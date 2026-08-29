package native

import (
	"cmp"
	"slices"
	"strings"

	"github.com/zoster81/marksplice/internal/parser"
	sourcepkg "github.com/zoster81/marksplice/internal/source"
)

type semanticCapturedBlock struct {
	parent      int
	event       parser.SemanticEvent
	inlineRange parser.Range
}

type semanticBlockCapture struct {
	blocks               []semanticCapturedBlock
	paragraphs           []int
	paragraphSearchLimit int
	extraInlines         []inlineBlock
}

func newSemanticBlockCapture(lineCount int) semanticBlockCapture {
	blockCapacity := max(32, lineCount-lineCount/8)
	return semanticBlockCapture{
		blocks:     make([]semanticCapturedBlock, 0, blockCapacity),
		paragraphs: make([]int, 0, max(8, lineCount/3)),
	}
}

func (capture *semanticBlockCapture) add(parent int, event parser.SemanticEvent, inlineRange parser.Range) int {
	if capture == nil {
		return -1
	}
	index := len(capture.blocks)
	capture.blocks = append(capture.blocks, semanticCapturedBlock{parent: parent, event: event, inlineRange: inlineRange})
	if event.Kind == parser.SemanticParagraph {
		capture.paragraphs = append(capture.paragraphs, index)
	}
	return index
}

func (capture *semanticBlockCapture) update(index int, update func(*parser.SemanticEvent)) {
	if capture == nil || index < 0 || index >= len(capture.blocks) {
		return
	}
	update(&capture.blocks[index].event)
}

func (capture *semanticBlockCapture) addInlineBlocks(blocks []inlineBlock) {
	if capture == nil || len(blocks) == 0 {
		return
	}
	capture.extraInlines = append(capture.extraInlines, blocks...)
}

func (capture *semanticBlockCapture) replace(index int, event parser.SemanticEvent, inlineRange parser.Range) {
	if capture == nil || index < 0 || index >= len(capture.blocks) {
		return
	}
	capture.blocks[index].event = event
	capture.blocks[index].inlineRange = inlineRange
}

func (capture *semanticBlockCapture) freezeParagraphIndex() {
	if capture == nil {
		return
	}
	slices.SortStableFunc(capture.paragraphs, func(left, right int) int {
		return cmp.Compare(capture.blocks[left].event.Range.Start, capture.blocks[right].event.Range.Start)
	})
	capture.paragraphSearchLimit = len(capture.paragraphs)
}

func (capture *semanticBlockCapture) paragraphContaining(start, end int) int {
	if capture == nil || capture.paragraphSearchLimit == 0 {
		return -1
	}
	low, high := 0, capture.paragraphSearchLimit
	for low < high {
		middle := low + (high-low)/2
		index := capture.paragraphs[middle]
		if capture.blocks[index].event.Range.Start <= start {
			low = middle + 1
		} else {
			high = middle
		}
	}
	if low == 0 {
		return -1
	}
	index := capture.paragraphs[low-1]
	range_ := capture.blocks[index].event.Range
	if range_.Start <= start && range_.End >= end {
		return index
	}
	return -1
}

// WalkSemantic emits an on-demand semantic event stream from Native parser
// ownership. It does not retain source or semantic events after return.
func (*Backend) WalkSemantic(source []byte, visit parser.SemanticVisitor) error {
	if visit == nil {
		return parser.ErrSemanticVisitorRequired
	}
	frontMatter, hasFrontMatter := sourcepkg.MapLeadingFrontMatter(source)
	lines := physicalLines(source)
	if hasFrontMatter {
		lines = semanticMarkdownLines(lines, frontMatter.Range.End)
	}
	capture := newSemanticBlockCapture(len(lines))
	blocks := parseBlockLinesSemantic(source, lines, true, &capture, -1)
	capture.freezeParagraphIndex()
	minimumOverlayStart := 0
	if hasFrontMatter {
		minimumOverlayStart = frontMatter.Range.End
	}
	footnotes := semanticFootnoteDefinitions(source, minimumOverlayStart)
	promoteSemanticFootnoteDefinitions(source, footnotes, &capture)
	inlineBlocks := blocks.inlines
	if len(capture.extraInlines) != 0 {
		inlineBlocks = append(append([]inlineBlock(nil), blocks.inlines...), capture.extraInlines...)
	}
	analyses := analyzeInlineBlocks(source, inlineBlocks, blocks.references)
	inline := mergeInlineAnalyses(analyses, 0)
	mathExpressions := nativeMathExpressionObservations(source, blocks, analyses, inline.nodes)
	mathExpressions = mergeNativeMathExpressions(mathExpressions, semanticCapturedBlockMathObservations(source, capture))
	promoteSemanticBlockMath(source, mathExpressions, &capture)
	footnoteReferences := semanticFootnoteReferences(source, blocks, footnotes)
	projection := newSemanticProjectionIndex(blocks, analyses)
	projection.terminals = semanticInlineTerminals(source, footnoteReferences, mathExpressions)
	if err := visit(parser.SemanticEvent{
		Phase: parser.SemanticEnter,
		Kind:  parser.SemanticDocument,
		Range: parser.Range{Start: 0, End: len(source)},
	}); err != nil {
		return err
	}
	if hasFrontMatter {
		if err := visit(semanticFrontMatterEvent(source, frontMatter)); err != nil {
			return err
		}
	}
	if err := emitSemanticCapturedBlocks(source, capture, projection, visit); err != nil {
		return err
	}
	return visit(parser.SemanticEvent{
		Phase: parser.SemanticExit,
		Kind:  parser.SemanticDocument,
		Range: parser.Range{Start: 0, End: len(source)},
	})
}

