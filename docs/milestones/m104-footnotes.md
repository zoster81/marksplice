# M104 — Footnotes

Status: complete implementation contract; pre-documentation verification is recorded below and the documented-tree release-quality freeze is recorded after source-of-truth alignment.

## Objective

M104 adds footnotes as a Marksplice core document capability with exact source ownership, immutable reference relationships, conservative construction, source-preserving mutation, and integration with existing query/composition/link/graph surfaces.

Footnotes are not part of Marksplice's pinned GFM 0.29 conformance baseline. The baseline GFM parser remains authoritative for normative GFM syntax. M104 uses an isolated temporary semantic pass for the explicitly reviewed footnote contract; this does not create a caller-selectable dialect mode, a first-party extension family, or permission to broaden unrelated Markdown semantics.

The reviewed syntax uses exact `[^label]` references and top-level `[^label]: body` definitions. Label matching is exact and case-sensitive. Ordinary reference-definition construction reserves caret-prefixed labels so the two syntax families cannot be emitted ambiguously by `DocumentBuilder`.

## Public read model

M104 introduces one public structural kind for source-proven top-level definitions:

```go
const KindFootnoteDefinition Kind = ...

type FootnoteDefinition struct { /* immutable */ }

func (f FootnoteDefinition) ID() NodeID
func (f FootnoteDefinition) Range() Range
func (f FootnoteDefinition) Label() string
func (f FootnoteDefinition) LabelRange() Range
func (f FootnoteDefinition) BodyRange() (Range, bool)

func (d *Document) FootnoteDefinitions() []FootnoteDefinition
func (d *Document) FootnoteDefinition(id NodeID) (FootnoteDefinition, bool)
func (d *Document) FootnoteDefinitionBodyRanges(id NodeID) ([]Range, bool)
```

`FootnoteDefinition.Range()` owns the complete source-proven physical definition container, including continuation/blank lines that belong to the definition. `LabelRange()` owns only the exact label bytes. `BodyRange()` is deliberately narrower: it is available only when the parser-backed semantic body is one non-empty source span on the opening physical line. Broader multiline/segmented bodies remain readable through caller-owned `FootnoteDefinitionBodyRanges` without gaining guessed mutation authority.

References are relationship data rather than public structural nodes:

```go
type FootnoteReference struct { /* immutable */ }

func (r FootnoteReference) Range() Range
func (r FootnoteReference) LabelRange() Range
func (r FootnoteReference) Label() string
func (r FootnoteReference) DefinitionID() (NodeID, bool)
func (r FootnoteReference) Occurrence() int

func (d *Document) FootnoteReferences() []FootnoteReference
```

`Range()` owns the exact `[^label]` token. `DefinitionID` is present only when the referenced definition also passed Marksplice's complete top-level source-ownership proof. `Occurrence` is zero-based in reference source order for the same parser-resolved definition. Returned slices/range vectors are caller-owned defensive copies.

Unused source-proven definitions remain readable. An unmatched differently-cased reference is not case-folded into a relationship.

## Parser and source-ownership boundary

The ordinary Goldmark adapter continues to parse Marksplice's normative GFM profile. M104 adds a second adapter-local parser instance configured only for the temporary footnote semantic pass. An AST transformer observes footnote definitions/references before Goldmark's footnote processing reorders/removes its temporary nodes; only Marksplice-owned anchors, labels, body ranges, reference ranges, definition anchors, occurrence values, and ordinary link usages leave the adapter.

The second pass is reconciled with the normative GFM observations rather than replacing them. Physical source regions claimed by parser-proven footnote definitions suppress conflicting baseline observations, link usages, and unresolved-reference observations only inside those exact regions. Outside those regions the baseline GFM parse is unchanged. Claimed ranges are normalized into sorted non-overlapping intervals; containment tests use ordered lookup rather than all-pairs scans.

Links/images/autolinks parsed inside footnote definitions are retained as ordinary parser-resolved link usages and merged into the authoritative source-ordered M99 relationship vector. This lets existing link intelligence and M100 graph construction see relationships contained in footnote bodies without creating a separate link subsystem.

