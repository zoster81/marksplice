package native

import "github.com/zoster81/marksplice/internal/parser"

type listMarker struct {
	ordered              bool
	marker               byte
	startNumber          int
	physicalStart        int
	contentStart         int
	contentIndent        int
	contentVirtualIndent int
	blank                bool
}

type listItemSource struct {
	marker         listMarker
	lines          []physicalLine
	separatedAfter bool
	trailingBlank  bool
	child          blockParseResult
	childReady     bool
}

type listSource struct {
	items []listItemSource
	next  int
}

type parsedListItem struct {
	source listItemSource
	child  blockParseResult
}

type listItemCollector struct {
	item            listItemSource
	pendingBlank    []physicalLine
	hasContent      bool
	nestedTailEmpty bool
}

func parseListMarker(source []byte, line physicalLine) (listMarker, bool) {
	indentBytes, indentColumns := leadingIndent(source, line)
	if indentColumns > 3 || sourceIndentHasTab(source, line, indentBytes) {
		return listMarker{}, false
	}
	position := line.start + indentBytes
	if position >= line.end {
		return listMarker{}, false
	}
	markerStart := position
	marker, position, ok := scanListMarkerToken(source, line, position)
	if !ok {
		return listMarker{}, false
	}
	marker.physicalStart = line.physicalStart
	return finishListMarker(source, line, marker, markerStart, position, indentColumns)
}

func scanListMarkerToken(source []byte, line physicalLine, position int) (listMarker, int, bool) {
	switch source[position] {
	case '-', '+', '*':
		return listMarker{marker: source[position]}, position + 1, true
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return scanOrderedListMarker(source, line, position)
	default:
		return listMarker{}, position, false
	}
}

func scanOrderedListMarker(source []byte, line physicalLine, position int) (listMarker, int, bool) {
	number := 0
	digits := 0
	for position < line.end && source[position] >= '0' && source[position] <= '9' && digits < 10 {
		number = number*10 + int(source[position]-'0')
		position++
		digits++
	}
	if digits == 0 || digits > 9 || position >= line.end || source[position] != '.' && source[position] != ')' {
		return listMarker{}, position, false
	}
	marker := listMarker{ordered: true, startNumber: number, marker: source[position]}
	return marker, position + 1, true
}

func finishListMarker(source []byte, line physicalLine, marker listMarker, markerStart, position, indentColumns int) (listMarker, bool) {
	markerWidth := position - markerStart
	if position == line.end {
		if bareListMarkerBeforeCRLF(source, line) {
			return listMarker{}, false
		}
		return blankListMarker(marker, position, indentColumns, markerWidth), true
	}
	if source[position] != ' ' && source[position] != '\t' {
		return listMarker{}, false
	}
	paddingStart := position
	columns := indentColumns + markerWidth
	position, columns = scanListMarkerPadding(source, line, position, columns)
	if position == line.end {
		return blankListMarker(marker, position, indentColumns, markerWidth), true
	}
	if columns-(indentColumns+markerWidth) > 4 {
		marker.contentVirtualIndent = fallbackListMarkerVirtualIndent(source, paddingStart, indentColumns+markerWidth)
		position = paddingStart + 1
		columns = indentColumns + markerWidth + 1
	}
	marker.contentStart = position
	marker.contentIndent = columns
	return marker, true
}

func fallbackListMarkerVirtualIndent(source []byte, paddingStart, column int) int {
	if paddingStart < 0 || paddingStart >= len(source) || source[paddingStart] != '\t' {
		return 0
	}
	return 4 - column%4 - 1
}

func bareListMarkerBeforeCRLF(source []byte, line physicalLine) bool {
	return line.end+1 < len(source) && line.next == line.end+2 && source[line.end] == '\r' && source[line.end+1] == '\n'
}

func blankListMarker(marker listMarker, position, indentColumns, markerWidth int) listMarker {
	marker.blank = true
	marker.contentStart = position
	marker.contentIndent = indentColumns + markerWidth + 1
	return marker
}

func scanListMarkerPadding(source []byte, line physicalLine, position, columns int) (int, int) {
	for position < line.end && (source[position] == ' ' || source[position] == '\t') {
		if source[position] == ' ' {
			columns++
		} else {
			columns += 4 - columns%4
		}
		position++
	}
	return position, columns
}

