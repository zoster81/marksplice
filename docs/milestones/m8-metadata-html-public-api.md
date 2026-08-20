# Milestone M8 — Metadata and HTML Public API

Status: green — metadata and HTML public API passed.

## Goal

Promote the M1-proven leading front-matter scalar and simple raw-HTML edit families into the reviewed public API without claiming full YAML/TOML or HTML semantics and without exposing parser/source-layer implementation details.

M8 continues the parse-time editable-capability boundary established by M5 through M7: callers see only nodes whose exact source-preserving mutation boundary is already known in the immutable parsed snapshot. Unsupported or opaque source remains parseable but is not publicly actionable.

## Scope

M8 includes four source-preserving slices:

- YAML simple scalar front-matter fields;
- TOML simple scalar front-matter fields;
- single-line valid HTML comment payloads;
- simple quoted `id` or `name` attributes on Goldmark-recognized `<a>` opening tags.

YAML and TOML share one public `FrontMatterField` type and one public `KindFrontMatterField`. The typed detail exposes `ID()`, the exact value `Range()`, `Key()`, and a Marksplice-owned `FrontMatterFormat` enum. The public surface does not expose scalar quote style, key ranges, delimiter ranges, or the internal YAML/TOML node split.

HTML comments and anchors remain distinct public types because their caller-facing semantics differ. `HTMLComment` exposes only `ID()` and its exact payload `Range()`. `HTMLAnchor` additionally exposes a semantic `HTMLAnchorAttribute` (`id` or `name`); source casing, quote character, spacing, tag spelling, and unrelated attributes remain private source data.

Full YAML/TOML parsing, nested metadata mutation, arrays/tables, multiline scalars/strings, duplicate target keys, arbitrary HTML attribute editing, opaque HTML editing, and an HTML DOM are outside M8.

## Parse-time capability model

Front matter remains a Marksplice-owned document-envelope layer rather than another Markdown dialect. `source.MapLeadingFrontMatter` recognizes only the conservative M1 shape: a closed byte-zero `---` YAML or `+++` TOML envelope with at least one unique simple scalar field whose lexical value boundary is provable. Ambiguous, duplicate-only, complex-only, non-leading, or unclosed shapes remain ordinary GFM source rather than being guessed as metadata.

During `Parse`, each safely mapped front-matter field is marked editable and retains the exact operation facts already represented by the internal node: field/value ranges, key, format, and scalar style. The immutable `Document` keeps only the envelope facts needed by candidate validation—format plus opening and closing ranges—rather than retaining a second copy of every field mapping.

For Goldmark `RawHTML` observations, parse-time source mapping attempts `source.MapHTMLComment` and then `source.MapSimpleHTMLAnchor`. Mapper success marks the node editable and stores the exact range plus the lexical facts already required for validation. Other raw HTML remains `KindHTMLOpaque` internally with `Editable=false` and is filtered from the public surface.

This design removes the M1 lazy rescans of the immutable original source from `PrepareReplaceFrontMatterValue`, `PrepareReplaceHTMLComment`, and `PrepareReplaceHTMLAnchor`. Candidate snapshots are still reparsed/remapped after replacement as the conservative fail-closed safety oracle.

## Public contracts

### FrontMatterField

`FrontMatterField.Range()` is the exact scalar value content replaced by `PrepareReplaceFrontMatterValue`. Envelope delimiters, key spelling, YAML `:` or TOML `=`, separator spacing, quote wrappers, inline comments, trailing spaces, unrelated fields, line endings, and the Markdown body remain outside the patch.

`Key()` identifies the unique simple scalar field selected by the conservative mapper. `Format()` reports YAML or TOML without exposing the internal source package type.

This is intentionally not a promise that Marksplice parses or edits arbitrary YAML/TOML documents.

### HTMLComment

`HTMLComment.Range()` is the exact single-line comment payload replaced by `PrepareReplaceHTMLComment`. `<!--` / `-->` and preserved horizontal padding immediately inside those delimiters remain outside the patch.

A candidate that introduces an invalid comment shape, including unsafe double-hyphen content, is rejected.

