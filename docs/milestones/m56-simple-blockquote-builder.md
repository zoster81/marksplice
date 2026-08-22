# Milestone M56 — Simple Blockquote Construction

Status: green — canonical single-line paragraph blockquote construction with parser-private source ownership proof.

## Goal

Add a conservative first blockquote constructor without pretending that the parsed public API already owns general container editing semantics.

M56 adds:

```go
func (b *DocumentBuilder) AppendBlockquote(inlineGFM string) error
```

The method constructs exactly one top-level blockquote containing one single-line paragraph.

## Supported shape and canonical source

`inlineGFM` must satisfy the existing non-empty single-line valid-UTF-8/NUL-free construction rule. The writer emits:

```text
> inlineGFM
```

with the shared final LF and inter-block separation.

Input such as a heading, list item, thematic break, multiline paragraph, or another block form after the `> ` prefix is rejected. M56 does not yet construct nested/multiblock blockquotes or expose container style options.

## Parser-private source ownership

M56 appends `KindBlockquote` only to the internal parser and splice taxonomies. It does not add a public parsed `Kind`, typed blockquote detail, or mutation.

Goldmark's public `Blockquote.Pos()` identifies the outer source-line start. For the reviewed simple shape, the blockquote must have exactly one `Paragraph` child with exactly one public line segment. The parser-independent observation stores:

- the complete blockquote source range from `>` through the final content byte;
- an exact `BlockquoteContentRange` for the paragraph bytes after the container prefix;
- top-level status.

When the observation enters `internal/splice`, the generic `Node.Range` owns the complete `> …` source while the existing generic `Node.ContentRange` owns the child paragraph bytes. A final refactor deliberately avoids retaining a second blockquote-specific range field in the splice node.

The node remains non-editable and is used only by construction proof.

## Construction proof

Both the block-local append check and final `Markdown()` pass require one top-level internal `KindBlockquote` whose:

- complete `Range` exactly equals the generated `> ` prefix plus requested content;
- `ContentRange` exactly equals caller-provided `inlineGFM`;
- editable flag remains false.

Nested inline observations are ignored by the block-level construction proof, while structural child kinds prevent the reviewed blockquote observation from being created at all.

## Compatibility and complexity

The new internal kinds are appended, preserving all previous internal ordinals and `NodeID` derivation. Public parsed taxonomy and mutation contracts remain unchanged.

Observation and writing are O(n) in the single source line; construction reparsing retains the established linear validation boundary.

## Devil's advocate review

### Risk: blockquote source looks simple but contains another block kind

Mitigation: the adapter observes M56 blockquotes only when there is exactly one paragraph child with exactly one line segment. `> # heading`, `> ---`, and `> - item` therefore fail closed.

### Risk: source ownership is inferred from container punctuation heuristics

Mitigation: the outer start comes from public Goldmark `Blockquote.Pos()` and the content range comes from the public paragraph child segment. Marksplice validates containment before assigning the generic splice `ContentRange`.

### Risk: construction-only blockquotes accidentally become editable/public

Mitigation: the internal node stays `Editable=false`, no public kind/detail is added, and the construction validator explicitly handles this private semantic proof family.

### Risk: family-specific proof state widens the central splice node unnecessarily

Mitigation: the post-green refactor consumes parser-specific `BlockquoteContentRange` while mapping and retains only the ordinary splice `Range`/`ContentRange` pair.

## TDD and verification evidence

The red run failed to compile on the intentionally missing parser kind, parser content metadata, and `AppendBlockquote` API. A bounded public Goldmark inspection then confirmed `Blockquote.Pos()` at the physical `>` byte and a one-line paragraph child segment starting after `> `.

Focused adapter and public construction tests pass for exact Unicode/inline-GFM source, canonical output, invalid structural child inputs, failed-append immutability, and nil receivers. The post-refactor focused tests and complete `go test ./... -count=1` repository suite pass. Production gocyclo reports no function above complexity 20.

Final combined M54–M56 verification passes five consecutive `go test ./... -count=1` runs, `go test -race ./... -count=1`, `go vet ./...`, `go build ./...`, generated `DocumentBuilder` documentation, the pinned published-GFM 0.29 conformance gate, `staticcheck ./...`, standard `golangci-lint run` with zero issues, production gocyclo with no function above complexity 20 across 33 production files, production/test-inclusive unparam, `govulncheck ./...` with no vulnerabilities, Gitleaks with no leaks, changed/untracked UTF-8/LF/no-trailing-whitespace hygiene across 50 paths, `git diff --check`, and repository-state checks. Final statement coverage is 92.8% for the public root package, 65.2% for `internal/parser/goldmark`, 79.3% for `internal/source`, and 66.7% for `internal/splice`.
