# Markdown Conformance Policy

Status: source of truth for Marksplice's Markdown syntax profile, normative hierarchy, conformance gates, and parser-update policy.

## Normative profile

Marksplice exposes one Markdown syntax profile. Its normative base grammar is **CommonMark 0.31.2** at `https://spec.commonmark.org/0.31.2/`. The published GFM specification at `https://github.github.com/gfm/` is layered on top for explicit GitHub Flavored Markdown extensions or corrections that are not already superseded by the newer CommonMark base.

The published GFM page is based on an older CommonMark line. Therefore inherited GFM core examples do not override CommonMark 0.31.2. When the two published documents disagree about base Markdown syntax, CommonMark 0.31.2 is authoritative. GFM remains authoritative for its explicit extension sections such as tables, task-list items, strikethrough, and extended autolinks. Marksplice does not expose separate CommonMark and GFM modes.

A future request for another Markdown dialect or a separately observable compatibility mode requires an explicit architecture decision. It must not be introduced implicitly through parser configuration or by copying one implementation's quirks.

GitHub's broader authoring surface includes features such as footnotes, alerts, mathematical expressions, and fenced technical/diagram blocks that are not all defined by either normative specification. Their product evaluation belongs to [`extension-strategy.md`](extension-strategy.md). Broadly useful capabilities may enter Marksplice core only through an explicit reviewed Marksplice contract without creating first-party extension modes. Dialect-specific syntax stays outside core and may be implemented by independent packages through the M110 opt-in SPI.

## Source hierarchy

When sources disagree, use this order:

1. the hash-pinned official CommonMark 0.31.2 specification for base Markdown syntax;
2. the hash-pinned published GFM specification for explicit GFM extension/correction sections not superseded by CommonMark 0.31.2;
3. explicit reviewed Marksplice contracts for core capabilities outside those specifications, provided they do not silently redefine the normative base grammar;
4. Marksplice conformance/focused tests that cite and implement the applicable contract above;
5. official reference/current implementations (`commonmark` reference implementation, `cmark-gfm`) and GitHub-maintained authoring guidance as compatibility, interpretation, and security evidence;
6. other documented Markdown specifications/implementations when the normative sources leave a case unspecified or materially ambiguous;
7. Goldmark behavior as temporary implementation evidence only.

Lower-ranked sources do not override higher-ranked sources. A parser-library mismatch is evidence to investigate, never by itself a reason to change Marksplice semantics. In particular, a Native-versus-Goldmark differential failure must first be classified against the applicable specification or Marksplice-owned contract before either backend is changed.

## Pinned specification snapshots

Conformance tests read externally provisioned snapshots of the official CommonMark and GFM pages. The snapshots are intentionally not vendored into this Apache-2.0 repository because upstream specification material is separately licensed validation input.

`internal/testutil/commonmarkspec/corpus.go` owns the approved CommonMark 0.31.2 snapshot SHA-256 and extracts its 652 numbered examples. `internal/testutil/gfmspec/corpus.go` continues to own the approved published GFM snapshot SHA-256 and extension-section classification. Do not duplicate either hash in documentation or configuration. Both loaders fail closed on changed bytes.

Set `MARKSPLICE_COMMONMARK_SPEC_HTML` to the approved CommonMark snapshot and `MARKSPLICE_GFM_SPEC_HTML` to the approved GFM snapshot. M114's parser-side normative chain is:

```text
go test ./internal/testutil/commonmarkspec -run '^TestPublishedCommonMark0312Corpus$' -count=1
go test ./internal/parser/goldmark -run '^TestCommonMark0312PublishedSpecificationAudit$' -count=1
go test ./internal/parser/differential -run '^TestNativeBackendMatchesPublishedCommonMark0312DifferentialCorpus$' -count=1
go test ./internal/parser/goldmark -run '^TestGFM029PublishedExtensionConformance$' -count=1
```

Marksplice does not expose an HTML renderer. The CommonMark acceptance chain therefore first proves the temporary reference renderer against all 652 official expected HTML examples, then requires the Native backend to match the complete parser-neutral `DocumentObservations` contract on those same 652 inputs. Goldmark remains evidence in the first step, not the normative source: any differential mismatch must still be classified against the specification hierarchy before implementation changes.

