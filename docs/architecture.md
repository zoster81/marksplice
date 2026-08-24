# Marksplice Architecture

Status: source of truth for durable architecture decisions.

## Mission

Marksplice is a Pure-Go library for creating and source-preserving structural manipulation of GitHub Flavored Markdown (GFM). GFM is the project's single normative Markdown syntax profile.

Marksplice has two deliberately separate core paths:

```text
                       caller intent
                            |
                 +----------+----------+
                 |                     |
          existing GFM bytes      new document
                 |                     |
               Parse            DocumentBuilder
                 |                     |
        immutable snapshot       canonical writer
                 |                     |
       semantic + source map      parser/model proof
                 |                     |
        structural operation            |
                 |                     |
        minimal byte patches            |
                 +----------+----------+
                            |
                         GFM bytes
```

Existing documents are immutable source snapshots. Ordinary edits use minimal source-bound patches and never authorize normalization of untouched source. New documents may use deterministic canonical GFM because there is no author formatting to preserve. New-document writing must never be substituted for the existing-document edit path.

M0 is the green repository-bootstrap baseline and M1–M103 are complete. Detailed milestone chronology, feature contracts, and verification evidence live in `docs/milestones/`; this document records only architecture that remains durable across those milestones.

## Package boundaries

- `internal/parser/goldmark` owns Goldmark-specific parsing, AST traversal, and narrowly scoped GFM compatibility behavior. No Goldmark type crosses this boundary.
- `internal/source` owns parser-independent byte ranges, lexical source proof, source fingerprints, validated patches, and patch application.
- `internal/splice` combines parser observations with source mappings, builds immutable snapshot indexes/relationships, and prepares structural mutations.
- `internal/publictest` owns black-box tests that import the root module API exactly as an external Go consumer would.
- root package `marksplice` owns reviewed public types and operations. It may wrap internal implementation values but must not expose Goldmark or `internal/*` types.

The public package deliberately remains at the Go module root because the canonical import path is `github.com/zoster81/marksplice`. Moving that package under a top-level `src/` directory would change the natural import path to `github.com/zoster81/marksplice/src` or require a forwarding facade whose only purpose is repository cosmetics. Root Go filenames are therefore grouped by responsibility instead: `api*.go` owns parsed-document/read/edit APIs and `builder*.go` owns new-document construction plus its package-private proof/writer helpers. Longer-form documentation stays under `docs/`, while external-style public API tests stay out of the root under `internal/publictest/`.

Keep orchestration separate from syntax-specific proof. Shared plumbing may centralize target lookup, patch transforms, candidate assembly, and candidate parsing, while lexical/source mappers and semantic validators remain feature-specific when their safety invariants differ. Reuse focused lexical primitives rather than duplicating scanners across families.

## Source model and public promotion

Source is represented as bytes. Ranges are half-open byte offsets `[start,end)` into one immutable snapshot. Byte offsets, not rune indexes, define mutation boundaries.

A parsed document may retain compact derived indexes and adjacency needed for efficient navigation, but the ordered node collection remains the source of structural iteration order. Snapshot `NodeID` values are deterministic and comparable only within the snapshot whose source facts produced them; they are not durable identities across arbitrary revisions. Duplicate derived IDs must fail closed rather than silently select one node.

Semantic recognition alone is not enough for public promotion. An actionable public kind requires reviewed caller-facing semantics plus exact source ownership for the operation it exposes. Unsupported or ambiguous semantic shapes remain internal/non-editable instead of receiving guessed ranges or partial mutation guarantees.

Typed public ranges are operation-oriented. Marksplice intentionally does not expose one generic "full node span" contract: a paragraph range, list-item content range, list subtree range, table-cell range, blockquote line range, or link destination range means exactly the bytes documented by that typed capability.

`Document.SourceRange` validates a snapshot-local public range and returns caller-owned bytes. Public accessors returning variable-length relationships or metadata return defensive copies so callers cannot mutate internal snapshot state.

## Structural query model

M97 adds bounded selectors over the same immutable snapshot model; it does not create a second AST, mutable query document, or persistent selector index.

`Document.QueryNodes` scans the existing source-ordered internal node array and returns only nodes already eligible for ordinary public promotion. Callers may filter by reviewed public `Kind`, optional complete containment in one snapshot-local byte range, and a mandatory positive result limit. `Document.QuerySections` reuses the existing compact section array and `Section` representation, with optional level and complete-subtree containment filters plus the same explicit positive result bound.

`NodeMatch.Range()` is intentionally not a generic full-node source span. It reuses the exact operation-oriented range already defined by that node kind's typed `Range()` accessor and exists only for selection/bounded reading. Mutation authority remains in the typed mutation operations and their source/candidate proof.

Queries preserve authoritative source/structural order and stop as soon as the caller's limit is reached. Kind and section-level filters are represented by fixed-size boolean arrays and reject oversized duplicate input vectors. Query-specific snapshot projection lives in `internal/splice/query.go`: `NodeSelectionAt` returns one shared scalar node summary plus the already-reviewed selection range without cloning the full internal node or its variable-length table/blockquote metadata, while `SourceLen` validates `Within` without copying source bytes. Ordinary `NodeSummaryAt` enumeration remains query-independent and reuses the same scalar summarization helper.

