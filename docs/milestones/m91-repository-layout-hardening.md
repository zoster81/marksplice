# M91 — Repository Layout Hardening

## Status

Complete and green.

## Objective

Make the repository immediately understandable to Go users and contributors before the first beta without changing the module path, public package, exported API, Markdown semantics, or source-preservation behavior.

## Requirements and constraints

The canonical consumer import must remain:

```text
github.com/zoster81/marksplice
```

For that reason the public `marksplice` package and `go.mod` remain at the module root. Moving the public package under a top-level `src/` directory would naturally create the import path `github.com/zoster81/marksplice/src`, or require a forwarding root package that duplicates the API boundary solely for cosmetic layout.

GitHub-conventional repository files also remain at the root: `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`, `LICENSE`, `NOTICE`, `go.mod`, and `go.sum`.

## Architecture and implementation

M91 performs structural relocation only:

1. all 44 `public_*_test.go` black-box API tests move from the module root to `internal/publictest/`;
2. the moved suite becomes package `publictest` and imports the root module exactly as an external consumer does;
3. `internal/publictest/doc.go` gives the package an explicit internal responsibility;
4. the single `example_test.go` remains at the module root so pkg.go.dev continues to publish executable examples for package `marksplice`;
5. root source filenames are grouped by responsibility without changing file content or package membership:
   - `api.go`, `api_types.go`, `api_details.go`, `api_sections.go`, `api_mutations.go`;
   - `builder.go`, `builder_frontmatter.go`, `builder_inline.go`, `builder_validation.go`, `builder_writer.go`, `builder_proof.go`;
   - `doc.go`;
6. `docs/README.md` becomes the documentation index and repository-layout map.

Private parser/source/splice packages remain under their existing `internal/` boundaries. No new runtime dependency or forwarding package is introduced.

## Test strategy

The relocation is proven in layers:

- `go test ./internal/publictest -count=1` proves the moved black-box suite is independently runnable;
- `go test . -run '^Example' -count=1` proves root pkg.go.dev examples remain executable;
- `go test ./... -count=1` proves normal repository discovery still executes every package and test family;
- the existing race, vet, build, static analysis, complexity, security, GFM conformance, and Go-version gates remain applicable to the exact same runtime code.

Because the black-box suite now lives in a different package directory, `go test . -cover` intentionally no longer represents project coverage. M91 switches maintainer coverage interpretation to explicit cross-package instrumentation. A first measurement over the complete explicit production package set produced 86.5% aggregate statement coverage; the `internal/publictest` run alone exercised 83.3% of that instrumented production set.

## Devil's advocate review

1. **Moving tests could silently stop `go test ./...` from running them.** `internal/publictest` has its own package declaration and `doc.go`; focused and full tests explicitly confirm discovery and execution.
2. **Moving tests could make coverage look artificially worse.** The old single-package coverage interpretation is no longer used; cross-package `-coverpkg` instrumentation measures code exercised by consumer-style tests.
3. **Adding `src/` could make the repository look familiar while breaking the Go import contract.** M91 deliberately avoids that layout and documents the Go module-root constraint instead.
4. **Renaming source files could accidentally imply semantic changes or lose history.** File contents and package declarations are unchanged; only responsibility-oriented filenames change, and `git` can detect the renames.
5. **Moving examples with the tests would degrade pkg.go.dev documentation.** `example_test.go` intentionally remains beside the public package at the module root.
6. **A cosmetic refactor immediately before release could introduce runtime risk.** M91 does not move runtime declarations across package boundaries; the complete release gate and GitHub workflow cycle must pass again before tagging.

## Verification

The exact M91 tree passed the complete local release gate with `M91_FULL_GATE_OK`: layout assertions, five consecutive `go test ./... -count=1` runs, race detection, explicit cross-package coverage, vet, build, package examples/documentation, the hash-pinned published GFM 0.29 conformance suite, Staticcheck, golangci-lint with zero issues, production complexity <=15, production/test `unparam`, `govulncheck` with no vulnerabilities found, Gitleaks with no leaks, `actionlint`, `go mod tidy -diff`, direct Go 1.27 compatibility, strict text hygiene, private-boundary scanning, `git diff --check`, and `git fsck --no-dangling`.

Cross-package statement coverage on that tree is 86.7% over the explicit production package set; the `internal/publictest` consumer-style suite alone exercises 83.3% of that instrumented set.

M91 was committed as `7c697fc544f80945a6b0ae90983d5d19d179df62` and pushed to public `main`. GitHub Actions run `32634536155` (`CI`) completed successfully for that exact commit. No beta tag existed when this evidence was recorded.

## Exit decision

M91 is complete. The repository layout is accepted for the first public beta. The release tag remains a separate publication boundary and may be created only after the final release-documentation commit also completes all GitHub workflows successfully.
