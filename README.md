# Marksplice

Source-preserving GitHub Flavored Markdown manipulation for Go.

Marksplice is an open-source Pure-Go library for understanding and structurally editing GitHub Flavored Markdown (GFM) while preserving untouched source bytes whenever an operation does not semantically require broader changes.

## Status

Marksplice has passed both its lossless-editing feasibility milestone (M1) and its public-API foundation milestone (M2). The published GitHub Flavored Markdown 0.29 specification is the project's single normative Markdown syntax profile. M1 established the Goldmark + Marksplice lossless architecture; M2 established the first reviewed Marksplice-owned public parse/read/lookup/change surface.

The public API remains intentionally narrow rather than feature-complete. Additional syntax families will be promoted only after their caller-facing semantics and source-preserving operations are reviewed.

## Design principles

- follow the published GitHub Flavored Markdown 0.29 specification as the single normative Markdown syntax profile;
- parse GFM for semantic understanding without implying whole-document normalization;
- preserve untouched author choices such as heading/list/fence styles, whitespace, line endings, and other lexical trivia;
- bind prepared edits to exact source snapshots and reject stale application;
- keep Goldmark behind an internal adapter and expose only Marksplice-owned types;
- keep filesystem, network, MCP, and host authorization concerns outside the core library.

See [`docs/architecture.md`](docs/architecture.md) for durable design decisions, [`docs/gfm-conformance.md`](docs/gfm-conformance.md) for the normative Markdown/conformance policy, [`docs/milestones/m1-lossless-editing.md`](docs/milestones/m1-lossless-editing.md) for the completed M1 evidence, and [`docs/milestones/m2-public-api-foundation.md`](docs/milestones/m2-public-api-foundation.md) for the completed public-API foundation.

## Public API foundation

The current public surface deliberately promotes only reviewed semantics: immutable snapshots, opaque snapshot-scoped node identities, headings and top-level paragraphs as generic node kinds, typed top-level paragraph ranges, source-bound paragraph replacement, and `errors.Is`-compatible public error categories. Internal M1 syntax coverage is broader than the public API and is not automatically a compatibility commitment.

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
