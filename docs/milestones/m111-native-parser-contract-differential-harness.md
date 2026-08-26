# Milestone M111 — Native Parser Contract and Differential Harness

Status: complete — parser contract, differential harness, pinned-corpus integration, focused parity regressions, and release-quality verification are green.

## Goal

Freeze the complete parser-independent semantic contract actually consumed by Marksplice before native parsing begins, and provide one reusable differential harness that can compare the temporary Goldmark oracle with the M112/M113 native candidate without changing public APIs or source-preservation authority.

M111 does not implement native Markdown grammar. M112 remains the first native-parser implementation milestone.

## Requirements

The M111 contract is:

- every runtime parser dependency consumed by `internal/splice` is represented by Marksplice-owned types in `internal/parser`;
- document parsing returns one immutable `DocumentObservations` value containing core nodes, resolved link usages, conservative unresolved reference usages, footnote definitions/references, and mathematical observations;
- construction-only semantic proof for nested blockquotes, typed-inline hierarchy, direct links/images, and reference links/images uses the same parser-independent backend boundary;
- construction reference resolution and reference-label normalization are part of the frozen semantic contract rather than hidden Goldmark calls;
- backend implementations must not mutate or retain caller source/expectation slices after return;
- byte ranges remain half-open offsets into the exact supplied source snapshot;
- successful observations/proofs are deterministic for identical input;
- production Marksplice retains no parser AST/context in immutable documents;
- no Goldmark type crosses `internal/parser/goldmark`;
- public API, `Kind` ordinals, source ownership, mutation authority, `DocumentBuilder` output, M110 extension behavior, and dependency versions remain unchanged.

## Internal backend contract

`internal/parser.Backend` now owns the complete temporary-parser substitution surface:

```go
type Backend interface {
    ParseDocument(source []byte) (DocumentObservations, error)
    ValidateNestedBlockquoteBlocks(source []byte, outer Range, innerSource []byte, depth int) error
    ValidateNestedBlockquoteParagraph(source []byte, outer Range, contentLines []Range, depth int) error
    ValidateConstructionInlineHierarchy(source []byte, expected []ConstructionInlineExpectation, references []ConstructionReferenceInlineExpectation) error
    ValidateConstructionLinkImages(source []byte, expected []ConstructionLinkImageExpectation) error
    ValidateConstructionReferenceInlines(source []byte, expected []ConstructionReferenceInlineExpectation) error
    ResolveConstructionReference(label string, definitions []ConstructionReferenceDefinition) (ConstructionReferenceDefinition, error)
    ReferenceLabelKey(label string) string
}
```

The interface is intentionally internal. It freezes what Marksplice consumes, not a third-party parser plugin API.

`internal/parser/goldmark.Adapter` implements this contract while Goldmark remains the temporary backend. `internal/splice/parser_backend.go` is the single direct Goldmark bridge inside `internal/splice`; ordinary parse, mutation candidate parse, construction proof, and construction reference resolution otherwise consume only `parser.Backend`.

Reference-label normalization keeps a small direct bridge function rather than constructing a complete parser instance for every normalization lookup. This preserves the centralized dependency boundary without adding avoidable parser allocation/work to relationship-heavy paths.

## Document-model injection boundary

`internal/splice.Parse` now delegates to private `parseWithBackend`. The injected path validates a non-nil backend and then calls a separate parse orchestrator so the production cyclomatic complexity gate remains at or below 15.

The Marksplice model builder consumes only `DocumentObservations`; parser-specific AST/context state cannot enter the retained snapshot. Focused tests prove an injected fake backend can construct a normal Marksplice document and that a nil backend fails deterministically.

## Differential harness

`internal/parser/differential.Harness` compares two `parser.Backend` implementations across:

- complete `DocumentObservations` in semantic source order;
- nested blockquote block/paragraph proof success versus failure;
- typed-inline hierarchy proof success versus failure;
- direct link/image proof success versus failure;
- reference inline proof success versus failure;
- construction reference resolution result/error parity;
- reference-label normalization keys.

For observation vectors, nil and empty slices are treated as equivalent because they encode the same absence of semantic facts. Scalar/range/order differences are not normalized away.

Each backend receives independent copies of mutable inputs. The harness verifies that source bytes, construction expectations, reference-definition vectors, nested inner source, and range vectors remain unchanged after each call. This prevents one backend from contaminating the other's input and makes mutation of caller-owned test input a differential-contract failure.

## Shared pinned GFM corpus

M111 extracts the published GFM snapshot loader into test-only `internal/testutil/gfmspec`.

This package is now the single repository authority for:

- the approved published-snapshot SHA-256;
- extraction/decoding of published examples;
- extension-section classification;
- the expected corpus shape.

