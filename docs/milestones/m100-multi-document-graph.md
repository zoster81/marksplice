# M100 — Multi-document graph

## Status

Complete. Focused TDD, architecture/performance review, full implementation-tree release-quality verification, exact pinned GFM conformance, corrected explicit cross-package coverage, and the final documented-tree regression/static/security/text/private-boundary/Git freeze are green. No commit or push is authorized by this milestone record.

## Objective

Build deterministic multi-document relationship intelligence over an explicit caller-provided set of immutable Marksplice documents without turning Marksplice into a filesystem crawler, URL client, path normalizer, or authorization layer.

M100 adds graph-level:

- caller-defined logical document identities;
- resolved cross-document edges over M99 relationships;
- local `#fragment` self-edges without resolver involvement;
- optional cross-document fragment targets using M98 semantics;
- outgoing-edge and backlink navigation;
- graph reachability;
- direct related-document discovery.

M100 remains deliberately narrower than M101. It does not diagnose unresolved relationships, missing documents, malformed resolver policy, missing/ambiguous fragments, or orphan-policy violations after a graph has been built. It produces the explicit resolved graph facts M101 can validate.

## Public contract

M100 adds:

```go
type DocumentKey string

type GraphDocument struct {
    Key      DocumentKey
    Document *Document
}

type DocumentResolution struct {
    Target   DocumentKey
    Fragment string
}

type DocumentResolver func(
    source DocumentKey,
    relationship LinkRelationship,
) (DocumentResolution, bool)

type GraphEdge struct { /* immutable */ }
func (e GraphEdge) SourceDocument() DocumentKey
func (e GraphEdge) TargetDocument() DocumentKey
func (e GraphEdge) Relationship() LinkRelationship
func (e GraphEdge) Fragment() (string, bool)
func (e GraphEdge) FragmentTarget() (FragmentTarget, bool)

type DocumentGraph struct { /* immutable */ }
func BuildDocumentGraph(documents []GraphDocument, resolver DocumentResolver) (*DocumentGraph, error)
func (g *DocumentGraph) DocumentKeys() []DocumentKey
func (g *DocumentGraph) Document(key DocumentKey) (*Document, bool)
func (g *DocumentGraph) Edges() []GraphEdge
func (g *DocumentGraph) Outgoing(key DocumentKey) ([]GraphEdge, bool)
func (g *DocumentGraph) Backlinks(key DocumentKey) ([]GraphEdge, bool)
func (g *DocumentGraph) ReachableFrom(key DocumentKey) ([]DocumentKey, bool)
func (g *DocumentGraph) RelatedDocuments(key DocumentKey) ([]DocumentKey, bool)
```

M100 also adds `ErrInvalidGraph` for malformed explicit graph inputs and resolver results.

`DocumentKey` is opaque caller data. Marksplice does not normalize it or interpret it as a path, URI, repository identity, or durable document ID.

## Resolution and authority model

`BuildDocumentGraph` validates the complete caller-provided document set before invoking the resolver:

- every key must be non-empty;
- keys must be unique;
- every document must be a real initialized immutable `Document` snapshot;
- an empty document set is valid.

Every input document then contributes its M99 `LinkRelationships` in source order.

A relationship whose semantic destination starts with `#` is local to the source document. It becomes a self-edge directly and **never invokes `DocumentResolver`**.

Every other relationship is offered synchronously to the caller-provided resolver in document-input order and relationship source order:

- returning `false` leaves that relationship outside the graph;
- returning `true` must identify a non-empty target key already present in the explicit input set;
- an empty or unknown target fails the entire build with `ErrInvalidGraph` rather than silently widening authority;
- optional `DocumentResolution.Fragment` is interpreted only against that already-supplied target snapshot using M98 fragment rules.

