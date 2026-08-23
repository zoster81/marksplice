# Changelog

All notable public changes to Marksplice will be documented in this file.

The project uses Semantic Versioning-compatible Go module tags. Until v1, the public API is intentionally unstable and may change between beta releases.

## Unreleased

- Continue API review and capability expansion toward a stable v1 contract.

## v0.1.0-beta.1 — 2026-08-23

- Initial public beta of `github.com/zoster81/marksplice`.
- Source-preserving structural parsing and reviewed mutation APIs for existing GFM.
- Deterministic `DocumentBuilder` construction for reviewed block, table, front-matter, blockquote, and typed-inline families.
- Typed full reference-link and reference-image construction with exact existing-definition proof.
- GFM 0.29 conformance policy with Goldmark isolated behind an internal adapter.
- Apache-2.0 licensing and Go 1.26 minimum language/toolchain requirement.
