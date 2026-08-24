package splice

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

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
	linkDeltas := make([]compositionDelta[string], 0, len(changes))
	originalNodes := compositionNodeViews(d)
	originalLinks := compositionLinkViews(d.linkUsages)
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
		nodeDeltas = append(nodeDeltas, newCompositionDelta(originalNodes, compositionNodeViews(candidateDocument), sourceStart))
		linkDeltas = append(linkDeltas, newCompositionDelta(originalLinks, compositionLinkViews(candidateDocument.linkUsages), sourceStart))
	}

	expectedNodes, ok := applyCompositionDeltas(originalNodes, nodeDeltas)
	if !ok {
		return ChangeSet{}, fmt.Errorf("%w: prepared mutations affect overlapping structural model regions", ErrInvalidReplacement)
	}
	expectedLinks, ok := applyCompositionDeltas(originalLinks, linkDeltas)
	if !ok {
		return ChangeSet{}, fmt.Errorf("%w: prepared mutations affect overlapping link relationships", ErrInvalidReplacement)
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
		!slices.Equal(compositionLinkViews(candidateDocument.linkUsages), expectedLinks) {
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
	semantic     string
	sourceHash   source.Fingerprint
	ownedLength  int
	rangeStart   int
	rangeEnd     int
	contentStart int
	contentEnd   int
}

func compositionNodeViews(document *Document) []compositionNodeView {
	if document == nil {
		return nil
	}
	indexes := make(map[NodeID]int, len(document.nodes))
	for index, node := range document.nodes {
		indexes[node.ID] = index
	}
	views := make([]compositionNodeView, len(document.nodes))
	for index, node := range document.nodes {
		owned := compositionOwnedRange(node)
		if !owned.Valid(len(document.source)) || owned.Start > node.Range.Start || owned.End < node.Range.End {
			owned = node.Range
		}
		view := compositionNodeView{
			semantic:    compositionNodeSemantic(node, index, indexes, owned),
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

func compositionOwnedRange(node Node) Range {
	switch node.Kind {
	case KindListItem:
		return node.ListItemSource.LineRange
	case KindTableCell:
		return node.TableCellSource.Range
	case KindTableRow:
		return node.TableRowSource.LineRange
	case KindTable:
		return node.TableSource.Range
	case KindReferenceDefinition:
		return node.ReferenceDefinitionSource.LineRange
	case KindThematicBreak:
		return node.ThematicBreakSource.LineRange
	case KindBlockquote:
		return node.BlockquoteSource.LineRange
	default:
		return node.Range
	}
}

func compositionNodeSemantic(node Node, index int, indexes map[NodeID]int, owned Range) string {
	var builder strings.Builder
	_, _ = fmt.Fprintf(&builder, "%d|%#v|align=%v|subtree=%t|children=%d", node.Kind, removalSurvivorSemanticSignature(node), node.TableAlignments, node.ListSubtreeComplete, node.ListChildCount)
	_, _ = fmt.Fprintf(&builder, "|list-parent=%d|table=%d|row=%d|prev-row=%d|next-row=%d", relativeNodeIndex(index, node.ListParentID, indexes), relativeNodeIndex(index, node.TableID, indexes), relativeNodeIndex(index, node.TableRowID, indexes), relativeNodeIndex(index, node.TablePreviousRowID, indexes), relativeNodeIndex(index, node.TableNextRowID, indexes))
	if node.Kind == KindBlockquote {
		_, _ = fmt.Fprintf(&builder, "|bq-marker=%s|bq-content=%s", compositionRelativeRange(node.BlockquoteSource.MarkerRange, owned), compositionRelativeRanges(node.BlockquoteSource.ContentRanges, owned))
	}
	return builder.String()
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

func compositionRelativeRange(range_, base Range) string {
	if range_ == (Range{}) {
		return "-"
	}
	return fmt.Sprintf("%d:%d", range_.Start-base.Start, range_.End-base.Start)
}

func compositionRelativeRanges(ranges []source.Range, base Range) string {
	if len(ranges) == 0 {
		return "-"
	}
	var builder strings.Builder
	for index, range_ := range ranges {
		if index != 0 {
			builder.WriteByte(',')
		}
		_, _ = fmt.Fprintf(&builder, "%d:%d", range_.Start-base.Start, range_.End-base.Start)
	}
	return builder.String()
}

func compositionLinkViews(usages []parser.LinkUsage) []string {
	views := make([]string, len(usages))
	for index, usage := range usages {
		views[index] = fmt.Sprintf("%d|%d|%s|%s|%s|%t|%t", usage.Kind, usage.Form, usage.Reference, usage.Destination, usage.Title, usage.HasTitle, usage.AutoLinkEmail)
	}
	return views
}

type compositionDelta[T comparable] struct {
	start       int
	end         int
	replacement []T
	sourceStart int
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
