# Milestone M34 — Section Child Identities

Status: green.

## Goal

Complete downward navigation for the existing M9 section hierarchy by exposing the snapshot-scoped heading identities of one section's immediate child sections in physical source order.

M34 must preserve the established `Section` value contract, including comparability, existing parent/range semantics, heading-based identity, and the O(h) section-building bound. It adds no new section identity namespace, parser pass, mutation, or filesystem/workspace behavior.

## Public contract

M34 adds:

```go
func (d *Document) SectionChildHeadingIDs(headingID NodeID) ([]NodeID, bool)
```

For a valid section heading ID, the method returns the governing heading IDs of that section's immediate child sections in source order. A valid leaf section returns an empty result with `true`.

The boolean is false for a nil document, a missing ID, or an ID that does not govern a promoted document section.

The returned slice is caller-owned. Mutating it cannot change the immutable document snapshot.

IDs remain snapshot-local `NodeID` values. M34 does not synthesize section IDs from heading text, levels, ranges, or ordinal positions.

## Why the accessor belongs to Document

The first TDD implementation stored `[]NodeID` directly in the public `Section` value and exposed `Section.ChildHeadingIDs()`.

That made `Section` non-comparable in Go and immediately broke an existing public regression that compares two `Section` values. A read-only navigation feature must not silently remove that source-level property from an established public value type.

M34 therefore keeps `Section` unchanged and places child lookup on `Document`, where the existing section index already owns snapshot-local hierarchy resolution.

## Internal model

M9 already processes supported headings in strict source order with a monotonic stack. At the moment a new section is created, the top of that stack is its immediate parent.

M34 reuses that same pass and stores only scalar adjacency metadata on the private `splice.Section` value:

- `firstChildIndex`;
- `nextSiblingIndex`;
- `childCount`.

A temporary `lastChildIndex` array lets the builder append each direct-child edge in O(1) while preserving source order. The temporary array is discarded after parsing.

No second heading-ID map, child-ID slice, hierarchy pass, recursive walk, or parser metadata is added. The existing `sectionIndex` remains the only heading-ID-to-section lookup map.

## Accessor validation

`SectionChildHeadingIDs` follows only the stored direct-sibling chain. It validates before returning data that:

- the requested parent index exists;
- child metadata uses valid sentinels and non-negative counts;
- every child index is strictly increasing in source order and within bounds;
- every returned child reports the requested section as its immediate parent;
- the chain contains exactly the stored child count.

Corrupt, cyclic, backward, truncated, or inconsistent internal adjacency fails closed with `(nil, false)` rather than panicking, looping, or returning a plausible but invalid hierarchy.

## Hierarchy semantics

Immediate-child semantics continue to come from M9 heading hierarchy, not from arithmetic level assumptions.

A level jump is valid. For example:

```text
# Root
### Deep child
#### Deeper
## Later child
```

`Deep child` and `Later child` are both immediate children of `Root`, while `Deeper` is an immediate child of `Deep child`.

Container headings remain outside the promoted section model, exactly as in M9.

## Complexity

Let `h` be the number of promoted document sections and `c` the number of immediate children returned for one section.

Section construction remains O(h) time. M34 adds one O(h) temporary integer array and O(1) scalar adjacency state per section. Lookup by heading ID remains O(1)-expected through the existing section index, followed by O(c) traversal and O(c) returned memory.

A complete traversal using repeated child lookups remains O(h) because each section has exactly one immediate parent edge.

## Devil's advocate review

### Risk: child identities make Section non-comparable

A slice field on public `Section` removes Go comparability and can break existing caller code at compile time.

Mitigation: keep the public `Section` representation unchanged and expose child lookup through `Document.SectionChildHeadingIDs`. The existing equality regression is retained and passes.

### Risk: skipped heading levels produce the wrong child relation

Treating `child.Level() == parent.Level()+1` as the definition of a child would reject valid hierarchy created by level jumps.