The resolver is a policy/authorization hook, not a retained service. Marksplice calls it only during `BuildDocumentGraph`, never stores it in `DocumentGraph`, and performs no filesystem access, path canonicalization, URL fetch, network request, command execution, or workspace traversal. A caller may choose to map an external-looking destination to a supplied document key, but that is explicit caller policy outside Marksplice's authority.

## Fragment semantics

Local edges reuse the exact M99 relationship fragment facts already derived from M98. `GraphEdge.Fragment` is therefore the relationship destination itself, including its leading `#`, and `FragmentTarget` is present only when M99/M98 resolved exactly one target.

For non-local relationships, the resolver may provide an optional fragment with or without a leading `#`. M100 evaluates it against the resolved target document with the same M98 normalization and target rules:

- percent-decoded heading fragments can resolve;
- promoted explicit HTML anchors can resolve;
- missing, ambiguous, or malformed fragments keep the document edge but expose no `FragmentTarget`;
- M100 does not reinterpret these states as diagnostics; M101 owns that distinction and reporting.

The fragment string returned by `GraphEdge.Fragment` remains the exact resolver-provided value for cross-document edges.

## Graph representation and determinism

`DocumentGraph` is immutable after construction.

Persistent graph state is intentionally compact:

- one input-order `[]DocumentKey`;
- one key-to-document map containing immutable document pointers;
- one source-ordered `[]GraphEdge`;
- outgoing and backlink maps containing **edge indices**, not duplicate `GraphEdge` copies.

Resolved edge occurrences are preserved individually. If two source relationships point to the same target, two graph edges remain visible and two backlinks are returned. This preserves source relationship multiplicity instead of collapsing semantic facts into a set.

Ordering is deterministic:

- `DocumentKeys` follows caller input order;
- `Edges` follows document input order then M99 relationship source order;
- `Outgoing` and `Backlinks` preserve the corresponding global edge order;
- `ReachableFrom` uses breadth-first discovery in outgoing-edge order and excludes the starting key even through self/cyclic paths;
- `RelatedDocuments` returns unique direct incoming-or-outgoing neighbors in original document input order and excludes self-edges.

All variable-length public results are caller-owned copies. `Document` returns the existing immutable snapshot pointer rather than copying source/model state.

## Performance and resource model

Let:

- `V` = caller-provided documents;
- `R` = total M99 parser-resolved relationships across those documents;
- `E` = relationships admitted as graph edges;
- `F` = admitted cross-document edges with resolver-provided fragments;
- `N + H` = aggregate source-model nodes plus promoted headings across documents whose fragment catalogs are needed.

The explicit document slice is the traversal/resource boundary. M100 never discovers additional documents.

Graph build is expected linear/near-linear in supplied state plus resolver cost:

1. O(V) document/key validation;
2. existing M99 relationship projection per document;
3. O(R) resolver/local-edge classification;
4. O(E) edge plus adjacency-index construction;
5. for cross-document fragments, at most one ephemeral M98 fragment catalog per targeted document, then expected O(1) lookup per fragment.

The first M100 implementation called `ResolveFragment` independently for every cross-document fragment, which could rebuild the same target catalog repeatedly and approach O(F·N). Review replaced that path with an ephemeral per-target `FragmentResolver` cache owned only by the build call. Neither `Document` nor `DocumentGraph` retains those fragment catalogs after construction.

Persistent graph memory is O(V + E) plus references to the caller's immutable documents. Query costs are:

- `DocumentKeys`: O(V) result copy;
- `Edges`: O(E) result copy;
- `Outgoing`: O(out-degree);
- `Backlinks`: O(in-degree);
- `ReachableFrom`: O(V + E) worst case with O(V) transient visited/queue state;
- `RelatedDocuments`: O(V + degree) with transient neighbor membership state.

M108 retains responsibility for measurement-driven graph benchmarks and resource budgets.

## Requirements and edge cases

Focused coverage includes:

- direct cross-document resolution;
- resolver rejection of external/unmapped relationships;
- local fragments that bypass the resolver;
- cross-document percent-decoded heading fragments;
- missing and ambiguous cross-document fragments;
- resolver-authorized self edges;
- graph cycles and self loops without duplicate reachability output;
- multiple backlinks from the same source/target pair;
- deterministic document, edge, BFS, and related-document ordering;
- caller-owned keys, edge lists, outgoing lists, backlink lists, reachability results, and related-document results;
- mutation of the caller's original `[]GraphDocument` after build without graph-state leakage;
- resolver non-retention after build;
- empty graph behavior;
- nil graph read behavior;
- empty keys;
- duplicate keys;
- nil and zero-value documents;
- resolver-returned empty/unknown targets;
- M98 ephemeral fragment resolver equivalence with ordinary `ResolveFragment`.

## TDD and implementation evidence

1. `tsk_76583aa38ba8a60f49497a9ef111385b` established the public RED: `internal/publictest` failed to compile because the M100 graph types/functions did not yet exist. The same task confirmed branch `main`, HEAD `e28926c4c89cba50692a32147fdd55716b640654`, and `origin/main` `671b07f172331071bc6ef9eb08a70588c700b3d5` before implementation.
2. `tsk_a8c72bdfab1758330875c1c989dde01a` produced the first GREEN: all public M100 tests, the M99 relationship regressions, and the complete repository regression passed.
3. `tsk_61afe38cb901787e79f1fda185569cba` passed the expanded resolver-lifetime, cross-document fragment, caller-owned result, nil/empty graph, full regression, and production complexity checks.
4. Performance review identified repeated target-fragment rescans. M100 added one internal ephemeral `FragmentResolver` primitive that captures one M98 catalog without storing it in `Document`; public conversion reuses one shared `publicFragmentTarget` helper. `tsk_d27380f665ebcef42d0c68b1acb7b3ac` passed focused M100, focused M98 resolver-equivalence, complete regression, and production `gocyclo <= 15` after that refactor.
5. `tsk_36452e72283e599addcff55c8fa21428` passed final formatting, focused M100, the executable `ExampleBuildDocumentGraph`, and `git diff --check` before release-quality freeze.

## Devil's advocate review

1. **Graph resolution could silently become filesystem/network authority.** Mitigation: keys are opaque; every document is supplied up front; core imports no filesystem/network package for M100; non-local destinations are passed only to the caller's resolver; resolver false means no edge.
2. **A resolver could escape the authorized set by returning an arbitrary target.** Mitigation: every successful resolver result must name a key already validated in the explicit document set; empty/unknown targets fail the complete build with `ErrInvalidGraph`.
3. **Local fragments could be overridden by caller policy and lose Markdown-local semantics.** Mitigation: destinations beginning with `#` are always self-edges and never call the resolver.
4. **Missing/ambiguous cross-document fragments could erase an otherwise valid document relationship.** Mitigation: the edge remains present with `Fragment` but no `FragmentTarget`; M101 owns diagnostics rather than M100 discarding the relationship.
5. **Resolver lifetime could accidentally turn graph queries into hidden I/O/policy callbacks.** Mitigation: resolver is a build-local parameter only and is not stored in `DocumentGraph`; focused tests make it panic if invoked after build and prove queries remain independent.
6. **Cross-document fragment resolution could become quadratic.** Mitigation: build-local per-target fragment resolvers reuse one ephemeral M98 catalog; repeated fragments do not trigger repeated target scans.
7. **Backlink/outgoing indexes could duplicate whole edges and waste memory.** Mitigation: one authoritative edge slice is stored; adjacency maps contain integer edge indices only.
8. **Cycles/self links could make reachability nondeterministic or non-terminating.** Mitigation: iterative BFS marks the source visited before traversal, marks each target before enqueue, and preserves deterministic outgoing-edge discovery order.
9. **Returning internal slices could let callers mutate graph state.** Mitigation: every variable-length accessor materializes caller-owned storage; input-slice mutation after build is also covered.
10. **M100 could overclaim unresolved/broken-link validation.** Mitigation: only explicitly resolved graph edges are represented. M99 retains all parser-resolved source relationships and M101 is the dedicated diagnostic/repair-planning milestone.

