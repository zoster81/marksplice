package splice

import "github.com/zoster81/marksplice/internal/source"

// PrepareRemoveSection prepares removal of one complete derived section subtree.
func (d *Document) PrepareRemoveSection(id NodeID) (ChangeSet, error) {
	section, err := d.sectionTarget(id)
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
	section, err := d.sectionTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	fragmentDocument, err := parseSectionFragment(replacement, section.Level)
	if err != nil {
		return ChangeSet{}, err
	}

	sectionIndex, ok := d.sectionIndex[id]
	if !ok {
		return ChangeSet{}, ErrInvalidTargetKind
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
	section, err := d.sectionTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	fragmentDocument, err := parseSectionFragment(fragment, section.Level)
	if err != nil {
		return ChangeSet{}, err
	}

	sectionIndex, ok := d.sectionIndex[id]
	if !ok {
		return ChangeSet{}, ErrInvalidTargetKind
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
	parent, err := d.sectionTarget(id)
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
	parentIndex, ok := d.sectionIndex[id]
	if !ok {
		return ChangeSet{}, ErrInvalidTargetKind
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

func (d *Document) prepareMoveSection(id, anchorID NodeID, after bool) (ChangeSet, error) {
	if id == anchorID {
		return ChangeSet{}, ErrInvalidReplacement
	}
	moved, err := d.sectionTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	anchor, err := d.sectionTarget(anchorID)
	if err != nil {
		return ChangeSet{}, err
	}
	if moved.Level != anchor.Level {
		return ChangeSet{}, ErrInvalidReplacement
	}

	movedIndex, ok := d.sectionIndex[id]
	if !ok {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	movedEnd := sectionSubtreeEndIndex(d.sections, movedIndex, moved.Range)
	expected, movedCandidateIndex, anchorCandidateIndex, ok := sectionOrderAfterMove(d.sections, movedIndex, movedEnd, anchorID, after)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}

	operation := "section move before"
	insertAt := anchor.Range.Start
	if after {
		operation = "section move after"
		insertAt = anchor.Range.End
	}
	if sameSectionOrder(d.sections, expected) {
		return d.newChanges(nil, operation)
	}
	if insertAt > moved.Range.Start && insertAt < moved.Range.End {
		return ChangeSet{}, ErrInvalidReplacement
	}

	fragment := d.source[moved.Range.Start:moved.Range.End]
	fragmentDocument, err := parseSectionFragment(fragment, moved.Level)
	if err != nil {
		return ChangeSet{}, err
	}
	patches := []source.Patch{
		{Range: moved.Range},
		{Range: Range{Start: insertAt, End: insertAt}, Replacement: fragment},
	}
	change, candidate, err := d.prepareCandidateChanges(patches, operation)
	if err != nil {
		return ChangeSet{}, err
	}
	candidateDocument, err := parseSectionMutationCandidate(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if candidateDocument.SectionCount() != len(expected) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if err := d.validateSectionHeadingSequence(candidate, candidateDocument, expected); err != nil {
		return ChangeSet{}, err
	}

	movedOffset, ok := movedRangeCandidateOffset(moved.Range, insertAt)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if err := validateInsertedSectionFragment(candidate, candidateDocument, fragment, fragmentDocument, movedCandidateIndex, movedOffset); err != nil {
		return ChangeSet{}, err
	}
	candidateMoved, ok := candidateDocument.SectionAt(movedCandidateIndex)
	if !ok || candidateMoved.Range != (Range{Start: movedOffset, End: movedOffset + len(fragment)}) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	candidateAnchor, ok := candidateDocument.SectionAt(anchorCandidateIndex)
	if !ok || candidateMoved.HasParent != candidateAnchor.HasParent ||
		(candidateMoved.HasParent && candidateMoved.ParentHeadingID != candidateAnchor.ParentHeadingID) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return change, nil
}

// PrepareReplaceSectionBody prepares replacement of one section's direct body while preserving its section hierarchy.
func (d *Document) PrepareReplaceSectionBody(id NodeID, replacement []byte) (ChangeSet, error) {
	section, err := d.sectionTarget(id)
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
	sectionIndex, ok := d.sectionIndex[id]
	if !ok {
		return ChangeSet{}, ErrInvalidTargetKind
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