func compatibleListMarker(left, right listMarker) bool {
	return left.ordered == right.ordered && left.marker == right.marker
}

func listInterruptsParagraph(source []byte, line physicalLine) bool {
	if _, ok := parseThematicBreak(source, line); ok {
		return false
	}
	marker, ok := parseListMarker(source, line)
	if !ok || marker.blank {
		return false
	}
	return !marker.ordered || marker.startNumber == 1
}

func parseListSemantic(source []byte, lines []physicalLine, index int, capture *semanticBlockCapture, parent int) (blockParseResult, int, bool) {
	if _, ok := parseThematicBreak(source, lines[index]); ok {
		return blockParseResult{}, index, false
	}
	first, ok := parseListMarker(source, lines[index])
	if !ok {
		return blockParseResult{}, index, false
	}
	collected := collectList(source, lines, index, first)
	if capture == nil {
		items, loose := parseListItems(source, collected.items)
		result := assembleList(source, items, loose)
		if len(collected.items) != 0 {
			result.trailingBlank = collected.items[len(collected.items)-1].trailingBlank
		}
		return result, collected.next, true
	}
	return parseCapturedList(source, collected, first, capture, parent)
}

func parseCapturedList(source []byte, collected listSource, first listMarker, capture *semanticBlockCapture, parent int) (blockParseResult, int, bool) {
	listRange := semanticListSourceRange(collected.items, first.physicalStart)
	start := 0
	if first.ordered {
		start = first.startNumber
	}
	listIndex := capture.add(parent, parser.SemanticEvent{
		Kind:    parser.SemanticList,
		Range:   listRange,
		Ordered: first.ordered,
		Start:   start,
		Marker:  first.marker,
	}, parser.Range{})
	parsed := make([]parsedListItem, len(collected.items))
	loose := false
	for index, item := range collected.items {
		itemRange := semanticListItemSourceRange(item)
		itemIndex := capture.add(listIndex, parser.SemanticEvent{
			Kind:    parser.SemanticListItem,
			Range:   itemRange,
			Ordered: item.marker.ordered,
			Marker:  item.marker.marker,
		}, parser.Range{})
		child := parseBlockLinesSemantic(source, item.lines, false, capture, itemIndex)
		if len(child.roots) != 0 {
			contentRange := itemRange
			if child.roots[0].range_ != (parser.Range{}) {
				contentRange = child.roots[0].range_
			}
			capture.update(itemIndex, func(event *parser.SemanticEvent) { event.ContentRange = contentRange })
		}
		parsed[index] = parsedListItem{source: item, child: child}
		if child.blankBetweenRoots || item.separatedAfter || referenceResidualMakesListLoose(child.roots) {
			loose = true
		}
	}
	capture.update(listIndex, func(event *parser.SemanticEvent) { event.Tight = !loose })
	result := assembleList(source, parsed, loose)
	if len(collected.items) != 0 {
		result.trailingBlank = collected.items[len(collected.items)-1].trailingBlank
	}
	return result, collected.next, true
}

func parseListItems(source []byte, items []listItemSource) ([]parsedListItem, bool) {
	parsed := make([]parsedListItem, len(items))
	loose := false
	for index, item := range items {
		child := item.child
		if !item.childReady {
			child = parseBlockLines(source, item.lines, false)
		}
		parsed[index] = parsedListItem{source: item, child: child}
		if child.blankBetweenRoots || item.separatedAfter || referenceResidualMakesListLoose(child.roots) {
			loose = true
		}
	}
	return parsed, loose
}

