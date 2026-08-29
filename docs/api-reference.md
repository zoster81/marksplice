# Public API Reference

This is the exhaustive exported callable reference for `github.com/zoster81/marksplice`, with the separate filesystem adapter `github.com/zoster81/marksplice/workspacefs` documented in its own section below. Go source and pkg.go.dev remain canonical for signatures. If you are learning the library, start with [Getting Started](getting-started.md); if you already know the task, use the goal index below before dropping into the alphabetical type/method reference.

## Find the API by goal

### Read and inspect a document

Start with [`Parse`](#parse), then use [`Document` methods](#document-methods). Common typed receivers include [`Heading`](#heading-methods), [`ListItem`](#listitem-methods), [`Task`](#task-methods), [`Table`](#table-methods), [`TableRow`](#tablerow-methods), [`TableCell`](#tablecell-methods), [`FencedBlock`](#fencedblock-methods), [`FootnoteDefinition`](#footnotedefinition-methods), [`MathExpression`](#mathexpression-methods), and [`FrontMatter`](#frontmatter-methods).

Recipe: [Inspect a document](recipes/inspect-document.md).

### Edit an existing document

Existing-document mutation is prepared through the `Prepare...` methods under [`Document`](#document-methods), optionally combined with `ComposeChanges`, then applied through [`ChangeSet.Apply`](#changeset-methods).

Recipe: [Edit an existing document](recipes/edit-existing-document.md).

### Create a new document

Use [`NewDocumentBuilder`](#newdocumentbuilder), [`DocumentBuilder` methods](#documentbuilder-methods), and the package-level typed inline constructors in [Package functions](#package-functions).

Recipe: [Create a document](recipes/create-document.md).

### Render HTML and source maps

Use `Document.RenderHTML` / `Document.HTML` for deterministic fragments and `Document.RenderHTMLDocument` / `Document.HTMLDocument` for standalone output. When preview/editor tooling needs snapshot-local Markdown-byte to output-byte correlation, use the corresponding `...WithSourceMap` variants and inspect `HTMLSourceMapEntry` / `HTMLOutputRange`.

Recipe: [Render HTML](recipes/render-html.md).

### Lists, sections, and tables

Use [`Document` methods](#document-methods) for queries/navigation/mutations, with typed detail from [`ListItem`](#listitem-methods), [`Task`](#task-methods), [`Section`](#section-methods), [`Table`](#table-methods), [`TableRow`](#tablerow-methods), and [`TableCell`](#tablecell-methods).

Recipe: [Lists, sections, and tables](recipes/lists-sections-tables.md).

### Links, navigation, and workspaces

Single-document navigation/relationship methods live under [`Document`](#document-methods). Cross-document APIs start with [`BuildDocumentGraph`](#builddocumentgraph) and [`ValidateWorkspace`](#validateworkspace), with results on [`DocumentGraph`](#documentgraph-methods), [`GraphEdge`](#graphedge-methods), [`WorkspaceReport`](#workspacereport-methods), and [`WorkspaceDiagnostic`](#workspacediagnostic-methods). Filesystem discovery/following lives only in the separate [`workspacefs`](#workspacefs-package) package. Knowledge metadata starts with [`BuildKnowledgeIndex`](#buildknowledgeindex) and [`KnowledgeIndex`](#knowledgeindex-methods).

Recipe: [Links and workspaces](recipes/links-workspaces.md).

### Read-only extensions

Start with [`ParseWithOptions`](#parsewithoptions), `Extension`, `ParseOptions`, and `ExtensionLimits`; read retained observations through [`Document.ExtensionNodes`](#document-methods) and [`ExtensionNode`](#extensionnode-methods).

Recipe: [Read-only extensions](recipes/extensions.md).

## Error model

Public operations classify failure families with `errors.Is`. The root-package sentinels are `ErrNodeNotFound`, `ErrInvalidReplacement`, `ErrInvalidTargetKind`, `ErrSourceConflict`, `ErrInvalidConstruction`, `ErrInvalidQuery`, `ErrInvalidGraph`, `ErrInvalidWorkspace`, `ErrInvalidKnowledge`, `ErrInvalidExtension`, and `ErrInvalidRender`. The `workspacefs` package separately exports `ErrInvalidInput` and `ErrBudgetExceeded`. Diagnostic strings are not compatibility contracts.

## Type index (alphabetical)

- `Alert` — Alert is immutable semantic detail layered over one promoted top-level blockquote. Its ID is the underlying blockquote NodeID; alerts do not introduce a second identity namespace.
- `AlertKind` — AlertKind identifies one reviewed GitHub alert semantic kind.
- `AutoLink` — AutoLink is immutable typed detail for one promoted single-line GFM autolink.
- `Blockquote` — Blockquote is immutable typed detail for one promoted complete top-level blockquote container.
- `ChangeSet` — ChangeSet is an opaque prepared change bound to one exact source snapshot. Its zero value is unbound and Apply reports ErrSourceConflict.
- `CodeSpan` — CodeSpan is immutable typed detail for one promoted simple single-line code span.
- `Document` — Document is an immutable parsed Markdown source snapshot.
- `DocumentBuilder` — DocumentBuilder constructs a new GFM document independently from parsed source snapshots. It is mutable and is not safe for concurrent use without caller synchronization; its zero value is a valid empty builder.
- `DocumentGraph` — DocumentGraph is an immutable graph over an explicit caller-provided document set. It stores resolved edges plus compact adjacency indexes and performs no I/O.
- `DocumentKey` — DocumentKey is a caller-defined logical identity for one document in a graph. Marksplice treats it as opaque data and does not interpret it as a filesystem path or URL.
- `DocumentResolution` — DocumentResolution is a caller-authorized resolution of one non-local relationship to a document that is already present in the explicit graph input set. Fragment is optional and uses the same optional-leading-# syntax accepted by Document.ResolveFragment.
- `DocumentResolver` — DocumentResolver resolves one non-local relationship against the caller's own authorization/domain model. Returning false leaves the relationship outside the graph. Marksplice invokes the resolver synchronously and never concurrently during one build, never retains it, and never performs filesystem or network access.
- `Emphasis` — Emphasis is immutable typed detail for one promoted simple emphasis span.
- `Extension` — Extension registers one explicitly opted-in third-party recognizer under one namespace. Registration does not grant Marksplice filesystem, network, command, mutation, or construction authority. Recognizers are ordinary statically linked caller code: Marksplice validates their returned observations but cannot sandbox or preempt their own CPU, memory, goroutine, filesystem, network, or command behavior.
- `ExtensionAttribute` — ExtensionAttribute is one extension-defined immutable scalar metadata entry. Attribute names must be non-empty tokens; values must be valid UTF-8 without NUL.
- `ExtensionID` — ExtensionID is the caller-defined namespace of one explicitly registered third-party syntax/semantic extension. It is separate from the closed core Kind namespace.
- `ExtensionKind` — ExtensionKind is one extension-local semantic kind name.
- `ExtensionLimits` — ExtensionLimits bounds extension observations retained by one ParseWithOptions call. Both limits must be positive when at least one extension is registered. MaxNodes is the total retained node count across all extensions. MaxMetadataBytes bounds the total bytes retained for each node's extension ID, kind, attribute names, and attribute values; it does not attempt to sandbox allocations performed inside third-party recognizers.
- `ExtensionMatch` — ExtensionMatch is one source-owned observation returned by a third-party recognizer. Range must be a non-empty byte range within the exact parsed source snapshot.
- `ExtensionNode` — ExtensionNode is one immutable validated third-party source observation attached to a Document snapshot. It never replaces or reclassifies a core Node.
- `ExtensionRecognizer` — ExtensionRecognizer observes extension-specific syntax or semantics over one exact source snapshot. Marksplice invokes recognizers synchronously and serially during ParseWithOptions and never retains the callback after the call returns. Returned slices must not be mutated concurrently after return.
- `ExtensionSource` — ExtensionSource is the immutable source view supplied to one extension recognizer. Text returns the exact parsed source bytes represented as a Go string.
- `FencedBlock` — FencedBlock is immutable read-only detail for one source-proven top-level GFM fenced block. It owns the complete physical container independently from the narrower historical FencedCode replacement capability.
- `FencedCode` — FencedCode is immutable typed detail for one fenced code block whose payload is proven to be one exact contiguous source span suitable for the historical source-preserving replacement API. Use Document.FencedBlocks for broader read-only fenced-container ownership.
- `FootnoteDefinition` — FootnoteDefinition is immutable typed detail for one source-proven top-level footnote definition. Range owns the complete physical definition container; BodyRange is available only for the conservative simple editable subset.
- `FootnoteReference` — FootnoteReference is one immutable parser-proven footnote reference occurrence. It is relationship data and does not grant generic mutation authority.
- `FragmentTarget` — FragmentTarget is one uniquely resolved fragment destination in this snapshot.
- `FragmentTargetKind` — FragmentTargetKind identifies one supported intra-document fragment target kind.
- `FrontMatter` — FrontMatter is immutable source ownership for one recognized document-leading metadata envelope. It is document-envelope state rather than a structural Markdown node.
- `FrontMatterField` — FrontMatterField is immutable typed detail for one promoted simple leading YAML/TOML scalar field.
- `FrontMatterFieldInput` — FrontMatterFieldInput is construction-only input for one canonical simple YAML/TOML string scalar.
- `FrontMatterFormat` — FrontMatterFormat identifies the source format of a recognized front-matter envelope or promoted field.
- `GraphDocument` — GraphDocument binds one caller-defined logical key to an immutable parsed document.
- `GraphEdge` — GraphEdge is one immutable resolved relationship between two caller-provided documents.
- `HTMLAnchor` — HTMLAnchor is immutable typed detail for one promoted simple quoted id/name attribute on an <a> tag.
- `HTMLAnchorAttribute` — HTMLAnchorAttribute identifies the semantic anchor attribute targeted by an HTMLAnchor.
- `HTMLComment` — HTMLComment is immutable typed detail for one promoted single-line HTML comment payload.
- `HTMLDocumentOptions` — HTMLDocumentOptions controls deterministic standalone HTML rendering; its zero value reuses fragment defaults and reviewed front-matter metadata mapping.
- `HTMLMetadataPolicy` — HTMLMetadataPolicy controls reviewed front-matter metadata mapping versus explicit omission.
- `HTMLOutputRange` — HTMLOutputRange is a half-open byte range in one exact emitted HTML result; its coordinate space is output bytes rather than Markdown source bytes.
- `HTMLRawPolicy` — HTMLRawPolicy controls preservation versus escaping of parser-proven raw HTML during fragment rendering.
- `HTMLRenderOptions` — HTMLRenderOptions controls deterministic HTML-fragment rendering. Its zero value preserves raw HTML, enables the GFM tag filter, and suppresses dangerous URL schemes.
- `HTMLSourceMapEntry` — HTMLSourceMapEntry correlates one snapshot-local Markdown source byte range with one contiguous byte range in the exact HTML output produced by the same successful render.
- `HTMLTagFilterPolicy` — HTMLTagFilterPolicy controls the GFM disallowed-raw-HTML tag filter.
- `HTMLUnsafeURLPolicy` — HTMLUnsafeURLPolicy controls suppression versus explicit allowance of dangerous URL schemes.
- `Heading` — Heading is immutable typed detail for one promoted top-level heading.
- `HeadingAnchor` — HeadingAnchor is an immutable GitHub-compatible anchor derived from one heading.
- `HeadingStyle` — HeadingStyle identifies the source syntax of a promoted heading.
- `Image` — Image is immutable typed detail for one promoted simple inline image.
- `Inline` — Inline is a construction-only typed inline value.
- `InlineLink` — InlineLink is immutable typed detail for one promoted simple inline link.
- `Kind` — Kind identifies a Marksplice core structural Markdown node category. Third-party extension identities are intentionally outside this core enum.
- `KnowledgeAlias` — KnowledgeAlias is one exact, syntax-independent alternate name for a document. Marksplice does not normalize, parse, or derive aliases from Markdown or metadata source.
- `KnowledgeDocument` — KnowledgeDocument supplies caller-owned semantic metadata for one document already present in the explicit DocumentGraph. Graph documents omitted from the input simply have no knowledge metadata; this value never changes the underlying document snapshot.
- `KnowledgeIndex` — KnowledgeIndex is an immutable syntax-independent semantic overlay on one DocumentGraph. It retains no parser, resolver callback, filesystem/network authority, or source mutation capability.
- `KnowledgeReference` — KnowledgeReference is one immutable caller-declared logical document relationship. It has no source offset because the knowledge layer does not infer Markdown or metadata syntax.
- `KnowledgeTag` — KnowledgeTag is one exact, syntax-independent classification value for a document. Matching is byte-exact and case-sensitive; Marksplice performs no normalization.
- `LinkFragmentStatus` — LinkFragmentStatus reports whether a relationship destination is an intra-document fragment and, when applicable, how it resolves in this snapshot.
- `LinkRelationship` — LinkRelationship is an immutable semantic outgoing link/image/autolink fact. It does not define generic source ownership or mutation authority.
- `LinkRelationshipKind` — LinkRelationshipKind identifies one semantic outgoing link/image relationship.
- `ListItem` — ListItem is immutable typed detail for one promoted single-line list item.
- `ListItemInput` — ListItemInput is construction-only structured input for one item in a nested list.
- `ManagedTOC` — ManagedTOC identifies one caller-designated section whose body is expected to use the conservative managed TOC shape recognized by TOCStale and PrepareSyncTOC.
- `MathExpression` — MathExpression is immutable typed detail for one reviewed mathematical source form. Mathematical payload is opaque data; Marksplice does not parse or render LaTeX.
- `MathExpressionStyle` — MathExpressionStyle identifies one reviewed GitHub-compatible mathematical source form.
- `Node` — Node is an immutable public summary of one promoted structural node.
- `NodeID` — NodeID identifies a node within one parsed source snapshot.
- `NodeMatch` — NodeMatch is one immutable structural query result.
- `NodeQuery` — NodeQuery selects promoted structural nodes from one immutable document snapshot.
- `Paragraph` — Paragraph is immutable typed detail for one promoted top-level paragraph.
- `ParseOptions` — ParseOptions configures optional third-party semantic/source overlays. Zero options are exactly equivalent to Parse.
- `Range` — Range is a half-open byte range [Start, End) in a source snapshot. The accessor returning a Range defines the semantic meaning of that span.
- `ReferenceDefinition` — ReferenceDefinition is immutable typed detail for one promoted single-line reference definition.
- `ReferenceForm` — ReferenceForm identifies one GFM reference-link/image source form.
- `Section` — Section is an immutable source-bound view governed by one promoted document heading.
- `SectionQuery` — SectionQuery selects derived document sections from one immutable snapshot.
- `Strikethrough` — Strikethrough is immutable typed detail for one promoted simple GFM strikethrough.
- `Strong` — Strong is immutable typed detail for one promoted simple strong-emphasis span.
- `Table` — Table is immutable typed detail for one promoted GFM table.
- `TableAlignment` — TableAlignment identifies the semantic alignment of a GFM table column. It is used for new-document construction and read-only parsed table alignment access.
- `TableCell` — TableCell is immutable typed detail for one promoted non-empty GFM table cell.
- `TableRow` — TableRow is immutable typed detail for one promoted GFM table body row.
- `Task` — Task is immutable typed detail for one promoted GFM task marker.
- `TaskListItem` — TaskListItem is structured input for one newly constructed GFM task-list item. InlineGFM is caller-provided inline GFM source; Checked selects the canonical '[x]' or '[ ]' task marker written before that content.
- `TaskListItemInput` — TaskListItemInput is construction-only structured input for one item in a nested task list. Depth follows the same structural contract as ListItemInput.
- `ThematicBreak` — ThematicBreak is immutable typed detail for one promoted top-level thematic break.
- `UnresolvedReference` — UnresolvedReference is immutable semantic metadata for one conservative explicit full/collapsed reference whose parser context contains no matching definition.
- `WorkspaceDiagnostic` — WorkspaceDiagnostic is one immutable workspace validation finding. Accessors expose only metadata meaningful for the diagnostic kind; absent metadata returns false.
- `WorkspaceDiagnosticKind` — WorkspaceDiagnosticKind identifies one deterministic workspace validation finding.
- `WorkspaceRepair` — WorkspaceRepair is one deterministic safe repair prepared through the ordinary snapshot-bound mutation machinery.
- `WorkspaceRepairPlan` — WorkspaceRepairPlan is an immutable ordered set of provably safe repairs.
- `WorkspaceReport` — WorkspaceReport combines the resolved DocumentGraph, deterministic diagnostics, and conservative repair plan for one explicit validation run.
- `WorkspaceResolution` — WorkspaceResolution is one caller-authorized classification of a non-local relationship. Target identities are opaque caller data; Marksplice performs no I/O.
- `WorkspaceResolutionKind` — WorkspaceResolutionKind describes how the caller classifies one non-local relationship while validating an explicit workspace document set.
- `WorkspaceResolver` — WorkspaceResolver classifies one non-local link relationship for workspace validation. It is invoked synchronously, never concurrently during one ValidateWorkspace call, and is never retained.
- `WorkspaceValidationOptions` — WorkspaceValidationOptions supplies explicit validation authority beyond the document set itself. Empty Roots disables orphan/reachability diagnostics.

## Functions and methods

### Package functions

#### `AutoLinkInline`

```go
func AutoLinkInline(value string) Inline
```

AutoLinkInline returns one canonical angle-autolink construction value.
Validation succeeds only when reparsing produces the existing source-proven AutoLink capability.

#### `BareAutoLinkInline`

```go
func BareAutoLinkInline(value string) Inline
```

BareAutoLinkInline returns one parser-proven GFM extended autolink token
without adding angle brackets. The complete requested token must be owned by
one AutoLink observation after reparsing or construction fails closed.

#### `BuildDocumentGraph`

```go
func BuildDocumentGraph(documents []GraphDocument, resolver DocumentResolver) (*DocumentGraph, error)
```

BuildDocumentGraph builds a deterministic graph over documents already supplied by
the caller. Local #fragment relationships resolve to their source document without
invoking resolver. Every other relationship is included only when resolver explicitly
maps it to a document key from the supplied set.

#### `BuildKnowledgeIndex`

```go
func BuildKnowledgeIndex(graph *DocumentGraph, documents []KnowledgeDocument) (*KnowledgeIndex, error)
```

BuildKnowledgeIndex builds syntax-independent aliases, tags, and logical references
over an already-authorized document graph. Metadata may be supplied for any subset of
graph documents. Every logical reference target must already belong to the graph;
aliases never resolve or discover additional documents.

#### `CodeInline`

```go
func CodeInline(code string) Inline
```

CodeInline returns one conservative single-line code span construction value.

The writer selects an adaptive backtick delimiter longer than every internal run.
Leading/trailing horizontal space and leading/trailing backticks are rejected
because supporting those shapes would require semantic whitespace or delimiter
normalization beyond the existing source-proven parsed CodeSpan capability.

#### `CollapsedReferenceImageInline`

```go
func CollapsedReferenceImageInline(alt ...Inline) Inline
```

CollapsedReferenceImageInline returns one `![alt][]` construction value.

#### `CollapsedReferenceLinkInline`

```go
func CollapsedReferenceLinkInline(label ...Inline) Inline
```

CollapsedReferenceLinkInline returns one `[label][]` construction value. The
emitted label must resolve to exactly one available normalized definition.

#### `DefaultHTMLDocumentOptions`

```go
func DefaultHTMLDocumentOptions() HTMLDocumentOptions
```

DefaultHTMLDocumentOptions returns the zero-value standalone policy: use the ordinary fragment-rendering defaults and map the reviewed front-matter metadata set when safely available.

#### `DefaultHTMLRenderOptions`

```go
func DefaultHTMLRenderOptions() HTMLRenderOptions
```

DefaultHTMLRenderOptions returns the documented zero-value HTML-fragment policy: preserve parser-proven raw HTML, apply the published GFM tag filter, and suppress dangerous URL schemes.

#### `EmphasisInline`

```go
func EmphasisInline(content ...Inline) Inline
```

EmphasisInline returns one conservative emphasis construction value.
It permits bounded nesting of code, emphasis, strong, and strikethrough children
while keeping links/images/autolinks outside this wrapper slice.

#### `FootnoteReferenceInline`

```go
func FootnoteReferenceInline(label string) Inline
```

FootnoteReferenceInline returns one typed `[^label]` reference. The label must
resolve to exactly one already-appended or explicitly deferred footnote definition
in the destination DocumentBuilder when the value is appended.

#### `ForwardReferenceImageInline`

```go
func ForwardReferenceImageInline(reference string, alt ...Inline) Inline
```

ForwardReferenceImageInline returns one full reference-image construction
value resolved only against one explicitly deferred top-level definition.

#### `ForwardReferenceLinkInline`

```go
func ForwardReferenceLinkInline(reference string, label ...Inline) Inline
```

ForwardReferenceLinkInline returns one full reference-link construction value
resolved only against one explicitly deferred top-level definition.

#### `ImageInline`

```go
func ImageInline(destination string, alt ...Inline) Inline
```

ImageInline returns one conservative inline-image construction value.

The destination is written in angle brackets and alt content may contain the reviewed
bounded structured-inline children; use ImageInlineWithTitle for a title.

#### `ImageInlineWithTitle`

```go
func ImageInlineWithTitle(destination, title string, alt ...Inline) Inline
```

ImageInlineWithTitle returns one conservative inline-image construction value
with a canonical double-quoted title. It applies the same conservative title policy
as LinkInlineWithTitle.

#### `LinkInline`

```go
func LinkInline(destination string, label ...Inline) Inline
```

LinkInline returns one conservative inline-link construction value.

The destination is written in angle brackets and labels may contain the reviewed
bounded structured-inline children; use LinkInlineWithTitle for a title.

#### `LinkInlineWithTitle`

```go
func LinkInlineWithTitle(destination, title string, label ...Inline) Inline
```

LinkInlineWithTitle returns one conservative inline-link construction value
with a canonical double-quoted title. The title must be non-empty and require no
GFM escape or entity interpretation.

#### `MathBacktickInline`

```go
func MathBacktickInline(payload string) Inline
```

MathBacktickInline returns one conservative GitHub-compatible `$`-backtick
construction value for payload that would otherwise overlap Markdown syntax.

#### `MathInline`

```go
func MathInline(payload string) Inline
```

MathInline returns one conservative GitHub-compatible `$...$` construction value.
Mathematical payload remains opaque and must fit on one physical line.

#### `NewDocumentBuilder`

```go
func NewDocumentBuilder() *DocumentBuilder
```

NewDocumentBuilder returns an empty new-document builder.

#### `Parse`

```go
func Parse(source []byte) (*Document, error)
```

Parse copies and parses source into an immutable document snapshot.

#### `ParseWithOptions`

```go
func ParseWithOptions(source []byte, options ParseOptions) (*Document, error)
```

ParseWithOptions copies and parses source using the ordinary Marksplice GFM core, then
optionally evaluates explicitly registered third-party read-only overlays. Extension
observations never alter core nodes, parser behavior, mutation authority, or construction.

#### `ReferenceImageInline`

```go
func ReferenceImageInline(reference string, alt ...Inline) Inline
```

ReferenceImageInline returns one conservative full reference-image construction value.
It follows the same exact-definition and structured-alt requirements as ReferenceLinkInline.

#### `ReferenceLinkInline`

```go
func ReferenceLinkInline(reference string, label ...Inline) Inline
```

ReferenceLinkInline returns one conservative full reference-link construction value.
The exact reference label must identify one already-appended top-level reference
definition in the destination DocumentBuilder when the value is appended. It permits
the same reviewed bounded structured-inline label children as direct links.

#### `ShortcutReferenceImageInline`

```go
func ShortcutReferenceImageInline(alt ...Inline) Inline
```

ShortcutReferenceImageInline returns one `![alt]` construction value.

#### `ShortcutReferenceLinkInline`

```go
func ShortcutReferenceLinkInline(label ...Inline) Inline
```

ShortcutReferenceLinkInline returns one `[label]` construction value. The
emitted label must resolve to exactly one available normalized definition.

#### `StrikethroughInline`

```go
func StrikethroughInline(content ...Inline) Inline
```

StrikethroughInline returns one conservative GFM strikethrough construction value.
It permits bounded code/emphasis/strong children but rejects direct
strikethrough-in-strikethrough nesting because adjacent tilde runs are ambiguous.

#### `StrongInline`

```go
func StrongInline(content ...Inline) Inline
```

StrongInline returns one conservative strong-emphasis construction value.
It applies the same bounded structured-child policy as EmphasisInline.

#### `TextInline`

```go
func TextInline(text string) Inline
```

TextInline returns semantic plain text for typed inline construction.

ASCII punctuation is encoded with canonical GFM backslash escapes so caller text
cannot become Markdown syntax implicitly. Validation occurs when the
value is appended to a DocumentBuilder.

#### `ValidateWorkspace`

```go
func ValidateWorkspace(documents []GraphDocument, resolver WorkspaceResolver, options WorkspaceValidationOptions) (*WorkspaceReport, error)
```

ValidateWorkspace validates relationships and explicitly managed generated indexes
over a finite caller-provided document set. Marksplice performs no filesystem or
network discovery and never retains resolver or validation authority callbacks.

### `Alert` methods

#### `ID`

```go
func (a Alert) ID() NodeID
```

ID returns the underlying blockquote's snapshot-scoped identity.

#### `Kind`

```go
func (a Alert) Kind() AlertKind
```

Kind returns the exact reviewed GitHub alert kind.

#### `MarkerRange`

```go
func (a Alert) MarkerRange() Range
```

MarkerRange returns the exact inner-source range containing the alert marker such as [!NOTE].

#### `Range`

```go
func (a Alert) Range() Range
```

Range returns the exact complete physical source owned by the underlying top-level blockquote.

### `AutoLink` methods

#### `ID`

```go
func (a AutoLink) ID() NodeID
```

ID returns the autolink's snapshot-scoped node identity.

#### `IsEmail`

```go
func (a AutoLink) IsEmail() bool
```

IsEmail reports whether the parser classified this as an email autolink.

#### `Range`

```go
func (a AutoLink) Range() Range
```

Range returns the exact autolink token content replaced by PrepareReplaceAutoLink.
Angle brackets, when present, and surrounding source are outside this range.

#### `Value`

```go
func (a AutoLink) Value() string
```

Value returns the parser-proven semantic autolink value.

### `Blockquote` methods

#### `ContentRange`

```go
func (b Blockquote) ContentRange() Range
```

ContentRange returns the historical single-line inner source span when the
promoted blockquote owns exactly one physical content segment. It returns the
zero Range for segmented multiline, lazy-continuation, or multi-block source;
use Document.BlockquoteContentRanges for those containers.

#### `ID`

```go
func (b Blockquote) ID() NodeID
```

ID returns the blockquote's snapshot-scoped node identity.

#### `Range`

```go
func (b Blockquote) Range() Range
```

Range returns the exact complete physical source owned by the top-level blockquote container.
Every owned physical line terminator is included when present.

### `ChangeSet` methods

#### `Apply`

```go
func (c ChangeSet) Apply(source []byte) ([]byte, error)
```

Apply applies the prepared change only when source matches its original snapshot.

### `CodeSpan` methods

#### `ID`

```go
func (c CodeSpan) ID() NodeID
```

ID returns the code span's snapshot-scoped node identity.

#### `Range`

```go
func (c CodeSpan) Range() Range
```

Range returns the exact code-span content span replaced by PrepareReplaceCodeSpan.

### `Document` methods

#### `Alert`

```go
func (d *Document) Alert(id NodeID) (Alert, bool)
```

Alert returns semantic alert detail when id identifies a promoted top-level blockquote
whose first inner physical line is one exact reviewed GitHub alert marker and whose
remaining owned source contains at least one non-empty body segment.

#### `AlertBodyRanges`

```go
func (d *Document) AlertBodyRanges(id NodeID) ([]Range, bool)
```

AlertBodyRanges returns caller-owned inner source segments after the alert marker line.
Marker-only blank lines are represented by valid empty ranges and lazy continuation
lines retain their source-proven blockquote inner ranges.

#### `Alerts`

```go
func (d *Document) Alerts() []Alert
```

Alerts returns all recognized top-level GitHub alerts in source order.
The returned slice is caller-owned. Recognition adds no persistent semantic index.

#### `AutoLink`

```go
func (d *Document) AutoLink(id NodeID) (AutoLink, bool)
```

AutoLink returns typed detail for one promoted single-line GFM autolink.

#### `Blockquote`

```go
func (d *Document) Blockquote(id NodeID) (Blockquote, bool)
```

Blockquote returns typed detail for one promoted complete top-level blockquote container.

#### `BlockquoteContentRanges`

```go
func (d *Document) BlockquoteContentRanges(id NodeID) ([]Range, bool)
```

BlockquoteContentRanges returns caller-owned inner source segments for every
physical line owned by one promoted top-level blockquote, in source order.
Marker-only lines are represented by valid empty ranges. Lazy continuation
lines have no synthetic marker removal: their complete physical content is returned.

#### `CodeSpan`

```go
func (d *Document) CodeSpan(id NodeID) (CodeSpan, bool)
```

CodeSpan returns typed detail for one promoted simple single-line code span.

#### `ComposeChanges`

```go
func (d *Document) ComposeChanges(changes ...ChangeSet) (ChangeSet, error)
```

ComposeChanges combines already-prepared mutations from this exact document
snapshot into one atomic source-bound change. Overlapping or semantically
interacting prepared mutations fail closed.

#### `Emphasis`

```go
func (d *Document) Emphasis(id NodeID) (Emphasis, bool)
```

Emphasis returns typed detail for one promoted simple emphasis span.

#### `ExtensionNodes`

```go
func (d *Document) ExtensionNodes() []ExtensionNode
```

ExtensionNodes returns caller-owned immutable extension observations in registration
order and each recognizer's returned order. Core structural nodes remain separate.

#### `FencedBlock`

```go
func (d *Document) FencedBlock(id NodeID) (FencedBlock, bool)
```

FencedBlock returns one source-proven top-level fenced block by snapshot ID.

#### `FencedBlockContentRanges`

```go
func (d *Document) FencedBlockContentRanges(id NodeID) ([]Range, bool)
```

FencedBlockContentRanges returns caller-owned source-backed payload ranges, one
per parser-proven physical body line. Empty payloads return an empty slice with
ok=true. These ranges are read-only source ownership, not generic mutation spans.

#### `FencedBlocks`

```go
func (d *Document) FencedBlocks() []FencedBlock
```

FencedBlocks returns every source-proven top-level fenced block in source order.
Readability is broader than the historical FencedCode edit capability: empty,
non-contiguous, or unclosed blocks may be returned without gaining mutation authority.

#### `FencedCode`

```go
func (d *Document) FencedCode(id NodeID) (FencedCode, bool)
```

FencedCode returns typed detail for one promoted supported fenced code block.

#### `FootnoteDefinition`

```go
func (d *Document) FootnoteDefinition(id NodeID) (FootnoteDefinition, bool)
```

FootnoteDefinition returns one source-proven top-level definition by snapshot ID.

#### `FootnoteDefinitionBodyRanges`

```go
func (d *Document) FootnoteDefinitionBodyRanges(id NodeID) ([]Range, bool)
```

FootnoteDefinitionBodyRanges returns caller-owned parser-proven body segments
in physical source order. They are read-only source metadata, not generic edit spans.

#### `FootnoteDefinitions`

```go
func (d *Document) FootnoteDefinitions() []FootnoteDefinition
```

FootnoteDefinitions returns every source-proven top-level footnote definition
in physical source order, including unused definitions.

#### `FootnoteReferences`

```go
func (d *Document) FootnoteReferences() []FootnoteReference
```

FootnoteReferences returns every parser-proven footnote reference in source order.
The returned slice is caller-owned and no relationship index is retained.

#### `FrontMatter`

```go
func (d *Document) FrontMatter() (FrontMatter, bool)
```

FrontMatter returns the recognized document-leading YAML/TOML metadata envelope.
Complex or duplicate metadata can be readable through this envelope even when no
individual field is safe to promote for source-preserving mutation.

#### `FrontMatterField`

```go
func (d *Document) FrontMatterField(id NodeID) (FrontMatterField, bool)
```

FrontMatterField returns typed detail for one promoted simple leading YAML/TOML scalar field.

#### `GenerateTOC`

```go
func (d *Document) GenerateTOC() []byte
```

GenerateTOC returns deterministic Markdown for the current section hierarchy.
Generated output uses LF line endings; source synchronization preserves the target body's line-ending style.

#### `HTMLAnchor`

```go
func (d *Document) HTMLAnchor(id NodeID) (HTMLAnchor, bool)
```

HTMLAnchor returns typed detail for one promoted simple quoted id/name attribute on an <a> tag.

#### `HTMLComment`

```go
func (d *Document) HTMLComment(id NodeID) (HTMLComment, bool)
```

HTMLComment returns typed detail for one promoted single-line HTML comment.

#### `Heading`

```go
func (d *Document) Heading(id NodeID) (Heading, bool)
```

Heading returns typed detail for one promoted top-level heading.

#### `HeadingAnchor`

```go
func (d *Document) HeadingAnchor(id NodeID) (HeadingAnchor, bool)
```

HeadingAnchor returns the derived anchor for one promoted heading.

#### `HeadingAnchors`

```go
func (d *Document) HeadingAnchors() []HeadingAnchor
```

HeadingAnchors derives all promoted heading anchors in source order.
Duplicate disambiguation is recomputed from the immutable snapshot on each call.

#### `HTML`

```go
func (d *Document) HTML(options HTMLRenderOptions) ([]byte, error)
```

HTML renders a deterministic HTML fragment into caller-owned bytes. It is the buffered convenience form of RenderHTML and returns `ErrInvalidRender` for an invalid receiver or options.

#### `HTMLDocument`

```go
func (d *Document) HTMLDocument(options HTMLDocumentOptions) ([]byte, error)
```

HTMLDocument renders a deterministic standalone HTML document into caller-owned bytes. It wraps the exact fragment renderer with doctype/html/head/charset/body markup and the selected reviewed metadata policy.

#### `HTMLDocumentWithSourceMap`

```go
func (d *Document) HTMLDocumentWithSourceMap(options HTMLDocumentOptions) ([]byte, []HTMLSourceMapEntry, error)
```

HTMLDocumentWithSourceMap renders a deterministic standalone HTML document into caller-owned bytes and returns snapshot-local Markdown-to-output correlations for that exact successful result. Output offsets are absolute from byte zero of the complete document; synthetic wrapper bytes can remain unmapped.

#### `HTMLWithSourceMap`

```go
func (d *Document) HTMLWithSourceMap(options HTMLRenderOptions) ([]byte, []HTMLSourceMapEntry, error)
```

HTMLWithSourceMap renders a deterministic HTML fragment into caller-owned bytes and returns the source-map entries for that exact successful output. Entries are semantic-event granular and may overlap when semantics nest.

#### `Image`

```go
func (d *Document) Image(id NodeID) (Image, bool)
```

Image returns typed detail for one promoted simple inline image.

#### `InlineLink`

```go
func (d *Document) InlineLink(id NodeID) (InlineLink, bool)
```

InlineLink returns typed detail for one promoted simple inline link.

#### `LinkRelationships`

```go
func (d *Document) LinkRelationships() []LinkRelationship
```

LinkRelationships returns all parser-resolved outgoing link/image/autolink
relationships in source order. The returned slice is caller-owned and does not
persist any relationship index or graph in the snapshot.

#### `ListItem`

```go
func (d *Document) ListItem(id NodeID) (ListItem, bool)
```

ListItem returns typed detail for one promoted single-line list item.

#### `MathExpression`

```go
func (d *Document) MathExpression(id NodeID) (MathExpression, bool)
```

MathExpression returns one reviewed mathematical expression by snapshot ID.

#### `MathExpressionPayloadRanges`

```go
func (d *Document) MathExpressionPayloadRanges(id NodeID) ([]Range, bool)
```

MathExpressionPayloadRanges returns caller-owned source-backed payload ranges.
Fenced math may expose zero, one, or multiple physical payload ranges.

#### `MathExpressions`

```go
func (d *Document) MathExpressions() []MathExpression
```

MathExpressions returns reviewed mathematical expressions in source order.
Exact-info `math` fenced blocks reuse their existing FencedBlock identity rather
than creating a second structural node.

#### `Node`

```go
func (d *Document) Node(id NodeID) (Node, bool)
```

Node returns one node summary by snapshot-local ID.

#### `Nodes`

```go
func (d *Document) Nodes() []Node
```

Nodes returns summaries for node kinds promoted into the public API.

#### `Paragraph`

```go
func (d *Document) Paragraph(id NodeID) (Paragraph, bool)
```

See the public Go signature and the task-oriented guide for usage constraints.

#### `PrepareAppendListItemChild`

```go
func (d *Document) PrepareAppendListItemChild(parentID NodeID, fragment []byte) (ChangeSet, error)
```

PrepareAppendListItemChild prepares appending one complete direct-child subtree to a fully supported list-item subtree.

#### `PrepareAppendSectionChild`

```go
func (d *Document) PrepareAppendSectionChild(parentHeadingID NodeID, fragment []byte) (ChangeSet, error)
```

PrepareAppendSectionChild prepares appending one direct child section subtree to a promoted parent section.

#### `PrepareAppendTableRow`

```go
func (d *Document) PrepareAppendTableRow(id NodeID, fragment []byte) (ChangeSet, error)
```

PrepareAppendTableRow prepares appending one caller-owned compatible body row to a promoted GFM table.

#### `PrepareInsertListItemAfter`

```go
func (d *Document) PrepareInsertListItemAfter(anchorID NodeID, fragment []byte) (ChangeSet, error)
```

PrepareInsertListItemAfter prepares insertion of one complete same-shape supported list-item subtree immediately after a complete supported anchor subtree.

#### `PrepareInsertListItemBefore`

```go
func (d *Document) PrepareInsertListItemBefore(anchorID NodeID, fragment []byte) (ChangeSet, error)
```

PrepareInsertListItemBefore prepares insertion of one complete same-shape supported list-item subtree immediately before a complete supported anchor subtree.

#### `PrepareInsertSectionAfter`

```go
func (d *Document) PrepareInsertSectionAfter(headingID NodeID, fragment []byte) (ChangeSet, error)
```

PrepareInsertSectionAfter prepares insertion of one sibling section subtree immediately after the target section subtree.

#### `PrepareInsertSectionBefore`

```go
func (d *Document) PrepareInsertSectionBefore(headingID NodeID, fragment []byte) (ChangeSet, error)
```

PrepareInsertSectionBefore prepares insertion of one sibling section subtree immediately before the target section.

#### `PrepareInsertTableColumn`

```go
func (d *Document) PrepareInsertTableColumn(id NodeID, column int, header []byte, alignment TableAlignment, body [][]byte) (ChangeSet, error)
```

PrepareInsertTableColumn prepares source-preserving insertion of one complete promoted GFM table column.

#### `PrepareInsertTableRowAfter`

```go
func (d *Document) PrepareInsertTableRowAfter(anchorID NodeID, fragment []byte) (ChangeSet, error)
```

PrepareInsertTableRowAfter prepares insertion of one complete compatible body row after a promoted row.

#### `PrepareInsertTableRowBefore`

```go
func (d *Document) PrepareInsertTableRowBefore(anchorID NodeID, fragment []byte) (ChangeSet, error)
```

PrepareInsertTableRowBefore prepares insertion of one complete compatible body row before a promoted row.

#### `PrepareMoveListItemAfter`

```go
func (d *Document) PrepareMoveListItemAfter(id, anchorID NodeID) (ChangeSet, error)
```

PrepareMoveListItemAfter prepares moving one complete supported list-item subtree immediately after a complete same-shape anchor subtree.

#### `PrepareMoveListItemBefore`

```go
func (d *Document) PrepareMoveListItemBefore(id, anchorID NodeID) (ChangeSet, error)
```

PrepareMoveListItemBefore prepares moving one complete supported list-item subtree immediately before a complete same-shape anchor subtree.

#### `PrepareMoveSectionAfter`

```go
func (d *Document) PrepareMoveSectionAfter(headingID, anchorHeadingID NodeID) (ChangeSet, error)
```

PrepareMoveSectionAfter prepares moving one complete promoted section subtree immediately after a same-level anchor subtree.

#### `PrepareMoveSectionBefore`

```go
func (d *Document) PrepareMoveSectionBefore(headingID, anchorHeadingID NodeID) (ChangeSet, error)
```

PrepareMoveSectionBefore prepares moving one complete promoted section subtree immediately before a same-level anchor section.

#### `PrepareMoveTableColumn`

```go
func (d *Document) PrepareMoveTableColumn(id NodeID, from, to int) (ChangeSet, error)
```

PrepareMoveTableColumn prepares moving one complete promoted GFM table column to a new zero-based position.

#### `PrepareMoveTableRowAfter`

```go
func (d *Document) PrepareMoveTableRowAfter(id, anchorID NodeID) (ChangeSet, error)
```

PrepareMoveTableRowAfter prepares moving one complete body row after another promoted row in the same table.

#### `PrepareMoveTableRowBefore`

```go
func (d *Document) PrepareMoveTableRowBefore(id, anchorID NodeID) (ChangeSet, error)
```

PrepareMoveTableRowBefore prepares moving one complete body row before another promoted row in the same table.

#### `PrepareRemoveBlockquote`

```go
func (d *Document) PrepareRemoveBlockquote(id NodeID) (ChangeSet, error)
```

PrepareRemoveBlockquote prepares source-preserving removal of one complete promoted top-level blockquote container.

#### `PrepareRemoveListItem`

```go
func (d *Document) PrepareRemoveListItem(id NodeID) (ChangeSet, error)
```

PrepareRemoveListItem prepares removal of one complete supported list-item subtree.

#### `PrepareRemoveReferenceDefinition`

```go
func (d *Document) PrepareRemoveReferenceDefinition(id NodeID) (ChangeSet, error)
```

PrepareRemoveReferenceDefinition prepares source-preserving removal of one complete promoted single-line reference-definition line.

#### `PrepareRemoveSection`

```go
func (d *Document) PrepareRemoveSection(headingID NodeID) (ChangeSet, error)
```

PrepareRemoveSection prepares source-preserving removal of one complete promoted section subtree.

#### `PrepareRemoveTableColumn`

```go
func (d *Document) PrepareRemoveTableColumn(id NodeID, column int) (ChangeSet, error)
```

PrepareRemoveTableColumn prepares source-preserving removal of one complete promoted GFM table column.

#### `PrepareRemoveTableRow`

```go
func (d *Document) PrepareRemoveTableRow(id NodeID) (ChangeSet, error)
```

PrepareRemoveTableRow prepares source-preserving removal of one promoted GFM table body row.

#### `PrepareRemoveThematicBreak`

```go
func (d *Document) PrepareRemoveThematicBreak(id NodeID) (ChangeSet, error)
```

PrepareRemoveThematicBreak prepares source-preserving removal of one complete promoted top-level thematic-break line.

#### `PrepareRenameFootnote`

```go
func (d *Document) PrepareRenameFootnote(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareRenameFootnote atomically renames one promoted footnote definition and
every parser-proven reference occurrence bound to that definition.

#### `PrepareRenameHeading`

```go
func (d *Document) PrepareRenameHeading(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareRenameHeading prepares a source-preserving rename of promoted heading content.

#### `PrepareReplaceAutoLink`

```go
func (d *Document) PrepareReplaceAutoLink(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceAutoLink prepares a source-preserving replacement of a promoted GFM autolink token.

#### `PrepareReplaceCodeSpan`

```go
func (d *Document) PrepareReplaceCodeSpan(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceCodeSpan prepares a source-preserving replacement of promoted code-span content.

#### `PrepareReplaceEmphasis`

```go
func (d *Document) PrepareReplaceEmphasis(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceEmphasis prepares a source-preserving replacement of promoted emphasis content.

#### `PrepareReplaceFencedCode`

```go
func (d *Document) PrepareReplaceFencedCode(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceFencedCode prepares a source-preserving replacement of promoted fenced-code content.

#### `PrepareReplaceFootnoteDefinitionBody`

```go
func (d *Document) PrepareReplaceFootnoteDefinitionBody(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceFootnoteDefinitionBody prepares a source-preserving replacement
of the conservative simple editable body of one promoted footnote definition.

#### `PrepareReplaceFrontMatterValue`

```go
func (d *Document) PrepareReplaceFrontMatterValue(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceFrontMatterValue prepares a source-preserving replacement of a promoted simple front-matter scalar value.

#### `PrepareReplaceHTMLAnchor`

```go
func (d *Document) PrepareReplaceHTMLAnchor(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceHTMLAnchor prepares a source-preserving replacement of a promoted HTML anchor id/name value.

#### `PrepareReplaceHTMLComment`

```go
func (d *Document) PrepareReplaceHTMLComment(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceHTMLComment prepares a source-preserving replacement of a promoted HTML comment payload.

#### `PrepareReplaceImageDestination`

```go
func (d *Document) PrepareReplaceImageDestination(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceImageDestination prepares a source-preserving replacement of a promoted image destination.

#### `PrepareReplaceInlineLinkDestination`

```go
func (d *Document) PrepareReplaceInlineLinkDestination(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceInlineLinkDestination prepares a source-preserving replacement of a promoted inline-link destination.

#### `PrepareReplaceListItem`

```go
func (d *Document) PrepareReplaceListItem(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceListItem prepares a source-preserving replacement of promoted list-item content.

#### `PrepareReplaceListItemSubtree`

```go
func (d *Document) PrepareReplaceListItemSubtree(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceListItemSubtree prepares replacement of one complete supported list-item subtree while preserving its external sibling shape and semantic parent.

#### `PrepareReplaceMathExpression`

```go
func (d *Document) PrepareReplaceMathExpression(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceMathExpression prepares a source-preserving replacement of one
reviewed mathematical payload while retaining its exact delimiter/container form.

#### `PrepareReplaceParagraph`

```go
func (d *Document) PrepareReplaceParagraph(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceParagraph prepares a source-preserving paragraph replacement.

#### `PrepareReplaceReferenceDefinitionDestination`

```go
func (d *Document) PrepareReplaceReferenceDefinitionDestination(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceReferenceDefinitionDestination prepares a source-preserving replacement of a promoted reference-definition destination.

#### `PrepareReplaceReferenceDefinitionTitle`

```go
func (d *Document) PrepareReplaceReferenceDefinitionTitle(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceReferenceDefinitionTitle prepares a source-preserving replacement of an existing promoted reference-definition title payload.

#### `PrepareReplaceSection`

```go
func (d *Document) PrepareReplaceSection(headingID NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceSection prepares source-preserving replacement of one complete promoted section subtree.

#### `PrepareReplaceSectionBody`

```go
func (d *Document) PrepareReplaceSectionBody(headingID NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceSectionBody prepares source-preserving replacement of one promoted section's direct body.

#### `PrepareReplaceStrikethrough`

```go
func (d *Document) PrepareReplaceStrikethrough(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceStrikethrough prepares a source-preserving replacement of promoted strikethrough content.

#### `PrepareReplaceStrong`

```go
func (d *Document) PrepareReplaceStrong(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceStrong prepares a source-preserving replacement of promoted strong-emphasis content.

#### `PrepareReplaceTableCell`

```go
func (d *Document) PrepareReplaceTableCell(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceTableCell prepares a source-preserving replacement of promoted table-cell content.

#### `PrepareReplaceTableRow`

```go
func (d *Document) PrepareReplaceTableRow(id NodeID, replacement []byte) (ChangeSet, error)
```

PrepareReplaceTableRow prepares source-preserving replacement of one complete promoted GFM table body row.

#### `PrepareSetTableAlignments`

```go
func (d *Document) PrepareSetTableAlignments(id NodeID, alignments []TableAlignment) (ChangeSet, error)
```

PrepareSetTableAlignments prepares one atomic source-preserving alignment update for every promoted GFM table column.

#### `PrepareSetTableColumnAlignment`

```go
func (d *Document) PrepareSetTableColumnAlignment(id NodeID, column int, alignment TableAlignment) (ChangeSet, error)
```

PrepareSetTableColumnAlignment prepares a source-preserving alignment change for one promoted GFM table column.

#### `PrepareSetTaskChecked`

```go
func (d *Document) PrepareSetTaskChecked(id NodeID, checked bool) (ChangeSet, error)
```

PrepareSetTaskChecked prepares a source-preserving GFM task state change.

#### `PrepareSyncTOC`

```go
func (d *Document) PrepareSyncTOC(headingID NodeID) (ChangeSet, error)
```

PrepareSyncTOC prepares source-preserving synchronization of an explicitly
designated empty/TOC-shaped section body. Arbitrary section bodies fail closed.

#### `QueryNodes`

```go
func (d *Document) QueryNodes(query NodeQuery) ([]NodeMatch, error)
```

QueryNodes returns at most query.Limit promoted nodes in existing structural
source order. The returned slice is caller-owned and no query state is retained.

#### `QuerySections`

```go
func (d *Document) QuerySections(query SectionQuery) ([]Section, error)
```

QuerySections returns at most query.Limit derived sections in source order.
The returned slice is caller-owned and contains the existing immutable Section
representation rather than a second query-specific section model.

#### `ReferenceDefinition`

```go
func (d *Document) ReferenceDefinition(id NodeID) (ReferenceDefinition, bool)
```

ReferenceDefinition returns typed detail for one promoted single-line reference definition.

#### `RenderHTML`

```go
func (d *Document) RenderHTML(writer io.Writer, options HTMLRenderOptions) error
```

RenderHTML streams a deterministic HTML fragment from this immutable snapshot. It consumes the Native semantic walk on demand, performs no filesystem/network access, and stops immediately on writer error. Invalid receiver, writer, or options report `ErrInvalidRender`.

#### `RenderHTMLDocument`

```go
func (d *Document) RenderHTMLDocument(writer io.Writer, options HTMLDocumentOptions) error
```

RenderHTMLDocument streams a deterministic standalone HTML document around the exact RenderHTML body. The zero metadata policy maps only exact lower-case `title`, `description`, `author`, and safe `lang` values from unique top-level source-proven simple front-matter scalars. `HTMLMetadataOmit` disables that mapping. The method performs no template, asset, filesystem, network, or command access and stops on writer error.

#### `RenderHTMLDocumentWithSourceMap`

```go
func (d *Document) RenderHTMLDocumentWithSourceMap(writer io.Writer, options HTMLDocumentOptions) ([]HTMLSourceMapEntry, error)
```

RenderHTMLDocumentWithSourceMap streams the same standalone HTML bytes and returns output-ordered source correlations only after successful completion. Eligible reviewed metadata maps to the wrapper bytes it emits; synthetic wrapper bytes are intentionally unmapped. Writer errors and short writes return a nil map.

#### `RenderHTMLWithSourceMap`

```go
func (d *Document) RenderHTMLWithSourceMap(writer io.Writer, options HTMLRenderOptions) ([]HTMLSourceMapEntry, error)
```

RenderHTMLWithSourceMap streams the same deterministic fragment bytes as RenderHTML and returns output-ordered source correlations only after successful completion. An outer mapping precedes a nested mapping that begins at the same output byte. Writer errors and short writes return a nil map.

#### `ResolveFragment`

```go
func (d *Document) ResolveFragment(fragment string) (FragmentTarget, bool)
```

ResolveFragment resolves an optional-leading-# URI fragment against heading-derived
and supported explicit HTML anchors. Zero or multiple matches fail closed.

#### `Section`

```go
func (d *Document) Section(headingID NodeID) (Section, bool)
```

Section returns the derived section governed by headingID.

#### `SectionChildHeadingIDs`

```go
func (d *Document) SectionChildHeadingIDs(headingID NodeID) ([]NodeID, bool)
```

SectionChildHeadingIDs returns one section's immediate child heading identities in source order.

#### `Sections`

```go
func (d *Document) Sections() []Section
```

Sections returns all derived document sections in source order.

#### `SourceRange`

```go
func (d *Document) SourceRange(range_ Range) ([]byte, bool)
```

SourceRange returns a copy of one valid byte range from the immutable source snapshot.
Caller mutations of the returned bytes do not affect the document.

#### `Strikethrough`

```go
func (d *Document) Strikethrough(id NodeID) (Strikethrough, bool)
```

Strikethrough returns typed detail for one promoted simple GFM strikethrough.

#### `Strong`

```go
func (d *Document) Strong(id NodeID) (Strong, bool)
```

Strong returns typed detail for one promoted simple strong-emphasis span.

#### `TOCStale`

```go
func (d *Document) TOCStale(headingID NodeID) (bool, bool)
```

TOCStale reports whether one explicitly designated section body is a recognized
TOC shape that differs from the TOC derived from this snapshot. The second result
is false when the target is missing or its direct body is not TOC-shaped.

#### `Table`

```go
func (d *Document) Table(id NodeID) (Table, bool)
```

Table returns typed detail for one promoted GFM table.

#### `TableAlignments`

```go
func (d *Document) TableAlignments(tableID NodeID) ([]TableAlignment, bool)
```

TableAlignments returns one semantic alignment per source-proven table column.
The returned slice is caller-owned.

#### `TableCell`

```go
func (d *Document) TableCell(id NodeID) (TableCell, bool)
```

TableCell returns typed detail for one promoted non-empty GFM table cell.

#### `TableHeaderCellIDs`

```go
func (d *Document) TableHeaderCellIDs(tableID NodeID) ([]NodeID, bool)
```

TableHeaderCellIDs returns the promoted non-empty header-cell identities owned by one promoted table in source order.
Empty or otherwise unpromoted header cells are omitted.

#### `TableRow`

```go
func (d *Document) TableRow(id NodeID) (TableRow, bool)
```

TableRow returns typed detail for one promoted GFM table body row.

#### `TableRowAlignments`

```go
func (d *Document) TableRowAlignments(rowID NodeID) ([]TableAlignment, bool)
```

TableRowAlignments returns the semantic column alignments for the table that owns one promoted body row.
The returned slice has exactly TableRow.ColumnCount entries and is caller-owned.

#### `TableRowCellIDs`

```go
func (d *Document) TableRowCellIDs(rowID NodeID) ([]NodeID, bool)
```

TableRowCellIDs returns the promoted non-empty cells owned by one promoted body row in source order.
Empty cells are omitted because they do not receive public cell identities.

#### `TableRowHeaderCellIDs`

```go
func (d *Document) TableRowHeaderCellIDs(rowID NodeID) ([]NodeID, bool)
```

TableRowHeaderCellIDs returns the promoted non-empty header cells for the table that owns one promoted body row.
Empty header cells are omitted because they do not receive public cell identities.

#### `TableRowIDs`

```go
func (d *Document) TableRowIDs(tableID NodeID) ([]NodeID, bool)
```

TableRowIDs returns the promoted body-row identities owned by one promoted table in source order.
The returned slice is caller-owned and can be empty even when BodyRowCount is non-zero.

#### `Task`

```go
func (d *Document) Task(id NodeID) (Task, bool)
```

Task returns typed detail for one promoted GFM task marker.

#### `ThematicBreak`

```go
func (d *Document) ThematicBreak(id NodeID) (ThematicBreak, bool)
```

ThematicBreak returns typed detail for one promoted top-level thematic break.

#### `ValidateFragment`

```go
func (d *Document) ValidateFragment(fragment string) bool
```

ValidateFragment reports whether fragment uniquely resolves in this exact snapshot.

### `DocumentBuilder` methods

#### `AppendAlert`

```go
func (b *DocumentBuilder) AppendAlert(kind AlertKind, inlineGFM string) error
```

AppendAlert appends one canonical top-level GitHub alert containing one
parser-proven paragraph. kind must be one of Note, Tip, Important, Warning,
or Caution. inlineGFM follows the same LF-only paragraph contract as AppendBlockquote.

#### `AppendAlertBlocks`

```go
func (b *DocumentBuilder) AppendAlertBlocks(kind AlertKind, content *DocumentBuilder) error
```

AppendAlertBlocks appends one canonical top-level GitHub alert from the current
reviewed body blocks of content. The child builder is snapshotted and later
changes do not affect this builder. Alerts cannot be nested inside blockquotes
or other alerts, so child alert blocks are rejected.

#### `AppendAlertContent`

```go
func (b *DocumentBuilder) AppendAlertContent(kind AlertKind, content ...Inline) error
```

AppendAlertContent appends one canonical top-level GitHub alert from typed
inline paragraph content.

#### `AppendBlockquote`

```go
func (b *DocumentBuilder) AppendBlockquote(inlineGFM string) error
```

AppendBlockquote appends one top-level blockquote containing one paragraph.

Non-empty LF-separated paragraph GFM is written with canonical '> ' on every
physical line. The block is retained only when construction-only source and
semantic proof reproduce exactly one top-level blockquote paragraph; broader
existing-source blockquote promotion remains unchanged.

#### `AppendBlockquoteBlocks`

```go
func (b *DocumentBuilder) AppendBlockquoteBlocks(depth int, content *DocumentBuilder) error
```

AppendBlockquoteBlocks appends one blockquote container from an existing child builder.

depth must be between 1 and 64. content is treated as an immutable construction
snapshot: its current reviewed body blocks are copied into the new container,
while later changes to content do not affect this builder. Every reviewed body-block
construction family is accepted, including recursive blockquote
children whose total structural depth remains at most 64. Front matter remains
a document envelope and is never accepted as a blockquote child.

#### `AppendBlockquoteContent`

```go
func (b *DocumentBuilder) AppendBlockquoteContent(content ...Inline) error
```

AppendBlockquoteContent appends one simple top-level blockquote from typed inline content.

#### `AppendFencedCode`

```go
func (b *DocumentBuilder) AppendFencedCode(content, info string) error
```

AppendFencedCode appends one top-level fenced code block.

Content may be empty or LF-separated multiline text. The canonical unindented
backtick fence is at least three bytes and grows beyond every potentially closing run in a
non-empty body. info is an optional single-line raw GFM info string and must not
contain backticks. Empty content produces adjacent opening/closing fence lines
without inventing a payload line.

#### `AppendFootnoteDefinition`

```go
func (b *DocumentBuilder) AppendFootnoteDefinition(label, body string) error
```

AppendFootnoteDefinition appends one canonical top-level footnote definition.
Body is one non-empty physical line; broader multiline parsed definitions remain
readable but are not synthesized by this conservative construction contract.

#### `AppendHeading`

```go
func (b *DocumentBuilder) AppendHeading(level int, inlineGFM string) error
```

AppendHeading appends one top-level ATX heading.

inlineGFM must be one non-empty physical line of valid UTF-8 GFM source. The
generated source is accepted only when reparsing proves the requested heading
level and exact content range.

#### `AppendHeadingContent`

```go
func (b *DocumentBuilder) AppendHeadingContent(level int, content ...Inline) error
```

AppendHeadingContent appends one top-level ATX heading from typed inline content.

#### `AppendMathBlock`

```go
func (b *DocumentBuilder) AppendMathBlock(payload string) error
```

AppendMathBlock appends one canonical top-level `$$...$$` mathematical block.
Multiline mathematical payload belongs in an exact-info `math` fenced block.

#### `AppendNestedBlockquote`

```go
func (b *DocumentBuilder) AppendNestedBlockquote(depth int, inlineGFM string) error
```

AppendNestedBlockquote appends one explicitly nested blockquote containing one paragraph.

depth is structural container depth and must be between 2 and 64. The writer
derives the canonical repeated "> " prefix on every physical line; caller
content remains raw paragraph GFM and must not introduce container structure
that changes the requested nesting hierarchy.

#### `AppendNestedBlockquoteContent`

```go
func (b *DocumentBuilder) AppendNestedBlockquoteContent(depth int, content ...Inline) error
```

AppendNestedBlockquoteContent appends one explicitly nested blockquote from typed inline content.

#### `AppendNestedOrderedList`

```go
func (b *DocumentBuilder) AppendNestedOrderedList(items ...ListItemInput) error
```

AppendNestedOrderedList appends one homogeneous nested ordered list.

The same structural depth contract as AppendNestedUnorderedList applies. Decimal
numbering starts at 1 in every list container and indentation follows the generated
parent marker width, including transitions such as '9.' to '10.'.

#### `AppendNestedOrderedTaskList`

```go
func (b *DocumentBuilder) AppendNestedOrderedTaskList(items ...TaskListItemInput) error
```

AppendNestedOrderedTaskList appends one homogeneous nested ordered GFM task list.
Numbering/indentation is container-local and each canonical task marker/state is proven.

#### `AppendNestedUnorderedList`

```go
func (b *DocumentBuilder) AppendNestedUnorderedList(items ...ListItemInput) error
```

AppendNestedUnorderedList appends one homogeneous nested unordered list.

Source-ordered ListItemInput values use Depth to describe the parent/child hierarchy.
The writer uses canonical '-' markers and derives
each nested indentation from the generated parent's exact content column.

#### `AppendNestedUnorderedTaskList`

```go
func (b *DocumentBuilder) AppendNestedUnorderedTaskList(items ...TaskListItemInput) error
```

AppendNestedUnorderedTaskList appends one homogeneous nested unordered GFM task list.
Structural depth follows ListItemInput and each canonical task marker/state is proven.

#### `AppendOrderedList`

```go
func (b *DocumentBuilder) AppendOrderedList(items ...string) error
```

AppendOrderedList appends one flat top-level ordered list.

The writer uses canonical sequential decimal markers beginning at 1 with '.' as the
delimiter. The generated items must reparse as one ordered list container.

#### `AppendOrderedTaskList`

```go
func (b *DocumentBuilder) AppendOrderedTaskList(items ...TaskListItem) error
```

AppendOrderedTaskList appends one flat top-level ordered GFM task list.

The writer combines canonical sequential '1.', '2.', ... list markers with the same
semantic task proof used by unordered task lists.

#### `AppendParagraph`

```go
func (b *DocumentBuilder) AppendParagraph(inlineGFM string) error
```

AppendParagraph appends one top-level paragraph.

The input must be non-empty valid UTF-8 GFM source with canonical LF line endings.
The complete input must reparse as exactly one top-level paragraph; input that
becomes multiple blocks or another block kind fails closed instead of being
escaped or normalized implicitly.

#### `AppendParagraphContent`

```go
func (b *DocumentBuilder) AppendParagraphContent(content ...Inline) error
```

AppendParagraphContent appends one top-level single-line paragraph from typed inline content.

#### `AppendReferenceDefinition`

```go
func (b *DocumentBuilder) AppendReferenceDefinition(label, destination string) error
```

AppendReferenceDefinition appends one top-level single-line link reference definition.

The writer uses canonical angle-bracket destination syntax without a title. The
generated definition is retained only when reparsing reproduces the exact
label, destination, and source mapping.

#### `AppendReferenceDefinitionWithTitle`

```go
func (b *DocumentBuilder) AppendReferenceDefinitionWithTitle(label, destination, title string) error
```

AppendReferenceDefinitionWithTitle appends one top-level single-line link
reference definition with a canonical double-quoted title.

The writer keeps canonical angle-bracket destination syntax and accepts the block
only when reparsing reproduces the exact label, destination, title, and
source mapping. title must not require escaping in the canonical form.

#### `AppendTable`

```go
func (b *DocumentBuilder) AppendTable(header []string, rows ...[]string) error
```

AppendTable appends one top-level unaligned GFM table.

At least one header column is required; body rows are optional. Every body row must
have the same width as header. Cell strings are caller-provided single-line
GFM source; empty cells are allowed. The builder writes canonical outer pipes
and '---' delimiter cells and retains the table only after exact table-container
proof plus body-row proof for every row that is present.

#### `AppendTableWithAlignments`

```go
func (b *DocumentBuilder) AppendTableWithAlignments(header []string, alignments []TableAlignment, rows ...[]string) error
```

AppendTableWithAlignments appends one top-level GFM table with explicit semantic column alignments.

The canonical outer-pipe/padding policy writes delimiter cells as '---', ':---',
'---:', or ':---:' for default, left, right, or center
alignment. alignments must have exactly one entry per header column; body rows
remain optional.

#### `AppendThematicBreak`

```go
func (b *DocumentBuilder) AppendThematicBreak() error
```

AppendThematicBreak appends one canonical top-level thematic break.

The builder writes exactly three hyphens and retains the block only when reparsing
observes one top-level thematic break over those exact bytes.

#### `AppendUnorderedList`

```go
func (b *DocumentBuilder) AppendUnorderedList(items ...string) error
```

AppendUnorderedList appends one flat top-level unordered list.

Each item must be one non-empty physical line of valid UTF-8 inline GFM. The writer
uses the canonical '-' marker and accepts the block only when reparsing proves
that every generated item belongs to the one requested list container.

#### `AppendUnorderedTaskList`

```go
func (b *DocumentBuilder) AppendUnorderedTaskList(items ...TaskListItem) error
```

AppendUnorderedTaskList appends one flat top-level unordered GFM task list.

The writer uses canonical '-' list markers and '[ ]'/'[x]' task markers. The block
is retained only when reparsing proves both the requested list container and
each requested semantic task marker/state.

#### `DeferFootnoteDefinition`

```go
func (b *DocumentBuilder) DeferFootnoteDefinition(label, body string) error
```

DeferFootnoteDefinition schedules one canonical top-level footnote definition
after ordinary body blocks and deferred ordinary reference definitions.

#### `DeferReferenceDefinition`

```go
func (b *DocumentBuilder) DeferReferenceDefinition(label, destination string) error
```

DeferReferenceDefinition schedules one canonical top-level reference definition
after the ordinary constructed body. ForwardReferenceLinkInline and
ForwardReferenceImageInline resolve only against explicitly deferred definitions;
ReferenceLinkInline and ReferenceImageInline still require prior definitions.

#### `DeferReferenceDefinitionWithTitle`

```go
func (b *DocumentBuilder) DeferReferenceDefinitionWithTitle(label, destination, title string) error
```

DeferReferenceDefinitionWithTitle schedules one canonical top-level reference
definition with a conservative double-quoted title after the ordinary body.

#### `Markdown`

```go
func (b *DocumentBuilder) Markdown() ([]byte, error)
```

Markdown returns newly generated canonical GFM source.

The returned bytes are caller-owned. The zero-value builder produces an empty
document. A nil builder reports ErrInvalidConstruction.

#### `SetTOMLFrontMatter`

```go
func (b *DocumentBuilder) SetTOMLFrontMatter(fields ...FrontMatterFieldInput) error
```

SetTOMLFrontMatter configures one canonical leading TOML front-matter envelope.
A DocumentBuilder can own at most one front-matter envelope.

#### `SetYAMLFrontMatter`

```go
func (b *DocumentBuilder) SetYAMLFrontMatter(fields ...FrontMatterFieldInput) error
```

SetYAMLFrontMatter configures one canonical leading YAML front-matter envelope.
A DocumentBuilder can own at most one front-matter envelope.

### `DocumentGraph` methods

#### `Backlinks`

```go
func (g *DocumentGraph) Backlinks(key DocumentKey) ([]GraphEdge, bool)
```

Backlinks returns resolved edges whose target is key in global edge order.

#### `Document`

```go
func (g *DocumentGraph) Document(key DocumentKey) (*Document, bool)
```

Document returns one immutable document snapshot by caller-defined key.

#### `DocumentKeys`

```go
func (g *DocumentGraph) DocumentKeys() []DocumentKey
```

DocumentKeys returns caller-defined document keys in graph-input order.

#### `Edges`

```go
func (g *DocumentGraph) Edges() []GraphEdge
```

Edges returns all resolved graph edges in deterministic document/source order.

#### `Outgoing`

```go
func (g *DocumentGraph) Outgoing(key DocumentKey) ([]GraphEdge, bool)
```

Outgoing returns resolved edges whose source is key in relationship source order.

#### `ReachableFrom`

```go
func (g *DocumentGraph) ReachableFrom(key DocumentKey) ([]DocumentKey, bool)
```

ReachableFrom returns every other document reachable from key using resolved graph
edges. Results are in deterministic breadth-first discovery order; self cycles are omitted.

#### `RelatedDocuments`

```go
func (g *DocumentGraph) RelatedDocuments(key DocumentKey) ([]DocumentKey, bool)
```

RelatedDocuments returns direct incoming-or-outgoing neighboring documents in the
original graph-input order. Self edges and duplicate neighbors are omitted.

### `Emphasis` methods

#### `ID`

```go
func (e Emphasis) ID() NodeID
```

ID returns the emphasis span's snapshot-scoped node identity.

#### `Range`

```go
func (e Emphasis) Range() Range
```

Range returns the exact emphasis content span replaced by PrepareReplaceEmphasis.

### `ExtensionNode` methods

#### `Attribute`

```go
func (n ExtensionNode) Attribute(name string) (string, bool)
```

Attribute returns the unique extension metadata value named name.

#### `Attributes`

```go
func (n ExtensionNode) Attributes() []ExtensionAttribute
```

Attributes returns caller-owned extension metadata in recognizer-provided order.

#### `ExtensionID`

```go
func (n ExtensionNode) ExtensionID() ExtensionID
```

ExtensionID returns the namespace that produced this observation.

#### `Kind`

```go
func (n ExtensionNode) Kind() ExtensionKind
```

Kind returns the extension-local semantic kind.

#### `Range`

```go
func (n ExtensionNode) Range() Range
```

Range returns the exact snapshot-local source range claimed by the extension.

### `ExtensionSource` methods

#### `Text`

```go
func (s ExtensionSource) Text() string
```

Text returns the complete immutable source snapshot as a string.

### `FencedBlock` methods

#### `Closed`

```go
func (f FencedBlock) Closed() bool
```

Closed reports whether a matching closing fence is present in source.

#### `ClosingFenceLength`

```go
func (f FencedBlock) ClosingFenceLength() (int, bool)
```

ClosingFenceLength returns the number of delimiter bytes in the closing fence.

#### `ClosingFenceRange`

```go
func (f FencedBlock) ClosingFenceRange() (Range, bool)
```

ClosingFenceRange returns the exact closing delimiter run when the block is closed.

#### `ClosingIndent`

```go
func (f FencedBlock) ClosingIndent() (int, bool)
```

ClosingIndent returns the source indentation before the closing delimiter.

#### `FenceChar`

```go
func (f FencedBlock) FenceChar() byte
```

FenceChar returns the opening delimiter byte, either '`' or '~'.

#### `ID`

```go
func (f FencedBlock) ID() NodeID
```

ID returns the snapshot-scoped identity shared with FencedCode when the same
block also satisfies the historical contiguous replacement contract.

#### `Info`

```go
func (f FencedBlock) Info() (string, bool)
```

Info returns the parser-proven trimmed info string when one is present.

#### `InfoRange`

```go
func (f FencedBlock) InfoRange() (Range, bool)
```

InfoRange returns the exact source bytes corresponding to Info.

#### `Language`

```go
func (f FencedBlock) Language() (string, bool)
```

Language returns the parser-proven language token derived from the info string.
Marksplice treats the value only as metadata and does not interpret the payload.

#### `OpeningFenceLength`

```go
func (f FencedBlock) OpeningFenceLength() int
```

OpeningFenceLength returns the number of delimiter bytes in the opening fence.

#### `OpeningFenceRange`

```go
func (f FencedBlock) OpeningFenceRange() Range
```

OpeningFenceRange returns the exact opening delimiter run, excluding indentation
and info-string source.

#### `OpeningIndent`

```go
func (f FencedBlock) OpeningIndent() int
```

OpeningIndent returns the source indentation before the opening delimiter.

#### `Range`

```go
func (f FencedBlock) Range() Range
```

Range returns the exact complete physical source owned by the fenced block.
A closing-fence line terminator is included when present; an unclosed block
owns source through EOF.

### `FencedCode` methods

#### `ID`

```go
func (f FencedCode) ID() NodeID
```

ID returns the fenced code block's snapshot-scoped node identity.

#### `Range`

```go
func (f FencedCode) Range() Range
```

Range returns the exact fenced-code content span replaced by PrepareReplaceFencedCode.
Internal body line endings are part of this span. Fence lines, info-string source,
and the final line ending immediately before a closing fence are outside it.
For an unclosed block the payload still excludes the preserved trailing source
line ending when one is present.

### `FootnoteDefinition` methods

#### `BodyRange`

```go
func (f FootnoteDefinition) BodyRange() (Range, bool)
```

BodyRange returns the exact simple body span suitable for source-preserving
replacement. Segmented or multiline definitions return false; use
Document.FootnoteDefinitionBodyRanges for read-only semantic body segments.

#### `ID`

```go
func (f FootnoteDefinition) ID() NodeID
```

ID returns the definition's snapshot-scoped structural identity.

#### `Label`

```go
func (f FootnoteDefinition) Label() string
```

Label returns the parser-proven footnote label.

#### `LabelRange`

```go
func (f FootnoteDefinition) LabelRange() Range
```

LabelRange returns the exact source bytes containing Label.

#### `Range`

```go
func (f FootnoteDefinition) Range() Range
```

Range returns the exact complete physical source owned by the definition.

### `FootnoteReference` methods

#### `DefinitionID`

```go
func (r FootnoteReference) DefinitionID() (NodeID, bool)
```

DefinitionID returns the promoted definition that owns this reference when
complete top-level source ownership is proven.

#### `Label`

```go
func (r FootnoteReference) Label() string
```

Label returns the parser-proven footnote label.

#### `LabelRange`

```go
func (r FootnoteReference) LabelRange() Range
```

LabelRange returns the exact source bytes containing Label.

#### `Occurrence`

```go
func (r FootnoteReference) Occurrence() int
```

Occurrence returns the zero-based source-order occurrence for this definition.

#### `Range`

```go
func (r FootnoteReference) Range() Range
```

Range returns the exact `[^label]` source token span.

### `FragmentTarget` methods

#### `Kind`

```go
func (t FragmentTarget) Kind() FragmentTargetKind
```

Kind returns whether this target is a derived heading anchor or supported explicit HTML anchor.

#### `NodeID`

```go
func (t FragmentTarget) NodeID() NodeID
```

NodeID returns the snapshot-scoped node identity owning the fragment target.

#### `Value`

```go
func (t FragmentTarget) Value() string
```

Value returns the resolved fragment value without a leading '#'.

### `FrontMatter` methods

#### `ClosingRange`

```go
func (f FrontMatter) ClosingRange() Range
```

ClosingRange returns the exact closing delimiter bytes.

#### `Format`

```go
func (f FrontMatter) Format() FrontMatterFormat
```

Format returns whether the envelope uses the reviewed YAML or TOML delimiters.

#### `OpeningRange`

```go
func (f FrontMatter) OpeningRange() Range
```

OpeningRange returns the exact opening delimiter bytes.

#### `Range`

```go
func (f FrontMatter) Range() Range
```

Range returns the complete envelope from the opening delimiter through the closing delimiter.
A physical line terminator following the closing delimiter is outside this range.

### `FrontMatterField` methods

#### `Format`

```go
func (f FrontMatterField) Format() FrontMatterFormat
```

Format returns whether the field belongs to a YAML or TOML front-matter envelope.

#### `ID`

```go
func (f FrontMatterField) ID() NodeID
```

ID returns the field's snapshot-scoped node identity.

#### `Key`

```go
func (f FrontMatterField) Key() string
```

Key returns the recognized simple scalar field key.

#### `Range`

```go
func (f FrontMatterField) Range() Range
```

Range returns the exact scalar value span replaced by PrepareReplaceFrontMatterValue.
Delimiters, key spelling, separator spacing, quote wrappers, comments, and line endings are outside this range.

### `GraphEdge` methods

#### `Fragment`

```go
func (e GraphEdge) Fragment() (string, bool)
```

Fragment returns the local target fragment when the edge carries one. For
cross-document relationships this is the fragment supplied by the resolver.

#### `FragmentTarget`

```go
func (e GraphEdge) FragmentTarget() (FragmentTarget, bool)
```

FragmentTarget returns the uniquely resolved target snapshot fragment when one exists.

#### `Relationship`

```go
func (e GraphEdge) Relationship() LinkRelationship
```

Relationship returns the immutable link relationship that produced this edge.

#### `SourceDocument`

```go
func (e GraphEdge) SourceDocument() DocumentKey
```

SourceDocument returns the caller-defined logical source document key.

#### `TargetDocument`

```go
func (e GraphEdge) TargetDocument() DocumentKey
```

TargetDocument returns the caller-defined logical target document key.

### `HTMLAnchor` methods

#### `Attribute`

```go
func (a HTMLAnchor) Attribute() HTMLAnchorAttribute
```

Attribute returns whether the promoted anchor targets an id or name attribute.

#### `ID`

```go
func (a HTMLAnchor) ID() NodeID
```

ID returns the HTML anchor's snapshot-scoped node identity.

#### `Range`

```go
func (a HTMLAnchor) Range() Range
```

Range returns the exact quoted attribute value span replaced by PrepareReplaceHTMLAnchor.
Tag/attribute spelling, spacing, quote wrappers, and other attributes are outside this range.

### `HTMLComment` methods

#### `ID`

```go
func (c HTMLComment) ID() NodeID
```

ID returns the HTML comment's snapshot-scoped node identity.

#### `Range`

```go
func (c HTMLComment) Range() Range
```

Range returns the exact comment payload span replaced by PrepareReplaceHTMLComment.
Comment delimiters and preserved inner horizontal padding are outside this range.

### `HTMLSourceMapEntry` methods

#### `OutputRange`

```go
func (m HTMLSourceMapEntry) OutputRange() HTMLOutputRange
```

OutputRange returns the half-open byte range in the exact rendered HTML output represented by this entry.

#### `SourceRange`

```go
func (m HTMLSourceMapEntry) SourceRange() Range
```

SourceRange returns the half-open byte range in the immutable Markdown snapshot used for this render. The range is snapshot-local correlation metadata, not mutation authority or durable identity.

### `HTMLOutputRange` methods

#### `Valid`

```go
func (r HTMLOutputRange) Valid(total int) bool
```

Valid reports whether the output range is ordered and contained in an HTML result of total bytes.

### `Heading` methods

#### `ID`

```go
func (h Heading) ID() NodeID
```

ID returns the heading's snapshot-scoped node identity.

#### `Level`

```go
func (h Heading) Level() int
```

Level returns the GFM heading level from 1 through 6.

#### `Range`

```go
func (h Heading) Range() Range
```

Range returns the exact heading-content byte span replaced by PrepareRenameHeading.
ATX markers, optional closing markers, Setext underlines, and line endings are outside this range.

#### `Style`

```go
func (h Heading) Style() HeadingStyle
```

Style returns whether the heading uses ATX or Setext source syntax.

### `HeadingAnchor` methods

#### `HeadingID`

```go
func (a HeadingAnchor) HeadingID() NodeID
```

HeadingID returns the snapshot-scoped heading identity that owns this anchor.

#### `Value`

```go
func (a HeadingAnchor) Value() string
```

Value returns the fragment value without a leading '#'.

### `Image` methods

#### `ID`

```go
func (i Image) ID() NodeID
```

ID returns the image's snapshot-scoped node identity.

#### `Range`

```go
func (i Image) Range() Range
```

Range returns the exact destination span replaced by PrepareReplaceImageDestination.
The image marker, alt text, parentheses, destination wrappers, title syntax, and surrounding source are outside this range.

### `InlineLink` methods

#### `Destination`

```go
func (l InlineLink) Destination() string
```

Destination returns the parser-proven semantic link destination.

#### `ID`

```go
func (l InlineLink) ID() NodeID
```

ID returns the inline link's snapshot-scoped node identity.

#### `Range`

```go
func (l InlineLink) Range() Range
```

Range returns the exact destination span replaced by PrepareReplaceInlineLinkDestination.
Label, parentheses, destination wrappers, title syntax, and surrounding source are outside this range.

#### `Title`

```go
func (l InlineLink) Title() (string, bool)
```

Title returns the parser-proven semantic link title when one is present.

### `KnowledgeIndex` methods

#### `Aliases`

```go
func (k *KnowledgeIndex) Aliases(key DocumentKey) ([]KnowledgeAlias, bool)
```

Aliases returns exact aliases for key in caller-provided order. The returned slice is caller-owned.

#### `DocumentsWithTag`

```go
func (k *KnowledgeIndex) DocumentsWithTag(tag KnowledgeTag) []DocumentKey
```

DocumentsWithTag returns graph documents carrying the exact tag in original graph-input order.

#### `ReachableFrom`

```go
func (k *KnowledgeIndex) ReachableFrom(key DocumentKey) ([]DocumentKey, bool)
```

ReachableFrom returns every other document reachable through the union of resolved
Markdown graph edges and caller-declared logical references. For each visited source,
graph edges are considered first, then logical references. Results are deterministic
breadth-first discovery order and self/cyclic paths never duplicate a document.

#### `ReferencedBy`

```go
func (k *KnowledgeIndex) ReferencedBy(key DocumentKey) ([]KnowledgeReference, bool)
```

ReferencedBy returns logical references whose target is key in global reference order.

#### `References`

```go
func (k *KnowledgeIndex) References() []KnowledgeReference
```

References returns all logical references in graph document order and per-document caller order.

#### `ReferencesFrom`

```go
func (k *KnowledgeIndex) ReferencesFrom(key DocumentKey) ([]KnowledgeReference, bool)
```

ReferencesFrom returns logical references whose source is key in caller order.

#### `RelatedDocuments`

```go
func (k *KnowledgeIndex) RelatedDocuments(key DocumentKey) ([]DocumentKey, bool)
```

RelatedDocuments returns unique direct neighbors across both resolved Markdown graph
edges and caller-declared logical references in original graph-input order. Self
relationships are omitted.

#### `ResolveAlias`

```go
func (k *KnowledgeIndex) ResolveAlias(alias KnowledgeAlias) (DocumentKey, bool)
```

ResolveAlias resolves one exact globally unique alias to an existing graph document.
Canonical DocumentKey values are not aliases and should be queried through DocumentGraph.

#### `Tags`

```go
func (k *KnowledgeIndex) Tags(key DocumentKey) ([]KnowledgeTag, bool)
```

Tags returns exact tags for key in caller-provided order. The returned slice is caller-owned.

### `KnowledgeReference` methods

#### `SourceDocument`

```go
func (r KnowledgeReference) SourceDocument() DocumentKey
```

SourceDocument returns the caller-defined logical source document key.

#### `TargetDocument`

```go
func (r KnowledgeReference) TargetDocument() DocumentKey
```

TargetDocument returns the caller-defined logical target document key.

### `LinkRelationship` methods

#### `Destination`

```go
func (r LinkRelationship) Destination() string
```

Destination returns the parser-resolved semantic destination.
Other-document paths and URLs remain opaque data; Marksplice does not access them.

#### `FragmentStatus`

```go
func (r LinkRelationship) FragmentStatus() LinkFragmentStatus
```

FragmentStatus reports intra-document fragment resolution using the same semantics
as ResolveFragment for destinations beginning with '#'. Other destinations return NotApplicable.

#### `FragmentTarget`

```go
func (r LinkRelationship) FragmentTarget() (FragmentTarget, bool)
```

FragmentTarget returns the resolved target when FragmentStatus is Resolved.

#### `IsEmail`

```go
func (r LinkRelationship) IsEmail() bool
```

IsEmail reports whether an autolink relationship was parser-classified as an
email autolink. It is false for every non-autolink relationship.

#### `Kind`

```go
func (r LinkRelationship) Kind() LinkRelationshipKind
```

Kind returns the semantic relationship/source-form family.

#### `Reference`

```go
func (r LinkRelationship) Reference() (string, ReferenceForm, bool)
```

Reference returns the parser-resolved reference label and source form for a
reference link/image. Direct links/images and autolinks return false.

#### `ReferenceDefinitionID`

```go
func (r LinkRelationship) ReferenceDefinitionID() (NodeID, bool)
```

ReferenceDefinitionID returns the promoted single-line definition that can be
proven to uniquely own this reference relationship. Unsupported or ambiguous
definition ownership returns false without invalidating the relationship.

#### `SourceNodeID`

```go
func (r LinkRelationship) SourceNodeID() (NodeID, bool)
```

SourceNodeID returns an existing promoted source node identity when this exact
relationship already belongs to the ordinary public node model.

#### `SourceOffset`

```go
func (r LinkRelationship) SourceOffset() int
```

SourceOffset returns the parser-proven byte offset where the relationship's
source syntax starts. It is diagnostic/ordering metadata, not a mutation range.

#### `Title`

```go
func (r LinkRelationship) Title() (string, bool)
```

Title returns the parser-resolved title when one is present.

### `ListItem` methods

#### `ChildIDs`

```go
func (i ListItem) ChildIDs() []NodeID
```

ChildIDs returns the immediate supported list-item child identities in source order.
Semantic children outside the promoted public subset are omitted.

#### `HasChildren`

```go
func (i ListItem) HasChildren() bool
```

HasChildren reports whether the supported single-line item owns one or more semantic direct child list items.
It can be true even when ChildIDs is empty because unsupported children are not assigned public identities.

#### `ID`

```go
func (i ListItem) ID() NodeID
```

ID returns the list item's snapshot-scoped node identity.

#### `Marker`

```go
func (i ListItem) Marker() byte
```

Marker returns the source marker/delimiter byte.
Unordered items use '-', '*', or '+'; ordered items use '.' or ')'.

#### `Ordered`

```go
func (i ListItem) Ordered() bool
```

Ordered reports whether the item belongs to an ordered list.

#### `ParentID`

```go
func (i ListItem) ParentID() (NodeID, bool)
```

ParentID returns the immediate supported list-item parent's snapshot-scoped identity.
The boolean is false for root items and when the semantic parent exists but is not publicly promoted.

#### `Range`

```go
func (i ListItem) Range() Range
```

Range returns the exact list-item content span replaced by PrepareReplaceListItem.
Indentation, list numbering, marker/delimiter bytes, post-marker spacing, and line endings are outside this range.

#### `SubtreeRange`

```go
func (i ListItem) SubtreeRange() (Range, bool)
```

SubtreeRange returns the exact complete supported subtree source span used by structural list-item operations.
The boolean is false when Marksplice cannot prove that every semantic descendant belongs to the supported list-item model.

### `MathExpression` methods

#### `ID`

```go
func (m MathExpression) ID() NodeID
```

ID returns the snapshot-scoped identity. Fenced math shares the underlying FencedBlock ID.

#### `PayloadRange`

```go
func (m MathExpression) PayloadRange() (Range, bool)
```

PayloadRange returns one exact contiguous payload span when available.
Dollar/backtick forms always expose it; fenced math may be non-contiguous or empty.

#### `Range`

```go
func (m MathExpression) Range() Range
```

Range returns the complete source-owned mathematical syntax/container.

#### `Style`

```go
func (m MathExpression) Style() MathExpressionStyle
```

Style returns the exact reviewed source delimiter/container form.

### `Node` methods

#### `ID`

```go
func (n Node) ID() NodeID
```

ID returns the snapshot-scoped node identity.

#### `Kind`

```go
func (n Node) Kind() Kind
```

Kind returns the structural node category.

### `NodeID` methods

#### `String`

```go
func (id NodeID) String() string
```

String returns a diagnostic representation of the snapshot-scoped ID.

### `NodeMatch` methods

#### `Node`

```go
func (m NodeMatch) Node() Node
```

Node returns the promoted structural node summary.

#### `Range`

```go
func (m NodeMatch) Range() Range
```

Range returns the same operation-oriented source range already exposed by the
matched node kind's typed Range() accessor. It is a query-selection span, not
independent mutation authority.

### `Paragraph` methods

#### `ID`

```go
func (p Paragraph) ID() NodeID
```

ID returns the paragraph's snapshot-scoped node identity.

#### `Range`

```go
func (p Paragraph) Range() Range
```

Range returns the exact paragraph byte span replaced by PrepareReplaceParagraph.
A line ending immediately following the paragraph is outside this range.

### `Range` methods

#### `Valid`

```go
func (r Range) Valid(total int) bool
```

Valid reports whether r is ordered and contained in a source of total bytes.

### `ReferenceDefinition` methods

#### `Destination`

```go
func (r ReferenceDefinition) Destination() string
```

Destination returns the parser-proven semantic reference destination.

#### `ID`

```go
func (r ReferenceDefinition) ID() NodeID
```

ID returns the reference definition's snapshot-scoped node identity.

#### `Label`

```go
func (r ReferenceDefinition) Label() string
```

Label returns the parser-proven reference-definition label as authored.

#### `Range`

```go
func (r ReferenceDefinition) Range() Range
```

Range returns the exact destination span replaced by PrepareReplaceReferenceDefinitionDestination.
Label, colon, destination wrappers, title syntax, indentation, trailing spaces, and line endings are outside this range.

#### `Title`

```go
func (r ReferenceDefinition) Title() (string, bool)
```

Title returns the parser-proven semantic reference title when one is present.

### `Section` methods

#### `BodyRange`

```go
func (s Section) BodyRange() Range
```

BodyRange returns the direct body source span after the heading line and before the next heading of any level.
Nested subsection headings and their content are outside this range.

#### `HeadingID`

```go
func (s Section) HeadingID() NodeID
```

HeadingID returns the snapshot-scoped heading node identity governing this section.

#### `Level`

```go
func (s Section) Level() int
```

Level returns the governing GFM heading level from 1 through 6.

#### `ParentHeadingID`

```go
func (s Section) ParentHeadingID() (NodeID, bool)
```

ParentHeadingID returns the governing heading ID of the nearest enclosing section.

#### `Range`

```go
func (s Section) Range() Range
```

Range returns the complete section subtree source span, including its heading.
The range ends immediately before the next heading of equal or higher level, or at end of source.

### `Strikethrough` methods

#### `ID`

```go
func (s Strikethrough) ID() NodeID
```

ID returns the strikethrough's snapshot-scoped node identity.

#### `Range`

```go
func (s Strikethrough) Range() Range
```

Range returns the exact strikethrough content span replaced by PrepareReplaceStrikethrough.

### `Strong` methods

#### `ID`

```go
func (s Strong) ID() NodeID
```

ID returns the strong span's snapshot-scoped node identity.

#### `Range`

```go
func (s Strong) Range() Range
```

Range returns the exact strong content span replaced by PrepareReplaceStrong.

### `Table` methods

#### `BodyRowCount`

```go
func (t Table) BodyRowCount() int
```

BodyRowCount returns the semantic number of body rows, including rows outside the promoted public row subset.

#### `ColumnCount`

```go
func (t Table) ColumnCount() int
```

ColumnCount returns the semantic/source-proven number of table columns.

#### `ID`

```go
func (t Table) ID() NodeID
```

ID returns the table's snapshot-scoped node identity.

#### `Range`

```go
func (t Table) Range() Range
```

Range returns the exact complete table source span.
It owns the header row, delimiter row, and every semantic body row; when present, the final owned line terminator is included.

### `TableCell` methods

#### `Column`

```go
func (c TableCell) Column() int
```

Column returns the zero-based column index within the mapped table row.

#### `Header`

```go
func (c TableCell) Header() bool
```

Header reports whether the cell belongs to the table header row.

#### `ID`

```go
func (c TableCell) ID() NodeID
```

ID returns the table cell's snapshot-scoped node identity.

#### `Range`

```go
func (c TableCell) Range() Range
```

Range returns the exact table-cell content span replaced by PrepareReplaceTableCell.
Pipes, cell padding, alignment syntax, neighboring cells, and line endings are outside this range.

#### `RowID`

```go
func (c TableCell) RowID() (NodeID, bool)
```

RowID returns the promoted GFM body row that owns this cell.
The boolean is false for header cells and when no promoted body-row identity is available.

#### `TableID`

```go
func (c TableCell) TableID() (NodeID, bool)
```

TableID returns the promoted GFM table that owns this cell.
The boolean is false when no promoted table identity is available.

### `TableRow` methods

#### `ColumnCount`

```go
func (r TableRow) ColumnCount() int
```

ColumnCount returns the semantic/source-proven number of columns in the body row.

#### `ID`

```go
func (r TableRow) ID() NodeID
```

ID returns the table row's snapshot-scoped node identity.

#### `NextID`

```go
func (r TableRow) NextID() (NodeID, bool)
```

NextID returns the nearest promoted body row after this row in the same table.

#### `PreviousID`

```go
func (r TableRow) PreviousID() (NodeID, bool)
```

PreviousID returns the nearest promoted body row before this row in the same table.

#### `Range`

```go
func (r TableRow) Range() Range
```

Range returns the exact complete physical body-row span used by structural row operations.
When present, the row's own line terminator is included; header and delimiter rows are never part of this range.

#### `TableID`

```go
func (r TableRow) TableID() (NodeID, bool)
```

TableID returns the promoted GFM table that owns this body row.
The boolean is false when no promoted table identity is available.

### `Task` methods

#### `Checked`

```go
func (t Task) Checked() bool
```

Checked reports the semantic task state.

#### `ID`

```go
func (t Task) ID() NodeID
```

ID returns the task's snapshot-scoped node identity.

#### `Range`

```go
func (t Task) Range() Range
```

Range returns the exact one-byte task state span changed by PrepareSetTaskChecked.
The surrounding brackets and list-item source are outside this range.

### `ThematicBreak` methods

#### `ID`

```go
func (t ThematicBreak) ID() NodeID
```

ID returns the thematic break's snapshot-scoped node identity.

#### `Range`

```go
func (t ThematicBreak) Range() Range
```

Range returns the exact complete physical line owned by structural thematic-break operations.
When present, the line terminator is included.

### `UnresolvedReference` methods

#### `Form`

```go
func (r UnresolvedReference) Form() ReferenceForm
```

Form returns the explicit full or collapsed reference form.

#### `IsImage`

```go
func (r UnresolvedReference) IsImage() bool
```

IsImage reports whether the unresolved reference is an image reference.

#### `Reference`

```go
func (r UnresolvedReference) Reference() string
```

Reference returns the unresolved reference label.

### `WorkspaceDiagnostic` methods

#### `Fragment`

```go
func (d WorkspaceDiagnostic) Fragment() (string, bool)
```

Fragment returns the fragment associated with a fragment/document diagnostic.

#### `Kind`

```go
func (d WorkspaceDiagnostic) Kind() WorkspaceDiagnosticKind
```

Kind returns the diagnostic category.

#### `NodeID`

```go
func (d WorkspaceDiagnostic) NodeID() (NodeID, bool)
```

NodeID returns the snapshot-local node associated with a generated-index diagnostic.

#### `Relationship`

```go
func (d WorkspaceDiagnostic) Relationship() (LinkRelationship, bool)
```

Relationship returns the link relationship associated with this diagnostic.

#### `SourceDocument`

```go
func (d WorkspaceDiagnostic) SourceDocument() (DocumentKey, bool)
```

SourceDocument returns the caller-defined source document identity associated with this finding.

#### `SourceOffset`

```go
func (d WorkspaceDiagnostic) SourceOffset() (int, bool)
```

SourceOffset returns source-order diagnostic metadata when one exact source anchor exists.

#### `TargetDocument`

```go
func (d WorkspaceDiagnostic) TargetDocument() (DocumentKey, bool)
```

TargetDocument returns the caller-defined target document identity associated with this finding.

#### `UnresolvedReference`

```go
func (d WorkspaceDiagnostic) UnresolvedReference() (UnresolvedReference, bool)
```

UnresolvedReference returns conservative explicit unresolved reference metadata.
Shortcut bracket text is never reported because it is ambiguous with ordinary text.

### `WorkspaceRepair` methods

#### `Change`

```go
func (r WorkspaceRepair) Change() ChangeSet
```

Change returns the ordinary source-bound prepared change for this repair.

#### `Document`

```go
func (r WorkspaceRepair) Document() DocumentKey
```

Document returns the caller-defined document key to which the repair applies.

### `WorkspaceRepairPlan` methods

#### `Repairs`

```go
func (p WorkspaceRepairPlan) Repairs() []WorkspaceRepair
```

Repairs returns caller-owned repair values in deterministic planning order.

### `WorkspaceReport` methods

#### `Diagnostics`

```go
func (r *WorkspaceReport) Diagnostics() []WorkspaceDiagnostic
```

Diagnostics returns caller-owned diagnostics in deterministic validation order.

#### `Graph`

```go
func (r *WorkspaceReport) Graph() *DocumentGraph
```

Graph returns the immutable document graph produced by this validation run.

#### `RepairPlan`

```go
func (r *WorkspaceReport) RepairPlan() WorkspaceRepairPlan
```

RepairPlan returns the immutable conservative repair plan.

## `workspacefs` package

Import path:

```go
import "github.com/zoster81/marksplice/workspacefs"
```

`workspacefs` is a read-only adapter over a caller-supplied `fs.FS`. It performs no network access, command execution, or filesystem writes. Filesystem discovery remains outside the root document core.

### Exported limits and errors

```go
const (
    DefaultMaxDocuments     = 10_000
    DefaultMaxBytes   int64 = 256 << 20
    DefaultMaxDepth         = 64
    DefaultMaxRelationships = 250_000
)

var (
    ErrInvalidInput    error
    ErrBudgetExceeded error
)

type Limits struct {
    MaxDocuments     int
    MaxBytes         int64
    MaxDepth         int
    MaxRelationships int
}

type Options struct {
    Limits Limits
}
```

All operations require positive document/byte/relationship limits and a non-negative depth limit. `MaxDepth` means directory depth below `root` for `Scan` and relationship-hop depth for `Follow`.

### `DefaultOptions`

```go
func DefaultOptions() Options
```

Returns finite default limits for ordinary documentation workspaces.

### `Scan`

```go
func Scan(fsys fs.FS, root string, options Options) (*Workspace, error)
```

Discovers `.md` and `.markdown` files under `root`, parses them through ordinary `marksplice.Parse`, and assigns deterministic slash-relative `DocumentKey` values. Discovery is read-only and budget bounded.

### `Follow`

```go
func Follow(fsys fs.FS, root string, entries []string, options Options) (*Workspace, error)
```

Loads explicit Markdown entries and follows reviewed relative Markdown URI-path relationships. Entry paths are validated, deduplicated, and deterministically ordered; cycles are visited once. Relationship paths normalize literal dot segments relative to the source document, percent-decode components exactly once, ignore query text for filesystem lookup, and preserve fragments. Absolute/scheme/protocol-relative/backslash/encoded-traversal-or-separator/directory/extensionless forms are not filesystem targets. Missing reviewed local targets remain available to workspace validation instead of causing discovery to invent a document.

### `Workspace` methods

`Workspace` is immutable after a successful load and is safe for concurrent reads.

#### `Documents`

```go
func (w *Workspace) Documents() []marksplice.GraphDocument
```

Returns a caller-owned copy of the parsed graph inputs in deterministic workspace order.

#### `BuildGraph`

```go
func (w *Workspace) BuildGraph() (*marksplice.DocumentGraph, error)
```

Delegates to the existing immutable document-graph implementation using the workspace's reviewed local relationship mapping.

#### `Validate`

```go
func (w *Workspace) Validate(options marksplice.WorkspaceValidationOptions) (*marksplice.WorkspaceReport, error)
```

Delegates to the existing workspace validator through the same local relationship resolver used by `Follow` and `BuildGraph`. Reviewed local targets present in the loaded workspace are resolved; reviewed local targets absent from the loaded workspace are classified as missing; non-filesystem destination forms are ignored by the adapter.

Path resolution is slash-based and relative to the source document key. Path components are percent-decoded once, query text is excluded from lookup, and fragments are preserved for ordinary fragment resolution. Absolute/scheme/protocol-relative/backslash/encoded-traversal-or-separator/directory/extensionless forms are not filesystem targets. Case sensitivity, symlink behavior, and other host semantics remain properties of the caller-supplied `fs.FS`.

For path-policy details and examples, see [Links and Workspaces](recipes/links-workspaces.md).

## Maintenance rule

When the exported API changes, update this reference in the same change. Maintainer verification compares the exported callable inventories from `go doc -all .` and `go doc -all ./workspacefs` with this file so undocumented functions or methods fail the documentation gate.
