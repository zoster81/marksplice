package source

import (
	"bytes"
	"fmt"
)

// BlockquoteMapping binds one reviewed simple top-level blockquote to its exact source line.
type BlockquoteMapping struct {
	Range        Range
	LineRange    Range
	ContentRange Range
	MarkerRange  Range
}

// MapSimpleTopLevelBlockquote maps one source-proven single-line, single-paragraph top-level blockquote.
func MapSimpleTopLevelBlockquote(input []byte, observed, content Range) (BlockquoteMapping, error) {
	if !observed.Valid(len(input)) || !content.Valid(len(input)) || content.Start == content.End ||
		observed.Start >= content.Start || observed.End != content.End {
		return BlockquoteMapping{}, fmt.Errorf("%w: invalid observed/content ranges", ErrUnsupportedBlockquoteShape)
	}
	if containsLineBreak(input[content.Start:content.End]) {
		return BlockquoteMapping{}, fmt.Errorf("%w: content crosses a physical line", ErrUnsupportedBlockquoteShape)
	}

	lineStart := physicalLineStart(input, observed.Start)
	lineEnd := physicalLineEnd(input, content.End)
	if observed.End != lineEnd || content.End != lineEnd {
		return BlockquoteMapping{}, fmt.Errorf("%w: observed blockquote does not own one complete physical content line", ErrUnsupportedBlockquoteShape)
	}
	markerStart, ok := simpleBlockquoteMarker(input, lineStart, content.Start)
	if !ok || observed.Start < lineStart || observed.Start > markerStart {
		return BlockquoteMapping{}, fmt.Errorf("%w: unsupported blockquote marker prefix", ErrUnsupportedBlockquoteShape)
	}

	lineRangeEnd := lineEnd
	if next, ok := nextPhysicalLineStart(input, lineEnd); ok {
		lineRangeEnd = next
	}
	return BlockquoteMapping{
		Range:        observed,
		LineRange:    Range{Start: lineStart, End: lineRangeEnd},
		ContentRange: content,
		MarkerRange:  Range{Start: markerStart, End: markerStart + 1},
	}, nil
}

// ValidateCanonicalBlockquoteParagraph proves the exact source emitted for one
// constructed depth-1 blockquote paragraph.
func ValidateCanonicalBlockquoteParagraph(input []byte, outer Range, contentLines []Range) error {
	return ValidateCanonicalNestedBlockquoteParagraph(input, outer, contentLines, 1)
}

// ValidateCanonicalNestedBlockquoteBlocks proves one exact canonical quoted
// block sequence. innerSource must be canonical LF-terminated source produced
// for the child blocks before blockquote prefixes are added.
func ValidateCanonicalNestedBlockquoteBlocks(input []byte, outer Range, innerSource []byte, depth int) error {
	if depth < 1 || !outer.Valid(len(input)) || outer.Start == outer.End || len(innerSource) == 0 || innerSource[len(innerSource)-1] != '\n' {
		return fmt.Errorf("%w: invalid canonical blockquote block source", ErrUnsupportedBlockquoteShape)
	}
	if depth > (outer.End-outer.Start)/2 {
		return fmt.Errorf("%w: canonical marker depth exceeds source range", ErrUnsupportedBlockquoteShape)
	}

	inputPos := outer.Start
	innerPos := 0
	for innerPos < len(innerSource) {
		var err error
		inputPos, innerPos, err = validateCanonicalBlockquoteBlockLine(input, outer, innerSource, inputPos, innerPos, depth)
		if err != nil {
			return err
		}
	}
	if inputPos != outer.End {
		return fmt.Errorf("%w: canonical blockquote end changed", ErrUnsupportedBlockquoteShape)
	}
	return nil
}

