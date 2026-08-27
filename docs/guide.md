# Marksplice User Guide

Marksplice is a Pure-Go library for creating, understanding, querying, and source-preservingly editing GitHub Flavored Markdown. It uses a Marksplice-owned Native CommonMark/GFM parser and never requires callers to work with parser-specific AST types.

This guide is task-oriented. For the complete callable surface, including every exported function and method, see [`api-reference.md`](api-reference.md). Executable examples also live in the root [`example_test.go`](../example_test.go) and are published by pkg.go.dev.

## Install and import

Marksplice requires Go 1.26 or newer. The currently published beta can be installed explicitly:

```text
go get github.com/zoster81/marksplice@v0.1.0-beta.1
```

Import the root package:

```go
import "github.com/zoster81/marksplice"
```

Marksplice is pre-v1 software, so review the changelog when moving between v0 releases.

## The three core concepts

Most applications can be designed around three values:

1. `Document` is an immutable parsed source snapshot. It owns a copy of the input, source-mapped public nodes, semantic relationships, and reviewed higher-level views.
2. `ChangeSet` is an opaque mutation prepared against one exact `Document` snapshot. Applying it to different bytes fails with `ErrSourceConflict`.
3. `DocumentBuilder` creates new canonical GFM from reviewed structured intent. It is separate from existing-document editing so Marksplice never needs to normalize an existing document just to change one element.

The separation is intentional: **parse to understand, prepare a minimal change to edit, use a builder to create new source**.

## Parse and read Markdown

`Parse` copies the input and returns an immutable snapshot:

```go
source := []byte("# Title\n\nBody.\n")
doc, err := marksplice.Parse(source)
if err != nil {
    return err
}

for _, node := range doc.Nodes() {
    if node.Kind() != marksplice.KindHeading {
        continue
    }
    heading, ok := doc.Heading(node.ID())
    if !ok {
        continue
    }
    text, ok := doc.SourceRange(heading.Range())
    if !ok {
        return errors.New("heading range is not readable")
    }
    fmt.Printf("level=%d text=%s\n", heading.Level(), text)
}
```

`Nodes` intentionally returns only promoted public node kinds. `Node(id)` retrieves one summary; typed accessors such as `Heading`, `Paragraph`, `ListItem`, `Task`, `Table`, `TableRow`, `TableCell`, `FencedCode`, `CodeSpan`, `Emphasis`, `Strong`, `Strikethrough`, `InlineLink`, `Image`, `AutoLink`, `ReferenceDefinition`, `ThematicBreak`, and `Blockquote` expose the operation-specific detail that Marksplice can prove safely.

A `Range` is a half-open byte range `[Start, End)`. Use `SourceRange` to obtain a caller-owned copy. `NodeID` values are deterministic only for the exact snapshot and must not be treated as durable IDs across arbitrary reparses.

## Read broader semantic/source views

Some useful information is intentionally broader than ordinary mutation authority.

### Blockquotes and alerts

`Blockquote` exposes a complete promoted top-level blockquote. For multiline, lazy-continuation, nested, or multi-block source, use `BlockquoteContentRanges` rather than assuming one contiguous editable payload.

GitHub alerts are a semantic overlay on those blockquotes:

```go
doc, _ := marksplice.Parse([]byte("> [!WARNING]\n> Back up first.\n"))
for _, alert := range doc.Alerts() {
    marker, _ := doc.SourceRange(alert.MarkerRange())
    bodyRanges, _ := doc.AlertBodyRanges(alert.ID())
    fmt.Printf("marker=%s body-parts=%d\n", marker, len(bodyRanges))
}
```

`Alert(id)` retrieves one alert; `Alert.Kind`, `MarkerRange`, `Range`, and `ID` describe its source-backed identity and marker.

### Fenced blocks

`FencedBlocks` is the broad read-only fenced-container view. `FencedBlock` exposes opening/closing delimiter ranges and lengths, indentation, info string, language token, closure state, complete range, and per-line payload ranges through `FencedBlockContentRanges`.

`FencedCode` is deliberately narrower: it exists only when Marksplice can prove one contiguous payload suitable for `PrepareReplaceFencedCode`.

### Front matter

`FrontMatter` recognizes a document-leading YAML or TOML envelope without pretending Marksplice is a general YAML/TOML parser:

