package native

import (
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestDelimiterTopologyTransferFactsForCandidate(t *testing.T) {
	t.Parallel()

	source := []byte("***x***")
	expected := []DelimiterTopologyPair{
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
	facts, ok := DelimiterTopologyTransferFactsForCandidate(source, nil, expected, false)
	if !ok {
		t.Fatal("transfer-facts derivation failed")
	}
	boundary, ok := DelimiterTopologyBoundaryFactsForCandidate(source, nil, expected)
	if !ok || facts.Boundary != boundary {
		t.Fatalf("boundary facts = %+v, want %+v", facts.Boundary, boundary)
	}
	incoming, ok := DelimiterTopologyIncomingOpenerMask(source, nil, expected)
	if !ok || facts.IncomingOpener != incoming {
		t.Fatalf("incoming mask = %02x, want %02x", facts.IncomingOpener, incoming)
	}
	star, underscore, ok := DelimiterTopologyLeadingRunForbiddenMasks(source, nil, expected)
	if !ok || facts.StarForbidden != star || facts.UnderscoreForbidden != underscore {
		t.Fatalf("forbidden masks = %02x/%02x, want %02x/%02x",
			facts.StarForbidden, facts.UnderscoreForbidden, star, underscore)
	}
}

func TestDelimiterTopologyTransferTildeMaskMatchesIndividualProbes(t *testing.T) {
	for _, source := range []string{"", "text", "*x*", "~x~", "~~x~~", "~x~ ~~y~~", "~~~x~~~", "*~x~*"} {
		runs, _, ok := delimiterTopologyLeadingRunPreparation([]byte(source), nil, nil)
		if !ok {
			t.Fatal("source preparation failed")
		}
		var expected []DelimiterTopologyPair
		for _, match := range processDelimiters(runs) {
			expected = append(expected, DelimiterTopologyPair{Marker: match.marker, Opening: match.openingConsumed, Closing: match.closingConsumed})
		}
		facts, ok := DelimiterTopologyTransferFactsForCandidate([]byte(source), nil, expected, true)
		if !ok {
			t.Fatalf("source %q rejected", source)
		}
		var want uint8
		for _, category := range []int{1, 2, 4, 5} {
			if !DelimiterTopologyPreservedWithLeadingRun([]byte(source), nil, expected, '~', category%3, category >= 3) {
				want |= 1 << category
			}
		}
		if facts.TildeForbidden != want {
			t.Fatalf("source %q: tilde mask %02x, want %02x", source, facts.TildeForbidden, want)
		}
		without, ok := DelimiterTopologyTransferFactsForCandidate([]byte(source), nil, expected, false)
		facts.TildeForbidden = 0
		if !ok || facts != without {
			t.Fatalf("source %q: tilde probes changed unrelated transfer facts", source)
		}
	}
}

func TestDelimiterTopologyIncomingOpenerMask(t *testing.T) {
	t.Parallel()

	source := []byte("*x*")
	expected := []DelimiterTopologyPair{{
		Marker:  '*',
		Opening: parser.Range{Start: 0, End: 1},
		Closing: parser.Range{Start: 2, End: 3},
	}}
	got, ok := DelimiterTopologyIncomingOpenerMask(source, nil, expected)
	if !ok {
		t.Fatal("incoming-opener derivation failed")
	}
	const want uint8 = 0x3b
	if got != want {
		t.Fatalf("incoming-opener mask = %02x, want %02x", got, want)
	}
}

func TestDelimiterTopologyBoundaryFactsForCandidate(t *testing.T) {
	t.Parallel()

	source := []byte("***x***")
	expected := []DelimiterTopologyPair{
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
	facts, ok := DelimiterTopologyBoundaryFactsForCandidate(source, nil, expected)
	if !ok {
		t.Fatal("boundary-facts derivation failed")
	}
	if facts.Empty || facts.SingleRun {
		t.Fatalf("boundary facts = %+v", facts)
	}
	if facts.First.Start != 0 || facts.First.End != 3 || facts.First.Marker != '*' {
		t.Fatalf("first run = %+v", facts.First)
	}
	if facts.Last.Start != 4 || facts.Last.End != 7 || facts.Last.Marker != '*' {
		t.Fatalf("last run = %+v", facts.Last)
	}
	if facts.FirstDemand.OpenLevelOne != 1 || facts.FirstDemand.CloseLevelOne != 0 {
		t.Fatalf("first demand = %+v", facts.FirstDemand)
	}
	if facts.LastDemand.OpenLevelOne != 0 || facts.LastDemand.CloseLevelOne != 1 {
		t.Fatalf("last demand = %+v", facts.LastDemand)
	}
}

func TestDelimiterTopologyLeadingRunForbiddenMasksReuseOneTopologySnapshot(t *testing.T) {
	t.Parallel()

	source := []byte("***x***")
	expected := []DelimiterTopologyPair{
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
	star, underscore, ok := DelimiterTopologyLeadingRunForbiddenMasks(source, nil, expected)
	if !ok {
		t.Fatal("forbidden-mask topology preparation failed")
	}
	for marker, got := range map[byte]uint8{'*': star, '_': underscore} {
		var want uint8
		for category := 0; category < 6; category++ {
			width := category % 3
			if width == 0 {
				width = 3
			}
			if !DelimiterTopologyPreservedWithLeadingRun(
				source, nil, expected, marker, width, category >= 3,
			) {
				want |= 1 << category
			}
		}
		if got != want {
			t.Fatalf("marker %q mask = %02x, want %02x", marker, got, want)
		}
	}
}

func TestDelimiterTopologyPreparedProbeScratchReuseIsEquivalent(t *testing.T) {
	t.Parallel()

	source := []byte("***x***")
	expected := []DelimiterTopologyPair{
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
	runs, want, ok := delimiterTopologyLeadingRunPreparation(source, nil, expected)
	if !ok {
		t.Fatal("delimiter topology preparation failed")
	}
	original := append([]delimiterRun(nil), runs...)
	scratch := make([]delimiterRun, 0, len(runs)+1)
	var resolver delimiterResolver

	for repeat := 0; repeat < 3; repeat++ {
		incoming, ok := delimiterTopologyIncomingOpenerMaskPrepared(source, runs, want, len(expected))
		if !ok {
			t.Fatal("incoming opener mask preparation failed")
		}
		baselineIncoming, ok := DelimiterTopologyIncomingOpenerMask(source, nil, expected)
		if !ok || incoming != baselineIncoming {
			t.Fatalf("incoming mask = %02x, want %02x", incoming, baselineIncoming)
		}
		for marker := range map[byte]struct{}{'*': {}, '_': {}} {
			for category := 0; category < 6; category++ {
				width := category % 3
				if width == 0 {
					width = 3
				}
				baseline := delimiterTopologyPreservedWithPreparedLeadingRun(
					runs, want, len(expected), marker, width, category >= 3,
				)
				got := delimiterTopologyPreservedWithPreparedLeadingRunUsing(
					scratch, runs, want, len(expected), marker, width, category >= 3, &resolver,
				)
				if got != baseline {
					t.Fatalf(
						"repeat=%d marker=%q category=%d reused=%v baseline=%v",
						repeat, marker, category, got, baseline,
					)
				}
			}
		}
		if len(runs) != len(original) {
			t.Fatalf("prepared runs length = %d, want %d", len(runs), len(original))
		}
		for index := range runs {
			if runs[index] != original[index] {
				t.Fatalf("prepared run %d mutated: got %+v want %+v", index, runs[index], original[index])
			}
		}
	}
}

func TestDelimiterTopologyMatchesOwnedObjectBoundary(t *testing.T) {
	t.Parallel()

	source := []byte("~~www.example.com~.~.~~")
	owners := []parser.Range{{Start: 2, End: 17}}
	expected := []DelimiterTopologyPair{
		{
			Marker:  '~',
			Opening: parser.Range{Start: 0, End: 2},
			Closing: parser.Range{Start: 21, End: 23},
		},
		{
			Marker:  '~',
			Opening: parser.Range{Start: 17, End: 18},
			Closing: parser.Range{Start: 19, End: 20},
		},
	}

	if !DelimiterTopologyMatches(source, owners, expected) {
		t.Fatal("owned-object delimiter topology did not match")
	}
}

func TestDelimiterTopologyMatchesCoalescedEmphasisConsumption(t *testing.T) {
	t.Parallel()

	source := []byte("***x***")
	expected := []DelimiterTopologyPair{
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

	if !DelimiterTopologyMatches(source, nil, expected) {
		t.Fatal("coalesced emphasis topology did not match")
	}
}

func TestDelimiterTopologyMatchesInContextCompletesDeferredSubtree(t *testing.T) {
	t.Parallel()

	source := []byte("*\\!*\\!*\\)*")
	expected := []DelimiterTopologyPair{
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

	if DelimiterTopologyMatches(source, nil, expected) {
		t.Fatal("deferred subtree unexpectedly matched without ancestor context")
	}
	if !DelimiterTopologyMatchesInContext(source, nil, expected, []byte("*"), nil) {
		t.Fatal("leading ancestor delimiter did not complete deferred subtree topology")
	}
	if DelimiterTopologyMatchesInContext(source, nil, expected, []byte("_"), nil) {
		t.Fatal("unrelated ancestor delimiter incorrectly completed deferred subtree topology")
	}
}

func TestDelimiterTopologyMatchesInContextRejectsUnexpectedCandidateConsumption(t *testing.T) {
	t.Parallel()

	source := []byte("*x*")
	expected := []DelimiterTopologyPair{{
		Marker:  '*',
		Opening: parser.Range{Start: 0, End: 1},
		Closing: parser.Range{Start: 2, End: 3},
	}}
	if DelimiterTopologyMatchesInContext(source, nil, expected, []byte("*a"), nil) {
		t.Fatal("context that reinterprets candidate delimiter bytes was accepted")
	}
}

func TestDelimiterTopologyPreservedWithLeadingRun(t *testing.T) {
	t.Parallel()

	source := []byte("*x*")
	expected := []DelimiterTopologyPair{{
		Marker:  '*',
		Opening: parser.Range{Start: 0, End: 1},
		Closing: parser.Range{Start: 2, End: 3},
	}}
	if !DelimiterTopologyPreservedWithLeadingRun(source, nil, expected, '~', 1, false) {
		t.Fatal("unrelated leading run changed delimiter topology")
	}
	if DelimiterTopologyPreservedWithLeadingRun(source, nil, expected, '!', 1, false) {
		t.Fatal("invalid leading marker was accepted")
	}
}

func TestDelimiterTopologyRejectsWrongExpectedConsumption(t *testing.T) {
	t.Parallel()

	source := []byte("***x***")
	expected := []DelimiterTopologyPair{
		{
			Marker:  '*',
			Opening: parser.Range{Start: 0, End: 2},
			Closing: parser.Range{Start: 5, End: 7},
		},
	}

	if DelimiterTopologyMatches(source, nil, expected) {
		t.Fatal("wrong expected delimiter consumption matched")
	}
}
