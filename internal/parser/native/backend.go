package native

import (
	"bytes"
	"cmp"
	"fmt"
	"slices"

	"github.com/zoster81/marksplice/internal/parser"
)

// Backend implements Marksplice's frozen parser.Backend contract with the native parser.
// Production selection remains outside this package until the M115 cutover.
type Backend struct{}

// New returns a stateless native parser backend.
func New() *Backend {
	return &Backend{}
}

// ParseDocument returns parser-independent observations from one native block/inline pass.
func (*Backend) ParseDocument(source []byte) (parser.DocumentObservations, error) {
	blocks := parseBlockLines(source, physicalLines(source), true)
	analyses := analyzeInlineBlocks(source, blocks.inlines, blocks.references)
	completeBackendBlockFacts(source, blocks.nodes, analyses)
	inline := mergeInlineAnalyses(analyses, len(blocks.nodes))
	mathExpressions := nativeMathExpressionObservations(source, blocks, analyses, inline.nodes)
	nodes := removeNativeMathGFMConflicts(mergeDocumentNodes(blocks.nodes, inline.nodes), mathExpressions)
	footnoteDefinitions, footnoteReferences, footnoteUsages := nativeFootnoteObservations(source, blocks)
	nodes, usages, unresolved := reconcileNativeFootnotes(
		source,
		nodes,
		inline.usages,
		inline.unresolved,
		footnoteDefinitions,
		footnoteUsages,
	)
	details, err := attachNodeDetails(nodes, sparseNodeDetails{
		blockquotes: blocks.blockquoteDetails,
		fencedCode:  blocks.fencedCodeDetails,
		tables:      blocks.tableDetails,
		tableRows:   blocks.tableRowDetails,
		tableCells:  blocks.tableCellDetails,
	})
	if err != nil {
		return parser.DocumentObservations{}, fmt.Errorf("attach sparse node details: %w", err)
	}
	return parser.DocumentObservations{
		Nodes:                     nonNilNodes(nodes),
		BlockquoteDetails:         details.blockquotes,
		FencedCodeDetails:         details.fencedCode,
		TableDetails:              details.tables,
		TableRowDetails:           details.tableRows,
		TableCellDetails:          details.tableCells,
		LinkUsages:                nonNilLinkUsages(usages),
		UnresolvedReferenceUsages: nonNilUnresolvedReferences(unresolved),
		FootnoteDefinitions:       footnoteDefinitions,
		FootnoteReferences:        footnoteReferences,
		MathExpressions:           mathExpressions,
	}, nil
}

func nonNilNodes(nodes []parser.Node) []parser.Node {
	if nodes == nil {
		return []parser.Node{}
	}
	return nodes
}

type sparseNodeDetails struct {
	blockquotes []parser.BlockquoteDetail
	fencedCode  []parser.FencedCodeDetail
	tables      []parser.TableDetail
	tableRows   []parser.TableRowDetail
	tableCells  []parser.TableCellDetail
}

type sparseDetailCursor struct {
	read  int
	write int
}

type sparseDetailCursors struct {
	blockquotes sparseDetailCursor
	fencedCode  sparseDetailCursor
	tables      sparseDetailCursor
	tableRows   sparseDetailCursor
	tableCells  sparseDetailCursor
}

func attachNodeDetails(nodes []parser.Node, details sparseNodeDetails) (sparseNodeDetails, error) {
	cursors := sparseDetailCursors{}
	for index := range nodes {
		node := &nodes[index]
		if node.DetailIndex != 0 {
			return sparseNodeDetails{}, fmt.Errorf("node %d arrived with preassigned sparse detail index", index)
		}
		var err error
		switch node.Kind {
		case parser.KindBlockquote:
			if !node.TopLevel {
				continue
			}
			err = attachAnchoredSparseDetail(node, index, details.blockquotes, &cursors.blockquotes, node.Range.Start, blockquoteDetailAnchor, "blockquote")
		case parser.KindFencedCode:
			err = attachAnchoredSparseDetail(node, index, details.fencedCode, &cursors.fencedCode, node.Anchor, fencedCodeDetailAnchor, "fenced-code")
		case parser.KindTable:
			err = attachAnchoredSparseDetail(node, index, details.tables, &cursors.tables, node.Range.Start, tableDetailAnchor, "table")
		case parser.KindTableRow:
			err = attachAnchoredSparseDetail(node, index, details.tableRows, &cursors.tableRows, node.Range.Start, tableRowDetailAnchor, "table-row")
		case parser.KindTableCell:
			err = attachRangedSparseDetail(node, index, details.tableCells, &cursors.tableCells)
		}
		if err != nil {
			return sparseNodeDetails{}, err
		}
	}
	details.blockquotes = compactSparseDetails(details.blockquotes, cursors.blockquotes.write)
	details.fencedCode = compactSparseDetails(details.fencedCode, cursors.fencedCode.write)
	details.tables = compactSparseDetails(details.tables, cursors.tables.write)
	details.tableRows = compactSparseDetails(details.tableRows, cursors.tableRows.write)
	details.tableCells = compactSparseDetails(details.tableCells, cursors.tableCells.write)
	return details, nil
}

