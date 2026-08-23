# M97 — Structural query surface

## Status

Complete. Focused TDD, documented-tree release-quality verification, corrected cross-package coverage, final hygiene, and repository-state checks are green.

## Objective

Add a small bounded public query surface over immutable `Document` snapshots so callers can find reviewed structural content without manually walking every typed accessor.

M97 must not:

- create a second mutable document model;
- introduce a string/regex query language;
- create persistent query indexes without measurement;
- reinterpret public source ownership;
- expose internal/opaque nodes merely because the parser understands them;
- consume heading-anchor, fragment, TOC, or link-relationship contracts owned by M98–M99.

## Public contract

M97 adds:

```go
type NodeQuery struct {
    Kinds  []Kind
    Within *Range
    Limit  int
}

type NodeMatch struct { /* immutable */ }

func (m NodeMatch) Node() Node
func (m NodeMatch) Range() Range
func (d *Document) QueryNodes(query NodeQuery) ([]NodeMatch, error)

type SectionQuery struct {
    Levels []int
    Within *Range
    Limit  int
}

func (d *Document) QuerySections(query SectionQuery) ([]Section, error)
```

Malformed or unbounded requests report `ErrInvalidQuery`.

### Bounds

- `Limit` is mandatory and must be greater than zero.
- an empty `NodeQuery.Kinds` selects every currently promoted public node kind;
- a node-kind filter may contain at most the number of currently useful public kinds, so arbitrarily long duplicate vectors are rejected;
- every requested kind must be a reviewed public `Kind`, not `KindUnknown` or an unknown future ordinal;
- an empty `SectionQuery.Levels` selects levels 1 through 6;
- a section-level filter may contain at most six entries and every level must be 1 through 6;
- `Within == nil` means the whole snapshot;
- non-nil `Within` is copied immediately and must be a valid half-open byte range in the exact snapshot.

Duplicate kind/level entries within those small bounds are idempotent.

### Selection semantics

Results retain the source/structural order already used by `Document.Nodes()` and `Document.Sections()`. The scan stops immediately when `Limit` matches have been collected.

`Within` uses complete containment, not mere overlap:

```text
match.Start >= Within.Start && match.End <= Within.End
```

`NodeMatch.Range()` deliberately does **not** define a new generic full-node ownership model. It returns the same operation-oriented range already exposed by the matched kind's typed `Range()` accessor:

- paragraph -> `Paragraph.Range()`;
- heading -> `Heading.Range()`;
- list item -> `ListItem.Range()`;
- task -> `Task.Range()`;
- table cell -> `TableCell.Range()`;
- fenced code -> `FencedCode.Range()`;
- simple strikethrough/code/emphasis/strong -> the corresponding typed `Range()`;
- inline link/image/reference definition/autolink -> the corresponding destination/token typed `Range()`;
- front-matter field, HTML comment, HTML anchor -> the corresponding typed `Range()`;
- table row -> `TableRow.Range()`;
- table -> `Table.Range()`;
- thematic break -> `ThematicBreak.Range()`;
- blockquote -> `Blockquote.Range()`.

The query range is selection/read metadata only. Existing mutation methods and typed source-ownership gates remain authoritative.

`QuerySections` returns the existing immutable comparable `Section` representation directly. Its `Within` filter uses the complete `Section.Range()`.

## Architecture

The public query methods scan the immutable source-ordered arrays that already back `Nodes()` and `Sections()`.

`internal/splice` adds two small read-only primitives:

- `SourceLen()` validates query ranges without copying source bytes;
- `NodeSelectionAt(index)` returns a lightweight `NodeSummary` plus the existing typed selection range without cloning the full internal node or its variable-length metadata.

The historical `NodeSummaryAt` path remains lightweight and unchanged in responsibility; M97-specific range projection is performed only when a query needs it.

The private `nodeSelectionRange` switch is exhaustive for the currently promoted Marksplice kinds and fails closed for `KindUnknown` and opaque HTML. A white-box regression verifies the current mapping across the internal kind set.

No persistent kind/range query index, cache, alternate AST, or query state is retained in `Document`.

## Complexity and resource model

For `N` existing nodes/sections and caller limit `L`:

- `QueryNodes`: O(N) worst-case scan, early stop at `L`, O(L) result storage;
- `QuerySections`: O(N) worst-case scan, early stop at `L`, O(L) result storage;
- kind membership is a fixed-size boolean array;
- level membership is a fixed-size seven-slot boolean array;
- no result sort is required because the authoritative arrays are already source ordered;
- validating `Within` is O(1) and does not copy source bytes.

M108 owns measurement-driven performance hardening. M97 intentionally does not add persistent indexes before benchmarks demonstrate a need.

## Requirements and edge cases

The focused contract covers:

- filtering multiple node kinds inside a section body range;
- nested node matches such as an inline link contained by a paragraph;
- range equality between `NodeMatch.Range()` and representative typed accessors;
- table complete-range selection versus list/link content/destination selection;
- heading selection with early `Limit == 1` stop;
- source-order equivalence with `Document.Nodes()` for unfiltered prefix queries;
- section level and complete-subtree range filtering;
- duplicate filter entries;
- caller-owned result slices;
- empty snapshots with an explicit positive limit and valid zero range;
- nil documents;
- zero/negative limits;
- unknown node kinds and invalid section levels;
- oversized duplicate filter vectors;
- negative and out-of-bounds `Within` ranges.

## TDD evidence

1. `tsk_4f0a2c89da91f7d418ba0694ab040fa4` established the initial public RED: the tests failed to compile only because `NodeQuery`, `NodeMatch`, `SectionQuery`, `QueryNodes`, and `QuerySections` did not exist.
2. `tsk_ac84ad27465735ce350c8442865d842e` passed the first public/internal GREEN after implementing the bounded scans and exhaustive selection-range mapping.
3. Architecture review then removed query-specific fields from the historical `NodeSummaryAt` path and introduced `NodeSelectionAt`, so ordinary `Document.Nodes()` enumeration does not pay M97-specific range work.
4. `tsk_a50535f759624515bb0debaf97a6c920` established a second RED: arbitrarily long duplicate node-kind and section-level filter vectors were still accepted.
5. `tsk_2a02e85219c6f784b12c211292294314` passed after adding constant-size filter-vector bounds.
6. `tsk_75030328b0e88a8ba355d223a7bd075d` passed focused/public/internal tests, one complete repository regression, the executable query example, vet/build, Staticcheck, golangci-lint, production gocyclo <= 15, production and test-inclusive unparam, `go mod tidy -diff`, and `git diff --check` with the activated private toolchain.

An earlier regression task, `tsk_19e56718cc69261ed29482a74ab8a4d1`, had already passed focused tests, the full repository regression, and gocyclo before failing only because `unparam` was not on PATH in that shell. It is harness evidence, not the final maintainability gate; the corrected activated-toolchain task above is authoritative.

## Devil's advocate review

1. **A query `Range()` could silently become a second source-ownership contract and later be used as raw mutation authority.** Mitigation: `NodeMatch.Range()` is explicitly defined as the already-reviewed typed `Range()` for that kind, is documented as selection-only, and all mutations still require typed targets plus existing candidate/source proof.
2. **A convenience query could allocate full internal nodes or copy table/blockquote metadata for every scanned candidate.** Mitigation: `NodeSelectionAt` returns only scalar summary facts plus one range; it never calls the full cloning `Document.Node` path. Result allocation is capped by `min(L, available)` and scanning stops at `L`.
3. **A query subsystem could become a redundant persistent index that must be kept synchronized with the immutable structural model.** Mitigation: M97 stores no query index or cache. It reuses the authoritative source-ordered node and section arrays. M108 owns any future measurement-driven indexing decision.
4. **A future public kind could accidentally receive a zero or guessed query range.** Mitigation: the internal range mapper fails closed for unhandled kinds, and the current internal-kind regression requires every promoted kind to map to its reviewed typed span while opaque/unknown kinds remain unavailable.
5. **Input vectors could technically have bounded outputs while still causing avoidable CPU work through millions of duplicate filters.** Mitigation: kind and level filter vectors have small constant cardinality limits matching the useful domain; membership is stored in fixed-size arrays.

