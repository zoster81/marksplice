package source

import (
	"bytes"
	"errors"
)

var ErrUnsupportedHTMLShape = errors.New("unsupported HTML source shape")

// HTMLCommentMapping binds a parser-recognized inline comment to exact delimiters and editable payload bytes.
type HTMLCommentMapping struct {
	Range        Range
	ContentRange Range
}

// MapHTMLComment maps one single-line raw-HTML range containing an HTML comment.
// Horizontal padding immediately inside the delimiters is preserved outside ContentRange.
func MapHTMLComment(input []byte, raw Range) (HTMLCommentMapping, error) {
	if !raw.Valid(len(input)) || raw.End-raw.Start < 7 {
		return HTMLCommentMapping{}, ErrUnsupportedHTMLShape
	}
	value := input[raw.Start:raw.End]
	if !bytes.HasPrefix(value, []byte("<!--")) || !bytes.HasSuffix(value, []byte("-->")) {
		return HTMLCommentMapping{}, ErrUnsupportedHTMLShape
	}
	if containsLineBreak(value) {
		return HTMLCommentMapping{}, ErrUnsupportedHTMLShape
	}

	start := raw.Start + 4
	end := raw.End - 3
	for start < end && isHorizontalSpace(input[start]) {
		start++
	}
	for end > start && isHorizontalSpace(input[end-1]) {
		end--
	}
	if start == end {
		return HTMLCommentMapping{}, ErrUnsupportedHTMLShape
	}
	return HTMLCommentMapping{Range: raw, ContentRange: Range{Start: start, End: end}}, nil
}

// HTMLAnchorMapping binds one simple <a> opening tag to a quoted id/name attribute value.
type HTMLAnchorMapping struct {
	Range        Range
	ContentRange Range
	Attribute    string
	Quote        byte
}

type htmlAttribute struct {
	name     []byte
	value    Range
	quote    byte
	hasValue bool
}

type htmlOpeningTagScanner struct {
	input []byte
	pos   int
	limit int
}

// MapSimpleHTMLAnchor maps a parser-recognized single-line opening <a> tag with exactly one quoted id/name attribute.
func MapSimpleHTMLAnchor(input []byte, raw Range) (HTMLAnchorMapping, error) {
	if !raw.Valid(len(input)) || raw.End-raw.Start < 7 {
		return HTMLAnchorMapping{}, ErrUnsupportedHTMLShape
	}
	value := input[raw.Start:raw.End]
	if containsLineBreak(value) {
		return HTMLAnchorMapping{}, ErrUnsupportedHTMLShape
	}

	scanner, tagName, ok := newHTMLOpeningTagScanner(value)
	if !ok || !bytes.EqualFold(tagName, []byte("a")) {
		return HTMLAnchorMapping{}, ErrUnsupportedHTMLShape
	}

	var candidate HTMLAnchorMapping
	found := false
	for {
		attribute, done, ok := scanner.nextAttribute()
		if !ok {
			return HTMLAnchorMapping{}, ErrUnsupportedHTMLShape
		}
		if done {
			break
		}
		name, target := htmlAnchorAttributeName(attribute.name)
		if !target {
			continue
		}
		if found || !attribute.hasValue || attribute.quote == 0 || attribute.value.Start == attribute.value.End {
			return HTMLAnchorMapping{}, ErrUnsupportedHTMLShape
		}
		candidate = HTMLAnchorMapping{
			Range:        raw,
			ContentRange: Range{Start: raw.Start + attribute.value.Start, End: raw.Start + attribute.value.End},
			Attribute:    name,
			Quote:        attribute.quote,
		}
		found = true
	}
	if !found {
		return HTMLAnchorMapping{}, ErrUnsupportedHTMLShape
	}
	return candidate, nil
}

func newHTMLOpeningTagScanner(input []byte) (*htmlOpeningTagScanner, []byte, bool) {
	if len(input) < 3 || input[0] != '<' || input[len(input)-1] != '>' || input[1] == '/' {
		return nil, nil, false
	}
	pos := 1
	start := pos
	for pos < len(input)-1 && isHTMLNameByte(input[pos]) {
		pos++
	}
	if pos == start {
		return nil, nil, false
	}
	return &htmlOpeningTagScanner{input: input, pos: pos, limit: len(input) - 1}, input[start:pos], true
}

func (s *htmlOpeningTagScanner) nextAttribute() (htmlAttribute, bool, bool) {
	s.skipSpace()
	if s.pos >= s.limit {
		return htmlAttribute{}, true, true
	}
	if s.input[s.pos] == '/' && s.pos+1 == s.limit {
		s.pos++
		return htmlAttribute{}, true, true
	}

	name, ok := s.scanAttributeName()
	if !ok {
		return htmlAttribute{}, false, false
	}
	s.skipSpace()
	if s.pos >= s.limit || s.input[s.pos] != '=' {
		return htmlAttribute{name: name}, false, true
	}

	s.pos++
	s.skipSpace()
	value, quote, ok := s.scanAttributeValue()
	if !ok {
		return htmlAttribute{}, false, false
	}
	return htmlAttribute{name: name, value: value, quote: quote, hasValue: true}, false, true
}

func (s *htmlOpeningTagScanner) skipSpace() {
	for s.pos < s.limit && isHorizontalSpace(s.input[s.pos]) {
		s.pos++
	}
}

func (s *htmlOpeningTagScanner) scanAttributeName() ([]byte, bool) {
	if s.pos >= s.limit || !isHTMLAttributeNameStart(s.input[s.pos]) {
		return nil, false
	}
	start := s.pos
	s.pos++
	for s.pos < s.limit && isHTMLAttributeNameByte(s.input[s.pos]) {
		s.pos++
	}
	return s.input[start:s.pos], true
}

func (s *htmlOpeningTagScanner) scanAttributeValue() (Range, byte, bool) {
	if s.pos >= s.limit {
		return Range{}, 0, false
	}
	quote := s.input[s.pos]
	if quote == '\'' || quote == '"' {
		return s.scanQuotedAttributeValue(quote)
	}
	start := s.pos
	for s.pos < s.limit && !isHorizontalSpace(s.input[s.pos]) && s.input[s.pos] != '>' {
		s.pos++
	}
	if s.pos == start {
		return Range{}, 0, false
	}
	return Range{Start: start, End: s.pos}, 0, true
}

func (s *htmlOpeningTagScanner) scanQuotedAttributeValue(quote byte) (Range, byte, bool) {
	start := s.pos + 1
	s.pos++
	for s.pos < s.limit && s.input[s.pos] != quote {
		s.pos++
	}
	if s.pos >= s.limit {
		return Range{}, 0, false
	}
	end := s.pos
	s.pos++
	return Range{Start: start, End: end}, quote, true
}

func htmlAnchorAttributeName(name []byte) (string, bool) {
	if bytes.EqualFold(name, []byte("id")) {
		return "id", true
	}
	if bytes.EqualFold(name, []byte("name")) {
		return "name", true
	}
	return "", false
}

func isHTMLNameByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '-'
}

func isHTMLAttributeNameStart(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b == '_' || b == ':'
}

func isHTMLAttributeNameByte(b byte) bool {
	return isHTMLAttributeNameStart(b) || b >= '0' && b <= '9' || b == '.' || b == '-'
}