func assembleList(source []byte, items []parsedListItem, loose bool) blockParseResult {
	result := blockParseResult{nodes: make([]parser.Node, 0, len(items)*2)}
	containerAnchor := 0
	hasContainerAnchor := false
	for itemIndex, parsed := range items {
		item := parsed.source
		child := parsed.child
		if !loose {
			child.nodes = suppressTightListParagraphs(child.nodes, child.roots)
		}
		attachDirectListParents(child.nodes, child.roots)
		if itemIndex == 0 {
			containerAnchor, hasContainerAnchor = firstDirectLineAnchor(child.roots)
		}
		node, promoted := listItemObservation(item, child, containerAnchor, hasContainerAnchor)
		if promoted {
			result.nodes = append(result.nodes, node)
		}
		markTaskInlinePrefix(source, &child)
		if loose {
			result.nodes = append(result.nodes, child.nodes...)
		} else {
			result.nodes = appendTaskAndChildren(result.nodes, source, child)
		}
		result.blockquoteDetails = append(result.blockquoteDetails, child.blockquoteDetails...)
		result.fencedCodeDetails = append(result.fencedCodeDetails, child.fencedCodeDetails...)
		result.tableDetails = append(result.tableDetails, child.tableDetails...)
		result.tableRowDetails = append(result.tableRowDetails, child.tableRowDetails...)
		result.tableCellDetails = append(result.tableCellDetails, child.tableCellDetails...)
		result.semantic = append(result.semantic, child.semantic...)
		result.inlines = append(result.inlines, child.inlines...)
		result.references = append(result.references, child.references...)
		result.lastLeafParagraph = child.lastLeafParagraph && !item.trailingBlank
	}
	result.roots = append(result.roots, rootBlock{
		kind:                   rootBlockList,
		itemCount:              len(items),
		hasListContainerAnchor: hasContainerAnchor,
		listContainerAnchor:    containerAnchor,
	})
	return result
}

func referenceResidualMakesListLoose(roots []rootBlock) bool {
	for index := 1; index < len(roots); index++ {
		if roots[index].kind == rootBlockParagraph && roots[index-1].kind == rootBlockReference {
			return true
		}
	}
	return false
}

func suppressTightListParagraphs(nodes []parser.Node, roots []rootBlock) []parser.Node {
	write := 0
	rootIndex := 0
	for index, node := range nodes {
		for rootIndex < len(roots) && roots[rootIndex].kind != rootBlockParagraph {
			rootIndex++
		}
		if rootIndex < len(roots) && roots[rootIndex].nodeIndex == index {
			rootIndex++
			continue
		}
		nodes[write] = node
		write++
	}
	clear(nodes[write:])
	return nodes[:write]
}

func collectList(source []byte, lines []physicalLine, index int, first listMarker) listSource {
	result := listSource{}
	for index < len(lines) {
		marker, ok := parseListMarker(source, lines[index])
		if !ok || !compatibleListMarker(first, marker) || thematicAtLine(source, lines[index]) {
			break
		}
		item, next := collectListItem(source, lines, index, marker, first)
		result.items = append(result.items, item)
		index = next
		if index >= len(lines) {
			break
		}
		if marker2, ok := parseListMarker(source, lines[index]); !ok || !compatibleListMarker(first, marker2) || thematicAtLine(source, lines[index]) {
			break
		}
	}
	result.next = index
	return result
}

func collectListItem(source []byte, lines []physicalLine, index int, marker, listStyle listMarker) (listItemSource, int) {
	collector := newListItemCollector(source, lines[index], marker)
	next := index + 1
	for next < len(lines) {
		line := lines[next]
		if blankLine(source, line) {
			collector.appendBlank(line)
			next++
			continue
		}
		_, columns := leadingIndent(source, line)
		if columns >= marker.contentIndent {
			if collector.consumeIndented(source, line, columns, marker, listStyle) {
				break
			}
			next++
			continue
		}
		if collector.stopsAtSibling(source, line, listStyle) || collector.stopsAfterPendingBlank() {
			break
		}
		if collector.consumeLazy(source, line) {
			next++
			continue
		}
		break
	}
	if collector.hasContent && len(collector.pendingBlank) != 0 {
		collector.item.trailingBlank = true
	}
	return collector.item, next
}

func newListItemCollector(source []byte, line physicalLine, marker listMarker) listItemCollector {
	firstLine := advancePhysicalLineStart(source, line, marker.contentStart, marker.contentVirtualIndent)
	return listItemCollector{
		item:            listItemSource{marker: marker, lines: []physicalLine{firstLine}},
		pendingBlank:    make([]physicalLine, 0),
		hasContent:      !blankLine(source, firstLine),
		nestedTailEmpty: blankNestedListLine(source, firstLine),
	}
}

func (collector *listItemCollector) appendBlank(line physicalLine) {
	collector.pendingBlank = append(collector.pendingBlank, line)
	collector.item.childReady = false
}

