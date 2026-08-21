# Milestone M27 — List-Item Parent-Anchor Move

Status: green — leaf-source movement around complete list-item anchors passed.

## Goal

Extend the existing atomic leaf list-item move operations so the destination anchor may be a complete supported parent subtree, while keeping the moved source itself exactly one M20 leaf physical line.

M27 reuses M24 subtree completeness/end ownership and M26 semantic sibling validation. At the M27 exit it does not introduce parent-subtree movement; M28 later broadens the moved source to a complete supported subtree using the same private ownership proof.

## Public contract

M27 does not add new methods. It broadens:

```go
func (d *Document) PrepareMoveListItemBefore(id, anchorID NodeID) (ChangeSet, error)
func (d *Document) PrepareMoveListItemAfter(id, anchorID NodeID) (ChangeSet, error)
```

The source `id` must still identify a promoted leaf item.

The anchor may be:

- a supported leaf, preserving M20 behavior; or
- a supported parent whose complete semantic descendant subtree is represented by M24's private supported model.

An incomplete parent anchor returns `ErrInvalidTargetKind`.

## Source and destination boundaries

The moved bytes remain exactly the source leaf's private `LineRange`.

Destination boundaries are:

```text
before: anchor.ListItemSource.LineRange.Start
after:  anchor.ListSubtreeEnd
```

For a leaf anchor, `ListSubtreeEnd == LineRange.End`, so M20's byte behavior is unchanged.

For a parent anchor, `after` is placed after all proven descendants rather than between the parent line and its child subtree.

No source byte is regenerated or normalized.

## Atomic move and same-shape proof

M27 preserves M20's one-operation/two-patch design:

1. delete the exact moved leaf `LineRange`;
2. insert the same bytes at the destination boundary.

The moved line still passes M19's standalone same-shape proof against the destination anchor:

- identical physical pre-marker prefix;
- same ordered/unordered state;
- same marker or ordered delimiter;
- caller/source-owned ordered numeric token preserved.

This prevents the broader anchor capability from becoming implicit indentation or marker rewriting.

## Semantic sibling proof

M26 introduced candidate validation requiring an inserted leaf and its anchor to have the same immediate semantic parent relation.

M27 generalizes that same private helper from one patch transform to an arbitrary validated transform slice. Both insertion and atomic move therefore use one implementation:

- M26 passes one zero-width insertion transform;
- M27 passes M20's deletion and insertion transforms.

After the combined candidate is parsed, the moved leaf must have the same immediate semantic parent presence and `ParentAnchor` as the candidate anchor.

This makes cross-parent movement explicit and candidate-proven rather than relying only on identical indentation bytes.

## Adjacent no-op behavior

M20's snapshot-bound no-op remains, but `after` adjacency now uses the complete anchor subtree boundary:

```text
before: moved.LineRange.End == anchor.LineRange.Start
after:  anchor.ListSubtreeEnd == moved.LineRange.Start
```

A zero-patch no-op is returned only when the original moved item and anchor already have the same semantic parent.

This prevents a purely lexical adjacency from being treated as a successful sibling move when the current GFM structure says the nodes have different parents.

The zero-patch change remains source-fingerprint bound and rejects stale input.

## Source remains leaf-only

At the M27 exit, only the destination anchor is broadened.

A supported parent cannot yet be the moved source. M28 later reviews and enables complete supported subtree sources with explicit overlap, descendant, and parent-count validation. The M27 deferral identified the required concerns:

- complete source subtree extraction;
- destination compatibility for a subtree fragment;
- descendant parent-anchor transforms;
- source/destination parent child-count transitions;
- no-op and overlap behavior when source and anchor subtrees interact.

`PrepareMoveListItemBefore/After` therefore continue to call the leaf-only target gate for the source.

## Candidate validation

A non-no-op M27 move requires:

1. the anchor subtree to be complete before patch construction;
2. the moved source to be a supported leaf;
3. the moved line to satisfy M19 lexical same-shape proof against the anchor;
4. all surviving original supported list items to preserve their transformed lexical mappings, source bytes, parent anchors, and allowed child-state transitions;
5. the moved leaf to reappear at the exact candidate offset with byte-identical source;
6. the moved leaf and candidate anchor to have the same immediate semantic parent relation.

The source parent's allowed `HasChildren` transition remains the M20 rule when the moved leaf was its final child.

## Compatibility

M27 preserves:

- M20 leaf-source extraction and atomic two-patch movement;
- M19 lexical same-shape destination proof;
- M24 private subtree completeness/end semantics;
- M26 source-owned parent-anchor transform and semantic sibling proof;
- caller-owned indentation, marker, numbering, Unicode, and line endings;
- snapshot-bound stale-source behavior;
- public `ListItem.Range()` and `ParentID()` semantics.

No public subtree range, generic batch API, renderer, or parser-specific type is introduced.

## Complexity

Let `n` be candidate source size, `l` the supported list-item count, and `k` the moved leaf-line size.

A non-no-op move remains:

- O(k) standalone leaf proof;
- O(n+k) candidate construction and semantic parse;
- O(l) survivor validation with expected O(1) mapping lookup;
- constant additional work for semantic sibling validation.

The destination subtree boundary is already resolved at parse time by M24, so M27 adds no descendant scan during mutation preparation.

## Devil's advocate review

### Risk: `after` inserts inside the anchor subtree

Using M20's `anchor.LineRange.End` on a parent would separate the parent head from its descendants.

Mitigation: a complete parent anchor uses `ListSubtreeEnd`; incomplete subtrees are rejected before patch creation.

### Risk: lexical prefix equality is mistaken for semantic siblinghood

Two lines can have compatible-looking indentation while host context assigns them different parents.

Mitigation: the final combined candidate must report the same immediate semantic parent for moved leaf and anchor.

### Risk: a no-op hides semantic mismatch

Two source ranges can be adjacent even when they are not semantic siblings.

Mitigation: zero-patch no-op requires original semantic-parent equality; otherwise the normal combined candidate path must prove the requested relation or fail closed.

### Risk: broadening the anchor silently enables subtree movement

A public parent is now accepted in one move argument, so source/anchor roles could be conflated.

Mitigation: source lookup remains `leafListItemTarget`; only destination lookup uses the complete-subtree gate.

### Risk: M26 and M27 sibling validators drift

Separate single-patch and multi-patch implementations could diverge.

Mitigation: M27 generalizes the existing M26 helper to accept `[]patchTransform`; both operations call the same semantic sibling validator.

## TDD evidence

M27 began with focused public tests that failed only because M20 still required a leaf destination anchor.

Focused tests now pass for:

- moving a root leaf after a parent with deep descendants;
- moving a root leaf before a parent subtree;
- nested cross-parent movement after a complete nested parent while preserving exact source bytes;
- ordered CRLF and Unicode movement while preserving the moved numeric token;
- adjacent before/after no-op detection using the complete subtree boundary;
- stale-source rejection for parent-anchor no-op changes;
- rejection of incomplete parent anchors;
- continued rejection of parent items as moved sources at the M27 exit, later superseded by M28 for complete supported source subtrees;
- focused M18–M27 list-item regressions together.

M27 is green. The complete strict repository verification stack passes on top of M21–M26 and the committed M18–M20 baseline: native `gofmt`, focused M18–M27 list regressions, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, text-hygiene checks, `git diff --check`, and `git fsck --no-dangling` after storage recovery.