func attachAnchoredSparseDetail[T any](node *parser.Node, nodeIndex int, details []T, cursor *sparseDetailCursor, nodeAnchor int, detailAnchor func(T) int, description string) error {
	for cursor.read < len(details) && detailAnchor(details[cursor.read]) < nodeAnchor {
		cursor.read++
	}
	if cursor.read >= len(details) || detailAnchor(details[cursor.read]) != nodeAnchor {
		return fmt.Errorf("node %d %s detail is missing", nodeIndex, description)
	}
	details[cursor.write] = details[cursor.read]
	cursor.write++
	cursor.read++
	if !assignNodeDetailIndex(node, cursor.write) {
		return fmt.Errorf("node %d %s detail index exceeds uint32", nodeIndex, description)
	}
	return nil
}

func attachRangedSparseDetail(node *parser.Node, nodeIndex int, details []parser.TableCellDetail, cursor *sparseDetailCursor) error {
	for cursor.read < len(details) && parserRangeLess(details[cursor.read].Range, node.Range) {
		cursor.read++
	}
	if cursor.read >= len(details) || details[cursor.read].Range != node.Range {
		return fmt.Errorf("node %d table-cell detail is missing", nodeIndex)
	}
	details[cursor.write] = details[cursor.read]
	cursor.write++
	cursor.read++
	if !assignNodeDetailIndex(node, cursor.write) {
		return fmt.Errorf("node %d table-cell detail index exceeds uint32", nodeIndex)
	}
	return nil
}

func blockquoteDetailAnchor(detail parser.BlockquoteDetail) int { return detail.Anchor }
func fencedCodeDetailAnchor(detail parser.FencedCodeDetail) int { return detail.Anchor }
func tableDetailAnchor(detail parser.TableDetail) int           { return detail.Anchor }
func tableRowDetailAnchor(detail parser.TableRowDetail) int     { return detail.RowAnchor }

func assignNodeDetailIndex(node *parser.Node, count int) bool {
	if count <= 0 || uint64(count) > uint64(^uint32(0)) {
		return false
	}
	node.DetailIndex = uint32(count)
	return true
}

func parserRangeLess(left, right parser.Range) bool {
	if left.Start != right.Start {
		return left.Start < right.Start
	}
	return left.End < right.End
}

func compactSparseDetails[T any](details []T, count int) []T {
	clear(details[count:])
	details = details[:count]
	if details == nil {
		return []T{}
	}
	return details
}

func nonNilLinkUsages(usages []parser.LinkUsage) []parser.LinkUsage {
	if usages == nil {
		return []parser.LinkUsage{}
	}
	return usages
}

func nonNilUnresolvedReferences(usages []parser.UnresolvedReferenceUsage) []parser.UnresolvedReferenceUsage {
	if usages == nil {
		return []parser.UnresolvedReferenceUsage{}
	}
	return usages
}

func mergeDocumentNodes(blocks, inlines []parser.Node) []parser.Node {
	if !documentNodesOrdered(blocks) || !documentNodesOrdered(inlines) {
		return copyAndSortDocumentNodes(blocks, inlines)
	}
	total := len(blocks) + len(inlines)
	if total == 0 {
		return []parser.Node{}
	}
	if len(blocks) == 0 {
		return inlines
	}
	if len(inlines) == 0 {
		return blocks
	}
	var result []parser.Node
	switch {
	case cap(inlines) >= total:
		result = inlines[:total]
	case cap(blocks) >= total:
		result = blocks[:total]
	default:
		result = make([]parser.Node, total)
	}
	return mergeOrderedDocumentNodes(result, blocks, inlines)
}

func documentNodesOrdered(nodes []parser.Node) bool {
	for index := 1; index < len(nodes); index++ {
		if documentNodeLess(nodes[index], nodes[index-1]) {
			return false
		}
	}
	return true
}

func mergeOrderedDocumentNodes(result, blocks, inlines []parser.Node) []parser.Node {
	blockIndex, inlineIndex := len(blocks)-1, len(inlines)-1
	write := len(result) - 1
	for blockIndex >= 0 && inlineIndex >= 0 {
		if !documentNodeLess(inlines[inlineIndex], blocks[blockIndex]) {
			result[write] = inlines[inlineIndex]
			inlineIndex--
		} else {
			result[write] = blocks[blockIndex]
			blockIndex--
		}
		write--
	}
	if blockIndex >= 0 {
		copy(result[:write+1], blocks[:blockIndex+1])
	} else if inlineIndex >= 0 {
		copy(result[:write+1], inlines[:inlineIndex+1])
	}
	return result
}

func copyAndSortDocumentNodes(blocks, inlines []parser.Node) []parser.Node {
	result := make([]parser.Node, 0, len(blocks)+len(inlines))
	result = append(result, blocks...)
	result = append(result, inlines...)
	if len(result) > 1 {
		slices.SortStableFunc(result, func(left, right parser.Node) int {
			if documentNodeLess(left, right) {
				return -1
			}
			if documentNodeLess(right, left) {
				return 1
			}
			return 0
		})
	}
	return result
}

