# Changelog

All notable public changes to Marksplice will be documented in this file.

The project uses Semantic Versioning-compatible Go module tags. Until v1, the public API is intentionally unstable and may change between beta releases.

## Unreleased

- Add source-proven footnote and mathematical-expression capabilities, generalized opaque YAML/TOML front-matter envelope reading, and syntax-independent knowledge-document indexing over explicit document graphs.
- Complete fuzz/pathological/performance hardening and stabilize the public API contract before third-party extensibility work.
- Replace the ambiguous four-value `WorkspaceDiagnostic.UnresolvedReference` result with typed immutable `UnresolvedReference` data; document concurrent-read safety for immutable public models, serialized build-time resolvers, caller-bounded operations, `errors.Is` error classification, and the closed core `Kind` namespace. This accessor change is intentionally source-incompatible during the v0 beta period.
- Add the explicit `ParseWithOptions` third-party read-only extension SPI with namespaced extension kinds, validated snapshot ranges/scalar metadata, caller-owned retention limits, serial non-retained recognizers, panic/error isolation through `ErrInvalidExtension`, and no core mutation/construction/parser/graph authority.
- Freeze the complete internal parser-backend substitution contract and add a reusable differential harness over the same pinned GFM corpus plus focused Marksplice semantic/source-position regressions, keeping Goldmark only as the temporary oracle for the native-parser transition.
- Complete the Native parser transition: implement and harden the full CommonMark/GFM backend, switch production parsing and construction proof to Native, freeze parser-neutral conformance fixtures tied to the approved external specification inputs, and remove the Goldmark adapter, differential scaffolding, and `github.com/yuin/goldmark` module dependency.
- Add a task-oriented module guide and an exhaustive exported-callable API reference, with documentation audits that keep the reference synchronized with the public Go declarations.

## v0.1.0-beta.1 — 2026-08-23

- Initial public beta of `github.com/zoster81/marksplice`.
- Source-preserving structural parsing and reviewed mutation APIs for existing GFM.
- Deterministic `DocumentBuilder` construction for reviewed block, table, front-matter, blockquote, and typed-inline families.
- Typed full reference-link and reference-image construction with exact existing-definition proof.
- GFM 0.29 conformance policy with Goldmark isolated behind an internal adapter.
- Apache-2.0 licensing and Go 1.26 minimum language/toolchain requirement.
