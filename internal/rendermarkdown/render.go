// Package rendermarkdown renders Native semantic events as deterministic canonical Markdown.
package rendermarkdown

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

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
	event                                parser.SemanticEvent
	inline                               []byte
	blocks                               []string
	items                                []listItem
	rows                                 []tableRow
	cells                                []tableCell
	task                                 bool
	checked                              bool
	delimiter                            string
	bareAutoLinkTail                     bool
	lastList                             listBoundary
	lastDelimiterSibling                 inlineDelimiterSibling
	lastTextSibling                      inlineTextSibling
	emphasisDescendantDepth              int
	directEmphasisChildren               int
	sameMarkerDirectEmphasisChildren     int
	boundarySensitiveDirectEmphasisChild bool
	onlyDirectEmphasisChild              inlineDelimiterSibling
}

type inlineDelimiterSibling struct {
	kind                       parser.SemanticKind
	sourceRange                parser.Range
	outputStart                int
	outputEnd                  int
	descendantDepth            int
	directEmphasisChildren     int
	onlyDirectChildKind        parser.SemanticKind
	onlyDirectChildSourceRange parser.Range
	onlyDirectChildOutputStart int
	onlyDirectChildOutputEnd   int
	onlyDirectChildValid       bool
	emphasisChain              *inlineEmphasisChain
	valid                      bool
}

type inlineEmphasisChain struct {
	kind        parser.SemanticKind
	outputStart int
	outputEnd   int
	child       *inlineEmphasisChain
}

type inlineTextSibling struct {
	sourceRange parser.Range
	marker      byte
	outputEnd   int
	valid       bool
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
		r.preserveAdjacentEmphasisDelimiters(&current)
		r.preserveNestedEmphasisTopology(&current)
	case parser.SemanticStrikethrough:
		current.delimiter = r.strikethroughDelimiter()
		r.preserveNestedStrikethroughDelimiters(&current)
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
		return r.appendText(event)
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
	case parser.SemanticEmphasis, parser.SemanticStrong:
		return r.appendEmphasisInline(current)
	case parser.SemanticStrikethrough:
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

func (r *renderer) appendText(event parser.SemanticEvent) error {
	if len(r.stack) == 0 {
		return fmt.Errorf("%w: text output outside container", ErrInvalidInput)
	}
	current := &r.stack[len(r.stack)-1]
	value := event.Value
	escaped := escapeText(value)
	if r.preserveUnconsumedEmphasisRunSuffix(current, event) {
		escaped = r.escapeUnconsumedEmphasisRunSuffix(current, event)
	} else if r.preserveLeadingTabAfterDelimiter(current, event) {
		escaped = "\t" + escapeText(value[1:])
	} else if inlineAtLineStart(current.inline) {
		escaped = escapeLeadingTextSpaces(value)
	}
	if current.bareAutoLinkTail {
		escaped = escapeTextAfterBareAutoLink(value)
	}
	if err := r.appendInline(escaped); err != nil {
		return err
	}
	current = &r.stack[len(r.stack)-1]
	if marker, ok := trailingEmphasisMarker(value); ok {
		current.lastTextSibling = inlineTextSibling{
			sourceRange: event.Range,
			marker:      marker,
			outputEnd:   len(current.inline),
			valid:       true,
		}
	}
	return nil
}

func (r *renderer) preserveLeadingTabAfterDelimiter(parent *frame, event parser.SemanticEvent) bool {
	previous, ok := r.delimiterSiblingBeforeSourceTab(parent, event)
	if !ok {
		return false
	}
	candidate := make([]byte, len(parent.inline)+1)
	copy(candidate, parent.inline)
	candidate[len(parent.inline)] = '&'
	return delimiterRunFlankingChanged(r.source, previous.sourceRange.End, candidate, previous.outputEnd)
}

func (r *renderer) delimiterSiblingBeforeSourceTab(parent *frame, event parser.SemanticEvent) (inlineDelimiterSibling, bool) {
	if parent == nil || event.Value == "" || event.Value[0] != '\t' || !event.Range.Valid(len(r.source)) ||
		event.Range.Start >= len(r.source) || r.source[event.Range.Start] != '\t' {
		return inlineDelimiterSibling{}, false
	}
	previous := parent.lastDelimiterSibling
	width := delimiterWidth(previous.kind)
	if !previous.valid || width == 0 || previous.sourceRange.End != event.Range.Start ||
		previous.outputEnd != len(parent.inline) || previous.outputEnd-previous.outputStart < 2*width {
		return inlineDelimiterSibling{}, false
	}
	return previous, true
}

func delimiterRunFlankingChanged(source []byte, sourceEnd int, candidate []byte, candidateEnd int) bool {
	sourceRange, marker, ok := delimiterRunRangeAtEnd(source, sourceEnd)
	if !ok {
		return false
	}
	candidateRange, candidateMarker, ok := delimiterRunRangeAtEnd(candidate, candidateEnd)
	return ok && marker == candidateMarker &&
		delimiterRangeFlankingChanged(source, sourceRange, candidate, candidateRange, marker)
}

func delimiterRangeFlankingChanged(source []byte, sourceRun parser.Range, candidate []byte, candidateRun parser.Range, marker byte) bool {
	sourceSegment := parser.Range{Start: 0, End: len(source)}
	candidateSegment := parser.Range{Start: 0, End: len(candidate)}
	sourceOpen, sourceClose := parser.DelimiterFlanking(source, sourceSegment, sourceRun.Start, sourceRun.End, marker)
	candidateOpen, candidateClose := parser.DelimiterFlanking(candidate, candidateSegment, candidateRun.Start, candidateRun.End, marker)
	return sourceOpen != candidateOpen || sourceClose != candidateClose
}

func delimiterRunRangeAtEnd(source []byte, end int) (parser.Range, byte, bool) {
	if end <= 0 || end > len(source) {
		return parser.Range{}, 0, false
	}
	marker := source[end-1]
	if marker != '*' && marker != '_' && marker != '~' {
		return parser.Range{}, 0, false
	}
	start := end - 1
	for start > 0 && source[start-1] == marker {
		start--
	}
	return parser.Range{Start: start, End: end}, marker, true
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
		current.lastDelimiterSibling = inlineDelimiterSibling{}
		current.lastTextSibling = inlineTextSibling{}
		return nil
	default:
		return fmt.Errorf("%w: inline output inside kind %d", ErrInvalidInput, current.event.Kind)
	}
}

func (r *renderer) appendEmphasisInline(current frame) error {
	if len(r.stack) == 0 {
		return fmt.Errorf("%w: inline output outside container", ErrInvalidInput)
	}
	parent := &r.stack[len(r.stack)-1]
	r.preserveUnconsumedEmphasisRunPrefix(parent, &current)
	r.repairSeparatedThreeLevelEmphasis(&current)
	r.reconcileFinalSharedEmphasisPair(&current)
	r.repairSharedCloseThreeLevelEmphasis(&current)
	r.repairDeepUniqueEmphasisChain(parent, &current)
	start := len(parent.inline)
	if err := r.appendInline(current.delimiter + string(current.inline) + current.delimiter); err != nil {
		return err
	}
	sibling := inlineDelimiterSibling{
		kind:                   current.event.Kind,
		sourceRange:            current.event.Range,
		outputStart:            start,
		outputEnd:              len(parent.inline),
		descendantDepth:        current.emphasisDescendantDepth,
		directEmphasisChildren: current.directEmphasisChildren,
		valid:                  true,
	}
	if current.directEmphasisChildren == 1 && current.onlyDirectEmphasisChild.valid {
		child := current.onlyDirectEmphasisChild
		offset := start + len(current.delimiter)
		sibling.onlyDirectChildKind = child.kind
		sibling.onlyDirectChildSourceRange = child.sourceRange
		sibling.onlyDirectChildOutputStart = offset + child.outputStart
		sibling.onlyDirectChildOutputEnd = offset + child.outputEnd
		sibling.onlyDirectChildValid = true
		sibling.emphasisChain = &inlineEmphasisChain{
			kind:        child.kind,
			outputStart: len(current.delimiter) + child.outputStart,
			outputEnd:   len(current.delimiter) + child.outputEnd,
			child:       child.emphasisChain,
		}
	}
	parent.lastDelimiterSibling = sibling
	parent.emphasisDescendantDepth = max(parent.emphasisDescendantDepth, current.emphasisDescendantDepth+1)
	r.recordDirectEmphasisChild(parent, current, sibling)
	return nil
}

