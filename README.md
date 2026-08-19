# Marksplice

Source-preserving GitHub Flavored Markdown manipulation for Go.

Marksplice is an open-source Pure-Go library for understanding and structurally editing GitHub Flavored Markdown (GFM) while preserving untouched source bytes whenever an operation does not semantically require broader changes.

## Status

Marksplice has passed milestones M1 through M6: lossless-editing feasibility, public-API foundation, heading/list public APIs, mapped block public APIs, and simple inline public APIs. The current reviewed surface covers top-level paragraphs/headings, M1-proven list items and GFM task markers, mapped non-empty table cells, supported single-line fenced code, and mapped simple strikethrough/code-span/emphasis/strong content. The published GitHub Flavored Markdown 0.29 specification is the project's single normative Markdown syntax profile.

The public API remains intentionally narrow rather than feature-complete. Additional syntax families will be promoted only after their caller-facing semantics and source-preserving operations are reviewed.

## Design principles

- follow the published GitHub Flavored Markdown 0.29 specification as the single normative Markdown syntax profile;
- parse GFM for semantic understanding without implying whole-document normalization;
- preserve untouched author choices such as heading/list/fence styles, whitespace, line endings, and other lexical trivia;
- bind prepared edits to exact source snapshots and reject stale application;
- keep Goldmark behind an internal adapter and expose only Marksplice-owned types;
- keep filesystem, network, MCP, and host authorization concerns outside the core library.

See [`docs/architecture.md`](docs/architecture.md) for durable design decisions, [`docs/gfm-conformance.md`](docs/gfm-conformance.md) for the normative Markdown/conformance policy, and [`docs/milestones/`](docs/milestones/) for milestone evidence and exit decisions.

## Public API foundation

The current public surface deliberately promotes only reviewed, source-mapped capabilities: immutable snapshots, opaque snapshot-scoped node identities, top-level paragraphs/headings, M1-proven single-line list items and GFM task markers, mapped non-empty table cells, supported single-line fenced code, and simple strikethrough/code-span/emphasis/strong spans. Typed details expose only operation-specific source ranges and reviewed semantic state; named operations prepare minimal source-bound changes. Unsupported semantic shapes such as normalized-space code spans or compound emphasis remain internal rather than appearing publicly actionable. Internal M1 syntax coverage remains broader than the public API and is not automatically a compatibility commitment.

## Development

The module path is:

```text
github.com/zoster81/marksplice
```

Run the standard checks with the Go toolchain selected for the repository:

```text
go test ./...
go test -race ./...
go vet ./...
```

Additional static, vulnerability, and secret-scanning checks are described in [`CONTRIBUTING.md`](CONTRIBUTING.md).

## Author

Marksplice was created by Giovanni Riccobene (`zoster81`).

## License

Licensed under the Apache License, Version 2.0. See [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE).

Goldmark is an MIT-licensed third-party dependency. Exact dependency versions are recorded in `go.mod` and `go.sum`.
