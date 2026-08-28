# M117 — Filesystem Resolution Hardening

Date: 2026-08-28
Status: Complete; unreleased

## Goal

M117 closes the filesystem path-resolution ambiguity intentionally left by M116 before later features build on `workspacefs`.

The milestone changes no exported API and does not change Markdown parsing. It hardens the existing read-only adapter over caller-supplied `fs.FS` authority, keeps graph and validation semantics delegated to the root package, and finishes with a measured traversal refactor rather than speculative caching.

## Requirements and edge cases

The filesystem resolver must define one deterministic policy for `workspacefs.Follow`, `Workspace.BuildGraph`, and `Workspace.Validate`.

Required behavior:

- resolve ordinary relative `.md` and `.markdown` paths, including `./`, nested paths, and literal `..` when source-relative normalization remains inside the supplied `fs.FS` namespace;
- treat the relationship destination as a URI reference, using the path component for file lookup;
- percent-decode each path component exactly once;
- preserve the target fragment for the existing fragment resolver;
- exclude query text from filesystem lookup;
- reject encoded traversal components and encoded path separators;
- reject malformed percent escapes, raw backslashes, empty path segments, absolute paths, URI schemes, protocol-relative targets, directories, and extensionless targets;
- do not invent directory indexes, extensions, case corrections, or host-path conversions;
- preserve the case, symlink, and other path semantics of the caller-provided `fs.FS` instead of claiming a universal filesystem sandbox;
- preserve finite document, byte, depth, and relationship budgets and cycle-safe traversal.

## Architecture and test strategy

M117 keeps one package-private `localTarget` resolution path as the single source of truth for filesystem relationship interpretation. `Follow` calls it before discovery, while `Workspace.BuildGraph` and `Workspace.Validate` use it through their existing resolver adapters. This avoids a second graph, parser, or validation implementation.

Resolution proceeds in this order:

1. ignore parser-classified email relationships;
2. split fragment and query from the relationship destination without interpreting query bytes;
3. validate the URI path shape and reject absolute, protocol-relative, backslash, empty-segment, and scheme forms;
4. percent-decode each slash-delimited path component exactly once;
5. reject decoded separators, encoded `.`/`..` traversal components, NUL, or malformed encoding;
6. normalize the decoded relative path against the source document with slash-based `path` semantics;
7. require the normalized result to satisfy `fs.ValidPath`, remain a Markdown file, and not resolve to the workspace root;
8. return the normalized document key plus the preserved fragment.

Black-box tests use `testing/fstest.MapFS` so behavior is independent of host-path syntax. TDD first established the expected failures against M116, then the smallest shared resolver change was implemented, followed by full regressions and profiling.

## Devil's advocate review

1. **Encoded input could bypass traversal or separator checks.**
   A generic unescape followed by path cleaning could turn `%2e%2e` or `%2F` into new authority after validation. M117 decodes components exactly once and rejects encoded traversal components and decoded separators before source-relative normalization. Root-escaping normalized paths still fail `fs.ValidPath`.

2. **Discovery, graph construction, and validation could disagree.**
   Separate path interpreters would allow a document to be followed under one rule but reported missing or ignored under another. All three paths use the same `localTarget` resolver.

3. **Cycles and repeated missing targets could amplify allocation and filesystem work.**
   `Follow` still marks discovered documents before enqueue. Profiling justified replacing queue reslicing with an index-based breadth-first queue and merging the prior `checked`/`availability` maps into one operation-local tri-state availability map that caches both existing and missing targets.

4. **`fs.FS` could be mistaken for a complete host security sandbox.**
   Marksplice never resolves native host paths, rewrites case, dereferences symlinks itself, or claims to constrain semantics implemented by the supplied filesystem. The caller owns the `fs.FS` instance and its case/symlink behavior; `workspacefs` only limits the slash-relative names it requests through that interface.

## TDD and implementation result

