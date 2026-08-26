# Milestone M110 — Public Third-Party Syntax/Semantic SPI

Status: complete — third-party read-only extension SPI and release-quality verification are green.

## Goal

Define the smallest parser-backend-independent public SPI for explicitly opted-in, statically linked Go packages to attach dialect-specific read-only syntax/semantic observations to an immutable Marksplice document.

M110 is an extensibility boundary, not a first-party extension bundle. Core GFM behavior, core `Kind` values, source-preserving mutation, and canonical `DocumentBuilder` behavior remain unchanged.

## Requirements

The M110 contract is:

- `Parse` remains the baseline GFM entrypoint;
- extensions are enabled only through `ParseWithOptions`;
- a zero `ParseOptions` value is equivalent to `Parse`;
- extension namespaces and local kinds are separate from core `Kind`;
- observations own only validated non-empty snapshot-local byte ranges;
- extension metadata is copied before retention and remains immutable;
- total retained node count and scalar metadata bytes require positive caller-provided limits;
- duplicate namespaces and malformed configuration fail before callbacks run;
- recognizers run synchronously and serially in registration order and are never retained;
- recognizer errors, recovered panics, invalid output, or exhausted limits fail closed with `ErrInvalidExtension` and no partial document;
- extension observations never suppress, replace, or reclassify core nodes;
- overlaps between extension observations are allowed because M110 is read-only;
- no generic extension mutation or construction hook is introduced.

Third-party recognizers are ordinary caller-linked Go code. Marksplice can validate and bound the observations it retains, but it does not claim to sandbox or preempt the recognizer's own execution. That distinction is part of the public security/resource contract.

## Public API

M110 adds:

```go
var ErrInvalidExtension error

type ExtensionID string
type ExtensionKind string

type ExtensionAttribute struct {
    Name  string
    Value string
}

type ExtensionMatch struct {
    Kind       ExtensionKind
    Range      Range
    Attributes []ExtensionAttribute
}

type ExtensionSource struct { /* private */ }
func (s ExtensionSource) Text() string

type ExtensionRecognizer func(ExtensionSource) ([]ExtensionMatch, error)

type Extension struct {
    ID        ExtensionID
    Recognize ExtensionRecognizer
}

type ExtensionLimits struct {
    MaxNodes         int
    MaxMetadataBytes int
}

type ParseOptions struct {
    Extensions      []Extension
    ExtensionLimits ExtensionLimits
}

func ParseWithOptions(source []byte, options ParseOptions) (*Document, error)

type ExtensionNode struct { /* private */ }
func (n ExtensionNode) ExtensionID() ExtensionID
func (n ExtensionNode) Kind() ExtensionKind
func (n ExtensionNode) Range() Range
func (n ExtensionNode) Attributes() []ExtensionAttribute
func (n ExtensionNode) Attribute(name string) (string, bool)
func (d *Document) ExtensionNodes() []ExtensionNode
```

`ExtensionSource.Text()` is one immutable string copy of the exact successfully parsed snapshot. Its byte indexes use the same coordinate system as public `Range` values.

`MaxNodes` limits the total retained observations across all registered extensions. `MaxMetadataBytes` limits the bytes retained for observation extension IDs, local kinds, attribute names, and attribute values. There are no hidden global extension caps.

## Architecture

M110 deliberately sits above the ordinary core parse:

```text
source bytes
   |
   v
ordinary core Parse
   |
   +-- immutable core model
   |
   v
immutable source string copy (only when extensions are enabled)
   |
   v
serial build-local recognizers
   |
   v
Marksplice validation and caller limits
   |
   v
immutable ExtensionNode overlay
```

No M110 change is required in the parser adapter, source mapper, splice model, mutation engine, or builder proof path. This keeps the SPI independent from Goldmark and suitable for the M111–M115 native-parser transition.

The extension overlay is intentionally separate from core structure. An extension may recognize a wikilink-like range that sits inside an ordinary core paragraph, but the paragraph remains the same core node and the extension observation receives no core `NodeID` or `Kind`.

## Read-only boundary

M110 does not expose generic extension mutation or builder integration.

This is intentional. A generic edit API based only on a third-party claimed range would weaken Marksplice-owned mutation authority. A generic builder hook would also change `DocumentBuilder` from canonical reviewed GFM construction into an arbitrary dialect writer.

