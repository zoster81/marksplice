# M113 — Marksplice-native inline parser

Status: **Complete**

## Goal

Implement the Marksplice-owned native inline parser beneath the parser-independent M111 contract without changing the production backend, public API, existing-document source authority, construction output, or the M110 third-party extension boundary.

M113 owns the GFM inline observations and link/reference relationship facts consumed by the current parsed-document model. Goldmark remains the temporary production backend and differential oracle. Full native `parser.Backend` completion, pathological/fuzz/cross-platform hardening, and production cutover remain later milestones.

## Requirements and edge cases

The candidate must derive inline syntax only from block regions already proven by the M112 native block pass. A whole-file inline scan is not acceptable because fenced/indented code, HTML blocks, reference definitions, and other block-owned regions can contain bytes that resemble inline syntax without being inline content.

The M113 parser must preserve original byte offsets and must not create normalized shadow offsets. Tight-list paragraphs remain inline-capable even when their paragraph node is suppressed structurally. Table cells require their GFM escaped-pipe semantics to remain distinct from ordinary paragraph text. Inline ownership and precedence must prevent child syntax from escaping code spans, raw HTML, autolinks, links, and images.

Reference usage must reuse M112 reference-definition facts rather than rescan definitions independently. GFM reference-label keys require ASCII-GFM whitespace folding plus full Unicode case folding, including multi-rune folds such as `ẞ` and `SS`.

## Architecture and test strategy

`internal/parser/native` now exposes M113 candidate entrypoints:

- `ParseInlines` projects the parser-independent inline node observations owned by M113;
- `ParseInlineObservations` projects those nodes together with resolved `LinkUsages` and conservative explicit `UnresolvedReferenceUsages`.

Both entrypoints run the M112 native block pass first and parse only its ephemeral `inlineBlock` regions. Paragraphs/headings, tight-list content, blockquote descendants, and table cells therefore share one block-authority source. Reference definitions collected by M112 propagate through the same block result and feed reference resolution directly.

The implementation is split by responsibility:

- backtick runs use a next-run-by-length index for monotonic code-span pairing;
- raw HTML and autolinks are primary opaque owners with explicit precedence;
- emphasis/strong use delimiter runs, Unicode flanking, the rule of three, and categorized opener stacks rather than recursive backtracking;
- strikethrough uses the same ownership model with its narrower delimiter rules;
- direct and reference links/images use shared composite syntax records, logical multi-segment cursors, and source-ordered activation rules;
- resolved link/image/autolink relationships and conservative unresolved full/collapsed references are emitted from the same parsed owners rather than reconstructed later;
- `inline_index.go` contains ephemeral source-ordered start/interval indexes used to remove repeated all-pairs owner/composite/delimiter lookups;
- all returned inline observations are source-ordered before comparison or downstream use.

The M110 extension SPI remains unchanged. `ParseWithOptions` continues to perform the ordinary core `Parse` first and only then evaluates caller-registered read-only overlays. M113 adds no parser hook, parser-specific extension callback, or Goldmark/native-specific extension contract.

TDD advanced through exact differential failures rather than speculative grammar expansion. Important RED boundaries included the missing native entrypoint and code-span example 338, raw-HTML/autolink precedence, emphasis/strong and strikethrough sections, direct link example 494, image example 581, reference-image example 591, Unicode reference folding example 549, cross-family ordering/promotion examples 525 and 532, and finally full-corpus table example 200. The final full-corpus gate compares every M113 inline node family and every M113 relationship family across all 676 parser-applicable published GFM examples.

## Native inline contract completed by M113

M113 now matches the temporary Goldmark oracle for the parser-independent observations consumed by the parsed-document model:

- code spans;
- emphasis and strong;
- strikethrough;
- direct simple inline links;
- direct and resolved-reference images where the current observation contract is source-provable;
- resolved full/collapsed/shortcut link/image relationship forms;
- conservative unresolved explicit full/collapsed reference forms;
- angle URI/email autolinks and the current GFM bare/extended autolink forms;
- inline raw HTML ownership;
- parser-normalized reference-label keys and resolved destination/title/reference facts.

