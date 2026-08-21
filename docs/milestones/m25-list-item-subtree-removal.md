# Milestone M25 — List-Item Subtree Removal

Status: green — supported list-item subtree removal passed.

## Goal

Extend the existing `PrepareRemoveListItem` operation from M18 leaf-line removal to complete removal of a supported list-item subtree, reusing the private subtree-completeness/end proof established by M24 rather than introducing a second removal API or a public subtree range.

## Public contract

M25 broadens the existing operation:

```go
func (d *Document) PrepareRemoveListItem(id NodeID) (ChangeSet, error)
```

Behavior is now:

- supported leaf item: unchanged M18 behavior; remove exactly its complete physical `LineRange`;
- supported parent with a complete M24 subtree: remove the parent and all supported descendants through its private `ListSubtreeEnd`;
- supported parent with an incomplete semantic subtree: fail closed with `ErrInvalidTargetKind`.

No new public method or range accessor is added.

## Removal boundary

The exact removal patch is:

```text
[parent.ListItemSource.LineRange.Start, parent.ListSubtreeEnd)
```

For a leaf, `ListSubtreeEnd == LineRange.End`, preserving M18 byte behavior.

For a complete parent, the range includes the parent physical line, every descendant physical line, and source bytes lying between those proven lines. It does not consume bytes after the final descendant line merely to normalize spacing. In particular, following blank lines remain caller source unless they are already inside the proven subtree span.

## Why subtree completeness is required

M22 may expose a simple parent even when one of its descendants is a complex unsupported list item. Removing only the visible supported lines would detach or leave behind source that semantically belonged to the requested parent.

M25 therefore uses the same `ListSubtreeComplete` gate as M24. An incomplete direct or deep descendant makes the operation non-actionable rather than causing Marksplice to guess ownership.

## Survivor validation with multiple removed IDs

M18's original candidate validator skipped one target ID. That is insufficient for subtree removal because every descendant inside the patch is intentionally absent from the candidate.

M25 generalizes the private list survivor validator to accept a set of skipped snapshot IDs.

A helper scans the original promoted list items once and collects all item IDs whose complete physical `LineRange` is contained in the removal range. The target must be present in that set.

The same set-aware validator is reused by existing operations:

- replacement skips exactly the replaced target while separately validating its new mapping;
- move skips exactly the moved leaf at its old location;
- insert/append skip no original items;
- subtree removal skips every removed supported item.

This removes a one-ID special case instead of adding subtree-specific validator duplication.

## Candidate count and surviving structure

After the removal candidate is parsed, Marksplice requires:

```text
candidate supported item count == original supported item count - removed supported item count
```

This rejects unexpected promotions or losses caused by an unsafe source join.

Every non-removed supported item must still preserve its transformed:

- complete physical-line mapping;
- marker/source and content ranges;
- ordered state and marker/delimiter;
- root/nested parent presence;
- parent physical-line anchor;
- physical-line bytes;
- child-state except at the explicitly authorized outer parent of the removed subtree.

## Outer-parent transition

When the removed subtree is nested beneath a supported parent, that parent loses exactly one direct semantic child subtree.

M25 therefore additionally requires the candidate outer parent's direct-child count to be:

```text
original direct-child count - 1
```

If the removed subtree was its only child, `HasChildren` may legitimately transition from true to false. That state change remains permitted only at the known immediate parent line anchor.

Root-level subtree removal has no outer-parent count check.

## Source preservation

M25 prepares one deletion patch only. It does not:

- renumber surviving ordered lists;
- rewrite list markers;
- regenerate indentation;
- collapse blank lines;
- synthesize separators or line endings;
- render a Markdown AST.

All source outside the exact subtree patch remains byte-identical.

## Complexity

Let `l` be the number of supported list items and `n` candidate source bytes.

Preparation adds:

- one O(l) pass to collect the supported IDs inside the already-proven subtree range;
- one candidate construction/parse/mapping pass, O(n);
- one O(l) survivor-validation pass with O(1)-expected lookup by physical-line start;
- O(1)-expected lookup of the supported outer parent through the existing snapshot node index.

No recursive descendant walk, persistent subtree index, or public generic batch operation is added.

## Devil's advocate review

### Risk: only the parent ID is skipped during validation

Descendants intentionally removed by the same patch would then be treated as lost survivors.

Mitigation: M25 generalizes the validator to an explicit skip-ID set and collects every supported item fully contained by the removal range.

### Risk: an unsupported descendant is silently left behind

The parent can be public even when the descendant model is incomplete.

Mitigation: removal requires `ListSubtreeComplete`; the same semantic-count completeness proof established by M24 detects unsupported direct and deep descendants.

### Risk: source joining promotes or absorbs another list item

A deletion may preserve bytes while changing neighboring GFM structure.

Mitigation: candidate supported-item count must be exactly original minus removed items, and every surviving promoted item must re-establish its exact transformed mapping and parent relation.

### Risk: removing a nested subtree leaves the outer parent semantic count unchanged

Checking only `HasChildren` would miss the difference when the outer parent has several children.

Mitigation: M25 requires the supported outer parent's semantic direct-child count to decrease by exactly one.

### Risk: M18 leaf semantics regress

Changing the operation to subtree-aware could broaden a leaf deletion unexpectedly.

Mitigation: every supported leaf is already a complete subtree whose `ListSubtreeEnd` equals `LineRange.End`; the same one-patch byte boundary is therefore retained.

## TDD evidence

M25 began with focused public tests that failed only because the existing M18 target gate rejected parent items with `ErrInvalidTargetKind`.

Focused tests now pass for:

- root parent removal with child and grandchild descendants;
- nested parent-subtree removal while preserving a sibling under the outer parent;
- ordered CRLF and Unicode source preservation;
- outer parent transition to `HasChildren == false` after removing its only child subtree;
- rejection of an unsupported direct child subtree;
- rejection when an unsupported grandchild propagates incompleteness;
- stale-source conflict behavior;
- unchanged M18 leaf-removal regressions;
- focused M18–M25 list-item regressions together.

M25 is green. The complete repository verification stack passes on top of M21–M24 and the committed M18–M20 baseline: native `gofmt`, focused M18–M25 list regressions, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, text-hygiene checks, `git diff --check`, and `git fsck --no-dangling` after storage recovery.
