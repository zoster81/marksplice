# M81 — Multiline Blockquote Paragraph Construction

## Status

Complete.

## Objective

Extend `DocumentBuilder.AppendBlockquote` from the M56 single-line construction subset to one canonical top-level blockquote containing exactly one LF-multiline paragraph. Existing parsed blockquote read/edit support remains limited to the M73/M74 simple single-line subset.

## Contract

M81 adds no new public method. `AppendBlockquote` now accepts non-empty valid UTF-8 paragraph GFM with LF line endings and writes canonical `> ` on every physical line. CR, NUL, invalid UTF-8, empty physical lines, and input that reparses into broader block structure fail with `ErrInvalidConstruction`.

The writer retains one exact content range per physical line instead of treating the intervening newline and blockquote markers as paragraph content.

## Architecture

M81 does not broaden `internal/parser/goldmark.Adapter.Parse` or the ordinary `observeBlockquote` path. Multiline construction uses a separate internal proof path: the source layer proves canonical markers, LF separators, line ranges, and the outer range; a Goldmark construction helper proves one top-level blockquote with exactly one paragraph and the requested semantic line starts/count/final end; an internal splice helper composes both proofs.

Single-line construction continues to use the existing M73 public source-mapping proof. Multiline expectations are validated separately and omitted only from the ordinary construction node-matching sequence because ordinary parsing intentionally does not promote them.

## TDD and risks

The initial focused test failed with the pre-M81 single-line validation error. Focused tests now cover canonical multiline output with front matter and surrounding blocks, Unicode/inline GFM, unchanged existing-source public promotion, invalid CR/blank/structural/nested inputs, failed-append immutability, exact source layout proof, and Goldmark rejection of multiple child blocks.

Key mitigations are: do not add multiline observations to normal parsing because mutation survivor checks compare ordinary internal nodes; keep per-line ranges because multiline paragraph bytes are discontiguous in source; keep lexical ownership in Marksplice because intermediate Goldmark segment ends can extend across continuation parsing; and fail closed when raw content becomes a heading, list, thematic break, nested blockquote, or multi-block container.

## Final verification

Focused public/source/Goldmark/splice tests pass. The final repository gate passed on 2026-08-22 after code and documentation alignment:

- `go test ./... -count=1` and `go test -race ./... -count=1`;
- `go vet ./...`, `staticcheck ./...`, and `golangci-lint run` with zero issues;
- production `gocyclo -over 15 -ignore '_test\\.go$' .` with no findings;
- production and test-inclusive `unparam` with no findings;
- pinned published GFM 0.29 conformance;
- `govulncheck ./...` with no vulnerabilities found;
- Gitleaks with no leaks found;
- `go build ./...` and public `go doc` checks including `DocumentBuilder.AppendBlockquote`;
- text hygiene over 224 relevant repository text files;
- `git diff --check` and `git fsck --no-dangling`.

Coverage on the final M81 tree is 92.6% for the root package, 69.1% for the Goldmark adapter, 79.1% for `internal/source`, 57.7% for `internal/splice`, and 71.0% aggregate. The parser interface package has no executable test target and reports 0.0%.

An initial source validator measured complexity 17 and was refactored into paragraph orchestration plus a focused per-line validator until the established <=15 production gate was restored.

The final Git state remains branch `main` at HEAD `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured remotes and the M63–M81 work intentionally uncommitted. No commit or push was performed.
