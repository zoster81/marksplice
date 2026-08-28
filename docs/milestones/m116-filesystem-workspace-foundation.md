# M116 - Filesystem workspace foundation

Status: **Complete**

## Goal

Make Marksplice's existing explicit multi-document graph and workspace validator practical for Markdown already stored in a caller-authorized filesystem, without moving hidden filesystem authority into the root document core.

M116 adds a separate public `workspacefs` package over caller-provided `fs.FS` authority. It does not change `DocumentGraph` into a crawler, does not add network access or writes, and does not attempt to settle the complete path-resolution policy reserved for M117.

## Requirements and edge cases

The filesystem adapter must:

- accept an explicit `fs.FS` and slash-valid workspace root;
- discover `.md` and `.markdown` files deterministically under that root;
- parse loaded bytes through ordinary `marksplice.Parse`;
- assign deterministic slash-relative `DocumentKey` values;
- optionally start from explicit entry documents and follow reviewed local Markdown relationships;
- visit relationship cycles once;
- preserve missing local targets as validation facts instead of inventing documents;
- reuse existing graph and workspace-validation semantics;
- bound documents, bytes, depth, and relationships;
- reject malformed roots/entries/options and fail closed on budget exhaustion;
- perform no filesystem writes, network access, or command execution.

M116 intentionally keeps relationship-path recognition conservative. Absolute paths, protocol/scheme targets, query-bearing destinations, percent-bearing destinations, backslash-looking destinations, root-escaping `..` forms, and other ambiguous/non-local shapes are not followed. Exact resolution semantics for those families, plus case/symlink/filesystem-specific behavior, remain M117 work.

## Architecture and test strategy

The implementation lives in the separate top-level `workspacefs` package. The root `marksplice` package remains the in-memory document/graph core.

The adapter has two loading modes:

1. `Scan` scopes the caller filesystem with `fs.Sub` when needed, traverses with `fs.WalkDir`, selects Markdown files, enforces directory/document depth and load budgets, and parses documents in deterministic slash-relative order.
2. `Follow` validates/deduplicates/sorts explicit Markdown entries, loads them through the same bounded loader, and performs queue-based relationship traversal with a visited set and target-existence cache.

Both modes produce one immutable `Workspace` containing existing `marksplice.GraphDocument` inputs. `Workspace.BuildGraph` delegates to `marksplice.BuildDocumentGraph`; `Workspace.Validate` delegates to `marksplice.ValidateWorkspace`. The package therefore owns filesystem adaptation and conservative local-path interpretation only, not a second graph implementation or parser.

File reads use `io.LimitReader` with one-byte overflow detection before `io.ReadAll` can consume beyond the remaining byte budget. Relationship/document counts are checked before accepting additional retained state. Public variable-length results are defensive copies.

Focused black-box tests were established before production implementation. The initial expected RED was a build failure because the new public package did not yet contain production Go files. The first GREEN then covered deterministic nested discovery, graph reuse, validation, cycle-safe following, missing targets, fragments, hostile destination forms, and all four budget families. Follow-up review added deterministic entry ordering/deduplication, defensive ownership, nil behavior, finite-default checks, and explicit root-escape coverage.

## Devil's advocate review

1. **Filesystem resolution could accidentally become broader authority than intended.** M116 follows only conservative slash-relative Markdown destinations contained by `fs.ValidPath` after source-relative normalization. Ambiguous/non-local forms are ignored rather than guessed, and all filesystem access comes from the caller's supplied `fs.FS`.
2. **Traversal could amplify CPU, memory, or I/O on large trees or cycles.** Every operation requires finite document, byte, depth, and relationship limits. `Follow` uses visited-before-enqueue behavior and cached target existence; `Scan` uses one deterministic `fs.WalkDir` traversal.
3. **The adapter could duplicate graph/validation semantics and drift from core.** `Workspace.BuildGraph` and `Workspace.Validate` directly delegate to the established root APIs with adapter-provided resolution facts. The adapter retains no second graph algorithm.
4. **A nominal byte budget could still allocate an oversized file before rejection.** Reads are bounded at the reader layer and request at most the remaining budget plus one overflow byte.
5. **Caller mutation could corrupt retained workspace order or keys.** `Documents` returns a caller-owned slice copy while parsed `Document` snapshots remain immutable.

The final focused refactor also removed an unused internal `addDocument` result found by `unparam`, leaving the helper contract limited to the relationship data actually consumed by `Follow`.

## Implementation

M116 adds:

- package `github.com/zoster81/marksplice/workspacefs`;
- `DefaultOptions` and finite default budgets;
- `Limits` for documents, bytes, depth, and relationships;
- `ErrInvalidInput` and `ErrBudgetExceeded` sentinel families;
- `Scan` for deterministic recursive Markdown discovery;
- `Follow` for deterministic entry-driven, cycle-safe local relationship traversal;
- immutable `Workspace.Documents`;
- `Workspace.BuildGraph` using the existing `DocumentGraph` implementation;
- `Workspace.Validate` using the existing workspace validator;
- conservative local Markdown destination mapping with fragment preservation;
- bounded file reads and target-existence caching.

The runnable workspace example now uses `workspacefs.Scan` over `os.DirFS` instead of manually listing files and maintaining duplicate resolver logic. Public README, Getting Started, guide, recipes, capabilities, API reference, architecture, roadmap, changelog, documentation map, contributor guidance, and repository instructions are aligned with the new explicit filesystem boundary.

## Verification

The completed documented implementation passes:

- focused `workspacefs` black-box tests;
- complete `go test ./... -count=1` regression;
- actual CGO/GCC `go test -race ./... -count=1`;
- `go vet ./...` and `go build ./...`;
- Staticcheck and golangci-lint with zero issues;
- production `gocyclo` with no function above complexity 15;
- production and test-inclusive `unparam`;
- `govulncheck` with no vulnerabilities found;
- Gitleaks with no leaks found;
- actionlint;
- `go mod tidy -diff` and `git diff --check`;
- Go 1.27.0 test/vet/build;
- CGO-disabled cross-builds for Linux amd64, macOS amd64, and macOS arm64;
- runnable filesystem-workspace example, producing four documents, five resolved edges, and the expected single orphan diagnostic;
- repository-wide `workspacefs` dogfood over 149 Markdown documents, producing 137 resolved edges and zero workspace diagnostics.

M116 does not modify `go.mod` or `go.sum` and adds no production dependency.

## Exit decision

M116 is complete. Marksplice now has an explicit, bounded, read-only `fs.FS` workspace adapter that feeds the existing document graph and workspace validator without weakening root-core authority boundaries or introducing duplicate parsing/graph implementations.

M117 is the next engineering boundary and owns the exact filesystem relationship-resolution policy plus the planned broad `workspacefs` refactor/profiling pass. No M117 behavior is part of this milestone.