type semanticProjectionIndex struct {
	analyses     []inlineAnalysis
	byStart      map[int]int
	terminals    []parser.SemanticEvent
	tasksByStart map[int][]parser.Node
}

func newSemanticProjectionIndex(blocks blockParseResult, analyses []inlineAnalysis) semanticProjectionIndex {
	index := semanticProjectionIndex{
		analyses:     analyses,
		byStart:      make(map[int]int, len(analyses)),
		tasksByStart: make(map[int][]parser.Node),
	}
	for analysisIndex, analysis := range analyses {
		if len(analysis.block.segments) == 0 {
			continue
		}
		start := analysis.block.segments[0].Start
		if _, exists := index.byStart[start]; !exists {
			index.byStart[start] = analysisIndex
		}
	}
	for _, node := range blocks.nodes {
		if node.Kind == parser.KindTask {
			index.tasksByStart[node.Range.Start] = append(index.tasksByStart[node.Range.Start], node)
		}
	}
	return index
}

func (index semanticProjectionIndex) analysis(range_ parser.Range) (inlineAnalysis, bool) {
	analysisIndex, ok := index.byStart[range_.Start]
	if !ok || analysisIndex < 0 || analysisIndex >= len(index.analyses) {
		return inlineAnalysis{}, false
	}
	return index.analyses[analysisIndex], true
}

func semanticFootnoteDefinitions(source []byte, minimumStart int) []nativeFootnoteDefinition {
	definitions := scanNativeFootnoteDefinitions(source)
	if minimumStart <= 0 || len(definitions) == 0 {
		return definitions
	}
	kept := definitions[:0]
	for _, definition := range definitions {
		if definition.observation.Anchor >= minimumStart {
			kept = append(kept, definition)
		}
	}
	return kept
}

func promoteSemanticFootnoteDefinitions(source []byte, definitions []nativeFootnoteDefinition, capture *semanticBlockCapture) {
	for _, definition := range definitions {
		observation := definition.observation
		end := observation.Anchor
		if len(observation.BodyRanges) != 0 {
			end = observation.BodyRanges[len(observation.BodyRanges)-1].End
		}
		index := capture.paragraphContaining(observation.Anchor, end)
		event := parser.SemanticEvent{
			Kind:         parser.SemanticFootnoteDefinition,
			Range:        parser.Range{Start: observation.Anchor, End: end},
			ContentRange: semanticRangesEnvelope(observation.BodyRanges),
			Label:        observation.Label,
		}
		if index < 0 {
			index = capture.add(-1, event, parser.Range{})
		} else {
			capture.replace(index, event, parser.Range{})
		}
		body := parseBlockLinesSemantic(source, definition.childLines, false, capture, index)
		capture.addInlineBlocks(body.inlines)
	}
}

func semanticFootnoteReferences(source []byte, blocks blockParseResult, definitions []nativeFootnoteDefinition) []parser.FootnoteReferenceObservation {
	if len(definitions) == 0 {
		return nil
	}
	observations := make([]parser.FootnoteDefinitionObservation, len(definitions))
	for index, definition := range definitions {
		observations[index] = definition.observation
	}
	ordinary := nativeOrdinaryReferenceDefinitions(source, blocks.references, observations)
	return scanNativeFootnoteReferences(source, blocks.inlines, definitions, ordinary)
}

func semanticCapturedBlockMathObservations(source []byte, capture semanticBlockCapture) []parser.MathExpressionObservation {
	result := make([]parser.MathExpressionObservation, 0)
	for _, block := range capture.blocks {
		if block.event.Kind != parser.SemanticParagraph {
			continue
		}
		if observation, ok := nativeBlockDollarMathObservation(source, block.event.Range); ok {
			result = append(result, observation)
		}
	}
	return result
}

func promoteSemanticBlockMath(source []byte, expressions []parser.MathExpressionObservation, capture *semanticBlockCapture) {
	for _, expression := range expressions {
		if expression.Style != parser.MathExpressionBlockDollar {
			continue
		}
		for index, block := range capture.blocks {
			if block.event.Kind != parser.SemanticParagraph || block.event.Range != expression.Range {
				continue
			}
			capture.replace(index, semanticMathEvent(source, expression), parser.Range{})
			break
		}
	}
}

func semanticInlineTerminals(source []byte, references []parser.FootnoteReferenceObservation, expressions []parser.MathExpressionObservation) []parser.SemanticEvent {
	result := make([]parser.SemanticEvent, 0, len(references)+len(expressions))
	for _, reference := range references {
		result = append(result, parser.SemanticEvent{
			Phase:            parser.SemanticLeaf,
			Kind:             parser.SemanticFootnoteReference,
			Range:            reference.Range,
			ContentRange:     reference.LabelRange,
			Label:            reference.Label,
			DefinitionAnchor: reference.DefinitionAnchor,
			Occurrence:       reference.Occurrence,
		})
	}
	for _, expression := range expressions {
		if expression.Style != parser.MathExpressionBlockDollar {
			result = append(result, semanticMathEvent(source, expression))
		}
	}
	if len(result) > 1 {
		slices.SortStableFunc(result, func(left, right parser.SemanticEvent) int {
			if byStart := cmp.Compare(left.Range.Start, right.Range.Start); byStart != 0 {
				return byStart
			}
			return cmp.Compare(right.Range.End, left.Range.End)
		})
	}
	return result
}