func (r *renderer) repairDeepUniqueEmphasisChain(parent *frame, current *frame) {
	if parent == nil || current == nil || delimiterWidth(parent.event.Kind) != 0 ||
		current.directEmphasisChildren != 1 || !current.onlyDirectEmphasisChild.valid {
		return
	}
	child := current.onlyDirectEmphasisChild
	if child.emphasisChain == nil || emphasisChainDepth(child.emphasisChain)+2 < 4 {
		return
	}
	candidate := make([]byte, len(current.inline)+2)
	candidate[0] = current.delimiter[0]
	copy(candidate[1:], current.inline)
	candidate[len(candidate)-1] = current.delimiter[0]
	pairs, ok := deepUniqueEmphasisPairs(len(candidate), child)
	if !ok {
		return
	}
	if r.repairSpecialDeepEmphasisTopology(current, child, candidate, pairs) ||
		delimiterCandidateMatches(candidate, pairs) {
		return
	}
	patterns := [][2]byte{
		{'*', '_'},
		{'*', '*'},
		{'_', '*'},
		{'_', '_'},
	}
	for _, pattern := range patterns {
		trial := append([]byte(nil), candidate...)
		for index, pair := range pairs {
			marker := pattern[0]
			if index == len(pairs)-1 {
				marker = pattern[1]
			}
			trial[pair[0].Start] = marker
			trial[pair[1].Start] = marker
		}
		if !delimiterCandidateMatches(trial, pairs) {
			continue
		}
		current.delimiter = string(trial[0])
		copy(current.inline, trial[1:len(trial)-1])
		return
	}
}

func (r *renderer) repairSpecialDeepEmphasisTopology(
	current *frame,
	child inlineDelimiterSibling,
	candidate []byte,
	pairs [][2]parser.Range,
) bool {
	return r.repairDeepSharedCloseEmphasis(current, child, candidate, pairs) ||
		r.repairSeparatedBoundaryFourLevelEmphasis(current, child, candidate, pairs)
}

func (r *renderer) repairSeparatedBoundaryFourLevelEmphasis(
	current *frame,
	child inlineDelimiterSibling,
	candidate []byte,
	pairs [][2]parser.Range,
) bool {
	if len(pairs) != 4 || !r.separatedBoundaryFourLevelSource(current, child) {
		return false
	}
	if delimiterCandidateMatchesPhysicalRuns(candidate, pairs) {
		return true
	}
	trial := append([]byte(nil), candidate...)
	markers := [4]byte{'*', '*', '_', '*'}
	for index, pair := range pairs {
		trial[pair[0].Start] = markers[index]
		trial[pair[1].Start] = markers[index]
	}
	trialPairs := pairs
	if !delimiterCandidateMatchesPhysicalRuns(trial, trialPairs) {
		var ok bool
		trial, trialPairs, ok = encodeSeparatedBoundaryASCIITail(trial, trialPairs)
		if !ok || !delimiterCandidateMatchesPhysicalRuns(trial, trialPairs) {
			return false
		}
	}
	current.delimiter = string(trial[0])
	current.inline = append(current.inline[:0], trial[1:len(trial)-1]...)
	return true
}

func encodeSeparatedBoundaryASCIITail(candidate []byte, pairs [][2]parser.Range) ([]byte, [][2]parser.Range, bool) {
	if len(pairs) != 4 {
		return nil, nil, false
	}
	closeRange := pairs[2][1]
	position := closeRange.End
	if !closeRange.Valid(len(candidate)) || closeRange.End-closeRange.Start != 1 ||
		candidate[closeRange.Start] != '_' || position >= len(candidate) || !asciiAlphaNumeric(candidate[position]) {
		return nil, nil, false
	}
	closeRun := sourceEmphasisRunAt(candidate, closeRange.Start, 1, '_')
	if !closeRun.Valid(len(candidate)) {
		return nil, nil, false
	}
	segment := parser.Range{Start: 0, End: len(candidate)}
	_, canClose := parser.DelimiterFlanking(candidate, segment, closeRun.Start, closeRun.End, '_')
	if canClose {
		return nil, nil, false
	}
	entity := []byte(fmt.Sprintf("&#%d;", candidate[position]))
	delta := len(entity) - 1
	encoded := make([]byte, 0, len(candidate)+delta)
	encoded = append(encoded, candidate[:position]...)
	encoded = append(encoded, entity...)
	encoded = append(encoded, candidate[position+1:]...)
	shifted := append([][2]parser.Range(nil), pairs...)
	for pairIndex := range shifted {
		for side := range shifted[pairIndex] {
			if shifted[pairIndex][side].Start > position {
				shifted[pairIndex][side].Start += delta
				shifted[pairIndex][side].End += delta
			}
		}
	}
	return encoded, shifted, true
}

func (r *renderer) separatedBoundaryFourLevelSource(
	current *frame,
	child inlineDelimiterSibling,
) bool {
	if current == nil || !child.onlyDirectChildValid ||
		delimiterWidth(current.event.Kind) != 1 || delimiterWidth(child.kind) != 1 ||
		delimiterWidth(child.onlyDirectChildKind) != 1 {
		return false
	}
	currentSource, currentOK := r.sourceEmphasisDelimiter(current.event.Kind, current.event.Range)
	childSource, childOK := r.sourceEmphasisDelimiter(child.kind, child.sourceRange)
	if !currentOK || !childOK || currentSource[0] != childSource[0] {
		return false
	}
	if current.event.Range.Start+1 == child.sourceRange.Start {
		return false
	}
	return child.sourceRange.End != current.event.Range.End-1
}

func (r *renderer) repairDeepSharedCloseEmphasis(
	current *frame,
	child inlineDelimiterSibling,
	candidate []byte,
	pairs [][2]parser.Range,
) bool {
	baseMarker, ok := r.sharedCloseEmphasisBaseMarker(current, child)
	if !ok {
		return false
	}
	if delimiterCandidateMatchesPhysicalRuns(candidate, pairs) {
		return true
	}
	trial := append([]byte(nil), candidate...)
	nestedMarker := alternateEmphasisMarker(baseMarker)
	for index, pair := range pairs {
		marker := nestedMarker
		if index < 2 {
			marker = baseMarker
		}
		trial[pair[0].Start] = marker
		trial[pair[1].Start] = marker
	}
	if !delimiterCandidateMatchesPhysicalRuns(trial, pairs) {
		return false
	}
	current.delimiter = string(trial[0])
	copy(current.inline, trial[1:len(trial)-1])
	return true
}

func (r *renderer) sharedCloseEmphasisBaseMarker(
	current *frame,
	child inlineDelimiterSibling,
) (byte, bool) {
	if current == nil || delimiterWidth(current.event.Kind) != 1 || delimiterWidth(child.kind) != 1 {
		return 0, false
	}
	currentSource, currentOK := r.sourceEmphasisDelimiter(current.event.Kind, current.event.Range)
	childSource, childOK := r.sourceEmphasisDelimiter(child.kind, child.sourceRange)
	if !currentOK || !childOK || currentSource[0] != childSource[0] {
		return 0, false
	}
	if !sharedCloseEmphasisPairRanges(current, child) ||
		!r.sourceEmphasisCloseRunShared(current, child, currentSource[0]) {
		return 0, false
	}
	return currentSource[0], true
}

func emphasisChainDepth(chain *inlineEmphasisChain) int {
	depth := 0
	for current := chain; current != nil; current = current.child {
		depth++
	}
	return depth
}

