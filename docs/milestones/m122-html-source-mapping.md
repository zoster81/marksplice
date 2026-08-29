# M122 — HTML Source Mapping

Status: **complete locally on 2026-08-29; unreleased pending milestone freeze commit, push, and exact remote CI closure.**

## Goal

M122 adds optional Markdown-source to HTML-output correlation for editor, IDE, preview, and tooling integrations without changing the ordinary HTML rendering path or retaining source-map state in `Document`.

The public surface is deliberately additive:

- `HTMLOutputRange` — half-open byte offsets in one exact HTML result;
- `HTMLSourceMapEntry` — one immutable source/output correlation;
- `Document.RenderHTMLWithSourceMap` — streaming fragment output plus a caller-owned map;
- `Document.HTMLWithSourceMap` — buffered fragment output plus the same map;
- `Document.RenderHTMLDocumentWithSourceMap` — streaming standalone output with absolute output offsets;
- `Document.HTMLDocumentWithSourceMap` — buffered standalone output plus mappings.

Existing `RenderHTML`, `HTML`, `RenderHTMLDocument`, and `HTMLDocument` remain the mapping-off APIs and do not allocate or retain the M122 result map.

## Mapping contract

Both coordinate spaces use **byte offsets**, not rune indexes:

- `HTMLSourceMapEntry.SourceRange()` is a half-open `Range` in the immutable Markdown snapshot used for the render;
- `HTMLSourceMapEntry.OutputRange()` is a half-open `HTMLOutputRange` in the exact HTML bytes emitted by that same successful render.

The map is semantic-event granular rather than total byte coverage. Markup synthesized by the renderer can therefore be unmapped. Nested semantics may intentionally overlap: for example, an emphasis owner can map to `<em>...</em>` while its text child maps to the inner text bytes. Results are sorted by output start; when two entries begin at the same output byte, the longer outer range precedes its nested range.

The map carries no `NodeID`. Source and output offsets are snapshot/result-local coordinates and must not be treated as durable cross-revision identities.

Fragment maps cover only emitted body semantics. Reference-definition and front-matter declarations that produce no visible fragment bytes do not receive fabricated output ranges. Standalone maps use absolute offsets from byte zero of the complete HTML document. Synthetic doctype/html/head/body wrapper bytes remain unmapped, while the reviewed M121 `title`, `description`, `author`, and safe `lang` front-matter scalars map to the exact wrapper bytes they emit when metadata mapping is enabled.

A failed mapped render returns no map. The supplied writer may already contain partial bytes when an error or short write occurs, so callers must treat both output and map as unsuccessful when the method returns an error.

## Architecture

M122 does not add a second parser, renderer AST, persistent source-map index, or document cache.

`internal/renderhtml` continues to consume the exact Native semantic walk used by M120–M121. Mapping-enabled rendering wraps the destination writer in a byte-counting writer that also preserves the `io.StringWriter` fast path. Semantic enter/leaf/exit handling records the output interval produced by the existing renderer operation and correlates it with the event's parser-owned source range.

Two existing deferred-output cases require explicit translation:

- image alt semantics are captured before one final `<img ...>` emission, so the complete image source range is correlated only when those bytes are actually written;
- footnote definition bodies are rendered into the existing local capture buffer and emitted later in the document footnote section. Their relative capture mappings are translated to the final root-output offset at emission time.

Standalone metadata projection remains owned by `internal/splice`, where the source-proven front-matter field ranges already exist. `internal/renderhtml` receives those ranges with the reviewed scalar values and correlates only metadata bytes that are actually emitted.

The public layer collects emitted internal correlations into caller-owned `HTMLSourceMapEntry` values. A chunked collector limits geometric slice-growth waste for large maps and performs one exact compaction before the deterministic output-order sort. Footnote-local capture maps remain temporary operation-local state and are discarded when rendering returns.

## TDD and edge cases

The initial focused RED failed only because the four mapped-rendering methods and public mapping types did not exist.

Focused coverage then proves:

- fragment streaming and buffered output are byte-identical to ordinary M120 rendering;
- standalone streaming and buffered output are byte-identical to ordinary M121 rendering;
- nested semantic ranges overlap intentionally while remaining valid and deterministic;
- Unicode coordinates are byte offsets;
- images and deferred footnote bodies map to their actual final output locations;
- reviewed standalone metadata receives absolute output offsets;
- `HTMLMetadataOmit` does not fabricate metadata mappings;
- tables, lists, links, dangerous-URL suppression, blockquotes, and raw HTML preserve exact renderer output while exposing event-granular mappings;
- 128-level nested blockquote input retains valid mapping/output correlation;
- pathological delimiter-heavy input is deterministic across repeated renders;
- nil receivers/writers and invalid body/metadata options preserve the public `ErrInvalidRender` contract;
- writer errors and `io.ErrShortWrite` return no map;
- returned variable-length mappings are caller-owned and a caller mutation cannot affect a later render.

