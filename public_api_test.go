package marksplice_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicDocumentSnapshotAndNodeLookup(t *testing.T) {
	t.Parallel()

	input := []byte("# Title\r\n\r\n> internal nested paragraph stays unpromoted\r\n\r\n- [ ] promoted task item\r\n\r\nParagraph.\r\n")
	snapshot := append([]byte(nil), input...)

	doc, err := marksplice.Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	input[0] = 'X'

	nodes := doc.Nodes()
	if len(nodes) != 4 {
		t.Fatalf("public node count = %d, want heading, list item, task, and paragraph", len(nodes))
	}

	var heading, paragraph, listItem, task marksplice.Node
	for _, node := range nodes {
		switch node.Kind() {
		case marksplice.KindHeading:
			heading = node
		case marksplice.KindParagraph:
			paragraph = node
		case marksplice.KindListItem:
			listItem = node
		case marksplice.KindTask:
			task = node
		}
	}
	if heading.ID().String() == "" || paragraph.ID().String() == "" || listItem.ID().String() == "" || task.ID().String() == "" {
		t.Fatalf("promoted IDs = heading %v paragraph %v list item %v task %v, want non-empty", heading.ID(), paragraph.ID(), listItem.ID(), task.ID())
	}
	for _, node := range nodes {
		switch node.Kind() {
		case marksplice.KindHeading, marksplice.KindParagraph, marksplice.KindListItem, marksplice.KindTask:
		default:
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
	want := []byte("# Title\r\n\r\n> internal nested paragraph stays unpromoted\r\n\r\n- [ ] promoted task item\r\n\r\nChanged paragraph.\r\n")
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
	var heading marksplice.Heading
	if heading.ID().String() != "" || heading.Range() != (marksplice.Range{}) || heading.Level() != 0 || heading.Style() != marksplice.HeadingStyleUnknown {
		t.Fatalf("zero Heading behavior = id %v range %v level %d style %v", heading.ID(), heading.Range(), heading.Level(), heading.Style())
	}
	var listItem marksplice.ListItem
	if listItem.ID().String() != "" || listItem.Range() != (marksplice.Range{}) || listItem.Ordered() || listItem.Marker() != 0 {
		t.Fatalf("zero ListItem behavior = id %v range %v ordered %v marker %q", listItem.ID(), listItem.Range(), listItem.Ordered(), listItem.Marker())
	}
	var task marksplice.Task
	if task.ID().String() != "" || task.Range() != (marksplice.Range{}) || task.Checked() {
		t.Fatalf("zero Task behavior = id %v range %v checked %v", task.ID(), task.Range(), task.Checked())
	}
	var tableCell marksplice.TableCell
	if tableCell.ID().String() != "" || tableCell.Range() != (marksplice.Range{}) || tableCell.Header() || tableCell.Column() != 0 {
		t.Fatalf("zero TableCell behavior = id %v range %v header %v column %d", tableCell.ID(), tableCell.Range(), tableCell.Header(), tableCell.Column())
	}
	var fencedCode marksplice.FencedCode
	if fencedCode.ID().String() != "" || fencedCode.Range() != (marksplice.Range{}) {
		t.Fatalf("zero FencedCode behavior = id %v range %v", fencedCode.ID(), fencedCode.Range())
	}
	var strikethrough marksplice.Strikethrough
	if strikethrough.ID().String() != "" || strikethrough.Range() != (marksplice.Range{}) {
		t.Fatalf("zero Strikethrough behavior = id %v range %v", strikethrough.ID(), strikethrough.Range())
	}
	var codeSpan marksplice.CodeSpan
	if codeSpan.ID().String() != "" || codeSpan.Range() != (marksplice.Range{}) {
		t.Fatalf("zero CodeSpan behavior = id %v range %v", codeSpan.ID(), codeSpan.Range())
	}
	var emphasis marksplice.Emphasis
	if emphasis.ID().String() != "" || emphasis.Range() != (marksplice.Range{}) {
		t.Fatalf("zero Emphasis behavior = id %v range %v", emphasis.ID(), emphasis.Range())
	}
	var strong marksplice.Strong
	if strong.ID().String() != "" || strong.Range() != (marksplice.Range{}) {
		t.Fatalf("zero Strong behavior = id %v range %v", strong.ID(), strong.Range())
	}
	var inlineLink marksplice.InlineLink
	if inlineLink.ID().String() != "" || inlineLink.Range() != (marksplice.Range{}) {
		t.Fatalf("zero InlineLink behavior = id %v range %v", inlineLink.ID(), inlineLink.Range())
	}
	var referenceDefinition marksplice.ReferenceDefinition
	if referenceDefinition.ID().String() != "" || referenceDefinition.Range() != (marksplice.Range{}) {
		t.Fatalf("zero ReferenceDefinition behavior = id %v range %v", referenceDefinition.ID(), referenceDefinition.Range())
	}
	var autoLink marksplice.AutoLink
	if autoLink.ID().String() != "" || autoLink.Range() != (marksplice.Range{}) {
		t.Fatalf("zero AutoLink behavior = id %v range %v", autoLink.ID(), autoLink.Range())
	}
	var frontMatterField marksplice.FrontMatterField
	if frontMatterField.ID().String() != "" || frontMatterField.Range() != (marksplice.Range{}) || frontMatterField.Key() != "" || frontMatterField.Format() != marksplice.FrontMatterFormatUnknown {
		t.Fatalf("zero FrontMatterField behavior = id %v range %v key %q format %v", frontMatterField.ID(), frontMatterField.Range(), frontMatterField.Key(), frontMatterField.Format())
	}
	var htmlComment marksplice.HTMLComment
	if htmlComment.ID().String() != "" || htmlComment.Range() != (marksplice.Range{}) {
		t.Fatalf("zero HTMLComment behavior = id %v range %v", htmlComment.ID(), htmlComment.Range())
	}
	var htmlAnchor marksplice.HTMLAnchor
	if htmlAnchor.ID().String() != "" || htmlAnchor.Range() != (marksplice.Range{}) || htmlAnchor.Attribute() != marksplice.HTMLAnchorAttributeUnknown {
		t.Fatalf("zero HTMLAnchor behavior = id %v range %v attribute %v", htmlAnchor.ID(), htmlAnchor.Range(), htmlAnchor.Attribute())
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