func semanticMathEvent(source []byte, expression parser.MathExpressionObservation) parser.SemanticEvent {
	return parser.SemanticEvent{
		Phase:        parser.SemanticLeaf,
		Kind:         parser.SemanticMath,
		Range:        expression.Range,
		ContentRange: expression.PayloadRange,
		Value:        semanticSourceValue(source, expression.PayloadRange),
		MathStyle:    expression.Style,
	}
}

func semanticMarkdownLines(lines []physicalLine, envelopeEnd int) []physicalLine {
	first := 0
	for first < len(lines) && lines[first].physicalStart < envelopeEnd {
		first++
	}
	return lines[first:]
}

func semanticFrontMatterEvent(source []byte, mapping sourcepkg.FrontMatterMapping) parser.SemanticEvent {
	range_ := parser.Range{Start: mapping.Range.Start, End: mapping.Range.End}
	contentStart := semanticNextLineStart(source, mapping.OpeningRange.End)
	contentRange := parser.Range{Start: contentStart, End: mapping.ClosingRange.Start}
	return parser.SemanticEvent{
		Phase:             parser.SemanticLeaf,
		Kind:              parser.SemanticFrontMatter,
		Range:             range_,
		ContentRange:      contentRange,
		Value:             semanticSourceValue(source, range_),
		FrontMatterFormat: semanticFrontMatterFormat(mapping.Format),
	}
}

func semanticNextLineStart(source []byte, end int) int {
	if end < len(source) && source[end] == '\r' {
		end++
		if end < len(source) && source[end] == '\n' {
			end++
		}
	} else if end < len(source) && source[end] == '\n' {
		end++
	}
	return end
}

func semanticFrontMatterFormat(format sourcepkg.FrontMatterFormat) parser.SemanticFrontMatterFormat {
	switch format {
	case sourcepkg.FrontMatterYAML:
		return parser.SemanticFrontMatterYAML
	case sourcepkg.FrontMatterTOML:
		return parser.SemanticFrontMatterTOML
	default:
		return parser.SemanticFrontMatterUnknown
	}
}

func semanticAlertEvent(source []byte, quoted blockquoteParseResult, range_ parser.Range) (parser.SemanticEvent, []physicalLine, bool) {
	if len(quoted.content) < 2 {
		return parser.SemanticEvent{}, nil, false
	}
	markerLine := quoted.content[0]
	if markerLine.start < 0 || markerLine.start > markerLine.end || markerLine.end > len(source) {
		return parser.SemanticEvent{}, nil, false
	}
	kind := semanticAlertKind(sourcepkg.AlertKindFromMarker(source[markerLine.start:markerLine.end]))
	if kind == parser.SemanticAlertUnknown {
		return parser.SemanticEvent{}, nil, false
	}
	body := quoted.content[1:]
	if !semanticAlertHasBody(body) {
		return parser.SemanticEvent{}, nil, false
	}
	contentRange := parser.Range{Start: body[0].start, End: body[len(body)-1].end}
	return parser.SemanticEvent{
		Kind:         parser.SemanticAlert,
		Range:        range_,
		ContentRange: contentRange,
		AlertKind:    kind,
	}, body, true
}

func semanticAlertHasBody(lines []physicalLine) bool {
	for _, line := range lines {
		if line.start < line.end {
			return true
		}
	}
	return false
}

func semanticAlertKind(kind sourcepkg.AlertKind) parser.SemanticAlertKind {
	switch kind {
	case sourcepkg.AlertNote:
		return parser.SemanticAlertNote
	case sourcepkg.AlertTip:
		return parser.SemanticAlertTip
	case sourcepkg.AlertImportant:
		return parser.SemanticAlertImportant
	case sourcepkg.AlertWarning:
		return parser.SemanticAlertWarning
	case sourcepkg.AlertCaution:
		return parser.SemanticAlertCaution
	default:
		return parser.SemanticAlertUnknown
	}
}

func semanticPhysicalBlockRange(lines []physicalLine, first, next, start int) parser.Range {
	if first < 0 || first >= len(lines) || next <= first {
		return parser.Range{Start: start, End: start}
	}
	last := min(next, len(lines)) - 1
	return parser.Range{Start: start, End: lines[last].end}
}

func semanticReferenceDefinitionRange(lines []physicalLine, reference referenceDefinitionParse, start int) parser.Range {
	if reference.firstLine < 0 || reference.firstLine >= len(lines) || reference.lastLine < reference.firstLine || reference.lastLine >= len(lines) {
		return parser.Range{Start: start, End: start}
	}
	return parser.Range{Start: start, End: lines[reference.lastLine].end}
}

func semanticIndentedCodeValue(source []byte, lines []physicalLine) string {
	lastContent := len(lines)
	for lastContent > 0 && blankLine(source, lines[lastContent-1]) {
		lastContent--
	}
	var result strings.Builder
	for _, line := range lines[:lastContent] {
		stripped := stripIndentColumns(source, line, 4)
		if stripped.virtualIndent > 0 {
			result.WriteString(strings.Repeat(" ", stripped.virtualIndent))
		}
		if stripped.start < stripped.end {
			result.Write(source[stripped.start:stripped.end])
		}
		if line.next > line.end {
			result.WriteByte('\n')
		}
	}
	return result.String()
}

func semanticListSourceRange(items []listItemSource, start int) parser.Range {
	if len(items) == 0 {
		return parser.Range{Start: start, End: start}
	}
	return parser.Range{Start: start, End: semanticListItemSourceRange(items[len(items)-1]).End}
}

