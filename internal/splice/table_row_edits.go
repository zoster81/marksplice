package splice

import "github.com/zoster81/marksplice/internal/source"

// PrepareReplaceTableRow prepares replacement of one complete promoted GFM table body row.
func (d *Document) PrepareReplaceTableRow(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.tableRowTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	if len(replacement) == 0 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	lineRange := target.TableRowSource.LineRange
	change, candidate, err := d.prepareCandidateChange(lineRange, replacement, "table row replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateIndex, err := parseTableRowMutationCandidate(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if candidateIndex.rowCount != d.promotedTableRowCount() {
		return ChangeSet{}, ErrInvalidReplacement
	}
	patches := []patchTransform{{Range: lineRange, ReplacementLength: len(replacement)}}
	if err := d.validateOriginalTableModelAfterPatches(candidate, candidateIndex, patches, &target); err != nil {
		return ChangeSet{}, err
	}
	if err := validateReplacedTableRow(candidate, candidateIndex, target, replacement, patches); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareRemoveTableRow prepares removal of one complete promoted GFM table body row.
func (d *Document) PrepareRemoveTableRow(id NodeID) (ChangeSet, error) {
	target, err := d.tableRowTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	change, candidate, err := d.prepareCandidateChange(target.TableRowSource.LineRange, nil, "table row removal")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateIndex, err := parseTableRowMutationCandidate(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if candidateIndex.rowCount != d.promotedTableRowCount()-1 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	patches := []patchTransform{{Range: target.TableRowSource.LineRange}}
	if err := d.validateOriginalTableModelAfterPatches(candidate, candidateIndex, patches, &target); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareInsertTableRowBefore prepares insertion of one complete body row before a promoted row in the same table.
func (d *Document) PrepareInsertTableRowBefore(anchorID NodeID, fragment []byte) (ChangeSet, error) {
	return d.prepareInsertTableRow(anchorID, fragment, false)
}

// PrepareInsertTableRowAfter prepares insertion of one complete body row after a promoted row in the same table.
func (d *Document) PrepareInsertTableRowAfter(anchorID NodeID, fragment []byte) (ChangeSet, error) {
	return d.prepareInsertTableRow(anchorID, fragment, true)
}

func (d *Document) prepareInsertTableRow(anchorID NodeID, fragment []byte, after bool) (ChangeSet, error) {
	anchor, err := d.tableRowTarget(anchorID)
	if err != nil {
		return ChangeSet{}, err
	}
	if len(fragment) == 0 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	insertAt := anchor.TableRowSource.LineRange.Start
	operation := "table row insertion before"
	if after {
		insertAt = anchor.TableRowSource.LineRange.End
		operation = "table row insertion after"
	}
	patch := Range{Start: insertAt, End: insertAt}
	change, candidate, err := d.prepareCandidateChange(patch, fragment, operation)
	if err != nil {
		return ChangeSet{}, err
	}
	candidateIndex, err := parseTableRowMutationCandidate(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if candidateIndex.rowCount != d.promotedTableRowCount()+1 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	patches := []patchTransform{{Range: patch, ReplacementLength: len(fragment)}}
	if err := d.validateOriginalTableModelAfterPatches(candidate, candidateIndex, patches, nil); err != nil {
		return ChangeSet{}, err
	}
	if err := validateInsertedTableRow(candidate, candidateIndex, anchor, fragment, insertAt, patches); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareMoveTableRowBefore prepares moving one complete body row before another row in the same table.
func (d *Document) PrepareMoveTableRowBefore(id, anchorID NodeID) (ChangeSet, error) {
	return d.prepareMoveTableRow(id, anchorID, false)
}

// PrepareMoveTableRowAfter prepares moving one complete body row after another row in the same table.
func (d *Document) PrepareMoveTableRowAfter(id, anchorID NodeID) (ChangeSet, error) {
	return d.prepareMoveTableRow(id, anchorID, true)
}

func (d *Document) prepareMoveTableRow(id, anchorID NodeID, after bool) (ChangeSet, error) {
	if id == anchorID {
		return ChangeSet{}, ErrInvalidReplacement
	}
	moved, err := d.tableRowTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	anchor, err := d.tableRowTarget(anchorID)
	if err != nil {
		return ChangeSet{}, err
	}
	if moved.TableAnchor != anchor.TableAnchor {
		return ChangeSet{}, ErrInvalidReplacement
	}
	operation := "table row move before"
	insertAt := anchor.TableRowSource.LineRange.Start
	if after {
		operation = "table row move after"
		insertAt = anchor.TableRowSource.LineRange.End
	}
	if !after && moved.TableRowSource.LineRange.End == anchor.TableRowSource.LineRange.Start ||
		after && anchor.TableRowSource.LineRange.End == moved.TableRowSource.LineRange.Start {
		return d.newChanges(nil, operation)
	}
	movedRange := moved.TableRowSource.LineRange
	if insertAt > movedRange.Start && insertAt < movedRange.End {
		return ChangeSet{}, ErrInvalidReplacement
	}
	fragment := d.source[movedRange.Start:movedRange.End]
	patchesSource := []source.Patch{
		{Range: movedRange},
		{Range: Range{Start: insertAt, End: insertAt}, Replacement: fragment},
	}
	change, candidate, err := d.prepareCandidateChanges(patchesSource, operation)
	if err != nil {
		return ChangeSet{}, err
	}
	candidateIndex, err := parseTableRowMutationCandidate(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if candidateIndex.rowCount != d.promotedTableRowCount() {
		return ChangeSet{}, ErrInvalidReplacement
	}
	patches := []patchTransform{
		{Range: movedRange},
		{Range: Range{Start: insertAt, End: insertAt}, ReplacementLength: len(fragment)},
	}
	if err := d.validateOriginalTableModelAfterPatches(candidate, candidateIndex, patches, &moved); err != nil {
		return ChangeSet{}, err
	}
	movedOffset, ok := movedRangeCandidateOffset(movedRange, insertAt)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if err := d.validateMovedTableRow(candidate, candidateIndex, moved, anchor, movedOffset, patches); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}