No query state is retained in `Document`. Navigation and relationship intelligence do not extend M97 into a string/regex/graph query language.

## Navigation model

M98 derives heading anchors, fragment targets, and TOCs from the same immutable structural snapshot rather than creating a navigation AST or persistent index. The parser adapter contributes one plain semantic `HeadingText` string for promoted document headings; parser-specific AST types remain confined to the adapter, and source ownership/ranges remain independent from that semantic text.

Heading anchors are recomputed in source order with an ephemeral duplicate-reservation map. Fragment resolution URI-decodes one local fragment and scans heading-derived anchors plus the already-promoted source-proven HTML anchor subset; zero or multiple matches fail closed. Explicit HTML anchors do not participate in heading duplicate numbering or generated TOC entries. Broader link/reference relationship enumeration remains M99.

TOC generation reuses the existing `Section` hierarchy and parent links. A generated standalone TOC is deterministic LF Markdown. Existing-source synchronization is narrower: the caller designates one section, its exact `BodyRange` must be empty/blank or match the conservative managed local-fragment-list grammar, its established line-ending form and leading/trailing blank trivia are preserved, and the replacement is then validated through the ordinary `PrepareReplaceSectionBody` candidate proof. Arbitrary prose/lists never become raw rewrite authority merely because they resemble navigation content.

No anchor map, fragment cache, TOC cache, or second section tree is retained in `Document`. M108 owns any later benchmark-driven indexing decision.

## Relationship intelligence model

M99 generalizes M93's compact parser-resolved reference-usage safety vector into one source-ordered `LinkUsage` vector covering direct/reference links, direct/reference images, and autolinks. The parser adapter contributes only Marksplice-owned scalar facts: semantic kind/form, source anchor, reference value when applicable, resolved destination/title, and autolink email classification. Recognized front-matter-envelope usages are filtered because front matter remains outside Markdown semantics in the snapshot.

`Document.LinkRelationships` is a read-only projection over that vector, not a new mutation model. A relationship may exist even when its source shape is outside the narrower editable public node subset. `SourceNodeID` is exposed only when the exact direct relationship already corresponds to an existing promoted node. `ReferenceDefinitionID` is exposed only when parser-defined normalized label plus resolved destination/title facts identify exactly one existing promoted definition; semantically valid but unpromoted definitions still resolve the relationship without gaining a fabricated public owner identity.

Only semantic destinations beginning with `#` are resolved inside the current snapshot. M99 reuses the M98 fragment normalization/target model through one ephemeral fragment catalog per relationship-enumeration call; other-document paths and external URLs remain opaque relationship data. Unresolved reference-looking source is not reinterpreted lexically as a relationship; that diagnostic boundary remains M101.

No normalized-definition map, fragment catalog, relationship index, backlink map, or graph is retained in `Document`. M99 builds ephemeral source-node/definition/fragment maps once per enumeration call so projection is expected O(N+H+R) rather than all-pairs. The authoritative retained relationship state remains the source-ordered `linkUsages` scalar vector.

## Multi-document graph model

M100 introduces `DocumentGraph` as a separate immutable aggregate over an **explicit caller-provided set** of already parsed `Document` snapshots. The graph does not discover documents. Callers assign opaque `DocumentKey` values and may provide a build-only `DocumentResolver` that maps one non-local M99 `LinkRelationship` to a `DocumentKey` already present in the input set plus an optional target fragment. Keys and destinations are data: Marksplice performs no path normalization, directory traversal, file reads, URL fetching, or network access.

A semantic destination beginning with `#` is always local to its source snapshot and becomes a self-edge without invoking the resolver. Every other relationship is admitted only when the resolver explicitly returns true. A successful resolution to an empty or unknown key fails the complete graph build with `ErrInvalidGraph`; returning false simply leaves that relationship outside the resolved graph. The resolver is never retained, so graph reads cannot trigger later policy callbacks or I/O.

`DocumentGraph` stores each resolved `GraphEdge` exactly once in deterministic document/source order. Outgoing and backlink adjacency store integer edge indices rather than duplicate edge values. Repeated relationships remain repeated edges because source occurrence multiplicity is semantic graph data. `ReachableFrom` is iterative breadth-first traversal with visited-before-enqueue cycle handling; `RelatedDocuments` is the unique direct incoming/outgoing neighbor set returned in caller document order. Public variable-length results are defensive copies.

Cross-document fragments reuse M98, but without rebuilding the same target catalog for every edge. During graph construction M100 creates a fragment resolver lazily for each target document that actually receives a resolver-provided fragment; that closure captures one ephemeral M98 fragment catalog and is discarded when `BuildDocumentGraph` returns. `DocumentGraph` retains only the resolved optional `FragmentTarget`, never the catalog or resolver. Missing/ambiguous/malformed fragments therefore keep their document edge but have no `FragmentTarget`.

## Workspace validation model

