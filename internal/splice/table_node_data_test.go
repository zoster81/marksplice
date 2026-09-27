package splice

import (
	"reflect"
	"testing"
)

func TestTableNodeDataKeepsSnapshotAndReturnedValuesIndependent(t *testing.T) {
	document, err := Parse([]byte("paragraph\n\n| left | right |\n| :--- | ---: |\n| a | b |\n| c | d |\n"))
	if err != nil {
		t.Fatal(err)
	}
	original := document.Nodes()
	returned := document.Nodes()
	var tableKinds [3]bool
	for index := range returned {
		node := &returned[index]
		switch node.Kind {
		case KindTable:
			tableKinds[0] = true
		case KindTableRow:
			tableKinds[1] = true
		case KindTableCell:
			tableKinds[2] = true
		default:
			if !reflect.DeepEqual(node.TableData(), TableNodeData{}) {
				t.Fatal("non-table node has table data")
			}
			if _, ok := document.TableNode(node.ID); ok {
				t.Fatal("non-table node has a table view")
			}
			continue
		}
		view, ok := document.TableNode(node.ID)
		if !ok || view.ID != node.ID || view.Kind != node.Kind || view.Range != node.Range || view.ContentRange != node.ContentRange || !reflect.DeepEqual(view.TableNodeData, node.TableData()) {
			t.Fatal("table view differs from the complete node")
		}
		view.TableColumnCount = -4
		if len(view.TableAlignments) != 0 {
			view.TableAlignments[0] = TableAlignmentCenter
		}
		metadata := node.TableData()
		metadata.TableColumnCount = -1
		if node.TableData().TableColumnCount == -1 {
			t.Fatal("scalar metadata aliases its node")
		}
		node.table.TableColumnCount = -2
		node.table.TableID = "changed"
		if len(metadata.TableAlignments) != 0 {
			metadata.TableAlignments[0] = TableAlignmentCenter
		}
		single, ok := document.Node(node.ID)
		if !ok {
			t.Fatal("missing table node")
		}
		single.table.TableColumnCount = -3
		if len(single.table.TableAlignments) != 0 {
			single.table.TableAlignments[0] = TableAlignmentCenter
		}
		indexed, ok := document.NodeAt(index)
		if !ok || !reflect.DeepEqual(indexed, original[index]) {
			t.Fatal("indexed table node differs from the complete snapshot")
		}
		indexed.table.TableColumnCount = -5
		if len(indexed.table.TableAlignments) != 0 {
			indexed.table.TableAlignments[0] = TableAlignmentCenter
		}
	}
	if tableKinds != [3]bool{true, true, true} {
		t.Fatalf("missing table kind coverage: %v", tableKinds)
	}
	if !reflect.DeepEqual(document.Nodes(), original) {
		t.Fatal("mutation of returned table data changed the snapshot")
	}
	if !reflect.DeepEqual((Node{}).TableData(), TableNodeData{}) {
		t.Fatal("zero node has nonzero table data")
	}
	for _, value := range []*Document{nil, {}, document} {
		if _, ok := value.TableNode("missing"); ok {
			t.Fatal("missing node has a table view")
		}
	}
}
