package source

import "fmt"

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