## Release-quality verification

The documented implementation tree passed the complete M97 freeze:

- `tsk_87f28c82f41df0a01ec4f463eb3a19bd`: five consecutive full `go test ./... -count=1` runs plus full race detection;
- `tsk_5c9d8ecd3da73872ae8926104dccbcf4`: formatting, vet/build, executable examples and query API docs, Staticcheck, golangci-lint, production gocyclo <= 15, production and test-inclusive unparam, `go mod tidy -diff`, `git diff --check`, govulncheck, Gitleaks, and actionlint. Its shortened GFM `-run` filter was later found not to select the real conformance test and is therefore not counted as conformance evidence;
- `tsk_4667b9e1029b9339f9756a6889d5c159`: direct Go 1.27.0 test, vet, and build;
- `tsk_5762211926441229f636f9bc6a810cd1`: corrected explicit cross-package coverage, 86.7% aggregate statement coverage over the production package set and 83.8% through `internal/publictest`, with private profiles removed after measurement;
- `tsk_b52d493b3f044e88009ee10c43066ab3`: strict text hygiene, private/public boundary checks, historical artifact absence, `git diff --check`, `git fsck --no-dangling`, branch/HEAD/origin verification, and confirmation that `go.mod`/`go.sum` remain unchanged;
- `tsk_85a8e7dab9f066570469a5aa5b55f721`: exact anchored `TestGFM029PublishedSpecificationConformance` execution against the approved pinned snapshot; verbose output proves that the test actually ran and passed.

## Post-completion refactor review

The post-M97 review keeps behavior and public contracts unchanged while tightening implementation responsibility boundaries:

- all reference-link/reference-image construction forms, resolution, writing, and proof are centralized in `builder_inline_reference.go`; generic typed-inline writing remains in `builder_inline.go`;
- query-only internal projection moved from the general snapshot implementation to `internal/splice/query.go`;
- `NodeSummaryAt` and `NodeSelectionAt` now reuse one scalar summarization helper;
- `QueryNodes` and `QuerySections` share nil-document/positive-limit validation and cache the immutable collection count for their bounded scans;
- the documented GFM gate now requires the exact anchored conformance test name because `go test -run` may otherwise succeed after selecting zero tests.

Focused query/reference regressions passed in `tsk_c8b5927023a7ba820cdd415e537be216`. Post-refactor complexity inventory `tsk_cee51a915486b3b73ef835ceaa425652` reports average production cyclomatic complexity 4.9, no function above the existing limit of 15, `builder_inline.go` reduced from 965 to 813 lines, and the reference-specific implementation isolated in a 298-line file.

The complete refactored tree then passed the pre-commit freeze:

- `tsk_5d71cf457182788df8e06ffa07062def`: five consecutive full tests plus race;
- `tsk_63edabb2255caa49286a8a72deecf631`: formatting, vet/build, examples, Staticcheck, golangci-lint, complexity/unparam, tidy/diff checks, the exact anchored GFM conformance test with verbose proof of execution, govulncheck, Gitleaks, and actionlint;
- `tsk_9ec199eebd3d70ea5506354dce5ff6ba`: direct Go 1.27.0 test/vet/build;
- `tsk_e7f6824f141c4cf4d78cf0d3d9bfe242`: corrected cross-package coverage at 86.7% aggregate and 83.8% through `internal/publictest`, with private profiles removed;
- `tsk_0ae35e583bf3ee5142d723f8528926d6`: corrected final pre-commit text/private-boundary/artifact/Git hygiene.

Two pre-commit harness runs are explicitly non-authoritative: `tsk_08f8de8cfeeddaf1bc8b3bca521710ef` passed literal PowerShell coverage variables and produced meaningless 0.0% totals, while `tsk_a1675834d02d8769f31c6a274355f555` correctly caught the resulting literal `$allPath` artifact. Both literal coverage artifacts were inspected as two-line atomic coverage profiles, removed, and the corrected coverage/hygiene gates above passed.

## Exit decision

M97 is complete. The query surface remains a bounded immutable-snapshot selector layer with no persistent query index or second document model. M98 — anchors, fragments, and TOC — is the next implementation milestone.
