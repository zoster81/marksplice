# Scripthold R30 integration gate

**Status:** in progress after `v1.0.0`; this gate must be resolved before M125 begins.

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
| MS-R30-001 | Variable-length YAML/TOML front-matter scalar replacement rejects non-zero byte deltas | **Implemented locally.** Closing-envelope validation now translates the complete closing range rather than changing only its end. Regression coverage includes positive/negative byte deltas, Unicode byte length, quotes/comments/spacing/neighbors, LF/CRLF, reparse, and stale-source rejection. | Available after this gate is released. |
| G01 | Insert/remove ordinary top-level paragraph/block content | **Deferred within this gate.** No competing Scripthold Markdown editor is an accepted workaround. | Withhold corresponding R30 operations until implemented or explicitly declined with another source-proven Marksplice primitive. |
| G02 | Set/promote/demote heading level | **Deferred within this gate.** Must validate the resulting section hierarchy rather than edit heading punctuation in isolation. | Withhold heading-level R30 operations. |
| G03 | Parser-derived semantic heading text | **Implemented locally** as `Heading.Text()`, reusing Native's existing semantic heading projection for ATX and Setext headings. | Available after this gate is released. |
| G04 | Blockquote content mutation | **Implemented locally for safely owned uniform-prefix blockquotes** as `PrepareReplaceBlockquoteContent`. Existing marker prefix and EOL style are reused exactly; lazy/mixed-prefix source fails closed. Alert-shaped blockquotes use the alert-specific API. | Available for the proven subset; unsupported shapes remain withheld. |
| G05 | GitHub alert type/body mutation | **Implemented locally** as `PrepareSetAlertKind` and `PrepareReplaceAlertBody`. Kind replacement owns only the alert marker payload; body replacement reuses a uniform source-proven blockquote prefix/EOL. Mixed/lazy bodies fail closed. | Available for the proven subset. |
| G06 | Fenced info string/language and safe empty-block population | **Implemented locally.** `PrepareSetFencedBlockInfo` sets/replaces/clears parser-proven info without regenerating fence trivia. `PrepareReplaceFencedCode` can populate a source-proven empty closed fenced block while preserving fence character/length, indentation, closing style, and EOL. | Available after this gate is released. |
| G07 | Direct-link/image label/alt/title mutation | **Partially implemented locally.** `PrepareReplaceInlineLinkLabel`, `PrepareReplaceImageAlt`, `PrepareReplaceInlineLinkTitle`, and `PrepareReplaceImageTitle` patch source-proven payloads and preserve destination/title delimiters and trivia. Existing-title replacement is covered; adding/removing title presence is a separate structural lifecycle and remains deferred. | Label/alt and existing-title replacement available; add/remove-title operations withheld. |
| G08 | Reference occurrence mutation and definition insertion/lifecycle | **Deferred within this gate.** Existing definition destination/title replacement and removal remain unchanged. | Withhold new occurrence/definition-lifecycle R30 operations. |
| G09 | Multiline footnote body mutation and definition lifecycle | **Deferred within this gate.** Existing simple-body replacement and coordinated rename remain unchanged. | Withhold multiline/lifecycle R30 operations. |
| G10 | Structural simple front-matter add/remove/rename/envelope lifecycle | **Deferred within this gate.** The simple scalar replacement bug is fixed, but structural metadata authority is not yet added. | Withhold structural metadata R30 operations. |
| G11 | First-child insertion into an existing list | **Deferred within this gate.** Any solution must derive layout from source-proven host context or expose reviewed layout facts without moving CommonMark indentation rules into Scripthold. | Withhold first-child insertion where no proven child trivia exists. |

## Current implementation boundaries

### Blockquote and alert replacement

Arbitrary blockquote reconstruction is not authorized. Multiline replacement is available only when every owned physical line in the replaced region has the same source-proven blockquote prefix and compatible EOL style. The operation reuses that exact prefix for each replacement line and reparses the complete candidate. Lazy continuation and mixed-prefix source intentionally return `ErrInvalidReplacement`.

Generic blockquote content replacement does not rewrite alert-shaped blockquotes. Alerts have dedicated kind/body operations so the semantic overlay cannot be changed accidentally through a generic blockquote edit.

### Fenced blocks

Info-string mutation patches only the parser-proven info payload. Clearing an info string leaves authored whitespace and all fence trivia untouched. Empty-payload population is limited to source-proven closed fenced blocks; unsupported/non-proven forms do not gain generic edit authority.

### Direct links and images

Label/alt replacement and existing title-payload replacement are exact source patches followed by candidate proof. They do not add or remove title syntax. A future title-presence lifecycle must own separator/delimiter syntax explicitly rather than infer it through a payload API.

## Tranche 1 refactor and profiling evidence

The first R30 implementation tranche was refactored before freeze after the production cyclomatic-complexity gate identified three hotspots. Blockquote line-profile extraction and link/image mapping proof were split by responsibility without weakening candidate validation; the production `gocyclo <= 15` gate then returned clean and the affected focused/historical regressions remained green.

A dedicated mutation-planning benchmark resolves its immutable target once before timing and measures only `Prepare...` work. Three corrected same-host runs on the Ryzen 9 5900X recorded approximately:

- front-matter variable-length value replacement: 0.82–0.85 us/op, 336 B/op, 8 allocs/op;
- uniform blockquote content replacement: 7.8–8.0 us/op, 7,520 B/op, 65 allocs/op;
- fenced info replacement: 3.4 us/op, 2,936 B/op, 34 allocs/op;
- inline-link label replacement: 4.1–4.3 us/op, 2,160 B/op, 33 allocs/op;
- existing image-title replacement: 3.8–3.9 us/op, 2,108 B/op, 30 allocs/op.

CPU/allocation profiles show candidate reparsing and ordinary immutable-model construction as the dominant cost. No R30-specific scanner or retained index emerged as a hotspot, so this tranche deliberately keeps candidate validation and adds no persistent mutation cache.

## Exit rule before M125

M125 must not start while an R30 operation that Scripthold intends to expose still depends on Scripthold reimplementing Markdown parsing/editing rules.

Before this gate can be frozen, every deferred item above must be resolved in one of these ways:

1. implemented and tested in Marksplice;
2. intentionally declined with a safe Marksplice-owned alternative; or
3. explicitly deferred with the corresponding Scripthold R30 operation withheld.

If Scripthold requires the deferred operation for its R30 schema, that item remains a blocker and this gate remains open.
