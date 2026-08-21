# Milestone M1 — Lossless Editing Feasibility

Status: green — feasibility gate passed.

## Question

Can Marksplice provide source-preserving structural editing for GitHub Flavored Markdown (GFM), using Goldmark for semantic understanding while preserving untouched source byte-for-byte across useful edits?

M1 is a feasibility gate. It does not freeze the full public API or implement the document graph. GFM is the only Markdown syntax profile in scope.

## Deliverables

- repository bootstrap with Apache-2.0 licensing and attribution, now recorded retrospectively as M0 because the bootstrap and first M1 slice landed together in the root commit;
- Goldmark isolated behind an internal parser adapter and configured with the single GFM profile;
- minimal source snapshot, range, fingerprint, node, and prepared-change model;
- representative structural edit tests that prove unchanged bytes remain identical;
- stale-source conflict handling;
- deterministic invalid-range/invalid-target behavior;
- capability matrix separating Goldmark-provided semantics from Marksplice-owned lossless source information;
- the reproducible GFM gate defined by `docs/gfm-conformance.md`, including the approved snapshot pin and focused parser-boundary regressions;
- initial fuzz targets for parser/source-map and patch boundaries when the focused model is stable enough to fuzz productively.

## First vertical slice

The bootstrap slice targets paragraph replacement discovered through semantic parsing.

The proof must:

1. parse a Markdown source containing surrounding structure;
2. obtain a Marksplice-owned paragraph node and exact byte range from the Goldmark-backed adapter;
3. prepare replacement bytes against the parsed source snapshot;
4. apply the change only to the exact original snapshot;
5. verify the semantic/local replacement result;
6. verify the prefix and suffix outside the changed range are byte-identical;
7. preserve the source's original line-ending form (LF, CRLF, or CR where exercised by the construct);
8. reject stale source;
9. assign distinct deterministic identities to structurally distinct nodes even when their human-readable content is duplicated.

This slice is deliberately smaller than the complete feasibility matrix. Its purpose is to validate the architecture before expanding syntax coverage.

## Proven list-item slice

The next proven slice maps and replaces the content of a single-line list item while preserving its existing marker, ordered-list number/delimiter, indentation, post-marker spacing, line ending, and all bytes outside the content range. The candidate is reparsed and rejected if the same single-line list-item structure cannot be re-established at the original source position.

This proof covers representative unordered, ordered, and nested list items. Multiline or otherwise structurally complex list-item replacement remains outside this slice and must fail closed until a broader source mapping is proven.

## Proven table-cell slice

The table proof maps non-empty GFM header/body cell content to a Marksplice-owned raw cell boundary and replaces only the semantic content bytes. Existing pipe delimiters, alignment rows, cell padding, neighboring cells, and line endings remain untouched. Rows with and without outer pipes are covered, as is an escaped `\|` within cell content.

The replacement candidate is reparsed and the same header/body column plus raw cell boundary must still be provable. An unescaped `|` that would split the target into multiple cells is therefore rejected. Empty cells and table shapes whose lexical boundary cannot yet be proven remain outside this slice and fail closed for mutation.

## Proven fenced-code slice

The fenced-code proof replaces only the semantic content bytes of one non-empty, single-line, top-level, explicitly closed fenced code block. Opening and closing fence bytes remain outside the patch, including backtick-versus-tilde choice, opening/closing fence lengths, indentation, info-string spelling/spacing, and the content line ending. Representative LF and CRLF cases pass, including a closing fence longer than the opener.

The candidate is reparsed and Marksplice must re-prove the same opening position, fence character and lengths, info-string range, opening/closing indentation, and content boundary. A replacement that would become an early closing fence is rejected. Multiline, empty, unclosed, or container-prefixed fenced-code mutation remains outside this slice; unsupported single-line source shapes remain parsable but fail closed when mutation is attempted.

## Proven strikethrough slice

The strikethrough proof replaces only the plain-text content of one simple, non-empty, single-line GFM strikethrough while preserving its existing one- or two-tilde delimiters and every byte outside the semantic content range. Representative single-tilde CRLF and double-tilde Unicode cases pass.

The replacement candidate is reparsed and must re-establish the same simple strikethrough at the same source position with the same delimiter length. Replacements that create a larger tilde run, introduce a line break, or change the inline child into nested Markdown markup are rejected. Nested/compound strikethrough content and multiline inline shapes remain outside this slice.

## Proven links/references batch

The links batch proves three related source-preserving behaviors in one pass. A simple single-line inline link with a plain-text label can replace only its destination bytes while preserving the label, parentheses, raw-versus-angle destination form, spacing, optional title syntax, surrounding paragraph bytes, and line ending. Representative raw, `<...>`, CRLF, and balanced-parenthesis destinations pass.

