# Milestone M61 — Titled Reference-Definition Construction

Status: green — canonical titled reference-definition construction reusing the existing M7/M50 source proof.

## Goal

Extend M50 new-document reference-definition construction to the title-bearing shape that the parsed/source model already supports, without widening the public parsed `ReferenceDefinition` detail.

M61 adds:

```go
func (b *DocumentBuilder) AppendReferenceDefinitionWithTitle(label, destination, title string) error
```

The historical two-argument `AppendReferenceDefinition` remains the canonical no-title API.

## Canonical source policy

M61 writes exactly:

```text
[label]: <destination> "title"
```

The destination retains M50 angle-bracket syntax. The title uses one canonical double-quoted spelling.

`label` and `destination` retain the M50 validation policy. `title` must be:

- non-empty;
- valid UTF-8;
- NUL-free;
- single-line;
- free of `"` and backslash.

The last rule deliberately avoids introducing an escaping layer whose semantic/raw-source contract has not been separately reviewed. Titles requiring escapes fail with `ErrInvalidConstruction` rather than being rewritten.

## Parser/model proof

M61 reuses the existing `KindReferenceDefinition` and `ReferenceDefinitionMapping`. Construction acceptance requires exact:

- complete generated source range;
- destination range;
- title range;
- label and destination semantic values;
- title semantic value;
- `HasTitle == true`;
- angle-destination state.

No new parser kind, source mapper, public `Kind`, snapshot identity, or public title accessor is introduced.

The public test additionally reparses the generated definition and performs `PrepareReplaceReferenceDefinitionDestination`. Applying that source-preserving edit must change only the destination bytes while leaving the generated double-quoted title byte-identical, demonstrating convergence between the construction output and the existing editing path.

## Compatibility

M50 no-title construction remains byte-for-byte unchanged. Its proof now passes through the same generalized reference expectation with `HasTitle == false` and an empty title range.

The public parsed `ReferenceDefinition` continues to expose only its snapshot ID and exact editable destination `Range()`. M61 is therefore an additive construction capability rather than a new parsed API commitment.

## Complexity

Validation and writing are O(n) in label, destination, and title bytes. Parser/model proof remains linear in generated document size. The generalized expectation adds constant title metadata per reference-definition block.

## Devil's advocate review

### Risk: automatic escaping changes caller title semantics

Mitigation: the canonical title form rejects quote and backslash rather than inventing escaping behavior. A future escaping policy can be reviewed separately if needed.

### Risk: source title and Goldmark semantic title diverge

Mitigation: both `node.Title` and `ReferenceDefinitionMapping.TitleRange` must reproduce the exact requested title bytes before construction succeeds.

### Risk: adding title construction accidentally changes M50 no-title output

Mitigation: the original method remains unchanged; the shared writer emits title syntax only when the private `hasTitle` flag is set. Existing M50 tests remain part of every repository regression.

### Risk: later destination editing damages the constructed title

Mitigation: the focused public test prepares and applies a destination-only replacement after reparsing M61 output and requires the title source to remain byte-identical.

## TDD and verification evidence

The initial red run failed to compile only because `AppendReferenceDefinitionWithTitle` did not yet exist. The green focused tests cover canonical Unicode title output, parsed promotion, destination replacement with title preservation, nil receivers, failed-append immutability, and rejection of empty, multiline/CR, NUL, invalid-UTF-8, quote-containing, and backslash-containing titles.

Focused root tests pass. The complete repository suite, `go vet ./...`, `go build ./...`, and production gocyclo are green before documentation; no production function exceeds complexity 20.

Final verification on the documented M1–M61 tree passes five consecutive `go test ./... -count=1` runs, `go test -race ./... -count=1`, coverage, `go vet ./...`, `go build ./...`, public `go doc`, the pinned published-GFM 0.29 conformance gate, `staticcheck ./...`, standard `golangci-lint run` with zero issues, production gocyclo with no function above complexity 20 across 33 production files, production and test-inclusive unparam, `govulncheck ./...` with no vulnerabilities, and Gitleaks with no leaks. Final statement coverage is 93.3% for the public root package, 65.2% for `internal/parser/goldmark`, 79.3% for `internal/source`, and 66.7% for `internal/splice`.

Strict changed/untracked text hygiene passes across 55 paths with valid UTF-8, no BOM, no CR, and no trailing whitespace. `git diff --check` and `git fsck --no-dangling` pass. The branch remains `main` at `5c016772b7583693b1f73770448fa22ec52832d5` with no configured remote. No commit or push was performed.