func documentNodeLess(left, right parser.Node) bool {
	leftStart := documentNodeStart(left)
	rightStart := documentNodeStart(right)
	if leftStart != rightStart {
		return leftStart < rightStart
	}
	leftBlock := documentBlockKind(left.Kind)
	rightBlock := documentBlockKind(right.Kind)
	if leftBlock != rightBlock {
		return leftBlock
	}
	return left.Range.End > right.Range.End
}

func documentNodeStart(node parser.Node) int {
	if node.Anchor > 0 || node.Range.Start == 0 {
		switch node.Kind {
		case parser.KindAutoLink, parser.KindCodeSpan, parser.KindEmphasis, parser.KindStrong, parser.KindInlineLink, parser.KindImage:
			return node.Anchor
		}
	}
	return node.Range.Start
}

func documentBlockKind(kind parser.Kind) bool {
	switch kind {
	case parser.KindParagraph, parser.KindHeading, parser.KindTask, parser.KindListItem,
		parser.KindTableCell, parser.KindFencedCode, parser.KindReferenceDefinition,
		parser.KindHTMLBlock, parser.KindTableRow, parser.KindThematicBreak,
		parser.KindBlockquote, parser.KindTable:
		return true
	default:
		return false
	}
}

// ResolveConstructionReference resolves one normalized GFM construction reference
// and fails closed when the normalized key is absent or ambiguous.
func (*Backend) ResolveConstructionReference(label string, definitions []parser.ConstructionReferenceDefinition) (parser.ConstructionReferenceDefinition, error) {
	key := ReferenceLabelKey(label)
	var match parser.ConstructionReferenceDefinition
	count := 0
	for _, definition := range definitions {
		if ReferenceLabelKey(definition.Label) != key {
			continue
		}
		match = definition
		count++
	}
	if count != 1 {
		return parser.ConstructionReferenceDefinition{}, fmt.Errorf("reference label %q must match exactly one normalized definition", label)
	}
	return match, nil
}

// ReferenceLabelKey returns the native GFM reference-label normalization key.
func (*Backend) ReferenceLabelKey(label string) string {
	return ReferenceLabelKey(label)
}

// ValidateNestedBlockquoteParagraph proves an exact nested blockquote hierarchy
// ending in one paragraph with the requested physical content-line anchors.
func (*Backend) ValidateNestedBlockquoteParagraph(source []byte, outer parser.Range, contentLines []parser.Range, depth int) error {
	if depth < 1 || !outer.Valid(len(source)) || outer.Start == outer.End || len(contentLines) == 0 {
		return fmt.Errorf("invalid blockquote construction ranges")
	}
	inner, err := exactNestedBlockquoteLines(source, outer, depth)
	if err != nil {
		return err
	}
	parsed := parseBlockLines(source, inner, false)
	if len(parsed.roots) != 1 || parsed.roots[0].kind != rootBlockParagraph || len(inner) != len(contentLines) {
		return fmt.Errorf("expected exact nested blockquote paragraph hierarchy")
	}
	for index, want := range contentLines {
		if !want.Valid(len(source)) || inner[index].start != want.Start {
			return fmt.Errorf("blockquote paragraph line %d start changed", index)
		}
	}
	if inner[len(inner)-1].end != contentLines[len(contentLines)-1].End {
		return fmt.Errorf("blockquote paragraph final line end changed")
	}
	if contentLines[len(contentLines)-1].End != outer.End {
		return fmt.Errorf("blockquote paragraph outer end changed")
	}
	return nil
}

// ValidateNestedBlockquoteBlocks proves exact nested blockquote ownership and that
// the innermost canonical child source is byte-identical to the reviewed fragment.
func (*Backend) ValidateNestedBlockquoteBlocks(source []byte, outer parser.Range, innerSource []byte, depth int) error {
	if depth < 1 || !outer.Valid(len(source)) || outer.Start == outer.End || len(innerSource) == 0 || innerSource[len(innerSource)-1] != '\n' {
		return fmt.Errorf("invalid blockquote block construction input")
	}
	inner, err := exactNestedBlockquoteLines(source, outer, depth)
	if err != nil {
		return err
	}
	dequoted := materializeNestedLines(source, inner)
	if !bytes.Equal(dequoted, innerSource) {
		return fmt.Errorf("blockquote child sequence changed")
	}
	actual := parseBlockLines(source, inner, false)
	expected := parseBlockLines(innerSource, physicalLines(innerSource), true)
	if len(actual.roots) != len(expected.roots) || actual.blankBetweenRoots != expected.blankBetweenRoots {
		return fmt.Errorf("blockquote child sequence changed")
	}
	return nil
}

