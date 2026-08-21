# Goldmark GFM Capability and Ownership Matrix

Status: source of truth for the parser/source-preservation ownership boundary established by M1 under Marksplice's single GitHub Flavored Markdown (GFM) profile.

Exact dependency versions belong in `go.mod` and `go.sum`; this document records responsibilities, not version pins.

| Area | Goldmark responsibility | Marksplice responsibility |
| --- | --- | --- |
| GFM core block/inline parsing | recognize the base GFM semantic structure | satisfy the gate defined in `docs/gfm-conformance.md`, then retain exact source spans/trivia needed for edits |
| GFM tables, task lists, strikethrough, extended autolinks | provide semantic nodes/extensions through `extension.GFM` | model and edit them without normalizing untouched source |
| Headings | identify heading semantics and levels | preserve ATX/Setext spelling, markers, spacing, and exact replacement boundaries |
| Paragraphs | identify paragraph nodes and line segments | map exact byte range and patch only selected content |
| Lists | identify lists/items and nesting semantics | preserve markers, numbering, indentation, task-marker spelling, and untouched item source |
| Fenced code | identify code blocks and info semantics | preserve fence character, length, indentation, spacing, and untouched body bytes |
| Tables | provide semantic table/header/body-row/cell hierarchy through GFM extension support | preserve table layout/alignment bytes; map editable non-empty cells plus complete body-row physical lines, retain parser-independent private table membership, resolve promoted body-cell↔row identities, same-table promoted row neighbors, and body-row→promoted-header identities, and validate complete row replacement/removal/insertion/movement while leaving delimiter rows and headers outside row-mutation targets |
| Links/images | identify link/image semantics and relationships; image destination/title details are not part of the pinned `ast.Image` public API | preserve inline/reference syntax choices and exact destination/source boundaries; derive simple image destination/title boundaries in Marksplice's source layer without private Goldmark access |
| Reference definitions | parse reference semantics | retain exact definition spelling/layout and safe mutation boundaries |
| YAML/TOML front matter | no responsibility in the default GFM profile | recognize only proven leading metadata envelopes in a separate source layer, preserve unknown envelope content opaquely, and patch exact safe scalar values without enabling another Markdown dialect |
| HTML/unsupported syntax | expose GFM `RawHTML` and `HTMLBlock` semantics/source segments | map proven comment/anchor boundaries for minimal edits and conservatively preserve all other raw/HTML block regions as opaque source |
| Source positions | provide AST segments where available | validate and supplement them; never assume AST positions encode all lexical trivia |
| Rendering | available for generated output | not used as the ordinary edit path for existing documents |
| Node identity | none suitable as Marksplice contract | deterministic snapshot-scoped Marksplice-owned identities |
| Stale-source safety | none | source fingerprints and conflict-on-mismatch |
| Patch application | none | validated minimal byte patches with unchanged-byte guarantees |

## Evidence rules established by M1

A construct is not considered losslessly supported merely because Goldmark parses it. Evidence for a source-preserving capability must show the exact source region needed for the operation, byte preservation outside changed spans, deterministic behavior for unsafe or ambiguous mutation cases, and no unresolved semantic mismatch against the approved GFM contract.

The parser decision gate currently retains Goldmark. The normative source hierarchy, approved snapshot, exact conformance evidence, advisory-source policy, and upgrade procedure are owned by [`gfm-conformance.md`](gfm-conformance.md); do not duplicate or reinterpret them here.

The default semantic parser profile must remain exactly GFM-oriented: `extension.GFM` plus only narrowly scoped compatibility behavior required to implement the published GFM syntax. Non-GFM extensions such as definition lists, footnotes, typographer transformations, custom heading attributes, or project-specific syntax are not enabled by default.

Where Goldmark source positions are insufficient, Marksplice may add a bounded lexical scan tied to the semantic node and source snapshot. Such scans must stay internal and must not copy or fork Goldmark implementation code.
