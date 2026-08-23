# M65 — Source-Preserving Table Column Alignment

## Status

Complete and green.

## Objective

Promote the first existing-document alignment mutation on the M64 public `Table` identity without exposing delimiter rows or regenerating table source.

## Public contract

```go
func (d *Document) PrepareSetTableColumnAlignment(tableID NodeID, column int, alignment TableAlignment) (ChangeSet, error)
```

The target is a promoted `Table`, `column` is zero-based and must be inside `Table.ColumnCount()`, and `alignment` uses the existing Marksplice-owned default/left/right/center vocabulary.

A request that leaves the alignment unchanged returns a source-bound no-op `ChangeSet`, so stale-source protection is preserved.

## Source ownership and algorithm

M65 reuses M64 `TableSource.Delimiter.Cells[column].ContentRange`. The source layer validates the existing delimiter token, preserves its exact dash run, and changes only the leading/trailing `:` required by the requested semantic alignment. Whitespace, outer/inter-cell pipes, dash count, line ending, header/body bytes, and unrelated document source are not regenerated.

The candidate is parsed once. Existing promoted rows/cells are validated through the established table mutation survivor pass, with one alignment override for rows owned by the target table. The candidate table must retain table count, column count and semantic body-row count while exposing exactly the requested alignment vector and the expected source-range length delta.

M67 later factors this implementation into the shared atomic alignment-vector engine; the M65 API remains the one-column adapter.

## Complexity

Patch planning is O(c) in the final shared implementation and reparsing/validation remains linear in candidate size. No persistent table-anchor index or delimiter-node identity is added.

## Risks and mitigations

1. Adding/removing colons changes byte offsets for every later row. Existing range-transform survivor validation is reused instead of assuming fixed offsets.
2. Rewriting the whole delimiter cell could normalize author trivia. The source helper copies the original dash run and emits only the required alignment colons.

## Evidence

Focused TDD covers CRLF, non-canonical dash lengths, semantic alignment reparse, invalid columns/alignment values, header-only tables, source-bound no-op behavior, stale snapshots, and invalid target kinds.

Final verification on the uncommitted M63–M67 working tree passed:

- `gofmt` on changed Go files;
- `go test ./... -count=1`;
- `go test -race ./... -count=1`;
- `go vet ./...`;
- `staticcheck ./...`;
- `golangci-lint run` with zero issues;
- `govulncheck ./...` with no vulnerabilities found;
- `gitleaks dir . --no-banner --redact` with no leaks found;
- `go build ./...`;
- coverage: root 93.2%, Goldmark adapter 66.2%, source 79.7%, splice 64.2%, aggregate 73.0%;
- `git diff --check`;
- final branch/HEAD review retained `main` at `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured remote and only intended M63–M67 working-tree changes.

No commit or push was performed.
