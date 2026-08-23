# Marksplice Capability Matrix and Roadmap

Status: source of truth for the current product-facing read/edit/create capability surface and forward roadmap.

This document answers three separate questions for each Markdown family:

1. can Marksplice understand the construct semantically under the GFM profile;
2. can callers read a reviewed public structural representation from a parsed immutable snapshot;
3. can callers safely edit existing source or construct new canonical source through reviewed APIs.

These are intentionally different capability levels. A construct may be semantically understood and preserved without being publicly actionable. Public promotion requires reviewed source ownership, caller-facing semantics, and fail-closed behavior.

Normative GFM behavior belongs to [`gfm-conformance.md`](gfm-conformance.md). Durable implementation boundaries belong to [`architecture.md`](architecture.md). Parser/source ownership belongs to [`goldmark-capability-matrix.md`](goldmark-capability-matrix.md). Milestone-specific contracts and verification evidence belong to [`milestones/`](milestones/).

## Current capability matrix

| Family | Semantic understanding | Public read | Existing-document edit | New-document construction | Current boundary |
| --- | --- | --- | --- | --- | --- |
| Paragraphs | Yes | Top-level promoted paragraphs | Replace top-level paragraph content | Raw GFM single-line/parser-proven LF-multiline; typed simple inline content | Container paragraphs are not automatically promoted |
| ATX headings | Yes | Level, style, content range | Rename content while preserving source style | Levels 1–6 from raw GFM or typed simple inline content | Construction uses canonical ATX syntax |
| Setext headings | Yes | Level, Setext style, content range | Rename content while preserving underline/source | Not dedicated | Existing Setext source is preserved; builder does not synthesize Setext headings |
| Sections | Derived from promoted headings | Parent, direct body, subtree range, immediate child heading IDs | Remove; replace body/subtree; insert/move siblings; append direct child | Through heading/block construction rather than a section builder | No separate section identity namespace |
| Unordered lists | Yes | Supported list items, parent/children, subtree range when complete | Content/subtree replace; remove; sibling insert/move; child/subtree append | Flat and homogeneous nested | Existing source markers/indentation are caller source; builder uses canonical `-` |
| Ordered lists | Yes | Supported list items, parent/children, subtree range when complete | Same structural operations as unordered lists | Flat and homogeneous nested | Existing numbering is preserved; builder numbers per container |
| Task lists | Yes | Checked state plus owning list structure | Set checked state; list structural operations reuse list proof | Flat/nested ordered and unordered | Builder writes canonical `[ ]`/`[x]` |
| Fenced code | Yes | Supported exact contiguous content range | Replace content | Single-line or LF-multiline with optional info string | Unsupported lossy body shapes remain non-editable |
| GFM tables | Yes | Comparable `Table`; promoted non-empty cells/body rows; table-owned row/header navigation | Cell replacement; row replace/remove/insert/move; table-level compatible row append; complete-column insert/remove/move | Canonical tables | `BodyRowCount` is semantic; row/cell IDs are promoted subsets; column edits require complete source mapping of every semantic row |
| Table alignment | Yes | `Document.TableAlignments` and `Document.TableRowAlignments` | Set one column or atomically replace full vector while preserving delimiter trivia | Explicit default/left/right/center | Existing edits change only source-proven delimiter syntax; column edits carry semantic alignment through structural changes |
| Emphasis | Yes | Simple source-proven spans | Replace span content | Raw GFM or typed `EmphasisInline` with semantic text plus bounded reviewed code/emphasis/strong/strikethrough children | Existing-source compound spans remain non-editable; ambiguous generated delimiter hierarchies fail closed |
| Strong emphasis | Yes | Simple source-proven spans | Replace span content | Raw GFM or typed `StrongInline` with the same bounded reviewed structured children | Existing-source simple delimiter boundary is unchanged |
| Strikethrough | Yes | Simple source-proven spans | Replace span content | Raw GFM or typed `StrikethroughInline` with semantic text plus bounded code/emphasis/strong children | Direct strikethrough-in-strikethrough construction is rejected; existing-source boundary is unchanged |
| Code spans | Yes | Simple source-proven spans | Replace span content | Raw GFM or typed `CodeInline` with adaptive backtick fences | Shapes requiring whitespace/delimiter normalization remain rejected |
| Inline links | Yes | Simple destination range | Replace destination | Raw GFM; typed inline form with canonical angle destination/title; or `ReferenceLinkInline` full-reference form targeting exactly one already-present exact definition | Structured labels, collapsed/shortcut references, and forward typed references remain deferred |
| Images | Yes | Simple inline-image destination range | Replace destination | Raw GFM; typed inline form with canonical angle destination/title; or `ReferenceImageInline` full-reference form targeting exactly one already-present exact definition | Structured alt text, collapsed/shortcut references, and forward typed references remain deferred |
| Reference definitions | Yes | Supported single-line destination range | Replace destination | Canonical no-title or conservative double-quoted title | Construction rejects titles requiring escaping |
| Autolinks | Yes | Supported token range | Replace supported token | Raw GFM or canonical typed `AutoLinkInline` angle token | Bare/extended generation remains deferred; typed construction requires source-proven reparsing |
| Thematic breaks | Yes | Promoted top-level source-proven physical-line range | Remove exact owned line with candidate survivor proof | Canonical `---` | Nested breaks remain internal; removal fails closed on unsafe joins |
| Blockquotes | Yes | Promoted top-level simple one-line blockquotes with exact line/content ranges | Remove exact owned line with whole-block survivor proof | One paragraph at depth 1 or explicit depth 2–64; multi-block depth 1–64 composition from reviewed child builders, including recursive blockquote children when every structural chain stays within 64 total levels | Existing-source read/edit remains the one-line depth-1 subset; front matter remains excluded as a document envelope; lazy-continuation and broader existing-source nested/multi-block promotion remain deferred |
| YAML front matter | Marksplice envelope recognition outside GFM parser | Unique simple scalar fields | Replace safe scalar value | Canonical document-leading envelope with conservative double-quoted string fields | Complex/ambiguous YAML remains opaque; construction is intentionally not a general YAML serializer |
| TOML front matter | Marksplice envelope recognition outside GFM parser | Unique simple scalar fields | Replace safe scalar value | Canonical document-leading envelope with conservative double-quoted string fields | Complex/ambiguous TOML remains opaque; construction is intentionally not a general TOML serializer |
| HTML | GFM raw/block HTML semantics | Simple comments and quoted `<a id>`/`<a name>` anchors only | Replace proven comment payload or anchor value | No dedicated builder | Other HTML is preserved conservatively as opaque source |

