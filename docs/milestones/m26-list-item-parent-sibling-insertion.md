# Milestone M26 — List-Item Parent Sibling Insertion

Status: green — complete-parent list-item sibling insertion passed.

## Goal

Extend the existing `PrepareInsertListItemBefore` and `PrepareInsertListItemAfter` operations from M19 leaf anchors to complete supported list-item parent subtrees, reusing M24's private subtree-completeness/end proof while keeping the inserted fragment itself a single same-shape leaf line.

## Public contract

M26 does not add new public methods. It broadens:

```go
func (d *Document) PrepareInsertListItemBefore(anchorID NodeID, fragment []byte) (ChangeSet, error)
func (d *Document) PrepareInsertListItemAfter(anchorID NodeID, fragment []byte) (ChangeSet, error)
```

The anchor may now be:

- a supported leaf item, preserving M19 behavior; or
- a supported parent whose complete semantic descendant subtree is represented by the M24 supported model.

An incomplete parent subtree returns `ErrInvalidTargetKind` rather than exposing a guessed boundary.

The fragment remains exactly one promoted-shape leaf list-item physical line and must satisfy M19's same-sibling-shape proof.

## Insertion boundaries

`before` inserts at the anchor's physical-line start:

```text
anchor.ListItemSource.LineRange.Start
```

`after` inserts at the complete private subtree end:

```text
anchor.ListSubtreeEnd
```

For a leaf, `ListSubtreeEnd == LineRange.End`, so M19 byte behavior is unchanged.

For a parent, `after` is therefore placed after all proven descendants rather than between the parent line and its first child.

No newline, blank line, indentation, marker, or numbering source is synthesized.

## Same-shape fragment proof

M26 retains M19's standalone fragment contract:

- exactly one complete physical leaf line;
- exact pre-marker prefix equal to the anchor's prefix;
- same ordered/unordered state;
- same unordered marker or ordered delimiter;
- caller-selected ordered numeric token preserved;
- no extra preamble, second item, or trailing blank-line source.

M26 broadens only the anchor boundary. It does not introduce list-item subtree fragments.

## Semantic sibling proof

Matching prefix and marker bytes are not sufficient to prove semantic siblinghood in every host context.

After candidate parsing, the inserted leaf must have the same immediate semantic list-parent relation as the candidate anchor:

- both root items, or
- both nested with the same `ListParentAnchor`.

The anchor remains part of ordinary survivor validation, so its own parent relationship must first re-establish exactly after the patch.

This gives the operation a semantic sibling invariant in addition to M19's lexical same-shape invariant.

## Structural parent-anchor transformation

M26 exposed an existing boundary ambiguity in list survivor validation. `ListParentAnchor` is a physical-line start, but it had been transformed as a generic zero-width range. A zero-width insertion exactly at the parent line start is intentionally left-biased by the shared generic range transform, while the actual parent line moves to the right.

Changing the generic range transform would affect section and move semantics, so M26 adds a list-specific source-owned anchor transform instead.

The parent anchor is represented by the first physical source byte:

```text
[ListParentAnchor, ListParentAnchor+1)
```

Transforming that byte provides the required behavior:

- insertion exactly before the parent shifts the anchor right;
- deletion before the parent shifts it left;
- replacement later in the same parent line leaves its start unchanged;
- a patch consuming the parent anchor fails closed.

The rule works identically for supported and intentionally unsupported immediate parents and does not require another public or persistent hierarchy index.

## Candidate validation

M26 requires:

1. exactly one additional supported list item;
2. every original supported item to preserve its transformed physical/source/content mappings and semantic parent relation;
3. the anchor subtree to have been complete before preparation;
4. the inserted item to match the standalone fragment mapping and caller bytes exactly;
5. the inserted item to remain a leaf;
6. the inserted item and candidate anchor to have identical immediate parent presence/anchor.

Unsafe joins, reparenting, or incomplete anchor subtrees fail closed.

## Compatibility

M26 preserves:

- M19 leaf-anchor before/after semantics;
- caller-owned ordered numbers and line endings;
- M24 private subtree completeness/end ownership;
- M25 subtree removal semantics;
- public `ListItem.Range()` content-only meaning;
- snapshot-local `NodeID` and `ParentID` semantics.

At the M26 exit, `PrepareMoveListItemBefore` and `PrepareMoveListItemAfter` still required leaf anchors. M27 later broadens only the destination anchor to a complete supported parent subtree while keeping the moved source leaf-only; parent-subtree movement remains deferred.

## Complexity

Let `l` be the number of supported list items, `n` the host candidate size, and `k` the fragment size.

Preparation remains:

- O(k) standalone fragment parse/mapping;
- O(n+k) candidate construction and semantic parse;
- O(l) survivor validation with O(1)-expected physical-line lookup.

The semantic sibling proof adds only constant-time candidate lookups. No recursive subtree walk, new persistent index, or public batch API is introduced.

## Devil's advocate review

### Risk: `after` splits a parent from its descendants

Using M19's `LineRange.End` on a parent would insert the sibling before the child subtree.

Mitigation: complete parent anchors use M24's `ListSubtreeEnd`.

### Risk: byte-compatible indentation still reparents the inserted item

Container/list context may make identical-looking source parse under a different immediate parent.

Mitigation: the inserted candidate item must report exactly the same immediate semantic parent relation as the candidate anchor.

### Risk: insertion before a parent falsely reports descendant reparenting

A generic zero-width parent anchor at the same byte as the insertion point is left-biased even though the parent line moves right.

Mitigation: transform the first physical byte of the parent line instead of the zero-width point. A dedicated internal regression covers insertion-at-anchor versus content replacement later in the line.

### Risk: broadening the anchor silently broadens the fragment to a subtree

That would combine two independent ownership problems in one milestone.

Mitigation: the fragment contract remains exactly the M19 single leaf line.

### Risk: an unsupported descendant gives a false `after` boundary

A public parent may still contain an unpromoted complex descendant.

Mitigation: both before and after require the M24 `ListSubtreeComplete` target gate; an incomplete parent returns `ErrInvalidTargetKind`.

## TDD evidence

M26 began with focused public tests that failed only because M19 still used the leaf-only target gate for parent anchors.

During the first green attempt, insertion before a root parent exposed the zero-width parent-anchor boundary issue described above. An initial whole-parent-line transform fixed that boundary but regressed parent content replacement because the edited content overlaps the parent line. The final source-owned one-byte anchor transform handles both cases and is covered directly by an internal regression.

Focused tests now pass for:

- insertion before a root parent subtree;
- insertion after a root parent with deep descendants;
- nested parent insertion retaining the same immediate parent;
- ordered CRLF and Unicode source preservation;
- rejection of incomplete parent anchors;
- stale-source conflict behavior;
- parent content replacement regression;
- M18–M25 list-item regressions;
- direct parent-anchor transform cases for insertion, replacement, earlier deletion, and anchor consumption.

M26 is green. The complete strict repository verification stack passes on top of M21–M25 and the committed M18–M20 baseline: native `gofmt`, focused M18–M26 list regressions, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, text-hygiene checks, `git diff --check`, and `git fsck --no-dangling` after storage recovery.
