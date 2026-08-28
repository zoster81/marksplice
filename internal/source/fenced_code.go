package source

import (
	"bytes"
	"fmt"
)

// FencedBlockMapping binds one parser-proven top-level fenced block to its exact
// source container, delimiter runs, info string, and per-line payload ranges.
type FencedBlockMapping struct {
	Range              Range
	OpeningFenceRange  Range
	InfoRange          Range
	ContentRanges      []Range
	ClosingFenceRange  Range
	FenceChar          byte
	OpeningFenceLength int
	ClosingFenceLength int
	OpeningIndent      int
	ClosingIndent      int
	Closed             bool
}

// FencedCodeMapping binds one supported contiguous fenced-code payload to its
// exact source fences and content. Range retains the historical M5/M52 meaning:
// it excludes the closing fence line terminator when a closing fence exists.
type FencedCodeMapping struct {
	Range              Range
	ContentRange       Range
	InfoRange          Range
	FenceChar          byte
	FenceLength        int
	ClosingFenceLength int
	OpeningIndent      int
	ClosingIndent      int
	Closed             bool
}

// MapFencedBlock maps one parser-proven top-level fenced block from its opening
// fence anchor and source-backed semantic payload-line ranges. Embedded payload
// bytes remain opaque; this function proves only Markdown/source ownership.
func MapFencedBlock(input []byte, anchor int, contentRanges []Range, info string) (FencedBlockMapping, error) {
	if anchor < 0 || anchor >= len(input) {
		return FencedBlockMapping{}, fmt.Errorf("%w: opening fence anchor is outside source", ErrUnsupportedFencedCodeShape)
	}
	openingStart := physicalLineStart(input, anchor)
	openingEnd := physicalLineEnd(input, anchor)
	opening, ok := parseTopLevelFenceOpening(input, openingStart, openingEnd)
	if !ok || openingStart+opening.indent != anchor {
		return FencedBlockMapping{}, fmt.Errorf("%w: opening fence is not source-proven", ErrUnsupportedFencedCodeShape)
	}
	if string(input[opening.infoRange.Start:opening.infoRange.End]) != info {
		return FencedBlockMapping{}, fmt.Errorf("%w: semantic info string disagrees with source", ErrUnsupportedFencedCodeShape)
	}
	return mapFencedBlockFromOpening(input, anchor, openingStart, openingEnd, opening, contentRanges)
}

func mapFencedBlockFromOpening(input []byte, anchor, openingStart, openingEnd int, opening fenceOpening, contentRanges []Range) (FencedBlockMapping, error) {
	mapping := FencedBlockMapping{
		Range:              Range{Start: openingStart, End: len(input)},
		OpeningFenceRange:  Range{Start: anchor, End: anchor + opening.length},
		InfoRange:          opening.infoRange,
		ContentRanges:      append([]Range(nil), contentRanges...),
		FenceChar:          opening.char,
		OpeningFenceLength: opening.length,
		OpeningIndent:      opening.indent,
	}
	bodyStart, ok := nextPhysicalLineStart(input, openingEnd)
	if !ok || bodyStart == len(input) {
		if len(contentRanges) != 0 {
			return FencedBlockMapping{}, fmt.Errorf("%w: semantic payload exists without a physical body line", ErrUnsupportedFencedCodeShape)
		}
		return mapping, nil
	}
	return mapFencedBlockBody(input, bodyStart, contentRanges, mapping)
}

