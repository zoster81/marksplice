package splice

import (
	"bytes"
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
)

// PrepareRemoveParagraph prepares removal of one complete promoted top-level paragraph physical block line span.
func (d *Document) PrepareRemoveParagraph(id NodeID) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindParagraph, "paragraph")
	if err != nil {
		return ChangeSet{}, err
	}
	owned, ok := paragraphRemovalRange(d.source, target.Range)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(owned, nil, "paragraph removal")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateNodeSurvivorsAfterRemoval(candidate, owned); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareInsertParagraphBefore prepares insertion of one top-level paragraph before a promoted paragraph while Marksplice owns the required blank-line separator.
func (d *Document) PrepareInsertParagraphBefore(id NodeID, content []byte) (ChangeSet, error) {
	return d.prepareInsertParagraph(id, content, false)
}

// PrepareInsertParagraphAfter prepares insertion of one top-level paragraph after a promoted paragraph while Marksplice owns the required blank-line separator.
func (d *Document) PrepareInsertParagraphAfter(id NodeID, content []byte) (ChangeSet, error) {
	return d.prepareInsertParagraph(id, content, true)
}

func (d *Document) prepareInsertParagraph(id NodeID, content []byte, after bool) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindParagraph, "paragraph")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateParagraphReplacement(content); err != nil {
		return ChangeSet{}, err
	}
	fragmentDocument, err := Parse(content)
	if err != nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	owned, eol, ok := paragraphPhysicalRange(d.source, target.Range)
	if !ok || len(eol) == 0 {
		return ChangeSet{}, ErrInvalidReplacement
	}

	insertAt := owned.Start
	fragment := make([]byte, 0, len(content)+2*len(eol))
	contentOffset := insertAt
	operation := "paragraph insertion before"
	if after {
		insertAt = owned.End
		contentOffset = insertAt + len(eol)
		fragment = append(fragment, eol...)
		fragment = append(fragment, content...)
		fragment = append(fragment, eol...)
		operation = "paragraph insertion after"
	} else {
		fragment = append(fragment, content...)
		fragment = append(fragment, eol...)
		fragment = append(fragment, eol...)
	}
	patch := Range{Start: insertAt, End: insertAt}
	change, candidate, err := d.prepareCandidateChange(patch, fragment, operation)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateParagraphInsertionCandidate(candidate, fragmentDocument, patch, len(fragment), contentOffset); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func paragraphPhysicalRange(input []byte, paragraph Range) (Range, []byte, bool) {
	if !paragraph.Valid(len(input)) || paragraph.Start >= paragraph.End {
		return Range{}, nil, false
	}
	start := paragraph.Start
	for start > 0 && input[start-1] != '\n' && input[start-1] != '\r' {
		start--
	}
	end := paragraph.End
	eol := physicalLineEndingAt(input, end)
	if len(eol) != 0 {
		end += len(eol)
	}
	return Range{Start: start, End: end}, eol, true
}

func paragraphRemovalRange(input []byte, paragraph Range) (Range, bool) {
	owned, _, ok := paragraphPhysicalRange(input, paragraph)
	if !ok {
		return Range{}, false
	}
	if owned.End >= len(input) {
		return owned, true
	}
	lineEnd, next, _ := physicalLine(input, owned.End)
	if len(bytes.Trim(input[owned.End:lineEnd], " \t")) != 0 {
		return owned, true
	}
	owned.End = next
	return owned, true
}

func physicalLineEndingAt(input []byte, offset int) []byte {
	if offset < 0 || offset >= len(input) {
		return nil
	}
	if input[offset] == '\r' {
		if offset+1 < len(input) && input[offset+1] == '\n' {
			return input[offset : offset+2]
		}
		return input[offset : offset+1]
	}
	if input[offset] == '\n' {
		return input[offset : offset+1]
	}
	return nil
}

func (d *Document) validateParagraphInsertionCandidate(candidate []byte, fragmentDocument *Document, patch Range, replacementLength, contentOffset int) error {
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return fmt.Errorf("%w: parse paragraph insertion candidate: %v", ErrInvalidReplacement, err)
	}
	if len(candidateDocument.nodes) != len(d.nodes)+len(fragmentDocument.nodes) {
		return ErrInvalidReplacement
	}
	patches := []patchTransform{{Range: patch, ReplacementLength: replacementLength}}
	matched := make([]bool, len(candidateDocument.nodes))
	for _, original := range d.nodes {
		index := matchingInsertionSurvivor(d, candidateDocument, matched, original, patches)
		if index < 0 {
			return ErrInvalidReplacement
		}
		matched[index] = true
	}
	if !originalLinkUsagesSurviveInsertion(d.linkUsages, candidateDocument.linkUsages, patches) {
		return ErrInvalidReplacement
	}
	return validateInsertedParagraphNodes(fragmentDocument, candidateDocument, matched, contentOffset)
}

