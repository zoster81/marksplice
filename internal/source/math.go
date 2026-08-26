package source

import (
	"bytes"
	"fmt"
)

// MathExpressionStyle identifies one reviewed GitHub-compatible source delimiter form.
type MathExpressionStyle uint8

const (
	MathExpressionUnknown MathExpressionStyle = iota
	MathExpressionInlineDollar
	MathExpressionInlineBacktick
	MathExpressionBlockDollar
)

// MathExpressionMapping binds one math expression to exact delimiter/container and payload source.
type MathExpressionMapping struct {
	Style        MathExpressionStyle
	Range        Range
	PayloadRange Range
}

// MapMathExpression independently proves one conservative mathematical source form.
// Block-dollar ownership includes the physical line terminator when one is present.
func MapMathExpression(input []byte, style MathExpressionStyle, syntax, payload Range) (MathExpressionMapping, error) {
	if !syntax.Valid(len(input)) || !payload.Valid(len(input)) || syntax.Start >= syntax.End || payload.Start >= payload.End ||
		payload.Start < syntax.Start || payload.End > syntax.End || containsLineBreak(input[payload.Start:payload.End]) {
		return MathExpressionMapping{}, fmt.Errorf("%w: invalid syntax or payload range", ErrUnsupportedMathExpressionShape)
	}
	var err error
	switch style {
	case MathExpressionInlineDollar:
		err = validateInlineDollarMath(input, syntax, payload)
	case MathExpressionInlineBacktick:
		err = validateInlineBacktickMath(input, syntax, payload)
	case MathExpressionBlockDollar:
		err = validateBlockDollarMath(input, syntax, payload)
	default:
		err = fmt.Errorf("%w: unsupported math style", ErrUnsupportedMathExpressionShape)
	}
	if err != nil {
		return MathExpressionMapping{}, err
	}
	owned := syntax
	if style == MathExpressionBlockDollar {
		owned.End = physicalLineRangeEnd(input, syntax.End)
	}
	return MathExpressionMapping{Style: style, Range: owned, PayloadRange: payload}, nil
}

func validateInlineDollarMath(input []byte, syntax, payload Range) error {
	if syntax.End-syntax.Start < 3 || payload.Start != syntax.Start+1 || payload.End != syntax.End-1 ||
		input[syntax.Start] != '$' || input[syntax.End-1] != '$' {
		return fmt.Errorf("%w: invalid inline-dollar delimiters", ErrUnsupportedMathExpressionShape)
	}
	if mathDollarEscaped(input, syntax.Start) || mathDollarEscaped(input, syntax.End-1) ||
		mathAdjacentDollar(input, syntax.Start) || mathAdjacentDollar(input, syntax.End-1) || mathContainsUnescapedDollar(input, payload) {
		return fmt.Errorf("%w: ambiguous inline-dollar delimiters", ErrUnsupportedMathExpressionShape)
	}
	return nil
}

func validateInlineBacktickMath(input []byte, syntax, payload Range) error {
	if syntax.End-syntax.Start < 5 || payload.Start != syntax.Start+2 || payload.End != syntax.End-2 ||
		input[syntax.Start] != '$' || input[syntax.Start+1] != '`' || input[syntax.End-2] != '`' || input[syntax.End-1] != '$' {
		return fmt.Errorf("%w: invalid inline-backtick delimiters", ErrUnsupportedMathExpressionShape)
	}
	if mathDollarEscaped(input, syntax.Start) || mathDollarEscaped(input, syntax.End-1) ||
		mathAdjacentDollar(input, syntax.Start) || mathAdjacentDollar(input, syntax.End-1) ||
		bytes.IndexByte(input[payload.Start:payload.End], '`') >= 0 || mathContainsUnescapedDollar(input, payload) {
		return fmt.Errorf("%w: ambiguous inline-backtick payload", ErrUnsupportedMathExpressionShape)
	}
	return nil
}

func validateBlockDollarMath(input []byte, syntax, payload Range) error {
	if syntax.End-syntax.Start < 5 || payload.Start != syntax.Start+2 || payload.End != syntax.End-2 ||
		!bytes.Equal(input[syntax.Start:syntax.Start+2], []byte("$$")) || !bytes.Equal(input[syntax.End-2:syntax.End], []byte("$$")) {
		return fmt.Errorf("%w: invalid block-dollar delimiters", ErrUnsupportedMathExpressionShape)
	}
	if physicalLineStart(input, syntax.Start) != syntax.Start || physicalLineEnd(input, syntax.End) != syntax.End ||
		mathContainsUnescapedDollar(input, payload) {
		return fmt.Errorf("%w: block-dollar source is not one complete physical line", ErrUnsupportedMathExpressionShape)
	}
	return nil
}

func mathContainsUnescapedDollar(input []byte, range_ Range) bool {
	for index := range_.Start; index < range_.End; index++ {
		if input[index] == '$' && !mathDollarEscaped(input, index) {
			return true
		}
	}
	return false
}

func mathDollarEscaped(input []byte, index int) bool {
	backslashes := 0
	for position := index - 1; position >= 0 && input[position] == '\\'; position-- {
		backslashes++
	}
	return backslashes%2 == 1
}

func mathAdjacentDollar(input []byte, index int) bool {
	return index > 0 && input[index-1] == '$' || index+1 < len(input) && input[index+1] == '$'
}