```go
doc, _ := marksplice.Parse([]byte("---\ntags:\n  - docs\n---\n\n# Guide\n"))
front, ok := doc.FrontMatter()
if ok {
    raw, _ := doc.SourceRange(front.Range())
    fmt.Printf("format=%v bytes=%d\n", front.Format(), len(raw))
}
```

`FrontMatterField(id)` is the narrower editable subset for unique simple top-level scalar fields. Complex or duplicate metadata can remain readable through the envelope while no unsafe field mutation is promoted.

### Footnotes and mathematics

`FootnoteDefinitions`, `FootnoteDefinition`, `FootnoteDefinitionBodyRanges`, and `FootnoteReferences` expose reviewed source-backed footnote relationships. `PrepareRenameFootnote` performs an atomic definition-plus-reference rename; `PrepareReplaceFootnoteDefinitionBody` is limited to the simple editable body subset.

`MathExpressions`, `MathExpression`, and `MathExpressionPayloadRanges` expose reviewed `$...$`, dollar-backtick, one-line `$$...$$`, and exact-info `math` fenced forms. The payload is opaque; Marksplice does not parse or render LaTeX.

### HTML anchors/comments

`HTMLComment` and `HTMLAnchor` expose only conservative source-proven forms. Their `Range` values are the exact payload/value spans accepted by `PrepareReplaceHTMLComment` and `PrepareReplaceHTMLAnchor`.

## Query nodes and sections

Use `QueryNodes` when you want bounded source-ordered structural selection without writing your own filter loop:

```go
matches, err := doc.QueryNodes(marksplice.NodeQuery{
    Kinds: []marksplice.Kind{marksplice.KindHeading, marksplice.KindParagraph},
    Limit: 100,
})
```

`Limit` must be positive. `Kinds` may be empty to select all promoted kinds. `Within` can restrict matches to an existing snapshot-local range. `NodeMatch.Node()` returns the summary and `NodeMatch.Range()` returns the already-reviewed typed range; it does not invent generic mutation authority.

`Sections` derives heading-governed section trees. `Section(id)` returns one section, `SectionChildHeadingIDs` returns direct child headings, and `QuerySections` supports bounded level/range filtering. `Section.Range()` owns the complete subtree while `BodyRange()` is only the direct body before the first child section.

## Source-preserving edits

Every ordinary existing-document edit follows the same pattern:

1. parse exact bytes;
2. select a promoted target;
3. call a named `Prepare...` operation;
4. apply the returned `ChangeSet` to the **same** source bytes.

Example:

```go
source := []byte("##  Old title  ##\n\nBody.\n")
doc, _ := marksplice.Parse(source)

var heading marksplice.Heading
for _, node := range doc.Nodes() {
    if node.Kind() == marksplice.KindHeading {
        heading, _ = doc.Heading(node.ID())
        break
    }
}

change, err := doc.PrepareRenameHeading(heading.ID(), []byte("New title"))
if err != nil {
    return err
}
updated, err := change.Apply(source)
if err != nil {
    return err
}
```

The result preserves the original heading style, surrounding spaces, optional closing ATX markers, line endings, and unrelated source.

### Scalar/content replacements

The public replacement family includes:

- `PrepareRenameHeading`;
- `PrepareReplaceParagraph`;
- `PrepareSetTaskChecked`;
- `PrepareReplaceFencedCode`;
- `PrepareReplaceCodeSpan`, `PrepareReplaceEmphasis`, `PrepareReplaceStrong`, `PrepareReplaceStrikethrough`;
- `PrepareReplaceInlineLinkDestination`, `PrepareReplaceImageDestination`, `PrepareReplaceAutoLink`;
- `PrepareReplaceReferenceDefinitionDestination`, `PrepareReplaceReferenceDefinitionTitle`;
- `PrepareReplaceFrontMatterValue`;
- `PrepareReplaceHTMLComment`, `PrepareReplaceHTMLAnchor`;
- `PrepareReplaceFootnoteDefinitionBody`, `PrepareRenameFootnote`;
- `PrepareReplaceMathExpression`;
- `PrepareReplaceTableCell` and `PrepareReplaceTableRow`.

Each operation validates its own source/semantic contract and fails closed when the requested replacement would change unsupported structure.

### Atomic composition

Independent changes prepared against the same snapshot can be combined:

```go
rename, _ := doc.PrepareRenameHeading(headingID, []byte("New title"))
replace, _ := doc.PrepareReplaceParagraph(paragraphID, []byte("New body."))
combined, err := doc.ComposeChanges(rename, replace)
if err != nil {
    return err
}
updated, err := combined.Apply(source)
```