A single-line link reference definition can likewise replace only its destination while preserving indentation, label, colon, raw-versus-angle form, spacing, title syntax, trailing spaces, and line ending. A definition edit is also proven to leave full (`[text][id]`), collapsed (`[id][]`), and shortcut (`[id]`) reference-link source byte-identical. Every replacement is reparsed and rejected unless the same semantic link/definition, destination boundary, wrapper form, label/title facts, and source position are re-established.

Compound inline-link labels, multiline links/reference definitions, reference-label rewrites, and relationship-wide renaming remain outside this batch and fail closed where an exact destination mapping cannot be proven.

## Proven inline-syntax batch

The inline-syntax batch proves four additional source-preserving behaviors together. Angle and bare autolinks can replace only their semantic source token while preserving angle brackets when present, surrounding paragraph bytes, and line endings. Representative angle URL, bare HTTPS, bare `www`, and published `mailto:` extended-autolink forms pass; replacements that cease to be the same autolink category are rejected.

Simple single-line code spans can replace only their content while preserving the exact backtick-run length, including a two-backtick span whose content contains a single backtick. Simple plain-text emphasis and strong spans likewise replace only content while preserving `*` versus `_` and one- versus two-character delimiter runs. All candidates are reparsed and must re-establish the same anchor, semantic kind, delimiter style, and source boundaries.

Code spans whose Goldmark semantics normalize surrounding spaces, multiline code spans, and compound/nested emphasis such as `***text***` remain outside this proof. They remain parsable but fail closed for mutation until their extra lexical trivia can be mapped exactly. Paragraph replacement regression coverage also proves that adding these inline observations does not prevent a valid paragraph replacement containing links, autolinks, code, emphasis, strong, or strikethrough.

## Proven front-matter/HTML batch

The front-matter proof keeps the Markdown body on the normal GFM parser profile and recognizes metadata only as a separate Marksplice-owned leading source envelope. A closed byte-zero `---` YAML or `+++` TOML envelope becomes metadata only when at least one unique simple scalar field has a provable lexical boundary; non-leading, unclosed, duplicate-only, or complex-only shapes remain ordinary GFM rather than being guessed as front matter. Unknown lines inside a recognized envelope remain opaque and byte-identical.

Representative YAML CRLF and TOML LF edits replace only one scalar value while preserving delimiter lines, key spelling, separator spacing, quote style, inline comments, trailing spaces, unrelated simple fields, complex nested YAML source, line endings, and the complete Markdown body. Candidate source is rescanned and must re-establish the same envelope format, field key, scalar style, and shifted boundaries. This is deliberately not full YAML/TOML semantic parsing; nested structures, arrays/tables, multiline scalars/strings, duplicate target keys, and other ambiguous values are not structurally editable in this proof.

The HTML proof uses Goldmark's GFM `RawHTML`/`HTMLBlock` recognition rather than an independent HTML parser. A simple single-line valid HTML comment can replace only its payload while preserving `<!--`/`-->` and inner horizontal padding. A simple `<a>` opening tag can replace only one quoted `id` or `name` value while preserving tag/attribute spelling, other attributes, spacing, and quote style. Other recognized inline raw HTML and HTML blocks are mapped conservatively as opaque source regions. Replacements are reparsed and must re-establish the same GFM raw-HTML construct and exact editable boundary.

## M1 consolidation

After the representative syntax proofs were complete, M1 was consolidated before declaring the gate green. The immutable document model now builds a snapshot-local `NodeID` index during parsing, giving structural target lookup constant expected time instead of scanning all nodes for every operation. Duplicate IDs fail closed while the ordered node slice remains the structural iteration source.

Mutation preparation now shares only genuinely common plumbing: target lookup, simple replacement preconditions, change-set/candidate construction, candidate parsing, and range-shift helpers. Syntax-specific source mapping and semantic validation remain typed rather than being hidden behind a generic mutation framework. The source layer is likewise organized into cohesive block, link, inline, front-matter, HTML, and shared lexical responsibilities.

The Goldmark adapter walk now delegates AST-specific extraction to small typed observation functions, keeping `Adapter.Parse` as orchestration rather than a monolithic semantic switch. Simple HTML opening-tag scanning uses a small reusable stateful scanner instead of embedding attribute parsing into anchor policy. Tests were split along the same feature boundaries so implementation and regression evidence remain locally discoverable.

The consolidation review stopped when the remaining branching was concentrated in bounded lexical parsers rather than orchestration or duplicated workflow code. Further splitting solely to reduce a metric would add indirection without improving the safety model. M1 also retains whole-candidate reparsing as a conservative single-mutation validation oracle; avoiding repeated full rescans in a future multi-edit batch is a post-M1 optimization requirement, not a reason to weaken the proven validation contract.

## Feasibility matrix

Each row records the M1 evidence for GFM semantic recognition, exact source mapping needed for the proven edit, unchanged-byte preservation, and relevant malformed/ambiguous behavior. GFM-specific constructs are first-class requirements rather than optional extensions.

