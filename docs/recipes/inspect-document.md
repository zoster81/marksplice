# Inspect a Document

Use this workflow when you want structured facts from Markdown without changing it.

Run the complete file-based example first:

```sh
go run ./examples/inspect
```

It reads [`../../examples/inspect/project-guide.md`](../../examples/inspect/project-guide.md), not an embedded string.

## Parse once

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

`Document` is immutable. Reuse it for several reads instead of reparsing for each question.

## Enumerate promoted nodes

`Nodes()` is the general source-ordered entrypoint:

```go
for _, node := range doc.Nodes() {
    switch node.Kind() {
    case marksplice.KindHeading:
        heading, _ := doc.Heading(node.ID())
        text, _ := doc.SourceRange(heading.Range())
        fmt.Printf("h%d %s\n", heading.Level(), text)
    case marksplice.KindTask:
        task, _ := doc.Task(node.ID())
        fmt.Printf("checked=%t\n", task.Checked())
    }
}
```

The typed accessor is important: a generic `Node` is intentionally small, while `Heading`, `Task`, `TableCell`, and other typed values expose the facts that are valid for that family.

## Use higher-level views when the question is higher-level

You do not always need `Nodes()`.

```go
sections := doc.Sections()
fences := doc.FencedBlocks()
links := doc.LinkRelationships()
frontMatter, hasFrontMatter := doc.FrontMatter()
```

Useful higher-level APIs include:

- `Sections()` and `SectionChildHeadingIDs()` for heading hierarchy;
- `FencedBlocks()` and `FencedBlockContentRanges()` for complete fenced containers;
- `Alerts()` and `AlertBodyRanges()` for GitHub alert semantics;
- `FootnoteDefinitions()`, `FootnoteReferences()`, and definition body ranges;
- `MathExpressions()` and payload ranges;
- `FrontMatter()` for a recognized YAML/TOML envelope;
- `HeadingAnchors()` and `ResolveFragment()` for navigation;
- `LinkRelationships()` for semantic outgoing relationships.

Broader read access does not automatically mean broader edit access. For example, `FencedBlock` can describe empty or non-contiguous bodies that the narrower `FencedCode` replacement contract does not authorize for generic payload editing.

## Read source through ranges

A `Range` is a byte span in the exact parsed snapshot:

```go
raw, ok := doc.SourceRange(heading.Range())
if !ok {
    return errors.New("range is outside the snapshot")
}
```

The accessor defines what that range means. `Heading.Range()` is heading content; `Task.Range()` is the one-byte task state; `TableCell.Range()` is cell content. Do not assume every range is a complete syntactic container.

## Use bounded queries for selection

When you need only a subset, prefer `QueryNodes` or `QuerySections`:

```go
matches, err := doc.QueryNodes(marksplice.NodeQuery{
    Kinds: []marksplice.Kind{
        marksplice.KindHeading,
        marksplice.KindTask,
    },
    Limit: 100,
})
```

Queries preserve source order and require a positive caller-owned limit. `Within` can restrict results to an existing source range.

See [`examples/query`](../../examples/query/) for a section-scoped task query.

## Need an exact accessor?

Use the [API Reference](../api-reference.md#read-and-inspect-a-document) for the domain index, then jump to the receiver/type you need.