M101 layers diagnostics and conservative repair planning over M98–M100 without turning Marksplice into a workspace crawler. `ValidateWorkspace` receives the complete explicit `GraphDocument` set, optional caller roots, explicit managed-TOC targets, and a build-time `WorkspaceResolver`. The resolver classifies each non-local M99 relationship as `Ignore`, `Resolved`, or `Missing`; Marksplice never derives path semantics from a destination and never retains the resolver after validation.

`WorkspaceReport` combines the resolved immutable M100 graph, defensive-copy diagnostics, and a repair plan. Source-bound findings carry the caller's `DocumentKey` plus exact source offset when one exists; orphan findings are target-only because no single source occurrence causes reachability failure. Local fragment findings reuse the existing M99/M98 status directly. Cross-document fragments use the same build-local target fragment catalogs as graph construction, and caller `Resolved` decisions are cached for the duration of validation so user policy is never invoked twice.

Unresolved references require one additional parser-independent scalar observation because a failed reference parse leaves no M99 relationship. The Goldmark adapter therefore records only conservative explicit full/collapsed unresolved link/image forms from eligible text runs, using the parser's real reference context and existing label normalization. Escaped, code/raw-HTML-owned, multiline/complex, front-matter, and shortcut-only bracket text are excluded rather than guessed. The immutable snapshot retains only the compact source-ordered unresolved-reference vector; no parser context or secondary AST survives parsing.

Root-relative orphan detection uses one private multi-source BFS over M100 adjacency, O(V+E) regardless of root count. Managed TOC ownership/status is batched per document; heading sets and generated TOC bytes are call-local. Repair authority is intentionally narrower than diagnostics: M101 automatically repairs only caller-designated M98-recognized stale TOCs. Multiple stale TOCs in one snapshot use one family-specific atomic multi-patch candidate proof and yield at most one ordinary snapshot-bound `ChangeSet` per document. Generic M95 `ComposeChanges` is not weakened when independently prepared generated-index changes overlap semantically.

## Semantic block pattern model

M102 treats GitHub alerts as a Marksplice-owned semantic overlay over already-promoted top-level blockquotes rather than as a new parser grammar or structural node kind. `Alert.ID()` is the underlying blockquote `NodeID`; `Alert.Range()` is the existing complete blockquote source range; the exact first M94 inner range supplies the reviewed uppercase alert marker; and body ranges are defensive copies of the remaining M94 physical inner ranges. Recognition is recomputed in source order when requested and stores no alert index or parser state in `Document`.

Construction keeps alert intent distinct from generic private blockquote intent so an alert cannot later be admitted as a child through `AppendBlockquoteBlocks` or another alert. The alert writer reuses the canonical depth-1 blockquote writer, and multi-block alerts reuse the existing child-builder snapshot plus blockquote inner-source proof. Candidate validation first proves the ordinary blockquote hierarchy/source mapping, then checks the exact expected marker and source mapping. No Goldmark alert extension, public `Kind`, separate identity namespace, renderer behavior, or alert-specific existing-source mutation authority is introduced.

## Fenced-block model

M103 separates complete fenced-container readability from the historical contiguous payload-mutation contract. `FencedBlock` is an immutable read-only view over one source-proven top-level GFM fenced container and shares the existing fenced-code snapshot identity rather than creating a second public `Kind` or identity namespace. Its complete range, opening/optional closing delimiter runs, indentation, parser-proven info string/language, and per-physical-line payload ranges are source facts; empty and unclosed forms are represented without synthetic payload bytes.

Goldmark continues to own fenced-code grammar, info semantics, and language derivation. A narrow adapter-local block-parser decorator delegates the complete fenced-code parser while retaining only the opening fence position on the resulting temporary AST node, including for empty bodies whose `Lines()` carry no source segment. That position is an anchor, not ownership: `internal/source` independently proves the opening fence, exact info bytes, every parser-backed physical payload line, optional matching closing fence, and complete container boundary against the immutable source snapshot. Parser/source disagreement fails closed.

The historical `FencedCode` view and `PrepareReplaceFencedCode` remain available only when the semantic payload is one exact contiguous source span. Broader `FencedBlock` readability therefore does not authorize generic rewrites of empty or non-contiguous payloads. Construction may emit an empty fenced body canonically with adjacent opening/closing fence lines, but embedded payload languages remain opaque data: core performs no embedded-language parsing, validation, execution, highlighting, or rendering.

Fenced-block proof is O(F) in the physical lines owned by the container and retains at most one source-backed payload range per semantic body line. Enumeration scans the existing source-ordered node collection and adds no persistent fenced-block index, language registry, secondary AST, or embedded-language parser state.

## Existing-document mutation model

A prepared change contains the source fingerprint it was created against and one or more validated, non-overlapping byte patches.

Application rules are:

1. fingerprint the supplied source;
2. reject stale source explicitly;
3. validate patch ranges and ordering;
4. apply patches in source coordinates without rendering unrelated source;
5. preserve every byte outside changed ranges.

