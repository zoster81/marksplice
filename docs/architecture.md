# Marksplice Architecture

Status: source of truth for durable architecture decisions.

## Mission

Marksplice is a Pure-Go library for source-preserving structural GitHub Flavored Markdown (GFM) manipulation. GFM is the project's single normative Markdown syntax profile. Parsing exists to understand source; it does not authorize normalization of untouched source.

For ordinary edits to an existing document, bytes outside the intended changed spans must remain byte-identical whenever the operation does not semantically require a broader change.

## Architecture

```text
original Markdown bytes
        |
        +--> semantic parser adapter (Goldmark initially)
        |         |
        |         v
        |    semantic observations
        |
        +--> Marksplice lossless source mapping
                  |
                  v
          Marksplice document model
                  |
                  v
        validated structural change
                  |
                  v
            minimal byte patches
```

The semantic parser and lossless mapping have different responsibilities. Goldmark may identify semantic constructs, but Marksplice owns lexical details and exact mutation boundaries needed to preserve source.

## Package boundaries

Current internal boundaries:

- `internal/parser/goldmark`: Goldmark-specific parsing and AST traversal. No Goldmark type may cross this adapter boundary.
- `internal/source`: snapshot fingerprints, byte ranges, validated patches, stale-source conflict detection, and patch application.
- `internal/splice`: feasibility-level document model that combines semantic observations with source snapshots and prepares structural edits.

Milestone M1 has passed its feasibility gate. The root public package remains intentionally small while the post-M1 public API is designed from the evidence gathered by the feasibility implementation; internal M1 types are not automatically promises of final public API shape.

Within the internal implementation, keep orchestration separate from syntax-specific proof logic. Shared mutation plumbing may centralize target lookup, simple replacement preconditions, candidate patch construction, and candidate parsing, while source mappers and semantic validators remain typed and feature-specific when their safety invariants differ. Likewise, shared lexical primitives belong in focused helpers rather than being duplicated across block, inline, link, front-matter, or HTML mapping code.

## Source model

Source is represented as bytes. Source ranges use half-open byte offsets `[start,end)` into a specific immutable source snapshot.

A structural node needs enough snapshot-local information to support deterministic targeting, conceptually including:

```text
id
kind
sourceRange
parentId
children
sourceFingerprint
properties
```

Human-readable labels such as heading text are not sufficient identities because duplicates are valid Markdown.

Node identities are snapshot-scoped, deterministic, and derived from source-bound structural facts. They are not durable identities across arbitrary document revisions.

An immutable parsed document may maintain a derived snapshot-local `NodeID` index so structural targeting does not require a linear scan for every operation. Building that index must reject duplicate IDs rather than silently choosing one node; the ordered node collection remains the source of structural iteration order.

## Mutation model

A prepared change contains the source fingerprint it was created against and one or more non-overlapping minimal byte patches.

Application rules:

1. fingerprint the supplied source;
2. reject the operation with an explicit conflict if the snapshot differs;
3. validate every patch range and ordering;
4. apply patches without rendering unrelated source;
5. preserve all bytes not covered by changed ranges.

Batch editing must reject overlapping or ambiguous patches rather than relying on application order to pick a winner. Efficient batch application should sort validated patches once and apply them without repeated whole-document rescans.

M1 deliberately reparses a prepared candidate snapshot when a mutation must prove that its semantic/source boundary still exists after replacement. This is a conservative O(n) safety oracle for one prepared mutation, not the desired implementation of a future multi-edit batch. Post-M1 batch planning should reuse parsed/indexed state or otherwise amortize validation without weakening the fail-closed semantic checks established by M1.

## Line endings, Unicode, and encoding

Marksplice core operates on Markdown bytes and must not normalize LF, CRLF, CR, or Unicode content as a side effect of unrelated edits. Byte ranges, not rune indexes, define mutation boundaries. The semantic parser may use a same-length shadow view for parser compatibility (for example, isolated CR mapped to LF) only when byte offsets remain exactly aligned with the original source.

Encoding/BOM preservation belongs to a host that provides decoded Markdown bytes plus an encoding policy, unless Marksplice later introduces an explicit encoding-aware layer. The core must not silently guess and rewrite file encodings.

