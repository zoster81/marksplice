package splice

// PrepareReplaceTableRow prepares replacement of one complete promoted GFM table body row.
func (d *Document) PrepareReplaceTableRow(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.tableRowTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	if len(replacement) == 0 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	lineRange := target.Range
	change, candidate, err := d.prepareCandidateChange(lineRange, replacement, "table row replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateIndex, err := parseTableRowMutationCandidate(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if candidateIndex.rowCount != d.promotedTableRowCount() {
		return ChangeSet{}, ErrInvalidReplacement
	}
	patches := []patchTransform{{Range: lineRange, ReplacementLength: len(replacement)}}
	if err := d.validateOriginalTableModelAfterPatches(candidate, candidateIndex, patches, &target); err != nil {
		return ChangeSet{}, err
	}
	if err := validateReplacedTableRow(candidate, candidateIndex, target, replacement, patches); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareRemoveTableRow prepares removal of one complete promoted GFM table body row.
func (d *Document) PrepareRemoveTableRow(id NodeID) (ChangeSet, error) {
	target, err := d.tableRowTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	change, candidate, err := d.prepareCandidateChange(target.Range, nil, "table row removal")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateIndex, err := parseTableRowMutationCandidate(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if candidateIndex.rowCount != d.promotedTableRowCount()-1 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	patches := []patchTransform{{Range: target.Range}}
	if err := d.validateOriginalTableModelAfterPatches(candidate, candidateIndex, patches, &target); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareInsertTableRowBefore prepares insertion of one complete body row before a promoted row in the same table.
func (d *Document) PrepareInsertTableRowBefore(anchorID NodeID, fragment []byte) (ChangeSet, error) {
	return d.prepareInsertTableRow(anchorID, fragment, false)
}

// PrepareInsertTableRowAfter prepares insertion of one complete body row after a promoted row in the same table.
func (d *Document) PrepareInsertTableRowAfter(anchorID NodeID, fragment []byte) (ChangeSet, error) {
	return d.prepareInsertTableRow(anchorID, fragment, true)
}

func (d *Document) prepareInsertTableRow(anchorID NodeID, fragment []byte, after bool) (ChangeSet, error) {
	anchor, err := d.tableRowTarget(anchorID)
	if err != nil {
		return ChangeSet{}, err
	}
	if len(fragment) == 0 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	insertAt := anchor.Range.Start
	operation := "table row insertion before"
	if after {
		insertAt = anchor.Range.End
		operation = "table row insertion after"
	}
	patch := Range{Start: insertAt, End: insertAt}
	change, candidate, err := d.prepareCandidateChange(patch, fragment, operation)
	if err != nil {
		return ChangeSet{}, err
	}
	candidateIndex, err := parseTableRowMutationCandidate(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if candidateIndex.rowCount != d.promotedTableRowCount()+1 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	patches := []patchTransform{{Range: patch, ReplacementLength: len(fragment)}}
	if err := d.validateOriginalTableModelAfterPatches(candidate, candidateIndex, patches, nil); err != nil {
		return ChangeSet{}, err
	}
	if err := validateInsertedTableRow(candidate, candidateIndex, anchor, fragment, insertAt, patches); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareMoveTableRowBefore prepares moving one complete body row before another row in the same table.
func (d *Document) PrepareMoveTableRowBefore(id, anchorID NodeID) (ChangeSet, error) {
	return d.prepareMoveTableRow(id, anchorID, false)
}

// PrepareMoveTableRowAfter prepares moving one complete body row after another row in the same table.
func (d *Document) PrepareMoveTableRowAfter(id, anchorID NodeID) (ChangeSet, error) {
	return d.prepareMoveTableRow(id, anchorID, true)
}

type tableRowMovePlan struct {
	moved      Node
	anchor     Node
	movedRange Range
	insertAt   int
	operation  string
	fragment   []byte
}

func (d *Document) prepareMoveTableRow(id, anchorID NodeID, after bool) (ChangeSet, error) {
	plan, noOp, err := d.planTableRowMove(id, anchorID, after)
	if err != nil {
		return ChangeSet{}, err
	}
	if noOp {
		return d.newChanges(nil, plan.operation)
	}
	change, candidate, insertRange, err := d.prepareMoveCandidate(plan.movedRange, plan.insertAt, plan.fragment, plan.operation)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateTableRowMoveCandidate(candidate, plan, insertRange); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func (d *Document) planTableRowMove(id, anchorID NodeID, after bool) (tableRowMovePlan, bool, error) {
	if id == anchorID {
		return tableRowMovePlan{}, false, ErrInvalidReplacement
	}
	moved, err := d.tableRowTarget(id)
	if err != nil {
		return tableRowMovePlan{}, false, err
	}
	anchor, err := d.tableRowTarget(anchorID)
	if err != nil {
		return tableRowMovePlan{}, false, err
	}
	if moved.TableAnchor != anchor.TableAnchor {
		return tableRowMovePlan{}, false, ErrInvalidReplacement
	}
	plan := tableRowMovePlan{
		moved:      moved,
		anchor:     anchor,
		movedRange: moved.Range,
		insertAt:   anchor.Range.Start,
		operation:  "table row move before",
	}
	if after {
		plan.insertAt = anchor.Range.End
		plan.operation = "table row move after"
	}
	if !after && plan.movedRange.End == anchor.Range.Start ||
		after && anchor.Range.End == plan.movedRange.Start {
		return plan, true, nil
	}
	if plan.insertAt > plan.movedRange.Start && plan.insertAt < plan.movedRange.End {
		return tableRowMovePlan{}, false, ErrInvalidReplacement
	}
	plan.fragment = d.source[plan.movedRange.Start:plan.movedRange.End]
	return plan, false, nil
}

func (d *Document) validateTableRowMoveCandidate(candidate []byte, plan tableRowMovePlan, insertRange Range) error {
	candidateIndex, err := parseTableRowMutationCandidate(candidate)
	if err != nil {
		return err
	}
	if candidateIndex.rowCount != d.promotedTableRowCount() {
		return ErrInvalidReplacement
	}
	transforms := []patchTransform{
		{Range: plan.movedRange},
		{Range: insertRange, ReplacementLength: len(plan.fragment)},
	}
	if err := d.validateOriginalTableModelAfterPatches(candidate, candidateIndex, transforms, &plan.moved); err != nil {
		return err
	}
	movedOffset, ok := movedRangeCandidateOffset(plan.movedRange, plan.insertAt)
	if !ok {
		return ErrInvalidReplacement
	}
	return d.validateMovedTableRow(candidate, candidateIndex, plan.moved, plan.anchor, movedOffset, transforms)
}
