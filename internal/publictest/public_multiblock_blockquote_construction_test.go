package publictest

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicDocumentBuilderWritesCanonicalMultiBlockBlockquote(t *testing.T) {
	t.Parallel()

	content := marksplice.NewDocumentBuilder()
	if err := content.AppendHeading(2, "Inside"); err != nil {
		t.Fatalf("content.AppendHeading() error = %v", err)
	}
	if err := content.AppendParagraph("first *line*\nsecond π"); err != nil {
		t.Fatalf("content.AppendParagraph() error = %v", err)
	}
	if err := content.AppendThematicBreak(); err != nil {
		t.Fatalf("content.AppendThematicBreak() error = %v", err)
	}
	if err := content.AppendFencedCode("fmt.Println(\"π\")", "go"); err != nil {
		t.Fatalf("content.AppendFencedCode() error = %v", err)
	}

	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendHeading(1, "Before"); err != nil {
		t.Fatalf("AppendHeading() error = %v", err)
	}
	if err := builder.AppendBlockquoteBlocks(1, content); err != nil {
		t.Fatalf("AppendBlockquoteBlocks() error = %v", err)
	}
	if err := builder.AppendParagraph("After."); err != nil {
		t.Fatalf("AppendParagraph() error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("# Before\n\n> ## Inside\n> \n> first *line*\n> second π\n> \n> ---\n> \n> ```go\n> fmt.Println(\"π\")\n> ```\n\nAfter.\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindBlockquote {
			t.Fatal("multi-block constructed blockquote unexpectedly entered the existing-source public blockquote subset")
		}
	}
}

