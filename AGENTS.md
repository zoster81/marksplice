# Marksplice Repository Guide

This repository contains the public `github.com/zoster81/marksplice` Go library.

## Sources of truth

Keep responsibilities separated and link rather than duplicate details:

- `docs/architecture.md` owns durable architecture, API-boundary, source-preservation, security, dependency, performance, and complexity decisions.
- `docs/gfm-conformance.md` owns the normative Markdown profile, GFM source hierarchy, conformance gate, and specification-update policy.
- `docs/capabilities.md` owns the current product-facing read/edit/create capability matrix and forward roadmap; it does not supersede architecture, conformance, parser/source ownership, or milestone evidence.
- `docs/goldmark-capability-matrix.md` owns the Goldmark-versus-Marksplice parser/source responsibility boundary.
- Each `docs/milestones/mNN-*.md` file owns the detailed contract, design record, verification evidence, and exit decision for that milestone. Historical gate results in milestone files remain historical evidence and should not be rewritten merely because a later gate becomes stricter.
- Milestone families are grouped only for navigation: M0 bootstrap; M1–M11 feasibility/public mapped capabilities; M12–M17 sections; M18–M34 list hierarchy plus section-child navigation; M35–M43 table row/cell model; M44–M62 new-document construction; M63–M70 public table/alignment/column editing; M71–M74 thematic-break/simple-blockquote promotion and removal; M75–M79 typed-inline construction; M80 front-matter construction; M81–M82 broader single-paragraph blockquote construction; M83–M86 multi-block/recursive blockquote construction; M87 typed link/image title construction; M88 bounded structured typed-inline nesting; M89 typed full-reference link/image construction; M90 first-public-beta readiness; M91 repository-layout hardening.
- `go.mod` and `go.sum` own exact Go dependency versions.
- `LICENSE` and `NOTICE` own licensing and project attribution.
- `CONTRIBUTING.md` owns contributor workflow and the current local verification commands.
- `docs/README.md` owns the tracked documentation index and repository-layout map.
- `docs/releasing.md` owns public module versioning, beta-release policy, first-push readiness, and publication verification.
- `CHANGELOG.md` owns public release notes; `SECURITY.md` owns vulnerability-reporting guidance.

Private operator state and private toolchain details are not repository content and must never be copied or linked into tracked documentation.

## Engineering rules

For substantive code, debugging, refactoring, architecture, performance, security, or testing work, use four phases:

1. requirements and edge cases;
2. architecture and test strategy;
3. devil's advocate review with at least two concrete failure risks and mitigations;
4. implementation and verification.

Prefer focused TDD: reproduce with a failing test, confirm the expected failure, make the smallest correct change, rerun the focused test, then run relevant regressions.

Before editing, inspect surrounding context and preserve unrelated changes, encoding, BOM, and line endings. Review the complete diff after editing. Run applicable formatting, tests, race tests, vet/static analysis, vulnerability checks, builds, `git diff --check`, and Git status. Report exactly what was and was not verified.

On Windows, when PowerShell is required, use `pwsh` (PowerShell 7+) whenever it is available and take advantage of its current capabilities rather than artificially restricting commands to Windows PowerShell compatibility. Use Windows PowerShell 5.1 (`powershell.exe`) only when a task specifically requires 5.1 compatibility or as the fallback when `pwsh` is unavailable. When automation starts an auxiliary user-visible window that does not require interaction, prefer starting it minimized when the launcher or application supports that behavior.

## Architecture guardrails

- GitHub Flavored Markdown (GFM) is Marksplice's single normative Markdown syntax profile. Follow `docs/gfm-conformance.md`; do not introduce separate CommonMark/GFM modes or dialect switches in core.
- Existing-document edits are source-preserving. Do not implement ordinary edits by rendering a complete Markdown AST back to source.
- New-document construction may write deterministic canonical GFM because no prior author source exists. Keep construction state separate from immutable parsed `Document` snapshots, and never route ordinary existing-document edits through the construction writer.
- YAML/TOML front matter is a Marksplice-owned document-leading envelope outside the GFM parser. Construction keeps at most one envelope as separate builder state and proves it through the existing source-layer mapper rather than treating it as an ordinary GFM block.
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

M0 is the green repository-bootstrap baseline and M1–M91 are complete engineering milestones. Their detailed contracts and historical evidence live only in the milestone records; current product status belongs in `docs/capabilities.md`.

Future work must extend, not bypass, the established invariants: source preservation, parser isolation, snapshot-bound identity and stale-source safety, explicit source ownership, mapped-capability promotion, host/candidate validation for structural edits, compact derived adjacency instead of redundant persistent indexes, canonical writing only for new source, parser/model proof of generated source, deterministic failure on ambiguous shapes, and bounded complexity.

The current production complexity gate is cyclomatic complexity **15 or lower per function** (`gocyclo -over 15` must report no production function), with production and test-inclusive `unparam` checks. Do not split cohesive lexical/state-machine logic merely to lower a metric; refactor when responsibility, reuse, or verification clarity improves.

Do not add Scripthold-specific MCP, filesystem authorization, preview/apply, release, or workspace-crawling behavior to Marksplice core.

## Language and style

Use English for code, identifiers, comments, documentation, commit messages, and repository metadata. Keep packages small, errors explicit, tests deterministic, and dependencies minimal.
