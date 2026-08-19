# Marksplice

Source-preserving GitHub Flavored Markdown manipulation for Go.

Marksplice is an open-source Pure-Go library for understanding and structurally editing GitHub Flavored Markdown (GFM) while preserving untouched source bytes whenever an operation does not semantically require broader changes.

## Status

Marksplice has passed its initial lossless-editing feasibility milestone (M1). The published GitHub Flavored Markdown 0.29 specification is the project's single normative Markdown syntax profile. M1 established that Goldmark can remain isolated behind a GFM-conformance adapter while Marksplice owns lossless source mapping, stale-source safety, and minimal byte patches for representative existing-document edits.

The public API is not yet stable; post-M1 API design will be derived from the proven feasibility invariants rather than exposing the internal proof model directly.

## Design principles

- follow the published GitHub Flavored Markdown 0.29 specification as the single normative Markdown syntax profile;
- parse GFM for semantic understanding without implying whole-document normalization;
- preserve untouched author choices such as heading/list/fence styles, whitespace, line endings, and other lexical trivia;
- bind prepared edits to exact source snapshots and reject stale application;
- keep Goldmark behind an internal adapter and expose only Marksplice-owned types;
- keep filesystem, network, MCP, and host authorization concerns outside the core library.

See [`docs/architecture.md`](docs/architecture.md) for durable design decisions, [`docs/gfm-conformance.md`](docs/gfm-conformance.md) for the normative Markdown/conformance policy, and [`docs/milestones/m1-lossless-editing.md`](docs/milestones/m1-lossless-editing.md) for the completed M1 feasibility evidence and exit decision.

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
