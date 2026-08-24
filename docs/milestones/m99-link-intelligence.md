# M99 — Link intelligence

## Status

Complete. Focused TDD, edge-case and performance review, full implementation-tree release-quality verification, corrected explicit cross-package coverage, exact pinned GFM conformance, and security/static/maintainability gates are green. The final documented-tree hygiene pass is recorded below. No commit or push is authorized by this milestone record.

## Objective

Complete single-document outgoing relationship intelligence over the immutable Marksplice snapshot without creating filesystem/network authority or a second document graph.

M99 covers parser-resolved:

- direct inline links;
- full/collapsed/shortcut reference links;
- direct images;
- full/collapsed/shortcut reference images;
- angle and GFM bare/extended autolinks;
- reference-definition ownership when one promoted definition can be proven uniquely;
- intra-document fragment status through the existing M98 heading/HTML-anchor resolver.

M99 remains narrower than M100 and M101. Other-document paths and external URLs stay opaque destination data. Unresolved reference-looking source is not reinterpreted as a relationship. No filesystem read, workspace traversal, network request, backlink graph, repair plan, or raw mutation authority is added.

## Public contract

M99 adds:

```go
type LinkRelationshipKind uint8
const (
    LinkRelationshipUnknown LinkRelationshipKind = iota
    LinkRelationshipInlineLink
    LinkRelationshipReferenceLink
    LinkRelationshipInlineImage
    LinkRelationshipReferenceImage
    LinkRelationshipAutoLink
)

type ReferenceForm uint8
const (
    ReferenceFormUnknown ReferenceForm = iota
    ReferenceFormFull
    ReferenceFormCollapsed
    ReferenceFormShortcut
)

type LinkFragmentStatus uint8
const (
    LinkFragmentNotApplicable LinkFragmentStatus = iota
    LinkFragmentResolved
    LinkFragmentMissing
    LinkFragmentAmbiguous
    LinkFragmentInvalid
)

type LinkRelationship struct { /* immutable */ }
func (r LinkRelationship) Kind() LinkRelationshipKind
func (r LinkRelationship) SourceOffset() int
func (r LinkRelationship) SourceNodeID() (NodeID, bool)
func (r LinkRelationship) Destination() string
func (r LinkRelationship) Title() (string, bool)
func (r LinkRelationship) Reference() (string, ReferenceForm, bool)
func (r LinkRelationship) ReferenceDefinitionID() (NodeID, bool)
func (r LinkRelationship) FragmentStatus() LinkFragmentStatus
func (r LinkRelationship) FragmentTarget() (FragmentTarget, bool)
func (r LinkRelationship) IsEmail() bool

func (d *Document) LinkRelationships() []LinkRelationship
```

`LinkRelationships` returns a caller-owned source-ordered slice. It is a read-only semantic projection and grants no generic source range or edit permission.

## Parser-independent relationship observation

M93 already required parser-resolved reference usage facts for mutation safety. M99 generalizes that single internal vector instead of introducing a parallel model.

`internal/parser.LinkUsage` records only Marksplice-owned scalars:

- semantic kind;
- source form (`direct`, `full`, `collapsed`, `shortcut`);
- parser/source-proven anchor offset;
- reference value when applicable;
- resolved destination;
- resolved title and title presence;
- autolink email classification.

The Goldmark adapter collects these facts during the existing AST walk. No Goldmark AST type leaves `internal/parser/goldmark`. The former M93-only `ReferenceUsage` aliases/wrapper were removed after the generalized vector replaced them; historical M93 behavior is still covered by the adapter regression.

Recognized YAML/TOML front-matter envelopes remain outside Markdown semantics in Marksplice. Link usages whose anchors lie inside the recognized envelope are removed from the snapshot relationship vector, extending the same boundary previously applied to M93 reference usages.

## Relationship classification

Direct link/image relationships expose parser-resolved destination/title facts independently from ordinary public node promotion. Therefore a complex label/alt can appear in `LinkRelationships` even when it does not satisfy the narrower source-mapped editable `InlineLink`/`Image` public subset.

