# M73 — Public Simple Blockquote Model

## Status

Complete and green.

## Objective

Promote the exact simple top-level blockquote shape already proven for M56 construction into the ordinary immutable parsed public model without implying support for general blockquote containers.

## Public contract

M73 appends `KindBlockquote` after `KindThematicBreak` and adds:

```go
func (d *Document) Blockquote(id NodeID) (Blockquote, bool)

type Blockquote struct { /* comparable scalar state */ }
func (b Blockquote) ID() NodeID
func (b Blockquote) Range() Range
func (b Blockquote) ContentRange() Range
```

`Range()` owns the complete physical blockquote line, including its existing line terminator. `ContentRange()` owns only the single inner paragraph source, excluding indentation, the `>` marker, its optional following ASCII space, and the line terminator.

## Capability boundary

Promotion remains intentionally narrower than GFM blockquote semantics. Goldmark must first report one top-level blockquote whose only child is one single-line paragraph. `internal/source.MapSimpleTopLevelBlockquote` then independently proves the physical prefix and line ownership.

The reviewed lexical subset allows zero through three leading ASCII spaces, one `>` marker, either no following marker space or one ASCII space, and a non-empty single physical-line paragraph. Multiline blockquotes, list-containing blockquotes, nested blockquotes, tab-indented prefixes, and other broader containers remain internal/non-editable.

The existing semantic `Node.Range` used by snapshot ID derivation is unchanged. Complete physical-line ownership is stored separately in `BlockquoteSource.LineRange`, preserving prior internal IDs and the M56 observation model.

## Risks and mitigations

1. Marker indentation can be confused with content. M73 keeps a separate exact marker range and public content range, and requires an independent source-layer prefix proof.
2. Promoting `KindBlockquote` could shift existing public ordinals. The new public kind is appended after M71 `KindThematicBreak`.
3. M56 construction proof previously expected a non-editable blockquote observation. M73 updates that proof to require the same exact editable source mapping now used by parsed promotion.
4. General GFM blockquotes can contain lazy continuations, multiple lines, and arbitrary block children. The parser observation and source mapper both fail closed outside the reviewed single-line paragraph subset.

## Evidence

Focused tests cover indented CRLF lines, exact inner content, `>quoted` without marker spacing, rejection of multiline and list-containing containers, public kind append compatibility, and M56 construction regression. Direct source-layer tests pin indentation/prefix rules and unsupported shapes.

No commit or push was performed.
