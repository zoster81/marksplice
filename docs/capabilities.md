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
| Paragraphs | Yes | Replace promoted top-level content | Yes | Container paragraphs are not automatically promoted |
| ATX headings | Yes | Rename content, preserving source style | Yes | Builder uses canonical ATX headings |
| Setext headings | Yes | Rename content, preserving underline/style | No dedicated Setext builder | Existing Setext source remains preserved |
| Sections | Yes, derived from headings | Body/subtree replace, remove, sibling insert/move, direct-child append | Via headings/blocks | Section identity is the governing heading ID |
| Ordered/unordered lists | Yes for reviewed items/hierarchy | Content/subtree replace, remove, sibling insert/move, child append | Flat and reviewed homogeneous nesting | Structural edits require complete supported subtree ownership |
| Task lists | Yes | Toggle state plus list structural operations | Flat/nested ordered and unordered | State edit owns only the task marker byte |
| Fenced blocks | Complete top-level read view | Payload replacement only through narrower editable `FencedCode` shapes | Yes, including empty payload | Embedded language is opaque data; no execution/rendering |
| GFM tables | Table/row/cell/alignments | Cell, row, alignment, and complete-column operations | Yes, including zero body rows | Column edits require complete table mapping |
| Code spans | Reviewed simple spans | Replace payload | Typed/raw construction | Ambiguous shapes fail closed |
| Emphasis / strong / strikethrough | Reviewed simple spans | Replace promoted payload | Typed/raw reviewed nesting | Existing-source compound shapes remain conservative |
| Direct links | Reviewed simple edit view plus broader semantic relationships | Replace promoted destination | Typed direct links with optional title | Relationship visibility can be broader than editability |
| Images | Reviewed simple edit view plus broader semantic relationships | Replace promoted destination | Typed direct images with optional title | Same relationship/edit split as links |
| Reference definitions | Reviewed promoted definitions | Replace destination/title; conservative complete-line removal | Immediate/deferred definitions | Valid unpromoted definitions can still resolve relationships |
| Reference links/images | Semantic relationship read | No generic existing-source reference-link/image edit | Full prior/forward and normalized collapsed/shortcut construction | Existing-source relationship read does not imply mutation authority |
| Autolinks | Yes | Replace supported token | Angle and parser-proven bare/extended construction | Exact token must remain a proven autolink |
| Thematic breaks | Promoted top-level line | Remove complete owned line | Yes | Removal validates surviving Markdown |
| Blockquotes | Complete promoted top-level containers plus per-line content ranges | Remove complete promoted container | Single/multi-block reviewed forms, bounded nesting | Nested internal blockquotes do not receive separate top-level identities |
| GitHub alerts | Semantic overlay on promoted blockquotes | No alert-specific rewrite API | `NOTE`, `TIP`, `IMPORTANT`, `WARNING`, `CAUTION` | Reuses blockquote identity/ownership |
| Footnotes | Definitions, body ranges, references | Simple body replace; coordinated definition/reference rename | Immediate/deferred definitions and typed references | Exact case-sensitive contract; multiline bodies are broader read-only data |
| Mathematical expressions | Reviewed inline/block/fenced forms | Replace proven payload | Reviewed typed/block forms; fenced `math` via fenced code | Payload is opaque; no LaTeX/Math renderer |
| YAML/TOML front matter | Complete recognized envelope plus safe simple fields | Replace unique simple top-level scalar value | Conservative canonical string fields | No general YAML/TOML parser or serializer |
| HTML comments/anchors | Conservative promoted forms | Replace comment payload or quoted anchor value | No dedicated builder | Other HTML remains opaque source |

## HTML rendering

A parsed `Document` can be rendered explicitly without changing the source-preserving edit path.

| Capability | API | Boundary |
| --- | --- | --- |
| Streaming fragment output | `Document.RenderHTML` | Writes body fragments to caller `io.Writer`; stops on writer error |
| Buffered fragment output | `Document.HTML` | Returns caller-owned fragment bytes; convenient when whole-output buffering is acceptable |
| Streaming standalone output | `Document.RenderHTMLDocument` | Writes deterministic doctype/html/head/charset/body around the same fragment renderer; no template or asset system |
| Buffered standalone output | `Document.HTMLDocument` | Returns caller-owned complete-document bytes |
| Reviewed metadata mapping | `HTMLMetadataFrontMatter`, `HTMLMetadataOmit` | Exact lower-case `title`, `description`, `author`, `lang` only from unique top-level source-proven simple front-matter scalars; no general YAML/TOML interpretation |
| Raw HTML | `HTMLRawPreserve`, `HTMLRawEscape` | Preserve is not sanitization; escape explicitly for an HTML trust boundary |
| Dangerous URLs | `HTMLUnsafeURLSuppress`, `HTMLUnsafeURLAllow` | Default suppresses dangerous schemes by emitting an empty destination; no URL is fetched |
| GFM tag filter | `HTMLTagFilterEnabled`, `HTMLTagFilterDisabled` | Default applies the published GFM disallowed-tag filter to preserved raw HTML |
| Code blocks | Deterministic `<pre><code>` fragments | Language metadata may become a class; no syntax-highlighting engine runs |
| Footnotes, tasks, tables | Deterministic semantic HTML | Reuses the Native semantic walk; renderer does not reparse Markdown |
| Mathematical forms | Deterministic opaque wrappers | No LaTeX/MathJax/KaTeX interpretation or execution |

Front matter and reference-definition declarations are source/semantic metadata and are not emitted as visible fragment blocks. Rendering performs no filesystem discovery, asset fetch, network access, command execution, template execution, or embedded-language execution.

## Query and navigation

| Capability | Available | Boundary |
| --- | --- | --- |
| Bounded node queries | `QueryNodes` | Positive result limit required; source ordered; no persistent query index |
| Bounded section queries | `QuerySections` | Positive limit; optional level/range filters |
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

New-document construction is intentionally different: `DocumentBuilder` emits canonical LF GFM because there is no existing author formatting to preserve.

## What Marksplice deliberately does not do

Marksplice does not provide:

- PDF rendering;
- Markdown-to-Markdown whole-document formatting/normalization as the ordinary edit path;
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
