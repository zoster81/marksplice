package marksplice_test

import (
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicSectionChildHeadingIDsExposeImmediateChildrenInSourceOrder(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\r\nintro\r\n\r\n## Alpha\r\nalpha\r\n\r\n### Deep Ω\r\ndeep\r\n\r\n## Beta\r\nbeta\r\n\r\n# Other\r\ntail\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sections := doc.Sections()
	byHeading := publicSectionsByHeadingText(t, doc, source, sections)

	root := byHeading["Root"]
	alpha := byHeading["Alpha"]
	deep := byHeading["Deep Ω"]
	beta := byHeading["Beta"]
	other := byHeading["Other"]

	rootChildren, ok := doc.SectionChildHeadingIDs(root.HeadingID())
	if !ok {
		t.Fatal("SectionChildHeadingIDs(root) ok = false, want true")
	}
	assertNodeIDs(t, rootChildren, []marksplice.NodeID{alpha.HeadingID(), beta.HeadingID()})
	assertSectionChildHeadingIDs(t, doc, alpha, []marksplice.NodeID{deep.HeadingID()})
	assertSectionChildHeadingIDs(t, doc, deep, nil)
	assertSectionChildHeadingIDs(t, doc, beta, nil)
	assertSectionChildHeadingIDs(t, doc, other, nil)

	rootChildren[0] = marksplice.NodeID{}
	assertSectionChildHeadingIDs(t, doc, root, []marksplice.NodeID{alpha.HeadingID(), beta.HeadingID()})

	rootAgain, ok := doc.Section(root.HeadingID())
	if !ok || rootAgain != root {
		t.Fatalf("Section(root heading) = %+v, %v; want %+v, true", rootAgain, ok, root)
	}
}

func TestPublicSectionChildHeadingIDsRefreshAfterAppendAndReparse(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n## First\none\n# Tail\ntail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	root := publicSectionsByHeadingText(t, doc, source, doc.Sections())["Root"]
	change, err := doc.PrepareAppendSectionChild(root.HeadingID(), []byte("## Second\ntwo\n"))
	if err != nil {
		t.Fatalf("PrepareAppendSectionChild() error = %v", err)
	}
	updatedSource, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	updated, err := marksplice.Parse(updatedSource)
	if err != nil {
		t.Fatalf("Parse(updated) error = %v", err)
	}
	sections := publicSectionsByHeadingText(t, updated, updatedSource, updated.Sections())
	assertSectionChildHeadingIDs(t, updated, sections["Root"], []marksplice.NodeID{sections["First"].HeadingID(), sections["Second"].HeadingID()})
}

func TestPublicSectionChildHeadingIDsHandleSkippedLevelsAndInvalidTargets(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n### Deep child\n#### Deeper\n## Later child\nparagraph\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sections := doc.Sections()
	byHeading := publicSectionsByHeadingText(t, doc, source, sections)

	root := byHeading["Root"]
	deep := byHeading["Deep child"]
	deeper := byHeading["Deeper"]
	later := byHeading["Later child"]

	assertSectionChildHeadingIDs(t, doc, root, []marksplice.NodeID{deep.HeadingID(), later.HeadingID()})
	assertSectionChildHeadingIDs(t, doc, deep, []marksplice.NodeID{deeper.HeadingID()})

	if got, ok := doc.SectionChildHeadingIDs(marksplice.NodeID{}); ok || got != nil {
		t.Fatalf("SectionChildHeadingIDs(zero) = %v, %v; want nil, false", got, ok)
	}
	var paragraph marksplice.NodeID
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraph = node.ID()
			break
		}
	}
	if paragraph.String() == "" {
		t.Fatal("paragraph node not found")
	}
	if got, ok := doc.SectionChildHeadingIDs(paragraph); ok || got != nil {
		t.Fatalf("SectionChildHeadingIDs(paragraph) = %v, %v; want nil, false", got, ok)
	}

	var nilDoc *marksplice.Document
	if got, ok := nilDoc.SectionChildHeadingIDs(root.HeadingID()); ok || got != nil {
		t.Fatalf("nil Document.SectionChildHeadingIDs() = %v, %v; want nil, false", got, ok)
	}
}

func assertSectionChildHeadingIDs(t *testing.T, doc *marksplice.Document, section marksplice.Section, want []marksplice.NodeID) {
	t.Helper()
	got, ok := doc.SectionChildHeadingIDs(section.HeadingID())
	if !ok {
		t.Fatalf("SectionChildHeadingIDs(%v) ok = false, want true", section.HeadingID())
	}
	assertNodeIDs(t, got, want)
}

func assertNodeIDs(t *testing.T, got, want []marksplice.NodeID) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("NodeID count = %d, want %d: got %v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("NodeID[%d] = %v, want %v", index, got[index], want[index])
		}
	}
}
