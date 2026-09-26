package splice

import "fmt"

type listItemModel struct {
	itemIndexes []int
	childIDs    []NodeID
}

type listItemModelBuilder struct {
	nodes              []Node
	itemIndexes        []int
	ordinalByLineStart map[int]int
	parentOrdinals     []int
	supportedChildren  []int
}

func resolveListItemModel(nodes []Node) (listItemModel, error) {
	builder := listItemModelBuilder{
		nodes:              nodes,
		ordinalByLineStart: make(map[int]int),
	}
	if err := builder.collectItems(); err != nil {
		return listItemModel{}, err
	}
	builder.parentOrdinals = make([]int, len(builder.itemIndexes))
	for ordinal := range builder.parentOrdinals {
		builder.parentOrdinals[ordinal] = -1
	}
	builder.supportedChildren = make([]int, len(builder.itemIndexes))
	builder.resolveParents()
	childIDs, err := builder.buildChildAdjacency()
	if err != nil {
		return listItemModel{}, err
	}
	if err := builder.resolveSubtrees(); err != nil {
		return listItemModel{}, err
	}
	return listItemModel{itemIndexes: builder.itemIndexes, childIDs: childIDs}, nil
}

func (b *listItemModelBuilder) collectItems() error {
	seenIDs := make(map[NodeID]int)
	lastLineStart := -1
	for index := range b.nodes {
		node := &b.nodes[index]
		if node.Kind != KindListItem || !node.Editable {
			continue
		}
		if node.listData().ListDirectChildCount < 0 || node.listData().ListHasChildren != (node.listData().ListDirectChildCount != 0) {
			return fmt.Errorf("inconsistent supported list-item child metadata for %q", node.ID)
		}
		if _, exists := seenIDs[node.ID]; exists {
			return fmt.Errorf("%w: %q", errDuplicateNodeID, node.ID)
		}
		lineStart := node.listData().ListItemLineRange.Start
		if previousOrdinal, exists := b.ordinalByLineStart[lineStart]; exists {
			previous := b.nodes[b.itemIndexes[previousOrdinal]]
			return fmt.Errorf("duplicate supported list-item line start %d for %q and %q", lineStart, previous.ID, node.ID)
		}
		if lineStart <= lastLineStart {
			return fmt.Errorf("supported list-item nodes are not in strict source order at %q", node.ID)
		}
		ordinal := len(b.itemIndexes)
		seenIDs[node.ID] = ordinal
		b.ordinalByLineStart[lineStart] = ordinal
		b.itemIndexes = append(b.itemIndexes, index)
		node.list.ListParentID = ""
		node.list.ListChildStart = 0
		node.list.ListChildCount = 0
		node.list.ListSubtreeComplete = false
		node.list.ListSubtreeEnd = node.listData().ListItemLineRange.End
		lastLineStart = lineStart
	}
	return nil
}

func (b *listItemModelBuilder) resolveParents() {
	for ordinal, nodeIndex := range b.itemIndexes {
		node := &b.nodes[nodeIndex]
		if !node.listData().ListHasParent {
			continue
		}
		parentOrdinal, ok := b.ordinalByLineStart[node.listData().ListParentAnchor]
		if !ok {
			continue
		}
		parent := b.nodes[b.itemIndexes[parentOrdinal]]
		node.list.ListParentID = parent.ID
		b.parentOrdinals[ordinal] = parentOrdinal
		b.supportedChildren[parentOrdinal]++
	}
}

func (b *listItemModelBuilder) buildChildAdjacency() ([]NodeID, error) {
	childOffsets := make([]int, len(b.itemIndexes)+1)
	for ordinal, childCount := range b.supportedChildren {
		childOffsets[ordinal+1] = childOffsets[ordinal] + childCount
	}
	childIDs := make([]NodeID, childOffsets[len(b.itemIndexes)])
	childCursors := append([]int(nil), childOffsets[:len(b.itemIndexes)]...)
	for ordinal, nodeIndex := range b.itemIndexes {
		parentOrdinal := b.parentOrdinals[ordinal]
		if parentOrdinal < 0 {
			continue
		}
		cursor := childCursors[parentOrdinal]
		if cursor < childOffsets[parentOrdinal] || cursor >= childOffsets[parentOrdinal+1] {
			parent := b.nodes[b.itemIndexes[parentOrdinal]]
			return nil, fmt.Errorf("supported list-item child adjacency overflow for parent %q", parent.ID)
		}
		childIDs[cursor] = b.nodes[nodeIndex].ID
		childCursors[parentOrdinal]++
	}
	for ordinal, nodeIndex := range b.itemIndexes {
		if childCursors[ordinal] != childOffsets[ordinal+1] {
			return nil, fmt.Errorf("incomplete supported list-item child adjacency for %q", b.nodes[nodeIndex].ID)
		}
		b.nodes[nodeIndex].list.ListChildStart = childOffsets[ordinal]
		b.nodes[nodeIndex].list.ListChildCount = b.supportedChildren[ordinal]
	}
	return childIDs, nil
}

