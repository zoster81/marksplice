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

M0 records the green repository-bootstrap baseline retrospectively. Engineering milestones M1 through M30 have passed. M1 established feasibility, M2 established the first durable public surface, M3 promoted top-level headings, M4 promoted M1-proven leaf single-line list items and tasks, M5 introduced a parse-time editable-capability gate for mapped table cells and fenced code, M6 applied that gate to simple inline spans, M7 applied it to M1-proven link destinations and autolinks, M8 applied it to M1-proven simple front-matter fields and HTML comment/anchor edits, M9 added the first read-only hierarchical section view, M10 added copied bounded reads from the immutable source snapshot, M11 promoted simple inline-image destination editing through the same mapped-capability boundary, M12 added fail-closed removal of complete M9 section subtrees, M13 added direct-section-body replacement while preserving the existing document section hierarchy, M14 added complete section-subtree replacement from a standalone validated same-level fragment, M15 added same-level sibling insertion immediately before/after a section boundary, M16 added atomic same-level subtree movement using coordinated delete/insert patches and one combined candidate validation, M17 added direct-child subtree append with explicit parent+1 level semantics, M18 added complete physical-line removal for promoted leaf list items, M19 added same-shape leaf sibling insertion, and M20 added atomic same-shape leaf movement with shared original-coordinate multi-patch range transforms, M21 added direct leaf-child append validated by semantic parent anchors in the host candidate, M22 promoted supported single-line-head list parents while keeping structural line operations conservative, M23 resolves public immediate supported-parent identities once during parse without a persistent hierarchy index, M24 extends child append to existing fully supported parent subtrees using private completeness/end proof, M25 reuses that proof to extend list-item removal to complete supported subtrees with set-based survivor validation, M26 extends same-shape leaf sibling insertion around complete supported parent subtrees with semantic sibling validation, M27 extends atomic leaf movement around complete supported parent anchors, M28 extends the moved source to complete supported subtrees with non-overlap, descendant, and parent-count validation, M29 extends caller-provided sibling insertion fragments to complete supported subtrees with standalone ownership proof and shared subtree-placement validation, and M30 extends direct-child append to complete supported child subtrees validated only in host context; internal M1 types and taxonomies are not automatically public API commitments.

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

M1 deliberately reparses a prepared candidate snapshot when a mutation must prove that its semantic/source boundary still exists after replacement. M12–M16 apply the same conservative oracle to section mutations: M12/M13 preserve or remove known headings, M14/M15 parse caller-provided standalone section fragments and compare inserted ranges in source order, and M16 validates a coordinated two-patch move only after rendering the complete candidate. Snapshot-owned moved source is not reparsed independently: the existing M9 section ownership/index supplies the subtree model, and candidate validation compares moved bytes, subtree-relative section/body/heading ranges, and internal descendant parentage directly. The internal source `ChangeSet` may hold multiple disjoint patches, but this is not a public batch API: one named structural operation owns all its patches and one semantic validation. These remain single-operation O(n+k) safety oracles. Future batch planning should reuse parsed/indexed state or otherwise amortize joint validation without weakening the fail-closed checks established by M1/M12–M16.

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

M16 adds `PrepareMoveSectionBefore` and `PrepareMoveSectionAfter`. A move is one atomic named mutation containing two disjoint source patches: removal of the exact source `Section.Range()` and insertion of those same bytes at the original anchor boundary. Source and anchor roots must have equal levels; moving across parents is allowed and explicitly validated as reparenting to the anchor's parent. The expected section order is computed by snapshot IDs before candidate parsing, then the candidate must reproduce every expected heading's level/style/lexical boundary. A retrospective audit reuses M9's already-proven snapshot-owned section window for the moved subtree instead of reparsing those bytes standalone: moved sections, direct-body ranges, headings, exact bytes, and internal descendant parents must all reproduce their source-relative structure at the destination, while only the moved root may adopt the anchor's parent. Already-satisfied adjacent moves return a zero-patch `ChangeSet` that remains fingerprint-bound. No whitespace repair, heading-level rewriting, or public generic batch semantics are introduced.

