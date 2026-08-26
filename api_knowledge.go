package marksplice

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// KnowledgeAlias is one exact, syntax-independent alternate name for a document.
// Marksplice does not normalize, parse, or derive aliases from Markdown or metadata source.
type KnowledgeAlias string

// KnowledgeTag is one exact, syntax-independent classification value for a document.
// Matching is byte-exact and case-sensitive; Marksplice performs no normalization.
type KnowledgeTag string

// KnowledgeDocument supplies caller-owned semantic metadata for one document already
// present in the explicit DocumentGraph. Graph documents omitted from the input simply
// have no knowledge metadata; this value never changes the underlying document snapshot.
type KnowledgeDocument struct {
	Document   DocumentKey
	Aliases    []KnowledgeAlias
	Tags       []KnowledgeTag
	References []DocumentKey
}

// KnowledgeReference is one immutable caller-declared logical document relationship.
// It has no source offset because the knowledge layer does not infer Markdown or metadata syntax.
type KnowledgeReference struct {
	sourceDocument DocumentKey
	targetDocument DocumentKey
}

// SourceDocument returns the caller-defined logical source document key.
func (r KnowledgeReference) SourceDocument() DocumentKey { return r.sourceDocument }

// TargetDocument returns the caller-defined logical target document key.
func (r KnowledgeReference) TargetDocument() DocumentKey { return r.targetDocument }

type knowledgeDocumentState struct {
	aliases []KnowledgeAlias
	tags    []KnowledgeTag
}

// KnowledgeIndex is an immutable syntax-independent semantic overlay on one DocumentGraph.
// It retains no parser, resolver callback, filesystem/network authority, or source mutation capability.
type KnowledgeIndex struct {
	graph       *DocumentGraph
	documents   map[DocumentKey]knowledgeDocumentState
	aliasOwners map[KnowledgeAlias]DocumentKey
	references  []KnowledgeReference
	outgoing    map[DocumentKey][]int
	backlinks   map[DocumentKey][]int
}

// BuildKnowledgeIndex builds syntax-independent aliases, tags, and logical references
// over an already-authorized document graph. Metadata may be supplied for any subset of
// graph documents. Every logical reference target must already belong to the graph;
// aliases never resolve or discover additional documents.
func BuildKnowledgeIndex(graph *DocumentGraph, documents []KnowledgeDocument) (*KnowledgeIndex, error) {
	if graph == nil {
		return nil, fmt.Errorf("%w: nil document graph", ErrInvalidKnowledge)
	}
	states := make(map[DocumentKey]knowledgeDocumentState, len(documents))
	aliasOwners := make(map[KnowledgeAlias]DocumentKey)
	referenceTargets := make(map[DocumentKey][]DocumentKey, len(documents))
	for position, document := range documents {
		if !graph.hasDocument(document.Document) {
			return nil, fmt.Errorf("%w: metadata document %q at position %d is outside the graph", ErrInvalidKnowledge, document.Document, position)
		}
		if _, exists := states[document.Document]; exists {
			return nil, fmt.Errorf("%w: duplicate metadata for document %q", ErrInvalidKnowledge, document.Document)
		}
		state, references, err := validateKnowledgeDocument(document, graph, aliasOwners)
		if err != nil {
			return nil, err
		}
		states[document.Document] = state
		referenceTargets[document.Document] = references
	}

	index := &KnowledgeIndex{
		graph:       graph,
		documents:   states,
		aliasOwners: aliasOwners,
		outgoing:    make(map[DocumentKey][]int),
		backlinks:   make(map[DocumentKey][]int),
	}
	for _, source := range graph.keys {
		for _, target := range referenceTargets[source] {
			index.addReference(KnowledgeReference{sourceDocument: source, targetDocument: target})
		}
	}
	return index, nil
}

