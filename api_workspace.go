package marksplice

import (
	"fmt"
	"strings"
)

// WorkspaceResolutionKind describes how the caller classifies one non-local
// relationship while validating an explicit workspace document set.
type WorkspaceResolutionKind uint8

const (
	WorkspaceResolutionUnknown WorkspaceResolutionKind = iota
	// WorkspaceResolutionIgnore leaves the relationship outside workspace validation.
	// This is appropriate for intentionally external or otherwise out-of-scope targets.
	WorkspaceResolutionIgnore
	// WorkspaceResolutionResolved maps the relationship to a document already present
	// in the explicit caller-provided set, optionally with a target fragment.
	WorkspaceResolutionResolved
	// WorkspaceResolutionMissing states that the caller expected a workspace document
	// target but that target is absent from the explicit document set.
	WorkspaceResolutionMissing
)

// WorkspaceResolution is one caller-authorized classification of a non-local
// relationship. Target identities are opaque caller data; Marksplice performs no I/O.
type WorkspaceResolution struct {
	Kind     WorkspaceResolutionKind
	Target   DocumentKey
	Fragment string
}

// WorkspaceResolver classifies one non-local link relationship for workspace validation.
// It is invoked synchronously, never concurrently during one ValidateWorkspace call,
// and is never retained.
type WorkspaceResolver func(source DocumentKey, relationship LinkRelationship) WorkspaceResolution

// ManagedTOC identifies one caller-designated section whose body is expected to use
// the conservative managed TOC shape recognized by TOCStale and PrepareSyncTOC.
type ManagedTOC struct {
	Document  DocumentKey
	HeadingID NodeID
}

// WorkspaceValidationOptions supplies explicit validation authority beyond the
// document set itself. Empty Roots disables orphan/reachability diagnostics.
type WorkspaceValidationOptions struct {
	Roots       []DocumentKey
	ManagedTOCs []ManagedTOC
}

type managedTOCKey struct {
	document DocumentKey
	heading  NodeID
}

// WorkspaceDiagnosticKind identifies one deterministic workspace validation finding.
type WorkspaceDiagnosticKind uint8

const (
	WorkspaceDiagnosticUnknown WorkspaceDiagnosticKind = iota
	WorkspaceDiagnosticMissingFragment
	WorkspaceDiagnosticAmbiguousFragment
	WorkspaceDiagnosticInvalidFragment
	WorkspaceDiagnosticMissingDocument
	WorkspaceDiagnosticUnresolvedReference
	WorkspaceDiagnosticOrphanDocument
	WorkspaceDiagnosticStaleGeneratedIndex
	WorkspaceDiagnosticUnrecognizedGeneratedIndex
)

// UnresolvedReference is immutable semantic metadata for one conservative explicit
// full/collapsed reference whose parser context contains no matching definition.
type UnresolvedReference struct {
	reference string
	form      ReferenceForm
	image     bool
}

// Reference returns the unresolved reference label.
func (r UnresolvedReference) Reference() string { return r.reference }

// Form returns the explicit full or collapsed reference form.
func (r UnresolvedReference) Form() ReferenceForm { return r.form }

// IsImage reports whether the unresolved reference is an image reference.
func (r UnresolvedReference) IsImage() bool { return r.image }

// WorkspaceDiagnostic is one immutable workspace validation finding. Accessors expose
// only metadata meaningful for the diagnostic kind; absent metadata returns false.
type WorkspaceDiagnostic struct {
	kind                   WorkspaceDiagnosticKind
	relationship           LinkRelationship
	hasRelationship        bool
	sourceOffset           int
	hasSourceOffset        bool
	sourceDocument         DocumentKey
	hasSourceDocument      bool
	targetDocument         DocumentKey
	hasTargetDocument      bool
	fragment               string
	hasFragment            bool
	unresolvedReference    UnresolvedReference
	hasUnresolvedReference bool
	nodeID                 NodeID
	hasNodeID              bool
}

