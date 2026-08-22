# Milestone M57 — Nested Unordered-List Construction

Status: green — canonical homogeneous nested unordered-list construction with exact hierarchy proof.

## Goal

Extend new-document construction from M45 flat unordered lists to supported nested list-item trees without exposing indentation as caller-owned syntax and without introducing a second list model.

M57 adds:

```go
type ListItemInput struct {
    InlineGFM string
    Depth     int
}

func (b *DocumentBuilder) AppendNestedUnorderedList(items ...ListItemInput) error
```

`ListItemInput` is construction-only input and has no snapshot identity/range semantics.

## Structural input contract

Items are supplied in source/preorder order. `Depth` is structural depth, not a number of spaces:

- the first item must have depth 0;
- depth must never be negative;
- one item may descend at most one level from its predecessor;
- returning to any earlier depth is allowed;
- every `InlineGFM` value must satisfy the existing single-line list-item construction rules.

This flat structural representation avoids recursive Go input graphs, cyclic slice aliasing, recursion-depth hazards, and caller-visible indentation arithmetic.

## Canonical writer

M57 writes `- ` markers. A child line is indented to the exact generated content column of its parent item. The writer maintains transient per-depth list-container frames and active parent indexes; it does not build or retain a second persistent tree.

For example:

```text
- root
  - child one
    - grandchild
  - child two
- tail
```

is derived from depths `0,1,2,1,0`.

## Parser/model proof

M57 generalizes the existing construction list proof. Every generated item must reparse with exact:

- marker-start and content ranges;
- complete physical line range;
- unordered `-` marker semantics;
- semantic list-container anchor;
- parent-present state and physical parent anchor;
- direct semantic child count;
- complete supported subtree status and subtree end.

The generated parsed model must therefore reproduce the same `ParentID`, source-ordered `ChildIDs`, and complete subtree boundaries already established by M22–M33.

## Complexity

Validation is O(i) for `i` input items. Writing is O(n) in generated output bytes and uses O(d) transient state for maximum structural depth `d`; parser/model proof remains O(n). Deep nesting necessarily emits increasing indentation, so output size itself may grow faster than item count, but no extra asymptotic pass over ancestors is performed.

## Devil's advocate review

### Risk: caller depth encodes an impossible hierarchy

Mitigation: negative depth, nonzero first depth, and jumps greater than one level fail before generation and do not mutate the builder.

### Risk: fixed indentation guesses break list semantics

Mitigation: indentation is derived from the actual generated parent content column rather than a hard-coded number of spaces. M58 additionally proves this across changing ordered-marker width.

### Risk: nested proof trusts lexical indentation only

Mitigation: final acceptance requires semantic parent/container/direct-child/subtree metadata from the existing list model, not just source spelling.

### Risk: recursive public input can be cyclic or overflow the Go stack

Mitigation: M57 intentionally uses source-ordered `ListItemInput` plus numeric structural depth instead of recursive child slices.

## TDD and verification evidence

The red run failed to compile because `ListItemInput` and `AppendNestedUnorderedList` did not exist. The green test constructs a three-level tree, mutates the original caller slice after append to prove input copying, reparses the exact canonical source, validates `ParentID`/`ChildIDs`, and confirms the root `SubtreeRange` owns all descendants and not the following root sibling.

Invalid tests cover empty input, negative depth, nonzero initial depth, depth jumps, invalid multiline inline source, failed-append immutability, and nil receivers. Focused root tests and the full repository regression pass.

Final combined M57–M60 verification passes five consecutive `go test ./... -count=1` runs, `go test -race ./... -count=1`, coverage, `go vet ./...`, `go build ./...`, generated `DocumentBuilder`/`ListItemInput`/`TaskListItemInput` documentation, the pinned published-GFM 0.29 conformance gate, `staticcheck ./...`, standard `golangci-lint run` with zero issues, production gocyclo with no function above complexity 20 across 33 production files, production and test-inclusive unparam, `govulncheck ./...` with no vulnerabilities, and Gitleaks with no leaks. Final statement coverage is 93.2% for the public root package, 65.2% for `internal/parser/goldmark`, 79.3% for `internal/source`, and 66.7% for `internal/splice`.

The first all-in-one verification harness stopped only after those gates because its PowerShell path collector accidentally treated the modified-file list as one path. A corrected hygiene-only follow-up then passed strict UTF-8/no-BOM/LF/no-trailing-whitespace checks across all 54 changed or untracked paths, `git diff --check`, `git fsck --no-dangling`, and final repository-state inspection. The branch remains `main` at `5c016772b7583693b1f73770448fa22ec52832d5`, with no configured remote; no commit or push was performed.
