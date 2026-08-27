# Capability and Third-Party Extensibility Strategy

Status: source of truth for evaluating useful Markdown ecosystem ideas, deciding what belongs in Marksplice core, defining the third-party extensibility boundary, and recording the completed Native-parser cutover strategy.

## Core rule

Marksplice does not maintain a collection of first-party syntax extensions.

A feature belongs in Marksplice core only when it is sufficiently general to improve Markdown/document understanding, source-preserving editing, deterministic construction, navigation, validation, or relationship intelligence without turning the library into a collection of dialects.

Dialect-specific, product-specific, presentation-only, renderer-only, or integration-specific behavior stays outside core. M110 defines the public read-only observation SPI through which independent Go packages may attach such semantics explicitly without becoming Marksplice dependencies.

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

M104 completes footnotes as a core capability because they add useful document relationships rather than presentation-only behavior. The reviewed contract is exact and case-sensitive: parser-backed references expose definition identity and occurrence order; top-level definitions require independent complete source ownership; multiline semantic body segments remain readable without becoming broad rewrite spans; simple bodies can be replaced source-preservingly; coordinated rename updates every parser-bound occurrence; canonical immediate/deferred definitions support typed references; and ordinary links inside footnote bodies participate in existing relationship/graph intelligence. The Native backend integrates this reviewed footnote observation pass while keeping footnotes outside Marksplice's normative GFM grammar and outside any first-party extension mode.

### Mathematical expressions

M105 completes the reviewed mathematical core overlay. Marksplice owns only conservative Markdown-level source semantics for non-empty single-line `$...$`, dollar-backtick, and one-line `$$...$$`, plus exact-info `math` fenced projection through the existing M103 identity. Dedicated forms are independently source-proven, queryable, source-preservingly payload-editable, and constructible through typed/canonical APIs; ambiguous Markdown-owned source fails closed. Mathematical payload remains opaque, and MathJax, KaTeX, MathML, LaTeX parsing/rendering, execution, or network-backed behavior remains outside core.

### Metadata and front matter

M106 completes the metadata/front-matter generalization audit by separating document-envelope ownership from field editability. `Document.FrontMatter()` can report an empty or conservatively metadata-evidenced YAML/TOML envelope even when every value remains opaque; duplicate/complex metadata does not gain mutation authority, and TOML table scope prevents nested members from being misrepresented as top-level fields. Empty leading envelopes have explicit metadata precedence while non-empty delimiter pairs without metadata evidence remain GFM. The useful ecosystem lesson is therefore implemented at the source-ownership layer only: Marksplice still has no generic metadata AST, YAML/TOML parser/serializer, schema system, or normalization path.

### Knowledge-document primitives

M107 implements the broadly useful subset as a syntax-independent overlay over M100: exact globally unique aliases, exact tags, direct logical references to existing `DocumentKey` targets, logical outgoing/backlink queries, and combined reachability/related-document traversal. The overlay does not infer wikilinks, hashtags, front-matter values, paths, or URLs; it does not add source mutation or a second graph. Arbitrary metadata schemas and free-form relationship attributes remain outside the reviewed core contract until a later explicit need justifies generally defined semantics.

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

This is not a prohibition on those features existing in the Marksplice ecosystem. It is a boundary: independent packages may recognize them through the M110 opt-in read-only overlay unless a later explicit core decision establishes that a concept has become sufficiently general.

## Rendering and integration boundary

A renderer layer is now explicitly approved for the post-M115 roadmap, but it remains separate from the source-preserving parser/editing path. M118–M124 add an on-demand semantic walk, HTML rendering, optional source mapping, and canonical Markdown output before the v1.0 gate. Canonical rendering is never ordinary existing-source mutation, and renderer code must consume Native semantic decisions rather than become a second Markdown parser.

PDF is intentionally deferred to M125–M126 for the v1.5 line and must use a separately reviewed backend/resource-authority boundary. The following remain outside the approved core/rendering roadmap unless separately justified:

- syntax highlighting engines;
- LaTeX/MathML/KaTeX/MathJax execution/rendering engines;
- Telegram/product-specific presentation;
- Mermaid/D2/Pikchr/chart execution/rendering;
- base64 image rewriting;
- YouTube/media embeds;
- network-backed enclave/embed behavior;
- site-generation/template systems.

Marksplice may understand the surrounding Markdown structure while treating those payloads/targets as data. The parser/editing core must not gain network, implicit filesystem-write, command-execution, browser, or renderer-backend authority from them.

