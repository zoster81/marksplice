package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM104FootnotesExposeDefinitionsReferencesAndExactSourceOwnership(t *testing.T) {
	t.Parallel()

	source := []byte("before[^b] and again[^a] and again[^a]\n\n[^a]: first line\n\n    second paragraph\n\n[^unused]: unused\n[^b]: bee\n")
	doc := mustParseM104Document(t, source)
	definitions := doc.FootnoteDefinitions()
	if len(definitions) != 3 {
		t.Fatalf("len(FootnoteDefinitions()) = %d, want 3", len(definitions))
	}
	if definitions[0].Label() != "a" || definitions[1].Label() != "unused" || definitions[2].Label() != "b" {
		t.Fatalf("definition labels = %q/%q/%q", definitions[0].Label(), definitions[1].Label(), definitions[2].Label())
	}
	for _, definition := range definitions {
		if definition.ID().String() == "" {
			t.Fatal("definition has empty NodeID")
		}
		if node, ok := doc.Node(definition.ID()); !ok || node.Kind() != marksplice.KindFootnoteDefinition {
			t.Fatalf("Node(definition) = %+v/%v", node, ok)
		}
		if got, ok := doc.FootnoteDefinition(definition.ID()); !ok || got.ID() != definition.ID() {
			t.Fatalf("FootnoteDefinition(ID) = %+v/%v", got, ok)
		}
	}

	a := definitions[0]
	assertM104RangeSource(t, doc, a.LabelRange(), "a")
	assertM104RangeSource(t, doc, a.Range(), "[^a]: first line\n\n    second paragraph\n\n")
	if _, ok := a.BodyRange(); ok {
		t.Fatal("multiline BodyRange() ok = true")
	}
	aBody, ok := doc.FootnoteDefinitionBodyRanges(a.ID())
	if !ok || len(aBody) != 2 {
		t.Fatalf("FootnoteDefinitionBodyRanges(a) = %v/%v", aBody, ok)
	}
	assertM104RangeSource(t, doc, aBody[0], "first line")
	assertM104RangeSource(t, doc, aBody[1], "second paragraph")

	b := definitions[2]
	body, ok := b.BodyRange()
	if !ok {
		t.Fatal("simple BodyRange() ok = false")
	}
	assertM104RangeSource(t, doc, body, "bee")

	references := doc.FootnoteReferences()
	if len(references) != 3 {
		t.Fatalf("len(FootnoteReferences()) = %d, want 3", len(references))
	}
	wantDefinitions := []marksplice.NodeID{b.ID(), a.ID(), a.ID()}
	wantOccurrences := []int{0, 0, 1}
	for index, reference := range references {
		definitionID, ok := reference.DefinitionID()
		if !ok || definitionID != wantDefinitions[index] {
			t.Fatalf("DefinitionID[%d] = %v/%v, want %v", index, definitionID, ok, wantDefinitions[index])
		}
		if reference.Occurrence() != wantOccurrences[index] {
			t.Fatalf("Occurrence[%d] = %d, want %d", index, reference.Occurrence(), wantOccurrences[index])
		}
		assertM104RangeSource(t, doc, reference.Range(), "[^"+reference.Label()+"]")
		assertM104RangeSource(t, doc, reference.LabelRange(), reference.Label())
	}

	if links := doc.LinkRelationships(); len(links) != 0 {
		t.Fatalf("footnote syntax leaked into LinkRelationships(): %+v", links)
	}
	copyRefs := doc.FootnoteReferences()
	copyRefs[0] = marksplice.FootnoteReference{}
	if again := doc.FootnoteReferences(); len(again) != 3 || again[0].Label() != "b" {
		t.Fatal("caller mutation leaked into FootnoteReferences")
	}
}

func TestM104FootnoteDefinitionsRemainTopLevelAndUnusedDefinitionsAreReadable(t *testing.T) {
	t.Parallel()

	source := []byte("[^unused]: value\n")
	doc := mustParseM104Document(t, source)
	definitions := doc.FootnoteDefinitions()
	if len(definitions) != 1 || definitions[0].Label() != "unused" {
		t.Fatalf("FootnoteDefinitions() = %+v", definitions)
	}
	if refs := doc.FootnoteReferences(); len(refs) != 0 {
		t.Fatalf("FootnoteReferences() = %+v, want none", refs)
	}
	if nodes := doc.Nodes(); len(nodes) != 1 || nodes[0].Kind() != marksplice.KindFootnoteDefinition {
		t.Fatalf("Nodes() = %+v, want one footnote definition", nodes)
	}
}

