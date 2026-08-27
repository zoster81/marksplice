// Package native implements Marksplice's native CommonMark/GFM parser.
package native

import (
	"bytes"
	"strings"

	"github.com/zoster81/marksplice/internal/parser"
)

type rootBlockKind uint8

const (
	rootBlockOther rootBlockKind = iota
	rootBlockParagraph
	rootBlockReference
	rootBlockList
)

type rootBlock struct {
	kind                   rootBlockKind
	range_                 parser.Range
	lineCount              int
	itemCount              int
	nodeIndex              int
	hasLineAnchor          bool
	lineAnchor             int
	hasListContainerAnchor bool
	listContainerAnchor    int
}

type blockParseResult struct {
	nodes             []parser.Node
	semantic          []parser.Range
	inlines           []inlineBlock
	references        []referenceDefinitionParse
	roots             []rootBlock
	blankBetweenRoots bool
	trailingBlank     bool
	lastLeafParagraph bool
}

type fenceOpening struct {
	anchor int
	indent int
	marker byte
	length int
	info   string
}

type blockquoteParseResult struct {
	node       parser.Node
	content    []physicalLine
	child      blockParseResult
	childReady bool
	next       int
}

// ParseBlocks returns the parser-independent block observations owned by the
// native block parser. Inline-derived observations are added by the native
// inline parser in the following parser milestone.
func ParseBlocks(source []byte) ([]parser.Node, error) {
	result := parseBlockLines(source, physicalLines(source), true)
	return result.nodes, nil
}

func parseBlockLines(source []byte, lines []physicalLine, topLevel bool) blockParseResult {
	result := blockParseResult{nodes: make([]parser.Node, 0, len(lines)/2)}
	blankBeforeRoot := false
	referenceParagraphOpen := false
	for index := 0; index < len(lines); {
		line := lines[index]
		if blankLine(source, line) {
			result.lastLeafParagraph = false
			referenceParagraphOpen = false
			if len(result.roots) != 0 {
				blankBeforeRoot = true
			}
			index++
			continue
		}
		if referenceParagraphOpen && referenceDefinitionParagraphContinuation(source, lines, index) {
			index = appendParagraphBlock(&result, source, lines, index, topLevel, blankBeforeRoot)
			blankBeforeRoot = false
			referenceParagraphOpen = false
			continue
		}
		referenceParagraphOpen = false
		if next, handled, reference := parseLeafBlock(&result, source, lines, index, topLevel, blankBeforeRoot); handled {
			index = next
			blankBeforeRoot = false
			referenceParagraphOpen = reference
			continue
		}
		if next, blankAfter, handled := parseStructuralBlock(&result, source, lines, index, topLevel, blankBeforeRoot); handled {
			index = next
			blankBeforeRoot = blankAfter
			continue
		}
		index = appendParagraphBlock(&result, source, lines, index, topLevel, blankBeforeRoot)
		blankBeforeRoot = false
	}
	return result
}

func referenceDefinitionParagraphContinuation(source []byte, lines []physicalLine, index int) bool {
	line := lines[index]
	if !referenceParagraphContinuationLine(source, line) {
		return false
	}
	_, ok := scanReferenceDefinition(source, lines, index)
	return !ok
}

func referenceParagraphContinuationLine(source []byte, line physicalLine) bool {
	if blankLine(source, line) || startsBlockquote(source, line) || interruptsParagraph(source, line) {
		return false
	}
	_, setext := parseSetextUnderline(source, line)
	return !setext
}

func appendParagraphBlock(result *blockParseResult, source []byte, lines []physicalLine, index int, topLevel, blankBeforeRoot bool) int {
	line := lines[index]
	node, semantic, next := parseParagraphOrSetext(source, lines, index)
	node.TopLevel = topLevel
	inlineLast := next
	if node.Kind == parser.KindHeading {
		inlineLast--
	}
	appendInlineLines(result, source, lines, index, inlineLast)
	if node.Kind == parser.KindParagraph {
		nodeIndex := len(result.nodes)
		result.nodes = append(result.nodes, node)
		recordRoot(result, rootBlock{kind: rootBlockParagraph, range_: node.Range, lineCount: len(semantic), nodeIndex: nodeIndex, hasLineAnchor: true, lineAnchor: line.physicalStart}, blankBeforeRoot)
	} else {
		if topLevel {
			result.nodes = append(result.nodes, node)
		}
		recordRoot(result, rootBlock{kind: rootBlockOther, hasLineAnchor: true, lineAnchor: line.physicalStart}, blankBeforeRoot)
	}
	result.semantic = append(result.semantic, semantic...)
	result.lastLeafParagraph = node.Kind == parser.KindParagraph
	return next
}

