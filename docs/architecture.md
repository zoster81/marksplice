# Marksplice Architecture

Status: source of truth for durable architecture decisions.

## Mission

Marksplice is a Pure-Go library for source-preserving structural GitHub Flavored Markdown (GFM) manipulation. GFM is the project's single normative Markdown syntax profile. Parsing exists to understand source; it does not authorize normalization of untouched source.

For ordinary edits to an existing document, bytes outside the intended changed spans must remain byte-identical whenever the operation does not semantically require a broader change.

## Architecture

```text
original Markdown bytes
        |
        +--> semantic parser adapter (Goldmark initially)
        |         |
        |         v
        |    semantic observations
        |
        +--> Marksplice lossless source mapping
                  |
                  v
          Marksplice document model
                  |
                  v
        validated structural change
                  |
                  v
            minimal byte patches
```

The semantic parser and lossless mapping have different responsibilities. Goldmark may identify semantic constructs, but Marksplice owns lexical details and exact mutation boundaries needed to preserve source.

## Package boundaries

Current internal boundaries:

- `internal/parser/goldmark`: Goldmark-specific parsing and AST traversal. No Goldmark type may cross this adapter boundary.
- `internal/source`: snapshot fingerprints, byte ranges, validated patches, stale-source conflict detection, and patch application.
- `internal/splice`: implementation document model that combines semantic observations with source snapshots and prepares structural edits.
- root package `marksplice`: reviewed public API values and operations only; it may wrap internal implementation objects but must not expose Goldmark or `internal/*` types. Keep public implementation responsibilities separated: core snapshot/enumeration/error plumbing, typed details, and named mutations belong in cohesive files rather than one growing API monolith.

Milestones M1 through M6 have passed. M1 established feasibility, M2 established the first durable public surface, M3 promoted top-level headings, M4 promoted M1-proven list items and tasks, M5 introduced a parse-time editable-capability gate for mapped table cells and fenced code, and M6 applied that gate to simple inline spans; internal M1 types and taxonomies are not automatically public API commitments.

Within the internal implementation, keep orchestration separate from syntax-specific proof logic. Shared mutation plumbing may centralize target lookup, simple replacement preconditions, candidate patch construction, and candidate parsing, while source mappers and semantic validators remain typed and feature-specific when their safety invariants differ. Likewise, shared lexical primitives belong in focused helpers rather than being duplicated across block, inline, link, front-matter, or HTML mapping code.

## Source model

Source is represented as bytes. Source ranges use half-open byte offsets `[start,end)` into a specific immutable source snapshot.

A structural node needs enough snapshot-local information to support deterministic targeting, conceptually including:

```text
id
kind
sourceRange
parentId
children
sourceFingerprint
properties
```

Human-readable labels such as heading text are not sufficient identities because duplicates are valid Markdown.

Node identities are snapshot-scoped, deterministic, and derived from source-bound structural facts. They are not durable identities across arbitrary document revisions. Public `NodeID` values are opaque and comparable; their diagnostic string form is not a persistence or round-trip format.

An immutable parsed document may maintain a derived snapshot-local `NodeID` index so structural targeting does not require a linear scan for every operation. Building that index must reject duplicate IDs rather than silently choosing one node; the ordered node collection remains the source of structural iteration order.

## Mutation model

A prepared change contains the source fingerprint it was created against and one or more non-overlapping minimal byte patches.

Application rules:

1. fingerprint the supplied source;
2. reject the operation with an explicit conflict if the snapshot differs;
3. validate every patch range and ordering;
4. apply patches without rendering unrelated source;
5. preserve all bytes not covered by changed ranges.

Batch editing must reject overlapping or ambiguous patches rather than relying on application order to pick a winner. Efficient batch application should sort validated patches once and apply them without repeated whole-document rescans.

M1 deliberately reparses a prepared candidate snapshot when a mutation must prove that its semantic/source boundary still exists after replacement. This is a conservative O(n) safety oracle for one prepared mutation, not the desired implementation of a future multi-edit batch. Post-M1 batch planning should reuse parsed/indexed state or otherwise amortize validation without weakening the fail-closed semantic checks established by M1.

## Line endings, Unicode, and encoding

