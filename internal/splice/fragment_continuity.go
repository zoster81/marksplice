package splice

import (
	"sort"

	"github.com/zoster81/marksplice/internal/source"
)

// FragmentContinuityStatus classifies one previously resolved local-fragment
// relationship after applying a source-bound prepared change.
type FragmentContinuityStatus uint8

const (
	FragmentContinuityUnknown FragmentContinuityStatus = iota
	FragmentContinuityPreserved
	FragmentContinuityRetargeted
	FragmentContinuityTargetChanged
	FragmentContinuityMissing
	FragmentContinuityAmbiguous
	FragmentContinuityInvalid
)

// FragmentContinuity correlates one previously resolved local-fragment relationship
// with the final candidate produced by a prepared change.
type FragmentContinuity struct {
	Before   LinkRelationship
	After    LinkRelationship
	HasAfter bool
	Status   FragmentContinuityStatus
}

// LocalFragmentContinuity evaluates previously resolved local-fragment relationships
// against the final candidate. Relationships whose source syntax is intentionally
// removed or replaced are not reported.
func (d *Document) LocalFragmentContinuity(change ChangeSet) ([]FragmentContinuity, error) {
	if d == nil {
		return nil, ErrSourceConflict
	}
	candidateSource, err := change.Apply(d.source)
	if err != nil {
		return nil, ErrSourceConflict
	}
	candidate, err := Parse(candidateSource)
	if err != nil {
		return nil, ErrInvalidReplacement
	}
	index, ok := newContinuityChangeIndex(change)
	if !ok {
		return nil, ErrInvalidReplacement
	}
	before, ok := d.LinkRelationships()
	if !ok {
		return nil, ErrInvalidReplacement
	}
	after, ok := candidate.LinkRelationships()
	if !ok {
		return nil, ErrInvalidReplacement
	}
	return correlateFragmentContinuity(d, candidate, before, after, index), nil
}

type continuityRelationshipKey struct {
	offset int
	kind   LinkRelationshipKind
}

func correlateFragmentContinuity(original, candidate *Document, before, after []LinkRelationship, index continuityChangeIndex) []FragmentContinuity {
	afterByKey := make(map[continuityRelationshipKey]LinkRelationship, len(after))
	for _, relationship := range after {
		key := continuityRelationshipKey{offset: relationship.SourceOffset, kind: relationship.Kind}
		if _, exists := afterByKey[key]; exists {
			delete(afterByKey, key)
			continue
		}
		afterByKey[key] = relationship
	}

	result := make([]FragmentContinuity, 0)
	for _, relationship := range before {
		if relationship.FragmentStatus != LinkFragmentResolved || !relationship.HasFragmentTarget {
			continue
		}
		offset, survives := index.mapOffset(relationship.SourceOffset)
		if !survives {
			continue
		}
		current, exists := afterByKey[continuityRelationshipKey{offset: offset, kind: relationship.Kind}]
		if !exists {
			result = append(result, FragmentContinuity{Before: relationship, Status: FragmentContinuityInvalid})
			continue
		}
		result = append(result, FragmentContinuity{
			Before:   relationship,
			After:    current,
			HasAfter: true,
			Status:   fragmentContinuityStatus(original, candidate, relationship, current, index),
		})
	}
	return result
}

func fragmentContinuityStatus(original, candidate *Document, before, after LinkRelationship, index continuityChangeIndex) FragmentContinuityStatus {
	switch after.FragmentStatus {
	case LinkFragmentMissing:
		return FragmentContinuityMissing
	case LinkFragmentAmbiguous:
		return FragmentContinuityAmbiguous
	case LinkFragmentInvalid:
		return FragmentContinuityInvalid
	case LinkFragmentNotApplicable:
		if before.Destination != after.Destination {
			return FragmentContinuityRetargeted
		}
		return FragmentContinuityInvalid
	case LinkFragmentResolved:
		return resolvedFragmentContinuityStatus(original, candidate, before, after, index)
	default:
		return FragmentContinuityInvalid
	}
}

func resolvedFragmentContinuityStatus(original, candidate *Document, before, after LinkRelationship, index continuityChangeIndex) FragmentContinuityStatus {
	if !after.HasFragmentTarget {
		return FragmentContinuityInvalid
	}
	if before.Destination != after.Destination {
		return FragmentContinuityRetargeted
	}
	if sameFragmentTargetAfterChange(original, candidate, before.FragmentTarget, after.FragmentTarget, index) {
		return FragmentContinuityPreserved
	}
	return FragmentContinuityTargetChanged
}

func sameFragmentTargetAfterChange(original, candidate *Document, before, after FragmentTarget, index continuityChangeIndex) bool {
	if before.Kind != after.Kind {
		return false
	}
	beforeNode, ok := original.nodeByID(before.NodeID)
	if !ok {
		return false
	}
	afterNode, ok := candidate.nodeByID(after.NodeID)
	if !ok || afterNode.Kind != beforeNode.Kind {
		return false
	}
	expected, ok := index.mapTargetRange(before.NodeID, beforeNode.Range)
	return ok && afterNode.Range == expected
}

type continuityChangeIndex struct {
	patches         []source.Patch
	prefixDelta     []int
	relocations     []continuityRelocation
	correspondences map[NodeID]Range
}

type continuityRelocation struct {
	source         Range
	candidateStart int
}

func newContinuityChangeIndex(change ChangeSet) (continuityChangeIndex, bool) {
	index := continuityChangeIndex{patches: change.Patches()}
	index.prefixDelta = make([]int, len(index.patches)+1)
	for patchIndex, patch := range index.patches {
		removed := patch.Range.End - patch.Range.Start
		index.prefixDelta[patchIndex+1] = index.prefixDelta[patchIndex] + len(patch.Replacement) - removed
	}
	if !index.buildRelocations(change.relocations) || !index.buildCorrespondences(change.correspondences) {
		return continuityChangeIndex{}, false
	}
	return index, true
}

