package splice

import "bytes"

type headingLevelPatch struct {
	range_      Range
	replacement []byte
	style       HeadingStyle
}

// PrepareSetHeadingLevel prepares a source-preserving heading-level change for one source-proven top-level heading.
func (d *Document) PrepareSetHeadingLevel(id NodeID, level int) (ChangeSet, error) {
	_, sectionIndex, err := d.sectionTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	target, ok := d.nodeByID(id)
	if !ok || level < 1 || level > 6 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if target.Level == level {
		content := d.source[target.ContentRange.Start:target.ContentRange.End]
		if change, ok := d.unchangedRangeChange(target.ContentRange, content); ok {
			return change, nil
		}
		return ChangeSet{}, ErrInvalidReplacement
	}

	plan, ok := planHeadingLevelPatch(d.source, target, level)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(plan.range_, plan.replacement, "heading level change")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateHeadingLevelCandidate(candidate, target, sectionIndex, level, plan); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func planHeadingLevelPatch(input []byte, target Node, level int) (headingLevelPatch, bool) {
	if !target.Range.Valid(len(input)) || !target.ContentRange.Valid(len(input)) || target.ContentRange.Start >= target.ContentRange.End {
		return headingLevelPatch{}, false
	}
	switch target.HeadingStyle {
	case HeadingStyleATX:
		return planATXHeadingLevelPatch(input, target, level)
	case HeadingStyleSetext:
		if level <= 2 {
			return planSetextUnderlinePatch(input, target, level)
		}
		return planSetextToATXPatch(input, target, level)
	default:
		return headingLevelPatch{}, false
	}
}

func planATXHeadingLevelPatch(input []byte, target Node, level int) (headingLevelPatch, bool) {
	markerStart := target.Range.Start
	for markerStart < target.ContentRange.Start && markerStart-target.Range.Start < 3 && input[markerStart] == ' ' {
		markerStart++
	}
	markerEnd := markerStart + target.Level
	if markerEnd > target.ContentRange.Start || !bytes.Equal(input[markerStart:markerEnd], bytes.Repeat([]byte{'#'}, target.Level)) {
		return headingLevelPatch{}, false
	}
	return headingLevelPatch{
		range_:      Range{Start: markerStart, End: markerEnd},
		replacement: bytes.Repeat([]byte{'#'}, level),
		style:       HeadingStyleATX,
	}, true
}

func planSetextUnderlinePatch(input []byte, target Node, level int) (headingLevelPatch, bool) {
	underlineStart := lastPhysicalLineStart(input, target.Range)
	if underlineStart < target.Range.Start || underlineStart >= target.Range.End {
		return headingLevelPatch{}, false
	}
	markerStart := underlineStart
	for markerStart < target.Range.End && markerStart-underlineStart < 3 && input[markerStart] == ' ' {
		markerStart++
	}
	oldMarker := byte('=')
	if target.Level == 2 {
		oldMarker = '-'
	}
	markerEnd := markerStart
	for markerEnd < target.Range.End && input[markerEnd] == oldMarker {
		markerEnd++
	}
	if markerEnd == markerStart {
		return headingLevelPatch{}, false
	}
	newMarker := byte('=')
	if level == 2 {
		newMarker = '-'
	}
	return headingLevelPatch{
		range_:      Range{Start: markerStart, End: markerEnd},
		replacement: bytes.Repeat([]byte{newMarker}, markerEnd-markerStart),
		style:       HeadingStyleSetext,
	}, true
}

func planSetextToATXPatch(input []byte, target Node, level int) (headingLevelPatch, bool) {
	content := input[target.ContentRange.Start:target.ContentRange.End]
	if bytes.ContainsAny(content, "\r\n") {
		return headingLevelPatch{}, false
	}
	lineEnd := target.ContentRange.End
	for lineEnd < target.Range.End && input[lineEnd] != '\r' && input[lineEnd] != '\n' {
		lineEnd++
	}
	if lineEnd >= target.Range.End {
		return headingLevelPatch{}, false
	}
	prefix := input[target.Range.Start:target.ContentRange.Start]
	if len(prefix) > 3 {
		return headingLevelPatch{}, false
	}
	for _, value := range prefix {
		if value != ' ' {
			return headingLevelPatch{}, false
		}
	}
	suffix := input[target.ContentRange.End:lineEnd]
	replacement := make([]byte, 0, len(prefix)+level+1+len(content)+len(suffix))
	replacement = append(replacement, prefix...)
	replacement = append(replacement, bytes.Repeat([]byte{'#'}, level)...)
	replacement = append(replacement, ' ')
	replacement = append(replacement, content...)
	replacement = append(replacement, suffix...)
	return headingLevelPatch{range_: target.Range, replacement: replacement, style: HeadingStyleATX}, true
}

func lastPhysicalLineStart(input []byte, range_ Range) int {
	if !range_.Valid(len(input)) || range_.Start >= range_.End {
		return -1
	}
	start := range_.End
	for start > range_.Start {
		previous := input[start-1]
		if previous == '\n' || previous == '\r' {
			break
		}
		start--
	}
	return start
}

func (d *Document) validateHeadingLevelCandidate(candidate []byte, target Node, targetSectionIndex, level int, plan headingLevelPatch) error {
	candidateDocument, err := parseSectionMutationCandidate(candidate)
	if err != nil || candidateDocument.SectionCount() != len(d.sections) {
		return ErrInvalidReplacement
	}
	candidateSection, ok := candidateDocument.SectionAt(targetSectionIndex)
	if !ok || candidateSection.Level != level {
		return ErrInvalidReplacement
	}
	candidateHeading, ok := candidateDocument.nodeByID(candidateSection.HeadingID)
	if !ok || candidateHeading.Level != level || candidateHeading.HeadingStyle != plan.style || candidateHeading.Range.Start != target.Range.Start {
		return ErrInvalidReplacement
	}
	if !sameHeadingContentBytes(d.source, target, candidate, candidateHeading) || candidateHeading.HeadingText != target.HeadingText {
		return ErrInvalidReplacement
	}
	if err := d.validateOtherHeadingsAfterLevelPatch(candidate, candidateDocument, targetSectionIndex, plan); err != nil {
		return err
	}
	return nil
}

func sameHeadingContentBytes(original []byte, originalHeading Node, candidate []byte, candidateHeading Node) bool {
	if !originalHeading.ContentRange.Valid(len(original)) || !candidateHeading.ContentRange.Valid(len(candidate)) {
		return false
	}
	return bytes.Equal(
		original[originalHeading.ContentRange.Start:originalHeading.ContentRange.End],
		candidate[candidateHeading.ContentRange.Start:candidateHeading.ContentRange.End],
	)
}

func (d *Document) validateOtherHeadingsAfterLevelPatch(candidate []byte, candidateDocument *Document, targetIndex int, plan headingLevelPatch) error {
	for index, section := range d.sections {
		if index == targetIndex {
			continue
		}
		originalHeading, ok := d.nodeByID(section.HeadingID)
		if !ok {
			return ErrInvalidReplacement
		}
		candidateSection, ok := candidateDocument.SectionAt(index)
		if !ok {
			return ErrInvalidReplacement
		}
		candidateHeading, ok := candidateDocument.nodeByID(candidateSection.HeadingID)
		if !ok {
			return ErrInvalidReplacement
		}
		expectedRange, ok := rangeAfterPatch(originalHeading.Range, plan.range_, len(plan.replacement))
		if !ok {
			return ErrInvalidReplacement
		}
		expectedContent, ok := rangeAfterPatch(originalHeading.ContentRange, plan.range_, len(plan.replacement))
		if !ok || candidateHeading.Level != originalHeading.Level || candidateHeading.HeadingStyle != originalHeading.HeadingStyle ||
			candidateHeading.Range != expectedRange || candidateHeading.ContentRange != expectedContent {
			return ErrInvalidReplacement
		}
		if !bytes.Equal(d.source[originalHeading.Range.Start:originalHeading.Range.End], candidate[expectedRange.Start:expectedRange.End]) {
			return ErrInvalidReplacement
		}
	}
	return nil
}
