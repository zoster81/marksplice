package marksplice

import "github.com/zoster81/marksplice/internal/splice"

// FragmentContinuityStatus classifies one previously resolved local-fragment
// relationship after a prepared change is applied to its exact source snapshot.
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

// FragmentContinuity describes one previously resolved local-fragment relationship
// and its correlated state in the final candidate produced by a prepared change.
type FragmentContinuity struct {
	before   LinkRelationship
	after    LinkRelationship
	hasAfter bool
	status   FragmentContinuityStatus
}

// Before returns the relationship from the document snapshot that prepared the change.
func (c FragmentContinuity) Before() LinkRelationship { return c.before }

// After returns the correlated candidate relationship when it still exists semantically.
func (c FragmentContinuity) After() (LinkRelationship, bool) { return c.after, c.hasAfter }

// Status returns the continuity classification for the correlated relationship.
func (c FragmentContinuity) Status() FragmentContinuityStatus { return c.status }

// LocalFragmentContinuity evaluates every previously resolved local-fragment
// relationship whose source syntax survives the prepared change. The change must be
// bound to this exact snapshot. Candidate parsing and target correspondence are owned
// by Marksplice; snapshot-scoped NodeID values are never compared across snapshots.
func (d *Document) LocalFragmentContinuity(change ChangeSet) ([]FragmentContinuity, error) {
	if d == nil || d.document == nil {
		return nil, ErrSourceConflict
	}
	internal, err := d.document.LocalFragmentContinuity(change.change)
	if err != nil {
		return nil, publicError(err)
	}
	result := make([]FragmentContinuity, len(internal))
	for index, continuity := range internal {
		before, ok := publicLinkRelationship(continuity.Before)
		if !ok {
			return nil, ErrInvalidReplacement
		}
		current := FragmentContinuity{
			before: before,
			status: publicFragmentContinuityStatus(continuity.Status),
		}
		if continuity.HasAfter {
			after, ok := publicLinkRelationship(continuity.After)
			if !ok {
				return nil, ErrInvalidReplacement
			}
			current.after = after
			current.hasAfter = true
		}
		result[index] = current
	}
	return result, nil
}

func publicFragmentContinuityStatus(status splice.FragmentContinuityStatus) FragmentContinuityStatus {
	switch status {
	case splice.FragmentContinuityPreserved:
		return FragmentContinuityPreserved
	case splice.FragmentContinuityRetargeted:
		return FragmentContinuityRetargeted
	case splice.FragmentContinuityTargetChanged:
		return FragmentContinuityTargetChanged
	case splice.FragmentContinuityMissing:
		return FragmentContinuityMissing
	case splice.FragmentContinuityAmbiguous:
		return FragmentContinuityAmbiguous
	case splice.FragmentContinuityInvalid:
		return FragmentContinuityInvalid
	default:
		return FragmentContinuityUnknown
	}
}
