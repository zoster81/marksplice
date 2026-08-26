package native

import "github.com/zoster81/marksplice/internal/parser"

type tableRowScan struct {
	anchor  int
	end     int
	cells   []parser.Range
	hasPipe bool
}

func parseTable(source []byte, lines []physicalLine, index int) ([]parser.Node, []parser.Range, int, bool) {
	header, alignments, ok := parseTableOpening(source, lines, index)
	if !ok {
		return nil, nil, index, false
	}
	headerLine := lines[index]

	bodyNodes := make([]parser.Node, 0)
	semantic := tableRowSemanticRanges(source, header.cells, len(header.cells))
	bodyRowCount := 0
	lastBodyRowAnchor := 0
	next := index + 2
	for next < len(lines) && tableBodyContinues(source, headerLine, lines[next]) {
		row, ok := scanTableRow(source, lines[next], true)
		if !ok {
			break
		}
		bodyRowCount++
		lastBodyRowAnchor = row.anchor
		bodyNodes = append(bodyNodes, tableRowNode(row, header.anchor, len(header.cells), alignments))
		bodyNodes = append(bodyNodes, tableCellNodes(row.cells, false, row.anchor, header.anchor, len(header.cells))...)
		semantic = append(semantic, tableRowSemanticRanges(source, row.cells, len(header.cells))...)
		next++
	}

	nodes := make([]parser.Node, 0, 1+len(header.cells)+len(bodyNodes))
	nodes = append(nodes, parser.Node{
		Kind:                   parser.KindTable,
		Range:                  parser.Range{Start: header.anchor, End: header.end},
		TableAnchor:            header.anchor,
		TableColumnCount:       len(header.cells),
		TableAlignments:        append([]parser.TableAlignment(nil), alignments...),
		TableBodyRowCount:      bodyRowCount,
		TableLastBodyRowAnchor: lastBodyRowAnchor,
	})
	nodes = append(nodes, tableCellNodes(header.cells, true, header.anchor, header.anchor, len(header.cells))...)
	nodes = append(nodes, bodyNodes...)
	return nodes, semantic, next, true
}

func parseTableOpening(source []byte, lines []physicalLine, index int) (tableRowScan, []parser.TableAlignment, bool) {
	if index < 0 || index+1 >= len(lines) {
		return tableRowScan{}, nil, false
	}
	headerLine := lines[index]
	delimiterLine := lines[index+1]
	if _, ok := ordinaryIndent(source, headerLine); !ok {
		return tableRowScan{}, nil, false
	}
	if _, ok := ordinaryIndent(source, delimiterLine); !ok || !sameBlockContainer(headerLine, delimiterLine) {
		return tableRowScan{}, nil, false
	}
	if interruptsParagraph(source, delimiterLine) || startsBlockquote(source, delimiterLine) {
		return tableRowScan{}, nil, false
	}
	header, ok := scanTableRow(source, headerLine, true)
	if !ok {
		return tableRowScan{}, nil, false
	}
	delimiter, ok := scanTableRow(source, delimiterLine, true)
	if !ok || !header.hasPipe && !delimiter.hasPipe || len(header.cells) != len(delimiter.cells) {
		return tableRowScan{}, nil, false
	}
	alignments, ok := tableDelimiterAlignments(source, delimiter.cells)
	if !ok {
		return tableRowScan{}, nil, false
	}
	return header, alignments, true
}

func tableRowSemanticRanges(source []byte, cells []parser.Range, columnCount int) []parser.Range {
	limit := min(len(cells), columnCount)
	ranges := make([]parser.Range, 0, limit)
	for index := 0; index < limit; index++ {
		cell := cells[index]
		range_ := parser.Range{Start: cell.Start, End: blockLineSemanticEnd(source, cell.End)}
		if range_.Start != range_.End {
			ranges = append(ranges, range_)
		}
	}
	return ranges
}

func tableBodyContinues(source []byte, header, line physicalLine) bool {
	if blankLine(source, line) {
		return false
	}
	if !sameBlockContainer(header, line) && !tableLazyBodyContinuation(source, header, line) {
		return false
	}
	if startsBlockquote(source, line) {
		return false
	}
	return !interruptsParagraph(source, line)
}

func tableLazyBodyContinuation(source []byte, header, line physicalLine) bool {
	return header.start != header.physicalStart && line.start == line.physicalStart && lazyParagraphContinuation(source, line)
}

