# Marksplice

[![Go Reference](https://pkg.go.dev/badge/github.com/zoster81/marksplice.svg)](https://pkg.go.dev/github.com/zoster81/marksplice)
[![CI](https://github.com/zoster81/marksplice/actions/workflows/ci.yml/badge.svg)](https://github.com/zoster81/marksplice/actions/workflows/ci.yml)

Marksplice is a Pure-Go library for reading, creating, querying, and **source-preservingly editing** Markdown.

Use it when you need to change Markdown structurally without reformatting the rest of the file.

## Why use Marksplice?

- **Edit existing Markdown without a full rewrite.** Rename a heading, check a task, update a table cell, move a section, and keep unrelated bytes untouched.
- **Work with structure instead of string searches.** Inspect headings, sections, lists, tasks, tables, fenced blocks, links, fragments, footnotes, front matter, and more.
- **Create Markdown from structured Go values.** `DocumentBuilder` writes deterministic GFM for new documents.
- **Understand documentation sets.** Build caller-controlled document graphs, inspect backlinks, validate links/fragments, and plan conservative repairs.

Marksplice does not crawl your filesystem, fetch URLs, render HTML/PDF, or silently normalize existing documents.

## Install

Marksplice requires Go 1.26 or newer. The current published beta is `v0.1.0-beta.2`:

```sh
go get github.com/zoster81/marksplice@v0.1.0-beta.2
```

## Try a real file

Clone the repository and run the inspection example:

```sh
go run ./examples/inspect
```

It loads [`examples/inspect/project-guide.md`](examples/inspect/project-guide.md) from disk and reports its sections, tasks, fenced blocks, and links.

For a source-preserving edit over another tracked Markdown file:

```sh
go run ./examples/edit
```

That example prepares several changes, combines them atomically, applies them to the original bytes, and **does not overwrite the fixture**.

## Minimal edit flow

```go
source, err := os.ReadFile("README.md")
if err != nil {
    return err
}

doc, err := marksplice.Parse(source)
if err != nil {
    return err
}

// Select a heading ID from doc.Nodes(), QueryNodes(), or another typed view.
change, err := doc.PrepareRenameHeading(headingID, []byte("New title"))
if err != nil {
    return err
}

updated, err := change.Apply(source) // apply to the exact bytes that were parsed
if err != nil {
    return err
}
```

A prepared `ChangeSet` is bound to the parsed snapshot. Applying it to different bytes fails closed with `ErrSourceConflict` rather than guessing where the edit belongs.

## What can it do?

| Goal | Examples |
| --- | --- |
| Inspect | headings, sections, lists/tasks, tables, fenced blocks, links, front matter, alerts, footnotes, math |
| Query | bounded source-ordered node and section queries |
| Edit | content replacements plus structural section/list/table operations and atomic change composition |
| Create | headings, paragraphs, lists/tasks, tables, fenced code, front matter, blockquotes/alerts, typed inline content |
| Navigate | anchors, fragments, TOCs, link relationships |
| Work across documents | explicit document graphs, backlinks, reachability, workspace validation, knowledge metadata |
| Extend read-only semantics | opt-in namespaced observations through `ParseWithOptions` |

See the concise [capability matrix](docs/capabilities.md) for current boundaries and unsupported behavior.

## Start here

- **New to Marksplice?** Follow [Getting Started](docs/getting-started.md).
- **Looking for a task?** Use the [User Guide](docs/guide.md) or [Recipes](docs/recipes/README.md).
- **Want runnable programs?** Browse [`examples/`](examples/README.md).
- **Need an exact signature?** Open the [API Reference](docs/api-reference.md).
- **Maintaining or contributing?** See the [documentation map](docs/README.md), [Contributing](CONTRIBUTING.md), and advanced architecture/conformance material from there.

## Status

Marksplice is beta software under active development. Until v1, public APIs may change between releases. The production parser is Marksplice's native CommonMark/GFM implementation; ordinary users do not need parser internals to use the public API.

## License

Apache License 2.0. See [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE).

Marksplice was created by Giovanni Riccobene (`zoster81`).
