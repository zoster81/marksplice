# M101 — Workspace Validation and Repair Planning

## Status

**Complete.** Focused TDD, architecture/performance review, implementation-tree release-quality verification, source-of-truth alignment, and the documented-tree release-quality freeze are green. No commit or push is authorized by this milestone record.

## Objective

M101 turns the M98 single-document navigation model, M99 relationship projection, and M100 explicit document graph into bounded workspace diagnostics and conservative repair planning **without** granting Marksplice implicit filesystem, path-resolution, network, or document-discovery authority.

The caller supplies the complete document set and explicitly classifies non-local relationships. Marksplice then reports only facts that are either parser/source proven or caller-authorized. Repairs are produced only when the requested result is already deterministic under an established source-preserving mutation contract.

## Requirements and authority boundary

The public entrypoint is:

```go
func ValidateWorkspace(
    documents []GraphDocument,
    resolver WorkspaceResolver,
    options WorkspaceValidationOptions,
) (*WorkspaceReport, error)
```

The `documents` set has the same finite explicit authority model as M100. Every `DocumentKey` is opaque caller identity. Marksplice does not interpret a key or relationship destination as a path or URL and performs no directory traversal, file read, URL fetch, command execution, or network request.

M100's boolean resolver is intentionally not reused for diagnostics because `false` means only “do not include this relationship in this graph”; it cannot distinguish a deliberately external relationship from a missing workspace document. M101 therefore uses an explicit classification:

```go
type WorkspaceResolutionKind uint8

const (
    WorkspaceResolutionUnknown WorkspaceResolutionKind = iota
    WorkspaceResolutionIgnore
    WorkspaceResolutionResolved
    WorkspaceResolutionMissing
)

type WorkspaceResolution struct {
    Kind     WorkspaceResolutionKind
    Target   DocumentKey
    Fragment string
}

type WorkspaceResolver func(
    source DocumentKey,
    relationship LinkRelationship,
) WorkspaceResolution
```

The meanings are deliberately narrow:

- `Ignore`: the caller declares that the relationship is outside workspace validation. It is neither a graph edge nor a broken-document diagnostic.
- `Resolved`: the caller maps the relationship to a non-empty `DocumentKey` already present in the explicit input set, optionally with a fragment.
- `Missing`: the caller states that a non-empty logical workspace target is expected but absent from the explicit input set.
- `Unknown` or contradictory payloads fail with `ErrInvalidWorkspace`.

The resolver is invoked synchronously only during `ValidateWorkspace`, once for each non-local M99 relationship. It is not retained in `WorkspaceReport` or `DocumentGraph`.

## Public diagnostics and report

`WorkspaceReport` contains exactly three products from one validation run:

1. the resolved M100 `DocumentGraph`;
2. deterministic `WorkspaceDiagnostic` values;
3. a conservative `WorkspaceRepairPlan`.

Variable-length diagnostic and repair slices are defensive copies.

`WorkspaceDiagnosticKind` covers:

- `WorkspaceDiagnosticMissingFragment`;
- `WorkspaceDiagnosticAmbiguousFragment`;
- `WorkspaceDiagnosticInvalidFragment`;
- `WorkspaceDiagnosticMissingDocument`;
- `WorkspaceDiagnosticUnresolvedReference`;
- `WorkspaceDiagnosticOrphanDocument`;
- `WorkspaceDiagnosticStaleGeneratedIndex`;
- `WorkspaceDiagnosticUnrecognizedGeneratedIndex`.

A diagnostic exposes only metadata meaningful to its kind:

- `SourceDocument()` for source-bound findings;
- `SourceOffset()` when one exact source anchor exists;
- `Relationship()` when the finding derives from an M99 relationship;
- `TargetDocument()` for local/cross-document targets, missing documents, orphan documents, and managed indexes;
- `Fragment()` for fragment/document findings;
- `UnresolvedReference()` for the conservative unresolved-reference subset;
- `NodeID()` for managed generated-index findings.

Orphan findings are intentionally target-only: they identify an unreachable document rather than pretending that one source location caused the condition.

