package source

import (
	"errors"
	"fmt"
)

var (
	ErrUnsupportedHeadingShape             = errors.New("unsupported heading source shape")
	ErrUnsupportedTaskMarker               = errors.New("unsupported task-list marker source shape")
	ErrUnsupportedListItemShape            = errors.New("unsupported list-item source shape")
	ErrUnsupportedTableCellShape           = errors.New("unsupported table-cell source shape")
	ErrUnsupportedFencedCodeShape          = errors.New("unsupported fenced-code source shape")
	ErrUnsupportedStrikethroughShape       = errors.New("unsupported strikethrough source shape")
	ErrUnsupportedInlineLinkShape          = errors.New("unsupported inline-link source shape")
	ErrUnsupportedImageShape               = errors.New("unsupported image source shape")
	ErrUnsupportedReferenceDefinitionShape = errors.New("unsupported reference-definition source shape")
	ErrUnsupportedAutoLinkShape            = errors.New("unsupported autolink source shape")
	ErrUnsupportedCodeSpanShape            = errors.New("unsupported code-span source shape")
	ErrUnsupportedEmphasisShape            = errors.New("unsupported emphasis source shape")
)

// HeadingStyle identifies the source syntax used by a mapped heading.
type HeadingStyle uint8

const (
	HeadingStyleUnknown HeadingStyle = iota
	HeadingStyleATX
	HeadingStyleSetext
)

// HeadingMapping binds semantic heading content to its exact top-level source region.
type HeadingMapping struct {
	Range        Range
	ContentRange Range
	Style        HeadingStyle
	Level        int
}

// MapTopLevelHeading maps a Goldmark heading content range to top-level GFM source syntax.
// Container-prefixed headings are intentionally rejected until container-aware mapping exists.
func MapTopLevelHeading(input []byte, content Range, level int) (HeadingMapping, error) {
	if !content.Valid(len(input)) || level < 1 || level > 6 {
		return HeadingMapping{}, fmt.Errorf("%w: invalid content range or level", ErrUnsupportedHeadingShape)
	}

	lineStart := physicalLineStart(input, content.Start)
	lineEnd := physicalLineEnd(input, content.End)
	if isTopLevelATXPrefix(input[lineStart:content.Start], level) {
		return HeadingMapping{
			Range:        Range{Start: lineStart, End: lineEnd},
			ContentRange: content,
			Style:        HeadingStyleATX,
			Level:        level,
		}, nil
	}

	if level > 2 {
		return HeadingMapping{}, fmt.Errorf("%w: heading level %d is neither mapped ATX nor Setext", ErrUnsupportedHeadingShape, level)
	}

	underlineStart, ok := nextPhysicalLineStart(input, lineEnd)
	if !ok {
		return HeadingMapping{}, fmt.Errorf("%w: missing Setext underline", ErrUnsupportedHeadingShape)
	}
	underlineEnd := physicalLineEnd(input, underlineStart)
	if !isSetextUnderline(input[underlineStart:underlineEnd], level) {
		return HeadingMapping{}, fmt.Errorf("%w: invalid Setext underline", ErrUnsupportedHeadingShape)
	}

	return HeadingMapping{
		Range:        Range{Start: lineStart, End: underlineEnd},
		ContentRange: content,
		Style:        HeadingStyleSetext,
		Level:        level,
	}, nil
}

// TaskMapping binds a semantic GFM task checkbox to its exact marker and state byte.
type TaskMapping struct {
	Range        Range
	ContentRange Range
	Checked      bool
}

// MapTaskMarker maps a Goldmark task-line anchor to the exact GFM checkbox marker.
func MapTaskMarker(input []byte, anchor int) (TaskMapping, error) {
	if anchor < 0 || anchor+3 > len(input) {
		return TaskMapping{}, fmt.Errorf("%w: anchor %d is outside source length %d", ErrUnsupportedTaskMarker, anchor, len(input))
	}
	if input[anchor] != '[' || input[anchor+2] != ']' {
		return TaskMapping{}, fmt.Errorf("%w: expected bracketed marker at byte %d", ErrUnsupportedTaskMarker, anchor)
	}

	state := input[anchor+1]
	checked := false
	switch state {
	case 'x', 'X':
		checked = true
	case ' ', '\t', '\v', '\f':
		// GFM allows a whitespace character for an unchecked task marker.
	default:
		return TaskMapping{}, fmt.Errorf("%w: invalid task state byte 0x%02x at byte %d", ErrUnsupportedTaskMarker, state, anchor+1)
	}

	return TaskMapping{
		Range:        Range{Start: anchor, End: anchor + 3},
		ContentRange: Range{Start: anchor + 1, End: anchor + 2},
		Checked:      checked,
	}, nil
}

