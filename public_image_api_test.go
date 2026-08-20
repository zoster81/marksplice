package marksplice_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicImageDestinationReplacementPreservesSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		replacement []byte
		want        []byte
	}{
		{
			name:        "raw destination preserves alt title and CRLF",
			source:      []byte("before ![alt text](old/path  \"A title\") after\r\n"),
			replacement: []byte("new/path"),
			want:        []byte("before ![alt text](new/path  \"A title\") after\r\n"),
		},
		{
			name:        "angle destination preserves wrappers spacing and title",
			source:      []byte("![alt](  <old path> 'title' )\n"),
			replacement: []byte("new path"),
			want:        []byte("![alt](  <new path> 'title' )\n"),
		},
		{
			name:        "balanced parentheses remain valid",
			source:      []byte("![alt](foo(bar) \"title\")\n"),
			replacement: []byte("next(baz)"),
			want:        []byte("![alt](next(baz) \"title\")\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			var image marksplice.Node
			for _, node := range doc.Nodes() {
				if node.Kind() == marksplice.KindImage {
					image = node
					break
				}
			}
			if image.ID().String() == "" {
				t.Fatal("public image node not found")
			}
			detail, ok := doc.Image(image.ID())
			if !ok {
				t.Fatalf("Image(%q) ok = false, want true", image.ID())
			}
			if detail.ID() != image.ID() {
				t.Fatalf("Image.ID() = %q, want %q", detail.ID(), image.ID())
			}
			if got := string(tt.source[detail.Range().Start:detail.Range().End]); got == "" {
				t.Fatal("Image.Range() selected an empty destination")
			}

			prefix := append([]byte(nil), tt.source[:detail.Range().Start]...)
			suffix := append([]byte(nil), tt.source[detail.Range().End:]...)
			change, err := doc.PrepareReplaceImageDestination(image.ID(), tt.replacement)
			if err != nil {
				t.Fatalf("PrepareReplaceImageDestination() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result = %q, want %q", got, tt.want)
			}
			if !bytes.Equal(got[:len(prefix)], prefix) || !bytes.Equal(got[len(prefix)+len(tt.replacement):], suffix) {
				t.Fatal("image replacement modified bytes outside Image.Range()")
			}
		})
	}
}

func TestPublicImagesFilterUnsupportedShapesAndPreserveErrors(t *testing.T) {
	t.Parallel()

	for name, source := range map[string][]byte{
		"compound alt":      []byte("![**alt**](old/path)\n"),
		"reference image":   []byte("![alt][id]\n\n[id]: old/path\n"),
		"empty destination": []byte("![alt]()\n"),
	} {
		doc, err := marksplice.Parse(source)
		if err != nil {
			t.Fatalf("Parse(%s) error = %v", name, err)
		}
		for _, node := range doc.Nodes() {
			if node.Kind() == marksplice.KindImage {
				t.Fatalf("%s was promoted as a public image", name)
			}
		}
	}

	source := []byte("![alt](old/path \"title\")\n\nparagraph\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var image, paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindImage:
			image = node
		case marksplice.KindParagraph:
			if paragraph.ID().String() == "" {
				paragraph = node
			}
		}
	}
	if image.ID().String() == "" || paragraph.ID().String() == "" {
		t.Fatalf("image/paragraph IDs = %q/%q, want non-empty", image.ID(), paragraph.ID())
	}

	if _, err := doc.PrepareReplaceImageDestination(marksplice.NodeID{}, []byte("new/path")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("zero ID error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareReplaceImageDestination(paragraph.ID(), []byte("new/path")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("paragraph target error = %v, want ErrInvalidTargetKind", err)
	}
	for _, replacement := range [][]byte{nil, []byte("line one\nline two"), []byte("new path"), []byte("new)tail")} {
		if _, err := doc.PrepareReplaceImageDestination(image.ID(), replacement); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("unsafe replacement %q error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}

	var zero marksplice.Image
	if zero.ID().String() != "" || zero.Range() != (marksplice.Range{}) {
		t.Fatalf("zero Image = id %q range %v", zero.ID(), zero.Range())
	}
}