`SourceNodeID` is available only when the exact relationship already corresponds to a promoted direct link/image/autolink node. Reference link/image usages remain relationship facts rather than newly promoted editable nodes.

Autolinks expose the parser-resolved destination and existing email classification. For GFM `www.` autolinks the semantic destination may include the parser-supplied `http://` prefix even though the source token does not. `SourceOffset` continues to identify the source token start rather than a synthesized destination string.

## Reference-definition ownership

A resolved reference relationship always exposes the parser-resolved reference value/form/destination/title. `ReferenceDefinitionID` is a stricter optional ownership fact.

M99 builds one ephemeral definition-owner map keyed by:

- parser-defined normalized reference label key;
- resolved destination;
- title;
- title-presence bit.

An owner ID is returned only when exactly one matching definition observation is present and that definition belongs to the existing promoted single-line editable `ReferenceDefinition` subset. If the semantic definition is valid but not publicly promoted — for example a supported multiline definition title — the relationship remains available while `ReferenceDefinitionID` returns false.

Ambiguous matching candidates also return no owner ID rather than selecting one heuristically.

GFM reference-label normalization remains behind the parser adapter through the same internal normalization primitive already used by construction. The normalization key is not a public or persistent identity.

## Fragment integration

Only destinations whose semantic value begins with `#` are classified as intra-document fragment relationships. Other paths such as `guide.md#part` remain `LinkFragmentNotApplicable` in M99 because resolving another document belongs to M100.

M99 does not implement a second fragment parser. It reuses the M98 normalization and target model:

- malformed/empty local fragments => `LinkFragmentInvalid`;
- no target => `LinkFragmentMissing`;
- more than one heading/explicit-anchor target => `LinkFragmentAmbiguous`;
- exactly one target => `LinkFragmentResolved` plus the exact M98 `FragmentTarget`;
- non-local destinations => `LinkFragmentNotApplicable`.

To avoid repeated full-document scans, M98 target derivation now exposes an internal ephemeral fragment catalog. `ResolveFragment` builds that catalog for an individual call; `LinkRelationships` builds it once and reuses it for every local relationship. The catalog is never retained in `Document`.

## Mutation/composition safety reuse

M99 broadens the compact relationship vector already consulted by removal/composition safety. Whole-block removal and `Document.ComposeChanges` now compare surviving direct links, images, autolinks, and references rather than reference usages alone.

This is intentionally stricter internal proof, not new public edit authority:

- a relationship owned by a removed source range may disappear;
- surviving relationships must preserve transformed source anchor and semantic facts;
- composed independently validated changes must produce the expected combined relationship delta;
- cross-operation reinterpretation remains fail-closed.

## Complexity and resource model

Let:

- `N` = stored structural nodes;
- `H` = promoted headings;
- `R` = parser-resolved link/image/autolink usages;
- `D` = reference definitions, with `D <= N`.

`LinkRelationships` performs expected O(N + H + R) work and O(N + H + R) temporary/result memory:

1. one O(N) scan builds the promoted source-node lookup;
2. one O(N) scan builds the ephemeral normalized definition-owner map;
3. M98 heading anchors plus supported explicit HTML anchors build one O(H + N) ephemeral fragment catalog;
4. one O(R) relationship projection performs expected O(1) definition/fragment lookups.

The first implementation scanned every definition for every reference and called the complete M98 resolver for every local fragment, creating avoidable O(R·D + R·(H+N)) behavior. Review replaced both nested scans with the ephemeral maps above before release-quality freeze.

No relationship index, normalized-label map, fragment map, backlink map, or graph is persisted in `Document`. The existing source-ordered `linkUsages` vector is the sole retained relationship observation set. M108 retains responsibility for benchmark-driven performance changes.

## Requirements and edge cases

Focused coverage includes:

- simple promoted direct links;
- complex direct link labels that remain non-promoted but still produce relationships;
- direct images with structured alt content;
- full/collapsed/shortcut reference links;
- reference images;
- shared promoted definition owner IDs;
- semantic reference resolution through an unpromoted multiline definition with no fabricated owner ID;
- unresolved reference-looking source producing no relationship;
- angle URL/email autolinks;
- GFM bare/extended URL, `www.`, and email autolinks;
- exact source ordering and source offsets;
- caller-owned result slices;
- recognized front-matter exclusion;
- percent-decoded local fragments;
- heading targets;
- promoted explicit HTML-anchor targets;
- missing, ambiguous, and invalid local fragments;
- other-document `path#fragment` destinations remaining not-applicable;
- nil-document behavior.

## TDD and implementation evidence

1. `tsk_35130563826a74a7f3496bc9e8031895` established the public RED: the focused consumer package failed to compile only because the M99 relationship types/method were absent.
2. The first implementation generalized the M93 reference vector to `LinkUsage`, added the public projection, and reused M98 fragment semantics. `tsk_af99c9bc1f2d32c96c8e764c93c8627e` passed the corrected focused M98/M99 and relevant internal suites after a fixture was fixed to contain a structurally valid separated reference definition.
3. `tsk_598619205fcfa1a097a0b5f02a07b775` passed the first complete `go test ./... -count=1` regression after relationship safety was broadened from references to all link usages.
4. `tsk_f51dd4d34f131815fb924fbfd64e133b` passed the devil's-advocate edge tests proving semantic resolution without a promoted multiline-definition owner and proving unresolved reference-looking syntax creates no relationship.
5. `tsk_d848d7f30ef6e14e821e26a75673fbaa` passed focused plus full regressions after removing the now-redundant internal M93 `ReferenceUsage` aliases and `ParseWithReferenceUsages` wrapper.
6. `tsk_106035467bc42bd47a25b6d3f2d398a4` passed verbose M98/M99 focused tests, relevant internals, and the complete repository regression after the relationship projection was made expected-linear and the bare-autolink/front-matter cases were added.

One intermediate focused run failed because the test fixture placed a candidate reference definition directly after a paragraph line, so GFM correctly did not parse it as a standalone link-reference definition. The implementation was not widened to reinterpret that invalid structure; the fixture was corrected instead.

## Devil's advocate review

1. **Relationship enumeration could accidentally promote syntax that is not safely editable.** Mitigation: relationships are immutable semantic facts; source-node identity is optional and only references existing promoted nodes. No relationship carries a generic mutation span.
2. **Reference owner matching could select the wrong normalized definition.** Mitigation: match parser-defined normalized label plus resolved destination/title facts, require exactly one candidate, and require that candidate to be in the existing promoted definition subset. Otherwise omit the owner ID.
3. **Unresolved reference-looking source could be mistaken for a broken relationship.** Mitigation: M99 enumerates only parser-resolved semantic relationships. Diagnostics for unresolved source syntax belong to M101 rather than an ad-hoc lexical reinterpretation here.
4. **Local-fragment classification could drift from `ResolveFragment`.** Mitigation: both paths use the same M98 normalization/status/catalog primitives and exact `FragmentTarget` representation.
5. **Relationship intelligence could become quadratic on documents with many links/definitions/headings.** Mitigation: use ephemeral source-node, definition-owner, and fragment maps so enumeration is expected O(N+H+R), with no all-pairs scan.
6. **M99 could become implicit workspace/network authority.** Mitigation: only a destination beginning with `#` is resolved inside the current snapshot. Every other destination remains opaque data for an explicit caller-provided M100 document set or an external host.
7. **Generalizing M93 safety metadata could weaken removal/composition proof.** Mitigation: the same compact comparison mechanism is retained but now covers a superset of semantic relationships; owned removed usages may disappear while surviving usages must still match after anchor transformation.
8. **Temporary Goldmark details could leak into the future native-parser contract.** Mitigation: only Marksplice `LinkUsage` scalars and the parser-defined normalization operation cross the adapter boundary. No AST type or Goldmark enum is public or stored in splice/public values.

