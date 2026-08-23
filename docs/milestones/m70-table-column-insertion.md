# M70 — Source-Preserving Table Column Insertion

## Status

Complete and green.

## Objective

Complete the conservative existing-document table-column structural set by inserting one semantic/source-proven column without canonicalizing the surrounding table.

## Public contract

```go
func (d *Document) PrepareInsertTableColumn(
    tableID NodeID,
    column int,
    header []byte,
    alignment TableAlignment,
    body [][]byte,
) (ChangeSet, error)
```

`column` is a zero-based insertion position in the inclusive range `0..Table.ColumnCount()`. `body` contains exactly one caller-owned cell-content byte slice per semantic body row, in source order. Header and body cell contents may be empty. The alignment uses the existing Marksplice-owned default/left/right/center vocabulary.

## Source ownership and reuse

M70 reuses the M68/M69 `MapCompleteTableRows` mutation-time gate: header, delimiter, and every semantic body row must be source-mapped at the complete table width before insertion is allowed. No public row/cell identity is synthesized for empty cells.

`internal/source.TableColumnInsertion` prepares one zero-width patch per row. The new raw slot inherits horizontal padding from the adjacent destination slot and clones one existing row pipe; existing cells, outer-pipe style, separators outside the insertion point, and line endings are left untouched. Single-column rows reuse an existing outer pipe as the required new separator instead of introducing a canonical formatting policy.

The new delimiter token is produced by `TableDelimiterAlignmentReplacement` using the adjacent delimiter cell as a template, so its exact dash run is retained while only the requested alignment colons are selected. Canonical builder delimiters are deliberately not reused for existing-source mutation.

## Candidate proof

All row patches are prepared in original snapshot coordinates and applied atomically. One candidate parse must prove:

- the promoted table count is unchanged;
- the target table has exactly one additional semantic/source-proven column;
- semantic body-row count is unchanged;
- the complete table range grows by exactly the inserted source bytes;
- the alignment vector equals the original vector with the requested value inserted at `column`;
- every candidate row still passes `MapCompleteTableRows`;
- the inserted header/body cell contents exactly equal caller input, and the inserted delimiter content exactly equals the derived delimiter token;
- all non-target table rows/cells survive through the existing patch-transform validator.

Target-table row/cell unchanged-mapping comparison is intentionally skipped because every downstream column index/range may structurally change.

## Complexity and refactor

M70 triggered a consolidation of the M68–M70 column engine. Common candidate parsing, survivor validation, table-shape validation, row/cell source validation, and permutation validation are shared helpers. After consolidation, no function in `internal/source/table_columns.go` or `internal/splice/table_column_edits.go` exceeds cyclomatic complexity 15. Planning remains linear in the target table, followed by one candidate parse/validation; no persistent column index is added.

## Risks and mitigations

1. Insertion could silently reformat author source. New bytes clone local row-slot padding, one existing pipe, and an adjacent delimiter dash run instead of using canonical builder formatting.
2. Semantic rows can exist without full lexical width. The complete-row capability gate fails closed before any patch is prepared.
3. Caller cell bytes containing line breaks or unescaped column delimiters can alter table shape. The source helper rejects line breaks and candidate width/content proof rejects any remaining shape mismatch.
4. Multiple insertions shift later source coordinates. All patches stay in original snapshot coordinates and use the established atomic patch machinery.

## Evidence

Focused TDD covers CRLF, middle/prepend/append insertion, empty cells, single-column tables, header-only tables, exact alignment insertion and dash-run inheritance, caller-input isolation, invalid indexes/alignment/body counts, unsafe cell syntax, incomplete semantic rows, invalid target kinds, local uneven padding, and byte-identical preservation of a following table.

Final verification on the uncommitted M63–M70 working tree passed:

- `gofmt` on changed Go files;
- `go test ./... -count=1`;
- `go test -race ./... -count=1`;
- `go vet ./...`;
- `staticcheck ./...`;
- `golangci-lint run` with zero issues;
- `govulncheck ./...` with no vulnerabilities found;
- `gitleaks dir . --no-banner --redact` with no leaks found;
- `go build ./...`;
- coverage: root 93.3%, Goldmark adapter 66.2%, source 80.6%, splice 60.2%, aggregate 71.0%;
- `gocyclo -over 15` reports no M68–M70 column-edit functions above the threshold;
- repository documentation scan found no stale M69/column-insertion-deferred markers;
- `git diff --check`;
- final branch/HEAD review retained `main` at `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured remote and only intended M63–M70 working-tree changes.

No commit or push was performed.
