# Marksplice Roadmap

Status: approved post-M115 development roadmap.

This document records the planned product and engineering sequence after the Native parser cutover. It is maintainer-facing: current user-visible capability remains owned by [`capabilities.md`](capabilities.md), while durable design decisions remain owned by [`architecture.md`](architecture.md).

## Release targets

The roadmap has three explicit release horizons:

- **Pre-M116 performance campaign -> `v0.5.0-beta.1`**. This beta is dedicated to parser/document-model throughput, allocation, and memory improvements before new product surface is added. Additional `v0.5.0-beta.N` iterations are allowed only when further measured optimization or correctness work justifies them.
- **M116–M124 -> `v1.0.0`**. M124 is the v1.0 stabilization, performance, API-review, and release-readiness gate.
- **M125–M126 -> `v1.5.0`**. PDF work is intentionally queued after v1.0 and must not delay the first stable release.

A milestone number is an engineering boundary, not a public API version. A release is published only from an exact reviewed commit that passes the applicable local and public CI gates.

## Operating model

Every implementation milestone follows the same closure discipline:

1. verify branch, `HEAD`, `origin/main`, and a clean/understood working tree;
2. restate requirements, edge cases, authority boundaries, compatibility, and expected complexity;
3. define the architecture and focused test strategy before implementation;
4. establish focused TDD evidence where practical;
5. implement the smallest coherent change without bypassing Native parsing or source-preservation invariants;
6. perform a devil's-advocate review for corruption, traversal, nondeterminism, memory/CPU amplification, API leakage, and regressions;
7. run focused tests, relevant regressions, and milestone-specific conformance/performance checks;
8. perform a refactor pass for reuse, dead/redundant code, complexity, allocation, and responsibility boundaries;
9. update every affected source-of-truth, user guide, recipe, example, and API reference surface;
10. review the complete diff and run the applicable release-quality gates;
11. create one coherent milestone freeze commit only after the documented tree is green;
12. push only a reviewed milestone freeze, then verify the remote ref and public CI for that exact commit before treating the milestone as remotely complete.

Small adjacent milestones may share one commit only when they form one coherent change and both exit criteria are complete. Do not accumulate unrelated unfinished milestone work into a large catch-all commit.

## Refactor and profiling cadence

Refactoring is part of development, not deferred cleanup.

- **Pre-M116 / v0.5:** dedicated whole-parse-path performance campaign before feature development resumes. Prioritize compact internal node/storage layout, Native inline candidate scanning, parse/model allocation and copy reduction, then follow new profiler evidence rather than speculative rewrites.
- **Every milestone:** focused refactor before freeze; remove duplication/dead state, keep responsibilities narrow, and keep production cyclomatic complexity at 15 or lower per function.
- **M117:** first broad post-M115 review of filesystem discovery/resolution code, including traversal scaling and allocation behavior.
- **M119:** broad review of Native semantic-walk changes to prove there is no second parser or permanently retained rendering AST.
- **M122:** renderer/source-mapping refactor and profiling checkpoint after the complete HTML path exists.
- **M123:** focused canonical-Markdown writer profiling plus semantic round-trip/idempotence hardening.
- **M124:** full repository refactor, profiling, allocation/CPU review, pathological-input review, API stability audit, documentation audit, and release-quality recertification before v1.0.

Performance conclusions must come from measurements. Avoid persistent caches/indexes or architectural complexity merely to optimize an unmeasured path.

## Pre-M116 — `v0.5.0-beta.1` performance campaign

**Status: released as `v0.5.0-beta.1` on 2026-08-28.**

The campaign materially reduced parser/document-model CPU, allocation count, and allocated bytes without adding product surface, reintroducing Goldmark, or weakening source-preservation, stale-source, conformance, or authority-boundary contracts.

The frozen comparison workload is the byte-certified real-world corpus of 6,857 Markdown documents / 60,778,570 bytes from 195 repositories, preloaded in memory so timed results exclude filesystem I/O. Five complete same-host runs on the final tree measured:

| Path | Baseline throughput | Final throughput | Baseline B/op | Final B/op | Baseline allocs/op | Final allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `marksplice.Parse` | 15.04 MB/s | **25.06 MB/s** | ~4.49 GB | **~2.702 GB** | ~13.05 M | **~10.402 M** |
| Native parser only | 19.23 MB/s | **30.87 MB/s** | ~2.54 GB | **~1.902 GB** | ~10.62 M | **~9.015 M** |

