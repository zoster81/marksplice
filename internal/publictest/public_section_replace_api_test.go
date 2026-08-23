package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicReplaceSectionReplacesExactSubtreeAndMayChangeHeadingStyle(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\r\nintro\r\n\r\n## Old Child\r\nold body\r\n\r\n### Old Deep\r\ndeep\r\n\r\n## Sibling\r\nsibling\r\n\r\n# Tail\r\ntail\r\n")
	replacement := []byte("New Child\r\n---------\r\nnew π body\r\n\r\n### New Deep\r\nnew deep\r\n\r\n")
	want := []byte("# Root\r\nintro\r\n\r\nNew Child\r\n---------\r\nnew π body\r\n\r\n### New Deep\r\nnew deep\r\n\r\n## Sibling\r\nsibling\r\n\r\n# Tail\r\ntail\r\n")

	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	target := publicSectionsByHeadingText(t, doc, source, doc.Sections())["Old Child"]
	if target.HeadingID().String() == "" {
		t.Fatal("Old Child section not found")
	}
	prefix := append([]byte(nil), source[:target.Range().Start]...)
	suffix := append([]byte(nil), source[target.Range().End:]...)

	change, err := doc.PrepareReplaceSection(target.HeadingID(), replacement)
	if err != nil {
		t.Fatalf("PrepareReplaceSection() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
	if !bytes.Equal(got[:len(prefix)], prefix) || !bytes.Equal(got[len(prefix)+len(replacement):], suffix) {
		t.Fatal("section replacement modified bytes outside Section.Range()")
	}

	updated, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(result) error = %v", err)
	}
	byHeading := publicSectionsByHeadingText(t, updated, got, updated.Sections())
	newChild := byHeading["New Child"]
	if newChild.HeadingID().String() == "" || newChild.Level() != 2 {
		t.Fatalf("replacement root = %+v, want level-2 section", newChild)
	}
	heading, ok := updated.Heading(newChild.HeadingID())
	if !ok || heading.Style() != marksplice.HeadingStyleSetext {
		t.Fatalf("replacement heading = %+v, %v; want Setext", heading, ok)
	}
	newDeep := byHeading["New Deep"]
	parent, ok := newDeep.ParentHeadingID()
	if !ok || parent != newChild.HeadingID() {
		t.Fatalf("New Deep parent = %v, %v; want New Child", parent, ok)
	}
	if byHeading["Old Deep"].HeadingID().String() != "" || byHeading["Sibling"].HeadingID().String() == "" || byHeading["Tail"].HeadingID().String() == "" {
		t.Fatal("replacement did not replace only the target subtree")
	}
}

func TestPublicReplaceSectionSupportsRootAndFinalEOFTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		heading     string
		replacement []byte
		want        []byte
	}{
		{
			name:        "root keeps preamble and following root",
			source:      []byte("preamble\n\n# One\nbody\n## Child\nchild\n# Two\ntail\n"),
			heading:     "One",
			replacement: []byte("# Replacement\nnew\n## Nested\nnested\n"),
			want:        []byte("preamble\n\n# Replacement\nnew\n## Nested\nnested\n# Two\ntail\n"),
		},
		{
			name:        "final section may end at EOF",
			source:      []byte("# One\nbody\n# Last\nold"),
			heading:     "Last",
			replacement: []byte("# Final\nnew"),
			want:        []byte("# One\nbody\n# Final\nnew"),
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
			change, err := doc.PrepareReplaceSection(section.HeadingID(), tt.replacement)
			if err != nil {
				t.Fatalf("PrepareReplaceSection() error = %v", err)
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

func TestPublicReplaceSectionRejectsInvalidSubtreeFragmentsAndUnsafeJoin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		heading     string
		replacement []byte
	}{
		{
			name:        "empty fragment",
			source:      []byte("# Root\nbody\n# Tail\ntail\n"),
			heading:     "Root",
			replacement: nil,
		},
		{
			name:        "preamble before root",
			source:      []byte("# Root\nbody\n# Tail\ntail\n"),
			heading:     "Root",
			replacement: []byte("preamble\n# New\nbody\n"),
		},
		{
			name:        "wrong root level",
			source:      []byte("# Root\n## Child\nbody\n## Tail\ntail\n"),
			heading:     "Child",
			replacement: []byte("# New\nbody\n"),
		},
		{
			name:        "multiple sibling roots",
			source:      []byte("# Root\n## Child\nbody\n## Tail\ntail\n"),
			heading:     "Child",
			replacement: []byte("## First\na\n## Second\nb\n"),
		},
		{
			name:        "replacement merges into following Setext heading",
			source:      []byte("# Root\n## Child\nold\nSibling\n-------\ntail\n"),
			heading:     "Child",
			replacement: []byte("## New\njoined"),
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
			if _, err := doc.PrepareReplaceSection(section.HeadingID(), tt.replacement); !errors.Is(err, marksplice.ErrInvalidReplacement) {
				t.Fatalf("PrepareReplaceSection() error = %v, want ErrInvalidReplacement", err)
			}
		})
	}
}

func TestPublicReplaceSectionPreservesTargetAndStaleErrors(t *testing.T) {
	t.Parallel()

	source := []byte("# One\nbody\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if _, err := doc.PrepareReplaceSection(marksplice.NodeID{}, []byte("# New\nbody\n")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareReplaceSection(zero ID) error = %v, want ErrNodeNotFound", err)
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
	if _, err := doc.PrepareReplaceSection(paragraph.ID(), []byte("# New\nbody\n")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceSection(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}

	one := publicSectionsByHeadingText(t, doc, source, doc.Sections())["One"]
	change, err := doc.PrepareReplaceSection(one.HeadingID(), []byte("# New\nbody\n"))
	if err != nil {
		t.Fatalf("PrepareReplaceSection() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '!'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}