## GFM conformance and Goldmark boundary

GitHub Flavored Markdown (GFM) is the only Markdown dialect Marksplice targets. CommonMark is inherited as GFM's base syntax, not exposed as a separate Marksplice mode. The normative source hierarchy, approved specification snapshot, conformance procedure, and update policy are defined in [`gfm-conformance.md`](gfm-conformance.md).

`github.com/yuin/goldmark` is the selected semantic parser implementation. It is configured for GFM with narrowly scoped Marksplice compatibility behavior where required by the approved GFM contract. Goldmark remains an implementation dependency, not part of Marksplice's public contract.

Rules:

- do not fork or copy Goldmark merely to implement Marksplice;
- do not expose its AST or parser-specific types publicly;
- configure the semantic parser for GFM rather than assembling an ad hoc mixture of Markdown extensions;
- follow the conformance hierarchy and specification-update gate in `docs/gfm-conformance.md`;
- treat parser-library divergences as adapter/model gaps to resolve, not as reasons to expose another Markdown dialect or silently waive conformance cases;
- supplement it with Marksplice-owned lexical/source mapping wherever semantic AST information is insufficient for source-preserving edits;
- do not serialize the Goldmark AST as the ordinary existing-document edit path;
- do not add non-GFM syntax extensions to the default parser profile merely because Goldmark supports them.

GFM's disallowed-raw-HTML tag filtering is an HTML-rendering requirement. Marksplice core currently parses, models, validates, and edits Markdown source rather than rendering HTML; if HTML rendering becomes a Marksplice responsibility, GFM rendering conformance including tag filtering becomes part of that feature's acceptance criteria.

YAML and TOML front matter are document-envelope metadata, not additional Markdown dialects. The GFM body continues to be parsed through the normal Goldmark profile without enabling a front-matter extension. Marksplice may recognize a leading front-matter envelope in its own source layer and exclude that proven envelope from the GFM structural view while preserving all envelope bytes not explicitly edited. M1 deliberately recognizes only closed byte-zero `---`/`+++` envelopes that contain at least one unique scalar field whose lexical value boundary can be proven safely; ambiguous, duplicate-only, complex-only, non-leading, or unclosed shapes remain ordinary GFM source rather than being guessed as metadata.

The detailed capability/ownership matrix is maintained in `docs/goldmark-capability-matrix.md`.

## Structural semantics

The target model includes documents, sections, headings, paragraphs, lists/list items/tasks, blockquotes, fenced code, inline code, tables, thematic breaks, front matter, links/references/images, explicit anchors, and opaque preserved HTML/unsupported regions.

A section is governed by a heading until the next heading of equal or higher level. The model must distinguish the heading, direct section body, and complete subtree.

Milestone M1 intentionally implemented only the subset required to prove the architecture; later milestones may expand the structural model without weakening the established source-preservation invariants.

## Safety boundaries

Prepared mutations fail closed on stale source. Malformed ranges, overlapping patches, invalid structural targets, or ambiguous targeting must produce deterministic errors rather than best-effort mutation.

Marksplice core does not perform arbitrary filesystem traversal, network requests, or command execution. Future multi-document relationship resolution must be bounded and constrained to caller-authorized document sets or roots.

External URLs remain data unless an explicit caller outside core chooses to act on them.

## Complexity goals

Parsing and structural indexing should be linear or near-linear in source size where practical. Mutation planning should avoid quadratic rescanning. Callers should eventually be able to impose byte, node, depth, relationship, and output budgets rather than relying on hidden global limits.

## Post-M1 public API direction

M1 has demonstrated the feasibility requirements that gated broader API design:

- semantic parsing through the internal Goldmark adapter;
- exact source ranges sufficient for useful local edits;
- byte preservation outside changed spans across representative structures;
- stale-source conflict rejection;
- deterministic malformed/ambiguous behavior;
- acceptable complexity and testability of the combined semantic + lossless model.

Passing the gate does not freeze the current internal feasibility API. The stable public API and broader structural operations should now be designed from these proven invariants, keeping Goldmark details internal and preserving the minimal-patch/fail-closed model.
