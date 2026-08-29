// Package renderhtml renders Native semantic events as deterministic HTML fragments.
package renderhtml

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"

	"github.com/zoster81/marksplice/internal/parser"
)

// ErrInvalidInput classifies invalid renderer configuration or semantic input.
var ErrInvalidInput = errors.New("renderhtml: invalid input")

// RawHTMLPolicy controls renderer handling of parser-proven raw HTML.
type RawHTMLPolicy uint8

const (
	RawHTMLPreserve RawHTMLPolicy = iota
	RawHTMLEscape
)

// UnsafeURLPolicy controls renderer handling of dangerous URL schemes.
type UnsafeURLPolicy uint8

const (
	UnsafeURLSuppress UnsafeURLPolicy = iota
	UnsafeURLAllow
)

// TagFilterPolicy controls the GFM disallowed-raw-HTML tag filter.
type TagFilterPolicy uint8

const (
	TagFilterEnabled TagFilterPolicy = iota
	TagFilterDisabled
)

// Options controls deterministic HTML-fragment rendering.
type Options struct {
	RawHTML    RawHTMLPolicy
	UnsafeURLs UnsafeURLPolicy
	TagFilter  TagFilterPolicy
}

type frame struct {
	kind            parser.SemanticKind
	tight           bool
	ordered         bool
	header          bool
	level           int
	suppress        bool
	tableHeadOpen   bool
	tableBodyOpen   bool
	footnoteCapture bool
	tightBlockOpen  bool
	label           string
	anchor          int
}

type imageCapture struct {
	event parser.SemanticEvent
	alt   strings.Builder
	depth int
}

type footnoteDefinition struct {
	label string
	body  []byte
}

type renderer struct {
	writer          io.Writer
	options         Options
	stack           []frame
	image           *imageCapture
	capture         *bytes.Buffer
	definitions     map[int]footnoteDefinition
	footnoteNumbers map[int]int
	footnoteOrder   []int
	footnoteLabels  map[int]string
}

// Render walks source semantics on demand and streams one HTML fragment to writer.
func Render(writer io.Writer, source []byte, backend parser.SemanticBackend, options Options) error {
	if writer == nil || backend == nil {
		return ErrInvalidInput
	}
	if err := validateOptions(options); err != nil {
		return err
	}
	r := &renderer{
		writer:          writer,
		options:         options,
		definitions:     make(map[int]footnoteDefinition),
		footnoteNumbers: make(map[int]int),
		footnoteLabels:  make(map[int]string),
	}
	return backend.WalkSemantic(source, r.visit)
}

func validateOptions(options Options) error {
	if options.RawHTML > RawHTMLEscape || options.UnsafeURLs > UnsafeURLAllow || options.TagFilter > TagFilterDisabled {
		return ErrInvalidInput
	}
	return nil
}

func (r *renderer) visit(event parser.SemanticEvent) error {
	if r.image != nil {
		return r.visitImage(event)
	}
	if event.Kind == parser.SemanticImage && event.Phase == parser.SemanticEnter {
		r.image = &imageCapture{event: event, depth: 1}
		return nil
	}
	switch event.Phase {
	case parser.SemanticEnter:
		return r.enter(event)
	case parser.SemanticLeaf:
		return r.leaf(event)
	case parser.SemanticExit:
		return r.exit(event)
	default:
		return fmt.Errorf("%w: unknown semantic phase %d", ErrInvalidInput, event.Phase)
	}
}

func (r *renderer) visitImage(event parser.SemanticEvent) error {
	if event.Kind == parser.SemanticImage {
		switch event.Phase {
		case parser.SemanticEnter:
			r.image.depth++
			return nil
		case parser.SemanticExit:
			r.image.depth--
			if r.image.depth > 0 {
				return nil
			}
			if r.image.depth < 0 {
				return fmt.Errorf("%w: image capture underflow", ErrInvalidInput)
			}
			image := r.image
			r.image = nil
			return r.writeImage(image.event, image.alt.String())
		}
	}
	switch event.Phase {
	case parser.SemanticLeaf:
		switch event.Kind {
		case parser.SemanticText, parser.SemanticCodeSpan, parser.SemanticAutoLink, parser.SemanticMath:
			r.image.alt.WriteString(event.Value)
		case parser.SemanticSoftBreak, parser.SemanticHardBreak:
			r.image.alt.WriteByte('\n')
		case parser.SemanticRawHTML:
			r.image.alt.WriteString(stripHTMLTags(event.Value))
		}
	}
	return nil
}

