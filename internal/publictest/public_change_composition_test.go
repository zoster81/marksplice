package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicComposeChangesAppliesIndependentPreparedMutationsAtomically(t *testing.T) {
	t.Parallel()

	source := []byte("# Old\n\nParagraph old.\n\n***\n\n> quote\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	var heading, paragraph, thematic, blockquote marksplice.Node
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindHeading:
			heading = node
		case marksplice.KindParagraph:
			paragraph = node
		case marksplice.KindThematicBreak:
			thematic = node
		case marksplice.KindBlockquote:
			blockquote = node
		}
	}
	rename, err := doc.PrepareRenameHeading(heading.ID(), []byte("New"))
	if err != nil {
		t.Fatalf("PrepareRenameHeading() error = %v", err)
	}
	replace, err := doc.PrepareReplaceParagraph(paragraph.ID(), []byte("Paragraph new."))
	if err != nil {
		t.Fatalf("PrepareReplaceParagraph() error = %v", err)
	}
	removeBreak, err := doc.PrepareRemoveThematicBreak(thematic.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveThematicBreak() error = %v", err)
	}
	removeQuote, err := doc.PrepareRemoveBlockquote(blockquote.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveBlockquote() error = %v", err)
	}

	combined, err := doc.ComposeChanges(rename, replace, removeBreak, removeQuote)
	if err != nil {
		t.Fatalf("ComposeChanges() error = %v", err)
	}
	got, err := combined.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("# New\n\nParagraph new.\n\n\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
	reversed, err := doc.ComposeChanges(removeQuote, removeBreak, replace, rename)
	if err != nil {
		t.Fatalf("ComposeChanges(reversed) error = %v", err)
	}
	reversedResult, err := reversed.Apply(source)
	if err != nil || !bytes.Equal(reversedResult, want) {
		t.Fatalf("reversed Apply() = %q, %v; want %q", reversedResult, err, want)
	}
}

func TestPublicComposeChangesRejectsCombinedSemanticInteraction(t *testing.T) {
	t.Parallel()

	source := []byte("left\n***\n***\nright\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var breaks []marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindThematicBreak {
			breaks = append(breaks, node)
		}
	}
	if len(breaks) != 2 {
		t.Fatalf("thematic-break count = %d, want 2", len(breaks))
	}
	first, err := doc.PrepareRemoveThematicBreak(breaks[0].ID())
	if err != nil {
		t.Fatalf("PrepareRemoveThematicBreak(first) error = %v", err)
	}
	second, err := doc.PrepareRemoveThematicBreak(breaks[1].ID())
	if err != nil {
		t.Fatalf("PrepareRemoveThematicBreak(second) error = %v", err)
	}
	if _, err := doc.ComposeChanges(first, second); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("ComposeChanges(interacting removals) error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPublicComposeChangesRejectsOverlapAndDifferentSnapshot(t *testing.T) {
	t.Parallel()

	source := []byte("# Old\n\nParagraph.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var heading marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindHeading {
			heading = node
			break
		}
	}
	first, err := doc.PrepareRenameHeading(heading.ID(), []byte("One"))
	if err != nil {
		t.Fatalf("PrepareRenameHeading(first) error = %v", err)
	}
	second, err := doc.PrepareRenameHeading(heading.ID(), []byte("Two"))
	if err != nil {
		t.Fatalf("PrepareRenameHeading(second) error = %v", err)
	}
	if _, err := doc.ComposeChanges(first, second); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("ComposeChanges(overlap) error = %v, want ErrInvalidReplacement", err)
	}

	otherSource := []byte("# Other\n")
	other, err := marksplice.Parse(otherSource)
	if err != nil {
		t.Fatalf("Parse(other) error = %v", err)
	}
	var otherHeading marksplice.Node
	for _, node := range other.Nodes() {
		if node.Kind() == marksplice.KindHeading {
			otherHeading = node
			break
		}
	}
	foreign, err := other.PrepareRenameHeading(otherHeading.ID(), []byte("Foreign"))
	if err != nil {
		t.Fatalf("other.PrepareRenameHeading() error = %v", err)
	}
	if _, err := doc.ComposeChanges(first, foreign); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("ComposeChanges(foreign) error = %v, want ErrSourceConflict", err)
	}
}

