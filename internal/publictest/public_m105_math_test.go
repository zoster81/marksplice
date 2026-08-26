package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM105MathExpressionsExposeDollarBacktickBlockAndFencedForms(t *testing.T) {
	t.Parallel()

	source := []byte("inline $x+1$ and $`a*b`$\n\n$$x^2$$\n\n```math\na+b\n```\n")
	doc := mustParseM105Document(t, source)
	expressions := doc.MathExpressions()
	if len(expressions) != 4 {
		t.Fatalf("len(MathExpressions()) = %d, want 4", len(expressions))
	}
	wantStyles := []marksplice.MathExpressionStyle{
		marksplice.MathExpressionInlineDollar,
		marksplice.MathExpressionInlineBacktick,
		marksplice.MathExpressionBlockDollar,
		marksplice.MathExpressionFencedBlock,
	}
	wantSyntax := []string{"$x+1$", "$`a*b`$", "$$x^2$$\n", "```math\na+b\n```\n"}
	wantPayload := []string{"x+1", "a*b", "x^2", "a+b"}
	for index, expression := range expressions {
		if expression.Style() != wantStyles[index] {
			t.Fatalf("Style[%d] = %v, want %v", index, expression.Style(), wantStyles[index])
		}
		assertM105RangeSource(t, doc, expression.Range(), wantSyntax[index])
		payloadRanges, ok := doc.MathExpressionPayloadRanges(expression.ID())
		if !ok || len(payloadRanges) != 1 {
			t.Fatalf("MathExpressionPayloadRanges[%d] = %v/%v", index, payloadRanges, ok)
		}
		assertM105RangeSource(t, doc, payloadRanges[0], wantPayload[index])
		if payload, ok := expression.PayloadRange(); !ok || payload != payloadRanges[0] {
			t.Fatalf("PayloadRange[%d] = %v/%v, want %v", index, payload, ok, payloadRanges[0])
		}
		if got, ok := doc.MathExpression(expression.ID()); !ok || got.ID() != expression.ID() || got.Style() != expression.Style() {
			t.Fatalf("MathExpression(ID)[%d] = %+v/%v", index, got, ok)
		}
	}

	fences := doc.FencedBlocks()
	if len(fences) != 1 || fences[0].ID() != expressions[3].ID() {
		t.Fatalf("math fenced identity = %+v vs %+v", fences, expressions[3])
	}
	if node, ok := doc.Node(expressions[3].ID()); !ok || node.Kind() != marksplice.KindFencedCode {
		t.Fatalf("fenced math Node() = %+v/%v", node, ok)
	}
}

func TestM105MathRecognitionFailsClosedAroundMarkdownOwnedAndMalformedSource(t *testing.T) {
	t.Parallel()

	source := []byte("plain \\$5 and unmatched $x\n\n`$code$` and [$linked$](https://example.com)\n\n```text\n$fenced$\n```\n\n```math extra\n$also-fenced$\n```\n\nvalid $x\\$y$\n")
	doc := mustParseM105Document(t, source)
	expressions := doc.MathExpressions()
	if len(expressions) != 1 || expressions[0].Style() != marksplice.MathExpressionInlineDollar {
		t.Fatalf("MathExpressions() = %+v, want one inline-dollar expression", expressions)
	}
	payload, ok := expressions[0].PayloadRange()
	if !ok {
		t.Fatal("PayloadRange() ok = false")
	}
	assertM105RangeSource(t, doc, payload, "x\\$y")
}

