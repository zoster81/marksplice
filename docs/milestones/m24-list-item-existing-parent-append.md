# Milestone M24 — Existing-Parent List-Item Child Append

Status: green — existing-parent list-item child append passed.

## Goal

Extend `PrepareAppendListItemChild` from the M21 first-child case to an already supported parent that owns existing child lists, without exposing a public list-subtree range or assuming that every semantic descendant belongs to Marksplice's currently promoted list-item subset.

M24 must insert after the parent's complete supported descendant subtree rather than after the parent's own physical line.

## Public contract

M24 does not add a new public method. It broadens the existing:

```go
func (d *Document) PrepareAppendListItemChild(parentID NodeID, fragment []byte) (ChangeSet, error)
```

The target may now be:

- a supported leaf list item, preserving the M21 first-child behavior; or
- a supported parent list item whose entire semantic list-item descendant subtree is represented by the supported M22 list-item model.

The inserted fragment must still become exactly one direct leaf child of the requested parent. Caller bytes remain authoritative; Marksplice does not synthesize indentation, numbering, markers, separators, or line endings.

A supported parent whose descendant subtree is incomplete returns `ErrInvalidTargetKind` rather than guessing an insertion boundary.

## Why `HasChildren` is not enough

M22 deliberately allows a simple supported parent to remain public even when one of its nested child items has a complex unsupported shape. The parent can therefore report `HasChildren() == true` while not every semantic descendant has a public Marksplice `ListItem`.

Using the last promoted descendant as an insertion boundary in that case could insert bytes into the middle of an unmodelled subtree.

M24 therefore distinguishes:

- public parent visibility;
- private complete-subtree ownership required for structural append.

The latter is never exposed as `ListItem.Range()` and does not imply that remove/move subtree semantics are reviewed.

## Parser-independent direct-child count

For each supported list item, the Goldmark adapter now reports a Marksplice-owned `ListDirectChildCount` alongside the existing parent anchor and child-state metadata.

The count is the number of immediate semantic child `ListItem` nodes across the supported item's direct nested `List` blocks. It is computed from public Goldmark AST relationships while Goldmark remains confined to `internal/parser/goldmark`.

Unsupported child items are still counted semantically even though they are not promoted as Marksplice nodes. This is the key evidence needed to detect incomplete public subtrees.

## Private subtree resolution

After M23 resolves supported `ListParentID` values, `internal/splice` performs one additional Marksplice-owned hierarchy pass.

Each supported list-item node receives private facts:

- `ListDirectChildCount`;
- `ListSubtreeComplete`;
- `ListSubtreeEnd`.

Resolution uses a leaf-up queue rather than recursion or parser traversal order:

1. build a temporary supported `NodeID -> node index` map;
2. count supported direct children from resolved `ListParentID` relations;
3. initialize every item's candidate subtree end to its own physical `LineRange.End`;
4. enqueue supported leaves;
5. when a child resolves, propagate its subtree end and completeness to its supported parent;
6. mark a node complete only when its supported direct-child count equals its semantic direct-child count and every supported child subtree is complete;
7. reject an impossible cycle or inconsistent child metadata.

A complete parent's private subtree end is therefore the greatest physical-line end owned by its supported descendants.

The temporary maps/queues are discarded after parse. No second persistent list hierarchy index is introduced.

## Append boundary

`PrepareAppendListItemChild` now targets only `ListSubtreeComplete` items.

Insertion occurs at:

```text
parent.ListSubtreeEnd
```

For a leaf this equals the M21 `LineRange.End` boundary.

For an existing parent it lies after the deepest existing supported descendant and before following source. Blank-line source is not normalized or consumed; the host candidate parser remains the final semantic oracle.

## Candidate validation

M24 reuses the M21 host-context proof:

- exactly one additional supported list item must appear;
- every original supported item must preserve its transformed source mapping and semantic parent relation;
- unrelated `HasChildren` state must remain stable;
- a leaf target may transition from `HasChildren == false` to true;
- an existing parent must remain a parent;
- the target parent's semantic direct-child count must increase by exactly one;
- the inserted physical line must equal the caller fragment byte-for-byte;
- the inserted item must be a leaf whose `ListParentAnchor` is exactly the target parent's physical-line start.

No standalone-fragment indentation rule is introduced.

## Existing structural operations

M24 changes only child append.

The following operations remain line-only and leaf-only:

- `PrepareRemoveListItem`;
- `PrepareInsertListItemBefore`;
- `PrepareInsertListItemAfter`;
- `PrepareMoveListItemBefore`;
- `PrepareMoveListItemAfter`.

M24 does not claim a public complete-subtree range and does not promote parent removal, sibling insertion around a parent subtree, or subtree movement.

## Complexity

Let `l` be the number of supported list items.

Parse-time subtree resolution adds expected O(l) time and O(l) temporary memory:

- one temporary ID index;
- one supported-child count pass;
- one leaf-up queue pass.

`PrepareAppendListItemChild` remains one candidate construction plus one semantic candidate parse and O(l) survivor validation. No repeated per-parent scans or recursive hierarchy walks are added.

## Devil's advocate review

### Risk: a hidden unsupported descendant is skipped

A supported parent can contain an unsupported complex child, so using only promoted source order would produce a false subtree boundary.

Mitigation: the semantic direct-child count includes unsupported items. Supported-child count mismatch marks that node incomplete, and incompleteness propagates to all supported ancestors.

### Risk: a supported child hides an unsupported grandchild

The parent's immediate child count can still match even though a deeper subtree is incomplete.

Mitigation: completeness is propagated leaf-up; every supported child must itself be complete before its parent can be complete.

### Risk: deep nesting causes recursion failure or quadratic scans

Recursive parent walks or repeated descendant scans could become fragile or O(l²).

Mitigation: M24 uses an iterative leaf-up queue and O(1)-expected temporary ID lookups, processing each supported parent edge once.

### Risk: append starts implying general subtree ownership publicly

A private append boundary could be misread as a reviewed public subtree range suitable for delete/move.

Mitigation: `ListSubtreeEnd` and completeness remain private. M24 changes only `PrepareAppendListItemChild`; all line structural operations remain leaf-only.

### Risk: existing first-child behavior changes

M21 already proved leaf append semantics and boundary handling.

Mitigation: every supported leaf resolves as a complete subtree whose end is exactly its existing `LineRange.End`, so M21 follows the same candidate-validation path unchanged.

## TDD evidence

M24 began with focused public tests that failed only because an existing parent was still rejected by the M22 leaf-only target gate.

Focused tests now pass for:

- append after one existing direct child;
- append after the deepest nested descendant while creating a direct sibling child;
- ordered parents with variable marker width, CRLF, and Unicode;
- a second append after the M21 result is reparsed;
- rejection when the parent has an unsupported direct child;
- rejection when an unsupported grandchild makes an otherwise supported parent subtree incomplete;
- M21 first-child and M23 parent-identity regressions;
- focused M18–M24 list-item regressions together.

M24 is green. The complete repository verification stack passes with M21–M23 on the committed M18–M20 baseline: native `gofmt`, focused M18–M24 list regressions, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, text-hygiene checks, `git diff --check`, and a clean `git fsck --no-dangling` after storage recovery.
