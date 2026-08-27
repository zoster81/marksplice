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
	lineEnd, destinationStart, err := validateInlineLinkPrefix(input, anchor, label, destination)
	if err != nil {
		return InlineLinkMapping{}, err
	}
	policy := inlineTitleForbidden
	if hasTitle {
		policy = inlineTitleRequired
	}
	parsed, issue := scanInlineDestinationTail(input, destinationStart, lineEnd, policy)
	if issue == inlineTailInvalidDestination || string(input[parsed.destinationRange.Start:parsed.destinationRange.End]) != destination {
		return InlineLinkMapping{}, fmt.Errorf("%w: source destination does not match semantic destination", ErrUnsupportedInlineLinkShape)
	}
	if issue == inlineTailInvalidTitle || hasTitle && string(input[parsed.tail.titleRange.Start:parsed.tail.titleRange.End]) != title {
		return InlineLinkMapping{}, fmt.Errorf("%w: source title does not match semantic title", ErrUnsupportedInlineLinkShape)
	}
	if issue != inlineTailOK {
		return InlineLinkMapping{}, fmt.Errorf("%w: missing inline-link closing parenthesis", ErrUnsupportedInlineLinkShape)
	}
	return InlineLinkMapping{
		Range:            Range{Start: anchor, End: parsed.tail.end},
		LabelRange:       label,
		DestinationRange: parsed.destinationRange,
		TitleRange:       parsed.tail.titleRange,
		AngleDestination: parsed.angle,
		HasTitle:         hasTitle,
	}, nil
}

func validateInlineLinkPrefix(input []byte, anchor int, label Range, destination string) (int, int, error) {
	if anchor < 0 || anchor >= len(input) || input[anchor] != '[' || !label.Valid(len(input)) || label.Start == label.End {
		return 0, 0, fmt.Errorf("%w: invalid anchor or label range", ErrUnsupportedInlineLinkShape)
	}
	lineEnd, destinationStart, ok := inlineDestinationOpening(input, anchor, 1, label)
	if !ok {
		return 0, 0, fmt.Errorf("%w: label is not followed by an inline-link destination", ErrUnsupportedInlineLinkShape)
	}
	if containsLineBreak(input[anchor:destinationStart]) {
		return 0, 0, fmt.Errorf("%w: link prefix crosses a physical line", ErrUnsupportedInlineLinkShape)
	}
	if destination == "" {
		return 0, 0, fmt.Errorf("%w: unsupported empty destination", ErrUnsupportedInlineLinkShape)
	}
	return lineEnd, destinationStart, nil
}

// ImageMapping binds one simple inline image to its exact source destination.
type ImageMapping struct {
	Range            Range
	AltRange         Range
	DestinationRange Range
	TitleRange       Range
	AngleDestination bool
	HasTitle         bool
}

// MapSimpleImage maps a single-line plain-text-alt inline image to its source destination.
func MapSimpleImage(input []byte, anchor int, alt Range) (ImageMapping, error) {
	if anchor < 0 || anchor+1 >= len(input) || input[anchor] != '!' || input[anchor+1] != '[' || !alt.Valid(len(input)) || alt.Start == alt.End {
		return ImageMapping{}, fmt.Errorf("%w: invalid anchor or alt range", ErrUnsupportedImageShape)
	}
	lineEnd, destinationStart, ok := inlineDestinationOpening(input, anchor, 2, alt)
	if !ok {
		return ImageMapping{}, fmt.Errorf("%w: alt text is not followed by an inline-image destination", ErrUnsupportedImageShape)
	}
	if containsLineBreak(input[anchor:destinationStart]) {
		return ImageMapping{}, fmt.Errorf("%w: image prefix crosses a physical line", ErrUnsupportedImageShape)
	}

	parsed, issue := scanInlineDestinationTail(input, destinationStart, lineEnd, inlineTitleOptionalSeparated)
	switch issue {
	case inlineTailInvalidDestination:
		return ImageMapping{}, fmt.Errorf("%w: unsupported or empty image destination", ErrUnsupportedImageShape)
	case inlineTailUnseparatedTitle:
		return ImageMapping{}, fmt.Errorf("%w: image title is not separated from destination", ErrUnsupportedImageShape)
	case inlineTailInvalidTitle:
		return ImageMapping{}, fmt.Errorf("%w: unsupported image title", ErrUnsupportedImageShape)
	case inlineTailMissingClosing:
		return ImageMapping{}, fmt.Errorf("%w: missing inline-image closing parenthesis", ErrUnsupportedImageShape)
	}
	return ImageMapping{
		Range:            Range{Start: anchor, End: parsed.tail.end},
		AltRange:         alt,
		DestinationRange: parsed.destinationRange,
		TitleRange:       parsed.tail.titleRange,
		AngleDestination: parsed.angle,
		HasTitle:         parsed.tail.hasTitle,
	}, nil
}

