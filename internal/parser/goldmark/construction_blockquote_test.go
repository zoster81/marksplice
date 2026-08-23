package goldmark

import (
	"bytes"
	"testing"

	markparser "github.com/zoster81/marksplice/internal/parser"
)

func TestValidateTopLevelBlockquoteParagraphAcceptsExactMultilineRanges(t *testing.T) {
	t.Parallel()

	source := []byte("# Title\n\n> first *line*\n> second π\n\nTail.\n")
	start := bytes.Index(source, []byte("> first *line*"))
	firstStart := start + 2
	firstEnd := firstStart + len("first *line*")
	secondMarker := firstEnd + 1
	secondStart := secondMarker + 2
	secondEnd := secondStart + len("second π")

	err := ValidateTopLevelBlockquoteParagraph(
		source,
		markparser.Range{Start: start, End: secondEnd},
		[]markparser.Range{
			{Start: firstStart, End: firstEnd},
			{Start: secondStart, End: secondEnd},
		},
	)
	if err != nil {
		t.Fatalf("ValidateTopLevelBlockquoteParagraph() error = %v", err)
	}
}

func TestValidateNestedBlockquoteParagraphAcceptsExactHierarchy(t *testing.T) {
	t.Parallel()

	source := []byte("# Title\n\n> > first *line*\n> > second π\n\nTail.\n")
	start := bytes.Index(source, []byte("> > first *line*"))
	firstStart := start + 4
	firstEnd := firstStart + len("first *line*")
	secondMarker := firstEnd + 1
	secondStart := secondMarker + 4
	secondEnd := secondStart + len("second π")

	err := ValidateNestedBlockquoteParagraph(
		source,
		markparser.Range{Start: start, End: secondEnd},
		[]markparser.Range{
			{Start: firstStart, End: firstEnd},
			{Start: secondStart, End: secondEnd},
		},
		2,
	)
	if err != nil {
		t.Fatalf("ValidateNestedBlockquoteParagraph() error = %v", err)
	}
}

func TestValidateNestedBlockquoteParagraphRejectsExtraDepth(t *testing.T) {
	t.Parallel()

	source := []byte("> > > text\n")
	if err := ValidateNestedBlockquoteParagraph(
		source,
		markparser.Range{Start: 0, End: 10},
		[]markparser.Range{{Start: 6, End: 10}},
		2,
	); err == nil {
		t.Fatal("ValidateNestedBlockquoteParagraph() error = nil, want depth mismatch rejection")
	}
}

func TestValidateNestedBlockquoteBlocksAcceptsExactChildSequence(t *testing.T) {
	t.Parallel()

	inner := []byte("## Head\n\nfirst\nsecond π\n\n---\n\n```go\nx\n```\n")
	source := []byte("> > ## Head\n> > \n> > first\n> > second π\n> > \n> > ---\n> > \n> > ```go\n> > x\n> > ```\n")
	if err := ValidateNestedBlockquoteBlocks(source, markparser.Range{Start: 0, End: len(source) - 1}, inner, 2); err != nil {
		t.Fatalf("ValidateNestedBlockquoteBlocks() error = %v", err)
	}
}

func TestValidateNestedBlockquoteBlocksRejectsChangedChildKindAndHeadingLevel(t *testing.T) {
	t.Parallel()

	inner := []byte("## Head\n\ntext\n")
	changedKind := []byte("> > ## Head\n> > \n> > - item\n")
	if err := ValidateNestedBlockquoteBlocks(changedKind, markparser.Range{Start: 0, End: len(changedKind) - 1}, inner, 2); err == nil {
		t.Fatal("ValidateNestedBlockquoteBlocks() error = nil, want child-kind rejection")
	}
	changedLevel := []byte("> > ### Head\n> > \n> > text\n")
	if err := ValidateNestedBlockquoteBlocks(changedLevel, markparser.Range{Start: 0, End: len(changedLevel) - 1}, inner, 2); err == nil {
		t.Fatal("ValidateNestedBlockquoteBlocks() error = nil, want heading-level rejection")
	}
}

func TestValidateNestedBlockquoteBlocksAcceptsListHierarchy(t *testing.T) {
	t.Parallel()

	inner := []byte("- parent\n  - child\n\n1. [x] first\n2. [ ] second\n")
	source := []byte("> > - parent\n> >   - child\n> > \n> > 1. [x] first\n> > 2. [ ] second\n")
	if err := ValidateNestedBlockquoteBlocks(source, markparser.Range{Start: 0, End: len(source) - 1}, inner, 2); err != nil {
		t.Fatalf("ValidateNestedBlockquoteBlocks() error = %v", err)
	}
}

