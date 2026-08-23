package splice

import (
	"bytes"
	"fmt"
	"slices"
)

func (d *Document) tableRowTarget(id NodeID) (Node, error) {
	return d.editableTargetNode(id, KindTableRow, "table row")
}

type tableMutationIndex struct {
	rowsByLineStart     map[int]Node
	cellsByContentStart map[int]Node
	rowCount            int
}

func parseTableMutationCandidate(candidate []byte) (*Document, tableMutationIndex, error) {
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return nil, tableMutationIndex{}, fmt.Errorf("%w: table mutation candidate parse: %v", ErrInvalidReplacement, err)
	}
	index, ok := indexTableMutationModel(candidateDocument)
	if !ok {
		return nil, tableMutationIndex{}, ErrInvalidReplacement
	}
	return candidateDocument, index, nil
}

func parseTableRowMutationCandidate(candidate []byte) (tableMutationIndex, error) {
	_, index, err := parseTableMutationCandidate(candidate)
	return index, err
}

func (d *Document) promotedTableRowCount() int {
	if d == nil {
		return 0
	}
	return len(d.tableRowIndexes)
}

func indexTableMutationModel(document *Document) (tableMutationIndex, bool) {
	if document == nil {
		return tableMutationIndex{}, false
	}
	index := tableMutationIndex{
		rowsByLineStart:     make(map[int]Node, len(document.tableRowIndexes)),
		cellsByContentStart: make(map[int]Node, len(document.tableCellIndexes)),
		rowCount:            len(document.tableRowIndexes),
	}
	for _, nodeIndex := range document.tableRowIndexes {
		node, ok := document.indexedEditableNode(nodeIndex, KindTableRow)
		if !ok {
			return tableMutationIndex{}, false
		}
		start := node.TableRowSource.LineRange.Start
		if _, exists := index.rowsByLineStart[start]; exists {
			return tableMutationIndex{}, false
		}
		index.rowsByLineStart[start] = node
	}
	for _, nodeIndex := range document.tableCellIndexes {
		node, ok := document.indexedEditableNode(nodeIndex, KindTableCell)
		if !ok {
			return tableMutationIndex{}, false
		}
		start := node.TableCellSource.ContentRange.Start
		if _, exists := index.cellsByContentStart[start]; exists {
			return tableMutationIndex{}, false
		}
		index.cellsByContentStart[start] = node
	}
	return index, true
}

func anchorAfterPatches(anchor int, patches []patchTransform) (int, bool) {
	ordered, ok := orderedPatchTransforms(patches)
	if !ok {
		return 0, false
	}
	return anchorAfterOrderedPatches(anchor, ordered)
}

func anchorAfterOrderedPatches(anchor int, ordered []patchTransform) (int, bool) {
	if anchor < 0 {
		return 0, false
	}
	transformed, ok := rangeAfterOrderedPatches(Range{Start: anchor, End: anchor + 1}, ordered)
	return transformed.Start, ok
}

func sameTableRowMapping(candidate []byte, originalSource []byte, original, mapped Node, expectedLine, expectedRange Range, expectedTableAnchor int, expectedAlignments []TableAlignment) bool {
	if !expectedLine.Valid(len(candidate)) || !expectedRange.Valid(len(candidate)) || !original.TableRowSource.LineRange.Valid(len(originalSource)) {
		return false
	}
	return mapped.TableRowSource.LineRange == expectedLine &&
		mapped.TableRowSource.Range == expectedRange &&
		mapped.TableAnchor == expectedTableAnchor &&
		mapped.TableColumnCount == original.TableColumnCount &&
		slices.Equal(mapped.TableAlignments, expectedAlignments) &&
		bytes.Equal(originalSource[original.TableRowSource.LineRange.Start:original.TableRowSource.LineRange.End], candidate[expectedLine.Start:expectedLine.End])
}

type tableSurvivorPolicy struct {
	tableAnchor int
	alignments  []TableAlignment
	skipRows    bool
	skipCells   bool
}

