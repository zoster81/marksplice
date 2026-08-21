# Milestone M39 — Table-Cell Row Identity

Status: green.

## Goal

Expose the snapshot-scoped promoted body-row identity that owns one promoted GFM table body cell, reusing the M35 physical-row anchor rather than introducing a public table identity or recomputing row membership on every accessor.

## Public contract

M39 extends the comparable `TableCell` value with:

```go
func (c TableCell) RowID() (NodeID, bool)
```

For a promoted non-empty body cell whose M35 body row is also promoted, `RowID` returns that row's existing `NodeID`. Header cells return the zero ID and `false` because M35 intentionally does not promote the header as a `TableRow`. No synthetic row identity is created for unsupported/unpromoted row shapes.

The new field is a scalar `NodeID`, so the established public `TableCell` value remains comparable.

## Parse-time resolution

M5 already gives each mapped cell a parser-independent physical `TableRowAnchor`; M35 gives each promoted body row the same physical anchor and a real snapshot ID. After ordinary node IDs exist, M39 resolves body cells through a temporary `rowAnchor -> compact row ordinal` map and stores only the resolved `TableRowID` on the cell node.

This happens before the immutable node index is published and does not participate in node-ID derivation. Existing cell and row IDs are therefore unchanged.

A body cell whose row anchor does not resolve to a promoted body row remains a normal mapped cell with no public row identity. Header cells are deliberately excluded from resolution.

## Complexity

Resolution is part of the shared M39/M40 linear table-row model pass. It adds O(r) temporary row-anchor lookup state for `r` promoted body rows and O(1) persistent scalar identity per promoted body cell. `RowID()` is O(1).

## Devil's advocate review

### Risk: header cells receive a fake body-row identity

Mitigation: header cells are excluded from the resolver and always expose `(zero,false)`.

### Risk: row identity changes existing node IDs

Mitigation: row/cell IDs are derived before the relationship pass; `TableRowID` is post-ID metadata only.

### Risk: an unsupported body-row shape makes document parsing fail

Mitigation: an otherwise mapped body cell whose row anchor is not promoted remains unresolved rather than synthesizing an identity. Corrupt relations to a promoted row, however, fail closed.

## TDD evidence

The first public test failed to compile because `TableCell.RowID` did not exist. Current tests cover header/body distinction, empty cells, multiple rows, zero values, and exact identity equality with the corresponding `TableRow.ID()`.

M39 is green.