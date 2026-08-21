# Milestone M0 — Repository Bootstrap

Status: green — retrospective bootstrap record.

## Purpose

M0 records the repository-readiness baseline that existed before the public API milestones and alongside the first M1 feasibility work. It owns bootstrap evidence only: repository structure, legal attribution, module/dependency setup, contributor guidance, architecture/conformance sources of truth, parser-isolation scaffolding, and initial test discipline.

M0 does **not** claim a separate historical implementation phase or commit. The repository was created in root commit `f24b592` (`Bootstrap source-preserving GFM editing`), and that same commit already contained the first M1 source-preserving feasibility slice. This retrospective record separates the bootstrap obligations from M1's technical feasibility evidence without rewriting that history.

## Bootstrap contract

The bootstrap baseline requires:

- Go module path `github.com/zoster81/marksplice`;
- Apache-2.0 licensing through `LICENSE`;
- original-author attribution through `NOTICE` and repository metadata;
- Goldmark as a pinned Go module dependency, isolated behind `internal/parser/goldmark` rather than exposed through the public package;
- public repository guidance through `README.md`, `CONTRIBUTING.md`, and `AGENTS.md`;
- durable architecture and GFM-conformance sources of truth under `docs/`;
- repository hygiene for line endings, generated/build artifacts, local workspaces, editor state, and local reports through `.gitattributes` and `.gitignore`;
- package documentation through `doc.go`;
- focused tests present before broad implementation growth;
- no requirement for filesystem, network, MCP, or host-authorization behavior in Marksplice core.

Exact dependency versions remain owned by `go.mod` and `go.sum`. Licensing remains owned by `LICENSE` and `NOTICE`. Durable architecture and Markdown-profile decisions remain owned by `docs/architecture.md` and `docs/gfm-conformance.md` rather than by this milestone record.

## Historical evidence

Root commit `f24b592` created the initial repository in one atomic bootstrap commit. Its repository-level artifacts included:

- `.gitattributes` and `.gitignore`;
- `AGENTS.md`, `README.md`, and `CONTRIBUTING.md`;
- `LICENSE` and `NOTICE`;
- `doc.go`;
- `go.mod` and `go.sum`;
- `docs/architecture.md`, `docs/gfm-conformance.md`, and the Goldmark capability matrix;
- the M1 milestone record;
- the first internal parser/source/splice packages and their focused tests.

The initial commit therefore satisfied the repository bootstrap and began M1 at the same time. Later milestones refined the implementation and documentation but did not replace the M0 repository boundary.

## Current audit result

The post-M29 retrospective audit confirms that the bootstrap contract remains intact:

- the module path is still `github.com/zoster81/marksplice`;
- Goldmark is the only direct module dependency and its imports remain confined to `internal/parser/goldmark` plus its tests/conformance harness;
- Apache-2.0 and original-author attribution remain present;
- public documentation keeps private operator/tooling state out of the repository;
- `.gitattributes` enforces LF for tracked source/documentation classes while Marksplice runtime logic remains byte-preserving for input Markdown line endings;
- `.gitignore` excludes local build/test/coverage/workspace/editor/report artifacts without hiding tracked source-of-truth files;
- fuzz targets exist at the parser/source-map and patch boundaries established by M1;
- the repository remains a pure library boundary with no filesystem traversal, command execution, network access, or MCP host responsibilities in core.

## Exit decision

M0 is green as a repository bootstrap baseline. No production-code change is required by M0 itself. The correction is documentation traceability: bootstrap is now recorded separately from M1 while retaining the historical fact that both began in the same root commit.

The complete M0–M29 retrospective verification passes repeated full tests, race detection, coverage execution, vet, build, generated package documentation, the approved published-GFM conformance gate, Staticcheck, standard golangci-lint, production-only gocyclo/unparam, govulncheck, Gitleaks, text-format checks, `git diff --check`, and repository object-integrity checks.

Future bootstrap-policy changes belong in their durable source-of-truth documents and should update this record only when the historical/current bootstrap evidence itself changes.
