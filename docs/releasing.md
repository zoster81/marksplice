# Release and Versioning Policy

Status: source of truth for public Marksplice module versioning and publication.

## Versioning

Marksplice is published as:

```text
github.com/zoster81/marksplice
```

The project follows Go module Semantic Versioning. The stable v1 line preserves backward compatibility: compatible fixes use patch releases, compatible public additions use minor releases, and intentionally breaking public API changes require the next major version plus the corresponding Go module-path decision.

The minimum supported Go version is the `go` directive in `go.mod`, currently Go 1.26. Public CI also tests the current Go 1.27 line. Release gates should use the latest stable patch available in both supported lines without raising the public floor merely to use the newest compiler.

A version that has been observed by Go module tooling must never be reused for different source bytes. If a publication needs correction after the Go proxy has cached it, publish a new semantic version and retract the affected version when appropriate.

## Current release line

- `v1.0.0` established the first stable public API contract.
- `v1.1.0` contains the compatible structured-editing expansion but is retracted because it reached Go module tooling before public release metadata and maintenance naming were fully corrected.
- `v1.1.1` is the supported v1.1 release and contains the same public runtime/API capability set with corrected public documentation and maintenance naming.

Future release targets belong in [`roadmap.md`](roadmap.md). A development boundary is not a release by itself; publication always requires an exact reviewed commit and green release gates.

## Public release readiness

Before every public release, verify that the repository accurately contains:

- canonical `go.mod`/`go.sum` state;
- package documentation in `doc.go`;
- compiled examples suitable for pkg.go.dev;
- README installation/current-version guidance;
- Apache-2.0 `LICENSE` and `NOTICE`;
- human-readable `CHANGELOG.md` release notes;
- private vulnerability reporting guidance in `SECURITY.md`;
- contributor and conformance documentation;
- CI across supported Go versions and major operating systems;
- dependency-update configuration.

Release notes and current documentation describe user-visible behavior. Internal milestone numbers, downstream consumer names, task IDs, private tool paths, and operator state are not public release metadata.

## Local verification

Run from the repository root on the exact commit intended for publication. At minimum:

```text
go mod tidy -diff
go mod verify
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
go build ./...
staticcheck ./...
golangci-lint run
gocyclo -over 15 -ignore '_test\.go$' .
unparam ./...
unparam -tests ./...
govulncheck ./...
gitleaks dir . --no-banner --redact
git diff --check
git status --short
```

Run the approved CommonMark/GFM conformance gates when their separately provisioned snapshots are available. Run the retained real-world corpus and performance/resource gates required by the change. Test supported Go versions and relevant cross-platform builds.

Do not claim a check passed unless it was actually executed. A skipped gate is reported with its reason.

## External-consumer verification

Before publication, test the exact checkout from a separate temporary module using a `replace` directive. Compile and run a small program importing `github.com/zoster81/marksplice` with each supported Go line.

After publication, repeat the consumer test against the actual tagged module without a `replace` directive.

Temporary consumer modules are maintainer tooling, not repository content.

## Publishing a version

1. Create the exact release-state commit locally only after local verification is green.
2. Fetch the remote and ensure `origin/main` has not advanced unexpectedly.
3. Push the release-state commit to `main`.
4. Wait for every GitHub Actions run associated with that exact commit to finish successfully.
5. Create an annotated semantic-version tag on that exact commit.
6. Push the tag.
7. Wait for every workflow associated with the tag/ref to finish successfully.
8. Create the GitHub Release using the matching human-readable `CHANGELOG.md` section.
9. Resolve the exact version through `proxy.golang.org`.
10. Verify package documentation on pkg.go.dev when available.
11. Compile/test a clean external consumer against the published version.

Example tag commands:

```text
git tag -a v1.1.1 -m "Marksplice v1.1.1"
git push origin v1.1.1
```

For prereleases, use an exact semantic prerelease such as `v1.2.0-rc.1` and mark the GitHub Release accordingly.

## Go proxy verification

After the tag and tag CI are green:

```text
GOPROXY=https://proxy.golang.org go list -m github.com/zoster81/marksplice@v1.1.1
```

On PowerShell:

```powershell
$env:GOPROXY = 'https://proxy.golang.org'
go list -m github.com/zoster81/marksplice@v1.1.1
```

Then verify the corresponding package documentation:

```text
https://pkg.go.dev/github.com/zoster81/marksplice@v1.1.1
```

## Retractions

Use a `retract` directive only when a published version should not be selected by users, for example because its release metadata, module state, or behavior is materially unsuitable for consumption.

A retraction is published in a later module version. Keep the reason short and user-facing. Do not encode internal task names or development history into the directive.

A retraction does not erase the original module version from proxies or caches. Consumers that request the exact retracted version can still resolve it; ordinary version selection and tooling can warn about the retraction once the newer module metadata is available.

## Repository settings

The public repository should retain:

- default branch `main`;
- GitHub Actions enabled;
- branch rules requiring the public CI checks;
- private vulnerability reporting;
- repository description/topics appropriate to Go, Markdown, GFM, structural editing, and source preservation.

Ordinary development work must not create or replace remotes, tags, releases, or other publication state without explicit authorization.

## Release history versus engineering history

`CHANGELOG.md` records public release changes. [`roadmap.md`](roadmap.md) records planned/current engineering boundaries. [`milestones/`](milestones/) preserves detailed historical implementation and verification evidence.

Do not copy milestone-by-milestone chronology into release notes, installation guides, API reference, package comments, or current architecture documentation.
