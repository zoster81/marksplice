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

func nodeFromObservation(snapshot []byte, fingerprint source.Fingerprint, observation parser.Node, tableRows map[int]tableRowSourceResult) (Node, error) {
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
	if err := mapBlockNodeSource(snapshot, observation, contentRange, tableRows, &node); err != nil {
		return Node{}, err
	}
	if err := mapInlineNodeSource(snapshot, observation, contentRange, &node); err != nil {
		return Node{}, err
	}
	node.ID = makeNodeID(fingerprint, kind, node.Range)
	return node, nil
}

func baseNodeFromObservation(kind Kind, contentRange Range, observation parser.Node) Node {
	node := Node{
		Kind:                   kind,
		Range:                  contentRange,
		ContentRange:           contentRange,
		Level:                  observation.Level,
		Checked:                observation.Checked,
		ListOrdered:            observation.Ordered,
		ListMarker:             observation.Marker,
		ListHasParent:          observation.HasListParent,
		ListParentAnchor:       observation.ListParentAnchor,
		ListContainerAnchor:    observation.ListContainerAnchor,
		ListHasChildren:        observation.HasListChildren,
		ListDirectChildCount:   observation.ListDirectChildCount,
		TableHeader:            observation.TableHeader,
		TableColumn:            observation.TableColumn,
		TableRowAnchor:         observation.TableRowAnchor,
		TableAnchor:            observation.TableAnchor,
		TableColumnCount:       observation.TableColumnCount,
		TableAlignments:        append([]TableAlignment(nil), observation.TableAlignments...),
		TableBodyRowCount:      observation.TableBodyRowCount,
		TableLastBodyRowAnchor: observation.TableLastBodyRowAnchor,
		Anchor:                 observation.Anchor,
		Destination:            observation.Destination,
		Label:                  observation.Label,
		Title:                  observation.Title,
		HasTitle:               observation.HasTitle,
		Value:                  observation.Value,
		AutoLinkEmail:          observation.AutoLinkEmail,
		TopLevel:               observation.TopLevel,
	}
	if kind == KindParagraph && observation.TopLevel {
		node.Editable = true
	}
	return node
}

func mapBlockNodeSource(snapshot []byte, observation parser.Node, contentRange Range, tableRows map[int]tableRowSourceResult, node *Node) error {
	switch node.Kind {
	case KindHeading:
		return mapHeadingNodeSource(snapshot, observation, contentRange, node)
	case KindTask:
		return mapTaskNodeSource(snapshot, observation, node)
	case KindListItem:
		return mapListItemNodeSource(snapshot, observation, contentRange, node)
	case KindTableCell:
		return mapTableCellNodeSource(snapshot, observation, contentRange, tableRows, node)
	case KindTableRow:
		return mapTableRowNodeSource(snapshot, observation, tableRows, node)
	case KindTable:
		return mapTableNodeSource(snapshot, observation, node)
	case KindFencedCode:
		return mapFencedCodeNodeSource(snapshot, contentRange, node)
	case KindThematicBreak:
		return mapThematicBreakNodeSource(snapshot, observation, contentRange, node)
	case KindBlockquote:
		return mapBlockquoteNodeSource(snapshot, observation, node)
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
		return fmt.Errorf("map list item source: %w", err)
	}
	node.Range = mapping.Range
	node.ContentRange = mapping.ContentRange
	node.ListOrdered = mapping.Ordered
	node.ListMarker = mapping.Marker
	node.ListItemSource = mapping
	node.Editable = true
	return nil
}

func mapTableRowNodeSource(snapshot []byte, observation parser.Node, tableRows map[int]tableRowSourceResult, node *Node) error {
	mapping, editable, err := mapTableRowSource(snapshot, observation.TableRowAnchor, tableRows)
	if err != nil {
		return fmt.Errorf("map table row source: %w", err)
	}
	if !editable || observation.TableColumnCount <= 0 || len(mapping.Cells) != observation.TableColumnCount || len(observation.TableAlignments) != observation.TableColumnCount {
		return nil
	}
	node.Range = mapping.LineRange
	node.ContentRange = mapping.Range
	node.TableAnchor = observation.TableAnchor
	node.TableColumnCount = observation.TableColumnCount
	node.TableRowSource = mapping
	node.Editable = true
	return nil
}

