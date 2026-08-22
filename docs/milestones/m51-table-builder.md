# Milestone M51 — Canonical Unaligned Table Construction

Status: green — parser-proven construction of a complete GFM table with body rows.

## Goal

Add a useful structured GFM table constructor by reusing the existing M35–M43 table row/cell model, without introducing a public `Table` identity or prematurely exposing alignment/style options.

M51 adds:

```go
func (b *DocumentBuilder) AppendTable(header []string, rows ...[]string) error
```

## Public contract and canonical policy

M51 requires:

- at least one header column;
- at least one body row;
- every body row has exactly the header width;
- cells are caller-provided single-line GFM source;
- empty cells are allowed;
- non-empty cells are valid UTF-8, NUL-free, and have no physical line break;
- caller cell strings do not include outer horizontal padding; the canonical writer owns that padding.

The writer emits outer pipes, one padding space on each side of cell source, and unaligned delimiter cells spelled `---`:

```markdown
| Name | Value |
| --- | --- |
| alpha | *one* |
|  | two |
```

Alignment markers, alternate pipe/padding styles, header-only tables, and escaping of raw `|` cell syntax remain outside M51.

Header and body-row slices are deeply copied before they enter retained builder state.

## Parser/model proof

M51 does not create a public table node or table ID. It reuses the parser-independent `TableAnchor`, promoted M35 body `TableRow` mappings, and the M5 physical cell mappings embedded in each `TableRowSource`.

Every generated body row must prove:

- exact complete physical-line `Range`, including its LF;
- exact raw row range excluding the terminator;
- exact semantic/source-proven column count;
- exact private `TableAnchor` equal to the requested generated table start;
- exact per-column raw and content ranges, including physically represented empty cells;
- source-order column numbers matching the requested width.

The public regression additionally reparses successful output and verifies promoted body rows plus M43 header-cell navigation. Two adjacent separately requested tables are required to remain separate semantic table containers after final whole-document validation.

`TableAnchor` remains private parser-independent proof metadata only. M51 does not add a public `Table`, alter `NodeID` derivation, or change any `Kind` ordinal.

## Failure behavior

Missing header/body structure, mismatched row width, malformed cells, raw source that changes the requested table grammar, nil builders, or any parser/source-mapping disagreement fails with `ErrInvalidConstruction`. Rejected appends leave retained builder state unchanged.

M51 deliberately requires a body row because the current reviewed table-container proof is anchored by promoted M35 body rows. A header-only table constructor would need a separately reviewed ownership/proof contract rather than a weaker lexical assumption.

## Complexity

Deep copying and writing are O(k) in table source size. Each body row stores one temporary expected range pair per column for validation. Complete construction validation remains O(n) in generated document size and uses the existing parser/source table model.

## Devil's advocate review

### Equal-width text rows could belong to a different semantic table

Mitigation: proof checks the parser-derived private `TableAnchor` for every promoted body row in addition to exact physical row/cell mappings and semantic column count.

### Empty cells could disappear from public node enumeration

Mitigation: M51 does not require synthetic empty `TableCell` identities. Exact empty-cell ownership is proven through the physical `TableRowSource.Cells` mapping, while `ColumnCount` remains the semantic cardinality.

### A table API with alignment options could outrun the parser proof

Mitigation: M51 fixes canonical unaligned delimiter cells. Historical note: M62 later adds explicit default/left/right/center construction only after Goldmark's public semantic alignment state is mapped and proven per body row; the original `AppendTable` output remains unchanged.

### Caller mutation after append could change generated output

Mitigation: header and every body-row slice are copied before retention; public tests mutate caller storage after append and require unchanged output.

## TDD and verification evidence

The public M51 tests were written first. The red run failed to compile only because `AppendTable` did not exist. The first implementation run exposed only a private Go name collision between a construction kind and struct; renaming the kind resolved it without changing design. Focused M51 tests then passed.

After the M49–M51 proof-data refactor, the complete `TestPublicDocumentBuilder` suite and repository-wide `go test ./... -count=1` regression remain green. Production construction complexity stays below the project threshold.

M51 adds no parser dependency, parser metadata, public table identity, new public kind, `NodeID` change, or existing-document normalization path.

M62 later adds internal semantic alignment metadata and `AppendTableWithAlignments` without changing those public parsed-table identity decisions.
