# M108 — Fuzzing, pathological input, and performance hardening

Status: complete.

## Objective

Establish reproducible robustness and resource evidence before public API stabilization and before Marksplice begins native-parser ownership.

M108 is measurement-first. It does not add syntax, public parser modes, hidden size/depth limits, persistent performance indexes, or M111 native-parser work. It expands fuzz/pathological coverage, benchmarks the M97–M107 intelligence surface, profiles measured hotspots, fixes Marksplice-owned pathological behavior, and records explicit scaling budgets for later regression comparison.

## Failure-surface inventory

The pre-M108 repository had two fuzz targets and no benchmarks:

- `FuzzParseProducesValidSourceRanges` covered the temporary Goldmark adapter's parser/source-coordinate boundary;
- `FuzzSinglePatchPreservesOutsideRange` covered source-patch byte preservation.

M108 adds two black-box fuzz targets:

- `FuzzM108PublicReadSurfacesRemainSourceBound` reparses arbitrary fuzzed input and requires bounded M97 queries plus M98/M99/M102–M106 read projections to expose only valid snapshot-local ranges/offsets;
- `FuzzM108NoopMutationsPreserveExactSource` exercises reviewed mutation families against fuzzed snapshots and requires every supported byte-identical mutation to apply back to the exact original source.

The deterministic pathological suite covers:

- a one-MiB paragraph;
- a one-MiB unclosed fenced block;
- 1,024 nested blockquote markers;
- dense mixed delimiter storms;
- malformed inline-link storms;
- 4,096 duplicate headings;
- 4,096 local-fragment links;
- 2,048 footnote definitions plus references.

These cases are operability/invariant tests, not arbitrary acceptance caps. Larger caller inputs remain permitted.

## Benchmark surface

M108 adds allocation/CPU benchmarks for:

- realistic mixed-syntax parsing at 64, 256, and 1,024 KiB;
- M97 node/section queries;
- M98 heading anchors, TOC generation, and duplicate-heading collision scaling;
- M99 relationship projection, including 1,024/4,096/16,384 dense relationships;
- M95 mutation planning and 1/4/16-change composition;
- M100 graph construction;
- M101 workspace validation;
- M102 alerts;
- M103 fenced blocks;
- M104 footnotes;
- M105 mathematical expressions;
- M106 front matter;
- M107 knowledge build, combined reachability, tag lookup, and exact alias lookup;
- pathological dense-delimiter and deep-blockquote parsing;
- raw Goldmark versus adapter-local delimiter/depth scaling so backend cost can be distinguished from Marksplice-owned overhead.

Raw benchmark/profile outputs are transient verification artifacts and are not repository dependencies.

## Performance budgets

Budgets are regression envelopes, not guaranteed service-level latency. Wall-clock absolute thresholds are intentionally not used as test pass/fail conditions because CPU scheduling, GC, host hardware, and toolchain changes make such tests flaky. For scaling families, a 4x input increase is compared against three-sample benchmark evidence unless noted otherwise.

The M108 budgets are:

- **realistic parse:** median CPU <= 6x and `B/op` <= 5x for each 4x source-size step;
- **dense delimiter parse:** median CPU <= 6.5x and `B/op` <= 5x for each 4x source-size step;
- **M98 duplicate heading anchors:** median CPU <= 6x and `B/op` <= 5x for each 4x heading-count step;
- **M99 dense relationship projection:** CPU and `B/op` should remain near-linear; repeated fixed-iteration evidence should stay <= 6x for each 4x relationship-count step;
- **M100/M101/M107 graph/workspace/knowledge operations:** median CPU <= 8x and `B/op` <= 7x for each 4x document-count step. The looser envelope covers GC/map growth and caller-owned result-slice capacity without authorizing an all-pairs algorithm;
- **mutation planning:** median CPU <= 5x for each 4x source-size step and allocation must remain approximately source-size independent for the benchmarked local paragraph operation;
- **M95 composition:** for two-or-more changes over a fixed document, 4x constituent count should remain <= 5x CPU and <= 5x allocation. The single-change fast path is intentionally excluded from this ratio. M95's O(k·N) semantic proof remains explicit and may be expensive for large caller-selected `k`; no hidden constituent cap is introduced;
- **deep blockquote temporary-backend boundary:** total depth scaling is not declared Marksplice-linear while Goldmark remains the production backend. At depth 4,096, full Marksplice parsing must not add more than 1.5x CPU or 1.1x allocation over the adapter-only measurement. M114 must establish the native-parser depth/resource budget before cutover.

