# M98 — Anchors, fragments, and TOC

## Status

Complete. Focused TDD, maintainability refactor, full implementation-tree release-quality verification, corrected explicit cross-package coverage, and security/conformance checks are green. No commit or push is authorized by this milestone record.

## Objective

Add native single-document navigation primitives derived from the existing immutable heading, section, HTML-anchor, and source-mutation models:

- GitHub-compatible heading-anchor derivation;
- deterministic duplicate-anchor disambiguation;
- fragment resolution and validation;
- deterministic table-of-contents generation;
- stale-TOC detection;
- source-preserving synchronization of an explicitly designated managed TOC body.

M98 remains deliberately narrower than M99. It resolves one fragment inside one immutable snapshot, but does not enumerate link relationships, build backlinks, validate all links, or create a persistent relationship graph.

## Public contract

M98 adds:

```go
type HeadingAnchor struct { /* immutable */ }
func (a HeadingAnchor) HeadingID() NodeID
func (a HeadingAnchor) Value() string

func (d *Document) HeadingAnchors() []HeadingAnchor
func (d *Document) HeadingAnchor(id NodeID) (HeadingAnchor, bool)

type FragmentTargetKind uint8
const (
    FragmentTargetUnknown FragmentTargetKind = iota
    FragmentTargetHeading
    FragmentTargetHTMLAnchor
)

type FragmentTarget struct { /* immutable */ }
func (t FragmentTarget) Kind() FragmentTargetKind
func (t FragmentTarget) NodeID() NodeID
func (t FragmentTarget) Value() string

func (d *Document) ResolveFragment(fragment string) (FragmentTarget, bool)
func (d *Document) ValidateFragment(fragment string) bool
func (d *Document) GenerateTOC() []byte
func (d *Document) TOCStale(headingID NodeID) (stale bool, recognized bool)
func (d *Document) PrepareSyncTOC(headingID NodeID) (ChangeSet, error)
```

All values are snapshot-local. No new durable identity namespace is introduced.

## Heading semantic text and anchor derivation

The Goldmark adapter now emits one parser-independent `HeadingText` scalar for promoted document headings. Goldmark AST types remain confined to `internal/parser/goldmark`; the immutable splice model stores only the resulting semantic string. This observation is intentionally part of the parser contract that M111–M115 can reproduce when Goldmark is replaced.

Semantic heading text keeps rendered text content rather than raw Markdown delimiters:

- link labels contribute their label text, not their destination;
- emphasis/strong/strikethrough delimiters do not contribute punctuation;
- ordinary escaped punctuation and HTML/numeric entities are semantically resolved;
- code-span text remains literal code content;
- parser/source ownership ranges remain unchanged.

Anchor derivation follows the reviewed GitHub behavior:

1. trim surrounding Unicode whitespace;
2. lowercase letters with Go Unicode case mapping;
3. convert ASCII spaces to `-`;
4. preserve letters, numbers, combining marks, `-`, and `_`;
5. remove other punctuation, whitespace, and symbols;
6. assign the first free source-ordered duplicate suffix `-1`, `-2`, and so on.

The duplicate allocator reserves every already-produced slug, not merely the unsuffixed base. Therefore the sequence `Same`, `Same`, `Same-1`, `Same` deterministically yields `same`, `same-1`, `same-1-1`, `same-2`.

Explicit HTML anchors do not participate in heading duplicate numbering and are not generated as TOC entries.

## Fragment resolution

`ResolveFragment` accepts a fragment with or without one leading `#`. It URI-percent-decodes the fragment value and rejects:

- empty fragments;
- malformed percent escapes;
- an additional literal `#` after the optional leading delimiter.

Resolution scans two existing snapshot-owned target families:

- derived heading anchors;
- promoted source-proven simple HTML `<a id>` / `<a name>` anchor values.

Matching is exact after fragment decoding. Resolution succeeds only when exactly one target matches across both families. Missing targets and ambiguous heading/explicit-anchor collisions fail closed. `ValidateFragment` is the boolean form of the same resolver and therefore cannot drift to a second interpretation.

M98 does not scan links or reference usages to discover who points at a fragment. That broader relationship surface remains M99.

## TOC generation

`GenerateTOC` derives one deterministic Markdown unordered list from the existing source-ordered `Section` model:

- every promoted heading contributes exactly one entry;
- entry nesting follows the existing section parent hierarchy, not raw heading-level arithmetic;
- each structural depth uses two spaces of indentation;
- labels use semantic heading text and escape `\\`, `[` and `]` for the generated Markdown label;
- destinations use the M98 derived heading anchors;
- generated standalone output uses canonical LF line endings.

This creates no second section tree or AST. The existing `Section` array and `sectionIndex` remain authoritative.

## Managed TOC recognition and stale detection

M98 deliberately does not infer arbitrary prose/list bodies as safe rewrite authority. Callers must identify the section whose direct body is intended to hold the TOC, and that exact `Section.BodyRange()` must pass a conservative lexical ownership check.

