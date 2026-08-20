# Milestone M12 — Section Removal

Status: green — section removal passed.

## Goal

Add the first source-preserving structural mutation built directly on the M9 section model: removal of one complete section subtree by the governing heading's snapshot-scoped identity.

M12 deliberately starts with removal rather than generic section replacement, insertion, movement, or batch composition. The exact deletion boundary already exists as `Section.Range()`, so the milestone can prove structural mutation safety without inventing new whitespace ownership or patch-order semantics.

## Public contract

M12 adds:

- `Document.PrepareRemoveSection(NodeID) (ChangeSet, error)`.

The argument is the existing governing heading `NodeID`; no section-specific identity namespace is introduced.

The operation removes exactly the complete `Section.Range()` established by M9. Therefore the removed bytes include:

- the governing heading source;
- the direct body;
- every nested subsection in the section subtree;
- source trivia already contained in the M9 subtree range up to, but not including, the next heading of equal or higher level or EOF.

The operation does not independently rescan for headings or decide which blank lines "belong" to a section. M9 remains the sole source of truth for the structural deletion range.

Preamble bytes before a removed root section remain untouched because preamble is outside M9 section ranges.

## Mutation and validation model

`PrepareRemoveSection` first requires an editable top-level heading that is present in the immutable section index. A missing node returns `ErrNodeNotFound`; a node that is not a removable section target returns `ErrInvalidTargetKind`.

The implementation prepares one empty-replacement patch over the stored section range and renders one candidate snapshot. The candidate is parsed exactly once through the normal Marksplice document pipeline.

M12 then validates all surviving supported document headings in source order. For every heading outside the removed subtree it requires:

- the same heading level;
- the same ATX/Setext source style;
- the same complete mapped heading range, shifted only when it originally followed the removed bytes;
- the same mapped content range, shifted by the same deletion delta when applicable;
- byte-identical complete heading source.

The candidate must also contain exactly the expected number of surviving sections. Because M9 hierarchy is a deterministic function of source-ordered heading levels, preserving the complete surviving heading sequence and levels preserves the surviving section hierarchy as well.

All headings whose section starts inside the removed section range are expected to disappear; this naturally removes nested descendants together with their governing section.

## Why candidate validation is necessary

Deleting byte-identical source can still alter Markdown semantics at the new join boundary.

The important M12 regression is a following Setext heading. For example, removing an intervening ATX section can cause a preceding paragraph line to become part of the surviving Setext heading. The bytes outside the deletion range would remain unchanged, but the requested structural operation would have changed a surviving heading boundary.

M12 therefore fails closed when surviving mapped heading facts cannot be re-established. No separator or blank line is synthesized automatically because doing so would mutate bytes outside `Section.Range()` and introduce a new formatting policy.

## Error contract

M12 adds no new public sentinel:

- missing ID: `ErrNodeNotFound`;
- wrong or non-section target: `ErrInvalidTargetKind`;
- a deletion whose join cannot preserve required surviving structure: `ErrInvalidReplacement`;
- applying the prepared change to another snapshot: `ErrSourceConflict`.

`ErrInvalidReplacement` predates M12. Its public documentation is broadened from replacement-byte-specific wording to the existing underlying meaning: a requested mutation that cannot preserve the required structure. The sentinel value and `errors.Is` behavior remain unchanged.

## Architecture and complexity

The implementation lives in `internal/splice/section_edits.go`, separate from syntax-family edit code and from generic mutation plumbing.

M12 reuses:

- the M9 immutable section slice and heading-ID index;
- the existing O(1)-expected node-ID index;
- the existing source-bound `ChangeSet` patch engine;
- the normal Marksplice `Parse` pipeline as the conservative candidate safety oracle.

After the candidate parse, expected-survivor validation scans the original section slice once. Candidate section lookup is sequential and heading lookup uses the candidate node index. Validation is O(h) for `h` original supported headings, in addition to the existing O(n) candidate parse. It does not scan forward from each section and introduces no O(h²) behavior.

No parser configuration, renderer, filesystem/network capability, second section index, or generic mutation framework is added.

## Devil's advocate review

### Risk: deletion changes a surviving Setext heading boundary

A raw byte deletion can join a preceding paragraph directly to the title line of a following Setext heading, causing Goldmark to reinterpret the heading while every untouched byte remains identical.

Mitigation: candidate parsing must reproduce every surviving heading's level, style, mapped range, mapped content range, and complete source bytes at the expected shifted position. The unsafe Setext case is a focused regression and returns `ErrInvalidReplacement`.

### Risk: removal heuristics consume sibling or parent whitespace

Trying to rediscover section boundaries during mutation could diverge from M9 or create subjective blank-line ownership rules.

Mitigation: M12 deletes exactly the stored `Section.Range()` and performs no separate boundary scan. The M9 section contract remains authoritative.

### Risk: descendant sections survive accidentally

Removing only a heading and direct body would orphan nested subsections or change hierarchy unexpectedly.

Mitigation: the complete M9 subtree range is removed. Every section whose heading starts inside that range is expected to disappear from the candidate index.

### Risk: validation becomes quadratic on heading-heavy documents

Comparing every survivor against all candidate headings would make structural mutation expensive.

Mitigation: original and candidate sections are both source ordered. Validation walks them once and uses existing node-ID indexes for heading detail lookup, so the post-parse comparison is O(h).

### Risk: automatic whitespace repair hides semantic changes

Inserting a blank line when the candidate becomes unsafe could make deletion appear successful while modifying bytes outside the section range and silently establishing a formatter policy.

Mitigation: M12 performs no repair. Unsafe joins fail closed so a future explicitly reviewed operation can define broader ownership if needed.

## Evidence and exit decision

M12 began with public tests that failed to compile solely because `Document.PrepareRemoveSection` did not exist.

Focused tests now pass for:

- nested section removal including a deep descendant;
- exact byte preservation outside `Section.Range()`;
- CRLF and Unicode source;
- root-section removal while preserving preamble and following roots;
- final-section removal through EOF;
- isolated-CR boundaries;
- stale-source conflict behavior;
- missing/wrong target errors;
- fail-closed Setext join behavior;
- internal rejection of an editable heading that is not present in the derived section index;
- deterministic range-shift helper behavior.

M12 is green. The complete repository verification stack passes with M10–M12 together: `gofmt`, focused tests, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, and `git diff --check`.