That is approximately **+66.6%** public-Parse throughput, **+60.5%** Native throughput, **-39.8%** public allocated bytes, **-25.1%** Native allocated bytes, **-20.3%** public allocations, and **-15.1%** Native allocations relative to the frozen V05-A baseline. These are same-harness engineering ratios, not cross-machine performance promises.

The accepted implementation keeps the public API and source-ownership model intact while compacting broad internal node/detail storage, reducing temporary parse/model state, adding measured Native scanner/index fast paths, removing repeated source-mapping work, and preserving source-ordered/monotonic algorithms instead of adding persistent caches. Fuzzing during the freeze also found three identity-mutation edge cases; byte-identical replacements of already source-proven content now produce snapshot-bound no-op `ChangeSet`s while genuinely new invalid payloads remain rejected.

All six hard V05 performance gates are satisfied. The retained M108 pathological/scaling matrix has no unexplained regression greater than 10%; all 6,857 certified real-world documents parse through the public API; exact CommonMark 0.31.2 and published-GFM contracts pass; bounded fuzzing, malformed derivatives, race, vet/build, Staticcheck, golangci-lint, `gocyclo <= 15`, `unparam`, vulnerability/secret/workflow checks, Go 1.26/1.27 verification, and Linux/macOS cross-build gates are green on the freeze candidate.

Goldmark v2 remains only a private external comparison and is not a production dependency, semantic oracle, or release requirement. The campaign's stretch throughput targets of 30 MB/s public and 35 MB/s Native were intentionally not used to delay the beta once the hard gates were exceeded cleanly.

M116 is the first engineering boundary after the `v0.5.0-beta.1` performance release. It is not part of the v0.5 release.

## M116 — Filesystem workspace foundation

**Status: complete on 2026-08-28; unreleased.**

**Goal:** make the existing explicit multi-document graph practical for documentation already stored in a caller-authorized filesystem, without moving filesystem authority into Marksplice core.

M116 introduced the separate `workspacefs` package over an explicit `fs.FS` supplied by the caller.

Its foundation responsibilities are:

- discover Markdown documents under an authorized filesystem root;
- load and parse discovered documents through ordinary `marksplice.Parse`;
- assign deterministic slash-relative document keys;
- resolve reviewed local Markdown relationships;
- build the existing `DocumentGraph`/workspace inputs;
- expose explicit document, byte, depth, and relationship budgets;
- perform no network access, command execution, or writes.

Acceptance covers deterministic traversal, nested directories, cycles, missing targets, local fragments, bounded resource use, and hostile/malformed path input.

The implemented `workspacefs` package provides `Scan`, `Follow`, finite `Limits`, immutable `Workspace.Documents`, and direct `BuildGraph`/`Validate` delegation to the existing root APIs. It accepts only caller-provided `fs.FS` authority, performs no writes/network/commands, reads files through a bounded reader, visits followed cycles once, and uses deterministic slash-relative `.md`/`.markdown` keys. M116 deliberately left ambiguous relationship-path families fail-closed; M117 completed their reviewed resolution policy without changing the M116 public API.

## M117 — Filesystem resolution hardening

**Status: complete on 2026-08-28; unreleased.**

**Goal:** close path and traversal ambiguity before building more features on top of filesystem discovery.

The completed resolver uses one policy for `Follow`, `Workspace.BuildGraph`, and `Workspace.Validate`:

- ordinary relative `.md`/`.markdown` paths, `./`, nested paths, and literal `..` are resolved against the source document, but the normalized result must remain a valid path in the supplied `fs.FS` namespace;
- each URI path component is percent-decoded exactly once; encoded traversal components and encoded slash or backslash separators fail closed;
- query text is excluded from filesystem lookup, while the target fragment is preserved for the existing fragment-resolution machinery;
- raw backslashes, empty path segments, malformed encoding, absolute paths, URI schemes, protocol-relative targets, directories, and extensionless targets are not treated as filesystem documents;
- no implicit `index.md`, extension inference, case rewriting, or host-path conversion is performed;
- case sensitivity, symlink behavior, and other host semantics remain properties of the caller-provided `fs.FS`; Marksplice does not present `fs.FS` as a universal security sandbox.

