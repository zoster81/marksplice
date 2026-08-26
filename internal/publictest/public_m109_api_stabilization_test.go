package publictest

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM109WorkspaceUnresolvedReferenceUsesTypedValue(t *testing.T) {
	t.Parallel()

	document, err := marksplice.Parse([]byte("![image][missing]\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	report, err := marksplice.ValidateWorkspace([]marksplice.GraphDocument{{
		Key:      "doc",
		Document: document,
	}}, nil, marksplice.WorkspaceValidationOptions{})
	if err != nil {
		t.Fatalf("ValidateWorkspace() error = %v", err)
	}
	diagnostics := report.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Kind() != marksplice.WorkspaceDiagnosticUnresolvedReference {
		t.Fatalf("Diagnostics() = %#v, want one unresolved-reference diagnostic", diagnostics)
	}

	unresolved, ok := diagnostics[0].UnresolvedReference()
	if !ok {
		t.Fatal("UnresolvedReference() ok = false, want true")
	}
	if unresolved.Reference() != "missing" || unresolved.Form() != marksplice.ReferenceFormFull || !unresolved.IsImage() {
		t.Fatalf("UnresolvedReference() = %q/%v/image=%t, want missing/full/image", unresolved.Reference(), unresolved.Form(), unresolved.IsImage())
	}
}

func TestM109ImmutablePublicModelsSupportConcurrentReads(t *testing.T) {
	t.Parallel()

	sourceA := []byte("# A\n\n[to-b](b.md#b)\n\n[missing][nope]\n")
	sourceB := []byte("# B\n")
	docA, err := marksplice.Parse(sourceA)
	if err != nil {
		t.Fatalf("Parse(a) error = %v", err)
	}
	docB, err := marksplice.Parse(sourceB)
	if err != nil {
		t.Fatalf("Parse(b) error = %v", err)
	}
	documents := []marksplice.GraphDocument{{Key: "a", Document: docA}, {Key: "b", Document: docB}}
	graph, err := marksplice.BuildDocumentGraph(documents, func(source marksplice.DocumentKey, relationship marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
		if source == "a" && relationship.Destination() == "b.md#b" {
			return marksplice.DocumentResolution{Target: "b", Fragment: "#b"}, true
		}
		return marksplice.DocumentResolution{}, false
	})
	if err != nil {
		t.Fatalf("BuildDocumentGraph() error = %v", err)
	}
	knowledge, err := marksplice.BuildKnowledgeIndex(graph, []marksplice.KnowledgeDocument{
		{Document: "a", Aliases: []marksplice.KnowledgeAlias{"alpha"}, Tags: []marksplice.KnowledgeTag{"docs"}, References: []marksplice.DocumentKey{"b"}},
		{Document: "b", Tags: []marksplice.KnowledgeTag{"docs"}},
	})
	if err != nil {
		t.Fatalf("BuildKnowledgeIndex() error = %v", err)
	}
	report, err := marksplice.ValidateWorkspace(documents, func(source marksplice.DocumentKey, relationship marksplice.LinkRelationship) marksplice.WorkspaceResolution {
		if source == "a" && relationship.Destination() == "b.md#b" {
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionResolved, Target: "b", Fragment: "#b"}
		}
		return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionIgnore}
	}, marksplice.WorkspaceValidationOptions{Roots: []marksplice.DocumentKey{"a"}})
	if err != nil {
		t.Fatalf("ValidateWorkspace() error = %v", err)
	}
	anchors := docA.HeadingAnchors()
	if len(anchors) != 1 {
		t.Fatalf("HeadingAnchors() count = %d, want 1", len(anchors))
	}
	change, err := docA.PrepareRenameHeading(anchors[0].HeadingID(), []byte("A2"))
	if err != nil {
		t.Fatalf("PrepareRenameHeading() error = %v", err)
	}

	const workers = 8
	const iterations = 32
	errors := make(chan error, workers)
	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for iteration := 0; iteration < iterations; iteration++ {
				if err := exerciseM109ConcurrentReads(docA, graph, knowledge, report, change, sourceA); err != nil {
					errors <- err
					return
				}
			}
		}()
	}
	group.Wait()
	close(errors)
	for err := range errors {
		t.Fatal(err)
	}
}