`internal/source.MapTopLevelFootnoteDefinition` independently proves physical ownership against the immutable source snapshot:

- the definition has a top-level prefix of at most three ordinary spaces;
- the exact source token agrees with the parser-proven `[^label]:` spelling;
- the complete opening line and supported blank/continuation lines are owned without absorbing following unrelated source;
- every parser-proven semantic body range is valid, non-empty, single-physical-line, non-overlapping, and contained by the owned definition;
- only a single opening-line semantic body span becomes the editable `BodyRange`.

Parser/source disagreement or unsupported ownership remains unpromoted rather than producing a partial public definition.

## Existing-source mutation

M104 adds two source-bound operations:

```go
func (d *Document) PrepareReplaceFootnoteDefinitionBody(id NodeID, replacement []byte) (ChangeSet, error)
func (d *Document) PrepareRenameFootnote(id NodeID, replacement []byte) (ChangeSet, error)
```

Body replacement requires the conservative simple `BodyRange`, a non-empty single physical line, and reparses the candidate. The operation may legitimately change relationships introduced inside that owned body, but definitions, footnote references outside the body, and ordinary link relationships outside the body must survive with the exact transformed source/semantic model.

Rename is a coordinated multi-patch operation over the definition label plus every parser-proven reference occurrence bound to that exact definition. The replacement label must be non-empty, single-line, bracket-free, and must not collide with another promoted definition. Candidate proof requires the same definition/reference cardinality and ownership, the requested exact label, unchanged occurrence ordering, and unchanged ordinary link relationships after patch-coordinate transformation.

Both operations inherit ordinary snapshot fingerprinting and stale-source rejection. Untouched bytes are never regenerated.

## Construction

`DocumentBuilder` supports the same conservative core capability through:

```go
func (b *DocumentBuilder) AppendFootnoteDefinition(label, body string) error
func (b *DocumentBuilder) DeferFootnoteDefinition(label, body string) error
func FootnoteReferenceInline(label string) Inline
```

Definitions are canonical top-level one-line `[^label]: body` blocks. `DeferFootnoteDefinition` explicitly schedules the definition after ordinary body blocks and deferred ordinary reference definitions, enabling forward typed footnote references without hidden reordering of previously appended blocks.

`FootnoteReferenceInline` resolves to exactly one already-appended or explicitly deferred definition in the destination builder. Resolution is exact and case-sensitive. It is accepted only in the ordinary top-level typed-inline sequence; the existing structured-child policy continues to reject it inside emphasis/link/image label hierarchies where that composition has not been reviewed.

Construction proof reparses the generated definition and requires exact label/body/source mapping. Typed-reference proof builds temporary canonical definitions solely in proof input, checks exact source ranges, labels, definition ownership, and occurrence ordering, and never emits those proof definitions into the requested inline bytes. Quoted blockquote/alert child builders reject deferred footnotes rather than silently dropping deferred state.

## Integration with existing core surfaces

- **M95 composition:** footnote references are part of the semantic composition model. Equal-cardinality model changes can now be represented as multiple disjoint changed runs, allowing a single validated footnote rename to affect separated reference/definition regions without falsely claiming unchanged nodes between them. Cardinality-changing composition retains the older conservative contiguous-delta behavior.
- **M97 queries:** `KindFootnoteDefinition` participates in the existing fixed-size, caller-bounded `QueryNodes` kind filter and uses the definition's complete owned range as its selection span.
- **M99 link intelligence:** links/images/autolinks inside footnote bodies are present in the ordinary source-ordered `LinkRelationships` projection.
- **M100 graph:** no new graph edge type is required. Footnote-body M99 relationships participate in graph construction normally. The footnote reference-to-definition relation itself remains an intra-document `FootnoteReference.DefinitionID` relationship rather than a document graph edge.
- **M101 diagnostics:** caret footnote syntax is removed from conflicting ordinary GFM reference/unresolved-reference observations before workspace diagnostics consume the snapshot, preventing one source token from representing two unrelated relationship families.

No persistent footnote-definition lookup map, reverse-reference index, footnote graph, parser AST, or parser context is retained in `Document`.

