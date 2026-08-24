# M102 — Semantic Block Patterns

## Status

**Complete.** Focused TDD, architecture/complexity review, implementation verification, source-of-truth alignment, and the final documented-tree release-quality freeze are green. No commit or push is authorized by this milestone record.

## Objective

M102 adds a reviewed semantic layer for useful patterns that are already valid baseline GFM. The first pattern is GitHub alerts: `NOTE`, `TIP`, `IMPORTANT`, `WARNING`, and `CAUTION` represented over ordinary top-level blockquote source.

The feature does **not** add a Markdown grammar mode, a new parser extension, a new public structural `Kind`, a second node identity namespace, renderer behavior, or mutation authority beyond the blockquote/source contracts already established by M94.

## Public read model

The public semantic enum is:

```go
type AlertKind uint8

const (
    AlertKindUnknown AlertKind = iota
    AlertKindNote
    AlertKindTip
    AlertKindImportant
    AlertKindWarning
    AlertKindCaution
)
```

`Alert` is a comparable immutable overlay over one already-promoted top-level blockquote:

- `ID()` is the underlying blockquote `NodeID`;
- `Kind()` is one of the five reviewed alert kinds;
- `Range()` is exactly the complete physical source range already owned by `Blockquote.Range()`;
- `MarkerRange()` is the exact first inner source range containing the alert marker.

`Document.Alert(id)` resolves one semantic alert by the underlying blockquote identity. `Document.Alerts()` enumerates recognized alerts in authoritative source order and returns caller-owned storage. `Document.AlertBodyRanges(id)` returns caller-owned M94 inner source ranges after the marker line.

No alert metadata is retained in the immutable snapshot. Recognition is a call-local projection over already-source-proven blockquote nodes.

## Recognition boundary

M102 recognizes an alert only when all of the following are true:

1. the underlying node is an already-promoted editable top-level `KindBlockquote`;
2. M94 has proven at least two physical inner source segments;
3. the first inner segment is byte-for-byte exactly one of:
   - `[!NOTE]`;
   - `[!TIP]`;
   - `[!IMPORTANT]`;
   - `[!WARNING]`;
   - `[!CAUTION]`;
4. at least one subsequent owned inner segment is non-empty.

The matcher deliberately does not normalize case, leading/trailing spaces, trailing text, or unknown labels. Marker-only/empty-body shapes are outside this reviewed subset. Alerts nested inside another blockquote or list are not promoted because M94 exposes only complete top-level blockquote identities and GitHub alerts are not a nested-element construct.

Body source remains ordinary blockquote source. Blank marker-only lines are returned as valid empty ranges. Lazy continuation or other broader M94-owned lines retain their exact proven ranges rather than being normalized.

## Construction model

`DocumentBuilder` adds:

```go
func (b *DocumentBuilder) AppendAlert(kind AlertKind, inlineGFM string) error
func (b *DocumentBuilder) AppendAlertContent(kind AlertKind, content ...Inline) error
func (b *DocumentBuilder) AppendAlertBlocks(kind AlertKind, content *DocumentBuilder) error
```

`AppendAlert` reuses the existing parser-proven LF-only blockquote-paragraph contract for the body. `AppendAlertContent` reuses the typed-inline writer before delegating to that path.

`AppendAlertBlocks` snapshots the current reviewed body blocks of a child builder and reuses the M83–M86 canonical blockquote child writer/proof. Front matter and deferred reference definitions remain document-level state and are rejected as alert children.

Construction has private `constructionAlert` and `constructionAlertBlocks` intent kinds rather than disguising alerts as generic builder blockquotes. They still render and reparse as ordinary `KindBlockquote`, but the private intent prevents an alert block from being admitted as a child through `AppendBlockquoteBlocks` or another `AppendAlertBlocks` call. This preserves the non-nesting contract without changing GFM parsing.

The writer emits canonical depth-1 source:

```text
> [!WARNING]
> body
```

For multi-block content, every canonical child physical line—including blank separators—is prefixed by the ordinary canonical `> ` writer. Generated source must pass both the existing blockquote semantic/source proof and an additional exact alert-marker/mapping proof.

## Existing-source editing

M102 introduces no alert-specific rewrite API. An alert remains the same underlying blockquote structural node, so already-established blockquote operations such as `PrepareRemoveBlockquote(alert.ID())` retain their exact M94 source-preserving behavior, including preservation of surrounding blank trivia.

Changing an alert kind or rewriting arbitrary alert body source is not implicitly authorized by semantic recognition. Such APIs would require their own source-preserving operation contracts if introduced later.

## Performance and retained state

Let `N` be the number of existing promoted nodes and `L` the total number of M94 content-range entries inspected for recognized blockquotes.

- `Alert(id)` performs one snapshot lookup plus inspection of that blockquote's first/body ranges;
- `Alerts()` is O(N + L) worst case and returns O(A) alert values for `A` matches;
- `AlertBodyRanges` copies only the selected alert's body range vector;
- no alert index, marker table per document, parser state, or secondary tree is retained.

Construction adds only scalar alert intent plus ordinary child snapshots already used by blockquote construction. No new dependency or parser extension is introduced.

## Devil's advocate review

### Risk 1 — semantic overlay accidentally changes the parser contract