func semanticListItemSourceRange(item listItemSource) parser.Range {
	end := item.marker.physicalStart
	if len(item.lines) != 0 {
		end = item.lines[len(item.lines)-1].end
	}
	return parser.Range{Start: item.marker.physicalStart, End: end}
}

func captureSemanticTable(capture *semanticBlockCapture, parent int, nodes []parser.Node, details tableParseDetails) {
	if capture == nil || len(nodes) == 0 || len(details.tables) == 0 || len(details.cells) == 0 {
		return
	}
	table := details.tables[0]
	range_ := semanticTableRange(nodes[0].Range, details.cells)
	tableIndex := capture.add(parent, parser.SemanticEvent{Kind: parser.SemanticTable, Range: range_, Columns: table.ColumnCount}, parser.Range{})
	for first := 0; first < len(details.cells); {
		last := semanticTableRowEnd(details.cells, first)
		captureSemanticTableRow(capture, tableIndex, details.cells[first:last], table)
		first = last
	}
}

func captureSemanticTableRow(capture *semanticBlockCapture, parent int, row []parser.TableCellDetail, table parser.TableDetail) {
	if len(row) == 0 {
		return
	}
	range_ := semanticTableRowRange(row)
	rowIndex := capture.add(parent, parser.SemanticEvent{Kind: parser.SemanticTableRow, Range: range_, Header: row[0].Header, Columns: table.ColumnCount}, parser.Range{})
	for _, cell := range row {
		capture.add(rowIndex, parser.SemanticEvent{
			Kind:         parser.SemanticTableCell,
			Range:        cell.Range,
			ContentRange: cell.Range,
			Header:       cell.Header,
			Column:       cell.Column,
			Alignment:    semanticTableAlignment(table.Alignments, cell.Column),
		}, cell.Range)
	}
	for column := len(row); column < table.ColumnCount; column++ {
		emptyRange := parser.Range{Start: range_.End, End: range_.End}
		capture.add(rowIndex, parser.SemanticEvent{
			Kind:      parser.SemanticTableCell,
			Range:     emptyRange,
			Header:    row[0].Header,
			Column:    column,
			Alignment: semanticTableAlignment(table.Alignments, column),
		}, parser.Range{})
	}
}

func emitSemanticCapturedBlocks(source []byte, capture semanticBlockCapture, projection semanticProjectionIndex, visit parser.SemanticVisitor) error {
	firstChild, nextSibling := semanticCapturedChildIndex(capture.blocks)
	for childRef := firstChild[0]; childRef != 0; childRef = nextSibling[childRef-1] {
		if err := emitSemanticCapturedBlock(source, capture.blocks, firstChild, nextSibling, projection, childRef-1, visit); err != nil {
			return err
		}
	}
	return nil
}

func semanticCapturedChildIndex(blocks []semanticCapturedBlock) ([]int, []int) {
	firstChild := make([]int, len(blocks)+1)
	nextSibling := make([]int, len(blocks))
	for index := len(blocks) - 1; index >= 0; index-- {
		parent := blocks[index].parent
		if parent < -1 || parent >= len(blocks) {
			parent = -1
		}
		owner := parent + 1
		nextSibling[index] = firstChild[owner]
		firstChild[owner] = index + 1
	}
	return firstChild, nextSibling
}

func emitSemanticCapturedBlock(source []byte, blocks []semanticCapturedBlock, firstChild, nextSibling []int, projection semanticProjectionIndex, index int, visit parser.SemanticVisitor) error {
	block := blocks[index]
	event := block.event
	if !semanticCapturedContainer(event.Kind) {
		event.Phase = parser.SemanticLeaf
		return visit(event)
	}
	event.Phase = parser.SemanticEnter
	if err := visit(event); err != nil {
		return err
	}
	if event.Kind == parser.SemanticListItem {
		if err := emitSemanticCapturedTasks(event, projection, visit); err != nil {
			return err
		}
	}
	if block.inlineRange != (parser.Range{}) {
		if analysis, ok := projection.analysis(block.inlineRange); ok {
			if err := emitSemanticInlineProjected(source, analysis, projection.terminals, visit); err != nil {
				return err
			}
		}
	}
	for childRef := firstChild[index+1]; childRef != 0; childRef = nextSibling[childRef-1] {
		if err := emitSemanticCapturedBlock(source, blocks, firstChild, nextSibling, projection, childRef-1, visit); err != nil {
			return err
		}
	}
	return visit(parser.SemanticEvent{Phase: parser.SemanticExit, Kind: event.Kind, Range: event.Range, Level: event.Level})
}

func semanticCapturedContainer(kind parser.SemanticKind) bool {
	switch kind {
	case parser.SemanticParagraph, parser.SemanticHeading, parser.SemanticBlockquote, parser.SemanticAlert,
		parser.SemanticFootnoteDefinition, parser.SemanticList, parser.SemanticListItem, parser.SemanticTable,
		parser.SemanticTableRow, parser.SemanticTableCell:
		return true
	default:
		return false
	}
}

func emitSemanticCapturedTasks(item parser.SemanticEvent, projection semanticProjectionIndex, visit parser.SemanticVisitor) error {
	if item.ContentRange == (parser.Range{}) {
		return nil
	}
	for _, task := range projection.tasksByStart[item.ContentRange.Start] {
		if err := visit(parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticTaskItem, Range: task.Range, Checked: task.Checked}); err != nil {
			return err
		}
	}
	return nil
}

