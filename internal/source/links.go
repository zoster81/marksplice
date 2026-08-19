package source

import "fmt"

// InlineLinkMapping binds one simple inline link to its exact source destination.
type InlineLinkMapping struct {
	Range            Range
	LabelRange       Range
	DestinationRange Range
	TitleRange       Range
	AngleDestination bool
	HasTitle         bool
}

// MapSimpleInlineLink maps a single-line plain-text-label inline link to its source destination.
func MapSimpleInlineLink(input []byte, anchor int, label Range, destination, title string, hasTitle bool) (InlineLinkMapping, error) {
	if anchor < 0 || anchor >= len(input) || input[anchor] != '[' || !label.Valid(len(input)) || label.Start == label.End {
		return InlineLinkMapping{}, fmt.Errorf("%w: invalid anchor or label range", ErrUnsupportedInlineLinkShape)
	}
	lineStart := physicalLineStart(input, anchor)
	lineEnd := physicalLineEnd(input, label.End)
	if label.Start != anchor+1 || label.End >= lineEnd || input[label.End] != ']' || label.End+1 >= lineEnd || input[label.End+1] != '(' {
		return InlineLinkMapping{}, fmt.Errorf("%w: label is not followed by an inline-link destination", ErrUnsupportedInlineLinkShape)
	}
	for _, b := range input[anchor : label.End+2] {
		if b == '\r' || b == '\n' {
			return InlineLinkMapping{}, fmt.Errorf("%w: link prefix crosses a physical line", ErrUnsupportedInlineLinkShape)
		}
	}
	if lineStart > anchor || destination == "" {
		return InlineLinkMapping{}, fmt.Errorf("%w: unsupported empty destination", ErrUnsupportedInlineLinkShape)
	}

	pos := skipHorizontalSpace(input, label.End+2, lineEnd)
	destinationRange, angle, next, ok := scanMarkdownLinkDestination(input, pos, lineEnd)
	if !ok || string(input[destinationRange.Start:destinationRange.End]) != destination {
		return InlineLinkMapping{}, fmt.Errorf("%w: source destination does not match semantic destination", ErrUnsupportedInlineLinkShape)
	}

	pos = skipHorizontalSpace(input, next, lineEnd)
	mapping := InlineLinkMapping{
		LabelRange:       label,
		DestinationRange: destinationRange,
		AngleDestination: angle,
		HasTitle:         hasTitle,
	}
	if hasTitle {
		titleRange, titleNext, ok := scanMarkdownLinkTitle(input, pos, lineEnd)
		if !ok || string(input[titleRange.Start:titleRange.End]) != title {
			return InlineLinkMapping{}, fmt.Errorf("%w: source title does not match semantic title", ErrUnsupportedInlineLinkShape)
		}
		mapping.TitleRange = titleRange
		pos = skipHorizontalSpace(input, titleNext, lineEnd)
	}
	if pos >= lineEnd || input[pos] != ')' {
		return InlineLinkMapping{}, fmt.Errorf("%w: missing inline-link closing parenthesis", ErrUnsupportedInlineLinkShape)
	}
	mapping.Range = Range{Start: anchor, End: pos + 1}
	return mapping, nil
}

// ReferenceDefinitionMapping binds one single-line link reference definition to its exact destination.
type ReferenceDefinitionMapping struct {
	Range            Range
	DestinationRange Range
	TitleRange       Range
	AngleDestination bool
	HasTitle         bool
}

// MapSingleLineReferenceDefinition maps one Goldmark-recognized single-line reference definition to its destination bytes.
func MapSingleLineReferenceDefinition(input []byte, observation Range, label, destination, title string, hasTitle bool) (ReferenceDefinitionMapping, error) {
	if !observation.Valid(len(input)) || observation.Start == observation.End || destination == "" {
		return ReferenceDefinitionMapping{}, fmt.Errorf("%w: invalid observation or empty destination", ErrUnsupportedReferenceDefinitionShape)
	}
	lineStart := physicalLineStart(input, observation.Start)
	lineEnd := physicalLineEnd(input, observation.End)
	for _, b := range input[lineStart:lineEnd] {
		if b == '\r' || b == '\n' {
			return ReferenceDefinitionMapping{}, fmt.Errorf("%w: definition crosses a physical line", ErrUnsupportedReferenceDefinitionShape)
		}
	}

	pos := lineStart
	indent := 0
	for pos < lineEnd && input[pos] == ' ' && indent < 4 {
		pos++
		indent++
	}
	if indent > 3 || pos >= lineEnd || input[pos] != '[' {
		return ReferenceDefinitionMapping{}, fmt.Errorf("%w: unsupported definition indentation or label opener", ErrUnsupportedReferenceDefinitionShape)
	}
	labelRange, next, ok := scanBracketContent(input, pos, lineEnd)
	if !ok || string(input[labelRange.Start:labelRange.End]) != label || next >= lineEnd || input[next] != ':' {
		return ReferenceDefinitionMapping{}, fmt.Errorf("%w: source label does not match semantic label", ErrUnsupportedReferenceDefinitionShape)
	}
	pos = skipHorizontalSpace(input, next+1, lineEnd)
	destinationRange, angle, next, ok := scanMarkdownLinkDestination(input, pos, lineEnd)
	if !ok || string(input[destinationRange.Start:destinationRange.End]) != destination {
		return ReferenceDefinitionMapping{}, fmt.Errorf("%w: source destination does not match semantic destination", ErrUnsupportedReferenceDefinitionShape)
	}

	pos = next
	spacesStart := pos
	pos = skipHorizontalSpace(input, pos, lineEnd)
	mapping := ReferenceDefinitionMapping{
		Range:            Range{Start: lineStart, End: lineEnd},
		DestinationRange: destinationRange,
		AngleDestination: angle,
		HasTitle:         hasTitle,
	}
	if hasTitle {
		if pos == spacesStart {
			return ReferenceDefinitionMapping{}, fmt.Errorf("%w: title is not separated from destination", ErrUnsupportedReferenceDefinitionShape)
		}
		titleRange, titleNext, ok := scanMarkdownLinkTitle(input, pos, lineEnd)
		if !ok || string(input[titleRange.Start:titleRange.End]) != title {
			return ReferenceDefinitionMapping{}, fmt.Errorf("%w: source title does not match semantic title", ErrUnsupportedReferenceDefinitionShape)
		}
		mapping.TitleRange = titleRange
		pos = skipHorizontalSpace(input, titleNext, lineEnd)
	}
	if pos != lineEnd {
		return ReferenceDefinitionMapping{}, fmt.Errorf("%w: unexpected bytes after definition", ErrUnsupportedReferenceDefinitionShape)
	}
	return mapping, nil
}

