# Milestone M4 — List Public API

Status: green — list public API passed.

## Goal

Extend the reviewed public typed-detail and named-operation pattern to the two list-family shapes whose exact source mapping is already established during parsing: single-line list items and GFM task markers.

M4 must not promote unrelated M1 internals or syntax families whose editable source shape is only proven lazily when a mutation is prepared.

## Scope

M4 promotes:

- public `KindListItem` and `KindTask` summaries;
- immutable `ListItem` typed detail;
- immutable `Task` typed detail;
- `Document.ListItem(NodeID)` and `Document.Task(NodeID)`;
- `Document.PrepareReplaceListItem(...)`;
- `Document.PrepareSetTaskChecked(...)`.

Tables, fenced code, links, inline syntax, front matter, HTML, and other M1 families remain internal.

## List item contract

Public list items correspond only to the single-line source shapes already mapped by M1. Nested items are included because M1 proved their mapping and replacement while preserving indentation.

`ListItem.Range()` is the exact content span replaced by `PrepareReplaceListItem`. It excludes indentation, ordered-list numbers, marker/delimiter bytes, post-marker spacing, and the following line ending.

`ListItem.Ordered()` identifies ordered versus unordered syntax. `ListItem.Marker()` returns the existing ASCII marker/delimiter byte:

- `-`, `*`, or `+` for unordered items;
- `.` or `)` for ordered items.

The ordered-list number itself remains source trivia and is not promoted as semantic public state in M4.

## Task contract

Public tasks correspond to GFM task markers whose exact bracket/state mapping was already proved by M1, including nested list tasks.

`Task.Range()` is the exact one-byte state span changed by `PrepareSetTaskChecked`. The surrounding `[` and `]`, list marker, indentation, content, and line ending are outside the range.

`Task.Checked()` reports the semantic checked state. A no-op state preparation preserves the original source byte, including uppercase `X`; an actual transition to checked uses the existing M1 canonical changed byte while changing only that state span.

## Error and preservation contract

Both operations retain the public error categories established in M2:

- missing IDs report `ErrNodeNotFound`;
- wrong node kinds report `ErrInvalidTargetKind`;
- unsafe list-item replacements report `ErrInvalidReplacement`;
- prepared changes remain source-bound and stale application reports `ErrSourceConflict`.

Source-preservation tests must prove byte identity outside the exact public `Range()`.

## Architecture and complexity

M4 is a public-boundary promotion only. It reuses the existing snapshot-local node index, M1 source mappings, candidate reparsing, and `ChangeSet` implementation. It adds no parser pass, no source rescan for lookup, and no new dependency.

Generic `Node` remains a small summary. List/task syntax-specific state belongs only to typed details.

## Deferred families

M1 table-cell, fenced-code, and several inline/link operations perform important source-shape validation only during mutation preparation rather than storing a fully proved editable mapping in the parsed document model. M4 intentionally does not expose those semantic observations as public actionable nodes yet. A later milestone should first decide whether to persist/validate their editable detail at parse time or introduce another explicit public capability boundary.

## Devil's advocate review

### Risk: ordered marker semantics are misleading

The internal marker byte for an ordered item is the delimiter (`.` or `)`), not its numeric prefix. Calling it a list number would freeze incorrect semantics.

Mitigation: public `Marker()` is documented explicitly as marker/delimiter; the numeric source prefix remains unpromoted.

### Risk: nested items/tasks disappear at the public boundary

Reusing top-level filtering from paragraph/heading promotion would regress M1-proven nested list behavior.

Mitigation: list-item and task promotion does not require `TopLevel`; focused public tests exercise nested forms and unchanged indentation.

### Risk: generic `Node` grows syntax-specific fields

Adding ordered state, marker, or checked state to generic summaries would recreate the M1 union at the public boundary.

Mitigation: `Node` continues to expose only ID and kind; syntax-specific state is available through `ListItem` and `Task`.

## Exit decision

M4 is green. Focused list/task public tests pass for ordered and nested list items, nested tasks, exact operation ranges, no-op task source preservation, wrong-target/missing-target errors, and unsafe list-item replacement. The complete Go regression and race suites pass; vet, Staticcheck, golangci-lint, govulncheck, and Gitleaks report no findings; generated package documentation exposes only Marksplice-owned or standard-library public types.

The milestone also consolidates root-package lookup/target validation so M2/M3/M4 wrappers reuse one snapshot-index lookup path instead of accumulating duplicate nil/not-found/kind checks. No parser, source mapper, dependency, or M1 mutation implementation changed.

Table cells, fenced code, links, inline syntax, front matter, and HTML remain unpromoted. Their future public promotion must first resolve the capability-boundary issue recorded in `docs/architecture.md` where important editable source-shape validation is currently deferred until mutation preparation.
