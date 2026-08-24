package publictest

import (
	"errors"
	"reflect"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM100DocumentGraphBuildsResolvedEdgesBacklinksReachabilityAndRelatedDocuments(t *testing.T) {
	t.Parallel()

	a := mustParseGraphDocument(t, "# A\n\n[to-b](b.md#target) [to-c](c.md) [external](https://example.com) [local](#a)\n")
	b := mustParseGraphDocument(t, "# Target\n\n[back](a.md)\n")
	c := mustParseGraphDocument(t, "# C\n\n[to-b](b.md) [missing](b.md#missing)\n")

	resolverCalls := 0
	resolver := func(source marksplice.DocumentKey, relationship marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
		resolverCalls++
		switch relationship.Destination() {
		case "b.md#target":
			return marksplice.DocumentResolution{Target: "b", Fragment: "#target"}, true
		case "c.md":
			return marksplice.DocumentResolution{Target: "c"}, true
		case "a.md":
			return marksplice.DocumentResolution{Target: "a"}, true
		case "b.md":
			return marksplice.DocumentResolution{Target: "b"}, true
		case "b.md#missing":
			return marksplice.DocumentResolution{Target: "b", Fragment: "#missing"}, true
		default:
			return marksplice.DocumentResolution{}, false
		}
	}

	graph, err := marksplice.BuildDocumentGraph([]marksplice.GraphDocument{
		{Key: "a", Document: a},
		{Key: "b", Document: b},
		{Key: "c", Document: c},
	}, resolver)
	if err != nil {
		t.Fatalf("BuildDocumentGraph() error = %v", err)
	}
	if resolverCalls != 6 {
		t.Fatalf("resolver calls = %d, want 6 non-local relationships", resolverCalls)
	}

	if got := graph.DocumentKeys(); !reflect.DeepEqual(got, []marksplice.DocumentKey{"a", "b", "c"}) {
		t.Fatalf("DocumentKeys() = %#v", got)
	}
	if got, ok := graph.Document("b"); !ok || got != b {
		t.Fatalf("Document(b) = %p/%v, want %p/true", got, ok, b)
	}
	if got, ok := graph.Document("missing"); ok || got != nil {
		t.Fatalf("Document(missing) = %p/%v, want nil/false", got, ok)
	}

	edges := graph.Edges()
	if len(edges) != 6 {
		t.Fatalf("Edges() count = %d, want 6", len(edges))
	}
	wantSources := []marksplice.DocumentKey{"a", "a", "a", "b", "c", "c"}
	wantTargets := []marksplice.DocumentKey{"b", "c", "a", "a", "b", "b"}
	wantDestinations := []string{"b.md#target", "c.md", "#a", "a.md", "b.md", "b.md#missing"}
	for index, edge := range edges {
		if edge.SourceDocument() != wantSources[index] || edge.TargetDocument() != wantTargets[index] || edge.Relationship().Destination() != wantDestinations[index] {
			t.Fatalf("edge[%d] = %q -> %q destination %q", index, edge.SourceDocument(), edge.TargetDocument(), edge.Relationship().Destination())
		}
	}

	if fragment, ok := edges[0].Fragment(); !ok || fragment != "#target" {
		t.Fatalf("cross-document Fragment() = %q/%v, want #target/true", fragment, ok)
	}
	if target, ok := edges[0].FragmentTarget(); !ok || target.Value() != "target" || target.NodeID().String() == "" {
		t.Fatalf("cross-document FragmentTarget() = %+v/%v", target, ok)
	}
	if fragment, ok := edges[2].Fragment(); !ok || fragment != "#a" {
		t.Fatalf("local Fragment() = %q/%v, want #a/true", fragment, ok)
	}
	if target, ok := edges[2].FragmentTarget(); !ok || target.Value() != "a" {
		t.Fatalf("local FragmentTarget() = %+v/%v", target, ok)
	}
	if fragment, ok := edges[5].Fragment(); !ok || fragment != "#missing" {
		t.Fatalf("missing Fragment() = %q/%v, want #missing/true", fragment, ok)
	}
	if _, ok := edges[5].FragmentTarget(); ok {
		t.Fatal("missing cross-document fragment unexpectedly resolved")
	}

	outgoing, ok := graph.Outgoing("a")
	if !ok || len(outgoing) != 3 {
		t.Fatalf("Outgoing(a) = %d/%v, want 3/true", len(outgoing), ok)
	}
	if got := graphEdgesDestinations(outgoing); !reflect.DeepEqual(got, []string{"b.md#target", "c.md", "#a"}) {
		t.Fatalf("Outgoing(a) destinations = %#v", got)
	}
	if got, ok := graph.Outgoing("missing"); ok || got != nil {
		t.Fatalf("Outgoing(missing) = %#v/%v", got, ok)
	}

	backlinks, ok := graph.Backlinks("b")
	if !ok || len(backlinks) != 3 {
		t.Fatalf("Backlinks(b) = %d/%v, want 3/true", len(backlinks), ok)
	}
	if got := graphEdgesSources(backlinks); !reflect.DeepEqual(got, []marksplice.DocumentKey{"a", "c", "c"}) {
		t.Fatalf("Backlinks(b) sources = %#v", got)
	}

	if got, ok := graph.ReachableFrom("a"); !ok || !reflect.DeepEqual(got, []marksplice.DocumentKey{"b", "c"}) {
		t.Fatalf("ReachableFrom(a) = %#v/%v", got, ok)
	}
	if got, ok := graph.RelatedDocuments("b"); !ok || !reflect.DeepEqual(got, []marksplice.DocumentKey{"a", "c"}) {
		t.Fatalf("RelatedDocuments(b) = %#v/%v", got, ok)
	}

	// Returned variable-length data is caller-owned.
	keys := graph.DocumentKeys()
	keys[0] = "mutated"
	edges[0] = marksplice.GraphEdge{}
	outgoing[0] = marksplice.GraphEdge{}
	if got := graph.DocumentKeys(); got[0] != "a" {
		t.Fatalf("DocumentKeys caller mutation leaked: %#v", got)
	}
	if got := graph.Edges(); len(got) != 6 || got[0].SourceDocument() != "a" {
		t.Fatalf("Edges caller mutation leaked: %#v", got)
	}
	if got, _ := graph.Outgoing("a"); len(got) != 3 || got[0].SourceDocument() != "a" {
		t.Fatalf("Outgoing caller mutation leaked: %#v", got)
	}
	backlinks[0] = marksplice.GraphEdge{}
	reachable, _ := graph.ReachableFrom("a")
	reachable[0] = "mutated"
	related, _ := graph.RelatedDocuments("b")
	related[0] = "mutated"
	if got, _ := graph.Backlinks("b"); len(got) != 3 || got[0].SourceDocument() != "a" {
		t.Fatalf("Backlinks caller mutation leaked: %#v", got)
	}
	if got, _ := graph.ReachableFrom("a"); !reflect.DeepEqual(got, []marksplice.DocumentKey{"b", "c"}) {
		t.Fatalf("ReachableFrom caller mutation leaked: %#v", got)
	}
	if got, _ := graph.RelatedDocuments("b"); !reflect.DeepEqual(got, []marksplice.DocumentKey{"a", "c"}) {
		t.Fatalf("RelatedDocuments caller mutation leaked: %#v", got)
	}
}

func TestM100DocumentGraphResolverIsBuildOnlyAndCrossDocumentFragmentsReuseM98(t *testing.T) {
	t.Parallel()

	source := mustParseGraphDocument(t, "# Source\n\n[encoded](target.md#caf%C3%A9) [missing](target.md#missing) [ambiguous](target.md#dup) [self](alias.md)\n")
	target := mustParseGraphDocument(t, "# Café\n\n<a id=\"dup\"></a>\n\n## Dup\n")
	input := []marksplice.GraphDocument{{Key: "source", Document: source}, {Key: "target", Document: target}}

	active := true
	calls := 0
	resolver := func(_ marksplice.DocumentKey, relationship marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
		if !active {
			panic("resolver retained after BuildDocumentGraph")
		}
		calls++
		switch relationship.Destination() {
		case "target.md#caf%C3%A9":
			return marksplice.DocumentResolution{Target: "target", Fragment: "#caf%C3%A9"}, true
		case "target.md#missing":
			return marksplice.DocumentResolution{Target: "target", Fragment: "#missing"}, true
		case "target.md#dup":
			return marksplice.DocumentResolution{Target: "target", Fragment: "#dup"}, true
		case "alias.md":
			return marksplice.DocumentResolution{Target: "source"}, true
		default:
			return marksplice.DocumentResolution{}, false
		}
	}

	graph, err := marksplice.BuildDocumentGraph(input, resolver)
	if err != nil {
		t.Fatalf("BuildDocumentGraph() error = %v", err)
	}
	if calls != 4 {
		t.Fatalf("resolver calls = %d, want 4", calls)
	}
	active = false
	input[0] = marksplice.GraphDocument{Key: "mutated", Document: target}

	edges := graph.Edges()
	if len(edges) != 4 {
		t.Fatalf("Edges() count = %d, want 4", len(edges))
	}
	if fragment, ok := edges[0].Fragment(); !ok || fragment != "#caf%C3%A9" {
		t.Fatalf("encoded Fragment() = %q/%v", fragment, ok)
	}
	if fragmentTarget, ok := edges[0].FragmentTarget(); !ok || fragmentTarget.Value() != "café" {
		t.Fatalf("encoded FragmentTarget() = %+v/%v", fragmentTarget, ok)
	}
	if _, ok := edges[1].FragmentTarget(); ok {
		t.Fatal("missing cross-document fragment unexpectedly resolved")
	}
	if _, ok := edges[2].FragmentTarget(); ok {
		t.Fatal("ambiguous cross-document fragment unexpectedly resolved")
	}
	if edges[3].SourceDocument() != "source" || edges[3].TargetDocument() != "source" {
		t.Fatalf("resolver self edge = %q -> %q", edges[3].SourceDocument(), edges[3].TargetDocument())
	}

	if got := graph.DocumentKeys(); !reflect.DeepEqual(got, []marksplice.DocumentKey{"source", "target"}) {
		t.Fatalf("DocumentKeys changed after input mutation: %#v", got)
	}
	if got, ok := graph.ReachableFrom("source"); !ok || !reflect.DeepEqual(got, []marksplice.DocumentKey{"target"}) {
		t.Fatalf("ReachableFrom(source) = %#v/%v", got, ok)
	}
	if got, ok := graph.RelatedDocuments("source"); !ok || !reflect.DeepEqual(got, []marksplice.DocumentKey{"target"}) {
		t.Fatalf("RelatedDocuments(source) = %#v/%v", got, ok)
	}
	if got, ok := graph.Backlinks("source"); !ok || len(got) != 1 || got[0].Relationship().Destination() != "alias.md" {
		t.Fatalf("Backlinks(source) = %#v/%v", got, ok)
	}
	if calls != 4 {
		t.Fatalf("resolver was invoked after graph construction: %d calls", calls)
	}
}

func TestM100DocumentGraphWithoutResolverStillIncludesLocalFragmentsOnly(t *testing.T) {
	t.Parallel()

	doc := mustParseGraphDocument(t, "# Local\n\n[local](#local) [other](other.md#part)\n")
	graph, err := marksplice.BuildDocumentGraph([]marksplice.GraphDocument{{Key: "doc", Document: doc}}, nil)
	if err != nil {
		t.Fatalf("BuildDocumentGraph() error = %v", err)
	}
	edges := graph.Edges()
	if len(edges) != 1 || edges[0].SourceDocument() != "doc" || edges[0].TargetDocument() != "doc" || edges[0].Relationship().Destination() != "#local" {
		t.Fatalf("Edges() = %+v, want one local self edge", edges)
	}
	if target, ok := edges[0].FragmentTarget(); !ok || target.Value() != "local" {
		t.Fatalf("FragmentTarget() = %+v/%v", target, ok)
	}
	if got, ok := graph.ReachableFrom("doc"); !ok || len(got) != 0 {
		t.Fatalf("ReachableFrom(doc) = %#v/%v, want empty/true", got, ok)
	}
	if got, ok := graph.RelatedDocuments("doc"); !ok || len(got) != 0 {
		t.Fatalf("RelatedDocuments(doc) = %#v/%v, want empty/true", got, ok)
	}
}

func TestM100DocumentGraphRejectsMalformedInputsAndResolverTargets(t *testing.T) {
	t.Parallel()

	doc := mustParseGraphDocument(t, "# A\n\n[to-b](b.md)\n")
	zero := &marksplice.Document{}

	tests := []struct {
		name      string
		documents []marksplice.GraphDocument
		resolver  marksplice.DocumentResolver
	}{
		{name: "empty key", documents: []marksplice.GraphDocument{{Key: "", Document: doc}}},
		{name: "duplicate key", documents: []marksplice.GraphDocument{{Key: "a", Document: doc}, {Key: "a", Document: doc}}},
		{name: "nil document", documents: []marksplice.GraphDocument{{Key: "a", Document: nil}}},
		{name: "zero document", documents: []marksplice.GraphDocument{{Key: "a", Document: zero}}},
		{
			name:      "resolver returns unknown target",
			documents: []marksplice.GraphDocument{{Key: "a", Document: doc}},
			resolver: func(marksplice.DocumentKey, marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
				return marksplice.DocumentResolution{Target: "missing"}, true
			},
		},
		{
			name:      "resolver returns empty target",
			documents: []marksplice.GraphDocument{{Key: "a", Document: doc}},
			resolver: func(marksplice.DocumentKey, marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
				return marksplice.DocumentResolution{}, true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := marksplice.BuildDocumentGraph(tt.documents, tt.resolver)
			if !errors.Is(err, marksplice.ErrInvalidGraph) {
				t.Fatalf("BuildDocumentGraph() error = %v, want ErrInvalidGraph", err)
			}
		})
	}

	empty, err := marksplice.BuildDocumentGraph(nil, nil)
	if err != nil {
		t.Fatalf("BuildDocumentGraph(nil,nil) error = %v", err)
	}
	if keys := empty.DocumentKeys(); len(keys) != 0 {
		t.Fatalf("empty DocumentKeys() = %#v", keys)
	}
	if edges := empty.Edges(); len(edges) != 0 {
		t.Fatalf("empty Edges() = %#v", edges)
	}
	if _, ok := empty.Document("missing"); ok {
		t.Fatal("empty graph unexpectedly contains document")
	}
	if got, ok := empty.Outgoing("missing"); ok || got != nil {
		t.Fatalf("empty Outgoing(missing) = %#v/%v", got, ok)
	}
	if got, ok := empty.Backlinks("missing"); ok || got != nil {
		t.Fatalf("empty Backlinks(missing) = %#v/%v", got, ok)
	}
	if got, ok := empty.ReachableFrom("missing"); ok || got != nil {
		t.Fatalf("empty ReachableFrom(missing) = %#v/%v", got, ok)
	}
	if got, ok := empty.RelatedDocuments("missing"); ok || got != nil {
		t.Fatalf("empty RelatedDocuments(missing) = %#v/%v", got, ok)
	}

	var nilGraph *marksplice.DocumentGraph
	if nilGraph.DocumentKeys() != nil || nilGraph.Edges() != nil {
		t.Fatal("nil graph variable-length accessors should return nil")
	}
	if _, ok := nilGraph.Document("missing"); ok {
		t.Fatal("nil graph unexpectedly contains document")
	}
}

func mustParseGraphDocument(t *testing.T, source string) *marksplice.Document {
	t.Helper()
	doc, err := marksplice.Parse([]byte(source))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return doc
}

func graphEdgesDestinations(edges []marksplice.GraphEdge) []string {
	result := make([]string, len(edges))
	for index, edge := range edges {
		result[index] = edge.Relationship().Destination()
	}
	return result
}

func graphEdgesSources(edges []marksplice.GraphEdge) []marksplice.DocumentKey {
	result := make([]marksplice.DocumentKey, len(edges))
	for index, edge := range edges {
		result[index] = edge.SourceDocument()
	}
	return result
}
