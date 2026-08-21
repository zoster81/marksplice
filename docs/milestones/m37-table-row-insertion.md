# Milestone M37 — Table-Row Sibling Insertion

Status: green.

## Goal

Insert one caller-provided complete GFM body row immediately before or after an existing promoted body-row anchor without generating Markdown syntax or line separators.

## Public contract

M37 adds:

```go
func (d *Document) PrepareInsertTableRowBefore(anchorID NodeID, fragment []byte) (ChangeSet, error)
func (d *Document) PrepareInsertTableRowAfter(anchorID NodeID, fragment []byte) (ChangeSet, error)
```

The anchor must be an M35 body row. The caller owns the complete inserted source bytes.

## Host-context proof

A single body-row fragment is not a standalone GFM table, so M37 deliberately does not invent a standalone row grammar. It inserts the bytes into the host candidate and parses that candidate once.

The candidate must contain exactly one additional promoted body row. The inserted row must:

- start exactly at the requested zero-width anchor boundary;
- own exactly `[insertAt, insertAt+len(fragment))` as its complete physical `LineRange`;
- have the same semantic/source-proven column count as the anchor;
- resolve to the same candidate table as the transformed anchor;
- reproduce the caller fragment byte-for-byte.

All original mapped rows and cells must survive with exact transformed mappings and bytes.

## Line terminators

Marksplice does not append or normalize a newline. A fragment without a terminator may be valid when inserted after the final terminated row at EOF, because it can own exactly the new final row span. The same fragment inserted before another line fails closed if it merges with adjacent source instead of owning one exact row.

Multi-row fragments and wrong-column-count rows fail closed.

## Complexity

Insertion prepares one zero-width patch, parses one candidate, builds one temporary table mutation index, and validates surviving rows/cells linearly. No persistent table membership index is introduced.

## Devil's advocate review

### Risk: an unterminated fragment merges with the anchor

Mitigation: the inserted candidate row must own exactly the caller fragment span as its complete `LineRange`.

### Risk: a fragment inserts multiple rows

Mitigation: the candidate promoted-row count must increase by exactly one and the row at the insertion offset must own the complete fragment.

### Risk: the new row attaches to a neighboring table

Mitigation: its private candidate table anchor must equal the transformed host anchor's table anchor.

### Risk: column shape silently changes

Mitigation: semantic/source-proven column count must equal the anchor's count; surviving row and cell mappings remain exact.

## TDD evidence and exit decision

Tests cover before/after insertion, EOF insertion without a synthetic terminator, exact reparse ownership of the EOF row, rejection of unsafe unterminated middle insertion, multi-row fragments, wrong column count, and existing table-cell regressions.

M37 is green.