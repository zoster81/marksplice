package marksplice

import "github.com/zoster81/marksplice/internal/splice"

// LinkRelationshipKind identifies one semantic outgoing link/image relationship.
type LinkRelationshipKind uint8

const (
	LinkRelationshipUnknown LinkRelationshipKind = iota
	LinkRelationshipInlineLink
	LinkRelationshipReferenceLink
	LinkRelationshipInlineImage
	LinkRelationshipReferenceImage
	LinkRelationshipAutoLink
)

// ReferenceForm identifies one GFM reference-link/image source form.
type ReferenceForm uint8

const (
	ReferenceFormUnknown ReferenceForm = iota
	ReferenceFormFull
	ReferenceFormCollapsed
	ReferenceFormShortcut
)

// LinkFragmentStatus reports whether a relationship destination is an
// intra-document fragment and, when applicable, how it resolves in this snapshot.
type LinkFragmentStatus uint8

const (
	LinkFragmentNotApplicable LinkFragmentStatus = iota
	LinkFragmentResolved
	LinkFragmentMissing
	LinkFragmentAmbiguous
	LinkFragmentInvalid
)

// LinkRelationship is an immutable semantic outgoing link/image/autolink fact.
// It does not define generic source ownership or mutation authority.
type LinkRelationship struct {
	kind                   LinkRelationshipKind
	sourceOffset           int
	sourceNodeID           NodeID
	hasSourceNode          bool
	destination            string
	title                  string
	hasTitle               bool
	reference              string
	referenceForm          ReferenceForm
	hasReference           bool
	referenceDefinitionID  NodeID
	hasReferenceDefinition bool
	fragmentStatus         LinkFragmentStatus
	fragmentTarget         FragmentTarget
	hasFragmentTarget      bool
	email                  bool
}

// Kind returns the semantic relationship/source-form family.
func (r LinkRelationship) Kind() LinkRelationshipKind { return r.kind }

// SourceOffset returns the parser-proven byte offset where the relationship's
// source syntax starts. It is diagnostic/ordering metadata, not a mutation range.
func (r LinkRelationship) SourceOffset() int { return r.sourceOffset }

// SourceNodeID returns an existing promoted source node identity when this exact
// relationship already belongs to the ordinary public node model.
func (r LinkRelationship) SourceNodeID() (NodeID, bool) {
	return r.sourceNodeID, r.hasSourceNode
}

// Destination returns the parser-resolved semantic destination.
// Other-document paths and URLs remain opaque data; M99 does not access them.
func (r LinkRelationship) Destination() string { return r.destination }

// Title returns the parser-resolved title when one is present.
func (r LinkRelationship) Title() (string, bool) {
	return r.title, r.hasTitle
}

// Reference returns the parser-resolved reference label and source form for a
// reference link/image. Direct links/images and autolinks return false.
func (r LinkRelationship) Reference() (string, ReferenceForm, bool) {
	return r.reference, r.referenceForm, r.hasReference
}

// ReferenceDefinitionID returns the promoted single-line definition that can be
// proven to uniquely own this reference relationship. Unsupported or ambiguous
// definition ownership returns false without invalidating the relationship.
func (r LinkRelationship) ReferenceDefinitionID() (NodeID, bool) {
	return r.referenceDefinitionID, r.hasReferenceDefinition
}

// FragmentStatus reports M98-compatible intra-document fragment resolution for
// destinations beginning with '#'. Other destinations return NotApplicable.
func (r LinkRelationship) FragmentStatus() LinkFragmentStatus { return r.fragmentStatus }

// FragmentTarget returns the M98 target when FragmentStatus is Resolved.
func (r LinkRelationship) FragmentTarget() (FragmentTarget, bool) {
	return r.fragmentTarget, r.hasFragmentTarget
}

// IsEmail reports whether an autolink relationship was parser-classified as an
// email autolink. It is false for every non-autolink relationship.
func (r LinkRelationship) IsEmail() bool { return r.email }

// LinkRelationships returns all parser-resolved outgoing link/image/autolink
// relationships in source order. The returned slice is caller-owned and does not
// persist any relationship index or graph in the snapshot.
func (d *Document) LinkRelationships() []LinkRelationship {
	result, ok := d.linkRelationships()
	if !ok {
		return nil
	}
	return result
}