The inherited published-GFM corpus remains compatibility/regression evidence and includes the normative extension cases:

```text
go test ./internal/parser/differential -run '^TestNativeBackendMatchesPublishedGFMDifferentialCorpus$' -count=1
go test ./internal/parser/differential -run '^TestNativeBlockParserMatchesPublishedGFMBlockProjection$' -count=1
go test ./internal/parser/differential -run '^TestNativeInlineParserMatchesPublishedGFMInlineProjection$' -count=1
go test ./internal/parser/differential -run '^TestNativeInlineRelationshipMatchesPublishedGFMAllParserExamples$' -count=1
```

Use the anchored exact test names shown above. `go test -run` can exit successfully after selecting zero tests, so shortened or guessed filters are not acceptable conformance or differential evidence.

The pinned GFM page contains 677 examples. Its explicit extension sections remain normative according to the hierarchy above. The inherited GFM core examples are retained as legacy compatibility/regression input, but cannot override a conflicting CommonMark 0.31.2 requirement. The single `tagfilter` example is rendering-specific and remains outside parser conformance while Marksplice core exposes no HTML renderer.

## Updating normative snapshots

Do not update an approved SHA merely because an official page changed.

A CommonMark or GFM snapshot update requires all of the following:

1. obtain the changed official published page from its canonical source;
2. review the specification diff and identify semantic changes, not only example-count changes;
3. classify whether each change belongs to the CommonMark base or an explicit GFM extension/correction;
4. add or update focused regression tests that cite the changed normative rule;
5. update native/temporary adapter behavior only where necessary to match the reviewed contract;
6. run the full conformance and repository regression gates;
7. update this document, affected capability/milestone documentation, and the appropriate approved hash in the same reviewed change.

If an implementation conflicts with the applicable published specification, record the discrepancy and follow the specification. If the specifications themselves do not determine the case, document the Marksplice decision and the secondary evidence used.

## Goldmark boundary

`github.com/yuin/goldmark` is the current temporary semantic parser implementation. While it remains in use, Marksplice may add narrowly scoped compatibility behavior inside `internal/parser/goldmark` where Goldmark differs from the specification-first Marksplice contract. Goldmark is never the normative oracle. The approved roadmap replaces this backend with the Marksplice-native parser and removes Goldmark at M115.

Rules:

- no Goldmark AST or parser-specific type may cross the Marksplice public API boundary;
- Goldmark's additional syntax packages do not define Marksplice core capabilities merely because Goldmark provides them;
- parser-library differences are adapter/test problems to resolve, not reasons to expose multiple dialect modes;
- ordinary existing-document edits must never depend on serializing a Goldmark AST back to Markdown;
- Marksplice continues to own exact source mapping, lexical trivia, source fingerprints, and minimal byte patches.

Historical M111–M113 evidence established parity with the Goldmark-backed production parser across the applicable published GFM 0.29 corpus. M114 completes the specification-first recertification: the full 652-example CommonMark chain, explicit GFM extension gate, focused Marksplice contract tests, and Native invariant fuzzing are authoritative acceptance evidence. Historical differential-fuzz rounds are discovery history only; retained inputs without an independent specification/Marksplice contract are source-bound/determinism invariants, not semantic authority. Goldmark remains the temporary production backend until the explicit M115 cutover, but no correctness decision is justified solely by matching it.

## Rendering boundary

Semantic Markdown parsing and HTML rendering are separate responsibilities. Marksplice core currently uses rendering only inside the conformance harness to compare parser semantics with specification examples; it does not expose HTML rendering as a product capability.

If Marksplice later exposes HTML rendering, GFM `tagfilter` behavior and any other rendering-specific requirements become mandatory acceptance criteria for that feature before it can claim GFM rendering conformance.

## Compatibility monitoring

Current `cmark-gfm` releases and GitHub-maintained GFM authoring guidance should be monitored when parser behavior changes or a new edge case appears. They are useful for discovering security fixes, implementation changes, and missing tests.

They remain advisory unless the normative GFM source itself changes through the reviewed snapshot-update process above.