func matchingInsertionSurvivor(originalDocument, candidateDocument *Document, matched []bool, original Node, patches []patchTransform) int {
	expectedRange, ok := rangeAfterPatches(original.Range, patches)
	if !ok {
		return -1
	}
	expectedContent, ok := rangeAfterPatches(original.ContentRange, patches)
	if !ok {
		return -1
	}
	for index, candidate := range candidateDocument.nodes {
		if matched[index] || candidate.Kind != original.Kind || candidate.Range != expectedRange || candidate.ContentRange != expectedContent {
			continue
		}
		if sameRemovalSurvivorSemantics(original, candidate) && samePatchedSurvivorAnchors(original, candidate, patches) &&
			samePatchedBlockquoteSource(originalDocument, candidateDocument, original, candidate, patches) {
			return index
		}
	}
	return -1
}

func samePatchedSurvivorAnchors(original, candidate Node, patches []patchTransform) bool {
	return sameRemovalListAnchors(original, candidate, patches) &&
		sameRemovalTableAnchors(original, candidate, patches) &&
		sameRemovalInlineAnchor(original, candidate, patches)
}

func samePatchedBlockquoteSource(originalDocument, candidateDocument *Document, original, candidate Node, patches []patchTransform) bool {
	if original.Kind != KindBlockquote {
		return true
	}
	originalSource, originalOK := originalDocument.blockquoteSource(original)
	candidateSource, candidateOK := candidateDocument.blockquoteSource(candidate)
	if !originalOK || !candidateOK {
		return false
	}
	expectedLine, ok := rangeAfterPatches(originalSource.LineRange, patches)
	if !ok || candidateSource.LineRange != expectedLine {
		return false
	}
	expectedMarker, ok := rangeAfterPatches(originalSource.MarkerRange, patches)
	if !ok || candidateSource.MarkerRange != expectedMarker || len(candidateSource.ContentRanges) != len(originalSource.ContentRanges) {
		return false
	}
	for index, content := range originalSource.ContentRanges {
		expected, ok := rangeAfterPatches(content, patches)
		if !ok || candidateSource.ContentRanges[index] != expected {
			return false
		}
	}
	return true
}

func originalLinkUsagesSurviveInsertion(original, candidate []parser.LinkUsage, patches []patchTransform) bool {
	candidateIndex := 0
	for _, usage := range original {
		expectedAnchor, ok := anchorAfterPatches(usage.Anchor, patches)
		if !ok {
			return false
		}
		expected := usage
		expected.Anchor = expectedAnchor
		for candidateIndex < len(candidate) && candidate[candidateIndex] != expected {
			candidateIndex++
		}
		if candidateIndex == len(candidate) {
			return false
		}
		candidateIndex++
	}
	return true
}

func validateInsertedParagraphNodes(fragmentDocument, candidateDocument *Document, matched []bool, offset int) error {
	fragmentIndex := 0
	for candidateIndex, candidate := range candidateDocument.nodes {
		if matched[candidateIndex] {
			continue
		}
		if fragmentIndex >= len(fragmentDocument.nodes) {
			return ErrInvalidReplacement
		}
		fragment := fragmentDocument.nodes[fragmentIndex]
		if !sameInsertedParagraphNode(fragment, candidate, offset) {
			return ErrInvalidReplacement
		}
		fragmentIndex++
	}
	if fragmentIndex != len(fragmentDocument.nodes) {
		return ErrInvalidReplacement
	}
	return nil
}

func sameInsertedParagraphNode(fragment, candidate Node, offset int) bool {
	if fragment.Kind != candidate.Kind || shiftedRange(fragment.Range, offset) != candidate.Range || shiftedRange(fragment.ContentRange, offset) != candidate.ContentRange ||
		!sameRemovalSurvivorSemantics(fragment, candidate) {
		return false
	}
	switch fragment.Kind {
	case KindInlineLink, KindImage, KindAutoLink, KindCodeSpan, KindEmphasis, KindStrong:
		return candidate.Anchor == fragment.Anchor+offset
	default:
		return true
	}
}
