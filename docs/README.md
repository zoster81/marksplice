# Marksplice Documentation

This directory is the single home for tracked project documentation beyond the repository-facing files that GitHub conventionally keeps at the module root (`README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`, `LICENSE`, and `NOTICE`).

## Start here

- [`capabilities.md`](capabilities.md): current read/edit/create capability matrix and forward product roadmap.
- [`architecture.md`](architecture.md): durable architecture, package boundaries, source-preservation model, and repository-layout rationale.
- [`gfm-conformance.md`](gfm-conformance.md): normative GitHub Flavored Markdown profile and specification-update policy.
- [`goldmark-capability-matrix.md`](goldmark-capability-matrix.md): Goldmark-versus-Marksplice parser/source responsibility boundary.
- [`extension-strategy.md`](extension-strategy.md): selection of broadly useful core capabilities from ecosystem ideas, the future third-party extensibility/SPI boundary, and the mandatory M115 Goldmark removal plan.
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
│   ├── parser/              parser-independent contracts and Goldmark adapter
│   ├── publictest/          black-box tests of the public module API
│   ├── source/              lossless source ownership/mapping
│   └── splice/              source-preserving document and mutation engine
├── docs/                    project documentation
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

- `api*.go` owns parsed-document identity, typed views, sections, mutation preparation, and the bounded public query surface in `api_query.go`;
- `builder*.go` owns new-document construction and its private validation/writing/proof helpers; generic typed-inline writing stays in `builder_inline.go`, while reference-link/reference-image forms, resolution, and proof stay in `builder_inline_reference.go`;
- `doc.go` and `example_test.go` own package documentation and executable examples.

Do not introduce a second source tree or duplicate documentation tree merely for cosmetic separation. New code should extend the existing package boundaries or justify a new internal package with a concrete responsibility.
