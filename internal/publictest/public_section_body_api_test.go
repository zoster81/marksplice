package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicReplaceSectionBodyPreservesHeadingChildrenAndSurroundingBytes(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\r\nold body\r\n\r\n## Child\r\nchild body\r\n\r\n# Tail\r\ntail\r\n")
	replacement := []byte("new π body\r\n- item\r\n\r\n")
	want := []byte("# Root\r\nnew π body\r\n- item\r\n\r\n## Child\r\nchild body\r\n\r\n# Tail\r\ntail\r\n")

	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	root := publicSectionsByHeadingText(t, doc, source, doc.Sections())["Root"]
	if root.HeadingID().String() == "" {
		t.Fatal("Root section not found")
	}
	prefix := append([]byte(nil), source[:root.BodyRange().Start]...)
	suffix := append([]byte(nil), source[root.BodyRange().End:]...)

	change, err := doc.PrepareReplaceSectionBody(root.HeadingID(), replacement)
	if err != nil {
		t.Fatalf("PrepareReplaceSectionBody() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
	if !bytes.Equal(got[:len(prefix)], prefix) || !bytes.Equal(got[len(prefix)+len(replacement):], suffix) {
		t.Fatal("section body replacement modified bytes outside Section.BodyRange()")
	}

	updated, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(result) error = %v", err)
	}
	updatedRoot := publicSectionsByHeadingText(t, updated, got, updated.Sections())["Root"]
	body, ok := updated.SourceRange(updatedRoot.BodyRange())
	if !ok || !bytes.Equal(body, replacement) {
		t.Fatalf("updated Root body = %q, %v; want %q, true", body, ok, replacement)
	}
	if len(updated.Sections()) != len(doc.Sections()) {
		t.Fatalf("updated section count = %d, want %d", len(updated.Sections()), len(doc.Sections()))
	}
}

func TestPublicReplaceSectionBodySupportsEmptyAndEOFBodies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		heading     string
		replacement []byte
		want        []byte
	}{
		{
			name:        "empty direct body keeps nested subsection",
			source:      []byte("# Root\nintro\n## Child\nbody\n### Deep\ndeep\n## Sibling\nsibling\n"),
			heading:     "Child",
			replacement: nil,
			want:        []byte("# Root\nintro\n## Child\n### Deep\ndeep\n## Sibling\nsibling\n"),
		},
		{
			name:        "last section body may end at EOF without line ending",
			source:      []byte("# One\nbody\n# Last\nold"),
			heading:     "Last",
			replacement: []byte("new"),
			want:        []byte("# One\nbody\n# Last\nnew"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			section := publicSectionsByHeadingText(t, doc, tt.source, doc.Sections())[tt.heading]
			if section.HeadingID().String() == "" {
				t.Fatalf("section %q not found", tt.heading)
			}
			change, err := doc.PrepareReplaceSectionBody(section.HeadingID(), tt.replacement)
			if err != nil {
				t.Fatalf("PrepareReplaceSectionBody() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPublicReplaceSectionBodyRejectsNewOrChangedDocumentHeading(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		heading     string
		replacement []byte
	}{
		{
			name:        "new top-level heading",
			source:      []byte("# Root\nold\n# Tail\ntail\n"),
			heading:     "Root",
			replacement: []byte("new\n## Injected\ntext\n"),
		},
		{
			name:        "replacement merges into following Setext heading",
			source:      []byte("# Root\nold\nChild\n-----\nchild\n"),
			heading:     "Root",
			replacement: []byte("joined"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			section := publicSectionsByHeadingText(t, doc, tt.source, doc.Sections())[tt.heading]
			if _, err := doc.PrepareReplaceSectionBody(section.HeadingID(), tt.replacement); !errors.Is(err, marksplice.ErrInvalidReplacement) {
				t.Fatalf("PrepareReplaceSectionBody() error = %v, want ErrInvalidReplacement", err)
			}
		})
	}
}

func TestPublicReplaceSectionBodyPreservesTargetAndStaleErrors(t *testing.T) {
	t.Parallel()

	source := []byte("# One\nbody\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if _, err := doc.PrepareReplaceSectionBody(marksplice.NodeID{}, []byte("new\n")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareReplaceSectionBody(zero ID) error = %v, want ErrNodeNotFound", err)
	}

	var paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraph = node
			break
		}
	}
	if paragraph.ID().String() == "" {
		t.Fatal("paragraph not found")
	}
	if _, err := doc.PrepareReplaceSectionBody(paragraph.ID(), []byte("new\n")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceSectionBody(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}

	one := publicSectionsByHeadingText(t, doc, source, doc.Sections())["One"]
	change, err := doc.PrepareReplaceSectionBody(one.HeadingID(), []byte("new\n"))
	if err != nil {
		t.Fatalf("PrepareReplaceSectionBody() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '!'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}