## M110 third-party extensibility boundary

M110 implements the smallest reviewed public SPI as an explicitly opted-in, read-only semantic/source overlay for independent statically linked Go packages. This is not Go's runtime `plugin` mechanism and does not mean Marksplice ships first-party extensions.

`Parse` remains the baseline GFM entrypoint. `ParseWithOptions` first completes the ordinary Marksplice parse, then invokes caller-registered recognizers synchronously and serially over one immutable source string. Each extension has an exact `ExtensionID` namespace and returns only extension-local kinds, non-empty snapshot-local ranges, and scalar attributes. Marksplice validates and defensively copies all retained observations under caller-provided total node and metadata-byte limits. Duplicate namespaces, malformed output, recognizer errors, recovered panics, or exhausted limits fail the complete call with `ErrInvalidExtension`; no partial extension state is returned.

The overlay cannot suppress, replace, or reclassify core nodes and never consumes core `Kind` ordinals. It exposes no raw patch, `ChangeSet`, `DocumentBuilder`, parser-AST, graph-resolver, filesystem, network, or command authority. Overlapping observations from different extensions are allowed because they are independent read-only claims. Zero options are equivalent to `Parse`, and absence of an extension leaves baseline behavior unchanged.

Recognizers are ordinary caller-linked Go code. Marksplice can validate and bound only the observations it retains; it cannot sandbox or preempt an extension's own CPU, memory, goroutine, filesystem, network, or command behavior. Caller trust therefore governs recognizer execution, while Marksplice retains fail-closed ownership of its own document state.

This parser-backend-independent public boundary was deliberately defined before native-parser work. M111 froze the separate internal parser-substitution contract/differential harness, M112 completed the native block candidate, M113 completed native inline/reference parsed-document observations, and M114 hardened the complete Native backend beneath the unchanged SPI. M115 preserved that boundary during production cutover; extensions remain coupled only to the public read-only overlay, not to parser internals.

## Completed Goldmark exit strategy

The transition completed at M115 without a Goldmark upgrade or migration:

1. complete and harden the Marksplice product/source model;
2. define the M110 third-party boundary;
3. freeze the parser-independent observation/proof contract and historical differential harness in M111;
4. implement Native block parsing in M112;
5. implement Native inline parsing in M113;
6. prove complete Native conformance and hardening in M114;
7. switch the single production bridge to Native, preserve the accepted parser-neutral contract through a dual-proof transition gate, and remove Goldmark code/tests/module dependency in M115.

Goldmark now appears only in historical transition records. Current correctness is specification-first: CommonMark 0.31.2, explicit GFM extension/correction rules, and approved Marksplice-owned contracts govern Native behavior. Historical fuzz-round inputs without independent normative authority remain invariants only.

## Historical Native-parser pre-cutover gate

M114 satisfied the pre-cutover Native-parser gate:

- the complete applicable pinned CommonMark 0.31.2 conformance corpus plus the normative explicit GFM extension corpus;
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

M115 completes that cutover: production parsing and construction proof use Native, the former adapter/differential implementation and `github.com/yuin/goldmark` dependency are removed, and the parser-neutral CommonMark/GFM contracts remain versioned under Native tests. The M110 SPI remains unchanged above the core parser.

## Devil's advocate review

1. **Core capability creep could still turn Marksplice into a dialect collection.** Mitigation: require broad, syntax-independent value before adding core semantics; route product/dialect syntax to M110 third-party packages.
2. **A third-party SPI could weaken source-preservation guarantees.** Mitigation: extensions contribute reviewed syntax/semantic observations, never raw mutation authority; core retains source ownership validation, stale-source checks, candidate proof, and patch application.
3. **A third-party parser hook could create nondeterminism or conflicts.** Mitigation: require explicit opt-in, deterministic registration/order/conflict rules, namespacing, bounded work, and fail-closed ambiguity handling.
4. **The Native parser is now a permanent maintenance responsibility.** Mitigation: keep the specification-first snapshot/fixture gates, focused reviewed contracts, fuzzing, real-world/pathological corpus tests, benchmarks, and ordinary complexity/security gates active; historical differential evidence cannot substitute for current normative review.
5. **Renderer-oriented ideas could pull unsafe authority into core.** Mitigation: keep rendering/media/network integrations outside core; Marksplice treats external targets and fenced payloads as data.
