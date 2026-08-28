package splice

import (
	"slices"

	"github.com/zoster81/marksplice/internal/source"
)

func (d *Document) tableTarget(id NodeID) (Node, source.TableMapping, error) {
	target, err := d.editableTargetNode(id, KindTable, "table")
	if err != nil {
		return Node{}, source.TableMapping{}, err
	}
	mapping, ok := remapTableSource(d.source, target)
	if !ok {
		return Node{}, source.TableMapping{}, ErrInvalidReplacement
	}
	return target, mapping, nil
}

// PrepareSetTableColumnAlignment prepares a source-preserving alignment change for one GFM table column.
func (d *Document) PrepareSetTableColumnAlignment(id NodeID, column int, alignment TableAlignment) (ChangeSet, error) {
	target, mapping, err := d.tableTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	if column < 0 || column >= target.TableColumnCount {
		return ChangeSet{}, ErrInvalidReplacement
	}
	expected := append([]TableAlignment(nil), target.TableAlignments...)
	expected[column] = alignment
	return d.prepareSetTableAlignments(target, mapping, expected, "table column alignment")
}

// PrepareSetTableAlignments prepares one atomic source-preserving alignment update for every GFM table column.
func (d *Document) PrepareSetTableAlignments(id NodeID, alignments []TableAlignment) (ChangeSet, error) {
	target, mapping, err := d.tableTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	if len(alignments) != target.TableColumnCount {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return d.prepareSetTableAlignments(target, mapping, append([]TableAlignment(nil), alignments...), "table alignments")
}

type tableAlignmentPatchPlan struct {
	patches    []source.Patch
	transforms []patchTransform
	totalDelta int
}

func (d *Document) prepareSetTableAlignments(target Node, mapping source.TableMapping, expected []TableAlignment, operation string) (ChangeSet, error) {
	plan, err := d.planTableAlignmentPatches(target, mapping, expected)
	if err != nil {
		return ChangeSet{}, err
	}
	if len(plan.patches) == 0 {
		return d.newChanges(nil, operation)
	}
	change, candidate, err := d.prepareCandidateChanges(plan.patches, operation)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateTableAlignmentCandidate(target, expected, candidate, plan); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func (d *Document) planTableAlignmentPatches(target Node, mapping source.TableMapping, expected []TableAlignment) (tableAlignmentPatchPlan, error) {
	if len(expected) != target.TableColumnCount || len(mapping.Delimiter.Cells) != target.TableColumnCount {
		return tableAlignmentPatchPlan{}, ErrInvalidReplacement
	}
	plan := tableAlignmentPatchPlan{
		patches:    make([]source.Patch, 0, target.TableColumnCount),
		transforms: make([]patchTransform, 0, target.TableColumnCount),
	}
	for column, alignment := range expected {
		lexical, ok := sourceTableDelimiterAlignment(alignment)
		if !ok {
			return tableAlignmentPatchPlan{}, ErrInvalidReplacement
		}
		if alignment == target.TableAlignments[column] {
			continue
		}
		delimiterRange := mapping.Delimiter.Cells[column].ContentRange
		replacement, err := source.TableDelimiterAlignmentReplacement(d.source, delimiterRange, lexical)
		if err != nil {
			return tableAlignmentPatchPlan{}, ErrInvalidReplacement
		}
		plan.patches = append(plan.patches, source.Patch{Range: delimiterRange, Replacement: replacement})
		plan.transforms = append(plan.transforms, patchTransform{Range: delimiterRange, ReplacementLength: len(replacement)})
		plan.totalDelta += len(replacement) - (delimiterRange.End - delimiterRange.Start)
	}
	return plan, nil
}

func (d *Document) validateTableAlignmentCandidate(target Node, expected []TableAlignment, candidate []byte, plan tableAlignmentPatchPlan) error {
	candidateDocument, candidateIndex, err := parseTableMutationCandidate(candidate)
	if err != nil {
		return err
	}
	if candidateIndex.rowCount != d.promotedTableRowCount() {
		return ErrInvalidReplacement
	}
	policy := tableSurvivorPolicy{tableAnchor: target.TableAnchor, alignments: expected}
	if err := d.validateOriginalTableModelAfterPatchesWithPolicy(candidate, candidateIndex, plan.transforms, nil, &policy); err != nil {
		return err
	}
	candidateTable, ok := promotedTableAtAnchor(candidateDocument, target.TableAnchor)
	if !ok || promotedTableCount(candidateDocument) != promotedTableCount(d) ||
		candidateTable.TableColumnCount != target.TableColumnCount ||
		candidateTable.TableBodyRowCount != target.TableBodyRowCount ||
		candidateTable.Range != shiftedEnd(target.Range, plan.totalDelta) ||
		!slices.Equal(candidateTable.TableAlignments, expected) {
		return ErrInvalidReplacement
	}
	return nil
}

// PrepareAppendTableRow prepares appending one caller-owned compatible body row to a promoted GFM table.
func (d *Document) PrepareAppendTableRow(id NodeID, fragment []byte) (ChangeSet, error) {
	target, mapping, err := d.tableTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	insertAt := mapping.Range.End
	if len(fragment) == 0 || !tableAppendBoundary(d.source, insertAt) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	insertRange := Range{Start: insertAt, End: insertAt}
	change, candidate, err := d.prepareCandidateChange(insertRange, fragment, "table row append")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateAppendedTableRowCandidate(target, fragment, candidate, insertRange); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func (d *Document) validateAppendedTableRowCandidate(target Node, fragment, candidate []byte, insertRange Range) error {
	candidateDocument, candidateIndex, err := parseTableMutationCandidate(candidate)
	if err != nil {
		return err
	}
	if candidateIndex.rowCount != d.promotedTableRowCount()+1 {
		return ErrInvalidReplacement
	}
	patches := []patchTransform{{Range: insertRange, ReplacementLength: len(fragment)}}
	if err := d.validateOriginalTableModelAfterPatches(candidate, candidateIndex, patches, nil); err != nil {
		return err
	}
	insertAt := insertRange.Start
	if !candidateOwnedTableRow(candidate, candidateIndex, insertAt, fragment, target.TableColumnCount, target.TableAnchor, target.TableAlignments) {
		return ErrInvalidReplacement
	}
	candidateTable, ok := promotedTableAtAnchor(candidateDocument, target.TableAnchor)
	if !ok || promotedTableCount(candidateDocument) != promotedTableCount(d) ||
		candidateTable.TableColumnCount != target.TableColumnCount ||
		candidateTable.TableBodyRowCount != target.TableBodyRowCount+1 ||
		candidateTable.TableLastBodyRowAnchor != insertAt ||
		candidateTable.Range != (Range{Start: target.Range.Start, End: target.Range.End + len(fragment)}) ||
		!slices.Equal(candidateTable.TableAlignments, target.TableAlignments) {
		return ErrInvalidReplacement
	}
	return nil
}

func sourceTableDelimiterAlignment(alignment TableAlignment) (source.TableDelimiterAlignment, bool) {
	switch alignment {
	case TableAlignmentDefault:
		return source.TableDelimiterAlignmentDefault, true
	case TableAlignmentLeft:
		return source.TableDelimiterAlignmentLeft, true
	case TableAlignmentRight:
		return source.TableDelimiterAlignmentRight, true
	case TableAlignmentCenter:
		return source.TableDelimiterAlignmentCenter, true
	default:
		return source.TableDelimiterAlignmentDefault, false
	}
}

func tableAppendBoundary(snapshot []byte, end int) bool {
	if end <= 0 || end > len(snapshot) {
		return false
	}
	return snapshot[end-1] == '\n' || snapshot[end-1] == '\r'
}

func promotedTableAtAnchor(document *Document, anchor int) (Node, bool) {
	if document == nil || anchor < 0 {
		return Node{}, false
	}
	var result Node
	found := false
	for _, node := range document.nodes {
		if node.Kind != KindTable || !node.Editable || node.TableAnchor != anchor {
			continue
		}
		if found {
			return Node{}, false
		}
		result = node
		found = true
	}
	return result, found
}

func promotedTableCount(document *Document) int {
	if document == nil {
		return 0
	}
	count := 0
	for _, node := range document.nodes {
		if node.Kind == KindTable && node.Editable {
			count++
		}
	}
	return count
}
