# M120 — HTML Fragment Renderer

Date: 2026-08-29
Status: Complete locally; freeze-ready, unreleased pending milestone commit, push, and exact remote CI closure

## Goal

M120 exposes deterministic HTML-fragment rendering over the M118–M119 Native semantic event stream without adding a second Markdown parser, retained renderer AST, hidden I/O, or whole-document output buffering to the primary writer path.

The public boundary is deliberately small:

- `Document.RenderHTML(io.Writer, HTMLRenderOptions)` for streaming output;
- `Document.HTML(HTMLRenderOptions)` for caller-owned buffered bytes;
- explicit raw-HTML, dangerous-URL, and GFM tag-filter policies;
- `ErrInvalidRender` for invalid renderer input/options.

Rendering is an explicit export path. It does not replace source-preserving editing or new-document Markdown construction.

## Requirements and edge cases

The renderer must:

- consume `parser.SemanticBackend.WalkSemantic` rather than reparse Markdown delimiters or construct a second AST;
- produce deterministic HTML fragments, not standalone HTML documents;
- stream to the caller's writer and stop immediately on writer failure;
- retain no rendering state in `Document` after the operation returns;
- preserve parser-proven raw HTML by default while applying the published GFM tag filter;
- offer explicit full raw-HTML escaping for callers that need a stricter HTML trust boundary;
- suppress dangerous URL schemes by default and permit them only through an explicit option;
- normalize Markdown escapes/entities and percent-encode destinations consistently with the reviewed CommonMark/GFM rendering expectations;
- render task lists, tables, tight/loose lists, blockquotes, alerts, code, links/images/autolinks, footnotes, and reviewed mathematical forms from semantic facts;
- omit front matter and reference-definition declarations from visible HTML output;
- perform no URL/image/asset fetching, filesystem discovery, network access, command execution, syntax highlighting, template execution, or math-engine execution;
- remain deterministic for malformed, dense-delimiter, CRLF, Unicode, and large inputs without mutating source bytes.

Preserved raw HTML is active markup and is not a sanitizer. `HTMLRawEscape` or an application-appropriate downstream sanitization boundary is required when untrusted Markdown crosses an HTML security boundary.

## Architecture and test strategy

`internal/renderhtml` owns HTML emission only. It receives semantic events from Native through the existing `internal/splice` document bridge and keeps operation-local renderer stack state. Image-alt collection and footnote-body collection use bounded local buffers only where output semantics require deferred emission; the primary writer path does not build a complete HTML result in memory.

The renderer does not own Markdown grammar. During TDD, full expected-HTML comparison exposed several semantic projection defects that were fixed at their existing Native semantic boundary rather than patched with renderer-specific Markdown heuristics. Parser-neutral M115 contracts were kept frozen: when a rendering requirement was broader or stricter than an existing editable/parser-neutral shape, the change was made only in the M120 semantic projection or public mapper.

The permanent conformance strategy has two layers:

1. selected focused CommonMark/GFM examples remain fast targeted regressions for individual renderer behaviors;
2. full profile-aware tests compare every non-divergent official example byte-for-byte with the approved published expected HTML.

The full renderer gate is independent from the complete M115 parser-neutral fixture and from the selected M119 semantic-event gate. Current Native output is never accepted as its own HTML oracle.

## Semantic defects found and corrected

Expected-HTML TDD exposed and corrected the following shared semantic facts:

- exact consumed delimiter ranges for nested emphasis/strong runs, preserving unmatched residual delimiter bytes;
- empty ATX heading semantic containers;
- strict entity-reference grammar before standard HTML entity decoding;
- short GFM table rows padded semantically to the header width;
- table-level `\|` escape consumption inside code-span cell values without widening editable code-span promotion;
- indented-code blank-line whitespace preservation;
- fenced-code final body-line terminator preservation;
- empty list-item semantics and correct tight-list block separators;
- unsupported list-item editing shapes failing closed in the public mapper while parser-neutral observations remain unchanged;
- HTML block values inside blockquotes excluding container marker bytes;
- multiline inline HTML tags promoted from the existing Native cross-segment scanner for rendering semantics only;
- reviewed GFM HTML-comment validity applied at the semantic rendering boundary without changing the frozen M115 parser-neutral observation contract;
- link/image label precedence preventing emphasis delimiter pairs from matching across an active composite-label boundary while still allowing pairs wholly inside or wholly outside the label.

