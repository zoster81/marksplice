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

Milestones M1 through M17 have passed. M1 established feasibility, M2 established the first durable public surface, M3 promoted top-level headings, M4 promoted M1-proven list items and tasks, M5 introduced a parse-time editable-capability gate for mapped table cells and fenced code, M6 applied that gate to simple inline spans, M7 applied it to M1-proven link destinations and autolinks, M8 applied it to M1-proven simple front-matter fields and HTML comment/anchor edits, M9 added the first read-only hierarchical section view, M10 added copied bounded reads from the immutable source snapshot, M11 promoted simple inline-image destination editing through the same mapped-capability boundary, M12 added fail-closed removal of complete M9 section subtrees, M13 added direct-section-body replacement while preserving the existing document section hierarchy, M14 added complete section-subtree replacement from a standalone validated same-level fragment, M15 added same-level sibling insertion immediately before/after a section boundary, M16 added atomic same-level subtree movement using coordinated delete/insert patches and one combined candidate validation, and M17 added direct-child subtree append with explicit parent+1 level semantics; internal M1 types and taxonomies are not automatically public API commitments.

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

M1 deliberately reparses a prepared candidate snapshot when a mutation must prove that its semantic/source boundary still exists after replacement. M12–M16 apply the same conservative oracle to section mutations: M12/M13 preserve or remove known headings, M14/M15 parse standalone section fragments and compare inserted ranges in source order, and M16 validates a coordinated two-patch move only after rendering the complete candidate. The internal source `ChangeSet` may hold multiple disjoint patches, but this is not a public batch API: one named structural operation owns all its patches and one semantic validation. These remain single-operation O(n+k) safety oracles. Future batch planning should reuse parsed/indexed state or otherwise amortize joint validation without weakening the fail-closed checks established by M1/M12–M16.

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

M9 makes that distinction concrete for supported document-level headings. A derived public `Section` is anchored to the existing heading `NodeID` rather than inventing a second node identity. `BodyRange` begins after the complete heading line (after a Setext underline when applicable) and ends at the next supported heading of any level; the complete section range starts at the governing heading source and ends before the next heading of equal or higher level. Parent relationships are computed from heading levels. Container headings and preamble pseudo-sections remain deferred. The immutable section index is built once in O(h) time with a monotonic stack and uses an O(1)-expected heading-ID lookup map.

M10 makes public ranges directly readable through `Document.SourceRange`. The method validates the requested half-open byte range against the receiving immutable snapshot and returns a fresh copy rather than an alias of internal source storage. A read costs O(k) time and returned memory for a `k`-byte span, requires no parser pass or node/section scan, and composes with section body/subtree ranges and operation-oriented typed-detail ranges. It deliberately adds no filesystem, workspace, or network authority.

M11 applies the mapped-capability boundary to simple inline images. Goldmark provides semantic image recognition and public anchor/child boundaries; Marksplice does not inspect Goldmark's private image destination/title state. Its lossless source layer proves the `![alt](destination "title")` shape, reusing the established Markdown destination/title scanners, and only successful mappings become publicly editable. Reference images, compound alt text, empty destinations, and other unproven shapes remain internal/non-editable. Candidate replacements are reparsed and remapped before a change is returned.

M12 adds the first structural mutation over the M9 section model. `PrepareRemoveSection` targets the existing governing heading identity and deletes exactly the stored complete section subtree range; it does not rediscover boundaries or synthesize whitespace. The candidate document is reparsed once, then surviving supported headings are compared in source order for level, ATX/Setext style, exact shifted full/content ranges, and byte-identical heading source. This catches semantic join hazards such as a following Setext heading absorbing preceding paragraph text even when all untouched bytes are identical.

M13 adds `PrepareReplaceSectionBody`, which patches exactly `Section.BodyRange()` and requires the replacement to remain the complete direct body after candidate parsing. Existing document-level sections/headings must all survive; nested child sections therefore remain outside the patch. Heading-looking content inside blockquotes or fenced code remains valid because validation is semantic rather than character-based. M13 also consolidates M12 and M13 onto one `validateSectionHeadingPatch` safety path and one delta-aware `rangeAfterPatch` helper. After the single candidate parse, heading validation is O(h), with source-ordered comparison rather than repeated searches. Neither mutation introduces a second section identity or index.

