package rendermarkdown

import (
	"reflect"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

func TestCanonicalInlineDeferredRecoverableUsesNativeBoundedCompletion(t *testing.T) {
	t.Parallel()

	source := []byte("*\\!*\\!*\\)*")
	pairs := []native.DelimiterTopologyPair{
		{
			Marker:  '*',
			Opening: parser.Range{Start: 0, End: 1},
			Closing: parser.Range{Start: 9, End: 10},
		},
		{
			Marker:  '*',
			Opening: parser.Range{Start: 3, End: 4},
			Closing: parser.Range{Start: 6, End: 7},
		},
	}
	if native.DelimiterTopologyMatches(source, nil, pairs) {
		t.Fatal("precondition: deferred candidate unexpectedly valid in isolation")
	}
	if !canonicalInlineDeferredRecoverable(source, nil, pairs) {
		t.Fatal("Native-proven bounded completion was not retained")
	}
}

func TestCanonicalInlineDeferredRecoverableRejectsImpossibleTopology(t *testing.T) {
	t.Parallel()

	source := []byte("*x*")
	impossiblePairs := []native.DelimiterTopologyPair{{
		Marker:  '_',
		Opening: parser.Range{Start: 0, End: 1},
		Closing: parser.Range{Start: 2, End: 3},
	}}
	if canonicalInlineDeferredRecoverable(source, nil, impossiblePairs) {
		t.Fatal("bounded context incorrectly recovered a marker-mismatched topology")
	}
	if canonicalInlineDeferredRecoverable(source, nil, nil) {
		t.Fatal("empty topology must not be recoverable")
	}
}

func TestCanonicalInlineChoiceBetterUsesCanonicalCostTuple(t *testing.T) {
	t.Parallel()

	preferred := canonicalInlinePreferredPayload
	rawTab := canonicalInlineRawTabPayload
	tests := []struct {
		name  string
		left  canonicalInlineChoiceCost
		right canonicalInlineChoiceCost
		want  bool
	}{
		{
			name: "alternation cost first",
			left: canonicalInlineChoiceCost{
				alternation: 0, strikeSingle: 1,
				payloads: []canonicalInlinePayloadChoice{rawTab}, markers: []byte{'_'},
			},
			right: canonicalInlineChoiceCost{
				alternation: 1, strikeSingle: 0,
				payloads: []canonicalInlinePayloadChoice{preferred}, markers: []byte{'*'},
			},
			want: true,
		},
		{
			name: "double strike before single strike",
			left: canonicalInlineChoiceCost{
				alternation: 1, strikeSingle: 0,
				payloads: []canonicalInlinePayloadChoice{rawTab}, markers: []byte{'_'},
			},
			right: canonicalInlineChoiceCost{
				alternation: 1, strikeSingle: 1,
				payloads: []canonicalInlinePayloadChoice{preferred}, markers: []byte{'*'},
			},
			want: true,
		},
		{
			name: "preferred payload before marker orientation",
			left: canonicalInlineChoiceCost{
				payloads: []canonicalInlinePayloadChoice{preferred}, markers: []byte{'_'},
			},
			right: canonicalInlineChoiceCost{
				payloads: []canonicalInlinePayloadChoice{rawTab}, markers: []byte{'*'},
			},
			want: true,
		},
		{
			name: "earliest payload difference wins",
			left: canonicalInlineChoiceCost{
				payloads: []canonicalInlinePayloadChoice{preferred, rawTab},
			},
			right: canonicalInlineChoiceCost{
				payloads: []canonicalInlinePayloadChoice{rawTab, preferred},
			},
			want: true,
		},
		{
			name: "asterisk first in AST preorder",
			left: canonicalInlineChoiceCost{
				markers: []byte{'_', '*', '*'},
			},
			right: canonicalInlineChoiceCost{
				markers: []byte{'_', '_', '*'},
			},
			want: true,
		},
		{
			name:  "equal tuple is not better",
			left:  canonicalInlineChoiceCost{payloads: []canonicalInlinePayloadChoice{preferred}, markers: []byte{'*'}},
			right: canonicalInlineChoiceCost{payloads: []canonicalInlinePayloadChoice{preferred}, markers: []byte{'*'}},
			want:  false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := canonicalInlineChoiceBetter(tc.left, tc.right); got != tc.want {
				t.Fatalf("canonicalInlineChoiceBetter() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestCanonicalInlinePruneSelectionPreservesCollapseOrder(t *testing.T) {
	t.Parallel()

	keyA := canonicalInlinePruneKey{
		state: canonicalInlineSelectorState{
			base: canonicalInlineTransferState{starForbidden: 0x01},
		},
	}
	keyB := canonicalInlinePruneKey{
		state: canonicalInlineSelectorState{
			base: canonicalInlineTransferState{underscoreForbidden: 0x02},
		},
	}
	observations := []canonicalInlinePruneObservation{
		{disposition: canonicalInlinePruneKeyed, key: keyA, cost: canonicalInlineChoiceCost{alternation: 2}},
		{disposition: canonicalInlinePruneDeferred},
		{disposition: canonicalInlinePruneMarkerless, cost: canonicalInlineChoiceCost{strikeSingle: 1}},
		{disposition: canonicalInlinePruneKeyed, key: keyB, cost: canonicalInlineChoiceCost{alternation: 0}},
		{disposition: canonicalInlinePruneMarkerless, cost: canonicalInlineChoiceCost{strikeSingle: 0}},
		{disposition: canonicalInlinePruneKeyed, key: keyA, cost: canonicalInlineChoiceCost{alternation: 1}},
		{disposition: canonicalInlinePruneDeferred},
		{disposition: canonicalInlinePruneKeyed, key: keyB, cost: canonicalInlineChoiceCost{alternation: 0}},
	}
	got := canonicalInlinePruneSelection(observations)
	want := []int{4, 1, 6, 5, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selection = %v, want %v", got, want)
	}
}

func TestCanonicalInlinePruneAccumulatorMatchesSelectionOrder(t *testing.T) {
	t.Parallel()

	keyA := canonicalInlinePruneKey{
		state: canonicalInlineSelectorState{
			base: canonicalInlineTransferState{starForbidden: 0x01},
		},
	}
	keyB := canonicalInlinePruneKey{
		state: canonicalInlineSelectorState{
			base: canonicalInlineTransferState{underscoreForbidden: 0x02},
		},
	}
	observations := []canonicalInlinePruneObservation{
		{disposition: canonicalInlinePruneKeyed, key: keyA, cost: canonicalInlineChoiceCost{alternation: 2}},
		{disposition: canonicalInlinePruneDeferred},
		{disposition: canonicalInlinePruneMarkerless, cost: canonicalInlineChoiceCost{strikeSingle: 1}},
		{disposition: canonicalInlinePruneKeyed, key: keyB, cost: canonicalInlineChoiceCost{alternation: 0}},
		{disposition: canonicalInlinePruneMarkerless, cost: canonicalInlineChoiceCost{strikeSingle: 0}},
		{disposition: canonicalInlinePruneKeyed, key: keyA, cost: canonicalInlineChoiceCost{alternation: 1}},
		{disposition: canonicalInlinePruneDeferred},
		{disposition: canonicalInlinePruneKeyed, key: keyB, cost: canonicalInlineChoiceCost{alternation: 0}},
	}
	candidates := make([]canonicalInlineFrontierCandidate, len(observations))
	for index, observation := range observations {
		candidates[index].alternationCost = observation.cost.alternation
		candidates[index].strikeSingleCost = observation.cost.strikeSingle
		candidates[index].payloadRawTab = make([]bool, index+1)
	}

	accumulator := canonicalInlinePruneAccumulator{}
	for index, observation := range observations {
		accumulator.observe(candidates[index], observation.disposition, observation.key)
	}
	got := accumulator.result()
	selected := canonicalInlinePruneSelection(observations)
	if len(got) != len(selected) {
		t.Fatalf("accumulator result length = %d, selection length = %d", len(got), len(selected))
	}
	for index, selectedIndex := range selected {
		if len(got[index].payloadRawTab) != len(candidates[selectedIndex].payloadRawTab) {
			t.Fatalf(
				"accumulator result[%d] identity = %d, want candidate %d identity = %d",
				index,
				len(got[index].payloadRawTab),
				selectedIndex,
				len(candidates[selectedIndex].payloadRawTab),
			)
		}
	}
}

func TestCanonicalInlineAlternationCostCountsTouchingSameFamilyParents(t *testing.T) {
	t.Parallel()

	buildCandidate := func(events []parser.SemanticEvent, markers ...byte) (canonicalInlineAST, canonicalInlineCandidate) {
		t.Helper()
		ast, err := buildCanonicalInlineAST(events)
		if err != nil {
			t.Fatal(err)
		}
		normalization, err := normalizeCanonicalInlineAST(ast)
		if err != nil {
			t.Fatal(err)
		}
		plan := newCanonicalInlinePlan(ast)
		markerIndex := 0
		for nodeIndex := 1; nodeIndex < len(ast.nodes); nodeIndex++ {
			event := ast.events[ast.nodes[nodeIndex].eventIndex]
			if event.Phase == parser.SemanticEnter &&
				(event.Kind == parser.SemanticEmphasis || event.Kind == parser.SemanticStrong) {
				plan.nodes[nodeIndex].marker = markers[markerIndex]
				markerIndex++
			}
		}
		candidate, err := canonicalInlineCandidateForPlan(
			ast, normalization, plan, canonicalInlineEmitContext{}, false,
		)
		if err != nil {
			t.Fatal(err)
		}
		return ast, candidate
	}

	touching := []parser.SemanticEvent{
		{Kind: parser.SemanticEmphasis, Phase: parser.SemanticEnter},
		{Kind: parser.SemanticStrong, Phase: parser.SemanticEnter},
		{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "x"},
		{Kind: parser.SemanticStrong, Phase: parser.SemanticExit},
		{Kind: parser.SemanticEmphasis, Phase: parser.SemanticExit},
	}
	ast, candidate := buildCandidate(touching, '*', '*')
	if candidate.alternationCost != 1 {
		t.Fatalf("candidate same-family alternation cost = %d, want 1", candidate.alternationCost)
	}
	if cost, ok := canonicalInlineAlternationCost(ast, candidate.delimiterSites); !ok || cost != 1 {
		t.Fatalf("touching same-family cost = %d, ok=%t; want 1, true", cost, ok)
	}
	ast, candidate = buildCandidate(touching, '*', '_')
	if candidate.alternationCost != 0 {
		t.Fatalf("candidate alternating cost = %d, want 0", candidate.alternationCost)
	}
	if cost, ok := canonicalInlineAlternationCost(ast, candidate.delimiterSites); !ok || cost != 0 {
		t.Fatalf("touching alternating cost = %d, ok=%t; want 0, true", cost, ok)
	}

	separated := []parser.SemanticEvent{
		{Kind: parser.SemanticEmphasis, Phase: parser.SemanticEnter},
		{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "a"},
		{Kind: parser.SemanticStrong, Phase: parser.SemanticEnter},
		{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "x"},
		{Kind: parser.SemanticStrong, Phase: parser.SemanticExit},
		{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "b"},
		{Kind: parser.SemanticEmphasis, Phase: parser.SemanticExit},
	}
	ast, candidate = buildCandidate(separated, '*', '*')
	if cost, ok := canonicalInlineAlternationCost(ast, candidate.delimiterSites); !ok || cost != 0 {
		t.Fatalf("separated same-family cost = %d, ok=%t; want 0, true", cost, ok)
	}
	if _, ok := canonicalInlineAlternationCost(ast, nil); ok {
		t.Fatal("missing delimiter sites unexpectedly produced an alternation cost")
	}
}

func TestCanonicalInlineCandidateCarriesNativeTopologyProof(t *testing.T) {
	t.Parallel()

	ast, err := buildCanonicalInlineAST([]parser.SemanticEvent{
		{Kind: parser.SemanticEmphasis, Phase: parser.SemanticEnter},
		{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "x"},
		{Kind: parser.SemanticEmphasis, Phase: parser.SemanticExit},
	})
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	plan := newCanonicalInlinePlan(ast)
	plan.nodes[1].marker = '*'

	candidate, err := canonicalInlineCandidateForPlan(
		ast,
		normalization,
		plan,
		canonicalInlineEmitContext{},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(candidate.output); got != "*x*" {
		t.Fatalf("output = %q, want %q", got, "*x*")
	}
	if len(candidate.delimiterSites) != 2 {
		t.Fatalf("delimiter sites = %d, want 2", len(candidate.delimiterSites))
	}
	if len(candidate.owners) != 0 {
		t.Fatalf("owners = %v, want none", candidate.owners)
	}
	if len(candidate.pairs) != 1 {
		t.Fatalf("topology pairs = %v, want 1 pair", candidate.pairs)
	}
	pair := candidate.pairs[0]
	if pair.Marker != '*' ||
		pair.Opening != (parser.Range{Start: 0, End: 1}) ||
		pair.Closing != (parser.Range{Start: 2, End: 3}) {
		t.Fatalf("topology pair = %+v", pair)
	}
	if !native.DelimiterTopologyMatches(candidate.output, candidate.owners, candidate.pairs) {
		t.Fatal("candidate metadata does not reproduce Native topology")
	}
}
