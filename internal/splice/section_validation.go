package splice

import (
	"bytes"
	"fmt"
)

func (d *Document) sectionTarget(id NodeID) (Section, error) {
	target, err := d.editableTargetNode(id, KindHeading, "section")
	if err != nil {
		return Section{}, err
	}
	if !target.TopLevel {
		return Section{}, ErrInvalidTargetKind
	}
	section, ok := d.SectionByHeadingID(id)
	if !ok {
		return Section{}, ErrInvalidTargetKind
	}
	return section, nil
}

func parseSectionFragment(replacement []byte, level int) (*Document, error) {
	if len(replacement) == 0 {
		return nil, ErrInvalidReplacement
	}
	fragmentDocument, err := Parse(replacement)
	if err != nil {
		return nil, fmt.Errorf("%w: section fragment parse: %v", ErrInvalidReplacement, err)
	}
	root, ok := fragmentDocument.SectionAt(0)
	if !ok || root.Level != level || root.Range != (Range{Start: 0, End: len(replacement)}) {
		return nil, ErrInvalidReplacement
	}
	return fragmentDocument, nil
}

func sectionSubtreeEndIndex(sections []Section, start int, range_ Range) int {
	end := start
	for end < len(sections) && sectionStartsInside(sections[end], range_) {
		end++
	}
	return end
}

func (d *Document) validateSectionHeadingPatch(candidate []byte, patch Range, replacementLength int, survivors []Section) (*Document, error) {
	candidateDocument, err := parseSectionMutationCandidate(candidate)
	if err != nil {
		return nil, err
	}
	if candidateDocument.SectionCount() != len(survivors) {
		return nil, ErrInvalidReplacement
	}
	if err := d.validateOriginalSectionHeadings(candidate, candidateDocument, 0, survivors, patch, replacementLength); err != nil {
		return nil, err
	}
	return candidateDocument, nil
}

func parseSectionMutationCandidate(candidate []byte) (*Document, error) {
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return nil, fmt.Errorf("%w: section mutation candidate parse: %v", ErrInvalidReplacement, err)
	}
	return candidateDocument, nil
}

func (d *Document) validateOriginalSectionHeadings(candidate []byte, candidateDocument *Document, candidateStart int, sections []Section, patch Range, replacementLength int) error {
	for offset, section := range sections {
		originalHeading, ok := d.nodeByID(section.HeadingID)
		if !ok {
			return fmt.Errorf("%w: missing original section heading %q", ErrInvalidReplacement, section.HeadingID)
		}
		candidateSection, ok := candidateDocument.SectionAt(candidateStart + offset)
		if !ok {
			return ErrInvalidReplacement
		}
		candidateHeading, ok := candidateDocument.nodeByID(candidateSection.HeadingID)
		if !ok {
			return ErrInvalidReplacement
		}

		expectedRange, ok := rangeAfterPatch(originalHeading.Range, patch, replacementLength)
		if !ok {
			return ErrInvalidReplacement
		}
		expectedContentRange, ok := rangeAfterPatch(originalHeading.ContentRange, patch, replacementLength)
		if !ok {
			return ErrInvalidReplacement
		}
		if candidateHeading.Level != originalHeading.Level ||
			candidateHeading.HeadingStyle != originalHeading.HeadingStyle ||
			candidateHeading.Range != expectedRange ||
			candidateHeading.ContentRange != expectedContentRange {
			return ErrInvalidReplacement
		}
		if !bytes.Equal(d.source[originalHeading.Range.Start:originalHeading.Range.End], candidate[candidateHeading.Range.Start:candidateHeading.Range.End]) {
			return ErrInvalidReplacement
		}
	}
	return nil
}