## Complexity and retained state

The isolated semantic pass adds one additional parser pass while Goldmark remains the temporary backend. Footnote reconciliation retains one source-ordered definition observation vector and one source-ordered reference vector for the parse operation. Claimed physical ranges are normalized once and tested with ordered interval lookup, avoiding nested observation-by-definition scans.

The immutable `Document` retains only source-proven definition nodes in the existing node array plus one compact source-ordered `FootnoteReference` vector. Definition-ID resolution uses an ephemeral parse-time anchor map. Public enumeration/query methods reuse existing source order and allocate only caller-owned result storage.

Mutation proof performs the same deliberate candidate reparse used by other source-preserving structural operations. M95 retains its established O(k·N) composition proof model; the equal-cardinality multi-run delta representation adds no persistent index and avoids treating unchanged model spans between coordinated patches as modified.

## Devil's advocate review

### Risk 1 — footnote syntax corrupts the normative GFM profile

The pinned GFM 0.29 corpus does not define footnotes, while the temporary backend exposes them as an extension. Enabling that extension in the authoritative parser would silently change baseline Markdown. Mitigation: keep the normative GFM parser unchanged and run footnotes through an isolated semantic pass whose claimed regions are reconciled explicitly.

### Risk 2 — a baseline GFM reference definition and a footnote claim the same bytes

`[^label]: ...` can overlap ordinary reference-definition or paragraph interpretations in the baseline parser. Mitigation: parser-proven footnote regions suppress competing baseline nodes/usages/unresolved observations only within exact physical claimed ranges; ordinary GFM behavior outside remains authoritative.

### Risk 3 — multiline semantic bodies become broad rewrite authority

A parser can understand a multiline footnote even when no single contiguous semantic payload span can be safely replaced. Mitigation: expose complete definition ownership and caller-owned body segments for reading, but grant `PrepareReplaceFootnoteDefinitionBody` only when one opening-line `BodyRange` is independently proven.

### Risk 4 — rename updates the definition but misses one backlink occurrence

A lexical search for `[^label]` could touch code/text that is not a footnote or miss parser-resolved occurrences. Mitigation: rename patches only the definition's source-proven label and parser-proven reference label ranges bound to that definition, then reparses and proves cardinality, occurrences, ownership, and unrelated link relationships.

### Risk 5 — footnote-body links disappear from document intelligence

The baseline parser may classify the definition container differently, causing links inside it to vanish when conflicting observations are removed. Mitigation: the isolated footnote pass collects ordinary link usages within the footnote AST before reconciliation and merges them into M99 source order; M100 therefore requires no special-case crawler.

### Risk 6 — definition lookup becomes an all-pairs or persistent index

Repeated scans across many footnotes/observations could create poor scaling, while a retained reverse index would duplicate snapshot state. Mitigation: parse-time claims are sorted/merged and queried by ordered lookup; definition resolution uses ephemeral maps; retained state remains the ordinary node array plus compact reference vector.

## Verification

M104 was developed through focused RED/GREEN tests covering parser reconciliation, source mapping, body replacement, coordinated rename, builder construction, M95 composition, M97 queries, and M99/M100 relationship integration.

The documented tree passed repeated full repository regressions, the race detector, examples and public API documentation checks, `go vet`, `go build`, Staticcheck, golangci-lint, production `gocyclo <= 15`, production/test-inclusive unparam, `go mod tidy -diff`, published GFM 0.29 conformance, Go 1.27 test/vet/build, govulncheck, Gitleaks, actionlint, strict text/artifact hygiene, and `git diff --check`.

Cross-package statement coverage measured **86.7% aggregate** and **84.3% through `internal/publictest`**.

## Exit decision

M104 is **complete**. Marksplice now has a conservative core footnote capability with parser-proven exact references/occurrences, independently source-proven top-level definitions, readable multiline bodies, narrow simple-body replacement, coordinated source-preserving rename, deterministic construction/forward references, bounded query/composition integration, and ordinary M99/M100 relationship/graph visibility for links contained in footnotes.

The next roadmap boundary is **M105 — Mathematical expressions**.
