package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicLinkDetailsAndReplacementPreserveSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		kind        marksplice.Kind
		target      string
		replacement []byte
		want        []byte
	}{
		{
			name:        "inline link preserves angle destination title and CRLF",
			source:      []byte("before [label](<old/path> \"title\") after\r\n"),
			kind:        marksplice.KindInlineLink,
			target:      "old/path",
			replacement: []byte("new/path"),
			want:        []byte("before [label](<new/path> \"title\") after\r\n"),
		},
		{
			name:        "reference definition preserves label layout title and CRLF",
			source:      []byte("  [id]: <old/path> 'title'  \r\n\r\n[id]\r\n"),
			kind:        marksplice.KindReferenceDefinition,
			target:      "old/path",
			replacement: []byte("new/path"),
			want:        []byte("  [id]: <new/path> 'title'  \r\n\r\n[id]\r\n"),
		},
		{
			name:        "autolink preserves angle brackets",
			source:      []byte("before <https://old.example/path> after\n"),
			kind:        marksplice.KindAutoLink,
			target:      "https://old.example/path",
			replacement: []byte("https://new.example/path"),
			want:        []byte("before <https://new.example/path> after\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			var target marksplice.Node
			var sourceRange marksplice.Range
			for _, node := range doc.Nodes() {
				if node.Kind() != tt.kind {
					continue
				}
				candidateRange, ok := publicLinkRange(doc, node)
				if !ok {
					t.Fatalf("typed link lookup for %v failed", tt.kind)
				}
				if got := string(tt.source[candidateRange.Start:candidateRange.End]); got == tt.target {
					target = node
					sourceRange = candidateRange
					break
				}
			}
			if target.ID().String() == "" {
				t.Fatalf("public link node kind %v with target %q not found", tt.kind, tt.target)
			}

			prefix := append([]byte(nil), tt.source[:sourceRange.Start]...)
			suffix := append([]byte(nil), tt.source[sourceRange.End:]...)
			change, err := preparePublicLink(doc, target.ID(), tt.kind, tt.replacement)
			if err != nil {
				t.Fatalf("prepare link replacement error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result = %q, want %q", got, tt.want)
			}
			if !bytes.Equal(got[:len(prefix)], prefix) || !bytes.Equal(got[len(prefix)+len(tt.replacement):], suffix) {
				t.Fatal("link replacement modified bytes outside typed range")
			}
		})
	}
}

func TestPublicLinksFilterUnsupportedShapesAndPreserveErrors(t *testing.T) {
	t.Parallel()

	complex, err := marksplice.Parse([]byte("[**label**](old/path)\n"))
	if err != nil {
		t.Fatalf("Parse(complex inline link) error = %v", err)
	}
	for _, node := range complex.Nodes() {
		if node.Kind() == marksplice.KindInlineLink {
			t.Fatal("compound-label inline link was promoted publicly")
		}
	}

	source := []byte("[label](old/path)\n\n[id]: old/path\n\nhttps://old.example/path\n\nparagraph\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	var paragraph marksplice.Node
	nodes := make(map[marksplice.Kind]marksplice.Node)
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraph = node
		}
		switch node.Kind() {
		case marksplice.KindInlineLink, marksplice.KindReferenceDefinition, marksplice.KindAutoLink:
			nodes[node.Kind()] = node
		}
	}
	if paragraph.ID().String() == "" || len(nodes) != 3 {
		t.Fatalf("public link/paragraph discovery = paragraph %q link count %d, want non-empty/3", paragraph.ID(), len(nodes))
	}

	invalid := map[marksplice.Kind][]byte{
		marksplice.KindInlineLink:          []byte("new)tail"),
		marksplice.KindReferenceDefinition: []byte("new path"),
		marksplice.KindAutoLink:            []byte("not-a-link"),
	}
	for _, kind := range []marksplice.Kind{marksplice.KindInlineLink, marksplice.KindReferenceDefinition, marksplice.KindAutoLink} {
		if _, err := preparePublicLink(doc, marksplice.NodeID{}, kind, []byte("new/path")); !errors.Is(err, marksplice.ErrNodeNotFound) {
			t.Fatalf("prepare %v with zero ID error = %v, want ErrNodeNotFound", kind, err)
		}
		if _, err := preparePublicLink(doc, paragraph.ID(), kind, []byte("new/path")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
			t.Fatalf("prepare %v with paragraph error = %v, want ErrInvalidTargetKind", kind, err)
		}
		if _, err := preparePublicLink(doc, nodes[kind].ID(), kind, invalid[kind]); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("prepare %v unsafe replacement error = %v, want ErrInvalidReplacement", kind, err)
		}
	}
}

func publicLinkRange(doc *marksplice.Document, node marksplice.Node) (marksplice.Range, bool) {
	switch node.Kind() {
	case marksplice.KindInlineLink:
		detail, ok := doc.InlineLink(node.ID())
		return detail.Range(), ok
	case marksplice.KindReferenceDefinition:
		detail, ok := doc.ReferenceDefinition(node.ID())
		return detail.Range(), ok
	case marksplice.KindAutoLink:
		detail, ok := doc.AutoLink(node.ID())
		return detail.Range(), ok
	default:
		return marksplice.Range{}, false
	}
}

func preparePublicLink(doc *marksplice.Document, id marksplice.NodeID, kind marksplice.Kind, replacement []byte) (marksplice.ChangeSet, error) {
	switch kind {
	case marksplice.KindInlineLink:
		return doc.PrepareReplaceInlineLinkDestination(id, replacement)
	case marksplice.KindReferenceDefinition:
		return doc.PrepareReplaceReferenceDefinitionDestination(id, replacement)
	case marksplice.KindAutoLink:
		return doc.PrepareReplaceAutoLink(id, replacement)
	default:
		return marksplice.ChangeSet{}, marksplice.ErrInvalidTargetKind
	}
}
