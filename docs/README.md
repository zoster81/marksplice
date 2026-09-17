# Documentation Map

The repository [`README.md`](../README.md) is the single public entry point. This page helps readers jump directly to the right level of detail.

## User documentation

For normal library use:

1. [Getting Started](getting-started.md) — install, parse a real file, inspect, query, edit, apply, create, and render.
2. [User Guide](guide.md) — choose an API family by goal.
3. [Recipes](recipes/README.md) — focused workflows tied to runnable examples.
4. [Examples](../examples/README.md) — complete programs using tracked Markdown fixtures.
5. [Capabilities](capabilities.md) — concise current read/edit/create/render boundaries.
6. [API Reference](api-reference.md) — exhaustive exported callable signatures and descriptions.

Normal users should not need development chronology or parser-transition history to use Marksplice.

## Maintainer documentation

Use these when changing the project or when you need design rationale:

- [Architecture](architecture.md) — current package boundaries, source preservation, mutation, construction, performance, and authority rules.
- [Markdown Conformance Policy](gfm-conformance.md) — normative CommonMark/GFM hierarchy, approved external snapshots, and update procedure.
- [Capability and Third-Party Extensibility Strategy](extension-strategy.md) — what belongs in core versus independent extensions/integrations.
- [Roadmap](roadmap.md) — planned engineering/release sequence.
- [Release and Versioning Policy](releasing.md) — module versioning and publication procedure.
- [`CONTRIBUTING.md`](../CONTRIBUTING.md) — contributor workflow and verification gates.
- [`SECURITY.md`](../SECURITY.md) — private vulnerability reporting.

## Historical engineering records

These files preserve implementation decisions and verification evidence. They are useful when reconstructing why a contract exists, but they are not part of the normal learning path:

- [`milestones/`](milestones/) — detailed historical feature/design/test records.
- [Historical parser capability matrix](goldmark-capability-matrix.md) — retired parser/source transition evidence.
- [`CHANGELOG.md`](../CHANGELOG.md) — public release history and user-visible release notes.

Current documentation describes current behavior directly. Historical phase numbers and old implementation transitions stay in explicitly historical records rather than leaking into the user-facing API story.

## Repository layout

```text
.
├── api.go / api_*.go        parsed-document, read, query, graph, and edit API
├── builder.go / builder_*.go
│                            new-document construction API
├── doc.go                   package documentation
├── example_test.go          compact pkg.go.dev examples
├── examples/                runnable file-based user examples
├── workspacefs/             caller-authorized read-only fs.FS workspace adapter
├── internal/                private parser/source/splice/rendering implementation and tests
├── docs/                    user, reference, maintainer, roadmap, and historical documentation
└── README.md                single public entry point
```

The main package import path is:

```text
github.com/zoster81/marksplice
```

Filesystem-backed workspace discovery is an explicit separate package:

```text
github.com/zoster81/marksplice/workspacefs
```

New documentation should have one clear responsibility and link to the existing source of truth rather than duplicating it.
