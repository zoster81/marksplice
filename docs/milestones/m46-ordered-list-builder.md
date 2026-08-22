# Milestone M46 — Flat Ordered-List Construction

Status: green — canonical flat ordered-list construction.

## Goal

Extend the M45 list-family construction proof to ordered lists without introducing a second list writer or validator.

M46 adds:

```go
func (b *DocumentBuilder) AppendOrderedList(items ...string) error
```

The operation constructs exactly one flat top-level ordered list.

## Canonical source policy

M46 uses deterministic new-document spelling:

```markdown
1. first
2. second
3. third
```

Rules:

- numbering begins at `1`;
- each following item increments by one;
- the delimiter is `.`;
- one space follows the delimiter;
- LF is used by the surrounding construction writer.

M46 deliberately does not expose custom start numbers, `)` delimiters, or source-style controls yet. Existing parsed ordered-list source continues to preserve its original numbering and delimiter during source-preserving edits.

Caller item slices are defensively copied before they enter builder state.

## Shared list-family architecture

Unordered and ordered construction share:

- `validateConstructionListItems`;
- `writeConstructionList`;
- `constructionExpectation`;
- private `ListContainerAnchor` membership proof;
- final List-item structural validation.

Only the canonical source policy differs: unordered emits `-`, while ordered emits sequential decimal numbers with `.`.

For each ordered item, the final parser proof requires ordered state, `.` marker, exact ranges, the expected shared container anchor, no list-item parent, no children, and complete one-line subtree ownership.

Two adjacent `AppendOrderedList` calls therefore fail if GFM would merge them into one semantic list. An unordered list followed by an ordered list remains valid when the parser proves two distinct list containers.

## Failure behavior

Empty lists and the same invalid or structurally unsafe single-line item inputs rejected by M45 fail with `ErrInvalidConstruction`. Rejected appends leave builder state unchanged.

## Complexity

Writing an ordered list of total source size `k` is O(k). Decimal marker conversion is bounded by each item's ordinal width and is included in the output-size bound. Final construction validation remains O(n) in generated document size.

## Devil's advocate review

### Ordered and unordered construction could drift into duplicate implementations

Mitigation: both use one list writer, one item validator, and one structural proof path.

### Sequential source numbers might not match semantic ordered-list state

Mitigation: generated items must reparse with `ListOrdered=true` and marker `.` at the expected ranges and container anchor.

### Caller mutation after append could alter builder state

Mitigation: the variadic item slice is copied, and tests mutate the caller slice after append before `Markdown()`.

## Verification evidence

The public ordered-list tests were written first and the red state was executed; compilation failed only because `AppendOrderedList` did not yet exist.

After implementation, the focused ordered-list tests pass. Combined M45/M46 focused tests and the complete repository regression suite also pass.

M46 adds no public list-container identity, changes no existing mutation contract, and reuses the private M45 membership proof.
