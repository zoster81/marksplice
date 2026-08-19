# Milestone M1 — Lossless Editing Feasibility

Status: active.

## Question

Can Marksplice provide source-preserving structural editing for GitHub Flavored Markdown (GFM), using Goldmark for semantic understanding while preserving untouched source byte-for-byte across useful edits?

M1 is a feasibility gate. It does not freeze the full public API or implement the document graph. GFM is the only Markdown syntax profile in scope.

## Deliverables

- repository bootstrap with Apache-2.0 licensing and attribution;
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

## Feasibility matrix

Each row must eventually prove GFM semantic recognition, exact source mapping needed for the planned edit, unchanged-byte preservation, and relevant malformed/ambiguous behavior. GFM-specific constructs are first-class requirements rather than optional extensions.

| Construct | M1 expectation | Initial state |
| --- | --- | --- |
| ATX headings | recognize, map, rename content while preserving markers/spacing | initial top-level proof passes |
| Setext headings | recognize, map, rename content while preserving underline/spacing | initial top-level proof passes |
| Paragraphs | recognize, map, replace | initial proof passes |
| Unordered lists | recognize markers/items and edit representative item | planned |
| Ordered lists | preserve numbering/style outside edits | planned |
| Nested lists | preserve nesting/indentation outside edits | planned |
| GFM task lists | update task state without list normalization | initial proof passes; one-byte state patch preserves marker/indentation/case on no-op |
| GFM tables | update representative cell without reformatting untouched table source | planned |
| Fenced code | preserve fence delimiter style/length when untouched | planned |
| GFM strikethrough | recognize and preserve delimiters outside edits | published one/two-tilde parser cases pass; source-preserving edit planned |
| GFM extended autolinks | recognize source/destination boundaries | published parser cases pass, including `mailto:`/`xmpp:` and bare-FTP rejection; source-preserving model planned |
| Inline links | recognize destination/source boundaries | planned |
| Reference links | preserve reference style | planned |
| Reference definitions | recognize and update representative definition | planned |
| YAML front matter | preserve unrelated fields/source | planned |
| TOML front matter | preserve unrelated fields/source | planned |
| HTML comments/anchors/opaque regions | preserve conservatively | planned |
| LF | unchanged-byte proof | initial proof passes |
| CRLF | unchanged-byte proof | initial proof passes |
| CR | unchanged-byte proof | heading proof passes through byte-stable parser shadow normalization |
| Unicode | byte-range correctness | planned |
| Malformed/ambiguous cases | deterministic fail-closed behavior where mutation cannot be proven safe | invalid ranges/targets covered; syntax matrix pending |
| Inline emphasis/strong | GFM-compatible semantic model before structural editing | parser conformance gate passes; source-preserving structural model planned |

Update this table as tests establish evidence. Do not mark a row complete based only on parser support or AST inspection.

## Exit criteria

M1 is green only when representative tests demonstrate that the combined semantic + source-map model is workable without whole-document serialization and without unacceptable special-case coupling to Goldmark internals.

At minimum, the completed matrix must include all constructs listed above or document an explicit, reviewed reason a construct moved to a later milestone.

After the gate is green, design the stable public API and broader structural operations from the evidence gathered here rather than from parser-specific abstractions.
