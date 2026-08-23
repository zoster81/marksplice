# Marksplice Architecture

Status: source of truth for durable architecture decisions.

## Mission

Marksplice is a Pure-Go library for creating and source-preserving structural manipulation of GitHub Flavored Markdown (GFM). GFM is the project's single normative Markdown syntax profile.

Marksplice has two deliberately separate core paths:

```text
                       caller intent
                            |
                 +----------+----------+
                 |                     |
          existing GFM bytes      new document
                 |                     |
               Parse            DocumentBuilder
                 |                     |
        immutable snapshot       canonical writer
                 |                     |
       semantic + source map      parser/model proof
                 |                     |
        structural operation            |
                 |                     |
        minimal byte patches            |
                 +----------+----------+
                            |
                         GFM bytes
```

Existing documents are immutable source snapshots. Ordinary edits use minimal source-bound patches and never authorize normalization of untouched source. New documents may use deterministic canonical GFM because there is no author formatting to preserve. New-document writing must never be substituted for the existing-document edit path.

M0 is the green repository-bootstrap baseline and M1–M91 are complete. Detailed milestone chronology, feature contracts, and verification evidence live in `docs/milestones/`; this document records only architecture that remains durable across those milestones.

## Package boundaries

- `internal/parser/goldmark` owns Goldmark-specific parsing, AST traversal, and narrowly scoped GFM compatibility behavior. No Goldmark type crosses this boundary.
- `internal/source` owns parser-independent byte ranges, lexical source proof, source fingerprints, validated patches, and patch application.
- `internal/splice` combines parser observations with source mappings, builds immutable snapshot indexes/relationships, and prepares structural mutations.
- `internal/publictest` owns black-box tests that import the root module API exactly as an external Go consumer would.
- root package `marksplice` owns reviewed public types and operations. It may wrap internal implementation values but must not expose Goldmark or `internal/*` types.

The public package deliberately remains at the Go module root because the canonical import path is `github.com/zoster81/marksplice`. Moving that package under a top-level `src/` directory would change the natural import path to `github.com/zoster81/marksplice/src` or require a forwarding facade whose only purpose is repository cosmetics. Root Go filenames are therefore grouped by responsibility instead: `api*.go` owns parsed-document/read/edit APIs and `builder*.go` owns new-document construction plus its package-private proof/writer helpers. Longer-form documentation stays under `docs/`, while external-style public API tests stay out of the root under `internal/publictest/`.

Keep orchestration separate from syntax-specific proof. Shared plumbing may centralize target lookup, patch transforms, candidate assembly, and candidate parsing, while lexical/source mappers and semantic validators remain feature-specific when their safety invariants differ. Reuse focused lexical primitives rather than duplicating scanners across families.

## Source model and public promotion

Source is represented as bytes. Ranges are half-open byte offsets `[start,end)` into one immutable snapshot. Byte offsets, not rune indexes, define mutation boundaries.

A parsed document may retain compact derived indexes and adjacency needed for efficient navigation, but the ordered node collection remains the source of structural iteration order. Snapshot `NodeID` values are deterministic and comparable only within the snapshot whose source facts produced them; they are not durable identities across arbitrary revisions. Duplicate derived IDs must fail closed rather than silently select one node.

Semantic recognition alone is not enough for public promotion. An actionable public kind requires reviewed caller-facing semantics plus exact source ownership for the operation it exposes. Unsupported or ambiguous semantic shapes remain internal/non-editable instead of receiving guessed ranges or partial mutation guarantees.

Typed public ranges are operation-oriented. Marksplice intentionally does not expose one generic "full node span" contract: a paragraph range, list-item content range, list subtree range, table-cell range, blockquote line range, or link destination range means exactly the bytes documented by that typed capability.

`Document.SourceRange` validates a snapshot-local public range and returns caller-owned bytes. Public accessors returning variable-length relationships or metadata return defensive copies so callers cannot mutate internal snapshot state.

## Existing-document mutation model

