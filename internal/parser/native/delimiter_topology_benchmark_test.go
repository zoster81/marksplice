package native

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func repeatedTopologyWitness(count int) ([]byte, []DelimiterTopologyPair) {
	source := []byte(strings.Repeat("*x* ", count))
	pairs := make([]DelimiterTopologyPair, count)
	for i := range pairs {
		pairs[i] = DelimiterTopologyPair{
			Marker: '*', Opening: parser.Range{Start: i * 4, End: i*4 + 1},
			Closing: parser.Range{Start: i*4 + 2, End: i*4 + 3},
		}
	}
	return source, pairs
}

func BenchmarkDelimiterBoundaryFactsScaling(b *testing.B) {
	for _, count := range []int{64, 256, 1024} {
		source, pairs := repeatedTopologyWitness(count)
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, ok := DelimiterTopologyBoundaryFactsForCandidate(source, nil, pairs); !ok {
					b.Fatal("valid topology rejected")
				}
			}
		})
	}
}

func TestDelimiterContainingRunBoundaries(t *testing.T) {
	source, pairs := repeatedTopologyWitness(64)
	runs, _, ok := delimiterTopologyLeadingRunPreparation(source, nil, pairs)
	if !ok {
		t.Fatal("valid witness rejected")
	}
	for i := len(pairs) - 1; i >= 0; i-- {
		for side, span := range []parser.Range{pairs[i].Opening, pairs[i].Closing} {
			if got := delimiterTopologyContainingRun(runs, '*', span); got != i*2+side {
				t.Fatalf("range %+v: run %d, want %d", span, got, i*2+side)
			}
		}
	}
	for _, span := range []parser.Range{{Start: -1, End: 1}, {Start: 1, End: 2}, {Start: 0, End: 3}, {Start: len(source), End: len(source) + 1}} {
		if got := delimiterTopologyContainingRun(runs, '*', span); got != -1 {
			t.Fatalf("unowned range %+v accepted by run %d", span, got)
		}
	}
	if delimiterTopologyContainingRun(runs, '_', pairs[0].Opening) != -1 ||
		delimiterTopologyContainingRun(nil, '*', pairs[0].Opening) != -1 {
		t.Fatal("wrong marker or empty run set accepted")
	}
}
