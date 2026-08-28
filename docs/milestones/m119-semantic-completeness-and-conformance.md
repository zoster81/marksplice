# M119 — Semantic Completeness and Conformance

Date: 2026-08-28
Status: Complete

## Goal

M119 completes the internal Native semantic event stream required by deterministic renderers before Marksplice exposes an HTML or canonical-Markdown rendering API.

The milestone keeps Native as the only Markdown syntax authority. Rendering consumers receive already-decided semantic facts and must not reparse Markdown delimiters, list structure, reference syntax, task markers, alerts, front matter, footnotes, or mathematical overlays.

## Requirements and edge cases

The completed semantic projection must:

- preserve true recursive ownership for nested blockquotes, lists, list items, tables, alerts, and footnote bodies rather than reconstructing hierarchy from flattened observations;
- expose ordered-list start values and tight/loose list state;
- emit indented code blocks and reference definitions in source order;
- recognize the existing Marksplice front-matter envelope before Markdown block parsing so envelope bytes are not reinterpreted as thematic breaks/headings;
- promote exact GitHub alert markers through one shared source policy and remove the marker line from rendered alert body semantics;
- replace source text with footnote-reference and mathematical terminal events in place instead of appending supplemental events after the document body;
- replace one-line block-dollar paragraphs with mathematical leaves rather than emitting both paragraph text and math;
- emit renderer-ready autolink destinations, including `mailto:` for email autolinks and `http://` for GFM `www.` extended autolinks;
- omit task checkbox marker bytes from semantic text while retaining the following source whitespace;
- provide CommonMark code-span semantic values, including multiline normalization, even when the source payload is not representable by one contiguous content range;
- preserve deterministic source-local ranges and stop immediately on visitor errors;
- retain no semantic tree/index after `WalkSemantic` returns;
- leave ordinary `Parse`/`ParseDocument` free of retained semantic-capture state.

Malformed source, CRLF, invalid UTF-8 bytes, deep blockquotes, unclosed syntax, empty documents, and visitor failure remain deterministic and fail without source mutation.

## Architecture and test strategy

M119 moves block hierarchy capture to the exact recursive Native block-parse point. `parseBlockLinesSemantic` accepts an optional operation-local capture; ordinary `parseBlockLines` passes `nil`, so normal parsing pays no semantic-capture allocation.

The semantic capture stores compact block entries with parent indices. Container children are emitted from compact first-child/next-sibling index arrays built only for the walk. Paragraph lookup used by footnote promotion is indexed by source start instead of repeatedly scanning all captured blocks.

Front matter is mapped first through the existing source-layer envelope authority and only the body lines are passed to Markdown block parsing. Alert marker recognition is centralized in `internal/source` and reused by both the existing public alert projection and the internal semantic walk.

Footnote definitions are promoted in place from their source-owning paragraph region and their stored physical child lines are reparsed semantically under the definition container. Footnote references and inline math become source-ordered terminal replacements. Block-dollar math replaces its exact paragraph capture. Renderers therefore never need a second pass that reparses delimiters.

Inline emission reuses Native ownership/composite/delimiter facts. Task prefix exclusion, email-autolink classification, reference-link/image resolution, code-span delimiters, escapes/entities, breaks, and table-cell inline ownership come from existing parser-proven facts rather than renderer heuristics.

TDD established focused REDs for nested container ownership/list facts, indented code/reference definitions, front matter, alerts, footnote/math placement, task text, email destinations, and multiline code spans. The semantic conformance harness then added manually reviewed expectations tied by official example identity to the approved hash-pinned CommonMark/GFM snapshots. It never serializes current Native output into its own expectations.

## Devil's advocate review

1. **Post-hoc hierarchy reconstruction could become a second Markdown parser.**
   M119 captures parentage while Native recursively decides the block structure. No range-containment grammar is used to guess nesting later.

2. **Footnote/math overlays could duplicate or reorder source semantics.**
   Definitions, references, and math now replace their exact underlying paragraph/text regions in source order. Supplemental append-after-body emission was removed.

3. **Semantic completeness could make the walk accidentally quadratic.**
   Profiling exposed a linear `paragraphContaining` scan consuming roughly 31% of CPU on the 256 KiB workload. M119 replaced it with an ordered paragraph index and binary lookup, then compacted child adjacency and removed excess merge capacity.

4. **The richer capture could penalize every public parse.**
   Semantic capture remains optional and operation-local. Ordinary `ParseDocument` calls the same block parser with capture disabled, and public/Native allocation measurements remain structurally close to the M118 baseline.

5. **Shared alert policy could diverge between public alerts and rendering.**
   The five exact GitHub alert markers now have one parser-independent internal source policy used by both surfaces.

6. **Specification tests could become tautological.**
   The M119 semantic expectations are hand-reviewed against selected official CommonMark/GFM examples loaded from hash-pinned external snapshots. Current Native output is only the implementation under test.

## Implemented semantic completeness

The M119 stream now covers, in renderer-ready source order:

- document, paragraph, heading, blockquote, list/list-item, table/row/cell, alert, and footnote-definition containers;
- ordered-list start values, marker style, and tight/loose state;
- task state without checkbox-marker text duplication;
- semantic text with escape/entity decoding plus soft/hard breaks;
- emphasis, strong, strikethrough, direct/reference links and images;
- code spans with CommonMark whitespace/line-ending normalization;
- raw inline HTML and block HTML;
- angle, bare, and GFM extended autolinks with renderer-ready destinations and email classification;
- reference-definition leaves;
- fenced and indented code blocks;
- thematic breaks;
- table header/column/alignment facts with cell inline content;
- front-matter envelope leaves with YAML/TOML format identity;
- exact GitHub NOTE/TIP/IMPORTANT/WARNING/CAUTION alert containers;
- footnote definitions/references in source position;
- reviewed inline-dollar, dollar-backtick, block-dollar, and existing fenced-math semantics.

