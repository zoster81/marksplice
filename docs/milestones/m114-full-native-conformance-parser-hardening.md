# M114 - Full native conformance and parser hardening

Status: **Complete**

## Goal

Complete and harden the Marksplice-native `parser.Backend` before the explicit M115 production cutover. M114 does not change the public API, source-preservation model, M110 extension boundary, or production parser selection.

## Specification-first protocol

M114 uses the normative hierarchy in [`../gfm-conformance.md`](../gfm-conformance.md): CommonMark 0.31.2 first, explicit GFM extensions/corrections second, reviewed Marksplice contracts third, and Goldmark only as temporary implementation evidence.

The historical fuzz rounds R1-R59 are closed as discovery history, not a parallel conformance protocol. A retained round input has semantic authority only when CommonMark/GFM or a reviewed Marksplice contract independently defines it. Every other historical input remains only a Native source-bound/determinism invariant. R50-A and R50-D are covered by focused CommonMark-first regressions and are no longer unclassified.

## Full corpus conformance

Marksplice has no HTML renderer, so the CommonMark gate is two-stage:

1. the hash-pinned CommonMark 0.31.2 reference audit passes all **652** official expected-HTML examples;
2. Native matches the complete parser-neutral `DocumentObservations` contract on those same **652** Markdown inputs.

The explicit GFM gate passes all **27** parser-side extension examples. Native also matches all **676** parser-applicable examples from the published GFM corpus as compatibility evidence; `tagfilter` remains rendering-only.

No implementation change may be justified only because Goldmark differs.

## Backend and fuzz completion

Native implements the complete M111 backend contract: document observations, nested blockquote proof, typed-inline hierarchy proof, direct link/image proof, reference-inline proof, construction reference resolution, and reference-label normalization.

The final fuzz roles are explicit:

- `FuzzM114NativeBackendObservationsRemainSourceBound`: arbitrary-input source/range safety;
- `FuzzM114NativeBackendLegacyDifferentialCorpusRemainsSourceBound`: R1-R59 historical corpus as deterministic invariants;
- `FuzzM114NativeDirectLinkProofAcceptanceParity`: frozen direct-link construction acceptance;
- `FuzzM114NativeReferenceProofAcceptanceParity`: frozen reference construction acceptance.

The last two use Goldmark only as a secondary comparator for already-reviewed construction contracts.

## Performance and refactor

M114 shares one per-block inline analysis, delays heavy inline-owner materialization, uses source-ordered indexes, adds embedded single-segment fast paths for inline/delimiter indexes, replaces reflection-based `sort.Slice*` in hot Native paths with typed `slices` sorting, and removes superseded helpers/trivial sorts.

Representative realistic measurements on the recorded development host:

| Size | Hardened Native | Temporary production path |
| --- | ---: | ---: |
| 64 KiB | ~11.3-12.7 ms / 13.65 MB / 56.8k allocs | ~15.6-17.2 ms / 21.21 MB / 59.9k allocs |
| 256 KiB | ~49.2-54.2 ms / 54.60 MB / 224.0k allocs | ~60.4-78.6 ms / 92.12 MB / 233.8k allocs |
| 1 MiB | ~179.7-203.0 ms / 214.01 MB / 876.0k allocs | ~329.6-343.8 ms / 374.33 MB / 912.5k allocs |

Native 1 MiB allocations fell from roughly 1.476 million at the start of the profiling pass to roughly 876 thousand. The final pre-M115 profile also reserves block-node capacity in the multi-analysis inline result so the already-ordered final document merge reuses that backing array instead of allocating a second full node vector. On the measured host the recorded 1 MiB Native path is about 1.8x faster and uses about 43% less memory than the still-production path. No persistent parser cache/index or hidden syntax cap was introduced.

## Real-world corpus hardening

A broad Native-only corpus campaign exercised exact Git-tracked Markdown bytes from unrelated open-source repositories without executing third-party project code. The first expansions exposed and fixed two parser-owned repeated reference-definition normalization paths with O(B*D) behavior; discovery of either defect reset the stability counter.

After those fixes, three consecutive independent expansion waves completed without another parser-owned defect. The final accumulated corpus contained **195 repositories / 6,857 Markdown files / 60,778,570 bytes**, with every extracted file verified against its recorded Git blob. All files passed source immutability, source-bound observation, and double-parse determinism checks, and all **384** deterministic malformed derivatives passed. After the final backing-array reuse refactor, representative aggregate Native runs measured about **18.8-19.2 MB/s**, with about **2.535 GB/op** allocated and **10.304 million allocs/op** over the complete corpus; allocation density is about **41.7 allocated bytes per input byte**. CPU/allocation profiles and largest-file benchmarks showed no new unexplained parser-owned amplification, and normalized-time outliers remained small files below the absolute review threshold.

The three-stable-wave stop condition is therefore satisfied. This corpus remains regression evidence; M114 does not keep expanding it merely to increase repository count.

## Coverage and maintainability

Measured statement coverage is **86.3% aggregate**, **84.7% through `internal/publictest`**, and **83.8% directly in `internal/parser/native`**. Production cyclomatic complexity remains at or below 15 per function and both `unparam` modes are clean.

## Devil's advocate review

1. Full-corpus differential evidence must not restore Goldmark as normative. The official CommonMark output and GFM extension corpus are checked first; mismatches are classified against the documented hierarchy.
2. Historical R-round fixes must not survive as undocumented Goldmark quirks. R1-R59 are discovery history; invariant-only tests no longer claim Goldmark parity.
3. Performance fast paths must not weaken source ownership or stable ordering. Dedicated index tests plus full conformance/fuzz/regression gates cover the refactors.
4. M114 must not cut over early. Production remains on Goldmark until M115.

## Verification

The final M114 tree passes the 652-example CommonMark chain, 27 explicit GFM examples, 676 parser-applicable GFM compatibility examples, focused Native/source regressions, all four classified fuzz targets, repeated full tests, actual CGO/GCC race detection, vet/build/static/lint/complexity/unparam checks, isolated Go 1.27 and cross-build checks, vulnerability/secret/workflow checks, corrected explicit-package coverage, the three-wave real-world corpus gate above, module/text/artifact/Git hygiene, and complete diff review. Corrected final statement coverage is **86.3% aggregate**, **84.7% through `internal/publictest`**, and **83.8% directly in `internal/parser/native`**.

Host-dependent benchmark times are evidence rather than CI thresholds; allocation and scaling behavior remain the more stable resource signal.

## Exit decision

M114 is complete. The Native parser satisfies the pre-cutover conformance, construction-proof, fuzz, real-world corpus, resource, performance, cross-platform, security, and maintainability gates while preserving Marksplice source/public/M110 boundaries.

M115 - Goldmark removal and final cutover - is the next and final parser-roadmap boundary.
