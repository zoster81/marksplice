# M84 — Blockquote List Children

Status: complete.

## Objective

Extend M83 multi-block blockquote construction to the complete reviewed list/task construction family without adding public API, duplicating list writers, or widening existing-source blockquote support.

## Contract

`DocumentBuilder.AppendBlockquoteBlocks` additionally accepts child builders containing:

- flat unordered lists;
- flat ordered lists;
- flat unordered task lists;
- flat ordered task lists;
- homogeneous nested unordered lists, including task items;
- homogeneous nested ordered lists, including task items.

The existing list APIs keep their original contracts. Marker selection, ordered numbering, task state, structural `Depth`, and marker-width-aware indentation are still derived by the ordinary list construction writer before the resulting source is quoted by the M83 container writer.

## Architecture

M84 changes no writer. The supported-child gate admits the existing list construction kinds, so standalone list validation and source expectations are reused unchanged.

The construction-only Goldmark comparator is extended to understand `ast.List`, `ast.ListItem`, and the `ast.TextBlock` nodes Goldmark uses for tight list-item text. List equality requires the same marker, ordered start value, tightness, item hierarchy, and source-equivalent text blocks.

Child-tree comparison uses an explicit stack of expected/actual sibling sequences rather than recursive descent. This keeps proof work bounded by the parsed tree size without adding call-stack growth for nested lists.

Ordinary parsing may promote list-item observations even when they are inside a blockquote. Therefore construction validation now excludes every ordinary observation whose complete source range lies inside a construction-only multi-block blockquote expectation. This does not change `Adapter.Parse`; it only prevents the generic construction matcher from competing with the authoritative container proof.

## TDD and edge cases

The initial focused test failed with `ErrInvalidConstruction` because list children were not yet admitted. The public test then constructs all reviewed list/task families in one depth-2 blockquote and verifies exact canonical source, including list markers, task markers, nested indentation, ordered numbering, and quoted blank separators.

A temporary diagnostic test was used only to inspect Goldmark's standalone and quoted list trees. It established that tight list-item content is represented by `ast.TextBlock`; the diagnostic file was then deleted and is not repository content.

Permanent Goldmark tests prove acceptance of nested list/task hierarchy and rejection when list kind or nesting structure changes.

## Devil's advocate review

1. **Deep list nesting could make semantic proof recursively consume the Go stack.** Child comparison is iterative with an explicit stack.
2. **Quoting could alter marker width, numbering, task state, or indentation.** The ordinary list builder proves the standalone source first; the quoted proof compares list/container hierarchy and the lexical proof preserves every inner byte exactly.
3. **Nested list observations could appear as unexpected generic construction nodes.** Observations wholly contained by the construction-only blockquote range are skipped by the generic matcher and remain covered by the blockquote-specific proof.
4. **Adding list cases could push the semantic dispatcher over the complexity gate.** List-specific comparison was extracted into a focused helper after `gocyclo` measured the dispatcher at 16, restoring the <=15 production gate.

## Verification

Focused public tests, focused Goldmark hierarchy tests, the complete `go test ./... -count=1` regression, production `gocyclo`, and production/test-inclusive `unparam` passed before M85 began. The fully documented M85 tree subsequently passed the complete strict repository gate, including five repeated suites, race, static/lint/complexity analysis, vulnerability and secret scans, pinned GFM conformance, text hygiene, and Git integrity checks, so the final verification also covers all M84 regressions.

## Exit decision

M84 is complete. Multi-block blockquotes can now compose every reviewed list/task construction family through the existing `AppendBlockquoteBlocks` API while list source generation remains owned by the original canonical writer.