Independent extension packages may provide their own ordinary Go helpers, but such helpers are outside Marksplice's `ChangeSet` and `DocumentBuilder` contracts. Any future integration with source-preserving edits or canonical construction requires a separate reviewed design.

## Validation and failure behavior

Configuration is validated before core parsing and before any callback:

- extension limits are positive when extensions are present;
- each `ExtensionID` is non-empty valid UTF-8 without whitespace/control characters;
- recognizers are non-nil;
- extension IDs are unique by exact string equality.

The ordinary core parse then runs. Only a successful core document proceeds to extension recognition.

For every returned match, Marksplice validates:

- a valid non-empty extension-local kind;
- a non-empty range contained in the exact source snapshot;
- unique valid attribute names;
- valid UTF-8, NUL-free attribute values;
- remaining node budget;
- remaining metadata-byte budget using subtraction-based overflow-safe arithmetic.

Recognizer errors are preserved as causes while the operation is classified with `ErrInvalidExtension`. Recognizer panics are recovered at the extension-call boundary; error panic values remain discoverable with `errors.Is`. The document receives extension state only after the complete extension run succeeds.

## Ordering, ownership, and concurrency

Recognizers run in registration order. `ExtensionNodes()` preserves registration order and each recognizer's returned order. Duplicate extension IDs are rejected; otherwise overlapping observations remain independent overlays.

`ExtensionNodes()` returns caller-owned slice storage and `Attributes()` returns caller-owned metadata storage. The retained nodes are immutable and therefore participate in the M109 concurrent-read guarantee. The recognizer callbacks themselves are not retained after `ParseWithOptions` returns.

## Complexity

For source size `N`, retained extension nodes `E`, and retained scalar metadata bytes `M`:

- ordinary core parse cost is unchanged;
- extensions add one O(N) immutable source-string copy only when enabled;
- recognizer work is extension-defined;
- Marksplice validation/copying is O(E + M);
- retained extension state is O(E + M);
- no extension callback, registry, parser context, or secondary syntax tree is retained.

## TDD and review evidence

M110 began with a black-box missing-API RED, then added the minimal read-only extension overlay. Focused tests hardened configuration prevalidation, malformed kinds/ranges/metadata, callback panic isolation, overflow-safe metadata budgeting, zero-options parity, defensive source/metadata ownership, and concurrent immutable reads.

A follow-up allocation review removed one avoidable whole-snapshot copy from the extension path while preserving the public contract and all focused/full/race regressions.

## Devil's advocate review

1. **Third-party execution cannot be sandboxed by an ordinary in-process Go callback.** Mitigation: document this explicitly; caller trust governs extension code, while Marksplice validates and bounds only retained observations.
2. **Extension semantics could accidentally appear to replace GFM.** Mitigation: core parsing completes first and the overlay cannot suppress, replace, or reclassify core nodes.
3. **Malformed ranges or excessive metadata could corrupt or exhaust retained state.** Mitigation: validate every range/scalar and require caller-owned total output limits with overflow-safe accounting.
4. **One extension failure could leave partial state.** Mitigation: build extension state locally and assign it only after all recognizers/results pass.
5. **A parser-specific SPI would block the Goldmark exit.** Mitigation: the public boundary uses only source text plus Marksplice-owned types and contains no parser AST/context type.
6. **Generic extension edits/construction would weaken established authority boundaries.** Mitigation: M110 is read-only; any future write/construction integration requires a separate reviewed contract.

## Release-quality verification

The source-of-truth-aligned M110 tree passed repeated complete regressions, focused black-box tests, full race detection, examples and public GoDoc checks, `go vet`, `go build`, Staticcheck, golangci-lint, production `gocyclo <= 15`, production/test-inclusive unparam, `go mod tidy -diff`, published GFM conformance, Go 1.27 test/vet/build, govulncheck, Gitleaks, actionlint, strict text/artifact hygiene, and `git diff --check`.

Cross-package statement coverage measured **87.0% aggregate** and **84.7% through `internal/publictest`**.

## Exit decision

M110 is complete. The smallest reviewed third-party boundary is now frozen as an explicit parser-backend-independent read-only overlay with separate extension namespaces/kinds, validated/bounded immutable observations, serial non-retained recognizers, and no mutation/construction/parser/graph/host authority. The next roadmap boundary is **M111 — Native parser contract and differential harness**.