// ListItemMapping binds one single-line list item to its exact source marker and content.
type ListItemMapping struct {
	Range        Range
	ContentRange Range
	Ordered      bool
	Marker       byte
}

// MapSingleLineListItem maps a semantic single-line list item content range to its source marker.
func MapSingleLineListItem(input []byte, content Range, ordered bool, marker byte) (ListItemMapping, error) {
	if !content.Valid(len(input)) || content.Start == content.End {
		return ListItemMapping{}, fmt.Errorf("%w: invalid content range", ErrUnsupportedListItemShape)
	}
	if containsLineBreak(input[content.Start:content.End]) {
		return ListItemMapping{}, fmt.Errorf("%w: content crosses a physical line", ErrUnsupportedListItemShape)
	}
	lineStart := physicalLineStart(input, content.Start)
	lineEnd := physicalLineEnd(input, content.End)

	prefix := input[lineStart:content.Start]
	markerStart, ok := listItemMarkerStart(prefix, ordered, marker)
	if !ok {
		return ListItemMapping{}, fmt.Errorf("%w: marker %q does not match semantic list item", ErrUnsupportedListItemShape, marker)
	}

	return ListItemMapping{
		Range:        Range{Start: lineStart + markerStart, End: lineEnd},
		ContentRange: content,
		Ordered:      ordered,
		Marker:       marker,
	}, nil
}

func listItemMarkerStart(prefix []byte, ordered bool, marker byte) (int, bool) {
	end := len(prefix)
	for end > 0 && (prefix[end-1] == ' ' || prefix[end-1] == '\t') {
		end--
	}
	if end == len(prefix) || end == 0 {
		return 0, false
	}

	if !ordered {
		if marker != '-' && marker != '*' && marker != '+' || prefix[end-1] != marker {
			return 0, false
		}
		return end - 1, true
	}
	if marker != '.' && marker != ')' || prefix[end-1] != marker {
		return 0, false
	}

	digitsEnd := end - 1
	digitsStart := digitsEnd
	for digitsStart > 0 && prefix[digitsStart-1] >= '0' && prefix[digitsStart-1] <= '9' {
		digitsStart--
	}
	if digitsStart == digitsEnd || digitsEnd-digitsStart > 9 {
		return 0, false
	}
	return digitsStart, true
}

// TableCellMapping binds one GFM table cell to its raw cell span.
type TableCellMapping struct {
	Range        Range
	ContentRange Range
	Column       int
}

// TableRowMapping binds one physical GFM table row to all of its lossless cell spans.
type TableRowMapping struct {
	Range Range
	Cells []TableCellMapping
}

// MapTableRow maps all cells in one physical GFM table row with a single row scan.
func MapTableRow(input []byte, anchor int) (TableRowMapping, error) {
	if anchor < 0 || anchor >= len(input) {
		return TableRowMapping{}, fmt.Errorf("%w: row anchor %d is outside source length %d", ErrUnsupportedTableCellShape, anchor, len(input))
	}
	lineStart := physicalLineStart(input, anchor)
	lineEnd := physicalLineEnd(input, anchor)
	if lineStart == lineEnd {
		return TableRowMapping{}, fmt.Errorf("%w: empty physical row", ErrUnsupportedTableCellShape)
	}

	line := input[lineStart:lineEnd]
	spans := tableCellSpans(line)
	if len(spans) == 0 {
		return TableRowMapping{}, fmt.Errorf("%w: physical row contains no table-cell delimiters", ErrUnsupportedTableCellShape)
	}
	cells := make([]TableCellMapping, len(spans))
	for column, raw := range spans {
		trimmed := trimHorizontalSpaceRange(line, raw)
		cells[column] = TableCellMapping{
			Range:        Range{Start: lineStart + raw.Start, End: lineStart + raw.End},
			ContentRange: Range{Start: lineStart + trimmed.Start, End: lineStart + trimmed.End},
			Column:       column,
		}
	}
	return TableRowMapping{
		Range: Range{Start: lineStart, End: lineEnd},
		Cells: cells,
	}, nil
}

