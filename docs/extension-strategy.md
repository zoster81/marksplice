# Capability and Third-Party Extensibility Strategy

Status: source of truth for evaluating useful Markdown ecosystem ideas, deciding what belongs in Marksplice core, defining the future third-party extensibility boundary, and preserving the mandatory Goldmark exit plan.

## Core rule

Marksplice does not maintain a collection of first-party syntax extensions.

A feature belongs in Marksplice core only when it is sufficiently general to improve Markdown/document understanding, source-preserving editing, deterministic construction, navigation, validation, or relationship intelligence without turning the library into a collection of dialects.

Dialect-specific, product-specific, presentation-only, renderer-only, or integration-specific behavior stays outside core. After M110, independent Go packages may implement such behavior through the reviewed public SPI if they need it.

The Goldmark ecosystem is used only as a catalog of feature ideas. Its extension packages are not a dependency roadmap and are not an architecture model that Marksplice intends to reproduce.

## Evaluation rules

A candidate idea is suitable for core when it provides broad value in one or more of these areas:

1. structural understanding and navigation;
2. source-preserving editing;
3. deterministic new-document construction;
4. document relationship and graph intelligence;
5. validation and diagnostics;
6. generally useful technical-document authoring.

A candidate should stay outside core when its primary value is:

- rendering or presentation;
- execution of an embedded language;
- a niche or product-specific Markdown dialect;
- network/media embedding;
- arbitrary content transformation;
- behavior that weakens the pinned GFM baseline or source-preservation invariants.

Useful concepts should be implemented through Marksplice-owned semantic/source contracts. Upstream packages are references, not planned production dependencies.

## Ideas selected for Marksplice core

### Anchors and table of contents

Heading-anchor derivation, duplicate-anchor handling, fragment resolution, TOC generation, stale-TOC detection, and source-preserving synchronization are natural consequences of Marksplice headings, sections, links, and document intelligence. They belong in core and require no external extension model. M98 implements immutable single-document navigation with no persistent anchor index; M99 completes source-ordered single-document link/reference/image/autolink intelligence while reusing the same fragment targets; M100 implements deterministic outgoing edges, backlinks, reachability, and direct related-document relationships across an explicit caller-provided document set under a build-only caller resolver, with no hidden filesystem/network discovery. M101 implements diagnostics and conservative repair planning over that same authority boundary: caller resolution explicitly distinguishes ignored/resolved/missing non-local relationships, root-relative reachability uses the resolved graph, unresolved references are reported only for conservative explicit full/collapsed forms, and automatic repair is limited to caller-designated M98-recognized stale TOCs.

### Semantic block patterns

Patterns that are already valid baseline Markdown may receive higher-level semantics without changing the grammar. M102 implements the first concrete case: GitHub alerts `NOTE`, `TIP`, `IMPORTANT`, `WARNING`, and `CAUTION` are recognized only as an exact Marksplice-owned overlay over already-promoted top-level blockquote source and can be constructed canonically from single-paragraph, typed-inline, or reviewed multi-block bodies. Alerts reuse the underlying blockquote identity/ownership, retain no semantic index, and cannot be nested through the builder. No Goldmark alert extension or new Markdown grammar mode is involved.

### Fenced-block semantics

M103 completes the generic core capability. `FencedBlock` exposes a source-proven top-level fenced container with exact opening/optional closing fence metadata, info string/language, complete container range, and caller-owned per-physical-line payload ranges, including empty and unclosed forms. The historical contiguous `FencedCode` payload-replacement contract remains separate and narrower, and canonical construction now supports empty payloads without inventing a blank body line.

Names such as `mermaid`, `geojson`, `topojson`, `stl`, `math`, `d2`, `pikchr`, or other technical languages remain opaque data values. Marksplice does not parse, execute, render, syntax-highlight, or validate the embedded language merely because it recognizes the fenced block.

### Footnotes

Footnotes are a high-value candidate for core because they introduce useful document relationships rather than presentation-only behavior. If the source contract is approved, core support should include references, definitions, multiple-reference behavior, construction, source-preserving editing, navigation, and graph integration.

### Mathematical expressions

GitHub-compatible mathematical source semantics are useful enough to evaluate for core. Marksplice should own only the Markdown-level syntax and exact source ranges. The mathematical payload remains opaque. MathJax, KaTeX, MathML, LaTeX rendering, or other renderers remain outside core.

### Metadata and front matter

Marksplice already owns a conservative YAML/TOML document-envelope model. The useful ecosystem lesson is to audit whether the envelope/source model should become more general, not to replace it with a generic metadata AST or serializer. Unknown metadata must remain source-preserved rather than normalized.

### Knowledge-document primitives

Document aliases, logical references, tags, and relationship metadata may be useful as syntax-independent graph/query primitives. Core should add such concepts only when they remain useful independently of a specific dialect spelling.

