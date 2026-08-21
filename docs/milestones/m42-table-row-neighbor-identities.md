# Milestone M42 — Table-Row Neighbor Identities

Status: green.

## Goal

Complete horizontal navigation among promoted M35 GFM table body rows without introducing a public `Table` value, a public table identity namespace, cross-table movement, or another persistent lookup index.

M42 exposes only the nearest promoted body row before and after one promoted row in the same table. Unpromoted semantic rows are not synthesized.

## Public contract

M42 extends the existing comparable `TableRow` detail with:

```go
func (r TableRow) PreviousID() (NodeID, bool)
func (r TableRow) NextID() (NodeID, bool)
```

The methods return the nearest promoted body-row identity before or after the row within the same GFM table. The first promoted row has no previous identity, the last has no next identity, and a table with one promoted body row has neither.

A zero `TableRow` returns `(NodeID{},false)` for both methods. M42 adds only scalar `NodeID` fields, so `TableRow` remains comparable.

The identities remain snapshot-scoped under the established `NodeID` contract; M42 does not create table IDs or change any existing kind ordinal or node-ID derivation.

## Internal model

The existing post-ID M39/M40 table-row resolver now also groups promoted body rows by the private parser-independent `TableAnchor` already carried by M35 row observations. During its source-ordered row pass it stores only two scalar links on each promoted row:

- `TablePreviousRowID`;
- `TableNextRowID`.

The temporary table-anchor-to-last-row map is discarded after parsing. No persistent table-anchor map or public table object is added.

The internal accessor resolves every non-empty stored neighbor through the authoritative node index and verifies promoted-row kind/editability, the same private table anchor, source-order direction, and the reciprocal neighbor link. Corrupt non-empty links therefore fail closed.

## Complexity

Neighbor links are constructed in O(r) time for `r` promoted body rows during the already-required table-model build, with O(t) temporary state for `t` tables and O(1) persistent scalar state per promoted row. Public neighbor access is O(1)-expected through the existing node index and performs no document/table scan.

## Devil's advocate review

### Risk: adjacent source rows from different tables are linked

Mitigation: links are created only through the private semantic `TableAnchor`, not source proximity. Public tests place a one-row second table immediately after the first table's row sequence and prove that navigation stops at the first table boundary.

### Risk: adding navigation makes `TableRow` non-comparable

Mitigation: only scalar `NodeID` fields are added. A public test uses `TableRow` as a map key to preserve the established comparable-value contract.

### Risk: a corrupted stored link returns a plausible foreign row

Mitigation: the accessor requires authoritative neighbor lookup, same-table ownership, correct source direction, and a reciprocal link back to the queried row. Injected cross-table corruption returns `false`.

## TDD evidence

The initial focused public test failed to compile because `PreviousID` and `NextID` did not exist. Current focused tests cover first/middle/last rows, a separate one-row table, zero values, comparability, same-table internal link construction, and injected cross-table corruption. Repeated focused table regressions and a repository-wide `go test ./...` run are green after the final resolver/accessor refactor.

Final repository-wide verification on the exact post-refactor M42/M43 tree passes five consecutive `go test ./...` runs, `go test -race ./...`, coverage, `go vet ./...`, `go build ./...`, generated package/`TableRow`/header-accessor documentation, `staticcheck ./...`, standard `golangci-lint run` with zero issues, and both test-inclusive and production-only `gocyclo`/`unparam` with zero issues. The pinned published-GFM 0.29 conformance gate passes; `govulncheck ./...` reports no vulnerabilities and Gitleaks reports no leaks. Final statement coverage is 91.1% for the public root package, 64.1% for `internal/parser/goldmark`, 79.2% for `internal/source`, and 65.6% for `internal/splice`.

M42 is green.
