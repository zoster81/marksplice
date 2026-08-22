package splice

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/zoster81/marksplice/internal/parser"
	goldmarkparser "github.com/zoster81/marksplice/internal/parser/goldmark"
	"github.com/zoster81/marksplice/internal/source"
)

func (d *Document) targetNode(id NodeID, expected Kind) (Node, error) {
	target, ok := d.nodeByID(id)
	if !ok {
		return Node{}, ErrNodeNotFound
	}
	if target.Kind != expected {
		return Node{}, ErrInvalidTargetKind
	}
	return target, nil
}

func (d *Document) editableTargetNode(id NodeID, expected Kind, description string) (Node, error) {
	target, err := d.targetNode(id, expected)
	if err != nil {
		return Node{}, err
	}
	if !target.Editable {
		return Node{}, fmt.Errorf("%w: %s source shape is not editable", ErrInvalidTargetKind, description)
	}
	return target, nil
}

func validateNonEmpty(replacement []byte) error {
	if len(replacement) == 0 {
		return ErrInvalidReplacement
	}
	return nil
}

func validateNonEmptySingleLine(replacement []byte) error {
	if err := validateNonEmpty(replacement); err != nil {
		return err
	}
	if bytes.ContainsAny(replacement, "\r\n") {
		return ErrInvalidReplacement
	}
	return nil
}

func (d *Document) newChange(range_ Range, replacement []byte, operation string) (ChangeSet, error) {
	return d.newChanges([]source.Patch{{Range: range_, Replacement: replacement}}, operation)
}

func (d *Document) newChanges(patches []source.Patch, operation string) (ChangeSet, error) {
	change, err := source.NewChangeSet(d.source, patches)
	if err != nil {
		return ChangeSet{}, fmt.Errorf("prepare %s: %w", operation, err)
	}
	return change, nil
}

func (d *Document) prepareCandidateChange(range_ Range, replacement []byte, operation string) (ChangeSet, []byte, error) {
	return d.prepareCandidateChanges([]source.Patch{{Range: range_, Replacement: replacement}}, operation)
}

func (d *Document) prepareCandidateChanges(patches []source.Patch, operation string) (ChangeSet, []byte, error) {
	change, err := d.newChanges(patches, operation)
	if err != nil {
		return ChangeSet{}, nil, err
	}
	candidate, err := change.Apply(d.source)
	if err != nil {
		return ChangeSet{}, nil, fmt.Errorf("render %s candidate: %w", operation, err)
	}
	return change, candidate, nil
}

func (d *Document) prepareMoveCandidate(moved Range, insertAt int, fragment []byte, operation string) (ChangeSet, []byte, Range, error) {
	insertRange := rangeWithLength(insertAt, 0)
	patches := []source.Patch{
		{Range: moved},
		{Range: insertRange, Replacement: fragment},
	}
	change, candidate, err := d.prepareCandidateChanges(patches, operation)
	return change, candidate, insertRange, err
}

func parseCandidate(candidate []byte) ([]parser.Node, error) {
	observations, err := goldmarkparser.New().Parse(candidate)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidReplacement, err)
	}
	return observations, nil
}

func shiftedEnd(range_ Range, delta int) Range {
	return Range{Start: range_.Start, End: range_.End + delta}
}

func rangeWithLength(start, length int) Range {
	return Range{Start: start, End: start + length}
}

type patchTransform struct {
	Range             Range
	ReplacementLength int
}

func rangeAfterPatch(range_, patch Range, replacementLength int) (Range, bool) {
	return rangeAfterPatches(range_, []patchTransform{{Range: patch, ReplacementLength: replacementLength}})
}

func rangeAfterPatches(range_ Range, patches []patchTransform) (Range, bool) {
	ordered, ok := orderedPatchTransforms(patches)
	if !ok {
		return Range{}, false
	}
	return rangeAfterOrderedPatches(range_, ordered)
}

func rangeAfterOrderedPatches(range_ Range, ordered []patchTransform) (Range, bool) {
	delta := 0
	for _, patch := range ordered {
		switch {
		case range_.End <= patch.Range.Start:
			return shiftedRange(range_, delta), true
		case range_.Start >= patch.Range.End:
			delta += patch.ReplacementLength - (patch.Range.End - patch.Range.Start)
		default:
			return Range{}, false
		}
	}
	return shiftedRange(range_, delta), true
}

func orderedPatchTransforms(patches []patchTransform) ([]patchTransform, bool) {
	needsSort := false
	for i, patch := range patches {
		if patch.ReplacementLength < 0 || patch.Range.Start < 0 || patch.Range.End < patch.Range.Start {
			return nil, false
		}
		if i != 0 && patchTransformLess(patch, patches[i-1]) {
			needsSort = true
		}
	}

	ordered := patches
	if needsSort {
		ordered = append([]patchTransform(nil), patches...)
		sort.Slice(ordered, func(i, j int) bool {
			return patchTransformLess(ordered[i], ordered[j])
		})
	}
	for i := 1; i < len(ordered); i++ {
		previous := ordered[i-1].Range
		current := ordered[i].Range
		if current.Start == previous.Start || current.Start < previous.End {
			return nil, false
		}
	}
	return ordered, true
}

func patchTransformLess(left, right patchTransform) bool {
	if left.Range.Start != right.Range.Start {
		return left.Range.Start < right.Range.Start
	}
	return left.Range.End < right.Range.End
}

func movedRangeCandidateOffset(moved Range, insertAt int) (int, bool) {
	length := moved.End - moved.Start
	switch {
	case insertAt <= moved.Start:
		return insertAt, true
	case insertAt >= moved.End:
		return insertAt - length, true
	default:
		return 0, false
	}
}

func shiftedRange(range_ Range, delta int) Range {
	return Range{Start: range_.Start + delta, End: range_.End + delta}
}
