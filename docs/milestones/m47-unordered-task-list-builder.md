# Milestone M47 — Flat Unordered Task-List Construction

Status: green — structured unordered GFM task-list construction.

## Goal

Extend the M45/M46 list-family construction path with explicit task state instead of requiring callers to spell task markers inside raw item source.

M47 adds:

```go
type TaskListItem struct {
    InlineGFM string
    Checked   bool
}

func (b *DocumentBuilder) AppendUnorderedTaskList(items ...TaskListItem) error
```

`TaskListItem` is construction input only. It does not replace or alias the existing snapshot-bound `Task` detail and carries no `NodeID`, source range, or parsed-document identity.

## Canonical source policy

M47 writes deterministic task-list source:

```markdown
- [ ] todo
- [x] done
```

Rules:

- the unordered-list marker is `-`;
- unchecked state is `[ ]`;
- checked state is lowercase `[x]`;
- one space follows the list marker and one space follows the task marker;
- `InlineGFM` remains explicit caller-provided single-line GFM source;
- LF and block separation follow the existing `DocumentBuilder` writer policy.

The caller-provided task-item slice is converted into private builder values before it is retained, so later caller mutation cannot change builder state.

## Parser-proof architecture

M47 does not add task-specific parser metadata. It reuses two already-established proof layers.

First, each generated physical item must pass the M45/M46 list proof: exact source/content ranges, expected unordered marker, the requested private `ListContainerAnchor`, no list-item parent, no children, and complete one-line subtree ownership.

Second, the generated checkbox must reparse through the existing M4 `Task` model. The proof requires an editable `KindTask` node at exactly the generated marker start, exact three-byte task-marker range, exact one-byte state range, and the requested checked state. The task marker must begin at the generated list item's content start. Because the parser emits `Task` only for a GFM task marker attached to a list item, these exact shared byte boundaries tie the semantic task proof to the requested list item without introducing a public or persistent list/task relationship index.

The task proof is additive. Existing `AppendUnorderedList` behavior remains unchanged when caller-owned item source itself begins with task syntax such as `[ ] text`; M47 does not globally reject semantic task nodes that were not requested through `TaskListItem`.

## Failure behavior

An empty task list, empty/multiline/invalid-UTF-8/NUL-containing `InlineGFM`, a nil builder, or any generated shape that does not reparse as the requested flat task list fails with `ErrInvalidConstruction`.

As in M45, two separately requested adjacent unordered task-list blocks fail at final `Markdown()` validation if GFM merges them into one semantic list container despite the blank line between requested blocks. Rejected appends leave builder state unchanged.

## Complexity

Writing a task list of output size `k` is O(k). Final construction validation remains O(n) in generated document size. Task proof builds at most one temporary entry per promoted task node, so temporary task-index memory is O(t) for `t` tasks and is discarded before `Markdown()` returns.

## Devil's advocate review

### Text that looks like `[x]` could be accepted without being a semantic GFM task

Mitigation: source spelling is insufficient. M47 requires the existing parser/source mapper to produce an editable `KindTask` with the exact generated marker/state ranges and checked state.

### A semantic task could be proven but belong to the wrong list item

Mitigation: the expected task marker must start exactly at the corresponding generated list-item content start, while that list item separately passes exact physical-line/container proof. The parser only promotes `Task` from a task checkbox whose semantic parent is a list item.

### Task proof could accidentally narrow the older generic list API

Mitigation: validation requires task evidence only for expectations created by the structured task-list writer. A focused regression keeps caller-owned task-shaped content valid through `AppendUnorderedList`.

## Verification evidence

Public task-list tests were added first. The red run failed to compile because `TaskListItem` and `AppendUnorderedTaskList` did not yet exist.

After implementation, the complete `TestPublicDocumentBuilder` focused suite passes, including canonical checked/unchecked output, defensive-copy behavior, parsed `Task` state, invalid-input rejection, semantic list-container merge rejection, nil-builder handling, preservation of existing generic-list behavior, and separation from ordered task-list containers.

Final repository-wide verification on the M47/M48 tree passes five consecutive `go test ./... -count=1` runs, `go test -race ./... -count=1`, coverage, `go vet ./...`, `go build ./...`, generated `DocumentBuilder`/`TaskListItem` documentation, and the pinned published-GFM 0.29 conformance gate. `staticcheck ./...` and standard `golangci-lint run` are clean; production gocyclo reports no function above complexity 20, production/test-inclusive unparam report no findings, `govulncheck ./...` reports no vulnerabilities, and Gitleaks reports no leaks. Final statement coverage is 91.6% for the public root package, 64.3% for `internal/parser/goldmark`, 79.3% for `internal/source`, and 66.7% for `internal/splice`.

M47 adds no parser field, public list identity, `NodeID` input, new public `Kind`, or existing-document rendering path.
