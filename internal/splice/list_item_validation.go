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

	var root Node
	foundRoot := false
	for _, node := range fragmentDocument.nodes {
		if node.Kind != KindListItem || !node.Editable || node.ListItemSource.LineRange.Start != 0 {
			continue
		}
		if foundRoot {
			return nil, listItemSubtreeOwnership{}, ErrInvalidReplacement
		}
		root = node
		foundRoot = true
	}
	if !foundRoot || root.ListHasParent {
		return nil, listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	fragmentSubtree, err := fragmentDocument.ownedListItemSubtree(root)
	if err != nil {
		return nil, listItemSubtreeOwnership{}, err
	}
	if fragmentSubtree.Range != (Range{Start: 0, End: len(fragment)}) ||
		!sameListItemSiblingShape(fragment, root.ListItemSource, d.source, anchor) ||
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
	items := make(map[int]listItemCandidateMapping)
	for _, node := range document.nodes {
		if node.Kind != KindListItem {
			continue
		}
		mapping := node.ListItemSource
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
	for _, originalNode := range d.nodes {
		if originalNode.Kind != KindListItem {
			continue
		}
		if _, skip := skipIDs[originalNode.ID]; skip {
			continue
		}
		original := originalNode.ListItemSource
		expectedLine, ok := rangeAfterPatches(original.LineRange, patches)
		if !ok {
			return ErrInvalidReplacement
		}
		expectedRange, ok := rangeAfterPatches(original.Range, patches)
		if !ok {
			return ErrInvalidReplacement
		}
		expectedContent, ok := rangeAfterPatches(original.ContentRange, patches)
		if !ok {
			return ErrInvalidReplacement
		}

		candidateMapping, ok := candidateItems[expectedLine.Start]
		if !ok || !sameListItemLexicalMapping(candidate, candidateMapping.Mapping, d.source, original, expectedLine, expectedRange, expectedContent) ||
			candidateMapping.HasParent != originalNode.ListHasParent {
			return ErrInvalidReplacement
		}
		expectedChildCount := originalNode.ListDirectChildCount + childCountDeltas[originalNode.ID]
		if expectedChildCount < 0 || candidateMapping.DirectChildCount != expectedChildCount || candidateMapping.HasChildren != (expectedChildCount != 0) {
			return ErrInvalidReplacement
		}
		if originalNode.ListHasParent {
			expectedParentAnchor, ok := listParentAnchorAfterPatches(originalNode, patches)
			if !ok || candidateMapping.ParentAnchor != expectedParentAnchor {
				return ErrInvalidReplacement
			}
		}
	}
	return nil
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
	if !item.ListHasParent {
		return 0, true
	}

	// The parent anchor is the first physical source byte of the parent line.
	// Transforming that byte keeps the anchor right-biased when insertion occurs
	// exactly before the parent while remaining independent of edits later in the
	// same parent line. This works identically for supported and unsupported parents.
	expectedParentSource, ok := rangeAfterPatches(
		Range{Start: item.ListParentAnchor, End: item.ListParentAnchor + 1},
		patches,
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
	expectedAnchorLine, ok := rangeAfterPatches(anchor.ListItemSource.LineRange, patches)
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
	if subtreeLength <= 0 || candidateRange.Start < 0 || candidateRange.End > len(candidate) {
		return ErrInvalidReplacement
	}
	if !bytes.Equal(d.source[subtree.Range.Start:subtree.Range.End], candidate[candidateRange.Start:candidateRange.End]) {
		return ErrInvalidReplacement
	}

	candidateCount := 0
	for _, candidateMapping := range candidateItems {
		lineRange := candidateMapping.Mapping.LineRange
		if lineRange.Start >= candidateRange.Start && lineRange.End <= candidateRange.End {
			candidateCount++
		}
	}
	if candidateCount != len(subtree.IDs) {
		return ErrInvalidReplacement
	}

	delta := candidateOffset - subtree.Range.Start
	for _, originalNode := range d.nodes {
		if _, inSubtree := subtree.IDs[originalNode.ID]; !inSubtree {
			continue
		}
		original := originalNode.ListItemSource
		expectedLine := shiftedRange(original.LineRange, delta)
		expectedRange := shiftedRange(original.Range, delta)
		expectedContent := shiftedRange(original.ContentRange, delta)
		candidateMapping, ok := candidateItems[expectedLine.Start]
		if !ok || !sameListItemLexicalMapping(candidate, candidateMapping.Mapping, d.source, original, expectedLine, expectedRange, expectedContent) ||
			candidateMapping.HasChildren != originalNode.ListHasChildren || candidateMapping.DirectChildCount != originalNode.ListDirectChildCount {
			return ErrInvalidReplacement
		}
		if originalNode.ID != subtree.Root.ID {
			if candidateMapping.HasParent != originalNode.ListHasParent {
				return ErrInvalidReplacement
			}
			if originalNode.ListHasParent && candidateMapping.ParentAnchor != originalNode.ListParentAnchor+delta {
				return ErrInvalidReplacement
			}
		}
	}
	return validateCandidateListItemSibling(candidateItems, candidateOffset, anchor, patches)
}

func (d *Document) insertedListItemChildSubtree(fragment []byte, insertAt, parentAnchor int) (listItemSubtreeOwnership, error) {
	if len(fragment) == 0 || insertAt < 0 || insertAt > len(d.source) || len(fragment) > len(d.source)-insertAt {
		return listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	root, ok := d.listItemNodeAtLineStart(insertAt)
	if !ok || !root.ListHasParent || root.ListParentAnchor != parentAnchor || !root.ListSubtreeComplete {
		return listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	parent, ok := d.listItemNodeAtLineStart(parentAnchor)
	if !ok || root.ListParentID != parent.ID {
		return listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	subtree, err := d.ownedListItemSubtree(root)
	if err != nil {
		return listItemSubtreeOwnership{}, err
	}
	expectedRange := Range{Start: insertAt, End: insertAt + len(fragment)}
	if subtree.Range != expectedRange || !bytes.Equal(fragment, d.source[expectedRange.Start:expectedRange.End]) {
		return listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	return subtree, nil
}

func (d *Document) listItemNodeAtLineStart(lineStart int) (Node, bool) {
	for _, node := range d.nodes {
		if node.Kind == KindListItem && node.ListItemSource.LineRange.Start == lineStart {
			return node, true
		}
	}
	return Node{}, false
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