`ComposeChanges` rejects byte overlap and semantic/model interactions. It does not expose a generic raw-patch batching API.

## Structural list operations

`ListItem` exposes ordered/unordered marker information, parent/child identities, and a complete `SubtreeRange` only when every semantic descendant belongs to the supported model.

Supported structural operations are explicit:

- `PrepareReplaceListItem` changes only one promoted item content span;
- `PrepareReplaceListItemSubtree` replaces a complete supported subtree;
- `PrepareRemoveListItem` removes a complete subtree;
- `PrepareInsertListItemBefore` / `PrepareInsertListItemAfter` insert a caller-provided complete sibling subtree;
- `PrepareAppendListItemChild` appends a complete direct child subtree;
- `PrepareMoveListItemBefore` / `PrepareMoveListItemAfter` move a complete supported subtree.

The fragment is reparsed and must satisfy the required sibling/parent/container shape. Marksplice does not silently reindent arbitrary malformed fragments.

## Structural section operations

Section editing follows the same complete-subtree model:

- `PrepareReplaceSectionBody` edits only the direct body;
- `PrepareReplaceSection` replaces the complete governed section;
- `PrepareRemoveSection` removes it;
- `PrepareInsertSectionBefore` / `PrepareInsertSectionAfter` insert a sibling section;
- `PrepareAppendSectionChild` appends a direct child section;
- `PrepareMoveSectionBefore` / `PrepareMoveSectionAfter` move a same-level section subtree.

Use `Section`, `Sections`, `SectionChildHeadingIDs`, and `QuerySections` to inspect the hierarchy first.

## Tables

`Table`, `TableRow`, and `TableCell` form the public source-backed table model. Useful navigation/accessors include:

- `TableAlignments`;
- `TableHeaderCellIDs`;
- `TableRowIDs`;
- `TableRowCellIDs` and `TableRowHeaderCellIDs`;
- `TableRowAlignments`;
- `TableCell.TableID` / `RowID`;
- `TableRow.TableID`, `PreviousID`, and `NextID`.

Mutation operations include row replacement/insertion/append/removal/move, single/all-column alignment changes, and complete column insertion/removal/move:

```go
change, err := doc.PrepareSetTableColumnAlignment(
    table.ID(), 1, marksplice.TableAlignmentRight,
)
```

Column operations require complete source proof for the header, delimiter, and every semantic body row. Empty/unpromoted cells never receive fabricated identities.

## Thematic breaks and blockquote removal

`ThematicBreak` exposes the complete owned physical line and `PrepareRemoveThematicBreak` removes it only when candidate parsing proves surrounding Markdown remains acceptable.

`PrepareRemoveBlockquote` removes one complete promoted top-level blockquote container, not merely a guessed marker span.

## Anchors, fragments, and TOCs

`HeadingAnchors` derives GitHub-compatible anchors with duplicate disambiguation. `HeadingAnchor(id)` returns one heading-derived anchor.

`ResolveFragment` resolves an optional-leading-`#` fragment against heading-derived and supported explicit HTML anchors. `ValidateFragment` is the boolean convenience form; ambiguous or missing targets fail closed.

`GenerateTOC` returns deterministic Markdown for the current section hierarchy. `TOCStale` and `PrepareSyncTOC` work only on explicitly designated bodies that match the conservative managed-TOC shape; arbitrary content is never overwritten as a guessed TOC.

## Link intelligence

`LinkRelationships` returns source-ordered immutable relationships for parser-resolved links, images, references, and autolinks. `LinkRelationship` exposes:

- semantic destination and optional title;
- relationship kind;
- reference value/form when applicable;
- source offset and optional promoted source `NodeID`;
- optional owning reference-definition ID;
- email-autolink classification;
- local fragment status and target.

The relationship surface is semantic intelligence, not a generic mutation range.

## Build a document graph

`BuildDocumentGraph` combines only documents that the caller explicitly supplies:

```go
index, _ := marksplice.Parse([]byte("# Index\n\n[guide](guide.md#guide)\n"))
guide, _ := marksplice.Parse([]byte("# Guide\n"))

graph, err := marksplice.BuildDocumentGraph([]marksplice.GraphDocument{
    {Key: "index", Document: index},
    {Key: "guide", Document: guide},
}, func(_ marksplice.DocumentKey, rel marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
    if rel.Destination() == "guide.md#guide" {
        return marksplice.DocumentResolution{Target: "guide", Fragment: "#guide"}, true
    }
    return marksplice.DocumentResolution{}, false
})
```

