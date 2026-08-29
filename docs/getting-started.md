# Getting Started

This guide takes you from installation to a real source-preserving edit. It assumes you already know basic Go; you do not need to know Marksplice internals.

## 1. Install Marksplice

Marksplice requires Go 1.26 or newer. The current published beta is installed explicitly:

```sh
go get github.com/zoster81/marksplice@v0.5.0-beta.1
```

Import the root package:

```go
import "github.com/zoster81/marksplice"
```

## 2. Run a real example

From a Marksplice repository checkout:

```sh
go run ./examples/inspect
```

The program reads [`../examples/inspect/project-guide.md`](../examples/inspect/project-guide.md) from disk. That file contains front matter, sections, a task list, a fenced Go block, a table, and links.

Then run the editing example:

```sh
go run ./examples/edit
```

It reads [`../examples/edit/release-plan.md`](../examples/edit/release-plan.md), prepares several independent edits, combines them, and prints the updated Markdown. It never overwrites the committed fixture.

## 3. Parse the bytes you loaded

Marksplice parses bytes, not filenames. Your application decides how the bytes are obtained and where any result is written.

```go
source, err := os.ReadFile("project-guide.md")
if err != nil {
    return err
}

doc, err := marksplice.Parse(source)
if err != nil {
    return err
}
```

`Document` is an immutable snapshot of those bytes. Keep the original `source` if you intend to apply a prepared edit later.

`Parse` does not read other files, follow URLs, or crawl directories. For an explicit read-only multi-file workflow, the separate `workspacefs` package can scan or follow Markdown inside a caller-supplied `fs.FS` under finite resource limits; see [Links and workspaces](recipes/links-workspaces.md).

## 4. Inspect useful structure

`Document.Nodes()` returns source-ordered public structural nodes. Use a typed accessor when you need details for one kind.

```go
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
        return errors.New("heading source is unavailable")
    }
    fmt.Printf("level=%d text=%s\n", heading.Level(), text)
}
```

Other common views include:

- `Sections()` for heading-governed document sections;
- `Task(id)` for task state;
- `Table(id)`, `TableRow(id)`, and `TableCell(id)` for tables;
- `FencedBlocks()` for complete fenced-container metadata;
- `LinkRelationships()` for links, images, references, and autolinks;
- `FrontMatter()` for a recognized YAML/TOML document envelope.

See the runnable [`inspect` example](../examples/inspect/main.go) for these ideas on one file.

## 5. Query instead of scanning everything yourself

For bounded selection, use `QueryNodes` or `QuerySections`. Every query requires a positive limit.

```go
matches, err := doc.QueryNodes(marksplice.NodeQuery{
    Kinds: []marksplice.Kind{marksplice.KindHeading},
    Limit: 20,
})
if err != nil {
    return err
}
```

A `Range` is a half-open byte range `[Start, End)` within this exact snapshot. Queries can use `Within` to stay inside a section or another source region.

The [`query` example](../examples/query/main.go) finds unfinished tasks only inside one named section.

## 6. Prepare a source-preserving edit

Suppose you selected a heading and have its `NodeID`:

```go
change, err := doc.PrepareRenameHeading(headingID, []byte("Release Readiness"))
if err != nil {
    return err
}
```

This does not modify `doc` or `source`. It returns a `ChangeSet`: an opaque change prepared for this exact snapshot.

Now apply it to the original bytes:

```go
updated, err := change.Apply(source)
if err != nil {
    return err
}
```

Marksplice changes only the source spans owned by that operation. Unrelated bytes are not regenerated through a whole-document renderer.

If `source` has changed since parsing, `Apply` returns `ErrSourceConflict`. Parse the newer bytes and prepare the edit again; do not reuse stale `NodeID` values or a stale `ChangeSet`.

## 7. Combine independent edits

Changes prepared from the same snapshot can be combined atomically:

```go
combined, err := doc.ComposeChanges(rename, replaceParagraph, checkTask, updateCell)
if err != nil {
    return err
}

updated, err := combined.Apply(source)
```

`ComposeChanges` rejects overlapping or semantically interacting edits rather than applying them in a guessed order.

