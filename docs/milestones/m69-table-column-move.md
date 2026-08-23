# M69 — Source-Preserving Table Column Move

## Status

Complete and green.

## Objective

Move one complete GFM table column to a new zero-based position while reusing M68's complete-row capability gate and preserving author-owned row layout trivia.

## Public contract

```go
func (d *Document) PrepareMoveTableColumn(tableID NodeID, from, to int) (ChangeSet, error)
```

`to` is the final zero-based position. Equal `from`/`to` returns a source-bound no-op `ChangeSet`.

## Complete-row capability gate

M69 reuses `internal/source.MapCompleteTableRows`; header, delimiter, and every semantic body row must all be source-mapped at the table's full semantic column count. Partially understood tables remain readable through M64 but are not column-movable.

## Source ownership and algorithm

`ReorderTableRowColumns` reorders only source-proven cell **content** bytes. The destination slot's leading/trailing horizontal whitespace, inter-cell separators, outer-pipe style, and physical line ending stay in their original positions. On the delimiter row this moves the exact colon/dash token, so alignment semantics travel with the moved column while slot formatting remains author-owned.

Every table row is replaced atomically over its existing content-only physical `Range()`. Each replacement has exactly the original row length, so line endings and all source outside the target rows are untouched and no downstream byte offsets change.

The target table's promoted rows/cells are skipped by unchanged-survivor comparison because their columns intentionally reorder; all non-target table state still uses the shared survivor validator. Candidate proof requires unchanged table/column/body-row counts and complete range, the alignment vector reordered by the same permutation, and successful complete-row remapping after the move.

## Complexity

Permutation construction is O(c), row rewriting is linear in target-table source size, and validation performs one candidate parse. No repeated per-column parse and no persistent column index are introduced.

## Risks and mitigations

1. **Moving raw cell padding with content could unintentionally restyle the row.** M69 keeps whitespace in destination slots and moves only trimmed source-proven content; dedicated tests use uneven padding to prove this boundary.
2. **Independent row moves could disagree on column order.** One shared permutation is applied to header, delimiter, and every body row in one atomic change, then reparsed alignment/table semantics are checked.

## Evidence

Focused TDD covers content/alignment reordering, exact delimiter-token movement, uneven slot-whitespace preservation, invalid indexes, source-bound no-op/stale behavior, invalid target kinds, and fail-closed incomplete semantic rows.

Final verification on the uncommitted M63–M69 working tree passed:

- `gofmt` on changed Go files;
- `go test ./... -count=1`;
- `go test -race ./... -count=1`;
- `go vet ./...`;
- `staticcheck ./...`;
- `golangci-lint run` with zero issues;
- `govulncheck ./...` with no vulnerabilities found;
- `gitleaks dir . --no-banner --redact` with no leaks found;
- `go build ./...`;
- coverage: root 93.2%, Goldmark adapter 66.2%, source 80.3%, splice 61.8%, aggregate 71.8%;
- repository documentation scan found no stale M67/table-column-deferred markers;
- `git diff --check`;
- final branch/HEAD review retained `main` at `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured remote and only intended M63–M69 working-tree changes.

No commit or push was performed.