// ReferenceDefinitionMapping binds one single-line link reference definition to its exact destination.
type ReferenceDefinitionMapping struct {
	Range            Range
	LineRange        Range
	DestinationRange Range
	TitleRange       Range
	AngleDestination bool
	HasTitle         bool
}

// MapSingleLineReferenceDefinition maps one parser-recognized single-line reference definition to its destination bytes.
func MapSingleLineReferenceDefinition(input []byte, observation Range, label, destination, title string, hasTitle bool) (ReferenceDefinitionMapping, error) {
	if !observation.Valid(len(input)) || observation.Start == observation.End || destination == "" {
		return ReferenceDefinitionMapping{}, fmt.Errorf("%w: invalid observation or empty destination", ErrUnsupportedReferenceDefinitionShape)
	}
	lineStart := physicalLineStart(input, observation.Start)
	lineEnd := physicalLineEnd(input, observation.End)
	if containsLineBreak(input[lineStart:lineEnd]) {
		return ReferenceDefinitionMapping{}, fmt.Errorf("%w: definition crosses a physical line", ErrUnsupportedReferenceDefinitionShape)
	}
	pos, err := referenceDefinitionDestinationStart(input, lineStart, lineEnd, label)
	if err != nil {
		return ReferenceDefinitionMapping{}, err
	}
	destinationRange, angle, next, ok := scanMarkdownLinkDestination(input, pos, lineEnd)
	if !ok || string(input[destinationRange.Start:destinationRange.End]) != destination {
		return ReferenceDefinitionMapping{}, fmt.Errorf("%w: source destination does not match semantic destination", ErrUnsupportedReferenceDefinitionShape)
	}
	titleRange, end, err := referenceDefinitionTitleTail(input, next, lineEnd, title, hasTitle)
	if err != nil {
		return ReferenceDefinitionMapping{}, err
	}
	lineRangeEnd := lineEnd
	if next, ok := nextPhysicalLineStart(input, lineEnd); ok {
		lineRangeEnd = next
	}
	return ReferenceDefinitionMapping{
		Range:            Range{Start: lineStart, End: end},
		LineRange:        Range{Start: lineStart, End: lineRangeEnd},
		DestinationRange: destinationRange,
		TitleRange:       titleRange,
		AngleDestination: angle,
		HasTitle:         hasTitle,
	}, nil
}

func referenceDefinitionDestinationStart(input []byte, lineStart, lineEnd int, label string) (int, error) {
	pos := lineStart
	indent := 0
	for pos < lineEnd && input[pos] == ' ' && indent < 4 {
		pos++
		indent++
	}
	if indent > 3 || pos >= lineEnd || input[pos] != '[' {
		return 0, fmt.Errorf("%w: unsupported definition indentation or label opener", ErrUnsupportedReferenceDefinitionShape)
	}
	labelRange, next, ok := scanBracketContent(input, pos, lineEnd)
	if !ok || string(input[labelRange.Start:labelRange.End]) != label || next >= lineEnd || input[next] != ':' {
		return 0, fmt.Errorf("%w: source label does not match semantic label", ErrUnsupportedReferenceDefinitionShape)
	}
	return skipHorizontalSpace(input, next+1, lineEnd), nil
}

func referenceDefinitionTitleTail(input []byte, start, lineEnd int, title string, hasTitle bool) (Range, int, error) {
	pos := skipHorizontalSpace(input, start, lineEnd)
	if !hasTitle {
		if pos != lineEnd {
			return Range{}, 0, fmt.Errorf("%w: unexpected bytes after definition", ErrUnsupportedReferenceDefinitionShape)
		}
		return Range{}, lineEnd, nil
	}
	if pos == start {
		return Range{}, 0, fmt.Errorf("%w: title is not separated from destination", ErrUnsupportedReferenceDefinitionShape)
	}
	titleRange, next, ok := scanMarkdownLinkTitle(input, pos, lineEnd)
	if !ok || string(input[titleRange.Start:titleRange.End]) != title {
		return Range{}, 0, fmt.Errorf("%w: source title does not match semantic title", ErrUnsupportedReferenceDefinitionShape)
	}
	pos = skipHorizontalSpace(input, next, lineEnd)
	if pos != lineEnd {
		return Range{}, 0, fmt.Errorf("%w: unexpected bytes after definition", ErrUnsupportedReferenceDefinitionShape)
	}
	return titleRange, lineEnd, nil
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
	if containsLineBreak(input[content.Start:content.End]) {
		return AutoLinkMapping{}, fmt.Errorf("%w: content crosses a physical line", ErrUnsupportedAutoLinkShape)
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
		range_, next, ok := scanAngleLinkDestination(input, start, limit)
		return range_, true, next, ok
	}
	range_, next, ok := scanBareLinkDestination(input, start, limit)
	return range_, false, next, ok
}

