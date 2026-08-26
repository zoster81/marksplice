package marksplice

import (
	"fmt"
	"strings"
)

// MathInline returns one conservative GitHub-compatible `$...$` construction value.
// Mathematical payload remains opaque and must fit on one physical line.
func MathInline(payload string) Inline {
	return Inline{kind: inlineConstructionMathDollar, text: payload}
}

// MathBacktickInline returns one conservative GitHub-compatible `$`-backtick
// construction value for payload that would otherwise overlap Markdown syntax.
func MathBacktickInline(payload string) Inline {
	return Inline{kind: inlineConstructionMathBacktick, text: payload}
}

// AppendMathBlock appends one canonical top-level `$$...$$` mathematical block.
// Multiline mathematical payload belongs in an exact-info `math` fenced block.
func (b *DocumentBuilder) AppendMathBlock(payload string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{kind: constructionMathBlock, inlineGFM: payload})
}

func writeTypedInlineMath(output *strings.Builder, inline Inline, expected *[]typedInlineExpectation) error {
	style := MathExpressionInlineDollar
	if inline.kind == inlineConstructionMathBacktick {
		style = MathExpressionInlineBacktick
	}
	if err := validateMathConstructionPayload(inline.text, style == MathExpressionInlineBacktick); err != nil {
		return err
	}
	output.WriteByte('$')
	if style == MathExpressionInlineBacktick {
		output.WriteByte('`')
	}
	start := output.Len()
	output.WriteString(inline.text)
	end := output.Len()
	if style == MathExpressionInlineBacktick {
		output.WriteByte('`')
	}
	output.WriteByte('$')
	*expected = append(*expected, typedInlineExpectation{
		kind:         KindMathExpression,
		contentRange: Range{Start: start, End: end},
		mathStyle:    style,
	})
	return nil
}

func validateConstructionMathPayload(payload string) error {
	if err := validateMathConstructionPayload(payload, false); err != nil {
		return fmt.Errorf("%w: mathematical payload: %v", ErrInvalidConstruction, err)
	}
	return nil
}

func validateMathConstructionPayload(payload string, rejectBacktick bool) error {
	if err := validateTypedInlineText(payload); err != nil {
		return err
	}
	if rejectBacktick && strings.IndexByte(payload, '`') >= 0 {
		return fmt.Errorf("mathematical payload contains a backtick")
	}
	for index := 0; index < len(payload); index++ {
		if payload[index] != '$' {
			continue
		}
		backslashes := 0
		for position := index - 1; position >= 0 && payload[position] == '\\'; position-- {
			backslashes++
		}
		if backslashes%2 == 0 {
			return fmt.Errorf("mathematical payload contains an unescaped dollar")
		}
	}
	return nil
}
