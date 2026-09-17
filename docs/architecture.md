# Marksplice Architecture

Status: source of truth for durable architecture decisions.

## Mission

Marksplice is a Pure-Go library for source-preserving Markdown document work. It exposes one syntax profile: CommonMark 0.31.2 as the normative base grammar, with published GFM extensions and corrections layered on top.

Two workflows are deliberately separate:

```text
existing Markdown bytes                 new document intent
          |                                      |
        Parse                              DocumentBuilder
          |                                      |
 immutable Document                       canonical writer
          |                                      |
 typed structural operation               parser/model proof
          |                                      |
 minimal source-bound patches                     |
          +----------------------+---------------+
                                 |
                           Markdown bytes
```

Existing-document editing preserves author source and changes only operation-owned bytes. New-document construction may emit deterministic canonical Markdown because there is no prior author formatting to preserve. Canonical Markdown rendering is a third, explicit export path: it intentionally normalizes returned output and must never become the implementation of ordinary source-preserving edits.

Historical design and verification chronology lives in [`docs/milestones/`](milestones/). This document describes the current architecture only.

## Package boundaries

- The root `marksplice` package owns the public document, query, mutation, graph, validation, construction, and rendering API.
- `internal/parser` defines parser-independent semantic contracts.
- `internal/parser/native` is the production CommonMark/GFM parser and semantic-walk implementation. Parser-specific types never cross the public API.
- `internal/source` owns byte ranges, lexical source proof, source fingerprints, validated patches, and patch application.
- `internal/splice` combines parser observations with source proof, builds immutable document state, and prepares source-bound mutations.
- `internal/renderhtml` owns deterministic HTML emission and optional source-to-output correlation. It does not parse Markdown or perform I/O beyond the caller-provided writer.
- `internal/rendermarkdown` owns deterministic canonical Markdown emission from semantic events. It does not own a second Markdown parser or retained AST.
- `internal/publictest` exercises the public API exactly as an external Go consumer would.
- `workspacefs` is an opt-in read-only adapter over caller-supplied `fs.FS` authority. It delegates parsing, graph, and validation semantics to the root package.

The public package remains at the module root so consumers import `github.com/zoster81/marksplice`. Repository layout must not introduce forwarding packages merely for cosmetics.

## Immutable source model

A `Document` is an immutable snapshot of caller-provided bytes. Source ranges are half-open byte offsets `[start,end)` into that exact snapshot. Mutation boundaries are byte-based, not rune-based.

Snapshot `NodeID` values are deterministic within one snapshot, but they are not durable identities across arbitrary revisions. A caller must reparse changed source rather than reusing stale IDs or prepared changes.

Semantic recognition alone is not mutation authority. A public editable capability requires both:

1. a reviewed semantic contract; and
2. exact source ownership for the operation being exposed.

Ambiguous or unsupported shapes remain readable or internal rather than receiving guessed edit ranges.

Typed ranges are operation-oriented. Marksplice intentionally does not pretend that every Markdown construct has one universal “full node range.” A table-cell range, link-destination range, section subtree, blockquote container, or fenced payload range owns exactly the bytes documented by that API.

`Document.SourceRange` validates a snapshot-local range and returns caller-owned bytes. Variable-length public results are defensive copies unless an API explicitly states otherwise.

## Existing-document mutation

A prepared `ChangeSet` contains a fingerprint of the source snapshot plus one or more validated, non-overlapping byte patches.

Application follows this contract:

1. fingerprint the supplied source;
2. reject stale source with `ErrSourceConflict`;
3. validate patch ordering and ranges;
4. apply patches in original source coordinates;
5. preserve every byte outside the changed ranges.

A structural operation may own several coordinated patches when that is the semantic operation, for example a move or a reference rename. `Document.ComposeChanges` combines independently prepared changes only when their source and semantic deltas are proven compatible; it is not a generic patch API.

When a local byte edit could reinterpret surrounding Markdown, preparation reparses the candidate and verifies the expected semantic/source model. Candidate reparsing is a safety oracle, not permission to regenerate unrelated source. Unsafe joins, changed container ownership, ambiguous relationships, malformed output, or stale source fail closed.

Existing-source operations preserve line endings, whitespace, marker spelling, delimiter choices, numbering, and other lexical trivia unless changing those bytes is the explicit requested operation.

## Structural document model

