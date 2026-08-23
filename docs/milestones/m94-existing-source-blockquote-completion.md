# M94 — Existing-Source Blockquote Completion

## Status

Complete. Focused TDD/regression, complexity-refactor verification, and the full documented-tree release-quality gate are green. The final post-evidence tree is reverified below.

## Objective

Complete existing-source blockquote promotion without deriving source authority from construction support. Promote exactly one top-level public blockquote identity across multiline, nested, lazy-continuation, marker-only, and multi-block forms only when Marksplice can prove the complete physical container. Keep nested internal blockquotes semantic-only, preserve the historical simple `ContentRange` contract, and extend source-preserving removal to the complete owned container.

## Public contract

M94 keeps `Blockquote` comparable and preserves its existing accessors while broadening the source it can represent.

- `Blockquote.Range()` now returns the complete source-proven top-level blockquote container, including each owned line terminator when present.
- `Blockquote.ContentRange()` retains the historical single-line/single-paragraph inner-range contract. Broader forms return the zero `Range`; Marksplice does not synthesize one continuous content span across intervening `>` markers.
- `Document.BlockquoteContentRanges(id)` returns caller-owned per-physical-line inner ranges in source order. Marker-only lines are valid empty ranges. Nested markers remain part of the outer container's inner source. A lazy continuation line without an outer marker contributes its complete physical content.
- only the top-level blockquote container is publicly promoted. Nested blockquote descendants do not receive separate public identities.
- `PrepareRemoveBlockquote` removes the complete owned top-level container and remains source-bound/fail-closed.

Existing simple one-line blockquote IDs and `ContentRange` semantics remain unchanged. New-document construction APIs and their accepted/rejected input shapes remain unchanged.

## Requirements and edge cases

The source layer, not Goldmark container positions alone, owns the physical blockquote boundary. Zero through three leading ASCII spaces before an outer `>` remain source data. One optional ASCII space immediately after that outer marker is excluded from the per-line inner range; additional nested markers or other syntax remain inner source.

Every marker-bearing physical line is lexically source-proven. A physical line without an outer `>` is admitted only when a Goldmark descendant block line segment proves semantic content on that exact physical line. This is the lazy-continuation safety boundary; an arbitrary following markerless line cannot be absorbed merely because it is adjacent.

A marker-only `>` or `> ` line may belong to the container even though Goldmark exposes no child segment for that empty line. Such ownership is lexical and remains bounded by the surrounding parser-proven top-level blockquote.

An unmarked blank line terminates the owned container. A later `>` block therefore remains a separate top-level blockquote. LF, CRLF, Unicode, no-marker-space source, and up to three leading spaces are preserved byte-for-byte.

Construction support is not parsed-source authorization. For example, an existing `> - item` can be safely represented as one complete source-owned blockquote container, while `AppendBlockquote("- item")` remains invalid because that builder API promises one blockquote paragraph rather than an arbitrary quoted block.

## Architecture and test strategy

M94 adds one parser-independent semantic evidence vector and one source-owned physical mapping.

`internal/parser.Node` gains private `BlockquoteSemanticRanges`. For a top-level Goldmark `Blockquote`, the adapter walks only block descendants and records valid non-empty line segments. Inline nodes are deliberately excluded because Goldmark panics when `Lines()` is requested from inline nodes and inline evidence is unnecessary for physical-line ownership.

The parser observation's ordinary `Range` remains the first physical blockquote line. This keeps the historical simple observation/ID input stable rather than redefining snapshot identity around a new complete-container span. `BlockquoteContentRange` remains populated only for the old single-paragraph/single-line shape.

`internal/source.MapTopLevelBlockquote` proves the first outer marker, normalizes the parser semantic ranges, then scans subsequent physical lines. Marker-bearing lines are accepted lexically; markerless lines require semantic proof on that exact line. `BlockquoteMapping.LineRange` owns the complete physical container, `ContentRanges` owns the per-line inner segments, and `ContentRange` remains only the legacy single-segment value before `internal/splice` additionally confirms that the parser's historical content range matches it.

`internal/splice` promotes the mapping only for top-level observations, exposes a defensive-copy content-range accessor, and stores no second public hierarchy. Complete-container removal continues to use the shared whole-block candidate parser proof. M94 additionally requires any surviving blockquote's transformed complete `LineRange`, first marker, and every content range to match after removal.

The builder proof is updated only to recognize that generated valid blockquotes now also appear in the ordinary parsed public node set. Its construction-specific validators remain authoritative for which shapes a builder API is allowed to produce.

## TDD evidence

