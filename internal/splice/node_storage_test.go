package splice

import (
	"testing"
	"unsafe"
)

func TestNodeStorageBudget(t *testing.T) {
	t.Parallel()

	const maxNodeBytes = uintptr(568)
	if size := unsafe.Sizeof(Node{}); size > maxNodeBytes {
		t.Fatalf("Node size = %d bytes, want <= %d", size, maxNodeBytes)
	}
}
