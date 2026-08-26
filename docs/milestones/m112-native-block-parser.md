# Milestone M112 — Marksplice-Native Block Parser

Status: complete — native block grammar, parser-independent block observations, full published-GFM block-projection parity, maintainability refactor, and release-quality verification are green.

## Goal

Implement Marksplice-owned document/block parsing beneath the parser-independent M111 contract without changing public APIs, source-preservation authority, construction output, dependency versions, or the temporary production-backend decision.

M112 owns block structure only. Native inline parsing remains M113, complete native conformance/hardening remains M114, and Goldmark removal/cutover remains M115.

## Scope

The native candidate under `internal/parser/native` covers the reviewed CommonMark/GFM block families required by Marksplice:

- physical-line and indentation handling, including tabs and CR/LF/CRLF boundaries;
- paragraphs and Setext headings;
- ATX headings;
- thematic breaks;
- indented code;
- fenced code/container observations and info strings;
- blockquotes, including nested and lazy source ownership needed by Marksplice observations;
- unordered/ordered lists, nested lists, tight/loose semantics, list-item/container relationships, and task markers;
- GFM tables and alignments;
- GFM HTML block classes;
- link-reference-definition block grammar, including multiline labels/titles and parser-normalized semantic values;
- parser-independent source/range facts consumed by the M111 differential harness.

M112 deliberately does not implement native text/escape/code-span/emphasis/strong/strikethrough/link/image/reference/autolink/raw-HTML inline grammar. Those remain M113.

## Architecture

The candidate is intentionally internal and is not wired into production `Parse` yet.

`internal/parser/native` is split by responsibility:

- `blocks.go` owns block orchestration and block-family dispatch;
- `lines.go` owns physical-line views, indentation columns, tab expansion, blank-line classification, and virtual indentation used by nested containers;
- `lists.go` owns list markers, item collection, tight/loose classification, nested-container ownership, list relationships, and task projection;
- `tables.go` owns GFM table row/delimiter/cell/alignment block scanning;
- `references.go` owns link-reference-definition block scanning;
- `html.go` owns GFM HTML-block recognition/termination and ASCII-only case folding.

The parser operates on the caller's source bytes and half-open byte offsets. Nested containers derive source-aligned line views rather than copied normalized Markdown. No second retained AST, parser context, persistent line index, convenience graph, filesystem/network authority, or public runtime state is introduced.

Production `internal/splice` continues to use the single temporary M111 Goldmark bridge. Passing M112 does not authorize an early backend switch.

## Differential contract

M112 adds a native block projection to the M111 parser-neutral differential harness.

The authoritative published-corpus gate is:

```text
TestNativeBlockParserMatchesPublishedGFMBlockProjection
```

It traverses all 676 parser-applicable examples from the same approved published GFM snapshot used by conformance and compares every block observation owned by M112 against the temporary Goldmark oracle. The single renderer-only tagfilter example remains outside parser scope exactly as documented by the conformance policy.

The comparison intentionally excludes inline-derived observations that M113 has not implemented. Passing this gate is therefore native **block-parser parity**, not a claim that the complete native parser is ready for production cutover.

Lists also have a stronger full-projection gate over published examples 231–307 so tightness/container errors cannot hide behind a comparison restricted to `ListItem` or `Task` observations.

## TDD and edge cases

Implementation advanced family-by-family against focused tests and the pinned published corpus. Important defects exposed during integration included:

- published example 270: nested-list/container interaction exposed incomplete list projection;
- example 287: blank lines owned by a descendant list incorrectly propagated looseness to ancestor lists;
- example 305: a nested list consumed a trailing blank separator that the parent item needed to classify the following direct paragraph;
- example 561: a whitespace-only reference label was incorrectly accepted as a link-reference definition.

The list fix records blank separation between direct item roots rather than blindly propagating descendant physical blank lines. Tight-list paragraph suppression is applied only after every direct item has been parsed and the enclosing list's tight/loose state is known. A nested list can propagate only the trailing separator it actually consumes at its parent boundary.

