# Milestone M21 — Direct List-Item Child Append

Status: green — direct list-item child append passed.

## Goal

Add source-preserving insertion of one direct leaf list-item child beneath an existing promoted M4 leaf list item without hard-coding Markdown indentation widths.

M21 builds on M18 physical-line ownership and the M19/M20 candidate mapping pipeline. The caller supplies the complete child physical line, including indentation/container prefix and marker syntax. Marksplice does not synthesize indentation, line endings, or list numbering.

## Public contract

M21 adds:

- `Document.PrepareAppendListItemChild(parentID, fragment) (ChangeSet, error)`.

The target must be an existing promoted leaf `ListItem` in the immutable source snapshot.

The fragment must become exactly one promoted leaf list item whose immediate semantic parent in the candidate is the targeted list item.

The child may use a different list kind or marker from its parent. For example, an ordered parent may receive an unordered child, or an unordered parent may receive an ordered child when GFM permits that child list to begin in the host context.

## Why indentation is not calculated by a fixed rule

GFM list continuation indentation depends on the parent marker width `W` and the number `N` of post-marker spaces, with `1 <= N <= 4`. Subsequent blocks belonging to that list item are indented by `W + N` relative to the containing block context.

Therefore a fixed `+2` or `+4` indentation policy would be incorrect for ordered markers with variable digit width, different post-marker spacing, tabs, and container prefixes.

M21 treats indentation as caller-owned source and uses semantic candidate parsing to prove direct parentage instead of generating indentation.

A focused test covers a `10.  parent` item, where marker width plus two following spaces requires a five-space child continuation before the nested marker.

## Parser-independent parent metadata

The internal parser observation for a promoted leaf list item gains:

- `HasListParent`;
- `ListParentAnchor`.

Goldmark remains isolated inside `internal/parser/goldmark`. The adapter obtains the immediate parent relation through public AST relationships:

```text
leaf ListItem -> containing List -> parent ListItem
```

`ListParentAnchor` is a Marksplice-owned source fact: the physical-line start of the immediate parent list item's first source block. `HasListParent` distinguishes a root item from a nested item whose parent happens to begin at byte zero.

No Goldmark AST node, `ListItem.Offset`, or parser-specific type crosses the adapter boundary.

## Existing list-mutation strengthening

The immutable internal list-item node persists the semantic parent metadata.

M18–M20 candidate mappings now preserve both lexical mapping and semantic parent metadata. For every original promoted leaf survivor, candidate validation requires:

- the expected transformed `LineRange`;
- the expected transformed marker/source `Range`;
- the expected transformed `ContentRange`;
- identical ordered state and marker/delimiter;
- byte-identical physical-line source;
- identical root-versus-nested parent presence;
- the same parent physical-line anchor after patch-coordinate transformation.

This prevents a mutation from silently reparenting an otherwise byte-identical surviving leaf.

## Child insertion boundary

M21 inserts at:

```text
parent.ListItemSource.LineRange.End
```

Because the target is a leaf before the operation, this point is immediately after the complete parent physical line and before any following source.

The patch is zero-width and contains the caller fragment unchanged.

## Candidate validation

M21 intentionally does not require the child fragment to parse as a standalone root list item. A correctly indented child under a wide ordered marker can be too deeply indented to form a standalone root list item, while still being valid inside the real parent context.

Instead, Marksplice renders the host candidate once and requires:

1. every original promoted list item survives with its transformed lexical mapping and semantic parent relation;
2. the target parent changes from `HasChildren == false` to `HasChildren == true` while retaining its exact own-line source mapping;
3. exactly one additional candidate leaf appears at the insertion byte;
4. that leaf's complete `LineRange` exactly equals the caller fragment span;
5. its semantic `ListParentAnchor` equals the target parent's physical-line start;
6. candidate bytes across that span are byte-identical to the supplied fragment.

M22 later promoted the supported single-line-head parent shape, so the current candidate count increases by one rather than replacing the old leaf with the child. The original M21 feasibility result remains the basis for the parent-anchor proof; M22 owns the broader public parent surface and the `HasChildren` transition semantics.

