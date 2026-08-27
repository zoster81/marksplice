# Marksplice Repository Guide

This repository contains the public `github.com/zoster81/marksplice` Go library.

## Sources of truth

Keep responsibilities separated and link rather than duplicate details:

- `README.md` is the single public entry point. Keep it short: what Marksplice is, why to use it, installation, a minimal flow, and clear links onward.
- `docs/getting-started.md` owns the first-use journey. `docs/guide.md` routes users by goal, `docs/recipes/` owns focused workflows, and `examples/` owns runnable file-based examples used by those workflows.
- `docs/api-reference.md` must cover every exported callable and remains the exhaustive signature/reference document rather than the learning path. Exported API changes update it plus every affected getting-started/guide/recipe/example surface in the same change.
- `docs/capabilities.md` owns the concise current product-facing read/edit/create boundary. It must describe present capability and deliberate limitations, not milestone chronology or development history.
- `docs/roadmap.md` owns the approved post-M115 product/engineering sequence and release targets. Keep planned work there instead of presenting future capability as already shipped.
- `docs/README.md` is the documentation map for readers who already know what they need. It separates user documentation, advanced/maintainer documentation, and historical records; it is not a second public home page.
- `docs/architecture.md` owns durable architecture, API-boundary, source-preservation, security, dependency, performance, and complexity decisions.
- `docs/gfm-conformance.md` owns the normative Markdown profile, GFM source hierarchy, conformance gate, and specification-update policy.
- `docs/extension-strategy.md` owns selection of broadly useful core capabilities from wider Markdown ecosystem ideas and the reviewed third-party extensibility/SPI boundary.
- Each `docs/milestones/mNN-*.md` file owns its historical contract, design record, verification evidence, and exit decision. Historical gate results remain historical evidence and should not be rewritten merely because a later gate becomes stricter.
- `docs/goldmark-capability-matrix.md` is historical pre-M115 parser-transition evidence; current parser architecture belongs to `docs/architecture.md` and `docs/gfm-conformance.md`.
- `go.mod` and `go.sum` own exact Go dependency versions.
- `LICENSE` and `NOTICE` own licensing and project attribution.
- `CONTRIBUTING.md` owns contributor workflow and current local verification commands.
- `docs/releasing.md` owns public module versioning, beta-release policy, release readiness, and publication verification.
- `CHANGELOG.md` owns public release notes; `SECURITY.md` owns vulnerability-reporting guidance.

Documentation should reveal complexity progressively. Do not copy milestone-by-milestone development history into README, Getting Started, recipes, or the capability matrix. Preserve that history in the milestone/advanced records and link to it only when it helps the reader's current task.

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

