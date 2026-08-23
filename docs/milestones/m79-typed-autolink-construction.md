# M79 — Typed Autolink Construction

## Status

Complete and green.

## Objective

Add canonical typed construction for source-proven angle autolinks and close the current simple typed-inline roadmap without broadening bare/extended-autolink syntax or exposing parser-specific classification.

## Public contract

```go
func AutoLinkInline(value string) Inline
```

The constructor represents a requested autolink token. M79 writes exactly `<value>`. The value must be non-empty valid UTF-8 on one physical line, contain no NUL, and contain no literal `<` or `>`.

M79 does not ask callers to classify the value as URI versus email. The semantic parser remains responsible for that GFM distinction; construction succeeds only when the rendered token reparses as the existing promoted/source-proven `AutoLink` capability with the exact requested value.

## Architecture and proof

M79 reuses the existing M7 autolink observation/mapping and the M75–M78 typed-inline expectation engine. The generated expectation records `KindAutoLink`, the exact inner content range, the requested value, and angle-token ownership. Candidate extraction requires:

- a public/source-proven `AutoLink` detail;
- the exact content range;
- `internal.Value` equal to the requested value;
- `AutoLinkSource.Angle == true`.

A string that reparses as ordinary text, raw HTML, or another construct therefore cannot satisfy the typed autolink proof. Bare/extended GFM autolink generation is intentionally not introduced.

## Risks and mitigations

1. **Arbitrary angle text could be mistaken for a valid autolink.** The constructor's lexical checks are only a prefilter; the existing semantic/source-mapped autolink proof is authoritative.
2. **URI/email classification could leak parser-specific API surface.** M79 does not expose a classification parameter or new public enum; it only requires the requested token to be a valid GFM autolink.
3. **Angle delimiters inside the value could terminate the token early.** Literal `<` and `>` are rejected before rendering.

## TDD evidence

The red run failed only on the intentionally missing `AutoLinkInline` constructor. Focused green tests cover canonical URI and email autolinks, exact public content ranges, and fail-closed rejection of empty values, ordinary text, non-autolink `www` text, multiline/NUL/invalid-UTF-8 input, and embedded angle brackets.

After implementation, all typed-inline tests and the complete repository regression pass. Production `gocyclo -over 20` remains empty, production/test-inclusive `unparam` reports no findings, and the highest typed-inline function complexity is 12.

Final repository-wide verification on the documented uncommitted M63–M79 tree passed `gofmt`, `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `staticcheck ./...`, `golangci-lint run` with zero issues, production `gocyclo -over 20` with no functions above the threshold, production and test-inclusive `unparam` with no findings, the hash-pinned published GFM 0.29 conformance test, `govulncheck ./...` with no vulnerabilities found, Gitleaks with no leaks found, `go build ./...`, and public `go doc` resolution for the M75–M79 typed-inline API. Coverage was 92.6% for the root package, 68.6% for the Goldmark adapter, 78.7% for `internal/source`, 57.9% for `internal/splice`, and 70.5% aggregate; the parser interface package has no direct tests. Strict UTF-8/no-BOM/LF/no-trailing-whitespace hygiene passed on 60 changed/untracked text paths, followed by `git diff --check` and `git fsck --no-dangling`.

The final repository state remained branch `main` at pre-M63 HEAD `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured Git remote and only the intended M63–M79 working-tree changes.

No commit or push was performed.