func validateKnowledgeDocument(document KnowledgeDocument, graph *DocumentGraph, aliasOwners map[KnowledgeAlias]DocumentKey) (knowledgeDocumentState, []DocumentKey, error) {
	state := knowledgeDocumentState{
		aliases: append([]KnowledgeAlias(nil), document.Aliases...),
		tags:    append([]KnowledgeTag(nil), document.Tags...),
	}
	for _, alias := range state.aliases {
		if !validKnowledgeText(string(alias)) {
			return knowledgeDocumentState{}, nil, fmt.Errorf("%w: document %q has invalid alias", ErrInvalidKnowledge, document.Document)
		}
		if graph.hasDocument(DocumentKey(alias)) {
			return knowledgeDocumentState{}, nil, fmt.Errorf("%w: alias %q collides with a canonical document key", ErrInvalidKnowledge, alias)
		}
		if owner, exists := aliasOwners[alias]; exists {
			return knowledgeDocumentState{}, nil, fmt.Errorf("%w: alias %q is already owned by %q", ErrInvalidKnowledge, alias, owner)
		}
		aliasOwners[alias] = document.Document
	}
	if err := validateKnowledgeTags(document.Document, state.tags); err != nil {
		return knowledgeDocumentState{}, nil, err
	}
	references := append([]DocumentKey(nil), document.References...)
	if err := validateKnowledgeReferences(document.Document, references, graph); err != nil {
		return knowledgeDocumentState{}, nil, err
	}
	return state, references, nil
}

func validateKnowledgeTags(document DocumentKey, tags []KnowledgeTag) error {
	seen := make(map[KnowledgeTag]struct{}, len(tags))
	for _, tag := range tags {
		if !validKnowledgeText(string(tag)) {
			return fmt.Errorf("%w: document %q has invalid tag", ErrInvalidKnowledge, document)
		}
		if _, exists := seen[tag]; exists {
			return fmt.Errorf("%w: document %q repeats tag %q", ErrInvalidKnowledge, document, tag)
		}
		seen[tag] = struct{}{}
	}
	return nil
}

func validateKnowledgeReferences(document DocumentKey, references []DocumentKey, graph *DocumentGraph) error {
	seen := make(map[DocumentKey]struct{}, len(references))
	for _, target := range references {
		if !graph.hasDocument(target) {
			return fmt.Errorf("%w: document %q references unknown target %q", ErrInvalidKnowledge, document, target)
		}
		if _, exists := seen[target]; exists {
			return fmt.Errorf("%w: document %q repeats logical reference to %q", ErrInvalidKnowledge, document, target)
		}
		seen[target] = struct{}{}
	}
	return nil
}

func validKnowledgeText(value string) bool {
	return value != "" && utf8.ValidString(value) && !strings.ContainsAny(value, "\x00\r\n")
}

func (k *KnowledgeIndex) addReference(reference KnowledgeReference) {
	index := len(k.references)
	k.references = append(k.references, reference)
	k.outgoing[reference.sourceDocument] = append(k.outgoing[reference.sourceDocument], index)
	k.backlinks[reference.targetDocument] = append(k.backlinks[reference.targetDocument], index)
}

func (k *KnowledgeIndex) hasDocument(key DocumentKey) bool {
	return k != nil && k.graph != nil && k.graph.hasDocument(key)
}

// Aliases returns exact aliases for key in caller-provided order. The returned slice is caller-owned.
func (k *KnowledgeIndex) Aliases(key DocumentKey) ([]KnowledgeAlias, bool) {
	if !k.hasDocument(key) {
		return nil, false
	}
	return append([]KnowledgeAlias(nil), k.documents[key].aliases...), true
}

// Tags returns exact tags for key in caller-provided order. The returned slice is caller-owned.
func (k *KnowledgeIndex) Tags(key DocumentKey) ([]KnowledgeTag, bool) {
	if !k.hasDocument(key) {
		return nil, false
	}
	return append([]KnowledgeTag(nil), k.documents[key].tags...), true
}

// ResolveAlias resolves one exact globally unique alias to an existing graph document.
// Canonical DocumentKey values are not aliases and should be queried through DocumentGraph.
func (k *KnowledgeIndex) ResolveAlias(alias KnowledgeAlias) (DocumentKey, bool) {
	if k == nil {
		return "", false
	}
	key, ok := k.aliasOwners[alias]
	return key, ok
}

