# Recipes

Recipes are short, goal-oriented paths through the public API. They assume you already know the basic `Document` / `ChangeSet` / `DocumentBuilder` split from [Getting Started](../getting-started.md).

| Recipe | Use it when... | Full example |
| --- | --- | --- |
| [Inspect a document](inspect-document.md) | you need headings, tasks, sections, fenced blocks, metadata, links, or other parsed facts | [`examples/inspect`](../../examples/inspect/) |
| [Edit an existing document](edit-existing-document.md) | you need minimal source-preserving changes without reformatting unrelated source | [`examples/edit`](../../examples/edit/) |
| [Create a document](create-document.md) | you are generating new Markdown from structured application data | [`examples/build`](../../examples/build/) |
| [Render HTML](render-html.md) | you need deterministic fragments/standalone HTML or optional Markdown-to-output byte mapping with explicit body and metadata policy | [`examples/render`](../../examples/render/) |
| [Lists, sections, and tables](lists-sections-tables.md) | you need hierarchy-aware queries or structural mutation | [`examples/query`](../../examples/query/) |
| [Links and workspaces](links-workspaces.md) | you need explicit `fs.FS` discovery/following, fragments, backlinks, reachability, validation, or cross-document relationships | [`examples/workspace`](../../examples/workspace/) |
| [Read-only extensions](extensions.md) | your application wants to recognize its own syntax without changing Marksplice core | [`examples/extensions`](../../examples/extensions/) |

For exact signatures, use the [API Reference](../api-reference.md). For the complete support boundary, use [Capabilities](../capabilities.md).
