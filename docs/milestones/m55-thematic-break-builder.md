# Milestone M55 — Thematic-Break Construction

Status: green — canonical top-level thematic-break construction with parser-private semantic proof.

## Goal

Add a common GFM block to new-document construction without promoting a new parsed-document public kind or mutation contract.

M55 adds:

```go
func (b *DocumentBuilder) AppendThematicBreak() error
```

The method has no style argument. It writes one canonical top-level thematic break as exactly `---`.

## Parser-private construction capability

Goldmark exposes thematic-break semantics but the block is not an actionable parsed capability in Marksplice. M55 therefore appends `KindThematicBreak` only to the internal parser and splice taxonomies. Existing internal ordinals are preserved, and no public `Kind` value is added.

Goldmark's public `ThematicBreak.Pos()` identifies the physical line start; its `Lines()` collection is empty for this node. The adapter uses `Pos()` plus Marksplice-owned physical line-end scanning to produce the exact parser-independent source range. No Goldmark private field or implementation code is accessed.

The internal splice node remains `Editable=false`. It exists solely so construction validation can prove semantic ownership.

## Construction proof

The block-local and final-document passes require exactly one top-level internal `KindThematicBreak` covering the generated three hyphens. The expectation deliberately accepts the construction-only non-editable node while ordinary reviewed construction families continue to require their established editable mappings.

Blank-line separation owned by `DocumentBuilder` prevents the canonical `---` source from becoming a Setext underline for a neighboring paragraph.

## Compatibility and complexity

M55 appends internal enum values rather than inserting them, so existing internal kind ordinals and `NodeID` inputs do not change. The public parsed taxonomy is unchanged.

Writing is O(1); parser/model proof remains linear in generated block/document size under the existing construction boundary.

## Devil's advocate review

### Risk: `---` becomes Setext syntax instead of a thematic break

Mitigation: block-local proof and full-document proof both require a top-level thematic-break observation over the exact bytes; builder-owned blank separation prevents a neighboring paragraph from absorbing it as an underline.

### Risk: a construction-only parser kind leaks into public API

Mitigation: no public `Kind` is added and the internal node remains non-editable. The construction proof names it explicitly instead of routing it through public node promotion.

### Risk: Goldmark lacks a line segment for thematic breaks

Mitigation: M55 uses the public AST `Pos()` as the start anchor and Marksplice-owned physical line scanning for the end. A focused adapter test proves the exact source range.

## TDD and verification evidence

The initial red test failed to compile because `AppendThematicBreak` and `KindThematicBreak` did not exist. The first implementation using `BaseBlock.Lines()` then failed its focused semantic test because Goldmark leaves thematic-break lines empty. A bounded public-API inspection established that `Pos()` is the exact source line start; switching to that contract made the adapter and public builder tests green.

The complete `go test ./... -count=1` repository regression passes on the combined M54–M56 tree.

Final combined M54–M56 verification passes five consecutive `go test ./... -count=1` runs, `go test -race ./... -count=1`, `go vet ./...`, `go build ./...`, generated `DocumentBuilder` documentation, the pinned published-GFM 0.29 conformance gate, `staticcheck ./...`, standard `golangci-lint run` with zero issues, production gocyclo with no function above complexity 20 across 33 production files, production/test-inclusive unparam, `govulncheck ./...` with no vulnerabilities, Gitleaks with no leaks, changed/untracked UTF-8/LF/no-trailing-whitespace hygiene across 50 paths, `git diff --check`, and repository-state checks. Final statement coverage is 92.8% for the public root package, 65.2% for `internal/parser/goldmark`, 79.3% for `internal/source`, and 66.7% for `internal/splice`.
