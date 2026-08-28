# M118 — Native Semantic Walk Foundation

Date: 2026-08-28
Status: Complete

## Goal

M118 establishes the internal, on-demand semantic projection that later renderers can consume without exposing a second public AST or increasing the retained model of every parsed `Document`.

Native remains the only Markdown syntax authority. The semantic layer is an internal synchronous event walk over one caller-provided source snapshot. It is deliberately separate from the ordinary `parser.Backend` contract used by `Parse`, so rendering support is paid for only when explicitly requested.

## Requirements and edge cases

The M118 foundation must:

- expose a parser-independent internal event vocabulary without exporting parser-internal types;
- emit balanced document/block/inline enter/exit events plus leaf events for text and terminal semantics;
- preserve snapshot-local byte ranges where useful for diagnostics and later source mapping;
- decode ordinary semantic text through Native-owned escape/entity handling instead of asking renderers to reinterpret Markdown;
- preserve Native-owned inline hierarchy for emphasis, strong, strikethrough, links, images, code spans, raw HTML, and autolinks;
- expose foundation events for blockquotes, lists, list items, tasks, thematic breaks, tables, fenced code, block HTML, footnotes, and reviewed mathematical forms;
- stop immediately and return the visitor error unchanged;
- reject a nil visitor with a classifiable internal error;
- retain no source slice, visitor, event tree, parser AST, or operation-local lookup index after the walk returns;
- leave the ordinary `Parse`/`ParseDocument` path structurally unchanged when the semantic projection is unused.

M119 deliberately owns rendering-completeness policy and conformance, including complete nested block-container projection, list start/tightness, indented code, reference-definition emission policy, alert/front-matter policy, and specification-backed semantic fixture coverage.

## Architecture and test strategy

`internal/parser` defines the internal semantic vocabulary, event value, visitor callback, and optional `SemanticBackend` interface. It does not add methods to the existing `parser.Backend`, so the production parse path does not need to construct rendering state.

`internal/parser/native` implements `WalkSemantic` by running the same Native block and inline analysis used to decide Markdown semantics, then projecting those ephemeral ownership facts directly into events:

1. parse Native block ownership and inline analyses on demand;
2. build one operation-local `semanticProjectionIndex` for source-start, list, task, table, and fenced-code lookups;
3. emit a balanced document boundary;
4. emit the current M118 block foundation;
5. emit Native-owned inline hierarchy from existing owner/composite/delimiter relationships;
6. emit footnote and math overlay events from the existing Native observation passes;
7. return immediately on the first visitor error;
8. discard all projection state when the call returns.

The inline walker does not infer parentage from the flattened public node vector. It reuses Native's ephemeral delimiter/composite ownership and the existing construction-semantic hierarchy machinery, which is the same syntax authority already responsible for delimiter pairing and link/image ownership.

TDD first established a compile RED for the missing semantic contract, then a focused GREEN for document/heading/paragraph/text/break/inline nesting. A second RED introduced the wider block/supplemental vocabulary and became GREEN only after those foundation families were emitted. Hardening tests cover nil visitors, early visitor failure, failure during supplemental events, deterministic repeated walks, empty input, valid source ranges, and non-retention of caller source bytes.

## Devil's advocate review

1. **A flat-node reconstruction could create a second, incorrect Markdown interpreter.**
   Source-range containment is not sufficient to recover delimiter/link ownership for all inline shapes. M118 instead projects the ownership already calculated by Native. Renderers therefore receive semantic events and never reparse Markdown delimiters.

2. **A shared rendering-heavy parse path could penalize every `Parse` call.**
   `SemanticBackend` is separate from `parser.Backend`, `ParseDocument` was not modified to call the walk, and every semantic lookup map is local to `WalkSemantic`. Pre/post ordinary Parse benchmarks therefore test the actual unused-support cost rather than assuming it is free.

3. **Naive lookup could turn rendering into an accidental quadratic pass.**
   The first broader M118 draft repeatedly scanned all inline analyses, table details, and list nodes. A 256 KiB realistic benchmark exposed approximately 143–175 ms per walk. The implementation was rejected before freeze and replaced with one ephemeral projection index; the same workload fell to roughly 36–42 ms with no persistent cache.

