# Marksplice

Source-preserving GitHub Flavored Markdown manipulation for Go.

Marksplice is an open-source Pure-Go library for understanding and structurally editing GitHub Flavored Markdown (GFM) while preserving untouched source bytes whenever an operation does not semantically require broader changes.

## Status

Marksplice has a green retrospective M0 repository-bootstrap record and has passed engineering milestones M1 through M33: lossless-editing feasibility, the public-API foundation, reviewed block/inline/link APIs, conservative metadata/HTML APIs, a hierarchical section model, bounded immutable-snapshot source reading, simple inline-image destination editing, and reviewed structural mutations. The current surface covers top-level paragraphs/headings, M1-proven leaf single-line list items and GFM task markers, mapped non-empty table cells, supported single-line fenced code, simple strikethrough/code-span/emphasis/strong spans, simple inline links/reference definitions/autolinks and inline images, unique simple leading YAML/TOML front-matter fields, simple HTML comments/anchors, source-bound section hierarchy/ranges anchored to heading IDs, copied reads of any valid snapshot-local byte range, fail-closed section remove/body/subtree replacement, same-level section sibling insertion/movement, direct-child section append, same-shape complete list-item subtree insertion around complete compatible anchors, atomic movement of complete supported list-item subtrees around complete compatible anchors, direct-child list-subtree append validated in host context, complete list-item subtree replacement preserving the external sibling/parent relationship, complete removal of fully supported list-item subtrees, semantic `HasChildren` plus snapshot-local immediate supported `ParentID`/source-ordered `ChildIDs` navigation for promoted list items, and an exact public `SubtreeRange()` for complete supported list-item subtrees while preserving the content-only `Range()` contract. The published GitHub Flavored Markdown 0.29 specification is the project's single normative Markdown syntax profile.

The public API remains intentionally narrow rather than feature-complete. Additional syntax families will be promoted only after their caller-facing semantics and source-preserving operations are reviewed.

## Design principles

- follow the published GitHub Flavored Markdown 0.29 specification as the single normative Markdown syntax profile;
- parse GFM for semantic understanding without implying whole-document normalization;
- preserve untouched author choices such as heading/list/fence styles, whitespace, line endings, and other lexical trivia;
- bind prepared edits to exact source snapshots and reject stale application;
- keep Goldmark behind an internal adapter and expose only Marksplice-owned types;
- keep filesystem, network, MCP, and host authorization concerns outside the core library.

See [`docs/architecture.md`](docs/architecture.md) for durable design decisions, [`docs/gfm-conformance.md`](docs/gfm-conformance.md) for the normative Markdown/conformance policy, and [`docs/milestones/`](docs/milestones/) for milestone evidence and exit decisions.

## Public API foundation

The current public surface deliberately promotes only reviewed, source-mapped capabilities: immutable snapshots, opaque snapshot-scoped node identities, the reviewed block/inline/link/image families above, unique simple leading front-matter scalar fields, narrowly mapped HTML comments/anchors, sections derived from promoted heading identities, and copied bounded reads through `Document.SourceRange`. Section views expose exact direct-body and complete-subtree ranges plus parent hierarchy without introducing a second node-ID namespace; `PrepareRemoveSection`, `PrepareReplaceSectionBody`, `PrepareReplaceSection`, `PrepareInsertSectionBefore`/`After`, `PrepareMoveSectionBefore`/`After`, and `PrepareAppendSectionChild` provide the reviewed section mutation set. Promoted supported list items retain content-only `ListItem.Range()` replacement semantics; `HasChildren()` distinguishes leaf items from single-line-head parents and `ParentID()` returns the immediate parent identity when that parent is itself supported/promoted. Parent IDs are resolved once during parse from existing physical-line anchors and actual snapshot node IDs; unsupported complex parents are not synthesized. Private parse-time physical-line ownership powers `PrepareInsertListItemBefore`/`After` and atomic `PrepareMoveListItemBefore`/`After`. M26 allows the insertion anchor to be a complete supported parent subtree; M29 further allows the caller fragment to be one complete supported same-shape list-item subtree after standalone ownership/completeness proof. M27 applies the same complete-anchor rule to atomic moves, and M28 allows the moved source itself to be a complete supported subtree. `after` uses the private subtree end, candidate validation proves semantic siblinghood with the anchor, and every inserted or moved descendant keeps its subtree-relative mapping/parentage, overlapping source/anchor subtrees fail closed, and move no-ops remain subtree-aware and snapshot-bound. `PrepareRemoveListItem` removes a leaf line or, when M24's private completeness proof succeeds, the complete supported subtree. `PrepareAppendListItemChild` accepts caller-owned child syntax and proves the immediate parent relation in the host GFM candidate, so variable marker width/post-marker spacing and container prefixes are handled without fixed indentation rules. M24 additionally allows append on an existing parent only when private semantic direct-child counts prove that its complete descendant subtree is represented by supported list items; an unsupported descendant fails closed instead of producing a guessed insertion boundary. Parent content replacement is allowed and validates descendants. Structural mutations fail closed when candidate parsing cannot preserve required surviving mappings, never synthesize whitespace repairs, and never renumber lists. Typed mutation details expose only operation-specific source ranges and reviewed semantic state; named operations prepare minimal source-bound changes. Unsupported semantic shapes such as normalized-space code spans, compound emphasis, ambiguous/complex metadata, opaque HTML, multiline or multi-block non-list-child list items, and container-aware sections remain internal rather than appearing publicly actionable. Internal M1 syntax coverage remains broader than the public API and is not automatically a compatibility commitment.

## Development

The module path is:

```text
github.com/zoster81/marksplice
```

Run the standard checks with the Go toolchain selected for the repository:

```text
go test ./...
go test -race ./...
go vet ./...
```

Additional static, vulnerability, and secret-scanning checks are described in [`CONTRIBUTING.md`](CONTRIBUTING.md).

## Author

Marksplice was created by Giovanni Riccobene (`zoster81`).

## License

Licensed under the Apache License, Version 2.0. See [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE).

Goldmark is an MIT-licensed third-party dependency. Exact dependency versions are recorded in `go.mod` and `go.sum`.
