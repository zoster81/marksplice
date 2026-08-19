# Marksplice Repository Guide

This repository contains the public `github.com/zoster81/marksplice` Go library.

## Sources of truth

Keep responsibilities separated and link rather than duplicate details:

- `docs/architecture.md` owns durable architecture, API-boundary, source-preservation, security, dependency, and performance decisions.
- `docs/gfm-conformance.md` owns the normative Markdown profile, GFM source hierarchy, conformance gate, and specification-update policy.
- `docs/milestones/m1-lossless-editing.md` owns the completed M1 feasibility evidence, matrix, consolidation record, and exit decision.
- `docs/milestones/m2-public-api-foundation.md` owns the completed M2 public-API foundation, constraints, evidence, and exit decision.
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

M1 and M2 are complete. M1 records the feasibility proof; M2 records the first durable public API foundation. Future milestones must extend the model from the established source-preservation, parser-isolation, stale-source, typed-detail, and fail-closed invariants rather than bypassing them.

Do not add Scripthold-specific MCP, filesystem authorization, preview/apply, release, or workspace-crawling behavior to Marksplice core.

## Language and style

Use English for code, identifiers, comments, documentation, commit messages, and repository metadata. Keep packages small, errors explicit, tests deterministic, and dependencies minimal.