func (i *continuityChangeIndex) buildRelocations(relocations []changeRelocation) bool {
	i.relocations = make([]continuityRelocation, len(relocations))
	for index, relocation := range relocations {
		candidateStart, ok := i.insertionOffset(relocation.insertAt)
		if !ok {
			return false
		}
		i.relocations[index] = continuityRelocation{source: relocation.source, candidateStart: candidateStart}
	}
	sort.Slice(i.relocations, func(left, right int) bool {
		return i.relocations[left].source.Start < i.relocations[right].source.Start
	})
	for index := 1; index < len(i.relocations); index++ {
		if i.relocations[index-1].source.End > i.relocations[index].source.Start {
			return false
		}
	}
	return true
}

func (i *continuityChangeIndex) buildCorrespondences(correspondences []changeCorrespondence) bool {
	if len(correspondences) == 0 {
		return true
	}
	i.correspondences = make(map[NodeID]Range, len(correspondences))
	for _, correspondence := range correspondences {
		if _, exists := i.correspondences[correspondence.sourceID]; exists {
			return false
		}
		candidateRange, ok := i.composedCorrespondenceRange(correspondence)
		if !ok {
			return false
		}
		i.correspondences[correspondence.sourceID] = candidateRange
	}
	return true
}

func (i continuityChangeIndex) composedCorrespondenceRange(correspondence changeCorrespondence) (Range, bool) {
	first := sort.Search(len(i.patches), func(index int) bool {
		return i.patches[index].Range.End > correspondence.sourceRange.Start ||
			(i.patches[index].Range.Start == i.patches[index].Range.End &&
				i.patches[index].Range.Start >= correspondence.sourceRange.Start)
	})
	owned := continuityOwnedPatchRanges(correspondence.patchRanges)
	for patchIndex := first; patchIndex < len(i.patches); patchIndex++ {
		patch := i.patches[patchIndex]
		if patch.Range.Start >= correspondence.sourceRange.End {
			break
		}
		if patchTouchesRange(patch, correspondence.sourceRange) && !owned[patch.Range] {
			return Range{}, false
		}
	}
	before := sort.Search(len(i.patches), func(index int) bool {
		return i.patches[index].Range.Start >= correspondence.sourceRange.Start
	})
	return shiftedRange(correspondence.candidateRange, i.prefixDelta[before]), true
}

func continuityOwnedPatchRanges(ranges []Range) map[Range]bool {
	result := make(map[Range]bool, len(ranges))
	for _, range_ := range ranges {
		result[range_] = true
	}
	return result
}

func patchTouchesRange(patch source.Patch, range_ Range) bool {
	if patch.Range.Start == patch.Range.End {
		return patch.Range.Start >= range_.Start && patch.Range.Start < range_.End
	}
	return patch.Range.Start < range_.End && patch.Range.End > range_.Start
}

func (i continuityChangeIndex) mapOffset(offset int) (int, bool) {
	if relocation, ok := i.relocationForOffset(offset); ok {
		return relocation.candidateStart + offset - relocation.source.Start, true
	}
	return i.mapUnmovedOffset(offset)
}

func (i continuityChangeIndex) mapTargetRange(id NodeID, range_ Range) (Range, bool) {
	if correspondence, ok := i.correspondences[id]; ok {
		return correspondence, true
	}
	if relocation, ok := i.relocationForRange(range_); ok {
		return Range{
			Start: relocation.candidateStart + range_.Start - relocation.source.Start,
			End:   relocation.candidateStart + range_.End - relocation.source.Start,
		}, true
	}
	start, ok := i.mapUnmovedOffset(range_.Start)
	if !ok {
		return Range{}, false
	}
	end, ok := i.mapUnmovedOffset(range_.End)
	if !ok || end < start {
		return Range{}, false
	}
	return Range{Start: start, End: end}, true
}

func (i continuityChangeIndex) mapUnmovedOffset(offset int) (int, bool) {
	if offset < 0 {
		return 0, false
	}
	after := sort.Search(len(i.patches), func(index int) bool {
		return i.patches[index].Range.Start > offset
	})
	if after > 0 {
		previous := i.patches[after-1].Range
		if offset < previous.End {
			return 0, false
		}
	}
	return offset + i.prefixDelta[after], true
}

func (i continuityChangeIndex) insertionOffset(insertAt int) (int, bool) {
	if insertAt < 0 {
		return 0, false
	}
	index := sort.Search(len(i.patches), func(index int) bool {
		return i.patches[index].Range.Start >= insertAt
	})
	if index >= len(i.patches) {
		return 0, false
	}
	patch := i.patches[index]
	if patch.Range.Start != insertAt || patch.Range.End != insertAt {
		return 0, false
	}
	return insertAt + i.prefixDelta[index], true
}

func (i continuityChangeIndex) relocationForOffset(offset int) (continuityRelocation, bool) {
	after := sort.Search(len(i.relocations), func(index int) bool {
		return i.relocations[index].source.Start > offset
	})
	if after == 0 {
		return continuityRelocation{}, false
	}
	relocation := i.relocations[after-1]
	return relocation, offset >= relocation.source.Start && offset < relocation.source.End
}

func (i continuityChangeIndex) relocationForRange(range_ Range) (continuityRelocation, bool) {
	relocation, ok := i.relocationForOffset(range_.Start)
	if !ok || range_.End > relocation.source.End {
		return continuityRelocation{}, false
	}
	return relocation, true
}
