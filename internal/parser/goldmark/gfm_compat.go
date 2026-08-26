package goldmark

import (
	"bytes"
	"regexp"

	"github.com/yuin/goldmark/ast"
	goldmarkparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var gfmProtocolEmailRegexp = regexp.MustCompile(`^[A-Za-z0-9._+\-]+@[A-Za-z0-9_\-]+(?:\.[A-Za-z0-9_\-]+)+`)

// gfmProtocolAutolinkParser supplements Goldmark's Linkify extension with the
// published GFM mailto:/xmpp: protocol-autolink grammar.
type gfmProtocolAutolinkParser struct{}

func (p *gfmProtocolAutolinkParser) Trigger() []byte {
	return []byte{' ', '*', '_', '~', '('}
}

func (p *gfmProtocolAutolinkParser) Parse(parent ast.Node, block text.Reader, pc goldmarkparser.Context) ast.Node {
	if pc.IsInLinkLabel() {
		return nil
	}

	line, segment := block.PeekLine()
	if len(line) == 0 {
		return nil
	}

	prefixLength := 0
	start := segment.Start
	if isGFMExtendedAutolinkBoundary(line[0]) {
		prefixLength = 1
		start++
		line = line[1:]
	}

	scheme, ok := gfmProtocolScheme(line)
	if !ok {
		return nil
	}

	end, valid := gfmProtocolAutolinkEnd(line, scheme)
	if !valid {
		literalLength := gfmProtocolTokenLength(line)
		if prefixLength != 0 {
			ast.MergeOrAppendTextSegment(parent, segment.WithStop(segment.Start+prefixLength))
		}
		block.Advance(prefixLength + literalLength)
		return ast.NewTextSegment(text.NewSegment(start, start+literalLength))
	}

	if prefixLength != 0 {
		ast.MergeOrAppendTextSegment(parent, segment.WithStop(segment.Start+prefixLength))
	}
	block.Advance(prefixLength + end)
	label := ast.NewTextSegment(text.NewSegment(start, start+end))
	return ast.NewAutoLink(ast.AutoLinkURL, label)
}

func isGFMExtendedAutolinkBoundary(c byte) bool {
	return util.IsSpace(c) || c == '*' || c == '_' || c == '~' || c == '('
}

func gfmProtocolScheme(line []byte) (string, bool) {
	switch {
	case bytes.HasPrefix(line, []byte("mailto:")):
		return "mailto", true
	case bytes.HasPrefix(line, []byte("xmpp:")):
		return "xmpp", true
	default:
		return "", false
	}
}

func gfmProtocolAutolinkEnd(line []byte, scheme string) (int, bool) {
	schemeLength := len(scheme) + 1
	if schemeLength >= len(line) {
		return 0, false
	}

	match := gfmProtocolEmailRegexp.FindIndex(line[schemeLength:])
	if match == nil || match[0] != 0 {
		return 0, false
	}
	end := schemeLength + match[1]
	if line[end-1] == '-' || line[end-1] == '_' {
		return 0, false
	}
	if end < len(line) && (line[end] == '-' || line[end] == '_') {
		return 0, false
	}

	if scheme == "xmpp" && end < len(line) && line[end] == '/' {
		resourceStart := end + 1
		resourceEnd := resourceStart
		for resourceEnd < len(line) && isGFMXMPPResourceByte(line[resourceEnd]) {
			resourceEnd++
		}
		if resourceEnd > resourceStart {
			end = resourceEnd
		}
	}
	return end, true
}

func isGFMXMPPResourceByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '@' || c == '.'
}

func gfmProtocolTokenLength(line []byte) int {
	for i, c := range line {
		if util.IsSpace(c) {
			return i
		}
	}
	return len(line)
}
