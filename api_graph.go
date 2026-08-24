package marksplice

import (
	"fmt"
	"strings"
)

// DocumentKey is a caller-defined logical identity for one document in a graph.
// Marksplice treats it as opaque data and does not interpret it as a filesystem path or URL.
type DocumentKey string

// GraphDocument binds one caller-defined logical key to an immutable parsed document.
type GraphDocument struct {
	Key      DocumentKey
	Document *Document
}

// DocumentResolution is a caller-authorized resolution of one non-local relationship
// to a document that is already present in the explicit graph input set. Fragment is
// optional and uses the same optional-leading-# syntax accepted by Document.ResolveFragment.
type DocumentResolution struct {
	Target   DocumentKey
	Fragment string
}

// DocumentResolver resolves one non-local relationship against the caller's own
// authorization/domain model. Returning false leaves the relationship outside the graph.
// Marksplice never retains the resolver and never performs filesystem or network access.
type DocumentResolver func(source DocumentKey, relationship LinkRelationship) (DocumentResolution, bool)

// GraphEdge is one immutable resolved relationship between two caller-provided documents.
type GraphEdge struct {
	sourceDocument    DocumentKey
	targetDocument    DocumentKey
	relationship      LinkRelationship
	fragment          string
	hasFragment       bool
	fragmentTarget    FragmentTarget
	hasFragmentTarget bool
}

// SourceDocument returns the caller-defined logical source document key.
func (e GraphEdge) SourceDocument() DocumentKey { return e.sourceDocument }

// TargetDocument returns the caller-defined logical target document key.
func (e GraphEdge) TargetDocument() DocumentKey { return e.targetDocument }

// Relationship returns the immutable M99 relationship that produced this edge.
func (e GraphEdge) Relationship() LinkRelationship { return e.relationship }

// Fragment returns the local target fragment when the edge carries one. For
// cross-document relationships this is the fragment supplied by the resolver.
func (e GraphEdge) Fragment() (string, bool) { return e.fragment, e.hasFragment }

// FragmentTarget returns the uniquely resolved target snapshot fragment when one exists.
func (e GraphEdge) FragmentTarget() (FragmentTarget, bool) {
	return e.fragmentTarget, e.hasFragmentTarget
}

// DocumentGraph is an immutable graph over an explicit caller-provided document set.
// It stores resolved edges plus compact adjacency indexes and performs no I/O.
type DocumentGraph struct {
	keys      []DocumentKey
	documents map[DocumentKey]*Document
	edges     []GraphEdge
	outgoing  map[DocumentKey][]int
	backlinks map[DocumentKey][]int
}

// BuildDocumentGraph builds a deterministic graph over documents already supplied by
// the caller. Local #fragment relationships resolve to their source document without
// invoking resolver. Every other relationship is included only when resolver explicitly
// maps it to a document key from the supplied set.
func BuildDocumentGraph(documents []GraphDocument, resolver DocumentResolver) (*DocumentGraph, error) {
	return buildDocumentGraph(documents, resolver, nil)
}

func buildDocumentGraph(documents []GraphDocument, resolver DocumentResolver, fragmentResolvers graphFragmentResolvers) (*DocumentGraph, error) {
	graph := &DocumentGraph{
		keys:      make([]DocumentKey, 0, len(documents)),
		documents: make(map[DocumentKey]*Document, len(documents)),
		outgoing:  make(map[DocumentKey][]int, len(documents)),
		backlinks: make(map[DocumentKey][]int, len(documents)),
	}
	if fragmentResolvers == nil {
		fragmentResolvers = make(graphFragmentResolvers)
	}
	for index, item := range documents {
		if item.Key == "" {
			return nil, fmt.Errorf("%w: document %d has an empty key", ErrInvalidGraph, index)
		}
		if item.Document == nil || item.Document.document == nil {
			return nil, fmt.Errorf("%w: document %q is nil or uninitialized", ErrInvalidGraph, item.Key)
		}
		if _, exists := graph.documents[item.Key]; exists {
			return nil, fmt.Errorf("%w: duplicate document key %q", ErrInvalidGraph, item.Key)
		}
		graph.keys = append(graph.keys, item.Key)
		graph.documents[item.Key] = item.Document
	}

	for _, item := range documents {
		relationships, ok := item.Document.linkRelationships()
		if !ok {
			return nil, fmt.Errorf("%w: relationship projection failed for %q", ErrInvalidGraph, item.Key)
		}
		for _, relationship := range relationships {
			if strings.HasPrefix(relationship.Destination(), "#") {
				graph.addEdge(localGraphEdge(item.Key, relationship))
				continue
			}
			if resolver == nil {
				continue
			}
			resolution, resolved := resolver(item.Key, relationship)
			if !resolved {
				continue
			}
			target, ok := graph.documents[resolution.Target]
			if resolution.Target == "" || !ok {
				return nil, fmt.Errorf("%w: resolver target %q from %q is not in the document set", ErrInvalidGraph, resolution.Target, item.Key)
			}
			graph.addEdge(resolvedGraphEdge(item.Key, resolution, relationship, target, fragmentResolvers))
		}
	}
	return graph, nil
}

func localGraphEdge(key DocumentKey, relationship LinkRelationship) GraphEdge {
	edge := GraphEdge{
		sourceDocument: key,
		targetDocument: key,
		relationship:   relationship,
		fragment:       relationship.Destination(),
		hasFragment:    true,
	}
	if target, ok := relationship.FragmentTarget(); ok {
		edge.fragmentTarget = target
		edge.hasFragmentTarget = true
	}
	return edge
}

