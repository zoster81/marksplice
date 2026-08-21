# Milestone M20 — Leaf List-Item Move

Status: green — leaf list-item move passed.

## Goal

Add atomic source-preserving movement of one complete promoted leaf list-item line immediately before or after a same-shape promoted leaf anchor.

M20 reuses M18 physical-line ownership and M19 sibling-shape proof. It deliberately does not compose an independently prepared removal with an independently prepared insertion: one named move operation owns both patches and validates the combined candidate.

## Public contract

M20 adds:

- `Document.PrepareMoveListItemBefore(itemID, anchorID) (ChangeSet, error)`;
- `Document.PrepareMoveListItemAfter(itemID, anchorID) (ChangeSet, error)`.

At the M20 exit, both IDs had to identify promoted M4 leaf single-line list items. M27 later broadens the destination anchor to a complete supported parent subtree, and M28 broadens the moved source as well when its complete supported subtree is proven.

Self-move is invalid.

## Source-preserving move contract

The moved bytes are exactly the original item's private `ListItemMapping.LineRange`:

- physical indentation/container prefix;
- marker or ordered number/delimiter;
- post-marker spacing;
- content and inline/task source;
- trailing source on the physical line;
- line terminator when present.

Those bytes are reinserted unchanged at the destination anchor boundary.

`before` targets `anchor.LineRange.Start`.

`after` targets `anchor.LineRange.End` for a leaf. M27 later uses the equivalent private `ListSubtreeEnd` boundary, which remains the same for a leaf and extends past all proven descendants for a complete parent anchor.

M20 prepares one source-bound `ChangeSet` with two disjoint patches in original snapshot coordinates:

1. delete the moved `LineRange`;
2. insert the exact moved bytes at the anchor boundary.

No ordered-list renumbering, indentation rewrite, marker normalization, or line-ending repair occurs.

## Same-shape destination proof

Before candidate construction, the moved physical line is parsed as the same standalone leaf fragment used by M19 and validated against the destination anchor.

The moved line must therefore have:

- the same exact pre-marker structural prefix as the anchor;
- the same ordered/unordered state;
- the same marker/delimiter byte.

An ordered numeric token may differ and is preserved byte-for-byte.

This allows movement between distinct semantic list parents when their concrete sibling source shape matches—for example, moving a three-space-indented `-` child from one ordered parent to another—while rejecting accidental promotion/demotion caused by a different prefix.

## Combined candidate validation

M20 renders both patches together and parses the resulting candidate once.

All original promoted leaf list items except the moved item must survive with the expected ranges after both original-coordinate patch transforms:

- `LineRange`;
- marker/source `Range`;
- `ContentRange`;
- ordered state;
- marker/delimiter;
- byte-identical complete physical line.

The moved line must reappear at its exact calculated candidate offset and reproduce its standalone mapping and bytes.

M20 deliberately does not require total promoted leaf count equality. Removing the last nested child from its source parent can legitimately make that parent newly promotable as a leaf, just as established by M18.

## Generic patch-range transform

The single-patch range-shift helpers established by section mutations are generalized into `internal/splice/mutation.go` because M18–M20 also depend on them.

M20 introduces a private `patchTransform` and `rangeAfterPatches` helper that:

- accepts patch ranges in original snapshot coordinates;
- validates ordering/non-overlap after sorting a private copy;
- accumulates byte deltas across disjoint patches;
- treats zero-width insertion boundaries with half-open range semantics;
- rejects a structural range that overlaps a replaced/deleted range or spans an insertion point.

`rangeAfterPatch` now delegates to the generic multi-patch helper, preserving M12–M19 behavior.

The section-specific move offset helper is likewise generalized to `movedRangeCandidateOffset` and reused by list-item movement.

This remains internal mutation plumbing, not a public batch API.

## Adjacent no-op behavior

A requested move that is already exactly satisfied is represented as a zero-patch snapshot-bound `ChangeSet`:

- moving an item `before` an anchor when its `LineRange.End == anchor.LineRange.Start`;
- moving an item `after` an anchor when `anchor.LineRange.End == item.LineRange.Start`.

