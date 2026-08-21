# Milestone M33 — List-Item Subtree Range

Status: green — public complete supported list-item subtree range passed.

## Goal

Promote the complete supported list-item subtree boundary as a read-only typed-detail contract now that M25 removal, M28 movement, M30 child-subtree append, and M31 subtree replacement have independently exercised and validated the same private ownership model.

M33 must not change the historical meaning of `ListItem.Range()`, broaden list-item promotion, or expose partial ranges for semantically incomplete subtrees.

## Public contract

M33 extends `ListItem` with:

```go
func (i ListItem) SubtreeRange() (Range, bool)
```

The returned range is the exact half-open source span owned by the complete supported list-item subtree:

```text
[ListItemSource.LineRange.Start, ListSubtreeEnd)
```

The boolean is true only when the existing private `ListSubtreeComplete` proof succeeds.

`ListItem.Range()` remains unchanged: it is still the exact first-line content span replaced by `PrepareReplaceListItem`. `SubtreeRange()` is a distinct structural-source contract.

## Completeness boundary

M22 intentionally permits a supported simple list-item parent even when a semantic descendant has an unsupported complex shape. Such an item may therefore report `HasChildren() == true` while Marksplice cannot prove ownership of every descendant byte.

M33 does not guess a partial structural range. For an incomplete subtree:

- `SubtreeRange()` returns the zero `Range` and `false`;
- the item remains available through the existing public `ListItem` detail;
- content-only operations retain their existing behavior;
- structural operations that require complete ownership continue to fail closed through their existing target gates.

For a supported leaf, the complete subtree range is exactly its private physical `LineRange`, preserving M18 semantics.

## Relationship to earlier milestones

M24 through M29 deliberately kept `ListSubtreeEnd` private because complete-subtree structural semantics were still being established operation by operation. Those historical milestone statements remain correct at their respective exit points.

By M33, complete-subtree ownership has been exercised by:

- M25 complete subtree removal;
- M28 complete subtree movement;
- M29 complete caller subtree placement;
- M30 complete direct-child subtree append;
- M31 complete subtree replacement.

M33 therefore promotes only the already-proven read boundary. It adds no mutation semantics and no second subtree model.

## Snapshot and source semantics

`SubtreeRange()` is snapshot-local like every other Marksplice range.

The returned value contains byte offsets only and does not alias internal memory. It composes directly with `Document.SourceRange`, which returns a copied byte slice.

After a successful structural mutation and reparse, the new snapshot may expose a different subtree range. Existing typed details retain their original snapshot-local values.

The range includes preserved line terminators owned by subtree physical lines and handles a final unterminated leaf by ending exactly at EOF. LF, CRLF, container prefixes, Unicode, markers, numbering, indentation, and blank source inside the proven subtree remain byte data rather than normalized syntax.

## Implementation

M33 adds no parser metadata, hierarchy pass, persistent index, source rescan, or candidate parse.

`Document.ListItem` already receives the internal list node. When `ListSubtreeComplete` is true it copies the existing private structural boundary into the public immutable typed detail and records availability explicitly. `SubtreeRange()` then returns that stored value in O(1).

The zero value of `ListItem` has no subtree range.

## Refactor review

The M31–M33 review also identified duplicate temporary identity storage in the M32 hierarchy resolver: `seenIDs` was populated only to detect duplicates, followed by a second pass that rebuilt the same supported IDs into `ordinalByID`.

The resolver now populates `ordinalByID` directly during the strict source-order collection pass. The existing map performs duplicate detection and stores compact ordinals at once.

This removes:

- one O(l) temporary map;
- one O(l) pass over supported list indexes;

while preserving the same fail-closed duplicate/source-order checks, O(l) time and O(l) temporary-memory complexity class, child adjacency, subtree completeness, and public behavior.

No broader refactor was justified: the simple public range projection remains clearer than adding another internal API solely to avoid one structural-range expression.

## Complexity

Let `l` be the supported list-item count.

M33 adds O(1) stored typed-detail state and O(1) accessor time per requested `ListItem`. It does not add parse-time asymptotic cost.

The accompanying resolver refactor reduces constant-factor temporary work while keeping the established hierarchy construction at O(l) time and O(l) temporary memory.

## Devil's advocate review

### Risk: callers confuse content range with structural range

Using one overloaded range contract would make existing content replacement semantics ambiguous.

Mitigation: `Range()` remains content-only and `SubtreeRange()` is a separately named `(Range, bool)` structural accessor with explicit documentation and tests proving the ranges differ.

### Risk: a public parent exposes a partial subtree boundary

A supported parent can hide unsupported direct or deep descendants.

Mitigation: M33 exposes the range only when the existing semantic-count-based `ListSubtreeComplete` proof succeeds; incomplete parents return `false`.

### Risk: public range semantics drift from structural mutations

A separately recomputed public boundary could diverge from remove/move/replace ownership.

Mitigation: M33 projects the same stored `LineRange.Start` and `ListSubtreeEnd` facts already used by the established structural operations. No new range-discovery algorithm is introduced.

### Risk: hierarchy refactoring changes child order or completeness

Mitigation: only duplicate temporary ID storage is removed. The same strict physical source-order collection, parent ordinal values, supported-child counts, adjacency fill, and leaf-up completeness queue remain in place and are covered by repeated M32/M33/list mutation regressions.

## TDD evidence and exit decision

Focused public tests were written first and failed only because `ListItem.SubtreeRange()` did not exist.

The implementation passes tests for:

- exact root, nested-parent, child, and leaf subtree bytes;
- clear distinction from the historical content-only `Range()`;
- CRLF and Unicode source;
- a final unterminated leaf ending exactly at EOF;
- incomplete direct and deep descendant rejection;
- range changes after append and reparse while the original snapshot remains unchanged;
- zero-value typed-detail behavior.

The focused M33 regression and the complete list-item hierarchy/mutation regression set pass after the resolver refactor. Final repository-wide verification passes five complete suite runs, race tests, coverage, vet, build, generated package documentation, and the published GFM 0.29 conformance gate. Final statement coverage is 89.6% for the public root package, 63.6% for `internal/parser/goldmark`, 79.0% for `internal/source`, and 59.7% for `internal/splice`. `staticcheck`, standard `golangci-lint`, and production-only `gocyclo`/`unparam` pass with zero issues; `govulncheck` reports no vulnerabilities and `gitleaks` reports no leaks. UTF-8/LF/no-trailing-whitespace checks, `git diff --check`, and `git fsck --no-dangling` also pass.

M33 is green.