func TestM104FootnoteBodyReplacementAndRenameAreSourcePreserving(t *testing.T) {
	t.Parallel()

	source := []byte("see[^old] and again[^old] [target](#target)\n\n# Target\n\n[^old]: old body\n")
	doc := mustParseM104Document(t, source)
	definition := doc.FootnoteDefinitions()[0]

	bodyChange, err := doc.PrepareReplaceFootnoteDefinitionBody(definition.ID(), []byte("new body"))
	if err != nil {
		t.Fatalf("PrepareReplaceFootnoteDefinitionBody() error = %v", err)
	}
	bodySource, err := bodyChange.Apply(source)
	if err != nil {
		t.Fatalf("body Apply() error = %v", err)
	}
	wantBody := []byte("see[^old] and again[^old] [target](#target)\n\n# Target\n\n[^old]: new body\n")
	if !bytes.Equal(bodySource, wantBody) {
		t.Fatalf("body Apply() = %q, want %q", bodySource, wantBody)
	}
	bodyDoc := mustParseM104Document(t, bodySource)
	if refs := bodyDoc.FootnoteReferences(); len(refs) != 2 || refs[0].Occurrence() != 0 || refs[1].Occurrence() != 1 {
		t.Fatalf("body references = %+v", refs)
	}
	if links := bodyDoc.LinkRelationships(); len(links) != 1 || links[0].Destination() != "#target" {
		t.Fatalf("body link relationships = %+v", links)
	}

	renameChange, err := doc.PrepareRenameFootnote(definition.ID(), []byte("new"))
	if err != nil {
		t.Fatalf("PrepareRenameFootnote() error = %v", err)
	}
	renamed, err := renameChange.Apply(source)
	if err != nil {
		t.Fatalf("rename Apply() error = %v", err)
	}
	wantRename := []byte("see[^new] and again[^new] [target](#target)\n\n# Target\n\n[^new]: old body\n")
	if !bytes.Equal(renamed, wantRename) {
		t.Fatalf("rename Apply() = %q, want %q", renamed, wantRename)
	}
	renamedDoc := mustParseM104Document(t, renamed)
	if definitions := renamedDoc.FootnoteDefinitions(); len(definitions) != 1 || definitions[0].Label() != "new" {
		t.Fatalf("renamed definitions = %+v", definitions)
	}
	if refs := renamedDoc.FootnoteReferences(); len(refs) != 2 || refs[0].Label() != "new" || refs[1].Label() != "new" || refs[1].Occurrence() != 1 {
		t.Fatalf("renamed references = %+v", refs)
	}
	if links := renamedDoc.LinkRelationships(); len(links) != 1 || links[0].Destination() != "#target" {
		t.Fatalf("renamed link relationships = %+v", links)
	}
}

