# Scripthold R30 integration gate

**Status:** Complete after `v1.0.0`. All Scripthold R30 Markdown-authority findings are implemented within explicit fail-closed boundaries and the freeze candidate has passed the required local verification gates. M125 remains not started and requires its own explicit start boundary.

## Purpose

Scripthold R30 dogfooding exercises Marksplice as the Markdown authority for query/read semantics, structured construction, and source-preserving existing-document mutation. Scripthold remains responsible for filesystem safety, encoding/BOM policy, preview/apply, backup, and conflict lifecycle.

HTML rendering, HTML source maps, and PDF work are outside this gate.

The gate exists to prevent consumer-required Markdown editing authority from being hidden inside M125 PDF work or reimplemented in Scripthold.

## Invariants

Every R30 mutation keeps the existing Marksplice contract:

- immutable source snapshots and snapshot-bound identity;
- exact source ownership;
- stale-source rejection;
- no regeneration of unrelated source;
- candidate reparsing before returning a `ChangeSet` when structure can change;
- fail-closed handling of ambiguous or unsupported source;
- byte-range correctness for Unicode and LF/CRLF preservation;
- compatibility with `ComposeChanges` for independent operations;
- black-box public contract tests;
- explicit read/edit/create capability documentation.

## Dogfood findings and upstream disposition

| ID | Finding | Upstream disposition | R30 availability |
| --- | --- | --- | --- |
| MS-R30-001 | Variable-length YAML/TOML front-matter scalar replacement rejects non-zero byte deltas | **Implemented.** Closing-envelope validation now translates the complete closing range rather than changing only its end. Regression coverage includes positive/negative byte deltas, Unicode byte length, quotes/comments/spacing/neighbors, LF/CRLF, reparse, and stale-source rejection. | Available in the R30 post-v1.0 tree. |
| G01 | Insert/remove ordinary top-level paragraph/block content | **Implemented for promoted top-level paragraphs** as `PrepareInsertParagraphBefore`, `PrepareInsertParagraphAfter`, and `PrepareRemoveParagraph`. Insert payload must independently reparse as exactly one paragraph; Marksplice owns separators/EOL and validates original-node/link survivors. | Available for the proven paragraph subset. |
| G02 | Set/promote/demote heading level | **Implemented** as `PrepareSetHeadingLevel`. ATX marker changes and supported Setext transitions are source-owned; Setext-to-ATX conversion is limited to single-line source-proven headings. Candidate reparsing validates heading order/identity and rebuilt `Sections()`. | Available for the proven subset. |
| G03 | Parser-derived semantic heading text | **Implemented** as `Heading.Text()`, reusing Native's existing semantic heading projection for ATX and Setext headings. | Available. |
| G04 | Blockquote content mutation | **Implemented for safely owned uniform-prefix blockquotes** as `PrepareReplaceBlockquoteContent`. Existing marker prefix and EOL style are reused exactly; lazy/mixed-prefix source fails closed. Alert-shaped blockquotes use the alert-specific API. | Available for the proven subset; unsupported shapes remain fail-closed. |
| G05 | GitHub alert type/body mutation | **Implemented** as `PrepareSetAlertKind` and `PrepareReplaceAlertBody`. Kind replacement owns only the alert marker payload; body replacement reuses a uniform source-proven blockquote prefix/EOL. Mixed/lazy bodies fail closed. | Available for the proven subset. |
| G06 | Fenced info string/language and safe empty-block population | **Implemented.** `PrepareSetFencedBlockInfo` sets/replaces/clears parser-proven info without regenerating fence trivia. `PrepareReplaceFencedCode` can populate a source-proven empty closed fenced block while preserving fence character/length, indentation, closing style, and EOL. | Available. |
| G07 | Direct-link/image label/alt/title mutation | **Implemented.** Label/alt and existing-title payload replacement remain exact source patches; `PrepareAddInlineLinkTitle`, `PrepareRemoveInlineLinkTitle`, `PrepareAddImageTitle`, and `PrepareRemoveImageTitle` own title separator/delimiter lifecycle without regenerating destination or wrappers. | Available. |
| G08 | Reference occurrence mutation and definition insertion/lifecycle | **Implemented.** `PrepareRetargetReferenceOccurrence` handles simple parser-proven full/collapsed/shortcut references, `PrepareRenameReferenceDefinition` atomically renames one unique definition plus bound occurrences, append APIs add canonical definitions, and title add/remove owns definition title syntax. Ambiguous normalized definitions fail closed. | Available for source-proven simple references. |
| G09 | Multiline footnote body mutation and definition lifecycle | **Implemented.** `PrepareReplaceFootnoteDefinitionBodyMultiline` renders logical LF payload through the source-proven footnote EOL/continuation layout; `PrepareAppendFootnoteDefinition` and `PrepareRemoveFootnoteDefinition` add/remove complete containers. The historical simple-body API remains intentionally narrow. | Available for proven containers. |
| G10 | Structural simple front-matter add/remove/rename/envelope lifecycle | **Implemented.** `PrepareRenameFrontMatterField`, `PrepareRemoveFrontMatterField`, and `PrepareAppendFrontMatterField` operate only on conservative simple scalar mappings; `PrepareAddFrontMatter`/`PrepareRemoveFrontMatter` own the complete leading envelope. Canonical new scalar values use the existing no-escaping string contract; complex YAML/TOML stays opaque. | Available for the conservative simple-field/envelope subset. |
| G11 | First-child insertion into an existing list | **Implemented** as `PrepareAppendFirstListItemChild`. Marksplice derives container prefix, indentation, EOL and canonical child marker from the source-proven parent, then reuses the existing list subtree proof. | Available. |

## Current implementation boundaries

### Blockquote and alert replacement