A prepared change contains the source fingerprint it was created against and one or more validated, non-overlapping byte patches.

Application rules are:

1. fingerprint the supplied source;
2. reject stale source explicitly;
3. validate patch ranges and ordering;
4. apply patches in source coordinates without rendering unrelated source;
5. preserve every byte outside changed ranges.

One named structural operation may own multiple coordinated patches, such as delete+insert movement. This is not a generic public batch API: the operation owns the complete semantic proof and validates one combined candidate.

When a local byte patch could change surrounding Markdown interpretation, mutation preparation reparses a candidate and validates the required surviving source/semantic model. Candidate proof is a conservative safety oracle, not permission to rewrite the document. Unsafe joins, newly introduced promoted structure, changed container relationships, ambiguous ownership, and malformed source fail closed.

No operation synthesizes whitespace, line endings, numbering, or delimiters unless that synthesis is part of an explicit new-document construction contract. Existing lexical trivia is source data.

## Section model

Sections are derived from promoted document-level headings rather than assigned a second identity namespace. A section distinguishes the governing heading, direct body, and complete subtree. Parent/child relationships derive from heading levels.

The section hierarchy is built once from source-ordered headings using a monotonic stack. Heading `NodeID` remains the section identity. Immediate child navigation reuses the same hierarchy with compact scalar linkage rather than storing slices inside the comparable public `Section` value.

Section mutations operate on exact stored boundaries. Body replacement preserves the heading and child sections; subtree replacement/insertion/movement validate caller or snapshot-owned section structure; direct-child append requires exactly parent level + 1. Structural candidate validation preserves unaffected heading level/style/source mapping and rejects Setext/paragraph or other join reinterpretation. No operation rewrites heading levels or manufactures separator whitespace.

## List model

Promoted list items retain exact first-line content replacement semantics while internal source ownership records the complete physical line. Supported single-line-head parent items may expose semantic `HasChildren`, immediate supported `ParentID`/`ChildIDs`, and `SubtreeRange()` only when the complete supported descendant subtree is proven.

Parent identities are resolved after ordinary snapshot IDs exist. Downward navigation uses compact source-ordered adjacency. Temporary anchor/ordinal maps are discarded after parse; no second persistent hierarchy index is introduced.

Subtree completeness compares semantic direct-child counts with supported promoted children and propagates completeness leaf-up. Structural removal, movement, sibling insertion, child append, and subtree replacement require the relevant complete source-owned subtree. Unsupported descendants make the structural boundary unavailable rather than guessed.

Host context is authoritative where GFM indentation depends on parent marker width and container context. Child insertion/replacement is therefore validated in the host candidate instead of forcing standalone parseability. Marksplice never synthesizes indentation, list numbering, markers, or line endings for existing source.

Movement uses exact snapshot-owned subtree bytes and coordinated original-coordinate patches. Overlapping source/anchor subtrees are rejected. Same-parent no-ops remain fingerprint-bound. Candidate proof validates descendant-relative mappings, parentage, source bytes, and operation-known parent child-count deltas.

## Table model

Tables have a source-proven public identity independent of any body row. `Table.Range()` owns the complete header + delimiter + semantic body source block. `ColumnCount()` is the source-proven semantic width; `BodyRowCount()` is semantic and can exceed the number of promoted body-row identities.

Goldmark provides semantic table hierarchy/alignment values, but Marksplice derives lexical ownership independently. The source table anchor comes from the public header position rather than trusting container position. Header, delimiter, body rows, cell ranges, and delimiter alignment tokens are mapped in the source layer.

Public rows/cells keep comparable scalar values. Variable-length table relationships and alignment vectors are document accessors returning caller-owned storage. Parse-time temporary maps build compact row/header/cell adjacency and scalar table ownership; persistent anchor maps are avoided.

Row mutations preserve exact caller/source row bytes, validate compatible width/table ownership in the host candidate, and never synthesize a missing line separator. Row movement is restricted to the same table under the current contract.

