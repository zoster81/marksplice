package splice

// PrepareRemoveBlockquote prepares exact removal of one complete promoted top-level blockquote container.
func (d *Document) PrepareRemoveBlockquote(id NodeID) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindBlockquote, "blockquote")
	if err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := d.blockquoteSource(target)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	removeRange := mapping.LineRange
	if !removeRange.Valid(len(d.source)) || removeRange.Start >= removeRange.End || !rangesOverlap(target.Range, removeRange) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(removeRange, nil, "blockquote removal")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateNodeSurvivorsAfterRemoval(candidate, removeRange); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}
