package goldmark

import (
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestMathExpressionObservationsRecognizeReviewedFormsAndExcludeOwnedMarkdown(t *testing.T) {
	t.Parallel()

	source := []byte("plain $x+1$ and $`a*b`$\n\n$$x^2$$\r\n\n`$code$` [link $x$](https://example.com)\n\n> $$quoted$$\n\n```text\n$fenced$\n```\n")
	observed, err := New().ParseDocument(source)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}
	if len(observed.MathExpressions) != 3 {
		t.Fatalf("len(MathExpressions) = %d, want 3: %+v", len(observed.MathExpressions), observed.MathExpressions)
	}
	want := []parser.MathExpressionStyle{
		parser.MathExpressionInlineDollar,
		parser.MathExpressionInlineBacktick,
		parser.MathExpressionBlockDollar,
	}
	for index, expression := range observed.MathExpressions {
		if expression.Style != want[index] {
			t.Fatalf("MathExpressions[%d].Style = %v, want %v", index, expression.Style, want[index])
		}
	}
	if !observed.MathExpressions[2].TopLevel {
		t.Fatal("block-dollar expression TopLevel = false")
	}
}

func TestMathExpressionObservationsSuppressExactGFMConflictsOnly(t *testing.T) {
	t.Parallel()

	observed, err := New().ParseDocument([]byte("$`a*b`$\n\n$$x^2$$\n\n`ordinary`\n\nparagraph\n"))
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}
	codeSpans := 0
	paragraphs := 0
	for _, node := range observed.Nodes {
		switch node.Kind {
		case parser.KindCodeSpan:
			codeSpans++
		case parser.KindParagraph:
			paragraphs++
		}
	}
	if codeSpans != 1 {
		t.Fatalf("ordinary code-span count = %d, want 1", codeSpans)
	}
	if paragraphs != 3 {
		t.Fatalf("ordinary paragraph count = %d, want 3", paragraphs)
	}
}
