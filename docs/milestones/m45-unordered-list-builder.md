# Milestone M45 — Flat Unordered-List Construction

Status: green — parser-proven flat unordered-list construction.

## Goal

Extend the new-document `DocumentBuilder` with one semantic GFM container while keeping construction separate from existing-source editing.

M45 adds:

```go
func (b *DocumentBuilder) AppendUnorderedList(items ...string) error
```

The operation constructs exactly one flat top-level unordered list. The canonical marker is `-`. Each item is explicit caller-provided single-line inline GFM and is defensively copied into builder state.

## Container-membership proof

A blank line does not necessarily create a second GFM list. For example:

```markdown
- one

- two
```

is still one semantic list container under the selected GFM parser. Therefore construction cannot prove list boundaries from lexical spacing alone.

M45 adds a private parser-independent `ListContainerAnchor` fact to list-item observations and internal splice nodes. The Goldmark adapter derives it through public AST relationships from the first physical list-item line belonging to the containing `ast.List`.

The anchor:

- is private implementation metadata;
- is a byte offset into the parsed snapshot;
- is not a public `List` identity;
- is not included in `NodeID` derivation;
- does not change public `Kind` ordinals;
- remains distinct from `ListParentAnchor`, which identifies an immediate parent list item rather than a list container.

Adapter tests prove root, nested, deep, blank-line-continued, and paragraph-separated list-container cases.

## Construction validation

Each appended unordered-list block is written and reparsed before it is retained. The complete `Markdown()` output is reparsed again.

For every generated item, proof requires:

- `KindListItem` with an editable source mapping;
- exact content and complete physical-line ranges;
- unordered state;
- `-` marker;
- the expected shared `ListContainerAnchor`;
- no list-item parent;
- no semantic children;
- direct child count zero;
- a complete one-item subtree ending at the physical line end.

Consequently, two consecutive `AppendUnorderedList` calls fail at final `Markdown()` if GFM merges their generated source into one semantic list container. Marksplice does not silently reinterpret two requested structural blocks as one.

## Failure behavior

Empty lists, empty items, invalid UTF-8, CR/LF, NUL, and item content that changes the requested flat-list structure fail with `ErrInvalidConstruction`. A rejected append does not mutate builder state. Nil receivers fail deterministically.

## Complexity

For total generated list source size `k`, list writing and block-local parser proof are O(k). Final document writing and validation remain O(n) in total generated source size.

## Devil's advocate review

### Lexical spacing may not equal semantic container boundaries

Mitigation: validate private semantic container membership from the parser rather than guessing from blank lines.

### New parser metadata could alter established list editing

Mitigation: the anchor is additive private metadata, does not participate in IDs or public typed details, and the complete M18–M33 regression suite remains green.

### Item content may create nesting or another structure

Mitigation: exact list-item range, parent, child-count, subtree-completeness, marker, and container proof fails closed.

## Verification evidence

The adapter and public M45 tests were written first and their red state was executed: the adapter test failed because `ListContainerAnchor` did not exist, and the public test failed because `AppendUnorderedList` did not exist.

After implementation, the focused adapter and public builder tests pass, followed by the complete repository regression suite.

M45 does not add a public `List`, change any public `Kind` ordinal, change `NodeID` derivation, or weaken existing source-preserving mutation semantics.
