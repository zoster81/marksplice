# M93 — Reference and Autolink Completion

## Status

Complete. The focused TDD/regression suite, implementation-freeze gate, and full documented-tree release-quality gate are green. The final post-evidence tree is reverified below.

## Objective

Complete the reviewed single-document reference/autolink construction surface and add conservative reference-definition lifecycle operations without weakening M89's exact-prior-definition full-reference contract, widening ordinary parsed reference-link promotion, or turning reference management into an unvalidated batch-edit path.

## Public contract

M93 adds typed construction forms for GFM reference links/images and parser-proven bare autolinks:

- `ForwardReferenceLinkInline` and `ForwardReferenceImageInline` emit canonical full-reference syntax against exactly one explicitly deferred definition;
- `CollapsedReferenceLinkInline` and `CollapsedReferenceImageInline` emit `[label][]` / `![alt][]` forms whose written label resolves through the parser-defined GFM normalization rules;
- `ShortcutReferenceLinkInline` and `ShortcutReferenceImageInline` emit `[label]` / `![alt]` forms under the same normalized-resolution proof;
- `BareAutoLinkInline` emits the caller-provided token without angle brackets only when reparsing recognizes the entire requested token as the existing source-proven autolink capability.

`DocumentBuilder` adds:

- `DeferReferenceDefinition`;
- `DeferReferenceDefinitionWithTitle`.

Deferred definitions are validated when registered and written deterministically after the ordinary constructed body. They provide an explicit document-level forward-reference contract. A deferred definition may not collide under GFM reference-label normalization with an already-appended or already-deferred definition.

M89 remains intact: `ReferenceLinkInline` and `ReferenceImageInline` still require exactly one already-appended exact reference label and do not treat a deferred-only definition as satisfying that prior-definition requirement. Normalized ambiguity still fails closed because the generated reference must resolve to the intended definition under GFM semantics.

Existing-source reference-definition lifecycle adds:

- `Document.PrepareReplaceReferenceDefinitionTitle`, which replaces only the payload of an already-present, source-proven title;
- `Document.PrepareRemoveReferenceDefinition`, which removes the exact complete physical line only when candidate parsing proves that every surviving reference relationship and ordinary promoted node remains semantically/source-equivalent.

`ReferenceDefinition.Range()` keeps its historical destination-range meaning. M93 does not repurpose that public range into a full-line ownership contract; complete-line ownership remains internal to the removal operation.

## Requirements and edge cases

Collapsed/shortcut labels use the parser-defined GFM normalization behavior rather than a second Marksplice normalization algorithm. Case/whitespace-equivalent definition labels that would make resolution ambiguous fail closed.

Full forward references require an explicitly deferred exact label. Existing `ReferenceLinkInline` / `ReferenceImageInline` must continue to reject a definition that exists only in deferred state.

Deferred definitions are top-level document state. A blockquote child builder cannot carry deferred definitions into `AppendBlockquoteBlocks`, because doing so would silently change definition scope and output placement.

Bare autolink construction accepts no syntax merely because it looks URL-like or email-like. The complete caller token must reparse as exactly one source-proven non-angle `AutoLink`; plain text, unsupported protocol/token shapes, multiline input, and tokens for which GFM truncates trailing punctuation fail closed.

Reference-definition title replacement requires an existing title and patches only its proven payload range. Quote delimiter style, destination spelling/wrappers, label spelling, indentation, separator spacing, trailing whitespace, and line ending remain untouched. Candidate reparsing must reproduce the same label/destination/title-presence/source mapping with only the requested title value changed.

Reference-definition removal owns the mapped complete physical line, including LF/CRLF when present. It is allowed only when removing that line does not alter surviving full, collapsed, shortcut, or reference-image relationships. A reference usage inside another block that is itself legitimately removed may disappear with that block; unrelated surviving usages must retain the same form, source-relative anchor after the patch, reference value, destination, title, and title-presence semantics.

