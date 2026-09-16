// Package rendermarkdown renders Native semantic events as deterministic canonical Markdown.
package rendermarkdown

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/zoster81/marksplice/internal/parser"
)

// ErrInvalidInput classifies invalid renderer input or inconsistent semantic events.
var ErrInvalidInput = errors.New("rendermarkdown: invalid input")

// Backend is the semantic authority required by canonical Markdown rendering.
// Reference-label normalization remains parser-owned so synthesized relationship
// definitions use the exact same key semantics as the production parser.
type Backend interface {
	parser.SemanticBackend
	ReferenceLabelKey(label string) string
}

type frame struct {
	event            parser.SemanticEvent
	inline           []byte
	blocks           []string
	items            []listItem
	rows             []tableRow
	cells            []tableCell
	task             bool
	checked          bool
	delimiter        string
	bareAutoLinkTail bool
	lastList         listBoundary
}

type listItem struct {
	blocks  []string
	task    bool
	checked bool
}

type tableRow struct {
	header bool
	cells  []tableCell
}

type tableCell struct {
	value     string
	column    int
	alignment parser.TableAlignment
}

type listBoundary struct {
	valid     bool
	ordered   bool
	delimiter byte
}

type referenceTarget struct {
	label       string
	destination string
	title       string
	hasTitle    bool
}

type topLevelBlock struct {
	value            string
	range_           parser.Range
	sourceBacked     bool
	replaceContained bool
	exactEOF         bool
}

type renderer struct {
	writer         io.Writer
	source         []byte
	backend        Backend
	stack          []frame
	wroteBlock     bool
	documentOpen   bool
	topList        listBoundary
	references     map[string]referenceTarget
	referenceOrder []string
	emittedRefs    map[string]struct{}
	topBlocks      []topLevelBlock
	ownedTopRanges []parser.Range
}

// Render streams deterministic canonical Markdown from one immutable source snapshot.
func Render(writer io.Writer, source []byte, backend Backend) error {
	if writer == nil || backend == nil {
		return ErrInvalidInput
	}
	r := &renderer{
		writer:      writer,
		source:      source,
		backend:     backend,
		references:  make(map[string]referenceTarget),
		emittedRefs: make(map[string]struct{}),
	}
	return backend.WalkSemantic(source, r.visit)
}

