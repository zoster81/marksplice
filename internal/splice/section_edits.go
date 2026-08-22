package splice

// PrepareRemoveSection prepares removal of one complete derived section subtree.
func (d *Document) PrepareRemoveSection(id NodeID) (ChangeSet, error) {
	section, _, err := d.sectionTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}

	change, candidate, err := d.prepareCandidateChange(section.Range, nil, "section removal")
	if err != nil {
		return ChangeSet{}, err
	}
	survivors := make([]Section, 0, len(d.sections))
	for _, candidateSection := range d.sections {
		if !sectionStartsInside(candidateSection, section.Range) {
			survivors = append(survivors, candidateSection)
		}
	}
	if _, err := d.validateSectionHeadingPatch(candidate, section.Range, 0, survivors); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareReplaceSection prepares replacement of one complete derived section subtree.
func (d *Document) PrepareReplaceSection(id NodeID, replacement []byte) (ChangeSet, error) {
	section, sectionIndex, err := d.sectionTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	fragmentDocument, err := parseSectionFragment(replacement, section.Level)
	if err != nil {
		return ChangeSet{}, err
	}

	subtreeEnd := sectionSubtreeEndIndex(d.sections, sectionIndex, section.Range)

	change, candidate, err := d.prepareCandidateChange(section.Range, replacement, "section replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateDocument, err := parseSectionMutationCandidate(candidate)
	if err != nil {
		return ChangeSet{}, err
	}

	fragmentCount := fragmentDocument.SectionCount()
	expectedCount := sectionIndex + fragmentCount + (len(d.sections) - subtreeEnd)
	if candidateDocument.SectionCount() != expectedCount {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if err := d.validateOriginalSectionHeadings(candidate, candidateDocument, 0, d.sections[:sectionIndex], section.Range, len(replacement)); err != nil {
		return ChangeSet{}, err
	}
	if err := validateInsertedSectionFragment(candidate, candidateDocument, replacement, fragmentDocument, sectionIndex, section.Range.Start); err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateOriginalSectionHeadings(candidate, candidateDocument, sectionIndex+fragmentCount, d.sections[subtreeEnd:], section.Range, len(replacement)); err != nil {
		return ChangeSet{}, err
	}

	candidateRoot, ok := candidateDocument.SectionAt(sectionIndex)
	if !ok || candidateRoot.Range != (Range{Start: section.Range.Start, End: section.Range.Start + len(replacement)}) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return change, nil
}

// PrepareInsertSectionBefore prepares insertion of one sibling section subtree immediately before the target section.
func (d *Document) PrepareInsertSectionBefore(id NodeID, fragment []byte) (ChangeSet, error) {
	return d.prepareInsertSection(id, fragment, false)
}

// PrepareInsertSectionAfter prepares insertion of one sibling section subtree immediately after the target section subtree.
func (d *Document) PrepareInsertSectionAfter(id NodeID, fragment []byte) (ChangeSet, error) {
	return d.prepareInsertSection(id, fragment, true)
}

func (d *Document) prepareInsertSection(id NodeID, fragment []byte, after bool) (ChangeSet, error) {
	section, sectionIndex, err := d.sectionTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	fragmentDocument, err := parseSectionFragment(fragment, section.Level)
	if err != nil {
		return ChangeSet{}, err
	}

	insertionIndex := sectionIndex
	insertAt := section.Range.Start
	operation := "section insertion before"
	if after {
		insertionIndex = sectionSubtreeEndIndex(d.sections, sectionIndex, section.Range)
		insertAt = section.Range.End
		operation = "section insertion after"
	}
	patch := Range{Start: insertAt, End: insertAt}

	change, candidate, err := d.prepareCandidateChange(patch, fragment, operation)
	if err != nil {
		return ChangeSet{}, err
	}
	candidateDocument, err := parseSectionMutationCandidate(candidate)
	if err != nil {
		return ChangeSet{}, err
	}

	fragmentCount := fragmentDocument.SectionCount()
	if candidateDocument.SectionCount() != len(d.sections)+fragmentCount {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if err := d.validateOriginalSectionHeadings(candidate, candidateDocument, 0, d.sections[:insertionIndex], patch, len(fragment)); err != nil {
		return ChangeSet{}, err
	}
	if err := validateInsertedSectionFragment(candidate, candidateDocument, fragment, fragmentDocument, insertionIndex, insertAt); err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateOriginalSectionHeadings(candidate, candidateDocument, insertionIndex+fragmentCount, d.sections[insertionIndex:], patch, len(fragment)); err != nil {
		return ChangeSet{}, err
	}

	candidateRoot, ok := candidateDocument.SectionAt(insertionIndex)
	if !ok || candidateRoot.Range != (Range{Start: insertAt, End: insertAt + len(fragment)}) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return change, nil
}

// PrepareAppendSectionChild prepares appending one direct child section subtree to a parent section.
func (d *Document) PrepareAppendSectionChild(id NodeID, fragment []byte) (ChangeSet, error) {
	parent, parentIndex, err := d.sectionTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	if parent.Level >= 6 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	fragmentDocument, err := parseSectionFragment(fragment, parent.Level+1)
	if err != nil {
		return ChangeSet{}, err
	}
	insertionIndex := sectionSubtreeEndIndex(d.sections, parentIndex, parent.Range)
	insertAt := parent.Range.End
	patch := Range{Start: insertAt, End: insertAt}

	change, candidate, err := d.prepareCandidateChange(patch, fragment, "append section child")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateDocument, err := parseSectionMutationCandidate(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	fragmentCount := fragmentDocument.SectionCount()
	if candidateDocument.SectionCount() != len(d.sections)+fragmentCount {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if err := d.validateOriginalSectionHeadings(candidate, candidateDocument, 0, d.sections[:insertionIndex], patch, len(fragment)); err != nil {
		return ChangeSet{}, err
	}
	if err := validateInsertedSectionFragment(candidate, candidateDocument, fragment, fragmentDocument, insertionIndex, insertAt); err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateOriginalSectionHeadings(candidate, candidateDocument, insertionIndex+fragmentCount, d.sections[insertionIndex:], patch, len(fragment)); err != nil {
		return ChangeSet{}, err
	}

	candidateParent, ok := candidateDocument.SectionAt(parentIndex)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	candidateChild, ok := candidateDocument.SectionAt(insertionIndex)
	if !ok || !candidateChild.HasParent || candidateChild.ParentHeadingID != candidateParent.HeadingID ||
		candidateChild.Range != (Range{Start: insertAt, End: insertAt + len(fragment)}) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return change, nil
}

// PrepareMoveSectionBefore prepares moving one complete section subtree immediately before a same-level anchor section.
func (d *Document) PrepareMoveSectionBefore(id, anchorID NodeID) (ChangeSet, error) {
	return d.prepareMoveSection(id, anchorID, false)
}

// PrepareMoveSectionAfter prepares moving one complete section subtree immediately after a same-level anchor subtree.
func (d *Document) PrepareMoveSectionAfter(id, anchorID NodeID) (ChangeSet, error) {
	return d.prepareMoveSection(id, anchorID, true)
}

type sectionMovePlan struct {
	moved                Section
	movedIndex           int
	movedEnd             int
	expected             []Section
	movedCandidateIndex  int
	anchorCandidateIndex int
	insertAt             int
	operation            string
	fragment             []byte
}

func (d *Document) prepareMoveSection(id, anchorID NodeID, after bool) (ChangeSet, error) {
	plan, noOp, err := d.planSectionMove(id, anchorID, after)
	if err != nil {
		return ChangeSet{}, err
	}
	if noOp {
		return d.newChanges(nil, plan.operation)
	}
	change, candidate, _, err := d.prepareMoveCandidate(plan.moved.Range, plan.insertAt, plan.fragment, plan.operation)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateSectionMoveCandidate(candidate, plan); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func (d *Document) planSectionMove(id, anchorID NodeID, after bool) (sectionMovePlan, bool, error) {
	if id == anchorID {
		return sectionMovePlan{}, false, ErrInvalidReplacement
	}
	moved, movedIndex, err := d.sectionTarget(id)
	if err != nil {
		return sectionMovePlan{}, false, err
	}
	anchor, _, err := d.sectionTarget(anchorID)
	if err != nil {
		return sectionMovePlan{}, false, err
	}
	if moved.Level != anchor.Level {
		return sectionMovePlan{}, false, ErrInvalidReplacement
	}
	movedEnd := sectionSubtreeEndIndex(d.sections, movedIndex, moved.Range)
	expected, movedCandidateIndex, anchorCandidateIndex, ok := sectionOrderAfterMove(d.sections, movedIndex, movedEnd, anchorID, after)
	if !ok {
		return sectionMovePlan{}, false, ErrInvalidReplacement
	}
	plan := sectionMovePlan{
		moved:                moved,
		movedIndex:           movedIndex,
		movedEnd:             movedEnd,
		expected:             expected,
		movedCandidateIndex:  movedCandidateIndex,
		anchorCandidateIndex: anchorCandidateIndex,
		insertAt:             anchor.Range.Start,
		operation:            "section move before",
	}
	if after {
		plan.insertAt = anchor.Range.End
		plan.operation = "section move after"
	}
	if sameSectionOrder(d.sections, expected) {
		return plan, true, nil
	}
	if plan.insertAt > moved.Range.Start && plan.insertAt < moved.Range.End {
		return sectionMovePlan{}, false, ErrInvalidReplacement
	}
	plan.fragment = d.source[moved.Range.Start:moved.Range.End]
	return plan, false, nil
}

func (d *Document) validateSectionMoveCandidate(candidate []byte, plan sectionMovePlan) error {
	candidateDocument, err := parseSectionMutationCandidate(candidate)
	if err != nil {
		return err
	}
	if candidateDocument.SectionCount() != len(plan.expected) {
		return ErrInvalidReplacement
	}
	if err := d.validateSectionHeadingSequence(candidate, candidateDocument, plan.expected); err != nil {
		return err
	}
	movedOffset, ok := movedRangeCandidateOffset(plan.moved.Range, plan.insertAt)
	if !ok {
		return ErrInvalidReplacement
	}
	if err := d.validateMovedSectionSubtree(candidate, candidateDocument, plan.movedIndex, plan.movedEnd, plan.movedCandidateIndex, movedOffset); err != nil {
		return err
	}
	candidateMoved, ok := candidateDocument.SectionAt(plan.movedCandidateIndex)
	if !ok || candidateMoved.Range != (Range{Start: movedOffset, End: movedOffset + len(plan.fragment)}) {
		return ErrInvalidReplacement
	}
	candidateAnchor, ok := candidateDocument.SectionAt(plan.anchorCandidateIndex)
	if !ok || candidateMoved.HasParent != candidateAnchor.HasParent ||
		(candidateMoved.HasParent && candidateMoved.ParentHeadingID != candidateAnchor.ParentHeadingID) {
		return ErrInvalidReplacement
	}
	return nil
}

// PrepareReplaceSectionBody prepares replacement of one section's direct body while preserving its section hierarchy.
func (d *Document) PrepareReplaceSectionBody(id NodeID, replacement []byte) (ChangeSet, error) {
	section, sectionIndex, err := d.sectionTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}

	change, candidate, err := d.prepareCandidateChange(section.BodyRange, replacement, "section body replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateDocument, err := d.validateSectionHeadingPatch(candidate, section.BodyRange, len(replacement), d.sections)
	if err != nil {
		return ChangeSet{}, err
	}
	candidateSection, ok := candidateDocument.SectionAt(sectionIndex)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	expectedBodyRange := Range{Start: section.BodyRange.Start, End: section.BodyRange.Start + len(replacement)}
	if candidateSection.BodyRange != expectedBodyRange {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return change, nil
}
