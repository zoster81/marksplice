package parser

import (
	"reflect"
	"testing"
)

func TestDelimiterRunResolverReuse(t *testing.T) {
	// Complete witnesses: ***x***, _x_, plain text, and ~~x~~.
	inputs := [][]DelimiterRun{
		{{Start: 0, End: 3, Marker: '*', CanOpen: true}, {Start: 4, End: 7, Marker: '*', CanClose: true}},
		{{Start: 0, End: 1, Marker: '_', CanOpen: true}, {Start: 2, End: 3, Marker: '_', CanClose: true}},
		nil,
		{{Start: 0, End: 2, Marker: '~', CanOpen: true}, {Start: 3, End: 5, Marker: '~', CanClose: true}},
	}
	want := make([][]DelimiterRunMatch, len(inputs))
	for i, input := range inputs {
		want[i] = ResolveDelimiterRuns(input)
	}
	var resolver DelimiterRunResolver
	for pass := 0; pass < 4; pass++ {
		for i, input := range inputs {
			before := append([]DelimiterRun(nil), input...)
			if got := resolver.Resolve(input); !reflect.DeepEqual(got, want[i]) {
				t.Fatalf("pass %d input %d: got %#v, want %#v", pass, i, got, want[i])
			}
			if !reflect.DeepEqual(input, before) {
				t.Fatalf("input %d changed", i)
			}
		}
	}
}
