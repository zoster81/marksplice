package native

import (
	"bytes"
	"cmp"
	"html"
	"slices"
	"sort"
	"strings"

	"github.com/zoster81/marksplice/internal/parser"
)

type headingTerminalKind uint8

const (
	headingTerminalOmit headingTerminalKind = iota
	headingTerminalCode
)

type headingTerminal struct {
	range_  parser.Range
	content parser.Range
	kind    headingTerminalKind
}

func completeBackendBlockFacts(source []byte, nodes []parser.Node, analyses []inlineAnalysis) {
	byStart := inlineAnalysisStartIndex(analyses)
	for index := range nodes {
		if nodes[index].Kind != parser.KindHeading {
			continue
		}
		analysis, ok := headingInlineAnalysis(nodes[index].Range, analyses, byStart)
		if !ok {
			nodes[index].HeadingText = ""
			continue
		}
		nodes[index].HeadingText = nativeHeadingSemanticText(source, analysis)
	}
}

func inlineAnalysisStartIndex(analyses []inlineAnalysis) map[int]int {
	index := make(map[int]int, len(analyses))
	for analysisIndex, analysis := range analyses {
		if len(analysis.block.segments) == 0 || analysis.block.tableCell {
			continue
		}
		index[analysis.block.segments[0].Start] = analysisIndex
	}
	return index
}

func headingInlineAnalysis(range_ parser.Range, analyses []inlineAnalysis, byStart map[int]int) (inlineAnalysis, bool) {
	index, ok := byStart[range_.Start]
	if !ok || index < 0 || index >= len(analyses) {
		return inlineAnalysis{}, false
	}
	analysis := analyses[index]
	last := analysis.block.segments[len(analysis.block.segments)-1]
	if last.Start >= range_.End || last.End < range_.End {
		return inlineAnalysis{}, false
	}
	return analysis, true
}

func nativeHeadingSemanticText(source []byte, analysis inlineAnalysis) string {
	terminals := headingTerminals(source, analysis.owners)
	removed := headingRemovedRanges(analysis.delimiters)
	return renderHeadingSegments(source, analysis.block.segments, terminals, removed)
}

func headingTerminals(source []byte, owners []inlineSpan) []headingTerminal {
	result := make([]headingTerminal, 0, len(owners))
	for _, owner := range owners {
		terminal := headingTerminal{range_: parser.Range{Start: owner.start, End: owner.end}}
		switch owner.kind {
		case parser.KindCodeSpan:
			terminal.kind = headingTerminalCode
			terminal.content = owner.content
		case parser.KindAutoLink, parser.KindRawHTML:
			terminal.kind = headingTerminalOmit
		default:
			if owner.start >= 0 && owner.start < len(source) && source[owner.start] == '<' {
				terminal.kind = headingTerminalOmit
			} else if owner.start >= 0 && owner.start < len(source) && source[owner.start] == '`' {
				terminal.kind = headingTerminalCode
				terminal.content = parser.Range{Start: owner.end, End: owner.end}
			} else {
				continue
			}
		}
		result = append(result, terminal)
	}
	if len(result) > 1 {
		slices.SortStableFunc(result, func(left, right headingTerminal) int {
			if order := cmp.Compare(left.range_.Start, right.range_.Start); order != 0 {
				return order
			}
			return cmp.Compare(right.range_.End, left.range_.End)
		})
	}
	return result
}

func headingRemovedRanges(parsed delimiterParseResult) []parser.Range {
	result := make([]parser.Range, 0, len(parsed.matches)*2+len(parsed.composites)*2)
	for _, match := range parsed.matches {
		result = append(result, match.openingConsumed, match.closingConsumed)
	}
	for _, composite := range parsed.composites {
		if !composite.active {
			continue
		}
		result = appendMarkupShell(result,
			parser.Range{Start: composite.start, End: composite.end},
			composite.label,
		)
	}
	return mergeHeadingRanges(result)
}

func appendMarkupShell(ranges []parser.Range, syntax, content parser.Range) []parser.Range {
	if syntax.Start < content.Start {
		ranges = append(ranges, parser.Range{Start: syntax.Start, End: content.Start})
	}
	if content.End < syntax.End {
		ranges = append(ranges, parser.Range{Start: content.End, End: syntax.End})
	}
	return ranges
}

func mergeHeadingRanges(ranges []parser.Range) []parser.Range {
	if len(ranges) < 2 {
		return ranges
	}
	slices.SortFunc(ranges, func(left, right parser.Range) int {
		if order := cmp.Compare(left.Start, right.Start); order != 0 {
			return order
		}
		return cmp.Compare(left.End, right.End)
	})
	merged := ranges[:1]
	for _, current := range ranges[1:] {
		last := &merged[len(merged)-1]
		if current.Start <= last.End {
			if current.End > last.End {
				last.End = current.End
			}
			continue
		}
		merged = append(merged, current)
	}
	return merged
}