Destination shape is validated before recognizing the no-op. M27 additionally requires original semantic siblinghood and uses the complete anchor subtree end for `after` adjacency; M28 applies the same boundary to complete moved source subtrees.

The returned no-op remains fingerprint-bound and therefore rejects stale source.

A blank line or other bytes between two items means they are not already immediately adjacent; M20 does not silently treat source-order equality as a no-op.

## Boundary behavior

A final moved item without a line terminator may be a valid standalone leaf, but moving it before another line would concatenate the moved content with following source unless a separator already exists.

M20 does not synthesize a terminator. Combined candidate parsing and inserted-fragment validation reject such unsafe moves.

The same principle applies when moving a terminated item after a final unterminated anchor.

## Error contract

M20 adds no new public sentinel:

- missing source/anchor ID: `ErrNodeNotFound`;
- wrong source/anchor kind: `ErrInvalidTargetKind`;
- self-move, sibling-shape mismatch, unsafe boundary, or candidate mapping failure: `ErrInvalidReplacement`;
- stale application, including stale no-op: `ErrSourceConflict`.

## Architecture and complexity

Let `n` be candidate size and `l` the original promoted leaf-item count.

A non-no-op move performs:

- standalone proof of the moved line, O(k);
- one two-patch candidate construction, O(n+k);
- one candidate semantic parse/mapping pass, O(n+k);
- one O(l) survivor-validation pass with O(1)-expected mapping lookup.

Patch transforms are sorted once; M20 currently has exactly two.

No persistent list index, list AST renderer, generic public tree edit, or generic public batch operation is introduced.

## Devil's advocate review

### Risk: remove + insert validate independently but fail together

Two individually valid changes can interact semantically after both are applied.

Mitigation: M20 never composes independently prepared changes. It creates the two patches internally, renders their joint candidate, and validates that candidate once.

### Risk: forward move offsets are calculated after deletion incorrectly

Destination coordinates are expressed in the original snapshot, while the moved line's candidate offset changes when the source lies before the destination.

Mitigation: `movedRangeCandidateOffset` explicitly subtracts the removed length for forward moves and keeps the insertion offset for backward moves. Focused tests cover both directions.

### Risk: cross-parent move changes nesting accidentally

Equal marker style alone is insufficient to prove compatible nesting.

Mitigation: M19's exact pre-marker prefix proof is reused before movement. The moved item must remain a promoted leaf in the candidate at the exact destination mapping.

### Risk: source parent becomes a new leaf

Moving its only child can change the public leaf observation set.

Mitigation: survivor validation is identity/mapping based rather than count based; legitimate newly promoted source parents are allowed.

### Risk: adjacent no-op stops being stale-safe

Returning raw source or an unbound empty value could bypass snapshot conflict semantics.

Mitigation: no-op uses the normal zero-patch `ChangeSet` constructor on the immutable source snapshot.

### Risk: generic multi-patch helper weakens Section behavior

Moving range math out of `section_validation.go` could subtly alter half-open boundary semantics established in M12–M16.

Mitigation: existing section range tests remain green, and new direct `rangeAfterPatches` tests cover forward/backward moves, insertion boundaries, overlaps, and invalid patch sets.

## Evidence and exit decision

M20 began with focused public tests that failed to compile solely because the before/after move APIs did not exist. The generic range-helper relocation compiled before the move implementation.

Focused tests now pass for:

- forward unordered move after an anchor;
- backward ordered move before an anchor while preserving the original number;
- nested CRLF/Unicode move across two different ordered parents with matching child prefix;
- source-parent leaf promotion after extracting its only child;
- adjacent before/after zero-patch no-ops;
- stale-source rejection for no-op and non-no-op changes;
- self-move rejection;
- structural-prefix mismatch rejection;
- missing/wrong source or anchor errors;
- fail-closed movement of an unterminated final line;
- direct multi-patch range-transform tests in both directions;
- regression of existing single-patch Section range semantics.

M20 is green. The complete repository verification stack passes with M18 through M20 together: native `gofmt` diff checks, focused list-item, section-regression, and shared multi-patch range-transform tests, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, and `git diff --check`.
