package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestR30DirectLinkAndImageContentMutationsPreserveOwnedSyntax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  []byte
		kind marksplice.Kind
		repl []byte
		want []byte
	}{
		{
			name: "link label variable length CRLF",
			src:  []byte("before [old label](<old path>  \"keep title\") after\r\n"),
			kind: marksplice.KindInlineLink,
			repl: []byte("new"),
			want: []byte("before [new](<old path>  \"keep title\") after\r\n"),
		},
		{
			name: "image alt Unicode variable byte length",
			src:  []byte("before ![old](asset.png 'keep title') after\n"),
			kind: marksplice.KindImage,
			repl: []byte("東京 image"),
			want: []byte("before ![東京 image](asset.png 'keep title') after\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.src)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			node := findNodeOfKind(t, doc, tt.kind)
			var change marksplice.ChangeSet
			switch tt.kind {
			case marksplice.KindInlineLink:
				change, err = doc.PrepareReplaceInlineLinkLabel(node.ID(), tt.repl)
			case marksplice.KindImage:
				change, err = doc.PrepareReplaceImageAlt(node.ID(), tt.repl)
			default:
				t.Fatalf("unexpected kind %v", tt.kind)
			}
			if err != nil {
				t.Fatalf("prepare content mutation error = %v", err)
			}
			got, err := change.Apply(tt.src)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("Apply() = %q, want %q", got, tt.want)
			}
			if _, err := marksplice.Parse(got); err != nil {
				t.Fatalf("Parse(applied) error = %v", err)
			}
		})
	}
}

func TestR30DirectLinkAndImageExistingTitleMutationsPreserveDelimiterAndTrivia(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  []byte
		kind marksplice.Kind
		repl []byte
		want []byte
	}{
		{
			name: "link double quote title",
			src:  []byte("[label](dest   \"old title\")\n"),
			kind: marksplice.KindInlineLink,
			repl: []byte("new"),
			want: []byte("[label](dest   \"new\")\n"),
		},
		{
			name: "image parenthesized title CRLF",
			src:  []byte("![alt](dest  (old title))\r\n"),
			kind: marksplice.KindImage,
			repl: []byte("a longer title"),
			want: []byte("![alt](dest  (a longer title))\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.src)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			node := findNodeOfKind(t, doc, tt.kind)
			var change marksplice.ChangeSet
			switch tt.kind {
			case marksplice.KindInlineLink:
				change, err = doc.PrepareReplaceInlineLinkTitle(node.ID(), tt.repl)
			case marksplice.KindImage:
				change, err = doc.PrepareReplaceImageTitle(node.ID(), tt.repl)
			}
			if err != nil {
				t.Fatalf("prepare title mutation error = %v", err)
			}
			got, err := change.Apply(tt.src)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("Apply() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestR30DirectLinkAndImageRicherMutationsFailClosed(t *testing.T) {
	t.Parallel()

	source := []byte("[label](dest) and ![alt](image.png)\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	link := findNodeOfKind(t, doc, marksplice.KindInlineLink)
	image := findNodeOfKind(t, doc, marksplice.KindImage)

	if _, err := doc.PrepareReplaceInlineLinkLabel(link.ID(), nil); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceInlineLinkLabel(empty) error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareReplaceImageAlt(image.ID(), []byte("bad\nalt")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceImageAlt(multiline) error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareReplaceInlineLinkTitle(link.ID(), []byte("title")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceInlineLinkTitle(absent title) error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareReplaceImageTitle(image.ID(), []byte("title")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceImageTitle(absent title) error = %v, want ErrInvalidReplacement", err)
	}

	change, err := doc.PrepareReplaceInlineLinkLabel(link.ID(), []byte("new label"))
	if err != nil {
		t.Fatalf("PrepareReplaceInlineLinkLabel() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[len(stale)-2] = 'X'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}
