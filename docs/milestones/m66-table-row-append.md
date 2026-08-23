# M66 — Table-Level Body Row Append

## Status

Complete and green.

## Objective

Use the M64 table identity to append one compatible GFM body row even when the table has no promoted body-row identity to anchor the existing M37 insertion API.

## Public contract

```go
func (d *Document) PrepareAppendTableRow(tableID NodeID, fragment []byte) (ChangeSet, error)
```

The caller owns the complete row fragment. Marksplice does not synthesize indentation, pipes, spacing, or line separators.

## Boundary semantics

The insertion point is the exact end of `Table.Range()`. The existing table must already end on a physical line boundary (`CR`, `LF`, or `CRLF` ownership reflected by the final byte). This deliberately rejects a delimiter/final body row that reaches EOF without a terminator: inserting a new row there would require Marksplice to invent a separator not supplied by the caller.

When the table already owns a line terminator, a caller fragment at document EOF may itself omit a trailing terminator because the inserted body row can validly end at EOF.

## Candidate proof and reuse

M66 reuses the existing table-row mutation index and `candidateOwnedTableRow` proof. The candidate must:

- add exactly one promoted compatible body row;
- retain every original promoted row/cell through the existing survivor validator;
- preserve table count, column count and alignment vector;
- increase the target table semantic body-row count by exactly one;
- make the inserted row the semantic last body row at the insertion anchor;
- extend the exact table range by exactly the caller fragment length.

This works for header-only tables and for tables whose pre-existing semantic rows are outside the stricter promoted `TableRow` subset.

## Complexity

One candidate parse plus existing linear table-model validation. No new persistent index or alternate row parser is introduced.

## Risks and mitigations

1. A visually separate following line may still be a GFM body row when no blank line terminates the table. Tests use true block termination and M66 follows parser semantics rather than visual pipe heuristics.
2. Hidden line-separator synthesis could corrupt source style. Append is rejected without a pre-existing table line boundary; otherwise caller bytes are inserted unchanged.

## Evidence

Focused TDD covers LF/CRLF, header-only tables, existing unpromoted semantic rows, EOF append, missing line boundaries, incompatible column counts, multi-row fragments, stale/invalid targets, and reparsed body-row counts.

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
