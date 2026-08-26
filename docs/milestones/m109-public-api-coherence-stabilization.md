# M109 — Public API Coherence and Stabilization Review

Status: complete — public API coherence/stabilization audit and release-quality verification are green.

## Objective

Review the complete Marksplice public Go surface as one library before the third-party extensibility boundary and native-parser work. M109 is a stabilization milestone, not a feature-expansion milestone: it removes accidental API ambiguity, makes durable cross-cutting contracts explicit, and deliberately preserves asymmetries whose source-ownership or semantic boundaries are intentional.

The review covers naming and typed values, mutation composition, query/graph/workspace/knowledge APIs, builder behavior, error taxonomy, boundedness, concurrency, public documentation, and assumptions that M110 must not accidentally invalidate.

## Requirements and edge cases

The reviewed contract requires:

- preserve every established source-ownership, source-preservation, stale-source, parser-isolation, and fail-closed invariant;
- treat the current v0 beta as the last appropriate place to remove clearly accidental public ambiguity before a stable v1 compatibility burden forms;
- do not add APIs merely for visual symmetry when a capability family has intentionally narrower semantics or mutation authority;
- keep `Document`, `DocumentGraph`, `KnowledgeIndex`, `WorkspaceReport`, and prepared `ChangeSet` values immutable after construction and safe for concurrent reads;
- keep `DocumentBuilder` explicitly mutable and require caller synchronization for concurrent use;
- invoke caller resolvers synchronously during graph/workspace construction, never concurrently within one call, and never retain them;
- keep public variable-length results caller-owned so concurrent readers cannot mutate retained snapshot state through returned slices;
- retain the existing public sentinel-error families and require callers to classify them with `errors.Is` rather than diagnostic strings;
- keep structural queries explicitly caller-bounded by positive result limits;
- keep graph/workspace/knowledge work bounded by finite caller-provided document sets and avoid hidden discovery, I/O, or global result caps;
- do not add `context.Context` parameters that the current synchronous Goldmark-backed parse/proof path cannot honor reliably during parser execution;
- keep future third-party syntax identities outside the closed Marksplice core `Kind` enum so M110 does not consume or destabilize core kind ordinals/query tables;
- keep current-contract GoDoc independent from milestone chronology; historical M-number detail belongs in `docs/milestones/` and architecture/history records.

Nil/zero-value behavior is retained where already intentional: the zero `DocumentBuilder` remains usable, zero/unbound `ChangeSet` remains source-conflicting, immutable pointer models retain their established nil-safe read behavior where documented, and zero public typed detail values remain non-authoritative rather than fabricated identities.

## Audit decisions

### Typed workspace unresolved references

The main API defect found by the audit was:

```go
func (d WorkspaceDiagnostic) UnresolvedReference() (string, ReferenceForm, bool, bool)
```

The two adjacent booleans represented `isImage` and `present`, making call sites easy to misread and unlike the rest of the typed diagnostic surface. M109 replaces that tuple with one immutable semantic value:

```go
type UnresolvedReference struct { /* immutable */ }
func (r UnresolvedReference) Reference() string
func (r UnresolvedReference) Form() ReferenceForm
func (r UnresolvedReference) IsImage() bool
func (d WorkspaceDiagnostic) UnresolvedReference() (UnresolvedReference, bool)
```

No diagnostic data, ordering, parser observation, resolution authority, or workspace behavior changes. This is an intentional pre-v1 source-level compatibility change and is recorded in the changelog.

`TOCStale() (bool, bool)` remains unchanged because it is the conventional documented `(value, found)` shape. `LinkRelationship.Reference() (string, ReferenceForm, bool)` also remains unchanged because its trailing boolean is one unambiguous presence flag rather than two adjacent semantic booleans.

### Mutation and typed-view coherence

M95 `ComposeChanges` remains the generic composition boundary. The audit found no justification for exposing patches, mutable transactions, sequential rebasing, or a weaker composition proof. Family-specific atomic operations remain preferable when several intents deliberately affect one logical aggregate.

The reviewed typed-view asymmetries remain intentional. Complete `FencedBlock` readability does not broaden the narrower `FencedCode` mutation span; multiline footnote bodies remain readable without becoming generic replacement spans; mathematical payload remains opaque; complex front matter remains envelope-readable without a metadata AST/serializer; parsed reference links/images remain relationship-readable without generic source mutation authority. M109 therefore adds no symmetry-only getters, builders, mutation methods, or public structural kinds.

### Query, graph, workspace, and knowledge boundedness

`QueryNodes`/`QuerySections` retain mandatory positive result limits and no persistent selector index. `DocumentGraph`, workspace validation, and `KnowledgeIndex` continue to operate only on explicit finite caller-provided document sets and never discover files or fetch URLs.

M109 deliberately does not add `context.Context` to otherwise synchronous in-memory APIs. The temporary Goldmark backend offers no reliable mid-parse cancellation boundary, so accepting a context would imply cancellation semantics the implementation could not consistently provide. A future native parser may introduce explicit cancellation/budget checkpoints only under a separately reviewed contract.

No new hidden size/depth/result caps are introduced. Existing reviewed structural limits, such as bounded typed-inline/blockquote construction depth, remain unchanged.

### Concurrency contract

`Parse` copies caller source and immutable snapshot/graph/knowledge/workspace state has no post-build mutation. M109 therefore makes the existing property public: successful immutable `Document`, `DocumentGraph`, `KnowledgeIndex`, `WorkspaceReport`, and `ChangeSet` values may be read, queried, used for mutation planning, or applied concurrently.

