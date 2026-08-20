package splice

import (
	"bytes"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

// PrepareRemoveListItem prepares removal of one complete physical leaf list-item line.
func (d *Document) PrepareRemoveListItem(id NodeID) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindListItem, "list item")
	if err != nil {
		return ChangeSet{}, err
	}
	lineRange := target.ListItemSource.LineRange
	change, candidate, err := d.prepareCandidateChange(lineRange, nil, "list item removal")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateItems, err := candidateListItemMappings(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateOriginalListItemsAfterPatch(candidate, candidateItems, lineRange, 0, target.ID); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareInsertListItemBefore prepares insertion of one same-shape leaf sibling immediately before the anchor item.
func (d *Document) PrepareInsertListItemBefore(id NodeID, fragment []byte) (ChangeSet, error) {
	return d.prepareInsertListItem(id, fragment, false)
}

// PrepareInsertListItemAfter prepares insertion of one same-shape leaf sibling immediately after the anchor item.
func (d *Document) PrepareInsertListItemAfter(id NodeID, fragment []byte) (ChangeSet, error) {
	return d.prepareInsertListItem(id, fragment, true)
}

func (d *Document) prepareInsertListItem(id NodeID, fragment []byte, after bool) (ChangeSet, error) {
	anchor, err := d.editableTargetNode(id, KindListItem, "list item")
	if err != nil {
		return ChangeSet{}, err
	}
	fragmentMapping, err := d.parseListItemFragment(fragment, anchor.ListItemSource)
	if err != nil {
		return ChangeSet{}, err
	}

	insertAt := anchor.ListItemSource.LineRange.Start
	operation := "list item insertion before"
	if after {
		insertAt = anchor.ListItemSource.LineRange.End
		operation = "list item insertion after"
	}
	patch := Range{Start: insertAt, End: insertAt}
	change, candidate, err := d.prepareCandidateChange(patch, fragment, operation)
	if err != nil {
		return ChangeSet{}, err
	}
	candidateItems, err := candidateListItemMappings(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if len(candidateItems) != d.promotedListItemCount()+1 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if err := d.validateOriginalListItemsAfterPatch(candidate, candidateItems, patch, len(fragment), ""); err != nil {
		return ChangeSet{}, err
	}
	if err := validateInsertedListItemFragment(candidate, candidateItems, fragment, fragmentMapping, insertAt); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareMoveListItemBefore prepares moving one complete leaf list-item line immediately before a same-shape anchor item.
func (d *Document) PrepareMoveListItemBefore(id, anchorID NodeID) (ChangeSet, error) {
	return d.prepareMoveListItem(id, anchorID, false)
}

// PrepareMoveListItemAfter prepares moving one complete leaf list-item line immediately after a same-shape anchor item.
func (d *Document) PrepareMoveListItemAfter(id, anchorID NodeID) (ChangeSet, error) {
	return d.prepareMoveListItem(id, anchorID, true)
}

func (d *Document) prepareMoveListItem(id, anchorID NodeID, after bool) (ChangeSet, error) {
	if id == anchorID {
		return ChangeSet{}, ErrInvalidReplacement
	}
	moved, err := d.editableTargetNode(id, KindListItem, "list item")
	if err != nil {
		return ChangeSet{}, err
	}
	anchor, err := d.editableTargetNode(anchorID, KindListItem, "list item")
	if err != nil {
		return ChangeSet{}, err
	}

	fragment := d.source[moved.ListItemSource.LineRange.Start:moved.ListItemSource.LineRange.End]
	fragmentMapping, err := d.parseListItemFragment(fragment, anchor.ListItemSource)
	if err != nil {
		return ChangeSet{}, err
	}

	insertAt := anchor.ListItemSource.LineRange.Start
	operation := "list item move before"
	alreadyPlaced := moved.ListItemSource.LineRange.End == anchor.ListItemSource.LineRange.Start
	if after {
		insertAt = anchor.ListItemSource.LineRange.End
		operation = "list item move after"
		alreadyPlaced = anchor.ListItemSource.LineRange.End == moved.ListItemSource.LineRange.Start
	}
	if alreadyPlaced {
		return d.newChanges(nil, operation)
	}

	movedRange := moved.ListItemSource.LineRange
	movedOffset, ok := movedRangeCandidateOffset(movedRange, insertAt)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	insertRange := Range{Start: insertAt, End: insertAt}
	patches := []source.Patch{
		{Range: movedRange},
		{Range: insertRange, Replacement: fragment},
	}
	transforms := []patchTransform{
		{Range: movedRange},
		{Range: insertRange, ReplacementLength: len(fragment)},
	}
	change, candidate, err := d.prepareCandidateChanges(patches, operation)
	if err != nil {
		return ChangeSet{}, err
	}
	candidateItems, err := candidateListItemMappings(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateOriginalListItemsAfterPatches(candidate, candidateItems, transforms, moved.ID); err != nil {
		return ChangeSet{}, err
	}
	if err := validateInsertedListItemFragment(candidate, candidateItems, fragment, fragmentMapping, movedOffset); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func (d *Document) parseListItemFragment(fragment []byte, anchor source.ListItemMapping) (source.ListItemMapping, error) {
	if len(fragment) == 0 {
		return source.ListItemMapping{}, ErrInvalidReplacement
	}
	items, err := candidateListItemMappings(fragment)
	if err != nil {
		return source.ListItemMapping{}, err
	}
	if len(items) != 1 {
		return source.ListItemMapping{}, ErrInvalidReplacement
	}
	mapping, ok := items[0]
	if !ok || mapping.LineRange != (Range{Start: 0, End: len(fragment)}) ||
		mapping.Ordered != anchor.Ordered || mapping.Marker != anchor.Marker {
		return source.ListItemMapping{}, ErrInvalidReplacement
	}

	anchorPrefix := d.source[anchor.LineRange.Start:anchor.Range.Start]
	fragmentPrefix := fragment[mapping.LineRange.Start:mapping.Range.Start]
	if !bytes.Equal(fragmentPrefix, anchorPrefix) {
		return source.ListItemMapping{}, ErrInvalidReplacement
	}
	return mapping, nil
}

func candidateListItemMappings(candidate []byte) (map[int]source.ListItemMapping, error) {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return nil, err
	}
	items := make(map[int]source.ListItemMapping)
	for _, observation := range observations {
		if observation.Kind != parser.KindListItem {
			continue
		}
		mapping, err := source.MapSingleLineListItem(
			candidate,
			Range{Start: observation.Range.Start, End: observation.Range.End},
			observation.Ordered,
			observation.Marker,
		)
		if err != nil {
			continue
		}
		if _, exists := items[mapping.LineRange.Start]; exists {
			return nil, ErrInvalidReplacement
		}
		items[mapping.LineRange.Start] = mapping
	}
	return items, nil
}

func (d *Document) validateOriginalListItemsAfterPatch(candidate []byte, candidateItems map[int]source.ListItemMapping, patch Range, replacementLength int, skipID NodeID) error {
	return d.validateOriginalListItemsAfterPatches(candidate, candidateItems, []patchTransform{{Range: patch, ReplacementLength: replacementLength}}, skipID)
}

func (d *Document) validateOriginalListItemsAfterPatches(candidate []byte, candidateItems map[int]source.ListItemMapping, patches []patchTransform, skipID NodeID) error {
	for _, originalNode := range d.nodes {
		if originalNode.Kind != KindListItem || originalNode.ID == skipID {
			continue
		}
		original := originalNode.ListItemSource
		expectedLine, ok := rangeAfterPatches(original.LineRange, patches)
		if !ok {
			return ErrInvalidReplacement
		}
		expectedRange, ok := rangeAfterPatches(original.Range, patches)
		if !ok {
			return ErrInvalidReplacement
		}
		expectedContent, ok := rangeAfterPatches(original.ContentRange, patches)
		if !ok {
			return ErrInvalidReplacement
		}

		mapped, ok := candidateItems[expectedLine.Start]
		if !ok || mapped.LineRange != expectedLine || mapped.Range != expectedRange ||
			mapped.ContentRange != expectedContent || mapped.Ordered != original.Ordered || mapped.Marker != original.Marker {
			return ErrInvalidReplacement
		}
		if !bytes.Equal(d.source[original.LineRange.Start:original.LineRange.End], candidate[mapped.LineRange.Start:mapped.LineRange.End]) {
			return ErrInvalidReplacement
		}
	}
	return nil
}

func validateInsertedListItemFragment(candidate []byte, candidateItems map[int]source.ListItemMapping, fragment []byte, fragmentMapping source.ListItemMapping, insertAt int) error {
	mapped, ok := candidateItems[insertAt]
	if !ok || mapped.LineRange != shiftedRange(fragmentMapping.LineRange, insertAt) ||
		mapped.Range != shiftedRange(fragmentMapping.Range, insertAt) ||
		mapped.ContentRange != shiftedRange(fragmentMapping.ContentRange, insertAt) ||
		mapped.Ordered != fragmentMapping.Ordered || mapped.Marker != fragmentMapping.Marker {
		return ErrInvalidReplacement
	}
	if !bytes.Equal(fragment, candidate[insertAt:insertAt+len(fragment)]) {
		return ErrInvalidReplacement
	}
	return nil
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
