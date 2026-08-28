package splice

import (
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

// ComposeChanges combines already-prepared mutations from this exact snapshot.
// Constituent patch sets must not overlap, their independently validated model
// deltas must be composable, and the resulting source must pass one combined
// document-model proof.
func (d *Document) ComposeChanges(changes ...ChangeSet) (ChangeSet, error) {
	if d == nil {
		return ChangeSet{}, ErrSourceConflict
	}
	combined, err := source.ComposeChangeSets(d.source, changes...)
	if err != nil {
		return ChangeSet{}, compositionSourceError(err)
	}
	if len(changes) <= 1 {
		return combined, nil
	}

	nodeDeltas := make([]compositionDelta[compositionNodeView], 0, len(changes))
	linkDeltas := make([]compositionDelta[compositionLinkView], 0, len(changes))
	footnoteDeltas := make([]compositionDelta[compositionFootnoteReferenceView], 0, len(changes))
	originalNodes := compositionNodeViews(d)
	originalLinks := compositionLinkViews(d.linkUsages)
	originalFootnotes := compositionFootnoteReferenceViews(d)
	for _, change := range changes {
		candidate, err := change.Apply(d.source)
		if err != nil {
			return ChangeSet{}, compositionSourceError(err)
		}
		candidateDocument, err := Parse(candidate)
		if err != nil {
			return ChangeSet{}, fmt.Errorf("%w: parse independently prepared composition candidate: %v", ErrInvalidReplacement, err)
		}
		sourceStart := compositionPatchStart(change.Patches())
		nodeDeltas = append(nodeDeltas, newCompositionDeltas(originalNodes, compositionNodeViews(candidateDocument), sourceStart)...)
		linkDeltas = append(linkDeltas, newCompositionDeltas(originalLinks, compositionLinkViews(candidateDocument.linkUsages), sourceStart)...)
		footnoteDeltas = append(footnoteDeltas, newCompositionDeltas(originalFootnotes, compositionFootnoteReferenceViews(candidateDocument), sourceStart)...)
	}

	expectedNodes, ok := applyCompositionDeltas(originalNodes, nodeDeltas)
	if !ok {
		return ChangeSet{}, fmt.Errorf("%w: prepared mutations affect overlapping structural model regions", ErrInvalidReplacement)
	}
	expectedLinks, ok := applyCompositionDeltas(originalLinks, linkDeltas)
	if !ok {
		return ChangeSet{}, fmt.Errorf("%w: prepared mutations affect overlapping link relationships", ErrInvalidReplacement)
	}
	expectedFootnotes, ok := applyCompositionDeltas(originalFootnotes, footnoteDeltas)
	if !ok {
		return ChangeSet{}, fmt.Errorf("%w: prepared mutations affect overlapping footnote relationships", ErrInvalidReplacement)
	}
	candidate, err := combined.Apply(d.source)
	if err != nil {
		return ChangeSet{}, compositionSourceError(err)
	}
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ChangeSet{}, fmt.Errorf("%w: parse combined composition candidate: %v", ErrInvalidReplacement, err)
	}
	if !slices.Equal(compositionNodeViews(candidateDocument), expectedNodes) ||
		!slices.Equal(compositionLinkViews(candidateDocument.linkUsages), expectedLinks) ||
		!slices.Equal(compositionFootnoteReferenceViews(candidateDocument), expectedFootnotes) {
		return ChangeSet{}, fmt.Errorf("%w: combined candidate does not match independently validated model deltas", ErrInvalidReplacement)
	}
	return combined, nil
}

func compositionSourceError(err error) error {
	if errors.Is(err, source.ErrConflict) {
		return ErrSourceConflict
	}
	return fmt.Errorf("%w: compose prepared changes: %v", ErrInvalidReplacement, err)
}

type compositionNodeView struct {
	semantic     compositionNodeSemanticView
	sourceHash   source.Fingerprint
	ownedLength  int
	rangeStart   int
	rangeEnd     int
	contentStart int
	contentEnd   int
}

type compositionNodeSemanticView struct {
	kind                Kind
	survivor            removalSurvivorSignature
	tableAlignments     string
	listSubtreeComplete bool
	listChildCount      int
	listParent          int
	table               int
	row                 int
	previousRow         int
	nextRow             int
	blockquoteMarker    compositionRelativeRangeView
	blockquoteContent   string
	footnoteLabel       compositionRelativeRangeView
	footnoteBody        string
	mathStyle           MathExpressionStyle
	mathPayload         compositionRelativeRangeView
}

type compositionRelativeRangeView struct {
	present bool
	start   int
	end     int
}

