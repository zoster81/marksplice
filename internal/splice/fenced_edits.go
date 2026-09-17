package splice

import (
	"bytes"

	"github.com/zoster81/marksplice/internal/source"
)

// PrepareSetFencedBlockInfo prepares a source-preserving set, replacement, or
// clear of one source-proven top-level fenced block info string.
func (d *Document) PrepareSetFencedBlockInfo(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.targetNode(id, KindFencedCode)
	if err != nil {
		return ChangeSet{}, err
	}
	if !target.TopLevel {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	if bytes.ContainsAny(replacement, "\r\n") || !bytes.Equal(bytes.Trim(replacement, " \t"), replacement) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	original, _, _, ok := d.FencedBlockSource(id)
	if !ok {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	if change, ok := d.unchangedRangeChange(original.InfoRange, replacement); ok {
		return change, nil
	}
	change, candidate, err := d.prepareCandidateChange(original.InfoRange, replacement, "fenced block info replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateFencedBlockInfoReplacement(candidate, original, replacement); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func (d *Document) preparePopulateEmptyFencedBlock(target Node, replacement []byte) (ChangeSet, error) {
	if err := validateNonEmpty(replacement); err != nil {
		return ChangeSet{}, err
	}
	original, originalInfo, _, ok := d.FencedBlockSource(target.ID)
	if !ok || len(original.ContentRanges) != 0 || !original.Closed {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	insertAt := original.ClosingFenceRange.Start - original.ClosingIndent
	if insertAt < 0 || insertAt > len(d.source) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	lineEnding, ok := physicalLineEndingBefore(d.source, insertAt)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	fragment := make([]byte, 0, len(replacement)+len(lineEnding))
	fragment = append(fragment, replacement...)
	fragment = append(fragment, lineEnding...)
	patch := Range{Start: insertAt, End: insertAt}
	change, candidate, err := d.prepareCandidateChange(patch, fragment, "empty fenced block population")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validatePopulatedFencedBlock(candidate, original, originalInfo, len(fragment)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func validateFencedBlockInfoReplacement(candidate []byte, original source.FencedBlockMapping, replacement []byte) error {
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ErrInvalidReplacement
	}
	mapping, info, ok := fencedBlockCandidateAt(candidateDocument, original.OpeningFenceRange.Start)
	if !ok || info != string(replacement) {
		return ErrInvalidReplacement
	}
	delta := len(replacement) - (original.InfoRange.End - original.InfoRange.Start)
	if !sameFencedBlockEnvelope(original, mapping, delta) || !shiftedRangesEqual(original.ContentRanges, mapping.ContentRanges, delta) {
		return ErrInvalidReplacement
	}
	if len(replacement) == 0 {
		if mapping.InfoRange.Start != mapping.InfoRange.End {
			return ErrInvalidReplacement
		}
	} else if mapping.InfoRange != rangeWithLength(original.InfoRange.Start, len(replacement)) {
		return ErrInvalidReplacement
	}
	return nil
}

func validatePopulatedFencedBlock(candidate []byte, original source.FencedBlockMapping, originalInfo string, delta int) error {
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ErrInvalidReplacement
	}
	mapping, info, ok := fencedBlockCandidateAt(candidateDocument, original.OpeningFenceRange.Start)
	if !ok || len(mapping.ContentRanges) == 0 {
		return ErrInvalidReplacement
	}
	if info != originalInfo || !sameFencedBlockEnvelope(original, mapping, delta) || mapping.InfoRange != original.InfoRange {
		return ErrInvalidReplacement
	}
	return nil
}

func sameFencedBlockEnvelope(original, candidate source.FencedBlockMapping, delta int) bool {
	if candidate.Range != shiftedEnd(original.Range, delta) ||
		candidate.OpeningFenceRange != original.OpeningFenceRange ||
		candidate.FenceChar != original.FenceChar ||
		candidate.OpeningFenceLength != original.OpeningFenceLength ||
		candidate.ClosingFenceLength != original.ClosingFenceLength ||
		candidate.OpeningIndent != original.OpeningIndent ||
		candidate.ClosingIndent != original.ClosingIndent ||
		candidate.Closed != original.Closed {
		return false
	}
	if original.Closed && candidate.ClosingFenceRange != shiftedRange(original.ClosingFenceRange, delta) {
		return false
	}
	return true
}

func shiftedRangesEqual(original, candidate []source.Range, delta int) bool {
	if len(original) != len(candidate) {
		return false
	}
	for index := range original {
		if candidate[index] != shiftedRange(original[index], delta) {
			return false
		}
	}
	return true
}

func fencedBlockCandidateAt(document *Document, openingFenceStart int) (source.FencedBlockMapping, string, bool) {
	if document == nil {
		return source.FencedBlockMapping{}, "", false
	}
	for _, node := range document.nodes {
		if node.Kind != KindFencedCode || !node.TopLevel {
			continue
		}
		mapping, info, _, ok := document.FencedBlockSource(node.ID)
		if ok && mapping.OpeningFenceRange.Start == openingFenceStart {
			return mapping, info, true
		}
	}
	return source.FencedBlockMapping{}, "", false
}

func physicalLineEndingBefore(input []byte, offset int) ([]byte, bool) {
	if offset >= 2 && input[offset-2] == '\r' && input[offset-1] == '\n' {
		return []byte{'\r', '\n'}, true
	}
	if offset >= 1 && (input[offset-1] == '\n' || input[offset-1] == '\r') {
		return []byte{input[offset-1]}, true
	}
	return nil, false
}