// MapTableCell verifies a non-empty semantic table-cell range against one physical GFM table row.
func MapTableCell(input []byte, content Range, column int) (TableCellMapping, error) {
	if !content.Valid(len(input)) || content.Start == content.End || column < 0 {
		return TableCellMapping{}, fmt.Errorf("%w: invalid content range or column", ErrUnsupportedTableCellShape)
	}
	if containsLineBreak(input[content.Start:content.End]) {
		return TableCellMapping{}, fmt.Errorf("%w: content crosses a physical line", ErrUnsupportedTableCellShape)
	}

	row, err := MapTableRow(input, content.Start)
	if err != nil {
		return TableCellMapping{}, err
	}
	if column >= len(row.Cells) {
		return TableCellMapping{}, fmt.Errorf("%w: column %d is outside %d mapped cells", ErrUnsupportedTableCellShape, column, len(row.Cells))
	}
	mapping := row.Cells[column]
	if mapping.ContentRange != content {
		return TableCellMapping{}, fmt.Errorf("%w: semantic content [%d,%d) does not match mapped content [%d,%d)", ErrUnsupportedTableCellShape, content.Start, content.End, mapping.ContentRange.Start, mapping.ContentRange.End)
	}
	return mapping, nil
}

func tableCellSpans(line []byte) []Range {
	var delimiters []int
	for i, b := range line {
		if b != '|' || i > 0 && line[i-1] == '\\' {
			continue
		}
		delimiters = append(delimiters, i)
	}
	if len(delimiters) == 0 {
		return nil
	}

	start := 0
	firstDelimiter := 0
	if allHorizontalSpace(line[:delimiters[0]]) {
		start = delimiters[0] + 1
		firstDelimiter = 1
	}
	trailingDelimiter := allHorizontalSpace(line[delimiters[len(delimiters)-1]+1:])

	spans := make([]Range, 0, len(delimiters)+1)
	for i := firstDelimiter; i < len(delimiters); i++ {
		delimiter := delimiters[i]
		spans = append(spans, Range{Start: start, End: delimiter})
		start = delimiter + 1
		if trailingDelimiter && i == len(delimiters)-1 {
			return spans
		}
	}
	if !trailingDelimiter {
		spans = append(spans, Range{Start: start, End: len(line)})
	}
	return spans
}

// FencedCodeMapping binds one single-line fenced code block to its exact source fences and content.
type FencedCodeMapping struct {
	Range              Range
	ContentRange       Range
	InfoRange          Range
	FenceChar          byte
	FenceLength        int
	ClosingFenceLength int
	OpeningIndent      int
	ClosingIndent      int
}

// MapSingleLineFencedCode maps one non-empty semantic fenced-code line to top-level source syntax.
func MapSingleLineFencedCode(input []byte, content Range) (FencedCodeMapping, error) {
	if !content.Valid(len(input)) || content.Start == content.End {
		return FencedCodeMapping{}, fmt.Errorf("%w: invalid or empty content range", ErrUnsupportedFencedCodeShape)
	}
	if containsLineBreak(input[content.Start:content.End]) {
		return FencedCodeMapping{}, fmt.Errorf("%w: content crosses a physical line", ErrUnsupportedFencedCodeShape)
	}

	contentLineStart := physicalLineStart(input, content.Start)
	contentLineEnd := physicalLineEnd(input, content.End)
	if content.End != contentLineEnd {
		return FencedCodeMapping{}, fmt.Errorf("%w: semantic content does not reach the physical line end", ErrUnsupportedFencedCodeShape)
	}

	openingStart, ok := previousPhysicalLineStart(input, contentLineStart)
	if !ok {
		return FencedCodeMapping{}, fmt.Errorf("%w: missing opening fence line", ErrUnsupportedFencedCodeShape)
	}
	openingEnd := physicalLineEnd(input, openingStart)
	nextLine, ok := nextPhysicalLineStart(input, openingEnd)
	if !ok || nextLine != contentLineStart {
		return FencedCodeMapping{}, fmt.Errorf("%w: content is not immediately after the opening fence", ErrUnsupportedFencedCodeShape)
	}
	opening, ok := parseTopLevelFenceOpening(input, openingStart, openingEnd)
	if !ok {
		return FencedCodeMapping{}, fmt.Errorf("%w: opening fence is not a supported top-level fence", ErrUnsupportedFencedCodeShape)
	}

	closingStart, ok := nextPhysicalLineStart(input, contentLineEnd)
	if !ok {
		return FencedCodeMapping{}, fmt.Errorf("%w: missing closing fence line", ErrUnsupportedFencedCodeShape)
	}
	closingEnd := physicalLineEnd(input, closingStart)
	closingLength, closingIndent, ok := parseTopLevelFenceClosing(input, closingStart, closingEnd, opening.char, opening.length)
	if !ok {
		return FencedCodeMapping{}, fmt.Errorf("%w: closing fence does not match opening fence", ErrUnsupportedFencedCodeShape)
	}

	return FencedCodeMapping{
		Range:              Range{Start: openingStart, End: closingEnd},
		ContentRange:       content,
		InfoRange:          opening.infoRange,
		FenceChar:          opening.char,
		FenceLength:        opening.length,
		ClosingFenceLength: closingLength,
		OpeningIndent:      opening.indent,
		ClosingIndent:      closingIndent,
	}, nil
}

