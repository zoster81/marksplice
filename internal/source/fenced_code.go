package source

import (
	"bytes"
	"fmt"
)

// FencedCodeMapping binds one supported fenced code block to its exact source fences and content.
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

// MapFencedCode maps one supported non-empty semantic fenced-code body to top-level source syntax.
// Multiline bodies are supported only for an unindented opening fence so the semantic body remains one exact contiguous source span.
func MapFencedCode(input []byte, content Range) (FencedCodeMapping, error) {
	if !content.Valid(len(input)) || content.Start == content.End {
		return FencedCodeMapping{}, fmt.Errorf("%w: invalid or empty content range", ErrUnsupportedFencedCodeShape)
	}
	multiline := containsLineBreak(input[content.Start:content.End])

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
	if multiline && (opening.indent != 0 || content.Start != contentLineStart) {
		return FencedCodeMapping{}, fmt.Errorf("%w: multiline content requires an unindented contiguous body", ErrUnsupportedFencedCodeShape)
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
	char, length, indent, next, ok := scanTopLevelFenceRun(line, 0)
	if !ok || length < 3 {
		return fenceOpening{}, false
	}
	infoStart := skipHorizontalSpace(line, next, len(line))
	infoEnd := len(line)
	for infoEnd > infoStart && isHorizontalSpace(line[infoEnd-1]) {
		infoEnd--
	}
	if char == '`' && bytes.IndexByte(line[infoStart:infoEnd], '`') >= 0 {
		return fenceOpening{}, false
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
	_, length, indent, next, ok := scanTopLevelFenceRun(line, char)
	if !ok || length < minimumLength || !allHorizontalSpace(line[next:]) {
		return 0, 0, false
	}
	return length, indent, true
}

func scanTopLevelFenceRun(line []byte, expected byte) (byte, int, int, int, bool) {
	indent := 0
	for indent < len(line) && line[indent] == ' ' {
		indent++
	}
	if indent > 3 || indent == len(line) {
		return 0, 0, 0, 0, false
	}
	char := line[indent]
	if expected != 0 && char != expected || expected == 0 && char != '`' && char != '~' {
		return 0, 0, 0, 0, false
	}
	next := indent
	for next < len(line) && line[next] == char {
		next++
	}
	return char, next - indent, indent, next, true
}