func validateInsertedSectionFragment(candidate []byte, candidateDocument *Document, fragment []byte, fragmentDocument *Document, candidateStart, sourceOffset int) error {
	for i := 0; i < fragmentDocument.SectionCount(); i++ {
		fragmentSection, ok := fragmentDocument.SectionAt(i)
		if !ok {
			return ErrInvalidReplacement
		}
		candidateSection, ok := candidateDocument.SectionAt(candidateStart + i)
		if !ok {
			return ErrInvalidReplacement
		}
		if candidateSection.Level != fragmentSection.Level ||
			candidateSection.Range != shiftedRange(fragmentSection.Range, sourceOffset) ||
			candidateSection.BodyRange != shiftedRange(fragmentSection.BodyRange, sourceOffset) {
			return ErrInvalidReplacement
		}

		fragmentHeading, ok := fragmentDocument.nodeByID(fragmentSection.HeadingID)
		if !ok {
			return ErrInvalidReplacement
		}
		candidateHeading, ok := candidateDocument.nodeByID(candidateSection.HeadingID)
		if !ok {
			return ErrInvalidReplacement
		}
		if candidateHeading.Level != fragmentHeading.Level ||
			candidateHeading.HeadingStyle != fragmentHeading.HeadingStyle ||
			candidateHeading.Range != shiftedRange(fragmentHeading.Range, sourceOffset) ||
			candidateHeading.ContentRange != shiftedRange(fragmentHeading.ContentRange, sourceOffset) {
			return ErrInvalidReplacement
		}
		if !bytes.Equal(fragment[fragmentHeading.Range.Start:fragmentHeading.Range.End], candidate[candidateHeading.Range.Start:candidateHeading.Range.End]) {
			return ErrInvalidReplacement
		}
	}
	return nil
}

func (d *Document) validateMovedSectionSubtree(candidate []byte, candidateDocument *Document, movedStart, movedEnd, candidateStart, candidateOffset int) error {
	if movedStart < 0 || movedStart >= movedEnd || movedEnd > len(d.sections) || candidateStart < 0 {
		return ErrInvalidReplacement
	}
	movedSections := d.sections[movedStart:movedEnd]
	root := movedSections[0]
	subtreeLength := root.Range.End - root.Range.Start
	if !movedSectionBytesMatch(d.source, candidate, root.Range, candidateOffset) ||
		candidateStart > candidateDocument.SectionCount() || len(movedSections) > candidateDocument.SectionCount()-candidateStart {
		return ErrInvalidReplacement
	}

	for i, originalSection := range movedSections {
		candidateSection, ok := candidateDocument.SectionAt(candidateStart + i)
		if !ok || !sameMovedSectionRanges(originalSection, candidateSection, root.Range.Start, candidateOffset, subtreeLength) {
			return ErrInvalidReplacement
		}
		if err := d.validateMovedSectionHeading(candidate, candidateDocument, originalSection, candidateSection, root.Range.Start, candidateOffset, subtreeLength); err != nil {
			return err
		}
		if i != 0 && !d.sameMovedSectionParent(candidateDocument, originalSection, candidateSection, movedStart, candidateStart, i) {
			return ErrInvalidReplacement
		}
	}
	return nil
}

func movedSectionBytesMatch(original, candidate []byte, sourceRange Range, candidateOffset int) bool {
	length := sourceRange.End - sourceRange.Start
	if !sourceRange.Valid(len(original)) || length <= 0 || candidateOffset < 0 || candidateOffset > len(candidate) || length > len(candidate)-candidateOffset {
		return false
	}
	return bytes.Equal(original[sourceRange.Start:sourceRange.End], candidate[candidateOffset:candidateOffset+length])
}

func sameMovedSectionRanges(original, candidate Section, sourceStart, destinationStart, subtreeLength int) bool {
	expectedRange, ok := rebaseSectionRange(original.Range, sourceStart, destinationStart, subtreeLength)
	if !ok {
		return false
	}
	expectedBodyRange, ok := rebaseSectionRange(original.BodyRange, sourceStart, destinationStart, subtreeLength)
	return ok && candidate.Level == original.Level && candidate.Range == expectedRange && candidate.BodyRange == expectedBodyRange
}

func (d *Document) validateMovedSectionHeading(candidate []byte, candidateDocument *Document, originalSection, candidateSection Section, sourceStart, destinationStart, subtreeLength int) error {
	originalHeading, ok := d.nodeByID(originalSection.HeadingID)
	if !ok {
		return ErrInvalidReplacement
	}
	candidateHeading, ok := candidateDocument.nodeByID(candidateSection.HeadingID)
	if !ok {
		return ErrInvalidReplacement
	}
	expectedRange, ok := rebaseSectionRange(originalHeading.Range, sourceStart, destinationStart, subtreeLength)
	if !ok {
		return ErrInvalidReplacement
	}
	expectedContent, ok := rebaseSectionRange(originalHeading.ContentRange, sourceStart, destinationStart, subtreeLength)
	if !ok || candidateHeading.Level != originalHeading.Level || candidateHeading.HeadingStyle != originalHeading.HeadingStyle ||
		candidateHeading.Range != expectedRange || candidateHeading.ContentRange != expectedContent {
		return ErrInvalidReplacement
	}
	if !bytes.Equal(d.source[originalHeading.Range.Start:originalHeading.Range.End], candidate[candidateHeading.Range.Start:candidateHeading.Range.End]) {
		return ErrInvalidReplacement
	}
	return nil
}

