# Milestone M7 — Link Public API

Status: green — link public API passed.

## Goal

Apply the M5/M6 parse-time editable-capability boundary to the M1-proven link families without exposing M1 internals or Goldmark-specific data.

M7 promotes only link observations whose exact source-preserving mutation boundary is proven while parsing the immutable source snapshot. Semantically recognized but unsupported source shapes remain internal and fail closed.

## Scope

M7 starts with three public slices:

- `InlineLink` typed detail plus `PrepareReplaceInlineLinkDestination`;
- `ReferenceDefinition` typed detail plus `PrepareReplaceReferenceDefinitionDestination`;
- `AutoLink` typed detail plus `PrepareReplaceAutoLink`.

Each detail exposes only `ID()` and `Range()`. The range is the exact destination or autolink content span replaced by the named operation. Labels, titles, destination wrappers, parentheses, angle brackets, indentation, trailing spaces, and line endings remain private source data unless a future independently reviewed API requires them.

Reference-link relationship-wide renaming, images, front matter, and HTML remain outside the initial M7 scope.

## Internal mapping model

During `Parse`, M7 attempts the existing M1 source mapper for each semantic observation:

- `source.MapSimpleInlineLink`;
- `source.MapSingleLineReferenceDefinition`;
- `source.MapAutoLink`.

On success, the immutable node stores the validated original `InlineLinkMapping`, `ReferenceDefinitionMapping`, or `AutoLinkMapping`, sets the exact editable `ContentRange`, and is marked `Editable=true`.

Expected unsupported source shapes remain semantic internal nodes with `Editable=false`; they do not make parsing fail. For example, a simple-label inline link with an empty destination may be semantically observed but is not promoted because the M1 mapper does not prove it as an editable destination shape.

Mutation preparation reuses the stored original mapping instead of rescanning the immutable original source. Candidate replacement reparsing and remapping remain the conservative M1 fail-closed safety oracle.

## Public contracts

### InlineLink

`InlineLink.Range()` is the exact destination content span replaced by `PrepareReplaceInlineLinkDestination`. Raw versus angle-bracket destination form, label source, parentheses, spacing, optional title syntax, and surrounding bytes remain outside the range and are preserved.

Only simple single-line plain-text-label inline links whose destination mapping is proven at parse time are promoted.

### ReferenceDefinition

`ReferenceDefinition.Range()` is the exact destination content span replaced by `PrepareReplaceReferenceDefinitionDestination`. Indentation, label, colon, raw versus angle-bracket destination form, spacing, optional title syntax, trailing spaces, and line endings remain outside the range and are preserved.

Only the M1-proven single-line definition shape is promoted.

### AutoLink

`AutoLink.Range()` is the exact semantic source token content replaced by `PrepareReplaceAutoLink`. Angle brackets remain outside the range when present.

Candidate validation must re-establish the same supported autolink category and source wrapper facts. A replacement that ceases to be the expected GFM autolink fails closed.

## Error and preservation contract

The existing public error categories remain unchanged:

- missing ID: `ErrNodeNotFound`;
- wrong or non-editable target: `ErrInvalidTargetKind`;
- unsafe replacement: `ErrInvalidReplacement`;
- stale application: `ErrSourceConflict`.

Public source-preservation tests verify byte identity outside each typed detail's exact `Range()`.

## Architecture and complexity

M7 introduces no parser configuration, dependency, generic link abstraction, filesystem/network behavior, or new patch engine. Public `Node` remains ID + kind only.

Persisting the three original mappings removes repeated rescans of the immutable source from link mutation preparation. Parsing remains linear or near-linear in source size under the existing model; candidate reparsing remains O(n) per prepared mutation and is intentionally unchanged.

## Devil's advocate review

### Risk: public fields freeze source trivia

Publishing labels, titles, wrapper style, or other lexical facts would make implementation details part of the compatibility contract before callers demonstrate a need.

Mitigation: expose only snapshot identity and the operation-oriented replacement range. Preservation facts stay internal.

### Risk: semantic recognition is mistaken for editability

Goldmark may recognize a link shape whose exact destination boundary is not covered by the M1 mapper.

Mitigation: public promotion requires parse-time mapper success. Expected unsupported mappings remain internal with `Editable=false`.

### Risk: destination replacement changes Markdown structure

A replacement can introduce delimiters, whitespace, or other syntax that changes the link or reference definition rather than only its destination.

Mitigation: the candidate snapshot is reparsed and remapped; wrapper, title, label, position, and source-boundary facts must still match the original supported shape.

### Risk: autolink replacement changes semantic category

A replacement may stop being an autolink or switch to an incompatible category.

Mitigation: candidate validation requires the same expected autolink semantics and mapped wrapper facts before returning a change set.

## Initial evidence

The initial TDD slice first failed at the public boundary because the three kinds and typed operations did not exist. After implementation, focused public tests pass for inline links, reference definitions, and autolinks, including exact replacement ranges, CRLF preservation, angle-wrapper preservation, invalid replacement errors, wrong-target errors, and unsupported-shape filtering.

Focused internal evidence also proves that all three successful source mappings are persisted during `Parse`, while an unsupported empty-destination inline link remains internal/non-editable and is rejected by mutation preparation.

## Exit decision

M7 is green. The three M1-proven link families now use the same parse-time editable-capability boundary established by M5/M6: supported inline links, single-line reference definitions, and GFM autolinks persist their original source mappings, expose narrow typed public details, and reuse those mappings during mutation preparation. Unsupported semantic shapes remain internal/non-editable and fail closed.

The consolidation review found no justified additional link slice inside M7. Reference-link relationship-wide renaming and other graph-level behavior require broader relationship semantics and remain deferred rather than being pulled into this milestone. No generic public link abstraction was introduced because the three named operations retain distinct caller-facing semantics.

The repository checks applicable to the completed milestone pass: `gofmt`, focused public/internal tests, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `staticcheck ./...`, `golangci-lint run`, `govulncheck ./...`, `gitleaks detect`, the approved published-GFM conformance gate, and `git diff --check`.
