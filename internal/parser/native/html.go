package native

import (
	"bytes"
	"strings"

	"github.com/zoster81/marksplice/internal/parser"
)

type htmlBlockKind uint8

const (
	htmlBlockNone htmlBlockKind = iota
	htmlBlockRawTag
	htmlBlockComment
	htmlBlockProcessingInstruction
	htmlBlockDeclaration
	htmlBlockCDATA
	htmlBlockNamedTag
	htmlBlockCompleteTag
)

type htmlBlockOpening struct {
	kind   htmlBlockKind
	anchor int
}

func htmlBlockStart(source []byte, line physicalLine) (htmlBlockOpening, bool) {
	indent, ok := ordinaryIndent(source, line)
	if !ok {
		return htmlBlockOpening{}, false
	}
	anchor := line.start + indent
	if anchor >= line.end || source[anchor] != '<' {
		return htmlBlockOpening{}, false
	}
	text := source[anchor:line.end]
	switch {
	case htmlType1Start(text):
		return htmlBlockOpening{kind: htmlBlockRawTag, anchor: anchor}, true
	case bytes.HasPrefix(text, []byte("<!--")):
		return htmlBlockOpening{kind: htmlBlockComment, anchor: anchor}, true
	case bytes.HasPrefix(text, []byte("<?")):
		return htmlBlockOpening{kind: htmlBlockProcessingInstruction, anchor: anchor}, true
	case bytes.HasPrefix(text, []byte("<![CDATA[")):
		return htmlBlockOpening{kind: htmlBlockCDATA, anchor: anchor}, true
	case len(text) >= 3 && text[0] == '<' && text[1] == '!' && asciiUpper(text[2]):
		return htmlBlockOpening{kind: htmlBlockDeclaration, anchor: anchor}, true
	case htmlType6Start(text):
		return htmlBlockOpening{kind: htmlBlockNamedTag, anchor: anchor}, true
	case completeHTMLTag(text):
		return htmlBlockOpening{kind: htmlBlockCompleteTag, anchor: anchor}, true
	default:
		return htmlBlockOpening{}, false
	}
}

func htmlBlockInterruptsParagraph(source []byte, line physicalLine) bool {
	opening, ok := htmlBlockStart(source, line)
	return ok && opening.kind != htmlBlockCompleteTag
}

func parseHTMLBlock(source []byte, lines []physicalLine, index int, opening htmlBlockOpening) (parser.Node, []parser.Range, int) {
	semantic := make([]parser.Range, 0)
	ownedStart := lines[index].start
	end := ownedStart
	next := index
	for next < len(lines) {
		line := lines[next]
		if htmlBlockBlankTerminated(opening.kind) && blankLine(source, line) {
			break
		}
		start := line.start
		if start < line.end {
			semantic = append(semantic, parser.Range{Start: start, End: line.end})
		}
		end = line.next
		next++
		if htmlBlockEnds(source[line.start:line.end], opening.kind) {
			break
		}
	}
	semantic = finalizeHTMLSemanticRanges(source, lines, index, next, semantic)
	return parser.Node{Kind: parser.KindHTMLBlock, Range: parser.Range{Start: ownedStart, End: end}}, semantic, next
}

func finalizeHTMLSemanticRanges(source []byte, lines []physicalLine, first, next int, ranges []parser.Range) []parser.Range {
	if len(ranges) == 0 || first >= next || lines[first].start == lines[first].physicalStart {
		return ranges
	}
	for index := 0; index+1 < len(ranges); index++ {
		ranges[index].End = ranges[index+1].End
	}
	lastLine := lines[next-1]
	if !blankLine(source, lastLine) {
		ranges[len(ranges)-1].End = lastLine.next
	}
	return ranges
}

func htmlBlockBlankTerminated(kind htmlBlockKind) bool {
	return kind == htmlBlockNamedTag || kind == htmlBlockCompleteTag
}

func htmlBlockEnds(line []byte, kind htmlBlockKind) bool {
	switch kind {
	case htmlBlockRawTag:
		return containsASCIIFold(line, "</script>") || containsASCIIFold(line, "</pre>") || containsASCIIFold(line, "</style>")
	case htmlBlockComment:
		return bytes.Contains(line, []byte("-->"))
	case htmlBlockProcessingInstruction:
		return bytes.Contains(line, []byte("?>"))
	case htmlBlockDeclaration:
		return bytes.ContainsRune(line, '>')
	case htmlBlockCDATA:
		return bytes.Contains(line, []byte("]]>"))
	default:
		return false
	}
}

func htmlType1Start(text []byte) bool {
	for _, name := range []string{"script", "pre", "style"} {
		prefix := "<" + name
		if !hasASCIIFoldPrefix(text, prefix) {
			continue
		}
		if len(text) == len(prefix) || htmlType1Boundary(text[len(prefix):]) {
			return true
		}
	}
	return false
}

func htmlType6Start(text []byte) bool {
	name, rest, ok := htmlTagNameAfterOpen(text)
	return ok && htmlBlockTagName(name) && (len(rest) == 0 || htmlStartBoundary(rest))
}

func htmlTagNameAfterOpen(text []byte) ([]byte, []byte, bool) {
	if len(text) < 2 || text[0] != '<' {
		return nil, nil, false
	}
	position := 1
	if text[position] == '/' {
		position++
		if position >= len(text) {
			return nil, nil, false
		}
	}
	start := position
	if !asciiLetter(text[position]) {
		return nil, nil, false
	}
	position++
	for position < len(text) && (asciiLetter(text[position]) || asciiDigit(text[position]) || text[position] == '-') {
		position++
	}
	return text[start:position], text[position:], true
}

