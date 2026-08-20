package splice

import (
	"bytes"
	"errors"
	"testing"
)

func TestRemoveSectionDeletesCompleteSubtreeAndPreservesSurvivingHeadings(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\r\nintro\r\n\r\n## Remove\r\nbody\r\n\r\n### Deep\r\ndeep\r\n\r\n## Keep\r\nkeep\r\n\r\n# Tail\r\ntail\r\n")
	want := []byte("# Root\r\nintro\r\n\r\n## Keep\r\nkeep\r\n\r\n# Tail\r\ntail\r\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	remove := sectionByHeadingContent(t, doc, source, "Remove")

	change, err := doc.PrepareRemoveSection(remove.HeadingID)
	if err != nil {
		t.Fatalf("PrepareRemoveSection() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}

	candidate, err := Parse(got)
	if err != nil {
		t.Fatalf("Parse(result) error = %v", err)
	}
	if candidate.SectionCount() != 3 {
		t.Fatalf("result SectionCount() = %d, want Root/Keep/Tail", candidate.SectionCount())
	}
	for _, heading := range []string{"Root", "Keep", "Tail"} {
		if sectionByHeadingContent(t, candidate, got, heading).HeadingID == "" {
			t.Fatalf("surviving section %q not found", heading)
		}
	}
}

func TestRemoveSectionFailsClosedWhenSetextJoinChangesHeadingBoundary(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\nintro\n## Remove\ngone\nNext\n----\ntail\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	remove := sectionByHeadingContent(t, doc, source, "Remove")
	if _, err := doc.PrepareRemoveSection(remove.HeadingID); !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("PrepareRemoveSection() error = %v, want ErrInvalidReplacement", err)
	}
}

func TestRemoveSectionFailsClosedForHeadingWithoutDerivedSection(t *testing.T) {
	t.Parallel()

	source := []byte("# Heading\n")
	node := Node{
		ID:           "heading",
		Kind:         KindHeading,
		Range:        Range{Start: 0, End: 9},
		ContentRange: Range{Start: 2, End: 9},
		Level:        1,
		Editable:     true,
		TopLevel:     true,
	}
	doc := &Document{
		source:       source,
		nodes:        []Node{node},
		nodeIndex:    map[NodeID]int{node.ID: 0},
		sectionIndex: map[NodeID]int{},
	}
	if _, err := doc.PrepareRemoveSection(node.ID); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareRemoveSection(unindexed heading) error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestRangeAfterPatchShiftsOnlyDisjointFollowingRanges(t *testing.T) {
	t.Parallel()

	removed := Range{Start: 10, End: 20}
	tests := []struct {
		name  string
		input Range
		want  Range
		ok    bool
	}{
		{name: "before", input: Range{Start: 2, End: 8}, want: Range{Start: 2, End: 8}, ok: true},
		{name: "touches start", input: Range{Start: 2, End: 10}, want: Range{Start: 2, End: 10}, ok: true},
		{name: "after", input: Range{Start: 20, End: 30}, want: Range{Start: 10, End: 20}, ok: true},
		{name: "overlap", input: Range{Start: 8, End: 12}, ok: false},
		{name: "inside", input: Range{Start: 12, End: 18}, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := rangeAfterPatch(tt.input, removed, 0)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("rangeAfterPatch(%v, %v, 0) = %v, %v; want %v, %v", tt.input, removed, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestReplaceSectionBodyPreservesHierarchyAndAllowsContainerHeadings(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\nold\n## Child\nchild\n# Tail\ntail\n")
	replacement := []byte("> ## Quoted\n> body\n\n```text\n## fenced\n```\n\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	root := sectionByHeadingContent(t, doc, source, "Root")
	change, err := doc.PrepareReplaceSectionBody(root.HeadingID, replacement)
	if err != nil {
		t.Fatalf("PrepareReplaceSectionBody() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	candidate, err := Parse(got)
	if err != nil {
		t.Fatalf("Parse(result) error = %v", err)
	}
	if candidate.SectionCount() != doc.SectionCount() {
		t.Fatalf("candidate SectionCount() = %d, want %d", candidate.SectionCount(), doc.SectionCount())
	}
	updatedRoot := sectionByHeadingContent(t, candidate, got, "Root")
	if !bytes.Equal(got[updatedRoot.BodyRange.Start:updatedRoot.BodyRange.End], replacement) {
		t.Fatalf("updated body = %q, want %q", got[updatedRoot.BodyRange.Start:updatedRoot.BodyRange.End], replacement)
	}
}

func TestReplaceSectionBodyRejectsNewDocumentHeading(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\nold\n# Tail\ntail\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	root := sectionByHeadingContent(t, doc, source, "Root")
	if _, err := doc.PrepareReplaceSectionBody(root.HeadingID, []byte("new\n## Injected\ntext\n")); !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceSectionBody() error = %v, want ErrInvalidReplacement", err)
	}
}

func TestRangeAfterPatchSupportsPositiveReplacementDelta(t *testing.T) {
	t.Parallel()

	got, ok := rangeAfterPatch(Range{Start: 20, End: 30}, Range{Start: 10, End: 20}, 15)
	want := Range{Start: 25, End: 35}
	if !ok || got != want {
		t.Fatalf("rangeAfterPatch() = %v, %v; want %v, true", got, ok, want)
	}
}

func TestReplaceSectionAllowsContainerHeadingsAndChangesDescendantCount(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n## Old\nold\n### Deep\ndeep\n## Keep\nkeep\n")
	replacement := []byte("## New\n> ### Quoted\n> body\n\n```text\n### fenced\n```\n\n### Child A\na\n#### Child B\nb\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	old := sectionByHeadingContent(t, doc, source, "Old")
	change, err := doc.PrepareReplaceSection(old.HeadingID, replacement)
	if err != nil {
		t.Fatalf("PrepareReplaceSection() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	candidate, err := Parse(got)
	if err != nil {
		t.Fatalf("Parse(result) error = %v", err)
	}
	for _, heading := range []string{"Root", "New", "Child A", "Child B", "Keep"} {
		if sectionByHeadingContent(t, candidate, got, heading).HeadingID == "" {
			t.Fatalf("section %q not found after replacement", heading)
		}
	}
	if candidate.SectionCount() != 5 {
		t.Fatalf("candidate SectionCount() = %d, want 5", candidate.SectionCount())
	}
}

func TestParseSectionFragmentRejectsInvalidRootShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		replacement []byte
		level       int
	}{
		{name: "empty", replacement: nil, level: 1},
		{name: "preamble", replacement: []byte("text\n# Root\nbody\n"), level: 1},
		{name: "wrong level", replacement: []byte("## Root\nbody\n"), level: 1},
		{name: "sibling root", replacement: []byte("## One\na\n## Two\nb\n"), level: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseSectionFragment(tt.replacement, tt.level); !errors.Is(err, ErrInvalidReplacement) {
				t.Fatalf("parseSectionFragment() error = %v, want ErrInvalidReplacement", err)
			}
		})
	}
}

func TestSectionSubtreeEndIndexStopsAtFirstOutsideSection(t *testing.T) {
	t.Parallel()

	sections := []Section{
		{Range: Range{Start: 0, End: 100}},
		{Range: Range{Start: 10, End: 70}},
		{Range: Range{Start: 20, End: 40}},
		{Range: Range{Start: 70, End: 100}},
		{Range: Range{Start: 100, End: 120}},
	}
	if got := sectionSubtreeEndIndex(sections, 1, Range{Start: 10, End: 70}); got != 3 {
		t.Fatalf("sectionSubtreeEndIndex() = %d, want 3", got)
	}
}

func TestRangeAfterPatchHandlesZeroWidthInsertion(t *testing.T) {
	t.Parallel()

	patch := Range{Start: 10, End: 10}
	before, ok := rangeAfterPatch(Range{Start: 2, End: 10}, patch, 5)
	if !ok || before != (Range{Start: 2, End: 10}) {
		t.Fatalf("before range = %v, %v; want [2,10), true", before, ok)
	}
	after, ok := rangeAfterPatch(Range{Start: 10, End: 20}, patch, 5)
	if !ok || after != (Range{Start: 15, End: 25}) {
		t.Fatalf("after range = %v, %v; want [15,25), true", after, ok)
	}
}

func TestInsertSectionBeforeAndAfterKeepOriginalSectionsSourceOrdered(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n## Alpha\na\n### Deep\nd\n## Beta\nb\n# Tail\ntail\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	alpha := sectionByHeadingContent(t, doc, source, "Alpha")
	beta := sectionByHeadingContent(t, doc, source, "Beta")

	for _, tt := range []struct {
		name    string
		prepare func() (ChangeSet, error)
		want    []string
	}{
		{
			name: "before beta",
			prepare: func() (ChangeSet, error) {
				return doc.PrepareInsertSectionBefore(beta.HeadingID, []byte("## Inserted\n### Child\nbody\n"))
			},
			want: []string{"Root", "Alpha", "Deep", "Inserted", "Child", "Beta", "Tail"},
		},
		{
			name: "after alpha subtree",
			prepare: func() (ChangeSet, error) {
				return doc.PrepareInsertSectionAfter(alpha.HeadingID, []byte("## Inserted\n### Child\nbody\n"))
			},
			want: []string{"Root", "Alpha", "Deep", "Inserted", "Child", "Beta", "Tail"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			change, err := tt.prepare()
			if err != nil {
				t.Fatalf("prepare insert error = %v", err)
			}
			got, err := change.Apply(source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			candidate, err := Parse(got)
			if err != nil {
				t.Fatalf("Parse(result) error = %v", err)
			}
			if candidate.SectionCount() != len(tt.want) {
				t.Fatalf("SectionCount() = %d, want %d", candidate.SectionCount(), len(tt.want))
			}
			for i, want := range tt.want {
				section, ok := candidate.SectionAt(i)
				if !ok {
					t.Fatalf("SectionAt(%d) missing", i)
				}
				heading, ok := candidate.nodeByID(section.HeadingID)
				if !ok || string(got[heading.ContentRange.Start:heading.ContentRange.End]) != want {
					t.Fatalf("section %d heading = %q, want %q", i, got[heading.ContentRange.Start:heading.ContentRange.End], want)
				}
			}
		})
	}
}

func TestSectionOrderAfterMoveTracksMovedAndAnchorIndices(t *testing.T) {
	t.Parallel()

	sections := []Section{
		{HeadingID: "root", Level: 1, Range: Range{Start: 0, End: 100}},
		{HeadingID: "alpha", Level: 2, Range: Range{Start: 10, End: 40}},
		{HeadingID: "alpha-child", Level: 3, Range: Range{Start: 20, End: 40}},
		{HeadingID: "beta", Level: 2, Range: Range{Start: 40, End: 70}},
		{HeadingID: "gamma", Level: 2, Range: Range{Start: 70, End: 100}},
		{HeadingID: "tail", Level: 1, Range: Range{Start: 100, End: 120}},
	}

	forward, movedIndex, anchorIndex, ok := sectionOrderAfterMove(sections, 1, 3, "beta", true)
	if !ok || movedIndex != 2 || anchorIndex != 1 {
		t.Fatalf("forward indices = moved %d anchor %d ok %v; want 2,1,true", movedIndex, anchorIndex, ok)
	}
	assertSectionOrder(t, forward, []NodeID{"root", "beta", "alpha", "alpha-child", "gamma", "tail"})

	backward, movedIndex, anchorIndex, ok := sectionOrderAfterMove(sections, 4, 5, "alpha", false)
	if !ok || movedIndex != 1 || anchorIndex != 2 {
		t.Fatalf("backward indices = moved %d anchor %d ok %v; want 1,2,true", movedIndex, anchorIndex, ok)
	}
	assertSectionOrder(t, backward, []NodeID{"root", "gamma", "alpha", "alpha-child", "beta", "tail"})
}

func TestMovedSectionCandidateOffsetAccountsForRemoval(t *testing.T) {
	t.Parallel()

	moved := Range{Start: 20, End: 40}
	if got, ok := movedSectionCandidateOffset(moved, 5); !ok || got != 5 {
		t.Fatalf("backward offset = %d, %v; want 5,true", got, ok)
	}
	if got, ok := movedSectionCandidateOffset(moved, 80); !ok || got != 60 {
		t.Fatalf("forward offset = %d, %v; want 60,true", got, ok)
	}
	if _, ok := movedSectionCandidateOffset(moved, 30); ok {
		t.Fatal("inside-subtree offset ok = true, want false")
	}
}

func assertSectionOrder(t *testing.T, sections []Section, want []NodeID) {
	t.Helper()
	if len(sections) != len(want) {
		t.Fatalf("section count = %d, want %d", len(sections), len(want))
	}
	for i, id := range want {
		if sections[i].HeadingID != id {
			t.Fatalf("section %d ID = %q, want %q", i, sections[i].HeadingID, id)
		}
	}
}

func sectionByHeadingContent(t *testing.T, doc *Document, source []byte, content string) Section {
	t.Helper()
	for _, section := range doc.sections {
		heading, ok := doc.nodeByID(section.HeadingID)
		if !ok || !heading.ContentRange.Valid(len(source)) {
			continue
		}
		if string(source[heading.ContentRange.Start:heading.ContentRange.End]) == content {
			return section
		}
	}
	t.Fatalf("section governed by heading %q not found", content)
	return Section{}
}