func TestM105MathPayloadReplacementPreservesSourceForm(t *testing.T) {
	t.Parallel()

	source := []byte("$x$\n\n$`a*b`$\n\n$$z$$\n\n```math\nfenced\n```\n")
	doc := mustParseM105Document(t, source)
	expressions := doc.MathExpressions()
	if len(expressions) != 4 {
		t.Fatalf("len(MathExpressions()) = %d, want 4", len(expressions))
	}

	changes := make([]marksplice.ChangeSet, 0, len(expressions))
	for index, replacement := range []string{"y+1", "c*d", "q^2", "inside"} {
		change, err := doc.PrepareReplaceMathExpression(expressions[index].ID(), []byte(replacement))
		if err != nil {
			t.Fatalf("PrepareReplaceMathExpression[%d]() error = %v", index, err)
		}
		changes = append(changes, change)
	}
	combined, err := doc.ComposeChanges(changes...)
	if err != nil {
		t.Fatalf("ComposeChanges() error = %v", err)
	}
	got, err := combined.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("$y+1$\n\n$`c*d`$\n\n$$q^2$$\n\n```math\ninside\n```\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Apply() = %q, want %q", got, want)
	}

	if _, err := doc.PrepareReplaceMathExpression(expressions[0].ID(), []byte("bad$payload")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("inline unescaped dollar error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareReplaceMathExpression(expressions[1].ID(), []byte("bad`payload")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("backtick payload error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareReplaceMathExpression(expressions[2].ID(), []byte("two\nlines")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("block multiline replacement error = %v, want ErrInvalidReplacement", err)
	}
}

func TestM105MathCompositionRejectsTwoEditsInSameParagraphAggregate(t *testing.T) {
	t.Parallel()

	doc := mustParseM105Document(t, []byte("$x$ and $`y`$\n"))
	expressions := doc.MathExpressions()
	if len(expressions) != 2 {
		t.Fatalf("len(MathExpressions()) = %d, want 2", len(expressions))
	}
	first, err := doc.PrepareReplaceMathExpression(expressions[0].ID(), []byte("a"))
	if err != nil {
		t.Fatalf("PrepareReplaceMathExpression(first) error = %v", err)
	}
	second, err := doc.PrepareReplaceMathExpression(expressions[1].ID(), []byte("b"))
	if err != nil {
		t.Fatalf("PrepareReplaceMathExpression(second) error = %v", err)
	}
	if _, err := doc.ComposeChanges(first, second); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("ComposeChanges(same paragraph) error = %v, want ErrInvalidReplacement", err)
	}
}

func TestM105MathConstructionAndQueries(t *testing.T) {
	t.Parallel()

	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendParagraphContent(
		marksplice.TextInline("A "),
		marksplice.MathInline("x+1"),
		marksplice.TextInline(" B "),
		marksplice.MathBacktickInline("a*b"),
	); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}
	if err := builder.AppendMathBlock("x^2+y^2"); err != nil {
		t.Fatalf("AppendMathBlock() error = %v", err)
	}
	if err := builder.AppendFencedCode("z^2", "math"); err != nil {
		t.Fatalf("AppendFencedCode(math) error = %v", err)
	}
	markdown, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("A $x+1$ B $`a*b`$\n\n$$x^2+y^2$$\n\n```math\nz^2\n```\n")
	if !bytes.Equal(markdown, want) {
		t.Fatalf("Markdown() = %q, want %q", markdown, want)
	}

	doc := mustParseM105Document(t, markdown)
	matches, err := doc.QueryNodes(marksplice.NodeQuery{Kinds: []marksplice.Kind{marksplice.KindMathExpression}, Limit: 8})
	if err != nil {
		t.Fatalf("QueryNodes(math) error = %v", err)
	}
	if len(matches) != 3 {
		t.Fatalf("len(QueryNodes(math)) = %d, want 3 dollar/backtick nodes", len(matches))
	}
	for _, match := range matches {
		if match.Node().Kind() != marksplice.KindMathExpression {
			t.Fatalf("QueryNodes returned kind %v", match.Node().Kind())
		}
	}

	bad := marksplice.NewDocumentBuilder()
	if err := bad.AppendParagraphContent(marksplice.MathInline("bad$payload")); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("MathInline invalid error = %v, want ErrInvalidConstruction", err)
	}
	if err := bad.AppendParagraphContent(marksplice.MathBacktickInline("bad`payload")); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("MathBacktickInline invalid error = %v, want ErrInvalidConstruction", err)
	}
	if err := bad.AppendMathBlock("two\nlines"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendMathBlock multiline error = %v, want ErrInvalidConstruction", err)
	}
}

func mustParseM105Document(t *testing.T, source []byte) *marksplice.Document {
	t.Helper()
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return doc
}

func assertM105RangeSource(t *testing.T, doc *marksplice.Document, range_ marksplice.Range, want string) {
	t.Helper()
	got, ok := doc.SourceRange(range_)
	if !ok {
		t.Fatalf("SourceRange(%v) ok = false", range_)
	}
	if !bytes.Equal(got, []byte(want)) {
		t.Fatalf("SourceRange(%v) = %q, want %q", range_, got, want)
	}
}
