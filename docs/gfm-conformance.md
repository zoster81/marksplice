# GFM Conformance Policy

Status: source of truth for Marksplice's Markdown syntax profile, conformance hierarchy, and parser-upgrade gate.

## Normative profile

Marksplice targets GitHub Flavored Markdown (GFM) as its single Markdown syntax profile.

The normative language contract is the published GFM 0.29 specification at `https://github.github.com/gfm/`. GFM is defined as a strict superset of CommonMark, so CommonMark syntax is supported as the inherited base of GFM; Marksplice does not expose a separate strict-CommonMark mode.

A future request for a second Markdown dialect or a separately observable CommonMark mode requires an explicit architecture decision. It must not be introduced implicitly through parser configuration.

## Source hierarchy

When sources disagree, use this order:

1. the hash-pinned snapshot of the official published GFM 0.29 specification;
2. Marksplice conformance and focused regression tests that implement that pinned contract;
3. the current `github/cmark-gfm` implementation and release behavior as compatibility/security evidence;
4. GitHub-maintained authoring guidance, including `github/awesome-copilot` Markdown GFM instructions, as advisory evidence and a source of test ideas;
5. Goldmark behavior as an implementation detail.

Lower-ranked sources do not override higher-ranked sources. A difference found in `cmark-gfm`, GitHub authoring guidance, or Goldmark is evidence to investigate; it is not by itself a reason to change Marksplice semantics.

## Pinned specification snapshot

The conformance test reads an externally provisioned snapshot of the official published GFM page. The snapshot is intentionally not vendored into this Apache-2.0 repository because the upstream specification material is CC-BY-SA-4.0 validation input.

`internal/parser/goldmark/gfm_conformance_test.go` is the single repository source of truth for the approved snapshot SHA-256. Do not duplicate that hash in documentation or configuration. The test fails closed if the supplied snapshot has a different hash, even if it still identifies itself as `Version 0.29-gfm`.

To run the gate, set `MARKSPLICE_GFM_SPEC_HTML` to the approved snapshot and run:

```text
go test ./internal/parser/goldmark -run TestGFM029PublishedSpecificationConformance
```

The pinned page currently contains 677 examples. Marksplice exercises 676 parser/render examples through the production parser factory. The single `tagfilter` example is excluded because disallowed-raw-HTML filtering is an HTML-rendering responsibility and Marksplice core does not currently provide an HTML renderer.

## Updating the normative snapshot

Do not update the approved SHA merely because the official page changed.

A snapshot update requires all of the following:

1. obtain the changed official published GFM page from its canonical source;
2. review the specification diff and identify semantic changes, not only example-count changes;
3. add or update focused regression tests for changed behavior;
4. update the Marksplice Goldmark adapter only where necessary to match the reviewed GFM contract;
5. run the full conformance gate and repository regression suite;
6. update this document, affected capability/milestone documentation, and the approved SHA in the test in the same reviewed change.

If the new specification conflicts with current `cmark-gfm` or GitHub authoring guidance, record the discrepancy and follow the published specification unless GitHub publishes a superseding normative source.

## Goldmark boundary

`github.com/yuin/goldmark` is the selected semantic parser implementation. Marksplice's default parser uses GFM semantics and may add narrowly scoped compatibility behavior inside `internal/parser/goldmark` where Goldmark differs from the pinned GFM contract.

Rules:

- no Goldmark AST or parser-specific type may cross the Marksplice public API boundary;
- no non-GFM Goldmark extension is enabled by default merely because Goldmark provides it;
- parser-library differences are adapter/test problems to resolve, not reasons to expose multiple dialect modes;
- ordinary existing-document edits must never depend on serializing a Goldmark AST back to Markdown;
- Marksplice continues to own exact source mapping, lexical trivia, source fingerprints, and minimal byte patches.

The parser decision gate currently retains Goldmark because the production parser configuration matches all 676 applicable examples from the approved published-spec snapshot, with focused regression coverage for known parser-boundary cases such as GFM HTML comments and extended autolinks.

## Rendering boundary

Semantic Markdown parsing and HTML rendering are separate responsibilities. Marksplice core currently uses rendering only inside the conformance harness to compare parser semantics with specification examples; it does not expose HTML rendering as a product capability.

If Marksplice later exposes HTML rendering, GFM `tagfilter` behavior and any other rendering-specific requirements become mandatory acceptance criteria for that feature before it can claim GFM rendering conformance.

## Compatibility monitoring

Current `cmark-gfm` releases and GitHub-maintained GFM authoring guidance should be monitored when parser behavior changes or a new edge case appears. They are useful for discovering security fixes, implementation changes, and missing tests.

They remain advisory unless the normative GFM source itself changes through the reviewed snapshot-update process above.