### Headings, sections, and navigation

Headings expose source-proven style/level plus parser-derived semantic text through `Heading.Text()`. Sections are derived from source-ordered headings with heading IDs as section identities; the hierarchy is built with a monotonic stack rather than a second document tree.

Heading anchors, duplicate handling, fragment resolution, and TOC generation are derived on demand. Fragment resolution also considers source-proven HTML anchors. No persistent anchor or TOC index is retained in `Document`.

Heading-level mutation preserves source style when representable, rebuilds section hierarchy through candidate parsing, and fails closed when a requested conversion cannot be proven without rewriting unrelated content.

### Lists

List items retain exact marker/content/container facts needed for source-preserving mutation. Parent/child relationships use compact adjacency over source-proven supported items. Structural list operations require complete ownership of the affected subtree or host position.

When indentation depends on parent marker width or container context, Marksplice derives it from source facts and validates the candidate in its real host. Callers do not need to reproduce CommonMark indentation rules.

### Tables

Tables own a source-proven complete table range, semantic width, header, delimiter row, body rows, alignments, and promoted cell relationships. Table mutations preserve surrounding row trivia and validate width/table ownership in the reparsed candidate.

Column operations require complete row mapping across the table. Insertions clone compatible adjacent layout rather than applying a generic formatter. Alignment mutations patch only proven delimiter-token colons and preserve dash runs and surrounding trivia.

### Blockquotes and alerts

Top-level blockquotes own one complete source-proven physical container with caller-owned inner content ranges. Lazy and mixed-prefix forms may remain readable/removable without becoming generically rewritable.

Content replacement is allowed only when a uniform marker prefix and line-ending style are proven. GitHub-style alerts are a semantic overlay over an ordinary blockquote identity; dedicated APIs mutate the alert kind or body without introducing a separate parser mode or node identity.

### Fenced blocks

`FencedBlock` exposes the complete source-proven container: opening/closing fence facts, indentation, info string, language, closure state, and per-line payload ranges. Embedded language payload is opaque data.

`PrepareReplaceFencedCode` edits a proven contiguous payload and can populate a source-proven empty closed block. `PrepareSetFencedBlockInfo` changes only the proven info payload. Non-contiguous or otherwise unsupported shapes stay read-only.

### Links, images, references, and autolinks

Direct links and images expose source-proven label/alt, destination, and title facts when available. Mutation APIs patch only owned payloads and preserve wrappers, delimiters, spacing, and surrounding source. Title presence has explicit add/remove operations rather than forcing a whole-token rewrite.

Reference definitions and occurrences use parser-defined label normalization and source-proven occurrence mappings. Coordinated rename/retarget/lifecycle operations reparse the complete candidate and fail closed on normalized-label ambiguity or unintended relationship changes.

`Document.LinkRelationships` projects parser-resolved direct/reference link, image, and autolink semantics in source order. Relationship intelligence is broader than editability: a relationship may be readable even when its exact source shape is not editable.

### Footnotes

Footnotes are a reviewed core capability layered outside the normative GFM grammar. Definitions and references are source-proven independently and remain exact/case-sensitive according to the public footnote contract.

Definitions expose complete ownership plus source-backed body ranges. Existing simple-body replacement remains available, while multiline replacement owns the complete proven definition container and reconstructs only the required continuation syntax using source-proven layout. Definition add/remove and coordinated rename are explicit lifecycle operations.

Links inside footnote bodies participate in ordinary relationship intelligence; no second link graph is created.

### Mathematical expressions

Mathematical source is a reviewed opaque semantic overlay. Supported dedicated forms are source-proven and payload-editable; exact-info `math` fences reuse the ordinary fenced-block identity and payload ownership. Marksplice does not parse, validate, execute, or render TeX/LaTeX/MathJax/KaTeX/MathML semantics.

### Front matter

YAML/TOML front matter is a document-leading envelope outside the Markdown grammar. `Document.FrontMatter()` exposes source-proven envelope facts without creating a metadata AST.

Only conservative unique top-level scalar fields are promoted for typed value mutation. Complex, duplicate, nested, or otherwise ambiguous metadata stays opaque. Structural APIs can rename/remove/append a proven simple field and add/remove a conservative envelope without introducing a YAML/TOML parser or serializer.

TOML scalar promotion stops when table scope begins. Existing-source edits preserve every other envelope byte.