- Marksplice exposes one Markdown syntax profile: CommonMark 0.31.2 is the normative base grammar, with the published GFM specification layered on top only for explicit GFM extensions/corrections. When inherited GFM 0.29 core behavior conflicts with CommonMark 0.31.2, CommonMark 0.31.2 wins. Follow `docs/gfm-conformance.md`; do not introduce separate CommonMark/GFM modes or dialect switches in core.
- Existing-document edits are source-preserving. Do not implement ordinary edits by rendering a complete Markdown AST back to source.
- New-document construction may write deterministic canonical GFM because no prior author source exists. Keep construction state separate from immutable parsed `Document` snapshots, and never route ordinary existing-document edits through the construction writer.
- YAML/TOML front matter is a Marksplice-owned document-leading envelope outside the GFM parser. M106 separates envelope recognition from editable-field promotion: a closed byte-zero envelope may be readable even when all metadata values remain opaque, while only unique source-proven simple top-level scalar fields can become mutation targets. Empty leading envelopes have explicit metadata precedence; non-empty delimiter pairs without conservative metadata evidence remain GFM. TOML field promotion stops when table scope begins. Construction keeps at most one conservative simple-field envelope as separate builder state and proves it through the existing source-layer mapper rather than treating it as an ordinary GFM block.
- Untouched bytes must remain byte-identical whenever the requested semantic operation does not require a broader change.
- Treat line endings, whitespace, delimiters, markers, and other lexical trivia as source data, not formatting to normalize.
- `internal/parser/native` is the production semantic parser behind the parser-independent `internal/parser.Backend` substitution boundary. M115 removed the former Goldmark adapter, differential harness, compatibility implementation, and module dependency after the M114 conformance/hardening gate and an M115 dual-proof cutover. Parser implementation behavior is not a normative source: classify mismatches against CommonMark 0.31.2, explicit GFM rules, or reviewed Marksplice contracts. Never expose parser-internal types from Marksplice public APIs.
- Marksplice owns its lossless source mapping, source fingerprints, structural identities, and minimal patch generation.
- Prepared mutations must be bound to the exact source snapshot and fail closed on stale input.
- Prefer deterministic byte offsets and `[]byte` transformations for source patches.
- Keep parsing/indexing linear or near-linear where practical and avoid repeated whole-document rescans in edit batches. M95 composition may perform O(k·N) proof work for `k` explicitly supplied prepared changes because it compares each independently validated candidate model and one combined candidate. M97 structural queries may perform O(N) source-ordered scans with explicit positive result limits and O(limit) output. M99 relationship projection uses only ephemeral source-node/definition/fragment maps and expected O(N+H+R) work over nodes, headings, and relationships. M100 graph build stores each resolved edge once plus adjacency edge indices, reuses at most one build-local fragment catalog per cross-fragment target document, and performs no document discovery beyond the explicit caller-provided set. M101 workspace validation invokes caller resolution once per non-local relationship, computes orphan reachability with one multi-source BFS, batches managed-heading/TOC derivation per document, and retains no resolver, fragment catalog, heading set, traversal state, or TOC cache after return. M102 semantic alert recognition is a call-local O(N+L) projection over already-promoted blockquotes and retains no alert index or parser state. M103 fenced-block source proof is O(F) in the physical lines owned by the container, keeps at most one source-backed payload range per semantic body line, reuses the existing source-ordered node scan, and retains no persistent fence/language index or embedded-language parser state. M104 keeps footnotes outside the normative GFM grammar; the Native backend integrates the reviewed footnote observation pass, normalizes claimed definition regions for ordered lookup, retains only promoted definition nodes plus a compact source-ordered reference vector, and adds no persistent label/reverse-reference index or separate graph. M105 keeps the same normative parser profile, recognizes dedicated mathematical source forms through bounded Marksplice-owned observation/source proof, linearly merges source-ordered results, reuses M103 identity for exact-info `math` fences, and retains no mathematical parser/index/rendering state. M106 keeps front-matter mapping source-linear, retains only the compact envelope boundaries plus already-promoted safe field nodes, and adds no YAML/TOML AST, schema, persistent key index, or serializer state. M107 layers exact caller-provided aliases/tags/logical references over the existing immutable M100 graph, stores each logical reference once plus edge-index adjacency, reuses M100 traversal helpers, retains one exact alias lookup, and adds no second graph, parser/syntax authority, I/O, or source mutation. Do not add nested all-pairs matching, redundant graph copies, hidden crawling, persistent query/proof indexes, or hidden global result caps without measured evidence.
- Core code must not perform arbitrary network requests, command execution, or filesystem traversal.
- Immutable `Document`, `DocumentGraph`, `KnowledgeIndex`, `WorkspaceReport`, and prepared `ChangeSet` values are concurrent-read safe; keep variable-length public results caller-owned. `DocumentBuilder` remains mutable/caller-synchronized, and graph/workspace resolver callbacks remain synchronous, serial within one call, and non-retained.
- Public sentinel failures are classified with `errors.Is`; diagnostic strings are not compatibility contracts. Structural queries keep explicit positive result limits, graph/workspace/knowledge operations remain bounded by caller-supplied finite document sets, and do not add a nominal `context.Context` contract that the active parser cannot honor.
- Public `Kind` is the closed Marksplice core structural namespace. M110 extension identities are separately namespaced/typed and cannot consume, replace, or register core `Kind` ordinals. Extension recognizers are opt-in serial/non-retained read-only overlays; validate/bound retained observations, never grant raw patch/builder/parser/graph/host authority, and do not claim to sandbox ordinary caller-linked Go code.
- M1 has proved the lossless-editing feasibility gate; do not promote the feasibility internals wholesale into a stable public API. Design post-M1 APIs from the proven architecture and source-preservation invariants.

## Scope

M0 is the green repository-bootstrap baseline and M1–M115 are complete engineering milestones. The native-parser replacement roadmap is complete at M115. The approved post-M115 sequence is M116–M124 toward `v1.0.0`, with M125–M126 queued for the `v1.5.0` PDF line; `docs/roadmap.md` owns that future sequence. Detailed completed-milestone contracts and historical evidence live only in the milestone records; current shipped product status belongs in `docs/capabilities.md`.

Future work must extend, not bypass, the established invariants: source preservation, parser isolation, snapshot-bound identity and stale-source safety, explicit source ownership, mapped-capability promotion, host/candidate validation for structural edits, compact derived adjacency instead of redundant persistent indexes, canonical writing only for new source, parser/model proof of generated source, deterministic failure on ambiguous shapes, and bounded complexity.

The current production complexity gate is cyclomatic complexity **15 or lower per function** (`gocyclo -over 15` must report no production function), with production and test-inclusive `unparam` checks. Do not split cohesive lexical/state-machine logic merely to lower a metric; refactor when responsibility, reuse, or verification clarity improves.

Do not add Scripthold-specific MCP, filesystem authorization, preview/apply, release, or workspace-crawling behavior to Marksplice core. Marksplice does not maintain first-party dialect extensions; broadly useful capabilities belong in core, while dialect/product syntax may use the explicit M110 read-only third-party SPI.

## Language and style

Use English for code, identifiers, comments, documentation, commit messages, and repository metadata. Keep packages small, errors explicit, tests deterministic, and dependencies minimal.
