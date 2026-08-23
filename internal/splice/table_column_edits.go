package splice

import (
	"slices"

	"github.com/zoster81/marksplice/internal/source"
)

// PrepareInsertTableColumn prepares source-preserving insertion of one complete GFM table column.
func (d *Document) PrepareInsertTableColumn(id NodeID, column int, header []byte, alignment TableAlignment, body [][]byte) (ChangeSet, error) {
	target, err := d.tableTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	lexicalAlignment, ok := sourceTableDelimiterAlignment(alignment)
	if !validTableColumnInsertionRequest(target, column, ok, len(body)) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	rows, err := d.completeTableRows(target)
	if err != nil {
		return ChangeSet{}, err
	}

	templateColumn := column
	if templateColumn == target.TableColumnCount {
		templateColumn--
	}
	delimiterContent, err := source.TableDelimiterAlignmentReplacement(d.source, rows[1].Cells[templateColumn].ContentRange, lexicalAlignment)
	if err != nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	expectedContents := tableColumnInsertionContents(header, delimiterContent, body)
	patches, transforms, totalInserted, err := d.tableColumnInsertionPatches(rows, column, expectedContents)
	if err != nil {
		return ChangeSet{}, err
	}

	change, candidate, err := d.prepareCandidateChanges(patches, "table column insertion")
	if err != nil {
		return ChangeSet{}, err
	}
	expectedAlignments := insertTableAlignment(target.TableAlignments, column, alignment)
	candidateTable, err := d.validateTableColumnMutationCandidate(
		candidate, target, transforms, target.TableColumnCount+1,
		target.TableSource.Range.End+totalInserted, expectedAlignments,
	)
	if err != nil || !candidateInsertedTableColumn(candidate, candidateTable, column, expectedContents) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return change, nil
}

