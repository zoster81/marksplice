package native

import (
	"bytes"
	"cmp"
	"slices"

	"github.com/zoster81/marksplice/internal/parser"
)

type inlineBlock struct {
	segments        []parser.Range
	tableCell       bool
	prefixExclusion parser.Range
}

type backtickRun struct {
	segment int
	start   int
	end     int
	length  int
}

type inlineSpan struct {
	segment       int
	start         int
	endSegment    int
	end           int
	kind          parser.Kind
	content       parser.Range
	autoLinkEmail bool
}

type inlineParseResult struct {
	nodes      []parser.Node
	usages     []parser.LinkUsage
	unresolved []parser.UnresolvedReferenceUsage
}

type inlineAnalysis struct {
	block                  inlineBlock
	owners                 []inlineSpan
	delimiters             delimiterParseResult
	relationshipExclusions [][]parser.Range
	parsed                 inlineParseResult
}

// ParseInlines returns parser-independent inline node observations from the
// completed Marksplice-native parser. Production selection uses the same Native backend.
func ParseInlines(source []byte) ([]parser.Node, error) {
	observed, err := ParseInlineObservations(source)
	return observed.Nodes, err
}

// ParseInlineObservations returns the parser-independent inline observation subset.
// The complete native parser.Backend is implemented separately by Backend.
func ParseInlineObservations(source []byte) (parser.DocumentObservations, error) {
	blocks := parseBlockLines(source, physicalLines(source), true)
	inline := parseInlineBlocks(source, blocks.inlines, blocks.references)
	return parser.DocumentObservations{
		Nodes:                     inline.nodes,
		LinkUsages:                inline.usages,
		UnresolvedReferenceUsages: inline.unresolved,
	}, nil
}

func appendInlineRange(result *blockParseResult, range_ parser.Range) {
	if range_.Start >= range_.End {
		return
	}
	result.inlines = append(result.inlines, inlineBlock{segments: []parser.Range{range_}})
}

func appendInlineLines(result *blockParseResult, source []byte, lines []physicalLine, first, last int) {
	segments := inlineLineRanges(source, lines, first, last)
	if len(segments) == 0 {
		return
	}
	result.inlines = append(result.inlines, inlineBlock{segments: segments})
}

func appendTableInlineBlocks(result *blockParseResult, nodes []parser.Node) {
	for _, node := range nodes {
		if node.Kind == parser.KindTableCell && node.Range.Start < node.Range.End {
			result.inlines = append(result.inlines, inlineBlock{segments: []parser.Range{node.Range}, tableCell: true})
		}
	}
}

func inlineLineRanges(source []byte, lines []physicalLine, first, last int) []parser.Range {
	if first < 0 || last > len(lines) || first >= last {
		return nil
	}
	segments := make([]parser.Range, 0, last-first)
	containerizedFirst := lines[first].start != lines[first].physicalStart
	for index := first; index < last; index++ {
		line := lines[index]
		if blankLine(source, line) {
			continue
		}
		start := line.start
		if indent, ok := ordinaryIndent(source, line); ok {
			start += indent
		} else if containerizedFirst && index > first && line.start == line.physicalStart {
			start = stripIndentColumns(source, line, 4).start
		}
		if start < line.end {
			segments = append(segments, parser.Range{Start: start, End: line.end})
		}
	}
	return segments
}

func parseInlineBlocks(source []byte, blocks []inlineBlock, definitions []referenceDefinitionParse) inlineParseResult {
	return parseInlineBlocksIndexed(source, blocks, basicReferenceDefinitions(definitions))
}

func parseInlineBlocksIndexed(source []byte, blocks []inlineBlock, definitions referenceDefinitionIndex) inlineParseResult {
	result := inlineParseResult{}
	for _, block := range blocks {
		appendInlineAnalysis(&result, analyzeInlineBlock(source, block, definitions))
	}
	sortInlineParseResult(&result)
	return result
}

func analyzeInlineBlocks(source []byte, blocks []inlineBlock, definitions []referenceDefinitionParse) []inlineAnalysis {
	resolved := basicReferenceDefinitions(definitions)
	analyses := make([]inlineAnalysis, len(blocks))
	for index, block := range blocks {
		analyses[index] = analyzeInlineBlock(source, block, resolved)
	}
	return analyses
}

func mergeInlineAnalyses(analyses []inlineAnalysis, additionalNodeCapacity int) inlineParseResult {
	if len(analyses) == 0 {
		return inlineParseResult{}
	}
	if len(analyses) == 1 {
		return analyses[0].parsed
	}
	nodeCount, usageCount, unresolvedCount := 0, 0, 0
	for _, analysis := range analyses {
		nodeCount += len(analysis.parsed.nodes)
		usageCount += len(analysis.parsed.usages)
		unresolvedCount += len(analysis.parsed.unresolved)
	}
	result := inlineParseResult{
		nodes:      make([]parser.Node, 0, nodeCount+max(0, additionalNodeCapacity)),
		usages:     make([]parser.LinkUsage, 0, usageCount),
		unresolved: make([]parser.UnresolvedReferenceUsage, 0, unresolvedCount),
	}
	for _, analysis := range analyses {
		appendInlineAnalysis(&result, analysis)
	}
	sortInlineParseResult(&result)
	return result
}

