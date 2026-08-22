# Milestone M59 — Nested Unordered Task-List Construction

Status: green — nested unordered task-list construction reusing M57 hierarchy and M47 task proof.

## Goal

Combine the M57 structural-depth list model with the existing exact GFM task-marker/state proof without changing the historical flat `TaskListItem` type.

M59 adds:

```go
type TaskListItemInput struct {
    InlineGFM string
    Checked   bool
    Depth     int
}

func (b *DocumentBuilder) AppendNestedUnorderedTaskList(items ...TaskListItemInput) error
```

A separate input type avoids adding a field to `TaskListItem`, which could break existing unkeyed composite literals.

## Construction policy

`Depth` follows the M57 contract. Each item is written with canonical unordered `- ` syntax plus `[ ]` or lowercase `[x]` according to `Checked`.

Example:

```text
- [ ] parent
  - [x] child
  - [ ] sibling
- [x] tail
```

Caller inputs are copied into private construction state before the builder retains them.

## Shared proof

M59 introduces no new parser/splice kind and no separate task-list writer. The generalized list writer already supports task markers on each item. Final validation therefore requires both:

- M57 exact list marker/container/parent/child/subtree proof;
- M47 exact `KindTask` marker range, state range, and checked state proof.

A task item is accepted only if both structures agree over the same generated list item.

## Compatibility and complexity

Flat `AppendUnorderedTaskList` remains unchanged and still uses `TaskListItem`. Nested construction uses `TaskListItemInput` only. Runtime and memory complexity are the same as M57 plus O(1) task metadata per item expectation.

## Devil's advocate review

### Risk: nested list semantics are correct but task state changes

Mitigation: every requested task item has an exact task-marker expectation keyed by source range; checked/unchecked state must match the parser observation.

### Risk: changing `TaskListItem` breaks existing users

Mitigation: M59 adds a new construction-only `TaskListItemInput` instead of widening the existing public struct.

### Risk: failed input partially mutates the builder

Mitigation: depth/inline validation and complete standalone parser proof occur before the block is appended to retained builder state; invalid tests verify the builder remains empty.

## TDD and verification evidence

The M59/M60 red run failed to compile only because `TaskListItemInput` and the two nested task-list methods did not yet exist. The focused M59 test proves canonical unordered source, caller-input copying, nested `ParentID`/`ChildIDs`, and mixed checked states. Invalid tests cover empty input, invalid depth transitions, invalid multiline inline source, builder immutability after rejection, and nil receivers.

Focused root tests and the complete M57–M60 regression are green.

Final combined M57–M60 verification passes five consecutive `go test ./... -count=1` runs, `go test -race ./... -count=1`, coverage, `go vet ./...`, `go build ./...`, generated `DocumentBuilder`/`ListItemInput`/`TaskListItemInput` documentation, the pinned published-GFM 0.29 conformance gate, `staticcheck ./...`, standard `golangci-lint run` with zero issues, production gocyclo with no function above complexity 20 across 33 production files, production and test-inclusive unparam, `govulncheck ./...` with no vulnerabilities, and Gitleaks with no leaks. Final statement coverage is 93.2% for the public root package, 65.2% for `internal/parser/goldmark`, 79.3% for `internal/source`, and 66.7% for `internal/splice`.

The first all-in-one verification harness stopped only after those gates because its PowerShell path collector accidentally treated the modified-file list as one path. A corrected hygiene-only follow-up then passed strict UTF-8/no-BOM/LF/no-trailing-whitespace checks across all 54 changed or untracked paths, `git diff --check`, `git fsck --no-dangling`, and final repository-state inspection. The branch remains `main` at `5c016772b7583693b1f73770448fa22ec52832d5`, with no configured remote; no commit or push was performed.