func semanticTableRange(fallback parser.Range, cells []parser.TableCellDetail) parser.Range {
	range_ := fallback
	for _, cell := range cells {
		if cell.Range.Start < range_.Start {
			range_.Start = cell.Range.Start
		}
		if cell.Range.End > range_.End {
			range_.End = cell.Range.End
		}
	}
	return range_
}

func semanticTableRowEnd(cells []parser.TableCellDetail, first int) int {
	last := first + 1
	for last < len(cells) && cells[last].RowAnchor == cells[first].RowAnchor {
		last++
	}
	return last
}

func semanticTableRowRange(row []parser.TableCellDetail) parser.Range {
	range_ := row[0].Range
	for _, cell := range row[1:] {
		if cell.Range.End > range_.End {
			range_.End = cell.Range.End
		}
	}
	return range_
}

func semanticTableAlignment(alignments []parser.TableAlignment, column int) parser.TableAlignment {
	if column < 0 || column >= len(alignments) {
		return parser.TableAlignmentDefault
	}
	return alignments[column]
}

func semanticRangesEnvelope(ranges []parser.Range) parser.Range {
	if len(ranges) == 0 {
		return parser.Range{}
	}
	return parser.Range{Start: ranges[0].Start, End: ranges[len(ranges)-1].End}
}

func semanticLogicalLinesValue(source []byte, lines []physicalLine) string {
	if len(lines) == 0 {
		return ""
	}
	var result strings.Builder
	for _, line := range lines {
		if line.start < line.end {
			result.Write(source[line.start:line.end])
		}
		if line.end < line.next {
			result.Write(source[line.end:line.next])
		}
	}
	return result.String()
}

func semanticRangesValue(source []byte, ranges []parser.Range) string {
	if len(ranges) == 0 {
		return ""
	}
	var result strings.Builder
	for index, range_ := range ranges {
		if !range_.Valid(len(source)) || range_.Start > range_.End {
			continue
		}
		if index != 0 {
			result.WriteByte('\n')
		}
		result.Write(source[range_.Start:range_.End])
	}
	return result.String()
}

func semanticFencedCodeValue(source []byte, ranges []parser.Range) string {
	value := semanticRangesValue(source, ranges)
	if len(ranges) == 0 {
		return value
	}
	last := ranges[len(ranges)-1]
	if last.Valid(len(source)) && last.End < len(source) && (source[last.End] == '\r' || source[last.End] == '\n') {
		value += "\n"
	}
	return value
}

func emitSemanticInlineProjected(source []byte, analysis inlineAnalysis, terminals []parser.SemanticEvent, visit parser.SemanticVisitor) error {
	if len(analysis.block.segments) == 0 {
		return nil
	}
	nodes := semanticConstructionSemantics(source, analysis.block, analysis.owners, analysis.delimiters)
	assignConstructionParents(nodes)
	children := make([][]int, len(nodes)+1)
	for index, node := range nodes {
		children[node.parent+1] = append(children[node.parent+1], index)
	}
	first := analysis.block.segments[0].Start
	last := analysis.block.segments[len(analysis.block.segments)-1].End
	return emitSemanticInlineRegion(source, analysis.block, nodes, children, -1, parser.Range{Start: first, End: last}, terminals, visit)
}

func emitSemanticInlineRegion(source []byte, block inlineBlock, nodes []constructionSemantic, children [][]int, parent int, region parser.Range, terminals []parser.SemanticEvent, visit parser.SemanticVisitor) error {
	cursor := region.Start
	for _, child := range children[parent+1] {
		node := nodes[child]
		if node.syntax.Start < cursor || node.syntax.End > region.End {
			continue
		}
		if terminal, ok := semanticTerminalCoveringNode(terminals, node.syntax, region); ok {
			if err := emitSemanticTextGapProjected(source, block, parser.Range{Start: cursor, End: terminal.Range.Start}, terminals, visit); err != nil {
				return err
			}
			if err := visit(terminal); err != nil {
				return err
			}
			cursor = terminal.Range.End
			continue
		}
		if err := emitSemanticTextGapProjected(source, block, parser.Range{Start: cursor, End: node.syntax.Start}, terminals, visit); err != nil {
			return err
		}
		if err := emitSemanticInlineNode(source, block, nodes, children, child, terminals, visit); err != nil {
			return err
		}
		cursor = node.syntax.End
	}
	return emitSemanticTextGapProjected(source, block, parser.Range{Start: cursor, End: region.End}, terminals, visit)
}

func semanticTerminalCoveringNode(terminals []parser.SemanticEvent, syntax, region parser.Range) (parser.SemanticEvent, bool) {
	for index := semanticTerminalIndex(terminals, syntax.Start); index < len(terminals); index++ {
		terminal := terminals[index]
		if terminal.Range.Start > syntax.Start {
			break
		}
		if terminal.Range.Start >= region.Start && terminal.Range.End <= region.End &&
			terminal.Range.Start <= syntax.Start && terminal.Range.End >= syntax.End {
			return terminal, true
		}
	}
	return parser.SemanticEvent{}, false
}

func semanticTerminalIndex(terminals []parser.SemanticEvent, offset int) int {
	index, _ := slices.BinarySearchFunc(terminals, offset, func(event parser.SemanticEvent, target int) int {
		return cmp.Compare(event.Range.Start, target)
	})
	if index > 0 && terminals[index-1].Range.End > offset {
		index--
	}
	return index
}