// Kind returns the diagnostic category.
func (d WorkspaceDiagnostic) Kind() WorkspaceDiagnosticKind { return d.kind }

// Relationship returns the link relationship associated with this diagnostic.
func (d WorkspaceDiagnostic) Relationship() (LinkRelationship, bool) {
	return d.relationship, d.hasRelationship
}

// SourceOffset returns source-order diagnostic metadata when one exact source anchor exists.
func (d WorkspaceDiagnostic) SourceOffset() (int, bool) {
	return d.sourceOffset, d.hasSourceOffset
}

// SourceDocument returns the caller-defined source document identity associated with this finding.
func (d WorkspaceDiagnostic) SourceDocument() (DocumentKey, bool) {
	return d.sourceDocument, d.hasSourceDocument
}

// TargetDocument returns the caller-defined target document identity associated with this finding.
func (d WorkspaceDiagnostic) TargetDocument() (DocumentKey, bool) {
	return d.targetDocument, d.hasTargetDocument
}

// Fragment returns the fragment associated with a fragment/document diagnostic.
func (d WorkspaceDiagnostic) Fragment() (string, bool) {
	return d.fragment, d.hasFragment
}

// UnresolvedReference returns conservative explicit unresolved reference metadata.
// Shortcut bracket text is never reported because it is ambiguous with ordinary text.
func (d WorkspaceDiagnostic) UnresolvedReference() (UnresolvedReference, bool) {
	return d.unresolvedReference, d.hasUnresolvedReference
}

// NodeID returns the snapshot-local node associated with a generated-index diagnostic.
func (d WorkspaceDiagnostic) NodeID() (NodeID, bool) { return d.nodeID, d.hasNodeID }

// WorkspaceRepair is one deterministic safe repair prepared through the ordinary
// snapshot-bound mutation machinery.
type WorkspaceRepair struct {
	document DocumentKey
	change   ChangeSet
}

// Document returns the caller-defined document key to which the repair applies.
func (r WorkspaceRepair) Document() DocumentKey { return r.document }

// Change returns the ordinary source-bound prepared change for this repair.
func (r WorkspaceRepair) Change() ChangeSet { return r.change }

// WorkspaceRepairPlan is an immutable ordered set of provably safe repairs.
type WorkspaceRepairPlan struct {
	repairs []WorkspaceRepair
}

// Repairs returns caller-owned repair values in deterministic planning order.
func (p WorkspaceRepairPlan) Repairs() []WorkspaceRepair {
	if len(p.repairs) == 0 {
		return nil
	}
	return append([]WorkspaceRepair(nil), p.repairs...)
}

// WorkspaceReport combines the resolved DocumentGraph, deterministic diagnostics, and
// conservative repair plan for one explicit validation run.
type WorkspaceReport struct {
	graph       *DocumentGraph
	diagnostics []WorkspaceDiagnostic
	repairPlan  WorkspaceRepairPlan
}

// Graph returns the immutable document graph produced by this validation run.
func (r *WorkspaceReport) Graph() *DocumentGraph {
	if r == nil {
		return nil
	}
	return r.graph
}

// Diagnostics returns caller-owned diagnostics in deterministic validation order.
func (r *WorkspaceReport) Diagnostics() []WorkspaceDiagnostic {
	if r == nil || len(r.diagnostics) == 0 {
		return nil
	}
	return append([]WorkspaceDiagnostic(nil), r.diagnostics...)
}

// RepairPlan returns the immutable conservative repair plan.
func (r *WorkspaceReport) RepairPlan() WorkspaceRepairPlan {
	if r == nil {
		return WorkspaceRepairPlan{}
	}
	return r.repairPlan
}

type workspaceRelationshipKey struct {
	source      DocumentKey
	offset      int
	kind        LinkRelationshipKind
	destination string
}

type workspaceValidationState struct {
	documents      []GraphDocument
	documentIndex  map[DocumentKey]*Document
	resolved       map[workspaceRelationshipKey]DocumentResolution
	fragmentStatus map[DocumentKey]func(string) (FragmentTarget, LinkFragmentStatus)
	graphFragments graphFragmentResolvers
	diagnostics    []WorkspaceDiagnostic
}

