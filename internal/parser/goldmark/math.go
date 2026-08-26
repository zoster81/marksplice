package goldmark

import (
	"bytes"

	"github.com/yuin/goldmark/ast"

	"github.com/zoster81/marksplice/internal/parser"
)

func mathExpressionObservations(source []byte, root ast.Node) []parser.MathExpressionObservation {
	inline := mergeMathExpressionObservations(
		inlineDollarMathObservations(source, root),
		inlineBacktickMathObservations(source, root),
	)
	return mergeMathExpressionObservations(inline, blockDollarMathObservations(source, root))
}

func mergeMathExpressionObservations(left, right []parser.MathExpressionObservation) []parser.MathExpressionObservation {
	result := make([]parser.MathExpressionObservation, 0, len(left)+len(right))
	leftIndex, rightIndex := 0, 0
	for leftIndex < len(left) && rightIndex < len(right) {
		if mathExpressionBefore(left[leftIndex], right[rightIndex]) {
			result = append(result, left[leftIndex])
			leftIndex++
			continue
		}
		result = append(result, right[rightIndex])
		rightIndex++
	}
	result = append(result, left[leftIndex:]...)
	result = append(result, right[rightIndex:]...)
	return result
}

func mathExpressionBefore(left, right parser.MathExpressionObservation) bool {
	return left.Range.Start < right.Range.Start || left.Range.Start == right.Range.Start && left.Range.End <= right.Range.End
}

func inlineDollarMathObservations(source []byte, root ast.Node) []parser.MathExpressionObservation {
	runs := unresolvedReferenceTextRuns(source, root)
	result := make([]parser.MathExpressionObservation, 0)
	for _, run := range runs {
		for offset := run.start; offset < run.end; offset++ {
			observation, end, ok := inlineDollarMathAt(source, offset, run.end)
			if !ok {
				continue
			}
			result = append(result, observation)
			offset = end - 1
		}
	}
	return result
}

func inlineDollarMathAt(source []byte, anchor, limit int) (parser.MathExpressionObservation, int, bool) {
	if anchor < 0 || anchor >= limit || source[anchor] != '$' || !singleMathDollarDelimiter(source, anchor, limit) {
		return parser.MathExpressionObservation{}, anchor, false
	}
	for end := anchor + 1; end < limit; end++ {
		if source[end] != '$' || !singleMathDollarDelimiter(source, end, limit) {
			continue
		}
		if end == anchor+1 {
			return parser.MathExpressionObservation{}, anchor, false
		}
		return parser.MathExpressionObservation{
			Style:        parser.MathExpressionInlineDollar,
			Range:        parser.Range{Start: anchor, End: end + 1},
			PayloadRange: parser.Range{Start: anchor + 1, End: end},
		}, end + 1, true
	}
	return parser.MathExpressionObservation{}, anchor, false
}

func singleMathDollarDelimiter(source []byte, index, limit int) bool {
	if index < 0 || index >= limit || source[index] != '$' || sourceByteEscaped(source, index) {
		return false
	}
	return (index == 0 || source[index-1] != '$') && (index+1 >= limit || source[index+1] != '$')
}

func inlineBacktickMathObservations(source []byte, root ast.Node) []parser.MathExpressionObservation {
	result := make([]parser.MathExpressionObservation, 0)
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		span, ok := node.(*ast.CodeSpan)
		if !ok {
			return ast.WalkContinue, nil
		}
		if observation, ok := inlineBacktickMathObservation(source, span); ok {
			result = append(result, observation)
		}
		return ast.WalkContinue, nil
	})
	return result
}

func inlineBacktickMathObservation(source []byte, span *ast.CodeSpan) (parser.MathExpressionObservation, bool) {
	payload, ok := simplePlainTextInlineRange(source, span)
	anchor := span.Pos()
	if !ok || !validInlineBacktickMathDelimiters(source, anchor, payload) || !validInlineBacktickMathPayload(source, payload) {
		return parser.MathExpressionObservation{}, false
	}
	return parser.MathExpressionObservation{
		Style:        parser.MathExpressionInlineBacktick,
		Range:        parser.Range{Start: anchor - 1, End: payload.End + 2},
		PayloadRange: payload,
	}, true
}