These are review budgets. Benchmark data is retained as evidence and future material regressions must be investigated rather than mechanically widening the envelopes.

## Baseline measurements and findings

The first three-sample baseline exposed a severe Marksplice-owned delimiter pathology. Dense delimiter parsing at 16/64/256 KiB was approximately 22 ms / 236 ms / 3.70 s in the initial smoke measurements. A 256-KiB CPU profile attributed about 92.7% of CPU to repeated `physicalLineStart`/`physicalLineEnd` scans, with 93.2% cumulative time under `MapSimpleCodeSpan`.

After the focused fix, three-sample dense-delimiter measurements are approximately:

| Input | Median CPU | `B/op` | `allocs/op` |
| --- | ---: | ---: | ---: |
| 16 KiB | 6.37 ms | 11,915,977 | 32,261 |
| 64 KiB | 36.83 ms | 50,023,088 | 127,530 |
| 256 KiB | 156.63 ms | 209,898,032 | 508,572 |

The pathological 256-KiB case is therefore roughly 24–30x faster than the pre-fix profile/smoke results, depending on the compared run, and returns to near-linear scaling.

Realistic mixed-syntax parsing also exposed avoidable copying. The initial 1-MiB baseline allocated about 657 MB/op. Profiles identified repeated full `Node` merges, full-node section-heading copies, and two parser-observation conflict filters as dominant Marksplice-owned copies. After the M108 refactors the three-sample final baseline is about 374 MB/op with 906k allocations/op, a reduction of roughly 43% in allocated bytes while preserving the same parse model.

Final realistic parse scaling is approximately:

| Input | Median CPU | `B/op` |
| --- | ---: | ---: |
| 64 KiB | 17.17 ms | 21,187,288 |
| 256 KiB | 91.75 ms | 92,058,912 |
| 1,024 KiB | 431.75 ms | 374,063,808 |

The remaining dominant allocation is the temporary Goldmark observation/model material plus the authoritative retained structural model, not another duplicate full-model merge. M108 deliberately does not redesign the complete internal parser observation contract solely to optimize the temporary backend before M111–M115.

M98 duplicate heading anchors remain near-linear: 1,024 / 4,096 / 16,384 identical headings measure about 0.35 / 1.50 / 7.01 ms median with approximately 0.17 / 0.72 / 2.90 MB allocated.

M100/M101/M107 benchmarks show expected graph-order scaling. Exact M107 alias lookup remains approximately constant at 9–11 ns/op with zero allocation from 64 through 1,024 documents, so M108 does not add another alias index. Tag lookup remains an explicit scan with caller-owned result allocation; the measurements do not justify a persistent tag-to-document index.

M95 composition remains intentionally expensive but scales with constituent count rather than quadratically in `k`: over the fixed 256-KiB benchmark document, four changes measure roughly 0.62–0.71 s / 598 MB and sixteen changes roughly 2.22–2.50 s / 1.98 GB. The 4x `k` step is therefore about 3.5x CPU and 3.3x allocation. M108 keeps the fail-closed independently-derived semantic-delta proof rather than weakening it or moving equivalent full-document work into ordinary mutation preparation.

## Implementation hardening

### Code-span line-rescan elimination

`MapSimpleCodeSpan` previously searched backward/forward to the physical-line bounds for every parser-proven code span, even though a backtick run itself cannot cross CR or LF. Dense same-line syntax therefore rescanned the same long line once per code span and became quadratic.

M108 replaces those whole-line searches with local delimiter-run checks bounded by the source length. Immediate preceding-byte validation still rejects an anchor inside a larger opener run, including exact LF, CRLF, and isolated-CR line-start regressions. No line index, cache, retained state, or source cap is needed.

### In-place supplemental-node merge

`splice.Parse` preallocates node capacity for the ordinary observations plus supplemental math/footnote nodes, but `mergeSourceOrderedNodes` previously allocated and copied a complete new `[]Node` for each supplemental stream. M108 uses `slices.Grow` plus a backwards stable merge, reusing the already-reserved backing array when available and retaining the historical base-before-addition order for equal source starts. Insufficient capacity still grows safely.

### Lightweight section-heading projection

Section construction previously copied full internal `Node` values merely to retain heading ID, level, and range. M108 introduces a private three-field `sectionHeading` projection for that build-local pass. The authoritative node model and public `Section` semantics are unchanged.