type fenceOpening struct {
	char      byte
	length    int
	indent    int
	infoRange Range
}

func parseTopLevelFenceOpening(input []byte, start, end int) (fenceOpening, bool) {
	if start < 0 || end < start || end > len(input) {
		return fenceOpening{}, false
	}
	line := input[start:end]
	indent := 0
	for indent < len(line) && line[indent] == ' ' {
		indent++
	}
	if indent > 3 || indent == len(line) || line[indent] != '`' && line[indent] != '~' {
		return fenceOpening{}, false
	}

	char := line[indent]
	i := indent
	for i < len(line) && line[i] == char {
		i++
	}
	length := i - indent
	if length < 3 {
		return fenceOpening{}, false
	}

	infoStart := i
	for infoStart < len(line) && isHorizontalSpace(line[infoStart]) {
		infoStart++
	}
	infoEnd := len(line)
	for infoEnd > infoStart && isHorizontalSpace(line[infoEnd-1]) {
		infoEnd--
	}
	if char == '`' {
		for _, b := range line[infoStart:infoEnd] {
			if b == '`' {
				return fenceOpening{}, false
			}
		}
	}

	return fenceOpening{
		char:      char,
		length:    length,
		indent:    indent,
		infoRange: Range{Start: start + infoStart, End: start + infoEnd},
	}, true
}

func parseTopLevelFenceClosing(input []byte, start, end int, char byte, minimumLength int) (int, int, bool) {
	if start < 0 || end < start || end > len(input) {
		return 0, 0, false
	}
	line := input[start:end]
	indent := 0
	for indent < len(line) && line[indent] == ' ' {
		indent++
	}
	if indent > 3 || indent == len(line) || line[indent] != char {
		return 0, 0, false
	}

	i := indent
	for i < len(line) && line[i] == char {
		i++
	}
	length := i - indent
	if length < minimumLength {
		return 0, 0, false
	}
	for ; i < len(line); i++ {
		if !isHorizontalSpace(line[i]) {
			return 0, 0, false
		}
	}
	return length, indent, true
}

func isTopLevelATXPrefix(prefix []byte, level int) bool {
	i := 0
	for i < len(prefix) && i < 3 && prefix[i] == ' ' {
		i++
	}
	markerStart := i
	for i < len(prefix) && prefix[i] == '#' {
		i++
	}
	if i-markerStart != level {
		return false
	}
	if i == len(prefix) {
		return false
	}
	for ; i < len(prefix); i++ {
		if prefix[i] != ' ' && prefix[i] != '\t' {
			return false
		}
	}
	return true
}

func isSetextUnderline(line []byte, level int) bool {
	i := 0
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	if i == len(line) {
		return false
	}
	marker := byte('=')
	if level == 2 {
		marker = '-'
	}
	markerCount := 0
	for i < len(line) && line[i] == marker {
		i++
		markerCount++
	}
	if markerCount == 0 {
		return false
	}
	for ; i < len(line); i++ {
		if line[i] != ' ' && line[i] != '\t' {
			return false
		}
	}
	return true
}
