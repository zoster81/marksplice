package native

import (
	"strings"

	"github.com/zoster81/marksplice/internal/parser"
)

// WalkSemantic emits an on-demand semantic event stream from Native parser
// ownership. It does not retain source or semantic events after return.
func (*Backend) WalkSemantic(source []byte, visit parser.SemanticVisitor) error {
	if visit == nil {
		return parser.ErrSemanticVisitorRequired
	}
	blocks := parseBlockLines(source, physicalLines(source), true)
	analyses := analyzeInlineBlocks(source, blocks.inlines, blocks.references)
	projection := newSemanticProjectionIndex(blocks, analyses)
	if err := visit(parser.SemanticEvent{
		Phase: parser.SemanticEnter,
		Kind:  parser.SemanticDocument,
		Range: parser.Range{Start: 0, End: len(source)},
	}); err != nil {
		return err
	}
	if err := emitSemanticBlocks(source, blocks, projection, visit); err != nil {
		return err
	}
	if err := emitSemanticSupplemental(source, blocks, analyses, visit); err != nil {
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
	listItems    map[int][]parser.Node
	tasksByStart map[int][]parser.Node
	tableDetails map[int]parser.TableDetail
	tableCells   map[int][]parser.TableCellDetail
	fencedCode   map[int]parser.FencedCodeDetail
}

func newSemanticProjectionIndex(blocks blockParseResult, analyses []inlineAnalysis) semanticProjectionIndex {
	index := semanticProjectionIndex{
		analyses:     analyses,
		byStart:      make(map[int]int, len(analyses)),
		listItems:    make(map[int][]parser.Node),
		tasksByStart: make(map[int][]parser.Node),
		tableDetails: make(map[int]parser.TableDetail, len(blocks.tableDetails)),
		tableCells:   make(map[int][]parser.TableCellDetail, len(blocks.tableDetails)),
		fencedCode:   make(map[int]parser.FencedCodeDetail, len(blocks.fencedCodeDetails)),
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
		switch node.Kind {
		case parser.KindListItem:
			index.listItems[node.ListContainerAnchor] = append(index.listItems[node.ListContainerAnchor], node)
		case parser.KindTask:
			index.tasksByStart[node.Range.Start] = append(index.tasksByStart[node.Range.Start], node)
		}
	}
	for _, detail := range blocks.tableDetails {
		index.tableDetails[detail.Anchor] = detail
	}
	for _, detail := range blocks.tableCellDetails {
		index.tableCells[detail.TableAnchor] = append(index.tableCells[detail.TableAnchor], detail)
	}
	for _, detail := range blocks.fencedCodeDetails {
		index.fencedCode[detail.Anchor] = detail
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

type semanticBlockEmitter struct {
	source     []byte
	projection semanticProjectionIndex
	visit      parser.SemanticVisitor
	seenLists  map[int]struct{}
	seenTables map[int]struct{}
}

func emitSemanticBlocks(source []byte, blocks blockParseResult, projection semanticProjectionIndex, visit parser.SemanticVisitor) error {
	emitter := semanticBlockEmitter{
		source:     source,
		projection: projection,
		visit:      visit,
		seenLists:  make(map[int]struct{}),
		seenTables: make(map[int]struct{}),
	}
	for _, node := range blocks.nodes {
		if err := emitter.emit(node); err != nil {
			return err
		}
	}
	return nil
}

func (emitter *semanticBlockEmitter) emit(node parser.Node) error {
	switch node.Kind {
	case parser.KindParagraph, parser.KindHeading:
		return emitter.emitText(node)
	case parser.KindBlockquote:
		return emitter.emitBlockquote(node)
	case parser.KindListItem:
		return emitter.emitList(node)
	case parser.KindTable:
		return emitter.emitTable(node)
	case parser.KindFencedCode:
		return emitter.emitFencedCode(node)
	case parser.KindHTMLBlock:
		return emitter.visit(parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticHTMLBlock, Range: node.Range, ContentRange: node.Range, Value: semanticSourceValue(emitter.source, node.Range)})
	case parser.KindThematicBreak:
		return emitter.emitThematicBreak(node)
	default:
		return nil
	}
}

func (emitter *semanticBlockEmitter) emitText(node parser.Node) error {
	if !node.TopLevel {
		return nil
	}
	return emitSemanticTextBlock(emitter.source, node, emitter.projection, emitter.visit)
}

func (emitter *semanticBlockEmitter) emitBlockquote(node parser.Node) error {
	if !node.TopLevel {
		return nil
	}
	return emitSemanticPair(parser.SemanticBlockquote, node.Range, parser.SemanticEvent{}, emitter.visit)
}

func (emitter *semanticBlockEmitter) emitList(node parser.Node) error {
	anchor := node.ListContainerAnchor
	if _, emitted := emitter.seenLists[anchor]; emitted {
		return nil
	}
	emitter.seenLists[anchor] = struct{}{}
	return emitSemanticList(emitter.source, anchor, emitter.projection, emitter.visit)
}

func (emitter *semanticBlockEmitter) emitTable(node parser.Node) error {
	anchor := node.Range.Start
	if _, emitted := emitter.seenTables[anchor]; emitted {
		return nil
	}
	emitter.seenTables[anchor] = struct{}{}
	return emitSemanticTable(emitter.source, anchor, node.Range, emitter.projection, emitter.visit)
}

func (emitter *semanticBlockEmitter) emitFencedCode(node parser.Node) error {
	if !node.TopLevel {
		return nil
	}
	return emitSemanticFencedCode(emitter.source, node, emitter.projection, emitter.visit)
}

func (emitter *semanticBlockEmitter) emitThematicBreak(node parser.Node) error {
	if !node.TopLevel {
		return nil
	}
	return emitter.visit(parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticThematicBreak, Range: node.Range})
}

func emitSemanticTextBlock(source []byte, node parser.Node, projection semanticProjectionIndex, visit parser.SemanticVisitor) error {
	analysis, ok := projection.analysis(node.Range)
	if !ok {
		return nil
	}
	kind := parser.SemanticParagraph
	if node.Kind == parser.KindHeading {
		kind = parser.SemanticHeading
	}
	if err := visit(parser.SemanticEvent{Phase: parser.SemanticEnter, Kind: kind, Range: node.Range, ContentRange: node.Range, Level: node.Level}); err != nil {
		return err
	}
	if err := emitSemanticInline(source, analysis, visit); err != nil {
		return err
	}
	return visit(parser.SemanticEvent{Phase: parser.SemanticExit, Kind: kind, Range: node.Range, Level: node.Level})
}

func emitSemanticPair(kind parser.SemanticKind, range_ parser.Range, fields parser.SemanticEvent, visit parser.SemanticVisitor) error {
	fields.Phase = parser.SemanticEnter
	fields.Kind = kind
	fields.Range = range_
	if err := visit(fields); err != nil {
		return err
	}
	return visit(parser.SemanticEvent{Phase: parser.SemanticExit, Kind: kind, Range: range_})
}

func emitSemanticList(source []byte, anchor int, projection semanticProjectionIndex, visit parser.SemanticVisitor) error {
	items := projection.listItems[anchor]
	if len(items) == 0 {
		return nil
	}
	range_ := items[0].Range
	if anchor >= 0 && anchor < range_.Start {
		range_.Start = anchor
	}
	for _, item := range items[1:] {
		if item.Range.End > range_.End {
			range_.End = item.Range.End
		}
	}
	first := items[0]
	if err := visit(parser.SemanticEvent{Phase: parser.SemanticEnter, Kind: parser.SemanticList, Range: range_, Ordered: first.Ordered, Marker: first.Marker}); err != nil {
		return err
	}
	for _, item := range items {
		if err := visit(parser.SemanticEvent{Phase: parser.SemanticEnter, Kind: parser.SemanticListItem, Range: item.Range, ContentRange: item.Range, Ordered: item.Ordered, Marker: item.Marker}); err != nil {
			return err
		}
		for _, task := range projection.tasksByStart[item.Range.Start] {
			if semanticRangeContains(item.Range, task.Range) {
				if err := visit(parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticTaskItem, Range: task.Range, Checked: task.Checked}); err != nil {
					return err
				}
			}
		}
		if analysis, ok := projection.analysis(item.Range); ok {
			if err := emitSemanticInline(source, analysis, visit); err != nil {
				return err
			}
		}
		if err := visit(parser.SemanticEvent{Phase: parser.SemanticExit, Kind: parser.SemanticListItem, Range: item.Range}); err != nil {
			return err
		}
	}
	return visit(parser.SemanticEvent{Phase: parser.SemanticExit, Kind: parser.SemanticList, Range: range_})
}

