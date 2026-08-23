# M71 — Public Thematic-Break Model

## Status

Complete and green.

## Objective

Promote source-proven top-level GFM thematic breaks from construction-only parser metadata into the ordinary immutable parsed public model without changing existing snapshot-ID derivation.

## Public contract

M71 appends `KindThematicBreak` to the public kind enum and adds:

```go
func (d *Document) ThematicBreak(id NodeID) (ThematicBreak, bool)

type ThematicBreak struct { /* comparable scalar state */ }
func (t ThematicBreak) ID() NodeID
func (t ThematicBreak) Range() Range
```

`Range()` owns the complete physical thematic-break line, including its existing LF/CRLF/CR terminator when present. Nested thematic breaks remain outside the promoted public surface.

## Source proof

`internal/source.MapTopLevelThematicBreak` independently validates the physical line rather than trusting Goldmark source ownership. The reviewed lexical subset accepts the GFM `*`, `-`, and `_` marker families, at least three markers, optional horizontal spacing between markers, up to three leading spaces, trailing horizontal space, and an optional existing line terminator.

The semantic `Node.Range` used by snapshot ID derivation remains the pre-existing Goldmark/Marksplice thematic-break range excluding the line terminator. The complete physical `LineRange` is stored separately in `ThematicBreakSource`, so M55 construction behavior and existing internal kind ordinals/IDs are not rewritten.

## Risks and mitigations

1. Public promotion could trust a parser position that is insufficient for mutation ownership. Promotion requires the independent source-layer line proof before `Editable` becomes true.
2. Adding a public kind could shift existing ordinals. `KindThematicBreak` is appended after the M64 `KindTable` value.
3. Construction proof previously required thematic breaks to be non-editable. M71 updates M55 proof to require the same exact source mapping now used by parsed promotion.

## Evidence

Focused tests cover `***`, `- - -`, `___` at EOF, indentation/trailing whitespace, CRLF physical-line ownership, and non-promotion of nested thematic breaks. Existing thematic-break construction tests remain green.

No commit or push was performed.