func exactNestedBlockquoteLines(source []byte, outer parser.Range, depth int) ([]physicalLine, error) {
	lines := physicalLines(source)
	start := blockquoteOuterLineIndex(source, lines, outer.Start)
	if start < 0 {
		return nil, fmt.Errorf("expected exact nested blockquote hierarchy")
	}
	end := blockquoteOuterLineEnd(lines, start, outer.End)
	if end < 0 {
		return nil, fmt.Errorf("expected exact nested blockquote hierarchy")
	}
	current := append([]physicalLine(nil), lines[start:end]...)
	for level := 0; level < depth; level++ {
		quoted := parseBlockquote(source, current, 0)
		if quoted.next != len(current) || len(quoted.content) == 0 {
			return nil, fmt.Errorf("expected exact nested blockquote hierarchy")
		}
		current = quoted.content
	}
	return current, nil
}

func blockquoteOuterLineIndex(source []byte, lines []physicalLine, anchor int) int {
	for index, line := range lines {
		parsedAnchor, _, ok := parseBlockquoteOpening(source, line)
		if ok && parsedAnchor == anchor {
			return index
		}
	}
	return -1
}

func blockquoteOuterLineEnd(lines []physicalLine, start, outerEnd int) int {
	for index := start; index < len(lines); index++ {
		if lines[index].end == outerEnd {
			return index + 1
		}
		if lines[index].end > outerEnd {
			break
		}
	}
	return -1
}

func materializeNestedLines(source []byte, lines []physicalLine) []byte {
	result := make([]byte, 0)
	for _, line := range lines {
		result = append(result, source[line.start:line.end]...)
		if line.next > line.end {
			result = append(result, source[line.end:line.next]...)
		}
	}
	return result
}

type constructionSemantic struct {
	kind        parser.Kind
	syntax      parser.Range
	content     parser.Range
	form        parser.LinkUsageForm
	reference   string
	destination string
	title       string
	hasTitle    bool
	parent      int
}

type constructionSemanticKey struct {
	kind   parser.Kind
	syntax parser.Range
}

type constructionReferenceKey struct {
	kind   parser.Kind
	syntax parser.Range
	form   parser.LinkUsageForm
}

type constructionState struct {
	semantics      []constructionSemantic
	bySyntax       map[constructionSemanticKey]int
	byReference    map[constructionReferenceKey]int
	childCounts    []int
	referenceCount int
}

const constructionAmbiguousIndex = -2

func parseConstructionState(source []byte, definitions []referenceDefinitionParse) (constructionState, error) {
	block, err := constructionInlineBlock(source)
	if err != nil {
		return constructionState{}, err
	}
	runs := collectBacktickRuns(source, block)
	nextSame := nextSameLengthRuns(runs)
	spans := collectInlineSpans(source, block)
	owners, barriers := resolvePrimaryInlineOwners(source, block, runs, nextSame, spans)
	delimiters := parseDelimiterObservations(source, block, owners, barriers, definitions)
	semantics := collectConstructionSemantics(owners, delimiters)
	state := constructionState{
		semantics:   semantics,
		bySyntax:    make(map[constructionSemanticKey]int, len(semantics)),
		byReference: make(map[constructionReferenceKey]int, len(semantics)),
		childCounts: assignConstructionParents(semantics),
	}
	for index, semantic := range semantics {
		addConstructionIndex(state.bySyntax, constructionSemanticKey{kind: semantic.kind, syntax: semantic.syntax}, index)
		if semantic.form == parser.LinkUsageDirect || semantic.kind != parser.KindInlineLink && semantic.kind != parser.KindImage {
			continue
		}
		state.referenceCount++
		addConstructionIndex(state.byReference, constructionReferenceKey{kind: semantic.kind, syntax: semantic.syntax, form: semantic.form}, index)
	}
	return state, nil
}

func addConstructionIndex[K comparable](index map[K]int, key K, value int) {
	if _, exists := index[key]; exists {
		index[key] = constructionAmbiguousIndex
		return
	}
	index[key] = value
}

func (state constructionState) semanticIndex(kind parser.Kind, syntax parser.Range) int {
	index, ok := state.bySyntax[constructionSemanticKey{kind: kind, syntax: syntax}]
	if !ok || index < 0 {
		return -1
	}
	return index
}

func (state constructionState) referenceIndex(want parser.ConstructionReferenceInlineExpectation) int {
	form := parserReferenceUsageForm(want.Form)
	if form == parser.LinkUsageUnknown {
		return -1
	}
	index, ok := state.byReference[constructionReferenceKey{kind: want.Kind, syntax: want.SyntaxRange, form: form}]
	if !ok || index < 0 {
		return -1
	}
	return index
}

func constructionInlineBlock(source []byte) (inlineBlock, error) {
	parsed := parseBlockLines(source, physicalLines(source), true)
	if len(parsed.roots) != 1 || parsed.roots[0].kind != rootBlockParagraph || len(parsed.inlines) != 1 || len(parsed.inlines[0].segments) != 1 {
		return inlineBlock{}, fmt.Errorf("construction inline proof is not contained by one paragraph")
	}
	segment := parsed.inlines[0].segments[0]
	if segment.Start != 0 || segment.End != len(source) {
		return inlineBlock{}, fmt.Errorf("construction inline proof paragraph range changed")
	}
	return parsed.inlines[0], nil
}