func htmlType1Boundary(rest []byte) bool {
	return len(rest) == 0 || htmlWhitespace(rest[0]) || rest[0] == '>'
}

func htmlStartBoundary(rest []byte) bool {
	if htmlType1Boundary(rest) {
		return true
	}
	return len(rest) >= 2 && rest[0] == '/' && rest[1] == '>'
}

func htmlBlockTagName(name []byte) bool {
	switch strings.ToLower(string(name)) {
	case "address", "article", "aside", "base", "basefont", "blockquote", "body",
		"caption", "center", "col", "colgroup", "dd", "details", "dialog", "dir", "div", "dl", "dt",
		"fieldset", "figcaption", "figure", "footer", "form", "frame", "frameset", "h1", "h2", "h3", "h4", "h5", "h6",
		"head", "header", "hr", "html", "iframe", "legend", "li", "link", "main", "menu", "menuitem", "nav", "noframes",
		"ol", "optgroup", "option", "p", "param", "section", "source", "summary", "table", "tbody", "td", "tfoot", "th",
		"thead", "title", "tr", "track", "ul":
		return true
	default:
		return false
	}
}

func completeHTMLTag(text []byte) bool {
	name, position, closing, ok := scanCompleteHTMLTagName(text)
	if !ok || !closing && htmlRawTagName(name) {
		return false
	}
	if closing {
		position = skipHTMLWhitespace(text, position)
		return position < len(text) && text[position] == '>' && onlyHTMLWhitespace(text[position+1:])
	}
	return scanCompleteHTMLOpenTagTail(text, position)
}

func scanCompleteHTMLTagName(text []byte) ([]byte, int, bool, bool) {
	if len(text) < 3 || text[0] != '<' {
		return nil, 0, false, false
	}
	position := 1
	closing := text[position] == '/'
	if closing {
		position++
	}
	nameStart := position
	if position >= len(text) || !asciiLetter(text[position]) {
		return nil, 0, false, false
	}
	position++
	for position < len(text) && (asciiLetter(text[position]) || asciiDigit(text[position]) || text[position] == '-') {
		position++
	}
	return text[nameStart:position], position, closing, true
}

func htmlRawTagName(name []byte) bool {
	return asciiFoldEqual(name, "script") || asciiFoldEqual(name, "style") || asciiFoldEqual(name, "pre")
}

func scanCompleteHTMLOpenTagTail(text []byte, position int) bool {
	for position < len(text) {
		if text[position] == '>' {
			return onlyHTMLWhitespace(text[position+1:])
		}
		if text[position] == '/' && position+1 < len(text) && text[position+1] == '>' {
			return onlyHTMLWhitespace(text[position+2:])
		}
		if !htmlWhitespace(text[position]) {
			return false
		}
		position = skipHTMLWhitespace(text, position)
		if position >= len(text) {
			return false
		}
		next, ok := scanHTMLAttribute(text, position)
		if ok {
			position = next
		}
	}
	return false
}

func scanHTMLAttribute(text []byte, position int) (int, bool) {
	if position >= len(text) || !htmlAttributeNameStart(text[position]) {
		return position, false
	}
	position++
	for position < len(text) && htmlAttributeNameContinue(text[position]) {
		position++
	}
	nameEnd := position
	valueStart := skipHTMLWhitespace(text, position)
	if valueStart >= len(text) || text[valueStart] != '=' {
		return nameEnd, true
	}
	position = skipHTMLWhitespace(text, valueStart+1)
	if position >= len(text) {
		return position, false
	}
	return scanHTMLAttributeValue(text, position)
}

func scanHTMLAttributeValue(text []byte, position int) (int, bool) {
	if text[position] == '\'' || text[position] == '"' {
		quote := text[position]
		position++
		for position < len(text) && text[position] != quote {
			position++
		}
		if position >= len(text) {
			return position, false
		}
		return position + 1, true
	}
	start := position
	for position < len(text) && htmlUnquotedAttributeValueByte(text[position]) {
		position++
	}
	return position, position > start
}

func htmlUnquotedAttributeValueByte(value byte) bool {
	if htmlWhitespace(value) {
		return false
	}
	switch value {
	case '"', '\'', '=', '<', '>', '`':
		return false
	default:
		return true
	}
}

func skipHTMLWhitespace(text []byte, position int) int {
	for position < len(text) && htmlWhitespace(text[position]) {
		position++
	}
	return position
}

func onlyHTMLWhitespace(text []byte) bool {
	return skipHTMLWhitespace(text, 0) == len(text)
}

func htmlWhitespace(value byte) bool {
	return value == ' ' || value == '\t'
}

func htmlAttributeNameStart(value byte) bool {
	return asciiLetter(value) || value == '_' || value == ':'
}

func htmlAttributeNameContinue(value byte) bool {
	return htmlAttributeNameStart(value) || asciiDigit(value) || value == '.' || value == '-'
}

func asciiLetter(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func asciiUpper(value byte) bool {
	return value >= 'A' && value <= 'Z'
}

func asciiDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func hasASCIIFoldPrefix(text []byte, prefix string) bool {
	return len(text) >= len(prefix) && asciiFoldEqual(text[:len(prefix)], prefix)
}

func containsASCIIFold(text []byte, needle string) bool {
	if len(text) < len(needle) {
		return false
	}
	for position := 0; position <= len(text)-len(needle); position++ {
		if asciiFoldEqual(text[position:position+len(needle)], needle) {
			return true
		}
	}
	return false
}

func asciiFoldEqual(text []byte, value string) bool {
	if len(text) != len(value) {
		return false
	}
	for index, current := range text {
		if asciiFold(current) != asciiFold(value[index]) {
			return false
		}
	}
	return true
}

func asciiFold(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + ('a' - 'A')
	}
	return value
}
