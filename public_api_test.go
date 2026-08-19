package marksplice_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicDocumentSnapshotAndNodeLookup(t *testing.T) {
	t.Parallel()

	input := []byte("# Title\r\n\r\n> internal nested paragraph stays unpromoted\r\n\r\n- [ ] internal task kind stays unpromoted\r\n\r\nParagraph.\r\n")
	snapshot := append([]byte(nil), input...)

	doc, err := marksplice.Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	input[0] = 'X'

	nodes := doc.Nodes()
	if len(nodes) != 2 {
		t.Fatalf("public node count = %d, want only heading and paragraph", len(nodes))
	}

	var heading, paragraph marksplice.Node
	for _, node := range nodes {
		switch node.Kind() {
		case marksplice.KindHeading:
			heading = node
		case marksplice.KindParagraph:
			paragraph = node
		}
	}
	if heading.ID().String() == "" || paragraph.ID().String() == "" {
		t.Fatalf("heading/paragraph IDs = %v/%v, want non-empty", heading.ID(), paragraph.ID())
	}
	for _, node := range nodes {
		if node.Kind() != marksplice.KindHeading && node.Kind() != marksplice.KindParagraph {
			t.Fatalf("Nodes() exposed unpromoted kind %v", node.Kind())
		}
	}

	found, ok := doc.Node(paragraph.ID())
	if !ok || found != paragraph {
		t.Fatalf("Node(%q) = %+v, %v; want paragraph, true", paragraph.ID(), found, ok)
	}
	var missing marksplice.NodeID
	if _, ok := doc.Node(missing); ok {
		t.Fatal("Node(zero ID) ok = true, want false")
	}

	nodes[0] = marksplice.Node{}
	again := doc.Nodes()
	if len(again) == 0 || again[0].ID().String() == "" {
		t.Fatal("mutating returned Nodes() slice changed document state")
	}

	change, err := doc.PrepareReplaceParagraph(paragraph.ID(), []byte("Changed paragraph."))
	if err != nil {
		t.Fatalf("PrepareReplaceParagraph() error = %v", err)
	}
	got, err := change.Apply(snapshot)
	if err != nil {
		t.Fatalf("Apply(snapshot) error = %v", err)
	}
	want := []byte("# Title\r\n\r\n> internal nested paragraph stays unpromoted\r\n\r\n- [ ] internal task kind stays unpromoted\r\n\r\nChanged paragraph.\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestPublicPreparedChangePreservesErrorCategories(t *testing.T) {
	t.Parallel()

	source := []byte("# Heading\n\nParagraph.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	var heading, paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindHeading:
			heading = node
		case marksplice.KindParagraph:
			paragraph = node
		}
	}

	var missing marksplice.NodeID
	if _, err := doc.PrepareReplaceParagraph(missing, []byte("new")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareReplaceParagraph(zero ID) error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareReplaceParagraph(heading.ID(), []byte("new")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceParagraph(heading) error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareReplaceParagraph(paragraph.ID(), nil); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceParagraph(empty) error = %v, want ErrInvalidReplacement", err)
	}

	change, err := doc.PrepareReplaceParagraph(paragraph.ID(), []byte("new"))
	if err != nil {
		t.Fatalf("PrepareReplaceParagraph() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '!'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}

func TestPublicParagraphDetailExposesPreciseTopLevelByteRange(t *testing.T) {
	t.Parallel()

	source := []byte("# Title\r\n\r\nParagraph with *formatting*.  \r\n\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	var heading, paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindHeading:
			heading = node
		case marksplice.KindParagraph:
			paragraph = node
		}
	}
	if paragraph.ID().String() == "" || heading.ID().String() == "" {
		t.Fatalf("heading/paragraph IDs = %v/%v, want non-empty", heading.ID(), paragraph.ID())
	}

	detail, ok := doc.Paragraph(paragraph.ID())
	if !ok {
		t.Fatalf("Paragraph(%q) ok = false, want true", paragraph.ID())
	}
	marker := []byte("Paragraph with *formatting*.  ")
	start := bytes.Index(source, marker)
	wantRange := marksplice.Range{Start: start, End: start + len(marker)}
	if got := detail.Range(); got != wantRange {
		t.Fatalf("Paragraph.Range() = %v, want %v", got, wantRange)
	}
	if got := string(source[detail.Range().Start:detail.Range().End]); got != string(marker) {
		t.Fatalf("paragraph bytes = %q, want %q", got, marker)
	}
	if detail.ID() != paragraph.ID() {
		t.Fatalf("Paragraph.ID() = %q, want %q", detail.ID(), paragraph.ID())
	}
	if _, ok := doc.Paragraph(heading.ID()); ok {
		t.Fatal("Paragraph(heading) ok = true, want false")
	}
	var missing marksplice.NodeID
	if _, ok := doc.Paragraph(missing); ok {
		t.Fatal("Paragraph(zero ID) ok = true, want false")
	}
}

func TestPublicZeroAndEmptyReadValuesAreDeterministic(t *testing.T) {
	t.Parallel()

	var node marksplice.Node
	if node.ID().String() != "" || node.Kind() != marksplice.KindUnknown {
		t.Fatalf("zero Node accessors returned non-zero values: id=%v kind=%v", node.ID(), node.Kind())
	}
	var paragraph marksplice.Paragraph
	if paragraph.ID().String() != "" || paragraph.Range() != (marksplice.Range{}) || !(marksplice.Range{}).Valid(0) {
		t.Fatalf("zero Paragraph/Range behavior = id %v range %v", paragraph.ID(), paragraph.Range())
	}
	var change marksplice.ChangeSet
	if _, err := change.Apply(nil); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("zero ChangeSet.Apply(nil) error = %v, want ErrSourceConflict", err)
	}

	doc, err := marksplice.Parse(nil)
	if err != nil {
		t.Fatalf("Parse(nil) error = %v", err)
	}
	if got := doc.Nodes(); len(got) != 0 {
		t.Fatalf("Parse(nil).Nodes() = %+v, want empty", got)
	}
}