func parseLeafBlock(result *blockParseResult, source []byte, lines []physicalLine, index int, topLevel, blankBeforeRoot bool) (int, bool, bool) {
	line := lines[index]
	if _, _, ok := parseBlockquoteOpening(source, line); ok {
		quoted := parseBlockquote(source, lines, index)
		child := quoted.child
		if !quoted.childReady {
			child = parseBlockLines(source, quoted.content, false)
		}
		if topLevel {
			quoted.node.BlockquoteSemanticRanges = child.semantic
			if simpleBlockquoteParagraph(child) {
				quoted.node.BlockquoteContentRange = child.nodes[0].Range
			}
			result.nodes = append(result.nodes, quoted.node)
		}
		result.nodes = append(result.nodes, child.nodes...)
		result.semantic = append(result.semantic, child.semantic...)
		result.inlines = append(result.inlines, child.inlines...)
		result.references = append(result.references, child.references...)
		result.lastLeafParagraph = child.lastLeafParagraph
		recordRoot(result, rootBlock{kind: rootBlockOther}, blankBeforeRoot)
		return quoted.next, true, false
	}
	if indentedCodeLine(source, line) {
		semantic, next := parseIndentedCodeLines(source, lines, index)
		result.semantic = append(result.semantic, semantic...)
		result.lastLeafParagraph = false
		recordRoot(result, rootBlock{kind: rootBlockOther, hasLineAnchor: true, lineAnchor: line.physicalStart}, blankBeforeRoot)
		return next, true, false
	}
	if opening, ok := parseFenceOpening(source, line); ok {
		node, semantic, next := parseFencedBlock(source, lines, index, opening)
		node.TopLevel = topLevel
		if node.Anchor >= 0 && node.Anchor < len(source) {
			result.nodes = append(result.nodes, node)
		}
		result.semantic = append(result.semantic, semantic...)
		result.lastLeafParagraph = false
		recordRoot(result, rootBlock{kind: rootBlockOther}, blankBeforeRoot)
		return next, true, false
	}
	if opening, ok := htmlBlockStart(source, line); ok {
		node, semantic, next := parseHTMLBlock(source, lines, index, opening)
		result.nodes = append(result.nodes, node)
		result.semantic = append(result.semantic, semantic...)
		result.lastLeafParagraph = false
		recordRoot(result, rootBlock{kind: rootBlockOther, hasLineAnchor: true, lineAnchor: line.physicalStart}, blankBeforeRoot)
		return next, true, false
	}
	if node, semantic, reference, next, ok := parseReferenceDefinition(source, lines, index); ok {
		if node.Kind != parser.KindUnknown {
			result.nodes = append(result.nodes, node)
		}
		result.semantic = append(result.semantic, semantic...)
		result.references = append(result.references, reference)
		result.lastLeafParagraph = true
		recordRoot(result, rootBlock{kind: rootBlockReference, hasLineAnchor: true, lineAnchor: line.physicalStart}, blankBeforeRoot)
		return next, true, true
	}
	return index, false, false
}

func parseStructuralBlock(result *blockParseResult, source []byte, lines []physicalLine, index int, topLevel, blankBeforeRoot bool) (int, bool, bool) {
	line := lines[index]
	if node, ok := parseATXHeading(source, line); ok {
		result.lastLeafParagraph = false
		node.TopLevel = topLevel
		if node.Range.Start != node.Range.End {
			result.semantic = append(result.semantic, parser.Range{Start: node.Range.Start, End: line.end})
			appendInlineRange(result, node.Range)
			if topLevel {
				result.nodes = append(result.nodes, node)
			}
		}
		recordRoot(result, rootBlock{kind: rootBlockOther, hasLineAnchor: true, lineAnchor: line.physicalStart}, blankBeforeRoot)
		return index + 1, false, true
	}
	if list, next, ok := parseList(source, lines, index); ok {
		result.nodes = append(result.nodes, list.nodes...)
		result.semantic = append(result.semantic, list.semantic...)
		result.inlines = append(result.inlines, list.inlines...)
		result.references = append(result.references, list.references...)
		result.lastLeafParagraph = list.lastLeafParagraph
		recordRoots(result, list.roots, blankBeforeRoot)
		return next, list.trailingBlank, true
	}
	if node, ok := parseThematicBreak(source, line); ok {
		result.lastLeafParagraph = false
		node.TopLevel = topLevel
		result.nodes = append(result.nodes, node)
		recordRoot(result, rootBlock{kind: rootBlockOther}, blankBeforeRoot)
		return index + 1, false, true
	}
	if nodes, semantic, next, ok := parseTable(source, lines, index); ok {
		result.nodes = append(result.nodes, nodes...)
		result.semantic = append(result.semantic, semantic...)
		result.lastLeafParagraph = true
		appendTableInlineBlocks(result, nodes)
		recordRoot(result, rootBlock{kind: rootBlockOther, hasLineAnchor: true, lineAnchor: line.physicalStart}, blankBeforeRoot)
		return next, false, true
	}
	return index, false, false
}

