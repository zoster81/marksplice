package parser

import (
	"testing"
	"unsafe"
)

func TestNodeStorageBudget(t *testing.T) {
	t.Parallel()

	const maxNodeBytes = uintptr(176)
	if size := unsafe.Sizeof(Node{}); size > maxNodeBytes {
		t.Fatalf("Node size = %d bytes, want <= %d", size, maxNodeBytes)
	}
}
