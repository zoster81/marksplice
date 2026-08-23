# M62 — Aligned Table Construction

## Goal

Extend the M51 new-document table writer with explicit GFM column alignment while preserving the existing parsed table API, source-preserving row mutations, and fail-closed construction proof.

## Public contract

M62 adds the construction-only alignment type:

```go
type TableAlignment uint8

const (
    TableAlignmentDefault TableAlignment = iota
    TableAlignmentLeft
    TableAlignmentRight
    TableAlignmentCenter
)
```

and the additive builder operation:

```go
func (b *DocumentBuilder) AppendTableWithAlignments(
    header []string,
    alignments []TableAlignment,
    rows ...[]string,
) error
```

`AppendTable` remains unchanged and continues to produce the M51 all-default table form.

The alignment slice must contain exactly one valid value for every header column. Header cells, body rows, and alignments are copied at append time so later caller mutation cannot alter retained construction state.

Canonical delimiter cells are:

- default: `---`;
- left: `:---`;
- right: `---:`;
- center: `:---:`.

All other M51 table constraints remain in force: at least one header column and one body row, equal row widths, canonical LF output, outer pipes, canonical cell padding, and reviewed single-line cell source.

## Architecture

The pinned Goldmark public table AST exposes semantic `Alignment` values on `Table`, `TableHeader`, `TableRow`, and `TableCell`. M62 therefore does not infer alignment from delimiter punctuation after parsing. The Goldmark adapter maps the public `AlignNone`, `AlignLeft`, `AlignRight`, and `AlignCenter` values into Marksplice-owned internal alignment values.

Each promoted body-row observation now carries an alignment vector whose length must equal its semantic column count. The splice node retains a defensive copy of that vector. Construction proof requires every generated body row to reproduce the exact expected table anchor, column count, alignment vector, physical row mapping, and cell ranges.

No public parsed `Table`, table identity, `TableRow.Alignment`, or `TableCell.Alignment` accessor is introduced. Alignment is construction input and internal proof metadata in M62.

## Mutation-strengthening refactor

M62 reuses the new semantic metadata to strengthen existing table-row mutation validation. Surviving, inserted, replaced, and moved body rows must preserve the expected alignment vector in addition to the established table anchor, column count, source range, and row bytes.

While adding the alignment slice, the internal immutable-document copy boundary was reviewed. `splice.Document.Nodes()` and `splice.Document.Node()` now defensively copy both `TableAlignments` and the pre-existing `TableRowSource.Cells` slice, preventing returned internal node values from aliasing stored table metadata.

## TDD evidence

The initial focused red run failed to compile only because the M62 public/internal alignment symbols and `AppendTableWithAlignments` did not yet exist.

Focused green coverage proves:

- Goldmark exposes `[left, right, center, default]` for a four-column aligned table;
- the builder emits the exact canonical delimiter source;
- caller mutation of header, row, and alignment slices after append does not change output;
- invalid/missing/mismatched alignment vectors fail with `ErrInvalidConstruction` and leave the builder unchanged;
- `Nodes()` and `Node()` returned values cannot mutate stored alignment or table-cell mapping slices;
- replacing a body row in an aligned table preserves the delimiter bytes and semantic alignment vector after reparsing.

Focused root, Goldmark, and splice tests pass. The complete repository regression, `go vet ./...`, `go build ./...`, and production gocyclo also pass before documentation; no production function exceeds complexity 20 across 33 production Go files. One intermediate verification process reported an output-drain timeout after exit code 0; an explicit package rerun immediately afterward passed root, Goldmark, source, and splice tests, confirming this was a runner transport issue rather than a repository failure.

The documented M1–M62 tree then passed the complete project verification gate: five consecutive full `go test ./... -count=1` runs, `go test -race ./... -count=1`, coverage, `go vet ./...`, `go build ./...`, public API documentation checks, the pinned GFM 0.29 conformance suite, Staticcheck, golangci-lint with zero issues, production/test-inclusive unparam, govulncheck with no reported vulnerabilities, Gitleaks with no reported leaks, strict UTF-8/LF/text hygiene over all 58 changed/untracked paths, `git diff --check`, and `git fsck --no-dangling`. Coverage at that M62 boundary was 93.5% for the root package, 65.6% for the Goldmark adapter, 79.3% for `internal/source`, and 66.8% for `internal/splice`; the parser interface package has no direct tests. No production function exceeded complexity 20 across the 33 production Go files present at that boundary.

## Post-M62 consolidation refactor

After the M62 gate, the complete M1–M62 codebase was reviewed again for reuse, responsibility boundaries, dead complexity, and immutable-data ownership. The refactor deliberately keeps feature-specific safety proof separate while consolidating only responsibilities with identical semantics:

- the construction implementation is split into public builder/API declarations, input validation/retained construction state, canonical writers, and reparse proof rather than one approximately 1,200-line file;
- public typed detail value objects remain in `api_types.go`, while `Document` typed lookup/conversion plumbing is isolated in `api_details.go`;
- Goldmark observations are separated into dispatch/block, inline, and structural list/table/task files without changing the parser-independent observation contract;
- source mapping separates shared unsupported-shape errors and heading/task, list, table, and fenced-code mapping responsibilities while retaining the existing focused link/front-matter/HTML/inline files;
- list-item, section, and table-row moves now reuse one private `prepareMoveCandidate` helper for the identical two-patch delete+insert candidate assembly, while planning, no-op detection, and every family-specific candidate/survivor proof remain separate.

No public API, public `Kind` ordinal, `NodeID` derivation, source range contract, canonical output bytes, stale-source behavior, or source-preserving mutation policy changes as part of this consolidation. Intermediate post-refactor verification passes full repository tests, vet/build, Staticcheck, golangci-lint with zero issues, production/test-inclusive unparam, `git diff --check`, and the production gocyclo threshold with no function above 20 across 43 production Go files.

The final documented/refactored tree then passed a strict fail-fast gate: `gofmt` clean; five consecutive `go test ./... -count=1` runs; a real `go test -race ./... -count=1` run with CGO enabled and the project-private w64devkit GCC; coverage; `go vet ./...`; `go build ./...`; public `DocumentBuilder` and `TableAlignment` documentation checks; the pinned GFM 0.29 conformance suite; Staticcheck; golangci-lint with zero issues; production/test-inclusive unparam; govulncheck reporting zero reachable vulnerabilities in Marksplice code; Gitleaks with no leaks; strict UTF-8/LF/text hygiene over all 68 changed/untracked paths; `git diff --check`; and `git fsck --no-dangling`. Final coverage is 93.5% for the root package, 65.6% for the Goldmark adapter, 79.3% for `internal/source`, and 66.9% for `internal/splice`; the parser interface package has no direct tests. Govulncheck also reports one vulnerability in imported packages and seven in required modules that are not reached by Marksplice code. No production function exceeds complexity 20 across 43 production Go files.

## Exit decision

M62 is complete. The post-M62 consolidation is behavior-preserving and does not open M63 functionality. Future table-construction milestones may broaden style policy or expose reviewed parsed alignment navigation, but must reuse this semantic alignment proof rather than returning to delimiter-only inference.
