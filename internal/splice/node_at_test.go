package splice

import (
	"reflect"
	"testing"
)

func TestNodeAtPreservesOrderAndChecksBounds(t *testing.T) {
	document, err := Parse([]byte("# Heading\n\nText with *emphasis* and [link](target).\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := document.Nodes()
	for index := range document.NodeCount() {
		node, ok := document.NodeAt(index)
		if !ok || !reflect.DeepEqual(node, want[index]) {
			t.Fatalf("node %d differs from the source-ordered snapshot", index)
		}
		node.ID = "changed"
		node.Range.End = -1
	}
	if !reflect.DeepEqual(document.Nodes(), want) {
		t.Fatal("indexed node mutation changed the snapshot")
	}
	for _, item := range []*Document{nil, {}, document} {
		for _, index := range []int{-1, item.NodeCount(), item.NodeCount() + 1} {
			node, ok := item.NodeAt(index)
			if ok || !reflect.DeepEqual(node, Node{}) {
				t.Fatalf("out-of-range node %d returned %+v/%v", index, node, ok)
			}
		}
	}
}
