# Milestone M9 — Section Model

Status: green — read-only section model passed.

## Goal

Add the first reviewed hierarchical document view after the M1-derived syntax-family promotions: source-bound sections governed by document headings.

M9 is intentionally read-only. It establishes exact section identity, hierarchy, and source ranges needed for bounded reading and future structural operations without prematurely defining insert/remove/move semantics or weakening the existing mutation safety model.

## Section semantics

A section is governed by one supported document heading.

Its complete subtree begins at the first byte of the heading source and ends immediately before the next supported document heading of equal or higher level, or at end of source.

Its direct body begins immediately after the governing heading's physical line ending and ends immediately before the next supported document heading of any level, or at end of source. Therefore nested subsection headings and their content are outside the direct body but remain inside the complete subtree.

A section's parent is the nearest preceding open section governed by a lower-level heading. Skipped heading levels are valid: an `h3` may have an `h1` parent when no intervening `h2` exists.

Document preamble bytes before the first supported heading are not a section in M9.

Container headings remain outside M9. The section index derives only from the same supported document-level heading nodes already promoted by M3; a heading inside a blockquote or another unsupported container does not silently become a document section.

## Public contract

M9 adds the immutable public `Section` view plus:

- `Document.Sections()` for source-ordered section enumeration;
- `Document.Section(headingID)` for constant-expected-time lookup by the governing heading's snapshot-scoped `NodeID`.

`Section` exposes:

- `HeadingID()` — the existing heading node identity governing the section;
- `Level()` — GFM heading level 1 through 6;
- `Range()` — complete section subtree, including the heading;
- `BodyRange()` — direct body after the heading line and before the next heading;
- `ParentHeadingID()` — nearest enclosing section's heading identity when one exists.

M9 deliberately does not introduce `KindSection` or manufacture a second `NodeID` namespace. A section is a derived view anchored by an existing heading node. This avoids the ambiguity of passing a section identity to `Document.Node` and having it resolve as a different structural category.

Duplicate heading text is safe because lookup uses snapshot-scoped heading IDs, never human-readable labels.

## Internal model and algorithm

`splice.Parse` builds a derived section index after structural nodes and the normal node-ID index are complete.

The builder first collects only editable document-level heading nodes in source order. It validates their level/range invariants and rejects an impossible out-of-order heading sequence rather than sorting away an internal inconsistency.

Hierarchy and subtree ends are computed with a monotonic stack of open sections:

1. for each heading in source order, pop every open section whose level is greater than or equal to the new heading level;
2. close each popped section at the new heading's source start;
3. the remaining stack top, if any, is the new section's parent;
4. push the new section;
5. sections still open at end of input naturally end at end of source.

Each section enters and leaves the stack at most once, so hierarchy/subtree indexing is O(h) for `h` supported headings. The section slice and heading-ID index are also O(h) memory.

Direct-body end is simply the next supported heading start, independent of level. Body start is derived from the already validated full heading source range and consumes exactly one LF, CRLF, or isolated CR line ending when present. Setext headings therefore begin their body after the underline line, not after the title line.

The public wrapper copies immutable section values and reuses the internal heading-ID index rather than rebuilding hierarchy on each call.

## Source preservation boundary

M9 performs no source mutation and introduces no renderer or normalization path.

All ranges are half-open byte offsets into the immutable parsed snapshot. LF, CRLF, isolated CR, Unicode, ATX/Setext spelling, blank lines, and all other source trivia remain untouched.

The complete subtree includes source bytes governed by the section up to the next equal/higher heading; the direct body excludes the governing heading line and every nested subsection.

## Devil's advocate review

### Risk: naive subtree discovery becomes O(h²)

Scanning forward from every heading to find its closing heading would become quadratic for large heading-heavy documents.

Mitigation: one monotonic stack computes all parent and subtree boundaries in O(h).

### Risk: Setext or CRLF boundaries leak heading syntax into the body

Using the semantic heading-content end would produce the wrong body start for Setext headings and could split CRLF.

Mitigation: M9 starts from the existing lossless full heading range, then consumes exactly one recognized LF/CRLF/CR line ending. Public and internal tests cover ATX, Setext, CRLF, isolated CR, and EOF.

### Risk: container headings become false document sections

A heading-looking construct inside a blockquote/list could distort document hierarchy.

Mitigation: section construction accepts only the existing editable document-level heading nodes. Focused tests verify a blockquote heading is excluded.

### Risk: duplicate heading labels create ambiguous targeting

Human-readable heading text is not unique Markdown identity.

Mitigation: sections are anchored to the heading's snapshot-scoped `NodeID`; duplicate-text headings produce distinct sections and O(1)-expected lookup entries.

### Risk: inventing section node IDs creates public identity confusion

A derived section ID that overlaps or diverges from the heading's node identity would complicate `Document.Node`, stale-snapshot semantics, and future structural operations.

Mitigation: M9 exposes `HeadingID()` explicitly and does not add `KindSection` or a second node-ID algorithm.

## Deferred work

M9 does not define:

- section insert/remove/move/replace operations;
- a preamble pseudo-section;
- container-aware sections;
- arbitrary child-node lists inside a section;
- a multi-document section graph;
- batch mutation semantics.

Those features must build on the section ranges and hierarchy established here and retain the source-preserving/stale-source rules already proven by earlier milestones.

In particular, combining independently prepared mutations is not treated as a trivial next step: two non-overlapping patches can still interact semantically in Markdown. A future batch milestone must provide joint validation rather than merely concatenating patch lists.

## Evidence and exit decision

M9 began with a focused public test that failed to compile because `Section`, `Document.Sections`, and `Document.Section` did not exist.

The completed public tests prove source-ordered enumeration, exact direct-body and subtree ranges, parent hierarchy, ATX and Setext headings, CRLF, lookup by heading ID, deterministic zero values, returned-slice isolation, and rejection of non-heading/missing IDs.

Internal tests additionally cover blockquote/container-heading exclusion, isolated CR parsing, EOF bodies, malformed source-order/range fail-closed behavior, empty documents, and duplicate heading text with distinct snapshot identities.

The consolidation review confirms the index is O(h), uses one section slice plus one heading-ID map, and does not duplicate the public node taxonomy or add source rescans during reads.

M9 is green. The complete repository verification stack passes with the section model included: `gofmt`, focused tests, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks detect` with no leaks, and the approved published-GFM conformance gate. Final whitespace/status checks are recorded with the working-tree review.