M119 changes no exported Go API. The semantic contract remains internal until a renderer consumes it.

## Specification-backed semantic conformance

The permanent M119 harness loads the same approved external snapshots already used by parser conformance and checks manually reviewed semantic expectations for selected orthogonal examples:

- CommonMark 0.31.2 examples 80, 335, 527, 572, 604, and 633;
- published GFM examples 199, 279, 491, 622, and 630.

These cover Setext headings, multiline code spans, reference links, images, email autolinks, hard breaks, tables, task lists, strikethrough, extended `www.` autolinks, and extended email autolinks. The existing complete parser-neutral contract remains separately authoritative for all 652 CommonMark and 676 parser-applicable GFM examples.

A permanent real-world semantic invariant test also walks the retained corpus twice per document while checking valid event/content ranges, balanced nesting, deterministic event digest, and source immutability. On the certified corpus this covers **6,857 Markdown documents / 60,778,570 bytes / 3,313,867 semantic events per pass**. These are invariants, not expected semantic-output fixtures.

## Refactor and performance evidence

The first complete M119 draft regressed the 256 KiB realistic semantic benchmark to roughly **68.5 ms / 57.05 MB / 239.6k allocations**. CPU profiling identified repeated paragraph containment scans as the dominant hotspot; allocation profiling also exposed over-reserved capture/inline storage and allocation-heavy child adjacency.

The accepted refactor:

- replaces linear footnote paragraph lookup with a frozen ordered paragraph index;
- reserves semantic capture from measured line cardinality without imposing a size cap;
- removes unused extra capacity from merged inline observations;
- replaces `[][]int` child adjacency with compact first-child/next-sibling arrays;
- removes the superseded M118 block/supplemental emitter implementation so only one semantic path remains;
- splits code-span payload validation/normalization to keep production cyclomatic complexity within the project limit.

A stable post-refactor 256 KiB run measured about **40.05 ms / 40.26 MB / 233k allocations**. Relative to the M118 semantic freeze of about **38.82 ms / 36.66 MB / 224k allocations**, that is approximately **+3.2% time, +9.8% allocated bytes, and +4.0% allocations** while adding the missing semantic completeness.

An independent later same-host triplet measured medians of approximately **49.56 ms / 53.28 MB / 232.5k allocations** for public `Parse`, **36.11 ms / 38.38 MB / 197.8k allocations** for Native `ParseDocument`, and **44.66 ms / 40.26 MB / 233.0k allocations** for `WalkSemantic`. Wall-clock timing moves with host load; allocation bytes/counts are the more stable architectural signal. Ordinary parse allocation state remains close to the M118 boundary, while semantic-walk allocated bytes remain inside the planned representative 10% review budget.

## Final verification state

The final documented-tree local freeze gate on 2026-08-28 passes:

- focused semantic tests and complete `internal/parser/...` regression;
- production `gocyclo <= 15`, production/test-inclusive `unparam`, `go vet`, and `git diff --check` at the semantic refactor checkpoint;
- bounded semantic fuzzing with more than 240,000 executions and no discovered failure;
- explicit pathological deep-blockquote, CRLF/front-matter/alert, malformed/unclosed, and invalid-UTF-8 semantic invariants;
- exact CommonMark snapshot loading plus all Native CommonMark/GFM parser-neutral contract gates;
- the selected CommonMark/GFM semantic conformance harness;
- the complete 6,857-document Native parser corpus plus all 384 deterministic malformed derivatives;
- the complete 6,857-document permanent semantic invariant corpus gate;
- same-host public Parse, Native ParseDocument, and semantic-walk benchmark comparison;
- full `go test ./... -count=1` plus actual Go 1.26.6 CGO/GCC race detection;
- `go vet`, `go build`, Staticcheck, golangci-lint with zero issues, production `gocyclo <= 15`, and production/test-inclusive `unparam`;
- `go mod tidy -diff`, `git diff --check`, govulncheck, Gitleaks, and actionlint;
- Go 1.27.0 test/vet/build plus CGO-disabled linux/amd64, darwin/amd64, and darwin/arm64 cross-builds;
- documentation dogfood over **152 Markdown documents** with **137 graph edges**, zero workspace diagnostics, and **359/359 exported callables** represented in the API reference;
- strict M119 change-set UTF-8/no-BOM/LF/no-NUL/no-trailing-whitespace hygiene, private-boundary checks, unchanged `go.mod`/`go.sum`, `git fsck --no-dangling`, and reviewed Git-state inventory.

The freeze commit and ordinary push are performed only after a post-status recertification of this exact tracked state. Public CI on the exact pushed commit is the remote closure confirmation; it does not change the M119 architecture or require a second public milestone-content commit.

## Exit boundary

M119 is complete at the reviewed local freeze boundary above. The milestone freeze commit must preserve this exact content and pass an ordinary non-force push plus exact remote/CI verification before M120 work begins.

M120 is the next engineering boundary after that remote closure. It may consume this internal semantic stream to implement deterministic HTML fragments, but it must not add a second Markdown parser or silently weaken raw-HTML, unsafe-URL, GFM tag-filter, footnote, task-list, or authority policies.
