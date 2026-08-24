# M103 — Fenced-block semantics

Status: complete implementation contract; documented-tree verification is recorded below.

## Objective

M103 generalizes fenced-code reading around the complete source-owned GFM fenced container while preserving the narrower historical payload-replacement contract.

The milestone is intentionally about Markdown structure and source ownership. Info-string values such as `mermaid`, `geojson`, `topojson`, `stl`, `math`, `d2`, `pikchr`, or any other language identifier remain opaque data. Marksplice does not parse, execute, render, syntax-highlight, or validate embedded payload languages.

## Public read model

`FencedBlock` is a read-only immutable view over one source-proven top-level GFM fenced container. It exposes:

```go
type FencedBlock struct { /* immutable */ }

func (f FencedBlock) ID() NodeID
func (f FencedBlock) Range() Range
func (f FencedBlock) OpeningFenceRange() Range
func (f FencedBlock) FenceChar() byte
func (f FencedBlock) OpeningFenceLength() int
func (f FencedBlock) OpeningIndent() int
func (f FencedBlock) Info() (string, bool)
func (f FencedBlock) InfoRange() (Range, bool)
func (f FencedBlock) Language() (string, bool)
func (f FencedBlock) Closed() bool
func (f FencedBlock) ClosingFenceRange() (Range, bool)
func (f FencedBlock) ClosingFenceLength() (int, bool)
func (f FencedBlock) ClosingIndent() (int, bool)

func (d *Document) FencedBlocks() []FencedBlock
func (d *Document) FencedBlock(id NodeID) (FencedBlock, bool)
func (d *Document) FencedBlockContentRanges(id NodeID) ([]Range, bool)
```

`Range()` owns the complete physical top-level container. For a closed block it includes the opening line, body, closing fence line, and the closing line terminator when present. For an unclosed block it extends through source EOF. `FencedBlockContentRanges` returns caller-owned source-backed payload ranges in source order; an empty fenced block returns an empty slice with `ok=true` rather than a synthetic zero-width payload line.

`Info` is the parser-proven semantic info string after the GFM parser's ordinary surrounding-space handling. `InfoRange` owns the exact corresponding source bytes. `Language` is the parser-proven language token derived from that info string. These values are metadata only.

`FencedBlock` reuses the existing fenced-code snapshot `NodeID`; M103 does not introduce a second identity namespace or a new public structural `Kind`.

## Parser and source-ownership boundary

The temporary Goldmark adapter remains responsible for fenced-block grammar and semantic facts. M103 adds a narrow `BlockParser` decorator around Goldmark's public fenced-code parser solely to retain the opening fence position on the resulting AST node. The decorator delegates opening recognition, continuation, closure, and node construction to Goldmark instead of copying or forking its fenced-code grammar. This closes the empty-body position gap where `Lines()` contains no segment from which the opening position could otherwise be recovered.

Parser-independent observations passed into Marksplice contain only scalar/source facts: opening anchor, per-body-line source ranges, info string, language, and top-level status. Goldmark AST/types remain inside `internal/parser/goldmark`.

`internal/source.MapFencedBlock` then independently proves exact ownership against the immutable source snapshot:

- the parser anchor points at a backtick or tilde opening run of at least three characters;
- opening indentation is within the GFM top-level fenced-code allowance;
- exact source info bytes agree with the parser semantic info string;
- every parser-proven payload range belongs to the expected physical source line and reaches that line's physical content end;
- a closing fence, when present, uses the opening delimiter, has at least the opening run length, and satisfies the reviewed indentation/trailing-space form;
- source before/after the owned container is never absorbed by a guessed range.

Parser/source disagreement fails closed: the semantic node may still exist internally, but the broader `FencedBlock` public ownership is not fabricated.

## Legacy edit compatibility

M103 deliberately separates broader readability from mutation authority.

Historical `FencedCode` remains the typed view for a payload proven to be one exact contiguous source span. `PrepareReplaceFencedCode` continues to patch only that span and preserve all opening/closing fence bytes, info source, indentation, line endings, and surrounding document bytes.