Marksplice performs no filesystem or network discovery. `DocumentKey` is opaque caller data. The resolver runs synchronously during the build and is never retained.

Graph queries are `Document`, `DocumentKeys`, `Edges`, `Outgoing`, `Backlinks`, `ReachableFrom`, and `RelatedDocuments`. `GraphEdge` exposes source/target document keys, the originating relationship, and optional fragment resolution.

## Validate an explicit workspace

`ValidateWorkspace` adds deterministic diagnostics and conservative repair planning over a caller-provided document set. A `WorkspaceResolver` classifies non-local relationships as ignored, resolved to an already-supplied document, or expected-but-missing.

`WorkspaceValidationOptions` can provide root documents for orphan/reachability diagnostics and caller-designated `ManagedTOC` targets.

```go
report, err := marksplice.ValidateWorkspace(
    []marksplice.GraphDocument{{Key: "guide", Document: doc}},
    nil,
    marksplice.WorkspaceValidationOptions{},
)
```

Use `WorkspaceReport.Diagnostics`, `Graph`, and `RepairPlan`. `WorkspaceDiagnostic` accessors expose only metadata meaningful for each `WorkspaceDiagnosticKind`; `UnresolvedReference()` returns typed conservative full/collapsed reference metadata. A `WorkspaceRepair` contains a target `DocumentKey` and an ordinary source-bound `ChangeSet`.

## Add syntax-independent knowledge metadata

`BuildKnowledgeIndex` layers caller-declared aliases, tags, and direct logical references on an existing `DocumentGraph`:

```go
knowledge, err := marksplice.BuildKnowledgeIndex(graph, []marksplice.KnowledgeDocument{
    {Document: "index", Aliases: []marksplice.KnowledgeAlias{"home"}, Tags: []marksplice.KnowledgeTag{"docs"}},
    {Document: "guide", Tags: []marksplice.KnowledgeTag{"docs"}, References: []marksplice.DocumentKey{"index"}},
})
```

It does not infer wikilinks, hashtags, paths, front matter, or URLs. Queries include `ResolveAlias`, `Aliases`, `Tags`, `DocumentsWithTag`, `References`, `ReferencesFrom`, `ReferencedBy`, combined `ReachableFrom`, and combined `RelatedDocuments`.

## Create new Markdown with DocumentBuilder

`NewDocumentBuilder` returns a mutable builder; the zero value is also usable. `Markdown()` returns caller-owned canonical LF GFM.

A simple document:

```go
builder := marksplice.NewDocumentBuilder()
_ = builder.AppendHeadingContent(1, marksplice.TextInline("Marksplice"))
_ = builder.AppendParagraphContent(
    marksplice.TextInline("See "),
    marksplice.LinkInline("https://example.com", marksplice.TextInline("the guide")),
)
source, err := builder.Markdown()
```

### Block construction

Builder block APIs include:

- `AppendHeading` / `AppendHeadingContent`;
- `AppendParagraph` / `AppendParagraphContent`;
- `AppendThematicBreak`;
- `AppendFencedCode`;
- flat `AppendUnorderedList`, `AppendOrderedList`, `AppendUnorderedTaskList`, `AppendOrderedTaskList`;
- homogeneous nested `AppendNestedUnorderedList`, `AppendNestedOrderedList`, `AppendNestedUnorderedTaskList`, `AppendNestedOrderedTaskList` using structured depth inputs;
- `AppendBlockquote`, `AppendBlockquoteContent`, `AppendNestedBlockquote`, `AppendNestedBlockquoteContent`, and `AppendBlockquoteBlocks`;
- `AppendAlert`, `AppendAlertContent`, and `AppendAlertBlocks`;
- `AppendTable` / `AppendTableWithAlignments`;
- `AppendReferenceDefinition` / `AppendReferenceDefinitionWithTitle`;
- `AppendFootnoteDefinition` and `AppendMathBlock`;
- `SetYAMLFrontMatter` / `SetTOMLFrontMatter`.

Construction fails with `ErrInvalidConstruction` when generated source cannot be reparsed and proven to match the requested structure.

### Typed inline construction

Prefer typed inline constructors when the caller owns semantic content rather than raw inline GFM:

- `TextInline` escapes punctuation so plain text cannot accidentally become Markdown syntax;
- `CodeInline` creates adaptive-backtick code spans;
- `EmphasisInline`, `StrongInline`, `StrikethroughInline` support the reviewed bounded nesting model;
- `LinkInline` / `LinkInlineWithTitle` and `ImageInline` / `ImageInlineWithTitle` create direct links/images;
- `AutoLinkInline` creates angle autolinks and `BareAutoLinkInline` requests a parser-proven GFM extended autolink token;
- `MathInline` and `MathBacktickInline` create reviewed inline math forms;
- `FootnoteReferenceInline` targets an immediate or deferred footnote definition.

### Reference links and deferred definitions

`ReferenceLinkInline` and `ReferenceImageInline` require an already-appended exact definition. `ForwardReferenceLinkInline` and `ForwardReferenceImageInline` resolve only against definitions explicitly scheduled with `DeferReferenceDefinition` or `DeferReferenceDefinitionWithTitle`.

`CollapsedReferenceLinkInline`, `ShortcutReferenceLinkInline`, `CollapsedReferenceImageInline`, and `ShortcutReferenceImageInline` use the emitted label and require exactly one normalized available definition.

Footnotes have the analogous `DeferFootnoteDefinition` plus `FootnoteReferenceInline` flow.

## Third-party read-only extensions

`ParseWithOptions` lets caller-linked Go code add **read-only** extension observations after the core parse succeeds. Extensions cannot reclassify core nodes or gain mutation, construction, graph, filesystem, network, or command authority.

```go
wiki := marksplice.Extension{
    ID: "example.org/wiki",
    Recognize: func(source marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
        text := source.Text()
        // Recognize extension-specific syntax and return exact ranges.
        _ = text
        return nil, nil
    },
}

doc, err := marksplice.ParseWithOptions(input, marksplice.ParseOptions{
    Extensions: []marksplice.Extension{wiki},
    ExtensionLimits: marksplice.ExtensionLimits{
        MaxNodes: 100,
        MaxMetadataBytes: 16 << 10,
    },
})
```

`ExtensionNodes` returns validated caller-owned observations. `ExtensionNode` exposes `ExtensionID`, extension-local `Kind`, exact `Range`, `Attributes`, and `Attribute(name)`. Recognizers run synchronously and serially and are not retained. They are ordinary application code: Marksplice bounds retained observations but does not sandbox a recognizer's own CPU, memory, goroutines, filesystem, network, or command behavior.

## Errors

Use `errors.Is` with the public sentinel families:

- `ErrNodeNotFound`;
- `ErrInvalidReplacement`;
- `ErrInvalidTargetKind`;
- `ErrSourceConflict`;
- `ErrInvalidConstruction`;
- `ErrInvalidQuery`;
- `ErrInvalidGraph`;
- `ErrInvalidWorkspace`;
- `ErrInvalidKnowledge`;
- `ErrInvalidExtension`.

Do not compare diagnostic error strings.

## Concurrency and ownership

A successfully parsed `Document` and immutable `DocumentGraph`, `KnowledgeIndex`, `WorkspaceReport`, and prepared `ChangeSet` values may be read/queried concurrently. Public variable-length results are caller-owned unless an API explicitly states otherwise.

`DocumentBuilder` is mutable and requires caller synchronization for concurrent use. Graph/workspace resolver callbacks and extension recognizers are invoked synchronously and are never retained after their build/parse call returns.

Callers must not concurrently mutate byte slices while passing them to an operation.

## Boundedness and I/O authority

Core operations are synchronous. Marksplice performs no implicit filesystem, network, or command I/O. `QueryNodes` and `QuerySections` require positive limits. Graph/workspace/knowledge operations are bounded by the explicit document collections supplied by the caller.

## What Marksplice deliberately does not do

Marksplice does not provide HTML/PDF rendering, embedded-language execution, filesystem crawling, network resolution, YAML/TOML serialization, LaTeX rendering, or arbitrary Markdown normalization. Dialect-specific syntax can be observed by opt-in third-party read-only extensions without changing the core CommonMark/GFM contract.

## Complete API reference

The complete exported callable surface is documented in [`api-reference.md`](api-reference.md). That reference includes every top-level function, every `Document`/`DocumentBuilder`/graph/workspace/knowledge method, and every exported value-object accessor, with the exact current Go signature and public GoDoc explanation.