## Release-quality verification

The final refactored implementation tree passed the substantive M99 freeze:

- `tsk_a562e1e95868589477aece6334a2aed6`: five consecutive complete `go test ./... -count=1` runs plus full race detection;
- `tsk_627f19769807ff85bd06648aaf606e3f`: gofmt cleanliness, `go vet`, `go build`, the executable `ExampleDocument_LinkRelationships`, and `go doc` proof for the new public relationship contract;
- `tsk_2987a484a035eb2408fa87c996487f7c`: Staticcheck plus golangci-lint with zero issues;
- `tsk_3698ce66de9b732a62f5e282c40e2b4b`: production `gocyclo <= 15`, production and test-inclusive `unparam`, `go mod tidy -diff`, and `git diff --check`;
- `tsk_a1cd40f31736c0c509cd7531e6645301`: direct Go 1.27.0 test, vet, and build;
- `tsk_8b0cdde3fae4a631708a34b1e07ce751`: govulncheck, Gitleaks, and actionlint;
- `tsk_58007c429652adde704da5db6cce36c5`: exact anchored `^TestGFM029PublishedSpecificationConformance$` execution against the approved private published-GFM snapshot; verbose output proves the real test ran and passed;
- `tsk_229692374d1c475a024e914ec520b00a`: corrected explicit production-package cross coverage at **86.8%** aggregate and **84.1%** through `internal/publictest`; both temporary profiles were created only under the private tool root and the task proved `PROFILES_REMAIN=False`.

Two implementation-freeze invocations are intentionally non-authoritative harness evidence. `tsk_17cb54510efbe08ef2bbdb1e01bc9949` completed its commands with exit code 0 but the runner reported an output-drain timeout, so its quality/static/maintainability work was rerun in the three successful authoritative tasks above. `tsk_5fb392f3e9a4ac630846ddd7cd683e16` returned PASS only because the exact GFM test was skipped when `MARKSPLICE_GFM_SPEC_HTML` was absent; that false-green was detected from verbose output and is not conformance evidence. The explicit-snapshot rerun `tsk_58007c429652adde704da5db6cce36c5` is the authoritative GFM gate.

`go.mod` and `go.sum` remain unchanged. The final documented tree then passed `tsk_59cd832c2305a767dd4b31a4717a0bcd` (complete regression, verbose focused M99 suite, vet/build, executable example, API docs, and explicit-snapshot exact GFM `RUN/PASS`), `tsk_92500e017a668b74fc3ba337ae7d6d85` (gofmt, Staticcheck, golangci-lint with zero issues, production gocyclo <=15, both unparam modes, tidy/diff, govulncheck with no vulnerabilities, Gitleaks with no leaks, and actionlint), `tsk_e8bf656dbd3f9659f7e917553910f297` (no coverage artifacts, no private workspace/tool/brief path leakage, unchanged `go.mod`/`go.sum`, `git diff --check`, `git fsck --no-dangling`, and expected branch/HEAD/origin/working-tree inventory), and corrected strict text/state hygiene `tsk_fbd83904d2a1f2ae5819e4c2575d07bd` (UTF-8/no-BOM/LF/no-trailing-whitespace/NUL, no artifacts, unchanged modules, diff/fsck, HEAD/origin/status). The preceding `tsk_60f6e9dea5248e1ca3cf9f4ac891f267` did not execute any check because PowerShell rejected an ambiguous diagnostic string interpolation and is discarded as harness-only evidence.

## Exit decision

M99 is complete. Single-document relationship intelligence remains an immutable source-ordered projection over parser-resolved `LinkUsage` facts with optional existing promoted identities, exact M98 local-fragment integration, expected-linear ephemeral lookups, no generic mutation authority, and no filesystem/network access or persistent graph. M100 — multi-document graph over an explicit caller-provided document set — is the next milestone.