func TestPublicDocumentBuilderWritesNestedMultiBlockBlockquote(t *testing.T) {
	t.Parallel()

	content := marksplice.NewDocumentBuilder()
	if err := content.AppendParagraphContent(marksplice.TextInline("> literal")); err != nil {
		t.Fatalf("content.AppendParagraphContent() error = %v", err)
	}
	if err := content.AppendHeadingContent(3, marksplice.TextInline("Typed *heading*")); err != nil {
		t.Fatalf("content.AppendHeadingContent() error = %v", err)
	}

	var builder marksplice.DocumentBuilder
	if err := builder.AppendBlockquoteBlocks(3, content); err != nil {
		t.Fatalf("AppendBlockquoteBlocks() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("> > > \\> literal\n> > > \n> > > ### Typed \\*heading\\*\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicDocumentBuilderMultiBlockBlockquoteSnapshotsChildBuilder(t *testing.T) {
	t.Parallel()

	content := marksplice.NewDocumentBuilder()
	if err := content.AppendParagraph("first"); err != nil {
		t.Fatalf("content.AppendParagraph() error = %v", err)
	}

	var builder marksplice.DocumentBuilder
	if err := builder.AppendBlockquoteBlocks(1, content); err != nil {
		t.Fatalf("AppendBlockquoteBlocks() error = %v", err)
	}
	if err := content.AppendHeading(2, "later"); err != nil {
		t.Fatalf("content.AppendHeading() error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	if want := []byte("> first\n"); !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want snapshot %q", got, want)
	}
}

func TestPublicDocumentBuilderRejectsInvalidMultiBlockBlockquoteContainer(t *testing.T) {
	t.Parallel()

	valid := marksplice.NewDocumentBuilder()
	if err := valid.AppendParagraph("text"); err != nil {
		t.Fatalf("valid.AppendParagraph() error = %v", err)
	}

	for _, depth := range []int{-1, 0, 65} {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendBlockquoteBlocks(depth, valid); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendBlockquoteBlocks(depth=%d) error = %v, want ErrInvalidConstruction", depth, err)
		}
		if got, err := builder.Markdown(); err != nil || len(got) != 0 {
			t.Fatalf("builder after rejected depth %d Markdown() = %q, %v; want empty, nil", depth, got, err)
		}
	}

	var nilBuilder *marksplice.DocumentBuilder
	if err := nilBuilder.AppendBlockquoteBlocks(1, valid); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil AppendBlockquoteBlocks() error = %v, want ErrInvalidConstruction", err)
	}

	var builder marksplice.DocumentBuilder
	if err := builder.AppendBlockquoteBlocks(1, nil); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendBlockquoteBlocks(nil) error = %v, want ErrInvalidConstruction", err)
	}
	if err := builder.AppendBlockquoteBlocks(1, marksplice.NewDocumentBuilder()); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendBlockquoteBlocks(empty) error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicDocumentBuilderMultiBlockBlockquoteSupportsListChildren(t *testing.T) {
	t.Parallel()

	content := marksplice.NewDocumentBuilder()
	if err := content.AppendUnorderedList("alpha", "beta"); err != nil {
		t.Fatalf("AppendUnorderedList() error = %v", err)
	}
	if err := content.AppendOrderedList("one", "two"); err != nil {
		t.Fatalf("AppendOrderedList() error = %v", err)
	}
	if err := content.AppendUnorderedTaskList(
		marksplice.TaskListItem{InlineGFM: "todo"},
		marksplice.TaskListItem{InlineGFM: "done", Checked: true},
	); err != nil {
		t.Fatalf("AppendUnorderedTaskList() error = %v", err)
	}
	if err := content.AppendOrderedTaskList(
		marksplice.TaskListItem{InlineGFM: "first", Checked: true},
		marksplice.TaskListItem{InlineGFM: "second"},
	); err != nil {
		t.Fatalf("AppendOrderedTaskList() error = %v", err)
	}
	if err := content.AppendNestedUnorderedList(
		marksplice.ListItemInput{InlineGFM: "parent"},
		marksplice.ListItemInput{InlineGFM: "child", Depth: 1},
	); err != nil {
		t.Fatalf("AppendNestedUnorderedList() error = %v", err)
	}
	if err := content.AppendNestedOrderedTaskList(
		marksplice.TaskListItemInput{InlineGFM: "parent", Checked: true},
		marksplice.TaskListItemInput{InlineGFM: "child", Depth: 1},
	); err != nil {
		t.Fatalf("AppendNestedOrderedTaskList() error = %v", err)
	}

	var builder marksplice.DocumentBuilder
	if err := builder.AppendBlockquoteBlocks(2, content); err != nil {
		t.Fatalf("AppendBlockquoteBlocks() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte(
		"> > - alpha\n" +
			"> > - beta\n" +
			"> > \n" +
			"> > 1. one\n" +
			"> > 2. two\n" +
			"> > \n" +
			"> > - [ ] todo\n" +
			"> > - [x] done\n" +
			"> > \n" +
			"> > 1. [x] first\n" +
			"> > 2. [ ] second\n" +
			"> > \n" +
			"> > - parent\n" +
			"> >   - child\n" +
			"> > \n" +
			"> > 1. [x] parent\n" +
			"> >    1. [ ] child\n",
	)
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicDocumentBuilderMultiBlockBlockquoteSupportsReferenceAndTableChildren(t *testing.T) {
	t.Parallel()

	content := marksplice.NewDocumentBuilder()
	if err := content.AppendReferenceDefinitionWithTitle("doc", "https://example.test/a", "Title"); err != nil {
		t.Fatalf("AppendReferenceDefinitionWithTitle() error = %v", err)
	}
	if err := content.AppendTableWithAlignments(
		[]string{"A", "B"},
		[]marksplice.TableAlignment{marksplice.TableAlignmentLeft, marksplice.TableAlignmentCenter},
		[]string{"x", "y"},
		[]string{"π", "z"},
	); err != nil {
		t.Fatalf("AppendTableWithAlignments() error = %v", err)
	}

	var builder marksplice.DocumentBuilder
	if err := builder.AppendBlockquoteBlocks(2, content); err != nil {
		t.Fatalf("AppendBlockquoteBlocks() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte(
		"> > [doc]: <https://example.test/a> \"Title\"\n" +
			"> > \n" +
			"> > | A | B |\n" +
			"> > | :--- | :---: |\n" +
			"> > | x | y |\n" +
			"> > | π | z |\n",
	)
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicDocumentBuilderMultiBlockBlockquoteSupportsRecursiveChildren(t *testing.T) {
	t.Parallel()

	leaf := marksplice.NewDocumentBuilder()
	if err := leaf.AppendParagraph("deep"); err != nil {
		t.Fatalf("leaf.AppendParagraph() error = %v", err)
	}

	content := marksplice.NewDocumentBuilder()
	if err := content.AppendBlockquote("single"); err != nil {
		t.Fatalf("content.AppendBlockquote() error = %v", err)
	}
	if err := content.AppendBlockquoteBlocks(2, leaf); err != nil {
		t.Fatalf("content.AppendBlockquoteBlocks() error = %v", err)
	}

	var builder marksplice.DocumentBuilder
	if err := builder.AppendBlockquoteBlocks(3, content); err != nil {
		t.Fatalf("AppendBlockquoteBlocks() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("> > > > single\n> > > \n> > > > > deep\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicDocumentBuilderMultiBlockBlockquoteBoundsTotalRecursiveDepth(t *testing.T) {
	t.Parallel()

	leaf := marksplice.NewDocumentBuilder()
	if err := leaf.AppendParagraph("limit"); err != nil {
		t.Fatalf("leaf.AppendParagraph() error = %v", err)
	}
	inner := marksplice.NewDocumentBuilder()
	if err := inner.AppendBlockquoteBlocks(32, leaf); err != nil {
		t.Fatalf("inner.AppendBlockquoteBlocks() error = %v", err)
	}

	var accepted marksplice.DocumentBuilder
	if err := accepted.AppendBlockquoteBlocks(32, inner); err != nil {
		t.Fatalf("AppendBlockquoteBlocks(total depth 64) error = %v", err)
	}
	got, err := accepted.Markdown()
	if err != nil {
		t.Fatalf("accepted.Markdown() error = %v", err)
	}
	want := []byte(strings.Repeat("> ", 64) + "limit\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("accepted.Markdown() = %q, want depth-64 source %q", got, want)
	}

	var rejected marksplice.DocumentBuilder
	if err := rejected.AppendBlockquoteBlocks(33, inner); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendBlockquoteBlocks(total depth 65) error = %v, want ErrInvalidConstruction", err)
	}
	if got, err := rejected.Markdown(); err != nil || len(got) != 0 {
		t.Fatalf("rejected.Markdown() = %q, %v; want empty, nil", got, err)
	}
}

func TestPublicDocumentBuilderRejectsFrontMatterMultiBlockBlockquoteChild(t *testing.T) {
	t.Parallel()

	content := marksplice.NewDocumentBuilder()
	if err := content.SetYAMLFrontMatter(marksplice.FrontMatterFieldInput{Key: "title", Value: "metadata"}); err != nil {
		t.Fatalf("SetYAMLFrontMatter() error = %v", err)
	}
	if err := content.AppendParagraph("body"); err != nil {
		t.Fatalf("AppendParagraph() error = %v", err)
	}

	var builder marksplice.DocumentBuilder
	if err := builder.AppendBlockquoteBlocks(1, content); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendBlockquoteBlocks(front matter) error = %v, want ErrInvalidConstruction", err)
	}
	if got, err := builder.Markdown(); err != nil || len(got) != 0 {
		t.Fatalf("builder after rejected front matter Markdown() = %q, %v; want empty, nil", got, err)
	}
}

func TestPublicDocumentBuilderRejectsSelfMultiBlockBlockquote(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendParagraph("existing"); err != nil {
		t.Fatalf("AppendParagraph() error = %v", err)
	}
	if err := builder.AppendBlockquoteBlocks(1, &builder); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendBlockquoteBlocks(self) error = %v, want ErrInvalidConstruction", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	if want := []byte("existing\n"); !bytes.Equal(got, want) {
		t.Fatalf("Markdown() after self rejection = %q, want %q", got, want)
	}
}
