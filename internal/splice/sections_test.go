package splice

import (
	"strings"
	"testing"
)

func TestSectionIndexExcludesContainerHeadingsAndPreservesHierarchy(t *testing.T) {
	t.Parallel()

	source := []byte("> # Container heading\r\n> text\r\n\r\n# One\r\nintro\r\n\r\n## Child\r\nbody\r\n\r\n# Two\r\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if doc.SectionCount() != 3 {
		t.Fatalf("SectionCount() = %d, want 3", doc.SectionCount())
	}

	one, ok := doc.SectionAt(0)
	if !ok || one.Level != 1 || one.HasParent {
		t.Fatalf("first section = %+v, %v; want level 1 root", one, ok)
	}
	child, ok := doc.SectionAt(1)
	if !ok || child.Level != 2 || !child.HasParent || child.ParentHeadingID != one.HeadingID {
		t.Fatalf("child section = %+v, %v; want parent %q", child, ok, one.HeadingID)
	}
	two, ok := doc.SectionAt(2)
	if !ok || two.Level != 1 || two.HasParent {
		t.Fatalf("third section = %+v, %v; want level 1 root", two, ok)
	}
	if one.Range.End != two.Range.Start || child.Range.End != two.Range.Start {
		t.Fatalf("section subtree ends = one %d child %d, want next root start %d", one.Range.End, child.Range.End, two.Range.Start)
	}
	if got, ok := doc.SectionByHeadingID(child.HeadingID); !ok || got != child {
		t.Fatalf("SectionByHeadingID(child) = %+v, %v; want %+v, true", got, ok, child)
	}
}

func TestSectionBodyStartSupportsLineEndingFormsAndEOF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		end  int
		want int
	}{
		{name: "LF", src: "# h\nbody", end: 3, want: 4},
		{name: "CRLF", src: "# h\r\nbody", end: 3, want: 5},
		{name: "CR", src: "# h\rbody", end: 3, want: 4},
		{name: "EOF", src: "# h", end: 3, want: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := sectionBodyStart([]byte(tt.src), tt.end)
			if err != nil {
				t.Fatalf("sectionBodyStart() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("sectionBodyStart() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSectionsPreserveIsolatedCRBoundaries(t *testing.T) {
	t.Parallel()

	source := []byte("# One\rbody\r# Two\r")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if doc.SectionCount() != 2 {
		t.Fatalf("SectionCount() = %d, want 2", doc.SectionCount())
	}
	one, _ := doc.SectionAt(0)
	two, _ := doc.SectionAt(1)
	if got := string(source[one.Range.Start:one.Range.End]); got != "# One\rbody\r" {
		t.Fatalf("first section source = %q, want %q", got, "# One\\rbody\\r")
	}
	if got := string(source[one.BodyRange.Start:one.BodyRange.End]); got != "body\r" {
		t.Fatalf("first section body = %q, want %q", got, "body\\r")
	}
	if got := string(source[two.Range.Start:two.Range.End]); got != "# Two\r" {
		t.Fatalf("second section source = %q, want %q", got, "# Two\\r")
	}
	if two.BodyRange.Start != len(source) || two.BodyRange.End != len(source) {
		t.Fatalf("second empty body range = %v, want [%d,%d)", two.BodyRange, len(source), len(source))
	}
}

func TestBuildSectionsFailsClosedOnInvalidHeadingOrderOrBoundary(t *testing.T) {
	t.Parallel()

	source := []byte("# one\n# two\n")
	nodes := []Node{
		{ID: "one", Kind: KindHeading, Range: Range{Start: 6, End: 11}, Level: 1, Editable: true, TopLevel: true},
		{ID: "two", Kind: KindHeading, Range: Range{Start: 0, End: 5}, Level: 1, Editable: true, TopLevel: true},
	}
	if _, _, err := buildSections(source, nodes); err == nil || !strings.Contains(err.Error(), "out of source order") {
		t.Fatalf("buildSections(out-of-order) error = %v, want source-order failure", err)
	}

	badBoundary := []Node{{ID: "bad", Kind: KindHeading, Range: Range{Start: 0, End: 2}, Level: 1, Editable: true, TopLevel: true}}
	if _, _, err := buildSections([]byte("# h\n"), badBoundary); err == nil {
		t.Fatal("buildSections(non-line-end boundary) error = nil, want failure")
	}

	if _, err := sectionBodyStart([]byte("x"), 2); err == nil {
		t.Fatal("sectionBodyStart(out of range) error = nil, want failure")
	}
}

func TestSectionsRemainDistinctForDuplicateHeadingText(t *testing.T) {
	t.Parallel()

	source := []byte("# Same\nfirst\n\n# Same\nsecond\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if doc.SectionCount() != 2 {
		t.Fatalf("SectionCount() = %d, want 2", doc.SectionCount())
	}
	first, _ := doc.SectionAt(0)
	second, _ := doc.SectionAt(1)
	if first.HeadingID == second.HeadingID || first.HeadingID == "" || second.HeadingID == "" {
		t.Fatalf("duplicate-text section IDs = %q/%q, want distinct non-empty", first.HeadingID, second.HeadingID)
	}
	if first.Range == second.Range || first.Range.End != second.Range.Start {
		t.Fatalf("duplicate-text section ranges = %v/%v, want distinct adjacent ranges", first.Range, second.Range)
	}
	if got, ok := doc.SectionByHeadingID(first.HeadingID); !ok || got != first {
		t.Fatalf("SectionByHeadingID(first) = %+v, %v; want %+v, true", got, ok, first)
	}
	if got, ok := doc.SectionByHeadingID(second.HeadingID); !ok || got != second {
		t.Fatalf("SectionByHeadingID(second) = %+v, %v; want %+v, true", got, ok, second)
	}
}

func TestEmptyDocumentHasNoSections(t *testing.T) {
	t.Parallel()

	doc, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse(nil) error = %v", err)
	}
	if doc.SectionCount() != 0 {
		t.Fatalf("SectionCount() = %d, want 0", doc.SectionCount())
	}
	if _, ok := doc.SectionAt(0); ok {
		t.Fatal("SectionAt(0) ok = true, want false")
	}
	if _, ok := doc.SectionByHeadingID("missing"); ok {
		t.Fatal("SectionByHeadingID(missing) ok = true, want false")
	}
}
