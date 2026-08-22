# Milestone M50 — Reference-Definition Construction

Status: green — canonical single-line reference-definition construction without titles.

## Goal

Reuse the reviewed M7 reference-definition mapping to construct new link reference definitions without exposing source-style options or silently normalizing caller data.

M50 adds:

```go
func (b *DocumentBuilder) AppendReferenceDefinition(label, destination string) error
```

## Canonical source policy

M50 writes exactly:

```markdown
[label]: <destination>
```

Rules:

- labels and destinations are non-empty single-line valid UTF-8 strings;
- NUL is rejected;
- labels containing `[` or `]` are rejected rather than escaped;
- destinations containing `<` or `>` are rejected because M50 owns canonical angle-bracket destination spelling and does not synthesize escaping;
- titles are not part of M50;
- indentation, alternate raw-destination spelling, trailing spaces, and title style controls remain outside the construction API.

Caller input remains semantic/source intent. If Goldmark or the Marksplice source mapper would interpret the generated label or destination differently, the append fails rather than returning normalized source.

## Parser/model proof

M50 reuses the existing M7 `ReferenceDefinitionMapping`. The generated node must be an editable `KindReferenceDefinition` with:

- exact complete definition range;
- exact destination range;
- exact semantic label and destination;
- no semantic title;
- angle-bracket destination mapping;
- no mapped title range.

No reference-definition identity is created during construction. Ordinary snapshot-scoped `NodeID` values appear only after the caller passes successful output to `Parse`.

## Failure behavior

Empty, multiline, invalid-UTF-8, NUL-containing, structurally unsafe label/destination input, nil builders, or parser/mapping disagreement fails with `ErrInvalidConstruction`. Rejected appends leave the builder unchanged.

M50 does not perform relationship-wide reference resolution or require that a matching reference use already exists; it constructs one independently valid definition block.

## Complexity

Input validation and writing are O(k) in label+destination size. The established construction parser/model proof remains O(n) in generated document size.

## Devil's advocate review

### Parser label normalization could make source and caller intent disagree

Mitigation: proof compares the reparsed semantic label and destination to the exact requested strings as well as the lossless M7 mapping.

### A broad options struct would prematurely freeze destination/title style policy

Mitigation: M50 exposes only the smallest durable operation: canonical angle destination and no title. Title construction and source-style controls remain separate future decisions.

### Escaping unsafe caller data could hide semantic changes

Mitigation: M50 never manufactures escaping. Unsupported bracket/angle forms fail closed.

## TDD and verification evidence

M50 tests were introduced in the same red slice as M49. The initial compile failed only because `AppendReferenceDefinition` did not exist. After implementation, focused M49/M50 tests and the complete construction regression set pass.

M50 adds no parser metadata, public kind, dependency, `NodeID` change, or existing-document rendering path.

Historical note: M61 later adds `AppendReferenceDefinitionWithTitle` for one canonical double-quoted title shape already supported by the M7 source mapping. The M50 two-argument method and its exact no-title output remain unchanged.
