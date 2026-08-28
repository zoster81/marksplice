package native

import (
	"bytes"
	"sort"

	"github.com/zoster81/marksplice/internal/parser"
)

func nativeMathExpressionObservations(source []byte, blocks blockParseResult, analyses []inlineAnalysis, inlineNodes []parser.Node) []parser.MathExpressionObservation {
	inline := mergeNativeMathExpressions(
		nativeInlineDollarMathObservations(source, analyses),
		nativeInlineBacktickMathObservations(source, inlineNodes),
	)
	return mergeNativeMathExpressions(inline, nativeBlockDollarMathObservations(source, blocks.nodes))
}

func mergeNativeMathExpressions(left, right []parser.MathExpressionObservation) []parser.MathExpressionObservation {
	result := make([]parser.MathExpressionObservation, 0, len(left)+len(right))
	leftIndex, rightIndex := 0, 0
	for leftIndex < len(left) && rightIndex < len(right) {
		if nativeMathBefore(left[leftIndex], right[rightIndex]) {
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

func nativeMathBefore(left, right parser.MathExpressionObservation) bool {
	return left.Range.Start < right.Range.Start || left.Range.Start == right.Range.Start && left.Range.End <= right.Range.End
}

func nativeInlineDollarMathObservations(source []byte, analyses []inlineAnalysis) []parser.MathExpressionObservation {
	result := make([]parser.MathExpressionObservation, 0)
	for _, analysis := range analyses {
		boundaries := nativeDelimiterTextBoundaries(analysis.block, analysis.delimiters.matches)
		result = append(result, scanNativeInlineDollarRuns(source, analysis.block, analysis.relationshipExclusions, boundaries)...)
	}
	return result
}

func nativeDelimiterTextBoundaries(block inlineBlock, matches []delimiterMatch) [][]int {
	if len(matches) == 0 {
		return nil
	}
	boundaries := make([][]int, len(block.segments))
	for _, match := range matches {
		if match.startSegment >= 0 && match.startSegment < len(block.segments) {
			segment := block.segments[match.startSegment]
			if match.content.Start > segment.Start && match.content.Start < segment.End {
				boundaries[match.startSegment] = append(boundaries[match.startSegment], match.content.Start)
			}
		}
		if match.endSegment >= 0 && match.endSegment < len(block.segments) {
			segment := block.segments[match.endSegment]
			if match.content.End > segment.Start && match.content.End < segment.End {
				boundaries[match.endSegment] = append(boundaries[match.endSegment], match.content.End)
			}
		}
	}
	for segmentIndex := range boundaries {
		sort.Ints(boundaries[segmentIndex])
		boundaries[segmentIndex] = compactNativeMathBoundaries(boundaries[segmentIndex])
	}
	return boundaries
}

func compactNativeMathBoundaries(boundaries []int) []int {
	if len(boundaries) < 2 {
		return boundaries
	}
	result := boundaries[:1]
	for _, boundary := range boundaries[1:] {
		if result[len(result)-1] != boundary {
			result = append(result, boundary)
		}
	}
	return result
}

type nativeMathScanCursor struct {
	exclusions     []parser.Range
	boundaries     []int
	exclusionIndex int
	boundaryIndex  int
}

func scanNativeInlineDollarRuns(source []byte, block inlineBlock, exclusions [][]parser.Range, boundaries [][]int) []parser.MathExpressionObservation {
	result := make([]parser.MathExpressionObservation, 0)
	for segmentIndex, segment := range block.segments {
		result = scanNativeInlineDollarSegment(
			source,
			segment,
			nativeMathSegmentExclusions(exclusions, segmentIndex),
			nativeMathSegmentBoundaries(boundaries, segmentIndex),
			result,
		)
	}
	return result
}

func nativeMathSegmentExclusions(exclusions [][]parser.Range, index int) []parser.Range {
	if index >= len(exclusions) {
		return nil
	}
	return exclusions[index]
}

func nativeMathSegmentBoundaries(boundaries [][]int, index int) []int {
	if index >= len(boundaries) {
		return nil
	}
	return boundaries[index]
}

func scanNativeInlineDollarSegment(source []byte, segment parser.Range, exclusions []parser.Range, boundaries []int, result []parser.MathExpressionObservation) []parser.MathExpressionObservation {
	cursor := nativeMathScanCursor{exclusions: exclusions, boundaries: boundaries}
	for position := segment.Start; position < segment.End; {
		cursor.advance(position)
		if end, excluded := cursor.excludedEnd(position); excluded {
			position = end
			continue
		}
		limit := cursor.limit(position, segment.End)
		relative := bytes.IndexByte(source[position:limit], '$')
		if relative < 0 {
			position = limit
			continue
		}
		position += relative
		observation, end, ok := nativeInlineDollarMathAt(source, position, limit)
		if !ok {
			position++
			continue
		}
		result = append(result, observation)
		position = end
	}
	return result
}

func (cursor *nativeMathScanCursor) advance(position int) {
	for cursor.exclusionIndex < len(cursor.exclusions) && position >= cursor.exclusions[cursor.exclusionIndex].End {
		cursor.exclusionIndex++
	}
	for cursor.boundaryIndex < len(cursor.boundaries) && position >= cursor.boundaries[cursor.boundaryIndex] {
		cursor.boundaryIndex++
	}
}

func (cursor *nativeMathScanCursor) excludedEnd(position int) (int, bool) {
	if cursor.exclusionIndex >= len(cursor.exclusions) || position < cursor.exclusions[cursor.exclusionIndex].Start {
		return 0, false
	}
	return cursor.exclusions[cursor.exclusionIndex].End, true
}

func (cursor *nativeMathScanCursor) limit(position, end int) int {
	limit := end
	if cursor.exclusionIndex < len(cursor.exclusions) {
		exclusionStart := cursor.exclusions[cursor.exclusionIndex].Start
		if exclusionStart > position && exclusionStart < limit {
			limit = exclusionStart
		}
	}
	if cursor.boundaryIndex < len(cursor.boundaries) && cursor.boundaries[cursor.boundaryIndex] < limit {
		limit = cursor.boundaries[cursor.boundaryIndex]
	}
	return limit
}

func nativeInlineDollarMathAt(source []byte, anchor, limit int) (parser.MathExpressionObservation, int, bool) {
	if anchor < 0 || anchor >= limit || source[anchor] != '$' || !nativeSingleMathDollarDelimiter(source, anchor, limit) {
		return parser.MathExpressionObservation{}, anchor, false
	}
	for end := anchor + 1; end < limit; end++ {
		if source[end] != '$' || !nativeSingleMathDollarDelimiter(source, end, limit) {
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

func nativeSingleMathDollarDelimiter(source []byte, index, limit int) bool {
	if index < 0 || index >= limit || source[index] != '$' || nativeSourceByteEscaped(source, index) {
		return false
	}
	return (index == 0 || source[index-1] != '$') && (index+1 >= limit || source[index+1] != '$')
}

func nativeInlineBacktickMathObservations(source []byte, nodes []parser.Node) []parser.MathExpressionObservation {
	result := make([]parser.MathExpressionObservation, 0)
	for _, node := range nodes {
		if node.Kind != parser.KindCodeSpan {
			continue
		}
		if observation, ok := nativeInlineBacktickMathObservation(source, node); ok {
			result = append(result, observation)
		}
	}
	return result
}

func nativeInlineBacktickMathObservation(source []byte, span parser.Node) (parser.MathExpressionObservation, bool) {
	payload := span.Range
	anchor := span.Anchor
	if !payload.Valid(len(source)) || payload.Start == payload.End ||
		!validNativeInlineBacktickMathDelimiters(source, anchor, payload) ||
		!validNativeInlineBacktickMathPayload(source, payload) {
		return parser.MathExpressionObservation{}, false
	}
	return parser.MathExpressionObservation{
		Style:        parser.MathExpressionInlineBacktick,
		Range:        parser.Range{Start: anchor - 1, End: payload.End + 2},
		PayloadRange: payload,
	}, true
}

func validNativeInlineBacktickMathDelimiters(source []byte, anchor int, payload parser.Range) bool {
	if anchor <= 0 || payload.Start != anchor+1 || payload.End+1 >= len(source) {
		return false
	}
	if source[anchor] != '`' || source[payload.End] != '`' || source[anchor-1] != '$' || source[payload.End+1] != '$' {
		return false
	}
	if nativeSourceByteEscaped(source, anchor-1) || nativeSourceByteEscaped(source, payload.End+1) {
		return false
	}
	return (anchor <= 1 || source[anchor-2] != '$') && (payload.End+2 >= len(source) || source[payload.End+2] != '$')
}

func validNativeInlineBacktickMathPayload(source []byte, payload parser.Range) bool {
	return bytes.IndexByte(source[payload.Start:payload.End], '`') < 0 && !nativeHasUnescapedMathDollar(source, payload.Start, payload.End)
}

func nativeBlockDollarMathObservations(source []byte, nodes []parser.Node) []parser.MathExpressionObservation {
	result := make([]parser.MathExpressionObservation, 0)
	for _, node := range nodes {
		if node.Kind != parser.KindParagraph || !node.TopLevel {
			continue
		}
		if observation, ok := nativeBlockDollarMathObservation(source, node.Range); ok {
			result = append(result, observation)
		}
	}
	return result
}

func nativeBlockDollarMathObservation(source []byte, range_ parser.Range) (parser.MathExpressionObservation, bool) {
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.MathExpressionObservation{}, false
	}
	start := nativePhysicalLineStart(source, range_.Start)
	end := nativePhysicalLineEnd(source, range_.End)
	if range_.Start != start || range_.End != end || end-start < 5 || bytes.ContainsAny(source[start:end], "\r\n") ||
		!bytes.HasPrefix(source[start:end], []byte("$$")) || !bytes.HasSuffix(source[start:end], []byte("$$")) {
		return parser.MathExpressionObservation{}, false
	}
	payload := parser.Range{Start: start + 2, End: end - 2}
	if payload.Start >= payload.End || nativeHasUnescapedMathDollar(source, payload.Start, payload.End) {
		return parser.MathExpressionObservation{}, false
	}
	return parser.MathExpressionObservation{
		Style:        parser.MathExpressionBlockDollar,
		Range:        parser.Range{Start: start, End: end},
		PayloadRange: payload,
		TopLevel:     true,
	}, true
}

func removeNativeMathGFMConflicts(nodes []parser.Node, expressions []parser.MathExpressionObservation) []parser.Node {
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

func nativeHasUnescapedMathDollar(source []byte, start, end int) bool {
	for index := start; index < end; index++ {
		if source[index] == '$' && !nativeSourceByteEscaped(source, index) {
			return true
		}
	}
	return false
}

func nativeSourceByteEscaped(source []byte, index int) bool {
	backslashes := 0
	for position := index - 1; position >= 0 && source[position] == '\\'; position-- {
		backslashes++
	}
	return backslashes%2 == 1
}

func nativePhysicalLineStart(source []byte, offset int) int {
	if offset > len(source) {
		offset = len(source)
	}
	for offset > 0 && source[offset-1] != '\n' && source[offset-1] != '\r' {
		offset--
	}
	return offset
}

func nativePhysicalLineEnd(source []byte, offset int) int {
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