func (r *renderer) visit(event parser.SemanticEvent) error {
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

func (r *renderer) enter(event parser.SemanticEvent) error {
	if (event.Kind == parser.SemanticLink || event.Kind == parser.SemanticImage) && event.Label != "" &&
		!referenceLabelUsesFootnoteSyntax(event.Label) {
		if err := r.registerReference(event); err != nil {
			return err
		}
		label, err := r.canonicalReferenceLabel(event.Label)
		if err != nil {
			return err
		}
		event.Label = label
	}
	current := frame{event: event}
	switch event.Kind {
	case parser.SemanticDocument:
		if r.documentOpen || len(r.stack) != 0 {
			return fmt.Errorf("%w: nested document", ErrInvalidInput)
		}
		r.documentOpen = true
	case parser.SemanticParagraph, parser.SemanticHeading, parser.SemanticLink, parser.SemanticImage,
		parser.SemanticBlockquote, parser.SemanticAlert, parser.SemanticList, parser.SemanticListItem,
		parser.SemanticTable, parser.SemanticTableRow, parser.SemanticTableCell, parser.SemanticFootnoteDefinition:
	case parser.SemanticEmphasis, parser.SemanticStrong:
		current.delimiter = r.emphasisDelimiter(event.Kind)
	case parser.SemanticStrikethrough:
		current.delimiter = r.strikethroughDelimiter()
	default:
		return fmt.Errorf("%w: unsupported enter kind %d", ErrInvalidInput, event.Kind)
	}
	r.stack = append(r.stack, current)
	return nil
}

func (r *renderer) leaf(event parser.SemanticEvent) error {
	switch event.Kind {
	case parser.SemanticText, parser.SemanticSoftBreak, parser.SemanticHardBreak, parser.SemanticCodeSpan,
		parser.SemanticAutoLink, parser.SemanticRawHTML, parser.SemanticFootnoteReference:
		return r.leafInline(event)
	case parser.SemanticThematicBreak, parser.SemanticCodeBlock, parser.SemanticHTMLBlock,
		parser.SemanticReferenceDefinition, parser.SemanticFrontMatter, parser.SemanticMath:
		return r.leafBlock(event)
	case parser.SemanticTaskItem:
		return r.recordTask(event.Checked)
	default:
		return fmt.Errorf("%w: unsupported leaf kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) leafInline(event parser.SemanticEvent) error {
	switch event.Kind {
	case parser.SemanticText:
		return r.appendText(event.Value)
	case parser.SemanticSoftBreak:
		return r.appendInline("\n")
	case parser.SemanticHardBreak:
		return r.appendInline("\\\n")
	case parser.SemanticCodeSpan:
		return r.appendInline(renderCodeSpan(event.Value, r.inTableCell()))
	case parser.SemanticAutoLink:
		value, bare := r.renderAutoLink(event)
		if err := r.appendInline(value); err != nil {
			return err
		}
		if bare {
			return r.markBareAutoLinkTail()
		}
		return nil
	case parser.SemanticRawHTML:
		return r.appendRawHTML(event.Value)
	case parser.SemanticFootnoteReference:
		return r.appendInline("[^" + event.Label + "]")
	default:
		return fmt.Errorf("%w: unsupported inline leaf kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) leafBlock(event parser.SemanticEvent) error {
	switch event.Kind {
	case parser.SemanticThematicBreak:
		if r.currentContainerKind() == parser.SemanticListItem {
			return r.appendBlock("* * *\n", event.Range)
		}
		return r.appendBlock("---\n", event.Range)
	case parser.SemanticCodeBlock:
		if r.terminalTopLevelCodeBlock(event) {
			block, err := renderTerminalCodeBlock(event)
			if err != nil {
				return err
			}
			return r.appendTerminalTopLevelBlock(block, event.Range)
		}
		block, err := renderCodeBlock(event)
		if err != nil {
			return err
		}
		return r.appendBlock(block, event.Range)
	case parser.SemanticHTMLBlock:
		return r.appendBlock(canonicalOpaqueBlock(event.Value), event.Range)
	case parser.SemanticReferenceDefinition:
		r.markReferenceDefinition(event.Label)
		label, err := r.canonicalReferenceLabel(event.Label)
		if err != nil {
			return err
		}
		event.Label = label
		return r.appendBlock(renderReferenceDefinition(event), event.Range)
	case parser.SemanticFrontMatter:
		return r.appendBlock(canonicalOpaqueBlock(event.Value), event.Range)
	case parser.SemanticMath:
		value, block, err := renderMath(event)
		if err != nil {
			return err
		}
		if block {
			return r.appendBlock(value, event.Range)
		}
		return r.appendInline(value)
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
		return r.exitDocument(current)
	case parser.SemanticEmphasis, parser.SemanticStrong, parser.SemanticStrikethrough, parser.SemanticLink, parser.SemanticImage:
		return r.exitInline(current)
	case parser.SemanticBlockquote, parser.SemanticAlert, parser.SemanticListItem, parser.SemanticList, parser.SemanticFootnoteDefinition:
		return r.exitBlockContainer(current)
	case parser.SemanticTableCell, parser.SemanticTableRow, parser.SemanticTable:
		return r.exitTable(current)
	default:
		return fmt.Errorf("%w: unsupported exit kind %d", ErrInvalidInput, event.Kind)
	}
}

func (r *renderer) exitDocument(current frame) error {
	switch current.event.Kind {
	case parser.SemanticDocument:
		if len(r.stack) != 0 {
			return fmt.Errorf("%w: document closed with active frames", ErrInvalidInput)
		}
		if err := r.writeMissingReferenceDefinitions(); err != nil {
			return err
		}
		if err := r.flushTopLevelBlocks(); err != nil {
			return err
		}
		r.documentOpen = false
		return nil
	case parser.SemanticParagraph:
		return r.appendBlock(string(current.inline)+"\n", current.event.Range)
	case parser.SemanticHeading:
		return r.appendBlock(renderHeading(current.event.Level, string(current.inline)), current.event.Range)
	default:
		return fmt.Errorf("%w: unsupported document exit kind %d", ErrInvalidInput, current.event.Kind)
	}
}

func (r *renderer) exitInline(current frame) error {
	switch current.event.Kind {
	case parser.SemanticEmphasis, parser.SemanticStrong, parser.SemanticStrikethrough:
		return r.appendInline(current.delimiter + string(current.inline) + current.delimiter)
	case parser.SemanticLink:
		return r.appendInline(r.renderLinkLike(current.event, string(current.inline), false))
	case parser.SemanticImage:
		return r.appendInline(r.renderLinkLike(current.event, string(current.inline), true))
	default:
		return fmt.Errorf("%w: unsupported inline exit kind %d", ErrInvalidInput, current.event.Kind)
	}
}

func (r *renderer) exitBlockContainer(current frame) error {
	switch current.event.Kind {
	case parser.SemanticBlockquote:
		return r.appendBlock(renderQuotedBlocks(current.blocks, ""), current.event.Range)
	case parser.SemanticAlert:
		marker, ok := alertMarker(current.event.AlertKind)
		if !ok {
			return fmt.Errorf("%w: alert kind %d", ErrInvalidInput, current.event.AlertKind)
		}
		return r.appendBlock(renderQuotedBlocks(current.blocks, marker), current.event.Range)
	case parser.SemanticListItem:
		return r.appendListItem(listItem{blocks: current.blocks, task: current.task, checked: current.checked})
	case parser.SemanticList:
		return r.appendList(current.event, current.items)
	case parser.SemanticFootnoteDefinition:
		return r.appendFootnoteDefinition(current.event, current.blocks)
	default:
		return fmt.Errorf("%w: unsupported block-container exit kind %d", ErrInvalidInput, current.event.Kind)
	}
}

func (r *renderer) exitTable(current frame) error {
	switch current.event.Kind {
	case parser.SemanticTableCell:
		value := string(current.inline)
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("%w: multiline table cell", ErrInvalidInput)
		}
		return r.appendTableCell(tableCell{value: value, column: current.event.Column, alignment: current.event.Alignment})
	case parser.SemanticTableRow:
		return r.appendTableRow(tableRow{header: current.event.Header, cells: current.cells})
	case parser.SemanticTable:
		block, err := renderTable(current.event, current.rows)
		if err != nil {
			return err
		}
		return r.appendBlock(block, current.event.Range)
	default:
		return fmt.Errorf("%w: unsupported table exit kind %d", ErrInvalidInput, current.event.Kind)
	}
}

func (r *renderer) appendText(value string) error {
	if len(r.stack) == 0 {
		return fmt.Errorf("%w: text output outside container", ErrInvalidInput)
	}
	current := &r.stack[len(r.stack)-1]
	escaped := escapeText(value)
	if inlineAtLineStart(current.inline) {
		escaped = escapeLeadingTextSpaces(value)
	}
	if current.bareAutoLinkTail {
		escaped = escapeTextAfterBareAutoLink(value)
	}
	return r.appendInline(escaped)
}

func inlineAtLineStart(inline []byte) bool {
	return len(inline) == 0 || inline[len(inline)-1] == '\n'
}

func escapeLeadingTextSpaces(value string) string {
	spaces := 0
	for spaces < len(value) && value[spaces] == ' ' {
		spaces++
	}
	if spaces == 0 {
		return escapeText(value)
	}
	return strings.Repeat("&#32;", spaces) + escapeText(value[spaces:])
}

func (r *renderer) appendRawHTML(value string) error {
	if len(r.stack) == 0 {
		return fmt.Errorf("%w: raw HTML output outside container", ErrInvalidInput)
	}
	current := &r.stack[len(r.stack)-1]
	prefix := ""
	if len(current.inline) != 0 && inlineAtLineStart(current.inline) {
		prefix = "    "
	}
	return r.appendInline(prefix + normalizeLineEndings(value))
}

func (r *renderer) markBareAutoLinkTail() error {
	if len(r.stack) == 0 {
		return fmt.Errorf("%w: autolink output outside container", ErrInvalidInput)
	}
	current := &r.stack[len(r.stack)-1]
	switch current.event.Kind {
	case parser.SemanticParagraph, parser.SemanticHeading, parser.SemanticEmphasis, parser.SemanticStrong,
		parser.SemanticStrikethrough, parser.SemanticLink, parser.SemanticImage, parser.SemanticTableCell:
		current.bareAutoLinkTail = true
		return nil
	default:
		return fmt.Errorf("%w: autolink output inside kind %d", ErrInvalidInput, current.event.Kind)
	}
}

func (r *renderer) appendInline(value string) error {
	if len(r.stack) == 0 {
		return fmt.Errorf("%w: inline output outside container", ErrInvalidInput)
	}
	current := &r.stack[len(r.stack)-1]
	switch current.event.Kind {
	case parser.SemanticParagraph, parser.SemanticHeading, parser.SemanticEmphasis, parser.SemanticStrong,
		parser.SemanticStrikethrough, parser.SemanticLink, parser.SemanticImage, parser.SemanticTableCell:
		current.inline = append(current.inline, value...)
		current.bareAutoLinkTail = false
		return nil
	default:
		return fmt.Errorf("%w: inline output inside kind %d", ErrInvalidInput, current.event.Kind)
	}
}

func (r *renderer) appendBlock(value string, range_ parser.Range) error {
	value = ensureBlock(value)
	if len(r.stack) == 0 {
		return fmt.Errorf("%w: block output outside document", ErrInvalidInput)
	}
	current := &r.stack[len(r.stack)-1]
	switch current.event.Kind {
	case parser.SemanticDocument:
		r.topList = listBoundary{}
		r.bufferTopLevelBlock(value, range_, true, false)
		return nil
	case parser.SemanticListItem, parser.SemanticBlockquote, parser.SemanticAlert, parser.SemanticFootnoteDefinition:
		current.lastList = listBoundary{}
		current.blocks = append(current.blocks, value)
		return nil
	default:
		return fmt.Errorf("%w: block output inside kind %d", ErrInvalidInput, current.event.Kind)
	}
}

func (r *renderer) terminalTopLevelCodeBlock(event parser.SemanticEvent) bool {
	value := normalizeLineEndings(event.Value)
	return r.currentContainerKind() == parser.SemanticDocument && event.Range.Valid(len(r.source)) &&
		event.Range.End == len(r.source) && value != "" && !strings.HasSuffix(value, "\n")
}

func (r *renderer) appendTerminalTopLevelBlock(value string, range_ parser.Range) error {
	if len(r.stack) == 0 || r.stack[len(r.stack)-1].event.Kind != parser.SemanticDocument {
		return fmt.Errorf("%w: terminal block outside document", ErrInvalidInput)
	}
	r.topList = listBoundary{}
	r.bufferTopLevelBlockMode(value, range_, true, false, true)
	return nil
}

func (r *renderer) appendFootnoteDefinition(event parser.SemanticEvent, blocks []string) error {
	value := ensureBlock(renderFootnoteDefinition(event.Label, blocks))
	if len(r.stack) == 0 {
		return fmt.Errorf("%w: footnote output outside document", ErrInvalidInput)
	}
	current := &r.stack[len(r.stack)-1]
	if current.event.Kind == parser.SemanticDocument {
		r.topList = listBoundary{}
		r.bufferTopLevelBlock(value, event.Range, true, true)
		return nil
	}
	return r.appendBlock(value, event.Range)
}

func (r *renderer) bufferTopLevelBlock(value string, range_ parser.Range, sourceBacked, replaceContained bool) {
	r.bufferTopLevelBlockMode(value, range_, sourceBacked, replaceContained, false)
}

func (r *renderer) bufferTopLevelBlockMode(value string, range_ parser.Range, sourceBacked, replaceContained, exactEOF bool) {
	if replaceContained && r.hasSourceRange(range_, sourceBacked) {
		r.ownedTopRanges = append(r.ownedTopRanges, range_)
	}
	r.topBlocks = append(r.topBlocks, topLevelBlock{
		value:            value,
		range_:           range_,
		sourceBacked:     sourceBacked,
		replaceContained: replaceContained,
		exactEOF:         exactEOF,
	})
}

func (r *renderer) hasSourceRange(range_ parser.Range, sourceBacked bool) bool {
	return sourceBacked && range_.Valid(len(r.source))
}

func normalizeOwnedTopLevelRanges(ranges []parser.Range) []parser.Range {
	if len(ranges) < 2 {
		return ranges
	}
	sort.Slice(ranges, func(leftIndex, rightIndex int) bool {
		return ranges[leftIndex].Start < ranges[rightIndex].Start
	})
	result := ranges[:1]
	for _, current := range ranges[1:] {
		last := &result[len(result)-1]
		if current.Start <= last.End {
			if current.End > last.End {
				last.End = current.End
			}
			continue
		}
		result = append(result, current)
	}
	return result
}

func (r *renderer) flushTopLevelBlocks() error {
	sort.SliceStable(r.topBlocks, func(leftIndex, rightIndex int) bool {
		left, right := r.topBlocks[leftIndex], r.topBlocks[rightIndex]
		if left.exactEOF != right.exactEOF {
			return !left.exactEOF
		}
		leftBacked := r.hasSourceRange(left.range_, left.sourceBacked)
		rightBacked := r.hasSourceRange(right.range_, right.sourceBacked)
		if leftBacked != rightBacked {
			return leftBacked
		}
		return leftBacked && left.range_.Start < right.range_.Start
	})
	owned := normalizeOwnedTopLevelRanges(r.ownedTopRanges)
	ownedIndex := 0
	for index, block := range r.topBlocks {
		if block.exactEOF && index != len(r.topBlocks)-1 {
			return fmt.Errorf("%w: terminal EOF block is not last", ErrInvalidInput)
		}
		if r.hasSourceRange(block.range_, block.sourceBacked) {
			for ownedIndex < len(owned) && owned[ownedIndex].End <= block.range_.Start {
				ownedIndex++
			}
			if !block.replaceContained && ownedIndex < len(owned) && owned[ownedIndex].Start <= block.range_.Start {
				continue
			}
		}
		if err := r.writeTopLevelBlock(block.value); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) appendList(event parser.SemanticEvent, items []listItem) error {
	if len(r.stack) == 0 {
		return fmt.Errorf("%w: list output outside container", ErrInvalidInput)
	}
	boundary, err := r.listBoundaryForCurrentContainer()
	if err != nil {
		return err
	}
	event.Marker = canonicalListMarker(event.Ordered, *boundary)
	value, err := renderList(event, items)
	if err != nil {
		return err
	}
	value = ensureBlock(value)
	current := &r.stack[len(r.stack)-1]
	switch current.event.Kind {
	case parser.SemanticDocument:
		r.bufferTopLevelBlock(value, event.Range, true, false)
	case parser.SemanticListItem:
		current.blocks = append(current.blocks, indentBlock(value, 2))
	case parser.SemanticBlockquote, parser.SemanticAlert, parser.SemanticFootnoteDefinition:
		current.blocks = append(current.blocks, value)
	default:
		return fmt.Errorf("%w: list output inside kind %d", ErrInvalidInput, current.event.Kind)
	}
	*boundary = listBoundary{valid: true, ordered: event.Ordered, delimiter: event.Marker}
	return nil
}

func (r *renderer) listBoundaryForCurrentContainer() (*listBoundary, error) {
	current := &r.stack[len(r.stack)-1]
	if current.event.Kind == parser.SemanticDocument {
		return &r.topList, nil
	}
	switch current.event.Kind {
	case parser.SemanticListItem, parser.SemanticBlockquote, parser.SemanticAlert, parser.SemanticFootnoteDefinition:
		return &current.lastList, nil
	default:
		return nil, fmt.Errorf("%w: list output inside kind %d", ErrInvalidInput, current.event.Kind)
	}
}

func canonicalListMarker(ordered bool, previous listBoundary) byte {
	marker := byte('-')
	if ordered {
		marker = '.'
	}
	if !previous.valid || previous.ordered != ordered || previous.delimiter != marker {
		return marker
	}
	if ordered {
		return ')'
	}
	return '*'
}

func (r *renderer) appendListItem(item listItem) error {
	if len(r.stack) == 0 || r.stack[len(r.stack)-1].event.Kind != parser.SemanticList {
		return fmt.Errorf("%w: list item outside list", ErrInvalidInput)
	}
	current := &r.stack[len(r.stack)-1]
	current.items = append(current.items, item)
	return nil
}

func (r *renderer) appendTableCell(cell tableCell) error {
	if len(r.stack) == 0 || r.stack[len(r.stack)-1].event.Kind != parser.SemanticTableRow {
		return fmt.Errorf("%w: table cell outside row", ErrInvalidInput)
	}
	current := &r.stack[len(r.stack)-1]
	current.cells = append(current.cells, cell)
	return nil
}

func (r *renderer) appendTableRow(row tableRow) error {
	if len(r.stack) == 0 || r.stack[len(r.stack)-1].event.Kind != parser.SemanticTable {
		return fmt.Errorf("%w: table row outside table", ErrInvalidInput)
	}
	current := &r.stack[len(r.stack)-1]
	current.rows = append(current.rows, row)
	return nil
}

func (r *renderer) recordTask(checked bool) error {
	for index := len(r.stack) - 1; index >= 0; index-- {
		if r.stack[index].event.Kind != parser.SemanticListItem {
			continue
		}
		if r.stack[index].task {
			return fmt.Errorf("%w: duplicate task marker", ErrInvalidInput)
		}
		r.stack[index].task = true
		r.stack[index].checked = checked
		return nil
	}
	return fmt.Errorf("%w: task marker outside list item", ErrInvalidInput)
}

func (r *renderer) writeTopLevelBlock(value string) error {
	if r.wroteBlock {
		if err := writeAllString(r.writer, "\n"); err != nil {
			return err
		}
	}
	if err := writeAllString(r.writer, value); err != nil {
		return err
	}
	r.wroteBlock = true
	return nil
}

func (r *renderer) currentContainerKind() parser.SemanticKind {
	if len(r.stack) == 0 {
		return parser.SemanticUnknown
	}
	return r.stack[len(r.stack)-1].event.Kind
}

func (r *renderer) pop(kind parser.SemanticKind) (frame, error) {
	if len(r.stack) == 0 {
		return frame{}, fmt.Errorf("%w: unbalanced exit %d", ErrInvalidInput, kind)
	}
	last := len(r.stack) - 1
	current := r.stack[last]
	if current.event.Kind != kind {
		return frame{}, fmt.Errorf("%w: exit %d closes %d", ErrInvalidInput, kind, current.event.Kind)
	}
	r.stack = r.stack[:last]
	return current, nil
}

func (r *renderer) emphasisDelimiter(kind parser.SemanticKind) string {
	if kind == parser.SemanticStrong {
		return "**"
	}
	if len(r.stack) != 0 {
		parent := r.stack[len(r.stack)-1]
		switch parent.event.Kind {
		case parser.SemanticStrong:
			if len(parent.inline) == 0 {
				return "_"
			}
		case parser.SemanticEmphasis:
			if parent.delimiter == "*" {
				return "_"
			}
			if parent.delimiter == "_" {
				return "*"
			}
		}
	}
	return "*"
}

func (r *renderer) strikethroughDelimiter() string {
	if len(r.stack) != 0 {
		parent := r.stack[len(r.stack)-1]
		if parent.event.Kind == parser.SemanticStrikethrough && parent.delimiter == "~~" {
			return "~"
		}
	}
	return "~~"
}

func (r *renderer) inTableCell() bool {
	for index := len(r.stack) - 1; index >= 0; index-- {
		if r.stack[index].event.Kind == parser.SemanticTableCell {
			return true
		}
	}
	return false
}

func (r *renderer) renderAutoLink(event parser.SemanticEvent) (string, bool) {
	if event.Range.Valid(len(r.source)) && event.Range.End-event.Range.Start >= 2 &&
		r.source[event.Range.Start] == '<' && r.source[event.Range.End-1] == '>' {
		return "<" + event.Value + ">", false
	}
	return event.Value, true
}

func (r *renderer) renderLinkLike(event parser.SemanticEvent, label string, image bool) string {
	shortcutImage := image && event.Label != "" && !referenceLabelUsesFootnoteSyntax(event.Label) &&
		r.referenceLabelSemanticKey(event.Label) == r.referenceLabelSemanticKey(label)
	if shortcutImage {
		label = event.Label
	}
	var output strings.Builder
	if image {
		output.WriteByte('!')
	}
	output.WriteByte('[')
	output.WriteString(label)
	output.WriteByte(']')
	if event.Label != "" && !referenceLabelUsesFootnoteSyntax(event.Label) {
		if shortcutImage {
			return output.String()
		}
		output.WriteByte('[')
		output.WriteString(event.Label)
		output.WriteByte(']')
		return output.String()
	}
	output.WriteByte('(')
	output.WriteString(renderDestination(event.Destination))
	if event.HasTitle {
		output.WriteString(" \"")
		output.WriteString(escapeDoubleQuotedTitle(event.Title))
		output.WriteByte('"')
	}
	output.WriteByte(')')
	return output.String()
}

func (r *renderer) referenceLabelSemanticKey(label string) string {
	return r.backend.ReferenceLabelKey(decodeEscapedASCIIPunctuation(label))
}

func decodeEscapedASCIIPunctuation(value string) string {
	var output strings.Builder
	output.Grow(len(value))
	for position := 0; position < len(value); {
		if value[position] == '\\' && position+1 < len(value) && isASCIIPunctuation(value[position+1]) {
			output.WriteByte(value[position+1])
			position += 2
			continue
		}
		output.WriteByte(value[position])
		position++
	}
	return output.String()
}

func referenceLabelUsesFootnoteSyntax(label string) bool {
	return strings.HasPrefix(label, "^")
}

func renderDestination(value string) string {
	if strings.Contains(value, ">") && bareDestinationSafe(value) {
		return value
	}
	return "<" + value + ">"
}

func bareDestinationSafe(value string) bool {
	depth := 0
	for index := 0; index < len(value); index++ {
		current := value[index]
		if current == '\\' && index+1 < len(value) {
			index++
			continue
		}
		if current <= ' ' || current == '<' {
			return false
		}
		switch current {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

func escapeDoubleQuotedTitle(value string) string {
	var output strings.Builder
	output.Grow(len(value))
	backslashes := 0
	for index := 0; index < len(value); index++ {
		current := value[index]
		if current == '\\' {
			backslashes++
			output.WriteByte(current)
			continue
		}
		if current == '"' && backslashes%2 == 0 {
			output.WriteByte('\\')
		}
		output.WriteByte(current)
		backslashes = 0
	}
	return output.String()
}

func (r *renderer) registerReference(event parser.SemanticEvent) error {
	key := r.backend.ReferenceLabelKey(event.Label)
	if key == "" {
		return fmt.Errorf("%w: empty normalized reference label", ErrInvalidInput)
	}
	target := referenceTarget{
		label:       canonicalReferenceLabelSpelling(event.Label),
		destination: event.Destination,
		title:       event.Title,
		hasTitle:    event.HasTitle,
	}
	if existing, ok := r.references[key]; ok {
		if existing.destination != target.destination || existing.title != target.title || existing.hasTitle != target.hasTitle {
			return fmt.Errorf("%w: inconsistent reference target for %q", ErrInvalidInput, event.Label)
		}
		return nil
	}
	r.references[key] = target
	r.referenceOrder = append(r.referenceOrder, key)
	return nil
}

func (r *renderer) canonicalReferenceLabel(label string) (string, error) {
	if r.backend.ReferenceLabelKey(label) == "" {
		return "", fmt.Errorf("%w: empty normalized reference label", ErrInvalidInput)
	}
	return canonicalReferenceLabelSpelling(label), nil
}

func canonicalReferenceLabelSpelling(label string) string {
	var output strings.Builder
	output.Grow(len(label))
	pendingSpace := false
	wrote := false
	for _, value := range label {
		if referenceLabelWhitespace(value) {
			if wrote {
				pendingSpace = true
			}
			continue
		}
		if pendingSpace {
			output.WriteByte(' ')
			pendingSpace = false
		}
		output.WriteRune(value)
		wrote = true
	}
	return output.String()
}

func referenceLabelWhitespace(value rune) bool {
	switch value {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	default:
		return false
	}
}

func (r *renderer) markReferenceDefinition(label string) {
	key := r.backend.ReferenceLabelKey(label)
	if key != "" {
		r.emittedRefs[key] = struct{}{}
	}
}

func (r *renderer) writeMissingReferenceDefinitions() error {
	for _, key := range r.referenceOrder {
		if _, emitted := r.emittedRefs[key]; emitted {
			continue
		}
		target := r.references[key]
		event := parser.SemanticEvent{
			Label:       target.label,
			Destination: target.destination,
			Title:       target.title,
			HasTitle:    target.hasTitle,
		}
		r.bufferTopLevelBlock(renderReferenceDefinition(event), parser.Range{}, false, false)
		r.emittedRefs[key] = struct{}{}
	}
	return nil
}

func renderReferenceDefinition(event parser.SemanticEvent) string {
	var output strings.Builder
	output.WriteByte('[')
	output.WriteString(event.Label)
	output.WriteString("]: ")
	output.WriteString(renderDestination(event.Destination))
	if event.HasTitle {
		output.WriteString(" \"")
		output.WriteString(escapeDoubleQuotedTitle(event.Title))
		output.WriteByte('"')
	}
	output.WriteByte('\n')
	return output.String()
}

func renderCodeSpan(value string, tableCell bool) string {
	maxRun := longestRun(value, '`')
	delimiter := strings.Repeat("`", maxRun+1)
	if delimiter == "" {
		delimiter = "`"
	}
	payload := value
	if tableCell && strings.Contains(payload, "|") {
		payload = strings.ReplaceAll(payload, "|", "\\|")
	}
	if needsCodeSpanPadding(value) {
		payload = " " + payload + " "
	}
	return delimiter + payload + delimiter
}

func needsCodeSpanPadding(value string) bool {
	if value == "" {
		return false
	}
	if value[0] == '`' || value[len(value)-1] == '`' {
		return true
	}
	return value[0] == ' ' && value[len(value)-1] == ' ' && !onlySpaces(value)
}

func onlySpaces(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] != ' ' {
			return false
		}
	}
	return true
}

func renderCodeBlock(event parser.SemanticEvent) (string, error) {
	value := normalizeLineEndings(event.Value)
	info := normalizeLineEndings(event.Info)
	fence, err := canonicalCodeFence(value, info)
	if err != nil {
		return "", err
	}
	var output strings.Builder
	output.WriteString(fence)
	output.WriteString(info)
	output.WriteByte('\n')
	output.WriteString(value)
	if value != "" && !strings.HasSuffix(value, "\n") {
		output.WriteByte('\n')
	}
	output.WriteString(fence)
	output.WriteByte('\n')
	return output.String(), nil
}

func renderTerminalCodeBlock(event parser.SemanticEvent) (string, error) {
	value := normalizeLineEndings(event.Value)
	info := normalizeLineEndings(event.Info)
	if !event.Fenced && info == "" {
		return renderTerminalIndentedCode(value), nil
	}
	fence, err := canonicalCodeFence(value, info)
	if err != nil {
		return "", err
	}
	return fence + info + "\n" + value, nil
}

func canonicalCodeFence(value, info string) (string, error) {
	if strings.ContainsAny(info, "\r\n") {
		return "", fmt.Errorf("%w: multiline fence info", ErrInvalidInput)
	}
	fenceChar := byte('`')
	if strings.ContainsRune(info, '`') {
		fenceChar = '~'
	}
	fenceLength := max(3, longestLeadingFenceRun(value, fenceChar)+1)
	return strings.Repeat(string(fenceChar), fenceLength), nil
}

func renderTerminalIndentedCode(value string) string {
	lines := strings.Split(value, "\n")
	var output strings.Builder
	for index, line := range lines {
		if line != "" {
			output.WriteString("    ")
			output.WriteString(line)
		}
		if index+1 < len(lines) {
			output.WriteByte('\n')
		}
	}
	return output.String()
}

func longestLeadingFenceRun(value string, marker byte) int {
	maxRun := 0
	for start := 0; start <= len(value); {
		end := strings.IndexByte(value[start:], '\n')
		if end < 0 {
			end = len(value)
		} else {
			end += start
		}
		position := start
		for position < end && position-start < 3 && value[position] == ' ' {
			position++
		}
		run := 0
		for position+run < end && value[position+run] == marker {
			run++
		}
		if run > maxRun {
			maxRun = run
		}
		if end == len(value) {
			break
		}
		start = end + 1
	}
	return maxRun
}

func renderMath(event parser.SemanticEvent) (string, bool, error) {
	switch event.MathStyle {
	case parser.MathExpressionInlineDollar:
		return "$" + event.Value + "$", false, nil
	case parser.MathExpressionInlineBacktick:
		return "$`" + event.Value + "`$", false, nil
	case parser.MathExpressionBlockDollar:
		return "$$" + event.Value + "$$\n", true, nil
	default:
		return "", false, fmt.Errorf("%w: math style %d", ErrInvalidInput, event.MathStyle)
	}
}

func indentBlock(value string, width int) string {
	if width <= 0 {
		return ensureBlock(value)
	}
	value = strings.TrimSuffix(ensureBlock(value), "\n")
	prefix := strings.Repeat(" ", width)
	var output strings.Builder
	for _, line := range strings.Split(value, "\n") {
		if strings.Trim(line, " \t") != "" {
			output.WriteString(prefix)
		}
		output.WriteString(line)
		output.WriteByte('\n')
	}
	return output.String()
}

func renderQuotedBlocks(blocks []string, marker string) string {
	body := joinBlocks(blocks, true)
	if marker != "" {
		if body == "" {
			body = marker + "\n"
		} else {
			body = marker + "\n" + body
		}
	}
	if body == "" {
		return ">\n"
	}
	body = strings.TrimSuffix(body, "\n")
	lines := strings.Split(body, "\n")
	var output strings.Builder
	for _, line := range lines {
		if line == "" {
			output.WriteString(">\n")
			continue
		}
		output.WriteString("> ")
		output.WriteString(line)
		output.WriteByte('\n')
	}
	return output.String()
}

func alertMarker(kind parser.SemanticAlertKind) (string, bool) {
	switch kind {
	case parser.SemanticAlertNote:
		return "[!NOTE]", true
	case parser.SemanticAlertTip:
		return "[!TIP]", true
	case parser.SemanticAlertImportant:
		return "[!IMPORTANT]", true
	case parser.SemanticAlertWarning:
		return "[!WARNING]", true
	case parser.SemanticAlertCaution:
		return "[!CAUTION]", true
	default:
		return "", false
	}
}

func renderList(event parser.SemanticEvent, items []listItem) (string, error) {
	if err := validateCanonicalList(event, items); err != nil {
		return "", err
	}
	var output strings.Builder
	for index, item := range items {
		if index != 0 && !event.Tight {
			output.WriteByte('\n')
		}
		markerText := canonicalListMarkerText(event, index)
		body := canonicalListItemBody(item, !event.Tight)
		writeCanonicalListItem(&output, markerText, body)
	}
	return output.String(), nil
}

func validateCanonicalList(event parser.SemanticEvent, items []listItem) error {
	if len(items) == 0 {
		return fmt.Errorf("%w: empty list", ErrInvalidInput)
	}
	if event.Ordered && event.Start < 0 {
		return fmt.Errorf("%w: negative ordered-list start", ErrInvalidInput)
	}
	if event.Ordered && event.Marker != '.' && event.Marker != ')' {
		return fmt.Errorf("%w: invalid ordered-list marker %q", ErrInvalidInput, event.Marker)
	}
	if !event.Ordered && event.Marker != '-' && event.Marker != '*' && event.Marker != '+' {
		return fmt.Errorf("%w: invalid unordered-list marker %q", ErrInvalidInput, event.Marker)
	}
	return nil
}

func canonicalListMarkerText(event parser.SemanticEvent, index int) string {
	if event.Ordered {
		return strconv.Itoa(event.Start+index) + string(event.Marker) + " "
	}
	return string(event.Marker) + " "
}

func canonicalListItemBody(item listItem, loose bool) string {
	body := strings.TrimSuffix(joinBlocks(item.blocks, loose), "\n")
	if !item.task {
		return body
	}
	if item.checked {
		return "[x]" + body
	}
	return "[ ]" + body
}

func writeCanonicalListItem(output *strings.Builder, markerText, body string) {
	if body == "" {
		output.WriteString(strings.TrimSuffix(markerText, " "))
		output.WriteByte('\n')
		return
	}
	lines := strings.Split(body, "\n")
	output.WriteString(markerText)
	output.WriteString(lines[0])
	output.WriteByte('\n')
	indent := strings.Repeat(" ", len(markerText))
	for _, line := range lines[1:] {
		if strings.Trim(line, " \t") != "" {
			output.WriteString(indent)
		}
		output.WriteString(line)
		output.WriteByte('\n')
	}
}

func renderTable(event parser.SemanticEvent, rows []tableRow) (string, error) {
	if event.Columns <= 0 || len(rows) == 0 {
		return "", fmt.Errorf("%w: invalid table structure", ErrInvalidInput)
	}
	alignments, err := tableAlignments(rows, event.Columns)
	if err != nil {
		return "", err
	}
	bodyStart := 0
	header := make([]tableCell, event.Columns)
	for column := range header {
		header[column] = tableCell{column: column, alignment: alignments[column]}
	}
	if rows[0].header {
		header, err = canonicalTableCells(rows[0], event.Columns, alignments)
		if err != nil {
			return "", err
		}
		bodyStart = 1
	}
	var output strings.Builder
	writeTableLine(&output, header)
	delimiters := make([]tableCell, event.Columns)
	for column, alignment := range alignments {
		delimiters[column] = tableCell{column: column, value: tableDelimiter(alignment), alignment: alignment}
	}
	writeTableLine(&output, delimiters)
	for _, row := range rows[bodyStart:] {
		if row.header {
			return "", fmt.Errorf("%w: header row after table body", ErrInvalidInput)
		}
		cells, err := canonicalTableCells(row, event.Columns, alignments)
		if err != nil {
			return "", err
		}
		writeTableLine(&output, cells)
	}
	return output.String(), nil
}

func tableAlignments(rows []tableRow, columns int) ([]parser.TableAlignment, error) {
	alignments := make([]parser.TableAlignment, columns)
	for _, row := range rows {
		for _, cell := range row.cells {
			if cell.column < 0 || cell.column >= columns {
				return nil, fmt.Errorf("%w: table column %d/%d", ErrInvalidInput, cell.column, columns)
			}
			if alignments[cell.column] == parser.TableAlignmentDefault && cell.alignment != parser.TableAlignmentDefault {
				alignments[cell.column] = cell.alignment
			}
		}
	}
	return alignments, nil
}

func canonicalTableCells(row tableRow, columns int, alignments []parser.TableAlignment) ([]tableCell, error) {
	cells := make([]tableCell, columns)
	seen := make([]bool, columns)
	for column := range cells {
		cells[column] = tableCell{column: column, alignment: alignments[column]}
	}
	for _, cell := range row.cells {
		if cell.column < 0 || cell.column >= columns {
			return nil, fmt.Errorf("%w: table column %d/%d", ErrInvalidInput, cell.column, columns)
		}
		if !seen[cell.column] || cells[cell.column].value == "" && cell.value != "" {
			cells[cell.column].value = cell.value
			seen[cell.column] = true
		}
	}
	return cells, nil
}

func writeTableLine(output *strings.Builder, cells []tableCell) {
	output.WriteByte('|')
	for _, cell := range cells {
		output.WriteByte(' ')
		output.WriteString(cell.value)
		output.WriteString(" |")
	}
	output.WriteByte('\n')
}

func tableDelimiter(alignment parser.TableAlignment) string {
	switch alignment {
	case parser.TableAlignmentLeft:
		return ":---"
	case parser.TableAlignmentRight:
		return "---:"
	case parser.TableAlignmentCenter:
		return ":---:"
	default:
		return "---"
	}
}

func renderFootnoteDefinition(label string, blocks []string) string {
	var output strings.Builder
	output.WriteString("[^")
	output.WriteString(label)
	output.WriteString("]:\n")
	body := joinBlocks(blocks, true)
	body = strings.TrimSuffix(body, "\n")
	if body == "" {
		return output.String()
	}
	for _, line := range strings.Split(body, "\n") {
		if line != "" {
			output.WriteString("    ")
			output.WriteString(line)
		}
		output.WriteByte('\n')
	}
	return output.String()
}

func joinBlocks(blocks []string, blank bool) string {
	if len(blocks) == 0 {
		return ""
	}
	separator := "\n"
	if blank {
		separator = "\n\n"
	}
	var output strings.Builder
	for index, block := range blocks {
		if index != 0 {
			output.WriteString(separator)
		}
		output.WriteString(strings.TrimSuffix(ensureBlock(block), "\n"))
	}
	output.WriteByte('\n')
	return output.String()
}

func renderHeading(level int, value string) string {
	if level < 1 || level > 6 {
		return ""
	}
	if strings.Contains(value, "\n") && level <= 2 {
		underline := "---"
		if level == 1 {
			underline = "==="
		}
		return strings.TrimSuffix(value, "\n") + "\n" + underline + "\n"
	}
	return strings.Repeat("#", level) + " " + value + "\n"
}

func canonicalOpaqueBlock(value string) string {
	value = normalizeLineEndings(value)
	value = strings.TrimRight(value, "\n")
	return value + "\n"
}

func ensureBlock(value string) string {
	if strings.HasSuffix(value, "\n") {
		return value
	}
	return value + "\n"
}

func normalizeLineEndings(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(value, "\r", "\n")
}

func escapeText(value string) string {
	var output strings.Builder
	output.Grow(len(value))
	for index := 0; index < len(value); index++ {
		current := value[index]
		switch current {
		case '\t':
			output.WriteString("&#9;")
			continue
		case '\n':
			output.WriteString("&#10;")
			continue
		case '\r':
			output.WriteString("&#13;")
			continue
		}
		if isASCIIPunctuation(current) {
			output.WriteByte('\\')
		}
		output.WriteByte(current)
	}
	return output.String()
}

func escapeTextAfterBareAutoLink(value string) string {
	prefixEnd := bareAutoLinkTailPrefixEnd(value)
	if prefixEnd == 0 {
		return escapeText(value)
	}
	return value[:prefixEnd] + escapeText(value[prefixEnd:])
}

func bareAutoLinkTailPrefixEnd(value string) int {
	if strings.HasPrefix(value, "<") {
		return 1
	}
	position := 0
	for position < len(value) && value[position] == ')' {
		position++
	}
	for position < len(value) && extendedAutoLinkTrailingPunctuation(value[position]) {
		position++
	}
	if position != 0 {
		return position
	}
	if len(value) < 4 || value[0] != '&' {
		return 0
	}
	position = 1
	for position < len(value) && asciiAlphaNumeric(value[position]) {
		position++
	}
	if position > 1 && position < len(value) && value[position] == ';' {
		return position + 1
	}
	return 0
}

func extendedAutoLinkTrailingPunctuation(value byte) bool {
	switch value {
	case '?', '!', '.', ',', ':', '*', '_', '~':
		return true
	default:
		return false
	}
}

func asciiAlphaNumeric(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}

func isASCIIPunctuation(value byte) bool {
	return value >= '!' && value <= '/' || value >= ':' && value <= '@' || value >= '[' && value <= '`' || value >= '{' && value <= '~'
}

func longestRun(value string, marker byte) int {
	maxRun := 0
	current := 0
	for index := 0; index < len(value); index++ {
		if value[index] == marker {
			current++
			if current > maxRun {
				maxRun = current
			}
			continue
		}
		current = 0
	}
	return maxRun
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
