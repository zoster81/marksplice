package goldmark

import (
	extensionast "github.com/yuin/goldmark/extension/ast"

	"github.com/zoster81/marksplice/internal/parser"
)

func observeTable(source []byte, table *extensionast.Table) (parser.Node, bool, error) {
	if len(table.Alignments) == 0 {
		return parser.Node{}, false, nil
	}
	header, ok := table.FirstChild().(*extensionast.TableHeader)
	if !ok || header.ChildCount() != len(table.Alignments) {
		return parser.Node{}, false, nil
	}
	tableAnchor, ok := tableSourceAnchor(table)
	if !ok {
		return parser.Node{}, false, nil
	}
	alignments, ok := observeTableAlignments(table.Alignments, header.ChildCount())
	if !ok {
		return parser.Node{}, false, nil
	}

	bodyRowCount := 0
	lastBodyRowAnchor := 0
	previousAnchor := tableAnchor
	for child := header.NextSibling(); child != nil; child = child.NextSibling() {
		row, ok := child.(*extensionast.TableRow)
		if !ok || row.Pos() <= previousAnchor {
			return parser.Node{}, false, nil
		}
		bodyRowCount++
		lastBodyRowAnchor = row.Pos()
		previousAnchor = row.Pos()
	}

	range_ := parser.Range{Start: tableAnchor, End: paragraphContentEnd(source, tableAnchor)}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Node{}, false, nil
	}
	return parser.Node{
		Kind:                   parser.KindTable,
		Range:                  range_,
		TableAnchor:            tableAnchor,
		TableColumnCount:       header.ChildCount(),
		TableAlignments:        alignments,
		TableBodyRowCount:      bodyRowCount,
		TableLastBodyRowAnchor: lastBodyRowAnchor,
	}, true, nil
}

func tableSourceAnchor(table *extensionast.Table) (int, bool) {
	if table == nil {
		return 0, false
	}
	header, ok := table.FirstChild().(*extensionast.TableHeader)
	if !ok || header.Pos() < 0 {
		return 0, false
	}
	return header.Pos(), true
}
