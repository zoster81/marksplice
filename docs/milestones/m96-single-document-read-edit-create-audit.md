# M96 — Single-document read/edit/create audit

## Status

Complete. Focused TDD, full documented-tree release-quality verification, corrected cross-package coverage, artifact hygiene, and final diff review are green. M97 structural query surface is the next implementation boundary.

## Objective

Audit the single-document core family by family across:

1. semantic understanding;
2. reviewed public structural reading;
3. exact existing-source editing;
4. canonical new-document construction;
5. structural navigation/relationships;
6. exact source ownership.

M96 closes only gaps that are both high-value and already supported by a proven Marksplice semantic/source model. It does not add APIs merely to make the matrix symmetric and does not consume contracts deliberately assigned to later milestones.

## Audit result

| Family | M96 conclusion |
| --- | --- |
| Paragraphs/headings/sections | Existing read/edit/construction/navigation boundaries are coherent. Setext remains a parsed-source preservation/edit capability; construction intentionally uses canonical ATX. |
| Lists/tasks | Existing flat/homogeneous nested construction and complete-subtree edit/navigation model are coherent. A new mixed nested ordered/unordered construction model would be a separate API/structure contract and is not justified by this audit alone. |
| Fenced code | Existing non-empty contiguous-body contract remains unchanged. Empty-body promotion/construction is not a one-line gap because `internal/source.MapFencedCode` currently requires a non-empty semantic content range; M103 owns the broader fence/info/content ownership contract. |
| GFM tables | **Closed in M96:** canonical builder tables may now have zero body rows. Construction adds an authoritative `KindTable` expectation that proves complete table range, anchor, width, body-row count, alignments, header/delimiter mapping, plus the existing per-body-row proofs when rows are present. This strengthens existing table construction as well as enabling header-only tables. |
| Table alignment | Existing read/edit/construction semantics are coherent; header-only aligned tables are covered by the new table-container proof. |
| Emphasis/strong/strikethrough/code spans | Existing simple parsed-source promotion and bounded typed construction are coherent; widening compound existing-source editability is not required for single-document completeness. |
| Inline links | **Closed in M96:** promoted simple inline links now expose parser-proven semantic `Destination()` and optional `Title()` in addition to the exact destination replacement range. Broader reference relationships remain M99. |
| Images | Existing source-proven destination edit and typed construction remain coherent. Goldmark's public image AST does not provide the same pinned semantic destination/title facts used for links, so M96 does not invent a semantic image getter contract. |
| Reference definitions | **Closed in M96:** promoted definitions now expose parser-proven `Label()`, `Destination()`, and optional `Title()` while `Range()` remains the exact destination mutation span and complete-line ownership remains private removal proof. Relationship intelligence remains M99. |
| Autolinks | **Closed in M96:** promoted autolinks now expose parser-proven `Value()` and `IsEmail()` while retaining the exact token replacement range. |
| Thematic breaks/blockquotes | Existing complete physical ownership and fail-closed removal contracts are coherent. |
| YAML/TOML front matter | Existing public key/format/range plus safe replacement and canonical construction are coherent for the intentionally narrow scalar subset. A decoded/general metadata value contract is not stored today; M106 owns metadata/front-matter generalization. |
| HTML | Existing simple comment/anchor read/edit subset plus opaque preservation is deliberate. No dedicated HTML builder is added solely for API symmetry. |
| Mutation composition | M95 already provides atomic composition of independent same-snapshot prepared changes; M96 found no single-document mutation-family gap that requires raw patch exposure or a second mutation model. |

## Requirements and edge cases

### Semantic read closure

The new getters expose only immutable scalar facts already stored in `internal/splice.Node` from parser-proven observations:

- `InlineLink.Destination()`;
- `InlineLink.Title() (string, bool)`;
- `ReferenceDefinition.Label()`;
- `ReferenceDefinition.Destination()`;
- `ReferenceDefinition.Title() (string, bool)`;
- `AutoLink.Value()`;
- `AutoLink.IsEmail()`.

No parser or source-mapper behavior changes. The typed detail values remain comparable because only strings and booleans are added. `Range()` keeps its historical operation-oriented meaning; semantic getters do not redefine source ownership.

### Header-only table construction

GFM table semantics and Marksplice's parsed table/source model already support `BodyRowCount == 0`. M96 therefore removes only the builder-specific body-row requirement, while retaining all existing validation for:

- at least one header column;
- exact alignment-vector width;
- body-row width when body rows exist;
- valid single-line UTF-8 cell source;
- no raw pipe/newline/NUL ambiguity under the canonical writer contract.