// ValidateWorkspace validates relationships and explicitly managed generated indexes
// over a finite caller-provided document set. Marksplice performs no filesystem or
// network discovery and never retains resolver or validation authority callbacks.
func ValidateWorkspace(documents []GraphDocument, resolver WorkspaceResolver, options WorkspaceValidationOptions) (*WorkspaceReport, error) {
	index, err := validateWorkspaceInputs(documents, options)
	if err != nil {
		return nil, err
	}
	state := workspaceValidationState{
		documents:      documents,
		documentIndex:  index,
		resolved:       make(map[workspaceRelationshipKey]DocumentResolution),
		fragmentStatus: make(map[DocumentKey]func(string) (FragmentTarget, LinkFragmentStatus)),
		graphFragments: make(graphFragmentResolvers),
	}
	if err := state.scanRelationships(resolver); err != nil {
		return nil, err
	}
	graph, err := buildDocumentGraph(documents, state.graphResolver(), state.graphFragments)
	if err != nil {
		return nil, fmt.Errorf("%w: build resolved document graph: %v", ErrInvalidWorkspace, err)
	}
	if err := state.appendUnresolvedReferences(); err != nil {
		return nil, err
	}
	state.appendOrphanDiagnostics(graph, options.Roots)
	repairs, err := state.appendManagedTOCDiagnostics(options.ManagedTOCs)
	if err != nil {
		return nil, err
	}
	return &WorkspaceReport{
		graph:       graph,
		diagnostics: append([]WorkspaceDiagnostic(nil), state.diagnostics...),
		repairPlan:  WorkspaceRepairPlan{repairs: repairs},
	}, nil
}

func validateWorkspaceInputs(documents []GraphDocument, options WorkspaceValidationOptions) (map[DocumentKey]*Document, error) {
	index := make(map[DocumentKey]*Document, len(documents))
	for position, item := range documents {
		if item.Key == "" || item.Document == nil || item.Document.document == nil {
			return nil, fmt.Errorf("%w: invalid document at position %d", ErrInvalidWorkspace, position)
		}
		if _, exists := index[item.Key]; exists {
			return nil, fmt.Errorf("%w: duplicate document key %q", ErrInvalidWorkspace, item.Key)
		}
		index[item.Key] = item.Document
	}
	if err := validateWorkspaceRoots(index, options.Roots); err != nil {
		return nil, err
	}
	if err := validateManagedTOCs(index, options.ManagedTOCs); err != nil {
		return nil, err
	}
	return index, nil
}

func validateWorkspaceRoots(index map[DocumentKey]*Document, roots []DocumentKey) error {
	seen := make(map[DocumentKey]bool, len(roots))
	for _, root := range roots {
		if root == "" {
			return fmt.Errorf("%w: empty workspace root", ErrInvalidWorkspace)
		}
		if _, exists := index[root]; !exists {
			return fmt.Errorf("%w: unknown workspace root %q", ErrInvalidWorkspace, root)
		}
		if seen[root] {
			return fmt.Errorf("%w: duplicate workspace root %q", ErrInvalidWorkspace, root)
		}
		seen[root] = true
	}
	return nil
}

func validateManagedTOCs(index map[DocumentKey]*Document, managed []ManagedTOC) error {
	seen := make(map[managedTOCKey]bool, len(managed))
	headings := make(map[DocumentKey]map[NodeID]bool)
	for _, target := range managed {
		document, exists := index[target.Document]
		if target.Document == "" || !exists {
			return fmt.Errorf("%w: unknown managed TOC document %q", ErrInvalidWorkspace, target.Document)
		}
		if target.HeadingID == (NodeID{}) {
			return fmt.Errorf("%w: managed TOC for %q has zero heading identity", ErrInvalidWorkspace, target.Document)
		}
		available, ok := headings[target.Document]
		if !ok {
			available = documentHeadingIDs(document)
			headings[target.Document] = available
		}
		if !available[target.HeadingID] {
			return fmt.Errorf("%w: managed TOC heading does not belong to %q", ErrInvalidWorkspace, target.Document)
		}
		key := managedTOCKey{document: target.Document, heading: target.HeadingID}
		if seen[key] {
			return fmt.Errorf("%w: duplicate managed TOC target for %q", ErrInvalidWorkspace, target.Document)
		}
		seen[key] = true
	}
	return nil
}