func deepUniqueEmphasisPairs(candidateLen int, child inlineDelimiterSibling) ([][2]parser.Range, bool) {
	if candidateLen < 2 || child.emphasisChain == nil {
		return nil, false
	}
	pairs := make([][2]parser.Range, 0, emphasisChainDepth(child.emphasisChain)+2)
	pairs = append(pairs,
		[2]parser.Range{
			{Start: 0, End: 1},
			{Start: candidateLen - 1, End: candidateLen},
		},
		[2]parser.Range{
			{Start: 1 + child.outputStart, End: 2 + child.outputStart},
			{Start: child.outputEnd, End: 1 + child.outputEnd},
		},
	)
	parentStart := 1 + child.outputStart
	for chain := child.emphasisChain; chain != nil; chain = chain.child {
		start := parentStart + chain.outputStart
		end := parentStart + chain.outputEnd
		if chain.kind != parser.SemanticEmphasis || start < 0 || end > candidateLen || end-start < 2 {
			return nil, false
		}
		pairs = append(pairs, [2]parser.Range{
			{Start: start, End: start + 1},
			{Start: end - 1, End: end},
		})
		parentStart = start
	}
	return pairs, true
}

func delimiterCandidateMatches(candidate []byte, pairs [][2]parser.Range) bool {
	runs, ok := emphasisCandidateRuns(candidate, pairs)
	if !ok {
		return false
	}
	return delimiterMatchesExpectedPairs(parser.ResolveDelimiterRuns(runs), pairs)
}

func (r *renderer) reconcileFinalSharedEmphasisPair(current *frame) {
	child, width, ok := r.finalSharedEmphasisPair(current)
	if !ok {
		return
	}
	if candidate, rewrite := r.threeLevelEmphasisCanonicalCandidate(current, child, width); rewrite {
		current.delimiter = string(candidate.delimiter)
		current.inline = candidate.inline
		return
	}
	leafMarker, rewriteLeaf := r.sharedEmphasisLeafMarker(current, child, width)
	rewriteNested := r.sharedEmphasisPairNeedsNestedRewrite(child)
	if child.descendantDepth < 2 && !rewriteNested && !rewriteLeaf {
		return
	}
	rewriteInlineSiblingMarker(current.inline, child.outputStart, child.outputEnd, width, current.delimiter[0])
	if rewriteLeaf {
		rewriteInlineSiblingMarker(
			current.inline,
			child.onlyDirectChildOutputStart,
			child.onlyDirectChildOutputEnd,
			width,
			leafMarker,
		)
		return
	}
	if rewriteNested {
		rewriteInlineSiblingMarker(
			current.inline,
			child.onlyDirectChildOutputStart,
			child.onlyDirectChildOutputEnd,
			width,
			alternateEmphasisMarker(current.delimiter[0]),
		)
	}
}

type threeLevelEmphasisSourceMarkers struct {
	outer byte
	child byte
	leaf  byte
}

type threeLevelEmphasisCandidate struct {
	delimiter byte
	inline    []byte
}

func (r *renderer) repairSharedCloseThreeLevelEmphasis(current *frame) {
	child, ok := threeLevelEmphasisChild(current)
	if !ok {
		return
	}
	markers, ok := r.sharedCloseThreeLevelSourceMarkers(current, child)
	if !ok {
		return
	}
	expected := threeLevelEmphasisExpectedPairs(len(current.inline)+2, child)
	if delimiterCandidateMatchesPhysicalRuns(
		delimitedInlineCandidate(current.inline, current.delimiter[0]),
		expected,
	) {
		return
	}
	inline := append([]byte(nil), current.inline...)
	rewriteInlineSiblingMarker(inline, child.outputStart, child.outputEnd, 1, markers.child)
	rewriteInlineSiblingMarker(
		inline,
		child.onlyDirectChildOutputStart,
		child.onlyDirectChildOutputEnd,
		1,
		markers.leaf,
	)
	if !delimiterCandidateMatchesPhysicalRuns(
		delimitedInlineCandidate(inline, markers.outer),
		expected,
	) {
		return
	}
	current.delimiter = string(markers.outer)
	copy(current.inline, inline)
}

func threeLevelEmphasisChild(current *frame) (inlineDelimiterSibling, bool) {
	if current == nil || current.directEmphasisChildren != 1 || !current.onlyDirectEmphasisChild.valid {
		return inlineDelimiterSibling{}, false
	}
	child := current.onlyDirectEmphasisChild
	if delimiterWidth(current.event.Kind) != 1 || delimiterWidth(child.kind) != 1 {
		return inlineDelimiterSibling{}, false
	}
	if child.descendantDepth != 1 || child.directEmphasisChildren != 1 || !child.onlyDirectChildValid {
		return inlineDelimiterSibling{}, false
	}
	if child.onlyDirectChildKind != parser.SemanticEmphasis {
		return inlineDelimiterSibling{}, false
	}
	return child, true
}

func (r *renderer) sharedCloseThreeLevelSourceMarkers(
	current *frame,
	child inlineDelimiterSibling,
) (threeLevelEmphasisSourceMarkers, bool) {
	currentSource, currentOK := r.sourceEmphasisDelimiter(current.event.Kind, current.event.Range)
	childSource, childOK := r.sourceEmphasisDelimiter(child.kind, child.sourceRange)
	leafSource, leafOK := r.sourceEmphasisDelimiter(child.onlyDirectChildKind, child.onlyDirectChildSourceRange)
	if !currentOK || !childOK || !leafOK {
		return threeLevelEmphasisSourceMarkers{}, false
	}
	if currentSource[0] != childSource[0] || !sharedCloseEmphasisPairRanges(current, child) {
		return threeLevelEmphasisSourceMarkers{}, false
	}
	if !r.sourceEmphasisCloseRunShared(current, child, currentSource[0]) {
		return threeLevelEmphasisSourceMarkers{}, false
	}
	return threeLevelEmphasisSourceMarkers{
		outer: currentSource[0],
		child: childSource[0],
		leaf:  leafSource[0],
	}, true
}

func sharedCloseEmphasisPairRanges(current *frame, child inlineDelimiterSibling) bool {
	if current.event.Range.Start+1 == child.sourceRange.Start {
		return false
	}
	return child.sourceRange.End == current.event.Range.End-1
}

func (r *renderer) sourceEmphasisCloseRunShared(
	current *frame,
	child inlineDelimiterSibling,
	marker byte,
) bool {
	currentClose := sourceEmphasisRunAt(r.source, current.event.Range.End-1, 1, marker)
	childClose := sourceEmphasisRunAt(r.source, child.sourceRange.End-1, 1, marker)
	return currentClose.Valid(len(r.source)) && currentClose == childClose
}

func delimitedInlineCandidate(inline []byte, marker byte) []byte {
	candidate := make([]byte, len(inline)+2)
	candidate[0] = marker
	copy(candidate[1:], inline)
	candidate[len(candidate)-1] = marker
	return candidate
}

func (r *renderer) repairSeparatedThreeLevelEmphasis(current *frame) {
	if current == nil || current.directEmphasisChildren != 1 || !current.onlyDirectEmphasisChild.valid {
		return
	}
	child := current.onlyDirectEmphasisChild
	width := delimiterWidth(current.event.Kind)
	childEvent := parser.SemanticEvent{Kind: child.kind, Range: child.sourceRange}
	currentSource, currentOK := r.sourceEmphasisDelimiter(current.event.Kind, current.event.Range)
	childSource, childOK := r.sourceEmphasisDelimiter(child.kind, child.sourceRange)
	sharedSourceBoundary := emphasisDelimiterEventsShareSourceRun(current.event, childEvent, width)
	if width != 1 || delimiterWidth(child.kind) != width ||
		(sharedSourceBoundary && (!currentOK || !childOK || currentSource[0] == childSource[0])) ||
		r.threeLevelEmphasisCurrentMatches(current, child) {
		return
	}
	candidate, ok := r.threeLevelEmphasisCanonicalCandidate(current, child, width)
	if !ok {
		return
	}
	current.delimiter = string(candidate.delimiter)
	current.inline = candidate.inline
}

