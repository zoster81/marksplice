# M68 — Source-Preserving Table Column Removal

## Status

Complete and green.

## Objective

Add existing-document removal of one complete GFM table column on top of the M64 public `Table` model without inventing synthetic empty-cell identities or regenerating the table.

## Public contract

```go
func (d *Document) PrepareRemoveTableColumn(tableID NodeID, column int) (ChangeSet, error)
```

`column` is zero-based. Removing the only remaining column is rejected.

## Complete-row capability gate

Column removal has a stricter mutation-time gate than ordinary table reading. `internal/source.MapCompleteTableRows` must source-map the header, delimiter, and every semantic body row with exactly `Table.ColumnCount()` cells. If even one semantic row cannot satisfy the existing physical-row mapper, the operation fails closed.

This keeps M64's broad read model intact while preventing column surgery on partially understood rows.

## Source ownership and algorithm

For every mapped row, `TableColumnRemovalRange` owns the selected raw cell plus exactly one adjacent inter-cell pipe:

- first/middle columns remove through the next cell start;
- the last column removes from the previous cell end through the selected cell end.

This preserves the row's existing leading/trailing outer-pipe style and leaves every surviving cell byte unchanged. Header, delimiter, and body deletions are prepared in original snapshot coordinates as one atomic `ChangeSet`.

The target table's promoted rows/cells are intentionally excluded from unchanged-survivor comparison because their column indexes and ranges change. All other tables still pass the established survivor validator, including range/anchor transforms caused by the shorter target table. The reparsed target must retain table count and semantic body-row count, decrease column count by exactly one, expose the alignment vector with the same column removed, own the exact shortened table range, and satisfy complete-row mapping again.

## Complexity

Complete-row proof and patch planning are linear in target-table source size, followed by one candidate parse/validation. No persistent per-column index is introduced.

## Risks and mitigations

1. **Wrong separator ownership could damage first/last-column pipe style.** Dedicated source-layer tests exercise first, middle, and last removal with outer pipes.
2. **Shortening one table shifts every later table.** Existing patch-transform survivor validation remains active for non-target tables, and focused coverage proves a following table remains byte-identical after rebasing.

## Evidence

Focused TDD covers CRLF source, empty surviving cells, alignment-vector contraction, first/middle/last lexical removal spans, a following table shifted but byte-identical, invalid indexes, one-column rejection, invalid target kinds, and fail-closed incomplete semantic rows.

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
