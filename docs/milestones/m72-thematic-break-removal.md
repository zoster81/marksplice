# M72 — Source-Preserving Thematic-Break Removal

## Status

Complete and green.

## Objective

Add exact removal of one promoted top-level thematic-break physical line while failing closed when deleting that line changes surviving Markdown structure.

## Public contract

```go
func (d *Document) PrepareRemoveThematicBreak(id NodeID) (ChangeSet, error)
```

The operation deletes exactly `ThematicBreak.Range()`. It does not synthesize or remove neighboring blank lines and remains bound to the immutable source snapshot.

## Survivor proof

M72 introduces a reusable block-removal survivor validator in `internal/splice`. After rendering the single deletion candidate, one full Marksplice parse must prove:

- every original observed node whose semantic range does not overlap the removed block survives one-to-one;
- survivor kind, transformed `Range`/`ContentRange`, reviewed scalar semantics, alignment/child-count facts, editability/top-level state, and applicable source anchors remain unchanged modulo the deletion transform;
- no additional observed nodes appear in the candidate.

This rejects join hazards that byte-level deletion alone cannot see. For example, removing `***\n` from `before\n***\nafter\n` would merge the two neighboring paragraphs, so the operation returns `ErrInvalidReplacement`.

The validator deliberately skips only nodes whose semantic ranges overlap the owned removal span. This makes it reusable for a future simple blockquote removal where the blockquote's contained paragraph/inline observations must disappear together.

## Risks and mitigations

1. Deleting an exact line can still change the parse of untouched neighboring bytes. Candidate reparsing plus one-to-one survivor proof rejects those joins.
2. Comparing snapshot `NodeID` values across reparses would be invalid because fingerprints change. M72 compares transformed source/semantic facts and deliberately ignores snapshot IDs.
3. A broad generic survivor helper could silently weaken family-specific mutation proof. M72 uses it only for whole-block removal; existing list/table/section operations retain their specialized validators.

## Evidence

Focused TDD covers exact CRLF line removal, stale-source conflict behavior, and fail-closed paragraph merging. M71 thematic-break parsing/construction tests and all internal regressions pass.

Additional pre-M73 regression passed:

- `go test ./... -count=1`;
- `go vet ./...`;
- `staticcheck ./...`;
- `git diff --check`.

No commit or push was performed.