One named structural operation may own multiple coordinated patches, such as delete+insert movement. M95 additionally lets callers combine multiple **already-prepared** opaque `ChangeSet` values through `Document.ComposeChanges`, but this still is not a generic public patch API. Every constituent must be bound to the same exact snapshot; source-level patch overlap and same-offset insertion conflicts are rejected before semantic composition.

For two or more composed changes, Marksplice applies each constituent to the original snapshot to derive its independently validated compact node/link-relationship model delta, rejects overlapping logical deltas, composes the expected deltas in model/source order, then parses one combined candidate and requires the result to equal that expected composed model. Node IDs and absolute offsets are not composition identity; semantic facts, source-owned shape, relative relationships, table/list/blockquote ownership facts, and parser-resolved link relationships are. This catches cross-operation interactions such as two removals that are safe individually but together change paragraph structure. Family-specific atomic operations remain preferable when several intents deliberately change one logical model region.

When a local byte patch could change surrounding Markdown interpretation, mutation preparation reparses a candidate and validates the required surviving source/semantic model. Candidate proof is a conservative safety oracle, not permission to rewrite the document. Unsafe joins, newly introduced promoted structure, changed container relationships, ambiguous ownership, and malformed source fail closed.

No operation synthesizes whitespace, line endings, numbering, or delimiters unless that synthesis is part of an explicit new-document construction contract. Existing lexical trivia is source data.

## Section model

Sections are derived from promoted document-level headings rather than assigned a second identity namespace. A section distinguishes the governing heading, direct body, and complete subtree. Parent/child relationships derive from heading levels.

The section hierarchy is built once from source-ordered headings using a monotonic stack. Heading `NodeID` remains the section identity. Immediate child navigation reuses the same hierarchy with compact scalar linkage rather than storing slices inside the comparable public `Section` value.

Section mutations operate on exact stored boundaries. Body replacement preserves the heading and child sections; subtree replacement/insertion/movement validate caller or snapshot-owned section structure; direct-child append requires exactly parent level + 1. Structural candidate validation preserves unaffected heading level/style/source mapping and rejects Setext/paragraph or other join reinterpretation. No operation rewrites heading levels or manufactures separator whitespace.

## List model

Promoted list items retain exact first-line content replacement semantics while internal source ownership records the complete physical line. Supported single-line-head parent items may expose semantic `HasChildren`, immediate supported `ParentID`/`ChildIDs`, and `SubtreeRange()` only when the complete supported descendant subtree is proven.

Parent identities are resolved after ordinary snapshot IDs exist. Downward navigation uses compact source-ordered adjacency. Temporary anchor/ordinal maps are discarded after parse; no second persistent hierarchy index is introduced.

Subtree completeness compares semantic direct-child counts with supported promoted children and propagates completeness leaf-up. Structural removal, movement, sibling insertion, child append, and subtree replacement require the relevant complete source-owned subtree. Unsupported descendants make the structural boundary unavailable rather than guessed.

Host context is authoritative where GFM indentation depends on parent marker width and container context. Child insertion/replacement is therefore validated in the host candidate instead of forcing standalone parseability. Marksplice never synthesizes indentation, list numbering, markers, or line endings for existing source.

Movement uses exact snapshot-owned subtree bytes and coordinated original-coordinate patches. Overlapping source/anchor subtrees are rejected. Same-parent no-ops remain fingerprint-bound. Candidate proof validates descendant-relative mappings, parentage, source bytes, and operation-known parent child-count deltas.

## Table model

Tables have a source-proven public identity independent of any body row. `Table.Range()` owns the complete header + delimiter + semantic body source block. `ColumnCount()` is the source-proven semantic width; `BodyRowCount()` is semantic, may be zero, and can exceed the number of promoted body-row identities.

Goldmark provides semantic table hierarchy/alignment values, but Marksplice derives lexical ownership independently. The source table anchor comes from the public header position rather than trusting container position. Header, delimiter, body rows, cell ranges, and delimiter alignment tokens are mapped in the source layer.

Public rows/cells keep comparable scalar values. Variable-length table relationships and alignment vectors are document accessors returning caller-owned storage. Parse-time temporary maps build compact row/header/cell adjacency and scalar table ownership; persistent anchor maps are avoided.

Row mutations preserve exact caller/source row bytes, validate compatible width/table ownership in the host candidate, and never synthesize a missing line separator. Row movement is restricted to the same table under the current contract.

Alignment mutation changes only source-proven delimiter colons while preserving each exact dash run and surrounding trivia. Whole-vector alignment mutation batches changed delimiter tokens into one candidate validation cycle.

Column insert/remove/move uses a stricter operation-time capability gate: header, delimiter, and every semantic body row must all map at exactly the table width. Removal owns one complete slot plus an adjacent separator; movement permutes source-proven contents while destination-slot trivia remains fixed; insertion clones adjacent slot padding/separators and delimiter dash-run style. Target-table rows/cells may structurally change, while non-target table survivors still undergo ordinary transformed-range/source validation. These operations add no persistent column index.

## Blocks, inline syntax, metadata, and HTML

Paragraphs/headings, fenced blocks/code, simple strikethrough/code/emphasis/strong, links/reference definitions/autolinks/images, simple front-matter fields, and narrow HTML comment/anchor forms follow one mapped-capability rule: parser semantics identify the family, then Marksplice proves exact operation-oriented source ownership before public editability.

