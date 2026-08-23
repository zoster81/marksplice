# M76 — Typed Code, Emphasis, and Strong Construction

## Status

Complete and green.

## Objective

Extend M75 typed-inline construction with the first structured inline families while preserving the parsed source-proven capability boundary and keeping the renderer small enough to extend safely.

## Public contract

```go
func CodeInline(code string) Inline
func EmphasisInline(content ...Inline) Inline
func StrongInline(content ...Inline) Inline
```

These values compose with the M75 `AppendHeadingContent`, `AppendParagraphContent`, and `AppendBlockquoteContent` entrypoints.

## Canonical source policy

`CodeInline` chooses a backtick fence one byte longer than the longest internal backtick run. It currently rejects empty content, invalid UTF-8, CR/LF, NUL, leading/trailing horizontal space, and leading/trailing backticks because those source shapes would require the whitespace/delimiter normalization that the parsed M6 `CodeSpan` capability deliberately excludes.

`EmphasisInline` writes `*content*`; `StrongInline` writes `**content**`. M76 intentionally accepts only `TextInline` children inside those wrappers. Nested structured content is deferred rather than exposing delimiter interactions that have not yet received a dedicated construction contract.

## Standalone semantic proof

Rendering the requested bytes is not sufficient. Before the GFM is handed to a block constructor, the typed-inline candidate is reparsed as exactly one paragraph and must reproduce every requested source-proven `CodeSpan`, `Emphasis`, or `Strong` content range. Any unexpected promoted public kind fails with `ErrInvalidConstruction`.

The proof is Marksplice-owned and consumes only public/root abstractions; no Goldmark type enters the public API and no Goldmark rendering is used.

## Defensive ownership

`EmphasisInline` and `StrongInline` recursively copy their caller-provided child slices. Mutating the caller's input slice after construction therefore cannot change retained typed intent.

## Complexity and consolidation review

Restoring the project's historical `gocyclo`/`unparam` gate during M76 exposed three unrelated production hotspots: the M72 whole-block survivor semantic comparison at complexity 30 and the growing internal/public kind-conversion switches at 21/22. Before M76 exit:

- the survivor comparison was replaced by one comparable scalar semantic signature plus the existing explicit alignment-slice comparison;
- parser→splice and splice→public kind conversion use bounded enum-indexed lookup tables with explicit unknown/unmapped rejection;
- the typed-inline writer and validator were split into family-specific writer, candidate-collection, node-expectation, and expectation-matching helpers before adding further inline families.

Focused regression proves the refactor is behavior-preserving. Production `gocyclo -over 20` is empty; the reported production average is 5.03, and production plus test-inclusive `unparam` report no findings. The typed-inline functions no longer appear in the production top-15 complexity list.

## Risks and mitigations

1. **Adaptive code fences could still trigger CommonMark normalization rules.** M76 rejects boundary whitespace/backtick shapes and requires exact parsed source mapping of the generated semantic content.
2. **Emphasis delimiter interactions can be context-sensitive.** Structured nesting is intentionally rejected and every generated wrapper must reproduce one exact source-proven public node.
3. **A growing validator could become a complexity hotspot.** Responsibility-specific helpers were introduced before the next family; the restored complexity gate remains an exit criterion.
4. **Lookup-table conversion could accidentally publish unsupported kinds.** Zero/unmapped entries remain `KindUnknown`, bounds are checked, and opaque HTML is intentionally not entered in the public mapping table.

## TDD evidence

The red test failed only on the intentionally missing `CodeInline`, `EmphasisInline`, and `StrongInline` constructors. Focused green tests cover mixed typed text/code/emphasis/strong output, internal backtick runs, exact parsed public ranges, defensive child copying, zero/empty/normalization-prone shapes, unsupported nesting, and the existing parsed simple-inline regression suite.

Final repository-wide verification on the documented uncommitted M63–M76 tree passed `gofmt`, `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `staticcheck ./...`, `golangci-lint run` with zero issues, production `gocyclo -over 20` with no functions above the threshold, production and test-inclusive `unparam` with no findings, the hash-pinned published GFM 0.29 conformance test, `govulncheck ./...` with no vulnerabilities found, Gitleaks with no leaks found, `go build ./...`, and public `go doc` resolution for all M75–M76 typed-inline exports. Coverage was 92.7% for the root package, 68.6% for the Goldmark adapter, 78.7% for `internal/source`, 57.9% for `internal/splice`, and 70.2% aggregate; the parser interface package has no direct tests. Strict UTF-8/no-BOM/LF/no-trailing-whitespace hygiene passed on 57 changed/untracked text paths, followed by `git diff --check` and `git fsck --no-dangling`.

The final repository state remained branch `main` at pre-M63 HEAD `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured Git remote and only the intended M63–M76 working-tree changes.

No commit or push was performed.
