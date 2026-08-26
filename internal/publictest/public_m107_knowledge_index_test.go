package publictest

import (
	"errors"
	"reflect"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM107KnowledgeIndexAddsSyntaxIndependentAliasesTagsAndLogicalReferences(t *testing.T) {
	t.Parallel()

	a := mustParseGraphDocument(t, "# A\n\n[to-b](b.md)\n")
	b := mustParseGraphDocument(t, "# B\n")
	c := mustParseGraphDocument(t, "# C\n")
	graph, err := marksplice.BuildDocumentGraph([]marksplice.GraphDocument{
		{Key: "a", Document: a},
		{Key: "b", Document: b},
		{Key: "c", Document: c},
	}, func(_ marksplice.DocumentKey, relationship marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
		if relationship.Destination() == "b.md" {
			return marksplice.DocumentResolution{Target: "b"}, true
		}
		return marksplice.DocumentResolution{}, false
	})
	if err != nil {
		t.Fatalf("BuildDocumentGraph() error = %v", err)
	}

	aliases := []marksplice.KnowledgeAlias{"Alpha", "Entry A"}
	tags := []marksplice.KnowledgeTag{"project", "active"}
	references := []marksplice.DocumentKey{"b", "c"}
	// Metadata input order is deliberately different from graph order. Public reference
	// enumeration must remain graph-document ordered, not metadata-slice ordered.
	metadata := []marksplice.KnowledgeDocument{
		{Document: "c", Aliases: []marksplice.KnowledgeAlias{"Gamma"}, Tags: []marksplice.KnowledgeTag{"archive"}, References: []marksplice.DocumentKey{"b"}},
		{Document: "a", Aliases: aliases, Tags: tags, References: references},
		{Document: "b", Aliases: []marksplice.KnowledgeAlias{"Beta"}, Tags: []marksplice.KnowledgeTag{"project"}},
	}
	index, err := marksplice.BuildKnowledgeIndex(graph, metadata)
	if err != nil {
		t.Fatalf("BuildKnowledgeIndex() error = %v", err)
	}

	// Caller-owned input must not alias the retained index.
	aliases[0] = "mutated"
	tags[0] = "mutated"
	references[0] = "c"
	metadata[0] = marksplice.KnowledgeDocument{Document: "mutated"}

	if got, ok := index.Aliases("a"); !ok || !reflect.DeepEqual(got, []marksplice.KnowledgeAlias{"Alpha", "Entry A"}) {
		t.Fatalf("Aliases(a) = %#v/%v", got, ok)
	}
	if got, ok := index.Tags("a"); !ok || !reflect.DeepEqual(got, []marksplice.KnowledgeTag{"project", "active"}) {
		t.Fatalf("Tags(a) = %#v/%v", got, ok)
	}
	if got, ok := index.ResolveAlias("Beta"); !ok || got != "b" {
		t.Fatalf("ResolveAlias(Beta) = %q/%v", got, ok)
	}
	if got, ok := index.ResolveAlias("beta"); ok || got != "" {
		t.Fatalf("ResolveAlias(beta) = %q/%v, want exact case-sensitive miss", got, ok)
	}
	if got := index.DocumentsWithTag("project"); !reflect.DeepEqual(got, []marksplice.DocumentKey{"a", "b"}) {
		t.Fatalf("DocumentsWithTag(project) = %#v", got)
	}
	if got := index.DocumentsWithTag("Project"); got != nil {
		t.Fatalf("DocumentsWithTag(Project) = %#v, want nil", got)
	}

	all := index.References()
	if len(all) != 3 || all[0].SourceDocument() != "a" || all[0].TargetDocument() != "b" || all[1].SourceDocument() != "a" || all[1].TargetDocument() != "c" || all[2].SourceDocument() != "c" || all[2].TargetDocument() != "b" {
		t.Fatalf("References() = %#v", all)
	}
	if got, ok := index.ReferencesFrom("a"); !ok || len(got) != 2 || got[0].TargetDocument() != "b" || got[1].TargetDocument() != "c" {
		t.Fatalf("ReferencesFrom(a) = %#v/%v", got, ok)
	}
	if got, ok := index.ReferencedBy("b"); !ok || len(got) != 2 || got[0].SourceDocument() != "a" || got[1].SourceDocument() != "c" {
		t.Fatalf("ReferencedBy(b) = %#v/%v", got, ok)
	}
	if edges := graph.Edges(); len(edges) != 1 {
		t.Fatalf("BuildKnowledgeIndex changed M100 graph edges: %#v", edges)
	}

	// Combined reachability visits M100 Markdown edges before M107 logical references.
	// The logical a->b duplicates an existing Markdown target but must not duplicate b.
	if got, ok := index.ReachableFrom("a"); !ok || !reflect.DeepEqual(got, []marksplice.DocumentKey{"b", "c"}) {
		t.Fatalf("ReachableFrom(a) = %#v/%v", got, ok)
	}
	if got, ok := index.RelatedDocuments("c"); !ok || !reflect.DeepEqual(got, []marksplice.DocumentKey{"a", "b"}) {
		t.Fatalf("RelatedDocuments(c) = %#v/%v", got, ok)
	}

	// Variable-length outputs must be caller-owned.
	gotAliases, _ := index.Aliases("a")
	gotTags, _ := index.Tags("a")
	gotRefs, _ := index.ReferencesFrom("a")
	gotReachable, _ := index.ReachableFrom("a")
	gotRelated, _ := index.RelatedDocuments("c")
	gotAliases[0] = "changed"
	gotTags[0] = "changed"
	gotRefs[0] = marksplice.KnowledgeReference{}
	gotReachable[0] = "changed"
	gotRelated[0] = "changed"
	if again, _ := index.Aliases("a"); again[0] != "Alpha" {
		t.Fatalf("Aliases caller mutation leaked: %#v", again)
	}
	if again, _ := index.Tags("a"); again[0] != "project" {
		t.Fatalf("Tags caller mutation leaked: %#v", again)
	}
	if again, _ := index.ReferencesFrom("a"); len(again) != 2 || again[0].TargetDocument() != "b" || again[1].TargetDocument() != "c" {
		t.Fatalf("ReferencesFrom caller mutation leaked: %#v", again)
	}
}

func TestM107KnowledgeIndexAllowsSubsetMetadataSelfReferenceAndEmptyGraph(t *testing.T) {
	t.Parallel()

	doc := mustParseGraphDocument(t, "# A\n")
	graph, err := marksplice.BuildDocumentGraph([]marksplice.GraphDocument{{Key: "a", Document: doc}, {Key: "b", Document: doc}}, nil)
	if err != nil {
		t.Fatalf("BuildDocumentGraph() error = %v", err)
	}
	index, err := marksplice.BuildKnowledgeIndex(graph, []marksplice.KnowledgeDocument{{Document: "a", References: []marksplice.DocumentKey{"a"}}})
	if err != nil {
		t.Fatalf("BuildKnowledgeIndex() error = %v", err)
	}
	if aliases, ok := index.Aliases("b"); !ok || aliases != nil {
		t.Fatalf("Aliases(b) = %#v/%v, want nil/true", aliases, ok)
	}
	if tags, ok := index.Tags("b"); !ok || tags != nil {
		t.Fatalf("Tags(b) = %#v/%v, want nil/true", tags, ok)
	}
	if got, ok := index.ReachableFrom("a"); !ok || len(got) != 0 {
		t.Fatalf("ReachableFrom(self-only a) = %#v/%v", got, ok)
	}
	if got, ok := index.RelatedDocuments("a"); !ok || len(got) != 0 {
		t.Fatalf("RelatedDocuments(self-only a) = %#v/%v", got, ok)
	}

	emptyGraph, err := marksplice.BuildDocumentGraph(nil, nil)
	if err != nil {
		t.Fatalf("BuildDocumentGraph(nil,nil) error = %v", err)
	}
	empty, err := marksplice.BuildKnowledgeIndex(emptyGraph, nil)
	if err != nil {
		t.Fatalf("BuildKnowledgeIndex(empty,nil) error = %v", err)
	}
	if empty.References() != nil || empty.DocumentsWithTag("anything") != nil {
		t.Fatal("empty knowledge index returned data")
	}
	if _, ok := empty.Aliases("missing"); ok {
		t.Fatal("empty knowledge index unexpectedly knows missing key")
	}
}

func TestM107KnowledgeIndexRejectsMalformedMetadata(t *testing.T) {
	t.Parallel()

	doc := mustParseGraphDocument(t, "# A\n")
	graph, err := marksplice.BuildDocumentGraph([]marksplice.GraphDocument{{Key: "a", Document: doc}, {Key: "b", Document: doc}}, nil)
	if err != nil {
		t.Fatalf("BuildDocumentGraph() error = %v", err)
	}

	invalidUTF8 := string([]byte{0xff})
	tests := []struct {
		name     string
		graph    *marksplice.DocumentGraph
		metadata []marksplice.KnowledgeDocument
	}{
		{name: "nil graph", graph: nil},
		{name: "unknown document", graph: graph, metadata: []marksplice.KnowledgeDocument{{Document: "missing"}}},
		{name: "duplicate document metadata", graph: graph, metadata: []marksplice.KnowledgeDocument{{Document: "a"}, {Document: "a"}}},
		{name: "empty alias", graph: graph, metadata: []marksplice.KnowledgeDocument{{Document: "a", Aliases: []marksplice.KnowledgeAlias{""}}}},
		{name: "invalid UTF-8 alias", graph: graph, metadata: []marksplice.KnowledgeDocument{{Document: "a", Aliases: []marksplice.KnowledgeAlias{marksplice.KnowledgeAlias(invalidUTF8)}}}},
		{name: "multiline alias", graph: graph, metadata: []marksplice.KnowledgeDocument{{Document: "a", Aliases: []marksplice.KnowledgeAlias{"one\ntwo"}}}},
		{name: "duplicate alias same document", graph: graph, metadata: []marksplice.KnowledgeDocument{{Document: "a", Aliases: []marksplice.KnowledgeAlias{"same", "same"}}}},
		{name: "duplicate alias across documents", graph: graph, metadata: []marksplice.KnowledgeDocument{{Document: "a", Aliases: []marksplice.KnowledgeAlias{"same"}}, {Document: "b", Aliases: []marksplice.KnowledgeAlias{"same"}}}},
		{name: "alias collides with canonical key", graph: graph, metadata: []marksplice.KnowledgeDocument{{Document: "a", Aliases: []marksplice.KnowledgeAlias{"b"}}}},
		{name: "empty tag", graph: graph, metadata: []marksplice.KnowledgeDocument{{Document: "a", Tags: []marksplice.KnowledgeTag{""}}}},
		{name: "invalid UTF-8 tag", graph: graph, metadata: []marksplice.KnowledgeDocument{{Document: "a", Tags: []marksplice.KnowledgeTag{marksplice.KnowledgeTag(invalidUTF8)}}}},
		{name: "multiline tag", graph: graph, metadata: []marksplice.KnowledgeDocument{{Document: "a", Tags: []marksplice.KnowledgeTag{"one\ntwo"}}}},
		{name: "duplicate tag", graph: graph, metadata: []marksplice.KnowledgeDocument{{Document: "a", Tags: []marksplice.KnowledgeTag{"same", "same"}}}},
		{name: "unknown reference target", graph: graph, metadata: []marksplice.KnowledgeDocument{{Document: "a", References: []marksplice.DocumentKey{"missing"}}}},
		{name: "duplicate reference", graph: graph, metadata: []marksplice.KnowledgeDocument{{Document: "a", References: []marksplice.DocumentKey{"b", "b"}}}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := marksplice.BuildKnowledgeIndex(tt.graph, tt.metadata); !errors.Is(err, marksplice.ErrInvalidKnowledge) {
				t.Fatalf("BuildKnowledgeIndex() error = %v, want ErrInvalidKnowledge", err)
			}
		})
	}
}

