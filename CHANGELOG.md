# Changelog

All notable public changes to Marksplice are documented in this file.

Marksplice follows Semantic Versioning-compatible Go module tags. `v1.0.0` established the first stable public API contract; compatible fixes and additions remain within the v1 line, while intentionally breaking public API changes require a new major version.

## v1.1.1 — 2026-09-17

- Make `v1.1.1` the supported release for the v1.1 feature set and retract `v1.1.0`, which was observed by Go module tooling before its public release metadata and maintenance naming were fully cleaned up.
- Replace internal development-phase terminology in active tests, benchmarks, helper names, and current documentation with behavior-oriented names. Public Go API behavior is unchanged from `v1.1.0`.
- Simplify current architecture, conformance, contribution, and release documentation so it describes present Marksplice behavior directly; historical development chronology remains in explicitly historical records.

## v1.1.0 — 2026-09-17 — retracted; use v1.1.1

- Fix variable-length YAML/TOML front-matter scalar replacement for positive and negative byte deltas, Unicode values, LF/CRLF source, and stale-source-safe edits.
- Add source-preserving top-level paragraph insertion/removal, heading-level mutation with section revalidation, first-child list insertion, direct-link/image title lifecycle operations, reference occurrence/definition lifecycle operations, multiline footnote definition editing/lifecycle, and conservative front-matter field/envelope lifecycle operations.
- Expose parser-derived semantic heading text through `Heading.Text()`.
- Add source-proven blockquote and alert content mutation, fenced info-string editing, and population of source-proven empty closed fenced blocks without whole-document regeneration.
- Keep the new mutation proof paths within the production complexity gate and retain candidate reparsing as the safety boundary rather than adding a persistent mutation cache.

## v1.0.0 — 2026-09-16

- Establish the first stable Marksplice public API and compatibility contract after full parser/source-preservation, workspace, rendering, performance, fuzz/race, dependency, documentation, and release-readiness recertification.
- Require Go 1.26 or newer and refresh the direct `golang.org/x/text` dependency used for Unicode reference-label folding.
- Add the read-only `workspacefs` package for caller-authorized Markdown workspaces over `fs.FS`, with deterministic scanning/following, finite resource limits, and shared document-graph/workspace validation semantics.
- Harden filesystem relationship resolution with explicit URI-path rules, single percent-decoding, query/file separation, preserved fragments, and fail-closed handling for traversal, encoded separators, unsupported absolute/scheme forms, directories, and extensionless targets.
- Add deterministic HTML fragment and standalone-document rendering with explicit raw-HTML, dangerous-URL, tag-filter, and reviewed front-matter metadata policies.
- Add optional HTML source-to-output byte mapping for editor/preview tooling without retaining a persistent source-map index.
- Add deterministic canonical Markdown export, separate from source-preserving editing, with semantic reparse equivalence and byte idempotence across the approved CommonMark/GFM examples and retained real-world corpus.
- Add full-profile HTML conformance against all approved CommonMark and published-GFM examples, including rendering-only tag filtering and explicitly reviewed profile divergences.

## v0.5.0-beta.1 — 2026-08-28

- Improve public parsing throughput and reduce allocation on the retained 6,857-document / 60.8 MB engineering corpus through profile-guided parser/document-model optimization.
- Compact common parser/document storage, reduce transient projection/index state and repeated source mapping, and add measured scanner/index fast paths without adding persistent parse caches.
- Make byte-identical replacements of already source-proven content deterministic snapshot-bound no-ops while keeping genuinely invalid replacement content fail-closed.
- Fix source-preserving promotion of GFM table rows nested in indented block containers.
- Expand public engineering documentation for conformance, stale-source safety, corpus validation, measured performance, and authority boundaries.

## v0.1.0-beta.2 — 2026-08-27

- Add source-proven footnote and mathematical-expression capabilities, generalized opaque YAML/TOML front-matter envelope reading, and syntax-independent knowledge-document indexing over explicit document graphs.
- Expand fuzz/pathological/performance hardening and stabilize the public API shape ahead of v1.
- Replace an ambiguous unresolved-reference accessor tuple with typed immutable `UnresolvedReference` data and document concurrent-read safety, bounded operations, `errors.Is` classification, and the closed core `Kind` namespace.
- Add the explicit `ParseWithOptions` read-only third-party observation SPI with namespaced extension kinds, validated snapshot ranges/metadata, caller-provided retention limits, serial non-retained recognizers, and no mutation/construction/parser/graph authority.
- Complete the Marksplice-owned Native parser transition, freeze parser-neutral conformance fixtures tied to approved specification inputs, and remove the temporary third-party Markdown parser dependency.
- Add a task-oriented module guide, exhaustive exported-callable API reference, clear documentation entry point, goal-oriented recipes, concise capability matrix, and runnable file-based examples.

## v0.1.0-beta.1 — 2026-08-23

- Initial public beta of `github.com/zoster81/marksplice`.
- Source-preserving structural parsing and reviewed mutation APIs for existing GFM.
- Deterministic `DocumentBuilder` construction for reviewed block, table, front-matter, blockquote, and typed-inline families.
- Typed full reference-link and reference-image construction with exact existing-definition proof.
- GFM conformance policy, parser isolation, Apache-2.0 licensing, and Go 1.26 minimum toolchain requirement.
