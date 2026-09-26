package publictest

import (
	"reflect"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestGraphAndKnowledgeReachabilityPreserveBreadthFirstDiscovery(t *testing.T) {
	t.Parallel()
	graph, err := marksplice.BuildDocumentGraph([]marksplice.GraphDocument{
		{Key: "a", Document: mustParseGraphDocument(t, "[b](b) [c](c)\n")},
		{Key: "b", Document: mustParseGraphDocument(t, "[d](d) [c](c)\n")},
		{Key: "c", Document: mustParseGraphDocument(t, "[d](d) [e](e)\n")},
		{Key: "d", Document: mustParseGraphDocument(t, "[a](a)\n")},
		{Key: "e", Document: mustParseGraphDocument(t, "[e](e)\n")},
	}, func(_ marksplice.DocumentKey, relationship marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
		return marksplice.DocumentResolution{Target: marksplice.DocumentKey(relationship.Destination())}, true
	})
	if err != nil {
		t.Fatal(err)
	}
	knowledge, err := marksplice.BuildKnowledgeIndex(graph, []marksplice.KnowledgeDocument{
		{Document: "c", References: []marksplice.DocumentKey{"b"}},
		{Document: "a", References: []marksplice.DocumentKey{"e", "a"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name  string
		visit func(marksplice.DocumentKey) ([]marksplice.DocumentKey, bool)
		want  []marksplice.DocumentKey
	}{
		{"graph", graph.ReachableFrom, []marksplice.DocumentKey{"b", "c", "d", "e"}},
		{"knowledge", knowledge.ReachableFrom, []marksplice.DocumentKey{"b", "c", "e", "d"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			first, ok := test.visit("a")
			if !ok || !reflect.DeepEqual(first, test.want) {
				t.Fatalf("visit(a) = %v/%v, want %v", first, ok, test.want)
			}
			first[0] = "changed"
			again, ok := test.visit("a")
			if !ok || !reflect.DeepEqual(again, test.want) {
				t.Fatal("result mutation changed subsequent traversal")
			}
			selfOnly, ok := test.visit("e")
			if !ok || selfOnly == nil || len(selfOnly) != 0 {
				t.Fatalf("self-only visit = %v/%v, want non-nil empty result", selfOnly, ok)
			}
		})
	}
}