func (d *Document) sameMovedSectionParent(candidateDocument *Document, originalSection, candidateSection Section, movedStart, candidateStart, relativeIndex int) bool {
	parentIndex, ok := d.sectionIndex[originalSection.ParentHeadingID]
	parentOffset := parentIndex - movedStart
	if !originalSection.HasParent || !ok || parentOffset < 0 || parentOffset >= relativeIndex {
		return false
	}
	candidateParent, ok := candidateDocument.SectionAt(candidateStart + parentOffset)
	return ok && candidateSection.HasParent && candidateSection.ParentHeadingID == candidateParent.HeadingID
}

func rebaseSectionRange(range_ Range, sourceStart, destinationStart, subtreeLength int) (Range, bool) {
	if range_.Start < sourceStart || range_.End < range_.Start {
		return Range{}, false
	}
	startOffset := range_.Start - sourceStart
	endOffset := range_.End - sourceStart
	if endOffset > subtreeLength || destinationStart < 0 || destinationStart > int(^uint(0)>>1)-endOffset {
		return Range{}, false
	}
	return Range{Start: destinationStart + startOffset, End: destinationStart + endOffset}, true
}

func sectionOrderAfterMove(sections []Section, movedStart, movedEnd int, anchorID NodeID, after bool) ([]Section, int, int, bool) {
	if movedStart < 0 || movedStart >= movedEnd || movedEnd > len(sections) {
		return nil, 0, 0, false
	}
	moved := append([]Section(nil), sections[movedStart:movedEnd]...)
	remaining := make([]Section, 0, len(sections)-len(moved))
	remaining = append(remaining, sections[:movedStart]...)
	remaining = append(remaining, sections[movedEnd:]...)

	anchorIndex := -1
	for i, section := range remaining {
		if section.HeadingID == anchorID {
			anchorIndex = i
			break
		}
	}
	if anchorIndex < 0 {
		return nil, 0, 0, false
	}
	insertIndex := anchorIndex
	if after {
		insertIndex = sectionSubtreeEndIndex(remaining, anchorIndex, remaining[anchorIndex].Range)
	}

	result := make([]Section, 0, len(sections))
	result = append(result, remaining[:insertIndex]...)
	result = append(result, moved...)
	result = append(result, remaining[insertIndex:]...)
	anchorCandidateIndex := anchorIndex
	if insertIndex <= anchorIndex {
		anchorCandidateIndex += len(moved)
	}
	return result, insertIndex, anchorCandidateIndex, true
}

func sameSectionOrder(left, right []Section) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].HeadingID != right[i].HeadingID {
			return false
		}
	}
	return true
}

func (d *Document) validateSectionHeadingSequence(candidate []byte, candidateDocument *Document, expected []Section) error {
	if candidateDocument.SectionCount() != len(expected) {
		return ErrInvalidReplacement
	}
	for i, section := range expected {
		originalHeading, ok := d.nodeByID(section.HeadingID)
		if !ok {
			return ErrInvalidReplacement
		}
		candidateSection, ok := candidateDocument.SectionAt(i)
		if !ok || candidateSection.Level != section.Level {
			return ErrInvalidReplacement
		}
		candidateHeading, ok := candidateDocument.nodeByID(candidateSection.HeadingID)
		if !ok {
			return ErrInvalidReplacement
		}
		originalContentOffset := originalHeading.ContentRange.Start - originalHeading.Range.Start
		candidateContentOffset := candidateHeading.ContentRange.Start - candidateHeading.Range.Start
		if candidateHeading.Level != originalHeading.Level ||
			candidateHeading.HeadingStyle != originalHeading.HeadingStyle ||
			candidateHeading.Range.End-candidateHeading.Range.Start != originalHeading.Range.End-originalHeading.Range.Start ||
			candidateContentOffset != originalContentOffset ||
			candidateHeading.ContentRange.End-candidateHeading.ContentRange.Start != originalHeading.ContentRange.End-originalHeading.ContentRange.Start {
			return ErrInvalidReplacement
		}
		if !bytes.Equal(d.source[originalHeading.Range.Start:originalHeading.Range.End], candidate[candidateHeading.Range.Start:candidateHeading.Range.End]) {
			return ErrInvalidReplacement
		}
	}
	return nil
}

func sectionStartsInside(section Section, range_ Range) bool {
	return section.Range.Start >= range_.Start && section.Range.Start < range_.End
}
