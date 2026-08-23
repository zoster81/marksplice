# Marksplice Capability Matrix and Roadmap

Status: source of truth for the current product-facing read/edit/create capability surface and forward roadmap.

This document answers three separate questions for each Markdown family:

1. can Marksplice understand the construct semantically under the GFM profile;
2. can callers read a reviewed public structural representation from a parsed immutable snapshot;
3. can callers safely edit existing source or construct new canonical source through reviewed APIs.

These are intentionally different capability levels. A construct may be semantically understood and preserved without being publicly actionable. Public promotion requires reviewed source ownership, caller-facing semantics, and fail-closed behavior.

Normative GFM behavior belongs to [`gfm-conformance.md`](gfm-conformance.md). Durable implementation boundaries belong to [`architecture.md`](architecture.md). Parser/source ownership belongs to [`goldmark-capability-matrix.md`](goldmark-capability-matrix.md). Milestone-specific contracts and verification evidence belong to [`milestones/`](milestones/).

## Current capability matrix

| Family | Semantic understanding | Public read | Existing-document edit | New-document construction | Current boundary |
| --- | --- | --- | --- | --- | --- |
| Paragraphs | Yes | Top-level promoted paragraphs | Replace top-level paragraph content | Raw GFM single-line/parser-proven LF-multiline; typed simple inline content | Container paragraphs are not automatically promoted |
| ATX headings | Yes | Level, style, content range | Rename content while preserving source style | Levels 1–6 from raw GFM or typed simple inline content | Construction uses canonical ATX syntax |
| Setext headings | Yes | Level, Setext style, content range | Rename content while preserving underline/source | Not dedicated | Existing Setext source is preserved; builder does not synthesize Setext headings |
| Sections | Derived from promoted headings | Parent, direct body, subtree range, immediate child heading IDs | Remove; replace body/subtree; insert/move siblings; append direct child | Through heading/block construction rather than a section builder | No separate section identity namespace |
| Unordered lists | Yes | Supported list items, parent/children, subtree range when complete | Content/subtree replace; remove; sibling insert/move; child/subtree append | Flat and homogeneous nested | Existing source markers/indentation are caller source; builder uses canonical `-` |
| Ordered lists | Yes | Supported list items, parent/children, subtree range when complete | Same structural operations as unordered lists | Flat and homogeneous nested | Existing numbering is preserved; builder numbers per container |
| Task lists | Yes | Checked state plus owning list structure | Set checked state; list structural operations reuse list proof | Flat/nested ordered and unordered | Builder writes canonical `[ ]`/`[x]` |
| Fenced code | Yes | Supported exact contiguous content range | Replace content | Single-line or LF-multiline with optional info string | Unsupported lossy body shapes remain non-editable |
| GFM tables | Yes | Comparable `Table`; promoted non-empty cells/body rows; table-owned row/header navigation | Cell replacement; row replace/remove/insert/move; table-level compatible row append; complete-column insert/remove/move | Canonical tables, including header+delimiter tables with zero body rows | `BodyRowCount` is semantic and may be zero; row/cell IDs are promoted subsets; column edits require complete source mapping of every semantic row |
| Table alignment | Yes | `Document.TableAlignments` and `Document.TableRowAlignments` | Set one column or atomically replace full vector while preserving delimiter trivia | Explicit default/left/right/center | Existing edits change only source-proven delimiter syntax; column edits carry semantic alignment through structural changes |
| Emphasis | Yes | Simple source-proven spans | Replace span content | Raw GFM or typed `EmphasisInline` with semantic text plus bounded reviewed code/emphasis/strong/strikethrough children | Existing-source compound spans remain non-editable; ambiguous generated delimiter hierarchies fail closed |
| Strong emphasis | Yes | Simple source-proven spans | Replace span content | Raw GFM or typed `StrongInline` with the same bounded reviewed structured children | Existing-source simple delimiter boundary is unchanged |
| Strikethrough | Yes | Simple source-proven spans | Replace span content | Raw GFM or typed `StrikethroughInline` with semantic text plus bounded code/emphasis/strong children | Direct strikethrough-in-strikethrough construction is rejected; existing-source boundary is unchanged |
| Code spans | Yes | Simple source-proven spans | Replace span content | Raw GFM or typed `CodeInline` with adaptive backtick fences | Shapes requiring whitespace/delimiter normalization remain rejected |
| Inline links | Yes | Simple destination range plus parser-proven semantic destination and optional title | Replace destination | Raw GFM; typed direct form with canonical angle destination/title and bounded reviewed structured label children; full prior-reference via `ReferenceLinkInline`; explicit full forward-reference via `ForwardReferenceLinkInline`; normalized collapsed/shortcut forms via dedicated constructors | Existing-source read/edit remains the simple source-proven subset; broader reference syntax/relationships remain outside ordinary promotion and are completed by M99 |
| Images | Yes | Simple inline-image destination range | Replace destination | Raw GFM; typed direct form with canonical angle destination/title and bounded reviewed structured alt children; full prior-reference via `ReferenceImageInline`; explicit full forward-reference plus normalized collapsed/shortcut image forms | Existing-source read/edit remains the simple source-proven subset; broader reference-image syntax is not newly promoted for ordinary editing |
| Reference definitions | Yes | Supported single-line destination range plus parser-proven label, destination, and optional title | Replace destination or existing title payload; remove exact complete line only when surviving reference relationships remain unchanged | Canonical immediate or explicitly deferred no-title/conservative double-quoted definitions | Public `Range()` remains the destination span; removal uses private complete-line ownership and fails closed for used definitions; relationship intelligence remains M99 |
| Autolinks | Yes | Supported token range plus parser-proven semantic value and email classification | Replace supported token | Raw GFM; canonical typed angle `AutoLinkInline`; parser-proven exact bare/extended token via `BareAutoLinkInline` | Typed bare construction requires the complete requested token to reparse as one source-proven non-angle autolink |
| Thematic breaks | Yes | Promoted top-level source-proven physical-line range | Remove exact owned line with candidate survivor proof | Canonical `---` | Nested breaks remain internal; removal fails closed on unsafe joins |
| Blockquotes | Yes | Promoted complete top-level containers with exact owned physical range plus per-line inner source ranges; historical simple single-paragraph `ContentRange` retained when applicable | Remove the complete owned container with whole-block survivor proof | One paragraph at depth 1 or explicit depth 2–64; multi-block depth 1–64 composition from reviewed child builders, including recursive blockquote children when every structural chain stays within 64 total levels | Multiline, nested, lazy-continuation, marker-only, and multi-block existing source is promoted only when every physical line is source-proven; nested internal blockquotes do not receive separate public identities |
| YAML front matter | Marksplice envelope recognition outside GFM parser | Unique simple scalar fields | Replace safe scalar value | Canonical document-leading envelope with conservative double-quoted string fields | Complex/ambiguous YAML remains opaque; construction is intentionally not a general YAML serializer |
| TOML front matter | Marksplice envelope recognition outside GFM parser | Unique simple scalar fields | Replace safe scalar value | Canonical document-leading envelope with conservative double-quoted string fields | Complex/ambiguous TOML remains opaque; construction is intentionally not a general TOML serializer |
| HTML | GFM raw/block HTML semantics | Simple comments and quoted `<a id>`/`<a name>` anchors only | Replace proven comment payload or anchor value | No dedicated builder | Other HTML is preserved conservatively as opaque source |

