package splice

import (
	"errors"
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

type tableRowSourceResult struct {
	mapping  source.TableRowMapping
	editable bool
}

type parserNodeDetails struct {
	blockquotes []parser.BlockquoteDetail
	fencedCode  []parser.FencedCodeDetail
	tables      []parser.TableDetail
	tableRows   []parser.TableRowDetail
	tableCells  []parser.TableCellDetail
}

func (d parserNodeDetails) blockquote(observation parser.Node) (parser.BlockquoteDetail, error) {
	if !observation.TopLevel || observation.DetailIndex == 0 || uint64(observation.DetailIndex) > uint64(len(d.blockquotes)) {
		return parser.BlockquoteDetail{}, fmt.Errorf("blockquote parser detail index is invalid")
	}
	detail := d.blockquotes[int(observation.DetailIndex)-1]
	if detail.Anchor != observation.Range.Start {
		return parser.BlockquoteDetail{}, fmt.Errorf("blockquote parser detail anchor %d disagrees with node start %d", detail.Anchor, observation.Range.Start)
	}
	return detail, nil
}

func (d parserNodeDetails) fenced(observation parser.Node) (parser.FencedCodeDetail, error) {
	if observation.DetailIndex == 0 || uint64(observation.DetailIndex) > uint64(len(d.fencedCode)) {
		return parser.FencedCodeDetail{}, fmt.Errorf("fenced-code parser detail index is invalid")
	}
	detail := d.fencedCode[int(observation.DetailIndex)-1]
	if detail.Anchor != observation.Anchor {
		return parser.FencedCodeDetail{}, fmt.Errorf("fenced-code parser detail anchor %d disagrees with node anchor %d", detail.Anchor, observation.Anchor)
	}
	return detail, nil
}

func (d parserNodeDetails) table(observation parser.Node) (parser.TableDetail, error) {
	if observation.DetailIndex == 0 || uint64(observation.DetailIndex) > uint64(len(d.tables)) {
		return parser.TableDetail{}, fmt.Errorf("table parser detail index is invalid")
	}
	detail := d.tables[int(observation.DetailIndex)-1]
	if detail.Anchor != observation.Range.Start {
		return parser.TableDetail{}, fmt.Errorf("table parser detail anchor %d disagrees with node start %d", detail.Anchor, observation.Range.Start)
	}
	return detail, nil
}

func (d parserNodeDetails) tableRow(observation parser.Node) (parser.TableRowDetail, error) {
	if observation.DetailIndex == 0 || uint64(observation.DetailIndex) > uint64(len(d.tableRows)) {
		return parser.TableRowDetail{}, fmt.Errorf("table-row parser detail index is invalid")
	}
	detail := d.tableRows[int(observation.DetailIndex)-1]
	if detail.RowAnchor != observation.Range.Start {
		return parser.TableRowDetail{}, fmt.Errorf("table-row parser detail anchor %d disagrees with node start %d", detail.RowAnchor, observation.Range.Start)
	}
	return detail, nil
}

func (d parserNodeDetails) tableCell(observation parser.Node) (parser.TableCellDetail, error) {
	if observation.DetailIndex == 0 || uint64(observation.DetailIndex) > uint64(len(d.tableCells)) {
		return parser.TableCellDetail{}, fmt.Errorf("table-cell parser detail index is invalid")
	}
	detail := d.tableCells[int(observation.DetailIndex)-1]
	if detail.Range != observation.Range {
		return parser.TableCellDetail{}, fmt.Errorf("table-cell parser detail range %v disagrees with node range %v", detail.Range, observation.Range)
	}
	return detail, nil
}

func sourceDetailCapacities(observations []parser.Node) (fenced, blockquote int) {
	for _, observation := range observations {
		switch observation.Kind {
		case parser.KindFencedCode:
			fenced++
		case parser.KindBlockquote:
			blockquote++
		}
	}
	return fenced, blockquote
}

func nodeFromObservation(snapshot []byte, fingerprint source.Fingerprint, observation parser.Node, parserDetails parserNodeDetails, tableRows map[int]tableRowSourceResult, tableSources map[int]source.TableMapping, fencedSources *[]fencedSourceDetail, blockquoteSources *[]source.BlockquoteMapping) (Node, error) {
	if !parserKindUsesSparseDetail(observation.Kind) && observation.DetailIndex != 0 {
		return Node{}, fmt.Errorf("semantic node kind %d has unexpected parser detail index", observation.Kind)
	}
	if observation.Kind == parser.KindRawHTML {
		return nodeFromRawHTMLObservation(snapshot, fingerprint, observation)
	}
	kind, err := mapKind(observation.Kind)
	if err != nil {
		return Node{}, err
	}

	contentRange := Range{Start: observation.Range.Start, End: observation.Range.End}
	if !contentRange.Valid(len(snapshot)) {
		return Node{}, fmt.Errorf("semantic node range [%d,%d) is outside source length %d", contentRange.Start, contentRange.End, len(snapshot))
	}

	node := baseNodeFromObservation(kind, contentRange, observation)
	if err := mapBlockNodeSource(snapshot, observation, contentRange, parserDetails, tableRows, tableSources, fencedSources, blockquoteSources, &node); err != nil {
		return Node{}, err
	}
	if err := mapInlineNodeSource(snapshot, observation, contentRange, &node); err != nil {
		return Node{}, err
	}
	node.ID = makeNodeID(fingerprint, kind, node.Range)
	return node, nil
}

func parserKindUsesSparseDetail(kind parser.Kind) bool {
	switch kind {
	case parser.KindFencedCode, parser.KindBlockquote, parser.KindTable, parser.KindTableRow, parser.KindTableCell:
		return true
	default:
		return false
	}
}

func baseNodeFromObservation(kind Kind, contentRange Range, observation parser.Node) Node {
	node := Node{
		Kind:                 kind,
		Range:                contentRange,
		ContentRange:         contentRange,
		Level:                observation.Level,
		HeadingText:          observation.HeadingText,
		Checked:              observation.Checked,
		ListOrdered:          observation.Ordered,
		ListMarker:           observation.Marker,
		ListHasParent:        observation.HasListParent,
		ListParentAnchor:     observation.ListParentAnchor,
		ListContainerAnchor:  observation.ListContainerAnchor,
		ListHasChildren:      observation.HasListChildren,
		ListDirectChildCount: observation.ListDirectChildCount,
		Anchor:               observation.Anchor,
		Destination:          observation.Destination,
		Label:                observation.Label,
		Title:                observation.Title,
		HasTitle:             observation.HasTitle,
		Value:                observation.Value,
		AutoLinkEmail:        observation.AutoLinkEmail,
		TopLevel:             observation.TopLevel,
	}
	if kind == KindParagraph && observation.TopLevel {
		node.Editable = true
	}
	return node
}

func mapBlockNodeSource(snapshot []byte, observation parser.Node, contentRange Range, parserDetails parserNodeDetails, tableRows map[int]tableRowSourceResult, tableSources map[int]source.TableMapping, fencedSources *[]fencedSourceDetail, blockquoteSources *[]source.BlockquoteMapping, node *Node) error {
	switch node.Kind {
	case KindHeading:
		return mapHeadingNodeSource(snapshot, observation, contentRange, node)
	case KindTask:
		return mapTaskNodeSource(snapshot, observation, node)
	case KindListItem:
		return mapListItemNodeSource(snapshot, observation, contentRange, node)
	case KindTableCell:
		return mapTableCellNodeSource(snapshot, observation, contentRange, parserDetails, tableRows, node)
	case KindTableRow:
		return mapTableRowNodeSource(snapshot, observation, parserDetails, tableRows, node)
	case KindTable:
		return mapTableNodeSource(snapshot, observation, parserDetails, tableSources, node)
	case KindFencedCode:
		return mapFencedCodeNodeSource(snapshot, observation, contentRange, parserDetails, fencedSources, node)
	case KindThematicBreak:
		return mapThematicBreakNodeSource(snapshot, observation, contentRange, node)
	case KindBlockquote:
		return mapBlockquoteNodeSource(snapshot, observation, parserDetails, blockquoteSources, node)
	default:
		return nil
	}
}

func mapInlineNodeSource(snapshot []byte, observation parser.Node, contentRange Range, node *Node) error {
	switch node.Kind {
	case KindStrikethrough:
		return mapStrikethroughNodeSource(snapshot, contentRange, node)
	case KindInlineLink:
		return mapInlineLinkNodeSource(snapshot, observation, contentRange, node)
	case KindImage:
		return mapImageNodeSource(snapshot, observation, contentRange, node)
	case KindReferenceDefinition:
		return mapReferenceDefinitionNodeSource(snapshot, observation, contentRange, node)
	case KindAutoLink:
		return mapAutoLinkNodeSource(snapshot, observation, contentRange, node)
	case KindCodeSpan:
		return mapCodeSpanNodeSource(snapshot, observation, contentRange, node)
	case KindEmphasis, KindStrong:
		return mapEmphasisNodeSource(snapshot, observation, contentRange, node)
	default:
		return nil
	}
}

func mapHeadingNodeSource(snapshot []byte, observation parser.Node, contentRange Range, node *Node) error {
	mapping, err := source.MapTopLevelHeading(snapshot, contentRange, observation.Level)
	if err != nil {
		return fmt.Errorf("map heading source: %w", err)
	}
	node.Range = mapping.Range
	node.ContentRange = mapping.ContentRange
	node.HeadingStyle = HeadingStyle(mapping.Style)
	node.Editable = true
	return nil
}

func mapTaskNodeSource(snapshot []byte, observation parser.Node, node *Node) error {
	mapping, err := source.MapTaskMarker(snapshot, observation.Range.Start)
	if err != nil {
		return fmt.Errorf("map task marker: %w", err)
	}
	if mapping.Checked != observation.Checked {
		return fmt.Errorf("map task marker: semantic checked state %v disagrees with source state %v", observation.Checked, mapping.Checked)
	}
	node.Range = mapping.Range
	node.ContentRange = mapping.ContentRange
	node.Checked = mapping.Checked
	node.Editable = true
	return nil
}

func mapListItemNodeSource(snapshot []byte, observation parser.Node, contentRange Range, node *Node) error {
	mapping, err := source.MapSingleLineListItem(snapshot, contentRange, observation.Ordered, observation.Marker)
	if err != nil {
		if errors.Is(err, source.ErrUnsupportedListItemShape) {
			return nil
		}
		return fmt.Errorf("map list item source: %w", err)
	}
	node.Range = mapping.Range
	node.ContentRange = mapping.ContentRange
	node.ListOrdered = mapping.Ordered
	node.ListMarker = mapping.Marker
	node.ListItemLineRange = mapping.LineRange
	node.Editable = true
	return nil
}

func mapTableRowNodeSource(snapshot []byte, observation parser.Node, parserDetails parserNodeDetails, tableRows map[int]tableRowSourceResult, node *Node) error {
	detail, err := parserDetails.tableRow(observation)
	if err != nil {
		return fmt.Errorf("map table row source: %w", err)
	}
	mapping, editable, err := mapTableRowSource(snapshot, detail.RowAnchor, tableRows)
	if err != nil {
		return fmt.Errorf("map table row source: %w", err)
	}
	node.TableRowAnchor = detail.RowAnchor
	node.TableAnchor = detail.TableAnchor
	node.TableColumnCount = detail.ColumnCount
	node.TableAlignments = append([]TableAlignment(nil), detail.Alignments...)
	if !editable || detail.ColumnCount <= 0 || len(mapping.Cells) != detail.ColumnCount || len(detail.Alignments) != detail.ColumnCount {
		return nil
	}
	node.Range = mapping.LineRange
	node.ContentRange = mapping.Range
	node.TableRowSourceAnchor = mapping.Anchor
	node.Editable = true
	return nil
}

func mapTableCellNodeSource(snapshot []byte, observation parser.Node, contentRange Range, parserDetails parserNodeDetails, tableRows map[int]tableRowSourceResult, node *Node) error {
	detail, err := parserDetails.tableCell(observation)
	if err != nil {
		return fmt.Errorf("map table cell source: %w", err)
	}
	node.TableHeader = detail.Header
	node.TableColumn = detail.Column
	node.TableRowAnchor = detail.RowAnchor
	node.TableAnchor = detail.TableAnchor
	mapping, editable, err := mapTableCellSource(snapshot, detail, contentRange, tableRows)
	if err != nil {
		return fmt.Errorf("map table cell source: %w", err)
	}
	if editable {
		node.TableCellRange = mapping.Range
		node.Editable = true
	}
	return nil
}

func mapThematicBreakNodeSource(snapshot []byte, observation parser.Node, contentRange Range, node *Node) error {
	if !observation.TopLevel {
		return nil
	}
	mapping, err := source.MapTopLevelThematicBreak(snapshot, contentRange)
	if err != nil {
		if errors.Is(err, source.ErrUnsupportedThematicBreakShape) {
			return nil
		}
		return fmt.Errorf("map thematic break source: %w", err)
	}
	if mapping.Range != contentRange {
		return fmt.Errorf("map thematic break source: mapped range %v disagrees with semantic range %v", mapping.Range, contentRange)
	}
	node.Editable = true
	return nil
}

func mapBlockquoteNodeSource(snapshot []byte, observation parser.Node, parserDetails parserNodeDetails, blockquoteSources *[]source.BlockquoteMapping, node *Node) error {
	if !observation.TopLevel {
		if observation.DetailIndex != 0 {
			return fmt.Errorf("map blockquote source: non-top-level node has parser detail")
		}
		return nil
	}
	parserDetail, err := parserDetails.blockquote(observation)
	if err != nil {
		return fmt.Errorf("map blockquote source: %w", err)
	}
	semanticRanges := rangesFromParser(parserDetail.SemanticRanges)
	mapping, err := source.MapTopLevelBlockquote(snapshot, node.Range, semanticRanges)
	if err != nil {
		if errors.Is(err, source.ErrUnsupportedBlockquoteShape) {
			return nil
		}
		return fmt.Errorf("map blockquote source: %w", err)
	}
	legacyContent := Range{Start: parserDetail.ContentRange.Start, End: parserDetail.ContentRange.End}
	if !legacyContent.Valid(len(snapshot)) || legacyContent.Start == legacyContent.End || len(mapping.ContentRanges) != 1 || mapping.ContentRanges[0] != legacyContent {
		legacyContent = Range{}
	}
	mapping.ContentRange = legacyContent
	index, ok := appendSourceDetail(blockquoteSources, mapping)
	if !ok {
		return fmt.Errorf("map blockquote source: sidecar capacity exceeded")
	}
	node.ContentRange = legacyContent
	node.SourceDetailIndex = index
	node.Editable = true
	return nil
}

func mapFencedCodeNodeSource(snapshot []byte, observation parser.Node, contentRange Range, parserDetails parserNodeDetails, fencedSources *[]fencedSourceDetail, node *Node) error {
	parserDetail, err := parserDetails.fenced(observation)
	if err != nil {
		return fmt.Errorf("map fenced code source: %w", err)
	}
	detail := fencedSourceDetail{}
	hasDetail := false
	if observation.TopLevel {
		contentRanges := rangesFromParser(parserDetail.ContentRanges)
		mapping, err := source.MapFencedBlock(snapshot, observation.Anchor, contentRanges, parserDetail.Info)
		if err == nil {
			detail.block = mapping
			detail.info = parserDetail.Info
			detail.language = parserDetail.Language
			hasDetail = true
		} else if !errors.Is(err, source.ErrUnsupportedFencedCodeShape) {
			return fmt.Errorf("map fenced block source: %w", err)
		}
	}

	mapping, err := source.MapFencedCode(snapshot, contentRange)
	if err == nil {
		detail.code = mapping
		hasDetail = true
		node.Editable = true
	} else if !errors.Is(err, source.ErrUnsupportedFencedCodeShape) {
		return fmt.Errorf("map fenced code source: %w", err)
	}
	if hasDetail {
		index, ok := appendSourceDetail(fencedSources, detail)
		if !ok {
			return fmt.Errorf("map fenced source: sidecar capacity exceeded")
		}
		node.SourceDetailIndex = index
	}
	return nil
}

func mapStrikethroughNodeSource(snapshot []byte, contentRange Range, node *Node) error {
	_, err := source.MapSimpleStrikethrough(snapshot, contentRange)
	if err == nil {
		node.Editable = true
		return nil
	}
	if errors.Is(err, source.ErrUnsupportedStrikethroughShape) {
		return nil
	}
	return fmt.Errorf("map strikethrough source: %w", err)
}

func mapInlineLinkNodeSource(snapshot []byte, observation parser.Node, contentRange Range, node *Node) error {
	mapping, err := source.MapSimpleInlineLink(snapshot, observation.Anchor, contentRange, observation.Destination, observation.Title, observation.HasTitle)
	if err == nil {
		node.ContentRange = mapping.DestinationRange
		node.Editable = true
		return nil
	}
	if errors.Is(err, source.ErrUnsupportedInlineLinkShape) {
		return nil
	}
	return fmt.Errorf("map inline link source: %w", err)
}

func mapImageNodeSource(snapshot []byte, observation parser.Node, contentRange Range, node *Node) error {
	mapping, err := source.MapSimpleImage(snapshot, observation.Anchor, contentRange)
	if err == nil {
		node.ContentRange = mapping.DestinationRange
		node.Editable = true
		return nil
	}
	if errors.Is(err, source.ErrUnsupportedImageShape) {
		return nil
	}
	return fmt.Errorf("map image source: %w", err)
}

func mapReferenceDefinitionNodeSource(snapshot []byte, observation parser.Node, contentRange Range, node *Node) error {
	mapping, err := source.MapSingleLineReferenceDefinition(snapshot, contentRange, observation.Label, observation.Destination, observation.Title, observation.HasTitle)
	if err == nil {
		node.ContentRange = mapping.DestinationRange
		node.Editable = true
		return nil
	}
	if errors.Is(err, source.ErrUnsupportedReferenceDefinitionShape) {
		return nil
	}
	return fmt.Errorf("map reference definition source: %w", err)
}

func mapAutoLinkNodeSource(snapshot []byte, observation parser.Node, contentRange Range, node *Node) error {
	mapping, err := source.MapAutoLink(snapshot, observation.Anchor, contentRange, observation.Value, observation.AutoLinkEmail)
	if err == nil {
		node.ContentRange = mapping.ContentRange
		node.Editable = true
		return nil
	}
	if errors.Is(err, source.ErrUnsupportedAutoLinkShape) {
		return nil
	}
	return fmt.Errorf("map autolink source: %w", err)
}

func mapCodeSpanNodeSource(snapshot []byte, observation parser.Node, contentRange Range, node *Node) error {
	_, err := source.MapSimpleCodeSpan(snapshot, observation.Anchor, contentRange)
	if err == nil {
		node.Editable = true
		return nil
	}
	if errors.Is(err, source.ErrUnsupportedCodeSpanShape) {
		return nil
	}
	return fmt.Errorf("map code span source: %w", err)
}

func mapEmphasisNodeSource(snapshot []byte, observation parser.Node, contentRange Range, node *Node) error {
	_, err := source.MapSimpleEmphasis(snapshot, observation.Anchor, contentRange, observation.Level)
	if err == nil {
		node.Editable = true
		return nil
	}
	if errors.Is(err, source.ErrUnsupportedEmphasisShape) {
		return nil
	}
	return fmt.Errorf("map emphasis source: %w", err)
}

func rangesFromParser(values []parser.Range) []Range {
	result := make([]Range, len(values))
	for index, value := range values {
		result[index] = Range{Start: value.Start, End: value.End}
	}
	return result
}

func mapTableRowSource(snapshot []byte, anchor int, cache map[int]tableRowSourceResult) (source.TableRowMapping, bool, error) {
	result, ok := cache[anchor]
	if !ok {
		row, err := source.MapTableRow(snapshot, anchor)
		if err != nil {
			if errors.Is(err, source.ErrUnsupportedTableCellShape) {
				cache[anchor] = tableRowSourceResult{}
				return source.TableRowMapping{}, false, nil
			}
			return source.TableRowMapping{}, false, err
		}
		result = tableRowSourceResult{mapping: row, editable: true}
		cache[anchor] = result
	}
	return result.mapping, result.editable, nil
}

func mapTableCellSource(snapshot []byte, detail parser.TableCellDetail, contentRange Range, cache map[int]tableRowSourceResult) (source.TableCellMapping, bool, error) {
	row, editable, err := mapTableRowSource(snapshot, detail.RowAnchor, cache)
	if err != nil {
		return source.TableCellMapping{}, false, err
	}
	if !editable || detail.Column < 0 || detail.Column >= len(row.Cells) {
		return source.TableCellMapping{}, false, nil
	}
	mapping := row.Cells[detail.Column]
	if mapping.ContentRange != contentRange {
		return source.TableCellMapping{}, false, nil
	}
	return mapping, true, nil
}
