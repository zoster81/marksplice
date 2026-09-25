package rendermarkdown

import (
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

func TestCanonicalInlineIncomingOpenerMaskDelegatesToNative(t *testing.T) {
	t.Parallel()

	source := []byte("*x*")
	expected := []native.DelimiterTopologyPair{{
		Marker:  '*',
		Opening: parser.Range{Start: 0, End: 1},
		Closing: parser.Range{Start: 2, End: 3},
	}}
	got, ok := canonicalInlineIncomingOpenerMask(source, nil, expected)
	if !ok || got != 0x3b {
		t.Fatalf("incoming-opener mask = %02x, ok=%t", got, ok)
	}
}

func TestCanonicalInlineTildeOpenerMaskUsesNativeLeadingRunProof(t *testing.T) {
	t.Parallel()

	source := []byte("*x*")
	expected := []native.DelimiterTopologyPair{{
		Marker:  '*',
		Opening: parser.Range{Start: 0, End: 1},
		Closing: parser.Range{Start: 2, End: 3},
	}}
	got, ok := canonicalInlineTildeOpenerMask(source, nil, expected)
	if !ok {
		t.Fatal("tilde opener derivation failed")
	}
	var want uint8
	for _, probe := range []struct {
		category int
		width    int
		canClose bool
	}{
		{category: 1, width: 1},
		{category: 2, width: 2},
		{category: 4, width: 1, canClose: true},
		{category: 5, width: 2, canClose: true},
	} {
		if !native.DelimiterTopologyPreservedWithLeadingRun(
			source, nil, expected, '~', probe.width, probe.canClose,
		) {
			want |= 1 << probe.category
		}
	}
	if got != want {
		t.Fatalf("tilde opener mask = %02x, want %02x", got, want)
	}
}

func TestCanonicalInlineTransferBoundaryStateUsesNativeFacts(t *testing.T) {
	t.Parallel()

	source := []byte("***x***")
	expected := []native.DelimiterTopologyPair{
		{
			Marker:  '*',
			Opening: parser.Range{Start: 0, End: 1},
			Closing: parser.Range{Start: 6, End: 7},
		},
		{
			Marker:  '*',
			Opening: parser.Range{Start: 1, End: 3},
			Closing: parser.Range{Start: 4, End: 6},
		},
	}
	state, ok := canonicalInlineTransferBoundaryState(source, nil, expected)
	if !ok || state.empty || state.singleRun {
		t.Fatalf("boundary state = %+v, ok=%t", state, ok)
	}
	if state.first.marker != '*' || state.first.lengthMod3 != 0 || state.first.lengthBucket != 2 {
		t.Fatalf("first run = %+v", state.first)
	}
	if state.last.marker != '*' || state.last.lengthMod3 != 0 || state.last.lengthBucket != 2 {
		t.Fatalf("last run = %+v", state.last)
	}
	if state.firstDemand.openLevelOne != 1 || state.lastDemand.closeLevelOne != 1 {
		t.Fatalf("demands = %+v/%+v", state.firstDemand, state.lastDemand)
	}
	if !state.firstTouchesLeft || !state.lastTouchesRight {
		t.Fatalf("touches = %t/%t", state.firstTouchesLeft, state.lastTouchesRight)
	}
}

func TestCanonicalInlineBoundaryFactsUseParserClassifications(t *testing.T) {
	t.Parallel()

	output := []byte("a** ")
	run := parser.DelimiterRun{Start: 1, End: 3, Marker: '*'}
	facts := canonicalInlineBoundaryRunFromEmission(output, run)
	if !facts.present || facts.marker != '*' || facts.lengthMod3 != 2 || facts.lengthBucket != 2 {
		t.Fatalf("boundary run facts = %+v", facts)
	}
	if facts.beforeWhitespace || facts.beforePunctuation || !facts.afterWhitespace || facts.afterPunctuation {
		t.Fatalf("boundary run classes = %+v", facts)
	}
	if got := canonicalInlineBoundaryClassFromFlags(true, false); got != canonicalInlineBoundaryWhitespace {
		t.Fatalf("whitespace class = %d", got)
	}
	if got := canonicalInlineBoundaryClassFromFlags(false, true); got != canonicalInlineBoundaryPunctuation {
		t.Fatalf("punctuation class = %d", got)
	}
	if got := canonicalInlineBoundaryClassFromFlags(false, false); got != canonicalInlineBoundaryOther {
		t.Fatalf("other class = %d", got)
	}
	if got := canonicalInlineEdgeClass([]byte("x"), true); got != canonicalInlineBoundaryOther {
		t.Fatalf("left edge class = %d", got)
	}
	if got := canonicalInlineEdgeClass([]byte(" "), false); got != canonicalInlineBoundaryWhitespace {
		t.Fatalf("right edge class = %d", got)
	}
}

func TestCanonicalInlineForeignForbiddenMaskDelegatesToNative(t *testing.T) {
	t.Parallel()

	source := []byte("***x***")
	expected := []native.DelimiterTopologyPair{
		{
			Marker:  '*',
			Opening: parser.Range{Start: 0, End: 1},
			Closing: parser.Range{Start: 6, End: 7},
		},
		{
			Marker:  '*',
			Opening: parser.Range{Start: 1, End: 3},
			Closing: parser.Range{Start: 4, End: 6},
		},
	}
	var want uint8
	for category := 0; category < 6; category++ {
		width := category % 3
		if width == 0 {
			width = 3
		}
		if !native.DelimiterTopologyPreservedWithLeadingRun(
			source, nil, expected, '*', width, category >= 3,
		) {
			want |= 1 << category
		}
	}
	if got := canonicalInlineForeignForbiddenMask(source, nil, expected, '*'); got != want {
		t.Fatalf("forbidden mask = %02x, want %02x", got, want)
	}
}

func TestCanonicalInlinePruneKeyCarriesPendingRawTabResponse(t *testing.T) {
	t.Parallel()

	base := canonicalInlineSelectorState{
		base: canonicalInlineTransferState{underscoreForbidden: 0x02},
	}
	pending := canonicalInlineSelectorState{
		base:        canonicalInlineTransferState{starForbidden: 0x04},
		tildeOpener: 0x01,
	}
	key := canonicalInlinePruneKey{
		state:              base,
		pendingRawTab:      true,
		pendingRawTabState: pending,
	}

	seen := map[canonicalInlinePruneKey]struct{}{key: {}}
	if _, ok := seen[key]; !ok {
		t.Fatal("prune key must remain comparable")
	}
	if !key.pendingRawTab ||
		key.state.base.underscoreForbidden != 0x02 ||
		key.pendingRawTabState.base.starForbidden != 0x04 ||
		key.pendingRawTabState.tildeOpener != 0x01 {
		t.Fatalf("prune key = %+v", key)
	}
}

func TestCanonicalInlinePruneDispositionPreservesBranchOrder(t *testing.T) {
	t.Parallel()

	deferred := canonicalInlineSelectorState{deferredContinuation: true}
	if got := canonicalInlinePruneDispositionForEmitted(false, deferred); got != canonicalInlinePruneMarkerless {
		t.Fatalf("markerless disposition = %d, want %d", got, canonicalInlinePruneMarkerless)
	}
	if got := canonicalInlinePruneDispositionForEmitted(true, deferred); got != canonicalInlinePruneDeferred {
		t.Fatalf("deferred disposition = %d, want %d", got, canonicalInlinePruneDeferred)
	}
	if got := canonicalInlinePruneDispositionForEmitted(true, canonicalInlineSelectorState{}); got != canonicalInlinePruneKeyed {
		t.Fatalf("keyed disposition = %d, want %d", got, canonicalInlinePruneKeyed)
	}
}

func TestCanonicalInlinePruneKeyConstructionKeepsFullPendingResponse(t *testing.T) {
	t.Parallel()

	state := canonicalInlineSelectorState{
		base: canonicalInlineTransferState{underscoreForbidden: 0x02},
	}
	pending := canonicalInlineSelectorState{
		base:        canonicalInlineTransferState{starForbidden: 0x04},
		tildeOpener: 0x01,
	}

	got := canonicalInlinePruneKeyWithPending(state, pending, true)
	want := canonicalInlinePruneKey{
		state:              state,
		pendingRawTab:      true,
		pendingRawTabState: pending,
	}
	if got != want {
		t.Fatalf("ordinary key = %+v, want %+v", got, want)
	}

	pendingOnly, ok := canonicalInlinePendingOnlyPruneKey(pending, true)
	if !ok || pendingOnly != (canonicalInlinePruneKey{
		pendingRawTab:      true,
		pendingRawTabState: pending,
	}) {
		t.Fatalf("pending-only key = %+v, ok=%t", pendingOnly, ok)
	}
	if _, ok := canonicalInlinePendingOnlyPruneKey(pending, false); ok {
		t.Fatal("pending-only key must reject absent pending response")
	}
}

func TestCanonicalInlineSelectorStateIsComparableAndCarriesQualifiedContinuation(t *testing.T) {
	t.Parallel()

	state := canonicalInlineSelectorState{
		base: canonicalInlineTransferState{
			starForbidden: 0x01,
			leftEdge:      canonicalInlineBoundaryOther,
		},
		extendedWWW: canonicalBareWWWContinuation{
			requiredOwnerEnd: 3,
			active:           true,
		},
		tildeOpener:          0x03,
		deferredContinuation: true,
	}

	seen := map[canonicalInlineSelectorState]struct{}{state: {}}
	if _, ok := seen[state]; !ok {
		t.Fatal("selector state must remain a comparable prune key")
	}
	if state.base.starForbidden != 0x01 || state.tildeOpener != 0x03 || !state.deferredContinuation {
		t.Fatalf("selector state = %+v", state)
	}
	if !state.extendedWWW.active || state.extendedWWW.requiredOwnerEnd != 3 {
		t.Fatalf("extended WWW continuation = %+v", state.extendedWWW)
	}
}

func TestCanonicalInlineTransferStateIsComparableAndCarriesBoundaryFacts(t *testing.T) {
	t.Parallel()

	state := canonicalInlineTransferState{
		starForbidden:       0x12,
		underscoreForbidden: 0x24,
		first: canonicalInlineBoundaryRun{
			present: true, marker: '*', lengthMod3: 2, lengthBucket: 2,
			beforeWhitespace: true, afterPunctuation: true,
		},
		last: canonicalInlineBoundaryRun{
			present: true, marker: '_', lengthMod3: 1, lengthBucket: 1,
			beforePunctuation: true, afterWhitespace: true,
		},
		firstDemand:      canonicalInlineBoundaryDemand{openLevelOne: 1},
		lastDemand:       canonicalInlineBoundaryDemand{closeLevelOne: 2},
		incomingOpener:   0x15,
		leftEdge:         canonicalInlineBoundaryWhitespace,
		rightEdge:        canonicalInlineBoundaryPunctuation,
		firstTouchesLeft: true,
		lastTouchesRight: true,
		singleRun:        false,
	}

	seen := map[canonicalInlineTransferState]struct{}{state: {}}
	if _, ok := seen[state]; !ok {
		t.Fatal("transfer state must remain a comparable prune key")
	}
	if state.first.marker != '*' || state.last.marker != '_' {
		t.Fatalf("boundary markers = %q/%q, want */_", state.first.marker, state.last.marker)
	}
	if state.leftEdge != canonicalInlineBoundaryWhitespace ||
		state.rightEdge != canonicalInlineBoundaryPunctuation {
		t.Fatalf("boundary classes = %d/%d", state.leftEdge, state.rightEdge)
	}
	if state.firstDemand.openLevelOne != 1 || state.lastDemand.closeLevelOne != 2 {
		t.Fatalf("boundary demands = %+v/%+v", state.firstDemand, state.lastDemand)
	}
}
