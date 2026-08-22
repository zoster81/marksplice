# Milestone M54 — Multiline Paragraph Construction

Status: green — `DocumentBuilder.AppendParagraph` now accepts parser-proven LF-multiline paragraph source.

## Goal

Extend the M44 paragraph constructor to the multiline top-level paragraph shape already supported by the parsed document model, without adding a parallel API or normalizing caller source.

The public signature remains:

```go
func (b *DocumentBuilder) AppendParagraph(inlineGFM string) error
```

M54 broadens accepted input rather than introducing a second multiline method.

## Construction policy

Paragraph input must be non-empty valid UTF-8 GFM source, contain no NUL byte, and use LF rather than CR or CRLF. LF bytes may occur inside the requested paragraph.

The complete requested bytes must reparse as exactly one top-level paragraph occupying the exact generated content range. A blank line, list marker, thematic break, fenced block, heading, or any other structure that splits or reclassifies the input fails closed with `ErrInvalidConstruction`.

The LF-only rule is construction policy for source that does not yet exist; it does not alter existing-document source-preservation behavior.

## Parser/model proof

M54 reuses the established paragraph construction expectation and the same parser semantics already used by `PrepareReplaceParagraph`. The generated paragraph must remain one editable top-level `KindParagraph` and its exact `Range` must equal the caller-owned paragraph bytes.

No new parser kind, public type, `NodeID` input, or source mapper is required.

## Compatibility and complexity

All previously valid M44 single-line paragraph calls remain valid and produce identical source. Validation and writing remain O(n) in requested content and retain the existing block-local plus final-document parser proof.

## Devil's advocate review

### Risk: embedded LF creates multiple blocks

Mitigation: the exact generated range must reparse as one top-level paragraph. Structural splits or reclassification are rejected rather than escaped or repaired.

### Risk: CRLF input is silently normalized

Mitigation: paragraph construction rejects every CR byte. Callers requesting new source must provide canonical LF explicitly.

### Risk: creation exceeds parsed paragraph capability

Mitigation: M54 deliberately reuses the existing parsed paragraph semantics and exact range proof; the builder cannot retain output that the ordinary document model does not reproduce as one top-level paragraph.

## TDD and verification evidence

The pre-existing invalid-construction test classified LF paragraph input as invalid. M54 changes that expectation and adds a focused canonical multiline test that reparses the generated source and verifies the exact public `Paragraph.Range()` bytes.

Focused M54 tests pass after paragraph-specific validation replaced the generic single-line validator. The complete `go test ./... -count=1` repository regression also passes on the combined M54–M56 tree after the final construction-proof refactor.

Final combined M54–M56 verification passes five consecutive `go test ./... -count=1` runs, `go test -race ./... -count=1`, `go vet ./...`, `go build ./...`, generated `DocumentBuilder` documentation, the pinned published-GFM 0.29 conformance gate, `staticcheck ./...`, standard `golangci-lint run` with zero issues, production gocyclo with no function above complexity 20 across 33 production files, production/test-inclusive unparam, `govulncheck ./...` with no vulnerabilities, Gitleaks with no leaks, changed/untracked UTF-8/LF/no-trailing-whitespace hygiene across 50 paths, `git diff --check`, and repository-state checks. Final statement coverage is 92.8% for the public root package, 65.2% for `internal/parser/goldmark`, 79.3% for `internal/source`, and 66.7% for `internal/splice`.