func emitSemanticInlineNode(source []byte, block inlineBlock, nodes []constructionSemantic, children [][]int, index int, terminals []parser.SemanticEvent, visit parser.SemanticVisitor) error {
	node := nodes[index]
	kind := semanticKindForParserKind(node.kind)
	if kind == parser.SemanticUnknown {
		return nil
	}
	switch kind {
	case parser.SemanticEmphasis, parser.SemanticStrong, parser.SemanticStrikethrough, parser.SemanticLink, parser.SemanticImage:
		enter := parser.SemanticEvent{
			Phase:        parser.SemanticEnter,
			Kind:         kind,
			Range:        node.syntax,
			ContentRange: node.content,
			Destination:  node.destination,
			Title:        node.title,
			HasTitle:     node.hasTitle,
			Label:        node.reference,
		}
		if err := visit(enter); err != nil {
			return err
		}
		if err := emitSemanticInlineRegion(source, block, nodes, children, index, node.content, terminals, visit); err != nil {
			return err
		}
		return visit(parser.SemanticEvent{Phase: parser.SemanticExit, Kind: kind, Range: node.syntax})
	case parser.SemanticCodeSpan, parser.SemanticRawHTML, parser.SemanticAutoLink:
		value := semanticSourceValue(source, node.content)
		if kind == parser.SemanticRawHTML && node.value != "" {
			value = node.value
		}
		if kind == parser.SemanticCodeSpan {
			if normalized, ok := semanticCodeSpanValue(source, node.syntax); ok {
				value = normalized
			}
			if block.tableCell {
				value = semanticTableEscapedPipeValue(value)
			}
		}
		event := parser.SemanticEvent{
			Phase:        parser.SemanticLeaf,
			Kind:         kind,
			Range:        node.syntax,
			ContentRange: node.content,
			Value:        value,
		}
		if kind == parser.SemanticAutoLink {
			event.AutoLinkEmail = node.autoLinkEmail
			event.Destination = semanticAutoLinkDestination(value, node.autoLinkEmail)
		}
		return visit(event)
	default:
		return nil
	}
}

func semanticKindForParserKind(kind parser.Kind) parser.SemanticKind {
	switch kind {
	case parser.KindEmphasis:
		return parser.SemanticEmphasis
	case parser.KindStrong:
		return parser.SemanticStrong
	case parser.KindStrikethrough:
		return parser.SemanticStrikethrough
	case parser.KindCodeSpan:
		return parser.SemanticCodeSpan
	case parser.KindInlineLink:
		return parser.SemanticLink
	case parser.KindImage:
		return parser.SemanticImage
	case parser.KindAutoLink:
		return parser.SemanticAutoLink
	case parser.KindRawHTML:
		return parser.SemanticRawHTML
	default:
		return parser.SemanticUnknown
	}
}

func semanticConstructionSemantics(source []byte, block inlineBlock, owners []inlineSpan, delimiters delimiterParseResult) []constructionSemantic {
	withoutMatches := delimiters
	withoutMatches.matches = nil
	result := collectConstructionSemantics(owners, withoutMatches)
	result = filterSemanticRawHTML(source, result)
	result = appendSemanticMultilineRawHTML(source, block, owners, result)
	result = appendSemanticDelimiterSemantics(delimiters, result)
	result = appendSemanticCodeSpanOwners(source, owners, result)
	sortSemanticConstructions(result)
	return result
}

func filterSemanticRawHTML(source []byte, semantics []constructionSemantic) []constructionSemantic {
	kept := semantics[:0]
	for _, node := range semantics {
		if node.kind == parser.KindRawHTML && !semanticRawHTMLProjectable(source, node.syntax) {
			continue
		}
		kept = append(kept, node)
	}
	return kept
}

func appendSemanticMultilineRawHTML(source []byte, block inlineBlock, owners []inlineSpan, result []constructionSemantic) []constructionSemantic {
	for _, owner := range owners {
		if owner.kind != parser.KindUnknown || owner.endSegment <= owner.segment || owner.start < 0 || owner.start >= len(source) || source[owner.start] != '<' {
			continue
		}
		value, ok := semanticInlineSpanValue(source, block, owner)
		if !ok {
			continue
		}
		syntax := parser.Range{Start: owner.start, End: owner.end}
		result = append(result, constructionSemantic{kind: parser.KindRawHTML, syntax: syntax, content: syntax, value: value, parent: -1})
	}
	return result
}

func appendSemanticDelimiterSemantics(delimiters delimiterParseResult, result []constructionSemantic) []constructionSemantic {
	for _, match := range delimiters.matches {
		if semanticDelimiterCrossesCompositeLabel(match, delimiters.composites) {
			continue
		}
		kind := constructionDelimiterKind(match)
		syntax, content, ok := semanticDelimiterProjection(match)
		if kind == parser.KindUnknown || !ok {
			continue
		}
		result = append(result, constructionSemantic{kind: kind, syntax: syntax, content: content, parent: -1})
	}
	return result
}

func appendSemanticCodeSpanOwners(source []byte, owners []inlineSpan, result []constructionSemantic) []constructionSemantic {
	for _, owner := range owners {
		if owner.kind != parser.KindUnknown || owner.start < 0 || owner.start >= len(source) || source[owner.start] != '`' {
			continue
		}
		syntax := parser.Range{Start: owner.start, End: owner.end}
		if _, ok := semanticCodeSpanValue(source, syntax); !ok {
			continue
		}
		result = append(result, constructionSemantic{kind: parser.KindCodeSpan, syntax: syntax, parent: -1})
	}
	return result
}