A recognized managed body is either empty/blank-only or consists of:

- optional leading blank physical lines;
- a contiguous local-fragment list in `- [label](#fragment)` form;
- two-space indentation per nesting depth;
- first entry at depth zero;
- no depth jump greater than one;
- one consistent physical line-ending form;
- optional trailing blank physical lines.

Ordinary lists, external links, odd indentation, internal blank separators, mixed line endings, malformed fragments, and ambiguous shapes are rejected. Leading and trailing blank bytes are retained as body trivia during synchronization.

`TOCStale` returns `(false, false)` for a missing/non-managed target. For a recognized body, it compares exact body bytes with the current generated TOC rendered using that body's established line-ending form plus preserved leading/trailing blank trivia.

## Source-preserving synchronization

`PrepareSyncTOC` is not a raw patch API. It requires:

1. one promoted top-level heading/section target;
2. exact `BodyRange` ownership from the existing section model;
3. successful managed-TOC lexical recognition;
4. deterministic regenerated TOC content using the target body's line ending and preserved edge blank trivia;
5. the existing `PrepareReplaceSectionBody` candidate parse and section-hierarchy proof.

Non-managed bodies report `ErrInvalidTargetKind`. Stale-source rejection and patch application remain the ordinary `ChangeSet` contract. Bytes outside the exact section body are not regenerated.

## Architecture and resource model

M98 stores no anchor map, fragment index, TOC cache, or relationship graph in `Document`.

For `H` promoted headings and `N` promoted/internal source-ordered nodes:

- `HeadingAnchors`: O(H) expected time, O(H) result plus temporary duplicate map;
- `HeadingAnchor(id)`: O(H) through ephemeral derivation;
- `ResolveFragment`: O(H + N) expected time and O(H) temporary anchor state;
- `GenerateTOC`: O(H) expected time and O(H) generated output/depth state;
- managed-body recognition: O(B) for body byte length `B`;
- `TOCStale`: O(H + B) plus generated output;
- `PrepareSyncTOC`: M98 derivation/recognition cost plus the already-established section-body candidate proof.

M108 owns measurement-driven optimization. A persistent anchor/fragment index is intentionally not added before benchmark evidence demonstrates a need.

## Requirements and edge cases

Focused coverage includes:

- plain and formatted heading text;
- link labels and code-span text;
- HTML/numeric entity semantics;
- Unicode lowercasing and non-Latin text;
- emoji/punctuation removal;
- preserved underscore/hyphen behavior;
- repeated spaces and non-space whitespace;
- duplicate slugs whose suffixed spellings are themselves authored headings;
- URI-percent-decoded fragments;
- malformed/empty/missing fragments;
- heading versus explicit HTML-anchor ambiguity;
- TOC hierarchy with skipped numeric heading levels;
- escaped bracket/backslash labels;
- CRLF synchronization;
- preserved leading/trailing blank body trivia;
- managed-body rejection for ordinary/external/malformed/mixed-EOL list shapes;
- fail-closed synchronization of arbitrary prose bodies;
- nil-document read behavior.

## TDD evidence

1. `tsk_c2b9909661cdb12eba5b1637f8eebb17` established the public RED: the focused package failed to compile only because the M98 public types/methods did not yet exist.
2. `tsk_264f7bc4292219c826412d4e92c87914` produced the first public GREEN after implementing the semantic-heading, anchor, fragment, TOC, and managed-sync paths.
3. `tsk_7d9e1229ee85059ae471e7432bc63319` passed the first complete `go test ./... -count=1` regression on that implementation.
4. `tsk_3d4e2bd5966b36a283ac73e6059f3d40` passed the new white-box tests, but its combined `-run` expression selected zero public M98 tests. It is intentionally non-authoritative for the public package and records the same harness hazard addressed by the exact GFM gate policy.
5. `tsk_58bc2b8c74914e9273830ccfce59ec65` corrected the filter to `TestM98.*`; verbose output proves both white-box and every public M98 test actually executed and passed.
6. `tsk_0a36dfb2dd2c45fb320b19eb027063b7` passed the corrected focused suite again after fragment/trivia hardening, including the new internal-blank/raw-second-`#` negative cases.
7. `tsk_7fbc95abc837656c6e9d6a775cd4c12e` passed after the blank-only managed-body case was tightened so existing blank prefix trivia remains before the synchronized TOC.
8. The first full maintainability gate correctly measured `tocBodyProfile` at cyclomatic complexity 19. The recognizer was then split into `tocBodyScan`, line-ending validation, depth validation, and profile materialization without changing behavior. `tsk_2459c51545c86f05850dc577f1a9552e` passed the focused public/white-box regressions and the production `gocyclo <= 15` gate after that refactor.

## Maintainability review

