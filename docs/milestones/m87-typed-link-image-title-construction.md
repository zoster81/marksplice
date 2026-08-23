# M87 — Typed Link and Image Title Construction

## Status

Complete and green before the final documented repository gate.

## Objective

Extend the existing typed-inline link/image construction path with conservative canonical titles while reusing the established source-proven inline-link/image mappings and keeping ordinary parsing/editing unchanged.

## Public contract

M87 adds two construction-only helpers:

```go
func LinkInlineWithTitle(destination, title string, label ...Inline) Inline
func ImageInlineWithTitle(destination, title string, alt ...Inline) Inline
```

The existing `LinkInline` and `ImageInline` behavior is unchanged. All four constructors defensively clone their `Inline` child values through the same shared private helper.

M87 emits canonical angle destinations followed by one separated double-quoted title:

```text
[label](<destination> "title")
![alt](<destination> "title")
```

The title must be non-empty valid UTF-8 on one physical line, contain no NUL, and contain none of double quote, backslash, or `&`. This deliberately narrower policy avoids title escaping and entity interpretation so requested title bytes remain identical to the source-proven title range.

Link labels and image alt text remain `TextInline`-only under the M77 contract. Structured label/alt nesting and reference-style typed forms remain separate future work.

## Architecture and test strategy

M87 does not add a parser observation, source mapper, parsed public detail, or mutation operation.

The private `Inline` intent value gains title text plus a presence bit. `LinkInline`, `ImageInline`, and their new title variants share `newTypedLinkOrImage`, preventing duplicated construction state.

`writeTypedInlineLinkOrImage` keeps the M77 canonical angle-destination writer and conditionally writes ` "title"` before the closing parenthesis. The typed expectation records exact title range, title value, and title presence in addition to the existing destination/label facts.

Candidate proof reuses the existing parsed-source mappings:

- inline links already expose semantic title plus `InlineLinkSource.TitleRange` and `HasTitle`;
- images deliberately avoid Goldmark-private destination/title state, so M87 follows the M77 boundary and reads exact destination/title bytes from `ImageSource.DestinationRange` and `ImageSource.TitleRange`.

The ordinary typed expectation equality then requires the generated candidate to reproduce all requested title facts exactly.

## TDD evidence

The red run failed to compile only because `LinkInlineWithTitle` and `ImageInlineWithTitle` did not yet exist.

Focused green tests prove canonical output for one titled link and one titled image in the same paragraph. Negative tests reject, for both constructors:

- empty titles;
- LF-multiline titles;
- NUL;
- invalid UTF-8;
- literal double quotes;
- backslashes;
- ampersand/entity-like content.

The complete pre-existing typed-inline construction test set also passes, proving no-title constructors retain their previous canonical bytes and semantics.

After implementation, `go test ./... -count=1`, production `gocyclo -over 15 -ignore '_test\.go$' .`, `unparam ./...`, and `unparam -tests ./...` pass together with M86.

## Devil's advocate review

1. **A title could be normalized by GFM escaping or entity decoding.** M87 rejects bytes that would require those interpretations and then requires exact source-mapped title value/range after reparsing.
2. **Goldmark does not expose image destination/title as the same public semantic fields used for links.** M87 does not inspect private Goldmark state; it reuses the already-reviewed Marksplice `ImageMapping` and reads exact source bytes, matching the M77 trust boundary.
3. **Adding title support could silently change existing no-title output.** Existing constructors use the same shared helper with `hasTitle=false`; the full typed-inline regression remains byte-identical and green.
4. **New constructors could encourage another parallel inline AST.** M87 extends the existing private `Inline` intent and expectation engine only; no second parser/source model is introduced.

## Verification

Pre-final-gate statement coverage on the M86–M87 tree is 92.3% for the root package, 70.1% for `internal/parser/goldmark`, 79.2% for `internal/source`, 57.5% for `internal/splice`, and 71.4% aggregate. The parser interface package reports 0.0% because it has no executable test target.

The fully documented M87 tree passed the strict M86–M87 completion gate: five consecutive `go test ./... -count=1` runs, `go test -race ./... -count=1`, coverage, `go vet ./...`, `go build ./...`, public `go doc` resolution for `AppendBlockquoteBlocks`, `LinkInlineWithTitle`, `ImageInlineWithTitle`, `LinkInline`, and `ImageInline`, hash-pinned published GFM 0.29 conformance, Staticcheck, golangci-lint with zero issues, production `gocyclo` at the <=15 threshold, production and test-inclusive `unparam`, `govulncheck` with no vulnerabilities found, Gitleaks with no leaks, strict UTF-8/no-BOM/LF/no-trailing-whitespace hygiene over 233 repository text files, `git diff --check`, and `git fsck --no-dangling`.

The verified repository state remained branch `main` at HEAD `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured remotes and the M63–M87 work intentionally uncommitted.

## Exit decision

M87 completes conservative typed link/image title construction while preserving the original M77 constructors and parsed-source contracts. The next high-leverage construction boundary is broader structured typed-inline composition or reference-style typed forms, preferably by reusing the existing typed expectation engine rather than introducing a second inline AST.