### HTML

Parser-proven raw/block HTML is readable and renderable under explicit policy. Existing-source HTML mutation remains deliberately narrow: only source-proven supported comment payloads and quoted anchor attributes become editable.

Preserving raw HTML during rendering is not sanitization. Applications handling untrusted Markdown must choose escaping or a downstream sanitizer appropriate to their security boundary.

## Queries and document intelligence

`QueryNodes` and `QuerySections` are bounded source-ordered scans over immutable snapshot state. Callers must provide positive result limits. Querying does not create a second AST or persistent selector index.

`DocumentGraph` is an immutable graph over an explicit caller-provided finite set of parsed documents. It never discovers documents itself. Non-local resolution is delegated to a build-only caller resolver; local fragments are resolved within the source snapshot. Resolvers are synchronous, serial within one call, and never retained.

`ValidateWorkspace` layers deterministic diagnostics and conservative repair planning over the same explicit document set. Automatic repair is intentionally narrower than diagnostics and uses ordinary snapshot-bound `ChangeSet` values.

`KnowledgeIndex` is a syntax-independent overlay over an existing immutable graph. Caller-provided aliases, tags, and logical references remain distinct from source-backed Markdown relationships; Marksplice does not infer them from filenames, wikilinks, hashtags, front matter, or URLs.

## Filesystem workspace adapter

`workspacefs` is the only package that discovers Markdown through filesystem authority, and that authority must be supplied by the caller as `fs.FS` plus finite limits.

`Scan` performs deterministic `.md`/`.markdown` discovery. `Follow` performs cycle-safe relationship-driven loading. Reads are byte-bounded before complete allocation. The adapter performs no writes, network requests, commands, or host authorization.

Local relationship resolution uses slash-based URI-path semantics: relative dot segments normalize only inside the supplied namespace; path components are percent-decoded once; query text is not part of file lookup; fragments remain relationship metadata; malformed encoding, encoded traversal/separators, absolute/scheme/protocol-relative/backslash/directory/extensionless targets fail closed. Case and symlink semantics remain properties of the caller-provided `fs.FS`.

## New-document construction

`DocumentBuilder` is mutable construction intent, not a parsed snapshot. It has no source fingerprint or snapshot `NodeID` semantics.

The builder emits deterministic LF Markdown and validates generated structure through the same parser-independent semantic/source contracts used by parsed documents. It supports reviewed headings, paragraphs, thematic breaks, blockquotes/alerts, lists/tasks, fenced code, tables, front matter, reference definitions, footnotes, mathematical blocks, and typed inline content.

Typed inline construction represents semantic intent rather than raw Markdown injection. Text is escaped deterministically; delimiter/fence choices are adaptive where required; references are proven against exact definitions. Unsupported or ambiguous nested combinations fail closed.

Construction proof is operation-local. No parallel persistent construction AST is retained after output is produced.

## Rendering and export

Rendering is on demand and separate from ordinary parsing/editing. The Native semantic walk supplies rendering facts only when requested; normal parsing retains no renderer AST or rendering index.

`RenderHTML`/`HTML` emit deterministic fragments. `RenderHTMLDocument`/`HTMLDocument` add a small deterministic standalone wrapper and map only reviewed simple front-matter metadata. Raw-HTML, unsafe-URL, and GFM tag-filter behavior are explicit options.

Source-map variants correlate snapshot-local Markdown byte ranges with byte ranges in that exact output. Maps are optional, caller-owned, and operation-local; they do not create durable cross-revision identities.

`RenderCanonicalMarkdown`/`CanonicalMarkdown` emit one deterministic normalized Markdown representation. Acceptance requires semantic reparse equivalence and byte idempotence. Canonical output never replaces source-preserving mutation.

Rendering performs no URL fetching, asset loading, command execution, templates, syntax highlighting, browser automation, or mathematical-engine execution.

## Parser and conformance boundary

[`gfm-conformance.md`](gfm-conformance.md) owns the normative source hierarchy, approved specification snapshots, parser-neutral fixtures, rendering conformance, canonical-round-trip gates, and update procedure.

`internal/parser/native` is the production parser, but implementation behavior is never its own specification. Regressions are classified against CommonMark 0.31.2, explicit GFM extension/correction rules, or reviewed Marksplice-owned capabilities.

