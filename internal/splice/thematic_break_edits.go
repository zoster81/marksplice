package splice

// PrepareRemoveThematicBreak prepares exact removal of one promoted top-level thematic-break physical line.
func (d *Document) PrepareRemoveThematicBreak(id NodeID) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindThematicBreak, "thematic break")
	if err != nil {
		return ChangeSet{}, err
	}
	removeRange := target.ThematicBreakSource.LineRange
	if !removeRange.Valid(len(d.source)) || removeRange.Start >= removeRange.End || !rangesOverlap(target.Range, removeRange) {
		return ChangeSet{}, ErrInvalidReplacement
	}

	change, candidate, err := d.prepareCandidateChange(removeRange, nil, "thematic break removal")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateNodeSurvivorsAfterRemoval(candidate, removeRange); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}
