# M80 — Front-Matter Construction

## Status

Complete.

## Objective

Add conservative new-document construction for the leading YAML/TOML front-matter envelope already recognized and source-mapped by Marksplice, without introducing a general YAML/TOML parser or treating front matter as GFM syntax.

## Public contract

M80 adds construction-only:

```go
type FrontMatterFieldInput struct {
    Key   string
    Value string
}

func (b *DocumentBuilder) SetYAMLFrontMatter(fields ...FrontMatterFieldInput) error
func (b *DocumentBuilder) SetTOMLFrontMatter(fields ...FrontMatterFieldInput) error
```

A builder owns at most one front-matter envelope. The setter may be called before or after block append operations, but generated front matter is always written at byte zero before the GFM body.

The input slice is defensively copied before retention.

## Canonical source policy

M80 intentionally constructs only simple string scalars:

- YAML uses `key: "value"`;
- TOML uses `key = "value"`;
- the opening/closing delimiters are `---` or `+++` respectively;
- fields preserve caller order;
- generated source uses LF;
- one blank line separates a retained envelope from a non-empty GFM body;
- a front-matter-only document ends with the closing delimiter line and one LF.

Keys use the existing conservative Marksplice key alphabet: ASCII letters, digits, `_`, `-`, and `.`. Keys must be non-empty and unique.

Values must be non-empty valid UTF-8 and must not contain CR/LF, NUL/control bytes, double quotes, or backslashes. This deliberately avoids YAML/TOML escape semantics: the exact caller bytes inside the quotes are the exact source-mapped public field value.

## Architecture

Front matter is document-envelope state on `DocumentBuilder`, not a `constructionBlock`. This preserves the established architecture that YAML/TOML metadata is outside the GFM parser and must be document-leading.

The document writer writes the envelope into the final output buffer first and then writes ordinary construction blocks into the same buffer. Block expectations therefore receive their final absolute byte ranges directly; there is no post-hoc range shifting.

Generated envelope proof reuses `internal/source.MapLeadingFrontMatter`. It requires:

- the requested YAML/TOML format;
- byte-zero ownership;
- the exact field count/order;
- exact key bytes;
- exact value bytes;
- the existing source-layer double-quoted style/quote mapping.

The proof runs when the envelope is configured and again against the complete generated document. Ordinary `validateConstructionDocument` then validates all GFM body expectations at their final offsets.

## TDD and edge cases

The initial focused test failed only because `FrontMatterFieldInput`, `SetYAMLFrontMatter`, and `SetTOMLFrontMatter` did not exist.

Focused green coverage includes:

- canonical YAML plus a heading;
- canonical TOML plus a paragraph;
- Unicode and `:`/`#` inside safely quoted values;
- front-matter-only output;
- defensive input-copy behavior;
- empty/invalid keys and values;
- duplicate keys;
- newline/CR/NUL/invalid UTF-8;
- quote/backslash rejection;
- rejecting a second same-format or different-format envelope;
- nil-builder errors;
- reparsing generated fields through the existing public `FrontMatterField` detail.

Existing M8 source/edit regressions also remain green.

## Devil's advocate review

1. **Envelope insertion could invalidate every subsequent construction range.** Mitigation: the final writer computes body expectations in the already-prefixed output buffer; it never shifts previously calculated ranges.
2. **Quoted YAML/TOML values could silently reinterpret escapes.** Mitigation: M80 does not implement escaping; quote, backslash, control, and multiline values fail before retention, and generated value bytes are checked against the existing source mapper.
3. **Multiple envelopes could create ambiguous document ownership.** Mitigation: one builder can retain exactly one envelope; every subsequent setter fails without changing the retained envelope.
4. **A new metadata syntax parser could diverge from M8.** Mitigation: there is no new parser. Construction uses a deterministic writer and `MapLeadingFrontMatter` as the parser-independent proof oracle.

## Final verification

The final M63–M80 repository gate passed on 2026-08-22 after documentation alignment:

- `gofmt` on the M80 construction/test files;
- `go test ./... -count=1`;
- `go test -race ./... -count=1`;
- `go vet ./...`;
- `staticcheck ./...`;
- `golangci-lint run` with zero issues;
- production `gocyclo -over 15 -ignore '_test\\.go$' .` with no findings;
- production and test-inclusive `unparam` with no findings;
- the pinned published GFM 0.29 conformance test;
- `govulncheck ./...` with no vulnerabilities found;
- Gitleaks with no leaks found;
- `go build ./...`;
- public `go doc` checks for `FrontMatterFieldInput`, both front-matter setters, and representative existing APIs;
- repository text hygiene over 219 relevant text files;
- `git diff --check` and `git fsck --no-dangling`.

Coverage from the final tree is 92.4% for the root package, 66.5% for the Goldmark adapter, 79.1% for `internal/source`, 57.9% for `internal/splice`, and 70.7% aggregate. The parser interface package contains no executable test target and reports 0.0%.

The final Git state remains branch `main` at HEAD `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured remotes and the M63–M80 work intentionally uncommitted. No commit or push was performed.
