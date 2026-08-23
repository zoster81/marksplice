package source

import "fmt"

// ThematicBreakMapping binds one top-level semantic thematic break to its exact
// source token span and complete physical line.
type ThematicBreakMapping struct {
	Range     Range
	LineRange Range
}

// MapTopLevelThematicBreak independently proves one semantic thematic-break
// observation against its complete physical source line.
func MapTopLevelThematicBreak(input []byte, observed Range) (ThematicBreakMapping, error) {
	if !observed.Valid(len(input)) || observed.Start == observed.End || containsLineBreak(input[observed.Start:observed.End]) {
		return ThematicBreakMapping{}, fmt.Errorf("%w: invalid semantic range", ErrUnsupportedThematicBreakShape)
	}
	lineStart := physicalLineStart(input, observed.Start)
	lineEnd := physicalLineEnd(input, observed.End)
	if observed.End != lineEnd {
		return ThematicBreakMapping{}, fmt.Errorf("%w: semantic range does not reach line end", ErrUnsupportedThematicBreakShape)
	}
	markerStart, ok := thematicBreakMarkerStart(input[lineStart:lineEnd])
	if !ok || observed.Start > lineStart+markerStart {
		return ThematicBreakMapping{}, fmt.Errorf("%w: physical line is not a supported thematic break", ErrUnsupportedThematicBreakShape)
	}
	lineRangeEnd := lineEnd
	if next, ok := nextPhysicalLineStart(input, lineEnd); ok {
		lineRangeEnd = next
	}
	return ThematicBreakMapping{
		Range:     observed,
		LineRange: Range{Start: lineStart, End: lineRangeEnd},
	}, nil
}

func thematicBreakMarkerStart(line []byte) (int, bool) {
	pos := 0
	for pos < len(line) && pos < 3 && line[pos] == ' ' {
		pos++
	}
	if pos >= len(line) || (line[pos] != '*' && line[pos] != '-' && line[pos] != '_') {
		return 0, false
	}
	markerStart := pos
	marker := line[pos]
	count := 0
	for ; pos < len(line); pos++ {
		switch line[pos] {
		case marker:
			count++
		case ' ', '\t':
		default:
			return 0, false
		}
	}
	return markerStart, count >= 3
}