M103 exposes complete fenced-container metadata through the read-only `FencedBlock` model after independent source proof, while fenced-code payload replacement remains limited to the historical exact contiguous `FencedCode` span. Exact delimiter runs, indentation, info source, complete container ownership, closure state, and per-line payload source are therefore public/readable where proven without becoming generic mutation spans. Simple inline spans reject shapes that require normalization or ambiguous delimiter ownership. Link/image/reference/autolink mutations preserve labels, wrappers, titles, spacing, and other source trivia outside the exact destination/token range. M96 allows promoted simple `InlineLink`, `ReferenceDefinition`, and `AutoLink` values to expose scalar semantic facts that were already parser-proven and stored internally: link destination/optional title, definition label/destination/optional title, and autolink value/email classification. These getters do not redefine the operation-oriented `Range()` contracts or widen parsed-source promotion. M93 extends promoted single-line reference-definition ownership with an internal complete physical `LineRange`; M99 generalizes the same parser-resolved safety metadata across direct/reference links, images, and autolinks and exposes it only as immutable relationship intelligence. Complete-line removal and composition now preserve every surviving link relationship, while reference/direct shapes outside ordinary edit promotion remain non-editable.

Thematic breaks own one complete physical line. M94 blockquotes instead own one complete top-level physical container across source-proven multiline, nested, lazy-continuation, marker-only, and multi-block forms. The public `Blockquote.Range()` is the complete owned source span; `Document.BlockquoteContentRanges` returns caller-owned inner ranges for each owned physical line, preserving nested markers as inner source and representing marker-only lines with valid empty ranges. The historical single-paragraph `ContentRange()` remains available only when that exact old contract is parser-proven; broader forms return the zero range rather than a synthetic span across markers. A markerless lazy line is owned only when a parser semantic block range proves content on that exact physical line. Nested blockquotes remain semantic children of one promoted outer top-level identity rather than gaining separate public identities. Complete-container removal uses the shared whole-block survivor proof plus transformed blockquote line/marker/content-range validation so paragraph/Setext/container joins and surviving blockquote ownership cannot silently change.

Front matter is a Marksplice-owned document envelope outside the GFM parser. Only a closed byte-zero `---` YAML or `+++` TOML envelope with at least one unique source-proven simple scalar field is recognized. Complex, ambiguous, duplicate-only, non-leading, or unclosed forms remain ordinary GFM source. Existing-document front-matter edits patch only proven scalar values and preserve all other envelope bytes. New-document construction owns at most one envelope as separate `DocumentBuilder` state and writes conservative canonical double-quoted string fields before every GFM body block.

HTML remains conservative. Goldmark may recognize raw/block HTML semantically, but public edits are limited to source-proven simple comment payloads and quoted `id`/`name` values on supported `<a>` openings. Other HTML is preserved opaquely.

## New-document construction model

`DocumentBuilder` is deliberately separate from parsed `Document` snapshots. Construction state represents intent before authoritative source exists, so it has no snapshot `NodeID`, source fingerprint, or stale-source semantics.

The builder writes deterministic LF GFM, one blank line between retained GFM blocks, and one final LF for a non-empty document. An optional YAML/TOML front-matter envelope is separate document-leading state rather than a `constructionBlock`; when a body exists, exactly one blank line separates the closing envelope delimiter from the first GFM block. Each requested block is validated, written, reparsed, and checked against family-specific semantic/source expectations before retention; `Markdown()` validates the complete generated sequence again.

Historical block/list/table builder APIs accept caller-provided raw GFM where documented. Typed-inline APIs are a separate semantic-text path and do not change those raw-source contracts.

Canonical construction currently includes a single optional YAML/TOML front-matter envelope with conservative string fields, ATX headings, parser-proven paragraphs, thematic breaks, single-paragraph blockquotes at depth 1 or explicit nesting depth 2–64 with canonical repeated `> ` prefixes on every LF-separated physical line, multi-block blockquotes at depth 1–64 composed from every reviewed body-block family including recursively nested blockquotes with total structural depth bounded at 64, exact depth-1 GitHub alerts from single-paragraph/typed-inline or reviewed multi-block bodies, flat/homogeneous nested ordered/unordered lists and task lists, fenced code including canonical empty payloads, reference definitions, and GFM tables with optional explicit alignment and optional body rows.

Nested list construction accepts structural depth rather than source indentation. The writer derives indentation from the generated parent content column and uses container-local ordered numbering. Fenced-code construction selects adaptive backtick fences against every potentially closing body run. Reference definitions use angle destinations and a conservative no-escape title form. Tables use canonical outer pipes/padding and canonical alignment delimiters. M96 gives every generated table an explicit complete-table construction expectation, so a header+delimiter table with zero body rows is accepted only after the same table anchor/range/width/alignment/source mapping is proven; body-row expectations remain additive when rows exist.