func TestValidateNestedBlockquoteBlocksRejectsChangedListHierarchy(t *testing.T) {
	t.Parallel()

	inner := []byte("- parent\n  - child\n")
	changedKind := []byte("> > 1. parent\n> >    1. child\n")
	if err := ValidateNestedBlockquoteBlocks(changedKind, markparser.Range{Start: 0, End: len(changedKind) - 1}, inner, 2); err == nil {
		t.Fatal("ValidateNestedBlockquoteBlocks() error = nil, want list-kind rejection")
	}
	changedHierarchy := []byte("> > - parent\n> > - child\n")
	if err := ValidateNestedBlockquoteBlocks(changedHierarchy, markparser.Range{Start: 0, End: len(changedHierarchy) - 1}, inner, 2); err == nil {
		t.Fatal("ValidateNestedBlockquoteBlocks() error = nil, want list-hierarchy rejection")
	}
}

func TestValidateNestedBlockquoteBlocksAcceptsReferenceAndTableHierarchy(t *testing.T) {
	t.Parallel()

	inner := []byte("[doc]: <https://example.test/a> \"Title\"\n\n| A | B |\n| :--- | :---: |\n| x | y |\n")
	source := []byte("> > [doc]: <https://example.test/a> \"Title\"\n> > \n> > | A | B |\n> > | :--- | :---: |\n> > | x | y |\n")
	if err := ValidateNestedBlockquoteBlocks(source, markparser.Range{Start: 0, End: len(source) - 1}, inner, 2); err != nil {
		t.Fatalf("ValidateNestedBlockquoteBlocks() error = %v", err)
	}
}

func TestValidateNestedBlockquoteBlocksRejectsChangedReferenceOrTableSemantics(t *testing.T) {
	t.Parallel()

	innerReference := []byte("[doc]: <https://example.test/a> \"Title\"\n")
	changedReference := []byte("> > [doc]: <https://example.test/b> \"Title\"\n")
	if err := ValidateNestedBlockquoteBlocks(changedReference, markparser.Range{Start: 0, End: len(changedReference) - 1}, innerReference, 2); err == nil {
		t.Fatal("ValidateNestedBlockquoteBlocks() error = nil, want reference-destination rejection")
	}

	innerTable := []byte("| A | B |\n| :--- | :---: |\n| x | y |\n")
	changedTable := []byte("> > | A | B |\n> > | --- | :---: |\n> > | x | y |\n")
	if err := ValidateNestedBlockquoteBlocks(changedTable, markparser.Range{Start: 0, End: len(changedTable) - 1}, innerTable, 2); err == nil {
		t.Fatal("ValidateNestedBlockquoteBlocks() error = nil, want table-alignment rejection")
	}
}

func TestValidateNestedBlockquoteBlocksAcceptsRecursiveBlockquoteHierarchy(t *testing.T) {
	t.Parallel()

	inner := []byte("> single\n\n> > deep\n")
	source := []byte("> > > > single\n> > > \n> > > > > deep\n")
	if err := ValidateNestedBlockquoteBlocks(source, markparser.Range{Start: 0, End: len(source) - 1}, inner, 3); err != nil {
		t.Fatalf("ValidateNestedBlockquoteBlocks() error = %v", err)
	}
}

func TestValidateNestedBlockquoteBlocksRejectsChangedRecursiveBlockquoteDepth(t *testing.T) {
	t.Parallel()

	inner := []byte("> > deep\n")
	changed := []byte("> > > > deep\n")
	if err := ValidateNestedBlockquoteBlocks(changed, markparser.Range{Start: 0, End: len(changed) - 1}, inner, 3); err == nil {
		t.Fatal("ValidateNestedBlockquoteBlocks() error = nil, want recursive-depth rejection")
	}
}

func TestValidateTopLevelBlockquoteParagraphRejectsMultipleChildBlocks(t *testing.T) {
	t.Parallel()

	source := []byte("> first\n> - item\n")
	firstStart := 2
	firstEnd := firstStart + len("first")
	secondStart := firstEnd + 1 + 2
	secondEnd := secondStart + len("- item")
	if err := ValidateTopLevelBlockquoteParagraph(
		source,
		markparser.Range{Start: 0, End: secondEnd},
		[]markparser.Range{{Start: firstStart, End: firstEnd}, {Start: secondStart, End: secondEnd}},
	); err == nil {
		t.Fatal("ValidateTopLevelBlockquoteParagraph() error = nil, want structural rejection")
	}
}