func collectConstructionSemantics(owners []inlineSpan, delimiters delimiterParseResult) []constructionSemantic {
	result := make([]constructionSemantic, 0, len(owners)+len(delimiters.composites)+len(delimiters.matches))
	for _, owner := range owners {
		switch owner.kind {
		case parser.KindCodeSpan, parser.KindRawHTML, parser.KindAutoLink:
			result = append(result, constructionSemantic{
				kind:    owner.kind,
				syntax:  parser.Range{Start: owner.start, End: owner.end},
				content: owner.content,
				parent:  -1,
			})
		}
	}
	for _, composite := range delimiters.composites {
		if !composite.active {
			continue
		}
		result = append(result, constructionSemantic{
			kind:        composite.kind,
			syntax:      parser.Range{Start: composite.start, End: composite.end},
			content:     composite.label,
			form:        composite.form,
			reference:   composite.reference,
			destination: composite.destination,
			title:       composite.title,
			hasTitle:    composite.hasTitle,
			parent:      -1,
		})
	}
	for _, match := range delimiters.matches {
		kind := constructionDelimiterKind(match)
		if kind == parser.KindUnknown {
			continue
		}
		result = append(result, constructionSemantic{
			kind:    kind,
			syntax:  parser.Range{Start: match.syntaxStart, End: match.syntaxEnd},
			content: match.content,
			parent:  -1,
		})
	}
	if len(result) > 1 {
		slices.SortStableFunc(result, func(left, right constructionSemantic) int {
			if order := cmp.Compare(left.syntax.Start, right.syntax.Start); order != 0 {
				return order
			}
			return cmp.Compare(right.syntax.End, left.syntax.End)
		})
	}
	return result
}

func constructionDelimiterKind(match delimiterMatch) parser.Kind {
	switch match.marker {
	case '*', '_':
		if match.level == 2 {
			return parser.KindStrong
		}
		return parser.KindEmphasis
	case '~':
		return parser.KindStrikethrough
	default:
		return parser.KindUnknown
	}
}

func assignConstructionParents(nodes []constructionSemantic) []int {
	childCounts := make([]int, len(nodes))
	stack := make([]int, 0, 8)
	for child := range nodes {
		for len(stack) != 0 && !constructionContains(nodes[stack[len(stack)-1]], nodes[child]) {
			stack = stack[:len(stack)-1]
		}
		nodes[child].parent = -1
		if len(stack) != 0 {
			parent := stack[len(stack)-1]
			nodes[child].parent = parent
			childCounts[parent]++
		}
		if constructionCanContain(nodes[child]) {
			stack = append(stack, child)
		}
	}
	return childCounts
}

func constructionCanContain(node constructionSemantic) bool {
	if node.content.Start >= node.content.End {
		return false
	}
	switch node.kind {
	case parser.KindEmphasis, parser.KindStrong, parser.KindStrikethrough, parser.KindInlineLink, parser.KindImage:
		return true
	default:
		return false
	}
}

func constructionContains(parent, child constructionSemantic) bool {
	return constructionCanContain(parent) && child.syntax.Start >= parent.content.Start && child.syntax.End <= parent.content.End
}

// ValidateConstructionInlineHierarchy proves typed inline kinds, exact syntax/content
// ranges, and direct semantic parent/child relationships against the native parser.
func (*Backend) ValidateConstructionInlineHierarchy(source []byte, expected []parser.ConstructionInlineExpectation, references []parser.ConstructionReferenceInlineExpectation) error {
	if len(expected) == 0 {
		return nil
	}
	for index, want := range expected {
		if err := validateNativeInlineExpectationInput(source, want, index); err != nil {
			return err
		}
	}
	definitions, err := constructionReferenceDefinitions(references)
	if err != nil {
		return err
	}
	state, err := parseConstructionState(source, definitions)
	if err != nil {
		return err
	}
	matched, actualToExpected, err := matchNativeInlineSemantics(state, expected)
	if err != nil {
		return err
	}
	if err := validateNativeInlineParents(state, expected, matched); err != nil {
		return err
	}
	return validateNativeInlineChildSets(state, expected, actualToExpected)
}

func matchNativeInlineSemantics(state constructionState, expected []parser.ConstructionInlineExpectation) ([]int, map[int]int, error) {
	matched := make([]int, len(expected))
	actualToExpected := make(map[int]int, len(expected))
	for index, want := range expected {
		actual := state.semanticIndex(want.Kind, want.SyntaxRange)
		if actual < 0 || state.semantics[actual].content != want.ContentRange {
			return nil, nil, fmt.Errorf("typed inline hierarchy node %d changed", index)
		}
		if err := validateNativeInlineDelimiter(want); err != nil {
			return nil, nil, fmt.Errorf("typed inline hierarchy node %d: %w", index, err)
		}
		matched[index] = actual
		actualToExpected[actual] = index
	}
	return matched, actualToExpected, nil
}

