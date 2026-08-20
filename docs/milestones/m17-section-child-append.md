# Milestone M17 — Section Child Append

Status: green — section child append passed.

## Goal

Add a source-preserving operation for appending one new direct child section subtree to an existing parent section, including the important case where the parent currently has no child section that could serve as a sibling insertion anchor.

M17 reuses the section-fragment proof and zero-width candidate validation established by M14/M15. It does not introduce a generic arbitrary-depth insertion API.

## Public contract

M17 adds:

- `Document.PrepareAppendSectionChild(parentHeadingID, fragment) (ChangeSet, error)`.

The target is the governing heading identity of an existing supported parent section.

The fragment must be one complete standalone section subtree whose root level is exactly:

```text
parent.Level() + 1
```

The fragment may contain any number of deeper descendants below that root.

A level-6 section cannot receive a child and returns `ErrInvalidReplacement`.

## Why exactly one level deeper

A fragment rooted deeper than `parent+1` does not have stable direct-child semantics when existing descendants are present.

For example, appending an H3 subtree to an H1 that already ends in an H2 subtree could make the new H3 a child of that last H2 rather than a child of the requested H1.

M17 therefore does not infer missing levels or synthesize intermediate headings. Direct-child insertion means exactly one heading level deeper than the parent.

## Insertion point

The fragment is inserted at:

```text
parent.Section.Range().End
```

M9 defines that range end as the boundary after the complete existing parent subtree. Therefore the new child is appended after all current descendants and immediately before the next equal-or-higher heading, or at EOF.

This supports both:

- appending the first child of a section with only direct body content;
- appending another direct child after an existing descendant tree.

## Source-preservation contract

M17 prepares one zero-width source patch:

```text
[parent.Range().End, parent.Range().End)
```

using the caller-provided fragment bytes unchanged.

No existing byte is rewritten, moved, normalized, or rendered. Marksplice does not add blank lines or line endings to make the fragment fit.

## Fragment and candidate validation

The fragment first passes the existing `parseSectionFragment` proof:

- non-empty fragment;
- first section begins at byte zero;
- root section owns the complete fragment;
- root level is `parent+1`.

After preparing the zero-width patch, Marksplice parses the complete candidate once and validates:

1. every original heading before the insertion boundary with the existing range-shift validator;
2. every inserted fragment section/heading against its standalone source ranges at the insertion offset;
3. every original heading after the insertion boundary with the same validator;
4. the inserted root has the candidate parent heading ID of the requested parent;
5. the inserted root range is exactly the caller-provided fragment bytes.

This final parent-identity check proves direct-child semantics rather than relying only on heading levels.

## Boundary behavior

The insertion boundary must already be valid Markdown source.

Appending at EOF after a parent body that already ends with a line ending can safely introduce a heading fragment. Appending the same fragment immediately after unterminated body text can concatenate the heading marker into ordinary text and is rejected.

Setext and other context-sensitive joins follow the same fail-closed rule established by M12–M16. M17 never manufactures a separator automatically.

## Error contract

M17 adds no new public sentinel:

- missing parent ID: `ErrNodeNotFound`;
- non-section parent target: `ErrInvalidTargetKind`;
- level-6 parent, wrong fragment level, preamble/multiple-root fragment, or unsafe host boundary: `ErrInvalidReplacement`;
- stale application: `ErrSourceConflict`.

## Architecture and complexity

M17 uses only existing section mutation primitives:

- `sectionTarget`;
- `sectionSubtreeEndIndex`;
- `parseSectionFragment`;
- the normal source-bound zero-width `ChangeSet`;
- `parseSectionMutationCandidate`;
- `validateOriginalSectionHeadings`;
- `validateInsertedSectionFragment`.

Let `n` be document size, `k` fragment size, `h` original section count, and `r` fragment section count. Preparation performs O(k) fragment parsing, one O(n+k) candidate construction/parse, and O(h+r) source-ordered validation. No repeated heading searches or quadratic rescans are introduced.

As part of M17 consolidation, the growing section implementation is mechanically split by responsibility:

- `internal/splice/section_edits.go` contains the named section mutation operations;
- `internal/splice/section_validation.go` contains fragment parsing, candidate validation, range shifting, subtree/order, and move-position helpers.

The split changes no API or algorithm. It reduces local file complexity while preserving one shared validation implementation across M12–M17.

## Devil's advocate review

### Risk: appending after direct body instead of the complete subtree splits existing descendants

Using `BodyRange().End` would insert the new child before current child headings.

Mitigation: M17 inserts at `Section.Range().End`, after the complete existing descendant subtree.

### Risk: accepting a deeper heading creates the wrong parent

An H3 appended to an H1 can become a child of the final H2 rather than the requested H1.

Mitigation: the standalone fragment root must be exactly `parent.Level()+1`, and the candidate root must explicitly report the requested parent heading ID.

### Risk: level-6 parent silently creates a non-child sibling

Markdown has no H7 heading level.

Mitigation: level-6 parents are rejected before fragment preparation.

### Risk: EOF insertion concatenates source

A parent ending in non-line-terminated body text can turn `## Child` into ordinary text.

Mitigation: candidate section/range validation rejects the operation; M17 adds no newline automatically.

### Risk: another section mutation develops a parallel validator

A special child-insertion path could drift from M14/M15 insertion safety rules.

Mitigation: M17 reuses the same fragment parser, inserted-fragment validator, and original-heading validator. The M17 file split centralizes those helpers rather than duplicating them.

## Evidence and exit decision

M17 began with focused public tests that failed to compile solely because `PrepareAppendSectionChild` did not exist.

Focused public tests now pass for:

- appending the first direct child;
- appending after an existing child/deep descendant subtree;
- exact zero-width source preservation;
- CRLF and Unicode source;
- inserted descendant hierarchy;
- safe final-EOF insertion;
- fail-closed unterminated EOF insertion;
- empty, wrong-level, preamble, and multiple-root fragments;
- level-6 parent rejection;
- missing/wrong target errors;
- stale-source conflicts.

Focused M12–M17 public/internal regressions also pass after splitting validation helpers into `section_validation.go`.

M17 is green. The complete repository verification stack passes with M10 through M17 together: native `gofmt` diff checks, focused M12–M17 section regressions, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, and `git diff --check`.
