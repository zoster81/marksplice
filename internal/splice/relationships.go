package splice

import (
	"strings"

	"github.com/zoster81/marksplice/internal/parser"
)

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

// LinkFragmentStatus describes one relationship destination's applicability or
// exact resolution against this immutable snapshot.
type LinkFragmentStatus uint8

const (
	LinkFragmentNotApplicable LinkFragmentStatus = iota
	LinkFragmentResolved
	LinkFragmentMissing
	LinkFragmentAmbiguous
	LinkFragmentInvalid
)

// LinkRelationship is a read-only semantic projection over one parser-resolved
// outgoing link, image, or autolink usage.
type LinkRelationship struct {
	Kind                   LinkRelationshipKind
	SourceOffset           int
	SourceNodeID           NodeID
	HasSourceNode          bool
	Destination            string
	Title                  string
	HasTitle               bool
	Reference              string
	ReferenceForm          ReferenceForm
	HasReference           bool
	ReferenceDefinitionID  NodeID
	HasReferenceDefinition bool
	FragmentStatus         LinkFragmentStatus
	FragmentTarget         FragmentTarget
	HasFragmentTarget      bool
	AutoLinkEmail          bool
}

// LinkRelationships returns all parser-resolved outgoing link/image/autolink
// relationships in source order. The returned slice is caller-owned.
func (d *Document) LinkRelationships() ([]LinkRelationship, bool) {
	return MapLinkRelationships(d, func(relationship LinkRelationship) (LinkRelationship, bool) {
		return relationship, true
	})
}

// MapLinkRelationships projects source-ordered relationships directly into a
// caller-owned result. Conversion is synchronous and not retained; any semantic
// or conversion failure discards the partial result.
func MapLinkRelationships[T any](d *Document, convert func(LinkRelationship) (T, bool)) ([]T, bool) {
	if d == nil {
		return nil, true
	}
	if len(d.linkUsages) == 0 {
		return []T{}, true
	}
	sourceNodeCount := d.linkUsageSourceNodeCount()
	var sourceNodes map[linkUsageNodeKey]NodeID
	if sourceNodeCount != 0 {
		// Allocate outside the usage loop so small scratch maps can stay on the stack.
		sourceNodes = make(map[linkUsageNodeKey]NodeID, sourceNodeCount)
		d.indexLinkUsageSourceNodes(sourceNodes)
	}
	var definitionOwners map[referenceDefinitionKey]referenceDefinitionOwner
	var fragments fragmentCatalog
	result := make([]T, 0, len(d.linkUsages))
	for _, usage := range d.linkUsages {
		if usage.Form != parser.LinkUsageDirect && definitionOwners == nil {
			definitionOwners = d.referenceDefinitionOwners()
		}
		if strings.HasPrefix(usage.Destination, "#") && fragments == nil {
			fragments = d.fragmentCatalog()
		}
		relationship, ok := d.linkRelationship(usage, sourceNodes, definitionOwners, fragments)
		if !ok {
			return nil, false
		}
		converted, ok := convert(relationship)
		if !ok {
			return nil, false
		}
		result = append(result, converted)
	}
	return result, true
}

type linkUsageNodeKey struct {
	kind   Kind
	anchor int
}

func (d *Document) linkUsageSourceNodeCount() int {
	count := 0
	for index := range d.nodes {
		if isLinkUsageSourceNode(&d.nodes[index]) {
			count++
		}
	}
	return count
}

func (d *Document) indexLinkUsageSourceNodes(result map[linkUsageNodeKey]NodeID) {
	for index := range d.nodes {
		node := &d.nodes[index]
		if !isLinkUsageSourceNode(node) {
			continue
		}
		result[linkUsageNodeKey{kind: node.Kind, anchor: node.Anchor}] = node.ID
	}
}

func isLinkUsageSourceNode(node *Node) bool {
	return node.Editable && (node.Kind == KindInlineLink || node.Kind == KindImage || node.Kind == KindAutoLink)
}

type referenceDefinitionKey struct {
	labelKey    string
	destination string
	title       string
	hasTitle    bool
}

type referenceDefinitionOwner struct {
	id    NodeID
	count int
}

func (d *Document) referenceDefinitionOwners() map[referenceDefinitionKey]referenceDefinitionOwner {
	result := make(map[referenceDefinitionKey]referenceDefinitionOwner)
	for _, node := range d.nodes {
		if node.Kind != KindReferenceDefinition {
			continue
		}
		key := referenceDefinitionKey{
			labelKey:    referenceLabelKey(node.Label),
			destination: node.Destination,
			title:       node.Title,
			hasTitle:    node.HasTitle,
		}
		owner := result[key]
		owner.count++
		if owner.count == 1 && node.Editable {
			owner.id = node.ID
		} else {
			owner.id = ""
		}
		result[key] = owner
	}
	return result
}