func validateNativeInlineParents(state constructionState, expected []parser.ConstructionInlineExpectation, matched []int) error {
	for index, want := range expected {
		actualParent := state.semantics[matched[index]].parent
		if want.Parent < 0 && actualParent < 0 {
			continue
		}
		if want.Parent < 0 || actualParent != matched[want.Parent] {
			return fmt.Errorf("typed inline hierarchy parent %d changed", index)
		}
	}
	return nil
}

func validateNativeInlineExpectationInput(source []byte, want parser.ConstructionInlineExpectation, index int) error {
	if want.Kind == parser.KindInlineLink || want.Kind == parser.KindImage {
		return validateNativeInlineOwnerInput(source, want, index)
	}
	if !want.SyntaxRange.Valid(len(source)) || !want.ContentRange.Valid(len(source)) || want.ContentRange.Start == want.ContentRange.End ||
		want.DelimiterLength < 1 || want.SyntaxRange.Start+want.DelimiterLength != want.ContentRange.Start ||
		want.ContentRange.End+want.DelimiterLength != want.SyntaxRange.End || want.Parent < -1 || want.Parent >= index {
		return fmt.Errorf("invalid typed inline hierarchy expectation %d", index)
	}
	if want.Marker == 0 {
		return fmt.Errorf("typed inline hierarchy marker %d is empty", index)
	}
	for offset := 0; offset < want.DelimiterLength; offset++ {
		if source[want.SyntaxRange.Start+offset] != want.Marker || source[want.ContentRange.End+offset] != want.Marker {
			return fmt.Errorf("typed inline hierarchy delimiter %d changed", index)
		}
	}
	return nil
}

func validateNativeInlineOwnerInput(source []byte, want parser.ConstructionInlineExpectation, index int) error {
	if !want.SyntaxRange.Valid(len(source)) || !want.ContentRange.Valid(len(source)) || want.ContentRange.Start == want.ContentRange.End ||
		want.SyntaxRange.Start == want.SyntaxRange.End || want.Parent != -1 || want.Marker != 0 || want.DelimiterLength != 0 {
		return fmt.Errorf("invalid typed inline owner expectation %d", index)
	}
	prefix := 1
	if want.Kind == parser.KindImage {
		prefix = 2
		if source[want.SyntaxRange.Start] != '!' {
			return fmt.Errorf("typed inline owner image prefix %d changed", index)
		}
	}
	if want.ContentRange.Start != want.SyntaxRange.Start+prefix || source[want.ContentRange.Start-1] != '[' ||
		want.ContentRange.End >= want.SyntaxRange.End || source[want.ContentRange.End] != ']' {
		return fmt.Errorf("typed inline owner boundary %d changed", index)
	}
	return nil
}

func validateNativeInlineDelimiter(want parser.ConstructionInlineExpectation) error {
	switch want.Kind {
	case parser.KindCodeSpan:
		if want.Marker != '`' {
			return fmt.Errorf("code-span shape changed")
		}
	case parser.KindEmphasis:
		if want.DelimiterLength != 1 || want.Marker != '*' && want.Marker != '_' {
			return fmt.Errorf("emphasis delimiter changed")
		}
	case parser.KindStrong:
		if want.DelimiterLength != 2 || want.Marker != '*' && want.Marker != '_' {
			return fmt.Errorf("strong delimiter changed")
		}
	case parser.KindStrikethrough:
		if want.DelimiterLength != 2 || want.Marker != '~' {
			return fmt.Errorf("strikethrough delimiter changed")
		}
	case parser.KindInlineLink, parser.KindImage:
		return nil
	default:
		return fmt.Errorf("unsupported typed inline hierarchy kind %d", want.Kind)
	}
	return nil
}

func validateNativeInlineChildSets(state constructionState, expected []parser.ConstructionInlineExpectation, actualToExpected map[int]int) error {
	wantChildren := make([]int, len(expected))
	for _, want := range expected {
		if want.Parent >= 0 {
			wantChildren[want.Parent]++
		}
	}
	gotChildren := make([]int, len(expected))
	for actualIndex, node := range state.semantics {
		if node.parent < 0 {
			if _, expectedNode := actualToExpected[actualIndex]; expectedNode {
				continue
			}
			if constructionNestedProofKind(node.kind) {
				return fmt.Errorf("typed inline hierarchy root gained unexpected child")
			}
			continue
		}
		parentExpected, parentTracked := actualToExpected[node.parent]
		if !parentTracked {
			continue
		}
		if _, childTracked := actualToExpected[actualIndex]; !childTracked {
			return fmt.Errorf("typed inline hierarchy parent %d gained unsupported child", parentExpected)
		}
		gotChildren[parentExpected]++
	}
	for index, want := range expected {
		if gotChildren[index] != wantChildren[index] || want.Parent >= index {
			return fmt.Errorf("typed inline hierarchy child count %d changed", index)
		}
	}
	return nil
}

func constructionNestedProofKind(kind parser.Kind) bool {
	switch kind {
	case parser.KindCodeSpan, parser.KindEmphasis, parser.KindStrong, parser.KindStrikethrough:
		return true
	default:
		return false
	}
}