These fixes reuse Native ownership and lexical primitives. No second Markdown parser was introduced in `internal/renderhtml`.

## Public rendering policies

The zero value of `HTMLRenderOptions` is the documented default:

- `HTMLRawPreserve`: preserve parser-proven raw HTML;
- `HTMLUnsafeURLSuppress`: replace dangerous link/image destinations with an empty destination;
- `HTMLTagFilterEnabled`: apply the published GFM disallowed-tag filter to preserved raw HTML.

Callers can independently choose:

- `HTMLRawEscape` to emit all parser-proven raw HTML as escaped text;
- `HTMLUnsafeURLAllow` as an explicit trust decision;
- `HTMLTagFilterDisabled` when exact preserved raw HTML is required.

Policy validation fails closed with `ErrInvalidRender`. Writer errors remain writer errors and are returned immediately.

## Devil's advocate review

1. **A renderer could silently become a second Markdown parser.**
   The accepted implementation consumes only `SemanticEvent` facts. Rendering-specific conformance failures that revealed missing grammar facts were repaired in Native semantic projection, never by rescanning Markdown syntax in the renderer.

2. **Rendering fixes could destabilize the frozen parser/editing model.**
   Multiple early fixes demonstrated this risk: a list-item change and strict HTML-comment change initially affected M115 parser-neutral contracts. Both were revised so parser-neutral observations remain unchanged and the stricter behavior lives only in the public edit capability gate or semantic rendering projection.

3. **Raw HTML preservation could be mistaken for sanitization.**
   The public option documentation explicitly states that preserved raw HTML remains active markup. The default GFM tag filter is not presented as a general sanitizer, and `HTMLRawEscape` is available as an explicit stronger policy.

4. **Unsafe URLs could bypass policy through entities or escapes.**
   Destinations are Markdown-decoded before dangerous-scheme classification, so forms such as entity-encoded `javascript:` are suppressed by default. Percent encoding happens only after policy classification.

5. **Streaming could still hide whole-document buffering.**
   The writer benchmark measures `RenderHTML(io.Discard)` separately from the buffered `HTML` helper. Streaming allocation is only about 0.96 MiB above the semantic walk on the 256 KiB workload, while the helper allocates roughly another 2.0 MiB to accumulate output.

6. **Conformance exceptions could become a wildcard escape hatch.**
   The permanent full-profile tests contain exact example-number maps and a written reason for every deliberate Marksplice-profile divergence. Any other mismatch fails the test immediately.

## Full rendering conformance

The permanent M120 renderer gates use the same approved externally provisioned, hash-verified specification snapshots as parser conformance.

### CommonMark 0.31.2

All 652 examples are accounted for. Six are deliberate Marksplice-profile divergences:

- example 98 — leading empty YAML front matter has explicit Marksplice precedence over CommonMark thematic-break interpretation;
- examples 608, 611, 612 — the reviewed extended-autolink profile is always enabled;
- examples 625, 626 — Marksplice deliberately uses the reviewed newer GFM HTML-comment grammar.

Every other **646/646 applicable CommonMark example** renders byte-identically to the published expected HTML with the GFM tag filter disabled.

### Published GFM

All 677 examples are accounted for. Four inherited core examples are deliberate Marksplice-profile divergences:

- example 68 — leading empty YAML front matter has explicit Marksplice precedence;
- examples 617, 620, 621 — the reviewed extended-autolink profile is always enabled.

Every other **673/673 applicable published-GFM example** renders byte-identically to the published expected HTML. The rendering-only `tagfilter` example is included with tag filtering enabled even though it remains correctly absent from the 676-case M115 parser-neutral fixture.