Reference labels now require at least one non-whitespace character, matching the pinned GFM definition. Staticcheck later exposed a real multiline-label scanner shadowing defect: a loop-local `position` result could be discarded before the next physical line. The scanner now updates the enclosing position state explicitly.

## Complexity and refactor

The first full implementation exposed six production functions above the project cyclomatic-complexity limit of 15, with a peak of 26. M112 refactored responsibilities instead of weakening the gate:

- shared physical-line/indent primitives moved from block orchestration into `lines.go`;
- HTML complete-tag parsing was split into name/tail scanners and uses allocation-free ASCII byte folding for hot comparisons;
- list-marker parsing was split into marker-token, ordered-number, padding, and completion helpers;
- reference-definition scanning was split into definition start, destination, title, label-line, and destination scanners;
- block orchestration was split into focused leaf/container and structural dispatch helpers;
- dead parameters/results exposed by `unparam` were removed.

The final production `gocyclo -over 15` result is empty. Production and test-inclusive `unparam` are both clean.

## Performance and retained state

Ordinary block parsing is source-linear or near-linear by construction:

- physical source is scanned once into compact line views;
- block dispatch advances monotonically through those views;
- nested containers parse only their owned child line views;
- list tight/loose classification uses direct-root metadata rather than reparsing items;
- HTML/reference/list scanners advance monotonically over their local source;
- no persistent native parser AST/cache/index is retained by public documents.

M112 adds candidate-parser code only. Public `Parse` still uses the temporary Goldmark backend, so M112 does not claim a change in production parse CPU/allocation behavior.

## Devil's advocate review

1. **Nested blank lines could leak through list levels and change paragraph projection.** Mitigation: classify looseness from direct roots and propagate only an explicitly consumed trailing separator; protect the complete published list range with full projection parity.
2. **Tabs or nested virtual indentation could drift byte ownership away from original source.** Mitigation: `physicalLine` keeps physical byte offsets separate from virtual indentation; every observation remains an offset into the original source, with focused nested/tab/CRLF differential coverage.
3. **Reference-definition scanning could accept source that later native inline reference semantics reject.** Mitigation: implement only the GFM block-definition grammar required at M112, including label length/non-whitespace constraints; reference-link resolution remains M113 and must reuse the frozen M111 semantics.
4. **Rendered-GFM equality could hide Marksplice-specific source-position defects.** Mitigation: M112 compares parser-independent block observations/ranges rather than rendered HTML and keeps focused source-position tests in addition to the published corpus.
5. **A partial native parser could accidentally become the production backend too early.** Mitigation: `internal/splice` retains the single M111 Goldmark bridge; M112 exposes no production cutover, and M114/M115 remain mandatory.

## Verification

The final documented M112 tree passed focused native/differential regressions, the exact **676-example** native block projection, complete repository tests, repeated stability runs, actual race detection, `go vet`, `go build`, Staticcheck, golangci-lint, production `gocyclo <= 15`, production/test-inclusive unparam, `go mod tidy -diff`, published GFM conformance for the temporary production backend, Go 1.27 test/vet/build, govulncheck, Gitleaks, actionlint, strict text/artifact hygiene, and `git diff --check`.

Cross-package statement coverage measured **84.1% aggregate** and **84.7% through `internal/publictest`**. Direct native plus differential tests measured **94.7% native-candidate statement coverage**. The aggregate denominator includes the not-yet-production native candidate, while public API tests intentionally do not import that candidate.

## Exit decision

M112 is complete. The final documented-tree hygiene/state revalidation is green.

The Marksplice-native block parser now produces parser-independent source/range observations for every required block family and matches the temporary Goldmark oracle's M112 block projection on all 676 parser-applicable published GFM examples. No public API, existing-source mutation authority, construction output, production backend, or dependency version changed.

The roadmap boundary advances to **M113 — Marksplice-native inline parser**. Goldmark remains the temporary production backend and differential oracle until the later mandatory hardening/cutover/removal milestones.
