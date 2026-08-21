# Milestone M28 — List-Item Subtree Move

Status: green — complete supported list-item subtree movement passed.

## Goal

Extend the existing atomic list-item move operations from M27 leaf sources to complete supported list-item subtrees, reusing the private subtree ownership proved by M24/M25 rather than introducing a second subtree model or a new public move API.

## Public contract

M28 broadens the existing `PrepareMoveListItemBefore` and `PrepareMoveListItemAfter` methods. Both source and destination anchor may be a supported leaf or a supported parent whose entire semantic descendant subtree is represented by Marksplice's supported list-item model. An incomplete source or anchor fails closed with `ErrInvalidTargetKind`.

No new public subtree range or mutation method is added.

## Complete-subtree target gate

By M28 the old private name `appendableListItemTarget` no longer described its responsibility because the same proof is used by remove, insert, append, and move. M28 consolidates those roles onto `completeListItemTarget`, which verifies ordinary list-item editability plus `ListSubtreeComplete`.

The obsolete `leafListItemTarget` becomes unused and is removed rather than retained as dead code.

## Source and destination ranges

A complete supported subtree owns `[LineRange.Start, ListSubtreeEnd)`. M28 centralizes that expression in `listItemSubtreeRange`, shared by removal and movement.

For a leaf, `ListSubtreeEnd == LineRange.End`, so M20/M27 byte behavior is unchanged. For a parent, the moved fragment contains the parent physical line and every byte through its final proven descendant.

Destination boundaries remain the anchor physical-line start for `before` and `anchor.ListSubtreeEnd` for `after`.

## Non-overlap rule

Source and anchor subtree ranges must not overlap. Ancestor/descendant moves relative to each other combine subtree extraction with an intra-subtree destination and are outside M28's reviewed ownership contract. They fail with `ErrInvalidReplacement` before patch construction.

## Same-sibling-shape consolidation

M28 extracts one private lexical helper for the existing M19/M20 destination-shape rules: ordered state, marker/delimiter, and exact pre-marker physical-line prefix.

Caller-provided insertion fragments still require standalone parsing because their bytes are not yet mapped. Snapshot-owned moved subtrees reuse their already validated root mapping instead of reparsing an existing document fragment solely to rediscover the same root shape. Ordered numeric tokens remain caller/source-owned and are never normalized.

## Atomic move and candidate proof

The operation remains one named `ChangeSet` with two disjoint original-coordinate patches: delete the exact complete source subtree and insert those exact bytes at the destination boundary.

M25's skip-ID set is reused for every supported item contained in the moved range. The candidate must contain the exact moved byte span at the computed destination offset and exactly the same number of supported moved items.

Every moved supported item must preserve its subtree-relative physical, marker/source, and content ranges; ordered state; marker/delimiter; child state; direct-child count; and physical-line bytes. Descendants below the moved root must preserve immediate parent presence and `ParentAnchor` shifted by the common subtree delta. The root may change parent and must become a semantic sibling of the candidate anchor through the shared M26/M27 sibling proof.

## Parent direct-child counts

For supported parents, M28 validates explicit direct-child deltas: source parent `-1`, destination parent `+1`. When both are the same supported parent, the deltas cancel and the original count must remain unchanged.

Unpromoted complex parents have no public `NodeID`; Marksplice does not invent one. Their safety remains governed by the full candidate semantic sibling proof and survivor validation.

## No-op behavior

No-op detection uses complete subtree boundaries: `movedSubtree.End == anchor.LineRange.Start` for `before`, and `anchor.ListSubtreeEnd == movedSubtree.Start` for `after`. It is accepted only when moved root and anchor already share the same semantic parent. The zero-patch change remains fingerprint-bound.

## Compatibility and complexity

M28 preserves M20 atomic movement for leaves, M24 private subtree ownership, M25 skip-set validation, M26/M27 semantic sibling and parent-anchor transforms, exact caller source, snapshot conflict behavior, and the existing public `ListItem` detail semantics.

Let `n` be candidate size, `l` the supported item count, and `k` the moved subtree bytes. Preparation uses O(1) subtree boundary lookup, O(l) moved-ID collection, O(k) exact byte handling, O(n+k) candidate parse, and O(l) survivor/moved-subtree validation. No recursive descendant walk, second persistent hierarchy index, renderer, or generic public batch API is introduced.

## Devil's advocate review

### Risk: parent-only extraction leaves descendants behind

Mitigation: source extraction is exactly the already-proven M24/M25 complete subtree range.

### Risk: source and anchor overlap

Mitigation: overlapping subtree ranges fail before patch construction.

### Risk: identical bytes reappear with a different descendant hierarchy

Mitigation: every moved descendant must preserve subtree-relative ranges, child state/count, and parent anchor shifted by one common delta.

### Risk: cross-parent movement changes list cardinality incorrectly

Mitigation: supported source/destination parent direct-child counts are checked with explicit `-1/+1` deltas, including the zero-net same-parent case.

### Risk: move duplicates M19 fragment parsing

Mitigation: one lexical same-shape helper is shared; only untrusted caller fragments need standalone parsing, while snapshot-owned move roots reuse stored mappings.

## TDD evidence

M28 began with focused tests whose supported parent sources failed only because M27 still used the leaf-only source gate.

Focused tests now pass for root parent movement with deep descendants, complete parent anchors, nested cross-parent reparenting, nested same-parent reorder, ordered CRLF/Unicode preservation, subtree-aware stale-safe no-ops, incomplete source/anchor rejection, ancestor/descendant overlap rejection, unchanged M20/M27 leaf movement, and focused M18–M28 regressions together.

M28 is green. The complete strict repository verification stack passes on top of M21–M27 and the committed M18–M20 baseline: native `gofmt`, focused M18–M28 list regressions, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, text-hygiene checks, `git diff --check`, and `git fsck --no-dangling` after storage recovery.