Alignment mutation changes only source-proven delimiter colons while preserving each exact dash run and surrounding trivia. Whole-vector alignment mutation batches changed delimiter tokens into one candidate validation cycle.

Column insert/remove/move uses a stricter operation-time capability gate: header, delimiter, and every semantic body row must all map at exactly the table width. Removal owns one complete slot plus an adjacent separator; movement permutes source-proven contents while destination-slot trivia remains fixed; insertion clones adjacent slot padding/separators and delimiter dash-run style. Target-table rows/cells may structurally change, while non-target table survivors still undergo ordinary transformed-range/source validation. These operations add no persistent column index.

## Blocks, inline syntax, metadata, and HTML

Paragraphs/headings, fenced code, simple strikethrough/code/emphasis/strong, links/reference definitions/autolinks/images, simple front-matter fields, and narrow HTML comment/anchor forms follow one mapped-capability rule: parser semantics identify the family, then Marksplice proves exact operation-oriented source ownership before public editability.

Fenced-code bodies are editable only when the semantic body is one exact contiguous source span; delimiter/fence details remain private proof data. Simple inline spans reject shapes that require normalization or ambiguous delimiter ownership. Link/image/reference/autolink mutations preserve labels, wrappers, titles, spacing, and other source trivia outside the exact destination/token range.

Thematic breaks and the supported simple blockquote subset additionally own complete physical lines. Structural removal deletes exactly that line and uses a shared whole-block survivor proof so paragraph/Setext/container joins cannot silently reinterpret external source.

Front matter is a Marksplice-owned document envelope outside the GFM parser. Only a closed byte-zero `---` YAML or `+++` TOML envelope with at least one unique source-proven simple scalar field is recognized. Complex, ambiguous, duplicate-only, non-leading, or unclosed forms remain ordinary GFM source. Existing-document front-matter edits patch only proven scalar values and preserve all other envelope bytes. New-document construction owns at most one envelope as separate `DocumentBuilder` state and writes conservative canonical double-quoted string fields before every GFM body block.

HTML remains conservative. Goldmark may recognize raw/block HTML semantically, but public edits are limited to source-proven simple comment payloads and quoted `id`/`name` values on supported `<a>` openings. Other HTML is preserved opaquely.

## New-document construction model

`DocumentBuilder` is deliberately separate from parsed `Document` snapshots. Construction state represents intent before authoritative source exists, so it has no snapshot `NodeID`, source fingerprint, or stale-source semantics.

The builder writes deterministic LF GFM, one blank line between retained GFM blocks, and one final LF for a non-empty document. An optional YAML/TOML front-matter envelope is separate document-leading state rather than a `constructionBlock`; when a body exists, exactly one blank line separates the closing envelope delimiter from the first GFM block. Each requested block is validated, written, reparsed, and checked against family-specific semantic/source expectations before retention; `Markdown()` validates the complete generated sequence again.

Historical block/list/table builder APIs accept caller-provided raw GFM where documented. Typed-inline APIs are a separate semantic-text path and do not change those raw-source contracts.

Canonical construction currently includes a single optional YAML/TOML front-matter envelope with conservative string fields, ATX headings, parser-proven paragraphs, thematic breaks, single-paragraph blockquotes at depth 1 or explicit nesting depth 2–64 with canonical repeated `> ` prefixes on every LF-separated physical line, multi-block blockquotes at depth 1–64 composed from every reviewed body-block family including recursively nested blockquotes with total structural depth bounded at 64, flat/homogeneous nested ordered/unordered lists and task lists, supported fenced code, reference definitions, and GFM tables with optional explicit alignment.

Nested list construction accepts structural depth rather than source indentation. The writer derives indentation from the generated parent content column and uses container-local ordered numbering. Fenced-code construction selects adaptive backtick fences against every potentially closing body run. Reference definitions use angle destinations and a conservative no-escape title form. Tables use canonical outer pipes/padding and canonical alignment delimiters.

