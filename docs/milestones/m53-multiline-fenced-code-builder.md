# Milestone M53 — Multiline Fenced-Code Construction

Status: green — canonical multiline fenced-code construction converged with M52 parsed capability.

## Goal

Extend the existing M49 `DocumentBuilder.AppendFencedCode` operation to multiline content only after M52 establishes the corresponding parsed/editing capability.

The public signature is unchanged:

```go
func (b *DocumentBuilder) AppendFencedCode(content, info string) error
```

M53 therefore broadens accepted structured intent rather than introducing a parallel multiline method or options type.

## Canonical source policy

Construction writes one unindented backtick-fenced block. Content must be:

- non-empty;
- valid UTF-8;
- NUL-free;
- either one line or multiple lines separated by LF;
- free of CR bytes.

The LF-only rule is construction policy, not an existing-document normalization rule. A new document has no prior author bytes to preserve, while M52 replacement continues to accept caller-owned CR/LF bytes verbatim.

`info` remains optional, valid UTF-8, NUL-free, single-line source and may not contain a backtick.

## Adaptive fence selection

M49 selected a fence longer than a competing backtick run at the start of its single content line. M53 generalizes that rule across the entire body.

For every LF-delimited content line, the writer examines a potential GFM closing-fence position after zero through three leading spaces. The canonical fence length is:

- three backticks when no potentially closing body run reaches three;
- otherwise one longer than the maximum observed leading backtick run.

For example, body source containing:

```text
line one
  ````
line three
```

is written with five-backtick opening and closing fences. The scan is deliberately conservative: a body run may cause a longer fence even when trailing source would prevent it from being a legal closing fence. This trades minimal delimiter length for deterministic fail-safe writing.

## Parser/model proof

M53 reuses the M49 family proof but now against M52 `MapFencedCode` semantics. The generated block must reparse as one promoted fenced-code node with:

- exact caller content range;
- exact complete generated fence mapping;
- backtick delimiter;
- opening and closing fence lengths equal to the canonical selected length;
- opening/closing indentation zero;
- exact info range.

The complete `DocumentBuilder.Markdown()` pass repeats proof in full-document context, so interaction with neighboring requested blocks still fails closed rather than trusting the standalone write.

No snapshot `NodeID`, fingerprint, public fence-style option, or render round trip is introduced during construction.

## Compatibility

M49 single-line calls produce the same canonical source as before except that fence selection is now implemented by the generalized all-lines scanner. M53 only expands accepted `content`; CR-containing multiline input that would require normalization is rejected with `ErrInvalidConstruction`.

## Complexity

Fence selection is O(n) in content bytes. Writing is O(n), and parser/model validation remains O(d) in the generated document size under the established construction proof boundary. No persistent construction AST or secondary fence index is added.

## Devil's advocate review

### Risk: an internal body line closes the generated fence early

Mitigation: the writer scans every line at valid 0–3-space closing-fence indentation and chooses a strictly longer backtick run; the final parser/model proof is still authoritative.

### Risk: accepting CRLF construction silently normalizes code payload bytes

Mitigation: construction accepts LF separators only. CR/CRLF content fails explicitly rather than being rewritten.

### Risk: creation outpaces the parsed model again

Mitigation: M53 was implemented only after M52 made the same multiline shape public/editable through `Parse`. Generated output is reparsed through that exact capability before being returned.

## TDD evidence

The initial construction test failed with `inline GFM must stay on one physical line`. After introducing fenced-content-specific validation and all-lines adaptive fence selection, the test writes and reparses exact multiline LF content containing an indented four-backtick body line using five-backtick fences. Invalid tests retain empty/NUL/invalid-UTF-8 rejection and explicitly reject CR or CRLF construction content.

Focused fenced-code tests and a repository-wide `go test ./... -count=1` regression are green. Final verification on the combined M52/M53 tree passes five consecutive `go test ./... -count=1` runs, `go test -race ./... -count=1`, coverage, `go vet ./...`, `go build ./...`, generated `DocumentBuilder`/`FencedCode`/replacement documentation, and the pinned published-GFM 0.29 conformance gate. `staticcheck ./...` and standard `golangci-lint run` are clean; production gocyclo includes tracked and untracked Go sources and reports no function above complexity 20 across 33 production files; production and test-inclusive unparam report no findings; `govulncheck ./...` reports no vulnerabilities; Gitleaks reports no leaks; strict UTF-8/no-BOM/LF/no-trailing-whitespace checks are clean for all 46 changed or untracked paths; `git diff --check` and `git fsck --no-dangling` pass. Final statement coverage is 92.8% for the public root package, 64.4% for `internal/parser/goldmark`, 79.3% for `internal/source`, and 66.9% for `internal/splice`.
