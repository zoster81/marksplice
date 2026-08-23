# M63 — Parsed Table Row Alignments

## Goal

Promote the semantic GFM table-column alignment vector already retained and mutation-proven by M62 into a narrow read-only public API for parsed documents, without introducing a public `Table` identity or making the established comparable `TableRow` value non-comparable.

## Requirements and edge cases

M63 must expose exactly one semantic alignment per source/semantic-proven body-row column and reuse the existing Marksplice-owned `TableAlignment` vocabulary:

- `TableAlignmentDefault`;
- `TableAlignmentLeft`;
- `TableAlignmentRight`;
- `TableAlignmentCenter`.

The accessor must reject missing/non-row identities, fail closed if retained metadata is internally inconsistent, return caller-owned storage, and perform no parser pass, source rescan, or persistent table index construction. Existing table-row mutation behavior, `Kind` ordinals, `NodeID` derivation, source ranges, construction bytes, and public `TableRow` comparability must not change.

A public `Table` model is deliberately deferred. Anchoring this first parsed alignment accessor to an existing promoted body-row identity avoids inventing table identity semantics before tables with no promoted body rows and other ownership boundaries are reviewed.

## Architecture and test strategy

The parser adapter already maps Goldmark's public `AlignNone`, `AlignLeft`, `AlignRight`, and `AlignCenter` values into parser-independent alignment metadata. M62 retained that vector on promoted body rows, defensively copied it at immutable-document boundaries, and added alignment-aware table-row survivor/candidate validation.

M63 therefore adds only:

```go
func (d *Document) TableRowAlignments(rowID NodeID) ([]TableAlignment, bool)
```

The root API resolves the promoted `TableRow`, requires the retained vector length to equal the row's semantic/source-proven `ColumnCount`, converts each internal value through an explicit switch, and returns a newly allocated public slice. Complexity is O(c) time and O(c) caller-owned output for c columns.

TDD covers:

- all four semantic alignment values in source order;
- identical alignment results from multiple body rows in one table;
- all-default tables;
- caller mutation of a returned slice not affecting the snapshot;
- non-row and zero identities failing closed;
- preservation of `TableRow` comparability.

## Devil's advocate review

**Risk: adding `[]TableAlignment` directly to `TableRow` would break comparability.** M42 deliberately preserved `TableRow` as a comparable value. M63 therefore uses a `Document` accessor returning caller-owned slice storage rather than changing the value layout.

**Risk: numeric casting could silently couple public enum values to internal/parser ordinals.** M63 converts through an explicit value switch and fails closed on unknown internal values rather than assuming ordinal identity.

**Risk: returning retained internal slice storage would violate snapshot immutability.** The accessor always allocates public output and copies/converts every element.

**Risk: alignment exposure could accidentally imply a public table identity or alignment mutation contract.** The API is explicitly row-anchored and read-only; public `Table`, delimiter ownership, and alignment mutation remain future reviewed slices.

## TDD evidence

The focused red test failed only because `Document.TableRowAlignments` did not yet exist. After the minimal implementation and formatting, the focused M63 tests pass, followed by the complete repository regression. The first strict-gate attempt then stopped at Staticcheck because the new comparability test used a tautological zero-value equality expression (SA4000); the test was corrected to a compile-time map-key comparability proof, and the complete strict gate was rerun from the beginning.

## Documentation alignment

M63 establishes [`../capabilities.md`](../capabilities.md) as the product-facing source of truth for the current semantic-read/edit/create matrix and forward capability roadmap. That roadmap does not replace the architecture, GFM conformance, parser/source ownership, or milestone evidence documents.

## Verification

The M63 code/documentation tree passed the strict fail-fast project gate before this final evidence text was recorded: `gofmt` is clean; five consecutive `go test ./... -count=1` runs pass; `go test -race ./... -count=1` passes with task-scoped CGO and the project-private GCC; `go vet ./...` and `go build ./...` pass; public documentation for `TableAlignment` and `Document.TableRowAlignments` resolves; the pinned published GFM 0.29 conformance suite passes; Staticcheck passes; golangci-lint reports zero issues; production and test-inclusive unparam report no findings; and production gocyclo reports no function above complexity 20 across 43 production Go files. Coverage is 93.4% for the root package, 65.6% for the Goldmark adapter, 79.3% for `internal/source`, and 66.9% for `internal/splice`; the parser interface package has no direct tests.

Govulncheck reports zero vulnerabilities reachable from Marksplice code; it also reports one vulnerability in imported packages and seven in required modules that Marksplice does not appear to call. Gitleaks reports no leaks. Strict UTF-8/no-BOM/LF/no-trailing-whitespace hygiene passes over all nine changed/untracked M63 paths, `git diff --check` and `git fsck --no-dangling` pass, the branch remains `main` at pre-M63 HEAD `352d094fe6ada53b0d9c4c417dc36bd633642692`, and no Git remotes are configured. After recording the final evidence text, the focused M63 tests, one complete `go test ./... -count=1` regression, strict text hygiene over all nine paths, and `git diff --check` pass again on the final documented tree.

## Exit decision

M63 is complete. It adds read-only parsed table alignment access only. A public `Table` model, table alignment mutation, and column structural operations remain deferred for separate design/review.