M17 adds `PrepareAppendSectionChild`. The fragment root must be exactly one heading level deeper than the parent, and insertion occurs at the complete parent subtree end rather than its direct-body end. This makes the new root a direct child even when existing descendants are present; candidate validation explicitly checks the inserted root's parent heading identity. Level-6 parents and deeper-than-direct fragments are rejected rather than synthesizing intermediate headings. M17 also splits the section implementation by established responsibility: named operations remain in `section_edits.go`, while shared fragment/candidate/range/order proof helpers live in `section_validation.go`; no algorithm or public contract changes as part of that consolidation.

M18 extends the M4 private `ListItemMapping` with `LineRange`, the complete physical source line including its own line terminator when present. Existing marker/source `Range`, content-only `ContentRange`, public `ListItem.Range()`, and snapshot ID derivation remain unchanged. `PrepareRemoveListItem` deletes exactly this private physical-line range. Candidate validation indexes remapped leaf list items by physical-line start and requires every original promoted leaf survivor to preserve shifted line/marker/content ranges, ordered state, marker byte, and complete line source. Candidate leaf-count equality is deliberately not required because removing a final nested child may legitimately make its parent newly promotable as a leaf.

M19 adds `PrepareInsertListItemBefore` and `PrepareInsertListItemAfter`. A standalone fragment must be exactly one complete promoted leaf line and reproduce the anchor's exact pre-marker physical-line prefix plus ordered state and marker/delimiter byte; an ordered numeric token remains caller source and may differ. Insertion uses the anchor's private `LineRange` boundary, preserves all original leaf mappings through one candidate parse, and requires exactly one additional leaf mapping matching the standalone fragment. No line terminator is synthesized when host separation is unsafe.

M20 adds `PrepareMoveListItemBefore` and `PrepareMoveListItemAfter`. A move owns two coordinated patches—delete the exact source `LineRange` and insert those same bytes at the anchor boundary—and validates their joint candidate once. The moved line must pass the same M19 destination-shape proof; cross-parent movement is allowed only when the concrete sibling prefix/marker shape matches. Survivor validation deliberately permits newly promoted source-parent leaf items. Generic half-open patch range shifting now lives in `mutation.go`: `rangeAfterPatches` transforms structural ranges across disjoint original-coordinate patches, `rangeAfterPatch` delegates to it, and `movedRangeCandidateOffset` is shared by section and list moves. These helpers remain private mutation plumbing, not a public batch surface.

M21 adds `PrepareAppendListItemChild`. The caller supplies one complete child physical line; Marksplice does not manufacture indentation, numbering, markers, or line endings. The Goldmark adapter emits Marksplice-owned `HasListParent`/`ListParentAnchor` metadata, derived from public AST parent relationships and the immediate parent item's physical-line start. Candidate validation preserves that semantic parent relation for surviving supported list items. M21 inserts at a leaf parent's `LineRange.End` and accepts the operation only when exactly one candidate leaf occupies the inserted byte span and reports the target physical line as its immediate parent. This is intentionally host-context validation rather than standalone-fragment parsing: GFM child indentation depends on parent marker width `W`, 1–4 post-marker spaces `N`, and container context, and ordered sublists interrupting paragraph content must begin at `1`. Caller bytes are never normalized to satisfy those rules.

M22 broadens list-item promotion only to an item whose own first `TextBlock`/`Paragraph` maps to one physical content line and whose remaining direct AST blocks are nested lists only. Parser-independent `HasListChildren` becomes internal `ListHasChildren` and public `ListItem.HasChildren()` while `ListItem.Range()` remains the exact first-line content span. `PrepareReplaceListItem` supports these parents and validates all descendants/parent anchors after the content-only patch. M18–M21 line-bound structural operations use one `leafListItemTarget` gate and reject parent targets rather than treating the first physical line as a subtree boundary. Removing or moving a final leaf child may legitimately toggle the source parent's `HasChildren`; shared survivor validation permits that change only at the operation's explicitly known parent anchor, preserving child-state everywhere else.

