# Markdown Conformance Policy

Status: source of truth for Marksplice's Markdown syntax profile, normative hierarchy, conformance gates, and parser-update policy.

## Normative profile

Marksplice exposes one Markdown syntax profile. Its normative base grammar is **CommonMark 0.31.2** at `https://spec.commonmark.org/0.31.2/`. The published GFM specification at `https://github.github.com/gfm/` is layered on top for explicit GitHub Flavored Markdown extensions or corrections that are not already superseded by the newer CommonMark base.

The published GFM page is based on an older CommonMark line. Therefore inherited GFM core examples do not override CommonMark 0.31.2. When the two published documents disagree about base Markdown syntax, CommonMark 0.31.2 is authoritative. GFM remains authoritative for its explicit extension sections such as tables, task-list items, strikethrough, and extended autolinks. Marksplice does not expose separate CommonMark and GFM modes.

A future request for another Markdown dialect or a separately observable compatibility mode requires an explicit architecture decision. It must not be introduced implicitly through parser implementation changes or by copying one implementation's quirks.

GitHub's broader authoring surface includes features such as footnotes, alerts, mathematical expressions, and fenced technical/diagram blocks that are not all defined by either normative specification. Their product evaluation belongs to [`extension-strategy.md`](extension-strategy.md). Broadly useful capabilities may enter Marksplice core only through an explicit reviewed Marksplice contract without creating first-party extension modes. Dialect-specific syntax stays outside core and may be implemented by independent packages through the M110 opt-in SPI.

## Source hierarchy

When sources disagree, use this order:

1. the hash-pinned official CommonMark 0.31.2 specification for base Markdown syntax;
2. the hash-pinned published GFM specification for explicit GFM extension/correction sections not superseded by CommonMark 0.31.2;
3. explicit reviewed Marksplice contracts for core capabilities outside those specifications, provided they do not silently redefine the normative base grammar;
4. Marksplice conformance/focused tests that cite and implement the applicable contract above;
5. official reference/current implementations (`commonmark` reference implementation, `cmark-gfm`) and GitHub-maintained authoring guidance as compatibility, interpretation, and security evidence;
6. other documented Markdown specifications/implementations when the normative sources leave a case unspecified or materially ambiguous;
7. retired parser-implementation records as historical evidence only.

Lower-ranked sources do not override higher-ranked sources. A parser-library or historical implementation mismatch is evidence to investigate, never by itself a reason to change Marksplice semantics. The production parser is Marksplice-owned Native code, so a Native regression must be classified against the applicable specification or reviewed Marksplice contract rather than against a retired implementation.

## Pinned specification snapshots and parser-neutral contracts

Conformance tests read externally provisioned snapshots of the official CommonMark and GFM pages. The snapshots are intentionally not vendored into this Apache-2.0 repository because upstream specification material is separately licensed validation input.

`internal/testutil/commonmarkspec/corpus.go` owns the approved CommonMark 0.31.2 snapshot SHA-256 and extracts its 652 numbered examples. `internal/testutil/gfmspec/corpus.go` owns the approved published GFM snapshot SHA-256 and extension-section classification. Do not duplicate either hash in documentation or configuration. Both loaders fail closed on changed bytes.

M115 retains parser-neutral expected observations in tracked Marksplice-owned fixtures under `internal/parser/native/testdata/`. These files do **not** contain the upstream specification corpus. Each entry binds an official example number to the SHA-256 of its externally loaded Markdown input plus the expected `parser.DocumentObservations`. The CommonMark fixture contains all **652** examples. The GFM fixture contains the **676 parser-applicable** examples from the 677-example published page; the single `tagfilter` example remains rendering-specific and is correctly excluded from the parser-neutral fixture even though M120 now covers it in the separate renderer gate.

The fixtures were frozen only after the M115 dual-proof transition gate demonstrated, in the same tree, that the Native observations matched the previously validated M114 parser-neutral contract while the pre-removal specification/reference gates were still green. This transition evidence prevents removal of the old implementation from silently weakening the accepted parser contract. It does not elevate the retired implementation above the specifications.

## Current conformance procedure

Provision the approved external snapshots and set:

- `MARKSPLICE_COMMONMARK_SPEC_HTML` to the CommonMark 0.31.2 HTML snapshot;
- `MARKSPLICE_GFM_SPEC_HTML` to the published GFM HTML snapshot.

Then run the exact anchored gates:

```text
go test ./internal/testutil/commonmarkspec -run '^TestPublishedCommonMark0312Corpus$' -count=1
go test ./internal/parser/native -run '^TestM115NativeMatchesPublishedCommonMark0312Contract$' -count=1
go test ./internal/parser/native -run '^TestM115NativeMatchesPublishedGFM029Contract$' -count=1
go test ./internal/parser/native -run '^TestM119PublishedCommonMarkSemanticContract$' -count=1
go test ./internal/parser/native -run '^TestM119PublishedGFMSemanticContract$' -count=1
go test ./internal/publictest -run '^TestM120PublishedCommonMarkHTMLFullProfileContract$' -count=1
go test ./internal/publictest -run '^TestM120PublishedGFMHTMLFullProfileContract$' -count=1
```

Use the anchored exact test names shown above. `go test -run` can exit successfully after selecting zero tests, so shortened or guessed filters are not acceptable conformance evidence.

