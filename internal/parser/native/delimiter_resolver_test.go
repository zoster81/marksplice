package native

import (
	"reflect"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestNativeDelimiterAdapterPreservesSharedResolverMatches(t *testing.T) {
	t.Helper()

	state := uint32(0x9e3779b9)
	next := func(limit uint32) uint32 {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		return state % limit
	}
	matchedSamples := 0

	for sample := 0; sample < 10000; sample++ {
		count := int(next(8)) + 1
		runs := make([]delimiterRun, 0, count)
		shared := make([]parser.DelimiterRun, 0, count)
		position := 0
		for index := 0; index < count; index++ {
			length := int(next(4)) + 1
			marker := []byte{'*', '_', '~'}[next(3)]
			canOpen := next(2) == 1
			canClose := next(2) == 1
			segment := int(next(3))
			start := position
			end := start + length
			position = end + int(next(3))

			runs = append(runs, delimiterRun{
				segment:  segment,
				start:    start,
				end:      end,
				marker:   marker,
				length:   length,
				canOpen:  canOpen,
				canClose: canClose,
			})
			shared = append(shared, parser.DelimiterRun{
				Start:    start,
				End:      end,
				Marker:   marker,
				CanOpen:  canOpen,
				CanClose: canClose,
			})
		}

		nativeMatches := processDelimiters(append([]delimiterRun(nil), runs...))
		sharedMatches := parser.ResolveDelimiterRuns(shared)
		if len(nativeMatches) != 0 {
			matchedSamples++
		}
		if len(nativeMatches) != len(sharedMatches) {
			t.Fatalf("sample %d match count = %d, shared = %d; runs=%+v", sample, len(nativeMatches), len(sharedMatches), shared)
		}
		for index := range nativeMatches {
			nativeMatch := nativeMatches[index]
			sharedMatch := sharedMatches[index]
			want := parser.DelimiterRunMatch{
				Marker:          nativeMatch.marker,
				Level:           nativeMatch.level,
				OpenerRun:       findDelimiterRunByStart(t, runs, nativeMatch.opener),
				CloserRun:       findDelimiterRunByStart(t, runs, nativeMatch.closer),
				OpeningConsumed: nativeMatch.openingConsumed,
				ClosingConsumed: nativeMatch.closingConsumed,
			}
			if !reflect.DeepEqual(sharedMatch, want) {
				t.Fatalf("sample %d match %d = %+v, want %+v; runs=%+v", sample, index, sharedMatch, want, shared)
			}
		}
	}
	if matchedSamples < 1000 {
		t.Fatalf("generated only %d matched samples", matchedSamples)
	}
}

func findDelimiterRunByStart(t *testing.T, runs []delimiterRun, start int) int {
	t.Helper()
	for index, run := range runs {
		if run.start == start {
			return index
		}
	}
	t.Fatalf("delimiter run starting at %d not found", start)
	return -1
}