func mapFencedBlockBody(input []byte, bodyStart int, contentRanges []Range, mapping FencedBlockMapping) (FencedBlockMapping, error) {
	contentIndex := 0
	for lineStart := bodyStart; lineStart < len(input); {
		lineEnd := physicalLineEnd(input, lineStart)
		closingLength, closingIndent, closed := parseTopLevelFenceClosing(input, lineStart, lineEnd, mapping.FenceChar, mapping.OpeningFenceLength)
		if closed {
			if contentIndex != len(contentRanges) {
				return FencedBlockMapping{}, fmt.Errorf("%w: closing fence precedes parser-proven payload", ErrUnsupportedFencedCodeShape)
			}
			mapping.Closed = true
			mapping.ClosingFenceLength = closingLength
			mapping.ClosingIndent = closingIndent
			mapping.ClosingFenceRange = Range{Start: lineStart + closingIndent, End: lineStart + closingIndent + closingLength}
			mapping.Range.End = ownedPhysicalLineEnd(input, lineEnd)
			return mapping, nil
		}
		if contentIndex >= len(contentRanges) || !fencedContentRangeMatchesLine(input, contentRanges[contentIndex], lineStart, lineEnd) {
			return FencedBlockMapping{}, fmt.Errorf("%w: parser-proven payload line disagrees with source", ErrUnsupportedFencedCodeShape)
		}
		contentIndex++
		next, ok := nextPhysicalLineStart(input, lineEnd)
		if !ok {
			break
		}
		lineStart = next
	}
	if contentIndex != len(contentRanges) {
		return FencedBlockMapping{}, fmt.Errorf("%w: parser-proven payload extends beyond source", ErrUnsupportedFencedCodeShape)
	}
	return mapping, nil
}

func fencedContentRangeMatchesLine(input []byte, content Range, lineStart, lineEnd int) bool {
	return content.Valid(len(input)) && content.Start >= lineStart && content.Start <= lineEnd &&
		physicalLineStart(input, content.Start) == lineStart && content.End == lineEnd
}

func ownedPhysicalLineEnd(input []byte, lineEnd int) int {
	if next, ok := nextPhysicalLineStart(input, lineEnd); ok {
		return next
	}
	return lineEnd
}

// MapFencedCode maps one supported non-empty semantic fenced-code body to the
// historical contiguous replacement contract. Multiline bodies remain editable
// only for an unindented opening fence. M103 also permits an unclosed block when
// the complete semantic body is still one exact contiguous source span.
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

	closure, ok := scanContiguousFencedCodeBody(input, contentLineStart, content, opening)
	if !ok {
		return FencedCodeMapping{}, fmt.Errorf("%w: semantic content is not one contiguous physical body", ErrUnsupportedFencedCodeShape)
	}

	legacyRange := Range{Start: openingStart, End: len(input)}
	if closure.closed {
		legacyRange.End = closure.lineEnd
	}
	return FencedCodeMapping{
		Range:              legacyRange,
		ContentRange:       content,
		InfoRange:          opening.infoRange,
		FenceChar:          opening.char,
		FenceLength:        opening.length,
		ClosingFenceLength: closure.length,
		OpeningIndent:      opening.indent,
		ClosingIndent:      closure.indent,
		Closed:             closure.closed,
	}, nil
}

type fencedCodeClosure struct {
	length  int
	indent  int
	lineEnd int
	closed  bool
}

func scanContiguousFencedCodeBody(input []byte, bodyStart int, content Range, opening fenceOpening) (fencedCodeClosure, bool) {
	firstLine := true
	for lineStart := bodyStart; lineStart < len(input); {
		lineEnd := physicalLineEnd(input, lineStart)
		if _, _, closed := parseTopLevelFenceClosing(input, lineStart, lineEnd, opening.char, opening.length); closed {
			return fencedCodeClosure{}, false
		}
		if firstLine {
			if content.Start < lineStart || content.Start > lineEnd || physicalLineStart(input, content.Start) != bodyStart {
				return fencedCodeClosure{}, false
			}
			firstLine = false
		}
		if lineEnd > content.End {
			return fencedCodeClosure{}, false
		}
		if lineEnd == content.End {
			next, ok := nextPhysicalLineStart(input, lineEnd)
			if !ok || next >= len(input) {
				return fencedCodeClosure{}, true
			}
			closingEnd := physicalLineEnd(input, next)
			closingLength, closingIndent, closed := parseTopLevelFenceClosing(input, next, closingEnd, opening.char, opening.length)
			if !closed {
				return fencedCodeClosure{}, false
			}
			return fencedCodeClosure{length: closingLength, indent: closingIndent, lineEnd: closingEnd, closed: true}, true
		}
		next, ok := nextPhysicalLineStart(input, lineEnd)
		if !ok || next > content.End {
			return fencedCodeClosure{}, false
		}
		lineStart = next
	}
	return fencedCodeClosure{}, false
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