type compositionLinkView struct {
	kind          parser.Kind
	form          parser.LinkUsageForm
	reference     string
	destination   string
	title         string
	hasTitle      bool
	autoLinkEmail bool
}

type compositionFootnoteReferenceView struct {
	label           string
	occurrence      int
	hasDefinition   bool
	definitionIndex int
}

func compositionNodeViews(document *Document) []compositionNodeView {
	if document == nil {
		return nil
	}
	indexes := document.nodeIndex
	views := make([]compositionNodeView, len(document.nodes))
	for index, node := range document.nodes {
		owned := compositionOwnedRange(document, node)
		if !owned.Valid(len(document.source)) || owned.Start > node.Range.Start || owned.End < node.Range.End {
			owned = node.Range
		}
		view := compositionNodeView{
			semantic:    compositionNodeSemantic(document, node, index, indexes, owned),
			sourceHash:  source.Sum(document.source[owned.Start:owned.End]),
			ownedLength: owned.End - owned.Start,
			rangeStart:  node.Range.Start - owned.Start,
			rangeEnd:    node.Range.End - owned.Start,
		}
		if node.ContentRange != (Range{}) {
			view.contentStart = node.ContentRange.Start - owned.Start
			view.contentEnd = node.ContentRange.End - owned.Start
		}
		views[index] = view
	}
	return views
}

func compositionOwnedRange(document *Document, node Node) Range {
	input := document.source
	switch node.Kind {
	case KindListItem:
		return node.ListItemLineRange
	case KindTableCell:
		return node.TableCellRange
	case KindTableRow:
		return node.Range
	case KindTable:
		return node.Range
	case KindReferenceDefinition:
		if mapping, ok := remapReferenceDefinitionSource(input, node); ok {
			return mapping.LineRange
		}
		return node.Range
	case KindThematicBreak:
		if mapping, ok := remapThematicBreakSource(input, node); ok {
			return mapping.LineRange
		}
		return node.Range
	case KindBlockquote:
		if mapping, ok := document.blockquoteSource(node); ok {
			return mapping.LineRange
		}
		return Range{}
	default:
		return node.Range
	}
}

func compositionNodeSemantic(document *Document, node Node, index int, indexes map[NodeID]int, owned Range) compositionNodeSemanticView {
	view := compositionNodeSemanticView{
		kind:                node.Kind,
		survivor:            removalSurvivorSemanticSignature(node),
		tableAlignments:     compositionTableAlignments(node.TableAlignments),
		listSubtreeComplete: node.ListSubtreeComplete,
		listChildCount:      node.ListChildCount,
		listParent:          relativeNodeIndex(index, node.ListParentID, indexes),
		table:               relativeNodeIndex(index, node.TableID, indexes),
		row:                 relativeNodeIndex(index, node.TableRowID, indexes),
		previousRow:         relativeNodeIndex(index, node.TablePreviousRowID, indexes),
		nextRow:             relativeNodeIndex(index, node.TableNextRowID, indexes),
	}
	if node.Kind == KindBlockquote {
		if mapping, ok := document.blockquoteSource(node); ok {
			view.blockquoteMarker = compositionRelativeRange(mapping.MarkerRange, owned)
			view.blockquoteContent = compositionRelativeRanges(mapping.ContentRanges, owned)
		}
	}
	if node.Kind == KindFootnoteDefinition {
		if mapping, ok := document.footnoteSource(node); ok {
			view.footnoteLabel = compositionRelativeRange(mapping.LabelRange, owned)
			view.footnoteBody = compositionRelativeRanges(mapping.BodyRanges, owned)
		}
	}
	if node.Kind == KindMathExpression {
		view.mathStyle = node.MathStyle
		view.mathPayload = compositionRelativeRange(node.ContentRange, owned)
	}
	return view
}

func relativeNodeIndex(current int, id NodeID, indexes map[NodeID]int) int {
	if id == "" {
		return 0
	}
	index, ok := indexes[id]
	if !ok {
		return int(^uint(0) >> 1)
	}
	return index - current
}

func compositionRelativeRange(range_, base Range) compositionRelativeRangeView {
	if range_ == (Range{}) {
		return compositionRelativeRangeView{}
	}
	return compositionRelativeRangeView{present: true, start: range_.Start - base.Start, end: range_.End - base.Start}
}

func compositionTableAlignments(alignments []TableAlignment) string {
	if len(alignments) == 0 {
		return ""
	}
	encoded := make([]byte, len(alignments))
	for index, alignment := range alignments {
		encoded[index] = byte(alignment)
	}
	return string(encoded)
}

