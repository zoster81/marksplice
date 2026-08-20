# Milestone M11 — Image Public API

Status: green — image public API passed.

## Goal

Add the first reviewed image-editing capability without exposing Goldmark internals or weakening Marksplice's parse-time editable-capability boundary.

M11 promotes only simple single-line inline images whose exact destination boundary can be proven from the immutable source snapshot. Reference images, compound alt text, empty destinations, multiline shapes, and other unproven forms remain internal/non-editable.

## Public contract

M11 adds:

- `KindImage`;
- immutable typed detail `Image`;
- `Document.Image(NodeID)`;
- `Document.PrepareReplaceImageDestination(NodeID, []byte)`.

`Image.Range()` is the exact destination-content span replaced by `PrepareReplaceImageDestination`. The `!` marker, alt text, brackets, parentheses, raw-versus-angle destination wrapper, optional title syntax, spacing, surrounding source, and line endings remain outside the operation range.

The public detail intentionally exposes no alt text, title, or lexical wrapper fields. Those facts are preservation state rather than caller requirements for the reviewed operation.

## Parser and source-mapping boundary

The pinned Goldmark `ast.Image` type does not expose destination/title data through its public API. M11 therefore does not inspect or depend on Goldmark private fields.

Goldmark remains responsible for semantic recognition that a construct is an image. Through the public AST contract Marksplice uses the image `Pos()` anchor plus a single plain-text child range. Marksplice's own lossless source mapper then proves the exact inline-image source shape and destination/title boundaries.

`source.MapSimpleImage` accepts only:

- a `!` anchor immediately followed by `[`;
- one non-empty plain-text alt range directly inside the brackets;
- an immediately following inline destination `(...)` on the same physical line;
- a non-empty destination supported by the established Markdown destination scanner;
- an optional supported inline title separated from the destination.

The mapper reuses the established destination/title scanners already used by the M7 link family rather than introducing a second lexical grammar.

## Parse-time capability model

A parser image observation always remains Marksplice-owned. During `splice.Parse`, `source.MapSimpleImage` determines whether that observation is editable.

On mapper success the immutable node stores `ImageMapping`, sets `ContentRange` to the exact destination range, and marks the node `Editable=true`.

Expected unsupported shapes remain internal `KindImage` nodes with `Editable=false`; they are filtered from the public node surface and mutation preparation fails closed with the existing invalid-target category.

This preserves the M5–M8 rule that semantic recognition alone is not sufficient for public editability.

## Mutation validation

`PrepareReplaceImageDestination` first applies the established non-empty single-line replacement precondition and prepares exactly one destination patch.

The candidate snapshot is then reparsed through the normal GFM parser and remapped as a simple image. Validation requires:

- the image semantic observation to remain at the same source anchor;
- the complete image range to change only by the destination-length delta;
- the alt range to remain identical;
- the destination range to begin at the same byte and have exactly the replacement length;
- raw-versus-angle destination form to remain unchanged;
- title presence to remain unchanged;
- when a title exists, its source range to shift only by the destination-length delta.

If those facts cannot be re-established, the operation returns `ErrInvalidReplacement`.

## Error and preservation contract

M11 adds no new public error category:

- missing ID: `ErrNodeNotFound`;
- wrong or non-editable target: `ErrInvalidTargetKind`;
- unsafe replacement: `ErrInvalidReplacement`;
- stale application: `ErrSourceConflict`.

Public tests verify that bytes outside `Image.Range()` are byte-identical after a valid destination replacement, including CRLF, angle destinations, spacing, titles, and balanced raw-destination parentheses.

## Architecture and complexity

M11 adds no parser dependency, filesystem/network behavior, renderer, image loader, generic media abstraction, or reference-graph behavior.

The parser observation is O(1) beyond the existing AST walk. Source mapping scans only the containing physical inline-image tail and therefore remains linear in that bounded source span. Candidate validation retains the established O(n) reparse per prepared mutation.

No generic public `LinkLike` or `Media` type is introduced. Links and images share internal lexical scanners where their GFM source grammar overlaps while retaining distinct public semantics and mutation names.

## Devil's advocate review

### Risk: coupling to Goldmark private image fields

The pinned `ast.Image` hides destination/title details. Reflection, unsafe access, or dependency-source assumptions would make Marksplice brittle across Goldmark upgrades.

Mitigation: use only public AST node behavior for semantic recognition/anchor/child boundaries, then perform Marksplice-owned source mapping.

### Risk: treating every semantic image as safely editable

Reference images and compound alt text are valid Markdown but do not satisfy the reviewed inline-image mutation proof.

Mitigation: parser recognition and editability remain separate. `MapSimpleImage` must succeed before public promotion.

### Risk: destination replacement changes image structure

A raw replacement containing whitespace, unmatched delimiters, or other syntax can turn the intended destination bytes into a title or terminate the image early.

Mitigation: candidate GFM reparse plus lossless remapping must re-establish the original image structural facts before returning a `ChangeSet`.

### Risk: duplicated link grammar diverges over time

Copying destination/title lexical rules specifically for images could drift from the already reviewed link behavior.

Mitigation: image mapping reuses `scanMarkdownLinkDestination` and `scanMarkdownLinkTitle`; only the image-specific `![alt]` prefix proof is new.

## Evidence and exit decision

M11 began with focused public tests that failed to compile because `KindImage`, `Image`, `Document.Image`, and `PrepareReplaceImageDestination` did not exist.

Focused tests now pass at four boundaries:

- Goldmark adapter: simple-image public AST anchor/alt boundaries and compound-alt filtering;
- source mapper: exact raw/angle destination/title ranges and unsupported-shape rejection;
- splice: parse-time editable mapping persistence, unsupported reference/empty-destination filtering, and source-preserving mutation;
- public API: exact typed range, valid raw/angle replacements, CRLF preservation, unsupported-shape filtering, stable error categories, and deterministic zero value.

M11 is green. The complete repository verification stack passes with M10 and M11 together: `gofmt`, focused tests, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks` with no leaks, the approved published-GFM conformance gate, and `git diff --check`.
