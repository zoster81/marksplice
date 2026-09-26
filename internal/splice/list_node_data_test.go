package splice

import (
	"reflect"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestListNodeDataKeepsSnapshotAndReturnedValuesIndependent(t *testing.T) {
	document, err := Parse([]byte("paragraph\n\n- parent\n  - child\n- sibling\n"))
	if err != nil {
		t.Fatal(err)
	}
	original := document.Nodes()
	returned := document.Nodes()
	items := 0
	for index := range returned {
		node := &returned[index]
		if node.Kind != KindListItem {
			if node.list != nil || node.ListData() != (ListNodeData{}) {
				t.Fatal("non-list node has list metadata")
			}
			if _, ok := document.ListNode(node.ID); ok {
				t.Fatal("non-list node has a list view")
			}
			continue
		}
		items++
		view, ok := document.ListNode(node.ID)
		if !ok || view.ID != node.ID || view.ContentRange != node.ContentRange || view.ListNodeData != node.ListData() {
			t.Fatal("list view differs from complete node")
		}
		view.ListChildCount = -1
		data := node.ListData()
		data.ListParentID = "changed"
		if node.ListData().ListParentID == data.ListParentID || node.ListData().ListChildCount == view.ListChildCount {
			t.Fatal("scalar metadata aliases its node")
		}
		node.list.ListParentID = "changed"
		node.list.ListChildCount = -2
		single, ok := document.Node(node.ID)
		if !ok {
			t.Fatal("missing list node")
		}
		single.list.ListSubtreeEnd = -3
	}
	if items != 3 || !reflect.DeepEqual(document.Nodes(), original) {
		t.Fatal("missing list items or snapshot changed through returned metadata")
	}
	if (Node{}).ListData() != (ListNodeData{}) {
		t.Fatal("zero node has list metadata")
	}
	for _, value := range []*Document{nil, {}, document} {
		if _, ok := value.ListNode("missing"); ok {
			t.Fatal("missing node has a list view")
		}
	}
}

func TestListNodeDataPreservesNonListBackendFactsAndArenaGrowth(t *testing.T) {
	mapper := nodeMapper{snapshot: []byte("text"), listNodeData: make([]ListNodeData, 0, 1)}
	observation := parser.Node{
		Kind: parser.KindParagraph, Range: parser.Range{Start: 0, End: 4},
		Ordered: true, Marker: ')', HasListParent: true, ListParentAnchor: 2,
		ListContainerAnchor: 3, HasListChildren: true, ListDirectChildCount: 4,
	}
	want := ListNodeData{
		ListOrdered: true, ListMarker: ')', ListHasParent: true, ListParentAnchor: 2,
		ListContainerAnchor: 3, ListHasChildren: true, ListDirectChildCount: 4,
	}
	var nodes []Node
	for index := 0; index < 16; index++ {
		node, err := mapper.nodeFromObservation(observation)
		if err != nil {
			t.Fatal(err)
		}
		nodes = append(nodes, node)
	}
	for index := range nodes {
		if nodes[index].ListData() != want {
			t.Fatal("backend facts lost during metadata allocation")
		}
		nodes[index].list.ListMarker = '+'
	}
}