func sortSemanticConstructions(result []constructionSemantic) {
	if len(result) <= 1 {
		return
	}
	slices.SortStableFunc(result, func(left, right constructionSemantic) int {
		if order := cmp.Compare(left.syntax.Start, right.syntax.Start); order != 0 {
			return order
		}
		return cmp.Compare(right.syntax.End, left.syntax.End)
	})
}

func semanticRawHTMLProjectable(source []byte, syntax parser.Range) bool {
	if !syntax.Valid(len(source)) || syntax.Start >= syntax.End {
		return false
	}
	value := source[syntax.Start:syntax.End]
	if len(value) < 4 || value[0] != '<' || value[1] != '!' || value[2] != '-' || value[3] != '-' {
		return true
	}
	if len(value) < 7 || value[len(value)-3] != '-' || value[len(value)-2] != '-' || value[len(value)-1] != '>' {
		return false
	}
	return validHTMLCommentText(value[4 : len(value)-3])
}

func semanticInlineSpanValue(source []byte, block inlineBlock, owner inlineSpan) (string, bool) {
	if owner.segment < 0 || owner.endSegment < owner.segment || owner.endSegment >= len(block.segments) {
		return "", false
	}
	var result strings.Builder
	for segmentIndex := owner.segment; segmentIndex <= owner.endSegment; segmentIndex++ {
		segment := block.segments[segmentIndex]
		start, end := segment.Start, segment.End
		if segmentIndex == owner.segment {
			start = owner.start
		}
		if segmentIndex == owner.endSegment {
			end = owner.end
		}
		if start < segment.Start || end > segment.End || start > end {
			return "", false
		}
		result.Write(source[start:end])
		if segmentIndex < owner.endSegment {
			result.WriteByte('\n')
		}
	}
	return result.String(), true
}

func semanticDelimiterCrossesCompositeLabel(match delimiterMatch, composites []compositeInline) bool {
	for _, composite := range composites {
		if !composite.active {
			continue
		}
		openingInside := semanticRangeInside(match.openingConsumed, composite.label)
		closingInside := semanticRangeInside(match.closingConsumed, composite.label)
		if openingInside != closingInside {
			return true
		}
	}
	return false
}

func semanticRangeInside(inner, outer parser.Range) bool {
	return inner.Start >= outer.Start && inner.End <= outer.End && inner.Start < inner.End
}

func semanticDelimiterProjection(match delimiterMatch) (parser.Range, parser.Range, bool) {
	if match.level <= 0 || match.openingConsumed.End-match.openingConsumed.Start != match.level ||
		match.closingConsumed.End-match.closingConsumed.Start != match.level || match.closer > match.syntaxEnd {
		return parser.Range{}, parser.Range{}, false
	}
	consumedBefore := match.syntaxEnd - match.closingConsumed.End
	closingStart := match.closer + consumedBefore
	closingEnd := closingStart + match.level
	syntax := parser.Range{Start: match.openingConsumed.Start, End: closingEnd}
	content := parser.Range{Start: match.openingConsumed.End, End: closingStart}
	if syntax.Start < match.syntaxStart || syntax.End > match.syntaxEnd || content.Start > content.End ||
		content.Start < syntax.Start || content.End > syntax.End {
		return parser.Range{}, parser.Range{}, false
	}
	return syntax, content, true
}

func semanticTableEscapedPipeValue(value string) string {
	var result []byte
	backslashes := 0
	for index := 0; index < len(value); index++ {
		current := value[index]
		if current == '\\' {
			backslashes++
			if result != nil {
				result = append(result, current)
			}
			continue
		}
		if current == '|' && backslashes%2 != 0 {
			if result == nil {
				result = append(make([]byte, 0, len(value)-1), value[:index-1]...)
			} else {
				result = result[:len(result)-1]
			}
		}
		if result != nil {
			result = append(result, current)
		}
		backslashes = 0
	}
	if result == nil {
		return value
	}
	return string(result)
}

func semanticCodeSpanValue(source []byte, syntax parser.Range) (string, bool) {
	payload, ok := semanticCodeSpanPayload(source, syntax)
	if !ok {
		return "", false
	}
	value := semanticNormalizeCodeSpanPayload(payload)
	if len(value) >= 2 && value[0] == ' ' && value[len(value)-1] == ' ' && !onlyASCIISpaces([]byte(value)) {
		value = value[1 : len(value)-1]
	}
	return value, true
}

func semanticCodeSpanPayload(source []byte, syntax parser.Range) ([]byte, bool) {
	if !syntax.Valid(len(source)) || syntax.End-syntax.Start < 2 || source[syntax.Start] != '`' {
		return nil, false
	}
	run := 1
	for syntax.Start+run < syntax.End && source[syntax.Start+run] == '`' {
		run++
	}
	if syntax.End-syntax.Start < run*2 {
		return nil, false
	}
	closingStart := syntax.End - run
	for position := closingStart; position < syntax.End; position++ {
		if source[position] != '`' {
			return nil, false
		}
	}
	return source[syntax.Start+run : closingStart], true
}

func semanticNormalizeCodeSpanPayload(payload []byte) string {
	var normalized strings.Builder
	normalized.Grow(len(payload))
	for index := 0; index < len(payload); index++ {
		switch payload[index] {
		case '\r':
			if index+1 < len(payload) && payload[index+1] == '\n' {
				index++
			}
			normalized.WriteByte(' ')
		case '\n':
			normalized.WriteByte(' ')
		default:
			normalized.WriteByte(payload[index])
		}
	}
	return normalized.String()
}

