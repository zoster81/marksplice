# Milestone M49 — Single-Line Fenced-Code Construction

Status: green — parser-proven construction for the supported M5 fenced-code shape. M53 later broadens the same method to the M52-reviewed LF-multiline shape without changing this historical M49 contract.

## Goal

Extend `DocumentBuilder` with fenced code without creating a construction capability that cannot round-trip into the existing reviewed parsed-document surface.

M49 adds:

```go
func (b *DocumentBuilder) AppendFencedCode(content, info string) error
```

The operation constructs one top-level fenced code block whose content is exactly one non-empty physical line, matching the source-mapped `FencedCode` shape already promoted by M5.

## Scope and canonical source policy

M49 deliberately does not broaden parsed `FencedCode` support to multiline content. A constructor that emitted a valid multiline fence while `Parse` intentionally kept that shape outside the public `FencedCode` surface would make construction and snapshot capabilities disagree.

Canonical construction rules are:

- opening and closing fences use backticks;
- fences are unindented;
- the minimum fence length is three;
- if content begins with a backtick run of length three or greater, the emitted fence is one byte longer than that run;
- `content` is non-empty, valid UTF-8, NUL-free, and contains no physical line break;
- `info` is optional raw GFM info-string source, valid UTF-8, NUL-free, single-line, and contains no backtick;
- LF and inter-block separation remain owned by the shared construction writer.

For example, content consisting of four backticks is safely enclosed by five-backtick fences rather than rejected or escaped.

## Parser/model proof

M49 reuses the existing M5 `FencedCodeMapping`; no parser field or public type is added.

After writing, the generated node must be an editable `KindFencedCode` with:

- exact requested content range;
- exact complete source mapping from opening fence through closing fence;
- exact info-string range;
- backtick fence character;
- expected opening and closing fence lengths;
- zero opening and closing indentation.

Construction fails with `ErrInvalidConstruction` if the semantic parser or source mapper disagrees with any generated boundary or property.

## Failure behavior

Empty/multiline/invalid-UTF-8/NUL-containing content, invalid info strings, nil builders, or any generated shape that does not reparse to the requested supported fenced-code mapping fails closed. A rejected append does not mutate retained builder state.

## Complexity

Fence selection scans only the initial backtick run of the one-line content. Writing is O(k) in emitted source size and final validation remains O(n) in generated document size.

## Devil's advocate review

### Code content could accidentally terminate its own fence

Mitigation: the canonical backtick fence grows beyond a competing leading backtick run, and the existing source mapper must reproduce the exact opening/closing lengths.

### Construction could silently promise multiline support that parsed snapshots do not expose

Mitigation: M49 intentionally accepts only the already-reviewed M5 single-line content shape. Multiline fenced-code construction remains deferred until its parsed public semantics are independently reviewed.

### Info-string syntax could change the generated block shape

Mitigation: backticks, line breaks, NUL, and invalid UTF-8 are rejected before writing; the complete M5 mapping is then reparsed and compared exactly.

## TDD and verification evidence

The focused public tests were written first. The red run failed to compile only because `AppendFencedCode` did not yet exist. After implementation, focused M49/M50 tests and the complete `TestPublicDocumentBuilder` regression set pass.

M49 changes no `Kind` ordinal, `NodeID` derivation, parser metadata, dependency, or existing-document source-preserving mutation contract.
