# M106 — Metadata/front-matter generalization audit

Status: complete.

## Objective

M106 reassesses the conservative YAML/TOML front-matter model against common metadata shapes without turning Marksplice into a YAML/TOML parser, metadata AST, schema system, or serializer.

The audit found one high-value architectural gap: prior front-matter recognition was coupled to the existence of at least one unique simple scalar field safe for mutation. A closed leading metadata envelope containing only arrays/lists, maps/tables, duplicate keys, or no fields therefore fell back to ordinary Markdown even though its bytes were metadata for document-level purposes. That could expose metadata-looking links or references to Markdown intelligence and made the public model unable to report the envelope itself.

M106 separates two independent questions:

1. is this source a recognized document-leading metadata envelope?;
2. does this envelope contain any individually source-proven field that Marksplice may edit safely?

Only the second question grants mutation authority.

## Public read model

M106 adds one immutable document-envelope value:

```go
type FrontMatter struct { /* immutable */ }

func (f FrontMatter) Format() FrontMatterFormat
func (f FrontMatter) Range() Range
func (f FrontMatter) OpeningRange() Range
func (f FrontMatter) ClosingRange() Range

func (d *Document) FrontMatter() (FrontMatter, bool)
```

`FrontMatter` is deliberately not a structural Markdown `Node` and adds no public `Kind`. It reports only source ownership and the reviewed YAML/TOML envelope format. `Range()` owns the exact envelope from the opening delimiter through the closing delimiter; a following physical line terminator is outside that range. The opening/closing accessors expose only the exact delimiter bytes.

The zero value is deterministic and a nil/zero `Document` reports no front matter.

## Envelope recognition and precedence

`internal/source.MapLeadingFrontMatter` remains the single parser-independent source proof for front matter. M106 broadens envelope recognition without broadening value parsing.

A recognized envelope must still:

- begin at byte zero;
- use an exact `---` YAML or `+++` TOML opening physical line;
- contain an exact matching closing delimiter physical line.

After those physical boundaries are proven, Marksplice recognizes the envelope when either:

- it is empty; or
- its body contains conservative metadata evidence.

For YAML, metadata evidence is a source-proven simple top-level key separator. The value may remain complex/opaque; recognizing the envelope does not require that value to be editable.

For TOML, evidence is either a source-proven simple top-level key separator or a conservative simple table/array-table header such as `[params]` or `[[products]]`. The table-header recognizer requires complete balanced brackets and the existing conservative key alphabet; a malformed `[` prefix is not metadata evidence.

This deliberately establishes one narrow precedence rule: an empty byte-zero `---` / `---` pair is a YAML front-matter envelope rather than two public thematic breaks. A non-empty `--- ... ---` region with no metadata evidence remains ordinary GFM source. Non-leading or unclosed candidate envelopes also remain GFM.

No YAML/TOML semantic validity claim is made for opaque body content beyond the reviewed evidence needed to establish document-envelope ownership.

## Editable-field boundary

The historical `FrontMatterField` and `PrepareReplaceFrontMatterValue` contracts remain the only front-matter field mutation surface.

Only unique source-proven simple top-level scalar fields are promoted. Complex values, duplicate target keys, nested YAML members, TOML table members, arrays/tables, multiline values, and other unsupported shapes remain opaque source.

M106 additionally stops TOML simple-field promotion after the first recognized table/array-table header. A line such as `author = 'Ada'` following `[params]` is therefore not misrepresented as a top-level field. A safely mapped field before the table remains eligible under the established contract.

Existing replacements still patch only the proven scalar payload and candidate-remap the complete envelope. Format, delimiter ownership, field key/style, shifted boundaries, untouched bytes, and stale-source protection remain unchanged.

## Construction

M106 intentionally adds no generic metadata-construction API.