## Cross-cutting guarantees

Parsed documents are immutable source snapshots. Public `NodeID` values are deterministic only within one snapshot and are not durable identities across arbitrary revisions. Public ranges are half-open byte offsets into that snapshot, and `Document.SourceRange` returns caller-owned copies.

M97 adds bounded immutable-snapshot selectors: `Document.QueryNodes` filters promoted nodes by reviewed kind, optional containing source range, and mandatory positive result limit; `Document.QuerySections` filters the existing derived `Section` model by heading level, optional containing range, and the same explicit result bound. Query results remain in authoritative source order, are caller-owned, retain no query state, and create no persistent query index. `NodeMatch.Range()` reuses the matched kind's existing typed operation-oriented `Range()` semantics for selection only; it is not a new generic mutation span.

Existing-document mutations are prepared as minimal source-bound patches. Untouched bytes are not regenerated, prepared changes reject stale source, and operations fail closed when candidate parsing/source proof cannot preserve required invariants. `Document.ComposeChanges` can atomically combine independent opaque `ChangeSet` values already prepared against the same exact snapshot; source overlap, overlapping structural/reference deltas, or a combined candidate that differs from the independently validated model fail closed. It does not expose raw patches or arbitrary byte batching. Marksplice does not use whole-document AST serialization as the ordinary edit path.