M14 adds `PrepareReplaceSection`, which replaces exactly `Section.Range()` with a non-empty standalone fragment that must itself be one complete section subtree rooted at the same heading level as the target. The replacement may change root heading text/style and all descendant structure. Marksplice parses the fragment independently to prove ownership, then parses the host candidate and validates three source-ordered windows: untouched headings before the patch, all inserted fragment sections shifted by the patch start, and untouched headings after the removed subtree. Inserted section/body/heading ranges must match the standalone fragment at the expected offset, while external heading mappings/source remain exact modulo the patch delta. The original subtree end is discovered with one forward scan, so post-parse validation remains linear in original plus inserted section counts.

M15 adds `PrepareInsertSectionBefore` and `PrepareInsertSectionAfter`. Both reuse the same `parseSectionFragment` and inserted-fragment validator as M14, but prepare a zero-width patch. `before` inserts at the anchor section start; `after` inserts at the complete anchor subtree end, so descendants are never split. The fragment root level must equal the anchor level, making the operation a precise same-level sibling insertion. Existing headings are validated before/after the insertion window with half-open zero-width range shifting, and the candidate inserted root must occupy exactly the caller-provided fragment bytes. M15 never manufactures blank lines/newlines: Setext or EOF joins that change semantics fail closed.

M16 adds `PrepareMoveSectionBefore` and `PrepareMoveSectionAfter`. A move is one atomic named mutation containing two disjoint source patches: removal of the exact source `Section.Range()` and insertion of those same bytes at the original anchor boundary. Source and anchor roots must have equal levels; moving across parents is allowed and explicitly validated as reparenting to the anchor's parent. The expected section order is computed by snapshot IDs before candidate parsing, then the candidate must reproduce every expected heading's level/style/lexical boundary and the moved subtree's standalone fragment ranges. Already-satisfied adjacent moves return a zero-patch `ChangeSet` that remains fingerprint-bound. No whitespace repair, heading-level rewriting, or public generic batch semantics are introduced.

M17 adds `PrepareAppendSectionChild`. The fragment root must be exactly one heading level deeper than the parent, and insertion occurs at the complete parent subtree end rather than its direct-body end. This makes the new root a direct child even when existing descendants are present; candidate validation explicitly checks the inserted root's parent heading identity. Level-6 parents and deeper-than-direct fragments are rejected rather than synthesizing intermediate headings. M17 also splits the section implementation by established responsibility: named operations remain in `section_edits.go`, while shared fragment/candidate/range/order proof helpers live in `section_validation.go`; no algorithm or public contract changes as part of that consolidation.

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

M7 applies the same parse-time capability rule to simple inline links, single-line reference definitions, and supported GFM autolinks. Successful observations persist `InlineLinkMapping`, `ReferenceDefinitionMapping`, or `AutoLinkMapping`; the node's editable content range is the exact destination/token span used by the named operation. Public `InlineLink`, `ReferenceDefinition`, and `AutoLink` details expose only ID and that operation-oriented range. Labels, titles, destination wrappers, parentheses, angle brackets, indentation, trailing spaces, and line endings remain private preservation data. Expected unsupported source shapes remain internal with `Editable=false`; mutation preparation reuses the stored original mapping while candidate reparsing/remapping remains fail closed.

M8 promotes the conservative M1 front-matter and HTML edit shapes without turning Marksplice into a YAML/TOML or HTML parser. Unique simple scalar fields from a recognized leading YAML/TOML envelope become one public `FrontMatterField` family with exact value range, key, and Marksplice-owned format; the internal YAML/TOML kind split remains private. The immutable document retains only envelope format/opening/closing facts needed for candidate validation, while field ranges/key/style already live on the node. Goldmark-recognized raw HTML is public only when the existing source mapper proves a single-line comment payload or one simple quoted `id`/`name` attribute on an `<a>` opening tag; all other raw/block HTML remains opaque and non-editable. Public `HTMLComment` and `HTMLAnchor` details expose only operation-oriented ranges plus the anchor's semantic `id`/`name` attribute. Original-source rescans are removed from these three mutation paths; candidate reparsing/remapping remains fail closed.