func recordRoot(result *blockParseResult, root rootBlock, blankBefore bool) {
	if blankBefore && len(result.roots) != 0 {
		result.blankBetweenRoots = true
	}
	result.roots = append(result.roots, root)
}

func recordRoots(result *blockParseResult, roots []rootBlock, blankBefore bool) {
	if len(roots) == 0 {
		return
	}
	if blankBefore && len(result.roots) != 0 {
		result.blankBetweenRoots = true
	}
	result.roots = append(result.roots, roots...)
}

func parseIndentedCodeLines(source []byte, lines []physicalLine, index int) ([]parser.Range, int) {
	ranges := make([]parser.Range, 0)
	for index < len(lines) {
		line := lines[index]
		if blankLine(source, line) {
			index++
			continue
		}
		if !indentedCodeLine(source, line) {
			break
		}
		stripped := stripIndentColumns(source, line, 4)
		if stripped.start < line.end {
			ranges = append(ranges, parser.Range{Start: stripped.start, End: blockLineSemanticEnd(source, line.next)})
		}
		index++
	}
	return ranges, index
}

func blockLineSemanticEnd(source []byte, stop int) int {
	for stop < len(source) && source[stop] != '\r' && source[stop] != '\n' {
		stop++
	}
	return stop
}

func simpleBlockquoteParagraph(child blockParseResult) bool {
	return len(child.nodes) == 1 && child.nodes[0].Kind == parser.KindParagraph &&
		len(child.roots) == 1 && child.roots[0].kind == rootBlockParagraph && child.roots[0].lineCount == 1
}

func parseBlockquoteOpening(source []byte, line physicalLine) (anchor, contentStart int, ok bool) {
	indent, ordinary := ordinaryIndent(source, line)
	if !ordinary {
		return 0, 0, false
	}
	anchor = line.start + indent
	if anchor >= line.end || source[anchor] != '>' {
		return 0, 0, false
	}
	contentStart = anchor + 1
	if contentStart < line.end && (source[contentStart] == ' ' || source[contentStart] == '\t') {
		contentStart++
	}
	return anchor, contentStart, true
}

func blockquoteContentLine(source []byte, line physicalLine, anchor, contentStart int) physicalLine {
	_, indentColumns := leadingIndent(source, line)
	markerColumn := line.columnOffset + indentColumns
	virtualIndent := 0
	if contentStart > anchor+1 && source[anchor+1] == '\t' {
		virtualIndent = 4 - (markerColumn+1)%4 - 1
	}
	return advancePhysicalLineStart(source, line, contentStart, virtualIndent)
}

func parseBlockquote(source []byte, lines []physicalLine, index int) blockquoteParseResult {
	anchor, _, _ := parseBlockquoteOpening(source, lines[index])
	result := blockquoteParseResult{
		node: parser.Node{
			Kind:     parser.KindBlockquote,
			Range:    parser.Range{Start: anchor, End: lines[index].end},
			TopLevel: true,
		},
		content: make([]physicalLine, 0),
		next:    index,
	}
	allowLazy := false
	lazyKnown := false
	for result.next < len(lines) {
		line := lines[result.next]
		lineAnchor, contentStart, marked := parseBlockquoteOpening(source, line)
		if marked {
			line = blockquoteContentLine(source, line, lineAnchor, contentStart)
			result.content = append(result.content, line)
			result.childReady = false
			lazyKnown = false
			result.next++
			continue
		}
		if !lazyKnown {
			result.child = parseBlockLines(source, result.content, false)
			result.childReady = true
			allowLazy = result.child.lastLeafParagraph
			lazyKnown = true
		}
		if !allowLazy || blankLine(source, line) || !strictContainerLazyParagraphLine(source, line) {
			break
		}
		result.content = append(result.content, line)
		result.childReady = false
		result.next++
	}
	return result
}

func strictContainerLazyParagraphLine(source []byte, line physicalLine) bool {
	if blankLine(source, line) {
		return false
	}
	if _, _, ok := parseBlockquoteOpening(source, line); ok {
		return false
	}
	if _, ok := parseListMarker(source, line); ok {
		return false
	}
	return !interruptsParagraph(source, line)
}

