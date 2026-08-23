# M95 — Structural Mutation Composition

## Status

Complete. Focused TDD, implementation-freeze verification, corrected cross-package coverage, and the full documented-tree release-quality gate are green. The final post-evidence tree is reverified below.

## Objective

Allow callers to atomically compose multiple already-prepared Marksplice mutations without exposing a generic raw-patch batching API. Composition must preserve exact snapshot binding, reject overlapping source patches, reject semantic interactions that are safe only in isolation, and validate one combined candidate against the independently proven intent of every constituent change.

M95 does not replace family-specific atomic operations. When two prepared changes intentionally modify the same structural model region, callers should continue to use the dedicated operation where one exists, such as `PrepareSetTableAlignments` for a complete alignment vector.

## Public contract

M95 adds:

```go
func (d *Document) ComposeChanges(changes ...ChangeSet) (ChangeSet, error)
```

The input values must already have been prepared by Marksplice against the same exact source snapshot represented by `d`.

- zero input changes produce a snapshot-bound no-op `ChangeSet`;
- one input change is accepted only if it is bound to the same source snapshot;
- multiple changes are applied atomically in original source coordinates;
- caller order does not define patch order or output bytes for independent changes;
- a zero/unbound or foreign-snapshot `ChangeSet` reports `ErrSourceConflict`;
- overlapping source patches, overlapping semantic model deltas, or a combined candidate whose parsed model differs from the independently validated deltas report `ErrInvalidReplacement`;
- `ChangeSet.Apply` retains its existing stale-source behavior after composition.

No public patch type, patch range list, arbitrary byte batch, or mutable transaction object is introduced.

## Requirements and edge cases

Every constituent remains bound to the original immutable snapshot. M95 never rebases a prepared mutation onto the result of another mutation and never silently changes source coordinates according to caller order.

Source overlap is rejected before semantic composition. This includes ordinary overlapping replacement/removal ranges and same-offset insertions. Coordinated multi-patch operations such as one list/table/section move remain one constituent change and keep the proof established by their original operation.

Disjoint source patches are not automatically safe. Two individually valid removals can interact through Markdown parsing after both separators disappear. M95 therefore treats individual validation as necessary but not sufficient and reparses the single combined candidate.

Reference relationships participate in the combined proof. A reference-definition destination update may alter parser-resolved reference usages outside the changed definition line; independent reference deltas compose only when the resulting reference-usage model is exactly the union of the individually validated changes.

Aggregate structures are deliberately conservative. If two constituent operations both change the same logical table model, their model deltas overlap even when the underlying delimiter patches do not. M95 rejects that composition instead of inventing merge semantics. Separate list-item content updates, including parent and child lines, can compose when their source-owned lines and independently validated structural deltas remain distinct.

## Architecture and test strategy

### Source layer

`internal/source.ComposeChangeSets` accepts only already-created `source.ChangeSet` values. It verifies every constituent fingerprint against the supplied source, concatenates their private patch sets, and sends the result back through `NewChangeSet`. The existing patch validator therefore remains the single authority for source range validity, source ordering, same-offset insertion conflicts, overlap rejection, defensive replacement copies, and final snapshot fingerprinting.

`source.ChangeSet.Patches` returns a defensive copy solely inside Marksplice's internal boundary so `internal/splice` can derive composition evidence without exposing patches through the root package.

### Combined model proof

For two or more changes, `internal/splice` performs four steps:

1. apply every constituent independently to the original snapshot and parse that already-validated single-operation candidate;
2. derive a compact ordered node-model delta and reference-usage delta by comparing the original model with each individual candidate;
3. reject constituent deltas whose original structural/reference regions overlap, then compose the non-overlapping expected model deltas in source/model order;
4. apply all source patches together, parse exactly one combined candidate, and require its compact node and reference views to equal the composed expected model.

The node composition view is parser-independent. It contains the durable semantic facts already used by whole-block survivor validation, source-relative relationship topology rather than snapshot IDs, table alignment values, list completeness/child facts, source-owned range geometry, a compact fingerprint of the operation-owned source, and M94 blockquote marker/content segmentation. Absolute byte offsets and snapshot-specific `NodeID` values are deliberately excluded so unrelated earlier patches may shift a valid node without changing its model view.

The model-delta algorithm uses the longest common prefix and suffix of comparable views. A simple local edit therefore produces a small model interval. A move can conservatively produce one larger interval spanning source and destination; this may reject another otherwise-safe change inside that interval, but it cannot silently merge two overlapping structural intents. This is an intentional fail-closed tradeoff rather than an attempt to solve arbitrary semantic three-way merging.

Reference usage views similarly preserve kind, reference form, normalized reference value, resolved destination/title, and title presence while excluding absolute anchors. Anchor/source-coordinate safety remains owned by each original mutation proof and the exact combined source snapshot.

For `k` composed changes over a document/model of size `N`, the current proof is intentionally O(k·N): it compares each independently prepared candidate with the original and then parses one combined candidate. There is no nested all-pairs node comparison, subtree rehashing, or persistent composition index. M108 performance hardening will benchmark this explicit tradeoff before any attempt to cache or retain additional mutation proof state.