func TestPublicComposeChangesEmptyAndSingleRemainSnapshotBound(t *testing.T) {
	t.Parallel()

	var nilDocument *marksplice.Document
	if _, err := nilDocument.ComposeChanges(); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("nil Document.ComposeChanges() error = %v, want ErrSourceConflict", err)
	}

	source := []byte("Paragraph.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	var zeroChange marksplice.ChangeSet
	if _, err := doc.ComposeChanges(zeroChange); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("ComposeChanges(zero ChangeSet) error = %v, want ErrSourceConflict", err)
	}

	empty, err := doc.ComposeChanges()
	if err != nil {
		t.Fatalf("ComposeChanges() error = %v", err)
	}
	got, err := empty.Apply(source)
	if err != nil || !bytes.Equal(got, source) {
		t.Fatalf("empty Apply() = %q, %v; want unchanged source", got, err)
	}

	var paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraph = node
			break
		}
	}
	change, err := doc.PrepareReplaceParagraph(paragraph.ID(), []byte("Changed."))
	if err != nil {
		t.Fatalf("PrepareReplaceParagraph() error = %v", err)
	}
	combined, err := doc.ComposeChanges(change)
	if err != nil {
		t.Fatalf("ComposeChanges(single) error = %v", err)
	}
	got, err = combined.Apply(source)
	if err != nil || !bytes.Equal(got, []byte("Changed.\n")) {
		t.Fatalf("single Apply() = %q, %v", got, err)
	}
}

func TestPublicComposeChangesCombinesListMoveAndTableRowInsertion(t *testing.T) {
	t.Parallel()

	source := []byte("- a\n- b\n\n| H |\n| --- |\n| x |\n| y |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var items, rows []marksplice.Node
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindListItem:
			items = append(items, node)
		case marksplice.KindTableRow:
			rows = append(rows, node)
		}
	}
	if len(items) != 2 || len(rows) != 2 {
		t.Fatalf("items/rows = %d/%d, want 2/2", len(items), len(rows))
	}
	move, err := doc.PrepareMoveListItemBefore(items[1].ID(), items[0].ID())
	if err != nil {
		t.Fatalf("PrepareMoveListItemBefore() error = %v", err)
	}
	insert, err := doc.PrepareInsertTableRowAfter(rows[0].ID(), []byte("| z |\n"))
	if err != nil {
		t.Fatalf("PrepareInsertTableRowAfter() error = %v", err)
	}
	combined, err := doc.ComposeChanges(move, insert)
	if err != nil {
		t.Fatalf("ComposeChanges() error = %v", err)
	}
	got, err := combined.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("- b\n- a\n\n| H |\n| --- |\n| x |\n| z |\n| y |\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestPublicComposeChangesCombinesSectionInsertionAndTaskUpdate(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n\n- [ ] task\n\n## Alpha\na\n\n## Beta\nb\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var task marksplice.Node
	var headings []marksplice.Node
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindTask:
			task = node
		case marksplice.KindHeading:
			headings = append(headings, node)
		}
	}
	if len(headings) != 3 || task.ID().String() == "" {
		t.Fatalf("headings/task = %d/%v, want 3/non-zero", len(headings), task.ID())
	}
	check, err := doc.PrepareSetTaskChecked(task.ID(), true)
	if err != nil {
		t.Fatalf("PrepareSetTaskChecked() error = %v", err)
	}
	insert, err := doc.PrepareInsertSectionBefore(headings[2].ID(), []byte("## Inserted\nbody\n"))
	if err != nil {
		t.Fatalf("PrepareInsertSectionBefore() error = %v", err)
	}
	combined, err := doc.ComposeChanges(check, insert)
	if err != nil {
		t.Fatalf("ComposeChanges() error = %v", err)
	}
	got, err := combined.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("# Root\n\n- [x] task\n\n## Alpha\na\n\n## Inserted\nbody\n## Beta\nb\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestPublicComposeChangesCombinesSiblingListItemReplacements(t *testing.T) {
	t.Parallel()

	source := []byte("- one\n- two\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var items []marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindListItem {
			items = append(items, node)
		}
	}
	if len(items) != 2 {
		t.Fatalf("list-item count = %d, want 2", len(items))
	}
	first, err := doc.PrepareReplaceListItem(items[0].ID(), []byte("ONE"))
	if err != nil {
		t.Fatalf("PrepareReplaceListItem(first) error = %v", err)
	}
	second, err := doc.PrepareReplaceListItem(items[1].ID(), []byte("TWO"))
	if err != nil {
		t.Fatalf("PrepareReplaceListItem(second) error = %v", err)
	}
	combined, err := doc.ComposeChanges(first, second)
	if err != nil {
		t.Fatalf("ComposeChanges() error = %v", err)
	}
	got, err := combined.Apply(source)
	if err != nil || !bytes.Equal(got, []byte("- ONE\n- TWO\n")) {
		t.Fatalf("Apply() = %q, %v", got, err)
	}
}