Leading YAML/TOML front matter is outside Markdown semantics in Marksplice. Reference-like text inside a recognized front-matter envelope is therefore excluded from the internal reference-usage safety vector and cannot falsely block a body reference-definition removal.

## Architecture and test strategy

M93 keeps three responsibilities separate.

### Construction resolution

The builder stores prior and deferred reference definitions separately. Full historical references consult only prior definitions; explicit forward full references consult only deferred definitions. Collapsed/shortcut forms may resolve against the definitions available to the complete constructed document.

Reference-label normalization is isolated behind `internal/parser/goldmark` while Goldmark remains the temporary backend. `internal/splice` and the root builder exchange only Marksplice-owned construction DTOs; no Goldmark type crosses the adapter boundary.

Construction proof remains construction-only. Full/collapsed/shortcut form, written source boundaries, resolved destination/title, and structured label hierarchy are validated independently where appropriate. Ordinary parsed-source reference links/images are not promoted merely because construction can generate them.

### Deferred definition output

Deferred definitions reuse the ordinary validated reference-definition `constructionBlock` and canonical writer. `Markdown()` presents the writer with ordinary body blocks followed by a stable snapshot of deferred definitions. No second source writer or reference-specific document serializer is introduced.

### Existing-source lifecycle safety

`internal/source.ReferenceDefinitionMapping` now owns both its historical semantic/source `Range` and a separate complete `LineRange`. Destination/title replacements preserve `LineRange` under the exact byte-length delta; removal uses `LineRange` only.

The ordinary Goldmark AST walk now also records a compact, non-public `ReferenceUsage` vector for reference links/images: kind, full/collapsed/shortcut form, source anchor, written reference value, resolved destination/title, and title presence. This is mutation-safety metadata, not public promotion. It is collected during the same parser walk as ordinary observations, so M93 does not add a second whole-document parse to the steady-state snapshot path.

Recognized front-matter-envelope anchors are filtered from this vector to preserve Marksplice's existing envelope boundary. Whole-block removal proof compares only reference usages that should survive the removed span, transforming their anchors by the removal patch. This both protects reference-definition removal and avoids over-rejecting existing block-removal operations when a removed block legitimately owns a reference usage.

## TDD evidence

The construction RED initially failed only on the new missing APIs. Focused tests then established:

- parser-proven bare/extended autolink construction and exact rejection of non-link/truncated tokens;
- collapsed and shortcut reference link/image construction;
- GFM-normalized label resolution and rejection of normalized ambiguity;
- explicit deferred definitions and full forward link/image construction;
- M89 prior-definition behavior when only a deferred definition exists;
- normalized collision rejection between deferred and ordinary definitions.

The lifecycle RED initially failed only on the new missing public mutation APIs after correcting the test package name. Source-layer TDD separately failed exactly because `ReferenceDefinitionMapping` did not yet own a complete physical line.

Permanent lifecycle/regression tests prove:

- title payload replacement preserves quote style, tabs/spaces, destination bytes, and CRLF;
- definitions without titles and unsafe title replacements fail closed;
- an unused definition owns/removes exactly its physical line;
- a definition used by full, collapsed, shortcut, or image reference syntax cannot be removed;
- an unused definition before a used definition can be removed while the surviving usage anchor shifts correctly;
- reference-looking front-matter content is not treated as Markdown usage;
- a removed blockquote may legitimately remove a reference usage it owns without weakening the proof for external usages;
- the Goldmark adapter directly records full/collapsed/shortcut link usages and shortcut image usage as internal Marksplice-owned facts.

## Devil's advocate review