func (collector *listItemCollector) consumeIndented(source []byte, line physicalLine, columns int, marker, listStyle listMarker) bool {
	if len(collector.pendingBlank) != 0 {
		if !collector.hasContent {
			collector.markSeparatedEmptySibling(source, line, listStyle)
			return true
		}
		if emptyNestedListRejectsBlankContinuation(source, line, collector.nestedTailEmpty, columns) {
			return true
		}
		collector.item.lines = append(collector.item.lines, collector.pendingBlank...)
		collector.pendingBlank = collector.pendingBlank[:0]
	}
	line = stripIndentColumns(source, line, marker.contentIndent)
	collector.item.lines = append(collector.item.lines, line)
	collector.item.childReady = false
	collector.hasContent = true
	collector.nestedTailEmpty = blankNestedListLine(source, line)
	return false
}

func (collector *listItemCollector) markSeparatedEmptySibling(source []byte, line physicalLine, listStyle listMarker) {
	nextMarker, ok := parseListMarker(source, line)
	collector.item.separatedAfter = ok && compatibleListMarker(listStyle, nextMarker) && !thematicAtLine(source, line)
}

func (collector *listItemCollector) stopsAtSibling(source []byte, line physicalLine, listStyle listMarker) bool {
	nextMarker, ok := parseListMarker(source, line)
	if !ok || thematicAtLine(source, line) {
		return false
	}
	if len(collector.pendingBlank) != 0 {
		collector.item.separatedAfter = compatibleListMarker(listStyle, nextMarker)
		collector.item.trailingBlank = !collector.item.separatedAfter
	}
	return true
}

func (collector *listItemCollector) stopsAfterPendingBlank() bool {
	if len(collector.pendingBlank) == 0 {
		return false
	}
	collector.item.trailingBlank = true
	return true
}

func (collector *listItemCollector) consumeLazy(source []byte, line physicalLine) bool {
	if !lazyParagraphContinuation(source, line) {
		return false
	}
	collector.item.child = parseBlockLines(source, collector.item.lines, false)
	collector.item.childReady = true
	if !collector.item.child.lastLeafParagraph {
		return false
	}
	collector.item.lines = append(collector.item.lines, line)
	collector.item.childReady = false
	collector.hasContent = true
	collector.nestedTailEmpty = false
	return true
}

func blankNestedListLine(source []byte, line physicalLine) bool {
	marker, ok := parseListMarker(source, line)
	return ok && marker.blank && !thematicAtLine(source, line)
}

func emptyNestedListRejectsBlankContinuation(source []byte, line physicalLine, nestedTailEmpty bool, columns int) bool {
	if !nestedTailEmpty || columns >= 4 {
		return false
	}
	_, ok := parseListMarker(source, line)
	return !ok || thematicAtLine(source, line)
}

func thematicAtLine(source []byte, line physicalLine) bool {
	_, ok := parseThematicBreak(source, line)
	return ok
}

func nestedParagraphEligibleLine(source []byte, line physicalLine) bool {
	for {
		if anchor, contentStart, ok := parseBlockquoteOpening(source, line); ok {
			line = blockquoteContentLine(source, line, anchor, contentStart)
			continue
		}
		if thematicAtLine(source, line) {
			return false
		}
		marker, ok := parseListMarker(source, line)
		if !ok {
			break
		}
		if marker.blank {
			return false
		}
		line = advancePhysicalLineStart(source, line, marker.contentStart, marker.contentVirtualIndent)
	}
	return paragraphEligibleLine(source, line)
}

func paragraphEligibleLine(source []byte, line physicalLine) bool {
	if blankLine(source, line) || indentedCodeLine(source, line) {
		return false
	}
	if _, _, ok := parseBlockquoteOpening(source, line); ok {
		return false
	}
	if _, ok := parseATXHeading(source, line); ok {
		return false
	}
	if _, ok := parseFenceOpening(source, line); ok {
		return false
	}
	if _, ok := parseThematicBreak(source, line); ok {
		return false
	}
	if _, ok := parseListMarker(source, line); ok {
		return false
	}
	return true
}

func lazyParagraphContinuation(source []byte, line physicalLine) bool {
	if blankLine(source, line) || startsBlockquote(source, line) {
		return false
	}
	return !interruptsParagraph(source, line)
}