Implementing alerts as a Goldmark extension or a new Marksplice `Kind` would couple the public model to product syntax and complicate the mandatory native-parser cutover. M102 instead recognizes exact marker bytes only after ordinary GFM blockquote ownership is proven.

### Risk 2 — alert construction can be nested accidentally

If an alert were stored as the existing generic private blockquote kind, a child builder containing an alert could later be quoted by `AppendBlockquoteBlocks`, silently changing the intended GitHub semantic shape. Dedicated private alert construction kinds are deliberately absent from the supported blockquote-child set, so nesting fails closed.

### Risk 3 — permissive marker normalization invents unsupported semantics

Case folding, whitespace trimming, or accepting trailing custom titles would guess beyond the reviewed GitHub marker contract. M102 uses byte-exact uppercase markers and rejects all other spellings.

### Risk 4 — semantic recognition becomes mutation authority

Recognizing a marker does not establish safe source ranges for changing marker/type/body independently. M102 therefore adds read and construction semantics only; existing-source mutation remains exactly the underlying blockquote contract.

### Risk 5 — construction proof forks blockquote logic

A separate alert block writer/parser proof would duplicate mature M81–M86 machinery. The implementation factors the blockquote inner-source writer and reuses the existing paragraph/multi-block candidate proof, adding only exact marker/mapping checks.

## TDD and implementation evidence

- `tsk_446adf0d06f1bb901587c3e41ce0ca2e` is non-authoritative because its command referenced an obsolete toolchain activation path, although compilation still reached the expected missing-API errors.
- `tsk_97583ea48d914e768a26e6f96b5b232e` is the authoritative public missing-API RED: the private toolchain activated correctly and compilation failed only on absent M102 `Alert`/`AlertKind`/document APIs.
- `tsk_625a75f7907c9806b1bfcbf3165298f9` exercised the first implementation. All new recognition/construction cases passed; only one removal assertion expected normalization of surrounding blank lines contrary to the established M94 contract. The test was corrected rather than changing product behavior.
- `tsk_708a41c6449081f2d49d3117b0d69f33` is the corrected focused M102 plus complete repository GREEN and clean `git diff --check`.
- `tsk_c5e53410dd49adb559c3b0e147cd56e5` passed all functional tests but correctly rejected two production complexity hotspots at 19 and 16. It is retained as refactor evidence, not a final maintainability gate.
- `tsk_3ac454a780510a6b78ae7839dac2315c` verified the responsibility-based refactor: focused M102, complete regression, production `gocyclo <= 15`, and `git diff --check` all passed.
- `tsk_da6295d93674153c11253ff14aea4431` passed the pre-freeze API/example/build/static/maintainability stack: focused M102, executable `ExampleDocument_Alerts`, `go vet`, `go build`, public API docs, Staticcheck, golangci-lint with `0 issues.`, production `gocyclo <= 15`, production and test-inclusive unparam, `go mod tidy -diff`, and `git diff --check`.

## Documented-tree release-quality freeze

After source-of-truth alignment, the exact documented M102 tree passed:

- focused M102 public tests, complete repository regression, executable `ExampleDocument_Alerts`, public API documentation, module-diff check, and `git diff --check`: `tsk_76720a5b32ea16734b0a43414cfafc42`;
- branch/HEAD/origin, working-tree, unchanged `go.mod`/`go.sum`, and pre-freeze diff-state verification: `tsk_05a04c8e7e2e1b2fc38fd976e2ff5449`;
- five consecutive complete `go test ./... -count=1` runs plus the actual race detector with `CGO_ENABLED=1` and the private GCC toolchain: `tsk_273d637d7f3a217ad31e91c0981dac07`;
- Go-file formatting cleanliness, `go vet`, `go build`, executable alert example, public alert API docs, Staticcheck, golangci-lint with `0 issues.`, production `gocyclo <= 15`, production and test-inclusive unparam, `go mod tidy -diff`, and `git diff --check`: `tsk_848b41558de900db1c2dbbd2bf0a0cbe`;
- govulncheck (`No vulnerabilities found.`), Gitleaks (`no leaks found`), and actionlint: `tsk_be5d36075ef5321efb60a4b68fa08c4c`;
- exact published GFM 0.29 conformance using the explicit approved private snapshot, with verbose `=== RUN` and `--- PASS`: `tsk_f45ae76bf25fad4e9856301e4c36d3ba`;
- isolated Go 1.27.0 Windows test/vet/build with explicit matching `GOROOT`, private `GOPATH`/`GOCACHE`, and `GOTOOLCHAIN=local`: `tsk_632430bbec6d1182cb5b6649c66f9492`;
- corrected explicit production-package cross coverage: `tsk_c408b970bc27bccb76c8b2559e1c2c5c`, reporting **87.1% aggregate** statement coverage and **84.5% through `internal/publictest`**, with `PROFILES_REMAIN=False`.

`go.mod` and `go.sum` remain unchanged through the documented-tree freeze.

## Exit decision

M102 is **complete**. GitHub alert semantics remain a Marksplice-owned exact overlay over ordinary top-level blockquote source, reuse the established blockquote identity/ownership model, add no parser grammar or persistent semantic index, and keep existing-source mutation authority unchanged. The next roadmap boundary is **M103 — Fenced-block semantics**, generalizing exact fence, info-string/language, content, and source ownership while keeping embedded technical payloads opaque.
