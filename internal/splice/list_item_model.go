package splice

import "fmt"

func resolveListItemParentIDs(nodes []Node) error {
	byLineStart := make(map[int]NodeID)
	for _, node := range nodes {
		if node.Kind != KindListItem || !node.Editable {
			continue
		}
		lineStart := node.ListItemSource.LineRange.Start
		if existing, exists := byLineStart[lineStart]; exists {
			return fmt.Errorf("duplicate supported list-item line start %d for %q and %q", lineStart, existing, node.ID)
		}
		byLineStart[lineStart] = node.ID
	}
	for i := range nodes {
		if nodes[i].Kind != KindListItem || !nodes[i].ListHasParent {
			continue
		}
		if parentID, ok := byLineStart[nodes[i].ListParentAnchor]; ok {
			nodes[i].ListParentID = parentID
		}
	}
	return nil
}

func resolveListItemSubtrees(nodes []Node) ([]NodeID, error) {
	listIndexes := make([]int, 0)
	ordinalByID := make(map[NodeID]int)
	lastLineStart := -1
	for i := range nodes {
		node := &nodes[i]
		if node.Kind != KindListItem || !node.Editable {
			continue
		}
		if node.ListDirectChildCount < 0 || node.ListHasChildren != (node.ListDirectChildCount != 0) {
			return nil, fmt.Errorf("inconsistent supported list-item child metadata for %q", node.ID)
		}
		if _, exists := ordinalByID[node.ID]; exists {
			return nil, fmt.Errorf("%w: %q", errDuplicateNodeID, node.ID)
		}
		lineStart := node.ListItemSource.LineRange.Start
		if lineStart <= lastLineStart {
			return nil, fmt.Errorf("supported list-item nodes are not in strict source order at %q", node.ID)
		}
		lastLineStart = lineStart
		ordinalByID[node.ID] = len(listIndexes)
		listIndexes = append(listIndexes, i)
		node.ListSubtreeEnd = node.ListItemSource.LineRange.End
	}

	supportedChildren := make([]int, len(listIndexes))
	remainingChildren := make([]int, len(listIndexes))
	childrenComplete := make([]bool, len(listIndexes))
	for ordinal := range listIndexes {
		childrenComplete[ordinal] = true
	}
	for _, nodeIndex := range listIndexes {
		parentID := nodes[nodeIndex].ListParentID
		if parentID == "" {
			continue
		}
		parentOrdinal, ok := ordinalByID[parentID]
		if !ok {
			return nil, fmt.Errorf("supported list-item parent %q for %q is missing", parentID, nodes[nodeIndex].ID)
		}
		supportedChildren[parentOrdinal]++
	}
	copy(remainingChildren, supportedChildren)

	childOffsets := make([]int, len(listIndexes)+1)
	for ordinal, childCount := range supportedChildren {
		childOffsets[ordinal+1] = childOffsets[ordinal] + childCount
	}
	listChildIDs := make([]NodeID, childOffsets[len(listIndexes)])
	childCursors := append([]int(nil), childOffsets[:len(listIndexes)]...)
	for _, nodeIndex := range listIndexes {
		parentID := nodes[nodeIndex].ListParentID
		if parentID == "" {
			continue
		}
		parentOrdinal := ordinalByID[parentID]
		cursor := childCursors[parentOrdinal]
		if cursor < childOffsets[parentOrdinal] || cursor >= childOffsets[parentOrdinal+1] {
			return nil, fmt.Errorf("supported list-item child adjacency overflow for parent %q", parentID)
		}
		listChildIDs[cursor] = nodes[nodeIndex].ID
		childCursors[parentOrdinal]++
	}
	for ordinal, nodeIndex := range listIndexes {
		if childCursors[ordinal] != childOffsets[ordinal+1] {
			return nil, fmt.Errorf("incomplete supported list-item child adjacency for %q", nodes[nodeIndex].ID)
		}
		nodes[nodeIndex].ListChildStart = childOffsets[ordinal]
		nodes[nodeIndex].ListChildCount = supportedChildren[ordinal]
	}

	queue := make([]int, 0, len(listIndexes))
	for ordinal := range listIndexes {
		if remainingChildren[ordinal] == 0 {
			queue = append(queue, ordinal)
		}
	}

	processed := 0
	for len(queue) != 0 {
		ordinal := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		nodeIndex := listIndexes[ordinal]
		node := &nodes[nodeIndex]
		node.ListSubtreeComplete = childrenComplete[ordinal] && supportedChildren[ordinal] == node.ListDirectChildCount
		processed++

		if node.ListParentID == "" {
			continue
		}
		parentOrdinal := ordinalByID[node.ListParentID]
		parentIndex := listIndexes[parentOrdinal]
		if node.ListSubtreeEnd > nodes[parentIndex].ListSubtreeEnd {
			nodes[parentIndex].ListSubtreeEnd = node.ListSubtreeEnd
		}
		if !node.ListSubtreeComplete {
			childrenComplete[parentOrdinal] = false
		}
		remainingChildren[parentOrdinal]--
		if remainingChildren[parentOrdinal] == 0 {
			queue = append(queue, parentOrdinal)
		}
	}
	if processed != len(listIndexes) {
		return nil, fmt.Errorf("supported list-item parent relation contains a cycle")
	}
	return listChildIDs, nil
}

func listItemSubtreeRange(item Node) Range {
	return Range{Start: item.ListItemSource.LineRange.Start, End: item.ListSubtreeEnd}
}

type listItemSubtreeOwnership struct {
	Root  Node
	Range Range
	IDs   map[NodeID]struct{}
}

func (d *Document) ownedListItemSubtree(root Node) (listItemSubtreeOwnership, error) {
	if root.Kind != KindListItem || !root.Editable || !root.ListSubtreeComplete {
		return listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	range_ := listItemSubtreeRange(root)
	ids := d.listItemIDsWithinRange(range_)
	if _, ok := ids[root.ID]; !ok || len(ids) == 0 {
		return listItemSubtreeOwnership{}, ErrInvalidReplacement
	}
	return listItemSubtreeOwnership{Root: root, Range: range_, IDs: ids}, nil
}

func (d *Document) listItemIDsWithinRange(range_ Range) map[NodeID]struct{} {
	ids := make(map[NodeID]struct{})
	for _, node := range d.nodes {
		if node.Kind != KindListItem {
			continue
		}
		lineRange := node.ListItemSource.LineRange
		if lineRange.Start >= range_.Start && lineRange.End <= range_.End {
			ids[node.ID] = struct{}{}
		}
	}
	return ids
}

func (d *Document) promotedListItemCount() int {
	count := 0
	for _, node := range d.nodes {
		if node.Kind == KindListItem {
			count++
		}
	}
	return count
}
