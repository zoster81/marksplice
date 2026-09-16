# Release and Versioning Policy

Status: source of truth for public Marksplice module versioning and release preparation.

## Versioning policy

Marksplice is a Go module at:

```text
github.com/zoster81/marksplice
```

The project follows Go module semantic-version conventions. During public API development, releases remain in the `v0` series and may use explicit alpha/beta/RC pre-release identifiers. A v0 or pre-release version carries no compatibility or stability guarantee. Starting with v1, backward compatibility follows Semantic Versioning: compatible additions and fixes stay within v1, while an intentionally breaking public API change requires a new major version and the corresponding Go module-path decision.

The first public beta version is:

```text
v0.1.0-beta.1
```

The module path intentionally has no `/vN` suffix while the major version is v0 or v1. A future v2+ release would require the corresponding major-version module-path suffix and a separate architecture/release decision.

The minimum supported Go version is owned by the `go` directive in `go.mod`. Go 1.26 is the current compatibility floor for the v1 line. Public CI resolves the latest available patch in both the `1.26.x` compatibility line and the `1.27.x` primary line (`actions/setup-go` with `check-latest: true`). Maintainer release gates should likewise use the latest stable patch in both supported lines and record the exact compiler versions in milestone/release evidence rather than raising the public floor merely to use the newest release compiler.

Published tags are immutable. Never move, delete-and-recreate, or otherwise rewrite a version that has been made available to Go tooling. Publish a new version instead.

## Approved stable release targets

The approved post-M115 roadmap targets:

- `v0.5.0-beta.1` as the pre-M116 performance beta after the dedicated parser/document-model optimization campaign meets its measured throughput/allocation gates and full correctness/conformance verification;
- `v1.0.0` as the first stable release after M124 completed the full API-stability, refactor, profiling, conformance, documentation, and release-readiness gate;
- `v1.5.0` after the deferred M125–M126 PDF backend/adapter line completes its own release gate.

The v0.5 line was intentionally optimization-first. `v0.5.0-beta.1` is the completed pre-M116 performance beta. M124 closes the first stable v1.0 line; M125–M126 remain deferred v1.5 work and must not be folded into the v1.0 release. Reaching a milestone implementation boundary does not by itself publish a release: tags/releases are created only from the exact reviewed commit after the required local and GitHub Actions gates are green.

Official Go module references:

- https://go.dev/doc/modules/publishing
- https://go.dev/doc/modules/release-workflow
- https://go.dev/doc/modules/version-numbers

## Public release readiness

Before every public release, verify that the repository contains and accurately describes:

- `go.mod` and `go.sum` with the canonical module path;
- package documentation in `doc.go`;
- compiled package examples suitable for pkg.go.dev;
- `README.md` installation, current release status, and minimum-Go-version guidance;
- Apache-2.0 `LICENSE` plus project `NOTICE`;
- `CHANGELOG.md` with the planned release notes;
- `SECURITY.md` with non-public vulnerability reporting guidance;
- contributor guidance and the GFM conformance policy;
- a public GitHub Actions workflow covering supported Go versions and major operating systems;
- dependency-update configuration for Go modules and GitHub Actions.

Repository settings that should remain configured for the public GitHub repository:

- default branch: `main`;
- GitHub Actions enabled;
- private vulnerability reporting enabled;
- branch protection/ruleset requiring the public CI checks before merge;
- repository description and topics that identify Go, Markdown, GFM, source preservation, and structural editing;
- Issues enabled if community bug reports are desired.

## Pre-release verification

Run from the repository root on the exact commit intended for the tag:

```text
go mod tidy
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
go build ./...
git diff --check
git status --short
```

`go mod tidy` must not leave an unexplained `go.mod` or `go.sum` diff.

The project also maintains stricter local quality/security gates documented in `CONTRIBUTING.md`. The hash-pinned published GFM conformance gate requires the separately provisioned approved specification snapshot described in `docs/gfm-conformance.md`; that corpus is intentionally not vendored into the public repository.

Before publication, test the module from a separate temporary consumer module. For an unpublished local checkout, use a `replace` directive that points to the checkout, then compile and test a small program importing `github.com/zoster81/marksplice`. Remove the temporary consumer after verification; it is not repository content.

## Public repository status

The public GitHub repository and `origin` remote already exist. Ordinary development/finalization work must not create or replace remotes, push commits, create tags, or publish releases unless that action is explicitly authorized. Release preparation therefore starts from an already-configured public repository and an exact reviewed local commit.

## Publishing a module version

After the exact release commit exists on the public `main` branch, wait for every GitHub Actions run associated with that exact commit to complete successfully. Do not create a release tag while any run is queued/in progress or if any run concludes unsuccessfully. This commit-level workflow gate is mandatory even when the same tree already passed the stricter local maintainer gate.

Only after all workflows for the exact release commit are green, create and push an annotated immutable tag. Use the actual reviewed version; the stable v1 form is:

```text
git tag -a v1.0.0 -m "Marksplice v1.0.0"
git push origin v1.0.0
```

For a prerelease, use its exact semantic version instead, for example `v1.0.0-rc.1`. Tagging and pushing remain separately authorized actions; release preparation alone does not authorize either operation.

Because public CI also runs on tag pushes, wait for every GitHub Actions run associated with the tag's target commit/ref to complete successfully before creating a GitHub release or advertising the module as published. Mark alpha/beta/RC tags as GitHub pre-releases; publish a stable version such as `v1.0.0` as a normal release. In both cases, use the matching `CHANGELOG.md` notes.

Prompt the public Go proxy to discover the version and verify that the exact tag resolves:

```text
GOPROXY=https://proxy.golang.org go list -m github.com/zoster81/marksplice@v1.0.0
```

On PowerShell:

```powershell
$env:GOPROXY = 'https://proxy.golang.org'
go list -m github.com/zoster81/marksplice@v1.0.0
```

After proxy resolution succeeds, verify the matching package documentation, for example:

```text
https://pkg.go.dev/github.com/zoster81/marksplice@v1.0.0
```

Then compile and test a clean external consumer against the published version without a `replace` directive. Consumers can select the exact version explicitly:

```text
go get github.com/zoster81/marksplice@v1.0.0
```

or depend on it directly in `go.mod`:

```text
require github.com/zoster81/marksplice v1.0.0
```

Pre-release versions are not preferred over stable releases by default, so callers testing an alpha/beta/RC should specify that version explicitly.

## Subsequent releases

Use a new immutable semantic version for every published change. While the API remains under active v0 development, breaking API changes are permitted but must be called out in `CHANGELOG.md` and release notes. Starting with v1, compatible fixes and additions follow normal Semantic Versioning within the v1 line; an intentionally breaking public API change requires the next major version and the Go major-version module-path policy described above.

Typical progression examples are:

```text
v0.5.0-beta.1
v1.0.0-rc.1
v1.0.0
v1.0.1
v1.1.0
v2.0.0
```

The `v1.0.0` publication boundary requires the completed M124 stability review of the public API, compatibility policy, source-preservation guarantees, supported Go-version policy, rendering/workspace resource boundaries, and complete release-readiness evidence, followed by green public CI on both the M124 freeze and the exact release-state commit.