func renderHeadingSegments(source []byte, segments []parser.Range, terminals []headingTerminal, removed []parser.Range) string {
	var result strings.Builder
	for _, segment := range segments {
		end := headingSegmentTextEnd(source, segment)
		if segment.Start >= end {
			continue
		}
		renderHeadingSegment(&result, source, parser.Range{Start: segment.Start, End: end}, terminals, removed)
	}
	return result.String()
}

func headingSegmentTextEnd(source []byte, segment parser.Range) int {
	end := segment.End
	if end <= segment.Start || end > len(source) {
		return end
	}
	for end > segment.Start && (source[end-1] == ' ' || source[end-1] == '\t') {
		end--
	}
	return end
}

func renderHeadingSegment(result *strings.Builder, source []byte, segment parser.Range, terminals []headingTerminal, removed []parser.Range) {
	position := segment.Start
	terminalIndex := firstHeadingTerminal(terminals, position)
	removedIndex := firstHeadingRange(removed, position)
	for position < segment.End {
		terminalIndex = advanceHeadingTerminals(terminals, terminalIndex, position)
		removedIndex = advanceHeadingRanges(removed, removedIndex, position)
		next := nextHeadingEventPosition(segment.End, terminals, terminalIndex, removed, removedIndex)
		if next > position {
			appendDecodedHeadingText(result, source[position:next])
			position = next
			continue
		}
		if consumed, ok := consumeHeadingTerminal(result, source, segment.End, terminals, terminalIndex, position); ok {
			position = consumed
			continue
		}
		if consumed, ok := consumeHeadingRemoved(segment.End, terminals, terminalIndex, removed, removedIndex, position); ok {
			position = consumed
			continue
		}
		position++
	}
}

func nextHeadingEventPosition(limit int, terminals []headingTerminal, terminalIndex int, removed []parser.Range, removedIndex int) int {
	next := limit
	if terminalIndex < len(terminals) && terminals[terminalIndex].range_.Start < next {
		next = terminals[terminalIndex].range_.Start
	}
	if removedIndex < len(removed) && removed[removedIndex].Start < next {
		next = removed[removedIndex].Start
	}
	return next
}

func consumeHeadingTerminal(result *strings.Builder, source []byte, limit int, terminals []headingTerminal, terminalIndex, position int) (int, bool) {
	if terminalIndex >= len(terminals) {
		return position, false
	}
	terminal := terminals[terminalIndex]
	if terminal.range_.Start > position || terminal.range_.End <= position {
		return position, false
	}
	if position == terminal.range_.Start && terminal.kind == headingTerminalCode && terminal.content.Valid(len(source)) {
		result.Write(source[terminal.content.Start:terminal.content.End])
	}
	return minHeadingPosition(terminal.range_.End, limit), true
}

func consumeHeadingRemoved(limit int, terminals []headingTerminal, terminalIndex int, removed []parser.Range, removedIndex, position int) (int, bool) {
	if removedIndex >= len(removed) || removed[removedIndex].Start > position || removed[removedIndex].End <= position {
		return position, false
	}
	skipEnd := removed[removedIndex].End
	if terminalIndex < len(terminals) && terminals[terminalIndex].range_.Start > position && terminals[terminalIndex].range_.Start < skipEnd {
		skipEnd = terminals[terminalIndex].range_.Start
	}
	return minHeadingPosition(skipEnd, limit), true
}

func firstHeadingTerminal(terminals []headingTerminal, position int) int {
	return sort.Search(len(terminals), func(index int) bool { return terminals[index].range_.End > position })
}

func firstHeadingRange(ranges []parser.Range, position int) int {
	return sort.Search(len(ranges), func(index int) bool { return ranges[index].End > position })
}

func advanceHeadingTerminals(terminals []headingTerminal, index, position int) int {
	for index < len(terminals) && terminals[index].range_.End <= position {
		index++
	}
	return index
}

func advanceHeadingRanges(ranges []parser.Range, index, position int) int {
	for index < len(ranges) && ranges[index].End <= position {
		index++
	}
	return index
}

func minHeadingPosition(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func appendDecodedHeadingText(result *strings.Builder, source []byte) {
	for position := 0; position < len(source); {
		if source[position] == '\\' && position+1 < len(source) && asciiPunctuation(source[position+1]) {
			result.WriteByte(source[position+1])
			position += 2
			continue
		}
		if source[position] == '&' {
			if decoded, end, ok := decodeHeadingEntity(source, position); ok {
				result.WriteString(decoded)
				position = end
				continue
			}
		}
		result.WriteByte(source[position])
		position++
	}
}

func decodeHeadingEntity(source []byte, start int) (string, int, bool) {
	limit := minHeadingPosition(len(source), start+64)
	relative := bytes.IndexByte(source[start:limit], ';')
	if relative < 0 {
		return "", start, false
	}
	end := start + relative + 1
	raw := string(source[start:end])
	decoded := html.UnescapeString(raw)
	if decoded == raw {
		return "", start, false
	}
	return decoded, end, true
}