## Cross-cutting guarantees

Parsed documents are immutable source snapshots. Public `NodeID` values are deterministic only within one snapshot and are not durable identities across arbitrary revisions. Public ranges are half-open byte offsets into that snapshot, and `Document.SourceRange` returns caller-owned copies.

Existing-document mutations are prepared as minimal source-bound patches. Untouched bytes are not regenerated, prepared changes reject stale source, and operations fail closed when candidate parsing/source proof cannot preserve required invariants. Marksplice does not use whole-document AST serialization as the ordinary edit path.

New-document construction is deliberately separate. `DocumentBuilder` may use deterministic canonical GFM because there is no pre-existing author formatting to preserve. Reviewed constructed blocks and the final document are reparsed and checked against requested semantic/source expectations before bytes are returned.

Semantic parsing is broader than the public actionable API. Unsupported or not-yet-reviewed shapes remain understandable/preservable rather than being exposed with guessed source ranges or mutation semantics.

The current production maintainability gate is cyclomatic complexity 15 or lower per function (`gocyclo -over 15` must be empty for production code), plus production and test-inclusive `unparam`. The post-M79 whole-code review established this stricter gate without changing public APIs, kind ordinals, `NodeID` derivation, generated bytes, source ownership, parser profile, or fail-closed behavior.

## Completed capability families

The current conservative model is complete through M90. Detailed chronology belongs to milestone records; the durable capability families are:

- **Mapped public editing foundation (M1–M11):** source-preserving paragraph/heading/list/task/table-cell/fenced-code/simple-inline/link/image/front-matter/HTML capabilities, snapshot-bound identity, and bounded source reading.
- **Sections and list hierarchy (M12–M34):** exact section body/subtree operations, sibling/child structure, complete supported list-subtree ownership, source-preserving structural edits, and compact parent/child navigation.
- **Table structural model (M35–M43, M63–M70):** source-proven body rows and table identity, compact ownership/navigation, semantic alignments, row mutation/append, and conservative complete-column insert/remove/move.
- **New-document block construction (M44–M62):** canonical headings/paragraphs/lists/tasks/fenced code/reference definitions/tables/thematic breaks/simple blockquotes, including nested list depth, adaptive fences, titles, and table alignment.
- **Promoted line-owned blocks (M71–M74):** public top-level thematic-break/simple-blockquote ownership and exact fail-closed removal.
- **Typed inline construction (M75–M79, M87–M89):** semantic text, code, emphasis/strong, link/image, strikethrough, and canonical angle autolinks; M87 adds conservative source-proven double-quoted titles, M88 adds bounded construction-only code/emphasis/strong/strikethrough nesting, and M89 adds canonical full reference-link/reference-image forms resolved only against exactly one already-present exact top-level definition without widening parsed-source link/image promotion.
- **Front-matter construction (M80):** one optional leading YAML/TOML envelope with deterministic LF formatting, ordered unique simple fields, conservative double-quoted string values, and proof through the existing source-layer front-matter mapper.
- **Broader blockquote construction (M81–M86):** M81 extends `AppendBlockquote` to one parser-proven LF-multiline depth-1 paragraph; M82 adds explicit depth 2–64 single-paragraph construction; M83 adds `AppendBlockquoteBlocks` for depth 1–64 child-builder composition; M84–M85 add the remaining reviewed non-blockquote body families; M86 admits recursive blockquote children while bounding every total structural chain at 64. Marksplice derives every canonical `> ` prefix and proves the exact construction-only hierarchy without widening existing-source blockquote promotion.
- **First public beta readiness (M90):** keeps release/version state outside runtime code while adding portable pkg.go.dev examples, cross-platform Go 1.26/1.27 CI, dependency-update metadata, beta/release/security documentation, and an external consumer-module verification path for `github.com/zoster81/marksplice`.

## Forward roadmap

The roadmap is ordered by architectural leverage rather than by a promise that every item will become the immediately following numbered milestone. Each slice must preserve established source-preserving and fail-closed rules.

1. **Broader typed inline composition.** Continue from M89 toward structured link/image label/alt composition and bare/extended autolink generation only where source/generation proof remains explicit; collapsed/shortcut or forward typed references require a separate design rather than weakening the exact-definition M89 contract.
2. **Broader existing-source blockquote promotion.** Treat multiline, nested, lazy-continuation, and multi-block parsed-source ownership/editing as a separate review from the now-complete bounded construction hierarchy; do not infer editability from construction capability.
3. **Document graph/workspace intelligence.** After the single-document model is sufficiently complete, add bounded caller-authorized relationship navigation/validation without giving core arbitrary filesystem, network, or command-execution authority.

## Current next boundary: structured labels and broader autolinks

M89 extends the existing `Inline` intent with `ReferenceLinkInline` and `ReferenceImageInline` rather than introducing a reference-specific public AST. Only full reference syntax is generated. The reference label must match exactly one already-appended top-level builder definition by exact string; Goldmark case-normalization is not accepted as construction authority, and forward/collapsed/shortcut forms fail closed. Validation uses an ephemeral construction-only proof source so resolved destination/title semantics can be checked without widening ordinary parsed-source link/image editability.

M90 then prepares the same module for its first public beta without changing Markdown semantics. The next high-leverage product boundary is therefore structured typed link/image labels or broader autolink generation. Existing-source blockquote promotion and document graph/workspace intelligence remain independent roadmap items.
