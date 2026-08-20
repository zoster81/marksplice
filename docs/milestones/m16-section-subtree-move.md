# Milestone M16 — Section Subtree Move

Status: green — section subtree move passed.

## Goal

Add source-preserving movement of one complete section subtree before or after another same-level section, using one atomic source-bound operation rather than composing independently prepared remove/insert changes.

M16 completes the first coherent structural section primitive set established by M12–M15: remove, replace body, replace subtree, insert sibling, and move sibling.

## Public contract

M16 adds:

- `Document.PrepareMoveSectionBefore(sectionHeadingID, anchorHeadingID) (ChangeSet, error)`;
- `Document.PrepareMoveSectionAfter(sectionHeadingID, anchorHeadingID) (ChangeSet, error)`.

Both IDs are existing promoted governing heading identities in the same immutable document snapshot.

The moved and anchor section roots must have the same heading level. This gives `before`/`after` a precise sibling meaning without implicitly rewriting heading levels.

The moved source is exactly the existing `Section.Range()`, including all descendant subsections and source trivia already owned by that M9 subtree range.

## Same-level reparenting

The source and anchor do not need to have the same current parent.

A same-level section moved next to an anchor under another parent becomes a sibling of that anchor and therefore adopts the anchor's parent in the candidate hierarchy. The operation does not rewrite the moved heading or descendant levels.

For root sections, both moved and anchor remain root sections.

Moving between different levels is rejected. Promotion/demotion and heading-level rewriting require a separate reviewed operation.

## Atomic two-patch model

A non-no-op move prepares exactly two source patches against one original fingerprint:

1. replace the moved `Section.Range()` with empty bytes;
2. insert the exact original subtree bytes at the anchor boundary.

`before` uses the anchor `Section.Range().Start`.

`after` uses the anchor `Section.Range().End`, which is after the complete anchor subtree.

The existing internal source `ChangeSet` already supports sorted disjoint patches and applies them against original-source coordinates. M16 therefore extends only internal splice plumbing from a single-patch helper to a multi-patch helper; it does not add a public batch API.

The replacement bytes for the insertion patch are copied by the source `ChangeSet`, preserving immutability.

## No-op behavior

If the requested source order is already satisfied, such as moving `A` before immediately following `B`, or moving `B` after immediately preceding `A`, M16 returns a valid zero-patch `ChangeSet` bound to the current source fingerprint.

Applying that change returns byte-identical source. Applying it to a stale snapshot still returns `ErrSourceConflict`.

Moving a section relative to itself is invalid rather than treated as a no-op.

## Candidate validation

M16 does not validate the two patches independently. It renders the combined candidate once and validates that final document.

The validator first computes the expected source-ordered section sequence by:

1. removing the complete moved subtree from the original section slice;
2. finding the anchor in the remaining sequence;
3. inserting the moved subtree before the anchor or after the anchor's complete subtree.

The candidate must have exactly the original section count and reproduce that expected heading sequence.

For every expected heading M16 requires:

- identical heading level;
- identical ATX/Setext style;
- identical complete heading byte length and source bytes;
- identical content-range offset within the heading;
- identical content length.

This position-independent comparison is intentional: many original heading offsets legitimately change during a move, while their lexical/semantic heading boundaries must remain identical.

The moved subtree is additionally parsed as the same standalone section fragment proof used by M14/M15. In the candidate, every moved fragment section and heading must reproduce the standalone fragment ranges at the calculated destination offset. The moved root must occupy exactly the moved source bytes.

Finally, the candidate moved root and anchor must have identical parent presence and, when nested, the same candidate parent heading ID. This explicitly proves the requested sibling relation.

## Boundary safety

Moving existing bytes can still alter GFM semantics at both the removal and insertion joins.

For example, removing an intervening section can make preceding paragraph text become part of a following Setext heading. Likewise, moving a Setext subtree to a destination without sufficient existing separation can absorb neighboring text.