func (d *Document) linkRelationship(usage parser.LinkUsage, sourceNodes map[linkUsageNodeKey]NodeID, definitions map[referenceDefinitionKey]referenceDefinitionOwner, fragments fragmentCatalog) (LinkRelationship, bool) {
	kind, nodeKind, referenceForm, hasReference, ok := relationshipKinds(usage)
	if !ok || usage.Anchor < 0 {
		return LinkRelationship{}, false
	}
	relationship := LinkRelationship{
		Kind:           kind,
		SourceOffset:   usage.Anchor,
		Destination:    usage.Destination,
		Title:          usage.Title,
		HasTitle:       usage.HasTitle,
		Reference:      usage.Reference,
		ReferenceForm:  referenceForm,
		HasReference:   hasReference,
		AutoLinkEmail:  usage.AutoLinkEmail,
		FragmentStatus: LinkFragmentNotApplicable,
	}
	if id, exists := sourceNodes[linkUsageNodeKey{kind: nodeKind, anchor: usage.Anchor}]; exists && !hasReference {
		relationship.SourceNodeID = id
		relationship.HasSourceNode = true
	}
	if hasReference {
		if id, ok := referenceDefinitionID(usage, definitions); ok {
			relationship.ReferenceDefinitionID = id
			relationship.HasReferenceDefinition = true
		}
	}
	if strings.HasPrefix(usage.Destination, "#") {
		target, status := resolveFragmentFromCatalog(usage.Destination, fragments)
		relationship.FragmentStatus = linkFragmentStatus(status)
		if status == fragmentResolutionResolved {
			relationship.FragmentTarget = target
			relationship.HasFragmentTarget = true
		}
	}
	return relationship, true
}

func relationshipKinds(usage parser.LinkUsage) (LinkRelationshipKind, Kind, ReferenceForm, bool, bool) {
	hasReference := usage.Form != parser.LinkUsageDirect
	form, ok := relationshipReferenceForm(usage.Form)
	if hasReference && !ok {
		return LinkRelationshipUnknown, KindUnknown, ReferenceFormUnknown, false, false
	}
	switch usage.Kind {
	case parser.KindInlineLink:
		if hasReference {
			return LinkRelationshipReferenceLink, KindInlineLink, form, true, true
		}
		return LinkRelationshipInlineLink, KindInlineLink, ReferenceFormUnknown, false, true
	case parser.KindImage:
		if hasReference {
			return LinkRelationshipReferenceImage, KindImage, form, true, true
		}
		return LinkRelationshipInlineImage, KindImage, ReferenceFormUnknown, false, true
	case parser.KindAutoLink:
		if hasReference {
			return LinkRelationshipUnknown, KindUnknown, ReferenceFormUnknown, false, false
		}
		return LinkRelationshipAutoLink, KindAutoLink, ReferenceFormUnknown, false, true
	default:
		return LinkRelationshipUnknown, KindUnknown, ReferenceFormUnknown, false, false
	}
}

func relationshipReferenceForm(form parser.LinkUsageForm) (ReferenceForm, bool) {
	switch form {
	case parser.LinkUsageFull:
		return ReferenceFormFull, true
	case parser.LinkUsageCollapsed:
		return ReferenceFormCollapsed, true
	case parser.LinkUsageShortcut:
		return ReferenceFormShortcut, true
	default:
		return ReferenceFormUnknown, false
	}
}

func referenceDefinitionID(usage parser.LinkUsage, definitions map[referenceDefinitionKey]referenceDefinitionOwner) (NodeID, bool) {
	key := referenceDefinitionKey{
		labelKey:    referenceLabelKey(usage.Reference),
		destination: usage.Destination,
		title:       usage.Title,
		hasTitle:    usage.HasTitle,
	}
	owner, ok := definitions[key]
	return owner.id, ok && owner.count == 1 && owner.id != ""
}

func linkFragmentStatus(status fragmentResolutionStatus) LinkFragmentStatus {
	switch status {
	case fragmentResolutionResolved:
		return LinkFragmentResolved
	case fragmentResolutionMissing:
		return LinkFragmentMissing
	case fragmentResolutionAmbiguous:
		return LinkFragmentAmbiguous
	case fragmentResolutionInvalid:
		return LinkFragmentInvalid
	default:
		return LinkFragmentInvalid
	}
}