The complete [`edit` example](../examples/edit/main.go) combines a heading rename, paragraph replacement, task update, and table-cell update while checking that unrelated source remains present.

## 8. Save the result when your application chooses

Marksplice returns bytes; it does not own filesystem mutation.

```go
if err := os.WriteFile("project-guide.updated.md", updated, 0o644); err != nil {
    return err
}
```

For tools that require atomic writes, backups, encoding preservation, authorization, or other filesystem policies, implement those policies outside Marksplice.

## 9. Create a new document

Existing-document editing and new-document creation are intentionally separate. Use `DocumentBuilder` when no original author formatting needs to be preserved:

```go
builder := marksplice.NewDocumentBuilder()
if err := builder.AppendHeadingContent(1, marksplice.TextInline("Release brief")); err != nil {
    return err
}
if err := builder.AppendParagraphContent(marksplice.TextInline("Ready for review.")); err != nil {
    return err
}

source, err := builder.Markdown()
```

The builder writes deterministic canonical GFM and validates the generated structure before returning it.

Run the larger example:

```sh
go run ./examples/build
```

It creates front matter, typed inline content, tasks, a table, and fenced shell commands.

## 10. Render HTML when you need an export

Rendering is separate from editing and construction. Once you have a parsed `Document`, stream a deterministic HTML fragment to any `io.Writer`:

```go
if err := doc.RenderHTML(os.Stdout, marksplice.DefaultHTMLRenderOptions()); err != nil {
    return err
}
```

Or collect caller-owned bytes:

```go
fragment, err := doc.HTML(marksplice.DefaultHTMLRenderOptions())
```

The default policy preserves parser-proven raw HTML, enables the GFM tag filter, and suppresses dangerous URL schemes. Preserved raw HTML is not a sanitizer; use `HTMLRawEscape` or an application-appropriate downstream sanitizer when untrusted Markdown crosses an HTML security boundary. Rendering performs no URL or asset fetching, command execution, syntax highlighting, or math-engine execution.

For a complete HTML document, use `RenderHTMLDocument` with `DefaultHTMLDocumentOptions`. The standalone zero value reuses the fragment safety defaults and maps only exact lower-case `title`, `description`, `author`, and `lang` fields when they are already unique top-level source-proven simple front-matter scalars. It does not parse arbitrary YAML/TOML; escape-dependent values are omitted rather than guessed, and `HTMLMetadataOmit` disables front-matter-derived metadata entirely.

When a preview or editor needs to correlate Markdown with rendered HTML, use `HTMLWithSourceMap` or `HTMLDocumentWithSourceMap` (or their streaming `Render...WithSourceMap` forms). Each `HTMLSourceMapEntry` carries a snapshot-local Markdown byte `Range` and a byte range in that exact output. The result is semantic-event granular rather than complete coverage, so nested ranges may overlap and synthetic HTML may be unmapped.

See [Render HTML](recipes/render-html.md) for fragment, standalone, metadata, safety, and source-map options. The tracked render example also supports `go run ./examples/render --map`.

## The five names you will see most often

| Name | Plain-language meaning |
| --- | --- |
| `Document` | Immutable parsed snapshot of one Markdown byte slice |
| `Node` | Small public summary for one promoted structural item |
| `NodeID` | Identity valid for that snapshot; use it to ask for typed detail or prepare an operation |
| `Range` | Exact byte span in that snapshot, with meaning defined by the accessor that returned it |
| `ChangeSet` | Prepared source-bound edit that can be applied only to the matching original bytes |

`DocumentBuilder` is the separate mutable value used to create new Markdown.

## Where to go next

- [User Guide](guide.md): choose a task and find the right API family.
- [Recipes](recipes/README.md): focused workflows for inspection, editing, creation, HTML rendering/source mapping, tables/lists/sections, filesystem workspaces, and extensions.
- [Examples](../examples/README.md): all runnable file-based programs.
- [API Reference](api-reference.md): exact signatures and exhaustive callable coverage.
- [Capabilities](capabilities.md): what is supported today and where Marksplice intentionally stops.

For parser architecture, conformance policy, and engineering history, use the [advanced/maintainer documentation map](README.md#advanced-and-maintainer-documentation). Those documents are not required for normal library use.