New-document construction is deliberately separate. `DocumentBuilder` may use deterministic canonical GFM because there is no pre-existing author formatting to preserve. Reviewed constructed blocks and the final document are reparsed and checked against requested semantic/source expectations before bytes are returned.

Semantic parsing is broader than the public actionable API. Unsupported or not-yet-reviewed shapes remain understandable/preservable rather than being exposed with guessed source ranges or mutation semantics.

The current production maintainability gate is cyclomatic complexity 15 or lower per function (`gocyclo -over 15` must be empty for production code), plus production and test-inclusive `unparam`. The post-M79 whole-code review established this stricter gate without changing public APIs, kind ordinals, `NodeID` derivation, generated bytes, source ownership, parser profile, or fail-closed behavior.

## Completed capability families

The current conservative model is functionally complete through M97. Detailed chronology belongs to milestone records; the durable capability families are:

- **Mapped public editing foundation (M1–M11):** source-preserving paragraph/heading/list/task/table-cell/fenced-code/simple-inline/link/image/front-matter/HTML capabilities, snapshot-bound identity, and bounded source reading.
- **Sections and list hierarchy (M12–M34):** exact section body/subtree operations, sibling/child structure, complete supported list-subtree ownership, source-preserving structural edits, and compact parent/child navigation.
- **Table structural model (M35–M43, M63–M70):** source-proven body rows and table identity, compact ownership/navigation, semantic alignments, row mutation/append, and conservative complete-column insert/remove/move.
- **New-document block construction (M44–M62):** canonical headings/paragraphs/lists/tasks/fenced code/reference definitions/tables/thematic breaks/simple blockquotes, including nested list depth, adaptive fences, titles, and table alignment.
- **Promoted line-owned blocks (M71–M74):** public top-level thematic-break/simple-blockquote ownership and exact fail-closed removal.
- **Typed inline/reference construction (M75–M79, M87–M89, M92–M93):** semantic text, code, emphasis/strong, link/image, strikethrough, and autolinks; M87 adds conservative source-proven double-quoted titles, M88 bounded construction-only nesting, M89 exact-prior full references, M92 structured direct/full-reference labels and alt content, and M93 explicit deferred full-forward references, normalized collapsed/shortcut link/image forms, and parser-proven exact bare/extended autolink tokens without widening ordinary parsed reference-link promotion.
- **Front-matter construction (M80):** one optional leading YAML/TOML envelope with deterministic LF formatting, ordered unique simple fields, conservative double-quoted string values, and proof through the existing source-layer front-matter mapper.
- **Broader blockquote construction (M81–M86):** M81 extends `AppendBlockquote` to one parser-proven LF-multiline depth-1 paragraph; M82 adds explicit depth 2–64 single-paragraph construction; M83 adds `AppendBlockquoteBlocks` for depth 1–64 child-builder composition; M84–M85 add the remaining reviewed non-blockquote body families; M86 admits recursive blockquote children while bounding every total structural chain at 64. Marksplice derives every canonical `> ` prefix and proves the exact construction-only hierarchy; M94 later broadens parsed-source promotion through a separate ownership proof rather than treating construction support as authorization.
- **First public beta readiness (M90):** keeps release/version state outside runtime code while adding portable pkg.go.dev examples, cross-platform Go 1.26/1.27 CI, dependency-update metadata, beta/release/security documentation, and an external consumer-module verification path for `github.com/zoster81/marksplice`.
- **Repository-layout hardening (M91):** no capability/API change; removes black-box test clutter from the module root, groups root source files by API/builder responsibility, adds a documentation index, and adopts cross-package coverage instrumentation appropriate to consumer-style tests.
- **Existing-source blockquote completion (M94):** promotes one complete top-level blockquote container across source-proven multiline, nested, lazy-continuation, marker-only, and multi-block forms; exposes caller-owned per-physical-line inner ranges, preserves the historical simple `ContentRange` contract, and removes only the complete owned container after strengthened survivor proof.
- **Structural mutation composition (M95):** `Document.ComposeChanges` combines independent already-prepared same-snapshot mutations into one atomic `ChangeSet`, reuses the existing source-patch overlap authority, composes compact independently validated node/reference deltas, and proves one combined candidate while failing closed on logical overlap or cross-operation Markdown interaction.
- **Single-document audit closures (M96):** promoted simple inline links/reference definitions/autolinks expose already parser-proven scalar semantics, and canonical table construction supports zero body rows under a stronger complete-table construction expectation. The audit explicitly defers relationship intelligence to M99, generalized fenced-block ownership to M103, and metadata generalization to M106.
- **Bounded structural selectors (M97):** `QueryNodes` and `QuerySections` scan the existing immutable source-ordered structural arrays with explicit positive result limits, optional containing source ranges, fixed-size kind/level filters, and no persistent query index or second document model. Node matches reuse the existing typed `Range()` contract for their kind rather than inventing generic source ownership.

