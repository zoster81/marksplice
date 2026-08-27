# Marksplice User Guide

Use this page to choose the shortest path to the task you have. If this is your first time using Marksplice, start with [Getting Started](getting-started.md).

## I want to...

| Goal | Start here | Runnable example |
| --- | --- | --- |
| Load a Markdown file and inspect its structure | [Inspect a document](recipes/inspect-document.md) | `go run ./examples/inspect` |
| Rename, replace, check, remove, insert, move, or combine edits | [Edit an existing document](recipes/edit-existing-document.md) | `go run ./examples/edit` |
| Create Markdown from structured Go values | [Create a document](recipes/create-document.md) | `go run ./examples/build` |
| Work with list hierarchies, sections, or GFM tables | [Lists, sections, and tables](recipes/lists-sections-tables.md) | `go run ./examples/query` |
| Resolve fragments, inspect links, build backlinks, or validate a document set | [Links and workspaces](recipes/links-workspaces.md) | `go run ./examples/workspace` |
| Observe application-specific syntax without changing core GFM | [Read-only extensions](recipes/extensions.md) | `go run ./examples/extensions` |
| Check whether a feature is supported | [Capability matrix](capabilities.md) | — |
| Find an exact function or method signature | [API Reference](api-reference.md) | — |

## The model in one minute

Marksplice deliberately separates two jobs.

### Existing Markdown: `Parse` → inspect → `Prepare...` → `Apply`

A parsed `Document` is immutable and owns an exact source snapshot. Existing-document operations prepare narrow `ChangeSet` values against that snapshot. Applying a change to different bytes fails with `ErrSourceConflict`.

This is the path to use when author formatting matters.

### New Markdown: `DocumentBuilder` → `Markdown`

A `DocumentBuilder` represents construction intent. It writes deterministic canonical GFM because there is no existing author source to preserve.

Do not use the builder to round-trip an existing document when your goal is a small edit.

## Reading a parsed document

The public API exposes several levels of detail:

- `Nodes()` gives source-ordered promoted structural summaries.
- Typed accessors such as `Heading`, `Task`, `TableCell`, `FencedCode`, `InlineLink`, and `ReferenceDefinition` expose operation-specific detail.
- Higher-level views such as `Sections`, `FencedBlocks`, `Alerts`, `MathExpressions`, `FootnoteDefinitions`, and `FrontMatter` expose reviewed semantics that do not always imply mutation authority.
- `QueryNodes` and `QuerySections` provide bounded structural selection.
- `HeadingAnchors`, `ResolveFragment`, and `LinkRelationships` provide navigation and relationship intelligence.

A public `Range` always means exactly what the accessor documents. Marksplice intentionally does not define one universal "full node range" for every construct.

## Editing existing source

Most mutation APIs are named `Prepare...`. Typical families include:

- paragraph, heading, task, fenced-code, inline, link/image/autolink, reference-definition, front-matter, HTML, footnote, math, and table-cell replacements;
- section replacement/removal/insertion/movement/child append;
- list-item content/subtree replacement, removal, sibling insertion/movement, and child append;
- table row, alignment, and complete-column operations;
- thematic-break and complete-blockquote removal;
- managed TOC synchronization;
- `ComposeChanges` for independent operations prepared from the same snapshot.

The exact supported shapes are deliberately conservative. If Marksplice cannot prove the source ownership or surviving structure required by an operation, it returns an error instead of rewriting a wider region.

See [Edit an existing document](recipes/edit-existing-document.md).

## Creating new Markdown

`DocumentBuilder` supports reviewed GFM construction for common document families, including:

- headings and paragraphs;
- typed inline text, code, emphasis, strong, strikethrough, links, images, autolinks, references, footnote references, and reviewed math forms;
- ordered/unordered lists and task lists, including reviewed homogeneous nesting;
- tables and alignments;
- fenced code;
- blockquotes and GitHub alerts;
- reference and footnote definitions;
- YAML/TOML front-matter envelopes;
- thematic breaks and mathematical blocks.

Generated content is reparsed and checked against construction expectations before `Markdown()` returns it.

See [Create a document](recipes/create-document.md).

## Navigation and multi-document work

Marksplice can understand document relationships without owning your filesystem or URL policy.

For one document:

- derive GitHub-compatible heading anchors;
- resolve local fragments;
- generate and conservatively synchronize managed TOCs;
- enumerate semantic link/image/autolink relationships.

For several documents that your application already loaded:

- `BuildDocumentGraph` creates an immutable graph over explicit caller keys;
- a caller resolver decides which non-local relationships map to which already-supplied documents;
- graph queries expose outgoing edges, backlinks, reachability, and related documents;
- `ValidateWorkspace` adds deterministic link/fragment/reference/orphan/managed-TOC diagnostics and conservative repair planning;
- `BuildKnowledgeIndex` adds caller-declared aliases, tags, and logical references without inventing Markdown syntax.

See [Links and workspaces](recipes/links-workspaces.md).

## Extensions

`ParseWithOptions` can attach namespaced read-only observations produced by caller-linked recognizers. Extension nodes cannot replace core GFM nodes or gain generic editing, construction, graph, filesystem, network, or command authority.

Use this for product-specific syntax such as a private `[[wikilink]]` convention when observation is enough. See [Read-only extensions](recipes/extensions.md).

## Errors

Use `errors.Is` with public sentinel families rather than comparing messages:

- `ErrNodeNotFound`
- `ErrInvalidReplacement`
- `ErrInvalidTargetKind`
- `ErrSourceConflict`
- `ErrInvalidConstruction`
- `ErrInvalidQuery`
- `ErrInvalidGraph`
- `ErrInvalidWorkspace`
- `ErrInvalidKnowledge`
- `ErrInvalidExtension`

Diagnostic strings are not compatibility contracts.

## Concurrency and ownership

Successfully built immutable `Document`, `DocumentGraph`, `KnowledgeIndex`, `WorkspaceReport`, and prepared `ChangeSet` values may be read concurrently.

`DocumentBuilder` is mutable and requires caller synchronization for concurrent use. Resolver and extension callbacks are invoked synchronously and are not retained after the build/parse call returns.

Public variable-length results are caller-owned unless an API explicitly states otherwise.

## What Marksplice does not own

Marksplice performs no implicit filesystem, network, or command I/O. It does not render HTML/PDF, execute fenced languages, serialize arbitrary YAML/TOML, render LaTeX, or normalize an existing document as a side effect of a structural edit.

Those boundaries are summarized in [Capabilities](capabilities.md). Architecture and conformance rationale live in the [advanced documentation](README.md#advanced-and-maintainer-documentation).