Both the existing Goldmark conformance test and the differential harness use the same loader. The approved snapshot contains 677 examples: 649 core, 8 tables, 2 task-list, 3 strikethrough, 14 autolink, and 1 tagfilter example. The parser differential gate compares all 676 parser-applicable examples; tagfilter remains renderer-only and is intentionally outside Marksplice's parser contract.

The published specification snapshot is external test data and is not copied into the module.

## Focused semantic/source-position parity

The published specification corpus is necessary but not sufficient because Marksplice consumes source positions and reviewed overlays that are more specific than rendered-GFM conformance.

M111 therefore adds a focused parser-neutral differential corpus covering the same families already protected by explicit Goldmark oracle assertions, including:

- GFM compatibility guards for strikethrough/autolinks/comments;
- resolved and unresolved reference forms;
- thematic-break ownership;
- simple and lazy-continuation blockquotes;
- table ownership, alignments, and empty-cell column accounting;
- empty/indented fenced-container anchors, info/language, and payload ranges;
- raw/HTML block boundaries including CRLF;
- simple/compound image promotion boundaries;
- list container/parent promotion and rejected trailing-block shapes;
- footnote definition/reference source order and relationship reconciliation;
- links inside footnote definitions;
- mathematical overlay ownership/conflict suppression;
- Setext/CRLF source behavior.

Construction parity also includes accepted and rejected cases for nested blockquote depth/child semantics, typed-inline delimiter hierarchy, direct link destination semantics, reference-token semantics, normalized ambiguity, and label-key normalization.

The existing Goldmark-specific tests remain valuable explicit oracle assertions. The differential suite adds the reusable candidate comparison layer that M112/M113 need.

## Complexity and retained state

M111 adds no public runtime state and no persistent parser comparison state.

- production parse complexity remains the existing source/model work plus one backend call;
- differential comparisons are test-only;
- the shared GFM loader is test-only;
- no second AST, parser context, corpus cache, or compatibility index is retained by production documents;
- `internal/splice` has exactly one direct import of `internal/parser/goldmark` after the refactor;
- the production complexity gate remains `gocyclo <= 15`; the first injected parse wrapper measured 16 after adding nil validation, so validation was separated from the existing orchestration instead of weakening the gate.

## TDD and implementation evidence

M111 began with missing-contract RED tests for `parser.Backend`, backend injection, and the differential harness. The implementation then centralized the Goldmark bridge, shared one published-GFM corpus loader, added complete backend observation/proof parity, verified caller-input immutability, and refactored the injected parse path back below the production complexity limit.

## Devil's advocate review

1. **A supposedly parser-independent cutover could leave hidden Goldmark calls in mutation or construction proof.** Mitigation: the backend contract includes both document observations and every construction/reference semantic primitive consumed by `internal/splice`; direct Goldmark imports there are reduced to one explicit bridge file and verified by test tooling.
2. **Differential comparison could hide backend mutation by passing the same input to oracle and candidate.** Mitigation: the harness copies mutable inputs independently, verifies post-call immutability, and has a regression proving mutation is rejected without changing the caller source.
3. **A separate differential GFM loader could drift from the normative conformance gate.** Mitigation: conformance and differential tests share `internal/testutil/gfmspec`, including one approved hash and one example extractor/classifier.
4. **Rendered GFM parity alone could miss Marksplice source-position semantics.** Mitigation: keep explicit Goldmark oracle assertions and add a focused parser-neutral differential corpus over the source-position/overlay families Marksplice consumes.
5. **Freezing a broad public parser interface could create a compatibility burden.** Mitigation: `parser.Backend` is internal and represents only implementation substitution; M110 remains the public third-party extensibility boundary.
6. **M111 could accidentally turn into native parser implementation.** Mitigation: no native grammar package or parser algorithm is introduced; M112 remains the explicit native block-parser milestone.

## Release-quality verification

The source-of-truth-aligned M111 tree passed repeated complete regressions, the focused differential suite, race detection, `go vet`, `go build`, Staticcheck, golangci-lint, production `gocyclo <= 15`, production/test-inclusive unparam, `go mod tidy -diff`, published GFM 0.29 conformance, all **676 parser-applicable differential corpus cases**, focused source-position/proof parity, backend input-immutability checks, Go 1.27 test/vet/build, govulncheck, Gitleaks, actionlint, strict text/artifact hygiene, and `git diff --check`.

Cross-package statement coverage measured **87.1% aggregate** and **84.7% through `internal/publictest`**.

## Exit decision

M111 is complete. The full parser-independent substitution contract is frozen, Goldmark is centralized as the temporary backend/oracle, conformance and differential tests share one pinned-corpus loader, and the reusable harness covers parser observations, source-position regressions, construction proofs, reference semantics, and input immutability. The roadmap boundary advances to **M112 — Marksplice-native block parser**; M112 must preserve this contract rather than widening or bypassing it.
