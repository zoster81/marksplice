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
- `docs/milestones/m10-bounded-source-reading.md` owns the completed M10 snapshot-bound source-range reading contract, evidence, and exit decision.
- `docs/milestones/m11-image-public-api.md` owns the completed M11 simple-image parse-time mapping, public API scope, evidence, and exit decision.
- `docs/milestones/m12-section-removal.md` owns the completed M12 section-subtree removal contract, surviving-heading validation, evidence, and exit decision.
- `docs/milestones/m13-section-body-replacement.md` owns the completed M13 direct-section-body replacement contract, shared section-mutation validator, evidence, and exit decision.
- `docs/milestones/m14-section-subtree-replacement.md` owns the completed M14 complete-section-subtree replacement contract, standalone-fragment proof, evidence, and exit decision.
- `docs/milestones/m15-section-sibling-insertion.md` owns the completed M15 same-level section-sibling insertion contract, zero-width boundary validation, evidence, and exit decision.
- `docs/milestones/m16-section-subtree-move.md` owns the completed M16 atomic same-level section-subtree move contract, coordinated-patch validation, evidence, and exit decision.
- `docs/milestones/m17-section-child-append.md` owns the completed M17 direct-child section append contract, parent+1 fragment invariant, consolidation record, evidence, and exit decision.
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

M1 through M17 are complete. M1 records the feasibility proof; M2 records the first durable public API foundation; M3 applies the typed-detail/named-operation pattern to top-level headings; M4 applies it to M1-proven single-line list items and GFM task markers; M5 adds a parse-time editable-capability gate and promotes mapped non-empty table cells and supported single-line fenced code; M6 applies the same gate to simple strikethrough, code spans, emphasis, and strong; M7 applies it to simple inline links, single-line reference definitions, and supported GFM autolinks; M8 promotes M1-proven simple leading front-matter scalar fields plus simple HTML comments/anchors while preserving opaque HTML; M9 adds an O(h) read-only section hierarchy with exact direct-body/subtree ranges anchored to heading IDs; M10 adds copied snapshot-bound source reads for any valid public range, including section body/subtree ranges; M11 promotes simple single-line inline-image destinations through the same parse-time mapped-capability gate without exposing Goldmark image internals; M12 deletes exactly one complete M9 section subtree and fails closed if the new join changes surviving heading structure; M13 replaces only a section's direct body while preserving its heading/subsections; M14 replaces one complete section subtree with a validated standalone subtree at the same root level; M15 inserts validated same-level sibling subtrees immediately before or after an anchor section; M16 atomically moves one complete subtree before/after a same-level anchor with coordinated delete/insert patches and one combined candidate validation; M17 appends one validated direct-child subtree at a parent boundary with an exact parent+1 level invariant. Future milestones must extend the model from the established source-preservation, parser-isolation, stale-source, typed-detail, mapped-capability, section-hierarchy, bounded-reading, structural-mutation, validated-fragment, coordinated-patch, explicit-parentage, and fail-closed invariants rather than bypassing them.

Do not add Scripthold-specific MCP, filesystem authorization, preview/apply, release, or workspace-crawling behavior to Marksplice core.

## Language and style

Use English for code, identifiers, comments, documentation, commit messages, and repository metadata. Keep packages small, errors explicit, tests deterministic, and dependencies minimal.
