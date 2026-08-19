package splice

import (
	"errors"
	"testing"
)

func TestIndexNodesBuildsDeterministicLookup(t *testing.T) {
	t.Parallel()

	nodes := []Node{
		{ID: NodeID("first"), Kind: KindParagraph},
		{ID: NodeID("second"), Kind: KindHeading},
	}
	index, err := indexNodes(nodes)
	if err != nil {
		t.Fatalf("indexNodes() error = %v", err)
	}
	if len(index) != len(nodes) || index[nodes[0].ID] != 0 || index[nodes[1].ID] != 1 {
		t.Fatalf("index = %#v, want first->0 and second->1", index)
	}
}

func TestIndexNodesRejectsDuplicateIDs(t *testing.T) {
	t.Parallel()

	_, err := indexNodes([]Node{
		{ID: NodeID("duplicate"), Kind: KindParagraph},
		{ID: NodeID("duplicate"), Kind: KindHeading},
	})
	if !errors.Is(err, errDuplicateNodeID) {
		t.Fatalf("indexNodes() error = %v, want errDuplicateNodeID", err)
	}
}
