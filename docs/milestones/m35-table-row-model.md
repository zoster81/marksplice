# Milestone M35 — Table-Row Model

Status: green.

## Goal

Promote complete GFM table body rows as source-mapped public nodes by reusing the M5 physical-row mapping that already proves table-cell boundaries.

M35 must not promote the header row or delimiter row as mutable rows, renumber any existing public/internal node kind, introduce a second table parser, or rescan one physical row once per cell and once again per row.

## Public contract

M35 adds `KindTableRow`, immutable typed detail `TableRow`, and `Document.TableRow`.

```go
func (r TableRow) ID() NodeID
func (r TableRow) Range() Range
func (r TableRow) ColumnCount() int
func (d *Document) TableRow(id NodeID) (TableRow, bool)
```

`Range()` owns exactly one complete physical GFM body row. It includes that row's existing LF, CRLF, or CR line terminator when present and ends exactly at EOF for an unterminated final row. No terminator is synthesized.

`ColumnCount()` is exposed only after the Goldmark semantic body-row cell count agrees with the Marksplice physical row mapper's cell count. A mismatch keeps the observation non-editable and therefore outside the public surface.

The header remains available through existing M5 `TableCell` details with `Header() == true`; neither the header nor the GFM delimiter row is a `TableRow` target.

## Parser and source mapping

The Goldmark adapter observes `extension/ast.TableRow` and emits only Marksplice-owned metadata: the physical row anchor, a private table anchor, and semantic column count. The parent table identity is never exposed as a Goldmark object or public API.

`source.MapTableRow` now reports both its historical content-only physical `Range` and `LineRange`, the complete line ownership span. Existing table-cell mapping continues to use the same function.

`internal/splice` factors one `mapTableRowSource` cache lookup used by both row and cell promotion. A physical row is therefore scanned at most once during one document parse even when it contains many cells and is itself promoted.

## Compatibility

`KindTableRow` is appended to the existing public, parser, and internal kind enumerations rather than inserted between established constants. Existing kind ordinals and internal kind values used by snapshot ID derivation remain unchanged.

Existing table-cell IDs, ranges, and mutation contracts are unchanged.

## Complexity

Let `w` be the byte width of a physical row and `c` its cell count. The row mapper remains O(w + c), shared by row and cell mapping through the parse-local row cache. M35 adds O(1) row metadata per promoted row and no persistent second table index.

## Devil's advocate review

### Risk: row promotion duplicates the M5 row scan

Mitigation: row and cell mapping share the existing parse-local `MapTableRow` cache through one factored lookup helper.

### Risk: semantic and lexical cell counts disagree

Mitigation: public row promotion requires exact semantic/source-mapped column-count agreement and otherwise fails closed.

### Risk: adding a kind renumbers established constants or node IDs

Mitigation: the new parser/internal/public kind is appended rather than inserted. Existing constants and snapshot-ID inputs keep their previous numeric values.

### Risk: header/delimiter edits destroy table grammar

Mitigation: only Goldmark body `TableRow` nodes are observed for M35. Header cells retain M5 support; the delimiter row is not promoted.

## TDD evidence and exit decision

The first public tests failed to compile because `KindTableRow`, `TableRow`, and `Document.TableRow` did not exist. The implementation now passes focused and repository-wide regressions for CRLF body-row ownership, exact row bytes, column count, zero/missing targets, EOF line ownership, existing table-cell behavior, and deterministic zero values.

M35 is green.