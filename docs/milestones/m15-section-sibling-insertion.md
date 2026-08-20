# Milestone M15 — Section Sibling Insertion

Status: green — section sibling insertion passed.

## Goal

Add the first section insertion operations by inserting one validated standalone section subtree immediately before or after an existing section as a same-level sibling.

M15 builds directly on M14's standalone-fragment proof and source-ordered candidate validation. It deliberately avoids generic insertion positions, arbitrary heading levels, automatic whitespace repair, and multi-operation batch semantics.

## Public contract

M15 adds:

- `Document.PrepareInsertSectionBefore(NodeID, []byte) (ChangeSet, error)`;
- `Document.PrepareInsertSectionAfter(NodeID, []byte) (ChangeSet, error)`.

The `NodeID` is the governing heading identity of the anchor section.

Both operations require the inserted fragment to pass the same standalone section-fragment proof used by M14:

- fragment is non-empty;
- its first section begins at byte zero;
- its first/root section spans the entire fragment;
- its root heading level equals the anchor section's level.

Therefore the inserted root is structurally a sibling of the anchor. The fragment may contain any number of nested descendant sections and ordinary GFM body constructs.

## Before semantics

`PrepareInsertSectionBefore` inserts at:

```text
anchor.Section.Range().Start
```

The inserted root appears immediately before the anchor in section source order.

For a nested anchor this makes the fragment a sibling under the same enclosing lower-level heading. For a root anchor it inserts another root section after any existing document preamble and before the anchor heading.

No preamble byte is consumed or rewritten.

## After semantics

`PrepareInsertSectionAfter` inserts at:

```text
anchor.Section.Range().End
```

Because M9's complete section range includes the entire descendant subtree, this position is after all anchor descendants, not merely after the anchor's direct body or heading.

The inserted root therefore appears after the complete anchor subtree as a same-level sibling.

For a final section the insertion point may be EOF. If the existing final source does not end in a boundary that permits the fragment heading to remain a heading, the operation fails closed rather than synthesizing a line ending.

## Source-preservation contract

Each operation prepares one zero-width source patch:

```text
[insertAt, insertAt)
```

with the fragment as replacement bytes.

Every original byte remains byte-identical. The operation only adds caller-provided bytes at the exact structural boundary.

M15 does not:

- trim or add blank lines;
- normalize line endings;
- render Markdown;
- move existing source;
- alter the anchor section;
- infer formatting style from neighboring headings.

If the requested insertion is not structurally valid with the source exactly as it exists, it returns an error.

## Shared fragment proof

M14's internal standalone parser helper is generalized in name from a replacement-specific helper to `parseSectionFragment` because the proof now serves both subtree replacement and insertion.

No proof semantics change:

- one complete root subtree;
- no fragment preamble;
- root level fixed to the operation's structural level;
- semantic GFM parsing rather than character heuristics.

ATX and Setext fragments are both valid when their surrounding candidate context preserves the same standalone structure.

## Candidate validation

After constructing the zero-width patch, M15 parses the whole candidate document once and validates three source-ordered windows:

1. all original sections before the insertion index;
2. all inserted fragment sections;
3. all original sections from the insertion index onward.

For `before`, the insertion index is the anchor's section index.

For `after`, the insertion index is the first section outside the anchor's complete subtree, found with the existing linear `sectionSubtreeEndIndex` scan.

The candidate must contain exactly:

```text
original section count + fragment section count
```

Original headings are validated with the same `validateOriginalSectionHeadings` path established by M14. Headings before the zero-width patch remain at their original ranges; headings starting at or after the insertion point shift by exactly `len(fragment)`.

Inserted sections are validated against the standalone fragment with `validateInsertedSectionFragment`. The inserted root must occupy exactly:

```text
[insertAt, insertAt + len(fragment))
```

This ensures the inserted source remains one self-contained sibling subtree in host context.

## Boundary behavior

M15 deliberately exposes Markdown boundary reality instead of hiding it.

For example, a standalone Setext fragment can become unsafe when inserted immediately after a paragraph line because that preceding line may be absorbed into the Setext heading. Likewise, inserting a heading at EOF immediately after non-line-terminated body text can make the heading cease to be a heading.

Those candidates return `ErrInvalidReplacement`.