func parseATXHeading(source []byte, line physicalLine) (parser.Node, bool) {
	level, position, ok := scanATXHeadingOpening(source, line)
	if !ok {
		return parser.Node{}, false
	}
	contentStart := trimHorizontalStart(source, position, line.end)
	contentEnd := atxContentEnd(source, contentStart, line.end)
	return parser.Node{Kind: parser.KindHeading, Range: parser.Range{Start: contentStart, End: contentEnd}, Level: level, TopLevel: true}, true
}

func scanATXHeadingOpening(source []byte, line physicalLine) (int, int, bool) {
	indent, ok := ordinaryIndent(source, line)
	if !ok {
		return 0, line.start, false
	}
	position := line.start + indent
	markerStart := position
	for position < line.end && source[position] == '#' && position-markerStart < 7 {
		position++
	}
	level := position - markerStart
	if !validATXHeadingMarker(source, line, position, level) {
		return 0, line.start, false
	}
	return level, position, true
}

func validATXHeadingMarker(source []byte, line physicalLine, position, level int) bool {
	return level >= 1 && level <= 6 && (position == line.end || source[position] == ' ' || source[position] == '\t')
}

func atxContentEnd(source []byte, start, end int) int {
	trimmed := end
	for trimmed > start && (source[trimmed-1] == ' ' || source[trimmed-1] == '\t') {
		trimmed--
	}
	hashEnd := trimmed
	for trimmed > start && source[trimmed-1] == '#' {
		trimmed--
	}
	if trimmed == hashEnd {
		return hashEnd
	}
	if trimmed == start || source[trimmed-1] == ' ' || source[trimmed-1] == '\t' {
		for trimmed > start && (source[trimmed-1] == ' ' || source[trimmed-1] == '\t') {
			trimmed--
		}
		return trimmed
	}
	return hashEnd
}

func parseSetextUnderline(source []byte, line physicalLine) (int, bool) {
	indent, ok := ordinaryIndent(source, line)
	if !ok {
		return 0, false
	}
	position := line.start + indent
	if position >= line.end || source[position] != '=' && source[position] != '-' {
		return 0, false
	}
	marker := source[position]
	count := 0
	for position < line.end && source[position] == marker {
		position++
		count++
	}
	for position < line.end && (source[position] == ' ' || source[position] == '\t') {
		position++
	}
	if count == 0 || position != line.end {
		return 0, false
	}
	if marker == '=' {
		return 1, true
	}
	return 2, true
}

func parseThematicBreak(source []byte, line physicalLine) (parser.Node, bool) {
	indent, ok := ordinaryIndent(source, line)
	if !ok {
		return parser.Node{}, false
	}
	position := line.start + indent
	if position >= line.end || source[position] != '*' && source[position] != '-' && source[position] != '_' {
		return parser.Node{}, false
	}
	marker := source[position]
	count := 0
	for position < line.end {
		switch source[position] {
		case marker:
			count++
		case ' ', '\t':
		default:
			return parser.Node{}, false
		}
		position++
	}
	if count < 3 {
		return parser.Node{}, false
	}
	start := line.start + indent
	return parser.Node{Kind: parser.KindThematicBreak, Range: parser.Range{Start: start, End: line.end}, TopLevel: true}, true
}

func parseFenceOpening(source []byte, line physicalLine) (fenceOpening, bool) {
	indent, ok := ordinaryIndent(source, line)
	if !ok {
		return fenceOpening{}, false
	}
	position := line.start + indent
	if position >= line.end || source[position] != '`' && source[position] != '~' {
		return fenceOpening{}, false
	}
	marker := source[position]
	markerStart := position
	anchor := markerStart
	for position < line.end && source[position] == marker {
		position++
	}
	length := position - markerStart
	if length < 3 {
		return fenceOpening{}, false
	}
	rawInfo := source[position:line.end]
	if line.next == line.end && len(rawInfo) == 1 {
		rawInfo = nil
	}
	if marker == '`' && bytes.Contains(rawInfo, []byte{'`'}) {
		return fenceOpening{}, false
	}
	info := strings.Trim(string(rawInfo), " \t")
	return fenceOpening{anchor: anchor, indent: indent, marker: marker, length: length, info: info}, true
}