func validInlineBacktickMathDelimiters(source []byte, anchor int, payload parser.Range) bool {
	if anchor <= 0 || payload.Start != anchor+1 || payload.End+1 >= len(source) {
		return false
	}
	if source[anchor] != '`' || source[payload.End] != '`' || source[anchor-1] != '$' || source[payload.End+1] != '$' {
		return false
	}
	if sourceByteEscaped(source, anchor-1) || sourceByteEscaped(source, payload.End+1) {
		return false
	}
	return (anchor <= 1 || source[anchor-2] != '$') && (payload.End+2 >= len(source) || source[payload.End+2] != '$')
}

func validInlineBacktickMathPayload(source []byte, payload parser.Range) bool {
	return bytes.IndexByte(source[payload.Start:payload.End], '`') < 0 && !hasUnescapedMathDollar(source, payload.Start, payload.End)
}

func blockDollarMathObservations(source []byte, root ast.Node) []parser.MathExpressionObservation {
	result := make([]parser.MathExpressionObservation, 0)
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		paragraph, ok := node.(*ast.Paragraph)
		if !ok {
			return ast.WalkContinue, nil
		}
		if observation, ok := blockDollarMathObservation(source, paragraph); ok {
			result = append(result, observation)
		}
		return ast.WalkContinue, nil
	})
	return result
}

func blockDollarMathObservation(source []byte, paragraph *ast.Paragraph) (parser.MathExpressionObservation, bool) {
	if _, ok := paragraph.Parent().(*ast.Document); !ok {
		return parser.MathExpressionObservation{}, false
	}
	lines := paragraph.Lines()
	if lines == nil || lines.Len() != 1 {
		return parser.MathExpressionObservation{}, false
	}
	segment := lines.At(0)
	start := mathPhysicalLineStart(source, segment.Start)
	end := mathPhysicalLineEnd(source, segment.Stop)
	if segment.Start != start || end-start < 5 || !bytes.HasPrefix(source[start:end], []byte("$$")) || !bytes.HasSuffix(source[start:end], []byte("$$")) {
		return parser.MathExpressionObservation{}, false
	}
	payload := parser.Range{Start: start + 2, End: end - 2}
	if payload.Start >= payload.End || hasUnescapedMathDollar(source, payload.Start, payload.End) {
		return parser.MathExpressionObservation{}, false
	}
	return parser.MathExpressionObservation{
		Style:        parser.MathExpressionBlockDollar,
		Range:        parser.Range{Start: start, End: end},
		PayloadRange: payload,
		TopLevel:     true,
	}, true
}

func hasUnescapedMathDollar(source []byte, start, end int) bool {
	for index := start; index < end; index++ {
		if source[index] == '$' && !sourceByteEscaped(source, index) {
			return true
		}
	}
	return false
}

func mathPhysicalLineStart(source []byte, offset int) int {
	if offset > len(source) {
		offset = len(source)
	}
	for offset > 0 && source[offset-1] != '\n' && source[offset-1] != '\r' {
		offset--
	}
	return offset
}

func mathPhysicalLineEnd(source []byte, offset int) int {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source) {
		offset = len(source)
	}
	for offset < len(source) && source[offset] != '\n' && source[offset] != '\r' {
		offset++
	}
	return offset
}

func removeMathGFMConflicts(nodes []parser.Node, expressions []parser.MathExpressionObservation) []parser.Node {
	if len(expressions) == 0 {
		return nodes
	}
	backtickPayloads := make(map[parser.Range]struct{})
	blockRanges := make(map[parser.Range]struct{})
	for _, expression := range expressions {
		switch expression.Style {
		case parser.MathExpressionInlineBacktick:
			backtickPayloads[expression.PayloadRange] = struct{}{}
		case parser.MathExpressionBlockDollar:
			blockRanges[expression.Range] = struct{}{}
		}
	}
	result := nodes[:0]
	for _, node := range nodes {
		if node.Kind == parser.KindCodeSpan {
			if _, suppressed := backtickPayloads[node.Range]; suppressed {
				continue
			}
		}
		if node.Kind == parser.KindParagraph {
			if _, suppressed := blockRanges[node.Range]; suppressed {
				continue
			}
		}
		result = append(result, node)
	}
	clear(nodes[len(result):])
	return result
}