### HTMLAnchor

`HTMLAnchor.Range()` is the exact quoted `id`/`name` attribute value replaced by `PrepareReplaceHTMLAnchor`. `Attribute()` reports only the semantic target (`id` or `name`). Existing tag/attribute case, attribute ordering, whitespace, quote character, other attributes, and surrounding source remain untouched.

The candidate must re-establish the same simple `<a>` raw-HTML shape, attribute semantics, quote style, and exact shifted boundaries.

## Error and preservation contract

M8 adds no new public error category:

- missing ID: `ErrNodeNotFound`;
- wrong or non-editable target: `ErrInvalidTargetKind`;
- replacement that cannot re-establish the supported shape: `ErrInvalidReplacement`;
- stale application: `ErrSourceConflict`.

Public tests verify byte identity outside each typed detail's exact `Range()` for YAML, TOML, HTML comment, and HTML anchor replacements, including CRLF and preserved lexical wrappers.

## Architecture and complexity

M8 introduces no parser extension, YAML/TOML dependency, HTML parser dependency, network/filesystem behavior, or new patch engine.

The internal consolidation deliberately avoids storing redundant source mapping structs when the exact required facts are already fields of the immutable node. Front matter stores only one compact envelope record on the document; field identity/range/style stays on each node. HTML comment/anchor preparation likewise reuses node ranges and anchor lexical facts directly. This removes three original-source rescans and the now-redundant `frontMatterFieldForTarget` lookup.

Public YAML/TOML kinds are collapsed into one `KindFrontMatterField` through a small multi-kind promotion helper. Existing single-kind callers continue through the original helper, so the public boundary gains no duplicated target-validation logic.

Candidate reparsing remains O(n) per prepared mutation, matching the established M1 safety model. Parse-time mapping remains linear or near-linear in source size under the existing bounded scanners.

## Devil's advocate review

### Risk: front matter promotion is mistaken for full YAML/TOML support

A public metadata type could imply arbitrary nested YAML/TOML semantics.

Mitigation: the type is explicitly `FrontMatterField`, only unique M1-proven simple scalar fields are promoted, and unsupported envelopes/shapes remain ordinary GFM or non-actionable source.

### Risk: HTML promotion becomes an HTML parser contract

Recognizing a couple of raw-HTML edit shapes could accidentally imply DOM-level support.

Mitigation: Marksplice continues to depend on Goldmark's GFM `RawHTML` recognition, then applies only its own narrow exact-source mappers. All other raw/block HTML is preserved opaquely and is not publicly actionable.

### Risk: exposing lexical trivia freezes implementation details

Quote characters, attribute casing, YAML/TOML scalar style, or delimiter ranges could become unnecessary compatibility commitments.

Mitigation: public details expose only semantic format/key/attribute facts needed by callers plus the operation-oriented replacement range. Lexical preservation facts stay internal.

### Risk: parse-time capability state duplicates mappings and grows memory

Persisting full mapper structs both at document and node level would duplicate field data.

Mitigation: the consolidation stores only the front-matter envelope facts once and reuses facts already present on nodes; comment/anchor nodes likewise do not retain redundant mapper structs.

## Evidence and exit decision

M8 began with a focused public test that failed to compile because the metadata/HTML kinds, typed details, enums, and named operations did not yet exist. After implementation, focused public tests prove YAML/TOML typed field semantics, HTML comment/anchor semantics, exact replacement ranges, byte preservation, unsupported-shape filtering, deterministic zero values, and stable public error categories.

Focused internal evidence proves that safely mapped front-matter fields and simple comment/anchor observations are marked editable during `Parse`, while opaque HTML remains non-editable. Consolidation review confirms that the three M1 original-source rescans and the redundant front-matter target lookup are gone.

M8 is green. The completed repository verification stack passes: `gofmt`, focused tests, `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, generated package documentation, `staticcheck ./...`, `golangci-lint run` with zero issues, `govulncheck ./...` with no vulnerabilities, `gitleaks detect` with no leaks, and the approved published-GFM conformance gate. Final whitespace/status checks are recorded with the working tree review.
