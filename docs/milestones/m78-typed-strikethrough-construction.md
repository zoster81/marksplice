# M78 — Typed Strikethrough Construction

## Status

Complete and green.

## Objective

Complete the simple typed delimiter family by adding construction-only GFM strikethrough through the same source-proven path already used by M76 emphasis and strong emphasis.

## Public contract

```go
func StrikethroughInline(content ...Inline) Inline
```

M78 accepts only `TextInline` children and defensively copies the caller slice. Generated source uses canonical `~~` delimiters. Empty content and structured nested children remain fail-closed rather than guessing delimiter interactions.

## Architecture and reuse

The M76 emphasis writer was generalized into one private delimited-inline writer shared by:

- `EmphasisInline` → `*...*`;
- `StrongInline` → `**...**`;
- `StrikethroughInline` → `~~...~~`.

All three families reuse M75 semantic-text escaping for children and the same standalone typed-inline reparse proof. M78 adds no new source mapper or parser metadata: candidate validation requires the existing M6 promoted/source-proven `Strikethrough` detail to reproduce the exact generated content range.

## Risks and mitigations

1. **Delimiter interaction with nested structured children could produce a different GFM tree.** M78 retains the text-only child boundary; broader nesting remains a separate review.
2. **Leading/trailing whitespace can change delimiter flanking semantics.** The standalone reparse/source proof is authoritative and rejects shapes that do not reproduce the exact requested `KindStrikethrough` content range.
3. **Duplicating emphasis/strong logic would raise maintenance complexity.** M78 reuses one delimited writer instead of adding a separate strikethrough renderer.

## TDD evidence

The red run failed only on the intentionally missing `StrikethroughInline` constructor. Focused green tests cover exact canonical source, parsed public content range, defensive child copying, empty content, structured-child rejection, and whitespace shapes that fail to reproduce the source-proven simple strikethrough capability.

After implementation, the complete typed-inline regression passes and production `gocyclo -over 20` remains empty. The shared delimited writer reports cyclomatic complexity 6.

Final repository-wide verification on the documented uncommitted M63–M79 tree passed `gofmt`, `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `staticcheck ./...`, `golangci-lint run` with zero issues, production `gocyclo -over 20` with no functions above the threshold, production and test-inclusive `unparam` with no findings, the hash-pinned published GFM 0.29 conformance test, `govulncheck ./...` with no vulnerabilities found, Gitleaks with no leaks found, `go build ./...`, and public `go doc` resolution for the M75–M79 typed-inline API. Coverage was 92.6% for the root package, 68.6% for the Goldmark adapter, 78.7% for `internal/source`, 57.9% for `internal/splice`, and 70.5% aggregate; the parser interface package has no direct tests. Strict UTF-8/no-BOM/LF/no-trailing-whitespace hygiene passed on 60 changed/untracked text paths, followed by `git diff --check` and `git fsck --no-dangling`.

The final repository state remained branch `main` at pre-M63 HEAD `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured Git remote and only the intended M63–M79 working-tree changes.

No commit or push was performed.