func scanAngleLinkDestination(input []byte, start, limit int) (Range, int, bool) {
	for i := start + 1; i < limit; i++ {
		if input[i] == '\\' && i+1 < limit && isASCIIPunctuation(input[i+1]) {
			i++
			continue
		}
		if input[i] != '>' {
			continue
		}
		if i == start+1 {
			return Range{}, start, false
		}
		return Range{Start: start + 1, End: i}, i + 1, true
	}
	return Range{}, start, false
}

func scanBareLinkDestination(input []byte, start, limit int) (Range, int, bool) {
	depth := 0
	i := start
	for i < limit {
		b := input[i]
		switch {
		case b == '\\' && i+1 < limit && isASCIIPunctuation(input[i+1]):
			i += 2
		case b == '(':
			depth++
			i++
		case b == ')':
			if depth == 0 {
				if i == start {
					return Range{}, start, false
				}
				return Range{Start: start, End: i}, i, true
			}
			depth--
			i++
		case isHorizontalSpace(b):
			if i == start || depth != 0 {
				return Range{}, start, false
			}
			return Range{Start: start, End: i}, i, true
		default:
			i++
		}
	}
	if i == start || depth != 0 {
		return Range{}, start, false
	}
	return Range{Start: start, End: i}, i, true
}

func inlineDestinationOpening(input []byte, anchor, prefixLength int, content Range) (int, int, bool) {
	lineEnd := physicalLineEnd(input, content.End)
	if content.Start != anchor+prefixLength || content.End >= lineEnd || input[content.End] != ']' || content.End+1 >= lineEnd || input[content.End+1] != '(' {
		return 0, 0, false
	}
	return lineEnd, content.End + 2, true
}

type inlineTitlePolicy uint8

const (
	inlineTitleForbidden inlineTitlePolicy = iota
	inlineTitleRequired
	inlineTitleOptionalSeparated
)

type inlineTailIssue uint8

const (
	inlineTailOK inlineTailIssue = iota
	inlineTailInvalidDestination
	inlineTailUnseparatedTitle
	inlineTailInvalidTitle
	inlineTailMissingClosing
)

type inlineTitleTail struct {
	titleRange Range
	hasTitle   bool
	end        int
}

type inlineDestinationTail struct {
	destinationRange Range
	angle            bool
	tail             inlineTitleTail
}

func scanInlineDestinationTail(input []byte, start, limit int, policy inlineTitlePolicy) (inlineDestinationTail, inlineTailIssue) {
	pos := skipHorizontalSpace(input, start, limit)
	destinationRange, angle, next, ok := scanMarkdownLinkDestination(input, pos, limit)
	if !ok {
		return inlineDestinationTail{}, inlineTailInvalidDestination
	}
	tail, issue := scanInlineTitleTail(input, next, limit, policy)
	return inlineDestinationTail{destinationRange: destinationRange, angle: angle, tail: tail}, issue
}

func scanInlineTitleTail(input []byte, start, limit int, policy inlineTitlePolicy) (inlineTitleTail, inlineTailIssue) {
	spacesStart := start
	pos := skipHorizontalSpace(input, start, limit)
	tail := inlineTitleTail{}

	if policy == inlineTitleOptionalSeparated && pos < limit && input[pos] != ')' {
		if pos == spacesStart {
			return inlineTitleTail{}, inlineTailUnseparatedTitle
		}
		policy = inlineTitleRequired
	}
	if policy == inlineTitleRequired {
		titleRange, next, ok := scanMarkdownLinkTitle(input, pos, limit)
		if !ok {
			return inlineTitleTail{}, inlineTailInvalidTitle
		}
		tail.titleRange = titleRange
		tail.hasTitle = true
		pos = skipHorizontalSpace(input, next, limit)
	}
	if pos >= limit || input[pos] != ')' {
		return tail, inlineTailMissingClosing
	}
	tail.end = pos + 1
	return tail, inlineTailOK
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
