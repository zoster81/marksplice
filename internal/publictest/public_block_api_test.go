package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicHeadingDetailAndRenamePreserveSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		level       int
		style       marksplice.HeadingStyle
		content     []byte
		replacement []byte
		want        []byte
	}{
		{
			name:        "ATX keeps markers spacing and CRLF",
			source:      []byte("  ### Old *heading* ###  \r\n\r\nParagraph.\r\n"),
			level:       3,
			style:       marksplice.HeadingStyleATX,
			content:     []byte("Old *heading*"),
			replacement: []byte("New **heading**"),
			want:        []byte("  ### New **heading** ###  \r\n\r\nParagraph.\r\n"),
		},
		{
			name:        "Setext keeps underline spacing and CRLF",
			source:      []byte("Old heading\r\n-----------   \r\n\r\nParagraph.\r\n"),
			level:       2,
			style:       marksplice.HeadingStyleSetext,
			content:     []byte("Old heading"),
			replacement: []byte("New heading"),
			want:        []byte("New heading\r\n-----------   \r\n\r\nParagraph.\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			var summary marksplice.Node
			for _, node := range doc.Nodes() {
				if node.Kind() == marksplice.KindHeading {
					summary = node
					break
				}
			}
			if summary.ID().String() == "" {
				t.Fatal("heading summary not found")
			}

			heading, ok := doc.Heading(summary.ID())
			if !ok {
				t.Fatalf("Heading(%q) ok = false, want true", summary.ID())
			}
			if heading.ID() != summary.ID() || heading.Level() != tt.level || heading.Style() != tt.style {
				t.Fatalf("heading detail = id %q level %d style %v; want id %q level %d style %v", heading.ID(), heading.Level(), heading.Style(), summary.ID(), tt.level, tt.style)
			}
			start := bytes.Index(tt.source, tt.content)
			wantRange := marksplice.Range{Start: start, End: start + len(tt.content)}
			if got := heading.Range(); got != wantRange {
				t.Fatalf("Heading.Range() = %v, want %v", got, wantRange)
			}

			prefix := append([]byte(nil), tt.source[:heading.Range().Start]...)
			suffix := append([]byte(nil), tt.source[heading.Range().End:]...)
			change, err := doc.PrepareRenameHeading(summary.ID(), tt.replacement)
			if err != nil {
				t.Fatalf("PrepareRenameHeading() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result = %q, want %q", got, tt.want)
			}
			if !bytes.Equal(got[:len(prefix)], prefix) {
				t.Fatal("bytes before heading content changed")
			}
			if !bytes.Equal(got[len(prefix)+len(tt.replacement):], suffix) {
				t.Fatal("bytes after heading content changed")
			}
		})
	}
}