func semanticAutoLinkDestination(value string, email bool) string {
	if email {
		return "mailto:" + value
	}
	if strings.HasPrefix(value, "www.") {
		return "http://" + value
	}
	return value
}

func emitSemanticTextGapProjected(source []byte, block inlineBlock, gap parser.Range, terminals []parser.SemanticEvent, visit parser.SemanticVisitor) error {
	if gap.Start >= gap.End {
		return nil
	}
	for index, segment := range block.segments {
		if segment.End < gap.Start || segment.Start >= gap.End {
			continue
		}
		segmentStart := semanticInlineSegmentStart(source, segment, index > 0)
		start := max(gap.Start, segmentStart)
		end := min(gap.End, segment.End)
		hasNext := index+1 < len(block.segments)
		textEnd, breakKind := semanticLineEnd(source, segment, hasNext)
		if end > textEnd {
			end = textEnd
		}
		if err := emitSemanticTextRangeExcluding(source, parser.Range{Start: start, End: end}, block.prefixExclusion, terminals, visit); err != nil {
			return err
		}
		if !hasNext || gap.End < block.segments[index+1].Start || gap.Start > segment.End {
			continue
		}
		breakRange := semanticBreakRange(source, segment, textEnd, breakKind)
		if err := visit(parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: breakKind, Range: breakRange}); err != nil {
			return err
		}
	}
	return nil
}

func emitSemanticTextRangeExcluding(source []byte, range_, exclusion parser.Range, terminals []parser.SemanticEvent, visit parser.SemanticVisitor) error {
	if exclusion.Start >= exclusion.End || exclusion.End <= range_.Start || exclusion.Start >= range_.End {
		return emitSemanticTextRangeProjected(source, range_, terminals, visit)
	}
	before := parser.Range{Start: range_.Start, End: min(range_.End, exclusion.Start)}
	if err := emitSemanticTextRangeProjected(source, before, terminals, visit); err != nil {
		return err
	}
	after := parser.Range{Start: max(range_.Start, exclusion.End), End: range_.End}
	return emitSemanticTextRangeProjected(source, after, terminals, visit)
}

func emitSemanticTextRangeProjected(source []byte, range_ parser.Range, terminals []parser.SemanticEvent, visit parser.SemanticVisitor) error {
	if range_.Start >= range_.End {
		return nil
	}
	cursor := range_.Start
	for index := semanticTerminalIndex(terminals, cursor); index < len(terminals); index++ {
		terminal := terminals[index]
		if terminal.Range.Start >= range_.End {
			break
		}
		if terminal.Range.Start < cursor || terminal.Range.End > range_.End {
			continue
		}
		if err := emitSemanticTextRange(source, parser.Range{Start: cursor, End: terminal.Range.Start}, visit); err != nil {
			return err
		}
		if err := visit(terminal); err != nil {
			return err
		}
		cursor = terminal.Range.End
	}
	return emitSemanticTextRange(source, parser.Range{Start: cursor, End: range_.End}, visit)
}

func emitSemanticTextRange(source []byte, range_ parser.Range, visit parser.SemanticVisitor) error {
	if range_.Start >= range_.End {
		return nil
	}
	value := semanticDecodedText(source[range_.Start:range_.End])
	if value == "" {
		return nil
	}
	return visit(parser.SemanticEvent{
		Phase:        parser.SemanticLeaf,
		Kind:         parser.SemanticText,
		Range:        range_,
		ContentRange: range_,
		Value:        value,
	})
}

func semanticInlineSegmentStart(source []byte, segment parser.Range, continuation bool) int {
	start := segment.Start
	if !continuation {
		return start
	}
	for start < segment.End && (source[start] == ' ' || source[start] == '\t') {
		start++
	}
	return start
}

func semanticLineEnd(source []byte, segment parser.Range, hasNext bool) (int, parser.SemanticKind) {
	end := segment.End
	if hasNext && end > segment.Start && source[end-1] == '\\' && !inlineByteEscaped(source, segment.Start, end-1) {
		return end - 1, parser.SemanticHardBreak
	}
	spaces := 0
	for position := end; position > segment.Start && source[position-1] == ' '; position-- {
		spaces++
	}
	if hasNext && spaces >= 2 {
		return end - spaces, parser.SemanticHardBreak
	}
	for end > segment.Start && (source[end-1] == ' ' || source[end-1] == '\t') {
		end--
	}
	return end, parser.SemanticSoftBreak
}

func semanticBreakRange(source []byte, segment parser.Range, textEnd int, kind parser.SemanticKind) parser.Range {
	start := segment.End
	if kind == parser.SemanticHardBreak {
		start = textEnd
	}
	end := segment.End
	if end < len(source) && source[end] == '\r' {
		end++
		if end < len(source) && source[end] == '\n' {
			end++
		}
	} else if end < len(source) && source[end] == '\n' {
		end++
	}
	return parser.Range{Start: start, End: end}
}

func semanticDecodedText(source []byte) string {
	if len(source) == 0 {
		return ""
	}
	var result strings.Builder
	result.Grow(len(source))
	appendDecodedHeadingText(&result, source)
	return result.String()
}

func semanticSourceValue(source []byte, range_ parser.Range) string {
	if !range_.Valid(len(source)) || range_.Start >= range_.End {
		return ""
	}
	return string(source[range_.Start:range_.End])
}
