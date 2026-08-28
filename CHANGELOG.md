# Changelog

All notable public changes to Marksplice will be documented in this file.

The project uses Semantic Versioning-compatible Go module tags. Until v1, the public API is intentionally unstable and may change between beta releases.

## Unreleased

- Add the read-only `workspacefs` package for caller-authorized Markdown workspaces over `fs.FS`, with deterministic `.md`/`.markdown` scanning, cycle-safe relationship following, slash-relative document keys, finite document/byte/depth/relationship budgets, and direct reuse of the existing document graph/workspace validator.
- Keep filesystem authority outside the root document core: `workspacefs` performs no writes, network access, or command execution and deliberately ignores ambiguous/non-local destination forms pending the dedicated filesystem-resolution hardening milestone.

## v0.5.0-beta.1 — 2026-08-28

- Complete the pre-M116 v0.5 performance campaign: on the byte-certified 6,857-document / 60.8 MB corpus, same-host five-run medians improve public `Parse` from 15.04 to 25.06 MB/s and Native from 19.23 to 30.87 MB/s while reducing public/Native allocated bytes to about 2.702/1.902 GB per complete corpus pass and allocation counts to about 10.402/9.015 million.
- Compact common parser/document node storage through sparse detail sidecars and narrower common records, reduce transient projection/index state and repeated source mapping, and add measured Native scanner/index fast paths without reintroducing Goldmark or adding persistent parse caches.
- Make byte-identical replacements of already source-proven content deterministic snapshot-bound no-ops across reviewed mutation APIs, while keeping validators for genuinely new invalid content fail-closed; fuzzing found and regression-tested Setext-heading, math-NUL, and contextual-paragraph edge cases.
- Fix source-preserving promotion of GFM table rows nested in indented block containers by tracking the semantic row anchor separately from complete physical-line ownership.
- Surface Marksplice's verified engineering/agent-tooling characteristics more clearly in the public entry point, including conformance, stale-source safety, real-world corpus validation, measured parse performance, and explicit authority boundaries.

## v0.1.0-beta.2 — 2026-08-27

- Add source-proven footnote and mathematical-expression capabilities, generalized opaque YAML/TOML front-matter envelope reading, and syntax-independent knowledge-document indexing over explicit document graphs.
- Complete fuzz/pathological/performance hardening and stabilize the public API contract before third-party extensibility work.
- Replace the ambiguous four-value `WorkspaceDiagnostic.UnresolvedReference` result with typed immutable `UnresolvedReference` data; document concurrent-read safety for immutable public models, serialized build-time resolvers, caller-bounded operations, `errors.Is` error classification, and the closed core `Kind` namespace. This accessor change is intentionally source-incompatible during the v0 beta period.
- Add the explicit `ParseWithOptions` third-party read-only extension SPI with namespaced extension kinds, validated snapshot ranges/scalar metadata, caller-owned retention limits, serial non-retained recognizers, panic/error isolation through `ErrInvalidExtension`, and no core mutation/construction/parser/graph authority.
- Freeze the complete internal parser-backend substitution contract and add a reusable differential harness over the same pinned GFM corpus plus focused Marksplice semantic/source-position regressions, keeping Goldmark only as the temporary oracle for the native-parser transition.
- Complete the Native parser transition: implement and harden the full CommonMark/GFM backend, switch production parsing and construction proof to Native, freeze parser-neutral conformance fixtures tied to the approved external specification inputs, and remove the Goldmark adapter, differential scaffolding, and `github.com/yuin/goldmark` module dependency.
- Add a task-oriented module guide and an exhaustive exported-callable API reference, with documentation audits that keep the reference synchronized with the public Go declarations.
- Refactor the public documentation into one clear README entry point, a first-use Getting Started path, goal-oriented recipes, a concise current capability matrix, and runnable file-based examples for inspection, editing, construction, querying, workspaces, and read-only extensions; keep milestone/parser history in advanced historical records instead of the normal user journey.
- Document the approved path to stable v1.0: filesystem-backed workspace discovery, Native semantic rendering, HTML/source mapping, canonical Markdown, and a dedicated M124 stabilization/profile gate; defer PDF work to the planned v1.5 line.

## v0.1.0-beta.1 — 2026-08-23

- Initial public beta of `github.com/zoster81/marksplice`.
- Source-preserving structural parsing and reviewed mutation APIs for existing GFM.
- Deterministic `DocumentBuilder` construction for reviewed block, table, front-matter, blockquote, and typed-inline families.
- Typed full reference-link and reference-image construction with exact existing-definition proof.
- GFM 0.29 conformance policy with Goldmark isolated behind an internal adapter.
- Apache-2.0 licensing and Go 1.26 minimum language/toolchain requirement.