Diagnostics are deterministic but grouped by validation phase rather than globally merged by source offset: relationship/fragment/document findings first in caller document order and M99 source order, then unresolved-reference findings in caller document/source order, then orphan findings in caller document order, then managed-index findings in caller-supplied `ManagedTOCs` order.

## Fragment and missing-document diagnostics

Local semantic destinations beginning with `#` reuse the exact M99/M98 fragment status already proven for the source document. Resolved local fragments produce no diagnostic. Missing, ambiguous, or invalid local fragments produce the corresponding typed finding and retain the local source document as both source and target identity.

For non-local relationships, only the caller's `WorkspaceResolution` decides whether the relationship belongs to the workspace:

- `Ignore` adds no edge and no diagnostic;
- `Missing` adds `WorkspaceDiagnosticMissingDocument` and no graph edge;
- `Resolved` adds an M100 graph edge and, when a fragment is supplied, resolves that fragment against the target snapshot using exact M98 semantics.

A resolved document edge is retained even if its fragment is missing/ambiguous/invalid. Fragment failure does not erase already-authorized document topology.

M101 caches the caller's successful `Resolved` decisions and uses them to build the M100 graph without invoking the user resolver a second time. Cross-document fragment classification and M100 target attachment share one build-local fragment resolver per target document.

## Conservative unresolved-reference diagnostics

Goldmark does not retain an unresolved reference node when no matching definition exists. A focused pinned-backend probe established the residual AST behavior before implementation:

- explicit unresolved full/collapsed forms remain source-backed `Text` segments;
- a resolved reference becomes an ordinary `Link`/`Image` node;
- code spans retain their own container and are distinguishable from normal text.

M101 therefore adds one parser-independent observation:

```go
type UnresolvedReferenceUsage struct {
    Kind      Kind
    Form      LinkUsageForm
    Anchor    int
    Reference string
}
```

The Goldmark adapter parses with an explicit parser context and, after ordinary semantic walking, performs a conservative source scan only over contiguous eligible `Text` runs. The detector:

- recognizes explicit full `[label][reference]` and collapsed `[label][]` forms plus image counterparts;
- uses the same parser-backed `ReferenceLabelKey` normalization and actual parse context, so an existing definition prevents a false unresolved report;
- excludes text owned by code spans, links, images, autolinks, and raw HTML;
- rejects escaped opening markers, nested/complex labels, embedded backslashes, and multiline shapes rather than guessing;
- excludes the recognized Marksplice front-matter envelope from retained diagnostics;
- deliberately does **not** diagnose shortcut `[label]` source when no definition exists, because that spelling is indistinguishable from ordinary bracket text after the parser declines reference interpretation.

The retained snapshot state is one compact source-ordered `unresolvedReferenceUsages` scalar vector. No secondary AST or parser object is retained.

## Orphan and reachability diagnostics

`WorkspaceValidationOptions.Roots` is an explicit caller-provided root set. An empty root set means “do not perform orphan diagnostics”; Marksplice does not choose roots implicitly.

When roots are supplied they must be non-empty, unique, and present in the explicit document set. M101 computes their union reachability using one private multi-source BFS over the already-built M100 adjacency structure. This is O(V+E) rather than one full traversal per root. Every document not reached from any root becomes one `WorkspaceDiagnosticOrphanDocument` in caller document order.

The public M100 `ReachableFrom` contract and discovery ordering are unchanged.

## Managed generated indexes and repair planning

M101 does not scan arbitrary headings looking for generated content. The caller explicitly supplies:

```go
type ManagedTOC struct {
    Document  DocumentKey
    HeadingID NodeID
}
```

Each managed target must identify an existing heading in the named snapshot and may appear only once. M98's strict managed-body recognizer remains authoritative:

- an unrecognized body produces `WorkspaceDiagnosticUnrecognizedGeneratedIndex` and **no** repair;
- a recognized current body produces no finding;
- a recognized stale body produces `WorkspaceDiagnosticStaleGeneratedIndex` and is eligible for repair.

M101 batches managed-TOC status calculation per document. Heading identities are materialized once per document, and M98 generated TOC bytes are reused per line-ending form using call-local caches. No TOC index/cache is stored in the snapshot.

## Atomic repair plan

