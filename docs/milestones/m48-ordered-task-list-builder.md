# Milestone M48 — Flat Ordered Task-List Construction

Status: green — structured ordered GFM task-list construction.

## Goal

Apply the M47 structured task-item and semantic task proof to the ordered-list source policy already established by M46, without introducing a second task writer or validator.

M48 adds:

```go
func (b *DocumentBuilder) AppendOrderedTaskList(items ...TaskListItem) error
```

The operation constructs exactly one flat top-level ordered list whose every item has explicit checked state.

## Canonical source policy

M48 writes deterministic source:

```markdown
1. [x] first
2. [ ] second
```

Rules:

- numbering begins at `1` and increments sequentially;
- the ordered-list delimiter is `.`;
- checked state is lowercase `[x]` and unchecked state is `[ ]`;
- one space follows the list delimiter and one space follows the task marker;
- task content follows the same single-line valid-UTF-8/NUL-free `InlineGFM` contract as M47;
- the surrounding construction writer owns LF and block separation.

Custom start numbers, `)` delimiters, uppercase `[X]`, and task-marker style controls remain intentionally outside the construction API. Existing parsed source keeps its author spelling under the ordinary source-preserving edit path.

## Shared implementation and proof

M47 and M48 use the same private `constructionListItem`, `validateConstructionListItems`, `writeConstructionList`, and task expectation proof. Ordered/unordered behavior is only a source-policy flag passed to the common list writer.

Each generated ordered task item must therefore satisfy both proof families:

1. the M46 list-item proof requires ordered state, `.` marker, exact physical/content ranges, the expected private `ListContainerAnchor`, flatness, and complete one-line subtree ownership;
2. the M47 task proof requires the exact semantic `KindTask` marker/state mapping and requested checked state at the list-item content start.

No `Task` snapshot identity is fabricated during construction. Successful output receives ordinary snapshot-scoped identities only if the caller later passes it to `Parse`.

## Failure behavior

M48 uses the same `ErrInvalidConstruction` behavior as the preceding construction milestones. Empty lists, malformed task content, nil builders, parser/mapping disagreement, or semantic list-container merging fail closed without changing retained builder state.

Separately requested adjacent ordered task lists remain distinct construction blocks; if GFM merges them semantically, final `Markdown()` proof rejects the result rather than treating one blank line as a list-container boundary.

## Complexity

M48 does not change the complexity established by M47. Writing is O(k) in emitted source size, complete validation is O(n) in generated document size, and temporary task proof storage is O(t).

## Devil's advocate review

### Ordered task construction could drift from ordinary ordered-list numbering policy

Mitigation: both M46 and M48 use the same list writer and the same sequential decimal marker path; only the task prefix inside each item differs.

### Reusing the unordered task proof could miss ordered-list semantics

Mitigation: task proof is only the second layer. Every item must first pass the existing ordered list-item proof, including `ListOrdered=true`, `.` marker, exact source range, and shared semantic container anchor.

### Adding ordered tasks could duplicate public task state concepts

Mitigation: `TaskListItem` remains construction input, while the existing `Task` remains immutable parsed-document detail. They have different lifecycles and no shared identity semantics.

## Verification evidence

The M48 public test was introduced in the same TDD red slice as M47 and initially failed to compile because `AppendOrderedTaskList` and `TaskListItem` did not exist.

After the shared implementation, the complete `TestPublicDocumentBuilder` suite passes, including canonical sequential numbering, parsed task/list semantics, ordered/unordered task-container separation, and fail-closed rejection when separately requested adjacent ordered task lists merge semantically.

Final repository-wide verification on the M47/M48 tree passes five consecutive `go test ./... -count=1` runs, `go test -race ./... -count=1`, coverage, `go vet ./...`, `go build ./...`, generated `DocumentBuilder`/`TaskListItem` documentation, and the pinned published-GFM 0.29 conformance gate. `staticcheck ./...` and standard `golangci-lint run` are clean; production gocyclo reports no function above complexity 20, production/test-inclusive unparam report no findings, `govulncheck ./...` reports no vulnerabilities, and Gitleaks reports no leaks. Final statement coverage is 91.6% for the public root package, 64.3% for `internal/parser/goldmark`, 79.3% for `internal/source`, and 66.7% for `internal/splice`.

M48 adds no parser metadata, public list identity, new `Kind`, `NodeID` derivation, dependency, or existing-document rendering behavior.
