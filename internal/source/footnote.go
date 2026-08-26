package source

import (
	"bytes"
	"fmt"
	"sort"
)

// FootnoteDefinitionMapping binds one top-level footnote definition to its exact
// source-owned container, label, and parser-proven semantic body segments.
type FootnoteDefinitionMapping struct {
	Range      Range
	LabelRange Range
	BodyRange  Range
	BodyRanges []Range
}

// MapTopLevelFootnoteDefinition independently proves complete physical ownership
// for one parser-observed top-level footnote definition. Multiline definitions
// remain readable through BodyRanges; BodyRange is exposed only for one simple
// body segment on the opening physical line.
func MapTopLevelFootnoteDefinition(input []byte, anchor int, label string, semanticRanges []Range) (FootnoteDefinitionMapping, error) {
	owned, labelRange, bodyStart, err := mapTopLevelFootnoteContainer(input, anchor, label)
	if err != nil {
		return FootnoteDefinitionMapping{}, err
	}
	ranges, err := normalizedFootnoteBodyRanges(input, semanticRanges, owned)
	if err != nil {
		return FootnoteDefinitionMapping{}, err
	}
	bodyRange := Range{}
	if len(ranges) == 1 && physicalLineStart(input, ranges[0].Start) == owned.Start && ranges[0].Start >= bodyStart {
		bodyRange = ranges[0]
	}
	return FootnoteDefinitionMapping{
		Range:      owned,
		LabelRange: labelRange,
		BodyRange:  bodyRange,
		BodyRanges: ranges,
	}, nil
}

func mapTopLevelFootnoteContainer(input []byte, anchor int, label string) (Range, Range, int, error) {
	if anchor < 0 || anchor >= len(input) || label == "" {
		return Range{}, Range{}, 0, fmt.Errorf("%w: invalid definition anchor or label", ErrUnsupportedFootnoteShape)
	}
	lineStart := physicalLineStart(input, anchor)
	if !validTopLevelFootnotePrefix(input[lineStart:anchor]) {
		return Range{}, Range{}, 0, fmt.Errorf("%w: unsupported definition indentation", ErrUnsupportedFootnoteShape)
	}
	token := []byte("[^" + label + "]:")
	if anchor+len(token) > len(input) || !bytes.Equal(input[anchor:anchor+len(token)], token) {
		return Range{}, Range{}, 0, fmt.Errorf("%w: source label does not match parser label", ErrUnsupportedFootnoteShape)
	}

	openingEnd := physicalLineEnd(input, anchor)
	ownedEnd := physicalLineRangeEnd(input, openingEnd)
	for next, ok := nextPhysicalLineStart(input, openingEnd); ok && next < len(input); {
		lineEnd := physicalLineEnd(input, next)
		line := input[next:lineEnd]
		if len(bytes.Trim(line, " \t")) != 0 && footnoteContinuationIndent(line) < 4 {
			break
		}
		ownedEnd = physicalLineRangeEnd(input, lineEnd)
		next, ok = nextPhysicalLineStart(input, lineEnd)
		if !ok {
			break
		}
	}
	return Range{Start: lineStart, End: ownedEnd},
		Range{Start: anchor + 2, End: anchor + 2 + len(label)}, anchor + len(token), nil
}

func validTopLevelFootnotePrefix(prefix []byte) bool {
	if len(prefix) > 3 {
		return false
	}
	for _, value := range prefix {
		if value != ' ' {
			return false
		}
	}
	return true
}

func physicalLineRangeEnd(input []byte, lineEnd int) int {
	if next, ok := nextPhysicalLineStart(input, lineEnd); ok {
		return next
	}
	return lineEnd
}

func footnoteContinuationIndent(line []byte) int {
	column := 0
	for _, value := range line {
		switch value {
		case ' ':
			column++
		case '\t':
			column += 4 - column%4
		default:
			return column
		}
		if column >= 4 {
			return column
		}
	}
	return column
}

func normalizedFootnoteBodyRanges(input []byte, ranges []Range, owned Range) ([]Range, error) {
	result := append([]Range(nil), ranges...)
	for _, range_ := range result {
		if !range_.Valid(len(input)) || range_.Start == range_.End || range_.Start < owned.Start || range_.End > owned.End || containsLineBreak(input[range_.Start:range_.End]) {
			return nil, fmt.Errorf("%w: invalid semantic body range", ErrUnsupportedFootnoteShape)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Start != result[j].Start {
			return result[i].Start < result[j].Start
		}
		return result[i].End < result[j].End
	})
	for index := 1; index < len(result); index++ {
		if result[index].Start < result[index-1].End {
			return nil, fmt.Errorf("%w: overlapping semantic body ranges", ErrUnsupportedFootnoteShape)
		}
	}
	return result, nil
}