func TestM104FootnoteMutationsFailClosedOnUnsupportedBodyAndLabelCollision(t *testing.T) {
	t.Parallel()

	source := []byte("one[^one] two[^two]\n\n[^one]: first\n[^two]: second\n")
	doc := mustParseM104Document(t, source)
	definitions := doc.FootnoteDefinitions()
	if len(definitions) != 2 {
		t.Fatalf("len(FootnoteDefinitions()) = %d, want 2", len(definitions))
	}
	if _, err := doc.PrepareRenameFootnote(definitions[0].ID(), []byte("two")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareRenameFootnote(collision) error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareRenameFootnote(definitions[0].ID(), []byte("bad]label")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareRenameFootnote(invalid) error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareReplaceFootnoteDefinitionBody(definitions[0].ID(), []byte("line one\nline two")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceFootnoteDefinitionBody(multiline) error = %v, want ErrInvalidReplacement", err)
	}

	multi := mustParseM104Document(t, []byte("[^n]: first\n\n    second\n"))
	if _, err := multi.PrepareReplaceFootnoteDefinitionBody(multi.FootnoteDefinitions()[0].ID(), []byte("replacement")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceFootnoteDefinitionBody(multisegment) error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestM104FootnoteConstructionSupportsDeferredDefinitionsAndTypedReferences(t *testing.T) {
	t.Parallel()

	builder := marksplice.NewDocumentBuilder()
	if err := builder.DeferFootnoteDefinition("note", "footnote body"); err != nil {
		t.Fatalf("DeferFootnoteDefinition() error = %v", err)
	}
	if err := builder.AppendParagraphContent(marksplice.TextInline("See note"), marksplice.FootnoteReferenceInline("note")); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}
	markdown, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("See note[^note]\n\n[^note]: footnote body\n")
	if !bytes.Equal(markdown, want) {
		t.Fatalf("Markdown() = %q, want %q", markdown, want)
	}
	doc := mustParseM104Document(t, markdown)
	if defs := doc.FootnoteDefinitions(); len(defs) != 1 || defs[0].Label() != "note" {
		t.Fatalf("constructed definitions = %+v", defs)
	}
	if refs := doc.FootnoteReferences(); len(refs) != 1 || refs[0].Label() != "note" || refs[0].Occurrence() != 0 {
		t.Fatalf("constructed references = %+v", refs)
	}

	missing := marksplice.NewDocumentBuilder()
	if err := missing.AppendParagraphContent(marksplice.FootnoteReferenceInline("missing")); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("unresolved FootnoteReferenceInline error = %v, want ErrInvalidConstruction", err)
	}
}

func TestM104LinksInsideFootnotesParticipateInDocumentGraph(t *testing.T) {
	t.Parallel()

	sourceDoc := mustParseM104Document(t, []byte("note[^n]\n\n[^n]: [target](b.md#part)\n"))
	targetDoc := mustParseM104Document(t, []byte("# Part\n"))
	links := sourceDoc.LinkRelationships()
	if len(links) != 1 || links[0].Destination() != "b.md#part" {
		t.Fatalf("footnote LinkRelationships() = %+v", links)
	}
	graph, err := marksplice.BuildDocumentGraph([]marksplice.GraphDocument{
		{Key: "a", Document: sourceDoc},
		{Key: "b", Document: targetDoc},
	}, func(source marksplice.DocumentKey, relationship marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
		if source == "a" && relationship.Destination() == "b.md#part" {
			return marksplice.DocumentResolution{Target: "b", Fragment: "#part"}, true
		}
		return marksplice.DocumentResolution{}, false
	})
	if err != nil {
		t.Fatalf("BuildDocumentGraph() error = %v", err)
	}
	edges := graph.Edges()
	if len(edges) != 1 || edges[0].SourceDocument() != "a" || edges[0].TargetDocument() != "b" {
		t.Fatalf("graph edges = %+v", edges)
	}
	if target, ok := edges[0].FragmentTarget(); !ok || target.Value() != "part" {
		t.Fatalf("footnote graph FragmentTarget() = %+v/%v", target, ok)
	}
}

func TestM104FootnoteSyntaxSupersedesCaretReferenceDefinitions(t *testing.T) {
	t.Parallel()

	doc := mustParseM104Document(t, []byte("use[^n]\n\n[^n]: /legacy-reference-destination\n"))
	if defs := doc.FootnoteDefinitions(); len(defs) != 1 || defs[0].Label() != "n" {
		t.Fatalf("FootnoteDefinitions() = %+v", defs)
	}
	if links := doc.LinkRelationships(); len(links) != 0 {
		t.Fatalf("caret reference leaked into LinkRelationships() = %+v", links)
	}

	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendReferenceDefinition("^n", "/legacy-reference-destination"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendReferenceDefinition(^n) error = %v, want ErrInvalidConstruction", err)
	}
}

func TestM104FootnoteLabelsResolveExactly(t *testing.T) {
	t.Parallel()

	doc := mustParseM104Document(t, []byte("use[^Note]\n\n[^note]: body\n"))
	definitions := doc.FootnoteDefinitions()
	if len(definitions) != 1 || definitions[0].Label() != "note" {
		t.Fatalf("FootnoteDefinitions() = %+v, want exact lowercase definition", definitions)
	}
	if references := doc.FootnoteReferences(); len(references) != 0 {
		t.Fatalf("FootnoteReferences() = %+v, want no case-folded match", references)
	}

	builder := marksplice.NewDocumentBuilder()
	if err := builder.DeferFootnoteDefinition("note", "body"); err != nil {
		t.Fatalf("DeferFootnoteDefinition() error = %v", err)
	}
	if err := builder.AppendParagraphContent(marksplice.FootnoteReferenceInline("Note")); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("FootnoteReferenceInline(case mismatch) error = %v, want ErrInvalidConstruction", err)
	}
}

func TestM104FootnoteDefinitionsIntegrateWithStructuralQueries(t *testing.T) {
	t.Parallel()

	doc := mustParseM104Document(t, []byte("paragraph\n\n[^note]: body\n"))
	definitions := doc.FootnoteDefinitions()
	if len(definitions) != 1 {
		t.Fatalf("len(FootnoteDefinitions()) = %d, want 1", len(definitions))
	}
	matches, err := doc.QueryNodes(marksplice.NodeQuery{Kinds: []marksplice.Kind{marksplice.KindFootnoteDefinition}, Limit: 4})
	if err != nil {
		t.Fatalf("QueryNodes() error = %v", err)
	}
	if len(matches) != 1 || matches[0].Node().ID() != definitions[0].ID() || matches[0].Range() != definitions[0].Range() {
		t.Fatalf("QueryNodes() = %+v, want exact footnote definition", matches)
	}
}

func TestM104FootnoteChangesComposeAcrossIndependentDefinitions(t *testing.T) {
	t.Parallel()

	source := []byte("a[^a] b[^b]\n\n[^a]: one\n\n[^b]: two\n")
	doc := mustParseM104Document(t, source)
	definitions := doc.FootnoteDefinitions()
	if len(definitions) != 2 {
		t.Fatalf("len(FootnoteDefinitions()) = %d, want 2", len(definitions))
	}
	byLabel := make(map[string]marksplice.FootnoteDefinition, len(definitions))
	for _, definition := range definitions {
		byLabel[definition.Label()] = definition
	}
	body, err := doc.PrepareReplaceFootnoteDefinitionBody(byLabel["a"].ID(), []byte("ONE"))
	if err != nil {
		t.Fatalf("PrepareReplaceFootnoteDefinitionBody(a) error = %v", err)
	}
	rename, err := doc.PrepareRenameFootnote(byLabel["b"].ID(), []byte("bee"))
	if err != nil {
		t.Fatalf("PrepareRenameFootnote(b) error = %v", err)
	}
	combined, err := doc.ComposeChanges(body, rename)
	if err != nil {
		t.Fatalf("ComposeChanges() error = %v", err)
	}
	got, err := combined.Apply(source)
	if err != nil {
		t.Fatalf("combined Apply() error = %v", err)
	}
	want := []byte("a[^a] b[^bee]\n\n[^a]: ONE\n\n[^bee]: two\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("combined Apply() = %q, want %q", got, want)
	}
	updated := mustParseM104Document(t, got)
	refs := updated.FootnoteReferences()
	if len(refs) != 2 || refs[0].Label() != "a" || refs[1].Label() != "bee" {
		t.Fatalf("combined FootnoteReferences() = %+v", refs)
	}
}

func TestM104DeferredFootnotesAreRejectedFromQuotedChildBuilders(t *testing.T) {
	t.Parallel()

	child := marksplice.NewDocumentBuilder()
	if err := child.DeferFootnoteDefinition("n", "body"); err != nil {
		t.Fatalf("DeferFootnoteDefinition() error = %v", err)
	}
	parent := marksplice.NewDocumentBuilder()
	if err := parent.AppendBlockquoteBlocks(1, child); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendBlockquoteBlocks(deferred footnote) error = %v, want ErrInvalidConstruction", err)
	}
	if err := parent.AppendAlertBlocks(marksplice.AlertKindNote, child); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendAlertBlocks(deferred footnote) error = %v, want ErrInvalidConstruction", err)
	}
}

func mustParseM104Document(t *testing.T, source []byte) *marksplice.Document {
	t.Helper()
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return doc
}

func assertM104RangeSource(t *testing.T, doc *marksplice.Document, range_ marksplice.Range, want string) {
	t.Helper()
	got, ok := doc.SourceRange(range_)
	if !ok {
		t.Fatalf("SourceRange(%v) ok = false", range_)
	}
	if !bytes.Equal(got, []byte(want)) {
		t.Fatalf("SourceRange(%v) = %q, want %q", range_, got, want)
	}
}