func parseFencedBlock(source []byte, lines []physicalLine, index int, opening fenceOpening) (parser.Node, []parser.Range, int) {
	content := make([]parser.Range, 0)
	semantic := make([]parser.Range, 0)
	next := index + 1
	for next < len(lines) {
		line := lines[next]
		if fenceClosing(source, line, opening) {
			next++
			break
		}
		start := line.start
		removed := 0
		for start < line.end && removed < opening.indent && source[start] == ' ' {
			start++
			removed++
		}
		content = append(content, parser.Range{Start: start, End: line.end})
		semantic = append(semantic, parser.Range{Start: start, End: blockLineSemanticEnd(source, line.next)})
		next++
	}
	range_ := parser.Range{Start: opening.anchor, End: opening.anchor}
	if len(content) != 0 {
		range_ = parser.Range{Start: content[0].Start, End: content[len(content)-1].End}
	}
	language := opening.info
	if separator := strings.IndexByte(language, ' '); separator >= 0 {
		language = language[:separator]
	}
	return parser.Node{
		Kind:                    parser.KindFencedCode,
		Range:                   range_,
		Anchor:                  opening.anchor,
		FencedCodeContentRanges: content,
		FencedCodeInfo:          opening.info,
		FencedCodeLanguage:      language,
		TopLevel:                true,
	}, semantic, next
}

func fenceClosing(source []byte, line physicalLine, opening fenceOpening) bool {
	indent, ok := ordinaryIndent(source, line)
	if !ok {
		return false
	}
	position := line.start + indent
	start := position
	for position < line.end && source[position] == opening.marker {
		position++
	}
	if position-start < opening.length {
		return false
	}
	for position < line.end {
		if source[position] != ' ' && source[position] != '\t' {
			return false
		}
		position++
	}
	return true
}

func parseParagraphOrSetext(source []byte, lines []physicalLine, index int) (parser.Node, []parser.Range, int) {
	indent, _ := ordinaryIndent(source, lines[index])
	start := lines[index].start + indent
	end := lines[index].end
	next := index + 1
	for next < len(lines) {
		line := lines[next]
		if blankLine(source, line) {
			break
		}
		if _, _, ok := parseTableOpening(source, lines, next); ok {
			break
		}
		if level, ok := parseSetextUnderline(source, line); ok && sameBlockContainer(lines[index], line) {
			headingEnd := trimHorizontalEnd(source, lines[next-1].start, end)
			semantic := semanticLineRanges(source, lines, index, next)
			return parser.Node{Kind: parser.KindHeading, Range: parser.Range{Start: start, End: headingEnd}, Level: level, TopLevel: true}, semantic, next + 1
		}
		if interruptsParagraph(source, line) || startsBlockquote(source, line) {
			break
		}
		end = line.end
		next++
	}
	semantic := semanticLineRanges(source, lines, index, next)
	return parser.Node{Kind: parser.KindParagraph, Range: parser.Range{Start: start, End: end}, TopLevel: true}, semantic, next
}

func semanticLineRanges(source []byte, lines []physicalLine, first, last int) []parser.Range {
	ranges := make([]parser.Range, 0, last-first)
	containerizedFirst := first < last && lines[first].start != lines[first].physicalStart
	for index := first; index < last; index++ {
		line := lines[index]
		if blankLine(source, line) {
			continue
		}
		start := line.start
		if containerizedFirst && index > first && line.start == line.physicalStart {
			start = trimHorizontalStart(source, line.start, line.end)
		} else if indent, ok := ordinaryIndent(source, line); ok {
			start += indent
		}
		if start < line.end {
			ranges = append(ranges, parser.Range{Start: start, End: line.end})
		}
	}
	if containerizedFirst {
		for index := 0; index+1 < len(ranges); index++ {
			ranges[index].End = ranges[index+1].End
		}
	}
	return ranges
}

func startsBlockquote(source []byte, line physicalLine) bool {
	_, _, ok := parseBlockquoteOpening(source, line)
	return ok
}

func sameBlockContainer(first, candidate physicalLine) bool {
	return first.start != first.physicalStart == (candidate.start != candidate.physicalStart)
}

func trimHorizontalStart(source []byte, start, end int) int {
	for start < end && (source[start] == ' ' || source[start] == '\t') {
		start++
	}
	return start
}

func trimHorizontalEnd(source []byte, start, end int) int {
	for end > start && (source[end-1] == ' ' || source[end-1] == '\t') {
		end--
	}
	return end
}

func interruptsParagraph(source []byte, line physicalLine) bool {
	if _, ok := parseATXHeading(source, line); ok {
		return true
	}
	if _, ok := parseFenceOpening(source, line); ok {
		return true
	}
	if htmlBlockInterruptsParagraph(source, line) {
		return true
	}
	if _, ok := parseThematicBreak(source, line); ok {
		return true
	}
	return listInterruptsParagraph(source, line)
}