M101 does not automatically rewrite broken links, guess missing files, choose among ambiguous anchors, or invent unresolved reference definitions. Those findings do not carry enough caller intent for a safe repair.

The only M101 automatic repair is synchronization of caller-designated, M98-recognized stale TOCs.

A single stale TOC already had an established M98 `PrepareSyncTOC` operation. Multiple stale managed TOCs in one document exposed an important composition boundary during M101 review: independently prepared M98 changes can share generated link-relationship deltas, so generic M95 `ComposeChanges` correctly fails closed rather than assuming those semantic edits are independent.

M101 therefore adds a **family-specific atomic multi-TOC preparation path** instead of weakening M95:

1. validate every explicit TOC target against the same immutable snapshot;
2. derive deterministic M98 replacement bytes, reusing generated TOC bytes per EOL;
3. sort exact direct-body patches by source position;
4. create one ordinary source-bound `ChangeSet` over original coordinates;
5. parse one combined candidate;
6. prove every original heading/style/range after cumulative patch transforms;
7. prove each targeted section body has its exact expected post-patch range.

The repair plan therefore returns at most one `WorkspaceRepair` per document, and each repair contains one ordinary stale-source-protected `ChangeSet`. Callers remain responsible for applying that change to the source snapshot they own.

## Malformed input and fail-closed behavior

`ErrInvalidWorkspace` covers malformed validation authority or inconsistent resolver results. M101 rejects:

- empty or duplicate document keys;
- nil/uninitialized documents;
- empty, unknown, or duplicate root keys;
- managed TOCs naming an unknown document, zero/foreign heading identity, or duplicate target;
- `Ignore` carrying target/fragment data;
- `Missing` with an empty target or a target actually present in the set;
- `Resolved` with an empty/unknown target;
- unknown resolution kinds.

There is no hidden document-count, root-count, relationship-count, or managed-TOC cap. Resource use is proportional to the explicit caller input.

## Performance and retained state

Let:

- `V` = documents;
- `E` = resolved graph edges;
- `R` = M99 relationships;
- `U` = conservative unresolved reference usages;
- `H` = headings/sections;
- `T` = explicitly managed TOCs;
- `B` = bytes of their designated bodies.

Durable properties after review:

- unresolved-reference detection is one bounded pass over eligible parser text/source and stores O(U) scalar state in each immutable snapshot;
- relationship validation is source ordered and caller resolution is invoked once per non-local M99 relationship;
- M100 graph construction reuses cached caller decisions and build-local target-fragment resolvers; it performs no user resolver callback on the second graph projection;
- orphan computation is one multi-source O(V+E) BFS regardless of root count;
- managed heading ownership is derived once per document;
- managed TOC status/repair derivation reuses generated TOC bytes per EOL with call-local caches;
- multi-TOC repair uses O(T log T) patch ordering plus one combined parse/proof rather than sequential stale-source rebasing;
- persistent workspace state is the returned O(V+E) M100 graph, diagnostics, and repair values only. Resolver callbacks, fragment catalogs, heading sets, root traversal state, and TOC caches are discarded before return.

M101 does currently project M99 relationships once for diagnostics and once inside the shared M100 graph builder. Both passes are expected-linear and no caller resolver work is duplicated. Measurement-driven elimination of remaining constant-factor projection costs belongs to M108 unless benchmarks justify earlier specialization.

## Devil's advocate review and mitigations

### Risk 1 — “not in graph” accidentally means “broken”

Reusing M100's boolean resolver would conflate intentionally external links with missing workspace targets. M101 uses explicit `Ignore` / `Resolved` / `Missing` states instead. `Ignore` can never become a broken-document diagnostic.

### Risk 2 — unresolved-reference false positives

After a failed reference parse, ordinary bracket text and shortcut references are semantically ambiguous. M101 reports only source-proven explicit full/collapsed shapes, rejects escape/complex cases, consults the real parser reference context, and never diagnoses shortcut `[label]` spelling without a definition.

### Risk 3 — repair plan becomes raw rewrite authority

Only explicitly managed, strictly recognized M98 TOCs may be repaired. All other diagnostics are read-only. Repairs remain ordinary snapshot-bound `ChangeSet` values and preserve stale-source rejection.