## Forward roadmap

The roadmap is split into six ordered phases. The milestone numbers express the intended dependency order, not permission to weaken a gate to keep numbering intact. Every slice must preserve source ownership, stale-source rejection, deterministic construction, bounded resource use, parser isolation, and the production complexity gate.

Marksplice does **not** maintain a family of first-party syntax extensions. A broadly useful Markdown/document capability belongs in core when its contract is general enough. Dialect-specific or product-specific syntax stays outside core. M110 may expose a controlled public SPI so independent Go packages can implement such syntax without becoming Marksplice dependencies or bypassing Marksplice safety invariants.

### Phase A — complete the single-document core

**M92 — Structured link/image label and alt composition — complete.** Typed construction now lets direct and full-reference link labels/image alt content use the already-reviewed bounded inline composition model. Destination/title/reference semantic proof remains independent from child-inline hierarchy proof, nested link/image/autolink/reference children remain rejected, and ambiguous delimiter hierarchies fail closed.

**M93 — Reference and autolink completion — complete.** Typed construction now supports parser-normalized collapsed/shortcut reference links/images, explicit deferred definitions plus full forward-reference constructors, and exact parser-proven bare/extended autolink tokens while preserving M89's prior-definition full-reference contract. Existing promoted single-line reference definitions can replace an existing title payload and can remove their complete physical line only when candidate proof preserves every surviving full/collapsed/shortcut/image reference relationship; ordinary parsed reference links/images remain non-promoted.

**M94 — Existing-source blockquote completion — complete.** Parsed snapshots now promote one complete top-level blockquote container across multiline, nested, lazy-continuation, marker-only, and multi-block forms only when exact physical ownership is proven. `Document.BlockquoteContentRanges` exposes caller-owned inner per-line spans; lazy markerless lines require parser semantic proof on that physical line, nested blockquotes remain internal children rather than receiving public identities, and complete-container removal strengthens survivor validation across the full blockquote mapping. Historical single-paragraph `ContentRange` and construction authorization remain unchanged.

**M95 — Structural mutation composition — complete.** `Document.ComposeChanges` atomically combines independent opaque prepared mutations only when every constituent is bound to the same exact snapshot. The source layer revalidates all private patches as one non-overlapping original-coordinate set; splice composes compact independently validated node/reference model deltas and proves one combined candidate. Overlapping patch/model regions, same logical aggregate edits, and cross-operation Markdown interactions fail closed; arbitrary raw byte-patch batching is not public.

