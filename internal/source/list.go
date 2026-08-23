package source

import "fmt"

// ListItemMapping binds one single-line list item to its exact source marker and content.
type ListItemMapping struct {
	Range        Range
	LineRange    Range
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

	lineRangeEnd := lineEnd
	if next, ok := nextPhysicalLineStart(input, lineEnd); ok {
		lineRangeEnd = next
	}
	return ListItemMapping{
		Range:        Range{Start: lineStart + markerStart, End: lineEnd},
		LineRange:    Range{Start: lineStart, End: lineRangeEnd},
		ContentRange: content,
		Ordered:      ordered,
		Marker:       marker,
	}, nil
}

func listItemMarkerStart(prefix []byte, ordered bool, marker byte) (int, bool) {
	end, ok := listItemMarkerEnd(prefix)
	if !ok {
		return 0, false
	}
	if ordered {
		return orderedListItemMarkerStart(prefix, end, marker)
	}
	return unorderedListItemMarkerStart(prefix, end, marker)
}

func listItemMarkerEnd(prefix []byte) (int, bool) {
	end := len(prefix)
	for end > 0 && isHorizontalSpace(prefix[end-1]) {
		end--
	}
	return end, end != len(prefix) && end != 0
}

func unorderedListItemMarkerStart(prefix []byte, end int, marker byte) (int, bool) {
	if marker != '-' && marker != '*' && marker != '+' || prefix[end-1] != marker {
		return 0, false
	}
	return end - 1, true
}

func orderedListItemMarkerStart(prefix []byte, end int, marker byte) (int, bool) {
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