Typed inline construction represents semantic text and reviewed inline structures rather than raw Markdown injection. Text is deterministically escaped. Code uses adaptive backticks under the existing exact source-mapped code-span boundary. M88 permits bounded structured nesting of code, emphasis, strong, and strikethrough inside emphasis/strong/strikethrough wrappers. Link/image typed construction uses strict canonical angle destinations; M87 adds separate `WithTitle` constructors that write conservative non-empty double-quoted titles and prove exact title presence/value/source range through the existing mappings. M89 adds canonical full reference-link/reference-image construction whose exact reference label must match exactly one already-appended top-level reference definition in the same builder. M92 lets direct and full-reference link labels/image alt content reuse the M88 structured child family while keeping destination/title/reference semantic proof separate from child hierarchy proof. M93 adds explicit deferred top-level definitions plus `ForwardReference*Inline` for full forward references, parser-normalized collapsed/shortcut reference link/image forms, and `BareAutoLinkInline` for exact parser-proven non-angle autolink tokens. Historical M89 constructors still consult only already-appended definitions; normalized ambiguity and parser-truncated bare tokens fail closed. Nested links/images/autolinks/references inside structured labels remain rejected, and ambiguous delimiter hierarchies still fail closed.

Construction proof reuses the same parser-independent source mappings and semantic metadata as parsed snapshots rather than creating a parallel Markdown AST. Front-matter construction specifically reuses `MapLeadingFrontMatter`; body expectations are written into the already-prefixed final buffer so their byte ranges are final rather than shifted after envelope insertion. M81–M86 blockquote construction deliberately established construction-only semantic validators plus Marksplice lexical proof while ordinary parsed-source blockquotes were still narrow. M94 later broadens ordinary top-level blockquote promotion through a separate physical-ownership proof; the construction validators remain independent, so the ability to parse or construct a broader blockquote never by itself authorizes a different construction shape or existing-source edit. M82 proves explicit single-paragraph structural depth. M83 snapshots a child `DocumentBuilder`, renders its already-reviewed blocks through the ordinary construction writers, prefixes every physical line including canonical blank separators, and proves that removing those prefixes yields the exact canonical inner source. M84–M85 extend the construction-only Goldmark comparison to list/list-item hierarchies, reference definitions, and table/header/row/cell hierarchies with alignment values. M86 admits blockquote nodes into that same iterative child comparator and performs an explicit iterative pre-render traversal of the private construction tree so every nested blockquote chain remains within the 64-level bound before recursive canonical writing can occur. Observations wholly contained by a construction-only multi-block blockquote are excluded from ordinary construction-node matching because the container-specific proof is authoritative there. Front matter remains a document envelope. M87 extends the existing typed-inline expectation proof with title value/range facts only; it adds no parser/source mapper. M88 adds a separate ephemeral construction-inline hierarchy proof for code/emphasis/strong/strikethrough wrappers rather than widening ordinary simple-inline promotion. The writer bounds wrapper nesting at 64, alternates `*`/`_` for nested emphasis-family delimiters, rejects direct strikethrough-in-strikethrough, and still fails closed when GFM reparsing cannot reproduce an exact requested hierarchy. Goldmark matching is single-pass with anchor buckets and precomputed parent→children lists, while simple leaf/source-mapped expectations remain independently validated through the existing parsed snapshot path. M89 reuses the same construction-only boundary for full reference links/images: the builder resolves one exact existing definition, emits the requested reference token, and validates it against an ephemeral proof source containing canonical definitions. The proof requires Goldmark `ReferenceLinkFull`, exact reference value, resolved destination/title semantics, and exact label/reference source ranges; the proof definitions are never emitted into the requested block and ordinary parsed-source link/image promotion remains unchanged. M92 extends the hierarchy proof so a direct or full-reference link/image can own reviewed structured children, but does not move destination/title/reference facts into that hierarchy contract: direct link/image semantics use a separate construction-only expectation, full references retain the M89 semantic expectation, and the hierarchy validator receives reference facts only to synthesize the ephemeral definitions required to parse the requested reference owner. M93 keeps GFM reference-label normalization behind the parser adapter, separates prior and deferred builder scopes, and extends construction-only reference expectations with full/collapsed/shortcut source form. For existing-source removal safety the same ordinary AST walk also records a compact Marksplice-owned vector of non-public reference usages; recognized front-matter-envelope anchors are filtered because front matter is not Markdown in the Marksplice model. Whole-block removal compares only surviving usages with transformed anchors, so an owned block may remove its own relationships while external relationships cannot silently change. Ordinary public parsed reference-link/image promotion remains unchanged. Successful generated source may then be passed to `Parse` to enter the immutable snapshot/editing model.

## GFM and Goldmark boundary

`docs/gfm-conformance.md` owns the normative source hierarchy, pinned GFM snapshot, conformance procedure, and update policy. CommonMark is inherited as GFM's base syntax; Marksplice exposes no separate CommonMark mode.