M117 also completed the first broad `workspacefs` refactor/profiling pass. The URI-path resolver introduced no measured allocation-count regression in the frozen 256-document workloads. A profiler-driven traversal refactor replaced queue reslicing and two target-state maps with index-based breadth-first traversal and one operation-local availability map. Follow allocation bytes/counts decreased, and 4x document scaling from 256 to 1024 measured approximately 4.69x for `Scan`, 4.22x for chained `Follow`, and 4.31x for dense `Follow`; resource growth remained approximately proportional to input. Parser/document construction remains the dominant cost, so no persistent workspace cache or secondary index was added.

M117 changes no exported API and no Markdown parser semantics.

## M118 — Native semantic walk foundation

**Status: complete on 2026-08-28; unreleased.**

**Goal:** add an on-demand semantic rendering projection without adding a second public AST or increasing the normal retained `Document` model merely for future rendering.

Native remains the single syntax authority. The semantic path should expose an internal event/walk model rich enough for renderers, including explicit block/inline nesting, text, soft/hard breaks, list/container facts, tables, code, links/images, raw HTML, footnotes, math, and relevant source ranges.

Key constraints:

- semantic interpretation stays in Native; renderer code must not reparse Markdown delimiters;
- projection is built/walked on demand rather than retained by every parsed `Document`;
- source ranges are carried where useful for diagnostics and future output mapping, but are not durable identities;
- no parser-internal type crosses the public API.

Normal `Parse` CPU/allocation behavior must be benchmarked before and after this milestone to prove that unused rendering support does not create an avoidable permanent cost.

The implemented foundation adds a separate internal `SemanticBackend`/event vocabulary and Native `WalkSemantic`; it does not add methods to the ordinary `parser.Backend` or retained rendering state to public `Document`. The walk projects Native-owned block/inline hierarchy plus text/break, list/task, table, fenced-code, raw/block-HTML, footnote, and math foundation events synchronously, carries snapshot-local ranges, stops on visitor errors, and discards one operation-local projection index when complete. Reference-definition, alert, and front-matter event identities are reserved for M119 policy rather than guessed in M118.

Profiling rejected the first broader implementation because repeated analysis/detail scans made the 256 KiB realistic semantic walk roughly 143–175 ms. One ephemeral lookup index reduced the final path to roughly 36–42 ms, with a recorded median around 38.8 ms / 36.66 MB / 224k allocations. Ordinary public `Parse` and Native `ParseDocument` allocation bytes/counts remain effectively identical to the frozen pre-M118 baseline; later-run wall-clock movement affected both ordinary paths similarly and is treated as same-host noise rather than a retained semantic-support cost. The final freeze gate also exercised the semantic walk twice over the retained 6,857-document / 60,778,570-byte real-world corpus with deterministic, balanced, source-immutable output. M119 now owns semantic completeness/conformance and the first broad semantic-layer refactor.

## M119 — Semantic completeness and conformance harness

**Goal:** prove that the semantic walk is complete enough to support deterministic rendering before shipping a renderer API.

Cover at least:

- paragraphs/headings and semantic text;
- soft and hard line breaks;
- emphasis/strong/strikethrough/code spans;
- direct/reference links, images, autolinks, and definitions;
- ordered/unordered/tight/loose lists and task items;
- blockquotes and thematic breaks;
- fenced and indented code blocks;
- tables and alignments;
- raw/block HTML;
- footnotes, reviewed math forms, alerts, and front-matter envelope policy.

Run the existing parser conformance and add semantic-walk-focused fixtures/tests without generating expectations from current Native output alone. Perform the first broad semantic-layer refactor here: no duplicate Markdown parser, no permanent second AST, no pointer-heavy structure unless profiling justifies it.

## M120 — HTML fragment renderer

**Goal:** render a parsed Marksplice document as deterministic CommonMark/GFM-compatible HTML fragments through streaming output.

The preferred public shape is writer-oriented (`io.Writer`) with a convenience byte/string-returning helper only where useful.

The renderer must define explicit policies for:

- raw HTML preservation/escaping;
- unsafe URL handling;
- GFM tag filtering;
- tables and task-list markup;
- footnotes;
- code fences and language classes without syntax highlighting;
- mathematical payload as semantic/opaque content without executing a math engine;
- images/links as emitted references only, with no fetching.

CommonMark/GFM expected HTML becomes the primary rendering-conformance oracle. The currently rendering-only GFM `tagfilter` example becomes mandatory before Marksplice may claim GFM HTML-rendering conformance.

