# Marksplice Capabilities

This document is the source of truth for the **current user-visible capability boundary**. It answers what Marksplice can read, edit in existing source, and create today.

It is intentionally not a development diary. Historical design, verification, and parser-transition records are kept in the advanced/history section of the [documentation map](README.md#historical-engineering-records).

## Capability levels

Marksplice separates three questions:

1. **Read:** can a caller obtain a reviewed public representation or semantic view?
2. **Edit:** can Marksplice prepare a source-preserving edit for an existing document?
3. **Create:** can `DocumentBuilder` produce the construct deterministically?

A valid Markdown construct can be understood internally without receiving public edit authority. Existing-source mutation is promoted only when exact ownership and fail-closed behavior are proven.

## Markdown and document capabilities

| Family | Read | Edit existing source | Create new source | Important boundary |
| --- | --- | --- | --- | --- |
| Paragraphs | Yes | Replace, insert before/after, or remove promoted top-level paragraphs | Yes | Inserted payload must independently reparse as exactly one paragraph; container paragraphs are not automatically promoted |
| ATX headings | Yes, including parser-derived semantic text via `Heading.Text()` | Rename content and set level while preserving supported source style | Yes | Level changes reparse and validate the resulting section hierarchy |
| Setext headings | Yes, including parser-derived semantic text via `Heading.Text()` | Rename content; level 1↔2; single-line source-proven conversion to ATX for levels 3–6 | No dedicated Setext builder | Multiline Setext-to-ATX conversion remains fail-closed rather than rewriting heading content |
| Sections | Yes, derived from headings | Body/subtree replace, remove, sibling insert/move, direct-child append | Via headings/blocks | Section identity is the governing heading ID |
| Ordered/unordered lists | Yes for reviewed items/hierarchy | Content/subtree replace, remove, sibling insert/move, child append, and source-derived first-child insertion | Flat and reviewed homogeneous nesting | Structural edits require complete supported subtree ownership; first-child indentation/container trivia comes from the proven parent |
| Task lists | Yes | Toggle state plus list structural operations | Flat/nested ordered and unordered | State edit owns only the task marker byte |
| Fenced blocks | Complete top-level read view | Payload replacement, safe population of source-proven empty closed blocks, and info-string set/replace/clear | Yes, including empty payload | Fence character/length, indentation, closing style, and unrelated source are preserved; embedded language remains opaque data |
| GFM tables | Table/row/cell/alignments | Cell, row, alignment, and complete-column operations | Yes, including zero body rows | Column edits require complete table mapping |
| Code spans | Reviewed simple spans | Replace payload | Typed/raw construction | Ambiguous shapes fail closed |
| Emphasis / strong / strikethrough | Reviewed simple spans | Replace promoted payload | Typed/raw reviewed nesting | Existing-source compound shapes remain conservative |
| Direct links | Reviewed simple edit view plus broader semantic relationships | Replace promoted destination/label/title payload; add/remove title syntax for proven simple forms | Typed direct links with optional title | Title lifecycle owns only required separator/delimiters/payload; wrappers and destination spelling are preserved |
| Images | Reviewed simple edit view plus broader semantic relationships | Replace promoted destination/alt/title payload; add/remove title syntax for proven simple forms | Typed direct images with optional title | Title lifecycle preserves destination wrappers and unrelated source trivia |
| Reference definitions | Reviewed promoted definitions | Replace destination/title; add/remove title; coordinated rename; append canonical definitions; conservative complete-line removal | Immediate/deferred definitions | Definition/occurrence lifecycle requires unique normalized ownership; valid unpromoted definitions can still resolve relationships |
| Reference links/images | Semantic relationship read | Retarget parser-proven simple full/collapsed/shortcut occurrences to an existing unique definition | Full prior/forward and normalized collapsed/shortcut construction | Retargeting may promote collapsed/shortcut source to full form while preserving visible label text; unsupported/ambiguous occurrences fail closed |
| Autolinks | Yes | Replace supported token | Angle and parser-proven bare/extended construction | Exact token must remain a proven autolink |
| Thematic breaks | Promoted top-level line | Remove complete owned line | Yes | Removal validates surviving Markdown |
| Blockquotes | Complete promoted top-level containers plus per-line content ranges | Remove complete promoted container; replace complete content for uniform source-proven marker/EOL forms | Single/multi-block reviewed forms, bounded nesting | Lazy or mixed-prefix content replacement fails closed; alert-shaped blockquotes use alert-specific mutation |
| GitHub alerts | Semantic overlay on promoted blockquotes | Change alert kind and replace body for source-proven supported forms | `NOTE`, `TIP`, `IMPORTANT`, `WARNING`, `CAUTION` | Reuses blockquote identity/ownership; mixed/lazy body prefixes fail closed |
| Footnotes | Definitions, body ranges, references | Simple or source-proven multiline body replace; coordinated rename; append/remove complete definitions | Immediate/deferred definitions and typed references | Exact case-sensitive contract; multiline mutation requires uniform proven continuation/EOL layout |
| Mathematical expressions | Reviewed inline/block/fenced forms | Replace proven payload | Reviewed typed/block forms; fenced `math` via fenced code | Payload is opaque; no LaTeX/Math renderer |
| YAML/TOML front matter | Complete recognized envelope plus safe simple fields | Replace scalar value; rename/append/remove simple fields; add/remove complete leading envelope | Conservative canonical string fields | Structural authority remains limited to conservative simple fields/envelopes; no general YAML/TOML parser or serializer |
| HTML comments/anchors | Conservative promoted forms | Replace comment payload or quoted anchor value | No dedicated builder | Other HTML remains opaque source |

## Canonical Markdown rendering

A parsed `Document` can be exported as deterministic canonical Markdown without changing the source-preserving edit path or mutating its immutable source snapshot.

| Capability | API | Boundary |
| --- | --- | --- |
| Streaming canonical output | `Document.RenderCanonicalMarkdown` | Writes one deterministic Marksplice-profile Markdown representation to caller `io.Writer`; stops on writer error |
| Buffered canonical output | `Document.CanonicalMarkdown` | Returns caller-owned complete canonical bytes; convenient when whole-output buffering is acceptable |
| Semantic round-trip | Both APIs | Reparsing the canonical result must reproduce the Native semantic facts used by the writer; the renderer does not introduce a second Markdown parser or AST |
| Byte idempotence | Both APIs | Rendering an already canonical result produces the same bytes |
| Formatting policy | One built-in profile | LF output and deterministic block/list/table/fence/reference choices; no general style/pretty-printer knobs |
| Existing-source authority | None | Canonical output is a separate export; ordinary `Prepare...`/`ChangeSet` editing continues to preserve unrelated source bytes |

Opaque code/raw payload stays data under its semantic owner. Canonical rendering performs no filesystem discovery, URL fetching, network access, command execution, templates, syntax highlighting, or mathematical-engine execution.

## HTML rendering

A parsed `Document` can be rendered explicitly without changing the source-preserving edit path.

| Capability | API | Boundary |
| --- | --- | --- |
| Streaming fragment output | `Document.RenderHTML` | Writes body fragments to caller `io.Writer`; stops on writer error |
| Buffered fragment output | `Document.HTML` | Returns caller-owned fragment bytes; convenient when whole-output buffering is acceptable |
| Streaming standalone output | `Document.RenderHTMLDocument` | Writes deterministic doctype/html/head/charset/body around the same fragment renderer; no template or asset system |
| Buffered standalone output | `Document.HTMLDocument` | Returns caller-owned complete-document bytes |
| Fragment source mapping | `RenderHTMLWithSourceMap`, `HTMLWithSourceMap` | Optional snapshot-local Markdown byte ranges correlated with emitted fragment byte ranges; event-granular, overlapping where semantics nest, and not total byte coverage |
| Standalone source mapping | `RenderHTMLDocumentWithSourceMap`, `HTMLDocumentWithSourceMap` | Same correlation with absolute complete-document output offsets; synthetic wrapper bytes remain unmapped while eligible reviewed metadata maps to emitted head/lang bytes |
| Reviewed metadata mapping | `HTMLMetadataFrontMatter`, `HTMLMetadataOmit` | Exact lower-case `title`, `description`, `author`, `lang` only from unique top-level source-proven simple front-matter scalars; no general YAML/TOML interpretation |
| Raw HTML | `HTMLRawPreserve`, `HTMLRawEscape` | Preserve is not sanitization; escape explicitly for an HTML trust boundary |
| Dangerous URLs | `HTMLUnsafeURLSuppress`, `HTMLUnsafeURLAllow` | Default suppresses dangerous schemes by emitting an empty destination; no URL is fetched |
| GFM tag filter | `HTMLTagFilterEnabled`, `HTMLTagFilterDisabled` | Default applies the published GFM disallowed-tag filter to preserved raw HTML |
| Code blocks | Deterministic `<pre><code>` fragments | Language metadata may become a class; no syntax-highlighting engine runs |
| Footnotes, tasks, tables | Deterministic semantic HTML | Reuses the Native semantic walk; renderer does not reparse Markdown |
| Mathematical forms | Deterministic opaque wrappers | No LaTeX/MathJax/KaTeX interpretation or execution |

Front matter and reference-definition declarations are source/semantic metadata and are not emitted as visible fragment blocks. Source maps describe only emitted semantic correlations; they do not fabricate ranges for declarations or synthetic markup that produces no corresponding semantic output. Rendering performs no filesystem discovery, asset fetch, network access, command execution, template execution, or embedded-language execution.

## Query and navigation

| Capability | Available | Boundary |
| --- | --- | --- |
| Bounded node queries | `QueryNodes` | Positive result limit required; source ordered; no persistent query index |
| Bounded section queries | `QuerySections` | Positive limit; optional level/range filters |
| Heading semantic text | `Heading.Text` | Parser-derived human text without exposing parser-internal AST types |
| Heading anchors | `HeadingAnchors`, `HeadingAnchor` | GitHub-compatible derivation with duplicate handling |
| Fragment resolution | `ResolveFragment`, `ValidateFragment` | Heading-derived and supported explicit HTML anchors |
| TOC generation | `GenerateTOC` | Deterministic from current section hierarchy |
| Existing TOC synchronization | `TOCStale`, `PrepareSyncTOC` | Only caller-designated conservative managed-TOC bodies |
| Link intelligence | `LinkRelationships` | Read-only semantic relationships; destinations outside the current document remain caller-interpreted data unless an explicit adapter such as `workspacefs` is used |

## Multi-document capabilities

The root package keeps multi-document graph/validation APIs explicit and in-memory. The separate `workspacefs` package can load that input from a caller-supplied `fs.FS`; the filesystem object and finite limits are the caller's explicit authority boundary.

| Capability | API | Boundary |
| --- | --- | --- |
| Filesystem discovery | `workspacefs.Scan` | Read-only `.md`/`.markdown` discovery under one caller-supplied `fs.FS` root; deterministic slash-relative keys |
| Filesystem relationship following | `workspacefs.Follow` | Starts from explicit Markdown entries; resolves reviewed relative slash-based Markdown URI paths with source-relative dot-segment normalization, one percent-decode of path components, query/file separation, preserved fragments, cycle-safe traversal, and finite caller limits. Absolute/scheme/protocol-relative/backslash/encoded-traversal-or-separator/directory/extensionless forms are not filesystem targets; case/symlink semantics come from the supplied `fs.FS`. |
| Filesystem resource limits | `workspacefs.Options`, `workspacefs.Limits` | Positive document/byte/relationship limits plus a non-negative scan-depth or follow-hop limit; exhaustion fails with `workspacefs.ErrBudgetExceeded` |
| Explicit document graph | `BuildDocumentGraph` | No discovery in the root package; resolver may target only documents already supplied |
| Outgoing links/backlinks | `Outgoing`, `Backlinks` | Immutable graph results |
| Reachability/related documents | `ReachableFrom`, `RelatedDocuments` | Deterministic graph traversal |
| Workspace validation | `ValidateWorkspace` | Caller resolver classifies ignored/resolved/missing targets |
| Diagnostics | `WorkspaceReport.Diagnostics` | Fragments, missing docs, conservative unresolved references, roots/orphans, managed TOCs |
| Safe repair planning | `WorkspaceReport.RepairPlan` | Automatic repair limited to proven managed-TOC synchronization |
| Knowledge metadata | `BuildKnowledgeIndex` | Caller-declared aliases, tags, logical references only; no syntax inference |

## Third-party read-only observations

`ParseWithOptions` can run explicitly registered, namespaced recognizers after the core parse.

Extensions may retain validated snapshot ranges and scalar attributes under caller-provided limits. They cannot:

- replace/reclassify core GFM nodes;
- register core `Kind` values;
- gain generic mutation or builder authority;
- change graph/workspace resolution policy;
- expose parser-internal types;
- acquire filesystem, network, or command authority from Marksplice.

Recognizers are ordinary caller-linked Go code, so their own execution remains governed by caller trust.

## Source-preservation guarantees

For ordinary existing-document edits:

- `Document` is an immutable snapshot;
- public ranges are half-open byte offsets into that snapshot;
- a prepared `ChangeSet` is bound to the exact source it was prepared against;
- stale application reports `ErrSourceConflict`;
- bytes outside operation-owned patches are not regenerated;
- candidate reparsing/proof rejects edits that would create unsupported surrounding structural changes;
- `ComposeChanges` combines only independently compatible changes from the same snapshot.

New-document construction is intentionally different: `DocumentBuilder` emits canonical LF GFM because there is no existing author formatting to preserve. Canonical Markdown rendering is also intentionally separate: it normalizes only an explicitly requested export and never becomes the implementation path for a source-preserving edit.

## What Marksplice deliberately does not do

Marksplice does not provide:

- PDF rendering;
- implicit Markdown-to-Markdown whole-document formatting/normalization as the ordinary edit path; explicit canonical Markdown export is available only through `RenderCanonicalMarkdown` / `CanonicalMarkdown`;
- hidden/implicit filesystem crawling or file loading outside an explicitly supplied `workspacefs` `fs.FS`;
- URL fetching or network resolution;
- command execution or embedded-language execution;
- syntax highlighting, diagram rendering, or asset fetching;
- arbitrary YAML/TOML serialization;
- LaTeX/MathJax/KaTeX rendering;
- a collection of first-party dialect extensions.

Application-specific syntax can be observed through the opt-in read-only extension SPI without changing the core Markdown profile.

## Markdown profile

Marksplice exposes one Markdown profile: CommonMark 0.31.2 as the normative base grammar, with explicit published GFM extensions/corrections layered on top. Footnotes, alerts, mathematical expressions, and front matter are separately reviewed Marksplice capabilities rather than hidden parser modes.

For normal usage, continue with the [User Guide](guide.md) or [Recipes](recipes/README.md). For normative/parser details, see [Markdown Conformance Policy](gfm-conformance.md) and [Architecture](architecture.md).
