# Milestone M18 — Leaf List-Item Removal

Status: green — leaf list-item removal passed.

## Goal

Add source-preserving removal of one promoted leaf single-line list item without broadening M4 into multiline/list-subtree semantics.

M18 intentionally targets only the list-item shape already proven and promoted by M4: a list item whose Goldmark node contains exactly one single-line text/paragraph child. The item may itself be nested inside another list or container, but it has no child list subtree of its own.

## Public contract

M18 adds:

- `Document.PrepareRemoveListItem(NodeID) (ChangeSet, error)`.

At the M18 exit, the ID had to identify an existing promoted M4 leaf `ListItem`, and the operation removed exactly one complete physical source line owned by that leaf item:

- container/list indentation before the marker;
- ordered number or unordered marker;
- post-marker spacing;
- item content;
- trailing horizontal source on the item line;
- that line's LF, CRLF, or isolated CR terminator when one exists.

At EOF without a line terminator, removal ends exactly at EOF.

Blank lines before or after the item are not owned by the item and remain byte-identical.

M25 later broadens the same public operation to a parent only when M24's private subtree-completeness proof establishes the entire supported list-item subtree boundary. Leaf behavior remains byte-identical to the M18 contract.

## Public detail compatibility

`ListItem.Range()` does not change. It remains the exact content-only span used by `PrepareReplaceListItem` and continues to exclude indentation, marker/numbering, spacing, and line endings.

M18 adds no structural range accessor to the public `ListItem` type.

Snapshot-scoped node IDs also remain unchanged because existing node `Range` semantics used for identity are not replaced by the new private structural line range.

## Private source mapping

M4's `source.ListItemMapping` is extended with a private `LineRange`.

Existing fields retain their meanings:

- `Range`: marker/ordered-number start through physical line content end;
- `ContentRange`: editable item content;
- `Ordered` and `Marker`: existing source syntax.

`LineRange` starts at the physical line start and ends after the physical line terminator when present. This explicitly captures the source ownership required by structural removal while preserving M4 replacement semantics.

The immutable splice node persists the complete `ListItemMapping` at parse time so structural operations do not rescan the original source to rediscover ownership.

## Candidate validation

After deleting `LineRange`, Marksplice parses the complete candidate once.

M18 does not require candidate leaf-list-item count equality. Removing a final nested child can legitimately turn its previously non-leaf parent into a newly promoted leaf item.

Instead, every original promoted leaf list item other than the target must still be present with:

- the expected patch-shifted `LineRange`;
- the expected patch-shifted marker/source `Range`;
- the expected patch-shifted `ContentRange`;
- the same ordered/unordered state;
- the same marker/delimiter byte;
- byte-identical complete physical-line source.

Candidate leaf mappings are indexed once by physical-line start. New leaf items that become promotable as a legitimate consequence of the removal are allowed.

## Source-preservation contract

M18 prepares one patch replacing exactly the target `LineRange` with empty bytes.

No other byte is normalized, regenerated, or rewritten. In particular, Marksplice does not collapse blank lines, renumber ordered lists, change indentation, or rewrite neighboring markers.

## Error contract

M18 adds no public error sentinel:

- missing target ID: `ErrNodeNotFound`;
- non-list-item target: `ErrInvalidTargetKind`;
- candidate structure that cannot preserve surviving promoted leaf mappings: `ErrInvalidReplacement`;
- stale application: `ErrSourceConflict`.

## Architecture and complexity

M18 adds `internal/splice/list_item_edits.go` for structural list-item operations rather than growing the existing block-content replacement file.

Let `n` be candidate document size and `l` the number of promoted leaf list items. Preparation performs:

- one O(n) candidate construction/semantic parse;
- one O(n)-bounded mapping pass over candidate observations;
- one O(l) validation pass over original promoted leaf nodes.

Candidate mappings use an O(1)-expected map keyed by physical-line start. No O(l²) repeated matching or new persistent index is introduced.

## Devil's advocate review

### Risk: using the existing node range leaves orphan indentation or line endings

M4's node/source range starts at the list marker/number and ends before the physical line terminator. Deleting it directly would leave indentation and/or an empty physical line.

Mitigation: M18 explicitly persists a separate private `LineRange` that owns the entire physical leaf line including its terminator.

### Risk: structural removal changes which list items are publicly promotable

Removing the only child of a parent list item can make that parent become a leaf item in the candidate. A strict candidate list-item count comparison would reject a semantically correct removal.

Mitigation: validate only original promoted leaf survivors. Newly promoted leaf observations are allowed.

### Risk: nested indentation is accidentally preserved as stray whitespace

A nested leaf's indentation precedes the marker and is outside M4's content/marker range.

Mitigation: `LineRange` starts at the physical source line start, so indentation belongs to the removed structural line.

### Risk: removal silently normalizes adjacent blank lines

Treating neighboring blank lines as list-item trivia would change author spacing.

Mitigation: `LineRange` includes only the target physical line and its own line terminator. Adjacent blank physical lines are never consumed.

### Risk: ordered lists get renumbered

A structural editor might be tempted to normalize numbering after deletion.

Mitigation: M18 performs only the exact source deletion. Surviving ordered-number bytes and delimiters must remain byte-identical.

## Evidence and exit decision

M18 began with two focused red tests:

- source mapping failed to compile because `ListItemMapping.LineRange` did not exist;
- public tests failed to compile because `PrepareRemoveListItem` did not exist.

Focused tests now pass for:

- CRLF and Unicode middle-item removal;
- nested leaf removal with indentation ownership;
- final EOF removal without a synthetic newline;
- removal of the last nested child while allowing its parent to become a newly promoted leaf;
- preservation of surrounding blank lines;
- task-list item removal;
- missing/wrong target errors;
- stale-source conflict behavior;
- exact private `LineRange` mapping for indented CRLF and container-prefixed list items.

M18 is green. The complete repository verification stack passes on top of the committed M10–M17 baseline: native `gofmt` diff checks, focused source/public list-item tests, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, and `git diff --check`.