## Robustness and performance evidence

A permanent pathological renderer test uses a 64 KiB dense delimiter source, renders the same immutable document twice, requires byte-identical output, and verifies caller source bytes remain unchanged.

The M120 benchmark uses the same representative 256 KiB realistic source for all paths. On the recorded Windows/amd64 Ryzen 9 5900X host, five-run medians are approximately:

| Path | Median time | Allocated bytes/op | Allocation count |
| --- | ---: | ---: | ---: |
| public `Parse` | 45.3 ms | 53.3 MB | ~233k |
| Native `WalkSemantic` | 40.6 ms | 40.8 MB | ~235k |
| `RenderHTML(io.Discard)` | 52.9 ms | 41.7 MB | ~258k |
| buffered `HTML` | 54.3 ms | 43.8 MB | ~258k |

Wall-clock values remain host-sensitive engineering evidence. Allocation bytes are the more useful architecture signal: the primary streaming path does not allocate a complete HTML result, while the convenience helper pays the expected output-buffer cost.

## Documentation and public boundary

M120 adds one user-facing rendering workflow without turning project history into the primary documentation path:

- `README.md` lists deterministic HTML fragments as a current capability;
- `docs/getting-started.md` shows the first streaming/buffered calls;
- `docs/guide.md` routes rendering as a separate user goal;
- `docs/recipes/render-html.md` owns practical policy/safety guidance;
- `docs/capabilities.md` records the exact rendering boundary;
- `docs/api-reference.md` records the exported renderer API;
- `docs/gfm-conformance.md` owns the independent full renderer-conformance rules;
- `docs/architecture.md` records the durable event-stream/no-second-parser/no-hidden-I/O boundary.

## Final verification state

The complete local freeze candidate has passed the project release-quality stack on the exact M120 source state:

- focused Native and public renderer regressions, including HTML-comment grammar, invalid options, writer failure, raw-HTML/tagfilter/unsafe-URL policy, URI normalization, image-alt semantics, and pathological deterministic/source-preserving input;
- five consecutive complete Go 1.26.6 `go test ./... -count=1` runs plus the actual CGO/GCC race detector;
- the exact CommonMark/GFM parser-neutral gates, selected M119 semantic gates, selected M120 renderer examples, and both permanent full-profile M120 expected-HTML gates;
- `go vet`, `go build`, Staticcheck, golangci-lint with zero issues, production `gocyclo <= 15`, and production/test-inclusive `unparam`;
- `go mod tidy -diff`, govulncheck with no known vulnerabilities, Gitleaks, and actionlint;
- isolated Go 1.27.0 test/vet/build plus CGO-disabled `linux/amd64`, `darwin/amd64`, and `darwin/arm64` cross-builds;
- documentation dogfood over 154 Markdown documents with zero workspace diagnostics and **362/362** exported callables represented in the API reference;
- strict UTF-8/no-BOM/LF/no-NUL/no-trailing-whitespace hygiene on the exact M120 change set, private-boundary checks for that change set, `git diff --check`, `git fsck --no-dangling`, and complete changed/untracked inventory review.

The renderer dispatcher refactor also reduced every production function to the existing cyclomatic-complexity limit without changing the specification-backed HTML output. A final Staticcheck simplification in the Native HTML-comment state machine was recertified with its exact focused grammar test, the complete conformance stack, five full regressions, and race detection.

M120 is therefore locally freeze-ready. It is not remotely closed until the ordinary milestone commit is pushed and GitHub Actions completes successfully on that exact pushed SHA.

## Exit boundary

M120 is locally complete and release-quality gates are green on the reviewed freeze candidate. The authorized next actions are the ordinary milestone freeze commit and non-force push, followed by exact remote-SHA and GitHub CI verification.

M121 — standalone HTML and metadata — is the next engineering boundary after exact M120 remote closure. It may wrap the fragment renderer in a reviewed standalone-document/export contract, but it must not weaken source-preserving editing, raw-HTML/URL policy, conformance, or core authority boundaries.
