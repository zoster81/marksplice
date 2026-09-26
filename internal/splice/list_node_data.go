package splice

import "github.com/zoster81/marksplice/internal/parser"

// ListNodeData holds source and adjacency metadata for list-item nodes.
type ListNodeData struct {
	ListOrdered          bool
	ListMarker           byte
	ListHasParent        bool
	ListHasChildren      bool
	ListSubtreeComplete  bool
	ListParentID         NodeID
	ListParentAnchor     int
	ListContainerAnchor  int
	ListDirectChildCount int
	ListChildStart       int
	ListChildCount       int
	ListSubtreeEnd       int
	ListItemLineRange    Range
}

// Shared only as an immutable read view. Construction writes use owned entries.
var emptyListNodeData ListNodeData

func (node Node) listData() *ListNodeData {
	if node.list == nil {
		return &emptyListNodeData
	}
	return node.list
}

// ListData returns detached scalar metadata, zero when no list facts exist.
func (node Node) ListData() ListNodeData {
	return *node.listData()
}

// ListNode is the detached source and hierarchy view of a promoted list item.
type ListNode struct {
	ID           NodeID
	ContentRange Range
	ListNodeData
}

// ListNode returns promoted list metadata without allocating a complete Node.
func (d *Document) ListNode(id NodeID) (ListNode, bool) {
	if d == nil {
		return ListNode{}, false
	}
	node, ok := d.nodeByID(id)
	if !ok || node.Kind != KindListItem || !node.Editable || node.list == nil {
		return ListNode{}, false
	}
	return ListNode{ID: node.ID, ContentRange: node.ContentRange, ListNodeData: *node.list}, true
}

func (mapper *nodeMapper) mapListNodeData(observation parser.Node, node *Node) {
	data := ListNodeData{
		ListOrdered:          observation.Ordered,
		ListMarker:           observation.Marker,
		ListHasParent:        observation.HasListParent,
		ListParentAnchor:     observation.ListParentAnchor,
		ListContainerAnchor:  observation.ListContainerAnchor,
		ListHasChildren:      observation.HasListChildren,
		ListDirectChildCount: observation.ListDirectChildCount,
	}
	// Preserve unusual backend facts on other kinds as well as list-item data.
	if node.Kind != KindListItem && data == (ListNodeData{}) {
		return
	}
	mapper.listNodeData = append(mapper.listNodeData, data)
	node.list = &mapper.listNodeData[len(mapper.listNodeData)-1]
}
