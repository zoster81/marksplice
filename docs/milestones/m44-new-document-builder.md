# Milestone M44 — New-Document Builder Foundation

Status: green — first structured new-document construction slice. M54 later broadens the same `AppendParagraph` method to parser-proven LF-multiline paragraphs without changing this historical M44 contract.

## Goal

Open Marksplice's second core path: create new GitHub Flavored Markdown from structured caller intent without weakening the source-preserving snapshot/edit path.

M44 deliberately keeps the two models separate:

- `Document` remains an immutable parsed source snapshot with snapshot-scoped identities, exact source mappings, and stale-source mutation safety;
- `DocumentBuilder` is mutable construction state for source that does not exist yet and therefore has no `NodeID`, source fingerprint, or authoritative source ranges.

Existing documents are never converted through `DocumentBuilder` as the ordinary edit path.

## Public contract

M44 adds:

```go
var ErrInvalidConstruction error

type DocumentBuilder struct { /* private */ }

func NewDocumentBuilder() *DocumentBuilder
func (b *DocumentBuilder) AppendHeading(level int, inlineGFM string) error
func (b *DocumentBuilder) AppendParagraph(inlineGFM string) error
func (b *DocumentBuilder) Markdown() ([]byte, error)
```

The zero value of `DocumentBuilder` is ready to use. Its empty `Markdown` result is an empty document. A nil builder reports `ErrInvalidConstruction`.

Returned Markdown bytes are caller-owned; mutating them cannot change builder state.

## M44 construction subset

M44 intentionally supports only:

- top-level ATX headings at levels 1 through 6;
- top-level single-line paragraphs.

`inlineGFM` is caller-provided inline GFM source, not implicitly escaped plain text. This allows already-valid inline syntax such as emphasis and links while avoiding a premature public inline construction AST.

For this first slice, inline input must:

- be non-empty;
- be valid UTF-8;
- contain no CR or LF;
- contain no NUL byte.

A paragraph input that would parse as another block kind, such as a list item, thematic break, or fenced block, fails closed. A heading whose source would reinterpret part of the requested inline bytes as an ATX closing sequence also fails because the exact heading content range is no longer the requested range.

Rejected appends do not mutate the builder.

## Canonical source policy

Because new documents have no author source to preserve, M44 uses one deterministic canonical spelling:

- headings are ATX;
- line endings are LF;
- adjacent blocks are separated by exactly one blank line;
- every non-empty document ends with one LF.

M44 does not yet expose a style policy. Future construction milestones may add reviewed formatting choices without changing the rule that existing-source edits preserve existing bytes.

## Validation model

Construction does not trust source generation merely because the writer emitted syntactically plausible bytes.

Each appended block is written independently and reparsed through the existing `internal/splice` GFM model before it is stored. This provides early fail-closed validation without reparsing the whole accumulated document after every append.

`Markdown` then writes the complete block sequence and reparses the complete result once. Validation requires, in source order:

- the requested top-level block kind;
- an editable mapping in the existing snapshot model;
- exact expected content/source ranges;
- exact heading level;
- ATX heading style.

The final parser proof is the convergence point between constructed Markdown and the existing Marksplice document model. Callers that need snapshot navigation or mutation can pass the returned bytes to `Parse` and receive ordinary snapshot-scoped IDs.

## Complexity

Let `b` be the number of constructed blocks and `n` the total output byte size.

Each append performs O(k) write/parse validation for the new block of size `k`; across all appends this is O(n) work for block-local validation. `Markdown` performs O(n) generation plus one O(n) parse/model validation. Builder storage and returned source are O(n).

The design avoids reparsing the complete growing document on every append, which would otherwise make a long construction sequence quadratic.

## Devil's advocate review

### Risk: caller inline source changes the requested block type

For example, `- item` requested as a paragraph would silently become a list.

Mitigation: every block and the final complete document are reparsed and must reproduce the requested block kind and exact mapping.

### Risk: heading source is syntactically valid but changes semantic content

ATX closing markers can remove trailing source from the semantic heading content.

Mitigation: the parsed heading `ContentRange` must equal exactly the generated range assigned to caller `inlineGFM`.

### Risk: construction state becomes confused with snapshot state

Giving drafts `NodeID` values or source ranges before source exists would create false identity/staleness semantics.

Mitigation: `DocumentBuilder` stores only private construction blocks. Snapshot IDs, fingerprints, ranges, and `ChangeSet` remain properties of `Document`/`Parse`.

### Risk: a future writer replaces source-preserving editing

A complete writer can be tempting to reuse for existing documents.

Mitigation: architecture explicitly forbids using new-document canonical generation as the ordinary existing-document edit path. Existing edits remain minimal source patches.

### Risk: a builder grows into a second complete Markdown AST prematurely

Mitigation: M44 models only block-level heading/paragraph intent and deliberately keeps inline content as explicit GFM source. Typed inline/list/table construction is promoted only in later reviewed milestones.

## Verification evidence

The public M44 tests were authored before the implementation. The intended compile-red run could not be executed because the shared local task runner remained queued until after implementation; that queued task was cancelled to avoid misrepresenting a post-implementation run as red evidence.

After implementation, focused `TestPublicDocumentBuilder*` tests pass and the complete `go test ./... -count=1` repository suite passes. The final post-refactor verification uses the private Go 1.26.6 toolchain and passes five consecutive complete suites, race detection, coverage, vet, build, generated package/`DocumentBuilder` documentation, the pinned published-GFM 0.29 conformance gate, Staticcheck, standard golangci-lint with zero issues, production gocyclo/unparam, test-inclusive unparam, govulncheck with no vulnerabilities, Gitleaks with no leaks, UTF-8/LF/no-trailing-whitespace checks across all changed text files, `git diff --check`, and `git fsck --no-dangling`.

Final statement coverage is 90.8% for the public root package, 64.1% for `internal/parser/goldmark`, 79.3% for `internal/source`, and 66.7% for `internal/splice`. The post-verification construction-validator split removes M44 code from the production complexity analyzer's top 25 while keeping sequence scanning separate from one-block proof.

The focused tests cover canonical output, inline emphasis/link source, Unicode, generated-source reparsing, level/style/range semantics, invalid levels, empty/multiline/CR/NUL/invalid-UTF-8 input on both heading and paragraph construction, structural paragraph ambiguity, ATX closing-marker ambiguity, zero-value behavior, nil receivers, failed-append immutability, and caller-owned output bytes.

M44 does not change any existing public `Kind` ordinal, `NodeID` derivation, parser profile, source-preservation rule, or mutation contract.