### Risk 4 — generic composition weakened to make generated indexes work

Two managed TOCs correctly demonstrated that M95 may reject independently prepared changes whose generated link semantic deltas overlap. M101 does not bypass or relax M95. A family-specific multi-TOC candidate proof prepares the complete intent atomically.

### Risk 5 — many roots/managed indexes cause repeated whole-workspace scans

The first design called `ReachableFrom` per root and `HeadingAnchor`/TOC generation per managed target. Review replaced those paths with one multi-source BFS, one heading-ID set per document, and call-local batched TOC generation per line-ending form.

### Risk 6 — diagnostic cannot be located in a multi-document set

The first public model carried a source offset but no source `DocumentKey` for unresolved references. Pre-freeze API review added `WorkspaceDiagnostic.SourceDocument()` for every source-bound finding; orphan diagnostics remain target-only by design.

## TDD and implementation evidence

The public missing-API RED is `tsk_a6a077b2a977f2de4b23a470a276e667`.

Earlier parser investigation:

- `tsk_61b87ac963110eba280f2728b6b47518` is discarded as harness-only because it launched the private probe outside a Go module and never executed the parser;
- corrected pinned-Goldmark AST/source probe `tsk_c9cd4133073e138c515f99d93b27aad7` established the residual unresolved-reference behavior and the conservative detector boundary. The private probe file was then removed from the tool root.

Implementation progression:

- `tsk_511b9380da821906e2734fa41cf82a3f`: resume-state compile established only the expected in-progress adapter-signature/M101-API errors;
- `tsk_1ed6d64d4be61126c0b684035b9d2706`: parser detector and M98 focused tests green; the only public failure was a test expecting four resolver calls instead of the correct five non-local M99 relationships. The assertion was corrected because treating `/resolved` specially would violate opaque-destination semantics;
- `tsk_7745e6e45b090159e934bdc164950b88`: first corrected focused M101 plus full repository GREEN;
- `tsk_3356d2a7194babab3923d3e279b85ae6`: devil test exposed that its own `#old-*` test data also generated legitimate missing-fragment findings; the test was isolated rather than weakening diagnostics;
- `tsk_38137476c4d865984b947cb141d90e65`: corrected test exposed the real M95 semantic-overlap rejection for two separately prepared TOC changes;
- `tsk_82bdb927dab546d6981408b8bdb07d54`: family-specific atomic multi-TOC repair made all functional tests green, while the production complexity gate correctly reported the first multi-patch heading validator at cyclomatic complexity 17;
- `tsk_9f997c04d8d01954d20b11d908e74d7f`: cursor/helper refactor reduced production complexity below the <=15 gate, introduced multi-source reachability/heading caching, preserved M100 behavior, and passed focused/full regressions;
- `tsk_59f357417293a4d5d2ca6325e7bed021`: pre-freeze public M101 suite, executable example, API docs, vet/build, gocyclo, unparam, full regression, and diff check green;
- `tsk_f24f31137d25757015b2415c26a92a21`: source-document diagnostic ownership addition passed focused/full regression and complexity;
- `tsk_07e592f3276f610b340c4eb10068a1db`: batched managed-TOC status/render refactor passed M98/M101 focused tests, complexity, full regression, and diff check;
- `tsk_0c49895374de36354d089233b6105499`: pre-freeze inventory confirmed expected M98–M101 working-tree files, unchanged module files, branch/HEAD/origin state, and clean diff check.

## Implementation-tree release-quality freeze

The final implementation tree passed:

- five consecutive complete `go test ./... -count=1` runs plus the race detector: `tsk_2907805116987604b01ea7a421269a09`;
- gofmt cleanliness, `go vet`, `go build`, executable `ExampleValidateWorkspace`, and public API documentation checks: `tsk_9c37c0ea0e1b4e0890a46634fb94b4ef`;
- Staticcheck, golangci-lint with `0 issues.`, production `gocyclo <= 15`, production and test-inclusive unparam, `go mod tidy -diff`, and `git diff --check`: `tsk_c54fa495015d76b96ed1a2eafdf585bb`;
- govulncheck (`No vulnerabilities found.`), Gitleaks (`no leaks found`), and actionlint: `tsk_c107b4d397dcb1745b10195b66a929fc`;
- exact published GFM 0.29 conformance using the explicit approved private snapshot, with verbose `=== RUN` and `--- PASS`: `tsk_27dea0f09529fa1e0f68af9836397b01`;
- authoritative isolated Go 1.27.0 Windows test/vet/build with explicit matching `GOROOT`, `GOPATH`, `GOCACHE`, and `GOTOOLCHAIN=local`: `tsk_dfa69587aad055116f7ac1aa74993b7f`;
- corrected explicit production-package cross coverage: `tsk_88b26f04f4caea4a67612ea06fd5e4d6`, reporting **87.1% aggregate** statement coverage and **84.5% through `internal/publictest`**, with `PROFILES_REMAIN=False`.

`go.mod` and `go.sum` remain unchanged through the implementation freeze.

## Documented-tree release-quality freeze

After aligning `AGENTS.md`, the public README/documentation source-of-truth files, and this milestone record with the final M101 contract, the documented tree passed:

- focused M101 public tests, complete repository regression, executable `ExampleValidateWorkspace`, public API documentation, module-diff check, and `git diff --check`: `tsk_d9ca2daf64d66f0e440828f202a3d715`;
- five consecutive complete `go test ./... -count=1` runs plus the race detector: `tsk_b52a2bffb3efa7deae2d4578ef991dcb`;
- corrected explicit Go-file formatting check, `go vet`, `go build`, executable example, public API docs, Staticcheck, golangci-lint, production `gocyclo <= 15`, both unparam modes, `go mod tidy -diff`, and `git diff --check`: `tsk_f7da8d499800402e2ca2bf8316194ec4`;
- govulncheck (`No vulnerabilities found.`), Gitleaks (`no leaks found`), and actionlint: `tsk_a51c020159768dc279367bcb62e50211`;
- exact published GFM 0.29 conformance using the explicit approved private snapshot, with verbose `=== RUN` and `--- PASS`: `tsk_7bdf474b3b79ff1063aefaaec526c875`;
- authoritative isolated Go 1.27.0 Windows test/vet/build with matching toolchain root and private caches: `tsk_b915d239b2723d8dea3652d08a74b5cd`;
- corrected explicit production-package cross coverage: `tsk_22c11a6e5c2dc940477256ea8c941c2f`, reporting **87.1% aggregate** statement coverage and **84.5% through `internal/publictest`**, with `PROFILES_REMAIN=False`.

The following documented-tree attempts are explicitly **non-authoritative harness results** and do not represent product failures:

- `tsk_b8bfd16d161139f8284688340abc7a76` stopped before any product verification because the command referenced a nonexistent toolchain-activation path;
- `tsk_850d26ad2f84594443f62213a67bc5fb` stopped before tests because the harness expected `rg` on `PATH`; the stale-marker check was rerun through the connector and found no obsolete M101-boundary markers;
- `tsk_f31ee9779e10bb91f3c059d51a1f9ba4` passed a literal `$coverpkg` argument and therefore produced meaningless 0.0% coverage; `tsk_22c11a6e5c2dc940477256ea8c941c2f` supersedes it;
- `tsk_430a45fe659598e208e7b6a0c8e8594e` completed its substantive vet/build/docs/static-analysis gates, but its root gofmt subcommand used an unexpanded Windows glob. The complete corrected quality gate `tsk_f7da8d499800402e2ca2bf8316194ec4` supersedes it.

`go.mod` and `go.sum` remain unchanged through the documented-tree freeze.

## Exit decision

M101 is **complete**. The validated public contract preserves the explicit caller-authority boundary, reports only source/parser-proven or caller-declared workspace failures, retains no hidden resolution/crawling authority, and automatically repairs only deterministic caller-managed stale TOCs through ordinary snapshot-bound mutation machinery.

The next roadmap boundary is **M102 — Semantic block patterns**, beginning with reviewed GitHub alert semantics (`NOTE`, `TIP`, `IMPORTANT`, `WARNING`, `CAUTION`) represented as a semantic layer over ordinary blockquote source without changing the baseline GFM parser grammar or weakening existing blockquote ownership proof.