func validateCanonicalBlockquoteBlockLine(input []byte, outer Range, innerSource []byte, inputPos, innerPos, depth int) (int, int, error) {
	lineEnd := bytes.IndexByte(innerSource[innerPos:], '\n')
	if lineEnd < 0 {
		return 0, 0, fmt.Errorf("%w: child source lost final LF", ErrUnsupportedBlockquoteShape)
	}
	lineEnd += innerPos
	prefixEnd := inputPos + depth*2
	if prefixEnd > outer.End || !canonicalBlockquotePrefix(input[inputPos:prefixEnd], depth) {
		return 0, 0, fmt.Errorf("%w: canonical marker changed", ErrUnsupportedBlockquoteShape)
	}
	lineLength := lineEnd - innerPos
	contentEnd := prefixEnd + lineLength
	if contentEnd > outer.End || !bytes.Equal(input[prefixEnd:contentEnd], innerSource[innerPos:lineEnd]) {
		return 0, 0, fmt.Errorf("%w: quoted child source changed", ErrUnsupportedBlockquoteShape)
	}
	nextInner := lineEnd + 1
	if nextInner == len(innerSource) {
		return contentEnd, nextInner, nil
	}
	if contentEnd >= outer.End || input[contentEnd] != '\n' {
		return 0, 0, fmt.Errorf("%w: canonical line separator changed", ErrUnsupportedBlockquoteShape)
	}
	return contentEnd + 1, nextInner, nil
}

// ValidateCanonicalNestedBlockquoteParagraph proves the exact source emitted
// for one constructed blockquote paragraph at explicit structural depth. Each
// content range must own one non-empty physical line prefixed by depth copies of
// canonical "> ", with LF between lines. The outer range excludes the final
// document line ending, matching the construction expectation model.
func ValidateCanonicalNestedBlockquoteParagraph(input []byte, outer Range, contentLines []Range, depth int) error {
	if depth < 1 || !outer.Valid(len(input)) || outer.Start == outer.End || len(contentLines) == 0 {
		return fmt.Errorf("%w: invalid canonical blockquote ranges", ErrUnsupportedBlockquoteShape)
	}

	lineStart := outer.Start
	for index, content := range contentLines {
		next, err := validateCanonicalBlockquoteLine(input, outer, content, lineStart, depth, index == len(contentLines)-1)
		if err != nil {
			return fmt.Errorf("canonical blockquote content line %d: %w", index, err)
		}
		lineStart = next
	}
	return nil
}

func validateCanonicalBlockquoteLine(input []byte, outer, content Range, lineStart, depth int, last bool) (int, error) {
	if lineStart < 0 || lineStart > len(input) || depth > (len(input)-lineStart)/2 {
		return 0, fmt.Errorf("%w: invalid canonical marker depth", ErrUnsupportedBlockquoteShape)
	}
	contentStart := lineStart + depth*2
	if !content.Valid(len(input)) || content.Start == content.End || content.Start != contentStart || content.End > outer.End {
		return 0, fmt.Errorf("%w: invalid content range", ErrUnsupportedBlockquoteShape)
	}
	if !canonicalBlockquotePrefix(input[lineStart:contentStart], depth) {
		return 0, fmt.Errorf("%w: canonical marker changed", ErrUnsupportedBlockquoteShape)
	}
	if containsLineBreak(input[content.Start:content.End]) {
		return 0, fmt.Errorf("%w: content crosses a physical line", ErrUnsupportedBlockquoteShape)
	}
	if last {
		if content.End != outer.End {
			return 0, fmt.Errorf("%w: canonical end changed", ErrUnsupportedBlockquoteShape)
		}
		return content.End, nil
	}
	if content.End >= len(input) || input[content.End] != '\n' {
		return 0, fmt.Errorf("%w: canonical line separator changed", ErrUnsupportedBlockquoteShape)
	}
	return content.End + 1, nil
}

func canonicalBlockquotePrefix(prefix []byte, depth int) bool {
	if len(prefix) != depth*2 {
		return false
	}
	for level := 0; level < depth; level++ {
		offset := level * 2
		if prefix[offset] != '>' || prefix[offset+1] != ' ' {
			return false
		}
	}
	return true
}

func simpleBlockquoteMarker(input []byte, lineStart, contentStart int) (int, bool) {
	if lineStart < 0 || contentStart <= lineStart || contentStart > len(input) {
		return 0, false
	}
	pos := lineStart
	for pos < contentStart && pos-lineStart < 3 && input[pos] == ' ' {
		pos++
	}
	if pos >= contentStart || input[pos] != '>' {
		return 0, false
	}
	marker := pos
	pos++
	if pos == contentStart {
		return marker, true
	}
	if pos+1 == contentStart && input[pos] == ' ' {
		return marker, true
	}
	return 0, false
}
