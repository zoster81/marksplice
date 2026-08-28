# Contributing to Marksplice

Marksplice is in its public v0 beta series. Contributions should favor correctness, deterministic behavior, source preservation, and narrow evidence-backed changes over API breadth. Until v1, public APIs remain under active review and compatibility-breaking changes must be called out explicitly in the changelog and release notes.

## Before changing code

Read `AGENTS.md` first, then the source-of-truth documents relevant to the change. Architecture/parser/conformance work normally requires `docs/architecture.md` and `docs/gfm-conformance.md`; consult milestone records when their historical contract or evidence is relevant, not as a prerequisite for unrelated work. Inspect the relevant implementation and tests before editing. Public API changes must update `docs/api-reference.md` plus every affected getting-started, guide, recipe, capability, and runnable-example surface in the same change.

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

## Documentation changes

The repository `README.md` is the single public entry point. Keep learning material progressively disclosed instead of duplicating it:

- `docs/getting-started.md` owns the first successful workflow;
- `docs/guide.md` routes by user goal;
- `docs/recipes/` owns focused workflows;
- `examples/` owns runnable file-based examples and tracked Markdown fixtures;
- `docs/capabilities.md` describes current capability, not development chronology;
- `docs/api-reference.md` remains exhaustive and secondary to task-oriented learning;
- advanced architecture/conformance/release documents and `docs/milestones/` retain the detail/history needed by maintainers.

When changing documentation, check relative links, run every affected example, preserve fixture files, and run `git diff --check`. New examples should prefer real tracked `.md` inputs over toy Markdown embedded in Go strings when the example is teaching a file/document workflow.

## Repository layout

The canonical document package stays at the module root so its import path remains `github.com/zoster81/marksplice`. Root Go files are grouped by responsibility: `api*.go` for parsed-document/read/edit APIs, `builder*.go` for new-document construction, plus `doc.go` and `example_test.go`. The separate public `workspacefs/` package owns only caller-authorized read-only `fs.FS` workspace loading; it must delegate parsing, graph, and validation semantics instead of becoming a second document core. Private parser/source/splice implementation and black-box tests live under `internal/`; longer-form documentation lives under `docs/` and is indexed by [`docs/README.md`](docs/README.md).

Do not introduce a top-level `src/` package merely for visual separation: in a Go module that would change the natural consumer import path or force a redundant forwarding facade. New directories should represent real package or documentation boundaries.

## Markdown profile and dependency policy

Marksplice exposes one Markdown syntax profile: CommonMark 0.31.2 is the normative base grammar, with the published GFM extensions/corrections layered on top. Follow [`docs/gfm-conformance.md`](docs/gfm-conformance.md) for the normative source hierarchy, approved snapshots, advisory-source policy, and specification-update procedure. Do not add separate dialect modes or additional syntax without an explicit architecture decision and corresponding conformance tests.

Keep dependencies minimal. Production Markdown parsing and construction proof use the Marksplice-native `internal/parser/native` implementation behind the parser-independent `internal/parser.Backend` contract. M115 removed the former Goldmark adapter, differential oracle, compatibility implementation, and module dependency; reintroducing a third-party semantic parser or parser-specific public type requires an explicit architecture decision. `golang.org/x/text` remains the direct parser-owned dependency used for full Unicode case folding in GFM reference-label normalization, while GFM whitespace rules remain Marksplice-owned. Footnotes and mathematical expressions are reviewed Marksplice core overlays implemented by Native observations plus independent source proof, not separate dialect modes or renderer dependencies. YAML/TOML front matter remains a Marksplice-owned source envelope, not a reason to add a YAML/TOML parser or serializer dependency. Exact dependency versions belong in `go.mod` and `go.sum`.

When the approved specification snapshots are provisioned separately, set `MARKSPLICE_COMMONMARK_SPEC_HTML` and `MARKSPLICE_GFM_SPEC_HTML` to their HTML paths. Run the exact Native-only CommonMark/GFM contract gates documented in `docs/gfm-conformance.md`; the current anchored tests are `TestPublishedCommonMark0312Corpus`, `TestM115NativeMatchesPublishedCommonMark0312Contract`, and `TestM115NativeMatchesPublishedGFM029Contract`. Inherited GFM core examples do not override the newer CommonMark base grammar, and rendering-only `tagfilter` is outside parser conformance while Marksplice has no HTML renderer. Use anchored exact test names: an incorrect `-run` filter can select zero tests while `go test` still exits successfully. Each loader verifies the approved SHA-256 before evaluating examples; an upstream specification change requires the reviewed update process in `docs/gfm-conformance.md`, not a hash-only or mechanically regenerated-fixture update.

## Source preservation

Ordinary edits to existing Markdown must not render and replace the complete document. Prepared changes must use validated source ranges, preserve untouched bytes, and fail closed when applied to a different source snapshot.

Use byte offsets for source mutation boundaries. Be deliberate about LF/CRLF, Unicode, malformed input, duplicate human-readable labels, large inputs, and deterministic behavior.

## Releases and publication

Public module versioning, beta policy, release preparation, and publication verification are defined in [`docs/releasing.md`](docs/releasing.md). Contributors should not move or recreate published module tags. A release must be cut from the exact reviewed commit that passed the applicable verification gates.

## Scope discipline

Do not add Scripthold-specific MCP adapters or host-specific filesystem/security policy to this repository. `workspacefs` may consume only caller-supplied `fs.FS` read authority under explicit finite limits; host authorization, writes, URL fetching, commands, and security policy remain outside Marksplice. Future public API and document-intelligence work must extend the proven architecture and source-preservation invariants rather than bypassing them or promoting parser/source internals wholesale.