| Construct | M1 expectation | Evidence |
| --- | --- | --- |
| ATX headings | recognize, map, rename content while preserving markers/spacing | top-level proof passes |
| Setext headings | recognize, map, rename content while preserving underline/spacing | top-level proof passes |
| Paragraphs | recognize, map, replace | proof passes |
| Unordered lists | recognize markers/items and edit representative item | single-line content replacement proof passes |
| Ordered lists | preserve numbering/style outside edits | single-line proof preserves number/delimiter outside the content patch |
| Nested lists | preserve nesting/indentation outside edits | nested single-line item proof passes; complex parent items are outside M1 |
| GFM task lists | update task state without list normalization | proof passes; one-byte state patch preserves marker/indentation/case on no-op |
| GFM tables | update representative cell without reformatting untouched table source | non-empty header/body cell replacement proof passes; escaped pipes and no-outer-pipe rows covered |
| Fenced code | preserve fence delimiter style/length when untouched | single-line top-level closed-block replacement proof passes; multiline/empty/container-prefixed mutation is outside M1 |
| GFM strikethrough | recognize and preserve delimiters outside edits | simple single-line plain-text replacement proof passes for one/two-tilde delimiters; nested/compound inline content is outside M1 |
| GFM extended autolinks | recognize source/destination boundaries | angle/bare token replacement proof passes for HTTPS, `www`, and published `mailto:` forms; invalid bare-FTP replacement remains rejected |
| Inline links | recognize destination/source boundaries | simple single-line destination replacement proof passes for raw/angle destinations, spacing, titles, and balanced parentheses |
| Reference links | preserve reference style | preservation proof passes for full/collapsed/shortcut forms during a referenced-definition destination edit |
| Reference definitions | recognize and update representative definition | single-line destination replacement proof passes for raw/angle destinations, indentation, spacing, titles, and CRLF |
| YAML front matter | preserve unrelated fields/source | leading simple-scalar proof passes with CRLF, trailing spaces, unrelated fields, and complex nested source preserved opaquely; ambiguous/duplicate/complex-only envelopes remain GFM |
| TOML front matter | preserve unrelated fields/source | leading simple-scalar proof passes for quoted strings and bare booleans while preserving quote style, separator spacing, inline comments, and body source |
| HTML comments/anchors/opaque regions | preserve conservatively | proof passes for single-line valid comment payload edits, quoted `id`/`name` anchor values, and opaque GFM HTML-block/raw-HTML mapping |
| LF | unchanged-byte proof | proof passes |
| CRLF | unchanged-byte proof | proof passes |
| CR | unchanged-byte proof | heading proof passes through byte-stable parser shadow normalization |
| Unicode | byte-range correctness | inline strikethrough replacement proof passes with multibyte UTF-8 content |
| Malformed/ambiguous cases | deterministic fail-closed behavior where mutation cannot be proven safe | representative fail-closed coverage spans invalid ranges/targets, list/table/fence boundaries, unsafe links/autolinks, normalized/compound inline syntax, front-matter ambiguity/duplicates, and unsupported HTML shapes |
| Inline code spans | preserve backtick-run style outside content edits | simple single-line proof passes for one/two-backtick runs; normalized-space and multiline forms are outside M1 |
| Inline emphasis/strong | GFM-compatible semantic model before structural editing | simple plain-text proof passes for `*`/`_` emphasis and `**`/`__` strong; compound/nested delimiter runs are outside M1 |

The table is the final M1 evidence record. Future capability expansion belongs to later milestones; do not reinterpret a passing M1 row as feature completeness beyond the explicitly proven shapes.

## Retrospective hardening

The M0–M29 retrospective audit rechecked M1's foundational snapshot and patch primitives against the larger mutation surface built later. Snapshot copying, replacement-byte copying, stale-source fingerprinting, sorted disjoint patch validation, parser isolation, isolated-CR shadow parsing, and candidate reparsing remain valid. One fail-closed edge was strengthened: `ChangeSet.Apply` now computes the final result length with overflow-checked integer arithmetic and reports `ErrInvalidRange` rather than relying on allocation behavior if an extreme patch set would exceed representable `int` length. Focused tests cover growth, shrinkage, invalid operands, and `MaxInt` overflow without changing the public API or ordinary patch complexity.

## Exit decision

M1 is green. Representative tests demonstrate that the combined semantic + source-map model is workable without whole-document serialization and without unacceptable special-case coupling to Goldmark internals. The feasibility matrix contains evidence for every M1 construct, stale-source rejection is proven, malformed/ambiguous mutations fail closed across representative syntax families, the approved GFM conformance gate is in place, and initial parser/source-patch fuzz targets exist.

This decision is a feasibility result, not a feature-completeness claim. Complex multiline/container forms, broader structural insert/remove/move operations, batching, sections, images, document-graph behavior, and stable public API design remain post-M1 work unless separately proven.

The next design phase should derive the stable public API and broader operations from the M1 invariants rather than exposing the feasibility implementation or parser-specific abstractions directly.
