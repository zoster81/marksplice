package splice

import "testing"

func TestRangeAfterPatchesHandlesForwardAndBackwardMoves(t *testing.T) {
	t.Parallel()

	forward := []patchTransform{
		{Range: Range{Start: 10, End: 20}},
		{Range: Range{Start: 40, End: 40}, ReplacementLength: 10},
	}
	forwardCases := []struct {
		name  string
		input Range
		want  Range
		ok    bool
	}{
		{name: "before source", input: Range{Start: 0, End: 5}, want: Range{Start: 0, End: 5}, ok: true},
		{name: "between source and destination", input: Range{Start: 20, End: 30}, want: Range{Start: 10, End: 20}, ok: true},
		{name: "ends at destination", input: Range{Start: 30, End: 40}, want: Range{Start: 20, End: 30}, ok: true},
		{name: "starts at destination", input: Range{Start: 40, End: 50}, want: Range{Start: 40, End: 50}, ok: true},
		{name: "overlaps removed source", input: Range{Start: 15, End: 25}, ok: false},
	}
	for _, tt := range forwardCases {
		t.Run("forward/"+tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := rangeAfterPatches(tt.input, forward)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("rangeAfterPatches(%v) = %v, %v; want %v, %v", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}

	backward := []patchTransform{
		{Range: Range{Start: 20, End: 30}},
		{Range: Range{Start: 5, End: 5}, ReplacementLength: 10},
	}
	backwardCases := []struct {
		name  string
		input Range
		want  Range
		ok    bool
	}{
		{name: "before destination", input: Range{Start: 0, End: 5}, want: Range{Start: 0, End: 5}, ok: true},
		{name: "between destination and source", input: Range{Start: 10, End: 20}, want: Range{Start: 20, End: 30}, ok: true},
		{name: "after source", input: Range{Start: 30, End: 40}, want: Range{Start: 30, End: 40}, ok: true},
		{name: "contains insertion point", input: Range{Start: 0, End: 10}, ok: false},
	}
	for _, tt := range backwardCases {
		t.Run("backward/"+tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := rangeAfterPatches(tt.input, backward)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("rangeAfterPatches(%v) = %v, %v; want %v, %v", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestRangeAfterPatchesRejectsInvalidPatchSets(t *testing.T) {
	t.Parallel()

	invalid := [][]patchTransform{
		{{Range: Range{Start: -1, End: 0}}},
		{{Range: Range{Start: 5, End: 4}}},
		{{Range: Range{Start: 1, End: 1}, ReplacementLength: -1}},
		{{Range: Range{Start: 2, End: 5}}, {Range: Range{Start: 4, End: 6}}},
		{{Range: Range{Start: 2, End: 2}}, {Range: Range{Start: 2, End: 5}}},
	}
	for i, patches := range invalid {
		if _, ok := rangeAfterPatches(Range{Start: 10, End: 12}, patches); ok {
			t.Fatalf("invalid patch set %d accepted", i)
		}
	}
}
