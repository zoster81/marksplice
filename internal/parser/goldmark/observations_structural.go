package goldmark

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	extensionast "github.com/yuin/goldmark/extension/ast"

	"github.com/zoster81/marksplice/internal/parser"
)

func observeListItem(source []byte, item *ast.ListItem) (parser.Node, bool, error) {
	list, ok := item.Parent().(*ast.List)
	if !ok {
		return parser.Node{}, false, nil
	}
	range_, directChildCount, ok := simpleListItemContentRange(source, item)
	if !ok {
		return parser.Node{}, false, nil
	}
	containerAnchor, ok := listContainerAnchor(source, list)
	if !ok {
		return parser.Node{}, false, nil
	}
	parentAnchor, hasParent := immediateListItemParentAnchor(source, list)
	return parser.Node{
		Kind:                 parser.KindListItem,
		Range:                range_,
		Ordered:              list.IsOrdered(),
		Marker:               list.Marker,
		HasListParent:        hasParent,
		ListParentAnchor:     parentAnchor,
		ListContainerAnchor:  containerAnchor,
		HasListChildren:      directChildCount != 0,
		ListDirectChildCount: directChildCount,
	}, true, nil
}

func listContainerAnchor(source []byte, list *ast.List) (int, bool) {
	firstItem, ok := list.FirstChild().(*ast.ListItem)
	if !ok {
		return 0, false
	}
	return listItemPhysicalLineStart(source, firstItem)
}

func immediateListItemParentAnchor(source []byte, list *ast.List) (int, bool) {
	parentItem, ok := list.Parent().(*ast.ListItem)
	if !ok {
		return 0, false
	}
	return listItemPhysicalLineStart(source, parentItem)
}

func listItemPhysicalLineStart(source []byte, item *ast.ListItem) (int, bool) {
	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		lines := child.Lines()
		if lines == nil || lines.Len() == 0 {
			continue
		}
		start := lines.At(0).Start
		if start < 0 || start > len(source) {
			return 0, false
		}
		for start > 0 && source[start-1] != '\n' && source[start-1] != '\r' {
			start--
		}
		return start, true
	}
	return 0, false
}

func observeTableRow(source []byte, row *extensionast.TableRow) (parser.Node, bool, error) {
	table, ok := row.Parent().(*extensionast.Table)
	if !ok || row.Pos() < 0 || table.Pos() < 0 || row.ChildCount() == 0 {
		return parser.Node{}, false, nil
	}
	alignments, ok := observeTableAlignments(row.Alignments, row.ChildCount())
	if !ok {
		return parser.Node{}, false, nil
	}
	start := row.Pos()
	end := paragraphContentEnd(source, start)
	range_ := parser.Range{Start: start, End: end}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Node{}, false, nil
	}
	return parser.Node{
		Kind:             parser.KindTableRow,
		Range:            range_,
		TableRowAnchor:   start,
		TableAnchor:      table.Pos(),
		TableColumnCount: row.ChildCount(),
		TableAlignments:  alignments,
	}, true, nil
}

func observeTableAlignments(values []extensionast.Alignment, columnCount int) ([]parser.TableAlignment, bool) {
	if columnCount <= 0 || len(values) != columnCount {
		return nil, false
	}
	result := make([]parser.TableAlignment, len(values))
	for index, value := range values {
		switch value {
		case extensionast.AlignNone:
			result[index] = parser.TableAlignmentDefault
		case extensionast.AlignLeft:
			result[index] = parser.TableAlignmentLeft
		case extensionast.AlignRight:
			result[index] = parser.TableAlignmentRight
		case extensionast.AlignCenter:
			result[index] = parser.TableAlignmentCenter
		default:
			return nil, false
		}
	}
	return result, true
}

func observeTableCell(source []byte, cell *extensionast.TableCell, column int) (parser.Node, bool) {
	parent := cell.Parent()
	if parent == nil || parent.Kind() != extensionast.KindTableHeader && parent.Kind() != extensionast.KindTableRow || parent.Pos() < 0 {
		return parser.Node{}, false
	}
	table, ok := parent.Parent().(*extensionast.Table)
	if !ok || table.Pos() < 0 {
		return parser.Node{}, false
	}
	lines := cell.Lines()
	if lines.Len() != 1 {
		return parser.Node{}, false
	}
	segment := lines.At(0)
	range_ := parser.Range{Start: segment.Start, End: segment.Stop}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Node{}, false
	}
	return parser.Node{
		Kind:           parser.KindTableCell,
		Range:          range_,
		TableHeader:    parent.Kind() == extensionast.KindTableHeader,
		TableColumn:    column,
		TableRowAnchor: parent.Pos(),
		TableAnchor:    table.Pos(),
	}, true
}

func observeTask(source []byte, checkbox *extensionast.TaskCheckBox) (parser.Node, bool, error) {
	parent, ok := checkbox.Parent().(*ast.TextBlock)
	if !ok || parent.Parent() == nil || parent.Parent().Kind() != ast.KindListItem {
		return parser.Node{}, false, nil
	}
	lines := parent.Lines()
	if lines.Len() == 0 {
		return parser.Node{}, false, nil
	}
	first := lines.At(0)
	range_ := parser.Range{Start: first.Start, End: first.Stop}
	if !range_.Valid(len(source)) {
		return parser.Node{}, false, fmt.Errorf("goldmark task anchor range [%d,%d) is outside source length %d", range_.Start, range_.End, len(source))
	}
	return parser.Node{Kind: parser.KindTask, Range: range_, Checked: checkbox.IsChecked}, true, nil
}
