# Goldmark GFM Capability and Ownership Matrix

Status: source of truth for the M1 parser/source-preservation boundary under Marksplice's single GitHub Flavored Markdown (GFM) profile.

Exact dependency versions belong in `go.mod` and `go.sum`; this document records responsibilities, not version pins.

| Area | Goldmark responsibility | Marksplice responsibility |
| --- | --- | --- |
| GFM core block/inline parsing | recognize the base GFM semantic structure | satisfy the gate defined in `docs/gfm-conformance.md`, then retain exact source spans/trivia needed for edits |
| GFM tables, task lists, strikethrough, extended autolinks | provide semantic nodes/extensions through `extension.GFM` | model and edit them without normalizing untouched source |
| Headings | identify heading semantics and levels | preserve ATX/Setext spelling, markers, spacing, and exact replacement boundaries |
| Paragraphs | identify paragraph nodes and line segments | map exact byte range and patch only selected content |
| Lists | identify lists/items and nesting semantics | preserve markers, numbering, indentation, task-marker spelling, and untouched item source |
| Fenced code | identify code blocks and info semantics | preserve fence character, length, indentation, spacing, and untouched body bytes |
| Tables | provide table semantics through extension support | preserve table layout/alignment bytes outside explicitly changed cells/rows |
| Links/images | identify destinations/titles/relationships | preserve inline/reference syntax choices and exact destination/source boundaries |
| Reference definitions | parse reference semantics | retain exact definition spelling/layout and safe mutation boundaries |
| HTML/unsupported syntax | expose available semantic/opaque nodes | conservatively preserve source regions not proven safe to edit |
| Source positions | provide AST segments where available | validate and supplement them; never assume AST positions encode all lexical trivia |
| Rendering | available for generated output | not used as the ordinary edit path for existing documents |
| Node identity | none suitable as Marksplice contract | deterministic snapshot-scoped Marksplice-owned identities |
| Stale-source safety | none | source fingerprints and conflict-on-mismatch |
| Patch application | none | validated minimal byte patches with unchanged-byte guarantees |

## M1 evidence rules

A construct is not considered losslessly supported merely because Goldmark parses it. M1 evidence must show the exact source region needed for the operation, byte preservation outside changed spans, deterministic behavior for unsafe or ambiguous mutation cases, and no unresolved semantic mismatch against the approved GFM contract.

The parser decision gate currently retains Goldmark. The normative source hierarchy, approved snapshot, exact conformance evidence, advisory-source policy, and upgrade procedure are owned by [`gfm-conformance.md`](gfm-conformance.md); do not duplicate or reinterpret them here.

The default semantic parser profile must remain exactly GFM-oriented: `extension.GFM` plus only narrowly scoped compatibility behavior required to implement the published GFM syntax. Non-GFM extensions such as definition lists, footnotes, typographer transformations, custom heading attributes, or project-specific syntax are not enabled by default.

Where Goldmark source positions are insufficient, Marksplice may add a bounded lexical scan tied to the semantic node and source snapshot. Such scans must stay internal and must not copy or fork Goldmark implementation code.
