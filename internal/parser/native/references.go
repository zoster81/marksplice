package native

import (
	"strings"

	"github.com/zoster81/marksplice/internal/parser"
)

type referenceDefinitionParse struct {
	label       string
	destination string
	title       string
	hasTitle    bool
	firstLine   int
	lastLine    int
	anchor      int
	hasAnchor   bool
	projectable bool
}

func parseReferenceDefinition(source []byte, lines []physicalLine, index int) (parser.Node, []parser.Range, referenceDefinitionParse, int, bool) {
	parsed, ok := scanReferenceDefinition(source, lines, index)
	if !ok {
		return parser.Node{}, nil, referenceDefinitionParse{}, index, false
	}
	semantic := referenceDefinitionSemanticRanges(source, lines, parsed.firstLine, parsed.lastLine)
	next := parsed.lastLine + 1
	if !parsed.projectable || parsed.destination == "" {
		return parser.Node{}, semantic, parsed, next, true
	}
	line := lines[index]
	indent, _ := ordinaryIndent(source, line)
	return parser.Node{
		Kind:        parser.KindReferenceDefinition,
		Range:       parser.Range{Start: line.start + indent, End: line.end},
		Destination: parsed.destination,
		Label:       parsed.label,
		Title:       parsed.title,
		HasTitle:    parsed.hasTitle,
	}, semantic, parsed, next, true
}

func scanReferenceDefinition(source []byte, lines []physicalLine, index int) (referenceDefinitionParse, bool) {
	label, labelLine, destinationLine, position, ok := scanReferenceDefinitionStart(source, lines, index)
	if !ok {
		return referenceDefinitionParse{}, false
	}
	destination, position, ok := scanReferenceDestination(source, lines[destinationLine], position)
	if !ok {
		return referenceDefinitionParse{}, false
	}
	line := lines[index]
	indent, _ := ordinaryIndent(source, line)
	parsed := referenceDefinitionParse{
		label:       label,
		destination: destination,
		firstLine:   index,
		lastLine:    destinationLine,
		anchor:      line.start + indent,
		hasAnchor:   true,
		projectable: labelLine == index && destinationLine == index,
	}
	return scanReferenceDefinitionTitle(source, lines, destinationLine, position, parsed)
}

func scanReferenceDefinitionStart(source []byte, lines []physicalLine, index int) (string, int, int, int, bool) {
	if index < 0 || index >= len(lines) {
		return "", 0, 0, 0, false
	}
	line := lines[index]
	indent, ok := ordinaryIndent(source, line)
	if !ok {
		return "", 0, 0, 0, false
	}
	position := line.start + indent
	if position >= line.end || source[position] != '[' {
		return "", 0, 0, 0, false
	}
	label, labelLine, position, ok := scanReferenceLabel(source, lines, index, position+1)
	if !ok {
		return "", 0, 0, 0, false
	}
	position = skipReferenceSpace(source, lines[labelLine], position)
	destinationLine, position, ok := referenceDestinationLine(source, lines, labelLine, position)
	return label, labelLine, destinationLine, position, ok
}

func referenceDestinationLine(source []byte, lines []physicalLine, lineIndex, position int) (int, int, bool) {
	if position < lines[lineIndex].end {
		return lineIndex, position, true
	}
	lineIndex++
	if lineIndex >= len(lines) || !referenceParagraphContinuationLine(source, lines[lineIndex]) {
		return 0, 0, false
	}
	position = skipReferenceSpace(source, lines[lineIndex], lines[lineIndex].start)
	return lineIndex, position, true
}

func scanReferenceDefinitionTitle(source []byte, lines []physicalLine, destinationLine, position int, parsed referenceDefinitionParse) (referenceDefinitionParse, bool) {
	destinationEnd := position
	position = skipReferenceSpace(source, lines[destinationLine], position)
	if position < lines[destinationLine].end {
		if position == destinationEnd {
			return referenceDefinitionParse{}, false
		}
		return scanSameLineReferenceTitle(source, lines, destinationLine, position, parsed)
	}
	if title, end, recognized, consumed := scanReferenceTitleOnFollowingLine(source, lines, destinationLine+1); recognized {
		parsed.title = title
		parsed.hasTitle = true
		if consumed {
			parsed.lastLine = end.line
			parsed.projectable = false
		}
	}
	return parsed, true
}

func scanSameLineReferenceTitle(source []byte, lines []physicalLine, destinationLine, position int, parsed referenceDefinitionParse) (referenceDefinitionParse, bool) {
	title, end, ok := scanReferenceTitle(source, lines, destinationLine, position)
	if !ok || skipReferenceSpace(source, lines[end.line], end.position) != lines[end.line].end {
		return referenceDefinitionParse{}, false
	}
	parsed.title = title
	parsed.hasTitle = true
	parsed.lastLine = end.line
	if end.line != destinationLine {
		parsed.projectable = false
	}
	return parsed, true
}

type referencePosition struct {
	line     int
	position int
}

type referenceLabelScan struct {
	text             strings.Builder
	length           int
	hasNonWhitespace bool
}

func scanReferenceLabel(source []byte, lines []physicalLine, lineIndex, position int) (string, int, int, bool) {
	state := referenceLabelScan{}
	for lineIndex < len(lines) {
		line := lines[lineIndex]
		if blankLine(source, line) {
			return "", 0, 0, false
		}
		var closed, ok bool
		position, closed, ok = scanReferenceLabelLine(source, line, position, &state)
		if !ok {
			return "", 0, 0, false
		}
		if closed {
			return state.text.String(), lineIndex, position, true
		}
		lineIndex++
		if lineIndex >= len(lines) || !referenceParagraphContinuationLine(source, lines[lineIndex]) {
			return "", 0, 0, false
		}
		state.text.WriteByte('\n')
		state.length++
		position = lines[lineIndex].start
	}
	return "", 0, 0, false
}

