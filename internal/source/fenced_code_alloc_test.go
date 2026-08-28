package source

import "testing"

var fencedCodeAllocationSink FencedCodeMapping

func TestMapFencedCodeAllocationBudget(t *testing.T) {
	source := []byte("```go\nold\n```\n")
	content := Range{Start: len("```go\n"), End: len("```go\nold")}

	mapping, err := MapFencedCode(source, content)
	if err != nil {
		t.Fatalf("MapFencedCode() error = %v", err)
	}
	if mapping.ContentRange != content || mapping.FenceChar != '`' || mapping.FenceLength != 3 || !mapping.Closed {
		t.Fatalf("MapFencedCode() = %+v, want closed backtick mapping for %v", mapping, content)
	}

	allocations := testing.AllocsPerRun(1000, func() {
		var err error
		fencedCodeAllocationSink, err = MapFencedCode(source, content)
		if err != nil {
			panic(err)
		}
	})
	if allocations != 0 {
		t.Fatalf("MapFencedCode() allocations = %.0f, want 0", allocations)
	}
}
