# M75 — Typed Semantic-Text Inline Construction

## Status

Complete and green.

## Objective

Open the typed-inline construction path without changing the existing raw-GFM construction APIs or introducing a persistent construction AST. M75 adds semantic plain-text input that cannot accidentally become Markdown syntax.

## Public contract

```go
type Inline struct { /* construction-only */ }

func TextInline(text string) Inline

func (b *DocumentBuilder) AppendHeadingContent(level int, content ...Inline) error
func (b *DocumentBuilder) AppendParagraphContent(content ...Inline) error
func (b *DocumentBuilder) AppendBlockquoteContent(content ...Inline) error
```

`Inline` is construction-only and has an invalid zero value. Existing `AppendHeading`, `AppendParagraph`, and `AppendBlockquote` continue to accept caller-owned raw GFM exactly as before.

## Text semantics and canonical writing

`TextInline` means semantic text, not raw Markdown. The typed writer backslash-escapes every escapable ASCII punctuation byte before passing the generated inline GFM into the existing block-construction proof. This intentionally favors unambiguous deterministic source over minimal escaping because the source is new and has no author formatting to preserve.

The text value must be non-empty valid UTF-8, remain on one physical line, and contain no NUL byte. CR/LF input and the zero `Inline` value fail with `ErrInvalidConstruction`; Marksplice does not silently normalize them.

## Architecture and reuse

M75 does not add another document or AST model. Typed values are rendered immediately into canonical inline GFM, then the existing heading/paragraph/blockquote APIs retain their established complete-block reparse proof. Parsed-document APIs and existing-document source-preserving mutations are unchanged.

This establishes an explicit API distinction:

- raw-GFM construction methods are for callers that already own valid inline GFM source;
- typed-inline construction methods are for semantic intent whose Markdown syntax Marksplice writes deterministically.

## Risks and mitigations

1. **Semantic text could be reinterpreted as Markdown.** Canonical punctuation escaping prevents text such as `*`, `[`, `#`, or `!` from silently becoming inline/block syntax.
2. **Typed construction could accidentally normalize existing source.** The APIs exist only on `DocumentBuilder`; existing parsed snapshots and mutations never route through this renderer.
3. **A second construction AST could duplicate block state.** `Inline` is transient construction input; rendered GFM is delegated to the existing block writer/proof rather than retained as a parallel authoritative document model.
4. **Caller mutation could alter retained input.** Structured values are copied where they contain variable-length child storage; M75 text itself is immutable Go string data.

## TDD evidence

The red test failed only on the intentionally missing `Inline`, `TextInline`, and typed append methods. Focused green tests cover canonical punctuation escaping, Unicode text, heading/paragraph/blockquote integration, invalid UTF-8, NUL and physical-line rejection, zero-value input, nil builders, and preservation of the existing raw-GFM APIs.

Final repository-wide verification on the documented uncommitted M63–M76 tree passed `gofmt`, `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `staticcheck ./...`, `golangci-lint run` with zero issues, production `gocyclo -over 20` with no functions above the threshold, production and test-inclusive `unparam` with no findings, the hash-pinned published GFM 0.29 conformance test, `govulncheck ./...` with no vulnerabilities found, Gitleaks with no leaks found, `go build ./...`, and public `go doc` resolution for the typed-inline API. Coverage was 92.7% for the root package, 68.6% for the Goldmark adapter, 78.7% for `internal/source`, 57.9% for `internal/splice`, and 70.2% aggregate; the parser interface package has no direct tests. Strict UTF-8/no-BOM/LF/no-trailing-whitespace hygiene passed on 57 changed/untracked text paths, followed by `git diff --check` and `git fsck --no-dangling`.

The final repository state remained branch `main` at pre-M63 HEAD `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured Git remote and only the intended M63–M76 working-tree changes.

No commit or push was performed.
