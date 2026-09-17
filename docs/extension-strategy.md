# Capability and Third-Party Extensibility Strategy

Status: source of truth for deciding what belongs in Marksplice core and what belongs in independent extensions or host applications.

## Core rule

Marksplice does not maintain a collection of first-party Markdown dialect extensions.

A capability belongs in core when it has broad value for Markdown/document understanding, source-preserving editing, deterministic construction, navigation, validation, rendering, or relationship intelligence and can be defined without weakening the CommonMark/GFM baseline.

Dialect-specific, product-specific, presentation-only, execution-oriented, or integration-specific behavior stays outside core unless a later architecture review establishes a general contract.

The wider Markdown ecosystem is useful as a catalog of ideas. External parser/extension packages are not a dependency roadmap and do not define Marksplice architecture.

## Evaluation criteria

A candidate is a good core fit when it improves one or more of:

1. structural understanding and navigation;
2. exact source-preserving editing;
3. deterministic new-document construction;
4. document relationships and graphs;
5. validation and diagnostics;
6. broadly useful technical-document authoring;
7. deterministic export of already-understood document semantics.

A candidate should normally stay outside core when its main purpose is:

- a niche or application-specific Markdown dialect;
- presentation or site generation;
- executing an embedded language;
- network/media fetching or embedding;
- arbitrary transformation of user content;
- hidden filesystem or command authority;
- behavior that silently changes the normative CommonMark/GFM profile.

Core capabilities must use Marksplice-owned semantic/source contracts. Upstream implementations may provide evidence, but they do not become semantic authorities merely because they exist.

## Broad capabilities that belong in core

### Navigation and document relationships

Heading anchors, duplicate-anchor handling, local fragment resolution, TOC generation/synchronization, source-ordered link/image/reference/autolink relationships, explicit multi-document graphs, backlinks, reachability, workspace diagnostics, and conservative repair planning are natural extensions of the document model.

These features operate over explicit immutable document sets. Marksplice does not discover files or URLs implicitly and does not retain caller resolvers after graph/validation construction.

### Semantic block patterns

Higher-level meaning may be recognized over syntax that is already valid baseline Markdown without inventing a new grammar mode. GitHub-style alerts are the current example: they remain ordinary source-proven blockquotes with an additional semantic view and dedicated source-preserving mutations.

### Fenced-block semantics

Fenced containers are a generic Markdown capability. Marksplice exposes exact fence/container/info/payload facts and source-preserving operations where ownership is proven.

Info strings such as `mermaid`, `geojson`, `topojson`, `stl`, `math`, `d2`, `pikchr`, or any other language identifier remain opaque data. Recognizing the container does not authorize parsing, executing, validating, highlighting, or rendering its embedded language.

### Footnotes

Footnotes provide broadly useful document semantics and relationships. Marksplice owns exact definition/reference source contracts, source-preserving rename/body/lifecycle operations, deterministic construction, and integration with ordinary link/graph intelligence.

Footnotes remain an explicitly reviewed core capability outside the normative GFM grammar rather than a caller-selectable dialect switch.

### Mathematical source forms

Marksplice owns conservative Markdown-level source semantics for reviewed mathematical forms, including exact-info `math` fences. Mathematical payload remains opaque.

TeX/LaTeX parsing, MathJax/KaTeX/MathML rendering, execution, package resolution, fonts, or network-backed resources stay outside core.

### Front matter

A leading YAML/TOML envelope is useful document metadata structure, but Marksplice intentionally does not become a general YAML/TOML library.

Core owns conservative envelope recognition, unique simple top-level scalar fields, narrow source-preserving field/envelope lifecycle operations, and deterministic simple construction. Complex/nested/duplicate metadata remains opaque.

### Knowledge-document primitives

Aliases, tags, and logical references are useful when supplied explicitly by the caller over an existing document graph. They belong in a syntax-independent overlay.

Marksplice does not infer this metadata from wikilinks, hashtags, filenames, paths, front matter, or URLs and does not merge logical references into source-backed Markdown relationships.

## Syntax that should normally stay in independent packages

Examples include:

- wikilinks such as `[[page]]`;
- hashtag syntax;
- definition lists not covered by the normative profile;
- heading attributes;
- custom/Pandoc-style containers;
- emoji shortcodes;
- application-specific Markdown variants;
- wiki-table syntax;
- product-specific inline/block markers;
- syntax whose primary purpose is presentation or integration with another application.

This boundary does not prohibit an ecosystem around Marksplice. It means independent packages may recognize such syntax explicitly without silently expanding the core grammar.

## Read-only third-party extension boundary

`ParseWithOptions` exposes an opt-in, parser-independent, read-only observation overlay for statically linked Go packages.

Each extension has its own namespace and may return only:

- extension-local kinds;
- non-empty snapshot-local source ranges;
- bounded scalar metadata.

Marksplice validates configuration and retained observations, including namespaces, kinds, ranges, metadata names/values, and caller-provided total node/metadata budgets. Results are defensively copied. Recognizer errors, invalid observations, exhausted limits, or recovered panics fail the complete call with `ErrInvalidExtension`; partial extension state is not published.

Extensions cannot:

- suppress, replace, or reclassify core nodes;
- consume or register core `Kind` values;
- obtain raw patch or `ChangeSet` authority;
- write through `DocumentBuilder` on behalf of core;
- receive parser AST/context types;
- mutate graphs/workspaces through hidden hooks;
- gain filesystem, network, command, or rendering authority from Marksplice.

Recognizers are ordinary caller-linked Go code. Marksplice can validate and bound only the observations it retains; it cannot sandbox or preempt the recognizer's own CPU, memory, goroutines, filesystem, network, or command behavior. Caller trust governs extension execution.

Zero extension options are behaviorally equivalent to ordinary `Parse`.

## Rendering and integration boundary

Rendering/export is separate from source-preserving editing.

Core provides deterministic HTML and canonical Markdown because those outputs are direct representations of semantics Marksplice already owns. Rendering is on demand, has no second Markdown parser, and retains no renderer AST in every `Document`.

The following remain outside core unless a separately reviewed integration boundary justifies them:

- syntax highlighting engines;
- mathematical rendering engines;
- Mermaid/D2/Pikchr/chart execution;
- browser automation;
- media or URL fetching;
- base64/image rewriting;
- site generators and template engines;
- product-specific presentation layers;
- implicit filesystem writes.

A future PDF capability must sit behind a separately reviewed backend/resource boundary. The document parser/editing core must not gain browser, font, command, network, or host-resource authority merely because PDF output exists.

## Admission discipline

Before moving a concept into core, require evidence for all relevant questions:

- Is the concept broadly useful rather than tied to one consumer?
- Is its semantic contract clear independently of one parser implementation?
- Can exact source ownership be proven for any proposed mutation?
- Does it preserve the existing CommonMark/GFM grammar boundary?
- Can it be bounded without hidden global caps?
- Does it avoid new filesystem/network/command authority?
- Can it reuse existing document/query/graph/rendering primitives instead of creating a parallel subsystem?
- Is the maintenance cost justified by concrete usage and tests?

API symmetry alone is not sufficient reason to add capability. A readable shape does not automatically become editable, and an editable simple shape does not imply that every complex shape must be writable.

## Devil's advocate review

1. **Core capability creep could turn Marksplice into a dialect collection.** Mitigation: require broad, syntax-independent value and keep product-specific spellings in independent packages.
2. **Extensions could weaken source-preservation guarantees.** Mitigation: the public extension SPI is read-only; source ownership and mutation remain core-controlled.
3. **Extension execution could be mistaken for a sandbox.** Mitigation: document that recognizers are ordinary caller code and that only retained observations are validated/bounded.
4. **Rendering integrations could pull unsafe authority into core.** Mitigation: keep fetching, commands, browsers, templates, highlighting, and embedded-language execution outside the document core.
5. **A third-party parser could become an accidental semantic oracle.** Mitigation: keep correctness specification-first and retain parser-independent tests/source proof.

Historical implementation transitions and milestone sequencing are preserved in explicitly historical records, not in this current capability policy.