func TestPublicComposeChangesCombinesIndependentReferenceDefinitionUpdates(t *testing.T) {
	t.Parallel()

	source := []byte("[a]: <one>\n[b]: <two>\n\n[a] [b]\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var definitions []marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindReferenceDefinition {
			definitions = append(definitions, node)
		}
	}
	if len(definitions) != 2 {
		t.Fatalf("reference-definition count = %d, want 2", len(definitions))
	}
	first, err := doc.PrepareReplaceReferenceDefinitionDestination(definitions[0].ID(), []byte("ONE"))
	if err != nil {
		t.Fatalf("PrepareReplaceReferenceDefinitionDestination(first) error = %v", err)
	}
	second, err := doc.PrepareReplaceReferenceDefinitionDestination(definitions[1].ID(), []byte("TWO"))
	if err != nil {
		t.Fatalf("PrepareReplaceReferenceDefinitionDestination(second) error = %v", err)
	}
	combined, err := doc.ComposeChanges(first, second)
	if err != nil {
		t.Fatalf("ComposeChanges() error = %v", err)
	}
	got, err := combined.Apply(source)
	if err != nil || !bytes.Equal(got, []byte("[a]: <ONE>\n[b]: <TWO>\n\n[a] [b]\n")) {
		t.Fatalf("Apply() = %q, %v", got, err)
	}
}

func TestPublicComposeChangesRejectsSameTableSemanticDelta(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| --- | --- |\n| x | y |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var table marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindTable {
			table = node
			break
		}
	}
	left, err := doc.PrepareSetTableColumnAlignment(table.ID(), 0, marksplice.TableAlignmentLeft)
	if err != nil {
		t.Fatalf("PrepareSetTableColumnAlignment(left) error = %v", err)
	}
	right, err := doc.PrepareSetTableColumnAlignment(table.ID(), 1, marksplice.TableAlignmentRight)
	if err != nil {
		t.Fatalf("PrepareSetTableColumnAlignment(right) error = %v", err)
	}
	if _, err := doc.ComposeChanges(left, right); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("ComposeChanges(same table) error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPublicComposeChangesCombinesNestedListContentUpdates(t *testing.T) {
	t.Parallel()

	source := []byte("- parent\n  - child\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var items []marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindListItem {
			items = append(items, node)
		}
	}
	if len(items) != 2 {
		t.Fatalf("list-item count = %d, want 2", len(items))
	}
	parent, err := doc.PrepareReplaceListItem(items[0].ID(), []byte("PARENT"))
	if err != nil {
		t.Fatalf("PrepareReplaceListItem(parent) error = %v", err)
	}
	child, err := doc.PrepareReplaceListItem(items[1].ID(), []byte("CHILD"))
	if err != nil {
		t.Fatalf("PrepareReplaceListItem(child) error = %v", err)
	}
	combined, err := doc.ComposeChanges(parent, child)
	if err != nil {
		t.Fatalf("ComposeChanges() error = %v", err)
	}
	got, err := combined.Apply(source)
	if err != nil || !bytes.Equal(got, []byte("- PARENT\n  - CHILD\n")) {
		t.Fatalf("Apply() = %q, %v", got, err)
	}
}
