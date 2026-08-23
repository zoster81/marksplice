package splice

import (
	"errors"
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

func mapTableNodeSource(snapshot []byte, observation parser.Node, node *Node) error {
	mapping, err := source.MapTable(snapshot, observation.TableAnchor, observation.TableBodyRowCount, observation.TableLastBodyRowAnchor)
	if err != nil {
		if errors.Is(err, source.ErrUnsupportedTableShape) {
			return nil
		}
		return fmt.Errorf("map table source: %w", err)
	}
	if observation.TableAnchor != mapping.Range.Start || observation.TableColumnCount <= 0 || len(observation.TableAlignments) != observation.TableColumnCount || len(mapping.Header.Cells) != observation.TableColumnCount || len(mapping.Delimiter.Cells) != observation.TableColumnCount || len(mapping.DelimiterAlignments) != observation.TableColumnCount {
		return nil
	}
	for index, alignment := range observation.TableAlignments {
		if !tableDelimiterAlignmentMatches(alignment, mapping.DelimiterAlignments[index]) {
			return nil
		}
	}
	node.Range = mapping.Range
	node.ContentRange = mapping.Range
	node.TableAnchor = observation.TableAnchor
	node.TableColumnCount = observation.TableColumnCount
	node.TableAlignments = append([]TableAlignment(nil), observation.TableAlignments...)
	node.TableBodyRowCount = observation.TableBodyRowCount
	node.TableLastBodyRowAnchor = observation.TableLastBodyRowAnchor
	node.TableSource = mapping
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