func mapTableCellNodeSource(snapshot []byte, observation parser.Node, contentRange Range, tableRows map[int]tableRowSourceResult, node *Node) error {
	mapping, editable, err := mapTableCellSource(snapshot, observation, contentRange, tableRows)
	if err != nil {
		return fmt.Errorf("map table cell source: %w", err)
	}
	if editable {
		node.TableCellSource = mapping
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
	node.ThematicBreakSource = mapping
	node.Editable = true
	return nil
}

func mapBlockquoteNodeSource(snapshot []byte, observation parser.Node, node *Node) error {
	if !observation.TopLevel {
		return nil
	}
	semanticRanges := make([]Range, len(observation.BlockquoteSemanticRanges))
	for index, range_ := range observation.BlockquoteSemanticRanges {
		semanticRanges[index] = Range{Start: range_.Start, End: range_.End}
	}
	mapping, err := source.MapTopLevelBlockquote(snapshot, node.Range, semanticRanges)
	if err != nil {
		if errors.Is(err, source.ErrUnsupportedBlockquoteShape) {
			return nil
		}
		return fmt.Errorf("map blockquote source: %w", err)
	}
	legacyContent := Range{Start: observation.BlockquoteContentRange.Start, End: observation.BlockquoteContentRange.End}
	if !legacyContent.Valid(len(snapshot)) || legacyContent.Start == legacyContent.End || len(mapping.ContentRanges) != 1 || mapping.ContentRanges[0] != legacyContent {
		legacyContent = Range{}
	}
	mapping.ContentRange = legacyContent
	node.ContentRange = legacyContent
	node.BlockquoteSource = mapping
	node.Editable = true
	return nil
}

func mapFencedCodeNodeSource(snapshot []byte, contentRange Range, node *Node) error {
	mapping, err := source.MapFencedCode(snapshot, contentRange)
	if err == nil {
		node.FencedCodeSource = mapping
		node.Editable = true
		return nil
	}
	if errors.Is(err, source.ErrUnsupportedFencedCodeShape) {
		return nil
	}
	return fmt.Errorf("map fenced code source: %w", err)
}

func mapStrikethroughNodeSource(snapshot []byte, contentRange Range, node *Node) error {
	mapping, err := source.MapSimpleStrikethrough(snapshot, contentRange)
	if err == nil {
		node.StrikethroughSource = mapping
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
		node.InlineLinkSource = mapping
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
		node.ImageSource = mapping
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
		node.ReferenceDefinitionSource = mapping
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
		node.AutoLinkSource = mapping
		node.Editable = true
		return nil
	}
	if errors.Is(err, source.ErrUnsupportedAutoLinkShape) {
		return nil
	}
	return fmt.Errorf("map autolink source: %w", err)
}

func mapCodeSpanNodeSource(snapshot []byte, observation parser.Node, contentRange Range, node *Node) error {
	mapping, err := source.MapSimpleCodeSpan(snapshot, observation.Anchor, contentRange)
	if err == nil {
		node.CodeSpanSource = mapping
		node.Editable = true
		return nil
	}
	if errors.Is(err, source.ErrUnsupportedCodeSpanShape) {
		return nil
	}
	return fmt.Errorf("map code span source: %w", err)
}

func mapEmphasisNodeSource(snapshot []byte, observation parser.Node, contentRange Range, node *Node) error {
	mapping, err := source.MapSimpleEmphasis(snapshot, observation.Anchor, contentRange, observation.Level)
	if err == nil {
		node.EmphasisSource = mapping
		node.Editable = true
		return nil
	}
	if errors.Is(err, source.ErrUnsupportedEmphasisShape) {
		return nil
	}
	return fmt.Errorf("map emphasis source: %w", err)
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

func mapTableCellSource(snapshot []byte, observation parser.Node, contentRange Range, cache map[int]tableRowSourceResult) (source.TableCellMapping, bool, error) {
	row, editable, err := mapTableRowSource(snapshot, observation.TableRowAnchor, cache)
	if err != nil {
		return source.TableCellMapping{}, false, err
	}
	if !editable || observation.TableColumn < 0 || observation.TableColumn >= len(row.Cells) {
		return source.TableCellMapping{}, false, nil
	}
	mapping := row.Cells[observation.TableColumn]
	if mapping.ContentRange != contentRange {
		return source.TableCellMapping{}, false, nil
	}
	return mapping, true, nil
}