// ValidateConstructionLinkImages proves direct link/image syntax and resolved semantics.
func (*Backend) ValidateConstructionLinkImages(source []byte, expected []parser.ConstructionLinkImageExpectation) error {
	if len(expected) == 0 {
		return nil
	}
	for index, want := range expected {
		if err := validateNativeLinkImageInput(source, want, index); err != nil {
			return err
		}
	}
	state, err := parseConstructionState(source, nil)
	if err != nil {
		return err
	}
	for index, want := range expected {
		actual := state.semanticIndex(want.Kind, want.SyntaxRange)
		if actual < 0 || state.semantics[actual].form != parser.LinkUsageDirect {
			return fmt.Errorf("construction link/image %d changed", index)
		}
		semantic := state.semantics[actual]
		if semantic.content != want.LabelRange || semantic.destination != want.Destination || semantic.title != want.Title || semantic.hasTitle != want.HasTitle {
			return fmt.Errorf("construction link/image %d: destination or title changed", index)
		}
	}
	return nil
}

func validateNativeLinkImageInput(source []byte, want parser.ConstructionLinkImageExpectation, index int) error {
	prefix, ok := nativeLinkImagePrefixLength(source, want.Kind, want.SyntaxRange.Start)
	if !ok || !want.SyntaxRange.Valid(len(source)) || !want.LabelRange.Valid(len(source)) ||
		want.SyntaxRange.Start == want.SyntaxRange.End || want.LabelRange.Start == want.LabelRange.End || want.Destination == "" ||
		want.LabelRange.Start != want.SyntaxRange.Start+prefix || want.HasTitle != (want.Title != "") {
		return fmt.Errorf("invalid construction link/image expectation %d", index)
	}
	suffix := nativeLinkImageSuffix(want)
	if want.LabelRange.End+len(suffix) != want.SyntaxRange.End || !bytes.Equal(source[want.LabelRange.End:want.SyntaxRange.End], suffix) {
		return fmt.Errorf("construction link/image syntax %d changed", index)
	}
	return nil
}

func nativeLinkImagePrefixLength(source []byte, kind parser.Kind, start int) (int, bool) {
	if start < 0 || start >= len(source) {
		return 0, false
	}
	switch kind {
	case parser.KindInlineLink:
		return 1, source[start] == '['
	case parser.KindImage:
		return 2, start+1 < len(source) && source[start] == '!' && source[start+1] == '['
	default:
		return 0, false
	}
}

func nativeLinkImageSuffix(want parser.ConstructionLinkImageExpectation) []byte {
	suffix := "](<" + want.Destination + ">"
	if want.HasTitle {
		suffix += " \"" + want.Title + "\""
	}
	return []byte(suffix + ")")
}

// ValidateConstructionReferenceInlines proves exact reference source forms and
// their resolved destination/title semantics against the native parser.
func (*Backend) ValidateConstructionReferenceInlines(source []byte, expected []parser.ConstructionReferenceInlineExpectation) error {
	if len(expected) == 0 {
		return nil
	}
	for index, want := range expected {
		if err := validateNativeReferenceInlineInput(source, want); err != nil {
			return fmt.Errorf("reference inline %d: %w", index, err)
		}
	}
	definitions, err := constructionReferenceDefinitions(expected)
	if err != nil {
		return err
	}
	state, err := parseConstructionState(source, definitions)
	if err != nil {
		return err
	}
	if state.referenceCount != len(expected) {
		return fmt.Errorf("reference inline semantic count changed")
	}
	for index, want := range expected {
		actual := state.referenceIndex(want)
		if actual < 0 {
			return fmt.Errorf("reference inline %d semantic node changed", index)
		}
		if err := validateNativeReferenceSemantic(source, actual, want, state); err != nil {
			return fmt.Errorf("reference inline %d: %w", index, err)
		}
	}
	return nil
}

func constructionReferenceDefinitions(expected []parser.ConstructionReferenceInlineExpectation) ([]referenceDefinitionParse, error) {
	byRaw := make(map[string]parser.ConstructionReferenceInlineExpectation, len(expected))
	order := make([]string, 0, len(expected))
	for _, want := range expected {
		if previous, ok := byRaw[want.Reference]; ok {
			if previous.Destination != want.Destination || previous.Title != want.Title || previous.HasTitle != want.HasTitle {
				return nil, fmt.Errorf("reference %q resolves inconsistently", want.Reference)
			}
			continue
		}
		byRaw[want.Reference] = want
		order = append(order, want.Reference)
	}
	definitions := make([]referenceDefinitionParse, 0, len(order))
	for _, reference := range order {
		want := byRaw[reference]
		definitions = append(definitions, referenceDefinitionParse{
			label: reference, destination: want.Destination, title: want.Title,
			hasTitle: want.HasTitle, projectable: true,
		})
	}
	return definitions, nil
}