func (b *listItemModelBuilder) resolveSubtrees() error {
	remainingChildren := append([]int(nil), b.supportedChildren...)
	childrenComplete := make([]bool, len(b.itemIndexes))
	queue := make([]int, 0, len(b.itemIndexes))
	for ordinal := range b.itemIndexes {
		childrenComplete[ordinal] = true
		if remainingChildren[ordinal] == 0 {
			queue = append(queue, ordinal)
		}
	}

	processed := 0
	for len(queue) != 0 {
		ordinal := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		node := &b.nodes[b.itemIndexes[ordinal]]
		node.list.ListSubtreeComplete = childrenComplete[ordinal] && b.supportedChildren[ordinal] == node.listData().ListDirectChildCount
		processed++

		parentOrdinal := b.parentOrdinals[ordinal]
		if parentOrdinal < 0 {
			continue
		}
		parent := &b.nodes[b.itemIndexes[parentOrdinal]]
		if node.listData().ListSubtreeEnd > parent.listData().ListSubtreeEnd {
			parent.list.ListSubtreeEnd = node.listData().ListSubtreeEnd
		}
		if !node.listData().ListSubtreeComplete {
			childrenComplete[parentOrdinal] = false
		}
		remainingChildren[parentOrdinal]--
		if remainingChildren[parentOrdinal] == 0 {
			queue = append(queue, parentOrdinal)
		}
	}
	if processed != len(b.itemIndexes) {
		return fmt.Errorf("supported list-item parent relation contains a cycle")
	}
	return nil
}

func listItemSubtreeRange(item Node) Range {
	return Range{Start: item.listData().ListItemLineRange.Start, End: item.listData().ListSubtreeEnd}
}

type listItemSubtreeOwnership struct {
	Root  Node
	Range Range
	IDs   map[NodeID]struct{}
}

func (d *Document) ownedListItemSubtree(root Node) (listItemSubtreeOwnership, error) {
	if root.Kind != KindListItem || !root.Editable || !root.listData().ListSubtreeComplete {
		return listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	range_ := listItemSubtreeRange(root)
	ids, ok := d.listItemSubtreeIDs(root, range_)
	if !ok {
		return listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	return listItemSubtreeOwnership{Root: root, Range: range_, IDs: ids}, nil
}

func (d *Document) listItemSubtreeIDs(root Node, subtreeRange Range) (map[NodeID]struct{}, bool) {
	if d == nil || !subtreeRange.Valid(len(d.source)) || subtreeRange.Start == subtreeRange.End {
		return nil, false
	}
	ids := make(map[NodeID]struct{})
	stack := []Node{root}
	for len(stack) != 0 {
		last := len(stack) - 1
		node := stack[last]
		stack = stack[:last]
		if _, duplicate := ids[node.ID]; duplicate || !node.listData().ListSubtreeComplete {
			return nil, false
		}
		lineRange := node.listData().ListItemLineRange
		if lineRange.Start < subtreeRange.Start || lineRange.End > subtreeRange.End {
			return nil, false
		}
		children, ok := d.listItemChildIDSpan(node)
		if !ok || len(children) != node.listData().ListDirectChildCount {
			return nil, false
		}
		ids[node.ID] = struct{}{}
		for index := len(children) - 1; index >= 0; index-- {
			child, ok := d.nodeByID(children[index])
			if !ok {
				return nil, false
			}
			stack = append(stack, child)
		}
	}
	return ids, len(ids) != 0
}

func (d *Document) listItemChildIDSpan(parent Node) ([]NodeID, bool) {
	if !validListItemChildSpanRequest(d, parent) {
		return nil, false
	}
	start := parent.listData().ListChildStart
	count := parent.listData().ListChildCount
	if start > len(d.listChildIDs) || count > len(d.listChildIDs)-start {
		return nil, false
	}
	ids := d.listChildIDs[start : start+count]
	if !d.validListItemChildren(parent, ids) {
		return nil, false
	}
	return ids, true
}

func validListItemChildSpanRequest(d *Document, parent Node) bool {
	return d != nil && parent.Kind == KindListItem && parent.Editable &&
		parent.listData().ListChildStart >= 0 && parent.listData().ListChildCount >= 0 && parent.listData().ListChildCount <= parent.listData().ListDirectChildCount
}

func (d *Document) validListItemChildren(parent Node, ids []NodeID) bool {
	previousStart := parent.listData().ListItemLineRange.Start
	for _, id := range ids {
		child, ok := d.nodeByID(id)
		if !ok || !validListItemChild(parent, child, previousStart) {
			return false
		}
		previousStart = child.listData().ListItemLineRange.Start
	}
	return true
}

func validListItemChild(parent, child Node, previousStart int) bool {
	return child.Kind == KindListItem && child.Editable && child.listData().ListParentID == parent.ID &&
		child.listData().ListParentAnchor == parent.listData().ListItemLineRange.Start && child.listData().ListItemLineRange.Start > previousStart
}

func (d *Document) promotedListItemCount() int {
	if d == nil {
		return 0
	}
	return len(d.listItemIndexes)
}