// AutoLinkMapping binds one semantic autolink value to its exact source token.
type AutoLinkMapping struct {
	Range        Range
	ContentRange Range
	Angle        bool
	Email        bool
}

// MapAutoLink maps one single-line angle or bare GFM autolink to its source token.
func MapAutoLink(input []byte, anchor int, content Range, value string, email bool) (AutoLinkMapping, error) {
	if anchor < 0 || anchor >= len(input) || !content.Valid(len(input)) || content.Start == content.End || string(input[content.Start:content.End]) != value {
		return AutoLinkMapping{}, fmt.Errorf("%w: invalid anchor/content or semantic value mismatch", ErrUnsupportedAutoLinkShape)
	}
	for _, b := range input[content.Start:content.End] {
		if b == '\r' || b == '\n' {
			return AutoLinkMapping{}, fmt.Errorf("%w: content crosses a physical line", ErrUnsupportedAutoLinkShape)
		}
	}

	if input[anchor] == '<' {
		if content.Start != anchor+1 || content.End >= len(input) || input[content.End] != '>' {
			return AutoLinkMapping{}, fmt.Errorf("%w: angle autolink boundary mismatch", ErrUnsupportedAutoLinkShape)
		}
		return AutoLinkMapping{
			Range:        Range{Start: anchor, End: content.End + 1},
			ContentRange: content,
			Angle:        true,
			Email:        email,
		}, nil
	}
	if content.Start != anchor {
		return AutoLinkMapping{}, fmt.Errorf("%w: bare autolink anchor does not match content start", ErrUnsupportedAutoLinkShape)
	}
	return AutoLinkMapping{
		Range:        content,
		ContentRange: content,
		Email:        email,
	}, nil
}

func scanBracketContent(input []byte, opener, limit int) (Range, int, bool) {
	if opener < 0 || opener >= limit || input[opener] != '[' {
		return Range{}, opener, false
	}
	for i := opener + 1; i < limit; i++ {
		if input[i] == '\\' && i+1 < limit && isASCIIPunctuation(input[i+1]) {
			i++
			continue
		}
		if input[i] == ']' {
			return Range{Start: opener + 1, End: i}, i + 1, i > opener+1
		}
	}
	return Range{}, opener, false
}

func scanMarkdownLinkDestination(input []byte, start, limit int) (Range, bool, int, bool) {
	if start < 0 || start >= limit {
		return Range{}, false, start, false
	}
	if input[start] == '<' {
		for i := start + 1; i < limit; i++ {
			if input[i] == '\\' && i+1 < limit && isASCIIPunctuation(input[i+1]) {
				i++
				continue
			}
			if input[i] == '>' {
				if i == start+1 {
					return Range{}, true, start, false
				}
				return Range{Start: start + 1, End: i}, true, i + 1, true
			}
		}
		return Range{}, true, start, false
	}

	depth := 0
	i := start
	for i < limit {
		b := input[i]
		if b == '\\' && i+1 < limit && isASCIIPunctuation(input[i+1]) {
			i += 2
			continue
		}
		if b == '(' {
			depth++
			i++
			continue
		}
		if b == ')' {
			if depth == 0 {
				break
			}
			depth--
			i++
			continue
		}
		if isHorizontalSpace(b) {
			break
		}
		i++
	}
	if i == start || depth != 0 {
		return Range{}, false, start, false
	}
	return Range{Start: start, End: i}, false, i, true
}

func scanMarkdownLinkTitle(input []byte, start, limit int) (Range, int, bool) {
	if start < 0 || start >= limit {
		return Range{}, start, false
	}
	opener := input[start]
	closer := opener
	if opener == '(' {
		closer = ')'
	} else if opener != '\'' && opener != '"' {
		return Range{}, start, false
	}
	for i := start + 1; i < limit; i++ {
		if input[i] == '\\' && i+1 < limit && isASCIIPunctuation(input[i+1]) {
			i++
			continue
		}
		if input[i] == closer {
			return Range{Start: start + 1, End: i}, i + 1, true
		}
	}
	return Range{}, start, false
}

func isASCIIPunctuation(b byte) bool {
	return b >= '!' && b <= '/' || b >= ':' && b <= '@' || b >= '[' && b <= '`' || b >= '{' && b <= '~'
}
