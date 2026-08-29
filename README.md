# Marksplice

[![Go Reference](https://pkg.go.dev/badge/github.com/zoster81/marksplice.svg)](https://pkg.go.dev/github.com/zoster81/marksplice)
[![CI](https://github.com/zoster81/marksplice/actions/workflows/ci.yml/badge.svg)](https://github.com/zoster81/marksplice/actions/workflows/ci.yml)

Marksplice is a **Pure-Go, source-preserving Markdown document engine** for Go — built for editors, developer tooling, and AI agents that need to understand and modify Markdown without rewriting it.

An ordinary AST parser tells you what Markdown means. Marksplice also proves **which source bytes a structural operation owns**, so a change can stay local, preserve author formatting, and fail closed when the source has changed.

## Why use Marksplice?

- **Edit existing Markdown without a full rewrite.** Rename a heading, check a task, update a table cell, move a section, and keep unrelated bytes untouched.
- **Work with structure instead of string searches.** Inspect headings, sections, lists, tasks, tables, fenced blocks, links, fragments, footnotes, front matter, and more.
- **Create Markdown from structured Go values.** `DocumentBuilder` writes deterministic GFM for new documents.
- **Render deterministic HTML on demand.** Stream fragments with `RenderHTML`, complete documents with `RenderHTMLDocument`, or request optional source-to-output byte maps for preview/editor tooling; all reuse Native semantics instead of reparsing Markdown.
- **Understand documentation sets.** Use the read-only `workspacefs` adapter over your own `fs.FS`, or supply documents directly; then inspect backlinks, validate links/fragments, and plan conservative repairs.
- **Give tools and AI agents a safer editing surface.** Use bounded structural queries, snapshot-local identities, exact source ranges, typed operations, and source-bound `ChangeSet`s instead of fragile whole-file text rewrites.

## Engineering facts

| Property | Current Marksplice contract |
| --- | --- |
| Markdown engine | Marksplice-owned Native parser; CommonMark 0.31.2 base plus reviewed GFM behavior |
| Conformance evidence | 652 CommonMark + 676 parser-applicable GFM parser contracts; profile-aware HTML checked against all 652 CommonMark and 677 published-GFM examples |
| Source safety | Exact byte ranges, immutable snapshots, stale-source detection, minimal operation-owned patches |
| Real-world validation | Byte-certified corpus of 6,857 Markdown documents, 60.8 MB, from 195 open-source repositories |
| Measured parse performance | v0.5 engineering freeze: 25.06 MB/s public `Parse`, 30.87 MB/s Native on the same preloaded 60.8 MB corpus |
| Robustness | Focused/pathological tests, fuzz targets, race testing, static analysis, and cross-platform builds |
| Portability | Pure Go; Go 1.26+; no third-party Markdown parser dependency |
| Dependencies | One direct dependency: `golang.org/x/text`, used for full Unicode GFM reference-label folding |
| Authority boundary | No hidden filesystem traversal, URL fetching, network access, or command execution in the document core |

Performance is measured on real documents as well as focused benchmarks; correctness and source preservation are not traded away to win a parser-only microbenchmark. On the same-host v0.5 campaign, public `Parse` improved from 15.04 to **25.06 MB/s** while allocated bytes fell from about 4.49 GB/op to **2.70 GB/op**. These are engineering benchmark results for that corpus/host, not cross-machine guarantees.

## Built for tools and AI agents

Marksplice turns a document-editing workflow into a small structural protocol:

```text
bounded query -> exact target -> typed change -> optional atomic composition -> apply to exact source
```

An agent can ask for a limited set of sections or nodes, prepare a structural change, and apply it without regenerating the document. If another actor changed the bytes in the meantime, the prepared change fails with `ErrSourceConflict` instead of guessing. Graph, backlink, fragment, and workspace APIs provide the same explicit model across documentation sets. When filesystem discovery is useful, `workspacefs` requires an explicit caller-supplied `fs.FS` plus finite resource limits.

The document core does not crawl files or fetch URLs. `workspacefs` performs only explicit read-only Markdown discovery/following inside the `fs.FS` authority supplied by the caller. HTML rendering is pure in-memory/output-writer work: fragment and standalone paths do not fetch linked assets, execute commands, run templates, run syntax highlighting or a math engine, or silently normalize existing documents. PDF remains outside the current capability set.

## Install

Marksplice requires Go 1.26 or newer. The current published beta is `v0.5.0-beta.1`:

```sh
go get github.com/zoster81/marksplice@v0.5.0-beta.1
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
| Render | deterministic HTML fragments or standalone documents, with explicit raw-HTML, unsafe-URL, tag-filter, metadata policy, and optional source-to-output byte mapping |
| Navigate | anchors, fragments, TOCs, link relationships |
| Work across documents | explicit `fs.FS` scan/follow, document graphs, backlinks, reachability, workspace validation, knowledge metadata |
| Extend read-only semantics | opt-in namespaced observations through `ParseWithOptions` |

See the concise [capability matrix](docs/capabilities.md) for current boundaries and unsupported behavior.

## Start here

- **New to Marksplice?** Follow [Getting Started](docs/getting-started.md).
- **Looking for a task?** Use the [User Guide](docs/guide.md) or [Recipes](docs/recipes/README.md).
- **Want runnable programs?** Browse [`examples/`](examples/README.md).
- **Need an exact signature?** Open the [API Reference](docs/api-reference.md).
- **Maintaining or contributing?** See the [documentation map](docs/README.md), [Contributing](CONTRIBUTING.md), and advanced architecture/conformance material from there.

## Status

Marksplice is beta software under active development. Until v1, public APIs may change between releases. The current published beta remains `v0.5.0-beta.1`; newer APIs documented on `main`, including the filesystem workspace foundation and HTML/source-mapping path, are unreleased until a later tag is cut. The production parser is Marksplice's native CommonMark/GFM implementation; ordinary users do not need parser internals to use the public API.

## License

Apache License 2.0. See [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE).

Marksplice was created by Giovanni Riccobene (`zoster81`).
