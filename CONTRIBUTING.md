# Contributing to Marksplice

Marksplice has passed its initial source-preserving editing feasibility gate and is preparing its public v0 beta series. Contributions should continue to favor correctness, deterministic behavior, and narrow evidence-backed changes over API breadth. Until v1, public APIs remain under active review and compatibility-breaking changes must be called out explicitly in the changelog and release notes.

## Before changing code

Read `AGENTS.md`, `docs/architecture.md`, `docs/gfm-conformance.md`, and the relevant milestone records under `docs/milestones/`. When a milestone is marked active, treat it as the scope and acceptance source for new work. Inspect the relevant implementation and tests before editing.

For substantive changes, document the four engineering phases in the working discussion: requirements/edge cases, architecture/test strategy, devil's advocate risks, and implementation/verification.

## Test-first workflow

When practical:

1. add or refine a focused test that demonstrates the required behavior;
2. run it and confirm the expected failure;
3. implement the smallest coherent fix;
4. rerun the focused test;
5. run the relevant regression suite.

Source-preservation tests must verify bytes outside changed spans, not only semantic equivalence.

## Supported Go versions

The minimum supported Go version is defined by the `go` directive in `go.mod`, currently Go 1.26. Public CI also tests the current Go 1.27 release on Linux, Windows, and macOS. Changes should remain compatible with the minimum version unless the minimum-version policy is deliberately changed in the same reviewed update.

## Required local checks

For code changes, run the applicable subset of:

```text
gofmt -w <changed-go-files>
go test ./...
go test -race ./...
go vet ./...
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

Do not claim a check passed unless it was actually executed. A skipped check should be reported with the reason.

The black-box public API suite lives in `internal/publictest` and imports `github.com/zoster81/marksplice` like an external consumer. Consequently, `go test . -cover` measures only the root package's local examples and is not a meaningful project coverage figure. Maintainer coverage gates must instrument the module packages across package boundaries; normal functional verification remains `go test ./...`.

## Repository layout

The canonical public package stays at the module root so its import path remains `github.com/zoster81/marksplice`. Root Go files are grouped by responsibility: `api*.go` for parsed-document/read/edit APIs, `builder*.go` for new-document construction, plus `doc.go` and `example_test.go`. Private parser/source/splice implementation and black-box tests live under `internal/`; longer-form documentation lives under `docs/` and is indexed by [`docs/README.md`](docs/README.md).

Do not introduce a top-level `src/` package merely for visual separation: in a Go module that would change the natural consumer import path or force a redundant forwarding facade. New directories should represent real package or documentation boundaries.

## Markdown profile and dependency policy

GitHub Flavored Markdown (GFM) is the repository's single normative Markdown syntax profile. Follow [`docs/gfm-conformance.md`](docs/gfm-conformance.md) for the normative source hierarchy, approved snapshot, CommonMark relationship, advisory-source policy, and specification-update procedure. Do not add separate dialect modes or non-GFM syntax extensions without an explicit architecture decision and corresponding conformance tests.

Keep dependencies minimal. Goldmark is the current temporary semantic parser, configured for GFM plus narrowly scoped, tested compatibility behavior required by the approved GFM contract. It must remain behind the internal parser adapter until the mandatory M115 native-parser cutover removes it. Do not expose third-party parser types through public APIs. Exact dependency versions belong in `go.mod` and `go.sum`.

When the approved GFM specification snapshot is provisioned separately, set `MARKSPLICE_GFM_SPEC_HTML` to its HTML path and run `go test ./internal/parser/goldmark -run '^TestGFM029PublishedSpecificationConformance$' -count=1`. Use the anchored exact test name: an incorrect `-run` filter can select zero tests while `go test` still exits successfully. The test verifies the approved SHA-256 before evaluating examples; an upstream specification change requires the reviewed update process in `docs/gfm-conformance.md`, not a hash-only update.

## Source preservation

Ordinary edits to existing Markdown must not render and replace the complete document. Prepared changes must use validated source ranges, preserve untouched bytes, and fail closed when applied to a different source snapshot.

Use byte offsets for source mutation boundaries. Be deliberate about LF/CRLF, Unicode, malformed input, duplicate human-readable labels, large inputs, and deterministic behavior.

## Releases and publication

Public module versioning, beta policy, first-push preparation, and release verification are defined in [`docs/releasing.md`](docs/releasing.md). Contributors should not move or recreate published module tags. A release must be cut from the exact reviewed commit that passed the applicable verification gates.

## Scope discipline

Do not add Scripthold-specific MCP adapters or host filesystem/security responsibilities to this repository. M1 no longer blocks public API design, but the current feasibility internals must not be promoted wholesale into a stable API; post-M1 API and document-graph work should be designed explicitly from the proven architecture and source-preservation invariants.
