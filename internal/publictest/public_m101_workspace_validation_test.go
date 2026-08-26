package publictest

import (
	"errors"
	"reflect"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM101ValidateWorkspaceReportsBrokenRelationshipsOrphansAndStaleTOC(t *testing.T) {
	t.Parallel()

	sourceA := []byte("# A\n\n## Contents\n\n- [A](#a)\n\n## Good\n\n[local-missing](#missing) [local-ambiguous](#dup) [local-invalid](#)\n\n<a id=\"dup\"></a>\n\n## Dup\n\n[missing-doc](missing.md#part) [cross-missing](b.md#missing) [cross-ambiguous](b.md#dup) [external](https://example.com)\n\n[badref][nope] [collapsed][] ![image][missing-image]\n\n`[code][nope]` [ordinary]\n\n[resolved][ok]\n\n[ok]: /resolved\n")
	sourceB := []byte("# B\n\n<a id=\"dup\"></a>\n\n## Dup\n")
	sourceC := []byte("# Orphan\n")

	docA := mustParseGraphDocument(t, string(sourceA))
	docB := mustParseGraphDocument(t, string(sourceB))
	docC := mustParseGraphDocument(t, string(sourceC))
	contents := publicSectionsByHeadingText(t, docA, sourceA, docA.Sections())["Contents"]

	resolverCalls := 0
	resolverActive := true
	resolver := func(_ marksplice.DocumentKey, relationship marksplice.LinkRelationship) marksplice.WorkspaceResolution {
		if !resolverActive {
			panic("workspace resolver retained after ValidateWorkspace")
		}
		resolverCalls++
		switch relationship.Destination() {
		case "missing.md#part":
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionMissing, Target: "missing", Fragment: "#part"}
		case "b.md#missing":
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionResolved, Target: "b", Fragment: "#missing"}
		case "b.md#dup":
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionResolved, Target: "b", Fragment: "#dup"}
		case "https://example.com":
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionIgnore}
		default:
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionIgnore}
		}
	}

	report, err := marksplice.ValidateWorkspace([]marksplice.GraphDocument{
		{Key: "a", Document: docA},
		{Key: "b", Document: docB},
		{Key: "c", Document: docC},
	}, resolver, marksplice.WorkspaceValidationOptions{
		Roots:       []marksplice.DocumentKey{"a"},
		ManagedTOCs: []marksplice.ManagedTOC{{Document: "a", HeadingID: contents.HeadingID()}},
	})
	if err != nil {
		t.Fatalf("ValidateWorkspace() error = %v", err)
	}
	resolverActive = false
	if resolverCalls != 5 {
		t.Fatalf("resolver calls = %d, want 5 non-local relationships", resolverCalls)
	}
	if graph := report.Graph(); graph == nil {
		t.Fatal("Graph() = nil")
	} else if got, ok := graph.ReachableFrom("a"); !ok || !reflect.DeepEqual(got, []marksplice.DocumentKey{"b"}) {
		t.Fatalf("Graph().ReachableFrom(a) = %#v/%v", got, ok)
	}

	diagnostics := report.Diagnostics()
	wantKinds := []marksplice.WorkspaceDiagnosticKind{
		marksplice.WorkspaceDiagnosticMissingFragment,
		marksplice.WorkspaceDiagnosticAmbiguousFragment,
		marksplice.WorkspaceDiagnosticInvalidFragment,
		marksplice.WorkspaceDiagnosticMissingDocument,
		marksplice.WorkspaceDiagnosticMissingFragment,
		marksplice.WorkspaceDiagnosticAmbiguousFragment,
		marksplice.WorkspaceDiagnosticUnresolvedReference,
		marksplice.WorkspaceDiagnosticUnresolvedReference,
		marksplice.WorkspaceDiagnosticUnresolvedReference,
		marksplice.WorkspaceDiagnosticOrphanDocument,
		marksplice.WorkspaceDiagnosticStaleGeneratedIndex,
	}
	gotKinds := make([]marksplice.WorkspaceDiagnosticKind, len(diagnostics))
	for index, diagnostic := range diagnostics {
		gotKinds[index] = diagnostic.Kind()
	}
	if !reflect.DeepEqual(gotKinds, wantKinds) {
		t.Fatalf("diagnostic kinds = %#v, want %#v", gotKinds, wantKinds)
	}

	for _, index := range []int{0, 1, 2, 3, 4, 5} {
		if _, ok := diagnostics[index].Relationship(); !ok {
			t.Fatalf("diagnostic[%d] relationship unavailable", index)
		}
		if _, ok := diagnostics[index].SourceOffset(); !ok {
			t.Fatalf("diagnostic[%d] source offset unavailable", index)
		}
		if source, ok := diagnostics[index].SourceDocument(); !ok || source != "a" {
			t.Fatalf("diagnostic[%d] SourceDocument() = %q/%v, want a/true", index, source, ok)
		}
	}
	if target, ok := diagnostics[3].TargetDocument(); !ok || target != "missing" {
		t.Fatalf("missing-document TargetDocument() = %q/%v", target, ok)
	}
	if fragment, ok := diagnostics[4].Fragment(); !ok || fragment != "#missing" {
		t.Fatalf("cross-missing Fragment() = %q/%v", fragment, ok)
	}
	if target, ok := diagnostics[4].TargetDocument(); !ok || target != "b" {
		t.Fatalf("cross-missing TargetDocument() = %q/%v", target, ok)
	}

	wantReferences := []struct {
		value string
		form  marksplice.ReferenceForm
		image bool
	}{
		{value: "nope", form: marksplice.ReferenceFormFull},
		{value: "collapsed", form: marksplice.ReferenceFormCollapsed},
		{value: "missing-image", form: marksplice.ReferenceFormFull, image: true},
	}
	for offset, want := range wantReferences {
		diagnostic := diagnostics[6+offset]
		unresolved, ok := diagnostic.UnresolvedReference()
		if !ok || unresolved.Reference() != want.value || unresolved.Form() != want.form || unresolved.IsImage() != want.image {
			t.Fatalf("UnresolvedReference[%d] = %q/%v/%v/%v, want %q/%v/%v/true", offset, unresolved.Reference(), unresolved.Form(), unresolved.IsImage(), ok, want.value, want.form, want.image)
		}
		if _, ok := diagnostic.SourceOffset(); !ok {
			t.Fatalf("unresolved diagnostic[%d] missing source offset", offset)
		}
		if source, ok := diagnostic.SourceDocument(); !ok || source != "a" {
			t.Fatalf("unresolved diagnostic[%d] SourceDocument() = %q/%v, want a/true", offset, source, ok)
		}
		if _, ok := diagnostic.Relationship(); ok {
			t.Fatalf("unresolved diagnostic[%d] unexpectedly has M99 relationship", offset)
		}
	}
	if orphan, ok := diagnostics[9].TargetDocument(); !ok || orphan != "c" {
		t.Fatalf("orphan TargetDocument() = %q/%v", orphan, ok)
	}
	if _, ok := diagnostics[9].SourceDocument(); ok {
		t.Fatal("orphan diagnostic unexpectedly has a source document")
	}
	if source, ok := diagnostics[10].SourceDocument(); !ok || source != "a" {
		t.Fatalf("stale generated-index SourceDocument() = %q/%v, want a/true", source, ok)
	}
	if nodeID, ok := diagnostics[10].NodeID(); !ok || nodeID != contents.HeadingID() {
		t.Fatalf("stale generated-index NodeID() = %v/%v", nodeID, ok)
	}

	repairs := report.RepairPlan().Repairs()
	if len(repairs) != 1 || repairs[0].Document() != "a" {
		t.Fatalf("repairs = %#v, want one repair for a", repairs)
	}
	updated, err := repairs[0].Change().Apply(sourceA)
	if err != nil {
		t.Fatalf("repair Apply() error = %v", err)
	}
	updatedDoc := mustParseGraphDocument(t, string(updated))
	updatedContents := publicSectionsByHeadingText(t, updatedDoc, updated, updatedDoc.Sections())["Contents"]
	if stale, recognized := updatedDoc.TOCStale(updatedContents.HeadingID()); !recognized || stale {
		t.Fatalf("repaired TOC stale/recognized = %v/%v, want false/true", stale, recognized)
	}

	diagnostics[0] = marksplice.WorkspaceDiagnostic{}
	repairs[0] = marksplice.WorkspaceRepair{}
	if again := report.Diagnostics(); len(again) != len(wantKinds) || again[0].Kind() == marksplice.WorkspaceDiagnosticUnknown {
		t.Fatal("caller mutation leaked into report diagnostics")
	}
	if again := report.RepairPlan().Repairs(); len(again) != 1 || again[0].Document() != "a" {
		t.Fatal("caller mutation leaked into repair plan")
	}
}

