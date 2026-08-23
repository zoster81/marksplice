# M88 — Structured Typed-Inline Nesting

## Status

Complete and green.

## Objective

Extend the existing construction-only `Inline` intent so callers can compose reviewed nested inline structures without widening ordinary parsed-source simple-inline editability and without introducing a second public inline AST.

## Public contract

M88 adds no new exported functions. It extends the existing constructors:

- `EmphasisInline(content ...Inline)`;
- `StrongInline(content ...Inline)`;
- `StrikethroughInline(content ...Inline)`.

`EmphasisInline` and `StrongInline` may now contain semantic text plus `CodeInline`, `EmphasisInline`, `StrongInline`, and `StrikethroughInline` children. `StrikethroughInline` accepts semantic text plus code/emphasis/strong children and rejects direct strikethrough-in-strikethrough nesting because adjacent tilde runs can change GFM block/inline interpretation.

`LinkInline`, `LinkInlineWithTitle`, `ImageInline`, `ImageInlineWithTitle`, and `AutoLinkInline` remain top-level typed-inline forms for this slice and are rejected when nested inside emphasis/strong/strikethrough wrappers. Link/image label and alt construction remain semantic-text-only.

Structured wrapper depth is bounded at 64. This is a safety maximum, not a promise that every possible delimiter combination below 64 is representable. Generated source is accepted only when GFM reparsing reproduces the exact requested hierarchy; ambiguous delimiter combinations fail closed even at smaller depths.

Existing simple typed-inline byte output remains unchanged.

## Architecture and test strategy

M88 keeps the public `Inline` representation private and reuses the existing writer/expectation engine.

The writer now carries a private context containing:

- ordinary source-mapped expectations;
- construction-only hierarchy expectations;
- current nesting policy;
- parent hierarchy index;
- wrapper depth;
- the nearest emphasis-family marker;
- parent wrapper kind.

Top-level typed content keeps the historical capability set. Link/image labels use a text-only policy. Structured wrapper children use a reviewed policy that permits only text, code, emphasis, strong, and strikethrough.

For nested emphasis/strong generation, the writer alternates `*` and `_` relative to the nearest emphasis-family ancestor to reduce adjacent delimiter collisions. Strikethrough continues to use canonical `~~`. Code spans retain the adaptive-backtick writer from M76.

Every generated code/emphasis/strong/strikethrough semantic node receives an ephemeral construction-only expectation containing exact syntax/content ranges, marker, delimiter length, and direct parent index. The expectation crosses `internal/splice` into a parser-independent DTO under `internal/parser`; `internal/parser/goldmark` is the only package that interprets it through Goldmark AST nodes.

Goldmark validation does not call `Adapter.Parse` and does not widen ordinary observations. It validates expectation inputs, matches semantic nodes in one AST walk using anchor buckets, verifies exact kind/anchor/delimiter/source facts, checks direct parents, and compares root/direct child order from precomputed parent→children lists.

Simple leaf/source-mapped expectations continue through the existing parsed `Document` proof. Compound emphasis/strong/strikethrough nodes that remain intentionally non-editable in ordinary parsed snapshots are skipped only by that old simple-expectation collector because the M88 hierarchy proof is authoritative for them.

`cloneTypedInlineConstruction` now copies only the caller-provided top-level `[]Inline` slice. This preserves caller-slice snapshot semantics while avoiding recursive cloning: nested `Inline` child fields are private and cannot be mutated by callers after their constructor has already taken ownership of its own top-level slice copy.

## TDD evidence

The initial focused RED failed exactly on the pre-M88 guard `nested structured inline is not supported`.

Permanent public tests prove:

- emphasis containing mixed semantic text and adaptive code;
- strong containing emphasis with canonical marker alternation;
- emphasis containing strikethrough;
- an exact stable hierarchy at structural depth 64;
- depth 65 rejection with `ErrInvalidConstruction`;
- direct strikethrough-in-strikethrough rejection;
- nested link, image, and autolink rejection;
- existing simple structured-inline and caller-slice snapshot behavior remains green.

Direct Goldmark tests prove exact nested kind/parent/order/delimiter matching, rejection when parent/delimiter/child-set changes, and coexistence with unrelated top-level link nodes outside the M88 hierarchy proof.

During design, temporary diagnostics confirmed an important GFM boundary: delimiter-only deep combinations can become semantically ambiguous well below depth 64, while text-separated hierarchies can remain exact through depth 64. The diagnostic files were removed; permanent tests encode the resulting contract rather than the exploratory probes.

## Devil's advocate review

1. **Construction support could accidentally promote compound existing-source spans as editable.** M88 uses a separate construction-only Goldmark validator and does not change ordinary adapter observations, source mappers, public detail accessors, or mutation APIs.
2. **Recursive wrappers could overflow the stack or permit unbounded structures.** The writer rejects a 65th structured wrapper before descending further; successful recursion is bounded to at most 64 wrappers.
3. **Delimiter choices could parse into a different nested hierarchy.** Marker alternation reduces collisions but is not trusted as proof. Exact Goldmark kind/anchor/parent/child-order validation remains authoritative, so ambiguous GFM fails closed.
4. **Hierarchy validation could become quadratic with many sibling inline nodes.** The final validator walks the AST once, uses anchor buckets for matching, and precomputes parent→children expectation lists once. It does not rescan the complete AST or expectation set per node.
5. **Recursive copying of immutable `Inline` values could add unnecessary CPU/stack growth.** M88 replaces recursive copying with a shallow top-level slice snapshot; nested representation remains inaccessible to callers.

## Verification

Before final documentation gating, focused tests, direct Goldmark hierarchy tests, the complete `go test ./... -count=1` regression, production `gocyclo -over 15 -ignore '_test\.go$' .`, `unparam ./...`, and `unparam -tests ./...` pass after the final single-pass proof refactor.

M88 statement coverage is 92.2% for the root package, 72.1% for `internal/parser/goldmark`, 79.2% for `internal/source`, 57.2% for `internal/splice`, and 71.6% aggregate. The parser interface package reports 0.0% because it has no executable test target.

The fully documented M88 tree passed the strict completion gate: five consecutive `go test ./... -count=1` runs, `go test -race ./... -count=1`, statement coverage, `go vet ./...`, `go build ./...`, public `go doc` resolution for `EmphasisInline`, `StrongInline`, and `StrikethroughInline`, hash-pinned published GFM 0.29 conformance, Staticcheck, golangci-lint with zero issues, production `gocyclo` at the <=15 threshold, production and test-inclusive `unparam`, `govulncheck` with no vulnerabilities found, Gitleaks with no leaks, strict UTF-8/no-BOM/LF/no-trailing-whitespace hygiene over 238 repository text files, `git diff --check`, and `git fsck --no-dangling`.

The verified repository state remained branch `main` at HEAD `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured remotes and the M63–M88 work intentionally uncommitted.

## Exit decision

M88 completes the first structured typed-inline nesting slice without widening existing-source editability. Reference-style typed links/images, bare/extended autolink generation, and structured link/image label/alt nesting remain separate future reviews.