The contract excludes caller races on argument storage: callers must not mutate a byte slice while passing it to an operation. `DocumentBuilder` remains mutable and is not concurrently safe without caller synchronization. Resolver callbacks are synchronous, serialized within one build/validation call, and discarded before the immutable result is returned.

A black-box race test concurrently exercises document projections, graph/knowledge traversals, workspace diagnostics, defensive-copy ownership, and `ChangeSet.Apply` across multiple goroutines.

### Error taxonomy

The existing public sentinel families remain sufficient:

- `ErrNodeNotFound`;
- `ErrInvalidTargetKind`;
- `ErrInvalidReplacement`;
- `ErrSourceConflict`;
- `ErrInvalidConstruction`;
- `ErrInvalidQuery`;
- `ErrInvalidGraph`;
- `ErrInvalidWorkspace`;
- `ErrInvalidKnowledge`.

No new umbrella sentinel is added. Public package documentation now states that callers should use `errors.Is`; human-readable diagnostic wording is not a compatibility contract.

### Public documentation and M110 boundary

Current-package GoDoc no longer refers to internal milestone numbers. Comments describe present semantics directly, including builder construction rules, resolver lifetime/concurrency, boundedness, and core-kind scope. Milestone chronology remains in milestone/history documentation.

`Kind` is explicitly the Marksplice **core** structural enum. M110 must use a separately namespaced/typed extension identity rather than reserving or injecting third-party kinds into the compact core ordinal space used by query filters and internal/public kind translation.

## Architecture and test strategy

M109 intentionally changes almost no runtime architecture. The one API-shape change replaces three scalar diagnostic fields with one scalar immutable public value while retaining the same underlying semantic data. No new map, cache, lock, goroutine, parser pass, dependency, or persistent index is introduced.

Testing follows the compatibility-sensitive path:

1. establish a black-box compile RED for the desired typed unresolved-reference accessor;
2. implement only the typed value and migrate existing public workspace tests;
3. run focused M101/M109 and full repository regressions;
4. exercise the immutable-model concurrency contract under the real race detector;
5. scan the current public Go surface for milestone-number leakage;
6. run generated public documentation and complete release-quality gates before marking M109 complete.

## Devil's advocate review

### Risk 1 — changing a public accessor breaks beta consumers

`WorkspaceDiagnostic.UnresolvedReference` is a source-level breaking change. Keeping the four-value tuple would preserve short-term compatibility but freeze an avoidable ambiguous API into later releases.

Mitigation: make the change during the documented v0 beta stabilization window, keep all semantic data unchanged, use one small typed value consistent with the rest of the API, update existing black-box tests and changelog, and do not bundle unrelated API renames.

### Risk 2 — documenting concurrent reads constrains future implementation choices

A future lazy cache or callback could accidentally introduce mutation/races behind an API now promised to be immutable.

Mitigation: the guarantee is deliberately limited to successful immutable public models. Any future cache must be immutable-at-build, safely synchronized, or remain call-local. `DocumentBuilder` is explicitly excluded, and resolver callbacks remain build-local/serialized rather than becoming query-time hooks.

### Risk 3 — adding cancellation only in the signature creates a false safety guarantee

Passing `context.Context` through wrapper methods without parser checkpoints could make callers believe expensive parsing/proof work is cancellable when it is not.

Mitigation: keep the current synchronous bounded contract honest. Revisit cancellation only when the native parser can define deterministic checkpoints and resource semantics.

### Risk 4 — M110 extensions consume core `Kind` values

Allowing independent packages to append or register core kinds would destabilize ordinal-based filters, mapping arrays, serialization assumptions, and core semantics.

Mitigation: M109 explicitly freezes `Kind` as the core namespace. M110 must define a separate namespaced extension identity and cannot redefine baseline GFM/core kinds.

### Risk 5 — cleanup by symmetry widens unsafe source authority

An API audit can tempt addition of matching getters/mutators/builders where source proof is intentionally asymmetric.

Mitigation: require a demonstrated semantic/source-ownership gap, not visual API symmetry. M109 preserves the reviewed fenced/footnote/math/front-matter/reference asymmetries and adds no new source authority.

## Verification

M109 began with a black-box compatibility RED for the typed unresolved-reference accessor, then verified the migrated M101/M109 surface, generated public documentation, boundedness/error/concurrency documentation, and zero milestone-number leakage from current public Go comments.

The immutable public-model concurrency contract was exercised under the real race detector across document projections, graph/knowledge traversals, workspace diagnostics, defensive-copy ownership, and `ChangeSet.Apply`.

The documented tree passed repeated complete regressions, focused black-box tests, race detection, examples and public GoDoc checks, `go vet`, `go build`, Staticcheck, golangci-lint, production `gocyclo <= 15`, production/test-inclusive unparam, `go mod tidy -diff`, published GFM conformance, Go 1.27 test/vet/build, govulncheck, Gitleaks, actionlint, strict text/artifact hygiene, and `git diff --check`.

Cross-package statement coverage measured **87.0% aggregate** and **84.7% through `internal/publictest`**.

## Exit decision

M109 is complete. The public surface has one evidence-backed pre-v1 ambiguity removed, durable concurrency/error/boundedness contracts made explicit, and a clean M110 namespace boundary without adding extension implementation, new source authority, parser behavior, synchronization, caches, dependencies, or hidden limits. The next roadmap boundary is **M110 — Public third-party syntax/semantic SPI**. Native-parser ownership still begins only at M111.