func (d *Document) linkRelationships() ([]LinkRelationship, bool) {
	if d == nil || d.document == nil {
		return nil, false
	}
	internal, ok := d.document.LinkRelationships()
	if !ok {
		return nil, false
	}
	result := make([]LinkRelationship, len(internal))
	for index, relationship := range internal {
		public, ok := publicLinkRelationship(relationship)
		if !ok {
			return nil, false
		}
		result[index] = public
	}
	return result, true
}

func publicLinkRelationship(relationship splice.LinkRelationship) (LinkRelationship, bool) {
	kind, ok := publicLinkRelationshipKind(relationship.Kind)
	if !ok {
		return LinkRelationship{}, false
	}
	form, ok := publicReferenceForm(relationship.ReferenceForm, relationship.HasReference)
	if !ok {
		return LinkRelationship{}, false
	}
	status, ok := publicLinkFragmentStatus(relationship.FragmentStatus)
	if !ok {
		return LinkRelationship{}, false
	}
	result := LinkRelationship{
		kind:                   kind,
		sourceOffset:           relationship.SourceOffset,
		destination:            relationship.Destination,
		title:                  relationship.Title,
		hasTitle:               relationship.HasTitle,
		reference:              relationship.Reference,
		referenceForm:          form,
		hasReference:           relationship.HasReference,
		hasReferenceDefinition: relationship.HasReferenceDefinition,
		fragmentStatus:         status,
		hasFragmentTarget:      relationship.HasFragmentTarget,
		email:                  relationship.AutoLinkEmail,
	}
	if relationship.HasSourceNode {
		result.sourceNodeID = publicNodeID(relationship.SourceNodeID)
		result.hasSourceNode = true
	}
	if relationship.HasReferenceDefinition {
		result.referenceDefinitionID = publicNodeID(relationship.ReferenceDefinitionID)
	}
	if relationship.HasFragmentTarget {
		fragmentTarget, ok := publicFragmentTarget(relationship.FragmentTarget)
		if !ok {
			return LinkRelationship{}, false
		}
		result.fragmentTarget = fragmentTarget
	}
	return result, true
}

func publicLinkRelationshipKind(kind splice.LinkRelationshipKind) (LinkRelationshipKind, bool) {
	switch kind {
	case splice.LinkRelationshipInlineLink:
		return LinkRelationshipInlineLink, true
	case splice.LinkRelationshipReferenceLink:
		return LinkRelationshipReferenceLink, true
	case splice.LinkRelationshipInlineImage:
		return LinkRelationshipInlineImage, true
	case splice.LinkRelationshipReferenceImage:
		return LinkRelationshipReferenceImage, true
	case splice.LinkRelationshipAutoLink:
		return LinkRelationshipAutoLink, true
	default:
		return LinkRelationshipUnknown, false
	}
}

func publicReferenceForm(form splice.ReferenceForm, hasReference bool) (ReferenceForm, bool) {
	if !hasReference {
		return ReferenceFormUnknown, form == splice.ReferenceFormUnknown
	}
	switch form {
	case splice.ReferenceFormFull:
		return ReferenceFormFull, true
	case splice.ReferenceFormCollapsed:
		return ReferenceFormCollapsed, true
	case splice.ReferenceFormShortcut:
		return ReferenceFormShortcut, true
	default:
		return ReferenceFormUnknown, false
	}
}

func publicLinkFragmentStatus(status splice.LinkFragmentStatus) (LinkFragmentStatus, bool) {
	switch status {
	case splice.LinkFragmentNotApplicable:
		return LinkFragmentNotApplicable, true
	case splice.LinkFragmentResolved:
		return LinkFragmentResolved, true
	case splice.LinkFragmentMissing:
		return LinkFragmentMissing, true
	case splice.LinkFragmentAmbiguous:
		return LinkFragmentAmbiguous, true
	case splice.LinkFragmentInvalid:
		return LinkFragmentInvalid, true
	default:
		return LinkFragmentNotApplicable, false
	}
}
