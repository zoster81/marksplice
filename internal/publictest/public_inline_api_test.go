package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicSimpleInlineDetailsAndReplacementPreserveSource(t *testing.T) {
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
			name:        "strikethrough preserves Unicode and tilde delimiters",
			source:      []byte("prefix ~~caffè 東京~~ suffix\n"),
			kind:        marksplice.KindStrikethrough,
			target:      "caffè 東京",
			replacement: []byte("nuovo 東京"),
			want:        []byte("prefix ~~nuovo 東京~~ suffix\n"),
		},
		{
			name:        "code span preserves double backtick run",
			source:      []byte("before ``old`code`` after\n"),
			kind:        marksplice.KindCodeSpan,
			target:      "old`code",
			replacement: []byte("new`code"),
			want:        []byte("before ``new`code`` after\n"),
		},
		{
			name:        "emphasis preserves underscore delimiters and CRLF",
			source:      []byte("before _old_ after\r\n"),
			kind:        marksplice.KindEmphasis,
			target:      "old",
			replacement: []byte("new"),
			want:        []byte("before _new_ after\r\n"),
		},
		{
			name:        "strong preserves asterisk delimiters and CRLF",
			source:      []byte("before **old** after\r\n"),
			kind:        marksplice.KindStrong,
			target:      "old",
			replacement: []byte("new"),
			want:        []byte("before **new** after\r\n"),
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
				candidateRange, ok := publicSimpleInlineRange(doc, node)
				if !ok {
					t.Fatalf("typed inline lookup for %v failed", tt.kind)
				}
				if got := string(tt.source[candidateRange.Start:candidateRange.End]); got == tt.target {
					target = node
					sourceRange = candidateRange
					break
				}
			}
			if target.ID().String() == "" {
				t.Fatalf("public inline node kind %v with content %q not found", tt.kind, tt.target)
			}

			prefix := append([]byte(nil), tt.source[:sourceRange.Start]...)
			suffix := append([]byte(nil), tt.source[sourceRange.End:]...)
			change, err := preparePublicSimpleInline(doc, target.ID(), tt.kind, tt.replacement)
			if err != nil {
				t.Fatalf("prepare inline replacement error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result = %q, want %q", got, tt.want)
			}
			if !bytes.Equal(got[:len(prefix)], prefix) || !bytes.Equal(got[len(prefix)+len(tt.replacement):], suffix) {
				t.Fatal("inline replacement modified bytes outside typed content range")
			}
		})
	}
}

func TestPublicSimpleInlineFiltersUnsupportedShapesAndPreservesErrors(t *testing.T) {
	t.Parallel()

	normalizedCode, err := marksplice.Parse([]byte("` old `\n"))
	if err != nil {
		t.Fatalf("Parse(normalized code) error = %v", err)
	}
	for _, node := range normalizedCode.Nodes() {
		if node.Kind() == marksplice.KindCodeSpan {
			t.Fatal("normalized-space code span was promoted publicly")
		}
	}

	compound, err := marksplice.Parse([]byte("***old***\n"))
	if err != nil {
		t.Fatalf("Parse(compound emphasis) error = %v", err)
	}
	for _, node := range compound.Nodes() {
		if node.Kind() == marksplice.KindEmphasis || node.Kind() == marksplice.KindStrong {
			t.Fatalf("compound emphasis kind %v was promoted publicly", node.Kind())
		}
	}

	source := []byte("~~strike~~ and `code` and *em* and **strong**\n")
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
		case marksplice.KindStrikethrough, marksplice.KindCodeSpan, marksplice.KindEmphasis, marksplice.KindStrong:
			nodes[node.Kind()] = node
		}
	}
	if paragraph.ID().String() == "" || len(nodes) != 4 {
		t.Fatalf("public inline/paragraph discovery = paragraph %q inline count %d, want non-empty/4", paragraph.ID(), len(nodes))
	}

	invalid := map[marksplice.Kind][]byte{
		marksplice.KindStrikethrough: []byte("~new"),
		marksplice.KindCodeSpan:      []byte("`"),
		marksplice.KindEmphasis:      []byte("*new"),
		marksplice.KindStrong:        []byte("**new"),
	}
	for _, kind := range []marksplice.Kind{marksplice.KindStrikethrough, marksplice.KindCodeSpan, marksplice.KindEmphasis, marksplice.KindStrong} {
		if _, err := preparePublicSimpleInline(doc, marksplice.NodeID{}, kind, []byte("new")); !errors.Is(err, marksplice.ErrNodeNotFound) {
			t.Fatalf("prepare %v with zero ID error = %v, want ErrNodeNotFound", kind, err)
		}
		if _, err := preparePublicSimpleInline(doc, paragraph.ID(), kind, []byte("new")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
			t.Fatalf("prepare %v with paragraph error = %v, want ErrInvalidTargetKind", kind, err)
		}
		if _, err := preparePublicSimpleInline(doc, nodes[kind].ID(), kind, invalid[kind]); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("prepare %v unsafe replacement error = %v, want ErrInvalidReplacement", kind, err)
		}
	}
}

func publicSimpleInlineRange(doc *marksplice.Document, node marksplice.Node) (marksplice.Range, bool) {
	switch node.Kind() {
	case marksplice.KindStrikethrough:
		detail, ok := doc.Strikethrough(node.ID())
		return detail.Range(), ok
	case marksplice.KindCodeSpan:
		detail, ok := doc.CodeSpan(node.ID())
		return detail.Range(), ok
	case marksplice.KindEmphasis:
		detail, ok := doc.Emphasis(node.ID())
		return detail.Range(), ok
	case marksplice.KindStrong:
		detail, ok := doc.Strong(node.ID())
		return detail.Range(), ok
	default:
		return marksplice.Range{}, false
	}
}

func preparePublicSimpleInline(doc *marksplice.Document, id marksplice.NodeID, kind marksplice.Kind, replacement []byte) (marksplice.ChangeSet, error) {
	switch kind {
	case marksplice.KindStrikethrough:
		return doc.PrepareReplaceStrikethrough(id, replacement)
	case marksplice.KindCodeSpan:
		return doc.PrepareReplaceCodeSpan(id, replacement)
	case marksplice.KindEmphasis:
		return doc.PrepareReplaceEmphasis(id, replacement)
	case marksplice.KindStrong:
		return doc.PrepareReplaceStrong(id, replacement)
	default:
		return marksplice.ChangeSet{}, marksplice.ErrInvalidTargetKind
	}
}
