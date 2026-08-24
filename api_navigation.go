package marksplice

import "github.com/zoster81/marksplice/internal/splice"

// HeadingAnchor is an immutable GitHub-compatible anchor derived from one heading.
type HeadingAnchor struct {
	headingID NodeID
	value     string
}

// HeadingID returns the snapshot-scoped heading identity that owns this anchor.
func (a HeadingAnchor) HeadingID() NodeID { return a.headingID }

// Value returns the fragment value without a leading '#'.
func (a HeadingAnchor) Value() string { return a.value }

// FragmentTargetKind identifies one supported intra-document fragment target kind.
type FragmentTargetKind uint8

const (
	FragmentTargetUnknown FragmentTargetKind = iota
	FragmentTargetHeading
	FragmentTargetHTMLAnchor
)

// FragmentTarget is one uniquely resolved fragment destination in this snapshot.
type FragmentTarget struct {
	kind   FragmentTargetKind
	nodeID NodeID
	value  string
}

// Kind returns whether this target is a derived heading anchor or supported explicit HTML anchor.
func (t FragmentTarget) Kind() FragmentTargetKind { return t.kind }

// NodeID returns the snapshot-scoped node identity owning the fragment target.
func (t FragmentTarget) NodeID() NodeID { return t.nodeID }

// Value returns the resolved fragment value without a leading '#'.
func (t FragmentTarget) Value() string { return t.value }

// HeadingAnchors derives all promoted heading anchors in source order.
// Duplicate disambiguation is recomputed from the immutable snapshot on each call.
func (d *Document) HeadingAnchors() []HeadingAnchor {
	if d == nil || d.document == nil {
		return nil
	}
	internal := d.document.HeadingAnchors()
	result := make([]HeadingAnchor, len(internal))
	for index, anchor := range internal {
		result[index] = publicHeadingAnchor(anchor)
	}
	return result
}

// HeadingAnchor returns the derived anchor for one promoted heading.
func (d *Document) HeadingAnchor(id NodeID) (HeadingAnchor, bool) {
	if d == nil || d.document == nil {
		return HeadingAnchor{}, false
	}
	anchor, ok := d.document.HeadingAnchor(internalNodeID(id))
	if !ok {
		return HeadingAnchor{}, false
	}
	return publicHeadingAnchor(anchor), true
}

// ResolveFragment resolves an optional-leading-# URI fragment against heading-derived
// and supported explicit HTML anchors. Zero or multiple matches fail closed.
func (d *Document) ResolveFragment(fragment string) (FragmentTarget, bool) {
	if d == nil || d.document == nil {
		return FragmentTarget{}, false
	}
	target, ok := d.document.ResolveFragment(fragment)
	if !ok {
		return FragmentTarget{}, false
	}
	return publicFragmentTarget(target)
}

func (d *Document) fragmentResolver() func(string) (FragmentTarget, bool) {
	resolveStatus := d.fragmentStatusResolver()
	if resolveStatus == nil {
		return nil
	}
	return func(fragment string) (FragmentTarget, bool) {
		target, status := resolveStatus(fragment)
		return target, status == LinkFragmentResolved
	}
}

func (d *Document) fragmentStatusResolver() func(string) (FragmentTarget, LinkFragmentStatus) {
	if d == nil || d.document == nil {
		return nil
	}
	resolve := d.document.FragmentStatusResolver()
	return func(fragment string) (FragmentTarget, LinkFragmentStatus) {
		target, status := resolve(fragment)
		publicStatus, ok := publicLinkFragmentStatus(status)
		if !ok {
			return FragmentTarget{}, LinkFragmentInvalid
		}
		if publicStatus != LinkFragmentResolved {
			return FragmentTarget{}, publicStatus
		}
		publicTarget, ok := publicFragmentTarget(target)
		if !ok {
			return FragmentTarget{}, LinkFragmentInvalid
		}
		return publicTarget, publicStatus
	}
}

// ValidateFragment reports whether fragment uniquely resolves in this exact snapshot.
func (d *Document) ValidateFragment(fragment string) bool {
	_, ok := d.ResolveFragment(fragment)
	return ok
}

// GenerateTOC returns deterministic Markdown for the current section hierarchy.
// Generated output uses LF line endings; source synchronization preserves the target body's line-ending style.
func (d *Document) GenerateTOC() []byte {
	if d == nil || d.document == nil {
		return nil
	}
	return append([]byte(nil), d.document.GenerateTOC()...)
}

// TOCStale reports whether one explicitly designated section body is a recognized
// TOC shape that differs from the TOC derived from this snapshot. The second result
// is false when the target is missing or its direct body is not TOC-shaped.
func (d *Document) TOCStale(headingID NodeID) (bool, bool) {
	if d == nil || d.document == nil {
		return false, false
	}
	return d.document.TOCStale(internalNodeID(headingID))
}

// PrepareSyncTOC prepares source-preserving synchronization of an explicitly
// designated empty/TOC-shaped section body. Arbitrary section bodies fail closed.
func (d *Document) PrepareSyncTOC(headingID NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(headingID, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareSyncTOC(internalNodeID(headingID)))
}

func (d *Document) prepareSyncTOCs(headingIDs []NodeID) (ChangeSet, error) {
	if d == nil || d.document == nil {
		return ChangeSet{}, ErrSourceConflict
	}
	internal := make([]splice.NodeID, len(headingIDs))
	for index, headingID := range headingIDs {
		if _, err := d.promotedNode(headingID, splice.KindHeading, true); err != nil {
			return ChangeSet{}, err
		}
		internal[index] = internalNodeID(headingID)
	}
	return publicChangeSet(d.document.PrepareSyncTOCs(internal))
}

type managedTOCStatus struct {
	headingID  NodeID
	stale      bool
	recognized bool
}

func (d *Document) managedTOCStatuses(headingIDs []NodeID) []managedTOCStatus {
	if d == nil || d.document == nil {
		return nil
	}
	internal := make([]splice.NodeID, len(headingIDs))
	for index, headingID := range headingIDs {
		internal[index] = internalNodeID(headingID)
	}
	statuses := d.document.TOCStatuses(internal)
	result := make([]managedTOCStatus, len(statuses))
	for index, status := range statuses {
		result[index] = managedTOCStatus{
			headingID:  publicNodeID(status.HeadingID),
			stale:      status.Stale,
			recognized: status.Recognized,
		}
	}
	return result
}

func publicHeadingAnchor(anchor splice.HeadingAnchor) HeadingAnchor {
	return HeadingAnchor{headingID: publicNodeID(anchor.HeadingID), value: anchor.Value}
}

func publicFragmentTarget(target splice.FragmentTarget) (FragmentTarget, bool) {
	kind, ok := publicFragmentTargetKind(target.Kind)
	if !ok {
		return FragmentTarget{}, false
	}
	return FragmentTarget{kind: kind, nodeID: publicNodeID(target.NodeID), value: target.Value}, true
}

func publicFragmentTargetKind(kind splice.FragmentTargetKind) (FragmentTargetKind, bool) {
	switch kind {
	case splice.FragmentTargetHeading:
		return FragmentTargetHeading, true
	case splice.FragmentTargetHTMLAnchor:
		return FragmentTargetHTMLAnchor, true
	default:
		return FragmentTargetUnknown, false
	}
}
