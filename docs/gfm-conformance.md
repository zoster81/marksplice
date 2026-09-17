# Markdown Conformance Policy

Status: source of truth for Marksplice's Markdown syntax profile, normative hierarchy, conformance gates, and specification-update procedure.

## Normative profile

Marksplice exposes one Markdown syntax profile.

- **CommonMark 0.31.2** is the normative base grammar.
- The published **GitHub Flavored Markdown specification** adds explicit GFM extensions and corrections.

The published GFM document is based on an older CommonMark line. Inherited GFM core examples therefore do not override CommonMark 0.31.2. When the two sources disagree about base Markdown syntax, CommonMark 0.31.2 wins. GFM remains authoritative for explicit extensions such as tables, task-list items, strikethrough, and extended autolinks.

Marksplice does not expose separate CommonMark/GFM modes. Another dialect or compatibility mode requires an explicit architecture decision; it must not appear implicitly through implementation quirks.

Reviewed Marksplice capabilities such as alerts, footnotes, mathematical source forms, and front matter may exist outside the normative grammar. They must have explicit contracts and must not silently redefine base CommonMark/GFM behavior. See [`extension-strategy.md`](extension-strategy.md).

## Source hierarchy

When evidence disagrees, use this order:

1. the approved official CommonMark 0.31.2 specification snapshot for base Markdown;
2. the approved published GFM specification snapshot for explicit GFM extensions/corrections not superseded by CommonMark 0.31.2;
3. explicit reviewed Marksplice contracts for capabilities outside those specifications;
4. focused and conformance tests implementing the applicable contract;
5. official reference/current implementations such as the CommonMark reference implementation and `cmark-gfm`, plus GitHub-maintained guidance, as secondary compatibility/security evidence;
6. other documented implementations/specifications only when higher-ranked sources leave a case materially unspecified;
7. retired parser implementation records as historical evidence only.

A parser implementation is never its own normative source. A mismatch is something to classify against the applicable specification or reviewed Marksplice contract.

## Approved external snapshots

Conformance tests read separately provisioned snapshots of the official CommonMark and GFM pages. The snapshots are not vendored because upstream specification material is separately licensed validation input.

`internal/testutil/commonmarkspec` owns the approved CommonMark snapshot hash and extraction of its **652** numbered examples. `internal/testutil/gfmspec` owns the approved published-GFM snapshot hash, its **677** examples, and extension-section classification. The loaders fail closed if snapshot bytes change.

Set these environment variables to the approved HTML files:

```text
MARKSPLICE_COMMONMARK_SPEC_HTML
MARKSPLICE_GFM_SPEC_HTML
```

Do not copy snapshot hashes into additional documentation or configuration; the loaders are the single hash authority.

## Parser-neutral fixtures

Tracked fixtures under `internal/parser/native/testdata/` contain Marksplice-owned expected parser observations, not upstream corpus text.

Each fixture entry binds official example identity to the SHA-256 of the externally loaded Markdown plus expected parser-independent observations. The CommonMark fixture covers all **652** examples. The GFM fixture covers the **676 parser-applicable** examples; the published `tagfilter` example is rendering-only and is intentionally absent from parser-neutral fixtures.

Fixture expectations must come from reviewed specifications/contracts, not by serializing whatever the current parser happens to produce.

## Required conformance gates

With approved snapshots provisioned, the exact anchored gates are:

```text
go test ./internal/testutil/commonmarkspec -run '^TestPublishedCommonMark0312Corpus$' -count=1
go test ./internal/parser/native -run '^TestNativeMatchesPublishedCommonMark0312Contract$' -count=1
go test ./internal/parser/native -run '^TestNativeMatchesPublishedGFMContract$' -count=1
go test ./internal/parser/native -run '^TestPublishedCommonMarkSemanticContract$' -count=1
go test ./internal/parser/native -run '^TestPublishedGFMSemanticContract$' -count=1
go test ./internal/publictest -run '^TestPublishedCommonMarkHTMLFullProfileContract$' -count=1
go test ./internal/publictest -run '^TestPublishedGFMHTMLFullProfileContract$' -count=1
go test ./internal/rendermarkdown -run '^TestPublishedCommonMarkCanonicalSemanticRoundTrip$' -count=1
go test ./internal/rendermarkdown -run '^TestPublishedGFMCanonicalSemanticRoundTrip$' -count=1
```

