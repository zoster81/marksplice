package source

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
)

var (
	ErrConflict           = errors.New("source snapshot conflict")
	ErrInvalidRange       = errors.New("invalid source range")
	ErrOverlappingPatches = errors.New("overlapping source patches")
)

// Fingerprint binds structural observations and prepared changes to one source snapshot.
type Fingerprint [sha256.Size]byte

// Sum returns the SHA-256 fingerprint of source.
func Sum(source []byte) Fingerprint {
	return sha256.Sum256(source)
}

// Range is a half-open byte range [Start, End).
type Range struct {
	Start int
	End   int
}

// Valid reports whether the range is ordered and contained in a source of total bytes.
func (r Range) Valid(total int) bool {
	return r.Start >= 0 && r.End >= r.Start && r.End <= total
}

// Patch replaces Range with Replacement.
type Patch struct {
	Range       Range
	Replacement []byte
}

// ChangeSet is an immutable prepared set of source-bound minimal patches.
type ChangeSet struct {
	fingerprint Fingerprint
	patches     []Patch
}

// NewChangeSet validates and copies patches against source.
func NewChangeSet(source []byte, patches []Patch) (ChangeSet, error) {
	prepared := make([]Patch, len(patches))
	for i, patch := range patches {
		if !patch.Range.Valid(len(source)) {
			return ChangeSet{}, fmt.Errorf("%w: patch %d has range [%d,%d) for source length %d", ErrInvalidRange, i, patch.Range.Start, patch.Range.End, len(source))
		}
		prepared[i] = Patch{
			Range:       patch.Range,
			Replacement: append([]byte(nil), patch.Replacement...),
		}
	}

	sort.Slice(prepared, func(i, j int) bool {
		if prepared[i].Range.Start == prepared[j].Range.Start {
			return prepared[i].Range.End < prepared[j].Range.End
		}
		return prepared[i].Range.Start < prepared[j].Range.Start
	})
	for i := 1; i < len(prepared); i++ {
		previous := prepared[i-1].Range
		current := prepared[i].Range
		if current.Start == previous.Start || current.Start < previous.End {
			return ChangeSet{}, fmt.Errorf("%w: [%d,%d) conflicts with [%d,%d)", ErrOverlappingPatches, previous.Start, previous.End, current.Start, current.End)
		}
	}

	return ChangeSet{
		fingerprint: Sum(source),
		patches:     prepared,
	}, nil
}

// ComposeChangeSets combines already-prepared changes from exactly one source
// snapshot. The combined change is revalidated as one non-overlapping patch set.
func ComposeChangeSets(source []byte, changes ...ChangeSet) (ChangeSet, error) {
	fingerprint := Sum(source)
	patches := make([]Patch, 0)
	for _, change := range changes {
		if change.fingerprint != fingerprint {
			return ChangeSet{}, ErrConflict
		}
		patches = append(patches, change.patches...)
	}
	return NewChangeSet(source, patches)
}

// Patches returns a defensive copy of the prepared source-coordinate patches.
// This method is internal to Marksplice's implementation boundary.
func (c ChangeSet) Patches() []Patch {
	patches := make([]Patch, len(c.patches))
	for index, patch := range c.patches {
		patches[index] = Patch{Range: patch.Range, Replacement: append([]byte(nil), patch.Replacement...)}
	}
	return patches
}

// Apply applies the prepared patches only when source matches the prepared snapshot.
func (c ChangeSet) Apply(source []byte) ([]byte, error) {
	if c.fingerprint != Sum(source) {
		return nil, ErrConflict
	}

	capacity := len(source)
	for _, patch := range c.patches {
		removed := patch.Range.End - patch.Range.Start
		var ok bool
		capacity, ok = adjustedResultLength(capacity, removed, len(patch.Replacement))
		if !ok {
			return nil, fmt.Errorf("%w: result length overflow", ErrInvalidRange)
		}
	}

	result := make([]byte, 0, capacity)
	cursor := 0
	for _, patch := range c.patches {
		result = append(result, source[cursor:patch.Range.Start]...)
		result = append(result, patch.Replacement...)
		cursor = patch.Range.End
	}
	result = append(result, source[cursor:]...)
	return result, nil
}

func adjustedResultLength(current, removed, replacement int) (int, bool) {
	if current < 0 || removed < 0 || replacement < 0 || removed > current {
		return 0, false
	}
	base := current - removed
	maxInt := int(^uint(0) >> 1)
	if replacement > maxInt-base {
		return 0, false
	}
	return base + replacement, true
}