The final recognizer keeps orchestration separate from state-machine details: `tocBodyProfile` owns range validation and the bounded physical-line loop; `tocBodyScan` owns per-line state; dedicated helpers own line-ending consistency, nesting-depth admissibility, and immutable profile construction. This lowers complexity without introducing a second parser or making the TOC grammar more permissive.

The refactor does not alter public APIs, generated TOC bytes, fragment semantics, source ownership, candidate proof, or persistent snapshot state.

## Devil's advocate review

1. **Raw heading source could produce incorrect anchors for links, emphasis, code, entities, and escapes.** Mitigation: derive parser-independent semantic heading text in the adapter while Goldmark owns semantic interpretation; source ranges and mutation authority remain separate. Focused entity/code/structured-label regressions cover the boundary.
2. **A TOC convenience method could overwrite arbitrary caller prose or lists.** Mitigation: synchronization requires an explicitly designated section plus a conservative local-fragment-list ownership grammar, then reuses the existing section-body candidate proof. Unsupported shapes fail closed.
3. **Duplicate allocation could collide with an authored suffixed heading.** Mitigation: every emitted slug is reserved in one ephemeral map and candidate suffixes probe for the first unused value.
4. **Fragment resolution could silently choose one target when a heading and explicit anchor collide.** Mitigation: count all supported matches and return success only for exactly one target.
5. **TOC synchronization could normalize unrelated line endings or surrounding whitespace.** Mitigation: the target body establishes one exact EOL form; leading/trailing blank bytes are retained, and the ordinary section-body mutation leaves all bytes outside `BodyRange` untouched.
6. **Anchor convenience could become an unmeasured persistent index that complicates immutable-snapshot invariants.** Mitigation: anchors, duplicate maps, TOC depths, and fragment matching are recomputed from authoritative source-ordered arrays. M108 owns any benchmark-driven indexing decision.
7. **Goldmark-specific semantic-text extraction could obstruct M115.** Mitigation: only a plain `HeadingText string` crosses the adapter boundary; no Goldmark type/API is public or stored outside the parser adapter.

## Release-quality verification

The final refactored implementation tree passed the substantive M98 freeze:

- `tsk_353e8e3bd67ab932803d0c0acdfa509d`: five consecutive `go test ./... -count=1` runs plus full race detection;
- `tsk_1a6096aa56d168a8fbbe1009289519f8`: gofmt, `go vet`, `go build`, the executable `GenerateTOC` example, M98 `go doc` checks, Staticcheck, golangci-lint with zero issues, production `gocyclo <= 15`, production and test-inclusive `unparam`, `go mod tidy -diff`, and `git diff --check`;
- `tsk_3f5ae18c841629231f0ea3037394f6c3`: direct Go 1.27.0 test, vet, and build;
- `tsk_e744b718b3e1d23faaba358658cb4abd`: the exact anchored `^TestGFM029PublishedSpecificationConformance$` gate; verbose execution proves the real pinned GFM test ran and passed;
- `tsk_a5a0e4bcee262962d96758937218d921`, `tsk_f6023804e830fa7ec82f2139f1e46e3a`, and `tsk_0f9598a2174011442029600f0d8bafb0`: standalone govulncheck, Gitleaks, and actionlint all passed on the final refactored tree;
- `tsk_e3390a0dd3a2769b2535c666bed7638b`: corrected explicit cross-package statement coverage passed at **86.7%** over the production package set and **83.8%** through `internal/publictest` alone; temporary profiles were kept outside the repository and removed after measurement.

Several earlier invocations are intentionally non-authoritative harness evidence. One quality run exposed Staticcheck S1017 and was fixed with behavior-equivalent `strings.TrimPrefix`; the next quality run exposed the real complexity-19 recognizer and triggered the refactor above. A combined security task completed its commands with exit code 0 but hit an output-drain timeout, so the three security tools were rerun separately. Early coverage attempts either measured package-local/duplicated profiles at a meaningless 50.3% or misquoted PowerShell native arguments; only the explicit quoted production-package run above counts as M98 coverage evidence.

`go.mod` and `go.sum` remain unchanged. The first post-documentation verification (`tsk_79afe0a7345222ee79c3953a418f082f`) passed the full regression, vet/build, exact GFM gate, diff/module checks, and `git fsck`, but correctly exposed two literal dollar-named coverage profiles left by discarded PowerShell harness attempts. Both files were inspected and confirmed as coverage profiles before targeted removal. The clean final hygiene gate `tsk_d90d7899f1566831bb0138b66436f969` then passed artifact-absence checks, one complete regression, `git diff --check`, unchanged `go.mod`/`go.sum`, `git fsck --no-dangling`, and branch/HEAD/origin verification on the final source-of-truth tree.

## Exit decision

M98 is complete. Anchors, exact single-document fragment resolution, deterministic TOC generation, stale detection, and fail-closed source-preserving synchronization remain an immutable-snapshot navigation layer with no persistent navigation index or second document model. M99 — link intelligence — is the next milestone and retains responsibility for broader link/reference relationship enumeration and validation.