The initial RED failed at compile time because `Document.BlockquoteContentRanges` did not exist. The first implementation then exposed two real issues:

1. collecting `Lines()` from every AST descendant panicked on Goldmark inline nodes; the collector was restricted to block nodes;
2. parser-proven lazy continuation lines were initially represented as empty lexical ranges; they were corrected to own their complete physical line content.

Focused tests then proved:

- multiline, nested, lazy-continuation, marker-only, and multi-block public promotion;
- exactly one public top-level blockquote identity for nested source;
- exact outer range and per-line inner ranges;
- historical `ContentRange` preservation for simple one-line paragraphs and zero range for broader/structural children;
- no absorption of an unowned following line;
- CRLF, Unicode, indentation, and no-marker-space preservation;
- defensive-copy behavior of `BlockquoteContentRanges`;
- complete removal of multiline CRLF, lazy, nested, and multi-block containers;
- source-bound stale-change rejection and unsafe-join failure;
- preserved reference-usage behavior when a removed block owns a reference;
- transformed source mapping of a later surviving complex blockquote;
- generated multiline/nested/multi-block construction continues to pass its independent builder proof, while historical invalid construction shapes remain rejected.

## Devil's advocate review

1. **A markerless line could be absorbed past the real blockquote boundary.** M94 accepts such a line only when a parser descendant block segment lies wholly on that physical line. The source mapper stops at the first unproven markerless line.
2. **A continuous public content range could falsely include outer markers between lines.** `ContentRange()` is intentionally not widened. Broader source uses ordered `BlockquoteContentRanges`, with valid empty ranges for marker-only lines and full content for lazy lines.
3. **Nested blockquotes could create duplicate public identities for overlapping source.** Only blockquotes whose parent is the document are observed for promotion; nested containers remain semantic descendants of the one outer public identity.
4. **Removing one block could change a surviving blockquote's lazy/nesting/source ownership while ordinary semantic survivor checks still pass.** M94 extends removal survivor proof to the complete transformed blockquote `LineRange`, first marker, and every inner range.
5. **Broader parsed promotion could silently broaden builder authorization.** Construction validators remain separate and tests retain historical rejections such as structural child input to `AppendBlockquote`.
6. **The new source scan could violate the production complexity gate.** The first correct implementation measured cyclomatic complexity 18 in `MapTopLevelBlockquote`. It was refactored into first-line validation and continuation scanning without changing behavior; the focused suite remained green and `gocyclo -over 15` became empty.
7. **Subtree evidence collection could accidentally traverse inline APIs with unsupported operations.** The semantic range collector explicitly inspects block nodes only, avoiding Goldmark inline `Lines()` calls and keeping evidence limited to the facts needed for source ownership.

## Verification

Focused source, Goldmark adapter, and black-box public regression suites pass after the final complexity refactor. A complete `go test ./... -count=1` run also passed before documentation finalization. The first combined implementation-freeze command passed repeated tests, vet, build, Staticcheck, and golangci-lint with zero issues, then stopped because production gocyclo correctly reported the new `MapTopLevelBlockquote` at complexity 18. Independent diagnostics confirmed both `unparam` modes, `go mod tidy -diff`, and `git diff --check` were already green. After the responsibility refactor, focused tests, formatting, and the production gocyclo ≤15 gate passed again.

The exact documented tree then passed five consecutive `go test ./... -count=1` runs; full `go test -race ./... -count=1` with the private CGO-capable toolchain; `go vet ./...`; `go build ./...`; root executable examples; `go doc` for `Document.BlockquoteContentRanges` and `Blockquote`; Staticcheck; golangci-lint with zero issues; production gocyclo ≤15; both `unparam` modes; `go mod tidy -diff`; the hash-pinned published GFM 0.29 conformance suite; govulncheck with no vulnerabilities found; Gitleaks with no leaks; actionlint; direct Go 1.27.0 `go test ./...`, `go vet ./...`, and `go build ./...`; strict UTF-8/no-BOM/LF/no-trailing-whitespace hygiene; private-boundary scanning; `git diff --check`; and `git fsck --no-dangling`.

Explicit cross-package statement coverage is **86.4%** over the production package set when exercised by the complete repository suite and **83.5%** when the same production set is exercised through `internal/publictest` alone. Coverage profiles were written only under the private tool root and are not repository content.

After recording this evidence, the final tree is rechecked so no post-gate documentation edit is left unverified.

## Exit decision

M94 is complete. Existing-source blockquote promotion now covers the reviewed complete top-level container forms without inventing nested public identities, weakening source ownership, or broadening construction authorization. M95 structural mutation composition is the next roadmap boundary.