func appendInlineAnalysis(result *inlineParseResult, analysis inlineAnalysis) {
	result.nodes = append(result.nodes, analysis.parsed.nodes...)
	result.usages = append(result.usages, analysis.parsed.usages...)
	result.unresolved = append(result.unresolved, analysis.parsed.unresolved...)
}

func sortInlineParseResult(result *inlineParseResult) {
	if len(result.usages) > 1 {
		slices.SortStableFunc(result.usages, func(left, right parser.LinkUsage) int {
			return cmp.Compare(left.Anchor, right.Anchor)
		})
	}
	if len(result.unresolved) > 1 {
		slices.SortStableFunc(result.unresolved, func(left, right parser.UnresolvedReferenceUsage) int {
			return cmp.Compare(left.Anchor, right.Anchor)
		})
	}
}

func analyzeInlineBlock(source []byte, block inlineBlock, definitions referenceDefinitionIndex) inlineAnalysis {
	runs := collectBacktickRuns(source, block)
	spans := collectInlineSpans(source, block)
	owners, barriers := resolvePrimaryInlineOwners(source, block, runs, nextSameLengthRuns(runs), spans)
	ownerExclusions := inlineOwnerExclusions(block, owners, nil)
	composites := collectCompositeInlinesIndexed(source, block, owners, definitions, ownerExclusions)
	delimiterExclusions := appendCompositeDelimiterExclusions(ownerExclusions, block, composites)
	delimiters := parseDelimiterObservationsWithExclusions(source, block, owners, barriers, composites, delimiterExclusions)
	relationshipExclusions := promoteRelationshipExclusions(delimiterExclusions, block, composites)
	nodes := primaryInlineOwnerObservations(source, block, owners, delimiters.composites)
	nodes = append(nodes, projectCompositeObservations(source, block, owners, delimiters.composites, delimiters.matches)...)
	nodes = append(nodes, delimiters.nodes...)
	sortInlineNodes(nodes)
	usages := compositeLinkUsages(delimiters.composites)
	usages = append(usages, autoLinkUsages(nodes)...)
	return inlineAnalysis{
		block:                  block,
		owners:                 owners,
		delimiters:             delimiters,
		relationshipExclusions: relationshipExclusions,
		parsed: inlineParseResult{
			nodes:      nodes,
			usages:     usages,
			unresolved: unresolvedReferenceUsages(source, block, relationshipExclusions, delimiters.matches, definitions),
		},
	}
}

func sortInlineNodes(nodes []parser.Node) {
	if len(nodes) < 2 {
		return
	}
	slices.SortStableFunc(nodes, func(left, right parser.Node) int {
		if order := cmp.Compare(left.Range.Start, right.Range.Start); order != 0 {
			return order
		}
		return cmp.Compare(left.Range.End, right.Range.End)
	})
}

