# Milestone M32 — List-Item Child Identities

Status: green — public immediate supported-child identities passed.

## Goal

Complete bidirectional public navigation of the supported list-item hierarchy without forcing callers to scan every promoted node and without introducing quadratic hierarchy walks.

M23 added upward navigation through `ParentID()`. M32 adds source-ordered downward navigation for immediate supported children.

## Public contract

M32 extends `ListItem` with:

```go
func (i ListItem) ChildIDs() []NodeID
```

`ChildIDs` returns the snapshot-scoped identities of the immediate semantic child list items that are themselves supported/promoted by Marksplice, in physical source order.

The returned slice is an independent copy. Mutating it cannot change the immutable `Document` or another `ListItem` detail value.

## `HasChildren` versus `ChildIDs`

The two APIs intentionally answer different questions:

- `HasChildren()` reports whether the item has one or more immediate semantic child list items, including unsupported shapes;
- `ChildIDs()` reports only immediate children that have public Marksplice identities.

Therefore `HasChildren() == true` with `len(ChildIDs()) == 0` is valid when all semantic direct children are outside the promoted subset.

M32 does not synthesize identities for unsupported list items and does not broaden the M22 promotion boundary.

## Compact adjacency model

M23 deliberately avoided a persistent second hierarchy map. M32 preserves that principle while making downward traversal efficient.

During the existing M24 list-model resolution pass, Marksplice now builds one compact source-ordered adjacency representation:

- every supported `Node` stores `ListChildStart` and `ListChildCount` integers;
- `Document` owns one flat `[]NodeID` containing all supported direct-child edges;
- children of one parent occupy one contiguous span in that flat array.

No persistent `map[NodeID][]NodeID` is introduced.

The existing source-ordered list-index collection is validated as strictly increasing by physical-line start. M32 does not sort it: unexpected ordering fails closed rather than changing the established O(l) hierarchy pass into O(l log l).

## Construction

The M24 resolver already knows each supported item's resolved `ListParentID` and supported direct-child count.

M32 reuses those facts:

1. collect supported list-item node indexes in strict source order;
2. build the existing temporary `NodeID -> compact ordinal` map;
3. count supported direct children per parent;
4. prefix-sum those counts into one child-offset array;
5. scan supported items in source order and write each child ID into its parent's contiguous flat span;
6. persist only the flat child-ID array plus each node's start/count;
7. continue the existing leaf-up subtree completeness/end proof.

Temporary count/offset/cursor arrays remain O(l) and are discarded after parse.

## Access path

`internal/splice.Document.ListItemChildIDs` validates the requested node, validates its compact span bounds, and returns a copied internal ID slice.

The root package converts those IDs into public opaque `NodeID` values when constructing `ListItem` typed detail. `ListItem.ChildIDs()` then returns another copy, preserving typed-detail immutability even when a caller retains and mutates the returned slice.

## Complexity

Let `l` be the number of supported list items and `c` the supported direct-child count for one item.

Parse-time hierarchy construction remains O(l) time and O(l) temporary memory. Persistent child-edge storage is O(l) because every supported non-root item contributes at most one parent edge.

`ParentID()` remains O(1). `ChildIDs()` is O(c) because it copies only the requested child span. A complete supported hierarchy traversal is O(l), not O(l²).

## Devil's advocate review

### Risk: semantic children are confused with public children

A simple parent may own an unsupported complex child. Treating `HasChildren` as `len(ChildIDs()) != 0` would erase that semantic fact or force unsupported nodes into the public API.

Mitigation: semantic child state/count remain parser-derived facts; the compact adjacency contains supported identities only and is documented separately.

### Risk: child order depends on Go map iteration

A map-backed adjacency fill could make public child order nondeterministic.

Mitigation: child IDs are filled by the strict physical source-order list-index sequence; maps are used only for ID-to-ordinal lookup.

### Risk: downward traversal becomes quadratic

Implementing `ChildIDs()` by scanning all document nodes on every call would make broad/deep hierarchy traversal O(l²).

Mitigation: one O(l) compact adjacency is built during parse and each accessor copies only O(c) IDs.

### Risk: callers mutate hierarchy state through returned slices

Mitigation: both internal and public access boundaries return copies; no stored adjacency slice is exposed directly.

### Risk: corrupt compact offsets panic on access

Mitigation: internal access validates non-negative start/count, addition overflow, and final bounds before slicing; invalid state fails closed.

## TDD evidence and exit decision

Focused public tests were written first and failed only because `ListItem.ChildIDs()` did not exist.

The implementation now passes tests for immediate-versus-descendant identity, source order, ParentID/ChildIDs symmetry, mixed supported/unsupported direct children, all-unsupported children with semantic `HasChildren == true`, independent returned copies, and updated snapshot identities after child append and reparse.

Focused M32 tests and the complete list-item hierarchy/mutation regression set pass five consecutive runs after the implementation. The initial deterministic-order implementation briefly used sorting; architecture review removed it before milestone exit and replaced it with strict source-order validation so the existing O(l) model remains literal. The later M33 refactor removes a duplicate temporary ID map and reconstruction pass while preserving the same compact ordinals and adjacency contract. Final M31–M33 verification passes five complete suite runs, race, coverage, vet, build, package documentation, published GFM 0.29 conformance, `staticcheck`, standard `golangci-lint`, production-only `gocyclo`/`unparam`, `govulncheck`, `gitleaks`, text hygiene, `git diff --check`, and `git fsck --no-dangling`.
