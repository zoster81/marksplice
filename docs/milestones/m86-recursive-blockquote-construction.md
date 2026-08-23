# M86 — Recursive Blockquote Construction

## Status

Complete and green before the final documented repository gate.

## Objective

Close the reviewed new-document blockquote composition hierarchy by allowing `AppendBlockquoteBlocks` child builders to contain already-constructed blockquotes, without widening ordinary parsed-source blockquote promotion or introducing a second public block model.

## Public contract

M86 adds no new API. It extends the existing construction-only method:

```go
func (b *DocumentBuilder) AppendBlockquoteBlocks(depth int, content *DocumentBuilder) error
```

The outer `depth` remains 1 through 64. Child builders may now contain both single-paragraph blockquotes and multi-block blockquotes. Along every parent/child blockquote chain, structural depths are additive and the total must not exceed 64. Sibling depths are independent and are not accumulated across separate branches.

Front matter remains a document envelope rather than a body block and is still rejected as blockquote child content. Nil, empty, self-referential, or otherwise invalid child builders retain the existing fail-closed behavior.

## Architecture and test strategy

M86 reuses the M83–M85 private `constructionBlock` model, canonical writers, lexical quoted-to-inner proof, and construction-only Goldmark comparator.

Before any recursive blockquote source is written, `validateConstructionBlockquoteChildren` walks the private construction tree with an explicit stack of `constructionBlockquoteValidationFrame` values. Each frame carries the accumulated parent depth. For a blockquote child, validation adds the child's own structural depth and rejects totals above 64. Multi-block blockquote children push their child sequence onto the explicit stack; non-blockquote children reuse their existing construction validation.

The canonical writer itself is unchanged. It may recursively render nested `constructionBlockquoteBlocks`, but the pre-render validation proves that recursion is bounded to at most 64 structural levels before writing starts.

The construction-only Goldmark comparator remains iterative. `ast.Blockquote` becomes another reviewed container kind, so matching a blockquote node also schedules comparison of its child sequence. This preserves exact recursive hierarchy rather than accepting a blockquote merely because its outer type matches.

Ordinary `Adapter.Parse`, M73 existing-source blockquote promotion, and M74 removal semantics are unchanged.

## TDD evidence

The focused red run failed exactly because M85 still rejected a blockquote child as unsupported.

Focused public tests then prove:

- a depth-3 outer multi-block blockquote can contain both a depth-1 paragraph blockquote and a depth-2 multi-block blockquote with exact canonical source;
- total depth 64 is accepted and produces exactly 64 canonical `> ` prefixes;
- total depth 65 is rejected with `ErrInvalidConstruction` and does not mutate the destination builder;
- front matter remains rejected.

Focused Goldmark tests prove the recursive blockquote hierarchy is accepted when exact and rejected when the nested blockquote depth changes.

After implementation, the focused tests, `go test ./... -count=1`, production `gocyclo -over 15 -ignore '_test\.go$' .`, `unparam ./...`, and `unparam -tests ./...` all pass.

## Devil's advocate review

1. **Recursive validation/writing could grow the Go stack or become unbounded.** Tree validation is iterative and rejects any blockquote chain above 64 before the existing recursive writer runs. Writer recursion is therefore explicitly bounded.
2. **Sibling blockquotes could be incorrectly summed into one global depth budget.** Each validation frame carries only the depth of its own ancestor chain; sibling branches start from the same parent depth and are evaluated independently.
3. **A semantic proof could accept the right outer blockquote count but the wrong nested child shape.** The Goldmark comparator treats blockquotes as containers and iteratively compares their descendants as well as sibling ordering.
4. **Construction capability could accidentally widen existing-source editing.** No ordinary observation, source mapper, public parsed detail, or mutation path changes in M86; recursive support exists only in construction validation/proof.

## Verification

Pre-final-gate statement coverage on the M86–M87 tree is 92.3% for the root package, 70.1% for `internal/parser/goldmark`, 79.2% for `internal/source`, 57.5% for `internal/splice`, and 71.4% aggregate. The parser interface package reports 0.0% because it has no executable test target.

The fully documented M87 tree passed the strict completion gate and therefore also verifies M86: five consecutive `go test ./... -count=1` runs, `go test -race ./... -count=1`, coverage, `go vet ./...`, `go build ./...`, public API documentation checks, hash-pinned published GFM 0.29 conformance, Staticcheck, golangci-lint with zero issues, production `gocyclo` at the <=15 threshold, production and test-inclusive `unparam`, `govulncheck` with no vulnerabilities found, Gitleaks with no leaks, strict UTF-8/no-BOM/LF/no-trailing-whitespace hygiene over 233 repository text files, `git diff --check`, and `git fsck --no-dangling`.

## Exit decision

M86 completes the currently reviewed blockquote construction hierarchy. Broader multiline, nested, lazy-continuation, and multi-block **existing-source** blockquote ownership/editing remains a separate future review and must not be inferred from construction support.