Where semantic observations are insufficient for lossless source work, Marksplice may perform bounded parser-independent lexical proof tied to the immutable source snapshot. Such proof remains internal and does not create an alternate Markdown grammar.

The production dependency graph contains no third-party Markdown parser.

## Third-party extension boundary

`ParseWithOptions` provides an opt-in read-only observation overlay for independent statically linked packages. Extensions receive immutable source and may return namespaced kinds, snapshot-local ranges, and bounded scalar metadata.

Extensions cannot suppress or replace core nodes, consume core `Kind` values, obtain raw patch/builder/parser/graph authority, or alter baseline parsing. Marksplice validates retained observations but cannot sandbox ordinary caller-linked Go code.

Dialect- or product-specific syntax belongs in such independent packages unless a broadly useful, syntax-independent contract justifies core support. See [`extension-strategy.md`](extension-strategy.md).

## Concurrency, errors, and boundedness

Successfully built immutable `Document`, `DocumentGraph`, `KnowledgeIndex`, `WorkspaceReport`, `workspacefs.Workspace`, and prepared `ChangeSet` values are safe for concurrent reads. `DocumentBuilder` is mutable and caller-synchronized.

Public sentinel errors classify failure families; callers should use `errors.Is`. Diagnostic strings are not compatibility contracts.

Structural queries have explicit positive result limits. Workspace traversal/loading has explicit document/byte/depth/relationship limits. Graph/workspace/knowledge operations are bounded by caller-supplied finite document sets. The library does not add hidden global result caps to mask poor algorithms.

## Line endings, Unicode, and encoding

Existing-source edits do not normalize LF, CRLF, isolated CR, Unicode content, or unrelated byte sequences. Any parser compatibility shadow view must remain byte-offset aligned with original source.

Encoding and BOM preservation belong to the host that supplies Markdown bytes. Core does not guess or rewrite file encodings. Full Unicode GFM reference-label case folding uses `golang.org/x/text`; GFM whitespace normalization remains explicitly defined by Marksplice.

New-document construction emits LF by contract.

## Performance and complexity

The design prefers source-ordered vectors, compact scalar records, monotonic scans, adjacency indices, and operation-local maps over redundant persistent indexes.

Important complexity expectations include:

- parsing/index construction: linear or near-linear in source size where practical;
- structural node/section queries: O(N) worst-case with early stop at the caller limit;
- relationship projection: expected O(N+H+R) using ephemeral lookup maps;
- graph build/traversal: O(V+R+E) / O(V+E) plus explicit resolver cost;
- workspace orphan reachability: one multi-source O(V+E) BFS;
- knowledge-index build: expected O(V+A+T+L);
- prepared-change composition: deliberately conservative O(k·N) proof work for `k` supplied changes, without an all-pairs algorithm;
- fenced/container source proof: linear in owned physical lines;
- front-matter recognition: source-linear with temporary key counting only.

Measured optimization must precede architectural complexity. Persistent caches or secondary indexes require benchmark/profile evidence. The retained real-world corpus, pathological scaling benchmarks, fuzzing, race testing, and profile-driven refactors are engineering evidence; machine-specific wall-clock numbers are not cross-machine product guarantees.

Production functions must remain at cyclomatic complexity 15 or lower. Static analysis, production/test-inclusive unparam checks, vulnerability checks, and diff hygiene are part of release-quality verification.

## Safety and authority boundaries

Core treats paths, URLs, relationship targets, fenced payloads, metadata values, and raw HTML as data unless an explicit API grants a narrowly reviewed operation.

The root package performs no arbitrary filesystem traversal, network requests, command execution, host authorization, or implicit writes. `workspacefs` performs only caller-authorized read operations through `fs.FS`.

Prepared mutations fail closed on stale source, malformed/overlapping ranges, invalid target kinds, incomplete ownership, or ambiguous structure. Dependencies remain minimal and exact versions are owned by `go.mod`/`go.sum`.

## Public documentation and history

`README.md` is the public entry point. Getting Started, the user guide, recipes, examples, capabilities, and API reference describe the product as it exists today. They must not require readers to understand internal development phases.

This architecture document likewise describes current durable behavior. Historical sequencing, experimental evidence, and old implementation transitions belong in `docs/milestones/` and other explicitly historical records. The public changelog describes user-visible release changes, not internal phase labels or consumer-specific project history.