The 256 KiB benchmark provides large-document coverage and exercises roughly 37.7k emitted mappings per render.

## Devil's-advocate review

1. **Deferred output could produce believable but wrong offsets.** Image and footnote capture cannot use the current root writer position when semantic events are first observed. M122 records capture-relative positions and translates them only when the bytes are actually emitted; focused tests pin both cases.
2. **Counting output bytes could regress every mapped write.** The first implementation wrapped `io.Writer` but did not implement `io.StringWriter`, causing roughly 60k avoidable allocations per 256 KiB render. The accepted counting writer forwards `WriteString`, restoring the renderer fast path.
3. **A convenient global slice could multiply retained-memory cost.** The first public draft accumulated an internal map and then copied it into public values. Profiling rejected that design. The accepted callback path constructs only the public result, while a chunked collector avoids geometric growth and then compacts once.
4. **A whole-document mapping could imply ownership of non-emitted metadata.** The initial hardening test exposed that a document-wide semantic range could indirectly associate front matter with body HTML even when metadata was omitted. M122 deliberately emits no whole-document map entry; useful correlations remain event-specific.
5. **Snapshot IDs could be mistaken for durable source-map identity.** No `NodeID` is present in the mapping contract. Callers correlate only byte ranges from the exact source/output pair returned by one render.
6. **Mapping could silently become mandatory overhead.** The pre-M122 APIs remain separate mapping-off entry points and use the unchanged renderer walk without a counting writer or retained mapping collection.

## Profiling and refactor checkpoint

The same realistic 256 KiB source used by the renderer milestones was measured with fragment and standalone mapping disabled versus enabled.

The first mapped draft measured about **50.53 MB/op / 319k allocations**. Profiling and review found two implementation-created costs rather than intrinsic mapping cost:

- loss of `io.StringWriter` caused nearly one allocation per HTML string write;
- retaining an internal map and then copying it to public values duplicated the large result;
- geometric growth of one large result slice remained an `alloc_space` hotspot even after the duplicate-copy removal.

After restoring `io.StringWriter`, collecting directly into public values, and using chunked growth plus one exact compaction, representative final measurements are approximately:

| Path | Mapping | Allocated bytes/op | Allocations/op | Map entries/op |
| --- | --- | ---: | ---: | ---: |
| fragment streaming | off | 41.80 MB | 258.4k | 0 |
| fragment streaming | on | 44.52 MB | 261.8k | 37,725 |
| standalone streaming | off | 41.80 MB | 258.4k | 0 |
| standalone streaming | on | 44.52 MB | 261.8k | 37,729 |

The measured incremental allocation cost is therefore about **2.7 MB/op** and **3.3k allocations/op** for roughly 37.7k returned correlations on this workload. Mapping-off remains on the M120/M121 allocation profile. Wall-clock values varied materially with host load during the profiling session and are intentionally not treated as a product performance claim.

The broad checkpoint also covers large input, deep nesting, tables, links, raw HTML, deferred footnotes/images, metadata, and pathological inline input. No persistent cache, secondary mapping index, hidden source-size cap, or alternative parser/rendering path was justified by the evidence.

## Conformance boundary

M122 changes no Markdown grammar and no HTML bytes. Mapped fragment output must be byte-identical to `RenderHTML`/`HTML`; mapped standalone output must be byte-identical to `RenderHTMLDocument`/`HTMLDocument` under the same options.

The M120 expected-HTML CommonMark/GFM gates therefore remain the normative rendering oracle. Source-map tests verify correlation around that existing output rather than introducing a new HTML dialect or a snapshot-derived expectation generator.

## Exit boundary

M122 is locally complete when the documented tree passes focused source-map tests, complete repository regressions, race/static/security/dependency/cross-platform gates, the existing parser/semantic/renderer conformance stack, documentation/API dogfood, strict diff/hygiene review, and the final mapping-off/on benchmark checkpoint.

Only after the reviewed milestone freeze commit is pushed normally and public CI succeeds for that exact SHA may M123 — Canonical Markdown renderer — begin.
