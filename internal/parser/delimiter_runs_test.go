package parser

import (
	"reflect"
	"testing"
)

func TestResolveDelimiterRuns(t *testing.T) {
	tests := []struct {
		name string
		runs []DelimiterRun
		want []DelimiterRunMatch
	}{
		{
			name: "simple emphasis",
			runs: []DelimiterRun{
				{Start: 0, End: 1, Marker: '*', CanOpen: true},
				{Start: 2, End: 3, Marker: '*', CanClose: true},
			},
			want: []DelimiterRunMatch{
				{
					Marker:          '*',
					Level:           1,
					OpenerRun:       0,
					CloserRun:       1,
					OpeningConsumed: Range{Start: 0, End: 1},
					ClosingConsumed: Range{Start: 2, End: 3},
				},
			},
		},
		{
			name: "strong consumes two delimiters",
			runs: []DelimiterRun{
				{Start: 0, End: 2, Marker: '*', CanOpen: true},
				{Start: 3, End: 5, Marker: '*', CanClose: true},
			},
			want: []DelimiterRunMatch{
				{
					Marker:          '*',
					Level:           2,
					OpenerRun:       0,
					CloserRun:       1,
					OpeningConsumed: Range{Start: 0, End: 2},
					ClosingConsumed: Range{Start: 3, End: 5},
				},
			},
		},
		{
			name: "one closer run consumes nested strong then emphasis",
			runs: []DelimiterRun{
				{Start: 0, End: 3, Marker: '*', CanOpen: true},
				{Start: 4, End: 7, Marker: '*', CanClose: true},
			},
			want: []DelimiterRunMatch{
				{
					Marker:          '*',
					Level:           2,
					OpenerRun:       0,
					CloserRun:       1,
					OpeningConsumed: Range{Start: 1, End: 3},
					ClosingConsumed: Range{Start: 4, End: 6},
				},
				{
					Marker:          '*',
					Level:           1,
					OpenerRun:       0,
					CloserRun:       1,
					OpeningConsumed: Range{Start: 0, End: 1},
					ClosingConsumed: Range{Start: 6, End: 7},
				},
			},
		},
		{
			name: "one opener run can close nested emphasis layers",
			runs: []DelimiterRun{
				{Start: 0, End: 3, Marker: '*', CanOpen: true},
				{Start: 6, End: 7, Marker: '*', CanClose: true},
				{Start: 8, End: 9, Marker: '*', CanClose: true},
				{Start: 11, End: 12, Marker: '*', CanClose: true},
			},
			want: []DelimiterRunMatch{
				{
					Marker:          '*',
					Level:           1,
					OpenerRun:       0,
					CloserRun:       1,
					OpeningConsumed: Range{Start: 2, End: 3},
					ClosingConsumed: Range{Start: 6, End: 7},
				},
				{
					Marker:          '*',
					Level:           1,
					OpenerRun:       0,
					CloserRun:       2,
					OpeningConsumed: Range{Start: 1, End: 2},
					ClosingConsumed: Range{Start: 8, End: 9},
				},
				{
					Marker:          '*',
					Level:           1,
					OpenerRun:       0,
					CloserRun:       3,
					OpeningConsumed: Range{Start: 0, End: 1},
					ClosingConsumed: Range{Start: 11, End: 12},
				},
			},
		},
		{
			name: "modulo three conflict blocks a pair",
			runs: []DelimiterRun{
				{Start: 0, End: 2, Marker: '*', CanOpen: true, CanClose: true},
				{Start: 3, End: 4, Marker: '*', CanClose: true},
			},
			want: []DelimiterRunMatch{},
		},
		{
			name: "strikethrough single tilde requires matching width",
			runs: []DelimiterRun{
				{Start: 0, End: 1, Marker: '~', CanOpen: true},
				{Start: 2, End: 3, Marker: '~', CanClose: true},
			},
			want: []DelimiterRunMatch{
				{
					Marker:          '~',
					Level:           1,
					OpenerRun:       0,
					CloserRun:       1,
					OpeningConsumed: Range{Start: 0, End: 1},
					ClosingConsumed: Range{Start: 2, End: 3},
				},
			},
		},
		{
			name: "strikethrough double tilde requires matching width",
			runs: []DelimiterRun{
				{Start: 0, End: 2, Marker: '~', CanOpen: true},
				{Start: 3, End: 5, Marker: '~', CanClose: true},
			},
			want: []DelimiterRunMatch{
				{
					Marker:          '~',
					Level:           2,
					OpenerRun:       0,
					CloserRun:       1,
					OpeningConsumed: Range{Start: 0, End: 2},
					ClosingConsumed: Range{Start: 3, End: 5},
				},
			},
		},
		{
			name: "strikethrough double opener does not partially match single closer",
			runs: []DelimiterRun{
				{Start: 0, End: 2, Marker: '~', CanOpen: true},
				{Start: 3, End: 4, Marker: '~', CanClose: true},
			},
			want: []DelimiterRunMatch{},
		},
		{
			name: "strikethrough single opener does not partially match double closer",
			runs: []DelimiterRun{
				{Start: 0, End: 1, Marker: '~', CanOpen: true},
				{Start: 2, End: 4, Marker: '~', CanClose: true},
			},
			want: []DelimiterRunMatch{},
		},
		{
			name: "strikethrough nested equal-width runs preserve outer pair",
			runs: []DelimiterRun{
				{Start: 0, End: 2, Marker: '~', CanOpen: true},
				{Start: 17, End: 18, Marker: '~', CanOpen: true},
				{Start: 19, End: 20, Marker: '~', CanOpen: true, CanClose: true},
				{Start: 21, End: 23, Marker: '~', CanClose: true},
			},
			want: []DelimiterRunMatch{
				{Marker: '~', Level: 1, OpenerRun: 1, CloserRun: 2, OpeningConsumed: Range{Start: 17, End: 18}, ClosingConsumed: Range{Start: 19, End: 20}},
				{Marker: '~', Level: 2, OpenerRun: 0, CloserRun: 3, OpeningConsumed: Range{Start: 0, End: 2}, ClosingConsumed: Range{Start: 21, End: 23}},
			},
		},
		{
			name: "strikethrough runs longer than two are not delimiter candidates",
			runs: []DelimiterRun{
				{Start: 0, End: 3, Marker: '~', CanOpen: true},
				{Start: 4, End: 7, Marker: '~', CanClose: true},
			},
			want: []DelimiterRunMatch{},
		},
		{
			name: "closing an outer run invalidates inner openers above it",
			runs: []DelimiterRun{
				{Start: 0, End: 1, Marker: '*', CanOpen: true},
				{Start: 2, End: 3, Marker: '_', CanOpen: true},
				{Start: 4, End: 5, Marker: '*', CanClose: true},
				{Start: 6, End: 7, Marker: '_', CanClose: true},
			},
			want: []DelimiterRunMatch{
				{
					Marker:          '*',
					Level:           1,
					OpenerRun:       0,
					CloserRun:       2,
					OpeningConsumed: Range{Start: 0, End: 1},
					ClosingConsumed: Range{Start: 4, End: 5},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ResolveDelimiterRuns(test.runs)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("ResolveDelimiterRuns() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestResolveDelimiterRunsDoesNotMutateInput(t *testing.T) {
	runs := []DelimiterRun{
		{Start: 0, End: 3, Marker: '*', CanOpen: true},
		{Start: 4, End: 5, Marker: '*', CanClose: true},
	}
	before := append([]DelimiterRun(nil), runs...)
	_ = ResolveDelimiterRuns(runs)
	if !reflect.DeepEqual(runs, before) {
		t.Fatalf("ResolveDelimiterRuns mutated input: got %+v, want %+v", runs, before)
	}
}