M103 widens that historical mapping only where the same contiguous proof still exists for an unclosed fence. An unclosed block with a complete contiguous payload can therefore retain source-preserving payload replacement; the replacement candidate must remain unclosed. `FencedCodeMapping.Closed` is part of survivor proof so a candidate cannot silently gain or lose its closing fence.

Indented multiline bodies whose semantic payload is represented by non-contiguous source ranges are readable through `FencedBlock` but remain outside the ordinary payload-replacement API. Empty fenced blocks are also readable through `FencedBlock`; the historical replacement API still requires a non-empty replacement span and therefore does not invent an editable zero-width body.

This boundary prevents a read-model improvement from becoming implicit generic rewrite authority.

## Construction

`DocumentBuilder.AppendFencedCode` continues to emit canonical unindented backtick fences and to choose a fence longer than every potentially closing backtick run in a non-empty body.

M103 adds empty payload construction. Empty content emits an opening fence line followed immediately by the matching closing fence line; no synthetic blank payload line is inserted. Existing LF-only payload and conservative info-string validation remain unchanged.

Construction proof now validates the complete generated `FencedBlock` container first. Non-empty generated blocks additionally retain the historical contiguous `FencedCode` expectation. Empty generated blocks prove that the complete container has zero payload ranges.

## Complexity and retained state

Fenced-block source proof is linear in the physical lines owned by the fenced container and retains one source-backed range per semantic payload line. `Document.FencedBlocks` scans the existing source-ordered node collection and stores no persistent fence index, payload-language registry, renderer state, or secondary AST.

The final M103 refactor centralizes public/internal range conversions and factors the already-proven opening fence into one shared `mapFencedBlockFromOpening` path. The historical contiguous mapper therefore reuses its opening proof instead of reparsing the same fence. No exported behavior or generated bytes changed.

## Devil's advocate review

### Risk 1 — semantic parser positions are treated as complete lexical ownership

Container positions alone are insufficient for lossless source work, especially for empty or indented bodies. Mitigation: Goldmark contributes semantic facts and a source anchor; Marksplice independently proves opening/info/body/closing physical ownership before exposing `FencedBlock`.

### Risk 2 — broader reading accidentally widens mutation authority

A complete container can be source-proven even when the semantic payload is not one contiguous replaceable span. Mitigation: `FencedBlock` is read-only and separate from the historical contiguous `FencedCode` edit capability.

### Risk 3 — an empty body invents a payload location

Treating adjacent opening/closing fences as containing a synthetic blank line would create bytes that do not exist. Mitigation: empty blocks expose zero payload ranges, and canonical construction writes adjacent fence lines.

### Risk 4 — an embedded fence-looking payload line is mistaken for closure

A lexical scanner that ignores parser/source sequencing could truncate body ownership. Mitigation: closure is accepted only after all preceding parser-proven body lines agree with source and the physical line satisfies the reviewed matching closing-fence grammar.

### Risk 5 — language recognition gains execution or renderer authority

Technical info-string names can tempt renderer-specific behavior into core. Mitigation: info/language are immutable metadata values only; embedded payload parsing, execution, validation, highlighting, rendering, network, filesystem, and command authority remain outside Marksplice core.

## Implementation and verification evidence

Focused implementation/refactor evidence:

- `tsk_236ad6a7d31c62aa834b43caaaa14395` — focused public M103, historical fenced-code, builder, and direct source-mapping regressions all green;
- `tsk_c8b23a7f616c50bf2512c99526709686` — direct Goldmark-adapter proof for opening anchors, empty bodies, info/language, and indented payload source ranges;
- `tsk_141eed5650652f29b048165fcb8382cc` — complete repository regression after the historical unclosed-fence contract migration;
- `tsk_b2ec5d970fe2fd36904c7b4579047b11` — final reuse/performance refactor followed by focused/source/full tests, vet, Staticcheck, production complexity `<=15`, and `git diff --check`.

Pre-documentation release-quality freeze:

- `tsk_a126ea630af58c3dfc60e3d068a5c923` — five consecutive complete `go test ./... -count=1` runs plus the actual CGO/GCC race detector;
- `tsk_3452afb080a39357eb17c298dd0fea19` — formatting cleanliness, vet, build, executable `ExampleDocument_FencedBlocks`, public API docs, Staticcheck, golangci-lint (`0 issues.`), production `gocyclo <= 15`, production/test-inclusive unparam, `go mod tidy -diff`, and `git diff --check`;
- `tsk_25a66addd86163044ff57bf0deacd786` — govulncheck (`No vulnerabilities found.`), Gitleaks (`no leaks found`), and actionlint;
- `tsk_7a259dd05fa8415085c184d68a7750e6` — exact published GFM 0.29 conformance with explicit approved private snapshot and verbose `RUN`/`PASS` evidence;
- `tsk_28db65f2a450284a2dae0a0a0710214e` — isolated Go 1.27.0 Windows test/vet/build with matching toolchain root and private caches;
- `tsk_f4a9a033a39d6b0dd3a7b149b9ef6215` — corrected explicit production-package cross coverage: **87.1% aggregate** statement coverage and **84.6% through `internal/publictest`**, with temporary profiles removed (`PROFILES_REMAIN=False`).

## Documented-tree release-quality freeze

After source-of-truth alignment, the exact documented M103 tree passed:

- `tsk_08817336c21efdf80eb776a9933fc756` — focused public M103 tests, complete repository regression, executable `ExampleDocument_FencedBlocks`, public fenced-block/API documentation, unchanged `go.mod`/`go.sum`, and `git diff --check`;
- `tsk_a611730ac1ef2b0cca19427d63766ade` — five consecutive complete `go test ./... -count=1` runs plus the actual CGO/GCC race detector;
- `tsk_a8f07238b9b2855d4307bb0c2cb0ed9e` — Go formatting cleanliness, vet, build, fenced-block example/API docs, Staticcheck, golangci-lint, production `gocyclo <= 15`, production/test-inclusive unparam, `go mod tidy -diff`, and `git diff --check`;
- `tsk_54a872e8561a292a69cdaabe8436c597` — govulncheck (`No vulnerabilities found.`), Gitleaks (`no leaks found`), and actionlint;
- `tsk_b7b8d37507424fc4260e4e58821733f6` — exact published GFM 0.29 conformance with `MARKSPLICE_GFM_SPEC_HTML` set explicitly to the approved private snapshot and verified verbose `RUN`/`PASS` evidence;
- `tsk_b89bd736080b4246119b2fee7c26c02d` — isolated Go 1.27.0 Windows test/vet/build with an explicit matching toolchain root and private caches;
- `tsk_e11c26023debd06b55b2c28a17c06486` — corrected explicit production-package cross coverage: **87.1% aggregate** statement coverage and **84.6% through `internal/publictest`**, with private temporary profiles removed (`PROFILES_REMAIN=False`);
- `tsk_2ed12622232b1903c514a3eb82de6600` — repository-wide UTF-8/no-BOM/LF/no-trailing-whitespace/no-NUL verification across tracked and untracked files, private-path/artifact leakage checks, unchanged `go.mod`/`go.sum`, `git diff --check`, `git fsck --no-dangling`, and branch/HEAD/origin/status confirmation.

An earlier coverage harness attempt is non-authoritative because PowerShell passed literal `$all`/`$pub` profile arguments and created two 10-byte temporary files in the repository. Those task-created artifacts were identified, removed without touching existing work, and the corrected coverage plus hygiene gates above prove the clean state.

## Exit decision

M103 is **complete**. Marksplice now owns a generic source-proven top-level fenced-block read model with exact fence/container/info/language/body metadata while preserving the narrower contiguous `FencedCode` mutation boundary. Empty and unclosed forms are represented without synthetic source, embedded languages remain opaque data, and the temporary Goldmark grammar remains isolated behind the parser adapter.

The next roadmap boundary is **M104 — Footnotes**.
