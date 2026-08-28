package native

import (
	"bytes"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestScanNativeInlineDollarRunsWithoutSparseConstraints(t *testing.T) {
	t.Parallel()

	source := []byte("plain $x$ text")
	anchor := bytes.Index(source, []byte("$x$"))
	if anchor < 0 {
		t.Fatal("test fixture marker not found")
	}
	block := inlineBlock{segments: []parser.Range{{Start: 0, End: len(source)}}}
	got := scanNativeInlineDollarRuns(source, block, nil, nil)
	if len(got) != 1 || got[0].Range != (parser.Range{Start: anchor, End: anchor + 3}) || got[0].PayloadRange != (parser.Range{Start: anchor + 1, End: anchor + 2}) {
		t.Fatalf("scanNativeInlineDollarRuns(nil constraints) = %#v, want one inline-dollar observation", got)
	}
}

func TestScanNativeInlineDollarRunsRespectsExclusionsAndBoundaries(t *testing.T) {
	t.Parallel()

	source := []byte("lead $a$ hide $b$ split $c$")
	first := bytes.Index(source, []byte("$a$"))
	hidden := bytes.Index(source, []byte("$b$"))
	split := bytes.Index(source, []byte("$c$"))
	if first < 0 || hidden < 0 || split < 0 {
		t.Fatal("test fixture markers not found")
	}

	block := inlineBlock{segments: []parser.Range{{Start: 0, End: len(source)}}}
	exclusions := [][]parser.Range{{{Start: hidden, End: hidden + len("$b$")}}}
	boundaries := [][]int{{split + 2}}
	got := scanNativeInlineDollarRuns(source, block, exclusions, boundaries)

	wantRange := parser.Range{Start: first, End: first + len("$a$")}
	wantPayload := parser.Range{Start: first + 1, End: first + 2}
	if len(got) != 1 || got[0].Style != parser.MathExpressionInlineDollar || got[0].Range != wantRange || got[0].PayloadRange != wantPayload {
		t.Fatalf("scanNativeInlineDollarRuns() = %#v, want one inline-dollar observation range=%v payload=%v", got, wantRange, wantPayload)
	}
}
