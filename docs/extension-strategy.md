# Capability and Third-Party Extensibility Strategy

Status: source of truth for evaluating useful Markdown ecosystem ideas, deciding what belongs in Marksplice core, defining the future third-party extensibility boundary, and preserving the mandatory Goldmark exit plan.

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

M104 completes footnotes as a core capability because they add useful document relationships rather than presentation-only behavior. The reviewed contract is exact and case-sensitive: parser-backed references expose definition identity and occurrence order; top-level definitions require independent complete source ownership; multiline semantic body segments remain readable without becoming broad rewrite spans; simple bodies can be replaced source-preservingly; coordinated rename updates every parser-bound occurrence; canonical immediate/deferred definitions support typed references; and ordinary links inside footnote bodies participate in existing relationship/graph intelligence. The temporary backend uses an isolated footnote semantic pass rather than changing Marksplice's normative GFM profile or exposing a first-party extension mode.

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

M110 implements the smallest reviewed public SPI as an explicitly opted-in, read-only semantic/source overlay for independent statically linked Go packages. This is not Go's runtime `plugin` mechanism and does not mean Marksplice ships first-party extensions.

`Parse` remains the baseline GFM entrypoint. `ParseWithOptions` first completes the ordinary Marksplice parse, then invokes caller-registered recognizers synchronously and serially over one immutable source string. Each extension has an exact `ExtensionID` namespace and returns only extension-local kinds, non-empty snapshot-local ranges, and scalar attributes. Marksplice validates and defensively copies all retained observations under caller-provided total node and metadata-byte limits. Duplicate namespaces, malformed output, recognizer errors, recovered panics, or exhausted limits fail the complete call with `ErrInvalidExtension`; no partial extension state is returned.

The overlay cannot suppress, replace, or reclassify core nodes and never consumes core `Kind` ordinals. It exposes no raw patch, `ChangeSet`, `DocumentBuilder`, parser-AST, graph-resolver, filesystem, network, or command authority. Overlapping observations from different extensions are allowed because they are independent read-only claims. Zero options are equivalent to `Parse`, and absence of an extension leaves baseline behavior unchanged.

Recognizers are ordinary caller-linked Go code. Marksplice can validate and bound only the observations it retains; it cannot sandbox or preempt an extension's own CPU, memory, goroutine, filesystem, network, or command behavior. Caller trust therefore governs recognizer execution, while Marksplice retains fail-closed ownership of its own document state.

This parser-backend-independent public boundary was deliberately defined before native-parser work. M111 freezes the separate internal parser-substitution contract/differential harness, M112 completes the native block candidate, and M113 completes native inline/reference parsed-document observations beneath it without changing the SPI. M114–M115 must preserve the same M110 public overlay rather than coupling extensions to Goldmark or requiring a parser-specific SPI redesign.

## Goldmark exit strategy

Goldmark is temporary. Marksplice will replace it with a native CommonMark/GFM parser and remove the dependency completely.

The transition is staged:

1. complete and harden the Marksplice product/source model;
2. define the M110 third-party boundary;
3. freeze the native parser observation/proof contract and differential harness in M111 (complete);
4. implement native block parsing in M112 (complete);
5. implement native inline parsing in M113 (complete);
6. prove complete native conformance and hardening in M114;
7. remove Goldmark and cut production over in M115.

Goldmark is isolated behind the M111 parser-independent backend contract and remains only the temporary production backend/differential oracle while the replacement is developed. M112 and M113 now supply the native parsed-document block/inline/reference candidate: block, inline-node, and link/reference relationship projections each pass all 676 parser-applicable published-GFM examples through the shared harness. M114 owns complete native backend/conformance hardening before M115 cutover. No Goldmark upgrade or migration is part of this roadmap.

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