func emitSemanticTable(source []byte, anchor int, fallback parser.Range, projection semanticProjectionIndex, visit parser.SemanticVisitor) error {
	detail, ok := projection.tableDetails[anchor]
	if !ok {
		return emitSemanticPair(parser.SemanticTable, fallback, parser.SemanticEvent{}, visit)
	}
	cells := projection.tableCells[anchor]
	range_ := semanticTableRange(fallback, cells)
	if err := visit(parser.SemanticEvent{Phase: parser.SemanticEnter, Kind: parser.SemanticTable, Range: range_, Columns: detail.ColumnCount}); err != nil {
		return err
	}
	for first := 0; first < len(cells); {
		last := semanticTableRowEnd(cells, first)
		if err := emitSemanticTableRow(source, cells[first:last], detail, projection, visit); err != nil {
			return err
		}
		first = last
	}
	return visit(parser.SemanticEvent{Phase: parser.SemanticExit, Kind: parser.SemanticTable, Range: range_})
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

func emitSemanticTableRow(source []byte, row []parser.TableCellDetail, detail parser.TableDetail, projection semanticProjectionIndex, visit parser.SemanticVisitor) error {
	if len(row) == 0 {
		return nil
	}
	range_ := semanticTableRowRange(row)
	if err := visit(parser.SemanticEvent{Phase: parser.SemanticEnter, Kind: parser.SemanticTableRow, Range: range_, Header: row[0].Header, Columns: detail.ColumnCount}); err != nil {
		return err
	}
	for _, cell := range row {
		if err := emitSemanticTableCell(source, cell, detail, projection, visit); err != nil {
			return err
		}
	}
	return visit(parser.SemanticEvent{Phase: parser.SemanticExit, Kind: parser.SemanticTableRow, Range: range_})
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

func emitSemanticTableCell(source []byte, cell parser.TableCellDetail, detail parser.TableDetail, projection semanticProjectionIndex, visit parser.SemanticVisitor) error {
	alignment := semanticTableAlignment(detail.Alignments, cell.Column)
	enter := parser.SemanticEvent{Phase: parser.SemanticEnter, Kind: parser.SemanticTableCell, Range: cell.Range, ContentRange: cell.Range, Header: cell.Header, Column: cell.Column, Alignment: alignment}
	if err := visit(enter); err != nil {
		return err
	}
	if analysis, found := projection.analysis(cell.Range); found {
		if err := emitSemanticInline(source, analysis, visit); err != nil {
			return err
		}
	}
	return visit(parser.SemanticEvent{Phase: parser.SemanticExit, Kind: parser.SemanticTableCell, Range: cell.Range})
}

func semanticTableAlignment(alignments []parser.TableAlignment, column int) parser.TableAlignment {
	if column < 0 || column >= len(alignments) {
		return parser.TableAlignmentDefault
	}
	return alignments[column]
}

func emitSemanticFencedCode(source []byte, node parser.Node, projection semanticProjectionIndex, visit parser.SemanticVisitor) error {
	detail, ok := projection.fencedCode[node.Anchor]
	if !ok {
		return visit(parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticCodeBlock, Range: node.Range, Fenced: true})
	}
	content := parser.Range{}
	if len(detail.ContentRanges) == 1 {
		content = detail.ContentRanges[0]
	}
	return visit(parser.SemanticEvent{
		Phase:        parser.SemanticLeaf,
		Kind:         parser.SemanticCodeBlock,
		Range:        node.Range,
		ContentRange: content,
		Value:        semanticRangesValue(source, detail.ContentRanges),
		Info:         detail.Info,
		Language:     detail.Language,
		Fenced:       true,
	})
}

func emitSemanticSupplemental(source []byte, blocks blockParseResult, analyses []inlineAnalysis, visit parser.SemanticVisitor) error {
	definitions, references, _ := nativeFootnoteObservations(source, blocks)
	for _, definition := range definitions {
		range_ := parser.Range{Start: definition.Anchor, End: definition.Anchor}
		if len(definition.BodyRanges) != 0 {
			range_.End = definition.BodyRanges[len(definition.BodyRanges)-1].End
		}
		if err := emitSemanticPair(parser.SemanticFootnoteDefinition, range_, parser.SemanticEvent{Label: definition.Label, ContentRange: semanticRangesEnvelope(definition.BodyRanges)}, visit); err != nil {
			return err
		}
	}
	for _, reference := range references {
		if err := visit(parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticFootnoteReference, Range: reference.Range, ContentRange: reference.LabelRange, Label: reference.Label, DefinitionAnchor: reference.DefinitionAnchor, Occurrence: reference.Occurrence}); err != nil {
			return err
		}
	}
	inline := mergeInlineAnalyses(analyses, len(blocks.nodes))
	for _, expression := range nativeMathExpressionObservations(source, blocks, analyses, inline.nodes) {
		if err := visit(parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticMath, Range: expression.Range, ContentRange: expression.PayloadRange, Value: semanticSourceValue(source, expression.PayloadRange), MathStyle: expression.Style}); err != nil {
			return err
		}
	}
	return nil
}

