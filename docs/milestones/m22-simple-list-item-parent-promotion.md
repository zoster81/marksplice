# Milestone M22 — Simple List-Item Parent Promotion

Status: green — simple list-item parent promotion passed.

## Goal

Keep a list item publicly addressable after M21 appends its first child, without promoting arbitrary multiline or multi-block list items or silently broadening the structural mutation contracts established by M18–M21.

M22 promotes one additional proven source shape: a list item whose own head is one physical text/paragraph line and whose remaining direct blocks, if any, are nested lists only.

## Public contract

M22 extends `ListItem` with:

```go
func (i ListItem) HasChildren() bool
```

`HasChildren` reports whether the supported item currently owns one or more direct child lists.

Existing public semantics remain unchanged:

- `ListItem.ID()` is snapshot-scoped;
- `ListItem.Range()` is still only the exact first-line content span used by `PrepareReplaceListItem`;
- `Ordered()` and `Marker()` still describe the item's own containing list syntax;
- child-list source, indentation, markers, numbering, and subtree ranges are not exposed through `ListItem.Range()`.

## Supported parent shape

A list item is publicly promoted when:

1. its first direct block is a Goldmark `TextBlock` or `Paragraph` whose mapped content occupies exactly one physical source line;
2. every remaining direct block is a nested `List`;
3. the first-line source can still be mapped through Marksplice's existing `MapSingleLineListItem` proof.

The item may have one or more nested lists and those lists may contain deeper descendants.

M22 deliberately does not promote a list item whose remaining blocks include another paragraph, indented code, blockquote, fenced code, or another non-list block. Those shapes require separate source-ownership semantics.

Goldmark types remain confined to `internal/parser/goldmark`; the parser-independent observation adds only `HasListChildren`.

## Identity and source compatibility

The item ID remains derived from the existing Marksplice list-item marker/source range. M22 does not redefine leaf IDs or `ListItem.Range()`.

Existing leaf items therefore retain the same kind/range/ID derivation. M22 adds observations for previously hidden supported parent items rather than changing the identity of already promoted leaf items.

## Parent content replacement

`PrepareReplaceListItem` now supports both:

- supported leaf items;
- supported single-line-head parent items.

The mutation still patches only `ContentRange`.

After candidate parsing, Marksplice requires:

- the same list marker/delimiter and ordered state;
- the expected updated physical-line, marker/source, and content ranges;
- the same parent relation for the target when nested;
- the same `HasChildren` state for the target;
- every other promoted list item to survive with its transformed lexical mapping, parent relation, child state, and byte-identical physical line.

This allows renaming a parent line while preserving all descendant list source byte-for-byte.

## M22 structural gating and M24 extension

M22 does not redefine line-only structural operations as subtree operations.

At the M22 exit, a shared internal `leafListItemTarget` gate required `HasChildren == false` for:

- `PrepareRemoveListItem`;
- `PrepareInsertListItemBefore`;
- `PrepareInsertListItemAfter`;
- `PrepareMoveListItemBefore` source and anchor;
- `PrepareMoveListItemAfter` source and anchor;
- `PrepareAppendListItemChild`.

That restriction was intentional because M22 had no reviewed complete-subtree boundary. M24 later lifts the parent restriction for `PrepareAppendListItemChild` after adding a private semantic subtree-completeness/end proof, M25 reuses the same proof to broaden `PrepareRemoveListItem` to complete supported subtrees, M26 reuses it for sibling insertion around a complete parent anchor while keeping the inserted fragment a leaf, M27 reuses it for move anchors, and M28 finally permits complete supported move sources as well. None of these milestones exposes a public subtree range.

## M21 interaction

Before M22, M21 converted one promoted leaf into a non-promoted parent while introducing one promoted child, so the promoted leaf count remained unchanged.

With M22, the parent remains promoted and changes `HasChildren` from false to true. The inserted child is an additional promoted item. M21 candidate validation therefore now expects one additional list item, permits the target parent's child-state transition, and explicitly requires the candidate parent to report `HasChildren == true`.

The inserted child must still be one leaf line with the exact requested parent anchor and `HasChildren == false`.

## Remove/move child transitions

Parent promotion also changes M18/M20 validation semantics, and M25 later generalizes removal to complete supported child subtrees.

Removing a leaf or complete child subtree, or moving the final leaf child, can legitimately change the supported outer parent's `HasChildren` state while leaving its own physical line byte-identical.

The shared list-item survivor validator therefore accepts child-state differences only at explicitly authorized original parent line anchors:

- M18/M25 authorize the removed leaf/subtree's immediate supported parent, when one exists;
- M20 authorizes the moved leaf's source parent, when one exists;
- M21 authorizes its target leaf line for false-to-true transition;
- M19 and ordinary replacement authorize no unrelated child-state changes.

All other promoted list items must preserve `HasChildren` exactly.

## Complexity

The parser adapter still walks the Goldmark AST once. Determining the supported parent shape walks only the direct children of each observed list item; across the AST those child edges are already bounded by the tree size.

Candidate list mapping remains source-ordered/O(n) parsing plus O(l) validation with O(1)-expected lookup by physical-line start, where `l` is the number of promoted supported list items.

No new persistent hierarchy index, renderer, batch API, filesystem/network behavior, or dependency is introduced.

## Devil's advocate review

### Risk: arbitrary multi-block list items become publicly actionable

A first line plus extra paragraph/code/quote has source ownership that is not captured by the M18 `LineRange`.

Mitigation: only nested `List` blocks are allowed after the single-line head. Any other direct block keeps the item unpromoted.

### Risk: existing line-only structural operations delete or split child subtrees

Once parents are public, a generic `KindListItem` target check would let M18/M19/M20/M21 accept them accidentally.

Mitigation at the M22 exit: one shared `leafListItemTarget` gate rejected parent items with `ErrInvalidTargetKind` before patch preparation. M24–M28 later replace that role-specific restriction only where complete-subtree ownership has been separately proven; by M28 the obsolete leaf-only target gate has no remaining caller and is removed.

### Risk: removing the final child is rejected as semantic drift

With parents promoted, the parent legitimately changes from parent to leaf after M18/M20 source extraction.

Mitigation: child-state changes are allowed only at the known source-parent anchor for operations that can cause that transition.

### Risk: parent content replacement reparses or reparents descendants

Checking only the target's marker/content range would miss semantic changes below it.

Mitigation: replacement uses the shared candidate list mapping to validate every other promoted item, including parent anchors and child-state, and separately validates the target's own parent/child state.

### Risk: leaf IDs or existing public ranges change

Changing identity semantics would create an unnecessary compatibility break.

Mitigation: M22 leaves `Kind`, mapped source `Range`, `ContentRange`, and node-ID derivation unchanged for existing leaf items.

## Evidence and exit decision

M22 began with focused tests that failed only because `HasListChildren` and public `ListItem.HasChildren()` did not exist.

Focused tests now pass for:

- promotion of a single-line parent with nested child lists;
- nested leaf parent metadata alongside parent promotion;
- rejection of parent items with trailing paragraph/code/blockquote blocks;
- public parent `HasChildren()` detail;
- content-only parent replacement preserving CRLF, Unicode, and descendant bytes;
- the original M22 parent-target rejection for remove/append/sibling insert/move, later superseded role by role through M24–M28 only when private subtree completeness is proven;
- M21 result retaining the parent publicly with `HasChildren == true`;
- M18/M20 last-child parent transitions;
- focused M18–M22 list regressions together.

M22 is green. The complete repository verification stack passes with M21 on the committed M18–M20 baseline: native `gofmt` checks, focused parser/list regressions, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, and `git diff --check`.
