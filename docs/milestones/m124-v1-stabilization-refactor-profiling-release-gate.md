# M124 — v1.0 Stabilization, Refactor, Profiling, and Release Gate

Status: **complete and remotely closed on 2026-09-16; freeze commit `dad38c7ee6a793abce31c4f0ea7c4efe011943c7` passed the exact 7-job public CI matrix.**

## Goal

M124 freezes the first stable Marksplice contract after the M116–M123 workspace and rendering line. It adds no feature surface by default: changes must be justified by correctness, boundedness, measured performance, maintainability, API stability, documentation quality, or release readiness.

The reviewed starting point is the remotely closed M123 commit `9c144656cbe68726494531eb5d0782846000906c` on `main` with `origin/main` identical and a clean working tree.

## Required review

M124 covers the complete repository rather than one feature family:

- source, public API, dependency, architecture, reuse, dead-code, duplication, and complexity audit;
- CPU/allocation profiling of parse, representative edits, graph/workspace validation, `workspacefs` scan/follow, semantic walk, HTML, source mapping, and canonical Markdown;
- pathological, oversized, and deeply nested input review with explicit resource-bound checks;
- parser/source/rendering fuzzing where appropriate;
- CommonMark/GFM parser, semantic, HTML, and canonical conformance;
- retained real-world corpus regression;
- race, supported-Go, cross-platform, static/lint/security/dependency/documentation verification;
- public API compatibility and documentation/API-reference/example audit;
- release notes, exact-tree hygiene, and v1.0 release-readiness review.

The stable release remains a separate publication action. M124 may be considered complete only after its exact reviewed freeze commit is pushed and public CI is green for that SHA.

## Initial audit and profiling

The opening baseline confirms that `go.mod` retains one direct production dependency, `golang.org/x/text`. The initial Go 1.26.6 gate passed the full package tests, `go vet`, `go build`, Staticcheck, golangci-lint, production `gocyclo <= 15`, production/test-inclusive `unparam`, `govulncheck`, Gitleaks, and `git diff --check`.

Representative same-host measurements cover public parse, edit planning, graph/workspace validation, semantic walk, HTML, source mapping, canonical Markdown, and the retained private `workspacefs` scan/follow scaling harness. Workspace scan/follow remains approximately proportional over the reviewed 256-to-1024-document scaling interval, so no persistent filesystem cache or secondary workspace index is justified.

The first 256 KiB Native/public parse profile identified footnote work as a measurable parser cost. The most conservative accepted opening refactor targets only fenced-code membership during footnote-definition filtering: the previous implementation compared every candidate definition with every fenced-code content range. M124 now flattens and normalizes those already parser-proven ranges once, then uses ordered interval lookup for each candidate. The recognized fenced-code ownership is unchanged.

The ordinary 256 KiB realistic benchmark remains within same-host noise and keeps the same allocation count; the refactor adds one operation-local compact range vector when both footnote candidates and fenced content exist. A dedicated pathological gate with 64/256/1024/4096 fenced footnote-like regions measures approximately proportional time and allocation growth after the change, avoiding the previous nested candidate-by-range lookup shape.

The larger footnote-reference scanner also redoes inline analysis that overlaps with the ordinary parse. M124 does not merge those paths speculatively: the specialized pass intentionally removes footnote definitions from ordinary reference resolution, so reusing an analysis computed under a different definition set could change nested link/reference ownership. Further reuse requires equivalence evidence rather than a benchmark-only shortcut.

## Recertification and resource review

After the fenced-range refactor, the working tree passed focused Native footnote/fenced-code regressions, the complete package suite with the approved CommonMark 0.31.2 and published-GFM snapshots provisioned, and the retained real-world corpus suite. A full race run over that same conformance/corpus-enabled tree also passed.

The original authoritative bounded fuzz gate used Go 1.26.6 explicitly and exercised all eight permanent parser/source/public hardening targets: Native observation/source bounds, legacy differential-corpus source bounds, direct/reference construction proof stability, semantic-walk source bounds, public read surfaces, source-preserving no-op mutations, and single-patch outside-range preservation. After the release-toolchain refresh, all eight permanent targets were rerun under Go 1.27.1 and again completed without a product failure.

The release toolchain is now locally retained and versioned with Go 1.27.1 as the primary compiler and Go 1.26.8 as the compatibility compiler. The complete package/specification/real-world-corpus suite, module verification, vet, and build pass under both lines. The full CGO/GCC race suite passes under Go 1.27.1. CGO-disabled cross-builds pass under both Go 1.27.1 and Go 1.26.8 for linux/amd64, darwin/amd64, and darwin/arm64. Final explicit cross-package coverage over the current eight-package production set reports **86.4% aggregate**, **79.4% through the black-box `internal/publictest` suite**, and **82.8% direct Native** statement coverage; all temporary private coverage profiles were removed after reporting. Public CI keeps `1.26.x` and `1.27.x` with `check-latest: true`; the exact pushed freeze commit `dad38c7ee6a793abce31c4f0ea7c4efe011943c7` passed all seven required jobs in run `35128934025`, closing M124 remotely.

A consolidated three-sample same-host benchmark matrix confirms the existing resource envelopes rather than exposing a new optimization target. The matrix was rerun after the Go 1.27.1 / `golang.org/x/text v0.42.0` refresh; allocation profiles remain aligned with the earlier M124 measurements, with no new regression requiring an implementation change:

- realistic public parsing scales from 64 to 256 to 1024 KiB with approximately proportional CPU/allocation growth; the post-refresh 256 KiB case remains about 53.31 MB/op and 233k allocations/op, with representative runs around 39–42 ms on the recorded host;
- representative local mutation planning remains approximately linear in source size and uses **1 allocation / 24 B per operation** across 64/256/1024 KiB;
- graph construction, workspace validation, and knowledge-index operations grow approximately with document count from 64 to 256 to 1024 documents, while exact alias lookup remains allocation-free and effectively constant-time;
- the retained dense-delimiter pathological parse stays near-linear across 16/64/256 KiB, and deep-blockquote allocation grows approximately proportionally with depth;
- Native realistic parse scales from about 9.43 MB/op at 64 KiB to 38.41 MB/op at 256 KiB and 151.77 MB/op at 1024 KiB, with allocation counts scaling similarly; the post-refresh 256 KiB runs are around 31 ms on the recorded host;
- the footnote reference-definition workload scales from 64 definitions / 256 blocks to 256 definitions / 1024 blocks without allocation amplification beyond the expected approximately 4x input growth;
- 256 KiB HTML, source-map, semantic-walk, and canonical-Markdown allocation profiles remain aligned with their M120–M123 freeze boundaries: streaming HTML is about 41.85 MB/op, mapped fragment HTML about 44.56 MB/op for 37,725 mappings, semantic walk about 40.80 MB/op, and canonical streaming about 44.05 MB/op;
- the retained private `workspacefs` scaling harness was updated only for the current `x/text` module graph and rerun under Go 1.27.1: 256-to-1024-document allocation growth remains approximately 4x, with same-host median timing ratios of roughly 4.9x for `Scan`, 4.0x for chained `Follow`, and 4.3x for dense `Follow` over 4x input.

Wall-clock figures remain host-sensitive engineering evidence, not public service-level guarantees. No persistent cache, secondary index, hidden source-size cap, or benchmark-specific shortcut is justified by the M124 measurements.

## API, dependency, authority, and documentation audit

`go.mod` still has one direct production dependency, `golang.org/x/text`, refreshed from v0.41.0 to v0.42.0; the existing `NOTICE` attribution remains accurate and the compatibility floor remains Go 1.26. The active release toolset was refreshed before the v1 gate: Go 1.27.1 primary, Go 1.26.8 compatibility, govulncheck 1.8.0, Staticcheck 2026.2.1/v0.8.1, golangci-lint 2.13.2, current `unparam`, gocyclo 0.6.0, actionlint 1.7.12, Gitleaks 8.30.1, GitHub CLI 2.101.0, Node.js 26.8.2 with npm 12.0.2, and w64devkit 2.9.1/GCC 16.2.0. Staticcheck, golangci-lint, production `gocyclo <= 15`, production/test-inclusive `unparam`, `go mod tidy -diff`, `govulncheck`, Gitleaks, actionlint, `git fsck`, and `git diff --check` are green on the reviewed M124 tree.

The root public API and `workspacefs` public source files are unchanged from the remotely closed M123 baseline. Documentation dogfood covers **374/374 exported callables**, and M124 introduces no exported callable, additional production dependency, capability, `Kind`, error family, filesystem/network/command authority, or compatibility-driven signature change.

The full source review found a small amount of internal chronology debt in production comments. Those comments now describe current contracts rather than old milestone transitions; no code path changed. The normal user guide likewise no longer mentions an internal milestone. Example release prose now says “next release” rather than “next beta”; the v1 release-state commit updates installation/status documentation to `v1.0.0`.

The release policy now explicitly covers stable v1 semantics and publication: Go 1.26 remains the current compatibility floor, v1+ compatibility follows Semantic Versioning, an intentionally breaking public API requires the next major/module-path decision, release/tag actions remain separately authorized, exact commit and tag CI must be green, prereleases and stable GitHub releases are distinguished, and proxy/pkg.go.dev plus an external consumer without `replace` are post-publication checks.

No exported callable, additional production dependency, capability claim, or authority boundary changes in this M124 slice.

## Devil's-advocate review

1. **A broad cleanup can silently change v1 semantics while appearing simpler.** Refactors remain evidence-driven and small; Native changes require focused tests plus published/corpus recertification rather than relying on implementation equivalence by inspection.
2. **Microbenchmarks can reward code that is worse on real documents.** Allocation state, representative workloads, scaling behavior, and profiler attribution are reviewed together. A local timing win alone is not sufficient justification.
3. **Reuse across parser passes can conflate different semantic contexts.** In particular, footnote reference resolution intentionally differs from the ordinary reference-definition set. Shared intermediate state is accepted only when its semantic inputs are proven equivalent.
4. **v1 cleanup can accidentally become an API redesign.** Public signatures remain frozen unless a concrete compatibility defect justifies change before v1; documentation and examples must be audited against the exact final callable inventory.

## Remaining work

M124 is complete. Implementation/refactor, dependency/toolchain refresh, API freeze, conformance/corpus, race, fuzz, dual-Go compatibility, cross-build, post-refresh performance, coverage, documentation dogfood, external-consumer, and strict text/private-boundary/module/Git hygiene are green. The freeze tree was committed, integrated with the matching Dependabot `x/text v0.42.0` update without changing its tree hash, pushed as `dad38c7ee6a793abce31c4f0ea7c4efe011943c7`, and closed by the exact public 7-job CI matrix. The separately authorized stable `v1.0.0` publication proceeds from a release-state documentation commit only after that commit also passes public CI.