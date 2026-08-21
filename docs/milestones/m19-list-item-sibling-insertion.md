# Milestone M19 — Leaf List-Item Sibling Insertion

Status: green — leaf list-item sibling insertion passed.

## Goal

Add source-preserving insertion of one promoted-shape leaf list item immediately before or after an existing promoted leaf item, while proving that the inserted line has the same structural list shape as its anchor.

M19 builds on the private physical-line ownership introduced by M18. At its exit it was intentionally limited to M4 leaf single-line anchors and did not introduce multiline/list-subtree insertion. M26 later broadens the anchor to a complete supported parent subtree while keeping the inserted fragment itself a single M19 leaf line.

## Public contract

M19 adds:

- `Document.PrepareInsertListItemBefore(anchorID, fragment) (ChangeSet, error)`;
- `Document.PrepareInsertListItemAfter(anchorID, fragment) (ChangeSet, error)`.

The fragment is caller-provided Markdown source for exactly one complete physical leaf list-item line.

## Same-sibling-shape invariant

A valid fragment must reproduce the anchor's structural list shape:

- exact bytes before the marker/ordered number within the physical line, including indentation and container prefixes;
- same ordered versus unordered state;
- same unordered marker or ordered delimiter byte.

The ordered number itself is caller source and may differ from the anchor's number. Marksplice does not renumber ordered lists.

This invariant prevents an insertion operation named "sibling" from silently creating a differently indented child/parent or a separate list caused by a different bullet/delimiter style.

## Fragment proof

The fragment is parsed independently and must contain exactly one promoted leaf list item whose private `LineRange` spans the entire fragment from byte zero to `len(fragment)`.

Therefore fragments with:

- preamble bytes;
- multiple list items;
- plain text instead of a list item;
- unsupported non-leaf/multiline shapes;
- trailing blank lines outside the item line;

are rejected.

Inline/task syntax inside the single item content is allowed when the item remains an M4-proven leaf line.

## Insertion boundaries

`before` inserts at `anchor.LineRange.Start`.

`after` inserts at `anchor.LineRange.End`.

M26 later preserves those boundaries for a leaf but uses the complete private `ListSubtreeEnd` for `after` when the anchor is a proven parent subtree; `before` remains at the anchor physical-line start.

Both are zero-width patches using original-source coordinates.

M19 does not synthesize a newline. A fragment without a terminator may parse standalone but will be rejected if host candidate parsing merges it with the anchor. Likewise, insertion after a final unterminated anchor fails closed if the new item cannot remain a separate list item.

## Candidate validation

M18's candidate mapping logic is generalized and shared.

The candidate is parsed once into a map of promoted leaf `ListItemMapping` values keyed by physical-line start.

M19 requires:

1. every original promoted leaf list item survives with exactly patch-shifted `LineRange`, marker/source `Range`, and `ContentRange`, plus identical ordered state, marker byte, and physical-line bytes;
2. exactly one additional promoted leaf item exists;
3. the inserted item reproduces the standalone fragment mapping shifted to the insertion offset;
4. the inserted candidate bytes are exactly the caller fragment.

At the M19 exit, the leaf anchor itself was included in survivor validation, so candidate semantics that accidentally turned it into a parent/non-leaf item failed closed. M26 supersedes only that anchor restriction: a complete supported parent may be an anchor, and the inserted leaf must additionally prove the same immediate semantic parent relation as the candidate anchor.

## Error contract

M19 adds no new public sentinel:

- missing anchor: `ErrNodeNotFound`;
- non-list-item anchor: `ErrInvalidTargetKind`;
- invalid fragment shape, sibling-shape mismatch, or unsafe host boundary: `ErrInvalidReplacement`;
- stale application: `ErrSourceConflict`.

## Architecture and complexity

`internal/splice/list_item_edits.go` now owns M18/M19 structural list-item operations and shared leaf-mapping validation.

Let `n` be candidate bytes and `l` the promoted leaf count. M19 performs one standalone fragment parse/mapping O(k), one candidate construction/parse O(n+k), and O(l) survivor validation with O(1)-expected lookup by physical-line start.

No persistent list index, AST serialization, public source-trivia range, or generic batch API is introduced.

## Devil's advocate review

### Risk: matching only marker style allows wrong nesting

A nested `- item` and a root `- item` share the same marker but are not siblings.

Mitigation: the exact pre-marker physical-line prefix must match the anchor, including indentation/container bytes.

### Risk: requiring identical ordered numbers would normalize caller intent

Ordered-list source numbers may intentionally skip or repeat values.

Mitigation: only ordered state and delimiter are structural invariants; the numeric token remains caller source.

### Risk: fragment has two valid list items

Accepting both would make one API call ambiguous and complicate boundary ownership.

Mitigation: standalone fragment mapping must contain exactly one leaf item whose `LineRange` owns the entire fragment.

### Risk: fragment without EOL concatenates with anchor

Standalone parsing alone cannot prove host separation.

Mitigation: host candidate mapping must reproduce both all original items and the inserted fragment at exact shifted ranges. Unsafe joins fail closed.

### Risk: M18 and M19 validators drift

Independent survivor logic would create inconsistent structural guarantees.

Mitigation: M19 refactors M18 onto shared candidate mapping and original-survivor validation helpers.

## Evidence and exit decision

M19 began with focused public tests that failed to compile solely because the before/after insertion APIs did not exist.

Focused tests now pass for:

- top-level unordered insertion before;
- ordered insertion after while preserving a caller-selected number;
- nested CRLF/Unicode sibling insertion;
- blockquote/container prefix preservation;
- task-item fragment insertion;
- empty/wrong-prefix/different-marker/different-list-kind/multi-item/plain-text rejection;
- unterminated before-fragment rejection;
- fail-closed insertion after a final unterminated anchor;
- missing/wrong anchor errors;
- stale-source conflict behavior.

M19 is green. The complete repository verification stack passes with M18 through M20 together: native `gofmt` diff checks, focused list-item and shared range-transform regressions, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, and `git diff --check`.