Use exact anchored names. `go test -run` can succeed after selecting zero tests, so guessed filters are not conformance evidence.

The parser-neutral gates verify snapshot identity and all applicable expected observations. The semantic-walk gates verify reviewed rendering semantics without using current output as their expected-value generator.

## HTML rendering conformance

Renderer conformance is separate from parser conformance and uses the specifications' expected HTML where applicable.

The CommonMark full-profile gate accounts for all **652** examples with GFM tag filtering disabled. A small, explicitly enumerated set of deliberate Marksplice-profile divergences is permitted only where a reviewed Marksplice contract intentionally differs, such as front-matter precedence or reviewed GFM behavior layered on the newer CommonMark base. Every other example must match expected HTML byte-for-byte.

The GFM full-profile gate accounts for all **677** published examples and includes rendering-only `tagfilter`. Any deliberate profile divergence is explicitly named in the permanent test; there is no wildcard mismatch allowance.

`tagfilter` is not parser syntax, so it does not enter parser-neutral fixtures even though its HTML behavior is mandatory.

## Canonical Markdown conformance

Canonical Markdown is normalization, not source reproduction. Every approved CommonMark and GFM example is:

1. parsed;
2. rendered canonically;
3. reparsed;
4. compared for reviewed semantic facts;
5. rendered canonically again;
6. required to produce byte-identical output on the second render.

This proves semantic round-trip plus byte idempotence. It does not require canonical output to retain the original source spelling.

Canonical conformance uses no per-example “accept current output” allowlist.

## Updating a specification snapshot or fixture

Do not update an approved hash or regenerate fixtures merely because an upstream page changed or the current parser emits different observations.

A specification update requires:

1. obtain the changed official page from its canonical source;
2. review the specification diff and identify semantic changes;
3. classify each change as base CommonMark or explicit GFM behavior;
4. add or update focused tests for changed normative rules;
5. derive parser-neutral expectations from the reviewed specification/Marksplice contract, using reference implementations only as secondary evidence;
6. update Native behavior only where required by the reviewed contract;
7. update affected fixture entries and the owning snapshot hash together, preserving example identity and Markdown-hash checks;
8. run all conformance gates plus the complete repository regression suite;
9. update affected current documentation and, when useful, the historical engineering record.

Mechanically snapshotting current Native output is not sufficient because it would make the test tautological.

## Parser implementation boundary

`internal/parser/native` is the production semantic parser behind the parser-independent `internal/parser.Backend` contract consumed by the rest of Marksplice.

Rules:

- parser-internal types do not cross the public API;
- parser differences are classified as specification/contract questions, not reasons to expose multiple dialect modes;
- ordinary existing-document edits never serialize a parser AST back to Markdown;
- exact source mapping, lexical trivia, source fingerprints, structural identities, and minimal patches remain Marksplice-owned source responsibilities;
- construction proof and source-preserving mutation use parser-independent contracts plus independent source proof;
- the production dependency graph contains no third-party Markdown parser.

Historical parser-transition evidence remains available in explicitly historical records but is not active architecture or normative authority.

## Rendering boundary

The on-demand semantic walk is the shared semantic input for HTML and canonical Markdown rendering. Renderers do not reparse Markdown delimiters and do not retain a second rendering AST in each `Document`.

HTML source maps are optional correlation metadata. With identical options, mapping-on output must be byte-identical to ordinary HTML output. Source/output ranges belong only to the exact snapshot/result that produced them and are not durable identities.

Canonical Markdown is accepted by semantic reparse equivalence plus idempotence and must never replace ordinary source-preserving mutation.

Rendering policy is explicit: raw HTML preservation/escaping, dangerous-URL handling, and GFM tag filtering are public options. Preserved raw HTML is not sanitized content. Rendering performs no URL fetching, asset loading, command execution, template execution, syntax highlighting, or mathematical-engine execution.

## Compatibility monitoring

Current `cmark-gfm` behavior and GitHub-maintained authoring guidance are useful when investigating edge cases, security behavior, or missing tests. They remain advisory unless the normative specification or a reviewed Marksplice contract changes through the process above.