func TestM107KnowledgeIndexNilBehavior(t *testing.T) {
	t.Parallel()

	var index *marksplice.KnowledgeIndex
	if index.References() != nil || index.DocumentsWithTag("tag") != nil {
		t.Fatal("nil knowledge index variable-length accessors should return nil")
	}
	if got, ok := index.Aliases("a"); ok || got != nil {
		t.Fatalf("nil Aliases(a) = %#v/%v", got, ok)
	}
	if got, ok := index.Tags("a"); ok || got != nil {
		t.Fatalf("nil Tags(a) = %#v/%v", got, ok)
	}
	if got, ok := index.ResolveAlias("alias"); ok || got != "" {
		t.Fatalf("nil ResolveAlias(alias) = %q/%v", got, ok)
	}
	if got, ok := index.ReferencesFrom("a"); ok || got != nil {
		t.Fatalf("nil ReferencesFrom(a) = %#v/%v", got, ok)
	}
	if got, ok := index.ReferencedBy("a"); ok || got != nil {
		t.Fatalf("nil ReferencedBy(a) = %#v/%v", got, ok)
	}
	if got, ok := index.ReachableFrom("a"); ok || got != nil {
		t.Fatalf("nil ReachableFrom(a) = %#v/%v", got, ok)
	}
	if got, ok := index.RelatedDocuments("a"); ok || got != nil {
		t.Fatalf("nil RelatedDocuments(a) = %#v/%v", got, ok)
	}

	var reference marksplice.KnowledgeReference
	if reference.SourceDocument() != "" || reference.TargetDocument() != "" {
		t.Fatalf("zero KnowledgeReference = %q -> %q", reference.SourceDocument(), reference.TargetDocument())
	}
}