### In-place parser conflict filters

Math and footnote conflict suppression previously copied the complete parser-node observation vector even though each filter is a terminal source-order-preserving compaction. M108 compacts those slices in place and clears removed tails so pointers/slices from suppressed observations are not accidentally retained. No parser observation is exposed or mutated after the filtered result becomes authoritative.

### Paragraph/image no-op correctness

The new no-op mutation fuzz seed exposed a correctness bug independent of performance: a top-level paragraph containing a supported inline image was rejected even when the replacement bytes were identical. `validateParagraphReplacement` admitted the other reviewed nested inline observations but omitted `parser.KindImage`. The focused fix adds images to the compatible nested-inline family; the RED fuzz seed becomes GREEN without widening paragraph replacement to new block shapes.

## Temporary backend depth finding

Deep blockquotes are superlinear before Marksplice source mapping. At depth 4,096, representative measurements were approximately:

- raw Goldmark: 16.7 ms / 1.29 MB;
- Marksplice adapter: 34.0 ms / 2.55 MB;
- complete Marksplice parse: 33.8 ms / 2.56 MB.

Because the full Marksplice layer adds negligible overhead over the adapter at that depth, M108 does not introduce an arbitrary depth limit or copy/fork Goldmark behavior. The backend limitation is explicit evidence for the future native block-parser/resource gate; M112/M114 must not reproduce it silently.

## Devil's advocate review

### Risk 1 — benchmark thresholds become hardware-dependent CI flakes

Mitigation: budgets use input-scaling ratios and allocation counts rather than absolute wall-clock pass/fail tests. Raw numbers are evidence, not portable latency promises.

### Risk 2 — a performance fix weakens source ownership

Mitigation: the code-span optimization removes only redundant searches for line boundaries that delimiter runs cannot cross. Focused LF/CRLF/CR tests preserve the larger-run rejection contract, and ordinary candidate/source proof remains unchanged.

### Risk 3 — in-place compaction retains removed semantic data

Mitigation: compacted parser slices explicitly clear their unused tails. Supplemental node merging preserves stable source order and has separate reuse/fallback tests.

### Risk 4 — a hidden resource cap masks backend denial-of-service behavior

Mitigation: M108 adds no parse/document/depth/global relationship cap. Pathological cases remain accepted, measured, and regression-tested. The deep-blockquote Goldmark limitation is documented rather than hidden behind an arbitrary rejection threshold.

### Risk 5 — a new persistent index trades one benchmark win for long-lived memory

Mitigation: measurements do not justify new M97 query, M98 navigation, M99 relationship, M102–M106 semantic, or M107 tag indexes. Existing ephemeral maps and M107's already-retained exact alias map remain the chosen balance.

### Risk 6 — composition optimization could invalidate cross-operation safety

Mitigation: M108 records the O(k·N) cost and scaling budget but keeps M95's independently validated semantic deltas plus final combined candidate proof. A future redesign must preserve equivalent evidence rather than merely making the benchmark faster.

## Verification

M108 exercised the new public-read and no-op-mutation fuzz targets, the deterministic malformed/deep/oversized pathological suite, and the complete benchmark surface. The fuzz campaign completed thousands of bounded executions without a product failure after the discovered paragraph/image no-op bug was fixed and permanently regressed.

Profile-guided regressions confirmed the code-span line-rescan fix, in-place node merge, lightweight section-heading projection, and in-place parser conflict filters. Scaling measurements also covered realistic parsing, dense delimiters, duplicate headings, dense relationships, graph/workspace/knowledge operations, mutation planning, composition, and deep blockquotes.

The final tree passed repeated complete regressions, race detection, `go vet`, `go build`, Staticcheck, golangci-lint, production `gocyclo <= 15`, production/test-inclusive unparam, `go mod tidy -diff`, published GFM conformance, Go 1.27 test/vet/build, govulncheck, Gitleaks, actionlint, strict text/artifact hygiene, and `git diff --check`.

Cross-package statement coverage measured **87.0% aggregate** and **84.6% through `internal/publictest`**.

## Exit decision

M108 is complete. The repository now has reproducible fuzz/pathological coverage, explicit performance/resource regression envelopes, and profile-driven hardening evidence for the current Goldmark-backed architecture without hidden caps or speculative persistent indexes. The next roadmap boundary is **M109 — Public API coherence and stabilization review**; native-parser ownership still does not begin until M111.