Typed inline construction represents semantic text and reviewed inline structures rather than raw Markdown injection. Text is deterministically escaped. Code uses adaptive backticks under the existing exact source-mapped code-span boundary. M88 permits bounded structured nesting of code, emphasis, strong, and strikethrough inside emphasis/strong/strikethrough wrappers. Link/image typed construction uses semantic-text label/alt plus strict canonical angle destinations; M87 adds separate `WithTitle` constructors that write conservative non-empty double-quoted titles and prove exact title presence/value/source range through the existing mappings. M89 adds canonical full reference-link/reference-image construction whose exact reference label must match exactly one already-appended top-level reference definition in the same builder. Collapsed/shortcut reference forms, forward definitions, and structured link/image label/alt nesting remain deferred. Canonical autolinks are accepted only when reparsing produces the exact requested source-proven autolink.

Construction proof reuses the same parser-independent source mappings and semantic metadata as parsed snapshots rather than creating a parallel Markdown AST. Front-matter construction specifically reuses `MapLeadingFrontMatter`; body expectations are written into the already-prefixed final buffer so their byte ranges are final rather than shifted after envelope insertion. M81–M86 blockquote construction deliberately uses construction-only semantic validators plus Marksplice lexical proof instead of widening ordinary blockquote observations, because ordinary parsed-node sets participate in existing-document mutation validation. M82 proves explicit single-paragraph structural depth. M83 snapshots a child `DocumentBuilder`, renders its already-reviewed blocks through the ordinary construction writers, prefixes every physical line including canonical blank separators, and proves that removing those prefixes yields the exact canonical inner source. M84–M85 extend the construction-only Goldmark comparison to list/list-item hierarchies, reference definitions, and table/header/row/cell hierarchies with alignment values. M86 admits blockquote nodes into that same iterative child comparator and performs an explicit iterative pre-render traversal of the private construction tree so every nested blockquote chain remains within the 64-level bound before recursive canonical writing can occur. Observations wholly contained by a construction-only multi-block blockquote are excluded from ordinary construction-node matching because the container-specific proof is authoritative there. Front matter remains a document envelope. M87 extends the existing typed-inline expectation proof with title value/range facts only; it adds no parser/source mapper. M88 adds a separate ephemeral construction-inline hierarchy proof for code/emphasis/strong/strikethrough wrappers rather than widening ordinary simple-inline promotion. The writer bounds wrapper nesting at 64, alternates `*`/`_` for nested emphasis-family delimiters, rejects direct strikethrough-in-strikethrough, and still fails closed when GFM reparsing cannot reproduce an exact requested hierarchy. Goldmark matching is single-pass with anchor buckets and precomputed parent→children lists, while simple leaf/source-mapped expectations remain independently validated through the existing parsed snapshot path. M89 reuses the same construction-only boundary for full reference links/images: the builder resolves one exact existing definition, emits the requested reference token, and validates it against an ephemeral proof source containing canonical definitions. The proof requires Goldmark `ReferenceLinkFull`, exact reference value, resolved destination/title semantics, and exact label/reference source ranges; the proof definitions are never emitted into the requested block and ordinary parsed-source link/image promotion remains unchanged. Successful generated source may then be passed to `Parse` to enter the immutable snapshot/editing model.

## GFM and Goldmark boundary

`docs/gfm-conformance.md` owns the normative source hierarchy, pinned GFM snapshot, conformance procedure, and update policy. CommonMark is inherited as GFM's base syntax; Marksplice exposes no separate CommonMark mode.

Goldmark is the selected semantic parser implementation, configured for GFM plus only narrowly scoped compatibility behavior required by the approved contract. Do not expose Goldmark AST/types publicly, serialize its AST as the existing-document edit path, or enable non-GFM extensions merely because Goldmark supports them.

Where Goldmark positions or public semantic state are insufficient for lossless source work, Marksplice may perform bounded lexical scans tied to the source snapshot. Such scans remain parser-independent and internal; Marksplice must not copy/fork Goldmark implementation code to obtain lexical trivia.

