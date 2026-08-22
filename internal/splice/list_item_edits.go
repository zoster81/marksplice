package splice

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

// PrepareReplaceListItemSubtree prepares replacement of one complete supported list-item subtree while preserving its external sibling shape and semantic parent.
func (d *Document) PrepareReplaceListItemSubtree(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.completeListItemTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	if len(replacement) == 0 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	removedSubtree, err := d.ownedListItemSubtree(target)
	if err != nil {
		return ChangeSet{}, err
	}

	replaceRange := removedSubtree.Range
	change, candidate, err := d.prepareCandidateChange(replaceRange, replacement, "list item subtree replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateDocument, candidateItems, err := parseListItemMutationCandidate(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	replacementSubtree, err := candidateDocument.exactListItemSubtree(replacement, replaceRange.Start)
	if err != nil {
		return ChangeSet{}, err
	}
	if len(candidateItems) != d.promotedListItemCount()-len(removedSubtree.IDs)+len(replacementSubtree.IDs) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	patches := []patchTransform{{Range: replaceRange, ReplacementLength: len(replacement)}}
	if err := d.validateOriginalListItemsAfterPatches(candidate, candidateItems, patches, removedSubtree.IDs, nil); err != nil {
		return ChangeSet{}, err
	}
	if err := validateReplacedListItemSubtreeRoot(d.source, target, candidate, replacementSubtree, patches); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
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

type listItemMovePlan struct {
	moved        Node
	anchor       Node
	movedSubtree listItemSubtreeOwnership
	movedOffset  int
	insertAt     int
	operation    string
	fragment     []byte
}

func (d *Document) prepareMoveListItem(id, anchorID NodeID, after bool) (ChangeSet, error) {
	plan, noOp, err := d.planListItemMove(id, anchorID, after)
	if err != nil {
		return ChangeSet{}, err
	}
	if noOp {
		return d.newChanges(nil, plan.operation)
	}
	change, candidate, insertRange, err := d.prepareMoveCandidate(plan.movedSubtree.Range, plan.insertAt, plan.fragment, plan.operation)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateListItemMoveCandidate(candidate, plan, insertRange); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func (d *Document) planListItemMove(id, anchorID NodeID, after bool) (listItemMovePlan, bool, error) {
	if id == anchorID {
		return listItemMovePlan{}, false, ErrInvalidReplacement
	}
	moved, err := d.completeListItemTarget(id)
	if err != nil {
		return listItemMovePlan{}, false, err
	}
	anchor, err := d.completeListItemTarget(anchorID)
	if err != nil {
		return listItemMovePlan{}, false, err
	}
	movedSubtree, err := d.ownedListItemSubtree(moved)
	if err != nil {
		return listItemMovePlan{}, false, err
	}
	if rangesOverlap(movedSubtree.Range, listItemSubtreeRange(anchor)) ||
		!sameListItemSiblingShape(d.source, moved.ListItemSource, d.source, anchor.ListItemSource) {
		return listItemMovePlan{}, false, ErrInvalidReplacement
	}
	plan := listItemMovePlan{
		moved:        moved,
		anchor:       anchor,
		movedSubtree: movedSubtree,
		insertAt:     anchor.ListItemSource.LineRange.Start,
		operation:    "list item move before",
	}
	alreadyPlaced := movedSubtree.Range.End == anchor.ListItemSource.LineRange.Start
	if after {
		plan.insertAt = anchor.ListSubtreeEnd
		plan.operation = "list item move after"
		alreadyPlaced = anchor.ListSubtreeEnd == movedSubtree.Range.Start
	}
	if alreadyPlaced && listItemsShareSemanticParent(moved, anchor) {
		return plan, true, nil
	}
	movedOffset, ok := movedRangeCandidateOffset(movedSubtree.Range, plan.insertAt)
	if !ok {
		return listItemMovePlan{}, false, ErrInvalidReplacement
	}
	plan.movedOffset = movedOffset
	plan.fragment = d.source[movedSubtree.Range.Start:movedSubtree.Range.End]
	return plan, false, nil
}

func (d *Document) validateListItemMoveCandidate(candidate []byte, plan listItemMovePlan, insertRange Range) error {
	candidateItems, err := candidateListItemMappings(candidate)
	if err != nil {
		return err
	}
	transforms := []patchTransform{
		{Range: plan.movedSubtree.Range},
		{Range: insertRange, ReplacementLength: len(plan.fragment)},
	}
	if err := d.validateOriginalListItemsAfterPatches(candidate, candidateItems, transforms, plan.movedSubtree.IDs, listItemDirectChildCountDeltas(plan.moved.ListParentID, plan.anchor.ListParentID)); err != nil {
		return err
	}
	return d.validateListItemSubtreePlacement(candidate, candidateItems, plan.movedSubtree, plan.movedOffset, plan.anchor, transforms)
}
