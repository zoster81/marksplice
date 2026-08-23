# M74 — Source-Preserving Simple Blockquote Removal

## Status

Complete and green.

## Objective

Add exact removal of one promoted M73 simple top-level blockquote physical line while rejecting candidate joins that reinterpret surviving Markdown.

## Public contract

```go
func (d *Document) PrepareRemoveBlockquote(id NodeID) (ChangeSet, error)
```

The operation deletes exactly `Blockquote.Range()`. It never synthesizes or removes neighboring whitespace and remains bound to the immutable source snapshot.

## Reuse and candidate proof

M74 deliberately reuses the M72 whole-block removal survivor validator rather than adding a blockquote-specific candidate index. The deleted blockquote observation and all of its contained paragraph/inline observations overlap the owned physical line and are expected to disappear. Every original observation outside that line must survive one-to-one after range transformation with unchanged reviewed semantic facts and applicable source anchors; no unexpected observed node may appear.

This detects genuine GFM join hazards. In the focused adversarial case:

```text
before
> quoted
---
```

removing only the blockquote line would turn `before\n---` into a Setext heading, so preparation fails with `ErrInvalidReplacement`.

## Risks and mitigations

1. Contained paragraph/inline nodes disappear with the blockquote. M72's validator skips all original observations whose semantic range overlaps the exact owned removal span, not only the target container node.
2. A safe byte deletion can still reinterpret neighboring blocks. One full candidate parse plus survivor proof is authoritative.
3. Reusing a generic validator could weaken specialized operations. The helper remains scoped to whole-block deletion; list/table/section edits retain their family-specific validators.
4. Stale application could mutate a different snapshot. The ordinary `ChangeSet` fingerprint binding remains unchanged.

## Evidence

Focused TDD covers exact CRLF line removal, stale-source rejection, and the Setext-heading join hazard. Full Go regression, `go vet`, `staticcheck`, and `git diff --check` passed before documentation. A first invalid test fixture using a lazy blockquote continuation was corrected after GFM semantics showed that the following line belonged to the blockquote rather than being an external survivor.

Final verification on the uncommitted M63–M74 working tree passed:

- `gofmt` on changed Go files;
- `go test ./... -count=1`;
- `go test -race ./... -count=1`;
- `go vet ./...`;
- `staticcheck ./...`;
- `golangci-lint run` with zero issues;
- `govulncheck ./...` with no vulnerabilities found;
- `gitleaks dir . --no-banner --redact` with no leaks found;
- `go build ./...`;
- coverage: root 93.0%, Goldmark adapter 66.2%, source 78.7%, splice 58.1%, aggregate 69.5%;
- corrected stale-document scan with no stale M72/blockquote-deferred markers;
- `git diff --check`;
- final branch/HEAD review retained `main` at `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured remote and only intended M63–M74 working-tree changes.

The original M74 gate could not resolve `gocyclo` because the existing analysis-binary directory was missing from the toolchain activation `PATH`; the earlier diagnosis that the analyzer was not installed was therefore incorrect. A later M75–M76 audit rediscovered the existing `gocyclo`/`unparam` binaries and restored the project complexity gate. On the descendant tree, `gocyclo` exposed excessive complexity in M72 survivor comparison and two growing kind-conversion switches; those paths were refactored without changing M71–M74 APIs or semantics until production `gocyclo -over 20` was empty and production/test-inclusive `unparam` reported no findings. This is supplemental descendant-tree evidence and is not presented as if those analyzers had run during the exact original M74 gate.

No commit or push was performed.
