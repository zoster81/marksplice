# M123 — Canonical Markdown Renderer

Status: **complete locally on 2026-09-15; unreleased pending milestone freeze commit, push, and exact remote CI closure.**

## Goal

M123 adds an explicit deterministic Markdown-to-Markdown export path without weakening Marksplice's source-preserving editing model.

The public surface is deliberately small:

- `Document.RenderCanonicalMarkdown(io.Writer) error` — streaming canonical output;
- `Document.CanonicalMarkdown() ([]byte, error)` — caller-owned buffered output.

Canonical rendering intentionally normalizes formatting. It never mutates the parsed snapshot, does not produce a `ChangeSet`, and must not become the implementation path for ordinary existing-document edits.

## Architecture

`internal/rendermarkdown` consumes the parser-independent semantic-event contract introduced by M118–M119. Native remains the only Markdown syntax authority.

The renderer therefore:

- owns no second parser or AST;
- has no direct dependency on `internal/parser/native`;
- retains only operation-local stacks, small family-specific buffers, reference state, and top-level ordering/ownership state;
- performs no filesystem discovery, network access, asset fetching, command execution, template execution, syntax highlighting, or mathematical execution;
- leaves the immutable source bytes untouched.

`internal/splice` bridges the snapshot source and semantic backend to the canonical writer in the same way the HTML path remains separate from source-preserving mutation.

## Canonical contract

M123 exposes one built-in Marksplice profile rather than a configurable style formatter. The core acceptance properties are semantic round-trip equivalence:

```text
Parse(original) -> semantic A
semantic A -> canonical Markdown
Parse(canonical) -> semantic B
A == B
```

and byte idempotence:

```text
canonical(parse(canonical(x))) == canonical(x)
```

The writer uses deterministic LF output and deterministic block spacing, headings, list markers/indentation, tables, fences, references, links/images, autolinks, footnotes, front matter, raw HTML, and reviewed math forms. Opaque payload remains data under its semantic owner rather than being interpreted as another language.

Reference source spelling may change when the semantic destination/title remains the same. Canonical linked-reference images use a stable shortcut form where that is required to preserve the Native composite semantics. Footnote-shaped ordinary reference labels use an unambiguous direct form rather than relying on source syntax that Native would interpret as a footnote.

## TDD and real-world hardening

The initial focused RED established the missing canonical writer helpers and invalid-input/writer-error behavior. The implementation then expanded through focused semantic/idempotence tests and the complete published/real-world corpora.

Corpus-driven defects were fixed as general parser/renderer rules rather than per-document allowlists. The notable families included:

- top-level source ordering and synthetic reference-definition placement;
- multiline reference definitions and reference-label normalization;
- emphasis/strong/strikethrough delimiter stability;
- bare-autolink punctuation and entity boundaries;
- terminal fenced-code payload and EOF handling;
- nested-list indentation that must preserve following parent-depth opaque blocks;
- whitespace-only fenced-code payload lines nested inside lists;
- footnote-like syntax inside parser-proven fenced code;
- duplicate residual top-level blocks also owned by a semantic footnote overlay;
- linked reference-image composites and escaped punctuation in shortcut labels.

The Native footnote corrections are ownership fixes, not renderer exceptions: fenced-code ranges prevent the scanner from claiming opaque code payload, and promoted footnote ranges suppress residual root blocks whose start is owned by the semantic overlay. This gives consumers one authoritative semantic interpretation of those source bytes.

## Conformance and regression evidence

The permanent M123 canonical gates use the same hash-pinned external specification snapshots as the existing parser/renderer conformance infrastructure.

All **652 CommonMark 0.31.2 examples** and all **677 published-GFM examples** are rendered canonically, reparsed, compared for reviewed semantic facts, rendered again, and required to be byte-identical to the first canonical result. There is no per-example canonical mismatch allowlist.

The retained real-world gate covers **6,857 Markdown documents / approximately 60.8 MB**. For every document it verifies source immutability, panic-free rendering, semantic round-trip equivalence, and byte idempotence. The semantic oracle normalizes only representation-only differences that the canonical export is allowed to change, such as reference-definition source spelling/order and source-form labels; dedicated focused tests retain explicit coverage for definition retention/synthesis and footnote/reference ownership.

## Profiling and refactor checkpoint

M123 profiling was performed before freeze rather than assuming a streaming API was automatically linear.

The first realistic scaling run exposed two writer-owned superlinear algorithms:

1. every top-level block was inserted into a source-ordered slice by scanning/shifting prior blocks;
2. every semantic ownership overlay rescanned all accumulated top-level blocks.

Both were removed. The accepted writer appends blocks, performs one stable O(B log B) ordering step at flush, normalizes ownership ranges once, and filters ordered blocks in one pass. A follow-up CPU profile then exposed the equivalent Native semantic ownership lookup scanning every owned range for every root block; Native now normalizes those intervals once and performs ordered binary lookup.

A complexity review also split canonical list validation, marker generation, body preparation, and continuation-line emission after the initial `renderList` exceeded the repository production threshold. The final production `gocyclo -over 15 -ignore '_test\\.go$' .` gate reports no function above 15.

Representative final same-host measurements on the realistic scaling harness are approximately:

| Input | Native semantic walk | Canonical streaming |
| --- | ---: | ---: |
| 64 KiB | 10–11 ms | ~11 ms |
| 256 KiB | 41–45 ms | ~48 ms |
| 1024 KiB | ~191 ms | ~217 ms |

Allocated bytes and allocation counts scale approximately with input. At 256 KiB the streaming path uses about **44.04 MB/op**; the buffered helper adds roughly **1.05 MB/op** for the returned result. Five single-pass measurements of the complete retained 60.8 MB corpus were roughly **19.8–20.2 MB/s** before the final Native lookup micro-optimization; the final exact-source scaling confirms the writer overhead remains modest relative to `WalkSemantic`.

These are engineering measurements for one host/harness, not cross-machine throughput guarantees. No persistent render cache, second retained tree, or benchmark-specific source-size cap was added.

## Devil's-advocate review

1. **Semantic-oracle normalization could hide a real renderer regression.** Mitigation: all published examples, dedicated reference/footnote/image tests, and the full retained corpus are required together; normalization is limited to representation-only fields and no document/example allowlist exists.
2. **Overlay ownership could discard a legitimate adjacent root block.** Ownership is based on parser-proven half-open source ranges and block start offsets, not textual proximity. Focused tests cover boundaries, real definitions outside fences, and multiline footnote bodies.
3. **Whole-document canonicalization could leak into ordinary mutation.** The API is explicit and output-only; no `ChangeSet` or mutation path delegates to `internal/rendermarkdown`, and user documentation routes source-preserving edits separately.
4. **Streaming could still imply hidden whole-output buffering.** The writer does not retain the final result, but the semantic walk and deterministic top-level ordering do require operation-local working state. The buffered convenience helper additionally owns the complete output by contract.
5. **Reference/link spelling choices could become unstable.** Stable direct/reference/shortcut policies are tested by semantic round-trip plus second-render byte equality, including ambiguous footnote-shaped labels and linked badges.

## Exit boundary

M123 is locally complete when the documented tree passes focused/public/full tests, exact CommonMark/GFM canonical gates, the retained real-world canonical corpus, race/static/security/dependency/cross-platform gates, documentation/API dogfood, production complexity <=15, strict diff/text/private-boundary hygiene, and a complete final review.

Only after the reviewed M123 freeze commit is pushed normally and public CI succeeds for that exact SHA may M124 begin. M124 is the separate v1.0 stabilization/release gate and is not part of M123.