func scanReferenceLabelLine(source []byte, line physicalLine, position int, state *referenceLabelScan) (int, bool, bool) {
	for position < line.end {
		value := source[position]
		if value == '\\' && position+1 < line.end && asciiPunctuation(source[position+1]) {
			state.text.WriteByte(value)
			state.text.WriteByte(source[position+1])
			position += 2
			state.length += 2
			state.hasNonWhitespace = true
			continue
		}
		if value == '[' {
			return position, false, false
		}
		if value == ']' {
			if position+1 < line.end && source[position+1] == ':' {
				return position + 2, true, state.hasNonWhitespace && state.length <= 999
			}
			return position, false, false
		}
		state.text.WriteByte(value)
		if !referenceWhitespace(value) {
			state.hasNonWhitespace = true
		}
		position++
		state.length++
		if state.length > 999 {
			return position, false, false
		}
	}
	return position, false, true
}

func scanReferenceDestination(source []byte, line physicalLine, position int) (string, int, bool) {
	if position >= line.end {
		return "", position, false
	}
	if source[position] == '<' {
		return scanEnclosedReferenceDestination(source, line, position+1)
	}
	return scanBareReferenceDestination(source, line, position)
}

func scanEnclosedReferenceDestination(source []byte, line physicalLine, position int) (string, int, bool) {
	start := position
	for position < line.end {
		if source[position] == '\n' || source[position] == '\r' || source[position] == '<' {
			return "", position, false
		}
		if source[position] == '>' {
			return string(source[start:position]), position + 1, true
		}
		position++
	}
	return "", position, false
}

func scanBareReferenceDestination(source []byte, line physicalLine, position int) (string, int, bool) {
	start := position
	depth := 0
	for position < line.end {
		value := source[position]
		if value == ' ' || value == '\t' {
			break
		}
		if value == '\\' && position+1 < line.end && asciiPunctuation(source[position+1]) {
			position += 2
			continue
		}
		var ok bool
		depth, ok = referenceDestinationDepth(value, depth)
		if !ok {
			return "", position, false
		}
		position++
	}
	if position == start {
		return "", position, false
	}
	return string(source[start:position]), position, true
}

func referenceDestinationDepth(value byte, depth int) (int, bool) {
	switch value {
	case '(':
		return depth + 1, true
	case ')':
		return depth - 1, depth > 0
	case '\n', '\r':
		return depth, false
	default:
		return depth, true
	}
}

func scanReferenceTitleOnFollowingLine(source []byte, lines []physicalLine, lineIndex int) (string, referencePosition, bool, bool) {
	if lineIndex >= len(lines) || blankLine(source, lines[lineIndex]) {
		return "", referencePosition{}, false, false
	}
	position := skipReferenceSpace(source, lines[lineIndex], lines[lineIndex].start)
	if position >= lines[lineIndex].end || !referenceTitleOpener(source[position]) {
		return "", referencePosition{}, false, false
	}
	title, end, ok := scanReferenceTitle(source, lines, lineIndex, position)
	if !ok {
		return "", referencePosition{}, false, false
	}
	consumed := skipReferenceSpace(source, lines[end.line], end.position) == lines[end.line].end
	return title, end, true, consumed
}

func scanReferenceTitle(source []byte, lines []physicalLine, lineIndex, position int) (string, referencePosition, bool) {
	if lineIndex >= len(lines) || position >= lines[lineIndex].end || !referenceTitleOpener(source[position]) {
		return "", referencePosition{}, false
	}
	opener := source[position]
	closer := opener
	if opener == '(' {
		closer = ')'
	}
	position++
	var title strings.Builder
	for lineIndex < len(lines) {
		line := lines[lineIndex]
		for position < line.end {
			value := source[position]
			if value == '\\' && position+1 < line.end && asciiPunctuation(source[position+1]) {
				title.WriteByte(value)
				title.WriteByte(source[position+1])
				position += 2
				continue
			}
			if value == closer {
				return title.String(), referencePosition{line: lineIndex, position: position + 1}, true
			}
			if opener == '(' && value == '(' {
				return "", referencePosition{}, false
			}
			title.WriteByte(value)
			position++
		}
		lineIndex++
		if lineIndex >= len(lines) || blankLine(source, lines[lineIndex]) {
			return "", referencePosition{}, false
		}
		title.WriteByte('\n')
		position = lines[lineIndex].start
	}
	return "", referencePosition{}, false
}

func referenceDefinitionSemanticRanges(source []byte, lines []physicalLine, first, last int) []parser.Range {
	ranges := make([]parser.Range, 0, last-first+1)
	for index := first; index <= last; index++ {
		line := lines[index]
		if blankLine(source, line) {
			continue
		}
		ranges = append(ranges, parser.Range{Start: line.start, End: line.end})
	}
	return ranges
}

func skipReferenceSpace(source []byte, line physicalLine, position int) int {
	for position < line.end && (source[position] == ' ' || source[position] == '\t') {
		position++
	}
	return position
}

func referenceTitleOpener(value byte) bool {
	return value == '\'' || value == '"' || value == '('
}

func referenceWhitespace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\n' || value == '\r'
}

func asciiPunctuation(value byte) bool {
	return value >= '!' && value <= '/' || value >= ':' && value <= '@' || value >= '[' && value <= '`' || value >= '{' && value <= '~'
}
