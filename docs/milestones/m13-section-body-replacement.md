# Milestone M13 — Section Body Replacement

Status: green — section body replacement passed.

## Goal

Add the second structural section mutation: source-preserving replacement of a section's direct body while keeping its governing heading and nested subsection hierarchy intact.

M13 builds directly on the distinction established by M9:

- `Section.BodyRange()` is the direct body only;
- `Section.Range()` is the complete subtree.

The operation therefore edits content owned directly by a section without implicitly replacing or removing child sections.

## Public contract

M13 adds:

- `Document.PrepareReplaceSectionBody(NodeID, []byte) (ChangeSet, error)`.

The `NodeID` is the existing governing heading identity. The replacement bytes may be empty or multiline and are inserted exactly in place of the stored `Section.BodyRange()`.

A valid replacement must remain the complete direct body of the same section after candidate parsing. It must not introduce, remove, rename, restyle, or otherwise change supported document-level headings.

This restriction is structural rather than textual. Heading-looking content is allowed when GFM semantics keep it outside the document section hierarchy, for example inside a blockquote or fenced code block. A real new top-level heading is rejected.

## Source-preservation contract

The prepared change contains one patch over exactly the original `BodyRange`.

Bytes before and after that range remain byte-identical. In particular, the operation preserves:

- the governing ATX or Setext heading source;
- nested subsection headings and their complete source;
- following sibling/ancestor headings;
- unrelated line endings and formatting outside the direct body;
- preamble and other sections.

The replacement itself is caller-provided source and is not normalized or rendered through an AST.

## Shared M12/M13 safety oracle

M13 consolidates the M12 section-mutation validator rather than adding a parallel implementation.

`internal/splice/section_edits.go` now provides one shared heading-patch validation path. Given a patch range, replacement length, and the ordered set of sections expected to survive, it:

1. parses the candidate document once;
2. requires the expected number of sections;
3. compares surviving headings in source order;
4. requires identical heading level and ATX/Setext style;
5. maps each original complete/content heading range through the patch delta;
6. requires the candidate ranges to equal those expected ranges;
7. requires byte-identical complete heading source.

M12 supplies only sections outside the removed subtree as survivors. M13 supplies the complete existing section slice because body replacement must not add or remove document sections.

The generalized `rangeAfterPatch` helper replaces the removal-specific shift helper and handles negative, zero, or positive patch-length deltas while rejecting ranges that overlap the changed span.

## Body-specific validation

Preserving all headings is necessary but not sufficient for body replacement.

After shared heading validation succeeds, M13 locates the target section at the same source-order index in the candidate section list and requires:

```text
candidate BodyRange == [original BodyRange.Start, original BodyRange.Start + len(replacement))
```

This proves that the entire replacement remained direct body source. A replacement that becomes a new subsection or merges into a following Setext heading therefore fails closed.

## Accepted and rejected examples

Accepted replacement shapes include:

- an empty direct body;
- paragraphs and Unicode text;
- lists, tables, links, and other ordinary GFM body source;
- multiline content;
- blockquotes containing headings;
- fenced code containing heading-looking lines;
- an EOF body without a trailing line ending.

Rejected shapes include:

- a replacement that creates a new document-level ATX or Setext heading;
- a replacement whose final bytes merge into and change a following heading boundary;
- any candidate that loses or changes an existing supported heading;
- a wrong, missing, or non-section target.

M13 does not attempt to rewrite a replacement to make it safe and does not synthesize separators automatically.

## Error contract

M13 uses the existing public errors:

- missing ID: `ErrNodeNotFound`;
- wrong or non-section target: `ErrInvalidTargetKind`;
- replacement that cannot preserve the required section structure: `ErrInvalidReplacement`;
- stale application: `ErrSourceConflict`.

Empty replacement bytes are valid when they produce a structurally valid empty direct body.

## Architecture and complexity

The public wrapper remains in the cohesive named-mutations surface while implementation stays in `internal/splice/section_edits.go`.

No new parser pass is added beyond the one candidate `Parse` already required by the section safety oracle. After parsing, heading comparison is O(h) for `h` supported document sections. Target section lookup in the original document remains O(1)-expected through the M9 `sectionIndex`, and candidate lookup by source-order index is O(1).

The candidate allocation and parse remain O(n + k) in the size of the original document and replacement candidate, consistent with the existing single-mutation safety model. M13 adds no batch planner, renderer, filesystem/network behavior, second section index, or generic tree-edit abstraction.

## Devil's advocate review

### Risk: a replacement silently creates nested document sections

Allowing arbitrary Markdown without validation could turn part of the requested direct body into a new subsection, changing hierarchy while the source patch itself remains local.

Mitigation: candidate section count must remain unchanged and every original heading must survive with identical structural facts. The target candidate `BodyRange` must equal exactly the inserted replacement range.

### Risk: an over-restrictive textual filter rejects valid Markdown

Rejecting every replacement containing `#` or Setext-like bytes would incorrectly ban headings inside blockquotes, code fences, inline code, or ordinary text.

Mitigation: M13 relies on GFM semantic parsing and the section index, not character heuristics. Focused internal evidence covers blockquote and fenced heading-looking content.

### Risk: replacement merges into a following Setext heading

A replacement without a terminating line ending can become part of a following Setext title, changing untouched heading source semantics.

Mitigation: the shared heading validator requires exact mapped heading ranges/content and body-specific validation requires the replacement to remain the complete direct body.

### Risk: M12 and M13 validators drift

Duplicating heading-shift and candidate comparison logic would create two structural safety implementations with different edge behavior.

Mitigation: M13 refactors M12 onto one `validateSectionHeadingPatch` path and one `rangeAfterPatch` helper. Focused regressions run both mutation families together after the refactor.

### Risk: section validation becomes quadratic

Searching the candidate section list from the beginning for every original heading would be O(h²).

Mitigation: original/surviving and candidate sections are source ordered and compared by matching sequential indices. Node detail lookup uses existing snapshot indexes.

## Evidence and exit decision

M13 began with focused public tests that failed to compile solely because `Document.PrepareReplaceSectionBody` did not exist.

Focused public and internal tests now pass for:

- multiline CRLF and Unicode direct-body replacement;
- exact byte preservation outside `BodyRange`;
- nested subsection preservation;
- empty direct body;
- EOF body without trailing line ending;
- new document-level heading rejection;
- following Setext merge rejection;
- blockquote/fenced heading-looking content remaining valid body source;
- missing/wrong targets and stale-source conflicts;
- shared negative and positive range-delta behavior;
- M12 section-removal regressions after the shared-validator refactor.

M13 is green. The complete repository verification stack passes with M10 through M13 together: `gofmt`, focused public/internal section-mutation regressions, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, and `git diff --check`.