The initial M117 RED proved the expected M116 gaps for percent-encoded paths and query-bearing relationships. It also exposed one concrete M116 defect: `docs//inside.md` could be silently normalized to `docs/inside.md`. The hardened resolver rejects that empty-segment form.

Focused coverage now proves:

- `./` and parent-relative paths that stay inside the workspace;
- percent-encoded spaces and Unicode in path components;
- query/file separation even when query bytes are not valid percent escapes;
- preserved percent-encoded fragments resolved by the existing fragment machinery;
- a colon inside a non-leading path segment;
- encoded and mixed traversal attempts;
- encoded slash/backslash separators;
- malformed percent escapes, NUL, and invalid decoded path bytes;
- one-pass rather than recursive percent decoding;
- duplicate slashes, absolute paths, protocol-relative paths, URI schemes, drive-like scheme syntax, raw backslashes, directory targets, and extensionless targets;
- caller-filesystem case behavior and exact missing-document diagnostics.

No new public function, type, option, or error was needed.

## Refactor and profiling

The M116 baseline and final M117 measurements use the same private synthetic `fstest.MapFS` harness on Go 1.26.6 and an AMD Ryzen 9 5900X. Timing is engineering evidence rather than a portable performance guarantee; allocation deltas and scaling are the more stable signals.

| Workload | M116 median | M117 median | M116 B/op | M117 B/op | M116 allocs/op | M117 allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Scan 256 | 1,835,738 ns | 1,802,575 ns | ~1,663,811 | ~1,663,811 | 12,330 | 12,330 |
| Follow chain 256 | 3,475,977 ns | 3,481,799 ns | ~2,342,048 | ~2,324,769 | 18,964 | 18,704 |
| Follow dense 256 | 3,261,669 ns | 3,117,215 ns | ~1,919,790 | ~1,892,906 | 11,716 | 11,702 |

The URI-path semantic change itself produced no measurable allocation-count increase before the traversal refactor. The final allocation profile shows parsing/document construction remains dominant; the target-existence helper fell from roughly 3% to roughly 1.5% of allocation space in the dense workload.

Scaling from 256 to 1,024 documents for 4x input measured approximately:

- `Scan`: 4.69x;
- chained `Follow`: 4.22x;
- dense `Follow`: 4.31x.

Allocated bytes and allocation counts remain approximately proportional to input. No persistent cache, secondary graph, path index, or hidden size cap was added.

## Verification

Before documentation freeze, the implementation passed:

- focused `workspacefs` tests and full `go test ./... -count=1`;
- real CGO/GCC `go test -race ./... -count=1`;
- `go vet ./...` and `go build ./...`;
- Staticcheck and golangci-lint with zero issues;
- production cyclomatic complexity at 15 or lower;
- production and test-inclusive `unparam`;
- `govulncheck`, Gitleaks, and actionlint;
- `go mod tidy -diff` and `git diff --check`;
- Go 1.27 test/vet/build on Windows;
- CGO-disabled Linux amd64, macOS amd64, and macOS arm64 builds.

The CommonMark/GFM published-specification corpus and the private 6,857-document parser corpus were not rerun for the implementation-only gate because M117 does not touch Native parsing, parser observations, construction proof, or Markdown semantics. That omission is deliberate rather than a claim that those unrelated heavy gates were executed.

Repository dogfood on the documented tree scans 150 Markdown files, builds 137 resolved edges, and reports zero workspace diagnostics through the public `workspacefs.Scan`/`BuildGraph`/`Validate` flow. The final documented-tree freeze gate repeats the applicable product, race, static, security, documentation, supported-Go, and cross-platform checks on this same tracked tree.

## Exit decision

M117 is complete locally when the documented tree, repository dogfood, and final quality/hygiene gates are green. The milestone introduces no exported API change and keeps filesystem authority isolated in `workspacefs`.

M118 is the next engineering boundary. M118 must not be treated as started by this record.