The CommonMark parser gate verifies the approved snapshot identity and all 652 parser-neutral contracts. The GFM parser gate verifies the approved snapshot identity, the published corpus shape, the example/extension identities, and all 676 parser-applicable parser-neutral contracts. M119 adds separate semantic-walk gates for manually reviewed selected examples from those same hash-pinned snapshots. M120 adds independent full renderer gates using the published expected HTML as the oracle rather than current Native or semantic output.

The M120 CommonMark renderer gate accounts for all 652 examples with the GFM tag filter disabled. Six examples are explicit Marksplice-profile divergences: one leading empty-YAML-front-matter precedence case, three always-enabled reviewed extended-autolink cases, and two cases where Marksplice deliberately applies the reviewed newer GFM HTML-comment grammar. Every other CommonMark example must render byte-identically to the approved expected HTML.

The M120 GFM renderer gate accounts for all 677 published examples and enables the tag filter for the rendering-only `tagfilter` extension example. Four inherited core examples are explicit Marksplice-profile divergences: one leading empty-YAML-front-matter case and three always-enabled reviewed extended-autolink cases. Every other published-GFM example, including tables, tasks, strikethrough, autolinks, and `tagfilter`, must render byte-identically to the approved expected HTML. These profile exceptions are named in the permanent test and are not a wildcard mismatch allowance.

Parser and renderer conformance remain separate responsibilities. `tagfilter` is not parser syntax and therefore still does not enter the 676-case M115 parser-neutral fixture; its renderer behavior is nevertheless mandatory and covered by M120.

## Updating normative snapshots or observation fixtures

Do not update an approved SHA or regenerate observation fixtures merely because an official page changed or the current Native parser produces different output.

A CommonMark or GFM snapshot/contract update requires all of the following:

1. obtain the changed official published page from its canonical source;
2. review the specification diff and identify semantic changes, not only example-count changes;
3. classify whether each change belongs to the CommonMark base or an explicit GFM extension/correction;
4. add or update focused regression tests that cite the changed normative rule;
5. derive the changed parser-neutral expectation from the reviewed specification/Marksplice contract, using reference/current implementations only as secondary evidence where useful;
6. update Native behavior only where necessary to match the reviewed contract;
7. update the affected fixture entries and snapshot hash in the same reviewed change, preserving example identity and Markdown-hash checks;
8. run the complete conformance and repository regression gates;
9. update this document and affected capability/milestone documentation in the same change.

Mechanically serializing the current Native output and accepting it as a new parser fixture or semantic-walk expectation is not sufficient review: that would create a tautological test. If an implementation conflicts with the applicable published specification, record the discrepancy and follow the specification. If the specifications themselves do not determine the case, document the Marksplice decision and the secondary evidence used.

## Parser implementation boundary

`internal/parser/native` is the production semantic parser implementation and satisfies the parser-independent `internal/parser.Backend` contract consumed by `internal/splice`. M115 removed the former Goldmark adapter, differential harness, compatibility implementation, and `github.com/yuin/goldmark` module dependency after the dual-proof cutover gate.

Rules:

- no parser-internal type may cross the Marksplice public API boundary;
- third-party syntax packages do not define Marksplice core capabilities merely because they exist;
- parser differences are conformance/test problems to classify, not reasons to expose multiple dialect modes;
- ordinary existing-document edits must never depend on serializing a parser AST back to Markdown;
- Marksplice owns exact source mapping, lexical trivia, source fingerprints, deterministic identities, and minimal byte patches;
- construction proof uses the same Native parser-independent contracts as parsed documents and remains independently source-proven.

Historical M111–M114 records document the staged parser-substitution work and the retired differential oracle. They remain engineering history, not active architecture. M115 is the completed cutover boundary: production parsing and construction proof are Native-only and the dependency graph contains no Goldmark package.

## Rendering boundary

Semantic Markdown parsing and HTML rendering are separate responsibilities. M118–M119 provide the internal on-demand semantic walk; M120 exposes deterministic HTML-fragment rendering through `Document.RenderHTML` and `Document.HTML` without adding a second Markdown parser or retained renderer AST. M121 wraps that exact body renderer in deterministic standalone document markup; it does not alter Markdown parsing or the expected-HTML conformance oracle.

Renderer conformance uses the specification's expected HTML as normative evidence where applicable rather than treating Native or semantic-walk output as its own oracle. The permanent full-profile gates above account for every official CommonMark 0.31.2 and published-GFM example and encode the small deliberate Marksplice-profile divergence sets explicitly. GFM `tagfilter` is mandatory renderer evidence even though it remains outside parser-neutral conformance.

Rendering policy is also a product boundary: raw HTML preservation/escaping, dangerous-URL suppression/allowance, and GFM tag filtering are explicit public options. Raw-HTML preservation is not a sanitizer. Rendering performs no URL fetching, asset loading, command execution, syntax highlighting, template execution, or mathematical-engine execution.

## Compatibility monitoring

Current `cmark-gfm` releases and GitHub-maintained GFM authoring guidance should be monitored when parser behavior changes or a new edge case appears. They are useful for discovering security fixes, implementation changes, and missing tests.

They remain advisory unless the normative GFM source itself changes through the reviewed snapshot-update process above.