## TDD evidence

The first RED was in `internal/source`: new composition tests failed to compile because `ComposeChangeSets` did not exist. The second RED was black-box public API compilation: `Document.ComposeChanges` did not exist.

Focused tests then proved:

- source-level composition of disjoint same-snapshot changes;
- source-level rejection of foreign snapshots and overlapping patches;
- snapshot-bound zero-change composition;
- defensive-copy behavior of internal patch inspection;
- atomic heading + paragraph + thematic-break removal + blockquote removal composition;
- caller-order independence for the same prepared changes;
- rejection of two individually safe thematic-break removals whose combined result would merge two paragraphs;
- stale/zero/foreign `ChangeSet` rejection;
- list-item move composed with table-row insertion;
- section insertion composed with task state update;
- independent sibling list-item content replacements;
- independent nested parent/child list-item content replacements;
- independent reference-definition destination updates while resolved reference usages also change;
- conservative rejection of two separate alignment updates to the same table model;
- executable pkg.go.dev-style composition example.

## Devil's advocate review

1. **Patch disjointness could be mistaken for semantic independence.** Two consecutive thematic breaks demonstrate the failure: removing either one alone preserves the two surrounding paragraphs, while removing both merges them. M95 derives individual model deltas and validates one combined parsed candidate, so this interaction is rejected.
2. **A generic composition API could become a raw-patch escape hatch.** The public API accepts only opaque prepared `ChangeSet` values. Patch extraction exists only in `internal/source`; caller-provided ranges or replacement batches cannot enter M95 directly.
3. **Snapshot rebasing could make caller order observable and invalidate original proofs.** M95 never sequentially rebases changes. Every constituent must match the original source fingerprint, all patches stay in original coordinates, and the combined source layer sorts/validates them as one set.
4. **Shared logical models could be merged accidentally even when byte patches differ.** Composition compares compact structural deltas, not just patch ranges. Two separate changes to one table alignment vector are rejected; callers can use the existing dedicated vector operation instead.
5. **Reference-definition edits could silently change usages outside the changed line.** Reference-usage deltas are composed and checked independently from node deltas, so the final parser-resolved relationship set must be exactly the expected union.
6. **Composition proof could become superlinear on deep lists.** The initial view used a complete list-item subtree as the source-owned fingerprint span. Review identified repeated ancestor hashing as unnecessary. M95 now fingerprints only each list item's already-proven physical line while relationship/completeness facts remain semantic fields; parent/child content updates remain independently composable and the production complexity gate stays green.
7. **Move deltas may be broader than strictly necessary.** Longest-prefix/suffix differencing intentionally collapses a move's two changed positions and the intervening reordered model into one conservative interval. This can cause false rejection of another change in that interval but avoids unsafe partial merge semantics and keeps the algorithm simple and bounded.
8. **Composition can require repeated candidate-model scans as `k` grows.** The current O(k·N) proof is explicit and bounded by the number of caller-supplied prepared changes; it avoids quadratic node matching and stores no persistent proof cache. M108 must benchmark realistic batch sizes before any optimization that would add mutable/lazy evidence or long-lived indexes to `ChangeSet`.

## Verification

Implementation-freeze evidence obtained before documentation finalization:

- five consecutive `go test ./... -count=1` runs passed;
- full `go test -race ./... -count=1` passed with the private CGO-capable toolchain;
- `go vet ./...`, `go build ./...`, executable examples, and `go doc` for `Document.ComposeChanges` passed;
- Staticcheck passed;
- golangci-lint reported zero issues;
- production `gocyclo -over 15` is empty;
- production and test-inclusive `unparam` passed;
- `go mod tidy -diff` and `git diff --check` passed;
- govulncheck found no vulnerabilities and Gitleaks found no leaks;
- actionlint passed;
- direct Go 1.27.0 test/vet/build passed.

The first coverage command is explicitly discarded: PowerShell passed the literal `$coverpkg` token to Go, producing warnings and meaningless 0.0% profiles. The corrected rerun passed the package list literally and reports **86.5%** aggregate statement coverage over the explicit production package set and **83.7%** through `internal/publictest` alone.

The exact documented tree then passed five consecutive complete test runs; full race detection; vet/build; executable examples and `go doc` for `Document.ComposeChanges`; Staticcheck; golangci-lint with zero issues; production gocyclo ≤15; production and test-inclusive unparam; `go mod tidy -diff`; `git diff --check`; the hash-pinned published GFM 0.29 conformance gate; govulncheck; Gitleaks; actionlint; direct Go 1.27.0 test/vet/build; strict UTF-8/no-BOM/LF/no-trailing-whitespace/control-character hygiene; private-boundary scanning; and `git fsck --no-dangling`.

After recording this evidence, the final public tree is rechecked so the milestone-status/evidence edit itself is not left unverified.

## Exit decision

M95 is complete. Marksplice can atomically compose independent reviewed prepared mutations without exposing raw patch batching, rebasing prepared changes, or weakening semantic/source proof. M96 single-document read/edit/create audit is the next roadmap boundary.