The writer emits a `KindTable` construction expectation for every generated table, not only header-only tables. It proves the complete generated table span and semantic/source table facts. Existing `KindTableRow` expectations continue to prove each body row independently when present.

## Architecture and test strategy

The two closures are intentionally independent:

- semantic getters are root-package projection of existing immutable internal facts;
- header-only table construction extends builder validation/writer proof without changing parsed-table semantics or existing-document mutation code.

TDD sequence:

1. black-box header-only table tests failed with the historical `table requires header columns and a body row` validation error;
2. black-box semantic-read tests failed to compile because the getters did not exist;
3. semantic getters were added with no parser/source changes and their focused tests passed;
4. table construction gained a table-container expectation/proof and body rows became optional;
5. the only initial table regression was the historical test that deliberately classified zero body rows as invalid; that obsolete premise was removed while all other invalid-table cases remained;
6. the complete public table family then passed.

## Devil's advocate review

1. **Header-only construction could be enabled by merely deleting a validation guard, leaving no authoritative construction expectation.** Mitigation: every constructed table now emits and validates a `KindTable` expectation for complete source range, table anchor, width, body-row count, alignments, header cells, delimiter cells, and delimiter-alignment mapping. Existing body-row proof remains additive.
2. **Semantic getters could accidentally expose lexical source bytes as if they were parser semantics.** Mitigation: only fields already populated as semantic parser observations in `splice.Node` are exposed. Image semantics, front-matter decoded values, and reference-link relationship APIs are not inferred from source trivia.
3. **New semantic fields could make public detail values non-comparable or caller-mutable.** Mitigation: the added fields are only strings and booleans; no slice/map/reference storage is introduced.
4. **M96 could steal later milestone scope and create conflicting contracts.** Mitigation: empty/general fenced-block ownership is explicitly deferred to M103, link/reference relationship intelligence to M99, and metadata generalization to M106. HTML construction and mixed nested-list construction remain deliberate non-closures because the audit found no source-safety defect that requires them now.

## Verification

Focused TDD and regressions:

- RED: header-only table construction rejected by the historical body-row guard;
- RED: new semantic getter tests failed to compile because the methods were absent;
- GREEN: semantic getter focused tests;
- GREEN: complete public table-family tests after adding `KindTable` construction proof and updating the obsolete zero-body-row rejection premise;
- GREEN: combined M96/link/reference/autolink/table public tests;
- GREEN: one complete `go test ./... -count=1` regression;
- GREEN: `gofmt` clean and `git diff --check`;
- GREEN: production gocyclo <= 15;
- GREEN: production and test-inclusive unparam.

Documented-tree release-quality gate:

- `tsk_c68a20b069f2cbcaf40a2e5a5fe74817`: five consecutive complete `go test ./... -count=1` runs plus full `go test -race ./... -count=1` passed;
- `tsk_fb8588dc3d3ae884706e865a10fa719f`: `go vet ./...`, `go build ./...`, executable examples and package documentation checks, Staticcheck, golangci-lint, production gocyclo <= 15, production and test-inclusive unparam, `go mod tidy -diff`, `git diff --check`, hash-pinned published GFM 0.29 conformance, govulncheck, Gitleaks, and actionlint passed;
- `tsk_14a80cbdfd5a12669f1073a95322ee8d`: direct Go 1.27.0 test/vet/build passed;
- `tsk_e9bc380fe48ef9c4df73ff3b07d1cb24`: corrected explicit cross-package coverage passed with **86.5%** aggregate statement coverage over the production package set and **83.7%** through `internal/publictest`; private coverage profiles were deleted and their absence verified before exit;
- `tsk_1538ab92a6dfddb8027f5ca2647c79ee`: strict UTF-8/no-BOM/LF/no-trailing-whitespace/control-character hygiene, public/private-boundary scanning, coverage-artifact absence, `git diff --check`, `git fsck --no-dangling`, and branch/HEAD/status checks passed.

Several earlier coverage/hygiene invocations were discarded as harness failures rather than product evidence: one PowerShell command passed a literal `$coverpkg`, later variants created literal dollar-named coverage artifacts or contained PowerShell syntax errors. Every confirmed harness artifact was inspected inside the authorized workspace, removed, and the corrected coverage/hygiene gates above were rerun successfully.

## Exit decision

M96 is complete. Phase A of the post-beta roadmap is closed without adding symmetry-only APIs or consuming M99/M103/M106 contracts. M97 — structural query surface — is the next implementation boundary.