Complex or normalized syntax may be parsed as an owner without being promoted as a source-editable/simple observation when the frozen Marksplice contract cannot prove one contiguous source payload. This preserves the established separation between semantic understanding and public source authority.

For GFM tables, a code span whose semantic payload contains a table-escaped pipe is intentionally not promoted as a simple source-contiguous `KindCodeSpan`; the table extension consumes the escape semantically, matching the current parser-independent oracle projection.

## Unicode dependency decision

M113 adds a direct dependency on `golang.org/x/text v0.41.0`. The native parser uses `golang.org/x/text/cases.Fold` only for full Unicode reference-label case folding required by the GFM contract; standard-library simple case conversion cannot represent all multi-rune folds. GFM whitespace collapsing remains Marksplice-owned and intentionally uses the GFM ASCII whitespace definition rather than general Unicode whitespace.

The dependency is under the Go project's BSD-style three-clause license. Exact versions remain owned by `go.mod`/`go.sum`, and `NOTICE` records the third-party attribution. This dependency is independent of Goldmark and is not scheduled for automatic removal at M115.

## Performance and maintainability

The first correct composite implementation exposed repeated owner/composite/delimiter all-pairs lookups. Before freeze, those lookups were replaced by ephemeral source-ordered indexes and exact range suppression. No persistent parse index, AST, or cache was added.

`BenchmarkM113NativeInlineScaling` remains as review evidence rather than a wall-clock CI threshold. Representative measurements covered direct-link and delimiter-dense inputs at 256/1024/4096 items. CPU scaling remained near the expected input-growth envelope after the indexed refactor, while allocation growth tracked the larger candidate structures rather than quadratic pair matching. M114 owns broader resource/pathological/fuzz and cross-platform hardening.

Production cyclomatic complexity remains 15 or lower per function, and both production and test-inclusive `unparam` gates are clean.

## Devil's advocate review

1. **Inline scanning could classify syntax inside non-inline block regions.** Mitigation: the native block pass is the only source of inline-capable regions; no independent whole-document inline scanner is retained.
2. **Delimiter/composite precedence could become quadratic on dense malicious input.** Mitigation: backtick/opener processing is monotonic or categorized, and projection/ownership checks use ephemeral ordered indexes instead of nested all-pairs matching. A permanent scaling benchmark records the current behavior.
3. **Reference normalization could silently diverge for non-ASCII labels.** Mitigation: the native key uses full Unicode case folding through `x/text/cases.Fold`, GFM-specific ASCII whitespace collapse, and a differential test including `ẞ`/`SS`, `Straße`/`STRASSE`, escapes, and NBSP.
4. **Table-extension preprocessing could fabricate an editable code-span payload.** Mitigation: table cells carry explicit inline context; a code span containing a table-escaped pipe remains an owner but is not promoted as a simple source-contiguous observation.
5. **Passing inline parity could tempt an early production cutover.** Mitigation: M113 does not implement the complete construction-proof side of `parser.Backend`; Goldmark remains production backend/oracle until M114 hardening and the explicit M115 cutover/removal gate.

## Verification

The final M113 tree passed full **676-example** inline-node and relationship parity, indexed-precedence regressions, repeated complete repository tests, race detection, `go vet`, `go build`, Staticcheck, golangci-lint, production `gocyclo <= 15`, both unparam modes, `go mod tidy -diff`, published GFM production conformance, native block/inline/relationship full-corpus differential gates, Go 1.27 test/vet/build, govulncheck, Gitleaks, actionlint, strict text/artifact hygiene, and `git diff --check`.

Statement coverage measured **88.6% aggregate**, **84.7% through public black-box tests**, and **93.6% for the native candidate**.

## Exit decision

M113 is complete. The documented tree retains the parser/source/public boundaries above and passed the final product, quality/security, and hygiene/module/Git-state gates.

M113 does **not** switch `internal/splice` to the native parser. The temporary Goldmark bridge remains production-authoritative. M114 is the next roadmap boundary and must complete the full native `parser.Backend`/construction-proof surface and harden the native implementation with malformed/deep/oversized input tests, fuzz/resource work, benchmarks, and cross-platform evidence before M115 may perform production cutover and remove Goldmark.