func (d *Document) validateOriginalTableModelAfterPatches(candidate []byte, candidateIndex tableMutationIndex, patches []patchTransform, skipRow *Node) error {
	return d.validateOriginalTableModelAfterPatchesWithPolicy(candidate, candidateIndex, patches, skipRow, nil)
}

func (d *Document) validateOriginalTableModelAfterPatchesWithPolicy(candidate []byte, candidateIndex tableMutationIndex, patches []patchTransform, skipRow *Node, policy *tableSurvivorPolicy) error {
	ordered, ok := orderedPatchTransforms(patches)
	if !ok {
		return ErrInvalidReplacement
	}
	if err := d.validateOriginalTableRowsAfterPatches(candidate, candidateIndex, ordered, skipRow, policy); err != nil {
		return err
	}
	return d.validateOriginalTableCellsAfterPatches(candidate, candidateIndex, ordered, skipRow, policy)
}

func (d *Document) validateOriginalTableRowsAfterPatches(candidate []byte, candidateIndex tableMutationIndex, ordered []patchTransform, skipRow *Node, policy *tableSurvivorPolicy) error {
	for _, nodeIndex := range d.tableRowIndexes {
		original, ok := d.indexedEditableNode(nodeIndex, KindTableRow)
		if !ok {
			return ErrInvalidReplacement
		}
		if skipRow != nil && original.ID == skipRow.ID {
			continue
		}
		expectedAlignments, skip := tableRowSurvivorPolicy(original, policy)
		if skip {
			continue
		}
		if !d.originalTableRowSurvives(candidate, candidateIndex, original, ordered, expectedAlignments) {
			return ErrInvalidReplacement
		}
	}
	return nil
}

func tableRowSurvivorPolicy(original Node, policy *tableSurvivorPolicy) ([]TableAlignment, bool) {
	if policy == nil || original.TableAnchor != policy.tableAnchor {
		return original.TableAlignments, false
	}
	if policy.skipRows {
		return nil, true
	}
	if policy.alignments != nil {
		return policy.alignments, false
	}
	return original.TableAlignments, false
}

func (d *Document) validateOriginalTableCellsAfterPatches(candidate []byte, candidateIndex tableMutationIndex, ordered []patchTransform, skipRow *Node, policy *tableSurvivorPolicy) error {
	for _, nodeIndex := range d.tableCellIndexes {
		original, ok := d.indexedEditableNode(nodeIndex, KindTableCell)
		if !ok {
			return ErrInvalidReplacement
		}
		if skipRow != nil && tableCellInsideRow(original, *skipRow) {
			continue
		}
		if policy != nil && policy.skipCells && original.TableAnchor == policy.tableAnchor {
			continue
		}
		if !d.originalTableCellSurvives(candidate, candidateIndex, original, ordered) {
			return ErrInvalidReplacement
		}
	}
	return nil
}

func (d *Document) originalTableRowSurvives(candidate []byte, candidateIndex tableMutationIndex, original Node, ordered []patchTransform, expectedAlignments []TableAlignment) bool {
	expectedLine, ok := rangeAfterOrderedPatches(original.TableRowSource.LineRange, ordered)
	if !ok {
		return false
	}
	expectedRange, ok := rangeAfterOrderedPatches(original.TableRowSource.Range, ordered)
	if !ok {
		return false
	}
	expectedTableAnchor, ok := anchorAfterOrderedPatches(original.TableAnchor, ordered)
	if !ok {
		return false
	}
	mapped, ok := candidateIndex.rowsByLineStart[expectedLine.Start]
	return ok && sameTableRowMapping(candidate, d.source, original, mapped, expectedLine, expectedRange, expectedTableAnchor, expectedAlignments)
}

func tableCellInsideRow(cell, row Node) bool {
	start := cell.TableCellSource.Range.Start
	return start >= row.TableRowSource.LineRange.Start && start < row.TableRowSource.LineRange.End
}

