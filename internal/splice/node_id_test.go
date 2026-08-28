package splice

import (
	"testing"

	"github.com/zoster81/marksplice/internal/source"
)

var nodeIDAllocationSink NodeID

func TestMakeNodeIDStableAndSingleAllocation(t *testing.T) {
	var fingerprint source.Fingerprint
	for index := range fingerprint {
		fingerprint[index] = byte(index)
	}
	kind := KindHeading
	range_ := Range{Start: 10, End: 20}

	const want NodeID = "f8bbba17b7e890d763e6f0f2060a5968"
	if got := makeNodeID(fingerprint, kind, range_); got != want {
		t.Fatalf("makeNodeID() = %q, want %q", got, want)
	}

	allocations := testing.AllocsPerRun(1000, func() {
		nodeIDAllocationSink = makeNodeID(fingerprint, kind, range_)
	})
	if allocations > 1 {
		t.Fatalf("makeNodeID() allocations = %.0f, want <= 1", allocations)
	}
}