## Ideas that should normally stay outside core

The following concepts are useful examples for future third-party packages rather than reasons to expand the Marksplice grammar:

- wikilinks such as `[[page]]`;
- hashtag syntax;
- definition lists;
- heading attributes;
- Pandoc-style fenced divs;
- custom block tags;
- emoji shortcodes;
- Obsidian-specific syntax;
- Discord/Telegram-specific Markdown variants;
- wiki-table syntax;
- CJK-specific parser behavior when it changes baseline syntax rules;
- other project-specific Markdown dialects.

This is not a prohibition on those features existing in the Marksplice ecosystem. It is a boundary: they should be implemented by independent packages through M110 unless a later explicit core decision establishes that a concept has become sufficiently general.

## Rendering and integration ideas outside core

The following remain outside the source-preserving engine unless a separate renderer/integration project is explicitly approved:

- syntax highlighting;
- PDF output;
- HTML presentation helpers;
- LaTeX/MathML/KaTeX/MathJax rendering;
- Telegram rendering;
- Mermaid/D2/Pikchr/chart rendering;
- base64 image rewriting;
- YouTube/media embeds;
- network-backed enclave/embed behavior.

Marksplice may understand the surrounding Markdown structure while treating those payloads/targets as data. Core must not gain network, filesystem, command-execution, or renderer authority from them.

## M110 third-party extensibility boundary

M110 may expose a small public SPI for independent, statically linked Go packages. This is not Go's runtime `plugin` mechanism and does not mean Marksplice ships first-party extensions.

The exact public API remains a later design task, but any approved SPI must satisfy these constraints:

- core GFM semantics cannot be silently redefined;
- third-party syntax/semantic kinds must be isolated or namespaced;
- parsing and construction must remain deterministic and bounded;
- source ownership must be explicit enough for every exposed edit;
- third-party code cannot submit arbitrary unvalidated byte patches;
- mutations still use Marksplice stale-source, overlap, candidate-proof, and patch validation;
- filesystem/network/command authority is not granted by registration;
- failure of one third-party recognizer must not corrupt core parser state;
- callers must opt in explicitly to non-core syntax;
- absence of a third-party package must leave baseline Marksplice behavior unchanged.

The SPI should be designed together with the Marksplice-native parser boundary so extensibility is intentional rather than bolted onto the parser afterward.

## Goldmark exit strategy

Goldmark is temporary. Marksplice will replace it with a native CommonMark/GFM parser and remove the dependency completely.

The transition is staged:

1. complete and harden the Marksplice product/source model;
2. define the M110 third-party boundary;
3. freeze the native parser observation contract and differential harness in M111;
4. implement native block parsing in M112;
5. implement native inline parsing in M113;
6. prove complete native conformance and hardening in M114;
7. remove Goldmark and cut production over in M115.

Goldmark may be used as a temporary differential oracle while the replacement is developed. No Goldmark upgrade or migration is part of this roadmap.

## Native-parser cutover gate

Before M115 removes Goldmark, the Marksplice-native parser must satisfy all of the following:

- the complete applicable pinned GFM conformance corpus;
- every focused parser-boundary and source-position regression required by Marksplice;
- equivalent or better semantic/source observations for every public read/edit/create capability;
- no reduction in source-preservation or stale-source guarantees;
- bounded behavior on malformed, deeply nested, and oversized inputs;
- block/inline/source-position fuzz coverage;
- explicit CPU, allocation, and large-document scaling evidence;
- cross-platform verification;
- production complexity and maintainability gates no weaker than the rest of Marksplice;
- no parser-specific type leaking into public APIs;
- compatibility with the reviewed M110 third-party SPI boundary.

After this gate passes, M115 requires production cutover and removal of Goldmark from `go.mod`, `go.sum`, runtime code, active tests, and non-historical documentation paths.

## Devil's advocate review

1. **Core capability creep could still turn Marksplice into a dialect collection.** Mitigation: require broad, syntax-independent value before adding core semantics; route product/dialect syntax to M110 third-party packages.
2. **A third-party SPI could weaken source-preservation guarantees.** Mitigation: extensions contribute reviewed syntax/semantic observations, never raw mutation authority; core retains source ownership validation, stale-source checks, candidate proof, and patch application.
3. **A third-party parser hook could create nondeterminism or conflicts.** Mitigation: require explicit opt-in, deterministic registration/order/conflict rules, namespacing, bounded work, and fail-closed ambiguity handling.
4. **A native parser becomes a permanent maintenance responsibility.** Mitigation: delay implementation until the Marksplice contract is mature, then require complete conformance, differential parity, fuzzing, benchmarks, and ordinary complexity gates before removing Goldmark.
5. **Renderer-oriented ideas could pull unsafe authority into core.** Mitigation: keep rendering/media/network integrations outside core; Marksplice treats external targets and fenced payloads as data.