func TestM101ValidateWorkspaceReportsUnrecognizedManagedTOCWithoutRepair(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n\n## Notes\n\nordinary prose\n")
	doc := mustParseGraphDocument(t, string(source))
	notes := publicSectionsByHeadingText(t, doc, source, doc.Sections())["Notes"]

	report, err := marksplice.ValidateWorkspace([]marksplice.GraphDocument{{Key: "doc", Document: doc}}, nil, marksplice.WorkspaceValidationOptions{
		ManagedTOCs: []marksplice.ManagedTOC{{Document: "doc", HeadingID: notes.HeadingID()}},
	})
	if err != nil {
		t.Fatalf("ValidateWorkspace() error = %v", err)
	}
	diagnostics := report.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Kind() != marksplice.WorkspaceDiagnosticUnrecognizedGeneratedIndex {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if len(report.RepairPlan().Repairs()) != 0 {
		t.Fatalf("unrecognized generated index unexpectedly has repair: %#v", report.RepairPlan().Repairs())
	}
}

func TestM101ValidateWorkspaceComposesMultipleManagedTOCRepairsPerDocument(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n\n## Contents One\n\n## Child\n\n## Contents Two\n")
	doc := mustParseGraphDocument(t, string(source))
	sections := publicSectionsByHeadingText(t, doc, source, doc.Sections())

	report, err := marksplice.ValidateWorkspace([]marksplice.GraphDocument{{Key: "doc", Document: doc}}, nil, marksplice.WorkspaceValidationOptions{
		ManagedTOCs: []marksplice.ManagedTOC{
			{Document: "doc", HeadingID: sections["Contents One"].HeadingID()},
			{Document: "doc", HeadingID: sections["Contents Two"].HeadingID()},
		},
	})
	if err != nil {
		t.Fatalf("ValidateWorkspace() error = %v", err)
	}
	diagnostics := report.Diagnostics()
	if len(diagnostics) != 2 || diagnostics[0].Kind() != marksplice.WorkspaceDiagnosticStaleGeneratedIndex || diagnostics[1].Kind() != marksplice.WorkspaceDiagnosticStaleGeneratedIndex {
		t.Fatalf("diagnostics = %#v, want two stale generated indexes", diagnostics)
	}
	repairs := report.RepairPlan().Repairs()
	if len(repairs) != 1 || repairs[0].Document() != "doc" {
		t.Fatalf("repairs = %#v, want one composed repair for doc", repairs)
	}
	updated, err := repairs[0].Change().Apply(source)
	if err != nil {
		t.Fatalf("composed repair Apply() error = %v", err)
	}
	updatedDoc := mustParseGraphDocument(t, string(updated))
	updatedSections := publicSectionsByHeadingText(t, updatedDoc, updated, updatedDoc.Sections())
	for _, name := range []string{"Contents One", "Contents Two"} {
		if stale, recognized := updatedDoc.TOCStale(updatedSections[name].HeadingID()); !recognized || stale {
			t.Fatalf("%s repaired TOC stale/recognized = %v/%v, want false/true", name, stale, recognized)
		}
	}
}

func TestM101ValidateWorkspaceExcludesFrontMatterAndAmbiguousShortcutTextFromUnresolvedReferences(t *testing.T) {
	t.Parallel()

	source := []byte("---\ntitle: \"[front][missing]\"\n---\n\n[body][missing] [ordinary] \\[escaped][missing]\n")
	doc := mustParseGraphDocument(t, string(source))
	report, err := marksplice.ValidateWorkspace([]marksplice.GraphDocument{{Key: "doc", Document: doc}}, nil, marksplice.WorkspaceValidationOptions{})
	if err != nil {
		t.Fatalf("ValidateWorkspace() error = %v", err)
	}
	diagnostics := report.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Kind() != marksplice.WorkspaceDiagnosticUnresolvedReference {
		t.Fatalf("diagnostics = %#v, want one body unresolved reference", diagnostics)
	}
	unresolved, ok := diagnostics[0].UnresolvedReference()
	if !ok || unresolved.Reference() != "missing" || unresolved.Form() != marksplice.ReferenceFormFull || unresolved.IsImage() {
		t.Fatalf("UnresolvedReference() = %q/%v/%v/%v", unresolved.Reference(), unresolved.Form(), unresolved.IsImage(), ok)
	}
}

func TestM101ValidateWorkspaceUsesMultiRootReachabilityWithoutDuplicateTraversalSemantics(t *testing.T) {
	t.Parallel()

	docA := mustParseGraphDocument(t, "# A\n\n[to-c](c.md)\n")
	docB := mustParseGraphDocument(t, "# B\n")
	docC := mustParseGraphDocument(t, "# C\n")
	docD := mustParseGraphDocument(t, "# D\n")
	report, err := marksplice.ValidateWorkspace([]marksplice.GraphDocument{
		{Key: "a", Document: docA},
		{Key: "b", Document: docB},
		{Key: "c", Document: docC},
		{Key: "d", Document: docD},
	}, func(_ marksplice.DocumentKey, relationship marksplice.LinkRelationship) marksplice.WorkspaceResolution {
		if relationship.Destination() == "c.md" {
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionResolved, Target: "c"}
		}
		return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionIgnore}
	}, marksplice.WorkspaceValidationOptions{Roots: []marksplice.DocumentKey{"a", "b"}})
	if err != nil {
		t.Fatalf("ValidateWorkspace() error = %v", err)
	}
	diagnostics := report.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Kind() != marksplice.WorkspaceDiagnosticOrphanDocument {
		t.Fatalf("diagnostics = %#v, want one orphan", diagnostics)
	}
	if orphan, ok := diagnostics[0].TargetDocument(); !ok || orphan != "d" {
		t.Fatalf("orphan TargetDocument() = %q/%v, want d/true", orphan, ok)
	}
}