`DocumentBuilder.SetYAMLFrontMatter` and `SetTOMLFrontMatter` retain the M80 contract: at most one document-leading envelope containing ordered unique simple fields written as conservative double-quoted strings. This deterministic subset remains useful for new documents without implying the ability to serialize arbitrary existing metadata structures.

Opaque/complex existing front matter is readable and source-owned, not regeneratable through a new general serializer.

## Integration with document intelligence

Once a complex, duplicate-only, or empty envelope is recognized, the complete envelope range participates in the same existing snapshot exclusion boundary previously used for simple front matter.

Parser-observed Markdown nodes and M99 link relationships whose source lies inside that envelope are not promoted into the Markdown model. M101 unresolved explicit-reference observations are likewise excluded. Footnote and mathematical observations inside the envelope remain outside Markdown semantics as well.

This is a boundary correction, not new relationship semantics: metadata payload stays opaque and produces no Marksplice graph edges or references.

## Complexity and retained state

Envelope mapping remains source-linear. It performs one bounded scan to find the exact closing delimiter and one bounded scan over envelope physical lines to collect conservative metadata evidence and safe field candidates. Unique editable fields use one temporary key-count map.

The immutable snapshot still retains only the compact envelope format/opening/closing facts already needed for mutation validation plus any existing promoted simple field nodes. `FrontMatter()` derives its complete range from those retained boundaries. M106 adds no metadata AST, parsed value tree, schema, persistent key index, reverse map, filesystem/network authority, or cache.

Production cyclomatic complexity remains at or below the repository limit of 15.

## Devil's advocate review

### Risk 1 — ordinary thematic-break Markdown is captured as YAML metadata

A naive rule that treated every closed leading `--- ... ---` pair as front matter would reinterpret ordinary GFM. M106 requires either an actually empty envelope or conservative metadata evidence. Non-empty source with no such evidence remains GFM. The empty-envelope precedence is explicit and tested rather than accidental.

### Risk 2 — TOML table members are exposed as top-level editable fields

The historical line scanner could otherwise map `key = value` syntax without knowing that a preceding `[table]` changed scope. M106 recognizes conservative table/array-table headers and stops top-level field promotion at that point. The payload after the table remains opaque.

### Risk 3 — generalized envelope reading becomes a promise of YAML/TOML semantics

The public `FrontMatter` value exposes only format and source ranges. It does not expose decoded values, paths, arrays, maps, aliases, dates, numbers, or schema semantics. No YAML/TOML dependency or parser enters the module.

### Risk 4 — duplicate or complex metadata becomes accidentally editable

Envelope recognition and field promotion are separate. Duplicate-only or complex-only envelopes can be read as front matter while exposing zero `FrontMatterField` mutation targets.

### Risk 5 — metadata text contaminates Markdown relationship intelligence

Expanded envelope recognition is applied before snapshot promotion and relationship projection. Links, unresolved references, footnotes, and math-looking source inside a recognized opaque envelope remain metadata, not Markdown model facts.

## Verification

M106 used focused RED/GREEN tests for YAML/TOML envelope recognition, empty-envelope precedence, TOML table headers, conservative field promotion, malformed/metadata-free rejection, existing field mutation, and M99/M101 metadata exclusion.

The documented tree passed repeated complete regressions, the race detector, the executable front-matter example and API documentation checks, `go vet`, `go build`, Staticcheck, golangci-lint, production `gocyclo <= 15`, production/test-inclusive unparam, `go mod tidy -diff`, published GFM 0.29 conformance, Go 1.27 test/vet/build, govulncheck, Gitleaks, actionlint, strict text/artifact hygiene, and `git diff --check`.

Cross-package statement coverage measured **86.8% aggregate** and **84.4% through `internal/publictest`**.

## Exit decision

M106 is complete. Envelope ownership is now independent from field mutation authority without introducing a metadata parser/serializer, and the exact documented tree has passed the release-quality freeze. The next roadmap boundary is **M107 — Knowledge-document primitives**.