Marksplice core operates on Markdown bytes and must not normalize LF, CRLF, CR, or Unicode content as a side effect of unrelated edits. Byte ranges, not rune indexes, define mutation boundaries. The semantic parser may use a same-length shadow view for parser compatibility (for example, isolated CR mapped to LF) only when byte offsets remain exactly aligned with the original source.

Encoding/BOM preservation belongs to a host that provides decoded Markdown bytes plus an encoding policy, unless Marksplice later introduces an explicit encoding-aware layer. The core must not silently guess and rewrite file encodings.

## GFM conformance and Goldmark boundary

GitHub Flavored Markdown (GFM) is the only Markdown dialect Marksplice targets. CommonMark is inherited as GFM's base syntax, not exposed as a separate Marksplice mode. The normative source hierarchy, approved specification snapshot, conformance procedure, and update policy are defined in [`gfm-conformance.md`](gfm-conformance.md).

`github.com/yuin/goldmark` is the selected semantic parser implementation. It is configured for GFM with narrowly scoped Marksplice compatibility behavior where required by the approved GFM contract. Goldmark remains an implementation dependency, not part of Marksplice's public contract.

Rules:

- do not fork or copy Goldmark merely to implement Marksplice;
- do not expose its AST or parser-specific types publicly;
- configure the semantic parser for GFM rather than assembling an ad hoc mixture of Markdown extensions;
- follow the conformance hierarchy and specification-update gate in `docs/gfm-conformance.md`;
- treat parser-library divergences as adapter/model gaps to resolve, not as reasons to expose another Markdown dialect or silently waive conformance cases;
- supplement it with Marksplice-owned lexical/source mapping wherever semantic AST information is insufficient for source-preserving edits;
- do not serialize the Goldmark AST as the ordinary existing-document edit path;
- do not add non-GFM syntax extensions to the default parser profile merely because Goldmark supports them.

GFM's disallowed-raw-HTML tag filtering is an HTML-rendering requirement. Marksplice core currently parses, models, validates, and edits Markdown source rather than rendering HTML; if HTML rendering becomes a Marksplice responsibility, GFM rendering conformance including tag filtering becomes part of that feature's acceptance criteria.

YAML and TOML front matter are document-envelope metadata, not additional Markdown dialects. The GFM body continues to be parsed through the normal Goldmark profile without enabling a front-matter extension. Marksplice may recognize a leading front-matter envelope in its own source layer and exclude that proven envelope from the GFM structural view while preserving all envelope bytes not explicitly edited. M1 deliberately recognizes only closed byte-zero `---`/`+++` envelopes that contain at least one unique scalar field whose lexical value boundary can be proven safely; ambiguous, duplicate-only, complex-only, non-leading, or unclosed shapes remain ordinary GFM source rather than being guessed as metadata.

The detailed capability/ownership matrix is maintained in `docs/goldmark-capability-matrix.md`.

## Structural semantics

The target model includes documents, sections, headings, paragraphs, lists/list items/tasks, blockquotes, fenced code, inline code, tables, thematic breaks, front matter, links/references/images, explicit anchors, and opaque preserved HTML/unsupported regions.

A section is governed by a heading until the next heading of equal or higher level. The model must distinguish the heading, direct section body, and complete subtree.

Milestone M1 intentionally implemented only the subset required to prove the architecture; later milestones may expand the structural model without weakening the established source-preservation invariants.

## Safety boundaries

Prepared mutations fail closed on stale source. Malformed ranges, overlapping patches, invalid structural targets, or ambiguous targeting must produce deterministic errors rather than best-effort mutation.

Marksplice core does not perform arbitrary filesystem traversal, network requests, or command execution. Future multi-document relationship resolution must be bounded and constrained to caller-authorized document sets or roots.

External URLs remain data unless an explicit caller outside core chooses to act on them.

## Complexity goals

Parsing and structural indexing should be linear or near-linear in source size where practical. Mutation planning should avoid quadratic rescanning. Callers should eventually be able to impose byte, node, depth, relationship, and output budgets rather than relying on hidden global limits.

## Public API foundation and promotion

M1 demonstrated the feasibility requirements that gated broader API design. M2 established the following public-boundary rules:

- exported signatures contain only Marksplice-owned or standard-library types;
- generic public `Node` values expose only reviewed common semantics, initially snapshot-scoped identity and promoted kind;
- internal node kinds are not published automatically; each public kind is promoted only after its caller-facing semantics are reviewed;
- operation-oriented internal source ranges are not exposed as one generic node-span contract. Position/range semantics belong to typed details or focused accessors that define exactly what the bytes represent;
- the first typed detail is a top-level `Paragraph`, whose range is the exact span used by paragraph replacement and excludes the following line ending;
- public paragraph mutation is limited to top-level paragraphs because that is the M1 shape whose replacement semantics were proven; container paragraphs remain internal until separately proven;
- public prepared changes are opaque and preserve M1 snapshot-conflict behavior;
- public sentinel errors are owned by the root package and internal failure categories are translated while preserving `errors.Is` semantics;
- public document lookup reuses the internal snapshot index rather than maintaining a second index.

These rules intentionally keep the public surface narrower than the M1 implementation. Broader structural operations should be promoted one semantic family at a time without weakening the minimal-patch/fail-closed model.

M3 applies that pattern to top-level headings. Public `Heading` detail exposes snapshot identity, level, Marksplice-owned ATX/Setext style, and the exact content range replaced by heading rename. It does not expose the internal complete heading source range, Goldmark data, or M1 heading structs. `PrepareRenameHeading` delegates to the proven M1 minimal-patch candidate-reparse validation, preserving markers, Setext underlines, spacing, and line endings while rejecting replacements that do not re-establish the same supported heading shape. Public enumeration also requires top-level status so future internal container headings are not promoted automatically.

M4 applies the same pattern to single-line list items and GFM task markers because those families already have exact editable mappings established during document parsing. Public `ListItem` detail exposes only the content range used by replacement plus ordered state and the existing marker/delimiter byte; numeric ordered-list prefixes remain source trivia. Public `Task` detail exposes only the one-byte state range and checked state. Nested list items and tasks are promoted because M1 proved those source-preserving shapes; unlike paragraph and heading promotion, they are not filtered by top-level status.

A semantic observation is not sufficient by itself for public promotion as an actionable node. When an M1 family proves important source-shape facts only lazily during mutation preparation, Marksplice should first make that capability boundary explicit—by persisting/validating an editable mapping in the parsed model or by another reviewed mechanism—before exposing the family as an ordinary public typed detail plus named mutation. This prevents callers from receiving apparently actionable nodes whose source-preserving support is known only after an operation is attempted, and keeps internal source-mapper failure categories out of the public contract.

M5 implements that capability boundary for non-empty GFM table cells and supported single-line fenced code. The immutable internal node stores an `Editable` capability and the validated original source mapping needed by the mutation validator. Expected unsupported source shapes remain semantic internal nodes with `Editable=false`; they do not make document parsing fail and are filtered from the public actionable surface. The capability flag itself is not public API.

For M5 table cells, public detail exposes only ID, the exact content replacement range, header/body state, and zero-based column. Raw cell padding and delimiters remain internal source data. Parse-time table handling is linear in row width: the Goldmark adapter assigns cell columns incrementally during its single AST walk, then passes a parser-independent row anchor to the source layer. A snapshot-local row cache lets `source.MapTableRow` scan each physical table row once and derive all cell mappings for that row, avoiding both sibling back-scans and a full-row source rescan for every cell in wide tables. For fenced code, public detail exposes only ID and the exact content replacement range; fence character/length, indentation, closing-fence facts, and info-string boundaries remain internal validation data. Mutation preparation reuses the stored original mapping rather than rescanning the immutable original source, while candidate reparsing and mapping remain the conservative M1 fail-closed safety oracle.

M6 applies the same parse-time capability rule to simple GFM strikethrough, code spans, emphasis, and strong emphasis. Supported nodes persist their validated `StrikethroughMapping`, `CodeSpanMapping`, or `EmphasisMapping` and are public only when `Editable=true`. Public typed details intentionally expose only snapshot ID and the exact content replacement range; tilde, backtick, `*`/`_`, and delimiter-run facts remain private validation data. Semantic shapes requiring code-span whitespace normalization or compound/nested emphasis delimiters remain internal non-editable observations rather than public mutation targets. Original-source mappings are reused during preparation; candidate reparsing/remapping remains fail closed.