func compositionRelativeRanges(ranges []source.Range, base Range) string {
	if len(ranges) == 0 {
		return ""
	}
	encoded := make([]byte, len(ranges)*16)
	for index, range_ := range ranges {
		offset := index * 16
		binary.LittleEndian.PutUint64(encoded[offset:offset+8], uint64(int64(range_.Start-base.Start)))
		binary.LittleEndian.PutUint64(encoded[offset+8:offset+16], uint64(int64(range_.End-base.Start)))
	}
	return string(encoded)
}

func compositionFootnoteReferenceViews(document *Document) []compositionFootnoteReferenceView {
	if document == nil {
		return nil
	}
	definitions := footnoteDefinitionIndexes(document)
	views := make([]compositionFootnoteReferenceView, len(document.footnoteReferences))
	for index, reference := range document.footnoteReferences {
		definitionIndex := -1
		if reference.HasDefinition {
			if resolved, ok := definitions[reference.DefinitionID]; ok {
				definitionIndex = resolved
			}
		}
		views[index] = compositionFootnoteReferenceView{
			label:           reference.Label,
			occurrence:      reference.Occurrence,
			hasDefinition:   reference.HasDefinition,
			definitionIndex: definitionIndex,
		}
	}
	return views
}

func compositionLinkViews(usages []parser.LinkUsage) []compositionLinkView {
	views := make([]compositionLinkView, len(usages))
	for index, usage := range usages {
		views[index] = compositionLinkView{
			kind:          usage.Kind,
			form:          usage.Form,
			reference:     usage.Reference,
			destination:   usage.Destination,
			title:         usage.Title,
			hasTitle:      usage.HasTitle,
			autoLinkEmail: usage.AutoLinkEmail,
		}
	}
	return views
}

type compositionDelta[T comparable] struct {
	start       int
	end         int
	replacement []T
	sourceStart int
}

func newCompositionDeltas[T comparable](original, candidate []T, sourceStart int) []compositionDelta[T] {
	if len(original) != len(candidate) {
		return []compositionDelta[T]{newCompositionDelta(original, candidate, sourceStart)}
	}
	result := make([]compositionDelta[T], 0)
	for index := 0; index < len(original); {
		if original[index] == candidate[index] {
			index++
			continue
		}
		start := index
		for index < len(original) && original[index] != candidate[index] {
			index++
		}
		result = append(result, compositionDelta[T]{
			start:       start,
			end:         index,
			replacement: append([]T(nil), candidate[start:index]...),
			sourceStart: sourceStart,
		})
	}
	return result
}

func newCompositionDelta[T comparable](original, candidate []T, sourceStart int) compositionDelta[T] {
	prefix := 0
	for prefix < len(original) && prefix < len(candidate) && original[prefix] == candidate[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(original)-prefix && suffix < len(candidate)-prefix &&
		original[len(original)-1-suffix] == candidate[len(candidate)-1-suffix] {
		suffix++
	}
	return compositionDelta[T]{
		start:       prefix,
		end:         len(original) - suffix,
		replacement: append([]T(nil), candidate[prefix:len(candidate)-suffix]...),
		sourceStart: sourceStart,
	}
}

func applyCompositionDeltas[T comparable](original []T, deltas []compositionDelta[T]) ([]T, bool) {
	active := make([]compositionDelta[T], 0, len(deltas))
	for _, delta := range deltas {
		if delta.start == delta.end && len(delta.replacement) == 0 {
			continue
		}
		active = append(active, delta)
	}
	sort.Slice(active, func(i, j int) bool {
		if active[i].start != active[j].start {
			return active[i].start < active[j].start
		}
		if active[i].end != active[j].end {
			return active[i].end < active[j].end
		}
		return active[i].sourceStart < active[j].sourceStart
	})
	if !compositionDeltasDisjoint(active) {
		return nil, false
	}
	result := make([]T, 0, len(original))
	cursor := 0
	for _, delta := range active {
		result = append(result, original[cursor:delta.start]...)
		result = append(result, delta.replacement...)
		cursor = delta.end
	}
	result = append(result, original[cursor:]...)
	return result, true
}

func compositionDeltasDisjoint[T comparable](deltas []compositionDelta[T]) bool {
	for index, delta := range deltas {
		if delta.start < 0 || delta.end < delta.start {
			return false
		}
		if index == 0 {
			continue
		}
		previous := deltas[index-1]
		if delta.start < previous.end || delta.start == previous.start {
			return false
		}
	}
	return true
}

func compositionPatchStart(patches []source.Patch) int {
	if len(patches) == 0 {
		return -1
	}
	start := patches[0].Range.Start
	for _, patch := range patches[1:] {
		if patch.Range.Start < start {
			start = patch.Range.Start
		}
	}
	return start
}
