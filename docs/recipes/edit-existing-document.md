# Edit an Existing Document

Use this workflow when you want to change selected Markdown structure while preserving unrelated source bytes.

Run the complete example:

```sh
go run ./examples/edit
```

It reads [`../../examples/edit/release-plan.md`](../../examples/edit/release-plan.md), prepares a heading rename, paragraph replacement, task-state change, and table-cell replacement, composes them atomically, and leaves the fixture untouched.

## The edit contract

Every ordinary edit follows the same pattern:

1. load exact bytes;
2. `Parse` those bytes;
3. select a promoted target;
4. call the specific `Prepare...` operation;
5. apply the returned `ChangeSet` to the same bytes.

```go
change, err := doc.PrepareRenameHeading(headingID, []byte("Release Readiness"))
if err != nil {
    return err
}

updated, err := change.Apply(source)
if err != nil {
    return err
}
```

`Apply` verifies the source snapshot. If the supplied bytes differ from what was parsed, it reports `ErrSourceConflict`.

## Choose the narrowest operation

Prefer the operation that matches the intent instead of replacing a wider container.

Common scalar/content edits include:

- `PrepareRenameHeading`
- `PrepareReplaceParagraph`
- `PrepareSetTaskChecked`
- `PrepareReplaceFencedCode`
- `PrepareReplaceCodeSpan`, `PrepareReplaceEmphasis`, `PrepareReplaceStrong`, `PrepareReplaceStrikethrough`
- `PrepareReplaceInlineLinkDestination`, `PrepareReplaceImageDestination`, `PrepareReplaceAutoLink`
- `PrepareReplaceReferenceDefinitionDestination`, `PrepareReplaceReferenceDefinitionTitle`
- `PrepareReplaceFrontMatterValue`
- `PrepareReplaceHTMLComment`, `PrepareReplaceHTMLAnchor`
- `PrepareReplaceFootnoteDefinitionBody`, `PrepareRenameFootnote`
- `PrepareReplaceMathExpression`
- `PrepareReplaceTableCell`

Marksplice keeps syntax outside the operation-owned range untouched. A heading rename, for example, does not replace the whole heading line just to change its text.

## Combine independent edits

If several changes were prepared from the same `Document`, combine them before applying:

```go
combined, err := doc.ComposeChanges(
    rename,
    replaceParagraph,
    checkTask,
    updateCell,
)
if err != nil {
    return err
}

updated, err := combined.Apply(source)
```

Composition is conservative. Byte overlap, logical overlap, or a combined Markdown interpretation that differs from the independently validated operations fails closed.

Use a family-specific atomic operation instead when several source changes intentionally represent one logical edit, such as a coordinated footnote rename.

## Structural edits

Marksplice also supports reviewed structural operations where complete ownership can be proven.

### Sections

- replace a direct section body;
- replace or remove a complete section subtree;
- insert sibling sections;
- append a direct child section;
- move same-level section subtrees.

### List items

- replace item content or a complete supported subtree;
- remove a complete subtree;
- insert siblings;
- append a direct child subtree;
- move supported subtrees.

### Tables

- replace/insert/append/remove/move body rows;
- change one or all column alignments;
- insert/remove/move complete columns when the full table mapping is proven.

See [Lists, sections, and tables](lists-sections-tables.md) for selection and structural boundaries.

## Why an operation can fail on valid Markdown

Valid Markdown is not automatically safe to edit through every operation. Marksplice requires enough source ownership and semantic proof to know exactly what the requested edit means and what must survive around it.

Examples of conservative failure include:

- a replacement would create a different Markdown construct at a boundary;
- a list subtree contains descendants outside the reviewed structural model;
- a table column operation cannot map every semantic row safely;
- a reference-definition removal would change surviving reference resolution;
- two otherwise valid prepared changes interact when combined.

Treat these failures as a signal to choose a different explicit operation or let the user/application resolve the ambiguous source. Do not fall back to silent whole-document normalization if source preservation matters.

## Writing the result is your responsibility

Marksplice returns updated bytes. It does not choose paths, permissions, backup policy, encoding policy, or atomic-file-replacement behavior.

For exact mutation signatures, use the [API Reference](../api-reference.md#edit-an-existing-document).