func TestM101ValidateWorkspaceRejectsMalformedAuthorityAndTargets(t *testing.T) {
	t.Parallel()

	doc := mustParseGraphDocument(t, "# Root\n\n[other](other.md)\n")
	source := []marksplice.GraphDocument{{Key: "doc", Document: doc}}
	rootHeading := doc.Sections()[0].HeadingID()
	foreignDoc := mustParseGraphDocument(t, "# Foreign\n")
	foreignHeading := foreignDoc.Sections()[0].HeadingID()

	tests := []struct {
		name     string
		resolver marksplice.WorkspaceResolver
		options  marksplice.WorkspaceValidationOptions
	}{
		{name: "empty root", options: marksplice.WorkspaceValidationOptions{Roots: []marksplice.DocumentKey{""}}},
		{name: "unknown root", options: marksplice.WorkspaceValidationOptions{Roots: []marksplice.DocumentKey{"missing"}}},
		{name: "duplicate root", options: marksplice.WorkspaceValidationOptions{Roots: []marksplice.DocumentKey{"doc", "doc"}}},
		{name: "unknown managed document", options: marksplice.WorkspaceValidationOptions{ManagedTOCs: []marksplice.ManagedTOC{{Document: "missing"}}}},
		{name: "zero managed heading", options: marksplice.WorkspaceValidationOptions{ManagedTOCs: []marksplice.ManagedTOC{{Document: "doc"}}}},
		{name: "foreign managed heading", options: marksplice.WorkspaceValidationOptions{ManagedTOCs: []marksplice.ManagedTOC{{Document: "doc", HeadingID: foreignHeading}}}},
		{name: "duplicate managed target", options: marksplice.WorkspaceValidationOptions{ManagedTOCs: []marksplice.ManagedTOC{{Document: "doc", HeadingID: rootHeading}, {Document: "doc", HeadingID: rootHeading}}}},
		{name: "ignored relationship carries target", resolver: func(marksplice.DocumentKey, marksplice.LinkRelationship) marksplice.WorkspaceResolution {
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionIgnore, Target: "ignored"}
		}},
		{name: "missing empty target", resolver: func(marksplice.DocumentKey, marksplice.LinkRelationship) marksplice.WorkspaceResolution {
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionMissing}
		}},
		{name: "missing target is actually present", resolver: func(marksplice.DocumentKey, marksplice.LinkRelationship) marksplice.WorkspaceResolution {
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionMissing, Target: "doc"}
		}},
		{name: "resolved empty target", resolver: func(marksplice.DocumentKey, marksplice.LinkRelationship) marksplice.WorkspaceResolution {
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionResolved}
		}},
		{name: "resolved unknown target", resolver: func(marksplice.DocumentKey, marksplice.LinkRelationship) marksplice.WorkspaceResolution {
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionResolved, Target: "missing"}
		}},
		{name: "invalid resolution kind", resolver: func(marksplice.DocumentKey, marksplice.LinkRelationship) marksplice.WorkspaceResolution {
			return marksplice.WorkspaceResolution{Kind: 99}
		}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := marksplice.ValidateWorkspace(source, test.resolver, test.options); !errors.Is(err, marksplice.ErrInvalidWorkspace) {
				t.Fatalf("ValidateWorkspace() error = %v, want ErrInvalidWorkspace", err)
			}
		})
	}

	zero := &marksplice.Document{}
	invalidDocuments := []struct {
		name      string
		documents []marksplice.GraphDocument
	}{
		{name: "empty document key", documents: []marksplice.GraphDocument{{Document: doc}}},
		{name: "nil document", documents: []marksplice.GraphDocument{{Key: "doc"}}},
		{name: "zero document", documents: []marksplice.GraphDocument{{Key: "doc", Document: zero}}},
		{name: "duplicate document key", documents: []marksplice.GraphDocument{{Key: "doc", Document: doc}, {Key: "doc", Document: foreignDoc}}},
	}
	for _, test := range invalidDocuments {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := marksplice.ValidateWorkspace(test.documents, nil, marksplice.WorkspaceValidationOptions{}); !errors.Is(err, marksplice.ErrInvalidWorkspace) {
				t.Fatalf("ValidateWorkspace() error = %v, want ErrInvalidWorkspace", err)
			}
		})
	}
}

func TestM101ValidateWorkspaceEmptyAndNilReportBehavior(t *testing.T) {
	t.Parallel()

	report, err := marksplice.ValidateWorkspace(nil, nil, marksplice.WorkspaceValidationOptions{})
	if err != nil {
		t.Fatalf("ValidateWorkspace(empty) error = %v", err)
	}
	if graph := report.Graph(); graph == nil || len(graph.DocumentKeys()) != 0 {
		t.Fatalf("empty Graph() = %#v", graph)
	}
	if report.Diagnostics() != nil || report.RepairPlan().Repairs() != nil {
		t.Fatalf("empty report contains results: diagnostics=%#v repairs=%#v", report.Diagnostics(), report.RepairPlan().Repairs())
	}

	var zero marksplice.WorkspaceReport
	if zero.Graph() != nil || zero.Diagnostics() != nil || zero.RepairPlan().Repairs() != nil {
		t.Fatal("zero WorkspaceReport returned data")
	}
}
