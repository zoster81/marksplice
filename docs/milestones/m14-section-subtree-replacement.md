# Milestone M14 — Section Subtree Replacement

Status: green — section subtree replacement passed.

## Goal

Add source-preserving replacement of one complete section subtree while allowing the replacement to redefine the governing heading source and all descendants.

M14 complements M12 and M13:

- M12 removes `Section.Range()`;
- M13 replaces only `Section.BodyRange()` and preserves all headings;
- M14 replaces the complete `Section.Range()` with another single section subtree.

The operation is deliberately stricter than arbitrary raw-range replacement. The replacement must prove that it is one self-contained section subtree at the same structural heading level as the target.

## Public contract

M14 adds:

- `Document.PrepareReplaceSection(NodeID, []byte) (ChangeSet, error)`.

The `NodeID` is the existing governing heading identity of the target section.

The replacement bytes must be non-empty and, when parsed as a standalone Marksplice document, must satisfy all of the following:

- the first section starts at byte zero, so there is no fragment preamble;
- the first section has the same heading level as the target section;
- the first section's complete `Range()` is exactly `[0,len(replacement))`, so the fragment contains one complete root subtree rather than multiple same/higher-level sibling roots.

Within that root subtree the replacement may freely change:

- heading text;
- ATX versus Setext style;
- direct body source;
- descendant section count, levels, text, and style;
- ordinary GFM body constructs.

The root heading level itself remains fixed because changing that level could reparent source outside the replaced range.

## Source-preservation contract

The prepared change contains one patch over exactly the target's stored `Section.Range()`.

No blank-line ownership is rediscovered and no separator is synthesized. Bytes before and after the target subtree remain byte-identical.

The replacement bytes are inserted exactly as supplied. Marksplice parses them for structural validation but does not render or normalize them.

## Standalone-fragment proof

Before touching the host document, M14 parses the replacement as its own immutable document.

The standalone proof prevents several ambiguous inputs:

- an empty replacement, which belongs to M12 removal instead;
- text before the replacement root heading;
- a root at a different structural level;
- two sibling roots at the target level;
- a lower-level-number heading that closes the intended root before end of fragment.

Nested lower-priority headings are allowed and become descendants of the replacement root.

Heading-looking content inside blockquotes, fenced code, or other non-section contexts is accepted because the proof uses GFM semantics rather than byte-pattern filters.

## Candidate-document proof

Standalone validity is not sufficient because the fragment may interact with the bytes immediately before or after the target range.

M14 therefore applies the candidate patch in memory and parses the whole candidate document once. Validation then checks three source-ordered regions:

1. original sections before the target subtree;
2. the inserted standalone fragment sections;
3. original sections after the removed target subtree.

Original external headings must retain their expected shifted source ranges, content ranges, levels, ATX/Setext styles, and byte-identical heading source.

Every inserted fragment section must retain, in the host candidate, the same level, complete section range, direct-body range, heading style, heading source range, and heading content range as in the standalone fragment, shifted only by the target patch start.

Finally, the candidate replacement root must occupy exactly:

```text
[target.Range().Start, target.Range().Start + len(replacement))
```

This proves that host context did not enlarge, truncate, or reinterpret the intended replacement subtree.

## Shared section-mutation validation

M14 extends the M12/M13 safety implementation without creating a second structural validator.

`internal/splice/section_edits.go` now separates reusable responsibilities:

- `parseSectionMutationCandidate` performs the one whole-candidate parse;
- `validateOriginalSectionHeadings` validates a source-ordered window of untouched original headings;
- `validateInsertedSectionFragment` validates a source-ordered inserted subtree against its standalone parse;
- `rangeAfterPatch` maps original ranges across one patch delta;
- `sectionSubtreeEndIndex` finds the first original section outside the replaced subtree with one forward scan.

M12 and M13 continue to use the same original-heading validation path.

## Error contract

M14 adds no new public errors:

- missing ID: `ErrNodeNotFound`;
- wrong/non-section target: `ErrInvalidTargetKind`;
- invalid fragment or unsafe host interaction: `ErrInvalidReplacement`;
- stale application: `ErrSourceConflict`.

An empty replacement intentionally returns `ErrInvalidReplacement`; callers wanting deletion use `PrepareRemoveSection`.

## Architecture and complexity

Let `n` be document size, `k` replacement size, `h` original section count, and `r` replacement-fragment section count.

M14 performs:

- one standalone replacement parse, O(k);
- one forward scan over the replaced original subtree, O(h) worst case;
- one candidate construction/parse, O(n + k);
- source-ordered validation of outside headings and inserted fragment headings, O(h + r).

There are no nested section searches, no whole-document rendering, no second persistent section index, and no batch planner.

The standalone parse is intentional: it establishes fragment ownership independently from host-context effects. A future batch/fragment abstraction may amortize such work only if it preserves the same fail-closed proof.

## Devil's advocate review

### Risk: replacement contains multiple sibling roots

A fragment such as two `##` sections could silently replace one subtree with two siblings and change the surrounding hierarchy.

Mitigation: the standalone first section must span the entire replacement. A same/higher-level sibling closes that range early and is rejected.

### Risk: changing the replacement root level reparents outside source

Replacing an `h2` subtree with an `h1` or `h3` root could alter parent/sibling relationships beyond the patch.

Mitigation: the standalone replacement root level must equal the target section level.

### Risk: valid standalone fragment becomes different Markdown in host context

A fragment without a terminating line ending can merge with a following Setext heading or otherwise alter boundary semantics.

Mitigation: the entire candidate is reparsed. Inserted section ranges must match standalone ranges at the expected offset, and all external headings must survive with exact shifted mapping/source facts.

### Risk: validation becomes quadratic when descendants change substantially

Searching candidate sections independently for every original/inserted heading would scale poorly on large heading-heavy documents.

Mitigation: original and candidate sections are source ordered. M14 computes the original subtree end once, then validates before/inserted/after windows by sequential indices. Post-parse validation remains O(h + r).

### Risk: M14 duplicates M12/M13 safety code

A new independent subtree validator could drift from existing deletion/body-replacement behavior.

Mitigation: M14 factors the existing heading comparison into a windowed helper reused by M12/M13 and adds only the standalone-fragment comparison required for inserted structural source.

## Evidence and exit decision

M14 began with focused public tests that failed to compile solely because `Document.PrepareReplaceSection` did not exist.

Focused public/internal tests now pass for:

- exact replacement of one nested subtree;
- byte preservation outside `Section.Range()`;
- ATX-to-Setext root style change;
- Unicode and CRLF source;
- changing descendant count and depth;
- preserving sibling/ancestor/root sections;
- replacing a root section while preserving preamble;
- replacing a final section through EOF;
- container/fenced heading-looking content inside the replacement;
- rejection of empty fragments, fragment preambles, wrong root levels, and multiple sibling roots;
- rejection of unsafe joins with a following Setext heading;
- stale-source and wrong-target error behavior;
- deterministic original-subtree end-index discovery.

M14 is green. The complete repository verification stack passes with M10 through M14 together: native `gofmt -d` checks, focused section-mutation regressions, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, and `git diff --check`.