func (r *renderer) threeLevelEmphasisCurrentMatches(current *frame, child inlineDelimiterSibling) bool {
	if current == nil || len(current.delimiter) != 1 || (current.delimiter[0] != '*' && current.delimiter[0] != '_') {
		return false
	}
	candidate := make([]byte, len(current.inline)+2)
	candidate[0] = current.delimiter[0]
	copy(candidate[1:], current.inline)
	candidate[len(candidate)-1] = current.delimiter[0]
	expected := threeLevelEmphasisExpectedPairs(len(candidate), child)
	runs, ok := emphasisCandidateRuns(candidate, expected)
	if !ok {
		return false
	}
	return delimiterMatchesExpectedPairs(parser.ResolveDelimiterRuns(runs), expected)
}

func (r *renderer) threeLevelEmphasisCanonicalCandidate(
	current *frame,
	child inlineDelimiterSibling,
	width int,
) (threeLevelEmphasisCandidate, bool) {
	if current == nil || width != 1 || child.descendantDepth != 1 || child.directEmphasisChildren != 1 ||
		!child.onlyDirectChildValid || child.kind != parser.SemanticEmphasis ||
		child.onlyDirectChildKind != parser.SemanticEmphasis {
		return threeLevelEmphasisCandidate{}, false
	}
	markers := [][2]byte{
		{'*', '_'},
		{'*', '*'},
		{'_', '*'},
		{'_', '_'},
	}
	if candidate, ok := threeLevelEmphasisCandidateForInline(current.inline, child, markers); ok {
		return candidate, true
	}
	tabInline, ok := r.threeLevelEmphasisSourceTabInline(current, child)
	if !ok {
		return threeLevelEmphasisCandidate{}, false
	}
	return threeLevelEmphasisCandidateForInline(tabInline, child, markers)
}

func threeLevelEmphasisCandidateForInline(
	baseInline []byte,
	child inlineDelimiterSibling,
	markers [][2]byte,
) (threeLevelEmphasisCandidate, bool) {
	for _, marker := range markers {
		inline := append([]byte(nil), baseInline...)
		rewriteInlineSiblingMarker(inline, child.outputStart, child.outputEnd, 1, marker[0])
		rewriteInlineSiblingMarker(
			inline,
			child.onlyDirectChildOutputStart,
			child.onlyDirectChildOutputEnd,
			1,
			marker[1],
		)
		candidate := delimitedInlineCandidate(inline, marker[0])
		expected := threeLevelEmphasisExpectedPairs(len(candidate), child)
		if delimiterCandidateMatches(candidate, expected) {
			return threeLevelEmphasisCandidate{delimiter: marker[0], inline: inline}, true
		}
	}
	return threeLevelEmphasisCandidate{}, false
}

func (r *renderer) threeLevelEmphasisSourceTabInline(
	current *frame,
	child inlineDelimiterSibling,
) ([]byte, bool) {
	position := child.outputEnd
	if current == nil || child.sourceRange.End >= len(r.source) || r.source[child.sourceRange.End] != '\t' ||
		position < 0 || position+4 > len(current.inline) ||
		string(current.inline[position:position+4]) != "&#9;" {
		return nil, false
	}
	inline := make([]byte, 0, len(current.inline)-3)
	inline = append(inline, current.inline[:position]...)
	inline = append(inline, '\t')
	inline = append(inline, current.inline[position+4:]...)
	return inline, true
}

func threeLevelEmphasisExpectedPairs(candidateLen int, child inlineDelimiterSibling) [][2]parser.Range {
	return [][2]parser.Range{
		{{Start: 0, End: 1}, {Start: candidateLen - 1, End: candidateLen}},
		{
			{Start: child.outputStart + 1, End: child.outputStart + 2},
			{Start: child.outputEnd, End: child.outputEnd + 1},
		},
		{
			{Start: child.onlyDirectChildOutputStart + 1, End: child.onlyDirectChildOutputStart + 2},
			{Start: child.onlyDirectChildOutputEnd, End: child.onlyDirectChildOutputEnd + 1},
		},
	}
}

func emphasisCandidateRuns(source []byte, pairs [][2]parser.Range) ([]parser.DelimiterRun, bool) {
	type runKey struct {
		start  int
		end    int
		marker byte
	}
	byRun := make(map[runKey]parser.DelimiterRun, len(pairs)*2)
	segment := parser.Range{Start: 0, End: len(source)}
	for _, pair := range pairs {
		for _, endpoint := range pair {
			if endpoint.Start < 0 || endpoint.End > len(source) || endpoint.End-endpoint.Start != 1 {
				return nil, false
			}
			marker := source[endpoint.Start]
			run := sourceEmphasisRunAt(source, endpoint.Start, 1, marker)
			if !run.Valid(len(source)) {
				return nil, false
			}
			canOpen, canClose := parser.DelimiterFlanking(source, segment, run.Start, run.End, marker)
			key := runKey{start: run.Start, end: run.End, marker: marker}
			byRun[key] = parser.DelimiterRun{
				Start:    run.Start,
				End:      run.End,
				Marker:   marker,
				CanOpen:  canOpen,
				CanClose: canClose,
			}
		}
	}
	runs := make([]parser.DelimiterRun, 0, len(byRun))
	for _, run := range byRun {
		runs = append(runs, run)
	}
	sort.Slice(runs, func(left, right int) bool {
		if runs[left].Start != runs[right].Start {
			return runs[left].Start < runs[right].Start
		}
		return runs[left].End < runs[right].End
	})
	return runs, true
}