func tableRowNode(row tableRowScan, tableAnchor, columnCount int, alignments []parser.TableAlignment) parser.Node {
	return parser.Node{
		Kind:             parser.KindTableRow,
		Range:            parser.Range{Start: row.anchor, End: row.end},
		TableRowAnchor:   row.anchor,
		TableAnchor:      tableAnchor,
		TableColumnCount: columnCount,
		TableAlignments:  append([]parser.TableAlignment(nil), alignments...),
	}
}

func tableCellNodes(cells []parser.Range, header bool, rowAnchor, tableAnchor, columnCount int) []parser.Node {
	limit := min(len(cells), columnCount)
	nodes := make([]parser.Node, 0, limit)
	for column := 0; column < limit; column++ {
		range_ := cells[column]
		if range_.Start == range_.End {
			continue
		}
		nodes = append(nodes, parser.Node{
			Kind:           parser.KindTableCell,
			Range:          range_,
			TableHeader:    header,
			TableColumn:    column,
			TableRowAnchor: rowAnchor,
			TableAnchor:    tableAnchor,
		})
	}
	return nodes
}

func scanTableRow(source []byte, line physicalLine, allowNoPipe bool) (tableRowScan, bool) {
	positions := tablePipePositions(source, line)
	if len(positions) == 0 {
		if !allowNoPipe {
			return tableRowScan{}, false
		}
		cell := trimTableCellRange(source, parser.Range{Start: line.start, End: line.end})
		return tableRowScan{anchor: line.start, end: line.end, cells: []parser.Range{cell}}, true
	}

	start := line.start
	firstDelimiter := 0
	if tableHorizontalSpace(source, line.start, positions[0]) {
		start = positions[0] + 1
		firstDelimiter = 1
	}
	trailingDelimiter := tableHorizontalSpace(source, positions[len(positions)-1]+1, line.end)
	cells := make([]parser.Range, 0, len(positions)+1)
	for i := firstDelimiter; i < len(positions); i++ {
		delimiter := positions[i]
		cells = append(cells, trimTableCellRange(source, parser.Range{Start: start, End: delimiter}))
		start = delimiter + 1
		if trailingDelimiter && i == len(positions)-1 {
			break
		}
	}
	if !trailingDelimiter {
		cells = append(cells, trimTableCellRange(source, parser.Range{Start: start, End: line.end}))
	}
	if len(cells) == 0 {
		return tableRowScan{}, false
	}
	return tableRowScan{anchor: line.start, end: line.end, cells: cells, hasPipe: true}, true
}

func tablePipePositions(source []byte, line physicalLine) []int {
	positions := make([]int, 0)
	for position := line.start; position < line.end; position++ {
		if source[position] != '|' || tablePipeEscaped(source, line.start, position) {
			continue
		}
		positions = append(positions, position)
	}
	return positions
}

func tablePipeEscaped(source []byte, start, position int) bool {
	backslashes := 0
	for position > start && source[position-1] == '\\' {
		backslashes++
		position--
	}
	return backslashes%2 != 0
}

func trimTableCellRange(source []byte, range_ parser.Range) parser.Range {
	for range_.Start < range_.End && (source[range_.Start] == ' ' || source[range_.Start] == '\t') {
		range_.Start++
	}
	for range_.End > range_.Start && (source[range_.End-1] == ' ' || source[range_.End-1] == '\t') {
		range_.End--
	}
	return range_
}

func tableHorizontalSpace(source []byte, start, end int) bool {
	for position := start; position < end; position++ {
		if source[position] != ' ' && source[position] != '\t' {
			return false
		}
	}
	return true
}

func tableDelimiterAlignments(source []byte, cells []parser.Range) ([]parser.TableAlignment, bool) {
	if len(cells) == 0 {
		return nil, false
	}
	alignments := make([]parser.TableAlignment, len(cells))
	for index, cell := range cells {
		alignment, ok := tableDelimiterAlignment(source, cell)
		if !ok {
			return nil, false
		}
		alignments[index] = alignment
	}
	return alignments, true
}

func tableDelimiterAlignment(source []byte, cell parser.Range) (parser.TableAlignment, bool) {
	if cell.Start >= cell.End {
		return parser.TableAlignmentDefault, false
	}
	start, end := cell.Start, cell.End
	left := source[start] == ':'
	if left {
		start++
	}
	right := end > start && source[end-1] == ':'
	if right {
		end--
	}
	if start >= end {
		return parser.TableAlignmentDefault, false
	}
	for position := start; position < end; position++ {
		if source[position] != '-' {
			return parser.TableAlignmentDefault, false
		}
	}
	switch {
	case left && right:
		return parser.TableAlignmentCenter, true
	case left:
		return parser.TableAlignmentLeft, true
	case right:
		return parser.TableAlignmentRight, true
	default:
		return parser.TableAlignmentDefault, true
	}
}
