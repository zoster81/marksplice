package marksplice_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicDocumentBuilderWritesCanonicalHeadingAndParagraphGFM(t *testing.T) {
	t.Parallel()

	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendHeading(2, "Title *with emphasis*"); err != nil {
		t.Fatalf("AppendHeading() error = %v", err)
	}
	if err := builder.AppendParagraph("Paragraph with [link](https://example.com) and Unicode π."); err != nil {
		t.Fatalf("AppendParagraph() error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("## Title *with emphasis*\n\nParagraph with [link](https://example.com) and Unicode π.\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
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
		t.Fatalf("generated nodes = %+v, want heading and paragraph", doc.Nodes())
	}
	headingDetail, ok := doc.Heading(heading.ID())
	if !ok || headingDetail.Level() != 2 || headingDetail.Style() != marksplice.HeadingStyleATX {
		t.Fatalf("generated heading = %+v, %v; want level-2 ATX", headingDetail, ok)
	}
	paragraphDetail, ok := doc.Paragraph(paragraph.ID())
	if !ok || string(got[paragraphDetail.Range().Start:paragraphDetail.Range().End]) != "Paragraph with [link](https://example.com) and Unicode π." {
		t.Fatalf("generated paragraph = %+v, %v; want exact inline GFM", paragraphDetail, ok)
	}
}

func TestPublicDocumentBuilderWritesCanonicalMultilineParagraph(t *testing.T) {
	t.Parallel()

	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendHeading(1, "Title"); err != nil {
		t.Fatalf("AppendHeading() error = %v", err)
	}
	content := "first line\nsecond *line* with Unicode π"
	if err := builder.AppendParagraph(content); err != nil {
		t.Fatalf("AppendParagraph(multiline) error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("# Title\n\nfirst line\nsecond *line* with Unicode π\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindParagraph {
			continue
		}
		paragraph, ok := doc.Paragraph(node.ID())
		if !ok {
			t.Fatalf("Paragraph(%q) ok = false", node.ID())
		}
		if string(got[paragraph.Range().Start:paragraph.Range().End]) != content {
			t.Fatalf("multiline paragraph source = %q, want %q", got[paragraph.Range().Start:paragraph.Range().End], content)
		}
		return
	}
	t.Fatal("generated multiline paragraph was not promoted")
}

func TestPublicDocumentBuilderWritesCanonicalThematicBreak(t *testing.T) {
	t.Parallel()

	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendHeading(1, "Title"); err != nil {
		t.Fatalf("AppendHeading() error = %v", err)
	}
	if err := builder.AppendThematicBreak(); err != nil {
		t.Fatalf("AppendThematicBreak() error = %v", err)
	}
	if err := builder.AppendParagraph("Tail."); err != nil {
		t.Fatalf("AppendParagraph() error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("# Title\n\n---\n\nTail.\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	var nilBuilder *marksplice.DocumentBuilder
	if err := nilBuilder.AppendThematicBreak(); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil DocumentBuilder.AppendThematicBreak() error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicDocumentBuilderWritesCanonicalSimpleBlockquote(t *testing.T) {
	t.Parallel()

	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendHeading(1, "Title"); err != nil {
		t.Fatalf("AppendHeading() error = %v", err)
	}
	if err := builder.AppendBlockquote("quoted *text* with Unicode π"); err != nil {
		t.Fatalf("AppendBlockquote() error = %v", err)
	}
	if err := builder.AppendParagraph("Tail."); err != nil {
		t.Fatalf("AppendParagraph() error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("# Title\n\n> quoted *text* with Unicode π\n\nTail.\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	for _, inline := range []string{"", "line one\rline two", "# heading", "---", "- item", "contains\x00nul", string([]byte{0xff})} {
		var invalid marksplice.DocumentBuilder
		if err := invalid.AppendBlockquote(inline); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendBlockquote(%q) error = %v, want ErrInvalidConstruction", inline, err)
		}
		if output, err := invalid.Markdown(); err != nil || len(output) != 0 {
			t.Fatalf("builder after rejected blockquote Markdown() = %q, %v; want empty, nil", output, err)
		}
	}

	var nilBuilder *marksplice.DocumentBuilder
	if err := nilBuilder.AppendBlockquote("text"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil DocumentBuilder.AppendBlockquote() error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicDocumentBuilderWritesCanonicalUnorderedList(t *testing.T) {
	t.Parallel()

	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendHeading(1, "Items"); err != nil {
		t.Fatalf("AppendHeading() error = %v", err)
	}
	if err := builder.AppendUnorderedList("first", "second *with emphasis*"); err != nil {
		t.Fatalf("AppendUnorderedList() error = %v", err)
	}
	if err := builder.AppendParagraph("Tail."); err != nil {
		t.Fatalf("AppendParagraph() error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("# Items\n\n- first\n- second *with emphasis*\n\nTail.\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	items := make([]marksplice.ListItem, 0, 2)
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindListItem {
			continue
		}
		item, ok := doc.ListItem(node.ID())
		if !ok {
			t.Fatalf("ListItem(%q) ok = false", node.ID())
		}
		items = append(items, item)
	}
	if len(items) != 2 {
		t.Fatalf("list item count = %d, want 2", len(items))
	}
	for index, item := range items {
		if item.Ordered() || item.Marker() != '-' || item.HasChildren() {
			t.Fatalf("item %d = ordered %v marker %q children %v, want flat unordered '-'", index, item.Ordered(), item.Marker(), item.HasChildren())
		}
		if _, ok := item.ParentID(); ok {
			t.Fatalf("item %d unexpectedly has a list-item parent", index)
		}
	}
}

func TestPublicDocumentBuilderRejectsMergedOrInvalidUnorderedLists(t *testing.T) {
	t.Parallel()

	var adjacent marksplice.DocumentBuilder
	if err := adjacent.AppendUnorderedList("one"); err != nil {
		t.Fatalf("AppendUnorderedList(first) error = %v", err)
	}
	if err := adjacent.AppendUnorderedList("two"); err != nil {
		t.Fatalf("AppendUnorderedList(second) error = %v", err)
	}
	if _, err := adjacent.Markdown(); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("Markdown(adjacent lists) error = %v, want ErrInvalidConstruction", err)
	}

	invalid := [][]string{
		nil,
		{},
		{""},
		{"line one\nline two"},
		{"contains\x00nul"},
		{string([]byte{0xff})},
		{"- nested"},
		{"---"},
	}
	for _, items := range invalid {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendUnorderedList(items...); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendUnorderedList(%q) error = %v, want ErrInvalidConstruction", items, err)
		}
		if got, err := builder.Markdown(); err != nil || len(got) != 0 {
			t.Fatalf("builder after rejected list Markdown() = %q, %v; want empty, nil", got, err)
		}
	}

	var nilBuilder *marksplice.DocumentBuilder
	if err := nilBuilder.AppendUnorderedList("item"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil DocumentBuilder.AppendUnorderedList() error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicDocumentBuilderWritesCanonicalOrderedList(t *testing.T) {
	t.Parallel()

	items := []string{"first", "second *with emphasis*", "third"}
	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendOrderedList(items...); err != nil {
		t.Fatalf("AppendOrderedList() error = %v", err)
	}
	items[0] = "mutated after append"

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("1. first\n2. second *with emphasis*\n3. third\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	count := 0
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindListItem {
			continue
		}
		item, ok := doc.ListItem(node.ID())
		if !ok {
			t.Fatalf("ListItem(%q) ok = false", node.ID())
		}
		if !item.Ordered() || item.Marker() != '.' || item.HasChildren() {
			t.Fatalf("ordered item = ordered %v marker %q children %v, want true '.' false", item.Ordered(), item.Marker(), item.HasChildren())
		}
		if _, ok := item.ParentID(); ok {
			t.Fatal("ordered top-level item unexpectedly has a list-item parent")
		}
		count++
	}
	if count != 3 {
		t.Fatalf("ordered list item count = %d, want 3", count)
	}
}

func TestPublicDocumentBuilderSeparatesDifferentListKindsAndRejectsMergedOrderedLists(t *testing.T) {
	t.Parallel()

	var mixed marksplice.DocumentBuilder
	if err := mixed.AppendUnorderedList("bullet"); err != nil {
		t.Fatalf("AppendUnorderedList() error = %v", err)
	}
	if err := mixed.AppendOrderedList("numbered"); err != nil {
		t.Fatalf("AppendOrderedList() error = %v", err)
	}
	got, err := mixed.Markdown()
	if err != nil {
		t.Fatalf("Markdown(mixed lists) error = %v", err)
	}
	if want := []byte("- bullet\n\n1. numbered\n"); !bytes.Equal(got, want) {
		t.Fatalf("Markdown(mixed lists) = %q, want %q", got, want)
	}

	var adjacent marksplice.DocumentBuilder
	if err := adjacent.AppendOrderedList("one"); err != nil {
		t.Fatalf("AppendOrderedList(first) error = %v", err)
	}
	if err := adjacent.AppendOrderedList("two"); err != nil {
		t.Fatalf("AppendOrderedList(second) error = %v", err)
	}
	if _, err := adjacent.Markdown(); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("Markdown(adjacent ordered lists) error = %v, want ErrInvalidConstruction", err)
	}

	for _, items := range [][]string{nil, {}, {""}, {"line one\nline two"}, {"- nested"}} {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendOrderedList(items...); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendOrderedList(%q) error = %v, want ErrInvalidConstruction", items, err)
		}
	}
}

func TestPublicDocumentBuilderWritesCanonicalUnorderedTaskList(t *testing.T) {
	t.Parallel()

	items := []marksplice.TaskListItem{
		{InlineGFM: "todo *first*"},
		{InlineGFM: "done [link](https://example.com)", Checked: true},
	}
	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendUnorderedTaskList(items...); err != nil {
		t.Fatalf("AppendUnorderedTaskList() error = %v", err)
	}
	items[0].InlineGFM = "mutated after append"
	items[1].Checked = false

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("- [ ] todo *first*\n- [x] done [link](https://example.com)\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	listCount := 0
	taskStates := make([]bool, 0, 2)
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindListItem:
			item, ok := doc.ListItem(node.ID())
			if !ok || item.Ordered() || item.Marker() != '-' || item.HasChildren() {
				t.Fatalf("generated task-list item = %+v, %v; want flat unordered '-'", item, ok)
			}
			listCount++
		case marksplice.KindTask:
			task, ok := doc.Task(node.ID())
			if !ok {
				t.Fatalf("Task(%q) ok = false", node.ID())
			}
			taskStates = append(taskStates, task.Checked())
		}
	}
	if listCount != 2 || len(taskStates) != 2 || taskStates[0] || !taskStates[1] {
		t.Fatalf("generated task-list proof = list items %d task states %v, want 2/[false true]", listCount, taskStates)
	}
}

func TestPublicDocumentBuilderWritesCanonicalOrderedTaskList(t *testing.T) {
	t.Parallel()

	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendOrderedTaskList(
		marksplice.TaskListItem{InlineGFM: "first", Checked: true},
		marksplice.TaskListItem{InlineGFM: "second"},
	); err != nil {
		t.Fatalf("AppendOrderedTaskList() error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("1. [x] first\n2. [ ] second\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	listCount := 0
	taskCount := 0
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindListItem:
			item, ok := doc.ListItem(node.ID())
			if !ok || !item.Ordered() || item.Marker() != '.' || item.HasChildren() {
				t.Fatalf("generated ordered task-list item = %+v, %v; want flat ordered '.'", item, ok)
			}
			listCount++
		case marksplice.KindTask:
			taskCount++
		}
	}
	if listCount != 2 || taskCount != 2 {
		t.Fatalf("generated ordered task list = %d items/%d tasks, want 2/2", listCount, taskCount)
	}
}

func TestPublicDocumentBuilderSeparatesDifferentTaskListKinds(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendUnorderedTaskList(marksplice.TaskListItem{InlineGFM: "bullet"}); err != nil {
		t.Fatalf("AppendUnorderedTaskList() error = %v", err)
	}
	if err := builder.AppendOrderedTaskList(marksplice.TaskListItem{InlineGFM: "numbered", Checked: true}); err != nil {
		t.Fatalf("AppendOrderedTaskList() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	if want := []byte("- [ ] bullet\n\n1. [x] numbered\n"); !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicDocumentBuilderRejectsInvalidOrMergedTaskLists(t *testing.T) {
	t.Parallel()

	invalid := [][]marksplice.TaskListItem{
		nil,
		{},
		{{InlineGFM: ""}},
		{{InlineGFM: "line one\nline two"}},
		{{InlineGFM: "contains\x00nul"}},
		{{InlineGFM: string([]byte{0xff})}},
	}
	for _, items := range invalid {
		var unordered marksplice.DocumentBuilder
		if err := unordered.AppendUnorderedTaskList(items...); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendUnorderedTaskList(%v) error = %v, want ErrInvalidConstruction", items, err)
		}
		var ordered marksplice.DocumentBuilder
		if err := ordered.AppendOrderedTaskList(items...); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendOrderedTaskList(%v) error = %v, want ErrInvalidConstruction", items, err)
		}
	}

	var adjacent marksplice.DocumentBuilder
	if err := adjacent.AppendUnorderedTaskList(marksplice.TaskListItem{InlineGFM: "one"}); err != nil {
		t.Fatalf("AppendUnorderedTaskList(first) error = %v", err)
	}
	if err := adjacent.AppendUnorderedTaskList(marksplice.TaskListItem{InlineGFM: "two", Checked: true}); err != nil {
		t.Fatalf("AppendUnorderedTaskList(second) error = %v", err)
	}
	if _, err := adjacent.Markdown(); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("Markdown(adjacent unordered task lists) error = %v, want ErrInvalidConstruction", err)
	}

	var adjacentOrdered marksplice.DocumentBuilder
	if err := adjacentOrdered.AppendOrderedTaskList(marksplice.TaskListItem{InlineGFM: "one"}); err != nil {
		t.Fatalf("AppendOrderedTaskList(first) error = %v", err)
	}
	if err := adjacentOrdered.AppendOrderedTaskList(marksplice.TaskListItem{InlineGFM: "two", Checked: true}); err != nil {
		t.Fatalf("AppendOrderedTaskList(second) error = %v", err)
	}
	if _, err := adjacentOrdered.Markdown(); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("Markdown(adjacent ordered task lists) error = %v, want ErrInvalidConstruction", err)
	}

	var nilBuilder *marksplice.DocumentBuilder
	if err := nilBuilder.AppendUnorderedTaskList(marksplice.TaskListItem{InlineGFM: "item"}); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil DocumentBuilder.AppendUnorderedTaskList() error = %v, want ErrInvalidConstruction", err)
	}
	if err := nilBuilder.AppendOrderedTaskList(marksplice.TaskListItem{InlineGFM: "item"}); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil DocumentBuilder.AppendOrderedTaskList() error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicDocumentBuilderTaskProofKeepsGenericListBehavior(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendUnorderedList("[ ] caller-owned task syntax"); err != nil {
		t.Fatalf("AppendUnorderedList(task-shaped content) error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown(task-shaped generic list) error = %v", err)
	}
	if want := []byte("- [ ] caller-owned task syntax\n"); !bytes.Equal(got, want) {
		t.Fatalf("Markdown(task-shaped generic list) = %q, want %q", got, want)
	}
}

func TestPublicDocumentBuilderWritesCanonicalFencedCode(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendFencedCode("fmt.Println(`hello`)", "go meta"); err != nil {
		t.Fatalf("AppendFencedCode() error = %v", err)
	}
	if err := builder.AppendFencedCode("````", ""); err != nil {
		t.Fatalf("AppendFencedCode(long fence content) error = %v", err)
	}
	if err := builder.AppendFencedCode("line one\n  ````\nline three", "text"); err != nil {
		t.Fatalf("AppendFencedCode(multiline) error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("```go meta\nfmt.Println(`hello`)\n```\n\n`````\n````\n`````\n\n`````text\nline one\n  ````\nline three\n`````\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	contents := []string{"fmt.Println(`hello`)", "````", "line one\n  ````\nline three"}
	found := 0
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindFencedCode {
			continue
		}
		block, ok := doc.FencedCode(node.ID())
		if !ok {
			t.Fatalf("FencedCode(%q) ok = false", node.ID())
		}
		if found >= len(contents) || string(got[block.Range().Start:block.Range().End]) != contents[found] {
			t.Fatalf("generated fenced-code content %d = %q", found, got[block.Range().Start:block.Range().End])
		}
		found++
	}
	if found != len(contents) {
		t.Fatalf("generated fenced-code count = %d, want %d", found, len(contents))
	}
}

func TestPublicDocumentBuilderRejectsInvalidFencedCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		content string
		info    string
	}{
		{content: ""},
		{content: "line one\r\nline two"},
		{content: "line one\rline two"},
		{content: "contains\x00nul"},
		{content: string([]byte{0xff})},
		{content: "code", info: "go\nmeta"},
		{content: "code", info: "go`meta"},
		{content: "code", info: "contains\x00nul"},
		{content: "code", info: string([]byte{0xff})},
	}
	for _, tt := range tests {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendFencedCode(tt.content, tt.info); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendFencedCode(%q, %q) error = %v, want ErrInvalidConstruction", tt.content, tt.info, err)
		}
		if got, err := builder.Markdown(); err != nil || len(got) != 0 {
			t.Fatalf("builder after rejected fenced code Markdown() = %q, %v; want empty, nil", got, err)
		}
	}

	var nilBuilder *marksplice.DocumentBuilder
	if err := nilBuilder.AppendFencedCode("code", "go"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil DocumentBuilder.AppendFencedCode() error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicDocumentBuilderWritesCanonicalReferenceDefinition(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendReferenceDefinition("docs", "https://example.com/path"); err != nil {
		t.Fatalf("AppendReferenceDefinition() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("[docs]: <https://example.com/path>\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	found := 0
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindReferenceDefinition {
			continue
		}
		definition, ok := doc.ReferenceDefinition(node.ID())
		if !ok {
			t.Fatalf("ReferenceDefinition(%q) ok = false", node.ID())
		}
		if destination := string(got[definition.Range().Start:definition.Range().End]); destination != "https://example.com/path" {
			t.Fatalf("generated reference destination = %q", destination)
		}
		found++
	}
	if found != 1 {
		t.Fatalf("generated reference-definition count = %d, want 1", found)
	}
}

func TestPublicDocumentBuilderWritesCanonicalTitledReferenceDefinition(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendReferenceDefinitionWithTitle("docs", "https://example.com/path", "Documentation π"); err != nil {
		t.Fatalf("AppendReferenceDefinitionWithTitle() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("[docs]: <https://example.com/path> \"Documentation π\"\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindReferenceDefinition {
			continue
		}
		definition, ok := doc.ReferenceDefinition(node.ID())
		if !ok {
			t.Fatalf("ReferenceDefinition(%q) ok = false", node.ID())
		}
		change, err := doc.PrepareReplaceReferenceDefinitionDestination(definition.ID(), []byte("https://new.example/path"))
		if err != nil {
			t.Fatalf("PrepareReplaceReferenceDefinitionDestination() error = %v", err)
		}
		updated, err := change.Apply(got)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
		wantUpdated := []byte("[docs]: <https://new.example/path> \"Documentation π\"\n")
		if !bytes.Equal(updated, wantUpdated) {
			t.Fatalf("updated Markdown = %q, want %q", updated, wantUpdated)
		}
		return
	}
	t.Fatal("generated titled reference definition was not promoted")
}

func TestPublicDocumentBuilderRejectsInvalidReferenceDefinitionTitle(t *testing.T) {
	t.Parallel()

	for _, title := range []string{"", "line\nbreak", "line\rbreak", "quote\"inside", "back\\slash", "contains\x00nul", string([]byte{0xff})} {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendReferenceDefinitionWithTitle("docs", "https://example.com", title); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendReferenceDefinitionWithTitle(title=%q) error = %v, want ErrInvalidConstruction", title, err)
		}
		if got, err := builder.Markdown(); err != nil || len(got) != 0 {
			t.Fatalf("builder after rejected titled definition Markdown() = %q, %v; want empty, nil", got, err)
		}
	}

	var nilBuilder *marksplice.DocumentBuilder
	if err := nilBuilder.AppendReferenceDefinitionWithTitle("docs", "https://example.com", "title"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil AppendReferenceDefinitionWithTitle() error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicDocumentBuilderRejectsInvalidReferenceDefinition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		label       string
		destination string
	}{
		{label: "", destination: "https://example.com"},
		{label: "docs", destination: ""},
		{label: "line\nbreak", destination: "https://example.com"},
		{label: "docs", destination: "line\nbreak"},
		{label: "docs]", destination: "https://example.com"},
		{label: "docs", destination: "https://example.com/a>b"},
		{label: "docs", destination: "contains\x00nul"},
		{label: string([]byte{0xff}), destination: "https://example.com"},
	}
	for _, tt := range tests {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendReferenceDefinition(tt.label, tt.destination); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendReferenceDefinition(%q, %q) error = %v, want ErrInvalidConstruction", tt.label, tt.destination, err)
		}
		if got, err := builder.Markdown(); err != nil || len(got) != 0 {
			t.Fatalf("builder after rejected reference definition Markdown() = %q, %v; want empty, nil", got, err)
		}
	}

	var nilBuilder *marksplice.DocumentBuilder
	if err := nilBuilder.AppendReferenceDefinition("docs", "https://example.com"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil DocumentBuilder.AppendReferenceDefinition() error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicDocumentBuilderWritesCanonicalTable(t *testing.T) {
	t.Parallel()

	header := []string{"Name", "Value"}
	rows := [][]string{
		{"alpha", "*one*"},
		{"", "two"},
	}
	var builder marksplice.DocumentBuilder
	if err := builder.AppendTable(header, rows...); err != nil {
		t.Fatalf("AppendTable() error = %v", err)
	}
	header[0] = "mutated"
	rows[0][0] = "mutated"

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("| Name | Value |\n| --- | --- |\n| alpha | *one* |\n|  | two |\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	rowsFound := make([]marksplice.TableRow, 0, 2)
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindTableRow {
			continue
		}
		row, ok := doc.TableRow(node.ID())
		if !ok {
			t.Fatalf("TableRow(%q) ok = false", node.ID())
		}
		rowsFound = append(rowsFound, row)
	}
	if len(rowsFound) != 2 {
		t.Fatalf("generated table-row count = %d, want 2", len(rowsFound))
	}
	for index, row := range rowsFound {
		if row.ColumnCount() != 2 {
			t.Fatalf("generated table row %d column count = %d, want 2", index, row.ColumnCount())
		}
	}
	headerIDs, ok := doc.TableRowHeaderCellIDs(rowsFound[0].ID())
	if !ok || len(headerIDs) != 2 {
		t.Fatalf("TableRowHeaderCellIDs() = %v, %v; want 2 header cells", headerIDs, ok)
	}
}

func TestPublicDocumentBuilderWritesCanonicalAlignedTable(t *testing.T) {
	t.Parallel()

	header := []string{"Left", "Right", "Center", "Default"}
	alignments := []marksplice.TableAlignment{
		marksplice.TableAlignmentLeft,
		marksplice.TableAlignmentRight,
		marksplice.TableAlignmentCenter,
		marksplice.TableAlignmentDefault,
	}
	rows := [][]string{{"l", "r", "c", "d"}}
	var builder marksplice.DocumentBuilder
	if err := builder.AppendTableWithAlignments(header, alignments, rows...); err != nil {
		t.Fatalf("AppendTableWithAlignments() error = %v", err)
	}
	header[0] = "mutated"
	alignments[0] = marksplice.TableAlignmentDefault
	rows[0][0] = "mutated"

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("| Left | Right | Center | Default |\n| :--- | ---: | :---: | --- |\n| l | r | c | d |\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	rowsFound := 0
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindTableRow {
			continue
		}
		row, ok := doc.TableRow(node.ID())
		if !ok || row.ColumnCount() != len(alignments) {
			t.Fatalf("generated aligned row = %+v, %v; want %d columns", row, ok, len(alignments))
		}
		rowsFound++
	}
	if rowsFound != 1 {
		t.Fatalf("generated aligned table-row count = %d, want 1", rowsFound)
	}
}

func TestPublicDocumentBuilderRejectsInvalidTableAlignments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		alignments []marksplice.TableAlignment
	}{
		{alignments: nil},
		{alignments: []marksplice.TableAlignment{}},
		{alignments: []marksplice.TableAlignment{marksplice.TableAlignmentLeft}},
		{alignments: []marksplice.TableAlignment{marksplice.TableAlignmentLeft, marksplice.TableAlignment(255)}},
	}
	for _, tt := range tests {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendTableWithAlignments([]string{"A", "B"}, tt.alignments, []string{"one", "two"}); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendTableWithAlignments(%v) error = %v, want ErrInvalidConstruction", tt.alignments, err)
		}
		if got, err := builder.Markdown(); err != nil || len(got) != 0 {
			t.Fatalf("builder after rejected aligned table Markdown() = %q, %v; want empty, nil", got, err)
		}
	}

	var nilBuilder *marksplice.DocumentBuilder
	if err := nilBuilder.AppendTableWithAlignments([]string{"A"}, []marksplice.TableAlignment{marksplice.TableAlignmentDefault}, []string{"one"}); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil AppendTableWithAlignments() error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicDocumentBuilderKeepsAdjacentTablesDistinct(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendTable([]string{"A"}, []string{"one"}); err != nil {
		t.Fatalf("AppendTable(first) error = %v", err)
	}
	if err := builder.AppendTable([]string{"B"}, []string{"two"}); err != nil {
		t.Fatalf("AppendTable(second) error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("| A |\n| --- |\n| one |\n\n| B |\n| --- |\n| two |\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicDocumentBuilderRejectsInvalidTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		header []string
		rows   [][]string
	}{
		{header: nil, rows: [][]string{{"one"}}},
		{header: []string{"A"}},
		{header: []string{"A", "B"}, rows: [][]string{{"one"}}},
		{header: []string{"A\nB"}, rows: [][]string{{"one"}}},
		{header: []string{"A|B"}, rows: [][]string{{"one"}}},
		{header: []string{"A"}, rows: [][]string{{"line one\nline two"}}},
		{header: []string{"A"}, rows: [][]string{{"contains\x00nul"}}},
		{header: []string{"A"}, rows: [][]string{{string([]byte{0xff})}}},
	}
	for _, tt := range tests {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendTable(tt.header, tt.rows...); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendTable(%v, %v) error = %v, want ErrInvalidConstruction", tt.header, tt.rows, err)
		}
		if got, err := builder.Markdown(); err != nil || len(got) != 0 {
			t.Fatalf("builder after rejected table Markdown() = %q, %v; want empty, nil", got, err)
		}
	}

	var nilBuilder *marksplice.DocumentBuilder
	if err := nilBuilder.AppendTable([]string{"A"}, []string{"one"}); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil DocumentBuilder.AppendTable() error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicDocumentBuilderWritesCanonicalNestedUnorderedList(t *testing.T) {
	t.Parallel()

	items := []marksplice.ListItemInput{
		{InlineGFM: "root", Depth: 0},
		{InlineGFM: "child one", Depth: 1},
		{InlineGFM: "grandchild", Depth: 2},
		{InlineGFM: "child two", Depth: 1},
		{InlineGFM: "tail", Depth: 0},
	}
	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendNestedUnorderedList(items...); err != nil {
		t.Fatalf("AppendNestedUnorderedList() error = %v", err)
	}
	items[0].InlineGFM = "mutated after append"

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("- root\n  - child one\n    - grandchild\n  - child two\n- tail\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	root := publicListItemByContent(t, doc, got, "root")
	childOne := publicListItemByContent(t, doc, got, "child one")
	grandchild := publicListItemByContent(t, doc, got, "grandchild")
	childTwo := publicListItemByContent(t, doc, got, "child two")
	tail := publicListItemByContent(t, doc, got, "tail")

	rootChildren := root.ChildIDs()
	if len(rootChildren) != 2 || rootChildren[0] != childOne.ID() || rootChildren[1] != childTwo.ID() {
		t.Fatalf("root ChildIDs() = %v, want [%v %v]", rootChildren, childOne.ID(), childTwo.ID())
	}
	if childChildren := childOne.ChildIDs(); len(childChildren) != 1 || childChildren[0] != grandchild.ID() {
		t.Fatalf("child one ChildIDs() = %v, want [%v]", childChildren, grandchild.ID())
	}
	if parent, ok := childOne.ParentID(); !ok || parent != root.ID() {
		t.Fatalf("child one ParentID() = (%v,%v), want (%v,true)", parent, ok, root.ID())
	}
	if parent, ok := grandchild.ParentID(); !ok || parent != childOne.ID() {
		t.Fatalf("grandchild ParentID() = (%v,%v), want (%v,true)", parent, ok, childOne.ID())
	}
	if parent, ok := childTwo.ParentID(); !ok || parent != root.ID() {
		t.Fatalf("child two ParentID() = (%v,%v), want (%v,true)", parent, ok, root.ID())
	}
	if _, ok := tail.ParentID(); ok {
		t.Fatal("tail unexpectedly has a parent")
	}
	rootSubtree, ok := root.SubtreeRange()
	if !ok {
		t.Fatal("root SubtreeRange() ok = false")
	}
	rootSource, ok := doc.SourceRange(rootSubtree)
	if !ok || !bytes.Equal(rootSource, []byte("- root\n  - child one\n    - grandchild\n  - child two\n")) {
		t.Fatalf("root subtree source = %q, %v", rootSource, ok)
	}
}

func TestPublicDocumentBuilderWritesCanonicalNestedOrderedListAcrossMarkerWidth(t *testing.T) {
	t.Parallel()

	items := make([]marksplice.ListItemInput, 0, 12)
	for index := 1; index <= 10; index++ {
		items = append(items, marksplice.ListItemInput{InlineGFM: fmt.Sprintf("item %d", index), Depth: 0})
	}
	items = append(items,
		marksplice.ListItemInput{InlineGFM: "child of ten", Depth: 1},
		marksplice.ListItemInput{InlineGFM: "item 11", Depth: 0},
	)

	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendNestedOrderedList(items...); err != nil {
		t.Fatalf("AppendNestedOrderedList() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("1. item 1\n2. item 2\n3. item 3\n4. item 4\n5. item 5\n6. item 6\n7. item 7\n8. item 8\n9. item 9\n10. item 10\n    1. child of ten\n11. item 11\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	itemTen := publicListItemByContent(t, doc, got, "item 10")
	child := publicListItemByContent(t, doc, got, "child of ten")
	itemEleven := publicListItemByContent(t, doc, got, "item 11")
	if parent, ok := child.ParentID(); !ok || parent != itemTen.ID() {
		t.Fatalf("child ParentID() = (%v,%v), want (%v,true)", parent, ok, itemTen.ID())
	}
	if children := itemTen.ChildIDs(); len(children) != 1 || children[0] != child.ID() {
		t.Fatalf("item 10 ChildIDs() = %v, want [%v]", children, child.ID())
	}
	if _, ok := itemEleven.ParentID(); ok {
		t.Fatal("item 11 unexpectedly has a parent")
	}
}

func TestPublicDocumentBuilderWritesCanonicalNestedTaskLists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ordered bool
		want    []byte
	}{
		{
			name:    "unordered",
			ordered: false,
			want:    []byte("- [ ] parent\n  - [x] child\n  - [ ] sibling\n- [x] tail\n"),
		},
		{
			name:    "ordered",
			ordered: true,
			want:    []byte("1. [ ] parent\n   1. [x] child\n   2. [ ] sibling\n2. [x] tail\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			items := []marksplice.TaskListItemInput{
				{InlineGFM: "parent", Checked: false, Depth: 0},
				{InlineGFM: "child", Checked: true, Depth: 1},
				{InlineGFM: "sibling", Checked: false, Depth: 1},
				{InlineGFM: "tail", Checked: true, Depth: 0},
			}
			builder := marksplice.NewDocumentBuilder()
			var err error
			if tt.ordered {
				err = builder.AppendNestedOrderedTaskList(items...)
			} else {
				err = builder.AppendNestedUnorderedTaskList(items...)
			}
			if err != nil {
				t.Fatalf("append nested task list error = %v", err)
			}
			items[0].InlineGFM = "mutated after append"

			got, err := builder.Markdown()
			if err != nil {
				t.Fatalf("Markdown() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("Markdown() = %q, want %q", got, tt.want)
			}
			doc, err := marksplice.Parse(got)
			if err != nil {
				t.Fatalf("Parse(generated) error = %v", err)
			}
			parent := publicListItemByContent(t, doc, got, "[ ] parent")
			child := publicListItemByContent(t, doc, got, "[x] child")
			sibling := publicListItemByContent(t, doc, got, "[ ] sibling")
			if parentID, ok := child.ParentID(); !ok || parentID != parent.ID() {
				t.Fatalf("child ParentID() = (%v,%v), want (%v,true)", parentID, ok, parent.ID())
			}
			if parentID, ok := sibling.ParentID(); !ok || parentID != parent.ID() {
				t.Fatalf("sibling ParentID() = (%v,%v), want (%v,true)", parentID, ok, parent.ID())
			}
			children := parent.ChildIDs()
			if len(children) != 2 || children[0] != child.ID() || children[1] != sibling.ID() {
				t.Fatalf("parent ChildIDs() = %v, want [%v %v]", children, child.ID(), sibling.ID())
			}
		})
	}
}

func TestPublicDocumentBuilderRejectsInvalidNestedTaskListInput(t *testing.T) {
	t.Parallel()

	invalid := [][]marksplice.TaskListItemInput{
		nil,
		{{InlineGFM: "item", Depth: 1}},
		{{InlineGFM: "root", Depth: 0}, {InlineGFM: "grandchild", Depth: 2}},
		{{InlineGFM: "line one\nline two", Depth: 0}},
	}
	for _, items := range invalid {
		for _, appendList := range []func(*marksplice.DocumentBuilder) error{
			func(builder *marksplice.DocumentBuilder) error {
				return builder.AppendNestedUnorderedTaskList(items...)
			},
			func(builder *marksplice.DocumentBuilder) error { return builder.AppendNestedOrderedTaskList(items...) },
		} {
			var builder marksplice.DocumentBuilder
			if err := appendList(&builder); !errors.Is(err, marksplice.ErrInvalidConstruction) {
				t.Fatalf("nested task append error = %v, want ErrInvalidConstruction", err)
			}
			if got, err := builder.Markdown(); err != nil || len(got) != 0 {
				t.Fatalf("builder after rejected nested task list Markdown() = %q, %v; want empty, nil", got, err)
			}
		}
	}

	var nilBuilder *marksplice.DocumentBuilder
	item := marksplice.TaskListItemInput{InlineGFM: "item"}
	if err := nilBuilder.AppendNestedUnorderedTaskList(item); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil AppendNestedUnorderedTaskList() error = %v, want ErrInvalidConstruction", err)
	}
	if err := nilBuilder.AppendNestedOrderedTaskList(item); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil AppendNestedOrderedTaskList() error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicDocumentBuilderRejectsInvalidNestedListDepth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		items []marksplice.ListItemInput
	}{
		{name: "empty", items: nil},
		{name: "negative", items: []marksplice.ListItemInput{{InlineGFM: "item", Depth: -1}}},
		{name: "first nested", items: []marksplice.ListItemInput{{InlineGFM: "item", Depth: 1}}},
		{name: "depth jump", items: []marksplice.ListItemInput{{InlineGFM: "root", Depth: 0}, {InlineGFM: "grandchild", Depth: 2}}},
		{name: "invalid inline", items: []marksplice.ListItemInput{{InlineGFM: "line one\nline two", Depth: 0}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, appendList := range []func(*marksplice.DocumentBuilder) error{
				func(builder *marksplice.DocumentBuilder) error { return builder.AppendNestedUnorderedList(tt.items...) },
				func(builder *marksplice.DocumentBuilder) error { return builder.AppendNestedOrderedList(tt.items...) },
			} {
				var builder marksplice.DocumentBuilder
				if err := appendList(&builder); !errors.Is(err, marksplice.ErrInvalidConstruction) {
					t.Fatalf("nested append error = %v, want ErrInvalidConstruction", err)
				}
				if got, err := builder.Markdown(); err != nil || len(got) != 0 {
					t.Fatalf("builder after rejected nested list Markdown() = %q, %v; want empty, nil", got, err)
				}
			}
		})
	}

	var nilBuilder *marksplice.DocumentBuilder
	if err := nilBuilder.AppendNestedUnorderedList(marksplice.ListItemInput{InlineGFM: "item"}); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil AppendNestedUnorderedList() error = %v, want ErrInvalidConstruction", err)
	}
	if err := nilBuilder.AppendNestedOrderedList(marksplice.ListItemInput{InlineGFM: "item"}); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil AppendNestedOrderedList() error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicDocumentBuilderRejectsInvalidConstruction(t *testing.T) {
	t.Parallel()

	for _, level := range []int{0, 7} {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendHeading(level, "title"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendHeading(level=%d) error = %v, want ErrInvalidConstruction", level, err)
		}
	}

	invalidParagraph := []string{
		"",
		"line one\r\nline two",
		"line one\rline two",
		"contains\x00nul",
		string([]byte{0xff}),
	}
	for _, content := range invalidParagraph {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendParagraph(content); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendParagraph(%q) error = %v, want ErrInvalidConstruction", content, err)
		}
	}

	invalidHeading := append([]string{"line one\nline two"}, invalidParagraph...)
	for _, inline := range invalidHeading {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendHeading(1, inline); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendHeading(%q) error = %v, want ErrInvalidConstruction", inline, err)
		}
	}

	for _, inline := range []string{"- item", "---", "```"} {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendParagraph(inline); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendParagraph(structural %q) error = %v, want ErrInvalidConstruction", inline, err)
		}
	}

	var builder marksplice.DocumentBuilder
	if err := builder.AppendHeading(2, "Title #"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendHeading(closing marker) error = %v, want ErrInvalidConstruction", err)
	}
	if got, err := builder.Markdown(); err != nil || len(got) != 0 {
		t.Fatalf("builder after rejected appends Markdown() = %q, %v; want empty, nil", got, err)
	}
}

func TestPublicDocumentBuilderZeroValueAndOutputCopies(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	empty, err := builder.Markdown()
	if err != nil || len(empty) != 0 {
		t.Fatalf("zero DocumentBuilder.Markdown() = %q, %v; want empty, nil", empty, err)
	}
	if err := builder.AppendHeading(1, "Title"); err != nil {
		t.Fatalf("AppendHeading() error = %v", err)
	}
	first, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown(first) error = %v", err)
	}
	first[0] = '!'
	second, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown(second) error = %v", err)
	}
	if string(second) != "# Title\n" {
		t.Fatalf("mutating returned Markdown changed builder state: %q", second)
	}

	var nilBuilder *marksplice.DocumentBuilder
	if _, err := nilBuilder.Markdown(); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil DocumentBuilder.Markdown() error = %v, want ErrInvalidConstruction", err)
	}
	if err := nilBuilder.AppendParagraph("text"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil DocumentBuilder.AppendParagraph() error = %v, want ErrInvalidConstruction", err)
	}
}