Goldmark is the current temporary semantic parser implementation, configured for GFM plus only narrowly scoped compatibility behavior required by the approved contract. Do not expose Goldmark AST/types publicly, serialize its AST as the existing-document edit path, or expand Marksplice syntax merely because Goldmark supports additional constructs. M102 alert semantics intentionally require no Goldmark alert extension: ordinary blockquote parsing remains the semantic substrate and the marker overlay is Marksplice-owned. M103 adds only a narrow adapter-local decorator around Goldmark's public fenced-code block parser so the temporary AST retains the opening fence position; the delegate still owns the grammar and the recorded position is only an anchor for independent Marksplice source proof.

Where Goldmark positions or public semantic state are insufficient for lossless source work, Marksplice may perform bounded lexical scans tied to the source snapshot. Such scans remain parser-independent and internal; Marksplice must not copy/fork Goldmark implementation code to obtain lexical trivia.

YAML/TOML front matter remains a separate document-envelope layer, not an additional Markdown dialect. HTML rendering is not a product capability; if rendering is later added, GFM rendering-specific requirements such as tag filtering become explicit acceptance criteria.

### Core capability and third-party extensibility boundary

`docs/extension-strategy.md` owns the feature-selection and future extensibility policy. Marksplice does not maintain a collection of first-party syntax extensions.

Broadly useful capabilities belong directly in core when their contract is general enough: examples include heading anchors/TOC, semantic patterns over already-valid Markdown, generic fenced-block metadata, and other document/navigation primitives. Dialect-specific spellings such as wikilinks, hashtags, definition lists, custom containers, emoji shortcodes, or product-specific Markdown stay outside core unless a later explicit core decision establishes broad value.

M110 may expose a small public SPI so independent, statically linked Go packages can contribute dialect-specific syntax or semantics. The SPI must not let third-party code redefine baseline GFM semantics, bypass stale-source or patch validation, gain filesystem/network/command authority, or expose parser-specific types through Marksplice public APIs. Extensibility must be designed as part of the Marksplice-native parser boundary rather than as a collection of one-off parser switches.

### Goldmark exit architecture

Goldmark is a temporary implementation dependency. The public API, source model, mutation model, construction model, and conformance expectations must remain Marksplice-owned so the parser can be replaced without changing consumer-visible contracts.

The roadmap ends with a Marksplice-native CommonMark/GFM parser and complete removal of `github.com/yuin/goldmark` from the dependency graph. Goldmark remains only while the rest of the engine is completed and while the native parser is developed against a differential harness. No Goldmark upgrade or migration is part of the roadmap.

The native parser must pass the full applicable pinned GFM corpus, every focused semantic/source-position regression, malformed/deep/oversized-input safety tests, fuzz/resource tests, performance/allocation benchmarks, and maintainability gates before production cutover. During transition, the current Goldmark backend may be used as a temporary differential oracle; after cutover, Goldmark-specific adapter and compatibility code is removed.

## Line endings, Unicode, and encoding

Existing-source edits must not normalize LF, CRLF, isolated CR, Unicode content, or unrelated byte sequences. If parser compatibility requires a same-length shadow view, byte offsets must remain exactly aligned with original source.

Encoding/BOM preservation belongs to the host that supplies Markdown bytes unless Marksplice later introduces an explicit encoding-aware layer. Core must not guess and rewrite file encodings.

New-document construction currently emits canonical LF by contract.

## Performance and complexity

Parsing/index construction should remain linear or near-linear in source size where practical. Relationship models use compact source-ordered arrays/scalar links plus temporary maps rather than redundant persistent indexes. Mutation planning should avoid repeated whole-document rescans beyond what the operation's safety proof explicitly requires. M95 composition currently performs O(k·N) proof work for `k` supplied prepared changes over a document/model of size `N`: one independently prepared candidate comparison per constituent plus one combined candidate parse. It uses no all-pairs node matching, no repeated list-subtree hashing, and no persistent composition cache/index. M97 node/section queries are O(N) worst-case scans with early stop at the explicit caller limit and O(limit) result memory; they add no persistent kind/range index or result sort. M98 heading-anchor/TOC derivation is O(H) expected for `H` headings with O(H) temporary/result state; fragment resolution is O(H+N) over headings and existing nodes, and managed-TOC recognition is O(B) over the designated body bytes. M99 `LinkRelationships` builds ephemeral source-node, normalized-definition-owner, and fragment maps once, then projects `R` relationships in expected O(N+H+R) time and O(N+H+R) temporary/result memory; it retains no graph/index. M100 graph construction is O(V+R+E) plus M99 projection/resolver cost and at most one additional ephemeral fragment-catalog build per target document that receives cross-document fragments; persistent graph state is O(V+E) with one edge vector plus adjacency indices. `ReachableFrom` is O(V+E) worst case and `RelatedDocuments` is O(V+degree). M101 validation invokes caller resolution once per non-local relationship, computes all root reachability with one O(V+E) multi-source BFS, batches managed-heading ownership and generated TOC bytes per document/EOL, and plans multiple stale TOCs with O(T log T) patch ordering plus one combined candidate parse/proof per repaired document. Resolver callbacks, fragment catalogs, heading sets, traversal state, and TOC caches are call-local only. M102 alert enumeration is O(N+L) worst case over promoted nodes and inspected blockquote content-range entries, with O(A) caller-owned result storage and no persistent semantic index. M103 fenced-block source proof is O(F) in the owned physical fence/body lines and stores at most one source-backed range per semantic payload line; `FencedBlocks` reuses the existing source-ordered node scan and retains no persistent fence or language index. M108 owns measurement-driven performance hardening of realistic mutation, query, navigation, relationship, graph, workspace-validation, semantic-pattern, and fenced-block workloads.