func (r *renderer) enter(event parser.SemanticEvent) error {
	switch event.Kind {
	case parser.SemanticDocument, parser.SemanticParagraph, parser.SemanticHeading:
		return r.enterDocumentBlock(event)
	case parser.SemanticEmphasis, parser.SemanticStrong, parser.SemanticStrikethrough, parser.SemanticLink:
		return r.enterInline(event)
	case parser.SemanticBlockquote, parser.SemanticAlert:
		return r.enterQuote(event)
	case parser.SemanticList, parser.SemanticListItem:
		return r.enterList(event)
	case parser.SemanticTable, parser.SemanticTableRow, parser.SemanticTableCell:
		return r.enterTable(event)
	case parser.SemanticFootnoteDefinition:
		return r.enterFootnoteDefinition(event)
	default:
		return fmt.Errorf("%w: unsupported enter kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) enterDocumentBlock(event parser.SemanticEvent) error {
	switch event.Kind {
	case parser.SemanticDocument:
		r.push(frame{kind: event.Kind})
		return nil
	case parser.SemanticParagraph:
		suppress := r.directTightListParagraph()
		r.push(frame{kind: event.Kind, suppress: suppress})
		if suppress {
			return nil
		}
		return r.writeString("<p>")
	case parser.SemanticHeading:
		if err := r.prepareTightListBlock(); err != nil {
			return err
		}
		if event.Level < 1 || event.Level > 6 {
			return fmt.Errorf("%w: heading level %d", ErrInvalidInput, event.Level)
		}
		r.push(frame{kind: event.Kind, level: event.Level})
		return r.writeString("<h" + strconv.Itoa(event.Level) + ">")
	default:
		return fmt.Errorf("%w: unsupported document block enter kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) enterInline(event parser.SemanticEvent) error {
	r.push(frame{kind: event.Kind})
	switch event.Kind {
	case parser.SemanticEmphasis:
		return r.writeString("<em>")
	case parser.SemanticStrong:
		return r.writeString("<strong>")
	case parser.SemanticStrikethrough:
		return r.writeString("<del>")
	case parser.SemanticLink:
		return r.writeLinkOpen(event)
	default:
		return fmt.Errorf("%w: unsupported inline enter kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) enterQuote(event parser.SemanticEvent) error {
	if err := r.prepareTightListBlock(); err != nil {
		return err
	}
	r.push(frame{kind: event.Kind})
	if event.Kind == parser.SemanticBlockquote {
		return r.writeString("<blockquote>\n")
	}
	class, title := alertPresentation(event.AlertKind)
	if err := r.writeString("<div class=\"markdown-alert markdown-alert-" + class + "\">\n"); err != nil {
		return err
	}
	return r.writeString("<p class=\"markdown-alert-title\">" + title + "</p>\n")
}

func (r *renderer) enterList(event parser.SemanticEvent) error {
	switch event.Kind {
	case parser.SemanticList:
		if err := r.prepareTightListBlock(); err != nil {
			return err
		}
		r.push(frame{kind: event.Kind, tight: event.Tight, ordered: event.Ordered})
		if !event.Ordered {
			return r.writeString("<ul>\n")
		}
		if event.Start != 1 {
			return r.writeString("<ol start=\"" + strconv.Itoa(event.Start) + "\">\n")
		}
		return r.writeString("<ol>\n")
	case parser.SemanticListItem:
		loose := r.parentListLoose()
		r.push(frame{kind: event.Kind})
		if event.ContentRange == (parser.Range{}) {
			return r.writeString("<li>")
		}
		if loose {
			return r.writeString("<li>\n")
		}
		return r.writeString("<li>")
	default:
		return fmt.Errorf("%w: unsupported list enter kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) enterTable(event parser.SemanticEvent) error {
	switch event.Kind {
	case parser.SemanticTable:
		if err := r.prepareTightListBlock(); err != nil {
			return err
		}
		r.push(frame{kind: event.Kind})
		return r.writeString("<table>\n")
	case parser.SemanticTableRow:
		if err := r.openTableSection(event.Header); err != nil {
			return err
		}
		r.push(frame{kind: event.Kind})
		return r.writeString("<tr>\n")
	case parser.SemanticTableCell:
		r.push(frame{kind: event.Kind, header: event.Header})
		tag := "td"
		if event.Header {
			tag = "th"
		}
		if event.Alignment == parser.TableAlignmentDefault {
			return r.writeString("<" + tag + ">")
		}
		return r.writeString("<" + tag + " align=\"" + alignmentName(event.Alignment) + "\">")
	default:
		return fmt.Errorf("%w: unsupported table enter kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) enterFootnoteDefinition(event parser.SemanticEvent) error {
	if r.capture != nil {
		return fmt.Errorf("%w: nested footnote definition", ErrInvalidInput)
	}
	r.capture = &bytes.Buffer{}
	r.push(frame{kind: event.Kind, footnoteCapture: true, label: event.Label, anchor: event.Range.Start})
	return nil
}

func (r *renderer) leaf(event parser.SemanticEvent) error {
	switch event.Kind {
	case parser.SemanticText, parser.SemanticSoftBreak, parser.SemanticHardBreak, parser.SemanticCodeSpan, parser.SemanticAutoLink, parser.SemanticRawHTML:
		return r.leafInline(event)
	case parser.SemanticHTMLBlock, parser.SemanticTaskItem, parser.SemanticThematicBreak, parser.SemanticCodeBlock, parser.SemanticMath:
		return r.leafBlock(event)
	case parser.SemanticFootnoteReference:
		return r.writeFootnoteReference(event)
	case parser.SemanticReferenceDefinition, parser.SemanticFrontMatter:
		return nil
	default:
		return fmt.Errorf("%w: unsupported leaf kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) leafInline(event parser.SemanticEvent) error {
	switch event.Kind {
	case parser.SemanticText:
		return r.writeString(escapeText(event.Value))
	case parser.SemanticSoftBreak:
		return r.writeString("\n")
	case parser.SemanticHardBreak:
		return r.writeString("<br />\n")
	case parser.SemanticCodeSpan:
		return r.writeString("<code>" + escapeText(event.Value) + "</code>")
	case parser.SemanticAutoLink:
		return r.writeAnchor(event.Destination, "", false, event.Value, false)
	case parser.SemanticRawHTML:
		return r.writeRawHTML(event.Value)
	default:
		return fmt.Errorf("%w: unsupported inline leaf kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) leafBlock(event parser.SemanticEvent) error {
	switch event.Kind {
	case parser.SemanticHTMLBlock:
		if err := r.prepareTightListBlock(); err != nil {
			return err
		}
		return r.writeRawHTML(event.Value)
	case parser.SemanticTaskItem:
		if event.Checked {
			return r.writeString("<input checked=\"\" disabled=\"\" type=\"checkbox\">")
		}
		return r.writeString("<input disabled=\"\" type=\"checkbox\">")
	case parser.SemanticThematicBreak:
		if err := r.prepareTightListBlock(); err != nil {
			return err
		}
		return r.writeString("<hr />\n")
	case parser.SemanticCodeBlock:
		if err := r.prepareTightListBlock(); err != nil {
			return err
		}
		return r.writeCodeBlock(event)
	case parser.SemanticMath:
		if event.MathStyle == parser.MathExpressionBlockDollar {
			if err := r.prepareTightListBlock(); err != nil {
				return err
			}
		}
		return r.writeMath(event)
	default:
		return fmt.Errorf("%w: unsupported block leaf kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) exit(event parser.SemanticEvent) error {
	current, err := r.pop(event.Kind)
	if err != nil {
		return err
	}
	switch event.Kind {
	case parser.SemanticDocument, parser.SemanticParagraph, parser.SemanticHeading:
		return r.exitDocumentBlock(event, current)
	case parser.SemanticEmphasis, parser.SemanticStrong, parser.SemanticStrikethrough, parser.SemanticLink:
		return r.exitInline(event)
	case parser.SemanticBlockquote, parser.SemanticAlert:
		return r.exitQuote(event)
	case parser.SemanticList, parser.SemanticListItem:
		return r.exitList(event, current)
	case parser.SemanticTable, parser.SemanticTableRow, parser.SemanticTableCell:
		return r.exitTable(event, current)
	case parser.SemanticFootnoteDefinition:
		return r.exitFootnoteDefinition(current)
	default:
		return fmt.Errorf("%w: unsupported exit kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) exitDocumentBlock(event parser.SemanticEvent, current frame) error {
	switch event.Kind {
	case parser.SemanticDocument:
		return r.writeFootnotes()
	case parser.SemanticParagraph:
		if current.suppress {
			return nil
		}
		return r.writeString("</p>\n")
	case parser.SemanticHeading:
		if current.level < 1 || current.level > 6 {
			return fmt.Errorf("%w: heading level %d", ErrInvalidInput, current.level)
		}
		return r.writeString("</h" + strconv.Itoa(current.level) + ">\n")
	default:
		return fmt.Errorf("%w: unsupported document block exit kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) exitInline(event parser.SemanticEvent) error {
	switch event.Kind {
	case parser.SemanticEmphasis:
		return r.writeString("</em>")
	case parser.SemanticStrong:
		return r.writeString("</strong>")
	case parser.SemanticStrikethrough:
		return r.writeString("</del>")
	case parser.SemanticLink:
		return r.writeString("</a>")
	default:
		return fmt.Errorf("%w: unsupported inline exit kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) exitQuote(event parser.SemanticEvent) error {
	if event.Kind == parser.SemanticBlockquote {
		return r.writeString("</blockquote>\n")
	}
	return r.writeString("</div>\n")
}

func (r *renderer) exitList(event parser.SemanticEvent, current frame) error {
	if event.Kind == parser.SemanticListItem {
		return r.writeString("</li>\n")
	}
	if current.ordered {
		return r.writeString("</ol>\n")
	}
	return r.writeString("</ul>\n")
}

func (r *renderer) exitTable(event parser.SemanticEvent, current frame) error {
	switch event.Kind {
	case parser.SemanticTable:
		if current.tableHeadOpen {
			if err := r.writeString("</thead>\n"); err != nil {
				return err
			}
		}
		if current.tableBodyOpen {
			if err := r.writeString("</tbody>\n"); err != nil {
				return err
			}
		}
		return r.writeString("</table>\n")
	case parser.SemanticTableRow:
		return r.writeString("</tr>\n")
	case parser.SemanticTableCell:
		if current.header {
			return r.writeString("</th>\n")
		}
		return r.writeString("</td>\n")
	default:
		return fmt.Errorf("%w: unsupported table exit kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) exitFootnoteDefinition(current frame) error {
	if !current.footnoteCapture || r.capture == nil {
		return fmt.Errorf("%w: missing footnote capture", ErrInvalidInput)
	}
	r.definitions[current.anchor] = footnoteDefinition{label: current.label, body: append([]byte(nil), r.capture.Bytes()...)}
	r.capture = nil
	return nil
}
func (r *renderer) writeLinkOpen(event parser.SemanticEvent) error {
	return r.writeAnchorOpen(event.Destination, event.Title, event.HasTitle, true)
}

func (r *renderer) writeAnchor(destination, title string, hasTitle bool, label string, markdownEscapes bool) error {
	if err := r.writeAnchorOpen(destination, title, hasTitle, markdownEscapes); err != nil {
		return err
	}
	if err := r.writeString(escapeText(label)); err != nil {
		return err
	}
	return r.writeString("</a>")
}

func (r *renderer) writeAnchorOpen(destination, title string, hasTitle, markdownEscapes bool) error {
	destination = r.renderDestination(destination, markdownEscapes)
	if err := r.writeString("<a href=\"" + escapeAttribute(destination) + "\""); err != nil {
		return err
	}
	if hasTitle {
		title = decodeMarkdownString(title)
		if err := r.writeString(" title=\"" + escapeAttribute(title) + "\""); err != nil {
			return err
		}
	}
	return r.writeString(">")
}

func (r *renderer) writeImage(event parser.SemanticEvent, alt string) error {
	destination := r.renderDestination(event.Destination, true)
	if err := r.writeString("<img src=\"" + escapeAttribute(destination) + "\" alt=\"" + escapeAttribute(alt) + "\""); err != nil {
		return err
	}
	if event.HasTitle {
		title := decodeMarkdownString(event.Title)
		if err := r.writeString(" title=\"" + escapeAttribute(title) + "\""); err != nil {
			return err
		}
	}
	return r.writeString(" />")
}

func (r *renderer) writeRawHTML(value string) error {
	if r.options.RawHTML == RawHTMLEscape {
		return r.writeString(escapeText(value))
	}
	if r.options.TagFilter == TagFilterEnabled {
		value = filterDisallowedTags(value)
	}
	return r.writeString(value)
}

func (r *renderer) writeCodeBlock(event parser.SemanticEvent) error {
	if err := r.writeString("<pre><code"); err != nil {
		return err
	}
	language := decodeMarkdownString(event.Language)
	if language != "" {
		if err := r.writeString(" class=\"language-" + escapeAttribute(language) + "\""); err != nil {
			return err
		}
	}
	value := event.Value
	if value != "" && !strings.HasSuffix(value, "\n") {
		value += "\n"
	}
	if err := r.writeString(">" + escapeText(value)); err != nil {
		return err
	}
	return r.writeString("</code></pre>\n")
}

func (r *renderer) writeMath(event parser.SemanticEvent) error {
	switch event.MathStyle {
	case parser.MathExpressionInlineDollar, parser.MathExpressionInlineBacktick:
		return r.writeString("<span class=\"math math-inline\">" + escapeText(event.Value) + "</span>")
	case parser.MathExpressionBlockDollar:
		return r.writeString("<div class=\"math math-block\">" + escapeText(event.Value) + "</div>\n")
	default:
		return fmt.Errorf("%w: unknown math style %d", ErrInvalidInput, event.MathStyle)
	}
}

func (r *renderer) writeFootnoteReference(event parser.SemanticEvent) error {
	number, ok := r.footnoteNumbers[event.DefinitionAnchor]
	if !ok {
		number = len(r.footnoteOrder) + 1
		r.footnoteNumbers[event.DefinitionAnchor] = number
		r.footnoteOrder = append(r.footnoteOrder, event.DefinitionAnchor)
		r.footnoteLabels[event.DefinitionAnchor] = event.Label
	}
	label := footnoteID(event.Label)
	refID := "fnref:" + label
	if event.Occurrence > 0 {
		refID += "-" + strconv.Itoa(event.Occurrence+1)
	}
	return r.writeString("<sup class=\"footnote-ref\"><a href=\"#fn:" + escapeAttribute(label) + "\" id=\"" + escapeAttribute(refID) + "\">" + strconv.Itoa(number) + "</a></sup>")
}

func (r *renderer) writeFootnotes() error {
	if len(r.footnoteOrder) == 0 {
		return nil
	}
	if err := r.writeRootString("<section class=\"footnotes\">\n<ol>\n"); err != nil {
		return err
	}
	for _, anchor := range r.footnoteOrder {
		definition, ok := r.definitions[anchor]
		if !ok {
			continue
		}
		label := footnoteID(definition.label)
		if err := r.writeRootString("<li id=\"fn:" + escapeAttribute(label) + "\">\n"); err != nil {
			return err
		}
		if err := r.writeRootBytes(definition.body); err != nil {
			return err
		}
		if err := r.writeRootString("<a href=\"#fnref:" + escapeAttribute(label) + "\" class=\"footnote-backref\">↩</a>\n</li>\n"); err != nil {
			return err
		}
	}
	return r.writeRootString("</ol>\n</section>\n")
}

func (r *renderer) openTableSection(header bool) error {
	index := r.findFrame(parser.SemanticTable)
	if index < 0 {
		return fmt.Errorf("%w: table row outside table", ErrInvalidInput)
	}
	table := &r.stack[index]
	if header {
		if table.tableBodyOpen {
			return fmt.Errorf("%w: header row after table body", ErrInvalidInput)
		}
		if !table.tableHeadOpen {
			table.tableHeadOpen = true
			return r.writeString("<thead>\n")
		}
		return nil
	}
	if table.tableHeadOpen {
		if err := r.writeString("</thead>\n"); err != nil {
			return err
		}
		table.tableHeadOpen = false
	}
	if !table.tableBodyOpen {
		table.tableBodyOpen = true
		return r.writeString("<tbody>\n")
	}
	return nil
}

func (r *renderer) prepareTightListBlock() error {
	if len(r.stack) == 0 || r.stack[len(r.stack)-1].kind != parser.SemanticListItem {
		return nil
	}
	for index := len(r.stack) - 2; index >= 0; index-- {
		if r.stack[index].kind != parser.SemanticList {
			continue
		}
		if !r.stack[index].tight {
			return nil
		}
		item := &r.stack[len(r.stack)-1]
		if item.tightBlockOpen {
			return nil
		}
		item.tightBlockOpen = true
		return r.writeString("\n")
	}
	return nil
}

func (r *renderer) parentListLoose() bool {
	for index := len(r.stack) - 1; index >= 0; index-- {
		if r.stack[index].kind == parser.SemanticList {
			return !r.stack[index].tight
		}
	}
	return false
}

func (r *renderer) directTightListParagraph() bool {
	if len(r.stack) == 0 || r.stack[len(r.stack)-1].kind != parser.SemanticListItem {
		return false
	}
	for index := len(r.stack) - 2; index >= 0; index-- {
		if r.stack[index].kind == parser.SemanticList {
			return r.stack[index].tight
		}
	}
	return false
}

func (r *renderer) renderDestination(destination string, markdownEscapes bool) string {
	if markdownEscapes {
		destination = decodeMarkdownString(destination)
	}
	if r.options.UnsafeURLs != UnsafeURLAllow && unsafeDestination(destination) {
		return ""
	}
	return percentEncodeURL(destination)
}

func unsafeDestination(destination string) bool {
	trimmed := strings.TrimLeftFunc(destination, func(value rune) bool { return value <= ' ' || value == 0x7f })
	colon := strings.IndexByte(trimmed, ':')
	if colon < 0 {
		return false
	}
	for _, delimiter := range "/?#" {
		if index := strings.IndexRune(trimmed, delimiter); index >= 0 && index < colon {
			return false
		}
	}
	var scheme strings.Builder
	for _, value := range trimmed[:colon] {
		if value <= ' ' || value == 0x7f {
			continue
		}
		scheme.WriteRune(value)
	}
	switch strings.ToLower(scheme.String()) {
	case "javascript", "vbscript", "file", "data":
		return true
	default:
		return false
	}
}

func decodeMarkdownString(value string) string {
	var output strings.Builder
	output.Grow(len(value))
	for position := 0; position < len(value); {
		if value[position] == '\\' && position+1 < len(value) && isASCIIPunctuation(value[position+1]) {
			output.WriteByte(value[position+1])
			position += 2
			continue
		}
		if value[position] == '&' {
			if relative := strings.IndexByte(value[position:], ';'); relative > 1 && relative <= 64 {
				end := position + relative + 1
				raw := value[position:end]
				decoded := html.UnescapeString(raw)
				if decoded != raw {
					output.WriteString(decoded)
					position = end
					continue
				}
			}
		}
		output.WriteByte(value[position])
		position++
	}
	return output.String()
}

func isASCIIPunctuation(value byte) bool {
	return value >= '!' && value <= '/' || value >= ':' && value <= '@' || value >= '[' && value <= '`' || value >= '{' && value <= '~'
}

func percentEncodeURL(value string) string {
	const hex = "0123456789ABCDEF"
	authorityStart, authorityEnd, hasAuthority := urlAuthorityRange(value)
	var output strings.Builder
	output.Grow(len(value))
	for index := 0; index < len(value); index++ {
		current := value[index]
		if urlSafeByte(current, index, authorityStart, authorityEnd, hasAuthority) {
			output.WriteByte(current)
			continue
		}
		output.WriteByte('%')
		output.WriteByte(hex[current>>4])
		output.WriteByte(hex[current&0x0f])
	}
	return output.String()
}

func urlAuthorityRange(value string) (int, int, bool) {
	start := -1
	if strings.HasPrefix(value, "//") {
		start = 2
	} else if delimiter := strings.Index(value, "://"); delimiter >= 0 {
		start = delimiter + 3
	}
	if start < 0 || start > len(value) {
		return 0, 0, false
	}
	end := len(value)
	if relative := strings.IndexAny(value[start:], "/?#"); relative >= 0 {
		end = start + relative
	}
	return start, end, true
}

func urlSafeByte(value byte, index, authorityStart, authorityEnd int, hasAuthority bool) bool {
	if value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' {
		return true
	}
	if value == '[' || value == ']' {
		return hasAuthority && index >= authorityStart && index < authorityEnd
	}
	return strings.ContainsRune("-._~:/?#@!$&'()*+,;=%", rune(value))
}

func filterDisallowedTags(value string) string {
	var output strings.Builder
	output.Grow(len(value))
	for index := 0; index < len(value); {
		if value[index] != '<' {
			output.WriteByte(value[index])
			index++
			continue
		}
		nameStart := index + 1
		if nameStart < len(value) && value[nameStart] == '/' {
			nameStart++
		}
		nameEnd := nameStart
		for nameEnd < len(value) && isASCIILetter(value[nameEnd]) {
			nameEnd++
		}
		if nameEnd > nameStart && isTagBoundary(value, nameEnd) && disallowedTag(value[nameStart:nameEnd]) {
			output.WriteString("&lt;")
			index++
			continue
		}
		output.WriteByte('<')
		index++
	}
	return output.String()
}

func disallowedTag(name string) bool {
	switch strings.ToLower(name) {
	case "title", "textarea", "style", "xmp", "iframe", "noembed", "noframes", "script", "plaintext":
		return true
	default:
		return false
	}
}

func isTagBoundary(value string, index int) bool {
	if index >= len(value) {
		return true
	}
	c := value[index]
	return c == '>' || c == '/' || c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func isASCIILetter(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func escapeText(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	return strings.ReplaceAll(value, "\"", "&quot;")
}

func escapeAttribute(value string) string {
	value = escapeText(value)
	return strings.ReplaceAll(value, "\"", "&quot;")
}

func stripHTMLTags(value string) string {
	var output strings.Builder
	inside := false
	for _, rune_ := range value {
		switch rune_ {
		case '<':
			inside = true
		case '>':
			inside = false
		default:
			if !inside {
				output.WriteRune(rune_)
			}
		}
	}
	return output.String()
}

func alignmentName(alignment parser.TableAlignment) string {
	switch alignment {
	case parser.TableAlignmentLeft:
		return "left"
	case parser.TableAlignmentRight:
		return "right"
	case parser.TableAlignmentCenter:
		return "center"
	default:
		return ""
	}
}

func alertPresentation(kind parser.SemanticAlertKind) (string, string) {
	switch kind {
	case parser.SemanticAlertNote:
		return "note", "Note"
	case parser.SemanticAlertTip:
		return "tip", "Tip"
	case parser.SemanticAlertImportant:
		return "important", "Important"
	case parser.SemanticAlertWarning:
		return "warning", "Warning"
	case parser.SemanticAlertCaution:
		return "caution", "Caution"
	default:
		return "unknown", "Alert"
	}
}

func footnoteID(label string) string {
	return strings.ReplaceAll(label, " ", "-")
}

func (r *renderer) push(value frame) {
	r.stack = append(r.stack, value)
}

func (r *renderer) pop(kind parser.SemanticKind) (frame, error) {
	if len(r.stack) == 0 {
		return frame{}, fmt.Errorf("%w: unbalanced semantic exit %d", ErrInvalidInput, kind)
	}
	last := len(r.stack) - 1
	value := r.stack[last]
	if value.kind != kind {
		return frame{}, fmt.Errorf("%w: semantic exit %d closes %d", ErrInvalidInput, kind, value.kind)
	}
	r.stack = r.stack[:last]
	return value, nil
}

func (r *renderer) findFrame(kind parser.SemanticKind) int {
	for index := len(r.stack) - 1; index >= 0; index-- {
		if r.stack[index].kind == kind {
			return index
		}
	}
	return -1
}

func (r *renderer) writeString(value string) error {
	writer := r.writer
	if r.capture != nil {
		writer = r.capture
	}
	return writeAllString(writer, value)
}

func (r *renderer) writeRootString(value string) error {
	return writeAllString(r.writer, value)
}

func (r *renderer) writeRootBytes(value []byte) error {
	written, err := r.writer.Write(value)
	if err != nil {
		return err
	}
	if written != len(value) {
		return io.ErrShortWrite
	}
	return nil
}

func writeAllString(writer io.Writer, value string) error {
	written, err := io.WriteString(writer, value)
	if err != nil {
		return err
	}
	if written != len(value) {
		return io.ErrShortWrite
	}
	return nil
}
