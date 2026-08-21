# Milestone M31 — List-Item Subtree Replacement

Status: green — complete supported list-item subtree replacement passed.

## Goal

Add source-preserving replacement of one complete supported list-item subtree while keeping the surrounding list/container relationship stable.

M31 complements the established list mutation set:

- `PrepareReplaceListItem` continues to replace only the first-line semantic content range;
- `PrepareRemoveListItem` removes one complete supported subtree;
- M31 replaces that complete subtree with another complete supported subtree.

## Public contract

M31 adds:

```go
func (d *Document) PrepareReplaceListItemSubtree(id NodeID, replacement []byte) (ChangeSet, error)
```

The target must pass the existing M24 complete-subtree gate. Empty replacement is rejected because deletion already belongs to `PrepareRemoveListItem`.

The replacement may change root text, ordered numeric token, direct-child count, descendant depth, descendant markers, and descendant list kinds. The replacement root must preserve the target root's external sibling shape: ordered-versus-unordered state, marker/delimiter byte, and exact physical-line prefix before the marker. Ordered numbering remains caller-owned source and may change.

The replacement root must also preserve whether the target has an immediate semantic list-item parent and, when nested, must retain the same transformed parent physical-line anchor.

## Host-context proof

M31 deliberately does not parse the replacement as a standalone document.

A nested list-item subtree can be valid only because of indentation and container context supplied by its parent. Standalone parsing would therefore reject valid nested replacements or encourage Marksplice to invent indentation rules.

M31 instead:

1. obtains the target's private complete-subtree ownership;
2. replaces exactly `[LineRange.Start,ListSubtreeEnd)` with the caller bytes;
3. parses the resulting host candidate once through the normal Marksplice pipeline;
4. resolves one complete candidate subtree beginning exactly at the original target start;
5. requires that subtree to own exactly the complete replacement byte span;
6. validates all surviving original list items across the replacement patch;
7. proves the replacement root's external sibling shape and semantic parent relation.

The same candidate `Document` supplies both list-item mappings and private subtree completeness, so M31 adds no second candidate parse.

## Shared exact-subtree validation

M31 extracts a small `exactListItemSubtree` helper from the M30 child-append proof.

The helper proves only operation-neutral facts:

- non-empty candidate bytes;
- a supported root starts at the requested byte;
- the root is `ListSubtreeComplete`;
- private subtree ownership resolves successfully;
- `ListSubtreeEnd` produces exactly the requested byte span;
- candidate bytes in that span equal the caller bytes.

M30 then adds direct-child parent requirements. M31 separately adds replacement-specific external parent and sibling-shape requirements. This avoids merging distinct semantic policies into a generic mutation framework.

## Survivor and count invariants

All original supported list items inside the replaced subtree are intentionally removed from survivor validation.

Every original supported item outside the target subtree must retain its transformed line/source/content ranges, physical bytes, marker state, parent anchor, child state, and exact semantic direct-child count.

The candidate supported-item count must equal:

```text
original supported count - replaced subtree item count + replacement subtree item count
```

The external parent receives no child-count delta: one direct target root is replaced by exactly one direct replacement root. A changed external direct-child count therefore fails closed.

## Fail-closed boundaries

M31 rejects:

- empty replacement;
- multiple replacement sibling roots;
- leading/trailing bytes not owned by the replacement subtree;
- unsupported descendants that make subtree completeness unprovable;
- replacement roots with a different external marker/delimiter or prefix;
- nested roots that escape or change their semantic list parent;
- unterminated replacement bytes that merge with following untouched source;
- incomplete target subtrees.

No separator, indentation, marker, numbering, or line ending is synthesized.

## Errors

M31 adds no new public sentinel:

- missing ID: `ErrNodeNotFound`;
- wrong kind or incomplete target subtree: `ErrInvalidTargetKind`;
- ambiguous/unsafe replacement: `ErrInvalidReplacement`;
- stale application: `ErrSourceConflict`.

## Complexity

Let `n` be host source bytes, `k` replacement bytes, `l` original supported list items, and `r` replacement supported list items.

Preparation performs one candidate construction and one full candidate parse, O(n+k), plus linear list survivor/ownership validation O(l+r). The parse-time list hierarchy remains O(l) temporary memory. M31 adds no renderer, recursive hierarchy walk, persistent second list index, standalone replacement parse, or generic batch API.

## Devil's advocate review

### Risk: the replacement consumes the following sibling

A missing line ending or changed indentation can make untouched following source part of the replacement root's semantic subtree.

Mitigation: the candidate root's private subtree range must equal exactly the caller replacement span, while every following survivor must reappear at its transformed mapping.

### Risk: the replacement changes the surrounding list/container

Changing `-` to `*`, `.` to `)`, ordered to unordered, or the source prefix can split or regroup surrounding list source even if the new root remains parseable.

Mitigation: the candidate root must preserve the target root's same-sibling lexical shape and external semantic parent relation.

### Risk: an unsupported descendant is hidden inside an apparently valid root

Mitigation: the candidate root must pass the existing semantic direct-child-count-based `ListSubtreeComplete` proof before ownership is accepted.

### Risk: M30 and M31 drift into duplicate subtree-boundary algorithms

Mitigation: exact candidate subtree ownership is factored once; operation-specific parent/sibling policies remain separate.

## TDD evidence and exit decision

Focused public tests were introduced first and failed to compile because `PrepareReplaceListItemSubtree` did not exist.

After the minimal implementation, focused tests pass for root and nested subtree replacement, deeper replacement descendants, CRLF, Unicode, caller-owned ordered numbering, external-parent preservation, exact child identities, ambiguous fragment rejection, unsupported descendants, unsafe joins, incomplete targets, error categories, and stale-source behavior.

List-item mutation/parentage regressions pass five consecutive runs, including the existing content-only `PrepareReplaceListItem` contract. The final M31–M33 repository verification after the M33 resolver refactor passes five complete suite runs, race tests, coverage, vet, build, package documentation, and the published GFM 0.29 conformance gate. `staticcheck`, standard `golangci-lint`, and production-only `gocyclo`/`unparam` pass with zero issues; `govulncheck` reports no vulnerabilities and `gitleaks` reports no leaks. Text hygiene, `git diff --check`, and `git fsck --no-dangling` also pass.