A pre-existing blank line or line ending that makes the same fragment structurally valid is preserved and used as-is. Marksplice never inserts a repair separator automatically.

## Error contract

M15 adds no new public error category:

- missing anchor ID: `ErrNodeNotFound`;
- wrong/non-section anchor: `ErrInvalidTargetKind`;
- invalid fragment or unsafe insertion boundary: `ErrInvalidReplacement`;
- stale application: `ErrSourceConflict`.

An empty fragment is invalid. Section deletion remains the explicit M12 operation.

## Architecture and complexity

Both public operations delegate to one internal `prepareInsertSection` implementation.

Let `n` be original document size, `k` fragment size, `h` original section count, and `r` fragment section count.

The operation performs:

- one standalone fragment parse, O(k);
- for `after`, one forward subtree-end scan, O(h) worst case; `before` uses the existing O(1)-expected section index directly;
- one candidate construction/parse, O(n + k);
- one source-ordered validation pass across original and inserted sections, O(h + r).

The zero-width patch reuses the normal source-bound `ChangeSet` engine and its stale-source fingerprint checks.

M15 adds no persistent index, renderer, parser mode, filesystem/network behavior, generic patch-batch API, or move abstraction.

## Devil's advocate review

### Risk: `after` inserts inside the anchor's descendant subtree

Using the next heading or direct-body boundary instead of complete `Section.Range().End` could place the new section before nested descendants.

Mitigation: `after` uses the complete M9 subtree end and the same `sectionSubtreeEndIndex` used by M14. Focused tests anchor on a section with a deep child and prove the inserted sibling appears after that child.

### Risk: arbitrary fragment level changes parentage

Allowing an inserted `h1` next to an `h2`, or vice versa, would not have a stable "sibling" meaning and could reparent existing source.

Mitigation: standalone fragment root level must equal the anchor level. More general structural placement requires a separately reviewed operation.

### Risk: Setext fragment absorbs preceding paragraph text

A standalone valid Setext section inserted at a line boundary immediately after paragraph text may parse together with the preceding line.

Mitigation: host candidate validation must reproduce the standalone inserted ranges exactly. The unsafe case is rejected; a fixture with an existing blank line proves the same Setext fragment succeeds when the boundary is genuinely safe.

### Risk: EOF insertion silently concatenates a heading to body text

Appending `# New` after an unterminated body line would produce ordinary text rather than a new section.

Mitigation: candidate section count/range validation rejects the insertion. M15 does not invent a newline. A source that already ends with a line ending accepts the fragment unchanged.

### Risk: before/after implementations drift

Separate validation pipelines could disagree about fragment proof or source preservation.

Mitigation: both public operations share `prepareInsertSection`, differing only in the calculated insertion point/index and operation description.

### Risk: zero-width range shifting handles a boundary inconsistently

An original heading whose range begins exactly at the insertion point must shift, while a heading whose range ends exactly there must remain unchanged.

Mitigation: `rangeAfterPatch` already uses half-open range semantics. New focused tests cover zero-width positive-delta mapping on both sides of the boundary.

## Evidence and exit decision

M15 began with focused public tests that failed to compile solely because `PrepareInsertSectionBefore` and `PrepareInsertSectionAfter` did not exist.

The first implementation correctly rejected a supposedly valid Setext fixture. Investigation showed the fixture placed the Setext title directly after paragraph text, allowing GFM to absorb the preceding line. The fixture was corrected by using a pre-existing blank line; the unsafe behavior remains covered as an explicit fail-closed boundary case.

Focused public/internal tests now pass for:

- inserting before a nested sibling;
- inserting after a complete subtree with descendants;
- preserving exact original bytes around a zero-width insertion;
- preserving parent relationships for inserted roots and children;
- inserting before the first root while preserving document preamble;
- inserting after a final root when the existing EOF boundary is safe;
- rejecting final-EOF insertion when source lacks the required line boundary;
- CRLF and Unicode source;
- separated Setext sibling insertion;
- invalid fragment, wrong-level, preamble, and multi-root rejection;
- wrong/missing target errors;
- stale-source conflict behavior;
- source-order preservation of all original and inserted sections;
- zero-width range-shift semantics.

M15 is green. The complete repository verification stack passes with M10 through M15 together: native `gofmt -d` checks, focused section-mutation regressions, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, and `git diff --check`.