func TestPublicHeadingRejectsInvalidTargetsAndUnsafeRename(t *testing.T) {
	t.Parallel()

	source := []byte("Heading\n-------\n\nParagraph.\n")
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
	if heading.ID().String() == "" || paragraph.ID().String() == "" {
		t.Fatal("expected heading and paragraph summaries")
	}
	if _, ok := doc.Heading(paragraph.ID()); ok {
		t.Fatal("Heading(paragraph) ok = true, want false")
	}
	var missing marksplice.NodeID
	if _, ok := doc.Heading(missing); ok {
		t.Fatal("Heading(zero ID) ok = true, want false")
	}
	if _, err := doc.PrepareRenameHeading(missing, []byte("new")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareRenameHeading(zero ID) error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareRenameHeading(paragraph.ID(), []byte("new")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareRenameHeading(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
	for _, replacement := range [][]byte{nil, []byte("line one\nline two"), []byte("---")} {
		if _, err := doc.PrepareRenameHeading(heading.ID(), replacement); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("PrepareRenameHeading(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
}

func TestPublicListItemDetailAndReplacementPreserveSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		target      string
		ordered     bool
		marker      byte
		replacement []byte
		want        []byte
	}{
		{
			name:        "ordered item preserves number delimiter spacing and LF",
			source:      []byte("7)  old item\n8)  keep\n"),
			target:      "old item",
			ordered:     true,
			marker:      ')',
			replacement: []byte("new **item**"),
			want:        []byte("7)  new **item**\n8)  keep\n"),
		},
		{
			name:        "nested unordered item preserves indentation marker and CRLF",
			source:      []byte("1. parent\r\n   +  old nested\r\n2. tail\r\n"),
			target:      "old nested",
			ordered:     false,
			marker:      '+',
			replacement: []byte("new nested"),
			want:        []byte("1. parent\r\n   +  new nested\r\n2. tail\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			var summary marksplice.Node
			var detail marksplice.ListItem
			for _, node := range doc.Nodes() {
				if node.Kind() != marksplice.KindListItem {
					continue
				}
				candidate, ok := doc.ListItem(node.ID())
				if !ok {
					t.Fatalf("ListItem(%q) ok = false", node.ID())
				}
				if got := string(tt.source[candidate.Range().Start:candidate.Range().End]); got == tt.target {
					summary = node
					detail = candidate
					break
				}
			}
			if summary.ID().String() == "" {
				t.Fatalf("list item %q not found", tt.target)
			}
			if detail.ID() != summary.ID() || detail.Ordered() != tt.ordered || detail.Marker() != tt.marker {
				t.Fatalf("list item detail = id %q ordered %v marker %q; want id %q ordered %v marker %q", detail.ID(), detail.Ordered(), detail.Marker(), summary.ID(), tt.ordered, tt.marker)
			}

			prefix := append([]byte(nil), tt.source[:detail.Range().Start]...)
			suffix := append([]byte(nil), tt.source[detail.Range().End:]...)
			change, err := doc.PrepareReplaceListItem(summary.ID(), tt.replacement)
			if err != nil {
				t.Fatalf("PrepareReplaceListItem() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result = %q, want %q", got, tt.want)
			}
			if !bytes.Equal(got[:len(prefix)], prefix) {
				t.Fatal("bytes before list-item content changed")
			}
			if !bytes.Equal(got[len(prefix)+len(tt.replacement):], suffix) {
				t.Fatal("bytes after list-item content changed")
			}
		})
	}
}

func TestPublicListItemRejectsInvalidTargetsAndReplacement(t *testing.T) {
	t.Parallel()

	source := []byte("- item\n\nParagraph.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	var item, paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindListItem:
			item = node
		case marksplice.KindParagraph:
			paragraph = node
		}
	}
	if item.ID().String() == "" || paragraph.ID().String() == "" {
		t.Fatal("expected promoted list item and paragraph")
	}
	if _, ok := doc.ListItem(paragraph.ID()); ok {
		t.Fatal("ListItem(paragraph) ok = true, want false")
	}
	var missing marksplice.NodeID
	if _, ok := doc.ListItem(missing); ok {
		t.Fatal("ListItem(zero ID) ok = true, want false")
	}
	if _, err := doc.PrepareReplaceListItem(missing, []byte("new")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareReplaceListItem(zero ID) error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareReplaceListItem(paragraph.ID(), []byte("new")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceListItem(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
	for _, replacement := range [][]byte{nil, []byte("line one\nline two"), []byte("---")} {
		if _, err := doc.PrepareReplaceListItem(item.ID(), replacement); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceListItem(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
}

func TestPublicTaskDetailAndStateChangePreserveSource(t *testing.T) {
	t.Parallel()

	source := []byte("* [X] keep uppercase\r\n1. parent\r\n   - [ ] nested\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	var checkedSummary, nestedSummary marksplice.Node
	var checked, nested marksplice.Task
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindTask {
			continue
		}
		detail, ok := doc.Task(node.ID())
		if !ok {
			t.Fatalf("Task(%q) ok = false", node.ID())
		}
		state := string(source[detail.Range().Start:detail.Range().End])
		switch state {
		case "X":
			checkedSummary, checked = node, detail
		case " ":
			nestedSummary, nested = node, detail
		}
	}
	if checkedSummary.ID().String() == "" || nestedSummary.ID().String() == "" {
		t.Fatal("expected checked and nested task summaries")
	}
	if !checked.Checked() || nested.Checked() {
		t.Fatalf("task states = checked %v nested %v; want true/false", checked.Checked(), nested.Checked())
	}
	if checked.Range().End-checked.Range().Start != 1 || nested.Range().End-nested.Range().Start != 1 {
		t.Fatalf("task ranges = %v/%v, want one-byte state spans", checked.Range(), nested.Range())
	}

	noOp, err := doc.PrepareSetTaskChecked(checkedSummary.ID(), true)
	if err != nil {
		t.Fatalf("PrepareSetTaskChecked(no-op) error = %v", err)
	}
	unchanged, err := noOp.Apply(source)
	if err != nil {
		t.Fatalf("Apply(no-op) error = %v", err)
	}
	if !bytes.Equal(unchanged, source) {
		t.Fatalf("no-op changed uppercase task source: %q", unchanged)
	}

	prefix := append([]byte(nil), source[:nested.Range().Start]...)
	suffix := append([]byte(nil), source[nested.Range().End:]...)
	change, err := doc.PrepareSetTaskChecked(nestedSummary.ID(), true)
	if err != nil {
		t.Fatalf("PrepareSetTaskChecked(true) error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply(true) error = %v", err)
	}
	want := []byte("* [X] keep uppercase\r\n1. parent\r\n   - [x] nested\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
	if !bytes.Equal(got[:len(prefix)], prefix) || !bytes.Equal(got[len(prefix)+1:], suffix) {
		t.Fatal("task state change modified bytes outside the one-byte state span")
	}
}

func TestPublicTaskRejectsInvalidTargets(t *testing.T) {
	t.Parallel()

	source := []byte("- [ ] task\n\nParagraph.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	var task, paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindTask:
			task = node
		case marksplice.KindParagraph:
			paragraph = node
		}
	}
	if task.ID().String() == "" || paragraph.ID().String() == "" {
		t.Fatal("expected promoted task and paragraph")
	}
	if _, ok := doc.Task(paragraph.ID()); ok {
		t.Fatal("Task(paragraph) ok = true, want false")
	}
	var missing marksplice.NodeID
	if _, ok := doc.Task(missing); ok {
		t.Fatal("Task(zero ID) ok = true, want false")
	}
	if _, err := doc.PrepareSetTaskChecked(missing, true); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareSetTaskChecked(zero ID) error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareSetTaskChecked(paragraph.ID(), true); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareSetTaskChecked(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}
func TestPublicTableCellDetailAndReplacementPreserveSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		target      string
		header      bool
		column      int
		replacement []byte
		want        []byte
	}{
		{
			name:        "body cell preserves pipes padding alignment and CRLF",
			source:      []byte("| Name | Value |\r\n| :--- | ---: |\r\n| alpha | old **value**  |\r\n"),
			target:      "old **value**",
			header:      false,
			column:      1,
			replacement: []byte("new *value*"),
			want:        []byte("| Name | Value |\r\n| :--- | ---: |\r\n| alpha | new *value*  |\r\n"),
		},
		{
			name:        "header cell without outer pipes preserves padding",
			source:      []byte("Name   | Value  \n:---   | ---:\nalpha  | old\n"),
			target:      "Value",
			header:      true,
			column:      1,
			replacement: []byte("Amount"),
			want:        []byte("Name   | Amount  \n:---   | ---:\nalpha  | old\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			var summary marksplice.Node
			var detail marksplice.TableCell
			for _, node := range doc.Nodes() {
				if node.Kind() != marksplice.KindTableCell {
					continue
				}
				candidate, ok := doc.TableCell(node.ID())
				if !ok {
					t.Fatalf("TableCell(%q) ok = false", node.ID())
				}
				if got := string(tt.source[candidate.Range().Start:candidate.Range().End]); got == tt.target {
					summary, detail = node, candidate
					break
				}
			}
			if summary.ID().String() == "" {
				t.Fatalf("table cell %q not found", tt.target)
			}
			if detail.ID() != summary.ID() || detail.Header() != tt.header || detail.Column() != tt.column {
				t.Fatalf("table detail = id %q header %v column %d; want id %q header %v column %d", detail.ID(), detail.Header(), detail.Column(), summary.ID(), tt.header, tt.column)
			}

			prefix := append([]byte(nil), tt.source[:detail.Range().Start]...)
			suffix := append([]byte(nil), tt.source[detail.Range().End:]...)
			change, err := doc.PrepareReplaceTableCell(summary.ID(), tt.replacement)
			if err != nil {
				t.Fatalf("PrepareReplaceTableCell() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result = %q, want %q", got, tt.want)
			}
			if !bytes.Equal(got[:len(prefix)], prefix) || !bytes.Equal(got[len(prefix)+len(tt.replacement):], suffix) {
				t.Fatal("table-cell replacement modified bytes outside content range")
			}
		})
	}
}

func TestPublicTableCellRejectsInvalidTargetsAndReplacement(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| - | - |\n| old | keep |\n\nParagraph.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var cell, paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindTableCell:
			detail, ok := doc.TableCell(node.ID())
			if ok && string(source[detail.Range().Start:detail.Range().End]) == "old" {
				cell = node
			}
		case marksplice.KindParagraph:
			paragraph = node
		}
	}
	if cell.ID().String() == "" || paragraph.ID().String() == "" {
		t.Fatal("expected promoted table cell and paragraph")
	}
	if _, ok := doc.TableCell(paragraph.ID()); ok {
		t.Fatal("TableCell(paragraph) ok = true, want false")
	}
	var missing marksplice.NodeID
	if _, err := doc.PrepareReplaceTableCell(missing, []byte("new")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareReplaceTableCell(zero ID) error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareReplaceTableCell(paragraph.ID(), []byte("new")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceTableCell(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
	for _, replacement := range [][]byte{nil, []byte("line one\nline two"), []byte("new | split")} {
		if _, err := doc.PrepareReplaceTableCell(cell.ID(), replacement); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceTableCell(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
}

func TestPublicFencedCodeDetailAndReplacementPreserveSource(t *testing.T) {
	t.Parallel()

	source := []byte("before\r\n\r\n  ````go meta  \r\n  old()\r\n   `````\t\r\n\r\nafter\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var summary marksplice.Node
	var detail marksplice.FencedCode
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindFencedCode {
			continue
		}
		candidate, ok := doc.FencedCode(node.ID())
		if !ok {
			t.Fatalf("FencedCode(%q) ok = false", node.ID())
		}
		if got := string(source[candidate.Range().Start:candidate.Range().End]); got == "old()" {
			summary, detail = node, candidate
			break
		}
	}
	if summary.ID().String() == "" {
		t.Fatal("supported fenced code not promoted")
	}

	replacement := []byte("fmt.Println(\"new\")")
	prefix := append([]byte(nil), source[:detail.Range().Start]...)
	suffix := append([]byte(nil), source[detail.Range().End:]...)
	change, err := doc.PrepareReplaceFencedCode(summary.ID(), replacement)
	if err != nil {
		t.Fatalf("PrepareReplaceFencedCode() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("before\r\n\r\n  ````go meta  \r\n  fmt.Println(\"new\")\r\n   `````\t\r\n\r\nafter\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
	if !bytes.Equal(got[:len(prefix)], prefix) || !bytes.Equal(got[len(prefix)+len(replacement):], suffix) {
		t.Fatal("fenced-code replacement modified bytes outside content range")
	}
}

func TestPublicMultilineFencedCodeDetailAndReplacementPreserveSource(t *testing.T) {
	t.Parallel()

	source := []byte("before\r\n\r\n````go\r\nold one\r\nold two\r\n  `````\t\r\n\r\nafter\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var summary marksplice.Node
	var detail marksplice.FencedCode
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindFencedCode {
			continue
		}
		candidate, ok := doc.FencedCode(node.ID())
		if !ok {
			t.Fatalf("FencedCode(%q) ok = false", node.ID())
		}
		if got := string(source[candidate.Range().Start:candidate.Range().End]); got == "old one\r\nold two" {
			summary, detail = node, candidate
			break
		}
	}
	if summary.ID().String() == "" {
		t.Fatal("multiline fenced code not promoted")
	}

	replacement := []byte("new one\r\nnew two\r\nnew three")
	prefix := append([]byte(nil), source[:detail.Range().Start]...)
	suffix := append([]byte(nil), source[detail.Range().End:]...)
	change, err := doc.PrepareReplaceFencedCode(summary.ID(), replacement)
	if err != nil {
		t.Fatalf("PrepareReplaceFencedCode(multiline) error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply(multiline) error = %v", err)
	}
	want := []byte("before\r\n\r\n````go\r\nnew one\r\nnew two\r\nnew three\r\n  `````\t\r\n\r\nafter\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("multiline result = %q, want %q", got, want)
	}
	if !bytes.Equal(got[:len(prefix)], prefix) || !bytes.Equal(got[len(prefix)+len(replacement):], suffix) {
		t.Fatal("multiline fenced-code replacement modified bytes outside content range")
	}

	if _, err := doc.PrepareReplaceFencedCode(summary.ID(), []byte("safe\r\n````\r\nrest")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceFencedCode(early closing fence) error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPublicFencedCodeFiltersUnsupportedShapeAndRejectsInvalidTargets(t *testing.T) {
	t.Parallel()

	indentedMultiline, err := marksplice.Parse([]byte("  ```go\n  one\n  two\n  ```\n"))
	if err != nil {
		t.Fatalf("Parse(indented multiline) error = %v", err)
	}
	for _, node := range indentedMultiline.Nodes() {
		if node.Kind() == marksplice.KindFencedCode {
			t.Fatal("indented multiline fenced-code shape was promoted publicly")
		}
	}

	source := []byte("```go\nold\n```\n\nParagraph.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var block, paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindFencedCode:
			block = node
		case marksplice.KindParagraph:
			paragraph = node
		}
	}
	if block.ID().String() == "" || paragraph.ID().String() == "" {
		t.Fatal("expected promoted fenced code and paragraph")
	}
	if _, ok := doc.FencedCode(paragraph.ID()); ok {
		t.Fatal("FencedCode(paragraph) ok = true, want false")
	}
	var missing marksplice.NodeID
	if _, err := doc.PrepareReplaceFencedCode(missing, []byte("new")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareReplaceFencedCode(zero ID) error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareReplaceFencedCode(paragraph.ID(), []byte("new")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceFencedCode(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
	for _, replacement := range [][]byte{nil, []byte("````")} {
		if _, err := doc.PrepareReplaceFencedCode(block.ID(), replacement); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceFencedCode(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
}
