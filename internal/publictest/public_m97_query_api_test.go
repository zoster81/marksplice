package publictest

import (
	"errors"
	"slices"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM97QueryNodesFiltersKindsRangeAndLimit(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n\nfirst [link](<dest> \"title\").\n\n- parent\n  - child\n\n| A | B |\n| --- | --- |\n| one | two |\n\n## Child\n\n> quoted\n\n# Tail\n\ntail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	root := publicSectionsByHeadingText(t, doc, source, doc.Sections())["Root"]
	within := root.BodyRange()
	matches, err := doc.QueryNodes(marksplice.NodeQuery{
		Kinds:  []marksplice.Kind{marksplice.KindParagraph, marksplice.KindInlineLink, marksplice.KindListItem, marksplice.KindTable},
		Within: &within,
		Limit:  64,
	})
	if err != nil {
		t.Fatalf("QueryNodes() error = %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("QueryNodes() returned no matches")
	}
	for _, match := range matches {
		range_ := match.Range()
		if range_.Start < within.Start || range_.End > within.End {
			t.Fatalf("match %v range %v escapes Within %v", match.Node().Kind(), range_, within)
		}
	}

	var seenLink, seenList, seenTable bool
	for _, match := range matches {
		switch match.Node().Kind() {
		case marksplice.KindInlineLink:
			detail, ok := doc.InlineLink(match.Node().ID())
			if !ok || match.Range() != detail.Range() {
				t.Fatalf("link match range = %v, typed = %v/%v", match.Range(), detail.Range(), ok)
			}
			seenLink = true
		case marksplice.KindListItem:
			detail, ok := doc.ListItem(match.Node().ID())
			if !ok || match.Range() != detail.Range() {
				t.Fatalf("list match range = %v, typed = %v/%v", match.Range(), detail.Range(), ok)
			}
			seenList = true
		case marksplice.KindTable:
			detail, ok := doc.Table(match.Node().ID())
			if !ok || match.Range() != detail.Range() {
				t.Fatalf("table match range = %v, typed = %v/%v", match.Range(), detail.Range(), ok)
			}
			seenTable = true
		}
	}
	if !seenLink || !seenList || !seenTable {
		t.Fatalf("representative matches = link %v list %v table %v", seenLink, seenList, seenTable)
	}

	headings, err := doc.QueryNodes(marksplice.NodeQuery{Kinds: []marksplice.Kind{marksplice.KindHeading, marksplice.KindHeading}, Limit: 1})
	if err != nil {
		t.Fatalf("QueryNodes(headings) error = %v", err)
	}
	if len(headings) != 1 {
		t.Fatalf("QueryNodes(headings) count = %d, want 1", len(headings))
	}
	heading, ok := doc.Heading(headings[0].Node().ID())
	if !ok || heading.Level() != 1 || headings[0].Range() != heading.Range() {
		t.Fatalf("first heading match = %+v, typed = %+v/%v", headings[0], heading, ok)
	}

	all, err := doc.QueryNodes(marksplice.NodeQuery{Limit: 3})
	if err != nil || len(all) != 3 {
		t.Fatalf("QueryNodes(all, limit=3) = %d, %v", len(all), err)
	}
	publicNodes := doc.Nodes()
	for index, match := range all {
		if match.Node().ID() != publicNodes[index].ID() {
			t.Fatalf("QueryNodes source order[%d] = %v, Nodes() = %v", index, match.Node().ID(), publicNodes[index].ID())
		}
	}
	all[0] = marksplice.NodeMatch{}
	again, err := doc.QueryNodes(marksplice.NodeQuery{Limit: 1})
	if err != nil || len(again) != 1 || again[0].Node().ID().String() == "" {
		t.Fatalf("caller mutation leaked into query state: %v, %v", again, err)
	}
}

func TestM97QueryNodesRejectsInvalidQueries(t *testing.T) {
	t.Parallel()

	doc, err := marksplice.Parse([]byte("# A\n\nbody\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	negative := marksplice.Range{Start: -1, End: 1}
	pastEnd := marksplice.Range{Start: 0, End: 999}
	tooManyKinds := make([]marksplice.Kind, int(marksplice.KindMathExpression)+1)
	for index := range tooManyKinds {
		tooManyKinds[index] = marksplice.KindHeading
	}
	tests := []marksplice.NodeQuery{
		{},
		{Limit: -1},
		{Kinds: tooManyKinds, Limit: 1},
		{Kinds: []marksplice.Kind{marksplice.KindUnknown}, Limit: 1},
		{Kinds: []marksplice.Kind{marksplice.Kind(255)}, Limit: 1},
		{Within: &negative, Limit: 1},
		{Within: &pastEnd, Limit: 1},
	}
	for _, query := range tests {
		if _, err := doc.QueryNodes(query); !errors.Is(err, marksplice.ErrInvalidQuery) {
			t.Fatalf("QueryNodes(%+v) error = %v, want ErrInvalidQuery", query, err)
		}
	}
	var nilDoc *marksplice.Document
	if _, err := nilDoc.QueryNodes(marksplice.NodeQuery{Limit: 1}); !errors.Is(err, marksplice.ErrInvalidQuery) {
		t.Fatalf("nil Document.QueryNodes() error = %v, want ErrInvalidQuery", err)
	}
}

func TestM97QuerySectionsFiltersLevelsRangeAndLimit(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n\nintro\n\n## Child\n\nchild\n\n### Deep\n\ndeep\n\n# Tail\n\ntail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sections := publicSectionsByHeadingText(t, doc, source, doc.Sections())
	within := sections["Root"].Range()
	matches, err := doc.QuerySections(marksplice.SectionQuery{Levels: []int{2}, Within: &within, Limit: 10})
	if err != nil {
		t.Fatalf("QuerySections() error = %v", err)
	}
	if len(matches) != 1 || matches[0].HeadingID() != sections["Child"].HeadingID() {
		t.Fatalf("level-2 Root matches = %+v, want Child", matches)
	}

	first, err := doc.QuerySections(marksplice.SectionQuery{Limit: 1})
	if err != nil || len(first) != 1 || first[0].HeadingID() != sections["Root"].HeadingID() {
		t.Fatalf("first section = %+v, %v", first, err)
	}
	first[0] = marksplice.Section{}
	again, err := doc.QuerySections(marksplice.SectionQuery{Limit: 1})
	if err != nil || len(again) != 1 || again[0].HeadingID().String() == "" {
		t.Fatalf("caller mutation leaked into section query state: %+v, %v", again, err)
	}

	allLevelTwo, err := doc.QuerySections(marksplice.SectionQuery{Levels: []int{2, 2}, Limit: 10})
	if err != nil || len(allLevelTwo) != 1 || !slices.Equal([]marksplice.NodeID{allLevelTwo[0].HeadingID()}, []marksplice.NodeID{sections["Child"].HeadingID()}) {
		t.Fatalf("duplicate level filter result = %+v, %v", allLevelTwo, err)
	}
}

func TestM97QueriesAcceptEmptySnapshotsWithAnExplicitBound(t *testing.T) {
	t.Parallel()

	doc, err := marksplice.Parse(nil)
	if err != nil {
		t.Fatalf("Parse(nil) error = %v", err)
	}
	zero := marksplice.Range{}
	nodes, err := doc.QueryNodes(marksplice.NodeQuery{Within: &zero, Limit: 1})
	if err != nil || len(nodes) != 0 {
		t.Fatalf("empty QueryNodes() = %v, %v", nodes, err)
	}
	sections, err := doc.QuerySections(marksplice.SectionQuery{Within: &zero, Limit: 1})
	if err != nil || len(sections) != 0 {
		t.Fatalf("empty QuerySections() = %v, %v", sections, err)
	}
}

func TestM97QuerySectionsRejectsInvalidQueries(t *testing.T) {
	t.Parallel()

	doc, err := marksplice.Parse([]byte("# A\n\n## B\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	invalidRange := marksplice.Range{Start: 4, End: 999}
	tooManyLevels := []int{1, 1, 1, 1, 1, 1, 1}
	for _, query := range []marksplice.SectionQuery{
		{},
		{Limit: -1},
		{Levels: tooManyLevels, Limit: 1},
		{Levels: []int{0}, Limit: 1},
		{Levels: []int{7}, Limit: 1},
		{Within: &invalidRange, Limit: 1},
	} {
		if _, err := doc.QuerySections(query); !errors.Is(err, marksplice.ErrInvalidQuery) {
			t.Fatalf("QuerySections(%+v) error = %v, want ErrInvalidQuery", query, err)
		}
	}
	var nilDoc *marksplice.Document
	if _, err := nilDoc.QuerySections(marksplice.SectionQuery{Limit: 1}); !errors.Is(err, marksplice.ErrInvalidQuery) {
		t.Fatalf("nil Document.QuerySections() error = %v, want ErrInvalidQuery", err)
	}
}
