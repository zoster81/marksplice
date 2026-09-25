package native

import (
	"sort"

	"github.com/zoster81/marksplice/internal/parser"
)

// ExtendedAutolinkOwner describes the exact semantic owner expected for one
// GFM extended autolink in an emitted inline candidate.
type ExtendedAutolinkOwner struct {
	Range parser.Range
	Form  parser.AutoLinkForm
	Value string
	Email bool
}

// ExtendedAutolinkOwnersMatch proves that every expected extended-autolink
// owner is still recognized by Native with the exact emitted byte boundary and
// semantic form. Surrounding candidate bytes remain part of the proof because
// GFM boundary and path-trimming rules are contextual.
func ExtendedAutolinkOwnersMatch(source []byte, expected []ExtendedAutolinkOwner) bool {
	lines := physicalLines(source)
	for _, owner := range expected {
		if !extendedAutolinkOwnerMatches(source, lines, owner) {
			return false
		}
	}
	return true
}

// ExtendedAutolinkSafeTrailingTextPrefix returns the maximal leading text run
// that may remain raw immediately after a non-WWW extended autolink while
// Native continues to recognize the exact same owner.
func ExtendedAutolinkSafeTrailingTextPrefix(form parser.AutoLinkForm, value string, email bool, text string) int {
	switch form {
	case parser.AutoLinkExtendedURL, parser.AutoLinkExtendedEmail, parser.AutoLinkExtendedProtocol:
	default:
		return 0
	}
	prefixEnd := 0
	for prefixEnd < len(text) && !extendedAutolinkTerminator(text[prefixEnd]) &&
		text[prefixEnd] != '\r' && text[prefixEnd] != '\n' {
		prefixEnd++
	}
	if prefixEnd == 0 {
		return 0
	}
	candidate := make([]byte, 0, len(value)+prefixEnd)
	candidate = append(candidate, value...)
	candidate = append(candidate, text[:prefixEnd]...)
	node, end, ok := scanExtendedAutolink(candidate, 0, 0, len(candidate))
	if !ok || end != len(value) || node.Range != (parser.Range{Start: 0, End: len(value)}) ||
		node.AutoLinkForm != form || node.AutoLinkEmail != email || node.Value != value {
		return 0
	}
	return prefixEnd
}

func extendedAutolinkOwnerMatches(source []byte, lines []physicalLine, owner ExtendedAutolinkOwner) bool {
	if !owner.Range.Valid(len(source)) || owner.Range.Start >= owner.Range.End {
		return false
	}
	switch owner.Form {
	case parser.AutoLinkExtendedWWW,
		parser.AutoLinkExtendedURL,
		parser.AutoLinkExtendedEmail,
		parser.AutoLinkExtendedProtocol:
	default:
		return false
	}
	lineIndex := sort.Search(len(lines), func(index int) bool {
		return lines[index].next > owner.Range.Start
	})
	if lineIndex >= len(lines) {
		return false
	}
	line := lines[lineIndex]
	if owner.Range.Start < line.start || owner.Range.End > line.end {
		return false
	}
	node, end, ok := scanExtendedAutolink(
		source,
		owner.Range.Start,
		line.start,
		line.end,
	)
	return ok &&
		end == owner.Range.End &&
		node.Range == owner.Range &&
		node.AutoLinkForm == owner.Form &&
		node.AutoLinkEmail == owner.Email &&
		node.Value == owner.Value
}