Mitigation: reuse the exact M9 monotonic-stack parent relation. Focused tests cover skipped levels and later shallower siblings.

### Risk: corrupted adjacency loops or returns children out of source order

A malformed next-sibling index could create a cycle, backward edge, invalid sentinel, or count mismatch.

Mitigation: the accessor validates bounds, strict index increase, sentinel values, immediate parent identity, and exact child count. Focused internal tests inject each failure shape and require fail-closed behavior.

### Risk: callers mutate document hierarchy through the returned slice

Returning internal storage directly would violate snapshot immutability.

Mitigation: child IDs are materialized into a fresh public slice on every successful call.

## TDD evidence

The first focused public test failed to compile because no child-identity API existed.

The first implementation then exposed a compatibility regression: adding a child-ID slice to public `Section` made the type non-comparable and broke the existing `got != want` section regression at compile time. The design was revised before proceeding.

The current implementation passes focused public tests for:

- multiple immediate children in exact source order;
- nested child/grandchild navigation;
- skipped heading levels;
- CRLF and Unicode source;
- leaf sections;
- defensive-copy behavior;
- nil, missing, zero, and non-section targets;
- unchanged public `Section` comparability.

Focused internal tests additionally cover out-of-range links, wrong-parent links, cycles, backward siblings, invalid sentinels, and child-count mismatches.

## Post-M34 consolidation

The immediate post-green review tightened reuse and fail-closed behavior before opening another milestone.

Focused red tests first proved two related index-integrity gaps: a corrupted `sectionIndex` entry could point a valid heading ID at another in-range section and still make `SectionByHeadingID` return that wrong section, and child adjacency could expose an ID whose `sectionIndex` entry resolved to a different section. The private `sectionByHeadingID` lookup now validates map membership, bounds, and `HeadingID` identity together and returns both the section and its index. `SectionByHeadingID`, M34 child lookup, moved-parent validation, and section mutation targeting reuse that one path; child lookup additionally requires each adjacency index to equal the authoritative index resolved for that child ID.

`sectionTarget` now returns the already validated section index, removing repeated `sectionIndex[id]` lookups from replace, insert, append-child, move, and body-replacement preparation. This reduces duplicated target/index validation without changing error categories or mutation semantics.

The M34 growth in `buildSections` was also split by responsibility: `collectSectionHeadings` owns promoted-heading collection/order validation, `newSection` owns one section's range initialization, and `linkSectionChild` owns source-ordered adjacency wiring. The existing monotonic stack remains the hierarchy algorithm and construction remains O(h). Because first-child state is explicit through `childCount`, the temporary last-child array no longer needs a separate O(h) sentinel-initialization pass.

Public integration coverage now also appends a section child, applies the source-bound change, reparses the result, and verifies that `SectionChildHeadingIDs` exposes the refreshed child identities in source order. The refactor additionally splits two unrelated historical monolithic public tests by semantic responsibility; production and test-inclusive `gocyclo`/`unparam` checks report zero issues after the cleanup.

## Exit decision

M34 is green. Focused public/internal hierarchy regressions pass, including the compatibility test that keeps `Section` comparable and the post-consolidation target/child index-integrity fail-closed regressions. The final repository-wide verification after the consolidation passes five consecutive `go test ./...` runs, `go test -race ./...`, coverage execution, `go vet ./...`, `go build ./...`, generated package documentation, the pinned published-GFM 0.29 conformance gate, `staticcheck ./...`, standard `golangci-lint run`, and both test-inclusive and production-only `gocyclo`/`unparam` with zero issues. `govulncheck ./...` reports no vulnerabilities and Gitleaks reports no leaks.

Final statement coverage is 89.9% for the public root package, 63.6% for `internal/parser/goldmark`, 79.0% for `internal/source`, and 60.8% for `internal/splice`. Final text-format, `git diff --check`, working-tree, and repository-integrity checks are recorded with the closing diff review.
