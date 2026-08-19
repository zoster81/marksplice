package source

import "fmt"

// CodeSpanMapping binds one simple single-line code span to its exact backtick run.
type CodeSpanMapping struct {
	Range        Range
	ContentRange Range
	FenceLength  int
}

// MapSimpleCodeSpan maps a code span whose semantic content is directly adjacent to equal backtick runs.
func MapSimpleCodeSpan(input []byte, anchor int, content Range) (CodeSpanMapping, error) {
	if anchor < 0 || anchor >= len(input) || !content.Valid(len(input)) || content.Start == content.End || input[anchor] != '`' {
		return CodeSpanMapping{}, fmt.Errorf("%w: invalid anchor or content", ErrUnsupportedCodeSpanShape)
	}
	for _, b := range input[content.Start:content.End] {
		if b == '\r' || b == '\n' {
			return CodeSpanMapping{}, fmt.Errorf("%w: content crosses a physical line", ErrUnsupportedCodeSpanShape)
		}
	}

	lineStart := physicalLineStart(input, anchor)
	lineEnd := physicalLineEnd(input, content.End)
	if anchor > lineStart && input[anchor-1] == '`' {
		return CodeSpanMapping{}, fmt.Errorf("%w: anchor is inside a larger opener run", ErrUnsupportedCodeSpanShape)
	}
	openerLength := byteRunLength(input, anchor, lineEnd, '`')
	if openerLength == 0 || content.Start != anchor+openerLength {
		return CodeSpanMapping{}, fmt.Errorf("%w: semantic content is not directly after the opener", ErrUnsupportedCodeSpanShape)
	}
	closingLength := byteRunLength(input, content.End, lineEnd, '`')
	if closingLength != openerLength {
		return CodeSpanMapping{}, fmt.Errorf("%w: closing run length %d does not match opener %d", ErrUnsupportedCodeSpanShape, closingLength, openerLength)
	}

	return CodeSpanMapping{
		Range:        Range{Start: anchor, End: content.End + closingLength},
		ContentRange: content,
		FenceLength:  openerLength,
	}, nil
}

// EmphasisMapping binds one simple emphasis/strong node to exact '*' or '_' delimiters.
type EmphasisMapping struct {
	Range        Range
	ContentRange Range
	Marker       byte
	Level        int
}

// MapSimpleEmphasis maps a plain-text single-line emphasis/strong node with exact one/two-character delimiters.
func MapSimpleEmphasis(input []byte, anchor int, content Range, level int) (EmphasisMapping, error) {
	if anchor < 0 || anchor >= len(input) || !content.Valid(len(input)) || content.Start == content.End || level < 1 || level > 2 {
		return EmphasisMapping{}, fmt.Errorf("%w: invalid anchor, content, or level", ErrUnsupportedEmphasisShape)
	}
	marker := input[anchor]
	if marker != '*' && marker != '_' {
		return EmphasisMapping{}, fmt.Errorf("%w: unsupported delimiter %q", ErrUnsupportedEmphasisShape, marker)
	}
	for _, b := range input[content.Start:content.End] {
		if b == '\r' || b == '\n' {
			return EmphasisMapping{}, fmt.Errorf("%w: content crosses a physical line", ErrUnsupportedEmphasisShape)
		}
	}

	lineStart := physicalLineStart(input, anchor)
	lineEnd := physicalLineEnd(input, content.End)
	if anchor > lineStart && input[anchor-1] == marker {
		return EmphasisMapping{}, fmt.Errorf("%w: anchor is inside a larger delimiter run", ErrUnsupportedEmphasisShape)
	}
	if byteRunLength(input, anchor, lineEnd, marker) != level || content.Start != anchor+level {
		return EmphasisMapping{}, fmt.Errorf("%w: opening delimiter does not match semantic level", ErrUnsupportedEmphasisShape)
	}
	if byteRunLength(input, content.End, lineEnd, marker) != level {
		return EmphasisMapping{}, fmt.Errorf("%w: closing delimiter does not match semantic level", ErrUnsupportedEmphasisShape)
	}

	return EmphasisMapping{
		Range:        Range{Start: anchor, End: content.End + level},
		ContentRange: content,
		Marker:       marker,
		Level:        level,
	}, nil
}

func byteRunLength(input []byte, start, limit int, value byte) int {
	if start < 0 || start >= limit || start >= len(input) {
		return 0
	}
	end := start
	for end < limit && end < len(input) && input[end] == value {
		end++
	}
	return end - start
}

// StrikethroughMapping binds simple GFM strikethrough content to its exact tilde delimiters.
type StrikethroughMapping struct {
	Range           Range
	ContentRange    Range
	DelimiterLength int
}

// MapSimpleStrikethrough maps one non-empty single-line plain-text strikethrough content range to exact GFM delimiters.
func MapSimpleStrikethrough(input []byte, content Range) (StrikethroughMapping, error) {
	if !content.Valid(len(input)) || content.Start == content.End {
		return StrikethroughMapping{}, fmt.Errorf("%w: invalid or empty content range", ErrUnsupportedStrikethroughShape)
	}
	for _, b := range input[content.Start:content.End] {
		if b == '\r' || b == '\n' {
			return StrikethroughMapping{}, fmt.Errorf("%w: content crosses a physical line", ErrUnsupportedStrikethroughShape)
		}
	}

	left := 0
	for i := content.Start - 1; i >= 0 && input[i] == '~'; i-- {
		left++
	}
	right := 0
	for i := content.End; i < len(input) && input[i] == '~'; i++ {
		right++
	}
	if left < 1 || left > 2 || right != left {
		return StrikethroughMapping{}, fmt.Errorf("%w: expected matching one- or two-tilde delimiters, got %d/%d", ErrUnsupportedStrikethroughShape, left, right)
	}

	return StrikethroughMapping{
		Range:           Range{Start: content.Start - left, End: content.End + right},
		ContentRange:    content,
		DelimiterLength: left,
	}, nil
}