## M121 — Standalone HTML and metadata

**Goal:** build complete standalone HTML documents on the same renderer without turning Marksplice into a site generator or template engine.

Add a small deterministic document wrapper (`doctype`, `html`, `head`, charset, body) and a deliberately narrow metadata policy. Front matter remains an opaque/conservative Marksplice envelope: known metadata may be mapped only through an explicit reviewed contract, while arbitrary YAML/TOML interpretation is not introduced.

No implicit stylesheet download, asset fetch, template execution, or network behavior is allowed.

## M122 — HTML source mapping

**Goal:** optionally map Markdown source ranges to emitted HTML output ranges for editor/IDE/tooling integration.

The mapping must be optional so callers that only want HTML do not pay unnecessary retained-memory cost. It should support deterministic Markdown-to-output correlation without treating snapshot-local node IDs as durable cross-revision identities.

After implementation, run a broad renderer refactor/profiling checkpoint covering streaming behavior, mapping overhead, large documents, deep nesting, tables, links, raw HTML, and pathological inline input.

## M123 — Canonical Markdown renderer

**Goal:** add an explicit Markdown-to-Markdown canonical rendering path, separate from ordinary source-preserving editing.

Canonical rendering intentionally normalizes formatting. It must never become the implementation path for existing-document edits.

The writer should use one deterministic Marksplice profile with minimal formatting knobs. Acceptance requires:

```text
Parse(original) -> semantic A
semantic A -> canonical Markdown
Parse(canonical) -> semantic B
A == B
```

and idempotence:

```text
canonical(parse(canonical(x))) == canonical(x)
```

Canonical choices for headings, list markers, fences, tables, references, blank lines, line endings, front matter, and opaque/raw content must be documented and tested. Avoid turning this feature into a general-purpose style formatter.

## M124 — v1.0 stabilization, refactor, profiling, and release gate

**Goal:** freeze the first stable Marksplice contract after workspace discovery, semantic rendering, HTML, source mapping, and canonical Markdown are complete.

M124 is not a feature grab-bag. It is the `v1.0.0` quality boundary.

Required work includes:

- full source/API/dependency/architecture audit;
- complete reuse/dead-code/duplication/refactor pass across M1–M123 where justified;
- production complexity audit;
- CPU/allocation profiling for parse, workspace scan/follow, semantic walk, HTML, source mapping, canonical Markdown, graph/workspace validation, and representative edit paths;
- pathological/oversized/deep input review and explicit resource-bound verification;
- fuzzing of parser/source/rendering boundaries where appropriate;
- CommonMark/GFM parser and HTML-rendering conformance;
- full real-world Markdown corpus regression where the retained private corpus is available;
- race, supported-Go, cross-platform build, static/lint/security/dependency/documentation gates;
- public API stability and compatibility review;
- complete documentation UX/API-reference/example audit;
- release notes and release-readiness verification.

The v1.0 release must be cut only from the exact reviewed M124 freeze commit after its required public CI is green.

## M125 — PDF backend contract (`v1.5.0` queue)

**Goal:** define the PDF integration boundary without coupling Marksplice's parser/editing core to fonts, browsers, printers, operating-system resources, commands, or network access.

The intended architecture is:

```text
Markdown -> semantic walk -> HTML -> PDF backend
```

M125 decides the backend interface, resource-resolution authority, security model, fidelity expectations, dependency/packaging boundary, and test strategy. It does not have to ship a PDF engine merely to complete the contract.

## M126 — First PDF adapter (`v1.5.0` queue)

**Goal:** implement the first reviewed PDF adapter against the M125 contract, using measured fidelity, portability, dependency, and security evidence.

PDF work remains isolated from the v1.0 core contract. The exact implementation technology is intentionally deferred until M125 can evaluate real options against the already-stable HTML renderer.

M126 is the planned `v1.5.0` release gate, subject to the same exact-commit local verification, push, public CI, and release-readiness discipline as v1.0.

## Explicitly excluded from this roadmap

The following are not planned merely because the rendering pipeline exists:

- JSON/XML interchange formats;
- syntax highlighting engines;
- Markdown-site generation/template engines;
- implicit filesystem writes;
- URL/network fetching;
- arbitrary YAML/TOML serialization;
- mathematical execution/rendering engines;
- first-party dialect/plugin collections.

Add such work only after a separate concrete user need and architecture review justify it.
