# Lists, Sections, and Tables

These families expose hierarchy-aware read APIs and conservative structural mutation. Use them when a local text replacement is not enough.

The runnable [`query` example](../../examples/query/) demonstrates section-scoped task selection over a real Markdown file:

```sh
go run ./examples/query
```

## Sections

Sections are derived from promoted document headings. The heading `NodeID` is the section identity.

```go
for _, section := range doc.Sections() {
    fmt.Printf("level=%d range=%v body=%v\n",
        section.Level(), section.Range(), section.BodyRange())
}
```

Use:

- `Sections()` for source-ordered sections;
- `Section(headingID)` for one section;
- `SectionChildHeadingIDs(headingID)` for immediate child headings;
- `QuerySections` for bounded level/range selection.

`Section.Range()` is the complete governed subtree. `BodyRange()` stops before the first child section.

Structural operations are explicit:

- `PrepareReplaceSectionBody`
- `PrepareReplaceSection`
- `PrepareRemoveSection`
- `PrepareInsertSectionBefore` / `PrepareInsertSectionAfter`
- `PrepareAppendSectionChild`
- `PrepareMoveSectionBefore` / `PrepareMoveSectionAfter`

Insert/replace fragments must satisfy the requested hierarchy when reparsed in the host document. Moves require compatible section levels.

## List items and tasks

`ListItem` exposes the item content boundary plus reviewed hierarchy facts such as parent, direct children, and complete subtree range when available.

A structural operation requires a complete supported subtree. If an item contains descendants Marksplice cannot safely promote into that structural model, the subtree authority is withheld rather than guessed.

Available structural operations include:

- `PrepareReplaceListItem`
- `PrepareReplaceListItemSubtree`
- `PrepareRemoveListItem`
- `PrepareInsertListItemBefore` / `PrepareInsertListItemAfter`
- `PrepareAppendListItemChild`
- `PrepareMoveListItemBefore` / `PrepareMoveListItemAfter`

Task state is narrower and independent:

```go
change, err := doc.PrepareSetTaskChecked(task.ID(), true)
```

That operation changes only the proven task-state byte, not the list item text or marker style.

Marksplice does not silently reindent arbitrary fragments. Caller-provided list fragments must fit the requested sibling/parent/container shape after host-context validation.

## Query tasks inside one section

Use a section's complete range as `Within`:

```go
within := section.Range()
matches, err := doc.QueryNodes(marksplice.NodeQuery{
    Kinds:  []marksplice.Kind{marksplice.KindTask},
    Within: &within,
    Limit:  100,
})
```

The query is bounded and remains in source order. See [`examples/query/main.go`](../../examples/query/main.go) for the complete version.

## Tables

`Table`, `TableRow`, and `TableCell` separate semantic ownership from source spans.

Typical read APIs include:

- `TableAlignments(tableID)`
- `TableHeaderCellIDs(tableID)`
- `TableRowIDs(tableID)`
- `TableRowCellIDs(rowID)` and `TableRowHeaderCellIDs(rowID)`
- `TableRowAlignments(rowID)`
- `TableCell.TableID()`, `RowID()`, and `Column()`
- `TableRow.TableID()`, `PreviousID()`, and `NextID()`

A `TableCell.Range()` is cell content only. Pipes, padding, neighboring cells, and line endings remain outside that range.

### Cell and row operations

- `PrepareReplaceTableCell`
- `PrepareReplaceTableRow`
- `PrepareRemoveTableRow`
- `PrepareInsertTableRowBefore` / `PrepareInsertTableRowAfter`
- `PrepareAppendTableRow`
- `PrepareMoveTableRowBefore` / `PrepareMoveTableRowAfter`

Rows supplied by the caller must be compatible with the target table width and host structure.

### Alignment operations

```go
change, err := doc.PrepareSetTableColumnAlignment(
    table.ID(),
    1,
    marksplice.TableAlignmentRight,
)
```

Use `PrepareSetTableAlignments` to update the complete alignment vector atomically.

Existing-source alignment edits alter only source-proven delimiter syntax while preserving surrounding table trivia.

### Column operations

- `PrepareInsertTableColumn`
- `PrepareRemoveTableColumn`
- `PrepareMoveTableColumn`

These require the strictest table mapping: header, delimiter, and every semantic body row must be safely mappable at the table width. If that proof is incomplete, the operation fails instead of fabricating cells or rewriting the table canonically.

## Creating lists and tables

For new documents, `DocumentBuilder` can emit flat and reviewed homogeneous nested ordered/unordered lists and tasks, plus canonical tables with optional alignments and zero or more body rows. New-document construction is canonical; it is not the path for editing an existing table/list while preserving its style.

For exact methods and signatures, use the [API Reference](../api-reference.md#lists-sections-and-tables).