1. **Removing a definition could silently turn a reference link/image into plain text or resolve it to another target.** Ordinary promoted-node survivor proof cannot see non-promoted reference links. M93 therefore records non-public resolved reference usages and requires every surviving usage to retain form, transformed anchor, reference value, destination, title, and title-presence semantics.
2. **The new safety vector could over-reject removal of a block that itself contains a reference usage.** The final proof filters usages anchored inside the owned removal span and compares only survivors with transformed anchors. A dedicated blockquote regression proves this behavior.
3. **Goldmark could misinterpret reference-looking metadata as body Markdown.** Marksplice already owns front matter outside the GFM parser, so M93 filters reference usages anchored inside the recognized leading envelope. A public regression proves that such pseudo-usage does not block safe removal.
4. **Forward support could weaken M89 by making all definitions implicitly visible before their source position.** Prior and deferred scopes remain separate. Historical `ReferenceLinkInline` / `ReferenceImageInline` consult only already-appended definitions; forward full references require explicit `ForwardReference*` constructors.
5. **Duplicating GFM reference normalization in the builder could drift from the parser and later native-parser contract.** Normalization is delegated through a Marksplice-owned adapter DTO to the current parser backend; builder/splice code does not reproduce Goldmark's normalization algorithm.
6. **Broader autolink generation could accept a token whose parser only recognizes a prefix.** `BareAutoLinkInline` requires the entire requested byte range to map back to one source-proven non-angle autolink, so parser truncation fails closed.
7. **The added relationship proof could introduce a second parse or unbounded index.** Reference usages are collected in the existing AST walk and stored as one source-ordered compact slice. Candidate mutation still performs only the ordinary candidate parse required by the safety model.

## Verification

Before documentation finalization, the completed implementation passed focused parser/source/splice/public regression suites, including direct reference-usage collection, all reference-definition lifecycle cases, blockquote-removal interaction, front-matter exclusion, and shifted surviving usages.

The implementation-freeze gate passed `go test ./... -count=1`, `go test ./... -count=3`, `go vet ./...`, `go build ./...`, production `gocyclo -over 15 -ignore '_test\\.go$' .`, `unparam ./...`, `unparam -tests ./...`, and `git diff --check` using the repository's private tool installations where the analysis binaries are not on `PATH`.

The first full documented-tree attempt exposed two issues before completion: golangci-lint found an ineffectual initialization in the new reference-usage collector, and the front-matter regression used an envelope containing only an intentionally unsupported YAML plain scalar, so Marksplice correctly did not recognize that input as front matter. The collector initialization was simplified and the regression was corrected to use one recognized simple field plus opaque reference-looking content inside the same envelope. Focused regression, full suite, golangci-lint, and `git diff --check` passed after those corrections.

Two coverage/hygiene attempt failures were harness-only and are discarded: PowerShell native-argument interpolation passed literal profile-variable names to coverage tooling, and one hygiene command used an invalid `${variable}:` interpolation form. The coverage mistake created one accidental 10-byte literal `$allProfile` file inside the repository; it was verified as a regular file inside the authorized workspace and removed before continuing. Neither harness failure is counted as product verification evidence.

The corrected documented-tree release-quality gate passed five consecutive `go test ./... -count=1` runs; actual `go test -race ./... -count=1` with the private CGO-capable GCC toolchain; `go vet ./...`; `go build ./...`; executable examples and `go doc` checks for the M93 public APIs; explicit cross-package coverage; the hash-pinned published GFM 0.29 conformance suite; Staticcheck; golangci-lint with zero issues; production `gocyclo <=15`; both `unparam` modes; govulncheck with no vulnerabilities found; Gitleaks with no leaks; actionlint; `go mod tidy -diff`; direct Go 1.27.0 `go test ./...`, `go vet ./...`, and `go build ./...`; strict UTF-8/no-BOM/LF/no-trailing-whitespace hygiene; private-boundary scanning; `git diff --check`; and `git fsck --no-dangling`.

Corrected explicit cross-package statement coverage is **86.4%** over the production package set and **83.8%** when that same set is exercised through `internal/publictest` alone.

## Exit decision

M93 is complete. Collapsed/shortcut references, explicit safe forward definitions, broader parser-proven bare autolink construction, and conservative reference-definition title/removal lifecycle operations are implemented without weakening M89 or widening ordinary parsed reference-link promotion. M94 existing-source blockquote completion is the next roadmap boundary.
