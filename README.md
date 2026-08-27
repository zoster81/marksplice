# Marksplice

[![Go Reference](https://pkg.go.dev/badge/github.com/zoster81/marksplice.svg)](https://pkg.go.dev/github.com/zoster81/marksplice)
[![CI](https://github.com/zoster81/marksplice/actions/workflows/ci.yml/badge.svg)](https://github.com/zoster81/marksplice/actions/workflows/ci.yml)

Structured GitHub Flavored Markdown creation and source-preserving manipulation for Go.

Marksplice is an open-source Pure-Go library for understanding, creating, and structurally editing GitHub Flavored Markdown (GFM). New documents may be generated from reviewed structured intent, while edits to existing documents preserve untouched source bytes whenever the requested operation does not semantically require broader changes.

## Status

Marksplice is currently **beta software under active development**. The first public beta version is `v0.1.0-beta.1`; until v1, public APIs and behavior may change incompatibly between releases.

The repository has a green retrospective M0 bootstrap record and completed engineering milestones M1–M115. M111–M115 replaced the transitional third-party parser with the Marksplice-native CommonMark/GFM backend; production parsing and construction proof are now Native-only. The current core model has two deliberately separate paths:

- parsed `Document` snapshots expose reviewed source-mapped read/edit capabilities and prepare minimal source-bound changes that reject stale input;
- `DocumentBuilder` creates new deterministic GFM and validates generated structure through the same parser/source-model boundary before returning bytes.

The current public surface covers reviewed paragraphs/headings/sections, supported list and task hierarchies, fenced code, GFM tables with public table/row/cell ownership and conservative row/alignment/column structural edits plus canonical header-only table construction, simple inline spans/links/images/reference definitions/footnotes/autolinks, parser-proven semantic destination/title reading for promoted simple links, label/destination/title reading for promoted reference definitions, semantic value/email classification for promoted autolinks, bounded source-ordered `QueryNodes`/`QuerySections` selectors over immutable snapshots, source-proven top-level thematic breaks and complete existing-source blockquote containers including multiline, nested, lazy-continuation, and multi-block forms where every physical line is owned, recognized source-owned YAML/TOML front-matter envelopes including opaque complex/duplicate/empty metadata, unique simple top-level scalar field edits, and canonical new-document envelope construction, simple HTML comments/anchors, typed inline construction for semantic text/code/emphasis/strong/strikethrough/links/images/autolinks including conservative canonical link/image titles, bounded reviewed emphasis-family nesting, structured link labels/image alt content using that same bounded inline model, exact-prior full references, explicit deferred full-forward references, normalized collapsed/shortcut reference link/image construction, parser-proven bare/extended autolink construction, source-preserving existing reference-definition title replacement and fail-closed complete-line removal, atomic composition of independent already-prepared source-bound mutations with overlap/model-interaction rejection and one combined candidate proof, canonical single-paragraph blockquote construction at depth 1 or explicit nesting depth 2–64, and construction-only multi-block blockquotes composed from reviewed builder children including recursively nested blockquotes with total depth bounded at 64.

Document intelligence builds on that editing surface without adding hidden workspace authority: `HeadingAnchors`/fragment/TOC APIs provide single-document navigation, `LinkRelationships` exposes source-ordered parser-resolved link/image/autolink facts, and `BuildDocumentGraph` combines only an explicit caller-provided set of parsed documents under opaque caller keys. Non-local graph edges require a build-only caller resolver whose target must already belong to that set; outgoing edges, backlinks, reachability, and direct related-document queries perform no filesystem traversal, path normalization, or network access. `ValidateWorkspace` adds caller-authorized diagnostics for local/cross-document fragment failures, caller-declared missing documents, conservative unresolved explicit references, root-relative orphans, and explicitly managed TOCs. Its resolver is build-only and automatic repair is limited to deterministic source-preserving synchronization of caller-designated stale TOCs. `BuildKnowledgeIndex` can then layer exact caller-owned aliases, tags, and direct logical `DocumentKey` references over that same immutable graph without parsing dialect syntax or modifying Markdown edges.

M102 additionally recognizes the five reviewed GitHub alert markers (`NOTE`, `TIP`, `IMPORTANT`, `WARNING`, `CAUTION`) as a semantic overlay over already-source-proven top-level blockquotes. Alerts reuse the underlying blockquote identity and complete source ownership, expose exact marker/body ranges, and can be constructed canonically from single-paragraph, typed-inline, or reviewed multi-block bodies. No alert parser extension, new structural `Kind`, persistent alert index, or alert-specific existing-source rewrite authority is introduced.

M103 adds a complete read-only `FencedBlock` view over source-proven top-level GFM fenced containers, including exact opening/closing fence metadata, info string/language, complete container range, and per-line payload ranges for empty, indented/non-contiguous, closed, and unclosed forms. Existing payload replacement remains deliberately narrower through the historical contiguous `FencedCode` contract, and new-document construction can emit a canonical empty fenced block without inventing a blank payload line. Embedded languages remain opaque data.

M104 adds exact case-sensitive footnote relationships plus source-proven top-level definitions. Multiline semantic body segments are readable without becoming broad mutation spans; simple opening-line bodies can be replaced source-preservingly, definition rename updates every parser-bound reference occurrence atomically, canonical definitions may be immediate or explicitly deferred, and typed references can target those definitions. Links inside footnote bodies reuse the ordinary relationship/graph intelligence. Footnotes remain an explicitly reviewed core capability outside the pinned GFM 0.29 baseline rather than a caller-selectable Markdown mode.

M105 adds conservative GitHub-compatible mathematical source semantics for source-proven `$...$`, dollar-backtick, one-line `$$...$$`, and exact-info `math` fenced forms. Dedicated expressions expose immutable style/source/payload ranges, source-preserving payload replacement, `KindMathExpression` queries, and typed/canonical construction. Fenced math reuses the existing M103 fenced identity and edit boundary. Mathematical payload is opaque: Marksplice does not parse or render LaTeX/MathJax/KaTeX/MathML and adds no math dependency or runtime renderer.

M106 separates front-matter envelope recognition from field editability. `Document.FrontMatter()` exposes exact YAML/TOML envelope ownership even when metadata is complex, duplicate-only, or empty, while only unique source-proven simple top-level scalar fields remain editable. Empty leading envelopes have explicit metadata precedence; non-empty delimiter pairs without conservative metadata evidence remain GFM. TOML table scope is recognized conservatively so nested table members are not promoted as top-level fields. Marksplice still does not parse or serialize arbitrary YAML/TOML.

M107 adds syntax-independent aliases, tags, and logical document references over an explicit immutable graph without deriving dialect syntax or adding discovery authority. M108 adds fuzz/pathological/performance regression coverage and removes measured Marksplice-owned quadratic/copy costs without hidden caps or speculative persistent indexes. M109 stabilizes cross-cutting contracts: immutable public snapshot/graph/knowledge/workspace/change values are safe for concurrent reads, `DocumentBuilder` remains caller-synchronized mutable state, resolver callbacks are synchronous/non-retained, sentinel errors are classified with `errors.Is`, and `Kind` remains the closed core structural namespace. M110 adds explicit `ParseWithOptions` third-party read-only observations with separate namespaces/kinds, snapshot ranges, scalar metadata, caller-owned retention limits, serial non-retained recognizers, and fail-closed `ErrInvalidExtension`; core GFM, mutation/construction, graph semantics, parser internals, and host authority remain unchanged. The only M109 public API reshaping is the pre-v1 replacement of the ambiguous four-value workspace unresolved-reference accessor with typed immutable `UnresolvedReference` data.

The public API remains intentionally narrower than everything the semantic parser can recognize. Unsupported or ambiguous shapes are preserved or kept internal until exact source ownership and caller-facing semantics are proven.

Start with the task-oriented [`docs/guide.md`](docs/guide.md), then use [`docs/api-reference.md`](docs/api-reference.md) for the complete exported callable surface. [`docs/README.md`](docs/README.md) maps the remaining architecture, conformance, capability, release, and historical milestone documentation.

## Design principles

- use CommonMark 0.31.2 as the normative base grammar, layering the published GFM specification only for explicit extensions and corrections;
- create new GFM through deterministic reviewed construction rules and parser/model proof;
- parse existing GFM for semantic understanding without implying whole-document normalization;
- preserve untouched author choices such as heading/list/fence styles, whitespace, delimiters, numbering, and line endings during existing-document edits;
- bind prepared edits to exact source snapshots and reject stale application;
- promote public capabilities only after operation-oriented source ownership is proven;
- keep parser implementation details behind the internal parser-independent backend boundary and expose only Marksplice-owned types;
- keep filesystem, network, command-execution, and host authorization concerns outside the core library.

## Installation

Marksplice requires Go 1.26 or newer. The current published beta is installed explicitly because Go does not prefer pre-release versions by default:

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

Executable examples for construction, parsing, explicit third-party `ParseWithOptions` observation, bounded `QueryNodes` selection, link intelligence, explicit multi-document graph construction, syntax-independent knowledge indexing, workspace validation, GitHub alert recognition, fenced-block metadata reading, opaque front-matter envelope reading, source-preserving heading mutation, and atomic `ComposeChanges` mutation composition live in `example_test.go` and are published by pkg.go.dev.

## Construction and editing

`DocumentBuilder` writes canonical LF GFM for reviewed new-document families such as headings, parser-proven paragraphs, lists/tasks including homogeneous nesting, fenced code including empty payloads, reference definitions, tables with optional alignment and optional body rows, thematic breaks, blockquotes, and exact non-nested GitHub alerts. `AppendBlockquote` owns one paragraph at depth 1, `AppendNestedBlockquote` accepts one paragraph at explicit depths 2–64, and `AppendBlockquoteBlocks` snapshots another builder's reviewed body blocks and quotes that sequence at depth 1–64. Multi-block composition accepts every reviewed body-block construction family, including recursive blockquote children when total structural depth stays at most 64; front matter remains excluded because it is a document envelope. It can also own one document-leading canonical YAML or TOML front-matter envelope with conservative double-quoted string fields. Typed-inline entrypoints provide a semantic-text alternative to the historical raw-GFM block APIs, including `AppendNestedBlockquoteContent`; `LinkInlineWithTitle` and `ImageInlineWithTitle` add conservative canonical double-quoted titles. M88 allows bounded nesting of `CodeInline`, `EmphasisInline`, `StrongInline`, and `StrikethroughInline` inside emphasis/strong/strikethrough wrappers, while ambiguous GFM delimiter combinations fail closed. M89 adds `ReferenceLinkInline` and `ReferenceImageInline` for canonical full-reference forms whose exact label must match exactly one top-level reference definition already present in the same builder. M92 allows direct and full-reference link labels/image alt content to use the same bounded M88 structured-inline family while keeping destination/title/reference semantics independently proven from child hierarchy. M93 adds explicit deferred definitions with full forward-reference constructors, normalized collapsed/shortcut reference link/image constructors, and exact parser-proven bare/extended autolink construction without weakening the M89 prior-definition contract. Existing parsed-source reference links/images remain outside ordinary promotion; promoted single-line reference definitions gain exact title-payload replacement and complete-line removal only when surviving reference relationships are unchanged.

Parsed `Document` values instead retain exact immutable source. `QueryNodes` and `QuerySections` provide bounded source-ordered selection over that existing snapshot model: callers supply an explicit positive result limit plus optional kind/level and containing-range filters, and Marksplice retains no query state or persistent selector index. `NodeMatch.Range()` reuses the matched kind's existing typed operation-oriented range for selection/read purposes rather than defining generic mutation ownership. Mutations target operation-specific ranges and validate the candidate source when surrounding Markdown interpretation could change. `Document.ComposeChanges` can combine independent opaque prepared `ChangeSet` values from the same snapshot into one atomic change; source overlap, overlapping semantic/reference deltas, and combined parser/model interactions fail closed, and callers never receive a generic raw-patch batching API. Structural operations never use the construction writer to reformat an existing document.

## Documentation

- [`docs/guide.md`](docs/guide.md): task-oriented module guide with end-to-end examples.
- [`docs/api-reference.md`](docs/api-reference.md): exhaustive exported function/method reference verified against the public Go declarations.
- [`docs/README.md`](docs/README.md): documentation index and repository-layout map.
- [`docs/capabilities.md`](docs/capabilities.md): current product-facing read/edit/create matrix and completed parser roadmap.
- [`docs/architecture.md`](docs/architecture.md): durable architecture, source-preservation, performance, safety, and complexity decisions.
- [`docs/gfm-conformance.md`](docs/gfm-conformance.md): normative GFM profile, pinned conformance source hierarchy, and update procedure.
- [`docs/goldmark-capability-matrix.md`](docs/goldmark-capability-matrix.md): historical pre-M115 parser/source ownership transition record.
- [`docs/milestones/`](docs/milestones/): feature-specific historical contracts, design records, tests, and exit decisions.
- [`CONTRIBUTING.md`](CONTRIBUTING.md): contributor workflow and current verification commands.
- [`docs/releasing.md`](docs/releasing.md): public beta versioning, release readiness, and Go-module publication procedure.
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

M115 removed the former Goldmark parser dependency. The current direct third-party dependency is `golang.org/x/text`, used by the Native parser for full Unicode GFM reference-label case folding; exact dependency versions are recorded in `go.mod` and `go.sum`, with attribution in [`NOTICE`](NOTICE).