## Release-quality verification

The final implementation tree passed:

- `tsk_06ff79f415b0f1c699593e214a0da2ab`: five consecutive complete `go test ./... -count=1` runs plus full race detection;
- `tsk_62de61776d4d9d2494249455218c3a12`: repository Go formatting cleanliness, `go vet`, `go build`, executable `ExampleBuildDocumentGraph`, and `go doc` checks for `DocumentGraph`, `GraphEdge`, `BuildDocumentGraph`, and `DocumentResolver`;
- `tsk_e853938948c7cac57548397d1e5f0ae8`: Staticcheck, golangci-lint with zero issues, production `gocyclo <= 15`, production and test-inclusive `unparam`, `go mod tidy -diff`, and `git diff --check`;
- `tsk_cae9006a3e1502e7cb60aeb5c4e7c77d`: govulncheck with no vulnerabilities found, Gitleaks with no leaks, and actionlint;
- `tsk_c9b05ccbf0734814714ce67b4fdbf1a7`: exact anchored `^TestGFM029PublishedSpecificationConformance$` against the explicit approved private published-GFM snapshot; verbose output proves the actual test ran and passed;
- `tsk_98ea3f3d7dd80e76db02ac14af12e0d3`: direct Go 1.27.0 test, vet, and build with matching explicit Go 1.27 `GOROOT`, private `GOPATH`/module/build caches, and `GOTOOLCHAIN=local`;
- `tsk_936bba67827d3ca2e7fe8786e041ecd6`: corrected explicit production-package cross coverage at **86.9%** aggregate and **84.2%** through `internal/publictest`; both temporary profiles lived only under the private tool root and the task proved `PROFILES_REMAIN=False`.

The earlier `tsk_14f83b9317eb7620073b65834dfdf2a5` is intentionally non-authoritative harness evidence. It invoked the Go 1.27 executable after the ordinary private activation left `GOROOT` pointing at Go 1.26.6, producing mixed compiler/assembler version errors before meaningful product verification. No source change was made for that failure; the fully isolated rerun above is authoritative.

`go.mod` and `go.sum` remain unchanged through the implementation freeze.

The final documented tree then passed:

- `tsk_cc4a7c1c40296c98c0e67d6d1a8812e1`: complete regression, verbose focused M100 suite, focused M98 ephemeral-fragment-resolver equivalence, vet/build, executable graph example, public graph API documentation, and exact explicit-snapshot GFM `RUN/PASS`;
- `tsk_e86ce7dea241c0de4e1fa15851f01ddf`: gofmt cleanliness, Staticcheck, golangci-lint with zero issues, production `gocyclo <= 15`, both unparam modes, `go mod tidy -diff`, `git diff --check`, govulncheck with no vulnerabilities, Gitleaks with no leaks, and actionlint;
- `tsk_7841993dc4a2e2502098f784a5b67f14`: no coverage artifacts, no private workspace/tool/brief path leakage, strict UTF-8/no-BOM/LF/no-trailing-whitespace/NUL hygiene, unchanged `go.mod`/`go.sum`, `git diff --check`, `git fsck --no-dangling`, and expected branch/HEAD/origin/working-tree inventory.

## Exit decision

M100 is complete. Multi-document relationship intelligence remains an immutable graph over a finite explicit caller-provided document set, with opaque caller keys, a non-retained build-only resolver, automatic same-document fragment edges, exact M98 target reuse, compact edge-index adjacency, deterministic traversal, and no implicit filesystem/network/document-discovery authority. M101 — workspace validation and repair planning over the same caller-authorized boundary — is the next roadmap milestone.
