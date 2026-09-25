package rendermarkdown

import (
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

type countingCanonicalInlineReferenceLabels struct {
	calls int
}

func (labels *countingCanonicalInlineReferenceLabels) ReferenceLabelKey(value string) string {
	labels.calls++
	return value
}

func TestCanonicalInlineRenderHostUsesCanonicalBottomUpChoice(t *testing.T) {
	t.Parallel()

	ast, err := buildCanonicalInlineAST([]parser.SemanticEvent{
		{Kind: parser.SemanticEmphasis, Phase: parser.SemanticEnter},
		{Kind: parser.SemanticStrong, Phase: parser.SemanticEnter},
		{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "x"},
		{Kind: parser.SemanticStrong, Phase: parser.SemanticExit},
		{Kind: parser.SemanticEmphasis, Phase: parser.SemanticExit},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := canonicalInlineRenderHost(ast, canonicalInlineEmitContext{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("canonical inline host was not representable")
	}
	if want := "*__x__*"; string(got) != want {
		t.Fatalf("canonical inline = %q, want %q", got, want)
	}
}

func TestCanonicalInlineRenderHostUsesTableCellContext(t *testing.T) {
	t.Parallel()

	ast, err := buildCanonicalInlineAST([]parser.SemanticEvent{{
		Kind: parser.SemanticCodeSpan, Phase: parser.SemanticLeaf, Value: "a|b",
	}})
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := canonicalInlineRenderHost(ast, canonicalInlineEmitContext{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("table-cell inline host was not representable")
	}
	if want := renderCodeSpan("a|b", true); string(got) != want {
		t.Fatalf("canonical table-cell inline = %q, want %q", got, want)
	}
}

func TestCanonicalInlineRenderHostThreadsReferenceLabelAuthorityThroughEvaluation(t *testing.T) {
	t.Parallel()

	ast, err := buildCanonicalInlineAST([]parser.SemanticEvent{
		{Kind: parser.SemanticImage, Phase: parser.SemanticEnter, Label: "label"},
		{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "label"},
		{Kind: parser.SemanticImage, Phase: parser.SemanticExit},
	})
	if err != nil {
		t.Fatal(err)
	}
	labels := &countingCanonicalInlineReferenceLabels{}
	got, ok, err := canonicalInlineRenderHost(
		ast,
		canonicalInlineEmitContext{referenceLabels: labels},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("reference-image inline host was not representable")
	}
	if want := "![label]"; string(got) != want {
		t.Fatalf("canonical reference image = %q, want %q", got, want)
	}
	if labels.calls < 4 {
		t.Fatalf("ReferenceLabelKey calls = %d, want evaluation and final emission to share authority", labels.calls)
	}
}

func TestCanonicalInlineRenderHostPreservesExtendedURLBoundary(t *testing.T) {
	t.Parallel()

	const source = "(Visit https://encrypted.google.com/search?q=Markup+(business))\n"
	events := firstParagraphInlineEvents(t, native.New(), []byte(source))
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := canonicalInlineRenderHost(ast, canonicalInlineEmitContext{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("extended URL boundary-safe host rejected")
	}
	const want = "\\(Visit https://encrypted.google.com/search?q=Markup+(business))"
	if string(got) != want {
		t.Fatalf("canonical host = %q, want %q", got, want)
	}
}

func TestCanonicalInlineBottomUpFrontierUsesASTViewsWithoutEventHistory(t *testing.T) {
	t.Parallel()

	ast, err := buildCanonicalInlineAST([]parser.SemanticEvent{
		{Kind: parser.SemanticStrong, Phase: parser.SemanticEnter},
		{Kind: parser.SemanticEmphasis, Phase: parser.SemanticEnter},
		{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "x"},
		{Kind: parser.SemanticEmphasis, Phase: parser.SemanticExit},
		{Kind: parser.SemanticStrong, Phase: parser.SemanticExit},
	})
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}

	pruneCalls := 0
	prune := func(
		view canonicalInlineAST,
		candidates []canonicalInlineFrontierCandidate,
		needsTilde bool,
	) ([]canonicalInlineFrontierCandidate, error) {
		pruneCalls++
		if needsTilde {
			t.Fatal("unexpected tilde requirement")
		}
		if len(view.events) != len(ast.events) || &view.events[0] != &ast.events[0] {
			t.Fatal("frontier view copied semantic-event history")
		}
		return candidates, nil
	}

	frontier, supported, err := canonicalInlineBottomUpHostWithNormalization(
		ast, normalization, false, prune,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !supported {
		t.Fatal("supported Emphasis/Strong tree rejected")
	}
	if pruneCalls != 5 {
		t.Fatalf("prune calls = %d, want 5", pruneCalls)
	}
	if len(frontier) != 4 {
		t.Fatalf("frontier size = %d, want 4", len(frontier))
	}
	got := [][]byte{
		frontier[0].markers,
		frontier[1].markers,
		frontier[2].markers,
		frontier[3].markers,
	}
	want := [][]byte{{'*', '*'}, {'_', '*'}, {'*', '_'}, {'_', '_'}}
	for index := range want {
		if len(got[index]) != len(want[index]) {
			t.Fatalf("markers[%d] = %q, want %q", index, got[index], want[index])
		}
		for bit := range want[index] {
			if got[index][bit] != want[index][bit] {
				t.Fatalf("markers[%d] = %q, want %q", index, got[index], want[index])
			}
		}
	}
}

func TestCanonicalInlineBottomUpHostEvaluatesRawTabOnASTViews(t *testing.T) {
	t.Parallel()

	ast, err := buildCanonicalInlineAST([]parser.SemanticEvent{
		{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "a\t"},
		{Kind: parser.SemanticEmphasis, Phase: parser.SemanticEnter},
		{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "x"},
		{Kind: parser.SemanticEmphasis, Phase: parser.SemanticExit},
	})
	if err != nil {
		t.Fatal(err)
	}
	frontier, supported, err := canonicalInlineBottomUpHost(ast)
	if err != nil {
		t.Fatal(err)
	}
	if !supported || len(frontier) == 0 {
		t.Fatalf("supported=%t frontier=%d", supported, len(frontier))
	}
	for index, candidate := range frontier {
		if len(candidate.payloadRawTab) != 2 || !candidate.payloadRawTab[0] {
			t.Fatalf("candidate %d raw-tab capability = %v", index, candidate.payloadRawTab)
		}
		if len(candidate.payloadKnown) != 2 || !candidate.payloadKnown[0] {
			t.Fatalf("candidate %d payload-known = %v", index, candidate.payloadKnown)
		}
	}
}

func TestCanonicalInlineASTViewPreservesSelectedForestOnly(t *testing.T) {
	t.Parallel()

	ast, err := buildCanonicalInlineAST([]parser.SemanticEvent{
		{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "a"},
		{Kind: parser.SemanticEmphasis, Phase: parser.SemanticEnter},
		{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "b"},
		{Kind: parser.SemanticEmphasis, Phase: parser.SemanticExit},
		{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "c"},
	})
	if err != nil {
		t.Fatal(err)
	}
	first := ast.nodes[ast.root].firstChild
	second := ast.nodes[first].nextSibling
	view, err := canonicalInlineASTView(ast, []int{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.events) != len(ast.events) || &view.events[0] != &ast.events[0] {
		t.Fatal("AST view must share immutable event storage")
	}
	if len(view.nodes) != 4 {
		t.Fatalf("view nodes = %d, want synthetic root + text + emphasis + child", len(view.nodes))
	}
	if view.nodes[view.root].firstChild != 1 || view.nodes[1].nextSibling != 2 ||
		view.nodes[2].nextSibling != noCanonicalInlineASTNode {
		t.Fatalf("view sibling topology = %+v", view.nodes)
	}
}
