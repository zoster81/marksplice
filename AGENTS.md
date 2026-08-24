# Marksplice Repository Guide

This repository contains the public `github.com/zoster81/marksplice` Go library.

## Sources of truth

Keep responsibilities separated and link rather than duplicate details:

- `docs/architecture.md` owns durable architecture, API-boundary, source-preservation, security, dependency, performance, and complexity decisions.
- `docs/gfm-conformance.md` owns the normative Markdown profile, GFM source hierarchy, conformance gate, and specification-update policy.
- `docs/capabilities.md` owns the current product-facing read/edit/create capability matrix and forward roadmap; it does not supersede architecture, conformance, parser/source ownership, or milestone evidence.
- `docs/goldmark-capability-matrix.md` owns the Goldmark-versus-Marksplice parser/source responsibility boundary.
- `docs/extension-strategy.md` owns selection of broadly useful core capabilities from wider Markdown ecosystem ideas, the future third-party extensibility/SPI boundary, and the mandatory Goldmark-exit/native-parser cutover gate.
- Each `docs/milestones/mNN-*.md` file owns the detailed contract, design record, verification evidence, and exit decision for that milestone. Historical gate results in milestone files remain historical evidence and should not be rewritten merely because a later gate becomes stricter.
- Milestone families are grouped only for navigation: M0 bootstrap; M1–M11 feasibility/public mapped capabilities; M12–M17 sections; M18–M34 list hierarchy plus section-child navigation; M35–M43 table row/cell model; M44–M62 new-document construction; M63–M70 public table/alignment/column editing; M71–M74 thematic-break/simple-blockquote promotion and removal; M75–M79 typed-inline construction; M80 front-matter construction; M81–M82 broader single-paragraph blockquote construction; M83–M86 multi-block/recursive blockquote construction; M87 typed link/image title construction; M88 bounded structured typed-inline nesting; M89 typed full-reference link/image construction; M90 first-public-beta readiness; M91 repository-layout hardening; M92 structured link/image label/alt composition; M93 reference/autolink completion and reference-definition lifecycle; M94 existing-source blockquote completion; M95 structural mutation composition; M96 single-document read/edit/create audit; M97 structural query surface; M98 anchors/fragments/TOC; M99 single-document link intelligence; M100 explicit multi-document graph; M101 workspace validation and repair planning; M102 semantic block patterns; M103 fenced-block semantics.
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
- Goldmark is the current temporary semantic parser, configured for GFM, and must remain an implementation detail until its mandatory M115 removal. Keep it behind `internal/parser/goldmark` and never expose Goldmark AST nodes or Goldmark-specific types from Marksplice public APIs.
- Marksplice owns its lossless source mapping, source fingerprints, structural identities, and minimal patch generation.
- Prepared mutations must be bound to the exact source snapshot and fail closed on stale input.
- Prefer deterministic byte offsets and `[]byte` transformations for source patches.
- Keep parsing/indexing linear or near-linear where practical and avoid repeated whole-document rescans in edit batches. M95 composition may perform O(k·N) proof work for `k` explicitly supplied prepared changes because it compares each independently validated candidate model and one combined candidate. M97 structural queries may perform O(N) source-ordered scans with explicit positive result limits and O(limit) output. M99 relationship projection uses only ephemeral source-node/definition/fragment maps and expected O(N+H+R) work over nodes, headings, and relationships. M100 graph build stores each resolved edge once plus adjacency edge indices, reuses at most one build-local fragment catalog per cross-fragment target document, and performs no document discovery beyond the explicit caller-provided set. M101 workspace validation invokes caller resolution once per non-local relationship, computes orphan reachability with one multi-source BFS, batches managed-heading/TOC derivation per document, and retains no resolver, fragment catalog, heading set, traversal state, or TOC cache after return. M102 semantic alert recognition is a call-local O(N+L) projection over already-promoted blockquotes and retains no alert index or parser state. M103 fenced-block source proof is O(F) in the physical lines owned by the container, keeps at most one source-backed payload range per semantic body line, reuses the existing source-ordered node scan, and retains no persistent fence/language index or embedded-language parser state. Do not add nested all-pairs matching, redundant graph copies, hidden crawling, persistent query/proof indexes, or hidden global result caps without measured evidence.
- Core code must not perform arbitrary network requests, command execution, or filesystem traversal.
- M1 has proved the lossless-editing feasibility gate; do not promote the feasibility internals wholesale into a stable public API. Design post-M1 APIs from the proven architecture and source-preservation invariants.

## Scope

M0 is the green repository-bootstrap baseline and M1–M103 are complete engineering milestones. Their detailed contracts and historical evidence live only in the milestone records; current product status belongs in `docs/capabilities.md`.

Future work must extend, not bypass, the established invariants: source preservation, parser isolation, snapshot-bound identity and stale-source safety, explicit source ownership, mapped-capability promotion, host/candidate validation for structural edits, compact derived adjacency instead of redundant persistent indexes, canonical writing only for new source, parser/model proof of generated source, deterministic failure on ambiguous shapes, and bounded complexity.

The current production complexity gate is cyclomatic complexity **15 or lower per function** (`gocyclo -over 15` must report no production function), with production and test-inclusive `unparam` checks. Do not split cohesive lexical/state-machine logic merely to lower a metric; refactor when responsibility, reuse, or verification clarity improves.

Do not add Scripthold-specific MCP, filesystem authorization, preview/apply, release, or workspace-crawling behavior to Marksplice core. Marksplice does not maintain first-party dialect extensions; broadly useful capabilities belong in core, while dialect/product syntax is reserved for the future explicit M110 third-party SPI.

## Language and style

Use English for code, identifiers, comments, documentation, commit messages, and repository metadata. Keep packages small, errors explicit, tests deterministic, and dependencies minimal.
