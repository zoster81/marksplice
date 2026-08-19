# Milestone M6 — Simple Inline Public API

Status: green — simple inline public API passed.

## Goal

Apply the M5 parse-time editable-capability gate to four simple inline families already proven by M1: GFM strikethrough, code spans, emphasis, and strong emphasis.

M6 promotes only source shapes whose exact editable content mapping is known in the immutable parsed snapshot. Semantically recognized but lexically unsupported inline shapes remain internal and fail closed rather than appearing as public actionable nodes.

## Scope

M6 includes four public slices:

- `Strikethrough` typed detail plus `PrepareReplaceStrikethrough`;
- `CodeSpan` typed detail plus `PrepareReplaceCodeSpan`;
- `Emphasis` typed detail plus `PrepareReplaceEmphasis`;
- `Strong` typed detail plus `PrepareReplaceStrong`.

Each detail exposes only `ID()` and `Range()`. The range is the exact inline content span replaced by its named operation. Delimiter characters, delimiter-run lengths, and other lexical preservation facts remain internal.

Autolinks, ordinary links, reference definitions, front matter, and HTML remain outside M6.

## Internal mapping model

During `Parse`, M6 attempts the existing M1 source mapper for each semantic observation:

- `source.MapSimpleStrikethrough`;
- `source.MapSimpleCodeSpan`;
- `source.MapSimpleEmphasis` at level 1 or 2.

On success, the immutable node stores the validated original mapping and is marked `Editable=true`. Expected unsupported-shape errors retain the semantic node with `Editable=false` and do not make parsing fail.

Mutation preparation reuses the stored original mapping. Candidate replacement reparsing and remapping remain unchanged as the M1 fail-closed safety oracle.

## Public contracts

### Strikethrough

`Strikethrough.Range()` is the exact plain-text content span replaced by `PrepareReplaceStrikethrough`. Existing one- or two-tilde delimiters remain outside the range and are preserved.

### CodeSpan

`CodeSpan.Range()` is the exact content span replaced by `PrepareReplaceCodeSpan`. The backtick-run length remains private source trivia used only for validation.

Simple single-line spans whose semantic content maps directly to the source are promoted. Shapes requiring CommonMark/GFM code-span whitespace normalization or multiline handling remain non-editable and unpromoted.

### Emphasis and Strong

`Emphasis.Range()` and `Strong.Range()` are the exact plain-text content spans replaced by their named operations. Existing `*` versus `_` choice and one- versus two-character delimiter runs remain private source data.

Compound/nested delimiter shapes such as `***text***` remain semantic internal observations but are not promoted unless the simple mapper proves the individual node editable.

## Error and preservation contract

The existing public error categories remain unchanged:

- missing ID: `ErrNodeNotFound`;
- wrong or non-editable target: `ErrInvalidTargetKind`;
- unsafe replacement: `ErrInvalidReplacement`;
- stale application: `ErrSourceConflict`.

Every public source-preservation test verifies byte identity outside the typed detail's exact `Range()`.

## Architecture and complexity

M6 introduces no parser configuration, dependency, generic inline abstraction, or new patch engine. `Node` remains ID + kind only. The four typed details are intentionally separate so caller-facing semantics can evolve independently without exposing the M1 node union.

Persisting original inline mappings removes repeated rescans of the immutable original source for these operations. Candidate reparsing remains O(n) per prepared mutation and is intentionally unchanged.

## Devil's advocate review

### Risk: code-span semantic content differs from literal source

CommonMark/GFM code-span rules may normalize spaces, making a semantic range unsuitable as a direct replacement span.

Mitigation: only `MapSimpleCodeSpan` success marks a node editable. Normalized-space and multiline shapes stay internal/non-editable.

### Risk: compound emphasis is falsely promoted

Nested delimiter runs can produce semantic emphasis/strong observations whose lexical boundaries are not the simple one/two-delimiter contract.

Mitigation: `MapSimpleEmphasis` must independently prove the exact delimiter shape; expected failure leaves the observation non-editable.

### Risk: exposing delimiter style freezes lexical trivia

Publishing marker or run-length fields would make preservation implementation details part of the compatibility contract without a demonstrated caller need.

Mitigation: M6 exposes only operation-oriented content ranges. Delimiter facts remain stored internally for validation.

### Risk: four public types add repetitive wrapper code

A premature generic inline type would reduce lines at the cost of coupling unrelated future semantics.

Mitigation: keep tiny typed wrappers and share only internal target validation and mapping workflow. Revisit abstraction only if later families demonstrate a durable common public contract.

## Exit decision

M6 is green. Internal tests prove parse-time editable mappings for supported strikethrough, code-span, emphasis, and strong shapes while normalized-space code spans and compound emphasis observations remain parseable with `Editable=false` and zero stored mappings. Public tests prove all four typed details, exact operation ranges, source preservation, unsupported-shape filtering, and stable public error categories.

The complete Go regression and race suites pass; vet, Staticcheck, golangci-lint, govulncheck, and Gitleaks report no findings; generated package documentation exposes only Marksplice-owned or standard-library public types.

M6 reuses the M5 capability boundary and removes repeated original-source mapping from these four mutation paths. Candidate reparsing and remapping remain the conservative M1 fail-closed oracle. No Goldmark configuration, GFM profile, dependency, generic inline abstraction, or NodeID algorithm changed.
