# Marksplice Repository Guide

This repository contains the public `github.com/zoster81/marksplice` Go library.

## Sources of truth

Keep responsibilities separated and link rather than duplicate details:

- `docs/architecture.md` owns durable architecture, API-boundary, source-preservation, security, dependency, and performance decisions.
- `docs/gfm-conformance.md` owns the normative Markdown profile, GFM source hierarchy, conformance gate, and specification-update policy.
- `docs/milestones/m1-lossless-editing.md` owns the completed M1 feasibility evidence, matrix, consolidation record, and exit decision.
- `docs/milestones/m2-public-api-foundation.md` owns the completed M2 public-API foundation, constraints, evidence, and exit decision.
- `docs/milestones/m3-heading-public-api.md` owns the completed M3 heading-detail and source-preserving rename scope, evidence, and exit decision.
- `docs/milestones/m4-list-public-api.md` owns the completed M4 list-item/task public API scope, evidence, deferred-family rationale, and exit decision.
- `docs/milestones/m5-mapped-block-public-api.md` owns the completed M5 parse-time editable mapping, table-cell/fenced-code public API scope, evidence, and exit decision.
- `docs/milestones/m6-simple-inline-public-api.md` owns the completed M6 simple-inline parse-time mapping, public API scope, unsupported-shape filtering, evidence, and exit decision.
- `docs/milestones/m7-link-public-api.md` owns the completed M7 link parse-time mapping and public API scope, evidence, consolidation record, and exit decision.
- `docs/milestones/m8-metadata-html-public-api.md` owns the completed M8 front-matter/HTML public API scope, parse-time capability evidence, consolidation record, and exit decision.
- `docs/milestones/m9-section-model.md` owns the completed M9 read-only section semantics, hierarchy/range algorithm, evidence, and exit decision.
- `go.mod` and `go.sum` own exact Go dependency versions.
- `LICENSE` and `NOTICE` own licensing and project attribution.
- `CONTRIBUTING.md` owns contributor workflow and validation expectations.

Private operator state and private toolchain details are not repository content and must never be copied or linked into tracked documentation.

## Engineering rules

For substantive code, debugging, refactoring, architecture, performance, security, or testing work, use four phases:

1. requirements and edge cases;
2. architecture and test strategy;
3. devil's advocate review with at least two concrete failure risks and mitigations;
4. implementation and verification.

Prefer focused TDD: reproduce with a failing test, confirm the expected failure, make the smallest correct change, rerun the focused test, then run relevant regressions.

Before editing, inspect surrounding context and preserve unrelated changes, encoding, BOM, and line endings. Review the complete diff after editing. Run applicable formatting, tests, race tests, vet/static analysis, vulnerability checks, builds, `git diff --check`, and Git status. Report exactly what was and was not verified.

On Windows, when PowerShell is required, prefer `pwsh` (PowerShell 7+) when available and use legacy Windows PowerShell (`powershell.exe`) as a compatibility fallback. When automation starts an auxiliary user-visible window that does not require interaction, prefer starting it minimized when the launcher or application supports that behavior.

## Architecture guardrails

- GitHub Flavored Markdown (GFM) is Marksplice's single normative Markdown syntax profile. Follow `docs/gfm-conformance.md`; do not introduce separate CommonMark/GFM modes or dialect switches in core.
- Existing-document edits are source-preserving. Do not implement ordinary edits by rendering a complete Markdown AST back to source.
- Untouched bytes must remain byte-identical whenever the requested semantic operation does not require a broader change.
- Treat line endings, whitespace, delimiters, markers, and other lexical trivia as source data, not formatting to normalize.
- Goldmark is the selected semantic parser, configured for GFM, but it is an implementation detail. Keep it behind `internal/parser/goldmark` and never expose Goldmark AST nodes or Goldmark-specific types from Marksplice public APIs.
- Marksplice owns its lossless source mapping, source fingerprints, structural identities, and minimal patch generation.
- Prepared mutations must be bound to the exact source snapshot and fail closed on stale input.
- Prefer deterministic byte offsets and `[]byte` transformations for source patches.
- Keep parsing/indexing linear or near-linear where practical and avoid repeated whole-document rescans in edit batches.
- Core code must not perform arbitrary network requests, command execution, or filesystem traversal.
- M1 has proved the lossless-editing feasibility gate; do not promote the feasibility internals wholesale into a stable public API. Design post-M1 APIs from the proven architecture and source-preservation invariants.

## Scope

M1 through M9 are complete. M1 records the feasibility proof; M2 records the first durable public API foundation; M3 applies the typed-detail/named-operation pattern to top-level headings; M4 applies it to M1-proven single-line list items and GFM task markers; M5 adds a parse-time editable-capability gate and promotes mapped non-empty table cells and supported single-line fenced code; M6 applies the same gate to simple strikethrough, code spans, emphasis, and strong; M7 applies it to simple inline links, single-line reference definitions, and supported GFM autolinks; M8 promotes M1-proven simple leading front-matter scalar fields plus simple HTML comments/anchors while preserving opaque HTML; M9 adds an O(h) read-only section hierarchy with exact direct-body/subtree ranges anchored to heading IDs. Future milestones must extend the model from the established source-preservation, parser-isolation, stale-source, typed-detail, mapped-capability, section-hierarchy, and fail-closed invariants rather than bypassing them.

Do not add Scripthold-specific MCP, filesystem authorization, preview/apply, release, or workspace-crawling behavior to Marksplice core.

## Language and style

Use English for code, identifiers, comments, documentation, commit messages, and repository metadata. Keep packages small, errors explicit, tests deterministic, and dependencies minimal.