The current production complexity gate is **cyclomatic complexity 15 or lower per function**: `gocyclo -over 15 -ignore '_test\.go$' .` must report no production function. Production and test-inclusive `unparam` checks are also required. This gate is a maintainability constraint, not an instruction to fragment cohesive lexical/state-machine logic; split code when responsibility, reuse, or verification clarity improves.

The post-M79 whole-code review reduced prior complexity hotspots by separating Goldmark block/inline dispatch, list marker parsing, table row/cell survivor proof, whole-block anchor validation, table alignment planning/candidate proof, table/list adjacency validation, complete-table mapping, YAML scalar validation, and shared link/image destination-tail parsing. The post-M97 review additionally isolates reference-link/reference-image construction and proof in `builder_inline_reference.go`, keeps generic typed-inline writing in `builder_inline.go`, and separates query-specific internal projection from the general snapshot document implementation. M98 similarly keeps managed-TOC orchestration separate from its small line-scanning state machine after the initial recognizer exceeded the complexity gate. M99 removes the superseded reference-only parser aliases/wrapper and replaces its first repeated definition/fragment scans with ephemeral lookup maps. M100 reuses one shared public fragment-target conversion helper, replaces repeated cross-edge `ResolveFragment` rescans with build-local per-target fragment resolvers, and stores adjacency as edge indices rather than edge copies. M101 separates conservative unresolved-reference observation from M99 relationship semantics, replaces per-root reachability and per-target heading/TOC rescans with multi-source/batched derivation, and uses a family-specific atomic TOC repair path after generic M95 composition correctly rejected overlapping generated-link semantic deltas. M102 factors quoted-block construction dispatch and blockquote base/simple/alert proof responsibilities after the first implementation exceeded the complexity gate, while reusing one canonical inner-source writer rather than duplicating blockquote logic. M103 centralizes public/internal range conversion and reuses one already-proven opening-fence mapping path for both the complete `FencedBlock` proof and the historical contiguous `FencedCode` proof, avoiding a second opening-fence scan. These refactors change no kind ordinals, `NodeID` derivation, source ownership, parser profile, or fail-closed mutation behavior; M103's only construction widening is the explicit canonical empty fenced payload.

## Safety and authority boundaries

Prepared mutations fail closed on stale source, malformed/overlapping ranges, invalid target kinds, incomplete ownership, or ambiguous structure. Core code does not perform arbitrary filesystem traversal, network requests, command execution, or host authorization.

External paths, URLs, and relationship targets are data unless a caller outside core explicitly provides bounded authority. M100 implements this boundary by accepting only already-parsed caller-provided documents plus a build-only resolver whose successful target must already belong to that explicit set; it performs no filesystem/network discovery. M101 preserves that boundary: its resolver explicitly distinguishes ignored, resolved, and caller-declared missing relationships, is never retained, and cannot authorize filesystem/network discovery. Roots and managed indexes are caller-provided; Marksplice does not choose or crawl them implicitly.

Dependencies should remain minimal. Exact dependency versions belong in `go.mod` and `go.sum`; licensing belongs in `LICENSE` and `NOTICE`.

## Public module and release boundary

The Go module path is `github.com/zoster81/marksplice`. `go.mod` owns the minimum supported Go version and exact dependency graph; public release tags own module versions. Runtime code does not contain a second release-version authority. While the API remains under active development, public releases stay in the v0 series and may use explicit pre-release identifiers. Published Go module tags are immutable. Public versioning, first-push readiness, CI compatibility policy, and proxy/pkg.go.dev verification are owned by `docs/releasing.md` rather than by runtime packages.

M90 adds no Markdown/runtime behavior. It prepares public Go-module consumption through portable package examples, cross-platform public CI, dependency-update metadata, beta/release documentation, security-reporting guidance, and an external consumer-module compilation test. The separately licensed, hash-pinned GFM conformance corpus remains external to the repository and therefore remains a stricter maintainer gate rather than a vendored public-CI input.

M91 is a repository-layout refactor only. It moves the black-box public API tests from the module root to `internal/publictest`, groups root package filenames into `api*` and `builder*` families, and adds `docs/README.md` as the documentation index. It does not change the module path, package name, exported API, Markdown behavior, or ordinary consumer import path. Coverage gates after M91 must use cross-package instrumentation because the consumer-style tests now live in a separate package directory.

## Public capability and history references

`docs/README.md` is the documentation index. `docs/capabilities.md` is the authoritative current read/edit/create matrix and forward roadmap. `docs/goldmark-capability-matrix.md` records parser/source ownership by syntax family. `docs/releasing.md` records public module/version policy. `docs/milestones/` retains the detailed M0–M103 contracts, design evolution, and historical verification evidence.
