# M105 — Mathematical expressions

Status: complete; implementation, refactor, source-of-truth alignment, and exact documented-tree release-quality verification are recorded below.

## Objective

M105 adds conservative GitHub-compatible mathematical source semantics as a Marksplice core capability without adding a mathematical parser, renderer, network dependency, or caller-selectable Markdown dialect.

The reviewed Markdown-level forms are:

- inline dollar source: `$payload$`;
- inline dollar-backtick source: `$` followed by one backtick-delimited payload and a closing `$`;
- one-physical-line block dollar source: `$$payload$$`;
- exact-info top-level fenced blocks whose parsed info string is `math`.

Mathematical payload is opaque. Marksplice does not parse or validate LaTeX, TeX, MathJax, KaTeX, MathML, or any other mathematical language/rendering model. The pinned GFM 0.29 parser profile remains unchanged; M105 is an explicitly reviewed core semantic overlay outside that baseline.

## Public read model

M105 appends one structural kind for source-proven non-fenced mathematical forms:

```go
const KindMathExpression Kind = ...

type MathExpressionStyle uint8

const (
    MathExpressionInlineDollar MathExpressionStyle = ...
    MathExpressionInlineBacktick
    MathExpressionBlockDollar
    MathExpressionFencedBlock
)

type MathExpression struct { /* immutable */ }

func (m MathExpression) ID() NodeID
func (m MathExpression) Style() MathExpressionStyle
func (m MathExpression) Range() Range
func (m MathExpression) PayloadRange() (Range, bool)

func (d *Document) MathExpressions() []MathExpression
func (d *Document) MathExpression(id NodeID) (MathExpression, bool)
func (d *Document) MathExpressionPayloadRanges(id NodeID) ([]Range, bool)
```

`Range()` owns the exact reviewed syntax/container. Block-dollar ownership includes its physical line terminator when present. `PayloadRange()` is available when one contiguous payload span is proven. `MathExpressionPayloadRanges` returns caller-owned source-backed ranges and reuses M103 per-physical-line fenced payload ownership.

An exact-info `math` fenced block is a semantic projection over the existing M103 fenced node, not a second structural node. It therefore reuses the same `NodeID`; `Document.Node(id)` continues to report `KindFencedCode`, while `MathExpressions` can report the same identity with `MathExpressionFencedBlock`. This avoids two structural identities for one fenced container.

Dollar/backtick/block-dollar expressions are promoted as `KindMathExpression` and participate in `QueryNodes`. Fenced-math projection is intentionally not duplicated into `QueryNodes(KindMathExpression)` because its authoritative structural kind remains `KindFencedCode`.

## Parser and source-ownership boundary

Goldmark 1.8.5 has no enabled mathematical extension in Marksplice and M105 does not add one. The authoritative GFM parser instance remains unchanged.

`internal/parser/goldmark` performs three bounded Marksplice-owned observation passes over the already-parsed GFM AST/source:

- inline-dollar scanning is limited to eligible text runs already used by conservative unresolved-reference observation, excluding code spans, links, images, autolinks, and raw HTML ownership;
- dollar-backtick recognition is anchored to an exact simple Goldmark code-span observation and proves the surrounding dollar/backtick delimiters independently;
- block-dollar recognition requires one complete top-level one-physical-line paragraph whose entire source is exact `$$payload$$`.

Exact GFM observations that exist only because Goldmark does not know the reviewed mathematical overlay are suppressed narrowly: the code-span observation inside dollar-backtick math and the paragraph observation exactly equal to a block-dollar form. Ordinary code spans/paragraphs outside those exact forms remain unchanged.

`internal/source.MapMathExpression` independently proves delimiter/source ownership. It rejects empty payloads, multiline payloads for the three dedicated forms, escaped/adjacent ambiguous delimiters, unescaped dollar characters inside payload, malformed dollar-backtick boundaries, and block-dollar source that is not one complete physical line. CRLF block ownership includes the complete `\r\n` terminator.

The three source-ordered observation streams are merged linearly rather than sorted globally. No parser AST/context, math lookup table, persistent interval index, or second syntax tree survives snapshot construction.

## Existing-source mutation

M105 adds:

```go
func (d *Document) PrepareReplaceMathExpression(id NodeID, replacement []byte) (ChangeSet, error)
```

For `$...$`, dollar-backtick, and `$$...$$`, the operation patches only the proven payload span. Replacement must be non-empty and single-line; an unescaped dollar is rejected, and dollar-backtick payload additionally rejects backticks. Candidate reparsing must reproduce the same style, source ownership, top-level classification, and transformed payload/container ranges.

For exact-info `math` fenced blocks, the operation reuses the historical M103/M1 fenced-code replacement capability and therefore grants no broader edit authority than the existing contiguous `FencedCode` mapping. Broader readable fenced shapes remain readable without automatically becoming editable.

All edits retain snapshot fingerprinting/stale-source rejection and preserve every untouched byte.

## Construction

M105 adds typed inline construction and one canonical block constructor:

```go
func MathInline(payload string) Inline
func MathBacktickInline(payload string) Inline
func (b *DocumentBuilder) AppendMathBlock(payload string) error
```

