# Contributing to Marksplice

Marksplice is currently proving its source-preserving editing model. Contributions should favor correctness, deterministic behavior, and narrow changes over API breadth.

## Before changing code

Read `AGENTS.md`, `docs/architecture.md`, `docs/gfm-conformance.md`, and the active milestone under `docs/milestones/`. Inspect the relevant implementation and tests before editing.

For substantive changes, document the four engineering phases in the working discussion: requirements/edge cases, architecture/test strategy, devil's advocate risks, and implementation/verification.

## Test-first workflow

When practical:

1. add or refine a focused test that demonstrates the required behavior;
2. run it and confirm the expected failure;
3. implement the smallest coherent fix;
4. rerun the focused test;
5. run the relevant regression suite.

Source-preservation tests must verify bytes outside changed spans, not only semantic equivalence.

## Required local checks

For code changes, run the applicable subset of:

```text
gofmt -w <changed-go-files>
go test ./...
go test -race ./...
go vet ./...
staticcheck ./...
golangci-lint run
govulncheck ./...
gitleaks dir . --no-banner --redact
git diff --check
git status --short
```

Do not claim a check passed unless it was actually executed. A skipped check should be reported with the reason.

## Markdown profile and dependency policy

GitHub Flavored Markdown (GFM) is the repository's single normative Markdown syntax profile. Follow [`docs/gfm-conformance.md`](docs/gfm-conformance.md) for the normative source hierarchy, approved snapshot, CommonMark relationship, advisory-source policy, and specification-update procedure. Do not add separate dialect modes or non-GFM syntax extensions without an explicit architecture decision and corresponding conformance tests.

Keep dependencies minimal. Goldmark is the selected semantic parser, configured for GFM plus narrowly scoped, tested compatibility behavior required by the approved GFM contract. It must remain behind the internal parser adapter. Do not expose third-party parser types through public APIs. Exact dependency versions belong in `go.mod` and `go.sum`.

When the approved GFM specification snapshot is provisioned separately, set `MARKSPLICE_GFM_SPEC_HTML` to its HTML path and run `go test ./internal/parser/goldmark -run TestGFM029PublishedSpecificationConformance`. The test verifies the approved SHA-256 before evaluating examples; an upstream specification change requires the reviewed update process in `docs/gfm-conformance.md`, not a hash-only update.

## Source preservation

Ordinary edits to existing Markdown must not render and replace the complete document. Prepared changes must use validated source ranges, preserve untouched bytes, and fail closed when applied to a different source snapshot.

Use byte offsets for source mutation boundaries. Be deliberate about LF/CRLF, Unicode, malformed input, duplicate human-readable labels, large inputs, and deterministic behavior.

## Scope discipline

Do not add Scripthold-specific MCP adapters or host filesystem/security responsibilities to this repository. Broad public API design and document-graph features remain gated on milestone M1.