func semanticRangesEnvelope(ranges []parser.Range) parser.Range {
	if len(ranges) == 0 {
		return parser.Range{}
	}
	return parser.Range{Start: ranges[0].Start, End: ranges[len(ranges)-1].End}
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

func semanticRangeContains(outer, inner parser.Range) bool {
	return inner.Start >= outer.Start && inner.End <= outer.End
}

func emitSemanticInline(source []byte, analysis inlineAnalysis, visit parser.SemanticVisitor) error {
	if len(analysis.block.segments) == 0 {
		return nil
	}
	nodes := collectConstructionSemantics(analysis.owners, analysis.delimiters)
	assignConstructionParents(nodes)
	children := make([][]int, len(nodes)+1)
	for index, node := range nodes {
		children[node.parent+1] = append(children[node.parent+1], index)
	}
	first := analysis.block.segments[0].Start
	last := analysis.block.segments[len(analysis.block.segments)-1].End
	return emitSemanticInlineRegion(source, analysis.block, nodes, children, -1, parser.Range{Start: first, End: last}, visit)
}

func emitSemanticInlineRegion(source []byte, block inlineBlock, nodes []constructionSemantic, children [][]int, parent int, region parser.Range, visit parser.SemanticVisitor) error {
	cursor := region.Start
	for _, child := range children[parent+1] {
		node := nodes[child]
		if node.syntax.Start < cursor || node.syntax.End > region.End {
			continue
		}
		if err := emitSemanticTextGap(source, block, parser.Range{Start: cursor, End: node.syntax.Start}, visit); err != nil {
			return err
		}
		if err := emitSemanticInlineNode(source, block, nodes, children, child, visit); err != nil {
			return err
		}
		cursor = node.syntax.End
	}
	return emitSemanticTextGap(source, block, parser.Range{Start: cursor, End: region.End}, visit)
}

func emitSemanticInlineNode(source []byte, block inlineBlock, nodes []constructionSemantic, children [][]int, index int, visit parser.SemanticVisitor) error {
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
		if err := emitSemanticInlineRegion(source, block, nodes, children, index, node.content, visit); err != nil {
			return err
		}
		return visit(parser.SemanticEvent{Phase: parser.SemanticExit, Kind: kind, Range: node.syntax})
	case parser.SemanticCodeSpan, parser.SemanticRawHTML, parser.SemanticAutoLink:
		value := semanticSourceValue(source, node.content)
		event := parser.SemanticEvent{
			Phase:        parser.SemanticLeaf,
			Kind:         kind,
			Range:        node.syntax,
			ContentRange: node.content,
			Value:        value,
		}
		if kind == parser.SemanticAutoLink {
			event.Destination = value
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

func emitSemanticTextGap(source []byte, block inlineBlock, gap parser.Range, visit parser.SemanticVisitor) error {
	if gap.Start >= gap.End {
		return nil
	}
	for index, segment := range block.segments {
		if segment.End <= gap.Start || segment.Start >= gap.End {
			continue
		}
		start := max(gap.Start, segment.Start)
		end := min(gap.End, segment.End)
		breakKind := parser.SemanticSoftBreak
		textEnd := segment.End
		if index+1 < len(block.segments) {
			textEnd, breakKind = semanticLineEnd(source, segment)
		}
		if end > textEnd {
			end = textEnd
		}
		if start < end {
			value := semanticDecodedText(source[start:end])
			if value != "" {
				if err := visit(parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Range: parser.Range{Start: start, End: end}, ContentRange: parser.Range{Start: start, End: end}, Value: value}); err != nil {
					return err
				}
			}
		}
		if index+1 >= len(block.segments) || gap.End < block.segments[index+1].Start || gap.Start > segment.End {
			continue
		}
		breakRange := semanticBreakRange(source, segment, textEnd, breakKind)
		if err := visit(parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: breakKind, Range: breakRange}); err != nil {
			return err
		}
	}
	return nil
}

func semanticLineEnd(source []byte, segment parser.Range) (int, parser.SemanticKind) {
	end := segment.End
	if end > segment.Start && source[end-1] == '\\' && !inlineByteEscaped(source, segment.Start, end-1) {
		return end - 1, parser.SemanticHardBreak
	}
	spaces := 0
	for position := end; position > segment.Start && source[position-1] == ' '; position-- {
		spaces++
	}
	if spaces >= 2 {
		return end - spaces, parser.SemanticHardBreak
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