func documentHeadingIDs(document *Document) map[NodeID]bool {
	anchors := document.HeadingAnchors()
	result := make(map[NodeID]bool, len(anchors))
	for _, anchor := range anchors {
		result[anchor.HeadingID()] = true
	}
	return result
}

func managedTOCStatuses(index map[DocumentKey]*Document, managed []ManagedTOC) map[managedTOCKey]managedTOCStatus {
	headings := make(map[DocumentKey][]NodeID)
	for _, target := range managed {
		headings[target.Document] = append(headings[target.Document], target.HeadingID)
	}
	result := make(map[managedTOCKey]managedTOCStatus, len(managed))
	for key, ids := range headings {
		for _, status := range index[key].managedTOCStatuses(ids) {
			result[managedTOCKey{document: key, heading: status.headingID}] = status
		}
	}
	return result
}

func (s *workspaceValidationState) scanRelationships(resolver WorkspaceResolver) error {
	for _, item := range s.documents {
		relationships, ok := item.Document.linkRelationships()
		if !ok {
			return fmt.Errorf("%w: relationship projection failed for %q", ErrInvalidWorkspace, item.Key)
		}
		for _, relationship := range relationships {
			if strings.HasPrefix(relationship.Destination(), "#") {
				if diagnostic, ok := localFragmentDiagnostic(item.Key, relationship); ok {
					s.diagnostics = append(s.diagnostics, diagnostic)
				}
				continue
			}
			if resolver == nil {
				continue
			}
			if err := s.acceptResolution(item.Key, relationship, resolver(item.Key, relationship)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *workspaceValidationState) acceptResolution(source DocumentKey, relationship LinkRelationship, resolution WorkspaceResolution) error {
	switch resolution.Kind {
	case WorkspaceResolutionIgnore:
		if resolution.Target != "" || resolution.Fragment != "" {
			return fmt.Errorf("%w: ignored relationship from %q carries target data", ErrInvalidWorkspace, source)
		}
		return nil
	case WorkspaceResolutionMissing:
		return s.acceptMissingResolution(source, relationship, resolution)
	case WorkspaceResolutionResolved:
		return s.acceptResolvedResolution(source, relationship, resolution)
	default:
		return fmt.Errorf("%w: invalid workspace resolution kind %d", ErrInvalidWorkspace, resolution.Kind)
	}
}

func (s *workspaceValidationState) acceptMissingResolution(source DocumentKey, relationship LinkRelationship, resolution WorkspaceResolution) error {
	if resolution.Target == "" {
		return fmt.Errorf("%w: missing-document resolution from %q has empty target", ErrInvalidWorkspace, source)
	}
	if _, exists := s.documentIndex[resolution.Target]; exists {
		return fmt.Errorf("%w: target %q was reported missing but is present", ErrInvalidWorkspace, resolution.Target)
	}
	diagnostic := relationshipDiagnostic(WorkspaceDiagnosticMissingDocument, source, relationship)
	diagnostic.targetDocument = resolution.Target
	diagnostic.hasTargetDocument = true
	if resolution.Fragment != "" {
		diagnostic.fragment = resolution.Fragment
		diagnostic.hasFragment = true
	}
	s.diagnostics = append(s.diagnostics, diagnostic)
	return nil
}

func (s *workspaceValidationState) acceptResolvedResolution(source DocumentKey, relationship LinkRelationship, resolution WorkspaceResolution) error {
	target, exists := s.documentIndex[resolution.Target]
	if resolution.Target == "" || !exists {
		return fmt.Errorf("%w: resolved target %q from %q is outside the document set", ErrInvalidWorkspace, resolution.Target, source)
	}
	s.resolved[workspaceRelationshipKeyFor(source, relationship)] = DocumentResolution{
		Target:   resolution.Target,
		Fragment: resolution.Fragment,
	}
	if resolution.Fragment == "" {
		return nil
	}
	resolve := s.fragmentStatusResolver(resolution.Target, target)
	_, status := resolve(resolution.Fragment)
	if diagnostic, ok := crossFragmentDiagnostic(source, relationship, resolution.Target, resolution.Fragment, status); ok {
		s.diagnostics = append(s.diagnostics, diagnostic)
	}
	return nil
}

func (s *workspaceValidationState) fragmentStatusResolver(key DocumentKey, document *Document) func(string) (FragmentTarget, LinkFragmentStatus) {
	if resolve, ok := s.fragmentStatus[key]; ok {
		return resolve
	}
	resolve := document.fragmentStatusResolver()
	s.fragmentStatus[key] = resolve
	s.graphFragments[key] = func(fragment string) (FragmentTarget, bool) {
		target, status := resolve(fragment)
		return target, status == LinkFragmentResolved
	}
	return resolve
}

func (s *workspaceValidationState) graphResolver() DocumentResolver {
	return func(source DocumentKey, relationship LinkRelationship) (DocumentResolution, bool) {
		resolution, ok := s.resolved[workspaceRelationshipKeyFor(source, relationship)]
		return resolution, ok
	}
}

func workspaceRelationshipKeyFor(source DocumentKey, relationship LinkRelationship) workspaceRelationshipKey {
	return workspaceRelationshipKey{
		source:      source,
		offset:      relationship.SourceOffset(),
		kind:        relationship.Kind(),
		destination: relationship.Destination(),
	}
}

func localFragmentDiagnostic(source DocumentKey, relationship LinkRelationship) (WorkspaceDiagnostic, bool) {
	kind, ok := workspaceFragmentDiagnosticKind(relationship.FragmentStatus())
	if !ok {
		return WorkspaceDiagnostic{}, false
	}
	diagnostic := relationshipDiagnostic(kind, source, relationship)
	diagnostic.targetDocument = source
	diagnostic.hasTargetDocument = true
	diagnostic.fragment = relationship.Destination()
	diagnostic.hasFragment = true
	return diagnostic, true
}

func crossFragmentDiagnostic(source DocumentKey, relationship LinkRelationship, target DocumentKey, fragment string, status LinkFragmentStatus) (WorkspaceDiagnostic, bool) {
	kind, ok := workspaceFragmentDiagnosticKind(status)
	if !ok {
		return WorkspaceDiagnostic{}, false
	}
	diagnostic := relationshipDiagnostic(kind, source, relationship)
	diagnostic.targetDocument = target
	diagnostic.hasTargetDocument = true
	diagnostic.fragment = fragment
	diagnostic.hasFragment = true
	return diagnostic, true
}

func workspaceFragmentDiagnosticKind(status LinkFragmentStatus) (WorkspaceDiagnosticKind, bool) {
	switch status {
	case LinkFragmentMissing:
		return WorkspaceDiagnosticMissingFragment, true
	case LinkFragmentAmbiguous:
		return WorkspaceDiagnosticAmbiguousFragment, true
	case LinkFragmentInvalid:
		return WorkspaceDiagnosticInvalidFragment, true
	case LinkFragmentResolved:
		return WorkspaceDiagnosticUnknown, false
	default:
		return WorkspaceDiagnosticUnknown, false
	}
}

func relationshipDiagnostic(kind WorkspaceDiagnosticKind, source DocumentKey, relationship LinkRelationship) WorkspaceDiagnostic {
	return WorkspaceDiagnostic{
		kind:              kind,
		relationship:      relationship,
		hasRelationship:   true,
		sourceOffset:      relationship.SourceOffset(),
		hasSourceOffset:   true,
		sourceDocument:    source,
		hasSourceDocument: true,
	}
}

func (s *workspaceValidationState) appendUnresolvedReferences() error {
	for _, item := range s.documents {
		unresolved, ok := item.Document.document.UnresolvedReferences()
		if !ok {
			return fmt.Errorf("%w: unresolved-reference projection failed for %q", ErrInvalidWorkspace, item.Key)
		}
		for _, reference := range unresolved {
			form, ok := publicReferenceForm(reference.Form, true)
			if !ok || (form != ReferenceFormFull && form != ReferenceFormCollapsed) {
				return fmt.Errorf("%w: invalid unresolved-reference form for %q", ErrInvalidWorkspace, item.Key)
			}
			s.diagnostics = append(s.diagnostics, WorkspaceDiagnostic{
				kind:              WorkspaceDiagnosticUnresolvedReference,
				sourceOffset:      reference.SourceOffset,
				hasSourceOffset:   true,
				sourceDocument:    item.Key,
				hasSourceDocument: true,
				unresolvedReference: UnresolvedReference{
					reference: reference.Reference,
					form:      form,
					image:     reference.Image,
				},
				hasUnresolvedReference: true,
			})
		}
	}
	return nil
}

func (s *workspaceValidationState) appendOrphanDiagnostics(graph *DocumentGraph, roots []DocumentKey) {
	if len(roots) == 0 {
		return
	}
	reachable := graph.reachableFromRoots(roots)
	for _, item := range s.documents {
		if reachable[item.Key] {
			continue
		}
		s.diagnostics = append(s.diagnostics, WorkspaceDiagnostic{
			kind:              WorkspaceDiagnosticOrphanDocument,
			targetDocument:    item.Key,
			hasTargetDocument: true,
		})
	}
}

func (s *workspaceValidationState) appendManagedTOCDiagnostics(managed []ManagedTOC) ([]WorkspaceRepair, error) {
	statuses := managedTOCStatuses(s.documentIndex, managed)
	headings := make(map[DocumentKey][]NodeID)
	order := make([]DocumentKey, 0)
	for _, target := range managed {
		status := statuses[managedTOCKey{document: target.Document, heading: target.HeadingID}]
		if !status.recognized {
			s.diagnostics = append(s.diagnostics, generatedIndexDiagnostic(WorkspaceDiagnosticUnrecognizedGeneratedIndex, target))
			continue
		}
		if !status.stale {
			continue
		}
		s.diagnostics = append(s.diagnostics, generatedIndexDiagnostic(WorkspaceDiagnosticStaleGeneratedIndex, target))
		if len(headings[target.Document]) == 0 {
			order = append(order, target.Document)
		}
		headings[target.Document] = append(headings[target.Document], target.HeadingID)
	}
	return s.prepareManagedTOCRepairs(order, headings)
}

func (s *workspaceValidationState) prepareManagedTOCRepairs(order []DocumentKey, headings map[DocumentKey][]NodeID) ([]WorkspaceRepair, error) {
	repairs := make([]WorkspaceRepair, 0, len(order))
	for _, key := range order {
		change, err := s.documentIndex[key].prepareSyncTOCs(headings[key])
		if err != nil {
			return nil, fmt.Errorf("%w: prepare atomic managed TOC repair for %q: %v", ErrInvalidWorkspace, key, err)
		}
		repairs = append(repairs, WorkspaceRepair{document: key, change: change})
	}
	return repairs, nil
}

func generatedIndexDiagnostic(kind WorkspaceDiagnosticKind, target ManagedTOC) WorkspaceDiagnostic {
	return WorkspaceDiagnostic{
		kind:              kind,
		sourceDocument:    target.Document,
		hasSourceDocument: true,
		targetDocument:    target.Document,
		hasTargetDocument: true,
		nodeID:            target.HeadingID,
		hasNodeID:         true,
	}
}
