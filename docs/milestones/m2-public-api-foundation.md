# Milestone M2 — Public API Foundation

Status: green — public API foundation passed.

## Goal

Turn the M1 feasibility implementation into a small, durable Marksplice-owned public API without exposing Goldmark types or freezing the internal proof model.

M2 is an API-foundation milestone, not a syntax-expansion milestone. The source-preservation, snapshot, parser-isolation, and fail-closed invariants established by M1 remain mandatory.

## Requirements

The first public surface must:

- parse Markdown bytes into an immutable document snapshot;
- expose snapshot-scoped structural node identities using Marksplice-owned public types;
- keep public node data limited to durable common semantics rather than exposing feature-specific M1 fields or ambiguous generic source ranges;
- support deterministic node lookup by `NodeID`;
- expose prepared source-bound changes without exposing internal patch implementation types;
- preserve `errors.Is`-compatible public error identities for node lookup, invalid targets/replacements, and stale-source conflicts;
- prove one representative source-preserving mutation through the public API before additional mutation families are promoted;
- keep all Goldmark and `internal/*` types out of exported signatures.

## First vertical slice

The first slice contains:

- `Parse([]byte) (*Document, error)`;
- `Document.Nodes() []Node`;
- `Document.Node(NodeID) (Node, bool)`;
- public `NodeID`, a deliberately small initial `Kind` set, and immutable `Node` summary accessors;
- an opaque public `ChangeSet` with `Apply([]byte)`;
- `Document.PrepareReplaceParagraph(...)` as the representative mutation;
- public sentinel errors mapped to the established internal failure categories.

The slice must prove that mutating the caller's input after `Parse` does not mutate the document snapshot, that modifying a returned node slice cannot mutate document state, and that a prepared change rejects stale source.

The first slice now passes external-package tests. The initial public kind set intentionally contains only headings and top-level paragraphs; internal tasks and nested/container paragraphs remain unpromoted even though the M1 implementation can observe them.

## Second vertical slice — typed paragraph detail

The second slice tests the pattern for syntax-specific public information without widening generic `Node`. `Document.Paragraph(NodeID)` returns immutable detail only for a promoted top-level paragraph. `Paragraph.Range()` is a Marksplice-owned half-open byte range whose meaning is explicit: it is the exact paragraph span used by `PrepareReplaceParagraph`, including preserved trailing horizontal spaces and excluding the immediately following line ending.

A paragraph inside a blockquote or another container is not promoted by this slice because M1 did not prove the same replacement boundary for container paragraphs. The public mutation rejects such a target even if a caller somehow obtains its snapshot-local internal identity.

This typed-detail pattern is the candidate for future syntax-specific public information: common `Node` stays small, while each promoted detail defines its own source-position and semantic contract.

## API shape

`Node` is intentionally not an exported-field copy of the M1 internal node struct. Its first stable common facts are identity and kind. M1 range fields are operation-oriented and are not semantically uniform across all node kinds, so M2 does not promote them as a generic public range contract. Syntax-specific positions and details such as paragraph spans, heading level, table column, link destination, task state, front-matter style, and HTML attribute details require later typed APIs or focused accessors after their public semantics are reviewed.

The initial public kind set promotes only the node categories needed by the first slice. Internal M1 categories are not automatically public API commitments; additional kinds are promoted when their public semantics are reviewed.

`NodeID` is an opaque, comparable value and is deterministic only within one source snapshot. It is not a persistent identity across arbitrary edits or reparses. `NodeID.String()` exists for diagnostics only; M2 defines no public parser, serialization, persistence, or round-trip contract for that string.

`Document` owns an immutable parsed snapshot. Public read methods return values/copies rather than writable references into internal state.

## Error contract

Public sentinel errors identify stable categories; error strings are diagnostic rather than API contracts. Callers should use `errors.Is`.

The initial categories are:

- node not found;
- invalid target kind;
- invalid replacement;
- source snapshot conflict.

## Complexity and allocation constraints

Parsing may perform the M1 snapshot copy and structural indexing. Public node lookup must reuse the internal snapshot index rather than add another linear scan. `Nodes()` may allocate one public summary slice, but must not copy the complete source or feature-specific data for each node.

Prepared paragraph replacement retains M1's candidate validation and stale-source safety.

## Devil's advocate review

### Risk: freezing M1 node fields or range semantics

Exporting every M1 node field, or exposing its operation-specific ranges as one generic span contract, would make inconsistent feasibility details part of the compatibility contract and make later typed APIs harder to design.

Mitigation: expose only reviewed common semantics in `Node`; promote syntax-specific positions and details separately.

### Risk: wrapper duplication

A public wrapper could duplicate the source snapshot or maintain a second node index.

Mitigation: `Document` wraps the internal immutable document; lookup delegates to the existing index, and public summaries are created only at read boundaries.

### Risk: generic mutation API too early

A generic `Replace(NodeID, ...)` would hide syntax-specific safety rules and encourage semantics that M1 did not prove.

Mitigation: promote one named operation, paragraph replacement, then evaluate the public pattern before exposing additional mutation families.

### Risk: callers treating `NodeID` as durable across revisions

Snapshot-derived IDs can change after source changes.

Mitigation: keep the representation opaque, state snapshot scope in type documentation and tests, and make `String()` diagnostic only. Future cross-revision identity or persisted selectors, if needed, must be separate concepts.

## Exit decision

M2 is green. External-package tests cover the public parse/read/lookup/change lifecycle, snapshot immutability, filtered node promotion, precise typed paragraph range semantics, public error categories, zero values, and stale-source rejection. Generated `go doc` output contains only Marksplice-owned or standard-library public types and no Goldmark or `internal/*` implementation types.

The root package owns its sentinel errors, `NodeID` is opaque and snapshot-scoped, node enumeration reuses lightweight internal summaries rather than copying the full M1 node union, and the representative paragraph mutation preserves M1 minimal-patch and source-conflict behavior.

This is an API-foundation result, not feature completeness. Only reviewed kinds are promoted, only top-level paragraph detail/mutation is public, and additional structural families must be introduced in later milestones through the typed-detail and named-operation pattern rather than by exposing the M1 implementation wholesale.
