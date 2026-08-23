# M83 — Multi-Block Blockquote Construction

Status: complete.

## Objective

Extend new-document blockquote construction from one paragraph to an explicit sequence of reviewed child blocks without adding a second public block AST and without widening existing-source blockquote promotion or editing.

## Contract

M83 adds:

```go
func (b *DocumentBuilder) AppendBlockquoteBlocks(depth int, content *DocumentBuilder) error
```

`depth` is structural intent and must be between 1 and 64 inclusive. `content` is a child builder whose current body blocks are snapshotted when appended; later mutations of that child builder do not affect the parent. The child builder must be non-nil, non-empty, distinct from the destination builder, and have no front-matter envelope.

M83 initially reviews paragraph, ATX heading, thematic-break, and fenced-code children. Later milestones extend the same API to additional already-reviewed child families.

The writer derives every container marker. It renders the snapshotted child sequence through the existing canonical construction writers and then writes exactly `depth` copies of `> ` before every physical line, including canonical blank lines between child blocks. Caller-authored blockquote markers are never structural authority.

## Architecture

M83 introduces a construction-only multi-block blockquote state that owns the requested depth and a private copy of the child construction blocks. It deliberately reuses `constructionBlock` rather than exposing a new public block sum type.

The proof path has three layers:

- the ordinary child builder validates each block and the complete standalone child sequence before it may be quoted;
- `internal/source.ValidateCanonicalNestedBlockquoteBlocks` proves byte-for-byte that removing exactly the generated repeated marker prefix from every quoted physical line reconstructs the canonical LF-terminated child source;
- `internal/parser/goldmark.ValidateNestedBlockquoteBlocks` reparses the complete generated source only for construction proof, finds exactly the requested nested blockquote depth, reparses the standalone inner source, and compares the reviewed child sequence semantically.

`internal/splice.ValidateConstructionNestedBlockquoteBlocks` composes the lexical and semantic proof. Ordinary `Adapter.Parse`, M73 existing-source blockquote promotion, and M74 removal semantics remain unchanged.

Construction-only observations inside a multi-block blockquote are not ordinary construction-node expectations; the blockquote-specific proof is authoritative for that source region.

## TDD and edge cases

The initial focused test failed to compile only because `AppendBlockquoteBlocks` did not yet exist. Focused public coverage then proves:

- canonical heading + LF-multiline paragraph + thematic break + fenced code with canonical quoted blank separators;
- explicit nested depth and typed-inline child blocks;
- snapshot semantics after the child builder is subsequently mutated;
- nil destination, nil child, empty child, self-reference, invalid depths, front matter, and unsupported child families fail with `ErrInvalidConstruction` without mutating the destination;
- generated multi-block source does not widen the public existing-source `KindBlockquote` subset.

Focused source tests prove exact quoted-to-inner byte ownership and reject changed child bytes or pathological depth. Focused Goldmark tests prove exact child kind/heading-level hierarchy and reject semantic changes.

## Devil's advocate review

1. **A mutable child builder could make retained construction state nondeterministic.** The destination snapshots the child block slice before validation and retention.
2. **Quoting could alter block boundaries or fenced-code interpretation.** The generated quoted tree is compared against a separately parsed canonical inner document, while Marksplice independently proves every marker and child byte.
3. **Construction-only nodes could leak into ordinary node matching.** Multi-block expectations bypass ordinary blockquote matching; later child-family milestones generalize this to observations wholly contained by the construction-only range.
4. **Lexical proof could become a complexity hotspot.** The validator was refactored into sequence orchestration plus a focused per-line helper, restoring the production complexity limit.

## Verification

M83 established a complete strict gate before M84 work began. The verified tree passed five consecutive `go test ./... -count=1` runs, race tests, vet, build, public `go doc`, pinned published GFM 0.29 conformance, Staticcheck, golangci-lint with zero issues, production `gocyclo` at the <=15 threshold, production and test-inclusive `unparam`, `govulncheck`, Gitleaks, strict text hygiene, `git diff --check`, and `git fsck --no-dangling`.

During that gate, Staticcheck identified a deprecated Goldmark text accessor and `unparam` identified an unused variadic option on the private `newMarkdown` helper; both were removed and the affected gates rerun successfully. Harness-only PowerShell invocation errors were corrected and generated coverage artifacts were removed from the repository before continuing.

Statement coverage measured on the M83 code tree was 92.5% for the root package, 68.7% for `internal/parser/goldmark`, 79.2% for `internal/source`, 57.5% for `internal/splice`, and 71.2% aggregate. The parser interface package reported 0.0% because it has no executable test target.

## Exit decision

M83 is complete. Multi-block blockquote composition now has one compact public entrypoint and a construction-only lexical/semantic proof boundary. The next reviewed slices extend supported child families without changing that API or widening existing-source blockquote semantics.
