# Milestone M41 — Complete Table-Row Replacement

Status: green.

## Goal

Replace exactly one complete promoted GFM table body row with caller-provided source while preserving the M35–M40 table model and every unaffected row/cell byte.

M41 is row replacement, not cell replacement, header editing, delimiter rewriting, table rendering, or cross-table transformation.

## Public contract

M41 adds:

```go
func (d *Document) PrepareReplaceTableRow(id NodeID, replacement []byte) (ChangeSet, error)
```

The target must be a promoted M35 body row. Replacement is non-empty and is validated only in the host candidate because one table row is not a standalone GFM table document.

The candidate replacement must:

- own exactly the original row start through exactly `len(replacement)` bytes;
- remain one promoted body row;
- preserve the target row's semantic/source-proven column count;
- remain in the transformed original table;
- reproduce the caller bytes exactly.

All original rows and mapped cells outside the replaced row retain exact transformed mappings and bytes. Cells inside the target row may be created, removed, or change content as long as the complete replacement row satisfies the row contract.

Empty replacement belongs to M36 removal and fails with `ErrInvalidReplacement` here.

## Host-context safety

M41 patches exactly the M35 complete physical `LineRange`, including its existing terminator when present. Marksplice never manufactures a terminator.

Therefore:

- a final EOF row may be replaced by an unterminated final row when the candidate proves exact ownership;
- an unterminated replacement in the middle that merges with the next physical line fails closed;
- a multi-row replacement fails because no single candidate row owns the entire replacement span;
- a different column count fails even when the Markdown is otherwise parseable.

Prepared changes remain source-fingerprint bound and reject stale application.

## Final M35–M41 consolidation

The final table-family review made two concrete reuse/performance changes.

First, insert, replacement, and move now share one `candidateOwnedTableRow` proof for exact candidate span, byte ownership, column count, and private table anchor. Operation-specific logic is limited to how the expected start/table anchor is derived.

Second, the M39/M40 resolver originally returned a small `tableRowModel` containing the flat promoted-cell adjacency plus the already-known promoted row count. That removed a full original-node recount from M36–M41 mutations. The later post-M43 whole-codebase consolidation replaces the redundant scalar count with `len(tableRowIndexes)` and retains the builder's compact source-ordered row/cell node-index slices, so candidate indexing and original survivor validation operate only on the relevant table families instead of filtering the complete document-node collection.

No persistent table-anchor map is introduced. Durable table-family collections remain compact linear arrays: body/header adjacency IDs plus source-ordered row/editable-cell node indexes.

## Complexity

A replacement performs one source patch, one candidate parse, one operation-local candidate row/cell index, and family-scoped survivor validation. The safety oracle remains O(n+t) for source size `n` and relevant table-model work `t`; post-M43 consolidation no longer adds a separate complete-document-node filter pass for original rows/cells. The refactors also remove the old O(nodes) row recount and duplicated ownership-proof branches.

## Devil's advocate review

### Risk: parseable replacement silently consumes the next line

Mitigation: the candidate row must own exactly `[originalStart, originalStart+len(replacement))`; merged source cannot satisfy that exact `LineRange`.

### Risk: replacement migrates to another table or changes width

Mitigation: candidate ownership requires the transformed original private table anchor and exact original column count.

### Risk: survivor validation incorrectly requires old target cells to remain

Mitigation: the target row and cells physically inside it are intentionally excluded from survivor proof; every other mapped row/cell is still validated independently.

### Risk: refactor weakens insertion/movement safety

Mitigation: the shared ownership helper contains the same exact range/bytes/columns/table-anchor checks previously duplicated by M37/M38; focused and full regressions cover all three operations after consolidation.

## TDD evidence

The first public test failed to compile because `PrepareReplaceTableRow` did not exist. Current tests cover CRLF exact replacement, EOF unterminated replacement, stale source, zero/wrong-kind targets, wrong column count, unsafe middle joins, multi-row source, and preservation of neighboring rows and cells. M39/M40 regressions additionally cover header/body row identity, empty/all-empty body cells, defensive adjacency copies, unresolved unpromoted parents, and injected corrupt row anchors, columns, source containment, bounds, missing IDs, cross-row identities, and reversed child order.

Final repository-wide verification on the exact post-refactor M41 tree passed five consecutive `go test ./...` runs, `go test -race ./...`, coverage, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, standard `golangci-lint`, test-inclusive and production-only `gocyclo`/`unparam` with zero issues, `govulncheck ./...` with no vulnerabilities, Gitleaks with no leaks, and the pinned published-GFM 0.29 conformance gate. Final statement coverage is 90.9% for the public root package, 64.0% for `internal/parser/goldmark`, 79.2% for `internal/source`, and 64.2% for `internal/splice`.

M41 is green.