func exerciseM109ConcurrentReads(document *marksplice.Document, graph *marksplice.DocumentGraph, knowledge *marksplice.KnowledgeIndex, report *marksplice.WorkspaceReport, change marksplice.ChangeSet, source []byte) error {
	nodes := document.Nodes()
	if len(nodes) == 0 {
		return fmt.Errorf("Nodes() returned no nodes")
	}
	nodes[0] = marksplice.Node{}
	if again := document.Nodes(); len(again) == 0 || again[0].Kind() == marksplice.KindUnknown {
		return fmt.Errorf("caller mutation leaked into Nodes()")
	}
	sections := document.Sections()
	if len(sections) == 0 {
		return fmt.Errorf("Sections() returned no sections")
	}
	sections[0] = marksplice.Section{}
	if again := document.Sections(); len(again) == 0 || again[0].HeadingID() == (marksplice.NodeID{}) {
		return fmt.Errorf("caller mutation leaked into Sections()")
	}
	if toc := document.GenerateTOC(); len(toc) != 0 {
		toc[0] ^= 1
		if bytes.Equal(toc, document.GenerateTOC()) {
			return fmt.Errorf("caller mutation leaked into GenerateTOC()")
		}
	}
	relationships := document.LinkRelationships()
	if len(relationships) == 0 {
		return fmt.Errorf("LinkRelationships() returned no relationships")
	}
	relationships[0] = marksplice.LinkRelationship{}
	if again := document.LinkRelationships(); len(again) == 0 || again[0].Destination() == "" {
		return fmt.Errorf("caller mutation leaked into LinkRelationships()")
	}

	edges := graph.Edges()
	if len(edges) != 1 {
		return fmt.Errorf("Edges() count = %d, want 1", len(edges))
	}
	edges[0] = marksplice.GraphEdge{}
	if again := graph.Edges(); len(again) != 1 || again[0].TargetDocument() != "b" {
		return fmt.Errorf("caller mutation leaked into graph Edges()")
	}
	if reachable, ok := graph.ReachableFrom("a"); !ok || len(reachable) != 1 || reachable[0] != "b" {
		return fmt.Errorf("graph ReachableFrom(a) = %v/%v", reachable, ok)
	}

	aliases, ok := knowledge.Aliases("a")
	if !ok || len(aliases) != 1 {
		return fmt.Errorf("Aliases(a) = %v/%v", aliases, ok)
	}
	aliases[0] = "changed"
	if again, ok := knowledge.Aliases("a"); !ok || len(again) != 1 || again[0] != "alpha" {
		return fmt.Errorf("caller mutation leaked into knowledge aliases")
	}
	if resolved, ok := knowledge.ResolveAlias("alpha"); !ok || resolved != "a" {
		return fmt.Errorf("ResolveAlias(alpha) = %q/%v", resolved, ok)
	}
	if reachable, ok := knowledge.ReachableFrom("a"); !ok || len(reachable) != 1 || reachable[0] != "b" {
		return fmt.Errorf("knowledge ReachableFrom(a) = %v/%v", reachable, ok)
	}

	diagnostics := report.Diagnostics()
	if len(diagnostics) == 0 {
		return fmt.Errorf("Diagnostics() returned no findings")
	}
	diagnostics[0] = marksplice.WorkspaceDiagnostic{}
	if again := report.Diagnostics(); len(again) == 0 || again[0].Kind() == marksplice.WorkspaceDiagnosticUnknown {
		return fmt.Errorf("caller mutation leaked into workspace diagnostics")
	}
	if report.Graph() == nil {
		return fmt.Errorf("WorkspaceReport.Graph() returned nil")
	}

	updated, err := change.Apply(source)
	if err != nil {
		return fmt.Errorf("ChangeSet.Apply() error = %w", err)
	}
	if !bytes.Contains(updated, []byte("# A2\n")) {
		return fmt.Errorf("ChangeSet.Apply() did not produce expected heading")
	}
	return nil
}