func validateNativeReferenceInlineInput(source []byte, want parser.ConstructionReferenceInlineExpectation) error {
	prefix, ok := nativeLinkImagePrefixLength(source, want.Kind, want.SyntaxRange.Start)
	if !ok {
		return fmt.Errorf("reference inline prefix changed")
	}
	if !want.SyntaxRange.Valid(len(source)) || !want.LabelRange.Valid(len(source)) || !want.ReferenceRange.Valid(len(source)) ||
		want.SyntaxRange.Start == want.SyntaxRange.End || want.LabelRange.Start == want.LabelRange.End || want.Reference == "" ||
		want.LabelRange.Start != want.SyntaxRange.Start+prefix {
		return fmt.Errorf("invalid reference inline ranges")
	}
	if want.LabelRange.End >= len(source) || source[want.LabelRange.End] != ']' {
		return fmt.Errorf("reference inline label closing changed")
	}
	return validateNativeReferenceForm(source, want)
}

func validateNativeReferenceForm(source []byte, want parser.ConstructionReferenceInlineExpectation) error {
	switch want.Form {
	case parser.ConstructionReferenceInlineFull:
		return validateNativeFullReferenceForm(source, want)
	case parser.ConstructionReferenceInlineCollapsed:
		return validateNativeCollapsedReferenceForm(source, want)
	case parser.ConstructionReferenceInlineShortcut:
		return validateNativeShortcutReferenceForm(source, want)
	default:
		return fmt.Errorf("unsupported reference inline form %d", want.Form)
	}
}

func validateNativeFullReferenceForm(source []byte, want parser.ConstructionReferenceInlineExpectation) error {
	if want.ReferenceRange.Start != want.LabelRange.End+2 || want.ReferenceRange.Start == want.ReferenceRange.End || want.SyntaxRange.End != want.ReferenceRange.End+1 {
		return fmt.Errorf("full reference inline boundary or syntax changed")
	}
	if want.LabelRange.End+1 >= len(source) || source[want.LabelRange.End+1] != '[' || want.ReferenceRange.End >= len(source) || source[want.ReferenceRange.End] != ']' {
		return fmt.Errorf("full reference inline boundary or syntax changed")
	}
	if string(source[want.ReferenceRange.Start:want.ReferenceRange.End]) != want.Reference {
		return fmt.Errorf("full reference inline boundary or syntax changed")
	}
	return nil
}

func validateNativeCollapsedReferenceForm(source []byte, want parser.ConstructionReferenceInlineExpectation) error {
	if want.ReferenceRange.Start != want.LabelRange.End+2 || want.ReferenceRange.Start != want.ReferenceRange.End || want.SyntaxRange.End != want.LabelRange.End+3 {
		return fmt.Errorf("collapsed reference inline boundary or syntax changed")
	}
	if want.LabelRange.End+2 >= len(source) || source[want.LabelRange.End+1] != '[' || source[want.LabelRange.End+2] != ']' {
		return fmt.Errorf("collapsed reference inline boundary or syntax changed")
	}
	if string(source[want.LabelRange.Start:want.LabelRange.End]) != want.Reference {
		return fmt.Errorf("collapsed reference inline boundary or syntax changed")
	}
	return nil
}

func validateNativeShortcutReferenceForm(source []byte, want parser.ConstructionReferenceInlineExpectation) error {
	if want.ReferenceRange.Start != want.SyntaxRange.End || want.ReferenceRange.Start != want.ReferenceRange.End || want.SyntaxRange.End != want.LabelRange.End+1 {
		return fmt.Errorf("shortcut reference inline boundary or syntax changed")
	}
	if string(source[want.LabelRange.Start:want.LabelRange.End]) != want.Reference {
		return fmt.Errorf("shortcut reference inline boundary or syntax changed")
	}
	return nil
}

func parserReferenceUsageForm(form parser.ConstructionReferenceInlineForm) parser.LinkUsageForm {
	switch form {
	case parser.ConstructionReferenceInlineFull:
		return parser.LinkUsageFull
	case parser.ConstructionReferenceInlineCollapsed:
		return parser.LinkUsageCollapsed
	case parser.ConstructionReferenceInlineShortcut:
		return parser.LinkUsageShortcut
	default:
		return parser.LinkUsageUnknown
	}
}

func validateNativeReferenceSemantic(source []byte, actualIndex int, want parser.ConstructionReferenceInlineExpectation, state constructionState) error {
	actual := state.semantics[actualIndex]
	if actual.reference != want.Reference || actual.destination != want.Destination || actual.title != want.Title || actual.hasTitle != want.HasTitle {
		return fmt.Errorf("resolved reference semantics changed")
	}
	if actual.content != want.LabelRange {
		return fmt.Errorf("reference label range changed")
	}
	if !want.StructuredLabel && state.childCounts[actualIndex] != 0 {
		return fmt.Errorf("reference label range changed")
	}
	if want.Form == parser.ConstructionReferenceInlineFull && string(source[want.ReferenceRange.Start:want.ReferenceRange.End]) != want.Reference {
		return fmt.Errorf("reference source value changed")
	}
	return nil
}

var _ parser.Backend = (*Backend)(nil)
