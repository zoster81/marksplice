# Milestone M30 — List-Item Child Subtree Append

Status: green — complete direct-child list-item subtree append passed.

## Goal

Extend `PrepareAppendListItemChild` from the M21/M24 single-leaf child to one complete supported direct-child subtree without inventing GFM indentation rules or weakening M24 subtree completeness.

M30 adds no public method or type. A leaf child remains the one-node degenerate subtree.

## Host-context proof

M30 intentionally does not reuse M29 standalone-fragment parsing. Child indentation depends on the host parent's marker width, post-marker spacing, and container prefix; a valid child under a wide ordered marker may not be meaningful as a root document.

The caller still owns indentation, numbering, marker choice, line endings, and separators. Marksplice synthesizes none of them.

Insertion remains at `parent.ListSubtreeEnd`. Marksplice builds one zero-width patch, renders the host candidate, and runs one full `splice.Parse`. Candidate list mappings are then derived from that parsed `Document`, avoiding a second parser pass.

The candidate item beginning at the insertion offset must:

- be a supported list item with the requested parent anchor;
- resolve `ListParentID` to the candidate target parent;
- pass `ListSubtreeComplete`;
- own exactly `[insertAt, insertAt+len(fragment))` through `ListSubtreeEnd`;
- preserve the caller fragment byte-for-byte.

This rejects multiple direct roots, trailing bytes outside the subtree, unsupported descendants, invalid indentation, and unsafe joins.

All original supported items still pass the shared survivor proof. Only the requested parent receives a `+1` direct-child-count delta, regardless of descendant count. The candidate supported-item total must equal the original count plus the inserted subtree ID count.

## Complexity

For document size `n`, fragment size `k`, original supported list count `l`, and inserted subtree size `f`, preparation remains O(n+k) for candidate construction/parsing plus O(l+f) validation. There is no standalone child parse, second candidate parse, recursive hierarchy walk, persistent second index, renderer, or generic batch API.

## Devil's advocate review

### Multiple sibling roots

Two valid direct children could be mistaken for one subtree.

Mitigation: the inserted root's `ListSubtreeEnd` must equal the exact fragment end.

### Unsupported descendant

A supported root can hide a complex unsupported semantic child.

Mitigation: M24 semantic child counts and leaf-up completeness remain authoritative; the root must be `ListSubtreeComplete`.

### Standalone rejection of valid indentation

Deep child indentation can be invalid standalone while correct under the host parent.

Mitigation: M30 validates only the complete host candidate and requires direct semantic parentage there.

### Redundant candidate parsing

A full candidate parse followed by the old parser-only mapping helper would duplicate work.

Mitigation: survivor mappings are derived directly from the already parsed candidate `Document`.

## TDD evidence

Focused tests were added before implementation. The initial run rejected every valid child-subtree case with `ErrInvalidReplacement`, proving the old `+1 leaf` restriction remained active.

After implementation, tests pass for a leaf parent receiving a child/grandchild subtree, append after existing deep descendants, wide ordered-parent CRLF/Unicode source, preserved `ParentID` and `HasChildren`, and rejection of multiple direct roots, trailing source, unsupported descendants, and same-level roots. Focused M21–M30 list regressions pass together.

## Exit decision

M30 is green. Verification passes native `gofmt`, focused regressions, five consecutive full `go test ./...` runs, race detection, vet, build, generated package documentation, the published GFM 0.29 gate, Staticcheck, standard golangci-lint with zero issues, production-only gocyclo/unparam with zero issues, govulncheck with no vulnerabilities, and Gitleaks with no leaks.

Final text/diff/repository-integrity checks are recorded with the working-tree review. M30 changes no parser dependency, NodeID algorithm, public type, error sentinel, normalization policy, or complexity class.