// DocumentsWithTag returns graph documents carrying the exact tag in original graph-input order.
func (k *KnowledgeIndex) DocumentsWithTag(tag KnowledgeTag) []DocumentKey {
	if k == nil || k.graph == nil {
		return nil
	}
	result := make([]DocumentKey, 0)
	for _, key := range k.graph.keys {
		if containsKnowledgeTag(k.documents[key].tags, tag) {
			result = append(result, key)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func containsKnowledgeTag(tags []KnowledgeTag, target KnowledgeTag) bool {
	for _, tag := range tags {
		if tag == target {
			return true
		}
	}
	return false
}

// References returns all logical references in graph document order and per-document caller order.
func (k *KnowledgeIndex) References() []KnowledgeReference {
	if k == nil || len(k.references) == 0 {
		return nil
	}
	return append([]KnowledgeReference(nil), k.references...)
}

// ReferencesFrom returns logical references whose source is key in caller order.
func (k *KnowledgeIndex) ReferencesFrom(key DocumentKey) ([]KnowledgeReference, bool) {
	if !k.hasDocument(key) {
		return nil, false
	}
	return k.referencesAt(k.outgoing[key]), true
}

// ReferencedBy returns logical references whose target is key in global reference order.
func (k *KnowledgeIndex) ReferencedBy(key DocumentKey) ([]KnowledgeReference, bool) {
	if !k.hasDocument(key) {
		return nil, false
	}
	return k.referencesAt(k.backlinks[key]), true
}

func (k *KnowledgeIndex) referencesAt(indices []int) []KnowledgeReference {
	if len(indices) == 0 {
		return nil
	}
	result := make([]KnowledgeReference, len(indices))
	for position, index := range indices {
		result[position] = k.references[index]
	}
	return result
}

// ReachableFrom returns every other document reachable through the union of resolved
// Markdown graph edges and caller-declared logical references. For each visited source,
// graph edges are considered first, then logical references. Results are deterministic
// breadth-first discovery order and self/cyclic paths never duplicate a document.
func (k *KnowledgeIndex) ReachableFrom(key DocumentKey) ([]DocumentKey, bool) {
	if !k.hasDocument(key) {
		return nil, false
	}
	visited := make(map[DocumentKey]bool, len(k.graph.documents))
	visited[key] = true
	queue := []DocumentKey{key}
	result := make([]DocumentKey, 0, len(k.graph.documents)-1)
	for head := 0; head < len(queue); head++ {
		current := queue[head]
		for _, edgeIndex := range k.graph.outgoing[current] {
			appendUnvisitedDocument(k.graph.edges[edgeIndex].targetDocument, visited, &queue, &result)
		}
		for _, referenceIndex := range k.outgoing[current] {
			appendUnvisitedDocument(k.references[referenceIndex].targetDocument, visited, &queue, &result)
		}
	}
	return result, true
}

// RelatedDocuments returns unique direct neighbors across both resolved Markdown graph
// edges and caller-declared logical references in original graph-input order. Self
// relationships are omitted.
func (k *KnowledgeIndex) RelatedDocuments(key DocumentKey) ([]DocumentKey, bool) {
	if !k.hasDocument(key) {
		return nil, false
	}
	related := make(map[DocumentKey]bool)
	k.graph.markRelatedDocuments(key, related)
	k.markRelatedKnowledgeDocuments(key, related)
	return k.graph.orderedRelatedDocuments(key, related), true
}

func (k *KnowledgeIndex) markRelatedKnowledgeDocuments(key DocumentKey, related map[DocumentKey]bool) {
	for _, referenceIndex := range k.outgoing[key] {
		related[k.references[referenceIndex].targetDocument] = true
	}
	for _, referenceIndex := range k.backlinks[key] {
		related[k.references[referenceIndex].sourceDocument] = true
	}
}