type graphFragmentResolvers map[DocumentKey]func(string) (FragmentTarget, bool)

func (r graphFragmentResolvers) resolve(key DocumentKey, target *Document, fragment string) (FragmentTarget, bool) {
	resolve, ok := r[key]
	if !ok {
		resolve = target.fragmentResolver()
		r[key] = resolve
	}
	if resolve == nil {
		return FragmentTarget{}, false
	}
	return resolve(fragment)
}

func resolvedGraphEdge(source DocumentKey, resolution DocumentResolution, relationship LinkRelationship, target *Document, resolvers graphFragmentResolvers) GraphEdge {
	edge := GraphEdge{
		sourceDocument: source,
		targetDocument: resolution.Target,
		relationship:   relationship,
	}
	if resolution.Fragment == "" {
		return edge
	}
	edge.fragment = resolution.Fragment
	edge.hasFragment = true
	if fragmentTarget, ok := resolvers.resolve(resolution.Target, target, resolution.Fragment); ok {
		edge.fragmentTarget = fragmentTarget
		edge.hasFragmentTarget = true
	}
	return edge
}

func (g *DocumentGraph) addEdge(edge GraphEdge) {
	index := len(g.edges)
	g.edges = append(g.edges, edge)
	g.outgoing[edge.sourceDocument] = append(g.outgoing[edge.sourceDocument], index)
	g.backlinks[edge.targetDocument] = append(g.backlinks[edge.targetDocument], index)
}

// DocumentKeys returns caller-defined document keys in graph-input order.
func (g *DocumentGraph) DocumentKeys() []DocumentKey {
	if g == nil {
		return nil
	}
	return append([]DocumentKey(nil), g.keys...)
}

// Document returns one immutable document snapshot by caller-defined key.
func (g *DocumentGraph) Document(key DocumentKey) (*Document, bool) {
	if g == nil {
		return nil, false
	}
	document, ok := g.documents[key]
	return document, ok
}

// Edges returns all resolved graph edges in deterministic document/source order.
func (g *DocumentGraph) Edges() []GraphEdge {
	if g == nil {
		return nil
	}
	return append([]GraphEdge(nil), g.edges...)
}

// Outgoing returns resolved edges whose source is key in relationship source order.
func (g *DocumentGraph) Outgoing(key DocumentKey) ([]GraphEdge, bool) {
	if g == nil {
		return nil, false
	}
	if _, ok := g.documents[key]; !ok {
		return nil, false
	}
	return g.edgesAt(g.outgoing[key]), true
}

// Backlinks returns resolved edges whose target is key in global edge order.
func (g *DocumentGraph) Backlinks(key DocumentKey) ([]GraphEdge, bool) {
	if g == nil {
		return nil, false
	}
	if _, ok := g.documents[key]; !ok {
		return nil, false
	}
	return g.edgesAt(g.backlinks[key]), true
}

func (g *DocumentGraph) edgesAt(indices []int) []GraphEdge {
	if len(indices) == 0 {
		return nil
	}
	result := make([]GraphEdge, len(indices))
	for position, index := range indices {
		result[position] = g.edges[index]
	}
	return result
}

// ReachableFrom returns every other document reachable from key using resolved graph
// edges. Results are in deterministic breadth-first discovery order; self cycles are omitted.
func (g *DocumentGraph) ReachableFrom(key DocumentKey) ([]DocumentKey, bool) {
	if g == nil {
		return nil, false
	}
	if _, ok := g.documents[key]; !ok {
		return nil, false
	}
	visited := make(map[DocumentKey]bool, len(g.documents))
	visited[key] = true
	queue := make([]DocumentKey, 1, len(g.documents))
	queue[0] = key
	result := make([]DocumentKey, 0, len(g.documents)-1)
	for head := 0; head < len(queue); head++ {
		for _, edgeIndex := range g.outgoing[queue[head]] {
			target := g.edges[edgeIndex].targetDocument
			if visited[target] {
				continue
			}
			visited[target] = true
			queue = append(queue, target)
			result = append(result, target)
		}
	}
	return result, true
}

func (g *DocumentGraph) reachableFromRoots(roots []DocumentKey) map[DocumentKey]bool {
	visited := make(map[DocumentKey]bool, len(g.documents))
	queue := make([]DocumentKey, 0, len(g.documents))
	for _, root := range roots {
		if visited[root] {
			continue
		}
		visited[root] = true
		queue = append(queue, root)
	}
	for head := 0; head < len(queue); head++ {
		for _, edgeIndex := range g.outgoing[queue[head]] {
			target := g.edges[edgeIndex].targetDocument
			if visited[target] {
				continue
			}
			visited[target] = true
			queue = append(queue, target)
		}
	}
	return visited
}

// RelatedDocuments returns direct incoming-or-outgoing neighboring documents in the
// original graph-input order. Self edges and duplicate neighbors are omitted.
func (g *DocumentGraph) RelatedDocuments(key DocumentKey) ([]DocumentKey, bool) {
	if g == nil {
		return nil, false
	}
	if _, ok := g.documents[key]; !ok {
		return nil, false
	}
	related := make(map[DocumentKey]bool)
	for _, edgeIndex := range g.outgoing[key] {
		target := g.edges[edgeIndex].targetDocument
		if target != key {
			related[target] = true
		}
	}
	for _, edgeIndex := range g.backlinks[key] {
		source := g.edges[edgeIndex].sourceDocument
		if source != key {
			related[source] = true
		}
	}
	result := make([]DocumentKey, 0, len(related))
	for _, candidate := range g.keys {
		if related[candidate] {
			result = append(result, candidate)
		}
	}
	return result, true
}
