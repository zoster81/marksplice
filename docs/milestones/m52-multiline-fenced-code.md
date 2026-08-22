# Milestone M52 — Multiline Fenced-Code Capability

Status: green — conservative multiline fenced-code promotion and source-preserving replacement.

## Goal

Broaden the existing M5 `FencedCode` capability from one non-empty physical content line to a conservative multiline body without adding a new public kind, changing `NodeID` derivation, or weakening source-preserving replacement.

M52 does not introduce a second multiline-specific API. Existing public contracts remain:

```go
func (d *Document) FencedCode(id NodeID) (FencedCode, bool)
func (f FencedCode) Range() Range
func (d *Document) PrepareReplaceFencedCode(id NodeID, replacement []byte) (ChangeSet, error)
```

## Supported multiline shape

A multiline fenced-code block is publicly promoted only when Marksplice can represent its semantic body as one exact contiguous source span.

M52 therefore requires an unindented opening fence for multiline bodies. This avoids Goldmark's legal fence-indentation removal creating semantic content whose logical lines no longer correspond to one exact source slice. The existing M5 single-line capability continues to support its previously reviewed 0–3-space opening indentation.

The closing fence may retain supported indentation and a longer fence run under the existing source mapper.

`FencedCode.Range()` remains the exact operation-oriented content span. For multiline bodies:

- internal body CR, LF, or CRLF bytes are part of the range;
- the final physical line ending immediately before the closing fence is outside the range;
- opening/closing fence lines and info-string source remain outside the range.

This keeps the single-line M5 range semantics intact while making multiline replacement one contiguous patch.

## Parser and source mapping

The Goldmark adapter now observes all non-empty `FencedCodeBlock.Lines()` content lines. It records one parser-independent range from the first semantic content byte through the physical end of the last content line.

The source mapper is generalized from the historical `MapSingleLineFencedCode` name to:

```go
func MapFencedCode(input []byte, content Range) (FencedCodeMapping, error)
```

It retains the existing proof of opening/closing delimiter character, fence lengths, info range, and indentation. Multiline mapping additionally requires opening indentation zero and the content start to coincide with the first physical content-line start. Unsupported shapes remain semantic observations with `Editable=false` and are filtered from the public actionable surface.

No Goldmark type crosses the adapter boundary.

## Replacement behavior

`PrepareReplaceFencedCode` now accepts any non-empty caller-provided byte sequence rather than imposing the generic single-line precondition. Existing-document replacement remains source-preserving:

- only `FencedCode.Range()` is patched;
- original fence spelling, info string, indentation, closing line, and untouched bytes are retained;
- caller-provided internal line-ending bytes are not normalized;
- the candidate is reparsed and remapped before a change is returned.

A replacement containing a line that becomes a valid closing fence for the preserved delimiter fails closed because candidate mapping no longer reproduces the original fenced-code boundary.

## Compatibility

M52 broadens capability without changing public signatures, public kind ordinals, snapshot identity inputs, or single-line range semantics.

Previously unsupported multiline fenced code may now appear as an ordinary promoted `KindFencedCode` when it satisfies the reviewed contiguous-body rule. Callers already written to enumerate promoted capabilities should therefore treat the change as a capability expansion, not a new taxonomy entry.

## Complexity

Goldmark already enumerates fenced-code body segments. Observation is O(l) only to access first/last segment metadata, and source mapping remains linear in the bounded fence/body source needed to identify the opening and closing physical lines. Candidate replacement retains the existing single reparse/remap safety oracle.

## Devil's advocate review

### Risk: multiline replacement injects an early closing fence

Mitigation: candidate reparsing plus exact `MapFencedCode` comparison must reproduce the expected content/source/fence mapping. An early closing line changes that mapping and returns `ErrInvalidReplacement`.

### Risk: legal opening indentation makes semantic content non-contiguous in source

Mitigation: multiline promotion requires opening indentation zero. Indented multiline fences remain internal non-editable observations; existing single-line indented support is unchanged.

### Risk: widening replacement normalizes author line endings

Mitigation: parsed-document mutation accepts caller-owned CR/LF bytes verbatim and patches only the stored body range. The final original line terminator before the closing fence remains untouched.

## TDD evidence

The initial source test failed with `content crosses a physical line`; the public test could not find a promoted multiline fenced block. After generalizing observation/mapping and replacement preconditions, focused source/public tests pass for CRLF multiline content, preserved longer/indented closing fences, exact range ownership, multiline replacement, early-closing rejection, historical single-line behavior, and fail-closed indented multiline input.

A repository-wide `go test ./... -count=1` regression is green after the internal mapper rename/refactor. Final verification on the combined M52/M53 tree passes five consecutive `go test ./... -count=1` runs, `go test -race ./... -count=1`, coverage, `go vet ./...`, `go build ./...`, generated `DocumentBuilder`/`FencedCode`/replacement documentation, and the pinned published-GFM 0.29 conformance gate. `staticcheck ./...` and standard `golangci-lint run` are clean; production gocyclo includes tracked and untracked Go sources and reports no function above complexity 20 across 33 production files; production and test-inclusive unparam report no findings; `govulncheck ./...` reports no vulnerabilities; Gitleaks reports no leaks; strict UTF-8/no-BOM/LF/no-trailing-whitespace checks are clean for all 46 changed or untracked paths; `git diff --check` and `git fsck --no-dangling` pass. Final statement coverage is 92.8% for the public root package, 64.4% for `internal/parser/goldmark`, 79.3% for `internal/source`, and 66.9% for `internal/splice`.
