# Contributing to Marksplice

Marksplice is a stable v1 Go module. Contributions should favor correctness, deterministic behavior, source preservation, and narrow evidence-backed changes over unnecessary API breadth.

## Before changing code

Read `AGENTS.md`, then the current source-of-truth documents relevant to the change. Architecture/parser/conformance work normally requires [`docs/architecture.md`](docs/architecture.md) and [`docs/gfm-conformance.md`](docs/gfm-conformance.md). Historical milestone records are useful when reconstructing why an older contract exists, but they are not prerequisites for unrelated work.

Inspect the relevant implementation and tests before editing. Public API changes must update [`docs/api-reference.md`](docs/api-reference.md) plus every affected Getting Started, guide, recipe, capability, and runnable-example surface in the same change.

For substantive work, cover four engineering phases in the working discussion:

1. requirements and edge cases;
2. architecture and test strategy;
3. devil's-advocate risks and mitigations;
4. implementation and verification.

## Test-first workflow

When practical:

1. add or refine a focused test that demonstrates the required behavior;
2. run it and confirm the expected failure;
3. implement the smallest coherent fix;
4. rerun the focused test;
5. run relevant regressions.

Source-preservation tests must verify untouched bytes, not only semantic equivalence.

## Supported Go versions

The minimum supported Go version is the `go` directive in `go.mod`, currently Go 1.26. Public CI also tests the current Go 1.27 line across supported operating systems. Do not raise the compatibility floor merely to use a newer release compiler.

## Required local checks

Run the applicable subset for the change:

```text
gofmt -w <changed-go-files>
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
go build ./...
staticcheck ./...
golangci-lint run
gocyclo -over 15 -ignore '_test\.go$' .
unparam ./...
unparam -tests ./...
govulncheck ./...
gitleaks dir . --no-banner --redact
git diff --check
git status --short
```

Do not claim a check passed unless it was executed. Report skipped checks and their reason.

The black-box public API suite lives in `internal/publictest` and imports `github.com/zoster81/marksplice` like an external consumer. Consequently, `go test . -cover` is not a meaningful project-wide coverage figure; coverage gates must instrument packages across the module boundary.

## Documentation

The repository [`README.md`](README.md) is the single public entry point. Keep learning material progressively disclosed:

- `docs/getting-started.md` owns the first successful workflow;
- `docs/guide.md` routes readers by goal;
- `docs/recipes/` owns focused workflows;
- `examples/` owns runnable file-based examples;
- `docs/capabilities.md` describes the current product boundary;
- `docs/api-reference.md` is the exhaustive callable reference;
- `docs/architecture.md`, `docs/gfm-conformance.md`, and `docs/releasing.md` describe current maintainer contracts;
- `docs/milestones/` preserves historical engineering chronology.

Current documentation must explain present behavior directly. Do not require readers to know internal milestone numbers, development phases, or the name of a particular downstream consumer. Historical chronology belongs only in explicitly historical records or the maintainer roadmap.

When changing documentation, check relative links, run affected examples, preserve fixtures, and run `git diff --check`.

## Repository layout

The public document package stays at the module root so its import path remains:

```text
github.com/zoster81/marksplice
```

Root Go files are grouped by responsibility: `api*.go` for parsed-document/read/edit APIs and `builder*.go` for new-document construction. `workspacefs/` is the separate caller-authorized read-only filesystem adapter. Private parser/source/splice/rendering implementation and black-box tests live under `internal/`.

Do not add a cosmetic top-level `src/` package that changes the natural import path or requires a forwarding facade.

## Markdown profile and dependencies

Marksplice exposes one Markdown profile: CommonMark 0.31.2 plus explicit published-GFM extensions/corrections. Follow [`docs/gfm-conformance.md`](docs/gfm-conformance.md) for normative hierarchy, approved snapshots, and update procedure.

Production parsing uses the Marksplice-owned Native backend behind parser-independent internal contracts. Reintroducing a third-party semantic parser or exposing parser-specific public types requires an explicit architecture decision.

Keep dependencies minimal. `golang.org/x/text` is the direct production dependency used for full Unicode GFM reference-label folding. Exact versions belong in `go.mod` and `go.sum`.

Footnotes, mathematical source forms, alerts, and front matter are reviewed Marksplice capabilities with independent source proof; they are not hidden caller-selectable dialect modes.

Rendering consumes the on-demand Native semantic walk and must not introduce a second Markdown parser, retained renderer AST, hidden filesystem/network I/O, asset fetcher, template engine, syntax highlighter, or math engine. Source mapping remains optional and must not change emitted HTML bytes. Canonical Markdown remains an explicit normalization export rather than an edit path.

## Conformance

When approved specification snapshots are available, set:

```text
MARKSPLICE_COMMONMARK_SPEC_HTML
MARKSPLICE_GFM_SPEC_HTML
```

Then run the exact anchored parser, semantic, renderer, and canonical-Markdown gates documented in [`docs/gfm-conformance.md`](docs/gfm-conformance.md). An incorrect `-run` filter may select zero tests while `go test` still exits successfully; use the documented exact names.

Each loader verifies the approved snapshot identity before evaluating examples. Do not accept an upstream change by updating only a hash or by mechanically regenerating expectations from current Native output.

## Source preservation

Ordinary existing-document edits must not render and replace the complete document. Prepared changes use validated source ownership, preserve untouched bytes, and fail closed when applied to a different source snapshot.

Use byte offsets for mutation boundaries. Be deliberate about LF/CRLF, Unicode, malformed input, duplicate labels, large inputs, and deterministic failure behavior.

## Authority boundaries

The document core performs no arbitrary filesystem traversal, URL fetching, network access, command execution, or host authorization.

`workspacefs` may consume only caller-supplied `fs.FS` read authority under explicit finite limits. Host authorization, writes, backups, encoding/BOM policy, network fetching, commands, and application-specific security policy remain outside Marksplice.

Do not add consumer-specific adapters or workflow policy to core. A generally useful capability should be expressed through a product-neutral public contract; product-specific syntax or semantics belongs in an independent package or host application.

## Releases

Public module versioning and publication verification are defined in [`docs/releasing.md`](docs/releasing.md). A release is cut only from the exact reviewed commit that passed the applicable local and public CI gates.

A version already observed by Go module tooling must not be reused for different source bytes. If release metadata or documentation needs correction after proxy publication, publish a new version and retract the affected one when appropriate.
