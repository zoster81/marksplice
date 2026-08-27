# Documentation Map

The repository [`README.md`](../README.md) is the single public entry point. This page is a map for readers who already know what they need.

## User documentation

Follow this path for normal library use:

1. [Getting Started](getting-started.md) — install, parse a real file, inspect, query, edit, apply, and create.
2. [User Guide](guide.md) — choose an API family by goal.
3. [Recipes](recipes/README.md) — focused workflows tied to runnable file-based examples.
4. [Examples](../examples/README.md) — complete programs that load tracked Markdown fixtures.
5. [Capabilities](capabilities.md) — concise current read/edit/create boundaries.
6. [API Reference](api-reference.md) — exhaustive exported callable signatures and GoDoc-derived descriptions.

Normal users should not need milestone records or parser-transition documentation to get started.

## Advanced and maintainer documentation

Use these when changing Marksplice itself or when you need the design rationale behind a boundary:

- [Architecture](architecture.md) — durable package, source-preservation, mutation, construction, performance, and authority decisions.
- [Markdown Conformance Policy](gfm-conformance.md) — normative CommonMark/GFM hierarchy, pinned external specification inputs, and update procedure.
- [Capability and Third-Party Extensibility Strategy](extension-strategy.md) — what belongs in core versus independent read-only extensions.
- [Release and Versioning Policy](releasing.md) — public module/release procedure.
- [`CONTRIBUTING.md`](../CONTRIBUTING.md) — contributor workflow and verification gates.
- [`SECURITY.md`](../SECURITY.md) — private vulnerability reporting.

## Historical engineering records

These files preserve decisions and verification evidence but are not part of the normal user journey:

- [`milestones/`](milestones/) — M0–M115 feature/design/test records.
- [Goldmark capability matrix](goldmark-capability-matrix.md) — historical pre-M115 parser/source transition record.
- [`CHANGELOG.md`](../CHANGELOG.md) — public release history and unreleased user-visible changes.

History remains tracked so maintainers can reconstruct why a contract exists without forcing new users to read the development chronology.

## Repository layout

```text
.
├── api.go / api_*.go        parsed-document, read, query, graph, and edit API
├── builder.go / builder_*.go
│                            new-document construction API
├── doc.go                   package documentation
├── example_test.go          compact pkg.go.dev examples
├── examples/                runnable file-based user examples
├── internal/                private parser/source/splice implementation and tests
├── docs/                    user, reference, advanced, and historical documentation
└── README.md                single public entry point
```

The public Go package stays at the module root so consumers import exactly:

```text
github.com/zoster81/marksplice
```

Do not create a duplicate public documentation tree or a cosmetic top-level `src/` package. New documentation should have one clear responsibility and link to the existing source of truth instead of copying it.