M23 adds `ListItem.ParentID() (NodeID, bool)` for immediate semantic parents that are themselves supported/promoted list items. Parent identity is resolved only after ordinary list-item IDs already exist: `Parse` builds a temporary O(l) map from each supported `LineRange.Start` to its real snapshot `NodeID`, resolves existing M21 `ListParentAnchor` values, stores only the resulting `ListParentID` on child nodes, then discards the map. Root items and children of intentionally unsupported complex parents return no public parent ID. M23 does not synthesize identities from anchors, add a persistent second hierarchy index, change node-ID derivation, or alter mutation semantics; the public accessor is O(1) after the O(l) parse-time resolution.

M24 extends `PrepareAppendListItemChild` to a supported parent that already owns children, but only when Marksplice can prove the complete semantic list-item descendant subtree is represented by the supported M22 model. The Goldmark adapter reports a parser-independent immediate semantic child count for each supported item, including unsupported child items in that count. After M23 parent-ID resolution, `internal/splice` uses a non-recursive leaf-up O(l) pass to compare semantic vs supported child counts, propagate descendant completeness, and compute a private `ListSubtreeEnd`. A leaf remains complete with subtree end equal to its physical `LineRange.End`; an existing parent is appendable only when every child subtree is complete. The append patch is placed at the private subtree end and the existing M21 candidate parent-anchor proof remains authoritative. Unsupported direct/deep descendants fail closed with `ErrInvalidTargetKind`. No public subtree range is exposed.

M25 broadens the existing `PrepareRemoveListItem` operation from M18 leaf-line deletion to complete supported subtree removal. It reuses `ListSubtreeComplete`/`ListSubtreeEnd`, so a leaf retains its exact M18 line boundary while a complete parent deletes `[LineRange.Start, ListSubtreeEnd)`. The list survivor validator now accepts a set of intentionally removed snapshot IDs rather than one skip ID; replacement/move reuse the same set-aware path for their single target. Candidate supported-item count must equal original minus removed items, all other mappings/parent anchors remain exact modulo the patch, and a supported outer parent's semantic direct-child count must decrease by exactly one. Incomplete subtrees remain non-actionable, and no public list-subtree range is introduced.

M26 broadens `PrepareInsertListItemBefore` and `PrepareInsertListItemAfter` so their anchor may be a complete M24 list-item subtree while the caller fragment remains exactly one M19 same-shape leaf line. `before` inserts at the anchor physical-line start and `after` at `ListSubtreeEnd`; candidate validation additionally requires the inserted leaf and candidate anchor to have the same immediate semantic parent presence/anchor. M26 also makes list parent-anchor shifting source-owned: the first physical byte `[ListParentAnchor,ListParentAnchor+1)` is transformed instead of a generic zero-width point, so insertion exactly before a parent shifts descendant anchors while content replacement later in the same line leaves the parent start stable. No public list-subtree range is introduced.

M27 broadens `PrepareMoveListItemBefore` and `PrepareMoveListItemAfter` only on the destination side: the moved source remains one M20 leaf line, while the anchor may be a complete M24 subtree. `before` uses the anchor physical-line start and `after` uses `ListSubtreeEnd`. The M26 semantic sibling validator is generalized to accept one or more `patchTransform` values so insertion and atomic delete+insert movement share exactly one candidate-parent proof. Parent-anchor no-ops use the complete subtree boundary and are returned only when source and anchor already share the same semantic parent; otherwise the combined candidate must prove the requested sibling relation or fail closed.