M16 does not add whitespace or line endings to repair such joins. If the candidate does not reproduce the expected heading sequence and moved-fragment ranges, preparation returns `ErrInvalidReplacement`.

## Error contract

M16 adds no new public error sentinel:

- missing source or anchor ID: `ErrNodeNotFound`;
- source or anchor that is not a supported section heading: `ErrInvalidTargetKind`;
- self move, different root levels, or unsafe candidate boundary: `ErrInvalidReplacement`;
- stale application: `ErrSourceConflict`.

## Architecture and complexity

M16 adds reusable internal multi-patch preparation helpers in `internal/splice/mutation.go`. Existing single-patch callers delegate to them, so previous mutation semantics remain unchanged.

Let `n` be document bytes and `h` the number of supported sections. M16 performs:

- O(h) discovery/copy of the logical expected section order;
- O(k) standalone parsing of the moved `k`-byte subtree;
- one O(n) candidate construction and parse;
- O(h) source-ordered heading validation;
- O(r) inserted-fragment validation for `r` moved subtree sections.

There is no repeated candidate parse per patch and no O(h²) heading search. The operation adds no persistent index, renderer, filesystem/network behavior, or public generic batch abstraction.

## Devil's advocate review

### Risk: composing remove and insert `ChangeSet`s validates the wrong states

Two independently prepared changes would each be bound to the original snapshot and would not validate the combined semantic result.

Mitigation: M16 creates one `ChangeSet` containing both patches and parses only the final combined candidate.

### Risk: source-before-destination and source-after-destination use different offsets

Using post-removal coordinates as patch coordinates would misplace forward moves.

Mitigation: both patches use original source coordinates, as required by `source.NewChangeSet`. A dedicated helper computes the moved subtree's final candidate offset by accounting for removal only when the destination originally follows the source.

### Risk: moved subtree accidentally changes parent or splits descendants

Moving only the root heading or using the anchor direct-body boundary could orphan descendants or insert inside an anchor subtree.

Mitigation: source bytes are exactly `Section.Range()` and `after` targets the anchor's complete `Range().End`. Candidate parent equality explicitly proves the moved root is a sibling of the anchor.

### Risk: duplicate heading text hides an ordering error

Human-readable heading labels are not stable identities.

Mitigation: expected ordering is computed using snapshot-scoped heading IDs. Candidate lexical comparison uses source order and exact heading bytes only after that identity-based expected order has been constructed.

### Risk: already-satisfied moves collide at a patch boundary

A delete and zero-width insert can share the same source offset, which the source patch engine correctly rejects as overlapping.

Mitigation: M16 detects an unchanged logical section order first and returns a fingerprint-bound zero-patch change.

### Risk: Setext joins change untouched heading boundaries

Moving bytes may join paragraph lines to Setext underlines even though all non-moved bytes remain byte-identical.

Mitigation: the combined candidate must reproduce every expected heading's lexical boundary and moved-fragment source ranges exactly. Unsafe joins fail closed.

## Evidence and exit decision

M16 began with focused public tests that failed to compile solely because `PrepareMoveSectionBefore` and `PrepareMoveSectionAfter` did not exist.

Focused public/internal tests now pass for:

- moving a subtree forward after a same-level sibling;
- moving a subtree backward before a same-level sibling;
- preserving descendant source and hierarchy;
- same-level reparenting across different parent sections;
- CRLF and Unicode moved content;
- snapshot-bound no-op moves when the requested order already exists;
- rejecting self moves and different-level anchors;
- missing/wrong target errors;
- stale-source conflicts;
- fail-closed Setext source-join behavior;
- logical move-order/index calculation in both directions;
- candidate offset calculation before/after removal.

M16 is green. The complete repository verification stack passes with M10 through M16 together: native `gofmt` diff checks, focused M12–M16 section regressions, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, and `git diff --check`.
