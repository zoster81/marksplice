# Milestone M10 — Bounded Source Reading

Status: green — bounded snapshot reading passed.

## Goal

Turn the exact byte ranges established by the public typed-detail and M9 section APIs into directly consumable, snapshot-bound reads without exposing Marksplice's internal source buffer or introducing filesystem, workspace, or query-language responsibilities.

M10 adds one universal read primitive rather than separate source-returning methods for every structural family.

## Public contract

`Document.SourceRange(Range) ([]byte, bool)` returns a copy of the exact half-open byte range from the immutable parsed source snapshot.

The operation succeeds for any ordered range contained in the snapshot, including an empty range. It returns `nil, false` for a nil document or a range with a negative start, reversed bounds, or an end beyond the snapshot.

The returned slice never aliases the document's stored source. Mutating either the caller's original input after `Parse` or bytes returned by `SourceRange` cannot mutate the immutable document snapshot.

The method deliberately accepts the existing public `Range` type. In particular, callers can pass `Section.Range()` to read a complete section subtree or `Section.BodyRange()` to read the direct section body. Operation-oriented typed-detail ranges can be consumed in the same way when a caller needs their exact source bytes.

## Architecture and complexity

The internal document already owns one immutable copy of the parsed source. M10 does not add a second source representation, cache, parser pass, source scan, or index.

The internal `splice.Document.SourceRange` validates bounds against the snapshot and copies only the requested bytes. The public wrapper translates the Marksplice-owned range into the internal range type and delegates.

For a requested span of `k` bytes, time and returned-memory cost are O(k). Reads do not depend on total document node or section counts and do not reparse Markdown.

## Safety boundary

M10 is a read-only capability. It does not weaken stale-source mutation checks or expose a mutable view of internal source storage.

A valid range is always interpreted against the exact `Document` snapshot receiving the call. Marksplice does not attempt to apply a range from another snapshot or infer whether caller-supplied offsets originated from the same document; out-of-bounds input fails closed while in-bounds byte offsets have their ordinary snapshot-local meaning.

## Devil's advocate review

### Risk: returned bytes alias immutable document state

Returning `d.source[start:end]` directly would let a caller mutate the parsed snapshot and invalidate node IDs, ranges, fingerprints, and prepared-change assumptions.

Mitigation: every successful read returns a fresh copy. Focused tests mutate returned bytes and prove subsequent reads are unchanged.

### Risk: malformed ranges panic or expose the wrong bytes

Slicing before validation could panic for negative, reversed, or oversized bounds.

Mitigation: range validity is checked against the internal snapshot length before slicing. Invalid ranges return `nil, false` deterministically.

### Risk: convenience APIs multiply around every structural type

Adding `SectionBody`, `SectionSource`, `HeadingSource`, and similar methods would duplicate the same byte-copy operation and expand the public surface unnecessarily.

Mitigation: one `SourceRange` primitive composes with all reviewed range-bearing public types.

### Risk: bounded reading becomes a filesystem authority

A document read API could drift into path resolution or workspace traversal.

Mitigation: M10 operates only on bytes already copied into one parsed `Document`. It performs no filesystem, network, or multi-document access.

## Evidence and exit decision

M10 began with focused public tests that failed to compile because `Document.SourceRange` did not exist.

After the minimal implementation, focused tests pass for copied snapshot bytes, caller/input mutation isolation, CRLF and Unicode content, complete section-subtree reads, direct section-body reads, invalid range rejection, valid empty ranges, and nil-document behavior.

M10 is green. The capability is intentionally small: it closes the bounded-reading gap exposed by M9 without adding another identity model, parser behavior, or source cache. Full repository verification is recorded with the completed milestone working-tree review.
