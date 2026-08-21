package splice

import "github.com/zoster81/marksplice/internal/source"

func (d *Document) completeListItemTarget(id NodeID) (Node, error) {
	target, err := d.editableTargetNode(id, KindListItem, "list item")
	if err != nil {
		return Node{}, err
	}
	if !target.ListSubtreeComplete {
		return Node{}, ErrInvalidTargetKind
	}
	return target, nil
}

// PrepareRemoveListItem prepares removal of one complete supported list-item subtree.
// For a leaf item, the subtree is exactly its complete physical line.
func (d *Document) PrepareRemoveListItem(id NodeID) (ChangeSet, error) {
	target, err := d.completeListItemTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	removedSubtree, err := d.ownedListItemSubtree(target)
	if err != nil {
		return ChangeSet{}, err
	}
	removeRange := removedSubtree.Range
	removedIDs := removedSubtree.IDs

	change, candidate, err := d.prepareCandidateChange(removeRange, nil, "list item removal")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateItems, err := candidateListItemMappings(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if len(candidateItems) != d.promotedListItemCount()-len(removedIDs) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if err := d.validateOriginalListItemsAfterPatch(candidate, candidateItems, removeRange, 0, removedIDs, listItemDirectChildCountDeltas(target.ListParentID, "")); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareInsertListItemBefore prepares insertion of one complete same-shape list-item subtree immediately before a complete supported anchor subtree.
func (d *Document) PrepareInsertListItemBefore(id NodeID, fragment []byte) (ChangeSet, error) {
	return d.prepareInsertListItem(id, fragment, false)
}

// PrepareInsertListItemAfter prepares insertion of one complete same-shape list-item subtree immediately after a complete supported anchor subtree.
func (d *Document) PrepareInsertListItemAfter(id NodeID, fragment []byte) (ChangeSet, error) {
	return d.prepareInsertListItem(id, fragment, true)
}

func (d *Document) prepareInsertListItem(id NodeID, fragment []byte, after bool) (ChangeSet, error) {
	anchor, err := d.completeListItemTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	fragmentDocument, fragmentSubtree, err := d.parseListItemSubtreeFragment(fragment, anchor.ListItemSource)
	if err != nil {
		return ChangeSet{}, err
	}

	insertAt := anchor.ListItemSource.LineRange.Start
	operation := "list item insertion before"
	if after {
		insertAt = anchor.ListSubtreeEnd
		operation = "list item insertion after"
	}
	patch := Range{Start: insertAt, End: insertAt}
	change, candidate, err := d.prepareCandidateChange(patch, fragment, operation)
	if err != nil {
		return ChangeSet{}, err
	}
	candidateItems, err := candidateListItemMappings(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if len(candidateItems) != d.promotedListItemCount()+len(fragmentSubtree.IDs) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if err := d.validateOriginalListItemsAfterPatch(candidate, candidateItems, patch, len(fragment), nil, listItemDirectChildCountDeltas("", anchor.ListParentID)); err != nil {
		return ChangeSet{}, err
	}
	transforms := []patchTransform{{Range: patch, ReplacementLength: len(fragment)}}
	if err := fragmentDocument.validateListItemSubtreePlacement(candidate, candidateItems, fragmentSubtree, insertAt, anchor, transforms); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareAppendListItemChild prepares insertion of one complete direct-child subtree at the end of a fully supported list-item subtree.
func (d *Document) PrepareAppendListItemChild(id NodeID, fragment []byte) (ChangeSet, error) {
	parent, err := d.completeListItemTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	if len(fragment) == 0 {
		return ChangeSet{}, ErrInvalidReplacement
	}

	insertAt := parent.ListSubtreeEnd
	patch := Range{Start: insertAt, End: insertAt}
	change, candidate, err := d.prepareCandidateChange(patch, fragment, "list item child append")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateDocument, candidateItems, err := parseListItemMutationCandidate(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	insertedSubtree, err := candidateDocument.insertedListItemChildSubtree(fragment, insertAt, parent.ListItemSource.LineRange.Start)
	if err != nil {
		return ChangeSet{}, err
	}
	if len(candidateItems) != d.promotedListItemCount()+len(insertedSubtree.IDs) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if err := d.validateOriginalListItemsAfterPatch(candidate, candidateItems, patch, len(fragment), nil, listItemDirectChildCountDeltas("", parent.ID)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareMoveListItemBefore prepares moving one complete supported list-item subtree immediately before a complete same-shape anchor subtree.
func (d *Document) PrepareMoveListItemBefore(id, anchorID NodeID) (ChangeSet, error) {
	return d.prepareMoveListItem(id, anchorID, false)
}

// PrepareMoveListItemAfter prepares moving one complete supported list-item subtree immediately after a complete same-shape anchor subtree.
func (d *Document) PrepareMoveListItemAfter(id, anchorID NodeID) (ChangeSet, error) {
	return d.prepareMoveListItem(id, anchorID, true)
}

func (d *Document) prepareMoveListItem(id, anchorID NodeID, after bool) (ChangeSet, error) {
	if id == anchorID {
		return ChangeSet{}, ErrInvalidReplacement
	}
	moved, err := d.completeListItemTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	anchor, err := d.completeListItemTarget(anchorID)
	if err != nil {
		return ChangeSet{}, err
	}

	movedSubtree, err := d.ownedListItemSubtree(moved)
	if err != nil {
		return ChangeSet{}, err
	}
	movedRange := movedSubtree.Range
	anchorRange := listItemSubtreeRange(anchor)
	if rangesOverlap(movedRange, anchorRange) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if !sameListItemSiblingShape(d.source, moved.ListItemSource, d.source, anchor.ListItemSource) {
		return ChangeSet{}, ErrInvalidReplacement
	}

	fragment := d.source[movedRange.Start:movedRange.End]
	movedIDs := movedSubtree.IDs

	insertAt := anchor.ListItemSource.LineRange.Start
	operation := "list item move before"
	alreadyPlaced := movedRange.End == anchor.ListItemSource.LineRange.Start
	if after {
		insertAt = anchor.ListSubtreeEnd
		operation = "list item move after"
		alreadyPlaced = anchor.ListSubtreeEnd == movedRange.Start
	}
	if alreadyPlaced && listItemsShareSemanticParent(moved, anchor) {
		return d.newChanges(nil, operation)
	}

	movedOffset, ok := movedRangeCandidateOffset(movedRange, insertAt)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	insertRange := Range{Start: insertAt, End: insertAt}
	patches := []source.Patch{
		{Range: movedRange},
		{Range: insertRange, Replacement: fragment},
	}
	transforms := []patchTransform{
		{Range: movedRange},
		{Range: insertRange, ReplacementLength: len(fragment)},
	}
	change, candidate, err := d.prepareCandidateChanges(patches, operation)
	if err != nil {
		return ChangeSet{}, err
	}
	candidateItems, err := candidateListItemMappings(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateOriginalListItemsAfterPatches(candidate, candidateItems, transforms, movedIDs, listItemDirectChildCountDeltas(moved.ListParentID, anchor.ListParentID)); err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateListItemSubtreePlacement(candidate, candidateItems, movedSubtree, movedOffset, anchor, transforms); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}
