# M115 - Goldmark removal and final cutover

Status: **Complete**

## Goal

Complete the evidence-gated parser replacement by switching every production parsing and construction-proof path to the Marksplice-native `parser.Backend`, preserving the accepted parser-neutral contract, and removing `github.com/yuin/goldmark` from active code, tests, and the module dependency graph.

M115 changes parser implementation ownership, not the public Markdown profile, source-preservation model, public API, M110 extension SPI, or ordinary mutation/construction contracts.

## Requirements and edge cases

The cutover must preserve all reviewed parser-independent observations and construction proofs while avoiding three failure modes that a simple dependency deletion could hide:

- parser behavior accepted only because the old and new implementations happened to agree rather than because the specification or a reviewed Marksplice contract required it;
- loss of durable conformance evidence after deleting the historical comparator;
- public/source regressions that were not represented in the published specification corpus.

Production and test code must contain no dependency on Goldmark after the final removal. Parser-neutral fixtures must remain tied to the approved external CommonMark/GFM inputs rather than becoming self-generated Native snapshots. Source immutability, source-bound ranges, deterministic parsing, malformed-input handling, construction proof, cross-platform behavior, and the M110 parser-independent extension boundary remain unchanged.

## Architecture and test strategy

M115 uses the parser-independent M111 `Backend` boundary as the only production substitution point. `internal/splice` selects `native.New()` and Native reference-label normalization; no parser-specific type enters retained Marksplice state or public APIs.

The removal sequence is deliberately staged:

1. switch production parsing and construction proof to Native;
2. run black-box/public regressions with Native actually selected;
3. freeze Marksplice-owned parser-neutral expected observations for the approved CommonMark and GFM corpora, binding every entry to official example identity and a SHA-256 of the externally provisioned Markdown input;
4. while the historical comparator still exists, prove both the frozen fixtures and the previously accepted transition-time comparison gates;
5. only then delete the Goldmark adapter, differential harness, compatibility implementation, and parity-only tests;
6. remove the module dependency with `go mod tidy` and recertify the resulting Native-only tree.

This sequencing prevents the current Native implementation from silently becoming its own oracle. Future fixture changes require reviewed specification/contract evidence; serializing current Native output is not an acceptable update procedure.

## TDD cutover regression

The first production-Native black-box run exposed one reviewed public table contract that was not represented by the published corpus: when a valid GFM table header immediately follows an open paragraph line, that trailing line must become the table header rather than remain part of the paragraph.

A focused Native regression was added first, the expected failure was confirmed, and the smallest block-parser fix split the trailing paragraph line when the following delimiter row proves a table. Focused Native/public tests and the complete regression suite then passed with production still selecting Native.

## Implementation

M115 completes the following changes:

- production `internal/splice` uses the Native backend and Native reference-label normalization;
- the former `internal/parser/goldmark` implementation is removed;
- the former `internal/parser/differential` transition harness is removed;
- parity-only Native tests are replaced by explicit construction contracts and Native determinism/input-immutability hardening;
- tracked parser-neutral CommonMark/GFM expected-observation fixtures live under `internal/parser/native/testdata` without vendoring upstream specification corpus bytes;
- `github.com/yuin/goldmark` is removed from `go.mod`, `go.sum`, and the reachable `go list -deps ./...` graph;
- active code/comments and current documentation describe Native as the production implementation while historical milestone/transition records retain Goldmark evidence in past tense;
- `golang.org/x/text` remains the direct parser-owned dependency required for full Unicode GFM reference-label case folding.

## Devil's advocate review

1. **Deleting the historical oracle could weaken future conformance.** The mitigation is a specification-first Native gate over Marksplice-owned parser-neutral fixtures bound to the approved external corpus input hashes. The official CommonMark/GFM snapshots remain external, hash-pinned validation input.
2. **Frozen fixtures could become tautological if regenerated from Native.** Fixture updates are explicitly review-gated: changed specification input or a reviewed Marksplice contract must justify the expected observation change. Current Native output alone is insufficient evidence.
3. **Published corpora do not cover every public source-ownership contract.** The production cutover was tested through the root and black-box public suites before deleting the comparator; that step found the table-after-paragraph regression and converted it into a focused Native test.
4. **A removed direct dependency could survive through tests or transitive imports.** M115 checks `go.mod`, `go.sum`, active source imports, and the complete `go list -deps ./...` graph after removal.
5. **Removing parity fuzzing could reduce malformed-input protection.** Historical differential inputs remain Native source-bound/determinism invariants, and the real-world corpus plus deterministic malformed derivatives continue to exercise panic/error, immutability, range, and repeatability properties without relying on another parser.

## Final verification

The final Native-only M115 code tree passes:

- the hash-pinned CommonMark 0.31.2 loader and all **652** Native parser-neutral CommonMark contracts;
- all **676** parser-applicable published-GFM Native contracts; rendering-only `tagfilter` remains outside parser conformance;
- **5** consecutive complete `go test ./... -count=1` regressions;
- actual race detection on Go 1.26.6 / Windows amd64 with CGO enabled and GCC;
- the accumulated exact-byte-certified Native-only real-world corpus of **195 repositories / 6,857 Markdown files / 60,778,570 bytes**, plus all **384** deterministic malformed derivatives;
- `gofmt`, `go vet`, `go build`, `staticcheck`, `golangci-lint`, production cyclomatic complexity at or below 15, and production/test-inclusive `unparam`;
- `go mod tidy -diff` and `git diff --check`;
- `govulncheck`, `gitleaks`, and `actionlint`;
- isolated Go 1.27.0 test/vet/build plus CGO-disabled cross-builds for Linux amd64, macOS amd64, and macOS arm64;
- corrected statement coverage of **87.6% aggregate**, **83.2% through `internal/publictest`**, and **82.0% directly in `internal/parser/native`**;
- no reachable `github.com/yuin/goldmark` package in the final dependency graph.

Host-dependent benchmark times remain evidence rather than CI thresholds. M114 retains the detailed pre-cutover Native resource/performance baseline and real-world profiling record; M115 changes selection/removal rather than introducing a new parser-performance algorithm.

## Exit decision

M115 is complete. Production parsing and construction proof are Native-only, the public/source/M110 contracts remain unchanged, the specification-first conformance evidence survives removal of the historical comparator, and Goldmark is no longer an active implementation or module dependency.

The M111-M115 parser-replacement roadmap is therefore closed. Future parser changes are ordinary maintenance of the Marksplice-owned Native implementation under the established conformance, source-preservation, fuzzing, real-world corpus, performance, maintainability, security, and cross-platform gates.
