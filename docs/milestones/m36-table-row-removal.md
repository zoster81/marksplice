# Milestone M36 — Table-Row Removal

Status: green.

## Goal

Remove exactly one complete promoted GFM table body row while proving that all untouched table rows and cells retain their source mappings after the join.

## Public contract

M36 adds:

```go
func (d *Document) PrepareRemoveTableRow(id NodeID) (ChangeSet, error)
```

The target must be a promoted M35 body row. The patch deletes exactly `TableRow.Range()` and therefore owns the row's existing line terminator when one exists.

Missing IDs retain `ErrNodeNotFound`; wrong node kinds retain `ErrInvalidTargetKind`; structural candidate failures use `ErrInvalidReplacement`; applying the prepared change to a different snapshot retains `ErrSourceConflict`.

## Candidate proof

Removal renders one candidate and parses it once. The candidate must contain exactly one fewer promoted body row.

Every surviving promoted table row must preserve, modulo the single deletion transform:

- complete `LineRange`;
- raw physical row range;
- private table anchor;
- column count;
- exact physical row bytes.

Every surviving mapped table cell, including header cells, must preserve:

- raw cell range;
- content range;
- header/body state;
- zero-based column;
- exact raw cell bytes.

Cells belonging to the removed row are the only cells intentionally skipped.

## Final-body-row behavior

Removing the last body row is allowed only when candidate parsing still proves the table header/cell structure. This specifically prevents a following source line from being silently absorbed into the table or otherwise changing the surviving header mapping.

No blank line, row, or delimiter is synthesized to repair a candidate.

## Complexity

The operation performs one candidate parse and linear validation over mapped table rows/cells. It prepares one source patch. No persistent mutation index is added.

## Devil's advocate review

### Risk: deleting the final body row changes the following line into table content

Mitigation: surviving header cells must reappear with exact transformed mappings and bytes after candidate parsing.

### Risk: line-ending ownership is ambiguous

Mitigation: removal uses the M35 complete physical `LineRange`; tests cover CRLF and final-body-row deletion.

### Risk: untouched cells are reformatted while row bytes look plausible

Mitigation: cell mappings and raw bytes are validated independently of the row-level proof.

## TDD evidence and exit decision

Focused tests prove exact CRLF deletion, final-body-row removal, preserved header cells, invalid-target categories, and stale-source rejection. Existing M5 cell regressions and the full suite pass.

M36 is green.