func delimiterCandidateMatchesPhysicalRuns(candidate []byte, expected [][2]parser.Range) bool {
	runs, ok := emphasisCandidateRuns(candidate, expected)
	if !ok {
		return false
	}
	matches := parser.ResolveDelimiterRuns(runs)
	if len(matches) != len(expected) {
		return false
	}
	runPairs := make([][2]int, len(expected))
	for index, pair := range expected {
		for side, endpoint := range pair {
			if endpoint.Start < 0 || endpoint.Start >= len(candidate) {
				return false
			}
			marker := candidate[endpoint.Start]
			run := sourceEmphasisRunAt(candidate, endpoint.Start, 1, marker)
			runIndex := delimiterRunIndex(runs, run, marker)
			if runIndex < 0 {
				return false
			}
			runPairs[index][side] = runIndex
		}
	}
	seen := make([]bool, len(runPairs))
	for _, match := range matches {
		if match.Level != 1 {
			return false
		}
		found := false
		for index, pair := range runPairs {
			if !seen[index] && match.OpenerRun == pair[0] && match.CloserRun == pair[1] {
				seen[index] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func delimiterRunIndex(runs []parser.DelimiterRun, target parser.Range, marker byte) int {
	if !target.Valid(target.End) {
		return -1
	}
	for index, run := range runs {
		if run.Start == target.Start && run.End == target.End && run.Marker == marker {
			return index
		}
	}
	return -1
}

func delimiterMatchesExpectedPairs(matches []parser.DelimiterRunMatch, expected [][2]parser.Range) bool {
	if len(matches) != len(expected) {
		return false
	}
	seen := make([]bool, len(expected))
	for _, match := range matches {
		if match.Level != 1 {
			return false
		}
		found := false
		for index, pair := range expected {
			if !seen[index] && match.OpeningConsumed == pair[0] && match.ClosingConsumed == pair[1] {
				seen[index] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (r *renderer) sharedEmphasisLeafMarker(current *frame, child inlineDelimiterSibling, width int) (byte, bool) {
	if current == nil || width != 1 || child.descendantDepth != 1 || child.directEmphasisChildren != 1 ||
		!child.onlyDirectChildValid || child.kind != parser.SemanticEmphasis ||
		child.onlyDirectChildKind != parser.SemanticEmphasis {
		return 0, false
	}
	childSource, childOK := r.sourceEmphasisDelimiter(child.kind, child.sourceRange)
	leafSource, leafOK := r.sourceEmphasisDelimiter(child.onlyDirectChildKind, child.onlyDirectChildSourceRange)
	if !childOK || !leafOK || childSource[0] == leafSource[0] ||
		child.sourceRange.Start+width != child.onlyDirectChildSourceRange.Start {
		return 0, false
	}
	marker := current.delimiter[0]
	if child.onlyDirectChildSourceRange.End == child.sourceRange.End-width {
		marker = alternateEmphasisMarker(marker)
	}
	return marker, true
}

func (r *renderer) finalSharedEmphasisPair(current *frame) (inlineDelimiterSibling, int, bool) {
	if current == nil || current.directEmphasisChildren != 1 || !current.onlyDirectEmphasisChild.valid {
		return inlineDelimiterSibling{}, 0, false
	}
	child := current.onlyDirectEmphasisChild
	width := delimiterWidth(current.event.Kind)
	if width == 0 || delimiterWidth(child.kind) != width || len(current.delimiter) != width {
		return inlineDelimiterSibling{}, 0, false
	}
	currentSource, currentOK := r.sourceEmphasisDelimiter(current.event.Kind, current.event.Range)
	childSource, childOK := r.sourceEmphasisDelimiter(child.kind, child.sourceRange)
	childEvent := parser.SemanticEvent{Kind: child.kind, Range: child.sourceRange}
	if !currentOK || !childOK || currentSource[0] != childSource[0] ||
		!emphasisDelimiterEventsShareSourceRun(current.event, childEvent, width) {
		return inlineDelimiterSibling{}, 0, false
	}
	childMarker, ok := inlineSiblingMarker(current.inline, child, width)
	if !ok || childMarker == current.delimiter[0] {
		return inlineDelimiterSibling{}, 0, false
	}
	return child, width, true
}

func (r *renderer) sharedEmphasisPairNeedsNestedRewrite(child inlineDelimiterSibling) bool {
	return child.directEmphasisChildren == 1 &&
		child.onlyDirectChildValid &&
		r.sourceInlineDelimiterDualPurpose(child.onlyDirectChildKind, child.onlyDirectChildSourceRange)
}

func inlineSiblingMarker(inline []byte, sibling inlineDelimiterSibling, width int) (byte, bool) {
	if width <= 0 || sibling.outputStart < 0 || sibling.outputEnd > len(inline) ||
		sibling.outputEnd-sibling.outputStart < 2*width {
		return 0, false
	}
	marker := inline[sibling.outputStart]
	if marker != '*' && marker != '_' {
		return 0, false
	}
	for index := 0; index < width; index++ {
		if inline[sibling.outputStart+index] != marker ||
			inline[sibling.outputEnd-width+index] != marker {
			return 0, false
		}
	}
	return marker, true
}

func rewriteInlineSiblingMarker(inline []byte, start, end, width int, marker byte) {
	if width <= 0 || start < 0 || end > len(inline) || end-start < 2*width ||
		(marker != '*' && marker != '_') {
		return
	}
	for index := 0; index < width; index++ {
		inline[start+index] = marker
		inline[end-width+index] = marker
	}
}

func alternateEmphasisMarker(marker byte) byte {
	if marker == '*' {
		return '_'
	}
	return '*'
}

func (r *renderer) sourceInlineDelimiterDualPurpose(kind parser.SemanticKind, range_ parser.Range) bool {
	width := delimiterWidth(kind)
	delimiter, ok := r.sourceEmphasisDelimiter(kind, range_)
	if !ok || width == 0 {
		return false
	}
	openRun := sourceEmphasisRunAt(r.source, range_.Start, width, delimiter[0])
	closeRun := sourceEmphasisRunAt(r.source, range_.End-width, width, delimiter[0])
	if !openRun.Valid(len(r.source)) || !closeRun.Valid(len(r.source)) {
		return false
	}
	segment := parser.Range{Start: 0, End: len(r.source)}
	openCan, openClose := parser.DelimiterFlanking(r.source, segment, openRun.Start, openRun.End, delimiter[0])
	closeOpen, closeCan := parser.DelimiterFlanking(r.source, segment, closeRun.Start, closeRun.End, delimiter[0])
	return openCan && openClose && closeOpen && closeCan
}

func sourceEmphasisRunAt(source []byte, position, width int, marker byte) parser.Range {
	if width <= 0 || position < 0 || position+width > len(source) ||
		(marker != '*' && marker != '_') {
		return parser.Range{}
	}
	for index := 0; index < width; index++ {
		if source[position+index] != marker || sourceByteEscapedAt(source, position+index) {
			return parser.Range{}
		}
	}
	start, end := position, position+width
	for start > 0 && source[start-1] == marker && !sourceByteEscapedAt(source, start-1) {
		start--
	}
	for end < len(source) && source[end] == marker && !sourceByteEscapedAt(source, end) {
		end++
	}
	return parser.Range{Start: start, End: end}
}

func trailingEmphasisMarker(value string) (byte, bool) {
	if value == "" {
		return 0, false
	}
	marker := value[len(value)-1]
	return marker, marker == '*' || marker == '_'
}

func (r *renderer) preserveUnconsumedEmphasisRunSuffix(parent *frame, event parser.SemanticEvent) bool {
	if parent == nil {
		return false
	}
	marker, ok := r.sourceUnconsumedRunSuffixMarker(event)
	if !ok {
		return false
	}
	previous := parent.lastDelimiterSibling
	if !previous.valid || !unconsumedSuffixTopologySensitive(previous) || previous.sourceRange.End != event.Range.Start ||
		previous.outputEnd != len(parent.inline) {
		return false
	}
	sourceDelimiter, ok := r.sourceEmphasisDelimiter(previous.kind, previous.sourceRange)
	if !ok || sourceDelimiter[0] != marker || len(parent.inline) < len(sourceDelimiter) {
		return false
	}
	if string(parent.inline[len(parent.inline)-len(sourceDelimiter):]) != sourceDelimiter {
		return false
	}
	return !r.unconsumedSuffixCreatesModuloThreeConflict(parent, previous, event, marker)
}

func (r *renderer) unconsumedSuffixCreatesModuloThreeConflict(parent *frame, previous inlineDelimiterSibling, event parser.SemanticEvent, marker byte) bool {
	width := delimiterWidth(previous.kind)
	if parent == nil || width <= 0 || previous.outputStart < 0 || previous.outputEnd > len(parent.inline) ||
		previous.outputEnd-previous.outputStart < 2*width {
		return false
	}
	suffix := r.escapeUnconsumedEmphasisRunSuffix(parent, event)
	if suffix == "" || suffix[0] != marker {
		return false
	}
	candidate := make([]byte, len(parent.inline)+len(suffix))
	copy(candidate, parent.inline)
	copy(candidate[len(parent.inline):], suffix)
	openRun := sourceEmphasisRunAt(candidate, previous.outputStart, width, marker)
	closeRun := sourceEmphasisRunAt(candidate, previous.outputEnd-width, width, marker)
	if !openRun.Valid(len(candidate)) || !closeRun.Valid(len(candidate)) {
		return false
	}
	segment := parser.Range{Start: 0, End: len(candidate)}
	openCan, openClose := parser.DelimiterFlanking(candidate, segment, openRun.Start, openRun.End, marker)
	closeOpen, closeCan := parser.DelimiterFlanking(candidate, segment, closeRun.Start, closeRun.End, marker)
	return openCan && closeCan && parser.DelimiterRunsHaveModuloThreeConflict(
		openRun.End-openRun.Start,
		openClose,
		closeRun.End-closeRun.Start,
		closeOpen,
	)
}

func unconsumedSuffixTopologySensitive(previous inlineDelimiterSibling) bool {
	return previous.descendantDepth >= 2 || previous.directEmphasisChildren >= 2
}

func (r *renderer) escapeUnconsumedEmphasisRunSuffix(parent *frame, event parser.SemanticEvent) string {
	if r.preserveTabAfterUnconsumedRunSuffix(parent, event) {
		return event.Value[:2] + escapeText(event.Value[2:])
	}
	return event.Value[:1] + escapeText(event.Value[1:])
}

func (r *renderer) preserveTabAfterUnconsumedRunSuffix(parent *frame, event parser.SemanticEvent) bool {
	if parent == nil || len(event.Value) < 2 || event.Value[1] != '\t' ||
		event.Range.Start+1 >= len(r.source) || r.source[event.Range.Start+1] != '\t' {
		return false
	}
	candidate := make([]byte, len(parent.inline)+2)
	copy(candidate, parent.inline)
	candidate[len(parent.inline)] = event.Value[0]
	candidate[len(parent.inline)+1] = '&'
	return delimiterRunFlankingChanged(r.source, event.Range.Start+1, candidate, len(parent.inline)+1)
}

func (r *renderer) sourceUnconsumedRunSuffixMarker(event parser.SemanticEvent) (byte, bool) {
	if event.Value == "" || !event.Range.Valid(len(r.source)) || event.Range.Start >= len(r.source) {
		return 0, false
	}
	marker := event.Value[0]
	if (marker != '*' && marker != '_') || r.source[event.Range.Start] != marker ||
		sourceByteEscapedAt(r.source, event.Range.Start) {
		return 0, false
	}
	return marker, true
}

func (r *renderer) preserveUnconsumedEmphasisRunPrefix(parent *frame, current *frame) {
	previous, ok := textSiblingBeforeEmphasis(parent, current)
	if !ok {
		return
	}
	sourceDelimiter, sourceRunLength, ok := r.sourceUnconsumedRunDelimiter(previous, current.event)
	if !ok {
		return
	}
	if alternate, alternateOK := r.shallowUnconsumedPrefixAlternateDelimiter(current, sourceRunLength); alternateOK {
		current.delimiter = alternate
		return
	}
	preserveCount := 1
	if current.unconsumedPrefixTopologySensitive() {
		if r.multiChildPrefixConflict(current, sourceDelimiter) {
			return
		}
	} else {
		if !r.shallowUnconsumedPrefixTopologySensitive(current) {
			return
		}
		preserveCount = sourceRunLength
	}
	if !escapedTextMarkerRunTail(parent.inline, previous.marker, preserveCount) {
		return
	}
	start := len(parent.inline) - 2*preserveCount
	for index := 0; index < preserveCount; index++ {
		parent.inline[start+index] = previous.marker
	}
	parent.inline = parent.inline[:start+preserveCount]
	parent.lastTextSibling = inlineTextSibling{}
	current.delimiter = sourceDelimiter
}

func textSiblingBeforeEmphasis(parent *frame, current *frame) (inlineTextSibling, bool) {
	if parent == nil || current == nil {
		return inlineTextSibling{}, false
	}
	previous := parent.lastTextSibling
	ok := previous.valid &&
		previous.sourceRange.End == current.event.Range.Start &&
		previous.outputEnd == len(parent.inline) &&
		previous.sourceRange.Start < previous.sourceRange.End
	return previous, ok
}

func (current *frame) unconsumedPrefixTopologySensitive() bool {
	return current.emphasisDescendantDepth >= 2 ||
		(current.directEmphasisChildren >= 2 && current.boundarySensitiveDirectEmphasisChild)
}

func (r *renderer) multiChildPrefixConflict(current *frame, sourceDelimiter string) bool {
	if len(sourceDelimiter) != 1 || !multiChildPrefixTopology(current) {
		return false
	}
	marker := sourceDelimiter[0]
	if r.sourceMultiChildSharedOpenPrefixBoundary(current, marker) &&
		renderedMultiChildSharedOpenAlternate(current, marker) {
		return true
	}
	return r.sourceMultiChildSeparatedPrefixBoundary(current, marker)
}

func multiChildPrefixTopology(current *frame) bool {
	return current != nil && delimiterWidth(current.event.Kind) == 1 &&
		current.directEmphasisChildren == 2 && current.sameMarkerDirectEmphasisChildren == 2 &&
		current.emphasisDescendantDepth == 1 && current.boundarySensitiveDirectEmphasisChild && len(current.inline) >= 2
}

func (r *renderer) sourceMultiChildSharedOpenPrefixBoundary(current *frame, marker byte) bool {
	if !current.event.Range.Valid(len(r.source)) {
		return false
	}
	start, end := current.event.Range.Start, current.event.Range.End
	return start+1 < end-1 && r.source[start+1] == marker && !sourceByteEscapedAt(r.source, start+1)
}

func renderedMultiChildSharedOpenAlternate(current *frame, marker byte) bool {
	firstMarker := current.inline[0]
	return (firstMarker == '*' || firstMarker == '_') && firstMarker != marker
}

func (r *renderer) sourceMultiChildSeparatedPrefixBoundary(current *frame, marker byte) bool {
	if marker != '*' || !current.event.Range.Valid(len(r.source)) {
		return false
	}
	start, end := current.event.Range.Start, current.event.Range.End
	if start+1 >= end-1 || r.source[start+1] == marker ||
		(r.source[end-2] == marker && !sourceByteEscapedAt(r.source, end-2)) {
		return false
	}
	segment := parser.Range{Start: 0, End: len(r.source)}
	canOpen, canClose := parser.DelimiterFlanking(r.source, segment, start, start+1, marker)
	return canOpen && canClose
}

func (r *renderer) shallowUnconsumedPrefixAlternateDelimiter(current *frame, sourceRunLength int) (string, bool) {
	child, ok := shallowAlternatePrefixChild(current, sourceRunLength)
	if !ok || !r.shallowAlternatePrefixSourceBoundary(current, child) {
		return "", false
	}
	return "_", true
}

func shallowAlternatePrefixChild(current *frame, sourceRunLength int) (inlineDelimiterSibling, bool) {
	if sourceRunLength != 1 || current == nil {
		return inlineDelimiterSibling{}, false
	}
	if current.emphasisDescendantDepth != 1 || current.directEmphasisChildren != 1 ||
		!current.onlyDirectEmphasisChild.valid || delimiterWidth(current.event.Kind) != 1 {
		return inlineDelimiterSibling{}, false
	}
	child := current.onlyDirectEmphasisChild
	if child.descendantDepth != 0 || child.directEmphasisChildren != 0 || delimiterWidth(child.kind) != 1 {
		return inlineDelimiterSibling{}, false
	}
	return child, true
}

func (r *renderer) shallowAlternatePrefixSourceBoundary(current *frame, child inlineDelimiterSibling) bool {
	currentDelimiter, currentOK := r.sourceEmphasisDelimiter(current.event.Kind, current.event.Range)
	childDelimiter, childOK := r.sourceEmphasisDelimiter(child.kind, child.sourceRange)
	if !currentOK || !childOK || currentDelimiter != "*" || childDelimiter != "*" {
		return false
	}
	if child.sourceRange.Start <= current.event.Range.Start+1 || child.sourceRange.End >= current.event.Range.End-1 {
		return false
	}
	return r.shallowAlternatePrefixFlanking(current, child)
}

func (r *renderer) shallowAlternatePrefixFlanking(current *frame, child inlineDelimiterSibling) bool {
	outerOpen := sourceEmphasisRunAt(r.source, current.event.Range.Start, 1, '*')
	if !outerOpen.Valid(len(r.source)) {
		return false
	}
	outerClose := sourceEmphasisRunAt(r.source, current.event.Range.End-1, 1, '*')
	if !outerClose.Valid(len(r.source)) {
		return false
	}
	childOpen := sourceEmphasisRunAt(r.source, child.sourceRange.Start, 1, '*')
	if !childOpen.Valid(len(r.source)) {
		return false
	}
	childClose := sourceEmphasisRunAt(r.source, child.sourceRange.End-1, 1, '*')
	if !childClose.Valid(len(r.source)) {
		return false
	}
	afterWhitespace, _ := parser.DelimiterFollowingClass(r.source, childClose.Start, childClose.End, len(r.source))
	if afterWhitespace {
		return false
	}
	segment := parser.Range{Start: 0, End: len(r.source)}
	oo, oc := parser.DelimiterFlanking(r.source, segment, outerOpen.Start, outerOpen.End, '*')
	co, cc := parser.DelimiterFlanking(r.source, segment, outerClose.Start, outerClose.End, '*')
	io, ic := parser.DelimiterFlanking(r.source, segment, childOpen.Start, childOpen.End, '*')
	jo, jc := parser.DelimiterFlanking(r.source, segment, childClose.Start, childClose.End, '*')
	return oo && !oc && !co && cc && io && ic && !jo && jc
}

func (r *renderer) shallowUnconsumedPrefixTopologySensitive(current *frame) bool {
	if current == nil || current.emphasisDescendantDepth != 1 || current.directEmphasisChildren != 1 ||
		!current.onlyDirectEmphasisChild.valid || delimiterWidth(current.event.Kind) != 1 {
		return false
	}
	child := current.onlyDirectEmphasisChild
	if child.descendantDepth != 0 || child.directEmphasisChildren != 0 || delimiterWidth(child.kind) != 1 {
		return false
	}
	return r.shallowUnconsumedPrefixSourceBoundary(current, child)
}

func (r *renderer) shallowUnconsumedPrefixSourceBoundary(current *frame, child inlineDelimiterSibling) bool {
	currentDelimiter, currentOK := r.sourceEmphasisDelimiter(current.event.Kind, current.event.Range)
	childDelimiter, childOK := r.sourceEmphasisDelimiter(child.kind, child.sourceRange)
	if !currentOK || !childOK || currentDelimiter[0] != '*' || childDelimiter[0] != '*' {
		return false
	}
	tailPosition := child.sourceRange.End
	return tailPosition < current.event.Range.End-1 && tailPosition < len(r.source) &&
		asciiAlphaNumeric(r.source[tailPosition]) && r.sourceEmphasisCloserDualPurpose(child, '*')
}

func (r *renderer) sourceEmphasisCloserDualPurpose(child inlineDelimiterSibling, marker byte) bool {
	closeRun := sourceEmphasisRunAt(r.source, child.sourceRange.End-1, 1, marker)
	if !closeRun.Valid(len(r.source)) {
		return false
	}
	segment := parser.Range{Start: 0, End: len(r.source)}
	canOpen, canClose := parser.DelimiterFlanking(r.source, segment, closeRun.Start, closeRun.End, marker)
	return canOpen && canClose
}

func (r *renderer) recordDirectEmphasisChild(parent *frame, current frame, sibling inlineDelimiterSibling) {
	if parent == nil || delimiterWidth(parent.event.Kind) == 0 {
		return
	}
	parent.directEmphasisChildren++
	if parent.directEmphasisChildren == 1 {
		parent.onlyDirectEmphasisChild = sibling
	} else {
		parent.onlyDirectEmphasisChild = inlineDelimiterSibling{}
	}
	parentDelimiter, ok := r.sourceEmphasisDelimiter(parent.event.Kind, parent.event.Range)
	if !ok {
		return
	}
	childDelimiter, ok := r.sourceEmphasisDelimiter(current.event.Kind, current.event.Range)
	if !ok || parentDelimiter[0] != childDelimiter[0] {
		return
	}
	parent.sameMarkerDirectEmphasisChildren++
	width := delimiterWidth(current.event.Kind)
	if width == 0 || current.event.Range.Start+width > len(r.source) {
		return
	}
	segment := parser.Range{Start: 0, End: len(r.source)}
	canOpen, canClose := parser.DelimiterFlanking(
		r.source,
		segment,
		current.event.Range.Start,
		current.event.Range.Start+width,
		childDelimiter[0],
	)
	if canOpen && canClose {
		parent.boundarySensitiveDirectEmphasisChild = true
	}
}

func (r *renderer) sourceUnconsumedRunDelimiter(previous inlineTextSibling, event parser.SemanticEvent) (string, int, bool) {
	sourceDelimiter, ok := r.sourceEmphasisDelimiter(event.Kind, event.Range)
	if !ok || sourceDelimiter[0] != previous.marker {
		return "", 0, false
	}
	markerPosition := previous.sourceRange.End - 1
	if markerPosition < previous.sourceRange.Start || markerPosition >= len(r.source) ||
		r.source[markerPosition] != previous.marker || sourceByteEscapedAt(r.source, markerPosition) {
		return "", 0, false
	}
	start := markerPosition
	for start > previous.sourceRange.Start && r.source[start-1] == previous.marker && !sourceByteEscapedAt(r.source, start-1) {
		start--
	}
	return sourceDelimiter, markerPosition - start + 1, true
}

func escapedTextMarkerRunTail(inline []byte, marker byte, count int) bool {
	if count <= 0 || len(inline) < 2*count {
		return false
	}
	start := len(inline) - 2*count
	for index := 0; index < count; index++ {
		if inline[start+2*index] != '\\' || inline[start+2*index+1] != marker {
			return false
		}
	}
	return true
}

func sourceByteEscapedAt(source []byte, position int) bool {
	backslashes := 0
	for position > 0 && source[position-1] == '\\' {
		backslashes++
		position--
	}
	return backslashes%2 != 0
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

func (r *renderer) preserveAdjacentEmphasisDelimiters(current *frame) {
	if current == nil || len(r.stack) == 0 {
		return
	}
	parent := &r.stack[len(r.stack)-1]
	previous := parent.lastDelimiterSibling
	if !previous.valid || previous.sourceRange.End != current.event.Range.Start || previous.outputEnd != len(parent.inline) {
		return
	}
	previousDelimiter, ok := r.sourceEmphasisDelimiter(previous.kind, previous.sourceRange)
	if !ok {
		return
	}
	currentDelimiter, ok := r.sourceEmphasisDelimiter(current.event.Kind, current.event.Range)
	if !ok || len(previousDelimiter) != delimiterWidth(previous.kind) || len(currentDelimiter) != delimiterWidth(current.event.Kind) {
		return
	}
	if previous.outputStart < 0 || previous.outputEnd > len(parent.inline) ||
		previous.outputEnd-previous.outputStart < 2*len(previousDelimiter) {
		return
	}
	copy(parent.inline[previous.outputStart:previous.outputStart+len(previousDelimiter)], previousDelimiter)
	copy(parent.inline[previous.outputEnd-len(previousDelimiter):previous.outputEnd], previousDelimiter)
	current.delimiter = currentDelimiter
}

func (r *renderer) preserveNestedEmphasisTopology(current *frame) {
	if current == nil || len(r.stack) == 0 {
		return
	}
	r.preserveUnsafeDirectEmphasisDelimiter(current)
	r.preserveSharedEmphasisRunComponent(current)
	r.preserveAncestorEmphasisMarkerRelation(current)
}

func (r *renderer) preserveSharedEmphasisRunComponent(current *frame) {
	width := delimiterWidth(current.event.Kind)
	currentSource, ok := r.sourceEmphasisDelimiter(current.event.Kind, current.event.Range)
	if width == 0 || !ok {
		return
	}
	previous := current.event
	component := make([]int, 0, 2)
	sameMarker := true
	for index := len(r.stack) - 1; index >= 0; index-- {
		ancestor := &r.stack[index]
		ancestorWidth := delimiterWidth(ancestor.event.Kind)
		if ancestorWidth == 0 {
			continue
		}
		ancestorSource, ok := r.sourceEmphasisDelimiter(ancestor.event.Kind, ancestor.event.Range)
		if !ok || ancestorWidth != width ||
			!emphasisDelimiterEventsShareSourceRun(ancestor.event, previous, width) {
			break
		}
		sameMarker = sameMarker && ancestorSource[0] == currentSource[0]
		component = append(component, index)
		previous = ancestor.event
	}
	anchorIndex := -1
	if len(component) < 2 {
		return
	}
	if !sameMarker {
		var anchored bool
		anchorIndex, anchored = r.sharedEmphasisRunComponentAnchor(component, width)
		if !anchored {
			return
		}
	}
	current.delimiter = currentSource
	for _, index := range component {
		r.preserveFrameSourceEmphasisDelimiter(index)
	}
	if anchorIndex >= 0 {
		r.preserveFrameSourceEmphasisDelimiter(anchorIndex)
	}
}

func (r *renderer) sharedEmphasisRunComponentAnchor(component []int, width int) (int, bool) {
	if len(component) == 0 {
		return -1, false
	}
	rootIndex := component[len(component)-1]
	rootSource, ok := r.sourceEmphasisDelimiter(r.stack[rootIndex].event.Kind, r.stack[rootIndex].event.Range)
	if !ok {
		return -1, false
	}
	for index := rootIndex - 1; index >= 0; index-- {
		ancestor := &r.stack[index]
		if delimiterWidth(ancestor.event.Kind) != width {
			continue
		}
		ancestorSource, ok := r.sourceEmphasisDelimiter(ancestor.event.Kind, ancestor.event.Range)
		if ok && ancestorSource[0] == rootSource[0] {
			return index, true
		}
	}
	return -1, false
}

func (r *renderer) preserveFrameSourceEmphasisDelimiter(index int) {
	if index < 0 || index >= len(r.stack) {
		return
	}
	ancestor := &r.stack[index]
	sourceDelimiter, ok := r.sourceEmphasisDelimiter(ancestor.event.Kind, ancestor.event.Range)
	if ok {
		ancestor.delimiter = sourceDelimiter
	}
}

func emphasisDelimiterEventsShareSourceRun(outer, inner parser.SemanticEvent, width int) bool {
	if width <= 0 || outer.Range.Start > inner.Range.Start || inner.Range.End > outer.Range.End {
		return false
	}
	openShared := outer.Range.Start+width == inner.Range.Start
	closeShared := inner.Range.End == outer.Range.End-width
	return openShared || closeShared
}

func (r *renderer) preserveUnsafeDirectEmphasisDelimiter(current *frame) {
	parent := &r.stack[len(r.stack)-1]
	if delimiterWidth(parent.event.Kind) == 0 || delimiterWidth(current.event.Kind) == 0 ||
		r.sourceEmphasisDelimiterUsable(current.event, current.delimiter) {
		return
	}
	parentDelimiter, parentOK := r.sourceEmphasisDelimiter(parent.event.Kind, parent.event.Range)
	currentDelimiter, currentOK := r.sourceEmphasisDelimiter(current.event.Kind, current.event.Range)
	if !parentOK || !currentOK {
		return
	}
	parent.delimiter = parentDelimiter
	current.delimiter = currentDelimiter
}

func (r *renderer) preserveAncestorEmphasisMarkerRelation(current *frame) {
	width := delimiterWidth(current.event.Kind)
	if width == 0 || len(current.delimiter) != width {
		return
	}
	currentSource, ok := r.sourceEmphasisDelimiter(current.event.Kind, current.event.Range)
	if !ok {
		return
	}
	for index := len(r.stack) - 1; index >= 0; index-- {
		ancestor := &r.stack[index]
		if delimiterWidth(ancestor.event.Kind) != width || len(ancestor.delimiter) != width {
			continue
		}
		ancestorSource, ok := r.sourceEmphasisDelimiter(ancestor.event.Kind, ancestor.event.Range)
		if !ok || ancestorSource[0] == currentSource[0] {
			return
		}
		ancestor.delimiter = ancestorSource
		current.delimiter = currentSource
		return
	}
}

func (r *renderer) sourceEmphasisDelimiterUsable(event parser.SemanticEvent, delimiter string) bool {
	width := delimiterWidth(event.Kind)
	if width == 0 || len(delimiter) != width || !event.Range.Valid(len(r.source)) ||
		event.Range.End-event.Range.Start < 2*width {
		return false
	}
	marker := delimiter[0]
	if marker != '*' && marker != '_' {
		return false
	}
	segment := parser.Range{Start: 0, End: len(r.source)}
	openCan, _ := parser.DelimiterFlanking(r.source, segment, event.Range.Start, event.Range.Start+width, marker)
	_, closeCan := parser.DelimiterFlanking(r.source, segment, event.Range.End-width, event.Range.End, marker)
	return openCan && closeCan
}

func (r *renderer) sourceEmphasisDelimiter(kind parser.SemanticKind, range_ parser.Range) (string, bool) {
	width := delimiterWidth(kind)
	if width == 0 || !range_.Valid(len(r.source)) || range_.End-range_.Start < 2*width {
		return "", false
	}
	marker := r.source[range_.Start]
	if marker != '*' && marker != '_' {
		return "", false
	}
	for index := 0; index < width; index++ {
		if r.source[range_.Start+index] != marker || r.source[range_.End-width+index] != marker {
			return "", false
		}
	}
	return strings.Repeat(string(marker), width), true
}

func delimiterWidth(kind parser.SemanticKind) int {
	switch kind {
	case parser.SemanticEmphasis:
		return 1
	case parser.SemanticStrong:
		return 2
	default:
		return 0
	}
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

func (r *renderer) preserveNestedStrikethroughDelimiters(current *frame) {
	if current == nil || len(r.stack) == 0 {
		return
	}
	ancestor := r.nearestStrikethroughAncestor()
	if ancestor == nil {
		return
	}
	ancestorDelimiter, ok := r.sourceStrikethroughDelimiter(ancestor.event)
	if !ok {
		return
	}
	currentDelimiter, ok := r.sourceStrikethroughDelimiter(current.event)
	if !ok {
		return
	}
	parent := &r.stack[len(r.stack)-1]
	r.preserveNestedStrikethroughBoundaryFlanking(parent, current.event, currentDelimiter)
	ancestor.delimiter = ancestorDelimiter
	current.delimiter = currentDelimiter
}

func (r *renderer) nearestStrikethroughAncestor() *frame {
	for index := len(r.stack) - 1; index >= 0; index-- {
		if r.stack[index].event.Kind == parser.SemanticStrikethrough {
			return &r.stack[index]
		}
	}
	return nil
}

func (r *renderer) preserveNestedStrikethroughBoundaryFlanking(parent *frame, child parser.SemanticEvent, delimiter string) {
	if parent == nil || child.Range.Start <= 0 || child.Range.Start > len(r.source) ||
		r.source[child.Range.Start-1] != '\t' || delimiter == "" {
		return
	}
	const escapedTab = "&#9;"
	if len(parent.inline) < len(escapedTab) || string(parent.inline[len(parent.inline)-len(escapedTab):]) != escapedTab {
		return
	}
	sourceRun := parser.Range{Start: child.Range.Start, End: child.Range.Start + len(delimiter)}
	if !sourceRun.Valid(len(r.source)) {
		return
	}
	following := delimiterFollowingRuneBytes(r.source, sourceRun.End)
	candidateRun := parser.Range{Start: len(parent.inline), End: len(parent.inline) + len(delimiter)}
	candidate := make([]byte, candidateRun.End+len(following))
	copy(candidate, parent.inline)
	copy(candidate[candidateRun.Start:candidateRun.End], delimiter)
	copy(candidate[candidateRun.End:], following)
	if !delimiterRangeFlankingChanged(r.source, sourceRun, candidate, candidateRun, '~') {
		return
	}
	parent.inline = append(parent.inline[:len(parent.inline)-len(escapedTab)], '\t')
}

func delimiterFollowingRuneBytes(source []byte, position int) []byte {
	if position < 0 || position >= len(source) {
		return nil
	}
	_, size := utf8.DecodeRune(source[position:])
	if size <= 0 {
		return nil
	}
	return source[position:min(position+size, len(source))]
}

func (r *renderer) sourceStrikethroughDelimiter(event parser.SemanticEvent) (string, bool) {
	if !event.Range.Valid(len(r.source)) || !event.ContentRange.Valid(len(r.source)) ||
		event.ContentRange.Start < event.Range.Start || event.ContentRange.End > event.Range.End {
		return "", false
	}
	openingWidth := event.ContentRange.Start - event.Range.Start
	closingWidth := event.Range.End - event.ContentRange.End
	if openingWidth != closingWidth || openingWidth < 1 || openingWidth > 2 {
		return "", false
	}
	for index := 0; index < openingWidth; index++ {
		if r.source[event.Range.Start+index] != '~' || r.source[event.Range.End-openingWidth+index] != '~' {
			return "", false
		}
	}
	return strings.Repeat("~", openingWidth), true
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