// PrepareRemoveTableColumn prepares source-preserving removal of one complete GFM table column.
func (d *Document) PrepareRemoveTableColumn(id NodeID, column int) (ChangeSet, error) {
	target, err := d.tableTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	if target.TableColumnCount <= 1 || column < 0 || column >= target.TableColumnCount {
		return ChangeSet{}, ErrInvalidReplacement
	}
	rows, err := d.completeTableRows(target)
	if err != nil {
		return ChangeSet{}, err
	}

	patches := make([]source.Patch, 0, len(rows))
	transforms := make([]patchTransform, 0, len(rows))
	totalRemoved := 0
	for _, row := range rows {
		removeRange, err := source.TableColumnRemovalRange(row, column)
		if err != nil {
			return ChangeSet{}, ErrInvalidReplacement
		}
		patches = append(patches, source.Patch{Range: removeRange})
		transforms = append(transforms, patchTransform{Range: removeRange})
		totalRemoved += removeRange.End - removeRange.Start
	}

	change, candidate, err := d.prepareCandidateChanges(patches, "table column removal")
	if err != nil {
		return ChangeSet{}, err
	}
	expectedAlignments := removeTableAlignment(target.TableAlignments, column)
	if _, err := d.validateTableColumnMutationCandidate(
		candidate, target, transforms, target.TableColumnCount-1,
		target.TableSource.Range.End-totalRemoved, expectedAlignments,
	); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareMoveTableColumn prepares moving one complete GFM table column to a new zero-based position.
func (d *Document) PrepareMoveTableColumn(id NodeID, from, to int) (ChangeSet, error) {
	target, err := d.tableTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	if from < 0 || from >= target.TableColumnCount || to < 0 || to >= target.TableColumnCount {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if from == to {
		return d.newChanges(nil, "table column move")
	}
	rows, err := d.completeTableRows(target)
	if err != nil {
		return ChangeSet{}, err
	}
	order := tableColumnMoveOrder(target.TableColumnCount, from, to)
	if order == nil {
		return ChangeSet{}, ErrInvalidReplacement
	}

	patches := make([]source.Patch, 0, len(rows))
	transforms := make([]patchTransform, 0, len(rows))
	for _, row := range rows {
		replacement, err := source.ReorderTableRowColumns(d.source, row, order)
		if err != nil {
			return ChangeSet{}, ErrInvalidReplacement
		}
		patches = append(patches, source.Patch{Range: row.Range, Replacement: replacement})
		transforms = append(transforms, patchTransform{Range: row.Range, ReplacementLength: len(replacement)})
	}

	change, candidate, err := d.prepareCandidateChanges(patches, "table column move")
	if err != nil {
		return ChangeSet{}, err
	}
	expectedAlignments := reorderTableAlignments(target.TableAlignments, order)
	if _, err := d.validateTableColumnMutationCandidate(
		candidate, target, transforms, target.TableColumnCount,
		target.TableSource.Range.End, expectedAlignments,
	); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func validTableColumnInsertionRequest(target Node, column int, alignmentOK bool, bodyCount int) bool {
	if column < 0 || column > target.TableColumnCount {
		return false
	}
	if !alignmentOK {
		return false
	}
	return bodyCount == target.TableBodyRowCount
}

func tableColumnInsertionContents(header, delimiter []byte, body [][]byte) [][]byte {
	contents := make([][]byte, len(body)+2)
	contents[0] = append([]byte(nil), header...)
	contents[1] = append([]byte(nil), delimiter...)
	for index := range body {
		contents[index+2] = append([]byte(nil), body[index]...)
	}
	return contents
}

func (d *Document) tableColumnInsertionPatches(rows []source.TableRowMapping, column int, contents [][]byte) ([]source.Patch, []patchTransform, int, error) {
	if len(rows) != len(contents) {
		return nil, nil, 0, ErrInvalidReplacement
	}
	patches := make([]source.Patch, 0, len(rows))
	transforms := make([]patchTransform, 0, len(rows))
	totalInserted := 0
	for index, row := range rows {
		insertRange, replacement, err := source.TableColumnInsertion(d.source, row, column, contents[index])
		if err != nil {
			return nil, nil, 0, ErrInvalidReplacement
		}
		patches = append(patches, source.Patch{Range: insertRange, Replacement: replacement})
		transforms = append(transforms, patchTransform{Range: insertRange, ReplacementLength: len(replacement)})
		totalInserted += len(replacement)
	}
	return patches, transforms, totalInserted, nil
}

func (d *Document) validateTableColumnMutationCandidate(candidate []byte, target Node, transforms []patchTransform, expectedColumnCount, expectedEnd int, expectedAlignments []TableAlignment) (Node, error) {
	candidateDocument, candidateIndex, err := parseTableMutationCandidate(candidate)
	if err != nil {
		return Node{}, err
	}
	policy := tableSurvivorPolicy{tableAnchor: target.TableAnchor, skipRows: true, skipCells: true}
	if err := d.validateOriginalTableModelAfterPatchesWithPolicy(candidate, candidateIndex, transforms, nil, &policy); err != nil {
		return Node{}, err
	}
	candidateTable, ok := promotedTableAtAnchor(candidateDocument, target.TableAnchor)
	if !ok || promotedTableCount(candidateDocument) != promotedTableCount(d) {
		return Node{}, ErrInvalidReplacement
	}
	if candidateTable.TableColumnCount != expectedColumnCount || candidateTable.TableBodyRowCount != target.TableBodyRowCount {
		return Node{}, ErrInvalidReplacement
	}
	expectedRange := Range{Start: target.TableSource.Range.Start, End: expectedEnd}
	if candidateTable.TableSource.Range != expectedRange || !slices.Equal(candidateTable.TableAlignments, expectedAlignments) {
		return Node{}, ErrInvalidReplacement
	}
	if !completeCandidateTableRows(candidate, candidateTable) {
		return Node{}, ErrInvalidReplacement
	}
	return candidateTable, nil
}

func (d *Document) completeTableRows(target Node) ([]source.TableRowMapping, error) {
	rows, err := source.MapCompleteTableRows(d.source, target.TableSource, target.TableColumnCount, target.TableBodyRowCount)
	if err != nil {
		return nil, ErrInvalidReplacement
	}
	return rows, nil
}

func completeCandidateTableRows(candidate []byte, table Node) bool {
	_, err := source.MapCompleteTableRows(candidate, table.TableSource, table.TableColumnCount, table.TableBodyRowCount)
	return err == nil
}

func candidateInsertedTableColumn(candidate []byte, table Node, column int, expectedContents [][]byte) bool {
	rows, err := source.MapCompleteTableRows(candidate, table.TableSource, table.TableColumnCount, table.TableBodyRowCount)
	if err != nil || len(rows) != len(expectedContents) || column < 0 || column >= table.TableColumnCount {
		return false
	}
	for index, row := range rows {
		if column >= len(row.Cells) {
			return false
		}
		content := row.Cells[column].ContentRange
		if !content.Valid(len(candidate)) || !slices.Equal(candidate[content.Start:content.End], expectedContents[index]) {
			return false
		}
	}
	return true
}

func insertTableAlignment(alignments []TableAlignment, column int, alignment TableAlignment) []TableAlignment {
	result := make([]TableAlignment, len(alignments)+1)
	copy(result, alignments[:column])
	result[column] = alignment
	copy(result[column+1:], alignments[column:])
	return result
}

func removeTableAlignment(alignments []TableAlignment, column int) []TableAlignment {
	result := make([]TableAlignment, 0, len(alignments)-1)
	result = append(result, alignments[:column]...)
	result = append(result, alignments[column+1:]...)
	return result
}

func tableColumnMoveOrder(columnCount, from, to int) []int {
	if columnCount <= 0 || from < 0 || from >= columnCount || to < 0 || to >= columnCount {
		return nil
	}
	order := make([]int, columnCount)
	for index := range order {
		order[index] = index
	}
	moved := order[from]
	if from < to {
		copy(order[from:to], order[from+1:to+1])
	} else if from > to {
		copy(order[to+1:from+1], order[to:from])
	}
	order[to] = moved
	return order
}

func reorderTableAlignments(alignments []TableAlignment, order []int) []TableAlignment {
	result := make([]TableAlignment, len(order))
	for index, column := range order {
		result[index] = alignments[column]
	}
	return result
}
