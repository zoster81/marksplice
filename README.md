# Marksplice

[![Go Reference](https://pkg.go.dev/badge/github.com/zoster81/marksplice.svg)](https://pkg.go.dev/github.com/zoster81/marksplice)
[![CI](https://github.com/zoster81/marksplice/actions/workflows/ci.yml/badge.svg)](https://github.com/zoster81/marksplice/actions/workflows/ci.yml)

Structured GitHub Flavored Markdown creation and source-preserving manipulation for Go.

Marksplice is an open-source Pure-Go library for understanding, creating, and structurally editing GitHub Flavored Markdown (GFM). New documents may be generated from reviewed structured intent, while edits to existing documents preserve untouched source bytes whenever the requested operation does not semantically require broader changes.

## Status

Marksplice is currently **beta software under active development**. The first public module release is planned as `v0.1.0-beta.1`; until v1, public APIs and behavior may change incompatibly between releases.

The repository has a green retrospective M0 bootstrap record and completed engineering milestones M1–M91. The current model has two deliberately separate paths:

- parsed `Document` snapshots expose reviewed source-mapped read/edit capabilities and prepare minimal source-bound changes that reject stale input;
- `DocumentBuilder` creates new deterministic GFM and validates generated structure through the same parser/source-model boundary before returning bytes.

The current public surface covers reviewed paragraphs/headings/sections, supported list and task hierarchies, fenced code, GFM tables with public table/row/cell ownership and conservative row/alignment/column structural edits, simple inline spans/links/images/reference definitions/autolinks, source-proven top-level thematic breaks and simple one-line existing-source blockquotes, unique simple YAML/TOML front-matter fields with canonical new-document envelope construction, simple HTML comments/anchors, typed inline construction for semantic text/code/emphasis/strong/strikethrough/links/images/autolinks including conservative canonical link/image titles, bounded reviewed emphasis-family nesting, and full reference-link/reference-image construction against an already-present exact reference definition, canonical single-paragraph blockquote construction at depth 1 or explicit nesting depth 2–64, and construction-only multi-block blockquotes composed from reviewed builder children including recursively nested blockquotes with total depth bounded at 64.

The public API remains intentionally narrower than everything the semantic parser can recognize. Unsupported or ambiguous shapes are preserved or kept internal until exact source ownership and caller-facing semantics are proven.

See [`docs/README.md`](docs/README.md) for the documentation map and repository layout, [`docs/capabilities.md`](docs/capabilities.md) for the authoritative current read/edit/create matrix and roadmap, and [`docs/milestones/`](docs/milestones/) for detailed milestone contracts and historical verification evidence.

## Design principles

- follow the published GitHub Flavored Markdown 0.29 specification as the single normative Markdown syntax profile;
- create new GFM through deterministic reviewed construction rules and parser/model proof;
- parse existing GFM for semantic understanding without implying whole-document normalization;
- preserve untouched author choices such as heading/list/fence styles, whitespace, delimiters, numbering, and line endings during existing-document edits;
- bind prepared edits to exact source snapshots and reject stale application;
- promote public capabilities only after operation-oriented source ownership is proven;
- keep Goldmark behind an internal adapter and expose only Marksplice-owned types;
- keep filesystem, network, command-execution, and host authorization concerns outside the core library.

## Installation

Marksplice requires Go 1.26 or newer. After the first public beta tag is published, install it explicitly because Go does not prefer pre-release versions by default:

```text
go get github.com/zoster81/marksplice@v0.1.0-beta.1
```

A consuming `go.mod` may instead contain:

```text
require github.com/zoster81/marksplice v0.1.0-beta.1
```

The module path remains `github.com/zoster81/marksplice` throughout v0 and v1.

## Quick start

```go
builder := marksplice.NewDocumentBuilder()
_ = builder.AppendHeadingContent(1, marksplice.TextInline("Marksplice"))
_ = builder.AppendParagraphContent(marksplice.TextInline("Source preserving GFM"))
source, err := builder.Markdown()
```

Executable examples for construction, parsing, and source-preserving heading mutation live in `example_test.go` and are published by pkg.go.dev.

## Construction and editing

`DocumentBuilder` writes canonical LF GFM for reviewed new-document families such as headings, parser-proven paragraphs, lists/tasks including homogeneous nesting, fenced code, reference definitions, tables with optional alignment, thematic breaks, and blockquotes. `AppendBlockquote` owns one paragraph at depth 1, `AppendNestedBlockquote` accepts one paragraph at explicit depths 2–64, and `AppendBlockquoteBlocks` snapshots another builder's reviewed body blocks and quotes that sequence at depth 1–64. Multi-block composition accepts every reviewed body-block construction family, including recursive blockquote children when total structural depth stays at most 64; front matter remains excluded because it is a document envelope. It can also own one document-leading canonical YAML or TOML front-matter envelope with conservative double-quoted string fields. Typed-inline entrypoints provide a semantic-text alternative to the historical raw-GFM block APIs, including `AppendNestedBlockquoteContent`; `LinkInlineWithTitle` and `ImageInlineWithTitle` add conservative canonical double-quoted titles. M88 allows bounded nesting of `CodeInline`, `EmphasisInline`, `StrongInline`, and `StrikethroughInline` inside emphasis/strong/strikethrough wrappers, while ambiguous GFM delimiter combinations fail closed. M89 adds `ReferenceLinkInline` and `ReferenceImageInline` for canonical full-reference forms whose exact label must match exactly one top-level reference definition already present in the same builder; collapsed/shortcut forms and forward definitions remain outside this slice. Existing parsed-source editing contracts remain unchanged.

Parsed `Document` values instead retain exact immutable source. Mutations target operation-specific ranges and validate the candidate source when surrounding Markdown interpretation could change. Structural operations never use the construction writer to reformat an existing document.

## Documentation

- [`docs/README.md`](docs/README.md): documentation index and repository-layout map.
- [`docs/capabilities.md`](docs/capabilities.md): current product-facing read/edit/create matrix and forward roadmap.
- [`docs/architecture.md`](docs/architecture.md): durable architecture, source-preservation, performance, safety, and complexity decisions.
- [`docs/gfm-conformance.md`](docs/gfm-conformance.md): normative GFM profile, pinned conformance source hierarchy, and update procedure.
- [`docs/goldmark-capability-matrix.md`](docs/goldmark-capability-matrix.md): Goldmark-versus-Marksplice responsibility boundary.
- [`docs/milestones/`](docs/milestones/): feature-specific historical contracts, design records, tests, and exit decisions.
- [`CONTRIBUTING.md`](CONTRIBUTING.md): contributor workflow and current verification commands.
- [`docs/releasing.md`](docs/releasing.md): public beta versioning, first-push readiness, and Go-module publication procedure.
- [`SECURITY.md`](SECURITY.md): private vulnerability-reporting policy.
- [`CHANGELOG.md`](CHANGELOG.md): public release notes and beta history.

## Development

The module path is:

```text
github.com/zoster81/marksplice
```

The `go 1.26` directive is the current minimum compatibility floor. Public CI exercises Go 1.26 and Go 1.27 on Linux, Windows, and macOS.

The public `marksplice` package intentionally lives at the module root so consumers import exactly `github.com/zoster81/marksplice`. Root source files are grouped into `api*` and `builder*` families; black-box consumer-style API tests live under `internal/publictest/`, and longer-form project documentation lives under `docs/`. A top-level `src/` package is intentionally avoided because it would change the natural Go import path or require an artificial forwarding facade.

At minimum, normal development uses:

```text
go test ./...
go test -race ./...
go vet ./...
```

Additional static, complexity, vulnerability, secret-scanning, conformance, and hygiene checks are documented in [`CONTRIBUTING.md`](CONTRIBUTING.md).

## Author

Marksplice was created by Giovanni Riccobene (`zoster81`).

## License

Licensed under the Apache License, Version 2.0. See [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE).

Goldmark is an MIT-licensed third-party dependency. Exact dependency versions are recorded in `go.mod` and `go.sum`.
