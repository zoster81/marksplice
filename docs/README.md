# Marksplice Documentation

This directory is the single home for tracked project documentation beyond the repository-facing files that GitHub conventionally keeps at the module root (`README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`, `LICENSE`, and `NOTICE`).

## Start here

- [`guide.md`](guide.md): task-oriented module guide with installation, end-to-end examples, editing/construction flows, graphs/workspaces, knowledge indexing, extensions, errors, and concurrency rules.
- [`api-reference.md`](api-reference.md): exhaustive reference for every exported function and method, with exact current signatures and public GoDoc explanations.
- [`capabilities.md`](capabilities.md): current read/edit/create capability matrix and completed parser roadmap.
- [`architecture.md`](architecture.md): durable architecture, package boundaries, source-preservation model, and repository-layout rationale.
- [`gfm-conformance.md`](gfm-conformance.md): normative GitHub Flavored Markdown profile and specification-update policy.
- [`goldmark-capability-matrix.md`](goldmark-capability-matrix.md): historical pre-M115 parser/source ownership transition record and Native cutover context.
- [`extension-strategy.md`](extension-strategy.md): selection of broadly useful core capabilities from ecosystem ideas, the reviewed third-party extensibility/SPI boundary, and the completed Native-parser cutover strategy.
- [`releasing.md`](releasing.md): public Go-module versioning, beta-release procedure, and publication verification.
- [`milestones/`](milestones/): detailed milestone contracts, design records, tests, and historical verification evidence.

## Repository layout

```text
.
├── api.go
├── api_*.go                 public parsed-document/read/edit API
├── builder.go
├── builder_*.go             new-document construction API and package-private implementation
├── doc.go                   package documentation for pkg.go.dev
├── example_test.go          executable pkg.go.dev examples
├── internal/
│   ├── parser/              parser-independent contract, production Native parser, and tracked parser-neutral conformance fixtures
│   ├── publictest/          black-box tests of the public module API
│   ├── testutil/            shared test-only pinned specification loaders
│   ├── source/              lossless source ownership/mapping
│   └── splice/              source-preserving document and mutation engine
├── docs/                    user guide, API reference, architecture, conformance, and project documentation
├── .github/                 public CI and dependency-update automation
├── go.mod / go.sum          module identity and dependency graph
└── repository metadata      README, license, security, changelog, contributing
```

The public Go package intentionally remains at the module root. For a Go module whose canonical consumer import is:

```text
github.com/zoster81/marksplice
```

moving that package under a top-level `src/` directory would naturally change its import path to `github.com/zoster81/marksplice/src`, or require an artificial forwarding facade at the root. Marksplice therefore follows the idiomatic Go layout: the root contains the public package, private implementation packages live under `internal/`, black-box API tests live under `internal/publictest/`, and all longer-form documentation lives under `docs/`.

The root package filenames are grouped deliberately:

- `api*.go` owns parsed-document identity, typed views, sections, mutation preparation, the bounded public query surface in `api_query.go`, native anchor/fragment/TOC navigation in `api_navigation.go`, immutable single-document link intelligence in `api_relationships.go`, the explicit caller-provided multi-document graph in `api_graph.go`, bounded workspace validation/repair planning in `api_workspace.go`, syntax-independent knowledge-document indexing in `api_knowledge.go`, Marksplice-owned semantic block overlays such as GitHub alerts in `api_alerts.go`, the complete read-only fenced-container model in `api_fenced_blocks.go`, source-proven footnote definitions/reference relationships in `api_footnotes.go`, and mathematical-expression read/edit projection in `api_math.go`;
- `builder*.go` owns new-document construction and its private validation/writing/proof helpers; generic typed-inline writing stays in `builder_inline.go`, reference-link/reference-image forms, resolution, and proof stay in `builder_inline_reference.go`, footnote definitions/typed-reference construction stays in `builder_footnotes.go`, mathematical construction stays in `builder_math.go`, and alert construction entrypoints stay in `builder_alerts.go` while reusing the shared blockquote writer/proof machinery;
- `doc.go` and `example_test.go` own package documentation and executable examples;
- `guide.md` owns the task-oriented user journey, while `api-reference.md` mirrors every exported callable from the public Go declarations and must be updated whenever that callable surface changes.

Do not introduce a second source tree or duplicate documentation tree merely for cosmetic separation. New code should extend the existing package boundaries or justify a new internal package with a concrete responsibility.
