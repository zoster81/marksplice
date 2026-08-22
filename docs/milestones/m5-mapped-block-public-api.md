# Milestone M5 — Mapped Block Public API

Status: green — mapped block public API passed.

## Goal

Promote two additional M1-proven block families, non-empty GFM table cells and supported single-line fenced code blocks, without weakening the public capability boundary established after M4.

M5 must make editability a property already known from the immutable parsed snapshot. A public actionable node must not defer discovery of essential source-shape support until the caller attempts a mutation.

## Scope

M5 includes two vertical slices:

- mapped non-empty `TableCell` detail plus source-preserving cell replacement;
- mapped single-line `FencedCode` detail plus source-preserving content replacement.

Both slices require moving the existing M1 source mapping of the original node from mutation preparation into document parsing. Candidate replacement validation remains conservative and reparses the candidate exactly as M1 proved.

## Internal capability model

The internal node model gains a snapshot-local `Editable` capability flag. For source families that require an explicit lossless mapper, the node also retains the validated mapping produced during `Parse`.

For M5:

- a table cell is editable only when `source.MapTableCell` succeeds for the semantic observation;
- a fenced-code node is editable only when the source mapper succeeds; the mapper is now named `source.MapFencedCode`, while M5 itself proved only the original single-line subset;
- unsupported semantic fenced-code shapes remain present internally and do not make `Parse` fail, but remain non-editable;
- the stored original mapping is reused by mutation preparation instead of rescanning the immutable source snapshot.

Existing public families are also marked editable only for their already-reviewed public shape: top-level paragraphs, top-level mapped headings, mapped list items, and mapped task markers.

`Editable` is an internal capability, not public semantic data.

## Public API contract

M5 promotes:

- `KindTableCell`;
- `KindFencedCode`;
- immutable `TableCell` detail;
- immutable `FencedCode` detail;
- `Document.TableCell(NodeID)`;
- `Document.FencedCode(NodeID)`;
- `Document.PrepareReplaceTableCell(...)`;
- `Document.PrepareReplaceFencedCode(...)`.

Generic `Node` remains ID + kind only.

### TableCell

`TableCell.Range()` is the exact non-empty cell content span replaced by `PrepareReplaceTableCell`; pipes, alignment syntax, padding, neighboring cells, and line endings are outside the range.

`TableCell.Header()` reports header/body position. `TableCell.Column()` is zero-based within the mapped GFM row.

Empty or otherwise unsupported cells are not promoted as actionable M5 nodes.

### FencedCode

`FencedCode.Range()` is the exact single-line content span replaced by `PrepareReplaceFencedCode`. Opening/closing fences, indentation, info-string spelling/spacing, and line ending are outside the range.

M5 deliberately does not expose fence character, fence length, indentation, or info-string ranges as public fields; those remain lexical preservation data used internally by candidate validation.

Unsupported, multiline, empty, unclosed, or container-prefixed fenced-code shapes remain outside this public slice unless the existing M1 mapper proves them supported.

## Error and preservation contract

Publicly obtainable M5 nodes have already passed the editable-source gate. The existing public error categories remain sufficient:

- missing ID: `ErrNodeNotFound`;
- wrong/non-actionable target: `ErrInvalidTargetKind`;
- unsafe replacement: `ErrInvalidReplacement`;
- stale application: `ErrSourceConflict`.

Source-preservation tests must prove bytes outside each typed `Range()` remain identical.

## Architecture and performance

Persisting the original lossless mapping avoids one repeated scan of the immutable original source during each table/fence mutation preparation. Parse-time table handling is shared per physical row. The Goldmark adapter assigns cell columns incrementally during its single AST walk and supplies a parser-independent row anchor; `source.MapTableRow` derives all cell mappings with one row scan, and `splice.Parse` caches that result for sibling cells. This removes both the former per-cell previous-sibling scan and the later full-row source rescan, keeping the added table mapping work linear in table source size. Candidate reparsing remains O(n) per prepared mutation as the M1 safety oracle; batch amortization remains a later concern.

Node identity remains snapshot-scoped and opaque. M5 does not intentionally redefine existing node-ID inputs merely to store richer mapping metadata.

## Devil's advocate review

### Risk: parsing becomes stricter than semantic GFM parsing

If failure of the source-preserving mapper aborts `Parse`, previously accepted but unsupported fenced-code source would regress.

Mitigation: expected unsupported-shape mapper failures leave the semantic node internal with `Editable=false`; only unexpected mapper errors may abort parsing.

### Risk: `Editable` becomes a generic public promise

Different syntax families have different operations and safety rules; exposing one boolean publicly would be too vague.

Mitigation: `Editable` stays internal and is used only to filter reviewed typed public capabilities.

### Risk: storing mapping duplicates stale data

Persisted offsets would be dangerous if the document were mutable in place.

Mitigation: `Document` is an immutable source snapshot. Prepared changes produce new bytes but never mutate the parsed document or its stored mapping.

### Risk: mapping storage bloats the generic public node model

Fence/table lexical details are implementation data, not common semantics.

Mitigation: mappings remain on internal nodes; public `Node` stays unchanged and typed details expose only reviewed fields.

## Exit decision

M5 is green. Internal tests prove that supported table cells and fenced code retain their validated original source mappings and are marked editable at parse time, while an unsupported unclosed fenced-code semantic node remains parseable, retains no mapping, and stays non-editable. Public tests prove typed table/fence details, exact operation ranges, unsupported-shape filtering, public error categories, and byte-identical preservation outside replacements.

The complete Go regression and race suites pass; vet, Staticcheck, golangci-lint, govulncheck, and Gitleaks report no findings; generated package documentation exposes only Marksplice-owned or standard-library public types.

M5 also removes repeated original-source mapping from table/fence mutation preparation by reusing the immutable snapshot mapping. A consolidation review additionally replaced per-cell table-row rescans with a shared row mapping cache keyed by a parser-independent source anchor and replaced previous-sibling column counting with incremental column assignment during the AST walk; focused tests cover shared row anchors, ordered columns including skipped empty cells, and sibling reuse of one cached row mapping. Candidate reparsing and source mapping remain in place as the conservative fail-closed validation oracle. No Goldmark parser configuration, dependency, GFM profile, or NodeID algorithm was changed.