`MathInline` emits `$payload$`; `MathBacktickInline` emits the reviewed dollar-backtick form. Both are accepted only at the ordinary top-level typed-inline sequence and are not silently admitted inside structured emphasis/link/image child hierarchies that have not been reviewed.

`AppendMathBlock` emits one canonical `$$payload$$` physical line. Multiline mathematical construction uses the already-existing `AppendFencedCode(..., "math")` path rather than inventing multiline dollar-block semantics.

Construction rejects empty/non-UTF-8/NUL/multiline payloads, unescaped dollars, and backticks in the dollar-backtick payload. Generated bytes are reparsed and must reproduce exact Marksplice math style/ranges. Math construction therefore reuses the established builder parser/source-proof boundary rather than introducing a separate serializer or math AST.

## Integration with existing core surfaces

- **M95 composition:** dedicated math nodes contribute style/payload source facts to the structural composition view. Independently prepared changes in separate structural regions compose normally. Two math edits inside the same paragraph remain the same logical aggregate and are deliberately rejected, preserving M95's existing overlap invariant.
- **M97 queries:** `KindMathExpression` is appended to the fixed-size public kind filter and uses the complete math-owned range as its selection span. The oversized-filter regression derives its bound from the newest public kind rather than a historical literal.
- **M103 fenced blocks:** exact-info `math` is a semantic projection with the same identity/source ownership; no second fenced node/index exists.
- **M99/M100 relationships/graph:** mathematical payload is opaque. Marksplice does not parse links or graph edges from mathematical language text; fenced payload semantics remain governed by the ordinary M103 opaque-content rule.

## Complexity and retained state

The mathematical overlay performs a constant number of bounded passes over source/AST observations. Each individual scanner is source-linear for its inspected regions; the three already-source-ordered result vectors are merged in linear time. Snapshot state adds only promoted dedicated math nodes in the existing node vector. Fenced math uses existing fenced-node state.

Enumeration is an O(N) scan over the existing source-ordered node vector and allocates only caller-owned results. Candidate mutation proof uses the established deliberate reparse model. No mathematical parser, rendering state, persistent syntax index, reverse map, filesystem/network authority, or hidden cache is retained.

The first implementation temporarily raised four production functions above the complexity-15 gate. The refactor separated delimiter proof, supplemental-node promotion, auxiliary construction dispatch, and typed-inline relationship expectation dispatch; the complete repository then returned to `gocyclo <=15` before source-of-truth freeze.

## Devil's advocate review

### Risk 1 — ordinary currency/dollar text is guessed as mathematics

Dollar characters are common prose. M105 therefore requires exact unescaped paired delimiters in parser-eligible text, rejects adjacent dollar ambiguity and empty payloads, and does not guess unmatched or escaped dollar text. Source already owned by code/link/image/autolink/raw-HTML constructs is excluded.

### Risk 2 — mathematical overlay changes normative GFM behavior

M105 does not enable a Goldmark math extension or alter the pinned GFM parser profile. It observes/reconciles only Marksplice-owned math facts after ordinary parsing, and exact published GFM conformance remains a mandatory release gate.

### Risk 3 — one source region receives duplicate public identities

Dollar-backtick math can look like a code span and fenced `math` is already a fenced node. Exact code-span/paragraph conflicts are suppressed only for dedicated forms; fenced math deliberately reuses the M103 `NodeID` and keeps `KindFencedCode` as its structural kind.

### Risk 4 — math edit changes delimiter interpretation

Payload replacement is never accepted based on lexical substitution alone. Candidate reparsing and independent source mapping must reproduce the exact original style and transformed ownership. Ambiguous `$`/backtick/newline replacements fail closed.

### Risk 5 — composition weakens same-aggregate safety

Two payload edits inside the same paragraph could be source-disjoint but still jointly alter one Markdown aggregate. M105 does not special-case around M95: same-paragraph independent math edits remain rejected, while changes in separate structural regions compose and pass one combined candidate proof.

### Risk 6 — math support expands into a rendering/execution subsystem

Payload stays opaque. No MathJax/KaTeX/MathML/LaTeX parser, script execution, network fetch, rendering engine, or embedded-language validation enters core.

## Verification

M105 was developed through focused RED/GREEN tests covering parser/source observations, independent source proof, payload replacement, typed and block construction, M97 queries, M95 composition, CRLF handling, and ambiguous delimiter rejection.

The documented tree passed repeated complete regressions, the race detector, executable examples and public API documentation checks, `go vet`, `go build`, Staticcheck, golangci-lint, production `gocyclo <= 15`, production/test-inclusive unparam, `go mod tidy -diff`, published GFM 0.29 conformance, Go 1.27 test/vet/build, govulncheck, Gitleaks, actionlint, strict text/artifact hygiene, and `git diff --check`.

Cross-package statement coverage measured **86.7% aggregate** and **84.3% through `internal/publictest`**.

## Exit decision

M105 is **complete**. Marksplice now has conservative GitHub-compatible mathematical source semantics with independently source-proven dedicated forms, exact M103 fenced-math projection, opaque payload reading, source-form-preserving replacement, bounded M97 query integration, deterministic typed/canonical construction, and no mathematical parser/renderer/dependency or widened filesystem/network authority.

The next roadmap boundary is **M106 — Metadata/front-matter generalization audit**.
