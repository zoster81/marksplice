# Milestone M40 — Table-Row Cell Identities

Status: green.

## Goal

Complete downward navigation for the supported M35 table body-row model by exposing the promoted non-empty cells owned by one row in physical/source order, without making `TableRow` non-comparable or creating a persistent row-ID map.

## Public contract

M40 adds:

```go
func (d *Document) TableRowCellIDs(rowID NodeID) ([]NodeID, bool)
```

For a valid promoted body row, the method returns the IDs of its promoted M5 non-empty body cells in source/column order. A row whose semantic columns are all empty returns an empty slice with `true`.

The returned count is intentionally allowed to be smaller than `TableRow.ColumnCount()`: M35 column count describes every semantic/source-proven table column, whereas M5 assigns public cell identities only to non-empty cells. M40 does not synthesize cell nodes for empty columns.

Invalid, missing, zero, nil-document, or non-row targets return `(nil,false)`. The returned slice is caller-owned.

## Internal model

The shared M39/M40 parse-time resolver builds one compact flat `[]NodeID` adjacency array for promoted body cells. Each promoted row stores scalar `TableRowCellStart` and `TableRowCellCount` offsets. A temporary row-anchor-to-ordinal map, per-row counts, prefix starts, and fill cursors are discarded after parsing.

The resolver validates:

- promoted row anchors are non-negative, unique, equal to their physical `LineRange.Start`, and strictly increasing in source order;
- promoted body-cell columns are valid and strictly increasing inside the row;
- mapped cell source remains inside the owning physical row;
- the flat adjacency is filled exactly once for each resolved promoted body cell.

The accessor revalidates bounds, child existence/kind/editability, body-row identity, and strict column order so corrupt internal adjacency fails closed instead of returning plausible data.

## Complexity

Let `r` be promoted body rows and `c` promoted body cells. The shared resolver is O(r+c) over the already materialized node sequence with O(r+c) temporary work and O(c) persistent adjacency. One lookup is O(k) for `k` returned promoted cells and O(k) returned memory.

## Devil's advocate review

### Risk: empty cells make the child count appear inconsistent

Mitigation: public documentation explicitly distinguishes semantic `ColumnCount()` from promoted-cell identities; empty cells are omitted, not synthesized.

### Risk: a slice field makes `TableRow` non-comparable

Mitigation: adjacency storage belongs to `Document`; `TableRow` remains an immutable comparable scalar value.

### Risk: corrupted adjacency exposes another row's cell

Mitigation: the accessor resolves every stored cell ID through the authoritative node index and requires `TableRowID == row.ID` plus strictly increasing columns.

## TDD evidence

Tests cover non-empty cells surrounding an empty middle column, all-empty body rows, source order, defensive-copy behavior, nil/zero/wrong-kind targets, and injected out-of-bounds, missing-ID, cross-row, and reversed-order corruption.

M40 is green.