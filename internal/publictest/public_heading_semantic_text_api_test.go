package publictest

import (
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicHeadingExposesParserDerivedSemanticText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
		want   string
		style  marksplice.HeadingStyle
	}{
		{
			name:   "ATX structured inline content",
			source: []byte("## **Installation**\n"),
			want:   "Installation",
			style:  marksplice.HeadingStyleATX,
		},
		{
			name:   "Setext structured inline content",
			source: []byte("**Installation**\n----------------\n"),
			want:   "Installation",
			style:  marksplice.HeadingStyleSetext,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			doc, err := marksplice.Parse(test.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			var heading marksplice.Heading
			for _, node := range doc.Nodes() {
				if node.Kind() != marksplice.KindHeading {
					continue
				}
				var ok bool
				heading, ok = doc.Heading(node.ID())
				if !ok {
					t.Fatalf("Heading(%q) not available", node.ID())
				}
				break
			}
			if heading.ID().String() == "" {
				t.Fatal("heading not found")
			}
			if got := heading.Text(); got != test.want {
				t.Fatalf("Heading.Text() = %q, want %q", got, test.want)
			}
			if heading.Style() != test.style {
				t.Fatalf("Heading.Style() = %v, want %v", heading.Style(), test.style)
			}
		})
	}

	var zero marksplice.Heading
	if got := zero.Text(); got != "" {
		t.Fatalf("zero Heading.Text() = %q, want empty", got)
	}
}
