# Create a Document

Use `DocumentBuilder` when you are creating new Markdown and there is no original author formatting to preserve.

Run the complete example:

```sh
go run ./examples/build
```

The example creates a release brief with front matter, typed inline content, a task list, a table, and fenced shell commands.

## Start with a builder

```go
builder := marksplice.NewDocumentBuilder()
```

The zero value is also usable, but the constructor makes intent obvious.

## Prefer typed inline content for semantic text

```go
err := builder.AppendParagraphContent(
    marksplice.TextInline("Generated with "),
    marksplice.LinkInline(
        "https://github.com/zoster81/marksplice",
        marksplice.TextInline("Marksplice"),
    ),
    marksplice.TextInline("."),
)
```

`TextInline` escapes Markdown punctuation so caller text does not accidentally become syntax. Typed constructors are available for reviewed code, emphasis, strong, strikethrough, links, images, autolinks, references, footnote references, and mathematical forms.

Raw-GFM builder entrypoints remain useful when the caller intentionally owns already-reviewed inline GFM.

## Add document blocks

Common builder methods include:

- `AppendHeading` / `AppendHeadingContent`
- `AppendParagraph` / `AppendParagraphContent`
- `AppendThematicBreak`
- `AppendFencedCode`
- ordered/unordered lists and task lists
- homogeneous nested lists/tasks
- blockquotes and GitHub alerts
- tables with optional alignments and optional body rows
- reference definitions and footnote definitions
- mathematical blocks
- YAML/TOML front matter through `SetYAMLFrontMatter` / `SetTOMLFrontMatter`

Example task list:

```go
err := builder.AppendUnorderedTaskList(
    marksplice.TaskListItem{InlineGFM: "Review API changes", Checked: true},
    marksplice.TaskListItem{InlineGFM: "Publish migration notes", Checked: false},
)
```

Example table:

```go
err := builder.AppendTableWithAlignments(
    []string{"Area", "Owner", "Status"},
    []marksplice.TableAlignment{
        marksplice.TableAlignmentLeft,
        marksplice.TableAlignmentLeft,
        marksplice.TableAlignmentCenter,
    },
    []string{"Docs", "Team", "Ready"},
)
```

## Front matter is separate document state

```go
err := builder.SetYAMLFrontMatter(
    marksplice.FrontMatterFieldInput{Key: "project", Value: "marksplice"},
    marksplice.FrontMatterFieldInput{Key: "status", Value: "draft"},
)
```

Construction intentionally supports a conservative string-field subset. Marksplice is not a general YAML/TOML serializer.

## Render only after construction succeeds

```go
source, err := builder.Markdown()
if err != nil {
    return err
}
```

The builder emits canonical LF GFM and validates generated structure. `ErrInvalidConstruction` means the requested structured intent could not be proven through the reviewed construction contract.

## Builder output is canonical by design

Source preservation applies to editing **existing** documents. A new document has no author trivia to preserve, so the builder can make deterministic choices such as:

- LF line endings;
- canonical block spacing;
- canonical task markers;
- canonical table formatting;
- adaptive code fences where needed.

Do not use a builder as a formatter for an existing document when your goal is a local edit. Parse the existing bytes and use a `Prepare...` operation instead.

## References and forward references

For ordinary reference links/images, Marksplice distinguishes definitions that already exist in builder state from definitions explicitly deferred for forward-reference construction. Dedicated constructors make that intent explicit.

Footnotes have a similar immediate/deferred construction flow.

This prevents construction from silently depending on unresolved or ambiguous labels.

For the exact builder and inline constructor surface, use the [API Reference](../api-reference.md#create-a-new-document).
