# Marksplice Examples

These programs use tracked Markdown files instead of embedding toy documents in Go strings. Run them from the repository root so their fixture paths resolve consistently.

| Example | What it demonstrates | Run |
| --- | --- | --- |
| `inspect` | Load a realistic Markdown file and inspect sections, tasks, fenced blocks, and links | `go run ./examples/inspect` |
| `edit` | Prepare four independent source-preserving edits, compose them atomically, and leave the fixture untouched | `go run ./examples/edit` |
| `build` | Create a release brief from structured builder input | `go run ./examples/build` |
| `query` | Find unfinished tasks inside one section with a bounded structural query | `go run ./examples/query` |
| `workspace` | Build a graph from four documentation files, inspect backlinks/reachability, and report an orphan document | `go run ./examples/workspace` |
| `extensions` | Add a small read-only `[[wikilink]]` observation without changing the core Markdown grammar | `go run ./examples/extensions` |

The `edit` program reads `examples/edit/release-plan.md` but never writes to it. The workspace example keeps all local links valid but leaves `troubleshooting.md` unreachable from the configured root so `ValidateWorkspace` reports a real orphan-document diagnostic.

The smaller examples in the root `example_test.go` remain useful for pkg.go.dev. This directory is the practical, file-based learning path used by the public documentation.
