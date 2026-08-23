# M85 — Blockquote Reference and Table Children

Status: complete.

## Objective

Complete the first reviewed multi-block blockquote composition family by admitting existing canonical reference-definition and GFM-table construction blocks through the M83 API, while keeping front matter and recursive blockquote children outside the contract.

## Contract

`DocumentBuilder.AppendBlockquoteBlocks` additionally accepts child builders containing:

- canonical reference definitions, with or without the already-supported conservative title form;
- canonical GFM tables, including explicit default/left/right/center column alignments.

The API remains depth 1–64 and continues to snapshot the child builder. No table, reference-definition, blockquote, or parser API is changed.

Front matter remains a document-leading envelope rather than a body block. Child blockquotes remain rejected so recursive depth composition can be reviewed independently with an explicit total-depth policy.

## Architecture

M85 reuses the existing reference-definition and table writers and their standalone construction validation unchanged. The M83 lexical proof still proves that quoted source reconstructs those exact canonical child bytes after removing the repeated blockquote prefixes.

The construction-only Goldmark comparator is extended with reviewed semantic facts:

- `ast.LinkReferenceDefinition`: exact label, destination, and title bytes;
- GFM `Table`: exact alignment vector and child hierarchy;
- `TableHeader` and `TableRow`: exact alignment vectors and child sequence;
- `TableCell`: exact semantic alignment.

Table containers participate in the existing iterative child-sequence comparison. Cells remain semantic leaves because the independent lexical proof already guarantees byte-identical canonical cell source, avoiding a second inline parser or duplicated table source mapper.

No Goldmark type crosses the internal adapter boundary and ordinary `Adapter.Parse` behavior is unchanged.

## TDD and edge cases

The initial focused public test failed at the supported-child gate. After implementation it writes one depth-2 blockquote containing a titled reference definition and a two-column aligned table with Unicode body content, and verifies the exact canonical quoted bytes.

Permanent Goldmark tests prove:

- successful reference-definition + aligned-table hierarchy inside a nested constructed blockquote;
- rejection when the reference destination changes;
- rejection when a table alignment changes.

The unsupported-child regression now intentionally contains only the remaining boundaries: recursive blockquote content and a child builder with front matter. Both continue to fail closed without mutating the destination builder.

## Devil's advocate review

1. **A table could keep the same source shape while semantic alignment changes.** Table, header, row, and cell alignment facts are compared in addition to exact lexical quoted-to-inner proof.
2. **Reference syntax could reparse to a different destination/title while looking structurally similar.** The construction-only comparator checks label, destination, and title bytes from Goldmark semantics.
3. **Descending through every table node could create another recursive proof path.** M85 reuses the iterative M84 sibling-sequence stack and only marks table containers as child-bearing.
4. **Admitting front matter would blur the document-envelope boundary.** The child builder still fails immediately when it owns front matter; M85 does not treat metadata as a quoted body block.

## Verification

Focused public and Goldmark tests, full `go test ./... -count=1`, production complexity, and both `unparam` modes are green on the M85 implementation. Statement coverage is 92.5% for the root package, 70.0% for `internal/parser/goldmark`, 79.2% for `internal/source`, 57.5% for `internal/splice`, and 71.3% aggregate; the parser interface package reports 0.0% because it has no executable test target.

The fully documented M85 tree passed the strict completion gate: five consecutive `go test ./... -count=1` runs, `go test -race ./... -count=1`, coverage, `go vet ./...`, `go build ./...`, public `go doc` checks for all blockquote construction entrypoints, hash-pinned published GFM 0.29 conformance, Staticcheck, golangci-lint with zero issues, production `gocyclo` at the <=15 threshold, production and test-inclusive `unparam`, `govulncheck` with no vulnerabilities found, Gitleaks with no leaks, strict UTF-8/no-BOM/LF/no-trailing-whitespace hygiene over 229 repository text files, `git diff --check`, and `git fsck --no-dangling`.

The verified state remained branch `main` at HEAD `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured remotes and the M63–M85 work intentionally uncommitted.

## Exit decision

M85 completes non-recursive multi-block blockquote composition for every currently reviewed canonical body-block family that has a safe standalone builder representation: paragraphs, ATX headings, thematic breaks, fenced code, lists/tasks, reference definitions, and GFM tables.

The next blockquote-construction boundary is recursive blockquote children. It requires an explicit total-depth rule so independently valid nested builders cannot combine past the 64-level construction limit. Existing-source multiline/nested/multi-block promotion remains a separate source-ownership and editing review.
