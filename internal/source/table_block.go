package source

import "fmt"

// TableDelimiterAlignment identifies the alignment encoded by one mapped GFM table delimiter cell.
type TableDelimiterAlignment uint8

const (
	TableDelimiterAlignmentDefault TableDelimiterAlignment = iota
	TableDelimiterAlignmentLeft
	TableDelimiterAlignmentRight
	TableDelimiterAlignmentCenter
)

// TableMapping binds one semantic GFM table to its exact complete physical source span.
type TableMapping struct {
	Range               Range
	Header              TableRowMapping
	Delimiter           TableRowMapping
	DelimiterAlignments []TableDelimiterAlignment
}

// MapTable maps the source-owned header, delimiter, and complete table span from semantic table anchors.
// bodyRowCount may exceed the number of body rows that Marksplice can promote individually.
func MapTable(input []byte, anchor, bodyRowCount, lastBodyRowAnchor int) (TableMapping, error) {
	if bodyRowCount < 0 {
		return TableMapping{}, fmt.Errorf("%w: negative body-row count %d", ErrUnsupportedTableShape, bodyRowCount)
	}
	header, delimiter, alignments, err := mapTableHeaderAndDelimiter(input, anchor)
	if err != nil {
		return TableMapping{}, err
	}
	end, err := completeTableEnd(input, delimiter, bodyRowCount, lastBodyRowAnchor)
	if err != nil {
		return TableMapping{}, err
	}
	return TableMapping{
		Range:               Range{Start: header.LineRange.Start, End: end},
		Header:              header,
		Delimiter:           delimiter,
		DelimiterAlignments: alignments,
	}, nil
}

func mapTableHeaderAndDelimiter(input []byte, anchor int) (TableRowMapping, TableRowMapping, []TableDelimiterAlignment, error) {
	header, err := MapTableRow(input, anchor)
	if err != nil || header.Range.Start != anchor {
		return TableRowMapping{}, TableRowMapping{}, nil, wrapUnsupportedTableShape("map header row", err)
	}
	if header.LineRange.End >= len(input) {
		return TableRowMapping{}, TableRowMapping{}, nil, fmt.Errorf("%w: table header has no delimiter row", ErrUnsupportedTableShape)
	}
	delimiter, err := MapTableRow(input, header.LineRange.End)
	if err != nil || delimiter.Range.Start != header.LineRange.End {
		return TableRowMapping{}, TableRowMapping{}, nil, wrapUnsupportedTableShape("map delimiter row", err)
	}
	alignments := make([]TableDelimiterAlignment, len(delimiter.Cells))
	for index, cell := range delimiter.Cells {
		alignment, ok := tableDelimiterAlignment(input, cell.ContentRange)
		if !ok {
			return TableRowMapping{}, TableRowMapping{}, nil, fmt.Errorf("%w: invalid delimiter cell %d", ErrUnsupportedTableShape, index)
		}
		alignments[index] = alignment
	}
	return header, delimiter, alignments, nil
}

func completeTableEnd(input []byte, delimiter TableRowMapping, bodyRowCount, lastBodyRowAnchor int) (int, error) {
	if bodyRowCount == 0 {
		return delimiter.LineRange.End, nil
	}
	if lastBodyRowAnchor < delimiter.LineRange.End || lastBodyRowAnchor >= len(input) || physicalLineStart(input, lastBodyRowAnchor) != lastBodyRowAnchor {
		return 0, fmt.Errorf("%w: invalid last body-row anchor %d", ErrUnsupportedTableShape, lastBodyRowAnchor)
	}
	lastEnd := physicalLineEnd(input, lastBodyRowAnchor)
	end := lastEnd
	if next, ok := nextPhysicalLineStart(input, lastEnd); ok {
		end = next
	}
	if end <= delimiter.LineRange.End || end > len(input) {
		return 0, fmt.Errorf("%w: invalid complete table end %d", ErrUnsupportedTableShape, end)
	}
	return end, nil
}

func tableDelimiterAlignment(input []byte, content Range) (TableDelimiterAlignment, bool) {
	if !content.Valid(len(input)) || content.Start == content.End {
		return TableDelimiterAlignmentDefault, false
	}
	token := input[content.Start:content.End]
	left := token[0] == ':'
	if left {
		token = token[1:]
	}
	right := len(token) > 0 && token[len(token)-1] == ':'
	if right {
		token = token[:len(token)-1]
	}
	if len(token) == 0 {
		return TableDelimiterAlignmentDefault, false
	}
	for _, b := range token {
		if b != '-' {
			return TableDelimiterAlignmentDefault, false
		}
	}
	switch {
	case left && right:
		return TableDelimiterAlignmentCenter, true
	case left:
		return TableDelimiterAlignmentLeft, true
	case right:
		return TableDelimiterAlignmentRight, true
	default:
		return TableDelimiterAlignmentDefault, true
	}
}

// TableDelimiterAlignmentReplacement returns a replacement for one delimiter-cell token while preserving its exact dash run.
func TableDelimiterAlignmentReplacement(input []byte, content Range, alignment TableDelimiterAlignment) ([]byte, error) {
	current, ok := tableDelimiterAlignment(input, content)
	if !ok {
		return nil, fmt.Errorf("%w: invalid delimiter cell", ErrUnsupportedTableShape)
	}
	if alignment > TableDelimiterAlignmentCenter {
		return nil, fmt.Errorf("%w: invalid delimiter alignment %d", ErrUnsupportedTableShape, alignment)
	}
	if current == alignment {
		return append([]byte(nil), input[content.Start:content.End]...), nil
	}

	token := input[content.Start:content.End]
	dashStart := 0
	if token[0] == ':' {
		dashStart++
	}
	dashEnd := len(token)
	if token[dashEnd-1] == ':' {
		dashEnd--
	}
	if dashStart >= dashEnd {
		return nil, fmt.Errorf("%w: delimiter has no dash run", ErrUnsupportedTableShape)
	}

	result := make([]byte, 0, dashEnd-dashStart+2)
	if alignment == TableDelimiterAlignmentLeft || alignment == TableDelimiterAlignmentCenter {
		result = append(result, ':')
	}
	result = append(result, token[dashStart:dashEnd]...)
	if alignment == TableDelimiterAlignmentRight || alignment == TableDelimiterAlignmentCenter {
		result = append(result, ':')
	}
	return result, nil
}

func wrapUnsupportedTableShape(context string, err error) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrUnsupportedTableShape, context)
	}
	return fmt.Errorf("%w: %s: %v", ErrUnsupportedTableShape, context, err)
}
