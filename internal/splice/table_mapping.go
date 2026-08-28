package splice

import (
	"errors"
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

func mapTableNodeSource(snapshot []byte, observation parser.Node, parserDetails parserNodeDetails, tableSources map[int]source.TableMapping, node *Node) error {
	detail, err := parserDetails.table(observation)
	if err != nil {
		return fmt.Errorf("map table source: %w", err)
	}
	mapping, err := source.MapTable(snapshot, detail.Anchor, detail.BodyRowCount, detail.LastBodyRowAnchor)
	if err != nil {
		if errors.Is(err, source.ErrUnsupportedTableShape) {
			return nil
		}
		return fmt.Errorf("map table source: %w", err)
	}
	if detail.Anchor != mapping.Range.Start || detail.ColumnCount <= 0 || len(detail.Alignments) != detail.ColumnCount || len(mapping.Header.Cells) != detail.ColumnCount || len(mapping.Delimiter.Cells) != detail.ColumnCount || len(mapping.DelimiterAlignments) != detail.ColumnCount {
		return nil
	}
	for index, alignment := range detail.Alignments {
		if !tableDelimiterAlignmentMatches(alignment, mapping.DelimiterAlignments[index]) {
			return nil
		}
	}
	node.Range = mapping.Range
	node.ContentRange = mapping.Range
	node.TableAnchor = detail.Anchor
	node.TableColumnCount = detail.ColumnCount
	node.TableAlignments = append([]TableAlignment(nil), detail.Alignments...)
	node.TableBodyRowCount = detail.BodyRowCount
	node.TableLastBodyRowAnchor = detail.LastBodyRowAnchor
	if _, exists := tableSources[detail.Anchor]; exists {
		return fmt.Errorf("map table source: duplicate table anchor %d", detail.Anchor)
	}
	tableSources[detail.Anchor] = mapping
	node.Editable = true
	return nil
}

func tableDelimiterAlignmentMatches(semantic TableAlignment, lexical source.TableDelimiterAlignment) bool {
	switch semantic {
	case TableAlignmentDefault:
		return lexical == source.TableDelimiterAlignmentDefault
	case TableAlignmentLeft:
		return lexical == source.TableDelimiterAlignmentLeft
	case TableAlignmentRight:
		return lexical == source.TableDelimiterAlignmentRight
	case TableAlignmentCenter:
		return lexical == source.TableDelimiterAlignmentCenter
	default:
		return false
	}
}
