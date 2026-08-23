# M64 — Public Table Model and Ownership

## Status

Complete and green.

## Objective

Promote a parser-independent public GFM `Table` identity without deriving that identity from a promoted body row. The model must remain meaningful when a table has no semantic body rows, when semantic body rows exist but none can be promoted as `TableRow`, and when header cells are empty and therefore have no synthetic public cell identities.

M64 is read-only at table level. It does not add delimiter-row mutation, alignment mutation, or column structural editing.

## Public contract

M64 appends `KindTable` after the existing public kind ordinals and adds the comparable immutable value:

```go
type Table struct { /* scalar snapshot-owned detail */ }

func (t Table) ID() NodeID
func (t Table) Range() Range
func (t Table) ColumnCount() int
func (t Table) BodyRowCount() int
```

`Table.Range()` owns the exact complete source block from the physical header-row start through the last semantic table-owned line. The existing terminator of the last owned line is included when present. The range therefore includes the header row, delimiter row, and every semantic body row, but never an unrelated following blank or block line.

`BodyRowCount()` is semantic rather than promotional. It can be non-zero while `Document.TableRowIDs` returns an empty promoted subset.

Document-level navigation keeps variable-length storage outside the comparable `Table` value:

```go
func (d *Document) Table(id NodeID) (Table, bool)
func (d *Document) TableRowIDs(tableID NodeID) ([]NodeID, bool)
func (d *Document) TableHeaderCellIDs(tableID NodeID) ([]NodeID, bool)
func (d *Document) TableAlignments(tableID NodeID) ([]TableAlignment, bool)
```

Returned slices are caller-owned. `TableRowIDs` contains only promoted body rows in source order. `TableHeaderCellIDs` contains only promoted non-empty header cells in source order. Empty or unsupported cells are never synthesized.

Ownership is also exposed from existing comparable values:

```go
func (r TableRow) TableID() (NodeID, bool)
func (c TableCell) TableID() (NodeID, bool)
```

A promoted table cell can have a valid `TableID` while `RowID` is unavailable because its semantic body row is not itself promotable.

The M63 `Document.TableRowAlignments` API remains source-compatible and behavior-compatible.

## Identity and source proof

Goldmark provides the semantic `Table`/header/body-row/cell hierarchy and alignment vector. Marksplice does not use a body row as table identity and does not trust `Table.Pos()` as the source header boundary.

Goldmark's table paragraph transformer can split a preceding paragraph while retaining the original paragraph position on the semantic table container. M64 therefore derives one parser-independent `TableAnchor` from the public `TableHeader.Pos()` value and uses that same anchor for the table, its rows, and its cells.

`internal/source.MapTable` independently proves:

- the exact physical header row from the header anchor;
- the immediately following physical delimiter row;
- delimiter-cell lexical alignment syntax and one alignment value per mapped delimiter column;
- the exact complete table end using the semantic last-body-row anchor when body rows exist;
- CRLF/LF and unterminated-EOF ownership without newline synthesis.

Promotion fails closed when header/delimiter column counts or lexical delimiter alignments disagree with Goldmark's semantic table metadata.

## Internal ownership model

M64 intentionally preserves the mature M35–M63 row-centric model and adds a second table-centric post-ID pass rather than rewriting row mutation state.

The table-centric pass:

1. indexes promoted tables transiently by parser-independent `TableAnchor`;
2. verifies semantic body-row count and last-row anchor against all observed body rows, including unpromoted rows;
3. assigns scalar `TableID` ownership to promoted rows and cells;
4. creates compact flat source-ordered promoted-row and promoted-header-cell adjacency for public table navigation;
5. stores only scalar start/count spans on table nodes and discards temporary maps/cursors after parsing.

There is no persistent table-anchor map. Parse-time complexity remains linear in observed table nodes/rows/cells. Table adjacency reads are O(k) only for the returned copied identities; alignment reads are O(columns).

## Compatibility

- `KindTable` is append-only after `KindTableRow`; existing public kind ordinals do not move.
- Internal `KindTable` is also appended so the kind byte used by existing snapshot `NodeID` derivation remains unchanged for pre-M64 node kinds.
- Existing row/cell source ranges and row mutation contracts remain unchanged.
- `TableRow` remains comparable because table ownership is one scalar `NodeID`; slices remain document-owned.
- No Goldmark type crosses the parser adapter boundary.
- No empty cell or delimiter-row public node is synthesized.

## Edge cases covered

- tables with zero body rows;
- tables with semantic body rows that are not promotable as `TableRow`;
- promoted body cells whose row identity is unavailable but table identity is available;
- all-empty table headers;
- mixed default/left/right/center alignments;
- CRLF complete-table ownership;
- tables created from the trailing lines of a Goldmark paragraph transformation, where the semantic table container position precedes the actual header;
- invalid table IDs and source-mapping failures.

## Devil's advocate review

### Risk: body-row-derived identity disappears

Using the first promoted body row as a table handle would make a valid table unaddressable after final-row removal and would never represent header+delimiter-only tables. M64 instead promotes the semantic table itself and proves its independent source block.

### Risk: requiring every body row to be promotable hides valid tables

A semantic row can contain a source shape that fails the stricter public `TableRow` mapping. Requiring complete row promotion would make table identity brittle and would couple table reads to row-edit capability. M64 separates semantic `BodyRowCount` from the promoted `TableRowIDs` subset.

### Risk: Goldmark container position is not always the header position

Goldmark can preserve the original paragraph position when it splits leading paragraph lines away from a later table header. M64 uses the public header node position as the canonical table source anchor and verifies the complete block independently in the source layer.

### Risk: table-level navigation regresses mature row mutation behavior

Replacing the M35–M63 row model would enlarge the regression surface. M64 keeps that model intact and layers table ownership/adjacency after it; existing row operations continue to use their established proof path.

## Verification

Focused source-mapping and public-table tests are green, including the split-leading-paragraph anchor case.

Final M64 verification on the uncommitted M63+M64 working tree passed:

- `gofmt` on every changed Go file;
- `go test ./... -count=1`;
- `go test -race ./... -count=1`;
- `go vet ./...`;
- `staticcheck ./...`;
- `golangci-lint run` with zero issues;
- `govulncheck ./...` with no vulnerabilities found;
- `gitleaks dir . --no-banner --redact` with no leaks found;
- `go build ./...`;
- coverage run: root 93.1%, Goldmark adapter 66.2%, source 79.2%, splice 66.9%, aggregate 74.4%;
- `git diff --check`;
- final Git status review confirmed the pre-existing uncommitted M63 work plus the intended M64 files only.

No commit or push was performed.