4. **A new event dispatcher could become a maintainability hotspot.**
   The first complete dispatcher/table implementation exceeded the project cyclomatic-complexity gate at 22 and 20. The code was split by responsibility into block emitters and table row/cell helpers. Production `gocyclo -over 15` is now clean.

## Implemented foundation

M118 currently emits:

- document enter/exit;
- paragraph and heading enter/exit with semantic text;
- text leaves with reviewed escape/entity decoding;
- soft and hard line breaks;
- emphasis, strong, and strikethrough containers;
- link and image containers with destination/title metadata;
- code-span, raw-HTML, and autolink leaves;
- blockquote foundation boundaries;
- list/list-item boundaries and task leaves;
- thematic-break leaves;
- table/table-row/table-cell hierarchy with header/column/alignment facts and inline cell content;
- fenced-code leaves with info string, language, payload, and fenced classification;
- block-HTML leaves;
- footnote-definition boundaries and footnote-reference leaves;
- reviewed mathematical-expression leaves with style and payload.

The vocabulary also reserves reference-definition, alert, and front-matter concepts so M119 can complete their reviewed policy without forcing a renderer-facing vocabulary break. Reservation is not a claim that those policies are complete in M118.

## Performance and refactor evidence

The frozen pre-M118 256 KiB realistic baseline on the same Windows/amd64 Ryzen 9 5900X host measured approximately:

| Path | Median | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| public `Parse` | 44.86 ms | ~53.20 MB | ~231,672 |
| Native `ParseDocument` | 33.44 ms | ~38.30 MB | ~196,940 |

After the M118 implementation and complexity refactor, the same ordinary paths remain approximately:

| Path | Median | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| public `Parse` | 47.25 ms | ~53.20 MB | ~231,672 |
| Native `ParseDocument` | 35.45 ms | ~38.30 MB | ~196,940 |

Both ordinary timings moved by a similar roughly 5–6% on the later run while allocation bytes/counts remained structurally unchanged, so the wall-clock delta is treated as same-host noise/load rather than an M118 retained-cost regression. The important architectural signal is that unused semantic support added no new ordinary-path allocation state.

The final on-demand semantic walk itself measures a 256 KiB median of approximately **38.82 ms / 36.66 MB / 224,018 allocations**. Before the ephemeral index refactor the same draft path required roughly 143–175 ms, so the reviewed design removed the accidental near-quadratic behavior by more than 3.5x without adding a persistent cache or retained rendering model.

## Verification state

The final documented-tree freeze gate on 2026-08-28 passed:

- focused semantic-walk tests and full `go test ./... -count=1` regression;
- actual Go 1.26.6 CGO/GCC `go test -race ./... -count=1`;
- `go vet`, `go build`, Staticcheck, golangci-lint with zero issues, production `gocyclo <= 15`, and both production/test-inclusive `unparam` modes;
- `go mod tidy -diff` and `git diff --check`;
- `govulncheck`, Gitleaks, and actionlint;
- the exact anchored CommonMark 0.31.2 corpus gate plus Native CommonMark/GFM parser-contract gates;
- Go 1.27.0 test/vet/build plus CGO-disabled linux/amd64, darwin/amd64, and darwin/arm64 cross-builds;
- a retained real-world-corpus semantic-walk gate over **6,857 documents / 60,778,570 bytes / 2,490,828 emitted events**, walking every document twice while checking valid ranges, balanced event nesting, deterministic output, and source immutability without treating current Native output as a semantic oracle;
- strict UTF-8/no-BOM/LF/no-NUL/no-trailing-whitespace and private-boundary hygiene, unchanged `go.mod`/`go.sum`, `git fsck --no-dangling`, and the exact reviewed eight-file M118 whitelist;
- pre/post ordinary Parse allocation/CPU comparison and dedicated on-demand semantic-walk benchmarking.

M118 is therefore frozen as the internal semantic-walk foundation. Semantic completeness policy and specification-backed renderer-facing conformance remain explicitly assigned to M119.

## Exit boundary

M118 changes no exported Go API and retains no semantic tree in public `Document` values. It establishes only the internal renderer-ready walk foundation.

M119 is the next engineering boundary. It must close semantic completeness gaps against reviewed specification expectations and perform the first broad Native/semantic refactor and conformance campaign before any public renderer ships.
