# Milestone M29 — List-Item Subtree Insertion

Status: green — caller-provided complete list-item subtree sibling insertion passed.

## Goal

Extend the existing list-item sibling insertion operations from M26 leaf fragments to complete supported list-item subtrees supplied by the caller, while preserving M19 same-sibling lexical shape, M24 subtree completeness, M26 semantic sibling validation, and M28 descendant-placement proof.

## Public contract

M29 broadens the existing `PrepareInsertListItemBefore` and `PrepareInsertListItemAfter` methods. The anchor remains any complete supported list-item subtree. The caller fragment may now be a supported leaf or one complete supported parent subtree whose entire semantic list-item descendant structure can be proved independently.

No new public method, subtree range, generic fragment type, or normalization option is added.

## Standalone subtree proof

Caller bytes are untrusted and are not part of the host snapshot, so M29 parses the fragment as an independent Marksplice `Document` before candidate construction.

The standalone fragment must have exactly one supported root list item beginning at byte zero. That root must have no semantic list-item parent, pass the existing M24 `ListSubtreeComplete` proof, and own exactly `[0,len(fragment))` through `ListSubtreeEnd`. Every promoted list item in the standalone document must belong to that root subtree.

The root must also preserve the established M19/M26 same-sibling lexical shape relative to the host anchor: ordered state, marker/delimiter byte, and exact pre-marker physical-line prefix. Ordered numeric tokens remain caller-owned source and may differ.

Multiple root siblings, trailing bytes outside the subtree, unsupported descendants that make completeness unprovable, mismatched root marker/prefix shape, and empty fragments fail closed with `ErrInvalidReplacement`.

## Candidate insertion proof

`before` inserts at the anchor physical-line start; `after` inserts at the complete anchor subtree end. Marksplice never synthesizes indentation, numbering, whitespace, or line endings.

The host candidate must contain exactly the standalone subtree's supported item count in addition to the original supported items. All original host list-item mappings remain byte-for-byte/source-range stable modulo the insertion transform.

The inserted root must become a semantic sibling of the candidate anchor. Every inserted descendant must preserve its standalone subtree-relative physical-line, marker/source, and content ranges; ordered state; marker/delimiter; child state; direct-child count; exact physical-line bytes; and immediate parent anchor shifted by the common insertion delta.

## Validation consolidation

M29 generalizes M28's moved-subtree validator into one private subtree-placement proof used by both insertion and movement. Snapshot-owned moves still validate against the original document, while caller-owned insertions validate against the independently parsed fragment document.

This removes the obsolete leaf-only inserted-fragment validator rather than maintaining two algorithms for the same structural invariants.

## Post-M29 consolidation

Before continuing to new list capabilities, the M21-M29 implementation is consolidated without changing behavior or public API. List hierarchy/model ownership now lives in `list_item_model.go`, named mutation orchestration remains in `list_item_edits.go`, and fragment/candidate/sibling/subtree proof lives in `list_item_validation.go`.

The leaf-up subtree resolver uses an explicit source-ordered list-index collection instead of depending on Go map iteration order, and its temporary work arrays are indexed by compact supported-list ordinals rather than all document nodes. This preserves the documented O(l) temporary-memory bound. Remove, move, and caller-fragment insertion share one private subtree-ownership value containing the proven root, exact range, and supported-ID set.

Survivor and subtree-placement validation share one lexical mapping comparator for exact transformed ranges, marker state, and physical-line bytes. The survivor proof now also checks exact `DirectChildCount` for every supported item. The only permitted count changes are NodeID-keyed `-1/+1` deltas for operation-known source/destination parents, with same-parent movement reducing to an exact zero delta. This replaces the previous specialized remove/move/append parent-count checks and strengthens sibling insertion and content replacement, which now fail closed on a changed count even when `HasChildren` stays true.

A second-pass maintainability audit also moves observation-to-source capability mapping from `document.go` into `node_mapping.go`, where base node materialization and block/inline typed dispatch remain explicit per syntax family. The previous `nodeFromObservation` complexity hotspot is removed without introducing a generic mapper or changing unsupported-shape handling. Goldmark's table-cell observer drops an always-nil error return. The consolidation adds no new parser pass, persistent hierarchy index, public type, or complexity class.

## Source preservation and stale snapshots

The prepared insertion remains a single zero-width minimal patch containing the exact caller fragment bytes. Existing bytes are not rewritten or rendered. The resulting `ChangeSet` remains bound to the source fingerprint and rejects application to stale input with `ErrSourceConflict`.

## Compatibility and complexity

Leaf insertion remains a degenerate one-node subtree and therefore retains M19/M26 behavior. Complete-anchor semantics from M26 and complete moved-source semantics from M28 are unchanged.

Let `n` be host candidate size, `k` fragment bytes, `l` original supported list-item count, and `f` supported list items in the fragment. Standalone fragment parsing is O(k), host candidate parsing is O(n+k), and mapping/subtree validation is O(l+f). No recursive descendant walk, renderer, second persistent hierarchy index, or public batch surface is introduced.

## Devil's advocate review

### Risk: a fragment root appears valid while hiding unsupported descendants

Mitigation: the standalone root must pass the existing semantic direct-child-count-based `ListSubtreeComplete` proof and its subtree must own the complete fragment byte range.

### Risk: a valid standalone hierarchy reparses differently inside the host

Mitigation: every inserted descendant is compared against its standalone mapping at one common offset, including child state/count and shifted immediate parent anchors; the root is separately required to be a semantic sibling of the candidate anchor.

### Risk: multiple roots or trailing bytes are silently accepted

Mitigation: the unique supported root must begin at byte zero, its complete subtree range must equal `[0,len(fragment))`, and all promoted fragment list items must belong to that subtree.

### Risk: M28 movement regresses during validator reuse

Mitigation: the existing move path calls the same generalized validator with its original snapshot document/range/ID set, and focused M28 movement regressions are run together with M29 insertion tests.

### Risk: host joins require whitespace repair

Mitigation: no repair is attempted. The exact caller bytes are inserted and the full host candidate parse must prove the requested sibling structure or the operation fails closed.

## TDD evidence

M29 began with focused public tests for root/nested/deep subtrees, complete parent anchors, CRLF/Unicode and caller-owned ordered numbering, standalone-fragment rejection, and stale-source behavior. The initial test run failed at preparation with `ErrInvalidReplacement` because M26 still required exactly one leaf mapping.

After the minimal implementation and validator consolidation, those focused M29 tests pass. Focused M19/M26 insertion and M28 subtree-move regressions also pass together.

M29 is green. The strict repository verification stack passes: native `gofmt`, focused M19/M26/M28/M29 list regressions, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, UTF-8/LF/no-trailing-whitespace checks for the M29 implementation/test/milestone files, `git diff --check`, and `git fsck --no-dangling`.

The post-M29 consolidation reran the focused list regressions and the complete normal/race/test/vet/build/documentation/GFM/static/security stack successfully before proceeding to another milestone.

A second independent audit then strengthened the survivor proof with exact direct-child-count validation, confirmed the new invariant first with focused failing tests, and reran those tests green after the shared NodeID-delta implementation. List/model regressions pass 20 consecutive runs and the complete repository suite passes 5 consecutive runs without nondeterministic failures. The final verification passes normal and race tests, vet, build, generated package documentation, the published-GFM conformance gate, `staticcheck`, standard `golangci-lint` with zero issues, production-only `gocyclo`/`unparam` with zero issues, `govulncheck` with no vulnerabilities, `gitleaks` with no leaks, text-format checks, `git diff --check`, and `git fsck --no-dangling`.