func firstDirectLineAnchor(roots []rootBlock) (int, bool) {
	for _, root := range roots {
		if root.hasLineAnchor {
			return root.lineAnchor, true
		}
	}
	return 0, false
}

func attachDirectListParents(nodes []parser.Node, roots []rootBlock) {
	parentAnchor, hasParent := firstDirectLineAnchor(roots)
	if !hasParent {
		return
	}
	for _, root := range roots {
		if root.kind != rootBlockList || !root.hasListContainerAnchor {
			continue
		}
		for index := range nodes {
			if nodes[index].Kind == parser.KindListItem && nodes[index].ListContainerAnchor == root.listContainerAnchor {
				nodes[index].HasListParent = true
				nodes[index].ListParentAnchor = parentAnchor
			}
		}
	}
}

func listItemObservation(item listItemSource, child blockParseResult, containerAnchor int, hasContainerAnchor bool) (parser.Node, bool) {
	if !hasContainerAnchor || len(child.roots) == 0 || child.roots[0].kind != rootBlockParagraph || child.roots[0].lineCount != 1 {
		return parser.Node{}, false
	}
	directChildCount := 0
	for _, root := range child.roots[1:] {
		if root.kind != rootBlockList {
			return parser.Node{}, false
		}
		directChildCount += root.itemCount
	}
	return parser.Node{
		Kind:                 parser.KindListItem,
		Range:                child.roots[0].range_,
		Ordered:              item.marker.ordered,
		Marker:               item.marker.marker,
		ListContainerAnchor:  containerAnchor,
		HasListChildren:      directChildCount != 0,
		ListDirectChildCount: directChildCount,
	}, true
}

func appendTaskAndChildren(nodes []parser.Node, source []byte, child blockParseResult) []parser.Node {
	task, taskOK := taskObservation(source, child)
	if !taskOK {
		return append(nodes, child.nodes...)
	}
	inserted := false
	for _, node := range child.nodes {
		nodes = append(nodes, node)
		if !inserted && node.Kind == parser.KindParagraph && node.Range == child.roots[0].range_ {
			nodes = append(nodes, task)
			inserted = true
		}
	}
	if !inserted {
		start := len(nodes) - len(child.nodes)
		nodes = append(nodes, parser.Node{})
		copy(nodes[start+1:], nodes[start:len(nodes)-1])
		nodes[start] = task
	}
	return nodes
}

func markTaskInlinePrefix(source []byte, child *blockParseResult) {
	prefix, ok := taskMarkerRange(source, *child)
	if !ok {
		return
	}
	for index := range child.inlines {
		if len(child.inlines[index].segments) == 0 || child.inlines[index].segments[0].Start != prefix.Start {
			continue
		}
		child.inlines[index].prefixExclusion = prefix
		return
	}
}

func taskMarkerRange(source []byte, child blockParseResult) (parser.Range, bool) {
	if len(child.roots) == 0 || child.roots[0].kind != rootBlockParagraph {
		return parser.Range{}, false
	}
	range_ := child.roots[0].range_
	if range_.End-range_.Start < 3 {
		return parser.Range{}, false
	}
	prefix := source[range_.Start : range_.Start+3]
	if prefix[0] != '[' || prefix[2] != ']' || prefix[1] != ' ' && prefix[1] != 'x' && prefix[1] != 'X' {
		return parser.Range{}, false
	}
	return parser.Range{Start: range_.Start, End: range_.Start + 3}, true
}

func taskObservation(source []byte, child blockParseResult) (parser.Node, bool) {
	prefix, ok := taskMarkerRange(source, child)
	if !ok {
		return parser.Node{}, false
	}
	range_ := child.roots[0].range_
	if firstLineEnd := nativePhysicalLineRangeEnd(source, range_.Start); firstLineEnd < range_.End {
		range_.End = firstLineEnd
	}
	for range_.End > prefix.End && (source[range_.End-1] == ' ' || source[range_.End-1] == '\t') {
		range_.End--
	}
	checked := source[prefix.Start+1] == 'x' || source[prefix.Start+1] == 'X'
	return parser.Node{Kind: parser.KindTask, Range: range_, Checked: checked}, true
}