Arbitrary blockquote reconstruction is not authorized. Multiline replacement is available only when every owned physical line in the replaced region has the same source-proven blockquote prefix and compatible EOL style. The operation reuses that exact prefix for each replacement line and reparses the complete candidate. Lazy continuation and mixed-prefix source intentionally return `ErrInvalidReplacement`.

Generic blockquote content replacement does not rewrite alert-shaped blockquotes. Alerts have dedicated kind/body operations so the semantic overlay cannot be changed accidentally through a generic blockquote edit.

### Fenced blocks

Info-string mutation patches only the parser-proven info payload. Clearing an info string leaves authored whitespace and all fence trivia untouched. Empty-payload population is limited to source-proven closed fenced blocks; unsupported/non-proven forms do not gain generic edit authority.

### Direct links and images

Label/alt and title-payload replacement remain exact source patches followed by candidate proof. Title add/remove is a distinct lifecycle that owns only the required separator plus title delimiters/payload; destination spelling/wrappers and unrelated trivia are preserved.

### Structural paragraph, heading, list, reference, footnote, and metadata edits

Paragraph insertion/removal is limited to promoted top-level paragraphs and validates the complete resulting candidate rather than exposing arbitrary raw-block injection. Heading-level mutation proves the rebuilt section hierarchy. First-child list insertion derives layout exclusively from the source-proven parent instead of requiring the caller to know CommonMark indentation rules.

Reference lifecycle operations remain limited to source-proven simple occurrences/definitions and fail closed on normalized-label ambiguity. Footnote multiline replacement rewrites only the owned definition container using its proven continuation/EOL layout. Front-matter lifecycle operations remain a conservative Marksplice-owned envelope layer: they do not add a YAML/TOML parser, schema, AST, or serializer.

## Tranche 1 refactor and profiling evidence

The first R30 implementation tranche was refactored before freeze after the production cyclomatic-complexity gate identified three hotspots. Blockquote line-profile extraction and link/image mapping proof were split by responsibility without weakening candidate validation; the production `gocyclo <= 15` gate then returned clean and the affected focused/historical regressions remained green.

A dedicated mutation-planning benchmark resolves its immutable target once before timing and measures only `Prepare...` work. Three corrected same-host runs on the Ryzen 9 5900X recorded approximately:

- front-matter variable-length value replacement: 0.82–0.85 us/op, 336 B/op, 8 allocs/op;
- uniform blockquote content replacement: 7.8–8.0 us/op, 7,520 B/op, 65 allocs/op;
- fenced info replacement: 3.4 us/op, 2,936 B/op, 34 allocs/op;
- inline-link label replacement: 4.1–4.3 us/op, 2,160 B/op, 33 allocs/op;
- existing image-title replacement: 3.8–3.9 us/op, 2,108 B/op, 30 allocs/op.

CPU/allocation profiles show candidate reparsing and ordinary immutable-model construction as the dominant cost. No R30-specific scanner or retained index emerged as a hotspot, so this tranche deliberately keeps candidate validation and adds no persistent mutation cache.

## Tranche 2 refactor and profiling evidence

The structural tranche added paragraph/heading/list/title/reference/footnote/front-matter lifecycle authority. Before freeze, the production complexity gate identified new reference/footnote proof helpers above the repository limit. Responsibilities were split into mapping, patch planning, and candidate-proof helpers; the production-only `gocyclo -over 15` gate then returned clean while R30, M104, source, and splice regressions remained green.

Three corrected same-host benchmark runs on the Ryzen 9 5900X measured only prepared-mutation planning after resolving immutable targets:

- coordinated reference-definition rename: 10.7–11.1 us/op, 12,184 B/op, 122 allocs/op;
- multiline footnote-body replacement: 14.9–15.2 us/op, 13,560 B/op, 131 allocs/op;
- simple front-matter field rename: 8.7–9.2 us/op, 8,216 B/op, 56 allocs/op.

CPU/allocation profiling again attributes the majority of retained work to candidate reparsing/model construction (`parseWithValidatedBackend` dominates cumulative allocated space). R30-specific patch planning remains a minority cost, so no persistent cache or weaker candidate validation is justified by the measurements.

## Freeze verification

The completed R30 candidate passed the following local gates on Windows/amd64 with the pinned private toolchain:

- focused R30 public/internal tests plus historical regressions used during each TDD tranche;
- full `go test ./...`, `go vet ./...`, `go build ./...`, `go mod verify`, `go mod tidy -diff`, and `git diff --check`;
- full CGO/GCC race detector under Go 1.27.1;
- Staticcheck, golangci-lint, production `gocyclo <= 15`, unparam, and govulncheck with no vulnerabilities reported;
- explicit CommonMark 0.31.2, published-GFM, and retained 6,857-document real-world-corpus test activation with exit 0;
- documentation dogfood over 161 Markdown documents, 153 graph edges, zero workspace diagnostics, and 406/406 exported callables represented in the exhaustive API reference;
- full `go test ./...` and `go build ./...` under the supported Go 1.26.8 compatibility toolchain;
- Go 1.27.1 CGO-disabled cross-build coverage for the supported non-Windows release targets exercised by the freeze gate.

Two static-analysis findings discovered during freeze were resolved before the final gate: one idiomatic `bytes.ContainsAny` simplification and one unused internal fenced-block helper result. Neither changed public behavior.

## Exit rule before M125

R30 is complete: no Scripthold R30 operation represented by G01–G11 requires Scripthold to reimplement Markdown parsing/editing rules. Unsupported source shapes remain deliberately fail-closed inside Marksplice rather than delegated to the consumer.

This gate does not start M125. M125 remains a separate future v1.5 PDF milestone and requires an explicit development boundary.
