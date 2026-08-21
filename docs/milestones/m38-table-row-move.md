# Milestone M38 — Table-Row Move and Consolidation

Status: green.

## Goal

Atomically move one complete promoted GFM table body row before or after another body row in the same table, preserving exact moved bytes and then consolidate the M35–M38 implementation for reuse and lower repeated work.

## Public contract

M38 adds:

```go
func (d *Document) PrepareMoveTableRowBefore(id, anchorID NodeID) (ChangeSet, error)
func (d *Document) PrepareMoveTableRowAfter(id, anchorID NodeID) (ChangeSet, error)
```

Source and anchor must be distinct promoted body rows in the same snapshot and the same table. Cross-table movement intentionally fails with `ErrInvalidReplacement` even when column counts happen to match.

## Atomic move

The move owns two coordinated original-coordinate patches:

1. delete the exact source `TableRow.Range()`;
2. insert those exact bytes at the requested anchor boundary.

The candidate is rendered and parsed once. Row count must remain unchanged. All non-moved rows and cells retain their transformed mappings and exact bytes. The moved candidate row must own exactly the moved byte span, preserve column count, reproduce the original line bytes exactly, and resolve to the candidate anchor's table.

Already-satisfied adjacent before/after moves return a zero-patch `ChangeSet`; it remains bound to the exact source fingerprint and rejects stale application.

## Same-table boundary

M38 does not use column count as table identity. The parser adapter records a private table source anchor for M35 body rows. Source and anchor must share that anchor before movement, and candidate source/anchor must share the transformed table anchor after movement.

This intentionally defers cross-table row transfer because alignment/header semantics belong to the destination table and need a separate reviewed contract.

## Final consolidation

The first green M35–M38 implementation rebuilt candidate row maps for survivor, inserted-row, and moved-row validation and independently built a cell map. It also scanned original rows and cells in separate survivor passes.

The final refactor introduces one operation-local `tableMutationIndex` built in one pass over the parsed candidate. It contains only:

- body rows keyed by complete physical-line start;
- mapped cells keyed by content start;
- promoted body-row count.

One `validateOriginalTableModelAfterPatches` pass over the original document validates both surviving rows and cells. Inserted/moved-row proof reuses the same candidate index. The index is temporary and discarded after operation preparation; no persistent table hierarchy/index is added.

M35 mapping also factors `mapTableRowSource` so M35 rows and M5 cells share the same parse-local physical-row cache rather than rescanning.

## Complexity

Let `n` be document size and `t` the number of promoted table rows/cells relevant to validation. A move uses one candidate parse plus O(t) temporary indexing/validation and two source patches. The consolidation removes repeated candidate-map construction and duplicate survivor scans without changing the O(n+t) safety-oracle complexity class.

## Devil's advocate review

### Risk: same-width cross-table movement appears safe but changes semantics

Mitigation: source and anchor must have the same private table anchor; candidate proof repeats same-table membership.

### Risk: move duplicates or drops row bytes

Mitigation: candidate row count is unchanged and the moved candidate row is byte-equal to the exact original `LineRange`.

### Risk: adjacent no-op loses stale-source safety

Mitigation: no-op returns the normal fingerprint-bound zero-patch `ChangeSet`, matching established section/list behavior.

### Risk: consolidation weakens independent cell proof

Mitigation: the combined survivor scan still validates every mapped cell's header state, column, raw/content ranges, and raw bytes; it only shares the candidate index and iteration.

## TDD evidence and exit decision

Tests cover same-table reorder, exact moved bytes, cross-table rejection, adjacency no-op, stale no-op application, invalid/self targets, CRLF/source-range behavior inherited from M35, and full existing table-cell regressions. Static analysis reports zero `gocyclo`/`unparam` issues after consolidation.

The final M38 verification passed five complete `go test ./...` runs, `go test -race ./...`, coverage, `go vet ./...`, `go build ./...`, generated package/TableRow documentation, `staticcheck ./...`, standard `golangci-lint`, test-inclusive and production-only `gocyclo`/`unparam`, `govulncheck ./...` with no vulnerabilities, Gitleaks with no leaks, and the pinned published-GFM 0.29 conformance gate. Final M38 statement coverage was 90.6% for the public root package, 64.0% for `internal/parser/goldmark`, 79.2% for `internal/source`, and 62.9% for `internal/splice`. `git diff --check` and `git fsck --no-dangling` also passed.

M38 is green.