**M96 — Single-document read/edit/create audit — complete.** The family-by-family audit closed two high-value gaps already supported by proven internal models: promoted inline links/reference definitions/autolinks expose existing parser-proven semantic scalar facts, and canonical GFM table construction accepts zero body rows under a complete `KindTable` construction expectation. Broader link relationships stay M99, generalized fenced-block ownership stays M103, and metadata generalization stays M106; no symmetry-only APIs were added.

### Phase B — document intelligence

**M97 — Structural query surface — complete.** `Document.QueryNodes` and `Document.QuerySections` provide caller-bounded source-ordered selection over the existing immutable node/section arrays. Node queries filter reviewed public kinds plus optional complete containment in a snapshot range; section queries filter levels plus optional complete subtree containment. Both require an explicit positive result limit, reject malformed/oversized filters, retain no query state, and add no persistent query index or second document model.

**M98 — Anchors, fragments, and TOC.** Derive GitHub-compatible heading anchors, duplicate-anchor disambiguation, fragment resolution/validation, TOC generation, stale-TOC detection, and source-preserving TOC synchronization from Marksplice headings/sections. These are native document capabilities, not an extension dependency.

**M99 — Link intelligence.** Complete intra-document relationship understanding across inline/reference links, images where relevant, reference definitions, fragments, supported explicit anchors, and heading-derived anchors. Expose outgoing relationships and the facts needed by later graph validation without adding filesystem authority.

**M100 — Multi-document graph.** Build links, fragments, backlinks, reachability, and related-document relationships over an explicit caller-provided document set. Core receives documents/authorized resolution from the caller and does not silently crawl the filesystem or network.

**M101 — Workspace validation and repair planning.** Add bounded caller-authorized resolution plus diagnostics for broken local links, missing fragments, ambiguous anchors, unresolved references, orphan/reachability problems, and stale generated indexes. Where a repair is provably safe, produce a deterministic repair plan that compiles down to the ordinary validated mutation machinery.

### Phase C — broadly useful core capabilities inspired by ecosystem use cases

[`extension-strategy.md`](extension-strategy.md) records the feature ideas reviewed from the wider Markdown ecosystem. Those projects are idea sources only; Marksplice does not reproduce their extension model or adopt their packages as core architecture.

**M102 — Semantic block patterns.** Add reviewed semantic recognition/construction for broadly useful patterns that are already valid baseline Markdown. The first concrete target is GitHub alerts (`NOTE`, `TIP`, `IMPORTANT`, `WARNING`, `CAUTION`) represented over blockquote source. Pattern semantics must not change the underlying parser grammar or source-preservation rules.

**M103 — Fenced-block semantics.** Generalize fenced-code ownership around exact fence, info string/language, content, and source ranges. Technical payload names such as `mermaid`, `geojson`, `topojson`, `stl`, `math`, `d2`, or other languages remain data values: Marksplice does not parse, execute, render, or validate those embedded languages.

**M104 — Footnotes.** Add footnote references/definitions as a core capability if their exact syntax/source contract is approved: read, relationships, construction, source-preserving edits, multiple-reference behavior, and graph integration. This is a Marksplice capability, not a separately enabled first-party extension.

**M105 — Mathematical expressions.** Add the useful GitHub-compatible Markdown source semantics for inline and block mathematical expressions where the syntax contract is explicit. Mathematical payloads remain opaque; MathJax, KaTeX, MathML, and other rendering engines stay outside core.

**M106 — Metadata/front-matter generalization audit.** Reassess the current conservative YAML/TOML envelope against real metadata use cases. Generalize only source ownership and document-envelope semantics that are broadly useful; Marksplice must not become a general YAML/TOML serializer or normalize unknown metadata.

**M107 — Knowledge-document primitives.** Add only syntax-independent primitives that improve document relationships, aliases, logical references, tags, or knowledge-graph querying. Do not automatically put wikilink, hashtag, definition-list, heading-attribute, fenced-div, emoji, Discord, Obsidian, or other dialect syntax into core; those are primary candidates for third-party packages through M110 if users need them.