M28 broadens the moved source to any complete supported list-item subtree while retaining the same public move methods. The source range is the M24/M25 `[LineRange.Start,ListSubtreeEnd)` ownership span; source and anchor subtrees must not overlap. Every supported moved descendant is validated at one subtree-relative offset with unchanged lexical mapping, child state/count, physical bytes, and shifted internal parent anchor, while the moved root is separately required to become a semantic sibling of the candidate anchor. Supported source/destination parent direct-child counts are checked with `-1/+1` deltas, including a zero-net same-parent reorder. M28 also consolidates the private complete-subtree target gate and same-sibling lexical proof, removing the obsolete leaf-only target helper and avoiding a redundant standalone parse of snapshot-owned moved source.

M29 broadens `PrepareInsertListItemBefore` and `PrepareInsertListItemAfter` from one M19 leaf fragment to one complete supported list-item subtree supplied by the caller. The fragment is parsed as an independent Marksplice document; its root must begin at byte zero, have no semantic list-item parent, pass M24 subtree-completeness proof, own exactly `[0,len(fragment))`, and reproduce the anchor's same-sibling lexical shape. The host candidate must contain exactly the standalone subtree's supported item count at the insertion span. Insertion and M28 movement share one subtree-placement validator that preserves every descendant's relative line/source/content ranges, marker state, child state/count, physical bytes, and internal parent anchor while separately requiring the inserted root to become a semantic sibling of the candidate anchor. M29 therefore adds no second subtree model, public subtree range, whitespace synthesis, or generic batch API.

Post-M29 consolidation keeps the growing list-item family separated by responsibility without changing the public contract. `list_item_model.go` owns supported parent resolution, deterministic leaf-up subtree completeness/end derivation, and private complete-subtree ownership; `list_item_edits.go` owns named structural-operation orchestration; and `list_item_validation.go` owns standalone-fragment, candidate-mapping, sibling, survivor, and subtree-placement proof. A shared lexical-mapping comparison covers exact line/source/content ranges, marker state, and physical-line bytes for both survivor and subtree validation. Survivor proof also validates every supported item's exact direct-child count, with explicit NodeID-keyed `-1/+1` deltas only for operation-known source/destination parents; this replaces separate remove/move/append parent-count checks and makes unchanged child counts fail closed as well as `HasChildren`. The leaf-up resolver stores its work arrays by compact supported-list ordinal rather than total document-node index, so the documented O(l) temporary-memory bound is literal even in documents dominated by non-list nodes. The consolidation does not add a persistent hierarchy index or change the O(l) parse-time hierarchy and O(n+k) mutation-validation complexity classes.

Observation-to-source capability mapping is likewise separated from document lifecycle/index construction. `node_mapping.go` owns base observation materialization plus small typed block/inline source-mapping functions; unsupported-shape sentinels and existing error wrapping remain feature-specific rather than being hidden behind a generic mapper. This removes the previous high-complexity `nodeFromObservation` switch from `document.go` without changing node IDs, editability gates, source ranges, table-row mapping cache semantics, or parser dependencies. The Goldmark table-cell observer also exposes only the `(Node, bool)` result it can actually produce instead of carrying an always-nil error channel.

M30 broadens `PrepareAppendListItemChild` from one M21/M24 leaf child to one complete supported direct-child subtree. Unlike M29 sibling fragments, a child fragment is not parsed standalone: valid GFM child indentation depends on the host parent's marker width, post-marker spacing, and container context, so a correctly indented child subtree may be indented code or otherwise meaningless as a root document. M30 therefore constructs the host candidate and runs one full `splice.Parse`; the resulting list model supplies both survivor mappings and private subtree completeness/ownership. The inserted root must begin exactly at the append boundary, resolve to the requested supported parent, pass `ListSubtreeComplete`, and own exactly the caller fragment span through `ListSubtreeEnd`. The parent direct-child count increases by exactly one regardless of descendant count, all original supported items retain transformed lexical/parent facts, and the candidate supported-item total must increase by exactly the inserted subtree's ID count. Multiple direct roots, trailing bytes outside the subtree, unsupported descendants, invalid indentation, and unsafe joins fail closed without whitespace or numbering synthesis. A leaf child remains the one-node degenerate subtree, preserving M21 behavior while avoiding both standalone-fragment rejection and a second candidate parse.

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
