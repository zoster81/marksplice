# Milestone M23 — List-Item Parent Identity

Status: green — list-item parent identity passed.

## Goal

Make the M21/M22 list hierarchy directly navigable through snapshot-scoped Marksplice identities without exposing Goldmark nodes, inventing identities for unsupported parents, or adding a persistent second hierarchy index.

## Public contract

M23 extends `ListItem` with:

```go
func (i ListItem) ParentID() (NodeID, bool)
```

The returned ID identifies the immediate semantic parent list item in the same parsed source snapshot when that parent is itself a supported/promoted `ListItem`.

The boolean is false when:

- the item is a root list item; or
- an immediate semantic list-item parent exists but that parent is outside the supported public list-item subset.

M23 does not invent a public identity for an unsupported parent. Callers must not interpret `false` as proof that the item is semantically root-level; it means only that no public parent ID is available.

## Identity resolution

M21 already records a parser-independent `ListParentAnchor`, defined as the immediate parent list item's physical-line start. M22 ensures supported simple parents are promoted and therefore receive ordinary snapshot-scoped `NodeID` values.

M23 resolves those two facts after all semantic observations have been converted to internal nodes and their existing IDs have been assigned:

1. build a temporary map from each supported list item's `LineRange.Start` to its actual `NodeID`;
2. for each list item with `ListHasParent == true`, look up `ListParentAnchor` in that temporary map;
3. persist only the resolved `ListParentID` on the child node;
4. discard the temporary map before returning the immutable `Document`.

This is intentionally different from calculating a new ID from the parent anchor. Marksplice node IDs include the mapped node range as well as the source fingerprint and kind; an anchor alone is not the parent's identity.

## Unsupported parents

M4/M21 can observe a supported nested leaf even when its semantic parent has a more complex source shape that M22 deliberately does not promote, such as a parent with an additional paragraph or other non-list direct block.

In that case:

- the child keeps its internal semantic `ListHasParent`/`ListParentAnchor` facts for mutation safety;
- no supported parent node exists in the temporary line-start map;
- public `ParentID()` returns the zero `NodeID` and `false`.

This preserves the existing public child surface without silently promoting the complex parent.

## Snapshot semantics

`ParentID` is snapshot-local exactly like every other `NodeID`.

After a successful mutation and reparse, both parent and child receive IDs derived from the new source fingerprint. A child in the updated snapshot therefore returns the updated parent ID, not the parent ID from the pre-mutation document.

## Complexity

Let `l` be the number of supported list items in one parsed document.

Parent identity resolution adds:

- one O(l) pass to build a temporary line-start map;
- one O(l) pass to resolve parent IDs;
- O(l) temporary memory during parse;
- O(1) storage per list-item node for the resolved parent ID;
- O(1) public `ParentID()` access.

No persistent list hierarchy index, extra document lookup map, parser pass, source rescan, or Goldmark type is added.

## Devil's advocate review

### Risk: synthesizing an ID from the physical parent anchor produces a false identity

Node IDs are derived from source fingerprint, kind, and the mapped node range. The physical line start alone does not encode that range.

Mitigation: M23 resolves only actual IDs already assigned to supported parent nodes.

### Risk: resolving parents lazily by scanning all list items makes hierarchy walks quadratic

Repeated `ParentID()` calls over a deep or broad list could become O(l²).

Mitigation: resolve once during `Parse`; the public accessor is O(1).

### Risk: unsupported complex parents become accidentally public

A child may have a valid semantic parent anchor even when that parent is intentionally outside M22.

Mitigation: only already promoted/supported list-item nodes populate the temporary map. Missing entries remain unresolved rather than synthesizing nodes.

### Risk: a duplicate physical-line start makes parent resolution ambiguous

Two supported list items must not own the same physical line start.

Mitigation: temporary-map construction fails closed if a duplicate supported line start is observed.

## Evidence and exit decision

M23 began with focused public tests that failed only because `ListItem.ParentID()` did not exist.

Focused tests now pass for:

- root item returning no parent ID;
- child returning the immediate supported parent ID;
- grandchild returning the immediate child ID rather than the root ancestor;
- a supported child beneath an unsupported complex parent returning no public parent ID;
- blockquote/CRLF/Unicode container-aware resolution;
- an M21 append followed by reparse returning the updated snapshot parent ID.

M23 is green. The complete repository verification stack passes with M21–M22 on the committed M18–M20 baseline: `gofmt`, focused and full tests, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, text-hygiene checks, and `git diff --check`.
