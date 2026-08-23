# M82 — Nested Blockquote Paragraph Construction

Status: complete.

## Objective

Extend new-document blockquote construction from M81's depth-1 single-paragraph form to explicit structural nesting without widening ordinary parsed blockquote promotion or editing.

## Contract

M82 adds two public construction entrypoints:

```go
func (b *DocumentBuilder) AppendNestedBlockquote(depth int, inlineGFM string) error
func (b *DocumentBuilder) AppendNestedBlockquoteContent(depth int, content ...Inline) error
```

`depth` is structural intent rather than caller-authored Markdown indentation or marker syntax. It must be between 2 and 64 inclusive. The existing `AppendBlockquote` and `AppendBlockquoteContent` APIs remain the depth-1 entrypoints.

Raw `inlineGFM` follows the M81 paragraph contract: non-empty valid UTF-8, canonical LF line endings, no NUL, no empty physical line, and the complete generated content must remain exactly one paragraph at the requested nesting depth. Typed content reuses the existing construction-only inline renderer before entering the same nested blockquote proof.

The writer derives every marker and emits exactly `depth` copies of `> ` before every physical content line. Caller-provided source that changes the requested hierarchy, including an additional blockquote marker, heading, list, thematic break, blank-line split, or other child-block structure, fails closed with `ErrInvalidConstruction`. A rejected append does not mutate prior builder state.

## Construction-only architecture

M82 deliberately does not broaden `internal/parser/goldmark.Adapter.Parse`, ordinary blockquote observations, the M73 public existing-source subset, or the M74 removal contract.

The existing construction block state now carries blockquote depth. The writer uses one shared blockquote path for depth 1 and nested depths, derives the canonical repeated prefix once, and records the exact content range for every physical line.

The M81 proof is generalized rather than duplicated:

- `internal/source.ValidateCanonicalNestedBlockquoteParagraph` proves exact repeated `> ` marker bytes, per-line content ownership, LF separators, and the outer range; the M81 depth-1 source validator remains as a wrapper;
- `internal/parser/goldmark.ValidateNestedBlockquoteParagraph` reparses only for construction proof and requires exactly the requested number of nested `ast.Blockquote` containers terminating in one paragraph with the expected line count, starts, and final end; the M81 top-level validator remains as a depth-1 wrapper;
- `internal/splice.ValidateConstructionNestedBlockquoteParagraph` composes those lexical and semantic proofs;
- nested expectations bypass ordinary construction node matching because ordinary parsing intentionally does not promote nested blockquotes into the existing-source public model.

This keeps lexical ownership in Marksplice, semantic GFM hierarchy validation behind the Goldmark adapter boundary, and existing-document mutation behavior unchanged.

## TDD and edge-case evidence

The initial focused public test failed to compile only because `AppendNestedBlockquote` and `AppendNestedBlockquoteContent` did not yet exist.

Focused public tests then proved:

- canonical depth-2 LF-multiline output with front matter and surrounding blocks;
- depth 64 output and rejection outside the 2–64 public range;
- typed depth-3 construction with escaped literal blockquote punctuation plus structured inline emphasis;
- Unicode and ordinary inline GFM retention;
- rejection of CR/CRLF, blank physical lines, invalid UTF-8, NUL, heading/list input, and caller content that creates additional blockquote depth;
- failed-append immutability and nil-receiver behavior;
- reparsing generated nested output does not expose the existing-source public `KindBlockquote` subset.

Focused source and Goldmark tests separately prove exact depth-sensitive marker layout, successful nested hierarchy recognition, and rejection when the actual hierarchy is deeper than requested.

## Devil's advocate review

1. **Caller content could smuggle another `>` marker and silently increase depth.** Canonical prefix writing is not sufficient by itself, so the construction-only Goldmark proof requires exactly the requested container chain and one terminating paragraph.
2. **A single-line nested blockquote could accidentally flow through the M73 depth-1 source-mapping proof.** Construction expectations route every depth greater than 1 through the separate construction-only proof and omit them from ordinary node matching.
3. **Unbounded structural depth could create unnecessary source growth and proof work.** The public API is explicitly bounded to depths 2–64; the writer therefore has deterministic linear prefix cost within a small fixed maximum.
4. **Generalizing M81 could regress depth-1 behavior.** The previous source, splice, and Goldmark validator names remain as depth-1 wrappers, while focused M81 regressions and the complete repository suite stay green.

## Final verification

The complete documented M82 tree passes the strict project gate:

- `gofmt` is stable on all M82 Go files;
- five consecutive `go test ./... -count=1` runs;
- `go test -race ./... -count=1`;
- `go vet ./...` and `go build ./...`;
- public `go doc` resolution for `AppendNestedBlockquote`, `AppendNestedBlockquoteContent`, and the existing `AppendBlockquote` API;
- the hash-pinned published GFM 0.29 conformance test;
- `staticcheck ./...`;
- standard `golangci-lint run` with zero issues;
- production `gocyclo -over 15 -ignore '_test\.go$' .` with no findings;
- production and test-inclusive `unparam` with no findings;
- `govulncheck ./...` with no vulnerabilities found;
- Gitleaks with no leaks found;
- strict UTF-8/no-BOM/LF/no-trailing-whitespace hygiene over all relevant repository text files;
- `git diff --check` and `git fsck --no-dangling`.

Statement coverage on the M82 tree is 92.6% for the public root package, 67.3% for `internal/parser/goldmark`, 79.1% for `internal/source`, and 57.7% for `internal/splice`. The parser interface package has no executable test target and reports 0.0%.

The final repository state remains branch `main` at pre-M63 HEAD `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured Git remotes and the M63–M82 work intentionally uncommitted. No commit or push was performed.

## Exit decision

M82 is complete. New-document construction now owns one parser-proven blockquote paragraph at depth 1 or explicit depth 2–64 while existing-source read/edit remains the M73/M74 one-line depth-1 subset.

The next block-composition boundary is multi-block blockquote construction: represent child blocks explicitly, define canonical marker/blank-line writing, and prove the exact resulting child hierarchy without treating caller-authored container markers as structural authority. Broader existing-source blockquote promotion remains a separate source-ownership and editing review.
