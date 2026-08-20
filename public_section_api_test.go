package marksplice_test

import (
	"bytes"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicSectionsExposeHierarchyAndExactRanges(t *testing.T) {
	t.Parallel()

	source := []byte("preamble\r\n\r\n# One\r\nintro\r\n\r\n## Child\r\nchild body\r\n\r\n### Deep\r\ndeep body\r\n\r\nAnother Child\r\n-------------\r\nsecond body\r\n\r\n# Two\r\ntail\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	sections := doc.Sections()
	if len(sections) != 5 {
		t.Fatalf("Sections() count = %d, want 5", len(sections))
	}
	byHeading := publicSectionsByHeadingText(t, doc, source, sections)

	one := byHeading["One"]
	child := byHeading["Child"]
	deep := byHeading["Deep"]
	another := byHeading["Another Child"]
	two := byHeading["Two"]

	assertPublicSection(t, source, one, 1,
		byteRangeOf(t, source, "# One", "# Two"),
		byteRangeOf(t, source, "intro", "## Child"),
		marksplice.NodeID{}, false)
	assertPublicSection(t, source, child, 2,
		byteRangeOf(t, source, "## Child", "Another Child"),
		byteRangeOf(t, source, "child body", "### Deep"),
		one.HeadingID(), true)
	assertPublicSection(t, source, deep, 3,
		byteRangeOf(t, source, "### Deep", "Another Child"),
		byteRangeOf(t, source, "deep body", "Another Child"),
		child.HeadingID(), true)
	assertPublicSection(t, source, another, 2,
		byteRangeOf(t, source, "Another Child", "# Two"),
		byteRangeOf(t, source, "second body", "# Two"),
		one.HeadingID(), true)
	assertPublicSection(t, source, two, 1,
		marksplice.Range{Start: bytes.Index(source, []byte("# Two")), End: len(source)},
		marksplice.Range{Start: bytes.Index(source, []byte("tail")), End: len(source)},
		marksplice.NodeID{}, false)

	if got, ok := doc.Section(one.HeadingID()); !ok || got != one {
		t.Fatalf("Section(one heading) = %+v, %v; want %+v, true", got, ok, one)
	}
	var paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraph = node
			break
		}
	}
	if paragraph.ID().String() == "" {
		t.Fatal("public paragraph not found")
	}
	if _, ok := doc.Section(paragraph.ID()); ok {
		t.Fatal("Section(paragraph ID) ok = true, want false")
	}
	if _, ok := doc.Section(marksplice.NodeID{}); ok {
		t.Fatal("Section(zero ID) ok = true, want false")
	}

	sections[0] = marksplice.Section{}
	again := doc.Sections()
	if len(again) != 5 || again[0].HeadingID().String() == "" {
		t.Fatal("mutating returned Sections() slice changed document state")
	}
}

func TestPublicSectionZeroValueIsDeterministic(t *testing.T) {
	t.Parallel()

	var section marksplice.Section
	if section.HeadingID().String() != "" || section.Level() != 0 || section.Range() != (marksplice.Range{}) || section.BodyRange() != (marksplice.Range{}) {
		t.Fatalf("zero Section = heading %v level %d range %v body %v", section.HeadingID(), section.Level(), section.Range(), section.BodyRange())
	}
	if parent, ok := section.ParentHeadingID(); ok || parent.String() != "" {
		t.Fatalf("zero Section.ParentHeadingID() = %v, %v; want zero, false", parent, ok)
	}
}

func publicSectionsByHeadingText(t *testing.T, doc *marksplice.Document, source []byte, sections []marksplice.Section) map[string]marksplice.Section {
	t.Helper()

	result := make(map[string]marksplice.Section, len(sections))
	for _, section := range sections {
		heading, ok := doc.Heading(section.HeadingID())
		if !ok {
			t.Fatalf("Heading(%q) for section not found", section.HeadingID())
		}
		text := string(source[heading.Range().Start:heading.Range().End])
		result[text] = section
	}
	return result
}

func assertPublicSection(t *testing.T, source []byte, section marksplice.Section, level int, wantRange, wantBody marksplice.Range, wantParent marksplice.NodeID, wantParentOK bool) {
	t.Helper()

	if section.HeadingID().String() == "" {
		t.Fatal("section has zero heading ID")
	}
	if section.Level() != level {
		t.Fatalf("Section.Level() = %d, want %d", section.Level(), level)
	}
	if section.Range() != wantRange {
		t.Fatalf("Section.Range() = %v (%q), want %v (%q)", section.Range(), source[section.Range().Start:section.Range().End], wantRange, source[wantRange.Start:wantRange.End])
	}
	if section.BodyRange() != wantBody {
		t.Fatalf("Section.BodyRange() = %v (%q), want %v (%q)", section.BodyRange(), source[section.BodyRange().Start:section.BodyRange().End], wantBody, source[wantBody.Start:wantBody.End])
	}
	parent, ok := section.ParentHeadingID()
	if ok != wantParentOK || parent != wantParent {
		t.Fatalf("Section.ParentHeadingID() = %v, %v; want %v, %v", parent, ok, wantParent, wantParentOK)
	}
}

func byteRangeOf(t *testing.T, source []byte, startMarker, endMarker string) marksplice.Range {
	t.Helper()

	start := bytes.Index(source, []byte(startMarker))
	end := bytes.Index(source, []byte(endMarker))
	if start < 0 || end < 0 || end < start {
		t.Fatalf("range markers %q..%q not found in order", startMarker, endMarker)
	}
	return marksplice.Range{Start: start, End: end}
}
