# Milestone M43 — Table-Row Header Cell Identities

Status: green.

## Goal

Complete table-local read navigation from any promoted M35 body row to the promoted M5 non-empty header cells of its owning GFM table, without promoting a public `Table`, synthesizing empty cell identities, or adding a persistent table-ID index.

M43 is read-only navigation. It does not add header mutation, delimiter-row mutation, column operations, table rendering, or cross-table movement.

## Public contract

M43 adds:

```go
func (d *Document) TableRowHeaderCellIDs(rowID NodeID) ([]NodeID, bool)
```

For a valid promoted body row, the method returns the promoted non-empty header-cell identities of the same table in source/column order. Empty semantic header columns are omitted under the existing M5 identity rule; if every header cell is empty, the method returns an empty slice with `true`.

The returned slice is caller-owned. Header `TableCell` details retain `Header()==true` and continue to return no body `RowID()`.

Invalid, missing, zero, nil-document, wrong-kind, or internally inconsistent targets return `(nil,false)`.

## Parser and internal model

Goldmark already provides the semantic table/header/body hierarchy. M43 makes that relationship explicit in Marksplice-owned parser observations: `observeTableCell` derives the containing public Goldmark table node through the AST parent hierarchy and records its byte anchor as the existing private parser-independent `TableAnchor`. No Goldmark type or table identity escapes the adapter.

The refactored M39/M40 table-model builder then:

1. collects promoted body rows and their private table anchors;
2. resolves body-cell row ownership and validates body/header column order;
3. assigns one flat header-cell adjacency block per table that has promoted body rows;
4. stores only scalar header start/count metadata on each promoted row.

Rows in the same table share the same flat header block. Temporary maps, counts, starts, and fill cursors are discarded after parsing. M43 therefore adds one persistent promoted-header `[]NodeID` array, not duplicated per-row slices or a persistent table-anchor map. A later post-M43 whole-codebase consolidation additionally retains the builder's compact source-ordered body-row and editable-cell node-index slices; these arrays carry no table-anchor relationship and let mutation candidate/survivor validation operate on the table family without filtering every document node.

The public/internal accessor shares bounds/lookup plumbing with M40, revalidates every returned node through the authoritative node index, and requires header kind, no body-row identity, matching private table ownership, valid column bounds, and strictly increasing columns.

## Complexity

Let `n` be the already materialized promoted-node sequence, `r` promoted body rows, `c` promoted editable table cells, and `h` promoted non-empty header cells. The refactored builder remains linear: O(n+r+c) construction, O(r+h) temporary adjacency work, and O(r+c+h) persistent compact table-family indexes/adjacency. One header lookup is O(k) time and O(k) returned memory for `k` promoted header cells, with no table/document rescan; table mutation survivor validation iterates the retained row/cell families rather than all structural nodes.

The final table-model refactor split row collection, cell membership, range assignment, adjacency fill, and accessor validation into focused helpers. The project's production complexity analyzer no longer reports any `table_row_model.go` function among its highest-complexity entries.

## Devil's advocate review

### Risk: header cells leak across adjacent tables

Mitigation: table membership comes from the semantic table parent anchor, not row-source proximity. Both internal and public tests use two tables and require distinct header adjacency.

### Risk: parser-specific table objects leak into the public model

Mitigation: the Goldmark adapter exports only the existing integer `TableAnchor` observation. Public API exposes only existing `NodeID` values and Marksplice-owned `TableCell`/`TableRow` details.

### Risk: empty header cells force synthetic identities or break cardinality

Mitigation: M43 follows the M5/M40 distinction between semantic columns and promoted non-empty cell identities. Mixed-empty headers preserve source column numbers, and an all-empty header returns an empty list with `true`.

### Risk: corrupted adjacency returns reordered or foreign cells

Mitigation: bounds are checked before slicing; every ID is resolved authoritatively and validated for header state, same-table ownership, column bounds, and strict order. Injected reversed adjacency fails closed.

## TDD evidence

The initial focused public test failed to compile because `TableRowHeaderCellIDs` did not exist. Current tests cover mixed-empty headers, source/column order, defensive copies, two-table isolation, nil/zero targets, all-empty headers, preserved header `RowID()` behavior, parser table-anchor propagation, mismatched table membership, and injected reversed adjacency. Focused public/internal/parser regressions and a repository-wide `go test ./...` run are green after the final refactor.

Final repository-wide verification on the exact post-refactor M42/M43 tree passes five consecutive `go test ./...` runs, `go test -race ./...`, coverage, `go vet ./...`, `go build ./...`, generated package/`TableRow`/header-accessor documentation, `staticcheck ./...`, standard `golangci-lint run` with zero issues, and both test-inclusive and production-only `gocyclo`/`unparam` with zero issues. The pinned published-GFM 0.29 conformance gate passes; `govulncheck ./...` reports no vulnerabilities and Gitleaks reports no leaks. Final statement coverage is 91.1% for the public root package, 64.1% for `internal/parser/goldmark`, 79.2% for `internal/source`, and 65.6% for `internal/splice`.

M43 is green.
