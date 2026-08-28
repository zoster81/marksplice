package splice

import (
	"bytes"
	"errors"
	"testing"

	sourcepkg "github.com/zoster81/marksplice/internal/source"
)

func TestSetTaskCheckedChangesOnlyCheckboxStateByte(t *testing.T) {
	t.Parallel()

	source := []byte("- [ ] first\r\n* [X] second\r\n+ [x] third\r\n\r\nplain [ ] text\r\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	tasks := nodesOfKind(doc.Nodes(), KindTask)
	if len(tasks) != 3 {
		t.Fatalf("task count = %d, want 3", len(tasks))
	}
	if tasks[0].Checked || !tasks[1].Checked || !tasks[2].Checked {
		t.Fatalf("task checked states = %v, %v, %v; want false, true, true", tasks[0].Checked, tasks[1].Checked, tasks[2].Checked)
	}
	if got, want := string(source[tasks[0].Range.Start:tasks[0].Range.End]), "[ ]"; got != want {
		t.Fatalf("task marker range = %q, want %q", got, want)
	}
	if got, want := string(source[tasks[0].ContentRange.Start:tasks[0].ContentRange.End]), " "; got != want {
		t.Fatalf("task state range = %q, want %q", got, want)
	}

	change, err := doc.PrepareSetTaskChecked(tasks[0].ID, true)
	if err != nil {
		t.Fatalf("PrepareSetTaskChecked() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("- [x] first\r\n* [X] second\r\n+ [x] third\r\n\r\nplain [ ] text\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
}

func TestSetTaskUncheckedPreservesListMarkerAndNoOpPreservesUppercaseX(t *testing.T) {
	t.Parallel()

	source := []byte("* [X] Keep style\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	tasks := nodesOfKind(doc.Nodes(), KindTask)
	if len(tasks) != 1 {
		t.Fatalf("task count = %d, want 1", len(tasks))
	}

	noOp, err := doc.PrepareSetTaskChecked(tasks[0].ID, true)
	if err != nil {
		t.Fatalf("PrepareSetTaskChecked(no-op) error = %v", err)
	}
	unchanged, err := noOp.Apply(source)
	if err != nil {
		t.Fatalf("Apply(no-op) error = %v", err)
	}
	if !bytes.Equal(unchanged, source) {
		t.Fatalf("no-op changed source: %q", unchanged)
	}

	change, err := doc.PrepareSetTaskChecked(tasks[0].ID, false)
	if err != nil {
		t.Fatalf("PrepareSetTaskChecked(false) error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply(false) error = %v", err)
	}
	want := []byte("* [ ] Keep style\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestNestedTaskListItemIsMappedWithoutChangingIndentation(t *testing.T) {
	t.Parallel()

	source := []byte("1. parent\n   - [ ] nested\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	tasks := nodesOfKind(doc.Nodes(), KindTask)
	if len(tasks) != 1 {
		t.Fatalf("task count = %d, want 1", len(tasks))
	}

	change, err := doc.PrepareSetTaskChecked(tasks[0].ID, true)
	if err != nil {
		t.Fatalf("PrepareSetTaskChecked() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("1. parent\n   - [x] nested\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestReplaceSingleLineListItemContentPreservesSourceStyle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		targetText  string
		replacement []byte
		want        []byte
	}{
		{
			name:        "unordered CRLF preserves indentation marker and spacing",
			source:      []byte("intro\r\n\r\n  *   old item  \r\n  * keep\r\n"),
			targetText:  "old item  ",
			replacement: []byte("new **item**"),
			want:        []byte("intro\r\n\r\n  *   new **item**\r\n  * keep\r\n"),
		},
		{
			name:        "ordered preserves number and delimiter",
			source:      []byte("7)  old item\n8)  keep\n"),
			targetText:  "old item",
			replacement: []byte("new item"),
			want:        []byte("7)  new item\n8)  keep\n"),
		},
		{
			name:        "nested preserves parent and nested indentation",
			source:      []byte("1. parent\n   +  old nested\n2. tail\n"),
			targetText:  "old nested",
			replacement: []byte("new nested"),
			want:        []byte("1. parent\n   +  new nested\n2. tail\n"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, err := Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			items := nodesOfKind(doc.Nodes(), KindListItem)
			var target Node
			found := false
			for _, item := range items {
				if string(tt.source[item.ContentRange.Start:item.ContentRange.End]) == tt.targetText {
					target = item
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("list item with content %q not found; items = %+v", tt.targetText, items)
			}

			prefix := append([]byte(nil), tt.source[:target.ContentRange.Start]...)
			suffix := append([]byte(nil), tt.source[target.ContentRange.End:]...)
			change, err := doc.PrepareReplaceListItem(target.ID, tt.replacement)
			if err != nil {
				t.Fatalf("PrepareReplaceListItem() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, tt.want)
			}
			if !bytes.Equal(got[:len(prefix)], prefix) {
				t.Fatal("bytes before changed span were modified")
			}
			if !bytes.Equal(got[len(prefix)+len(tt.replacement):], suffix) {
				t.Fatal("bytes after changed span were modified")
			}
		})
	}
}

func TestPrepareReplaceListItemRejectsInvalidReplacementAndWrongTarget(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("- item\n\nparagraph\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	items := nodesOfKind(doc.Nodes(), KindListItem)
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(items) != 1 || len(paragraphs) != 1 {
		t.Fatalf("item/paragraph counts = %d/%d, want 1/1", len(items), len(paragraphs))
	}

	for _, replacement := range [][]byte{nil, []byte("line one\nline two"), []byte("---")} {
		if _, err := doc.PrepareReplaceListItem(items[0].ID, replacement); !errors.Is(err, ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceListItem(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
	if _, err := doc.PrepareReplaceListItem(paragraphs[0].ID, []byte("new")); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceListItem(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestPrepareSetTaskCheckedRejectsNonTask(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("Paragraph.\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(paragraphs) != 1 {
		t.Fatalf("paragraph count = %d, want 1", len(paragraphs))
	}
	if _, err := doc.PrepareSetTaskChecked(paragraphs[0].ID, true); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareSetTaskChecked(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestRenameHeadingPreservesATXSourceStyle(t *testing.T) {
	t.Parallel()

	source := []byte("  ### Old *heading* ###  \n\nParagraph.\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	headings := nodesOfKind(doc.Nodes(), KindHeading)
	if len(headings) != 1 {
		t.Fatalf("heading count = %d, want 1", len(headings))
	}
	target := headings[0]
	if target.Level != 3 || target.HeadingStyle != HeadingStyleATX {
		t.Fatalf("heading metadata = level %d style %d, want level 3 ATX", target.Level, target.HeadingStyle)
	}
	if got, want := string(source[target.Range.Start:target.Range.End]), "  ### Old *heading* ###  "; got != want {
		t.Fatalf("heading source range = %q, want %q", got, want)
	}
	if got, want := string(source[target.ContentRange.Start:target.ContentRange.End]), "Old *heading*"; got != want {
		t.Fatalf("heading content range = %q, want %q", got, want)
	}

	replacement := []byte("New **heading**")
	change, err := doc.PrepareRenameHeading(target.ID, replacement)
	if err != nil {
		t.Fatalf("PrepareRenameHeading() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("  ### New **heading** ###  \n\nParagraph.\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
}

func TestPrepareRenameHeadingNoOpAcceptsExistingMultilineSetextContent(t *testing.T) {
	t.Parallel()

	source := []byte("00\n0\n-")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	headings := nodesOfKind(doc.Nodes(), KindHeading)
	if len(headings) != 1 {
		t.Fatalf("heading count = %d, want 1", len(headings))
	}
	target := headings[0]
	if target.HeadingStyle != HeadingStyleSetext || target.ContentRange != (Range{Start: 0, End: 4}) {
		t.Fatalf("heading = %+v, want multiline Setext content [0,4)", target)
	}
	replacement := append([]byte(nil), source[target.ContentRange.Start:target.ContentRange.End]...)
	change, err := doc.PrepareRenameHeading(target.ID, replacement)
	if err != nil {
		t.Fatalf("PrepareRenameHeading(no-op) error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply(no-op) error = %v", err)
	}
	if !bytes.Equal(got, source) {
		t.Fatalf("no-op changed source: %q", got)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '1'
	if _, err := change.Apply(stale); !errors.Is(err, ErrSourceConflict) {
		t.Fatalf("Apply(stale no-op) error = %v, want ErrSourceConflict", err)
	}
}

func TestRenameHeadingPreservesSetextSourceStyleAndCRLF(t *testing.T) {
	t.Parallel()

	source := []byte("Old heading\r\n-----------   \r\n\r\nParagraph.\r\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	headings := nodesOfKind(doc.Nodes(), KindHeading)
	if len(headings) != 1 {
		t.Fatalf("heading count = %d, want 1", len(headings))
	}
	target := headings[0]
	if target.Level != 2 || target.HeadingStyle != HeadingStyleSetext {
		t.Fatalf("heading metadata = level %d style %d, want level 2 Setext", target.Level, target.HeadingStyle)
	}
	if got, want := string(source[target.Range.Start:target.Range.End]), "Old heading\r\n-----------   "; got != want {
		t.Fatalf("heading source range = %q, want %q", got, want)
	}
	if got, want := string(source[target.ContentRange.Start:target.ContentRange.End]), "Old heading"; got != want {
		t.Fatalf("heading content range = %q, want %q", got, want)
	}

	change, err := doc.PrepareRenameHeading(target.ID, []byte("New heading"))
	if err != nil {
		t.Fatalf("PrepareRenameHeading() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("New heading\r\n-----------   \r\n\r\nParagraph.\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
}

func TestRenameHeadingPreservesSetextLevelOneAndCRLineEndings(t *testing.T) {
	t.Parallel()

	source := []byte("Old heading\r=========  \r\rParagraph.\r")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	headings := nodesOfKind(doc.Nodes(), KindHeading)
	if len(headings) != 1 {
		t.Fatalf("heading count = %d, want 1", len(headings))
	}
	target := headings[0]
	if target.Level != 1 || target.HeadingStyle != HeadingStyleSetext {
		t.Fatalf("heading metadata = level %d style %d, want level 1 Setext", target.Level, target.HeadingStyle)
	}

	change, err := doc.PrepareRenameHeading(target.ID, []byte("New heading"))
	if err != nil {
		t.Fatalf("PrepareRenameHeading() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("New heading\r=========  \r\rParagraph.\r")
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
}

func TestPrepareRenameHeadingRejectsNonHeadingAndMultilineReplacement(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("# Heading\n\nParagraph.\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(paragraphs) != 1 {
		t.Fatalf("paragraph count = %d, want 1", len(paragraphs))
	}
	if _, err := doc.PrepareRenameHeading(paragraphs[0].ID, []byte("new")); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareRenameHeading(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}

	headings := nodesOfKind(doc.Nodes(), KindHeading)
	if len(headings) != 1 {
		t.Fatalf("heading count = %d, want 1", len(headings))
	}
	if _, err := doc.PrepareRenameHeading(headings[0].ID, []byte("line one\nline two")); !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("PrepareRenameHeading(multiline) error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPrepareRenameHeadingRejectsSetextReplacementThatChangesBlockKind(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("Heading\n-------\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	headings := nodesOfKind(doc.Nodes(), KindHeading)
	if len(headings) != 1 {
		t.Fatalf("heading count = %d, want 1", len(headings))
	}

	_, err = doc.PrepareRenameHeading(headings[0].ID, []byte("---"))
	if !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("PrepareRenameHeading() error = %v, want ErrInvalidReplacement", err)
	}
}

func TestReplaceTableCellPreservesUntouchedTableSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		targetText  string
		replacement []byte
		want        []byte
	}{
		{
			name:        "body cell preserves CRLF alignment and spacing",
			source:      []byte("| Name | Value |\r\n| :--- | ---: |\r\n| alpha | old **value**  |\r\n| beta | keep |\r\n"),
			targetText:  "old **value**",
			replacement: []byte("new *value*"),
			want:        []byte("| Name | Value |\r\n| :--- | ---: |\r\n| alpha | new *value*  |\r\n| beta | keep |\r\n"),
		},
		{
			name:        "header cell preserves table without outer pipes",
			source:      []byte("Name   | Value  \n:---   | ---:\nalpha  | old\n"),
			targetText:  "Value",
			replacement: []byte("Amount"),
			want:        []byte("Name   | Amount  \n:---   | ---:\nalpha  | old\n"),
		},
		{
			name:        "escaped pipe remains cell content",
			source:      []byte("| A | B |\n| - | - |\n| x | old \\| value |\n"),
			targetText:  "old \\| value",
			replacement: []byte("new \\| value"),
			want:        []byte("| A | B |\n| - | - |\n| x | new \\| value |\n"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, err := Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			cells := nodesOfKind(doc.Nodes(), KindTableCell)
			var target Node
			found := false
			for _, cell := range cells {
				if string(tt.source[cell.ContentRange.Start:cell.ContentRange.End]) == tt.targetText {
					target = cell
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("table cell with content %q not found; cells = %+v", tt.targetText, cells)
			}
			if !target.Editable {
				t.Fatal("mapped table cell Editable = false, want true")
			}
			mapping, ok := remapTableCellSource(tt.source, target)
			if !ok || mapping.ContentRange != target.ContentRange || mapping.Column != target.TableColumn {
				t.Fatalf("remapped table capability = %+v, %v; target content/column = %v/%d", mapping, ok, target.ContentRange, target.TableColumn)
			}

			prefix := append([]byte(nil), tt.source[:target.ContentRange.Start]...)
			suffix := append([]byte(nil), tt.source[target.ContentRange.End:]...)
			change, err := doc.PrepareReplaceTableCell(target.ID, tt.replacement)
			if err != nil {
				t.Fatalf("PrepareReplaceTableCell() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, tt.want)
			}
			if !bytes.Equal(got[:len(prefix)], prefix) {
				t.Fatal("bytes before changed cell content were modified")
			}
			if !bytes.Equal(got[len(prefix)+len(tt.replacement):], suffix) {
				t.Fatal("bytes after changed cell content were modified")
			}
		})
	}
}

func TestPrepareReplaceTableCellRejectsUnsafeReplacementAndWrongTarget(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| - | - |\n| old | keep |\n\nparagraph\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	cells := nodesOfKind(doc.Nodes(), KindTableCell)
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	var target Node
	found := false
	for _, cell := range cells {
		if string(source[cell.ContentRange.Start:cell.ContentRange.End]) == "old" {
			target = cell
			found = true
			break
		}
	}
	if !found || len(paragraphs) != 1 {
		t.Fatalf("target cell found = %v, paragraph count = %d; want true/1", found, len(paragraphs))
	}

	for _, replacement := range [][]byte{nil, []byte("line one\nline two"), []byte("new | split")} {
		if _, err := doc.PrepareReplaceTableCell(target.ID, replacement); !errors.Is(err, ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceTableCell(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
	if _, err := doc.PrepareReplaceTableCell(paragraphs[0].ID, []byte("new")); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceTableCell(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestReplaceSingleLineFencedCodePreservesFenceSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		targetText  string
		replacement []byte
		want        []byte
	}{
		{
			name:        "backtick CRLF preserves indentation info and longer closing fence",
			source:      []byte("before\r\n\r\n  ````go meta  \r\n  old()\r\n   `````\t\r\n\r\nafter\r\n"),
			targetText:  "old()",
			replacement: []byte("fmt.Println(\"new\")"),
			want:        []byte("before\r\n\r\n  ````go meta  \r\n  fmt.Println(\"new\")\r\n   `````\t\r\n\r\nafter\r\n"),
		},
		{
			name:        "tilde LF preserves three-space fence indentation",
			source:      []byte("   ~~~~ rust\n   old\n  ~~~~~\n"),
			targetText:  "old",
			replacement: []byte("new"),
			want:        []byte("   ~~~~ rust\n   new\n  ~~~~~\n"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, err := Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			blocks := nodesOfKind(doc.Nodes(), KindFencedCode)
			var target Node
			found := false
			for _, block := range blocks {
				if string(tt.source[block.ContentRange.Start:block.ContentRange.End]) == tt.targetText {
					target = block
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("fenced code block with content %q not found; blocks = %+v", tt.targetText, blocks)
			}
			if !target.Editable {
				t.Fatal("mapped fenced code Editable = false, want true")
			}
			mapping, ok := doc.FencedCodeSource(target.ID)
			if !ok || mapping.ContentRange != target.ContentRange || mapping.FenceChar == 0 || mapping.FenceLength < 3 {
				t.Fatalf("sidecar fenced-code mapping = %+v, %v; target content = %v", mapping, ok, target.ContentRange)
			}

			prefix := append([]byte(nil), tt.source[:target.ContentRange.Start]...)
			suffix := append([]byte(nil), tt.source[target.ContentRange.End:]...)
			change, err := doc.PrepareReplaceFencedCode(target.ID, tt.replacement)
			if err != nil {
				t.Fatalf("PrepareReplaceFencedCode() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, tt.want)
			}
			if !bytes.Equal(got[:len(prefix)], prefix) {
				t.Fatal("bytes before changed fenced-code content were modified")
			}
			if !bytes.Equal(got[len(prefix)+len(tt.replacement):], suffix) {
				t.Fatal("bytes after changed fenced-code content were modified")
			}
		})
	}
}

func TestPrepareReplaceFencedCodeRejectsUnsafeReplacementAndWrongTarget(t *testing.T) {
	t.Parallel()

	source := []byte("```go\nold\n```\n\nparagraph\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	blocks := nodesOfKind(doc.Nodes(), KindFencedCode)
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(blocks) != 1 || len(paragraphs) != 1 {
		t.Fatalf("fenced/paragraph counts = %d/%d, want 1/1", len(blocks), len(paragraphs))
	}

	for _, replacement := range [][]byte{nil, []byte("````")} {
		if _, err := doc.PrepareReplaceFencedCode(blocks[0].ID, replacement); !errors.Is(err, ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceFencedCode(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
	if _, err := doc.PrepareReplaceFencedCode(paragraphs[0].ID, []byte("new")); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceFencedCode(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestUnclosedSingleLineFencedCodeKeepsContiguousReplacement(t *testing.T) {
	t.Parallel()

	source := []byte("```go\nold\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	blocks := nodesOfKind(doc.Nodes(), KindFencedCode)
	if len(blocks) != 1 {
		t.Fatalf("fenced code count = %d, want 1 semantic observation", len(blocks))
	}
	block := blocks[0]
	codeMapping, codeOK := doc.FencedCodeSource(block.ID)
	if !block.Editable || !codeOK || codeMapping == (sourcepkg.FencedCodeMapping{}) || codeMapping.Closed {
		t.Fatalf("unclosed fenced code = editable %v mapping %+v, %v", block.Editable, codeMapping, codeOK)
	}
	blockMapping, _, _, blockOK := doc.FencedBlockSource(block.ID)
	if !blockOK || blockMapping.Closed || blockMapping.Range != (Range{Start: 0, End: len(source)}) {
		t.Fatalf("unclosed fenced block mapping = %+v, %v", blockMapping, blockOK)
	}
	change, err := doc.PrepareReplaceFencedCode(block.ID, []byte("new"))
	if err != nil {
		t.Fatalf("PrepareReplaceFencedCode() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if want := []byte("```go\nnew\n"); !bytes.Equal(got, want) {
		t.Fatalf("Apply() = %q, want %q", got, want)
	}
}
