# M90 — First Public Beta Readiness

## Status

Complete and green for internal first-public-beta readiness. Publication has not been performed.

## Objective

Prepare `github.com/zoster81/marksplice` for its first public Go-module beta without adding release state, network behavior, filesystem authority, or other publication concerns to the runtime library.

## Release contract

The planned first public module version is:

```text
v0.1.0-beta.1
```

Marksplice remains explicitly pre-v1 and under active API development. The initial compatibility floor remains the `go 1.26` directive in `go.mod`; public CI also exercises Go 1.27. The module path remains `github.com/zoster81/marksplice` for v0/v1.

Published Git tags are immutable. Runtime code does not expose a hard-coded release version; the module tag is the release-version authority.

M90 prepares publication but does not itself authorize or perform a commit, remote creation, push, tag, GitHub release, Go-proxy publication, or pkg.go.dev publication.

## Repository readiness

M90 adds or aligns:

- `.github/workflows/ci.yml` with read-only permissions, concurrency cancellation, Go 1.26/1.27 testing across Linux/Windows/macOS, tidy/vet/race quality checks, and immutable SHA pins for `actions/checkout` v6.0.2 and `actions/setup-go` v7.0.0;
- `.github/dependabot.yml` for weekly Go-module and GitHub Actions dependency updates;
- executable package examples in `example_test.go` for builder construction, parsing/source reading, and source-preserving heading mutation;
- `SECURITY.md` with private vulnerability-reporting guidance appropriate to an active beta;
- `CHANGELOG.md` with a planned `v0.1.0-beta.1` section whose release date remains unset until publication;
- `docs/releasing.md` as the release/versioning/first-push source of truth;
- README beta warning, minimum-Go policy, installation/`require` examples, quick start, and release/security documentation links;
- `CONTRIBUTING.md` supported-Go and publication guidance;
- `.gitattributes` LF policy for YAML workflow/configuration files.

The hash-pinned published GFM corpus remains externally provisioned because its upstream licensing differs from the repository license. Public CI therefore runs the ordinary repository tests, while the separately provisioned conformance gate remains part of the maintainer completion/release gate.

## External consumer strategy

Before publication, a temporary module outside the repository must import `github.com/zoster81/marksplice` using a local `replace` directive and compile/test representative public APIs. The temporary consumer lives only under the private Marksplice tool root and is deleted after verification.

After the tag is actually pushed, the same version must be verified through the public Go proxy and pkg.go.dev using the immutable `v0.1.0-beta.1` tag.

## Requirements and edge cases

- The public README must not imply the beta already exists before the tag is published; installation text therefore states that commands apply after the first public beta tag is published.
- The release documentation must distinguish preparation from authorization; documented push/tag commands are not executed automatically.
- Public CI must not depend on private machine paths, private tool inventory, or the privately retained GFM corpus.
- Release metadata must not introduce a second version authority that can drift from Go module tags.
- The Go compatibility floor should maximize community usability while remaining actually tested; the first beta keeps Go 1.26 and tests Go 1.27 in public CI.
- Public package examples must compile and reflect canonical typed-inline escaping rather than promising source bytes that the construction contract intentionally escapes.

## Devil's advocate review

1. **A hard-coded runtime version could drift from the immutable Go module tag.** M90 adds no runtime version constant; Git tags remain authoritative.
2. **Public CI could accidentally rely on private validation files.** The public workflow contains no private path and documents that the separately licensed GFM corpus is an external maintainer gate.
3. **A beta README could mislead users into assuming v1 compatibility.** The README, changelog, release policy, and contributing guide explicitly state active pre-v1 instability.
4. **A first push could expose secrets or machine-specific files.** Gitleaks, text/path review, Git status, `.gitignore`, and the final strict gate remain mandatory before publication.
5. **A module may build in-repo but fail when consumed externally.** M90 requires a separate consumer-module `replace` test before publication and proxy/pkg.go.dev verification after the public tag exists.
6. **CI metadata could be syntactically invalid despite repository tests passing.** Workflow/configuration syntax is independently validated during final readiness verification.

## Verification

The external consumer proof passed from a temporary separate Go 1.26 module using the canonical `require github.com/zoster81/marksplice v0.0.0` plus a local `replace` to the unpublished checkout. That consumer compiled and tested M89 reference construction and ordinary `Parse`, and the temporary module was deleted afterward.

The public workflow passed `actionlint` v1.7.12. The selected official Actions releases were resolved to immutable public tag SHAs before pinning. The repository also passed direct Go 1.27.0 `go test ./...` and `go vet ./...` in addition to the Go 1.26.6 maintainer gate.

The combined M89–M90 strict gate passed five full test runs, race detection, coverage, vet/build, package/examples/new-API documentation checks, hash-pinned published GFM 0.29 conformance, Staticcheck, golangci-lint with zero issues, production complexity <=15, production/test `unparam`, `govulncheck` with no vulnerabilities found, Gitleaks with no leaks, `actionlint`, `go mod tidy -diff`, Go 1.27 compatibility, strict UTF-8/no-BOM/LF/no-trailing-whitespace hygiene over 251 repository text files, public/private path-boundary scanning, `git diff --check`, and `git fsck --no-dangling`.

Final statement coverage is 92.3% for the root package, 73.6% for `internal/parser/goldmark`, 79.2% for `internal/source`, 57.0% for `internal/splice`, and 71.8% aggregate. The parser interface package reports 0.0% because it has no executable test target.

The dry-run first-commit inventory contains only the expected M63–M90/public-readiness files. At verification time the repository remained on `main` at HEAD `352d094fe6ada53b0d9c4c417dc36bd633642692`, with no configured remotes and no tags. No commit, push, repository creation, release, proxy publication, or pkg.go.dev publication was performed.

## Exit decision

M90 is complete: the repository is internally ready for a reviewed first public consolidation commit/push and subsequent `v0.1.0-beta.1` tag. The remaining publication actions are explicitly separated by authorization and were not performed by this milestone.