## GFM ordered-list interruption rule

During TDD an intentionally non-canonical fixture exposed an important GFM rule: an ordered list that interrupts a paragraph must begin with start number `1`.

A nested ordered child such as `7) child` immediately following parent paragraph content therefore does not necessarily create the requested child list. M21 does not renumber it to `1`; candidate validation rejects it with `ErrInvalidReplacement`.

Focused tests retain this as a fail-closed regression.

## Boundary behavior

M21 never inserts a separator automatically.

If the target parent is the final physical line and has no line terminator, appending a child fragment would concatenate bytes onto the existing line. Candidate validation rejects that shape.

A terminated final parent may receive a final child that itself has no terminating newline, provided the candidate parser proves the requested direct-child relation.

Fragments with a leading blank line, multiple child item lines, or plain continuation text are rejected because the inserted span is not exactly one direct promoted leaf line.

## Error contract

M21 adds no public error sentinel:

- missing parent ID: `ErrNodeNotFound`;
- non-list-item parent: `ErrInvalidTargetKind`;
- empty/multiple/non-child fragment, invalid GFM interruption, or unsafe boundary: `ErrInvalidReplacement`;
- stale application: `ErrSourceConflict`.

## Architecture and complexity

M21 extends the existing `internal/splice/list_item_edits.go` family rather than introducing a parallel hierarchy implementation.

Let `n` be candidate bytes and `l` the current promoted leaf count. Preparation performs:

- one zero-width candidate construction, O(n+k);
- one semantic parse/mapping pass, O(n+k);
- one O(l) survivor-validation pass with O(1)-expected lookup by physical-line start.

No standalone fragment parse is needed for this operation, no list renderer is introduced, and no public generic batch API or Goldmark type is exposed.

## Devil's advocate review

### Risk: fixed indentation rules corrupt ordered/nested syntax

Marker widths and post-marker spacing vary, and container prefixes change the effective source indentation.

Mitigation: Marksplice does not generate child indentation. The caller supplies exact source and the host candidate parser proves direct parentage.

### Risk: a same-level sibling is mistaken for a child

A syntactically valid list item inserted after the target is not necessarily nested beneath it.

Mitigation: the inserted candidate leaf must report `HasListParent=true` and `ListParentAnchor` equal to the target physical-line start.

### Risk: existing leafs are silently reparented

Range/byte preservation alone would not detect a semantic parent change.

Mitigation: semantic parent metadata is persisted and validated for every original surviving promoted leaf across M18–M21.

### Risk: standalone fragment validation rejects valid deeply indented children

A child line valid at `W+N` inside an ordered parent may look like indented code when parsed alone.

Mitigation: M21 validates the fragment only in the real host candidate and requires exact inserted line ownership there.

### Risk: ordered child numbering is silently normalized

Changing `7)` to `1)` would violate source ownership.

Mitigation: caller bytes are never rewritten. If GFM does not permit the supplied start number in context, preparation fails closed.

## Evidence and exit decision

M21 began with two focused red tests:

- parser tests failed to compile because `HasListParent`/`ListParentAnchor` did not exist;
- public tests failed to compile because `PrepareAppendListItemChild` did not exist.

Focused tests now pass for:

- root leaf parent metadata;
- nested and deeply nested immediate-parent anchors;
- unordered parent with differently marked child;
- ordered marker width/post-marker-spacing (`W+N`) child indentation;
- nested leaf becoming a parent;
- blockquote/container-aware parent relation;
- task child insertion;
- final child without a terminator after a terminated parent;
- same-level sibling/plain-text/multiple-line/preamble rejection;
- ordered non-`1` interruption rejection;
- unterminated-parent rejection;
- missing/wrong target errors;
- stale-source conflict behavior;
- focused M18–M20 regression coverage after parent-relation validation was added.

M21 is green. The complete repository verification stack passes on top of the committed M18–M20 baseline: native `gofmt` diff checks, focused parser/list-mutation regressions, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, and `git diff --check`.