YAML/TOML front matter remains a separate document-envelope layer, not an additional Markdown dialect. HTML rendering is not a product capability; if rendering is later added, GFM rendering-specific requirements such as tag filtering become explicit acceptance criteria.

## Line endings, Unicode, and encoding

Existing-source edits must not normalize LF, CRLF, isolated CR, Unicode content, or unrelated byte sequences. If parser compatibility requires a same-length shadow view, byte offsets must remain exactly aligned with original source.

Encoding/BOM preservation belongs to the host that supplies Markdown bytes unless Marksplice later introduces an explicit encoding-aware layer. Core must not guess and rewrite file encodings.

New-document construction currently emits canonical LF by contract.

## Performance and complexity

Parsing/index construction should remain linear or near-linear in source size where practical. Relationship models use compact source-ordered arrays/scalar links plus temporary maps rather than redundant persistent graph indexes. Mutation planning should avoid repeated whole-document rescans and should parse at most the candidate snapshots required by the operation's safety proof.

The current production complexity gate is **cyclomatic complexity 15 or lower per function**: `gocyclo -over 15 -ignore '_test\.go$' .` must report no production function. Production and test-inclusive `unparam` checks are also required. This gate is a maintainability constraint, not an instruction to fragment cohesive lexical/state-machine logic; split code when responsibility, reuse, or verification clarity improves.

The post-M79 whole-code review reduced prior complexity hotspots by separating Goldmark block/inline dispatch, list marker parsing, table row/cell survivor proof, whole-block anchor validation, table alignment planning/candidate proof, table/list adjacency validation, complete-table mapping, YAML scalar validation, and shared link/image destination-tail parsing. The refactor changes no public APIs, kind ordinals, `NodeID` derivation, source ownership, generated bytes, parser profile, or fail-closed behavior.

## Safety and authority boundaries

Prepared mutations fail closed on stale source, malformed/overlapping ranges, invalid target kinds, incomplete ownership, or ambiguous structure. Core code does not perform arbitrary filesystem traversal, network requests, command execution, or host authorization.

External paths, URLs, and relationship targets are data unless a caller outside core explicitly provides bounded authority. Future multi-document/workspace intelligence must operate only on caller-authorized document sets or roots.

Dependencies should remain minimal. Exact dependency versions belong in `go.mod` and `go.sum`; licensing belongs in `LICENSE` and `NOTICE`.

## Public module and release boundary

The Go module path is `github.com/zoster81/marksplice`. `go.mod` owns the minimum supported Go version and exact dependency graph; public release tags own module versions. Runtime code does not contain a second release-version authority. While the API remains under active development, public releases stay in the v0 series and may use explicit pre-release identifiers. Published Go module tags are immutable. Public versioning, first-push readiness, CI compatibility policy, and proxy/pkg.go.dev verification are owned by `docs/releasing.md` rather than by runtime packages.

M90 adds no Markdown/runtime behavior. It prepares public Go-module consumption through portable package examples, cross-platform public CI, dependency-update metadata, beta/release documentation, security-reporting guidance, and an external consumer-module compilation test. The separately licensed, hash-pinned GFM conformance corpus remains external to the repository and therefore remains a stricter maintainer gate rather than a vendored public-CI input.

M91 is a repository-layout refactor only. It moves the black-box public API tests from the module root to `internal/publictest`, groups root package filenames into `api*` and `builder*` families, and adds `docs/README.md` as the documentation index. It does not change the module path, package name, exported API, Markdown behavior, or ordinary consumer import path. Coverage gates after M91 must use cross-package instrumentation because the consumer-style tests now live in a separate package directory.

## Public capability and history references

`docs/README.md` is the documentation index. `docs/capabilities.md` is the authoritative current read/edit/create matrix and forward roadmap. `docs/goldmark-capability-matrix.md` records parser/source ownership by syntax family. `docs/releasing.md` records public module/version policy. `docs/milestones/` retains the detailed M0–M91 contracts, design evolution, and historical verification evidence.