### Phase D — hardening before parser ownership

**M108 — Fuzzing, pathological input, and performance hardening.** Expand block/inline/source-map/mutation fuzzing, deep/malformed/oversized-input tests, allocation/CPU benchmarks, and large-document scaling evidence. Establish explicit resource/performance budgets before the native parser is implemented.

**M109 — Public API coherence and stabilization review.** Review naming, typed views, mutation composition, query/graph APIs, builder behavior, error taxonomy, boundedness, concurrency assumptions, documentation, and future third-party extensibility as one library. Remove accidental redundancy before it becomes a long-term compatibility burden.

### Phase E — third-party extensibility boundary

**M110 — Public third-party syntax/semantic SPI.** Design the smallest safe Go extension contract that lets independent packages add dialect-specific syntax or semantics without becoming Marksplice core features. Prefer normal statically linked Go packages, not Go's runtime `plugin` mechanism. Any SPI must preserve core GFM behavior, namespacing/isolation, bounded parsing, exact source ownership, validated mutation, deterministic construction, and the filesystem/network authority boundary. An external package must not gain a bypass around raw patch validation or redefine core syntax semantics.

M110 is an extensibility boundary, not a bundle of Marksplice-maintained extensions. Wikilinks, hashtags, definition lists, custom/fenced containers, emoji shortcodes, product-specific Markdown, and similar dialects are examples of what third parties may choose to implement.

### Phase F — Marksplice-native parser and Goldmark removal

Goldmark is a temporary implementation dependency. The required end state is a Marksplice-native CommonMark/GFM parser with **no Goldmark dependency in `go.mod`**. No Goldmark upgrade or migration is part of this roadmap.

**M111 — Native parser contract and differential harness.** Freeze the parser observations actually required by Marksplice and build a differential harness over the complete applicable pinned GFM corpus plus every Marksplice semantic/source-position regression. The existing Goldmark backend is only a temporary oracle while the replacement is built.

**M112 — Marksplice-native block parser.** Implement native document/block parsing for the required CommonMark/GFM structures with source observations designed directly for Marksplice: headings, paragraphs, lists/tasks, blockquotes, fenced/indented code, thematic breaks, HTML blocks, reference definitions, tables, and associated container relationships.

**M113 — Marksplice-native inline parser.** Implement native text/escape, code span, emphasis/strong/strikethrough, links/images, references, autolinks, raw HTML, and required GFM inline semantics. Integrate the M110 third-party hook at a controlled parser boundary rather than attaching it after the native parser is complete.

**M114 — Full native conformance and parser hardening.** Require the native parser to pass every applicable pinned GFM example and focused Marksplice parser/source regression, plus fuzzing, malformed/pathological/deep input tests, resource bounds, CPU/allocation benchmarks, cross-platform tests, and the ordinary complexity/static-analysis gates. Production cutover is blocked until this gate is green.

**M115 — Goldmark removal and final cutover.** Switch all production parsing and construction proof to the Marksplice-native parser; remove `github.com/yuin/goldmark` from `go.mod`/`go.sum`; delete the Goldmark adapter and Goldmark-specific compatibility implementation; and rerun the complete release-quality verification stack. M115 is complete only when no non-historical runtime, test, documentation, or dependency path requires Goldmark.

## Current next boundary: M98 anchors, fragments, and TOC

M97 is complete with focused TDD, documented-tree release-quality verification, corrected cross-package coverage, and final hygiene recorded in its milestone file. The next implementation slice is M98 exactly as defined above: derive GitHub-compatible heading anchors, fragment resolution/validation, and TOC generation/synchronization without weakening snapshot/source-preservation boundaries.

Goldmark remains pinned at the current reviewed line only while Marksplice is completed and the native parser is developed. The upstream extension catalog is used only to identify generally useful product ideas. The roadmap ends at M115 with Goldmark removed completely.