func (d *Document) originalTableCellSurvives(candidate []byte, candidateIndex tableMutationIndex, original Node, ordered []patchTransform) bool {
	expectedRaw, ok := rangeAfterOrderedPatches(original.TableCellSource.Range, ordered)
	if !ok {
		return false
	}
	expectedContent, ok := rangeAfterOrderedPatches(original.TableCellSource.ContentRange, ordered)
	if !ok || !expectedRaw.Valid(len(candidate)) || !expectedContent.Valid(len(candidate)) || !original.TableCellSource.Range.Valid(len(d.source)) {
		return false
	}
	mapped, ok := candidateIndex.cellsByContentStart[expectedContent.Start]
	return ok && mapped.TableHeader == original.TableHeader && mapped.TableColumn == original.TableColumn &&
		mapped.TableCellSource.Range == expectedRaw && mapped.TableCellSource.ContentRange == expectedContent &&
		bytes.Equal(d.source[original.TableCellSource.Range.Start:original.TableCellSource.Range.End], candidate[expectedRaw.Start:expectedRaw.End])
}

func candidateOwnedTableRow(candidate []byte, candidateIndex tableMutationIndex, start int, fragment []byte, columnCount, tableAnchor int, alignments []TableAlignment) bool {
	if len(fragment) == 0 || start < 0 || start > len(candidate) || len(fragment) > len(candidate)-start {
		return false
	}
	row, ok := candidateIndex.rowsByLineStart[start]
	if !ok || row.TableRowSource.LineRange != (Range{Start: start, End: start + len(fragment)}) || row.TableColumnCount != columnCount || row.TableAnchor != tableAnchor || !slices.Equal(row.TableAlignments, alignments) {
		return false
	}
	return bytes.Equal(candidate[start:start+len(fragment)], fragment)
}

func validateReplacedTableRow(candidate []byte, candidateIndex tableMutationIndex, original Node, replacement []byte, patches []patchTransform) error {
	expectedTableAnchor, ok := anchorAfterPatches(original.TableAnchor, patches)
	if !ok {
		return ErrInvalidReplacement
	}
	if !candidateOwnedTableRow(candidate, candidateIndex, original.TableRowSource.LineRange.Start, replacement, original.TableColumnCount, expectedTableAnchor, original.TableAlignments) {
		return ErrInvalidReplacement
	}
	return nil
}

func candidateRowForOriginal(index tableMutationIndex, original Node, patches []patchTransform) (Node, bool) {
	expectedLine, ok := rangeAfterPatches(original.TableRowSource.LineRange, patches)
	if !ok {
		return Node{}, false
	}
	row, ok := index.rowsByLineStart[expectedLine.Start]
	return row, ok && row.TableRowSource.LineRange == expectedLine
}

func validateInsertedTableRow(candidate []byte, candidateIndex tableMutationIndex, anchor Node, fragment []byte, insertAt int, patches []patchTransform) error {
	candidateAnchor, ok := candidateRowForOriginal(candidateIndex, anchor, patches)
	if !ok {
		return ErrInvalidReplacement
	}
	if !candidateOwnedTableRow(candidate, candidateIndex, insertAt, fragment, anchor.TableColumnCount, candidateAnchor.TableAnchor, candidateAnchor.TableAlignments) {
		return ErrInvalidReplacement
	}
	return nil
}

func (d *Document) validateMovedTableRow(candidate []byte, candidateIndex tableMutationIndex, moved, anchor Node, movedOffset int, patches []patchTransform) error {
	originalLine := moved.TableRowSource.LineRange
	if !originalLine.Valid(len(d.source)) || originalLine.Start == originalLine.End {
		return ErrInvalidReplacement
	}
	candidateAnchor, ok := candidateRowForOriginal(candidateIndex, anchor, patches)
	if !ok {
		return ErrInvalidReplacement
	}
	fragment := d.source[originalLine.Start:originalLine.End]
	if !candidateOwnedTableRow(candidate, candidateIndex, movedOffset, fragment, moved.TableColumnCount, candidateAnchor.TableAnchor, candidateAnchor.TableAlignments) {
		return ErrInvalidReplacement
	}
	return nil
}
