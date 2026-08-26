package source

import (
	"errors"
	"testing"
)

func TestMapMathExpressionProvesReviewedFormsAndCRLFOwnership(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  string
		style   MathExpressionStyle
		syntax  Range
		payload Range
		owned   Range
	}{
		{name: "inline dollar", source: `$x\$y$`, style: MathExpressionInlineDollar, syntax: Range{0, 6}, payload: Range{1, 5}, owned: Range{0, 6}},
		{name: "inline backtick", source: "$`a*b`$", style: MathExpressionInlineBacktick, syntax: Range{0, 7}, payload: Range{2, 5}, owned: Range{0, 7}},
		{name: "block CRLF", source: "$$x^2$$\r\nnext", style: MathExpressionBlockDollar, syntax: Range{0, 7}, payload: Range{2, 5}, owned: Range{0, 9}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mapping, err := MapMathExpression([]byte(tt.source), tt.style, tt.syntax, tt.payload)
			if err != nil {
				t.Fatalf("MapMathExpression() error = %v", err)
			}
			if mapping.Style != tt.style || mapping.Range != tt.owned || mapping.PayloadRange != tt.payload {
				t.Fatalf("mapping = %+v, want style=%v range=%v payload=%v", mapping, tt.style, tt.owned, tt.payload)
			}
		})
	}
}

func TestMapMathExpressionRejectsAmbiguousOrMultilineShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		source  string
		style   MathExpressionStyle
		syntax  Range
		payload Range
	}{
		{source: `$$x$`, style: MathExpressionInlineDollar, syntax: Range{0, 4}, payload: Range{1, 3}},
		{source: `$x$y$`, style: MathExpressionInlineDollar, syntax: Range{0, 5}, payload: Range{1, 4}},
		{source: "$`a`b`$", style: MathExpressionInlineBacktick, syntax: Range{0, 7}, payload: Range{2, 5}},
		{source: "$$x\ny$$", style: MathExpressionBlockDollar, syntax: Range{0, 7}, payload: Range{2, 5}},
	}
	for _, tt := range tests {
		if _, err := MapMathExpression([]byte(tt.source), tt.style, tt.syntax, tt.payload); !errors.Is(err, ErrUnsupportedMathExpressionShape) {
			t.Fatalf("MapMathExpression(%q) error = %v, want ErrUnsupportedMathExpressionShape", tt.source, err)
		}
	}
}
