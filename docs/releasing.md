# Release and Versioning Policy

Status: source of truth for public Marksplice module versioning and release preparation.

## Versioning policy

Marksplice is a Go module at:

```text
github.com/zoster81/marksplice
```

The project follows Go module semantic-version conventions. During public API development, releases remain in the `v0` series and may use explicit alpha/beta/RC pre-release identifiers. A v0 or pre-release version carries no compatibility or stability guarantee.

The first public beta version is:

```text
v0.1.0-beta.1
```

The module path intentionally has no `/vN` suffix while the major version is v0 or v1. A future v2+ release would require the corresponding major-version module-path suffix and a separate architecture/release decision.

The minimum supported Go version is owned by the `go` directive in `go.mod`. The first beta targets Go 1.26 as the compatibility floor. Public CI also exercises Go 1.27.

Published tags are immutable. Never move, delete-and-recreate, or otherwise rewrite a version that has been made available to Go tooling. Publish a new version instead.

## Approved stable release targets

The approved post-M115 roadmap targets:

- `v1.0.0` after M124 completes the full API-stability, refactor, profiling, conformance, documentation, and release-readiness gate;
- `v1.5.0` after the deferred M125–M126 PDF backend/adapter line completes its own release gate.

M125–M126 must not delay v1.0. Reaching a milestone implementation boundary does not by itself publish a release: tags/releases are created only from the exact reviewed commit after the required local and GitHub Actions gates are green.

Official Go module references:

- https://go.dev/doc/modules/publishing
- https://go.dev/doc/modules/release-workflow
- https://go.dev/doc/modules/version-numbers

## Public release readiness

Before every public release, verify that the repository contains and accurately describes:

- `go.mod` and `go.sum` with the canonical module path;
- package documentation in `doc.go`;
- compiled package examples suitable for pkg.go.dev;
- `README.md` installation, beta-status, and minimum-Go-version guidance;
- Apache-2.0 `LICENSE` plus project `NOTICE`;
- `CHANGELOG.md` with the planned beta release notes;
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

## Publishing a new beta module version

The first public beta, `v0.1.0-beta.1`, is already published. For each later beta, after the release commit exists on the public `main` branch, wait for every GitHub Actions run associated with that exact commit to complete successfully. Do not create the release tag while any run is queued/in progress or if any run concludes unsuccessfully. This commit-level workflow gate is mandatory even when the same tree already passed the stricter local maintainer gate.

Only after all workflows for the exact release commit are green, create and push an annotated immutable tag. The examples below use `v0.1.0-beta.3`; replace it with the actual new version being released:

```text
git tag -a v0.1.0-beta.3 -m "Marksplice v0.1.0-beta.3"
git push origin v0.1.0-beta.3
```

Because the public CI also runs on tag pushes, wait for every GitHub Actions run associated with the tag's target commit/ref to complete successfully before creating the GitHub pre-release or advertising the module as published. A GitHub pre-release may then be created from that exact tag using the matching `CHANGELOG.md` notes.

Prompt the public Go proxy to discover the version and verify that the exact tag resolves:

```text
GOPROXY=https://proxy.golang.org go list -m github.com/zoster81/marksplice@v0.1.0-beta.3
```

On PowerShell:

```powershell
$env:GOPROXY = 'https://proxy.golang.org'
go list -m github.com/zoster81/marksplice@v0.1.0-beta.3
```

After proxy resolution succeeds, verify the package documentation at:

```text
https://pkg.go.dev/github.com/zoster81/marksplice@v0.1.0-beta.3
```

Consumers can then install the beta explicitly:

```text
go get github.com/zoster81/marksplice@v0.1.0-beta.3
```

or depend on it directly in `go.mod`:

```text
require github.com/zoster81/marksplice v0.1.0-beta.3
```

Because pre-release versions are not preferred over normal releases by default, callers should specify the beta version explicitly.

## Subsequent beta releases

Use a new immutable semantic version for every published change. While the API remains under active v0 development, breaking API changes are permitted but must be called out in `CHANGELOG.md` and release notes.

Typical progression examples are:

```text
v0.1.0-beta.1
v0.1.0-beta.2
v0.1.0-rc.1
v0.1.0
v0.2.0-beta.1
```

Do not publish v1 before the M124 stability gate has deliberately reviewed the public API, compatibility policy, source-preservation guarantees, supported Go-version policy, rendering/workspace resource boundaries, and complete release-readiness evidence.
