package splice

import (
	"bytes"
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

type listItemCandidateMapping struct {
	Mapping          source.ListItemMapping
	HasParent        bool
	ParentAnchor     int
	HasChildren      bool
	DirectChildCount int
}

func (d *Document) parseListItemSubtreeFragment(fragment []byte, anchor source.ListItemMapping) (*Document, listItemSubtreeOwnership, error) {
	if len(fragment) == 0 {
		return nil, listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	fragmentDocument, err := Parse(fragment)
	if err != nil {
		return nil, listItemSubtreeOwnership{}, fmt.Errorf("%w: list-item fragment parse: %v", ErrInvalidReplacement, err)
	}

	root, foundRoot := fragmentDocument.listItemNodeAtLineStart(0)
	if !foundRoot || root.ListHasParent {
		return nil, listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	fragmentSubtree, err := fragmentDocument.ownedListItemSubtree(root)
	if err != nil {
		return nil, listItemSubtreeOwnership{}, err
	}
	rootSource, ok := remapListItemSource(fragmentDocument.source, root)
	if !ok || fragmentSubtree.Range != (Range{Start: 0, End: len(fragment)}) ||
		!sameListItemSiblingShape(fragment, rootSource, d.source, anchor) ||
		len(fragmentSubtree.IDs) != fragmentDocument.promotedListItemCount() {
		return nil, listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	return fragmentDocument, fragmentSubtree, nil
}

func sameListItemSiblingShape(leftSource []byte, left source.ListItemMapping, rightSource []byte, right source.ListItemMapping) bool {
	if left.Ordered != right.Ordered || left.Marker != right.Marker {
		return false
	}
	leftPrefix := leftSource[left.LineRange.Start:left.Range.Start]
	rightPrefix := rightSource[right.LineRange.Start:right.Range.Start]
	return bytes.Equal(leftPrefix, rightPrefix)
}

func candidateListItemMappings(candidate []byte) (map[int]listItemCandidateMapping, error) {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return nil, err
	}
	items := make(map[int]listItemCandidateMapping)
	for _, observation := range observations {
		if observation.Kind != parser.KindListItem {
			continue
		}
		mapping, err := source.MapSingleLineListItem(
			candidate,
			Range{Start: observation.Range.Start, End: observation.Range.End},
			observation.Ordered,
			observation.Marker,
		)
		if err != nil {
			continue
		}
		if _, exists := items[mapping.LineRange.Start]; exists {
			return nil, ErrInvalidReplacement
		}
		items[mapping.LineRange.Start] = listItemCandidateMapping{
			Mapping:          mapping,
			HasParent:        observation.HasListParent,
			ParentAnchor:     observation.ListParentAnchor,
			HasChildren:      observation.HasListChildren,
			DirectChildCount: observation.ListDirectChildCount,
		}
	}
	return items, nil
}

func parseListItemMutationCandidate(candidate []byte) (*Document, map[int]listItemCandidateMapping, error) {
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: list-item mutation candidate parse: %v", ErrInvalidReplacement, err)
	}
	candidateItems, err := listItemCandidateMappingsFromDocument(candidateDocument)
	if err != nil {
		return nil, nil, err
	}
	return candidateDocument, candidateItems, nil
}

func listItemCandidateMappingsFromDocument(document *Document) (map[int]listItemCandidateMapping, error) {
	if document == nil {
		return nil, ErrInvalidReplacement
	}
	items := make(map[int]listItemCandidateMapping, len(document.listItemIndexes))
	for _, nodeIndex := range document.listItemIndexes {
		node, ok := document.indexedEditableNode(nodeIndex, KindListItem)
		if !ok {
			return nil, ErrInvalidReplacement
		}
		mapping, ok := remapListItemSource(document.source, node)
		if !ok {
			return nil, ErrInvalidReplacement
		}
		if _, exists := items[mapping.LineRange.Start]; exists {
			return nil, ErrInvalidReplacement
		}
		items[mapping.LineRange.Start] = listItemCandidateMapping{
			Mapping:          mapping,
			HasParent:        node.ListHasParent,
			ParentAnchor:     node.ListParentAnchor,
			HasChildren:      node.ListHasChildren,
			DirectChildCount: node.ListDirectChildCount,
		}
	}
	return items, nil
}

func (d *Document) validateOriginalListItemsAfterPatch(candidate []byte, candidateItems map[int]listItemCandidateMapping, patch Range, replacementLength int, skipIDs map[NodeID]struct{}, childCountDeltas map[NodeID]int) error {
	return d.validateOriginalListItemsAfterPatches(candidate, candidateItems, []patchTransform{{Range: patch, ReplacementLength: replacementLength}}, skipIDs, childCountDeltas)
}

func (d *Document) validateOriginalListItemsAfterPatches(candidate []byte, candidateItems map[int]listItemCandidateMapping, patches []patchTransform, skipIDs map[NodeID]struct{}, childCountDeltas map[NodeID]int) error {
	ordered, ok := orderedPatchTransforms(patches)
	if !ok {
		return ErrInvalidReplacement
	}
	for _, nodeIndex := range d.listItemIndexes {
		original, ok := d.indexedEditableNode(nodeIndex, KindListItem)
		if !ok {
			return ErrInvalidReplacement
		}
		if _, skip := skipIDs[original.ID]; skip {
			continue
		}
		if !d.originalListItemSurvives(candidate, candidateItems, original, ordered, childCountDeltas[original.ID]) {
			return ErrInvalidReplacement
		}
	}
	return nil
}

func (d *Document) originalListItemSurvives(candidate []byte, candidateItems map[int]listItemCandidateMapping, originalNode Node, ordered []patchTransform, childCountDelta int) bool {
	original, ok := remapListItemSource(d.source, originalNode)
	if !ok {
		return false
	}
	expectedLine, ok := rangeAfterOrderedPatches(original.LineRange, ordered)
	if !ok {
		return false
	}
	expectedRange, ok := rangeAfterOrderedPatches(original.Range, ordered)
	if !ok {
		return false
	}
	expectedContent, ok := rangeAfterOrderedPatches(original.ContentRange, ordered)
	if !ok {
		return false
	}
	candidateMapping, ok := candidateItems[expectedLine.Start]
	if !ok || !sameListItemLexicalMapping(candidate, candidateMapping.Mapping, d.source, original, expectedLine, expectedRange, expectedContent) ||
		candidateMapping.HasParent != originalNode.ListHasParent {
		return false
	}
	expectedChildCount := originalNode.ListDirectChildCount + childCountDelta
	if expectedChildCount < 0 || candidateMapping.DirectChildCount != expectedChildCount || candidateMapping.HasChildren != (expectedChildCount != 0) {
		return false
	}
	if !originalNode.ListHasParent {
		return true
	}
	expectedParentAnchor, ok := listParentAnchorAfterOrderedPatches(originalNode, ordered)
	return ok && candidateMapping.ParentAnchor == expectedParentAnchor
}

func sameListItemLexicalMapping(candidate []byte, mapped source.ListItemMapping, originalSource []byte, original source.ListItemMapping, expectedLine, expectedRange, expectedContent Range) bool {
	if mapped.LineRange != expectedLine || mapped.Range != expectedRange || mapped.ContentRange != expectedContent ||
		mapped.Ordered != original.Ordered || mapped.Marker != original.Marker {
		return false
	}
	if !original.LineRange.Valid(len(originalSource)) || !expectedLine.Valid(len(candidate)) {
		return false
	}
	return bytes.Equal(originalSource[original.LineRange.Start:original.LineRange.End], candidate[expectedLine.Start:expectedLine.End])
}

func listParentAnchorAfterPatches(item Node, patches []patchTransform) (int, bool) {
	ordered, ok := orderedPatchTransforms(patches)
	if !ok {
		return 0, false
	}
	return listParentAnchorAfterOrderedPatches(item, ordered)
}

func listParentAnchorAfterOrderedPatches(item Node, ordered []patchTransform) (int, bool) {
	if !item.ListHasParent {
		return 0, true
	}

	// The parent anchor is the first physical source byte of the parent line.
	// Transforming that byte keeps the anchor right-biased when insertion occurs
	// exactly before the parent while remaining independent of edits later in the
	// same parent line. This works identically for supported and unsupported parents.
	expectedParentSource, ok := rangeAfterOrderedPatches(
		Range{Start: item.ListParentAnchor, End: item.ListParentAnchor + 1},
		ordered,
	)
	if !ok {
		return 0, false
	}
	return expectedParentSource.Start, true
}

func validateCandidateListItemSibling(candidateItems map[int]listItemCandidateMapping, itemStart int, anchor Node, patches []patchTransform) error {
	item, ok := candidateItems[itemStart]
	if !ok {
		return ErrInvalidReplacement
	}
	expectedAnchorLine, ok := rangeAfterPatches(anchor.ListItemLineRange, patches)
	if !ok {
		return ErrInvalidReplacement
	}
	candidateAnchor, ok := candidateItems[expectedAnchorLine.Start]
	if !ok || item.HasParent != candidateAnchor.HasParent {
		return ErrInvalidReplacement
	}
	if item.HasParent && item.ParentAnchor != candidateAnchor.ParentAnchor {
		return ErrInvalidReplacement
	}
	return nil
}

func listItemsShareSemanticParent(left, right Node) bool {
	if left.ListHasParent != right.ListHasParent {
		return false
	}
	return !left.ListHasParent || left.ListParentAnchor == right.ListParentAnchor
}

func (d *Document) validateListItemSubtreePlacement(candidate []byte, candidateItems map[int]listItemCandidateMapping, subtree listItemSubtreeOwnership, candidateOffset int, anchor Node, patches []patchTransform) error {
	subtreeLength := subtree.Range.End - subtree.Range.Start
	candidateRange := Range{Start: candidateOffset, End: candidateOffset + subtreeLength}
	if subtreeLength <= 0 || candidateRange.Start < 0 || candidateRange.End > len(candidate) ||
		!bytes.Equal(d.source[subtree.Range.Start:subtree.Range.End], candidate[candidateRange.Start:candidateRange.End]) ||
		candidateListItemCountWithinRange(candidateItems, candidateRange) != len(subtree.IDs) {
		return ErrInvalidReplacement
	}
	if err := d.validatePlacedListItemSubtreeMappings(candidate, candidateItems, subtree, candidateOffset-subtree.Range.Start); err != nil {
		return err
	}
	return validateCandidateListItemSibling(candidateItems, candidateOffset, anchor, patches)
}

func candidateListItemCountWithinRange(candidateItems map[int]listItemCandidateMapping, range_ Range) int {
	count := 0
	for _, candidateMapping := range candidateItems {
		lineRange := candidateMapping.Mapping.LineRange
		if lineRange.Start >= range_.Start && lineRange.End <= range_.End {
			count++
		}
	}
	return count
}

func (d *Document) validatePlacedListItemSubtreeMappings(candidate []byte, candidateItems map[int]listItemCandidateMapping, subtree listItemSubtreeOwnership, delta int) error {
	for id := range subtree.IDs {
		originalNode, ok := d.nodeByID(id)
		if !ok || originalNode.Kind != KindListItem {
			return ErrInvalidReplacement
		}
		original, ok := remapListItemSource(d.source, originalNode)
		if !ok {
			return ErrInvalidReplacement
		}
		expectedLine := shiftedRange(original.LineRange, delta)
		expectedRange := shiftedRange(original.Range, delta)
		expectedContent := shiftedRange(original.ContentRange, delta)
		candidateMapping, ok := candidateItems[expectedLine.Start]
		if !ok || !sameListItemLexicalMapping(candidate, candidateMapping.Mapping, d.source, original, expectedLine, expectedRange, expectedContent) ||
			candidateMapping.HasChildren != originalNode.ListHasChildren || candidateMapping.DirectChildCount != originalNode.ListDirectChildCount {
			return ErrInvalidReplacement
		}
		if originalNode.ID == subtree.Root.ID {
			continue
		}
		if candidateMapping.HasParent != originalNode.ListHasParent ||
			originalNode.ListHasParent && candidateMapping.ParentAnchor != originalNode.ListParentAnchor+delta {
			return ErrInvalidReplacement
		}
	}
	return nil
}

func (d *Document) exactListItemSubtree(fragment []byte, start int) (listItemSubtreeOwnership, error) {
	if len(fragment) == 0 || start < 0 || start > len(d.source) || len(fragment) > len(d.source)-start {
		return listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	root, ok := d.listItemNodeAtLineStart(start)
	if !ok || !root.ListSubtreeComplete {
		return listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	subtree, err := d.ownedListItemSubtree(root)
	if err != nil {
		return listItemSubtreeOwnership{}, err
	}
	expectedRange := Range{Start: start, End: start + len(fragment)}
	if subtree.Range != expectedRange || !bytes.Equal(fragment, d.source[expectedRange.Start:expectedRange.End]) {
		return listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	return subtree, nil
}

func (d *Document) insertedListItemChildSubtree(fragment []byte, insertAt, parentAnchor int) (listItemSubtreeOwnership, error) {
	subtree, err := d.exactListItemSubtree(fragment, insertAt)
	if err != nil {
		return listItemSubtreeOwnership{}, err
	}
	root := subtree.Root
	if !root.ListHasParent || root.ListParentAnchor != parentAnchor {
		return listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	parent, ok := d.listItemNodeAtLineStart(parentAnchor)
	if !ok || root.ListParentID != parent.ID {
		return listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	return subtree, nil
}

func validateReplacedListItemSubtreeRoot(originalSource []byte, target Node, candidateSource []byte, replacement listItemSubtreeOwnership, patches []patchTransform) error {
	root := replacement.Root
	rootSource, rootOK := remapListItemSource(candidateSource, root)
	targetSource, targetOK := remapListItemSource(originalSource, target)
	if !rootOK || !targetOK || root.ListHasParent != target.ListHasParent || !sameListItemSiblingShape(candidateSource, rootSource, originalSource, targetSource) {
		return ErrInvalidReplacement
	}
	if !target.ListHasParent {
		return nil
	}
	expectedParentAnchor, ok := listParentAnchorAfterPatches(target, patches)
	if !ok || root.ListParentAnchor != expectedParentAnchor {
		return ErrInvalidReplacement
	}
	return nil
}

func (d *Document) listItemNodeAtLineStart(lineStart int) (Node, bool) {
	if d == nil || lineStart < 0 {
		return Node{}, false
	}
	low, high := 0, len(d.listItemIndexes)
	for low < high {
		middle := low + (high-low)/2
		nodeIndex := d.listItemIndexes[middle]
		node, ok := d.indexedEditableNode(nodeIndex, KindListItem)
		if !ok {
			return Node{}, false
		}
		if node.ListItemLineRange.Start < lineStart {
			low = middle + 1
		} else {
			high = middle
		}
	}
	if low >= len(d.listItemIndexes) {
		return Node{}, false
	}
	nodeIndex := d.listItemIndexes[low]
	node, ok := d.indexedEditableNode(nodeIndex, KindListItem)
	if !ok || node.ListItemLineRange.Start != lineStart {
		return Node{}, false
	}
	return node, true
}

func listItemIDSet(ids ...NodeID) map[NodeID]struct{} {
	set := make(map[NodeID]struct{}, len(ids))
	for _, id := range ids {
		if id != "" {
			set[id] = struct{}{}
		}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

func listItemDirectChildCountDeltas(removedParentID, addedParentID NodeID) map[NodeID]int {
	if removedParentID == addedParentID {
		return nil
	}
	deltas := make(map[NodeID]int, 2)
	if removedParentID != "" {
		deltas[removedParentID] = -1
	}
	if addedParentID != "" {
		deltas[addedParentID] = 1
	}
	return deltas
}
