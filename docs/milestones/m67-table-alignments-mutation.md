# M67 — Atomic Table Alignment Vector Mutation

## Status

Complete and green.

## Objective

Extend M65 from one alignment change to an atomic whole-table alignment update without performing one parse/validation cycle per column.

## Public contract

```go
func (d *Document) PrepareSetTableAlignments(tableID NodeID, alignments []TableAlignment) (ChangeSet, error)
```

The alignment slice must contain exactly one valid value per semantic table column. Caller storage is converted/copied before entering the internal mutation path.

## Shared implementation

M67 introduces one private alignment-mutation engine used by both M65 and M67. It walks columns once, creates a minimal source patch only for delimiter cells whose semantic alignment changes, and prepares all non-overlapping patches as one `ChangeSet`.

Every replacement is produced by the M65 source helper, so each cell retains its original dash count and surrounding delimiter trivia. The candidate is parsed exactly once regardless of the number of changed columns. A single alignment override is used while validating surviving promoted rows in the target table.

A vector identical to the current state returns a source-bound no-op `ChangeSet`.

## Complexity

Planning is O(c) for c columns, followed by one linear candidate parse/validation. Memory is O(c) only for changed-column patches and the copied expected alignment vector. No persistent index is added.

## Risks and mitigations

1. Multiple variable-length patches could be applied against already-shifted coordinates. All patches remain in original snapshot coordinates and flow through the existing sorted/non-overlap patch machinery.
2. Repeating M65 independently per column would create avoidable O(c*n) reparsing and intermediate snapshots. M67 batches the operation and makes M65 delegate to the same engine.

## Evidence

Focused TDD changes several CRLF delimiter cells with different dash counts in one operation, reparses the final semantic vector, verifies caller-input isolation, rejects wrong vector lengths/invalid enum values, and shares the M65/M66 target/error coverage.

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
