# Milestone M3 — Heading Public API

Status: green — heading public API passed.

## Goal

Extend the M2 typed-detail and named-operation pattern to top-level GFM headings without promoting the broader M1 feasibility model into the public API.

M3 is deliberately narrow. It promotes only the caller-facing heading semantics already proven by M1 and required by source-preserving rename.

## Requirements

The public heading family must:

- keep `Node` limited to common snapshot identity and promoted kind;
- expose immutable typed heading detail through `Document.Heading(NodeID)`;
- expose heading level and source syntax through Marksplice-owned public values;
- define one precise range contract for the exact heading-content bytes changed by rename;
- support a named `PrepareRenameHeading` operation returning the existing opaque `ChangeSet`;
- preserve ATX markers, optional closing markers, Setext underlines, indentation/spacing, line endings, and all unrelated bytes;
- preserve the M2 public error categories and stale-source behavior;
- fail closed when replacement bytes would cease to represent the same supported heading shape;
- keep Goldmark and internal M1 types out of exported signatures.

Only top-level headings are promoted. Container-prefixed heading mapping remains outside this milestone.

## Public shape

M3 adds:

- `HeadingStyle` with `HeadingStyleUnknown`, `HeadingStyleATX`, and `HeadingStyleSetext`;
- immutable `Heading` detail with `ID()`, `Range()`, `Level()`, and `Style()`;
- `Document.Heading(NodeID) (Heading, bool)`;
- `Document.PrepareRenameHeading(NodeID, []byte) (ChangeSet, error)`.

`Heading.Range()` is intentionally the exact content span replaced by `PrepareRenameHeading`. It does not mean the complete heading source span. ATX marker bytes, optional closing markers, Setext underline bytes, and following line endings remain outside this range.

The public `HeadingStyle` values are Marksplice-owned. Their representation is not an alias or compatibility promise for `internal/splice.HeadingStyle` or `internal/source.HeadingStyle`; conversion is explicit at the public boundary.

## Architecture and reuse

M3 does not add a parser, source mapper, or mutation engine. It reuses the M1 top-level heading observation, exact content mapping, minimal patch preparation, candidate reparse, source-style validation, and stale-source conflict handling.

The root package remains an API boundary rather than a second structural model. It translates only reviewed heading facts into immutable public values and delegates mutation safety to the established internal implementation.

Generic public enumeration is also guarded so a future internal container heading is not automatically promoted merely because it shares `KindHeading`.

## Test strategy and evidence

The public external-package tests cover:

- ATX level/style detail;
- Setext level/style detail;
- exact rename-content ranges;
- ATX marker/spacing and CRLF preservation;
- Setext underline/spacing and CRLF preservation;
- byte identity before and after the changed content span;
- typed lookup rejection for non-heading and missing IDs;
- public error categories for missing IDs, wrong target kinds, empty/multiline replacements, and a Setext replacement that would change block structure;
- deterministic zero-value `Heading` behavior.

TDD evidence for the first slice was explicit: the focused public tests first failed to compile because `HeadingStyle`, `Document.Heading`, and `PrepareRenameHeading` did not exist; after the minimal public implementation, the focused tests and full Go regression suite passed.

## Devil's advocate review

### Risk: exposing the wrong range semantics

The internal heading node carries both a complete mapped source range and a content range. Publishing the complete range as a generic-looking heading span would make rename callers believe marker/underline bytes are mutable or interchangeable with content.

Mitigation: `Heading.Range()` is defined only as the exact content span used by rename, mirroring the operation-oriented range discipline established by M2.

### Risk: public enum coupled to feasibility internals

Aliasing the internal heading-style enum would make M1 representation choices part of the compatibility contract.

Mitigation: M3 owns a separate public enum and uses explicit conversion.

### Risk: Setext rename changes Markdown block structure

Certain single-line replacements can turn a Setext content line plus underline into another GFM structure.

Mitigation: M3 delegates to the M1 candidate reparse and source-style revalidation, which rejects the change unless the same heading level/style and source position remain provable.

### Risk: future container headings leak through generic enumeration

If the internal parser later observes headings inside blockquotes or other containers, `KindHeading` alone would be insufficient to prove they share M3's top-level source contract.

Mitigation: public enumeration and typed lookup require top-level heading status; container heading semantics need their own proof before promotion.

## Exit decision

M3 is green. Focused public heading tests and the full Go regression/race suites pass; vet, Staticcheck, golangci-lint, govulncheck, and Gitleaks report no findings; and generated package documentation exposes only Marksplice-owned or standard-library public types.

The milestone promotes only reviewed top-level heading semantics. Container headings, unrelated M1 node kinds, internal source ranges, parser types, and generic mutation abstractions remain unpromoted. Future structural families should continue to use the typed-detail and named-operation pattern unless a separately reviewed abstraction proves clearer without weakening source preservation or fail-closed validation.