func primaryInlineOwnerObservations(source []byte, block inlineBlock, owners []inlineSpan, composites []compositeInline) []parser.Node {
	exclusions := activeCompositeSyntaxExclusions(block, composites)
	count := 0
	for _, owner := range owners {
		if primaryInlineOwnerObservable(owner, exclusions) {
			count++
		}
	}
	if count == 0 {
		return nil
	}
	nodes := make([]parser.Node, 0, count)
	for _, owner := range owners {
		if !primaryInlineOwnerObservable(owner, exclusions) {
			continue
		}
		if node, ok := primaryInlineOwnerObservation(source, owner); ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func activeCompositeSyntaxExclusions(block inlineBlock, composites []compositeInline) [][]parser.Range {
	for _, composite := range composites {
		if composite.active {
			return inlineOwnerExclusions(block, nil, composites)
		}
	}
	return nil
}

func primaryInlineOwnerObservable(owner inlineSpan, exclusions [][]parser.Range) bool {
	switch owner.kind {
	case parser.KindCodeSpan, parser.KindRawHTML, parser.KindAutoLink:
	default:
		return false
	}
	if owner.segment < 0 || owner.segment >= len(exclusions) && exclusions != nil {
		return false
	}
	return exclusions == nil || !inlineRangesContainPosition(exclusions[owner.segment], owner.start)
}

func primaryInlineOwnerObservation(source []byte, owner inlineSpan) (parser.Node, bool) {
	if !owner.content.Valid(len(source)) || owner.content.Start >= owner.content.End {
		return parser.Node{}, false
	}
	switch owner.kind {
	case parser.KindCodeSpan:
		return parser.Node{Kind: parser.KindCodeSpan, Range: owner.content, Anchor: owner.start}, true
	case parser.KindRawHTML:
		return parser.Node{Kind: parser.KindRawHTML, Range: owner.content}, true
	case parser.KindAutoLink:
		return parser.Node{
			Kind:          parser.KindAutoLink,
			Range:         owner.content,
			Anchor:        owner.start,
			Value:         string(source[owner.content.Start:owner.content.End]),
			AutoLinkEmail: owner.autoLinkEmail,
		}, true
	default:
		return parser.Node{}, false
	}
}

func resolvePrimaryInlineOwners(source []byte, block inlineBlock, runs []backtickRun, nextSame []int, spans []inlineSpan) ([]inlineSpan, []backtickRun) {
	if len(runs) == 0 {
		return spans, nil
	}
	owners := make([]inlineSpan, 0, len(spans)+len(runs)/2)
	barriers := make([]backtickRun, 0)
	for runIndex, spanIndex := 0, 0; runIndex < len(runs) || spanIndex < len(spans); {
		if nextSpanBeforeRun(spans, spanIndex, runs, runIndex) {
			span := spans[spanIndex]
			owners = append(owners, span)
			runIndex = skipRunsBefore(runs, runIndex, span.endSegment, span.end)
			spanIndex++
			continue
		}

		opener := runs[runIndex]
		segmentStart := block.segments[opener.segment].Start
		if inlineByteEscaped(source, segmentStart, opener.start) {
			runIndex++
			continue
		}
		if nextSame[runIndex] < 0 {
			barriers = append(barriers, opener)
			runIndex++
			continue
		}
		closeIndex := nextSame[runIndex]
		closer := runs[closeIndex]
		owner := inlineSpan{segment: opener.segment, start: opener.start, endSegment: closer.segment, end: closer.end}
		if node, ok := codeSpanObservation(source, block, opener, closer); ok {
			owner.kind = node.Kind
			owner.content = node.Range
		}
		owners = append(owners, owner)
		spanIndex = skipSpansBefore(spans, spanIndex, closer.segment, closer.end)
		runIndex = closeIndex + 1
	}
	return owners, barriers
}

func nextSpanBeforeRun(spans []inlineSpan, spanIndex int, runs []backtickRun, runIndex int) bool {
	if spanIndex >= len(spans) {
		return false
	}
	if runIndex >= len(runs) {
		return true
	}
	return inlinePositionBefore(spans[spanIndex].segment, spans[spanIndex].start, runs[runIndex].segment, runs[runIndex].start)
}

func inlinePositionBefore(leftSegment, leftOffset, rightSegment, rightOffset int) bool {
	return leftSegment < rightSegment || leftSegment == rightSegment && leftOffset < rightOffset
}

func skipRunsBefore(runs []backtickRun, index, segment, end int) int {
	for index < len(runs) && (runs[index].segment < segment || runs[index].segment == segment && runs[index].start < end) {
		index++
	}
	return index
}

func skipSpansBefore(spans []inlineSpan, index, segment, end int) int {
	for index < len(spans) && (spans[index].segment < segment || spans[index].segment == segment && spans[index].start < end) {
		index++
	}
	return index
}

func collectBacktickRuns(source []byte, block inlineBlock) []backtickRun {
	runs := make([]backtickRun, 0)
	for segmentIndex, segment := range block.segments {
		for position := segment.Start; position < segment.End; {
			if source[position] != '`' {
				position++
				continue
			}
			start := position
			for position < segment.End && source[position] == '`' {
				position++
			}
			runs = append(runs, backtickRun{
				segment: segmentIndex,
				start:   start,
				end:     position,
				length:  position - start,
			})
		}
	}
	return runs
}

func nextSameLengthRuns(runs []backtickRun) []int {
	next := make([]int, len(runs))
	for index := range next {
		next[index] = -1
	}
	last := make(map[int]int)
	for index := len(runs) - 1; index >= 0; index-- {
		if nextIndex, ok := last[runs[index].length]; ok {
			next[index] = nextIndex
		}
		last[runs[index].length] = index
	}
	return next
}

func codeSpanObservation(source []byte, block inlineBlock, opener, closer backtickRun) (parser.Node, bool) {
	if opener.segment != closer.segment {
		return multilineCodeSpanObservation(source, block, opener, closer)
	}
	start, end := opener.end, closer.start
	if start >= end || block.tableCell && codeSpanContainsTableEscapedPipe(source, start, end) {
		return parser.Node{}, false
	}
	if source[start] == ' ' && source[end-1] == ' ' && !onlyASCIISpaces(source[start:end]) {
		start++
		end--
	}
	if start >= end {
		return parser.Node{}, false
	}
	return parser.Node{
		Kind:   parser.KindCodeSpan,
		Range:  parser.Range{Start: start, End: end},
		Anchor: opener.start,
	}, true
}

func multilineCodeSpanObservation(source []byte, block inlineBlock, opener, closer backtickRun) (parser.Node, bool) {
	if block.tableCell || closer.segment != opener.segment+1 || opener.segment < 0 || closer.segment >= len(block.segments) {
		return parser.Node{}, false
	}
	first := block.segments[opener.segment]
	second := block.segments[closer.segment]
	if closer.start != second.Start || opener.end >= first.End || source[opener.end] != ' ' || !projectableMultilineCodeSpanSeparator(source, first, second) {
		return parser.Node{}, false
	}
	if len(bytes.Trim(source[opener.end:first.End], " \t")) == 0 {
		return parser.Node{}, false
	}
	content := parser.Range{Start: opener.end + 1, End: first.End}
	if content.Start >= content.End {
		return parser.Node{}, false
	}
	return parser.Node{Kind: parser.KindCodeSpan, Range: content, Anchor: opener.start}, true
}

func projectableMultilineCodeSpanSeparator(source []byte, first, second parser.Range) bool {
	if first.End < 0 || first.End >= len(source) || second.Start <= first.End {
		return false
	}
	switch source[first.End] {
	case '\n':
		return second.Start == first.End+1
	case '\r':
		return second.Start == first.End+1
	default:
		return false
	}
}

func collectInlineSpans(source []byte, block inlineBlock) []inlineSpan {
	spans := make([]inlineSpan, 0)
	if len(block.segments) == 0 {
		return spans
	}
	segmentIndex := 0
	position := block.segments[0].Start
	for segmentIndex < len(block.segments) {
		segment := block.segments[segmentIndex]
		if position >= segment.End {
			segmentIndex++
			if segmentIndex < len(block.segments) {
				position = block.segments[segmentIndex].Start
			}
			continue
		}
		if source[position] == '<' && !inlineByteEscaped(source, segment.Start, position) {
			if node, end, ok := scanAngleAutolink(source, position, segment.End); ok {
				spans = append(spans, observedInlineSpan(segmentIndex, position, end, node))
				position = end
				continue
			}
			if end, ok := scanInlineHTML(source, position, segment.End); ok {
				spans = append(spans, inlineSpan{
					segment: segmentIndex, start: position, endSegment: segmentIndex, end: end,
					kind: parser.KindRawHTML, content: parser.Range{Start: position, End: end},
				})
				position = end
				continue
			}
			if endSegment, end, ok := scanMultilineHTMLProcessingInstruction(source, block, segmentIndex, position); ok {
				spans = append(spans, inlineSpan{segment: segmentIndex, start: position, endSegment: endSegment, end: end})
				segmentIndex = endSegment
				position = end
				continue
			}
			if endSegment, end, ok := scanMultilineHTMLDeclaration(source, block, segmentIndex, position); ok {
				spans = append(spans, inlineSpan{segment: segmentIndex, start: position, endSegment: endSegment, end: end})
				segmentIndex = endSegment
				position = end
				continue
			}
			if endSegment, end, ok := scanMultilineHTMLTag(source, block, segmentIndex, position); ok {
				spans = append(spans, inlineSpan{segment: segmentIndex, start: position, endSegment: endSegment, end: end})
				segmentIndex = endSegment
				position = end
				continue
			}
		}
		if asciiAlphaNumeric(source[position]) && extendedAutolinkBoundary(source, segment.Start, position) {
			if node, end, ok := scanExtendedAutolink(source, position, segment.Start, segment.End); ok {
				spans = append(spans, observedInlineSpan(segmentIndex, position, end, node))
				position = end
				continue
			}
		}
		position++
	}
	return spans
}

func observedInlineSpan(segment, start, end int, node parser.Node) inlineSpan {
	return inlineSpan{
		segment:       segment,
		start:         start,
		endSegment:    segment,
		end:           end,
		kind:          node.Kind,
		content:       node.Range,
		autoLinkEmail: node.AutoLinkEmail,
	}
}

func scanExtendedAutolink(source []byte, start, segmentStart, limit int) (parser.Node, int, bool) {
	if !extendedAutolinkBoundary(source, segmentStart, start) {
		return parser.Node{}, start, false
	}
	if end, ok := scanProtocolAutolink(source, start, limit); ok {
		return bareAutolinkObservation(source, start, end, false), end, true
	}
	if end, ok := scanExtendedURLAutolink(source, start, limit); ok {
		return bareAutolinkObservation(source, start, end, false), end, true
	}
	if end, ok := scanExtendedEmailAutolink(source, start, limit); ok {
		return bareAutolinkObservation(source, start, end, true), end, true
	}
	return parser.Node{}, start, false
}

func bareAutolinkObservation(source []byte, start, end int, email bool) parser.Node {
	return parser.Node{
		Kind:          parser.KindAutoLink,
		Range:         parser.Range{Start: start, End: end},
		Anchor:        start,
		Value:         string(source[start:end]),
		AutoLinkEmail: email,
	}
}

func extendedAutolinkBoundary(source []byte, segmentStart, start int) bool {
	if start == segmentStart {
		return true
	}
	switch source[start-1] {
	case ' ', '\t', '*', '_', '~', '(':
		return true
	default:
		return false
	}
}

func scanExtendedURLAutolink(source []byte, start, limit int) (int, bool) {
	domainStart := start
	switch {
	case hasPrefixAt(source, start, limit, "www."):
		domainStart += len("www.")
	case hasPrefixAt(source, start, limit, "http://"):
		domainStart += len("http://")
	case hasPrefixAt(source, start, limit, "https://"):
		domainStart += len("https://")
	default:
		return start, false
	}
	domainEnd, domainTruncated := scanExtendedDomainEnd(source, domainStart, limit)
	if domainEnd <= domainStart || !validExtendedDomain(source[domainStart:domainEnd]) {
		return start, false
	}
	if domainTruncated {
		return domainEnd, true
	}
	end := domainEnd
	for end < limit && !extendedAutolinkTerminator(source[end]) {
		end++
	}
	end = trimExtendedAutolinkPath(source, start, end)
	if end < domainEnd {
		return start, false
	}
	return end, true
}

func scanExtendedDomainEnd(source []byte, start, limit int) (int, bool) {
	rawEnd := start
	for rawEnd < limit && extendedDomainByte(source[rawEnd]) {
		rawEnd++
	}
	domainEnd := rawEnd
	for domainEnd > start && source[domainEnd-1] == '.' {
		domainEnd--
	}
	return domainEnd, domainEnd != rawEnd
}

func validExtendedDomain(domain []byte) bool {
	if len(domain) == 0 || bytes.IndexByte(domain, '.') < 0 {
		return false
	}
	labels := bytes.Split(domain, []byte{'.'})
	for _, label := range labels {
		if !validExtendedDomainLabel(label) {
			return false
		}
	}
	return validExtendedDomainTail(labels)
}

func validExtendedDomainLabel(label []byte) bool {
	if len(label) == 0 {
		return false
	}
	for _, b := range label {
		if !asciiLetter(b) && !asciiDigit(b) && b != '_' && b != '-' {
			return false
		}
	}
	return true
}

func validExtendedDomainTail(labels [][]byte) bool {
	lastTwo := labels
	if len(lastTwo) > 2 {
		lastTwo = lastTwo[len(lastTwo)-2:]
	}
	for _, label := range lastTwo {
		if bytes.IndexByte(label, '_') >= 0 {
			return false
		}
	}
	return true
}

func extendedDomainByte(b byte) bool {
	return asciiLetter(b) || asciiDigit(b) || b == '_' || b == '-' || b == '.'
}

func extendedAutolinkTerminator(b byte) bool {
	return b == '<' || b == ' ' || b == '\t'
}

func trimExtendedAutolinkPath(source []byte, start, end int) int {
	end = trimExtendedTrailingPunctuation(source, start, end)
	end = trimExtendedUnbalancedParentheses(source, start, end)
	return trimExtendedEntitySuffix(source, start, end)
}

func trimExtendedTrailingPunctuation(source []byte, start, end int) int {
	for end > start && extendedTrailingPunctuation(source[end-1]) {
		end--
	}
	return end
}

func trimExtendedUnbalancedParentheses(source []byte, start, end int) int {
	if end <= start || source[end-1] != ')' {
		return end
	}
	open, close := 0, 0
	for _, b := range source[start:end] {
		switch b {
		case '(':
			open++
		case ')':
			close++
		}
	}
	for close > open && end > start && source[end-1] == ')' {
		end--
		close--
	}
	return end
}

func trimExtendedEntitySuffix(source []byte, start, end int) int {
	if end <= start || source[end-1] != ';' {
		return end
	}
	nameStart := end - 1
	for nameStart > start && asciiAlphaNumeric(source[nameStart-1]) {
		nameStart--
	}
	if nameStart > start && source[nameStart-1] == '&' && nameStart < end-1 {
		return nameStart - 1
	}
	return end
}

func extendedTrailingPunctuation(b byte) bool {
	switch b {
	case '?', '!', '.', ',', ':', '*', '_', '~':
		return true
	default:
		return false
	}
}

func scanExtendedEmailAutolink(source []byte, start, limit int) (int, bool) {
	if start >= limit || !asciiAlphaNumeric(source[start]) {
		return start, false
	}
	position := start
	for position < limit && emailLocalByte(source[position]) {
		position++
	}
	if position == start || position >= limit || source[position] != '@' {
		return start, false
	}
	domainStart := position + 1
	position = domainStart
	for position < limit && extendedEmailDomainByte(source[position]) {
		position++
	}
	domainEnd := position
	for domainEnd > domainStart && source[domainEnd-1] == '.' {
		domainEnd--
	}
	if !validExtendedEmailDomain(source[domainStart:domainEnd]) {
		return start, false
	}
	return domainEnd, true
}

func scanEmailAutolink(source []byte, start, limit int, localByte func(byte) bool) (int, bool) {
	position := start
	for position < limit && localByte(source[position]) {
		position++
	}
	if position == start || position >= limit || source[position] != '@' {
		return start, false
	}
	domainStart := position + 1
	position = domainStart
	for position < limit && extendedEmailDomainByte(source[position]) {
		position++
	}
	domainEnd := position
	for domainEnd > domainStart && source[domainEnd-1] == '.' {
		domainEnd--
	}
	if !validExtendedEmailDomain(source[domainStart:domainEnd]) {
		return start, false
	}
	if position > domainEnd && source[position-1] != '.' {
		return start, false
	}
	return domainEnd, true
}

func validExtendedEmailDomain(domain []byte) bool {
	if len(domain) == 0 || bytes.IndexByte(domain, '.') < 0 || domain[len(domain)-1] == '-' || domain[len(domain)-1] == '_' {
		return false
	}
	labelStart := 0
	for position := 0; position <= len(domain); position++ {
		if position < len(domain) && domain[position] != '.' {
			continue
		}
		if position == labelStart {
			return false
		}
		labelStart = position + 1
	}
	return true
}

func gfmProtocolEmailLocalByte(b byte) bool {
	return asciiLetter(b) || asciiDigit(b) || b == '.' || b == '-' || b == '_' || b == '+'
}

func extendedEmailDomainByte(b byte) bool {
	return asciiLetter(b) || asciiDigit(b) || b == '.' || b == '-' || b == '_'
}

func scanProtocolAutolink(source []byte, start, limit int) (int, bool) {
	scheme := ""
	switch {
	case hasPrefixAt(source, start, limit, "mailto:"):
		scheme = "mailto"
	case hasPrefixAt(source, start, limit, "xmpp:"):
		scheme = "xmpp"
	default:
		return start, false
	}
	addressStart := start + len(scheme) + 1
	addressEnd, ok := scanEmailAutolink(source, addressStart, limit, gfmProtocolEmailLocalByte)
	if !ok {
		return start, false
	}
	end := addressEnd
	if scheme == "xmpp" && end < limit && source[end] == '/' {
		resourceEnd := end + 1
		for resourceEnd < limit && xmppResourceByte(source[resourceEnd]) {
			resourceEnd++
		}
		if resourceEnd > end+1 {
			end = resourceEnd
		}
	}
	return end, true
}

func xmppResourceByte(b byte) bool {
	return asciiLetter(b) || asciiDigit(b) || b == '@' || b == '.'
}

func hasPrefixAt(source []byte, start, limit int, prefix string) bool {
	return start >= 0 && start+len(prefix) <= limit && bytes.Equal(source[start:start+len(prefix)], []byte(prefix))
}

func asciiAlphaNumeric(b byte) bool {
	return asciiLetter(b) || asciiDigit(b)
}

func scanAngleAutolink(source []byte, start, limit int) (parser.Node, int, bool) {
	if start+2 >= limit || source[start] != '<' {
		return parser.Node{}, start, false
	}
	relative := bytes.IndexByte(source[start+1:limit], '>')
	if relative < 0 {
		return parser.Node{}, start, false
	}
	end := start + 1 + relative
	value := source[start+1 : end]
	if validURIAutolink(value) {
		return parser.Node{
			Kind:   parser.KindAutoLink,
			Range:  parser.Range{Start: start + 1, End: end},
			Anchor: start,
			Value:  string(value),
		}, end + 1, true
	}
	if validEmailAutolink(value) {
		return parser.Node{
			Kind:          parser.KindAutoLink,
			Range:         parser.Range{Start: start + 1, End: end},
			Anchor:        start,
			Value:         string(value),
			AutoLinkEmail: true,
		}, end + 1, true
	}
	return parser.Node{}, start, false
}

func validURIAutolink(value []byte) bool {
	colon := bytes.IndexByte(value, ':')
	if colon < 2 || colon > 32 || !asciiLetter(value[0]) {
		return false
	}
	for _, b := range value[1:colon] {
		if !asciiLetter(b) && !asciiDigit(b) && b != '+' && b != '.' && b != '-' {
			return false
		}
	}
	for _, b := range value[colon+1:] {
		if b <= ' ' || b == 0x7f || b == '<' || b == '>' {
			return false
		}
	}
	return true
}

func validEmailAutolink(value []byte) bool {
	at := bytes.LastIndexByte(value, '@')
	if at <= 0 || at == len(value)-1 || bytes.IndexByte(value[:at], '@') >= 0 {
		return false
	}
	for _, b := range value[:at] {
		if !emailLocalByte(b) {
			return false
		}
	}
	return validEmailDomain(value[at+1:])
}

func emailLocalByte(b byte) bool {
	if asciiLetter(b) || asciiDigit(b) {
		return true
	}
	switch b {
	case '.', '!', '#', '$', '%', '&', '\'', '*', '+', '/', '=', '?', '^', '_', '`', '{', '|', '}', '~', '-':
		return true
	default:
		return false
	}
}

func validEmailDomain(domain []byte) bool {
	if len(domain) == 0 {
		return false
	}
	labelStart := 0
	for position := 0; position <= len(domain); position++ {
		if position < len(domain) && domain[position] != '.' {
			continue
		}
		label := domain[labelStart:position]
		if !validEmailDomainLabel(label) {
			return false
		}
		labelStart = position + 1
	}
	return true
}

func validEmailDomainLabel(label []byte) bool {
	if len(label) == 0 || len(label) > 63 || !asciiLetter(label[0]) && !asciiDigit(label[0]) || !asciiLetter(label[len(label)-1]) && !asciiDigit(label[len(label)-1]) {
		return false
	}
	for position := 1; position+1 < len(label); position++ {
		b := label[position]
		if !asciiLetter(b) && !asciiDigit(b) && b != '-' {
			return false
		}
	}
	return true
}

type inlineHTMLCursor struct {
	block    inlineBlock
	segment  int
	position int
}

func scanMultilineHTMLProcessingInstruction(source []byte, block inlineBlock, startSegment, start int) (int, int, bool) {
	if startSegment < 0 || startSegment >= len(block.segments) {
		return startSegment, start, false
	}
	segment := block.segments[startSegment]
	if start+1 >= segment.End || source[start] != '<' || source[start+1] != '?' {
		return startSegment, start, false
	}
	cursor := inlineHTMLCursor{block: block, segment: startSegment, position: start + 2}
	previousQuestion := false
	for {
		value, ok := cursor.peek(source)
		if !ok {
			return startSegment, start, false
		}
		cursor.advance()
		if previousQuestion && value == '>' {
			return multilineHTMLEnd(cursor, startSegment, start)
		}
		previousQuestion = value == '?'
	}
}

func scanMultilineHTMLDeclaration(source []byte, block inlineBlock, startSegment, start int) (int, int, bool) {
	if startSegment < 0 || startSegment >= len(block.segments) {
		return startSegment, start, false
	}
	segment := block.segments[startSegment]
	if start+2 >= segment.End || source[start] != '<' || source[start+1] != '!' || !asciiLetter(source[start+2]) {
		return startSegment, start, false
	}
	cursor := inlineHTMLCursor{block: block, segment: startSegment, position: start + 3}
	for {
		value, ok := cursor.peek(source)
		if !ok {
			return startSegment, start, false
		}
		cursor.advance()
		if value == '>' {
			return multilineHTMLEnd(cursor, startSegment, start)
		}
	}
}

func scanMultilineHTMLTag(source []byte, block inlineBlock, startSegment, start int) (int, int, bool) {
	segment := block.segments[startSegment]
	position, closing, ok := scanMultilineHTMLTagName(source[start:segment.End])
	if !ok {
		return startSegment, start, false
	}
	cursor := inlineHTMLCursor{block: block, segment: startSegment, position: start + position}
	if closing {
		skipInlineHTMLWhitespace(source, &cursor)
		return finishMultilineHTMLTag(source, cursor, startSegment, start)
	}
	for {
		if endSegment, end, ok := finishMultilineHTMLTag(source, cursor, startSegment, start); ok {
			return endSegment, end, true
		}
		if !skipInlineHTMLWhitespace(source, &cursor) {
			return startSegment, start, false
		}
		if endSegment, end, ok := finishMultilineHTMLTag(source, cursor, startSegment, start); ok {
			return endSegment, end, true
		}
		if !scanInlineHTMLCursorAttribute(source, &cursor) {
			return startSegment, start, false
		}
	}
}

func scanMultilineHTMLTagName(text []byte) (int, bool, bool) {
	if len(text) < 2 || text[0] != '<' {
		return 0, false, false
	}
	position := 1
	closing := text[position] == '/'
	if closing {
		position++
	}
	if position >= len(text) || !asciiLetter(text[position]) {
		return 0, false, false
	}
	position++
	for position < len(text) && (asciiLetter(text[position]) || asciiDigit(text[position]) || text[position] == '-') {
		position++
	}
	return position, closing, true
}

func finishMultilineHTMLTag(source []byte, cursor inlineHTMLCursor, startSegment, start int) (int, int, bool) {
	value, ok := cursor.peek(source)
	if !ok {
		return startSegment, start, false
	}
	if value == '>' {
		cursor.advance()
		return multilineHTMLEnd(cursor, startSegment, start)
	}
	if value != '/' {
		return startSegment, start, false
	}
	cursor.advance()
	value, ok = cursor.peek(source)
	if !ok || value != '>' {
		return startSegment, start, false
	}
	cursor.advance()
	return multilineHTMLEnd(cursor, startSegment, start)
}

func multilineHTMLEnd(cursor inlineHTMLCursor, startSegment, start int) (int, int, bool) {
	if cursor.segment <= startSegment {
		return startSegment, start, false
	}
	return cursor.segment, cursor.position, true
}

func scanInlineHTMLCursorAttribute(source []byte, cursor *inlineHTMLCursor) bool {
	value, ok := cursor.peek(source)
	if !ok || !htmlAttributeNameStart(value) {
		return false
	}
	cursor.advance()
	for {
		value, ok = cursor.peek(source)
		if !ok || !htmlAttributeNameContinue(value) {
			break
		}
		cursor.advance()
	}
	beforeWhitespace := *cursor
	skipInlineHTMLWhitespace(source, cursor)
	value, ok = cursor.peek(source)
	if !ok || value != '=' {
		*cursor = beforeWhitespace
		return true
	}
	cursor.advance()
	skipInlineHTMLWhitespace(source, cursor)
	return scanInlineHTMLCursorAttributeValue(source, cursor)
}

func scanInlineHTMLCursorAttributeValue(source []byte, cursor *inlineHTMLCursor) bool {
	value, ok := cursor.peek(source)
	if !ok {
		return false
	}
	if value == '\'' || value == '"' {
		quote := value
		cursor.advance()
		for {
			value, ok = cursor.peek(source)
			if !ok {
				return false
			}
			cursor.advance()
			if value == quote {
				return true
			}
		}
	}
	consumed := false
	for ok && !inlineHTMLWhitespace(value) && htmlUnquotedAttributeValueByte(value) {
		consumed = true
		cursor.advance()
		value, ok = cursor.peek(source)
	}
	return consumed
}

func skipInlineHTMLWhitespace(source []byte, cursor *inlineHTMLCursor) bool {
	consumed := false
	for {
		value, ok := cursor.peek(source)
		if !ok || !inlineHTMLWhitespace(value) {
			return consumed
		}
		consumed = true
		cursor.advance()
	}
}

func inlineHTMLWhitespace(value byte) bool {
	return htmlWhitespace(value) || value == '\n' || value == '\r'
}

func (cursor inlineHTMLCursor) peek(source []byte) (byte, bool) {
	if cursor.segment >= len(cursor.block.segments) {
		return 0, false
	}
	segment := cursor.block.segments[cursor.segment]
	if cursor.position < segment.End {
		return source[cursor.position], true
	}
	if cursor.segment+1 < len(cursor.block.segments) {
		return '\n', true
	}
	return 0, false
}

func (cursor *inlineHTMLCursor) advance() {
	if cursor.segment >= len(cursor.block.segments) {
		return
	}
	segment := cursor.block.segments[cursor.segment]
	if cursor.position < segment.End {
		cursor.position++
		return
	}
	cursor.segment++
	if cursor.segment < len(cursor.block.segments) {
		cursor.position = cursor.block.segments[cursor.segment].Start
	}
}

func scanInlineHTML(source []byte, start, limit int) (int, bool) {
	text := source[start:limit]
	switch {
	case bytes.HasPrefix(text, []byte("<!--")):
		return scanInlineHTMLComment(source, start, limit)
	case bytes.HasPrefix(text, []byte("<?")):
		return scanInlineHTMLUntil(source, start+2, limit, []byte("?>"))
	case bytes.HasPrefix(text, []byte("<![CDATA[")):
		return scanInlineHTMLUntil(source, start+9, limit, []byte("]]>"))
	case len(text) >= 3 && text[0] == '<' && text[1] == '!' && asciiLetter(text[2]):
		return scanInlineHTMLDeclaration(source, start, limit)
	default:
		return scanInlineHTMLTag(source, start, limit)
	}
}

func scanInlineHTMLComment(source []byte, start, limit int) (int, bool) {
	text := source[start:limit]
	if bytes.HasPrefix(text, []byte("<!-->")) {
		return start + len("<!-->"), true
	}
	if bytes.HasPrefix(text, []byte("<!--->")) {
		return start + len("<!--->"), true
	}
	bodyStart := start + len("<!--")
	relative := bytes.Index(source[bodyStart:limit], []byte("-->"))
	if relative < 0 {
		return start, false
	}
	return bodyStart + relative + len("-->"), true
}

func scanInlineHTMLUntil(source []byte, searchStart, limit int, closing []byte) (int, bool) {
	relative := bytes.Index(source[searchStart:limit], closing)
	if relative < 0 {
		return searchStart, false
	}
	return searchStart + relative + len(closing), true
}

func scanInlineHTMLDeclaration(source []byte, start, limit int) (int, bool) {
	if start+2 >= limit || source[start] != '<' || source[start+1] != '!' || !asciiLetter(source[start+2]) {
		return start, false
	}
	position := start + 3
	for position < limit && source[position] != '>' {
		position++
	}
	if position >= limit {
		return start, false
	}
	return position + 1, true
}

func scanInlineHTMLTag(source []byte, start, limit int) (int, bool) {
	text := source[start:limit]
	_, position, closing, ok := scanCompleteHTMLTagName(text)
	if !ok {
		return start, false
	}
	if closing {
		end, ok := scanInlineHTMLClosingTagEnd(text, position)
		return start + end, ok
	}
	end, ok := scanInlineHTMLOpenTagEnd(text, position)
	return start + end, ok
}

func scanInlineHTMLClosingTagEnd(text []byte, position int) (int, bool) {
	position = skipHTMLWhitespace(text, position)
	if position < len(text) && text[position] == '>' {
		return position + 1, true
	}
	return 0, false
}

func scanInlineHTMLOpenTagEnd(text []byte, position int) (int, bool) {
	for position < len(text) {
		if end, ok := scanInlineHTMLTagClose(text, position); ok {
			return end, true
		}
		if !htmlWhitespace(text[position]) {
			return 0, false
		}
		position = skipHTMLWhitespace(text, position)
		if end, ok := scanInlineHTMLTagClose(text, position); ok {
			return end, true
		}
		next, ok := scanHTMLAttribute(text, position)
		if !ok {
			return 0, false
		}
		position = next
	}
	return 0, false
}

func scanInlineHTMLTagClose(text []byte, position int) (int, bool) {
	if position < len(text) && text[position] == '>' {
		return position + 1, true
	}
	if position+1 < len(text) && text[position] == '/' && text[position+1] == '>' {
		return position + 2, true
	}
	return 0, false
}

func codeSpanContainsTableEscapedPipe(source []byte, start, end int) bool {
	for position := start; position < end; position++ {
		if source[position] == '|' && tablePipeEscaped(source, start, position) {
			return true
		}
	}
	return false
}

func onlyASCIISpaces(value []byte) bool {
	for _, b := range value {
		if b != ' ' {
			return false
		}
	}
	return true
}

func inlineByteEscaped(source []byte, segmentStart, position int) bool {
	backslashes := 0
	for position > segmentStart && source[position-1] == '\\' {
		backslashes++
		position--
	}
	return backslashes%2 != 0
}
