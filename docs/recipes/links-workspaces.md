# Links and Workspaces

Marksplice can reason about navigation and relationships while leaving filesystem, URL, and authorization policy to your application.

Run the complete multi-file example:

```sh
go run ./examples/workspace
```

It loads four Markdown files from [`../../examples/workspace/docs/`](../../examples/workspace/docs/), builds a graph, checks backlinks/reachability, and reports `troubleshooting.md` as unreachable from the configured root while all local links remain valid.

## Start with one document

### Heading anchors and fragments

```go
anchors := doc.HeadingAnchors()
target, status := doc.ResolveFragment("#configuration")
```

Heading anchors follow Marksplice's GitHub-compatible derivation and duplicate handling. Supported explicit HTML anchors also participate in fragment resolution.

Use `ValidateFragment` when you only need a boolean result.

### Table of contents

`GenerateTOC` creates deterministic Markdown from the existing section hierarchy.

`TOCStale` and `PrepareSyncTOC` are deliberately narrower: the caller must identify a managed section body, and Marksplice only synchronizes content that matches the conservative managed-TOC contract. Arbitrary lists or prose are never overwritten because they merely look like a TOC.

### Link relationships

```go
for _, relationship := range doc.LinkRelationships() {
    fmt.Println(relationship.Destination())
}
```

Relationships cover parser-resolved links, images, references, and autolinks in source order. The relationship view can be broader than ordinary editable link-node promotion; relationship intelligence does not grant a generic mutation span.

## Build a graph from documents you already loaded

Your application supplies documents and opaque keys:

```go
graph, err := marksplice.BuildDocumentGraph(
    []marksplice.GraphDocument{
        {Key: "index", Document: indexDoc},
        {Key: "guide", Document: guideDoc},
    },
    resolver,
)
```

The resolver decides whether a non-local relationship points to one of those already-supplied documents:

```go
func resolver(
    source marksplice.DocumentKey,
    relationship marksplice.LinkRelationship,
) (marksplice.DocumentResolution, bool) {
    // Apply your own path/URL/domain policy here.
    return marksplice.DocumentResolution{
        Target:   "guide",
        Fragment: "#configuration",
    }, true
}
```

Marksplice does not interpret `DocumentKey` as a path and does not open files or fetch URLs. A successful resolution must target a key already present in the explicit graph input.

Local `#fragment` destinations are self-relationships and do not require the resolver.

## Query the graph

Useful operations include:

- `Edges()` for all resolved relationships;
- `Outgoing(key)` for links from one document;
- `Backlinks(key)` for incoming links;
- `ReachableFrom(key)` for deterministic breadth-first reachability;
- `RelatedDocuments(key)` for unique direct neighbors.

The graph is immutable after construction.

## Validate an explicit workspace

`ValidateWorkspace` adds diagnostics over the same caller-controlled document set. Its resolver classifies each non-local relationship as ignored, resolved, or expected-but-missing.

```go
report, err := marksplice.ValidateWorkspace(
    documents,
    workspaceResolver,
    marksplice.WorkspaceValidationOptions{
        Roots: []marksplice.DocumentKey{"index"},
    },
)
```

Diagnostics can report:

- missing, ambiguous, or invalid fragments;
- caller-declared missing documents;
- conservative unresolved explicit references;
- documents unreachable from caller-provided roots;
- stale or unrecognized caller-managed TOCs.

The only automatic repair planning is for deterministic managed-TOC synchronization. Workspace validation does not crawl for files that were not supplied by the caller.

See [`examples/workspace/main.go`](../../examples/workspace/main.go) for a resolver that maps actual Markdown destinations to the files loaded by the example and uses caller-provided roots to surface an orphan-document diagnostic.

## Add syntax-independent knowledge metadata

If your application already has aliases, tags, or logical document relationships, layer them on the immutable graph:

```go
knowledge, err := marksplice.BuildKnowledgeIndex(graph, []marksplice.KnowledgeDocument{
    {
        Document: "guide",
        Aliases:  []marksplice.KnowledgeAlias{"start"},
        Tags:     []marksplice.KnowledgeTag{"docs"},
    },
})
```

Marksplice does not infer this data from wikilinks, filenames, front matter, hashtags, or URLs. It is caller-declared semantic metadata.

Knowledge queries include alias resolution, tags, logical references/backlinks, combined reachability, and combined related-document queries.

## Choosing the right layer

| Need | API |
| --- | --- |
| Resolve `#fragment` in one document | `ResolveFragment` |
| Enumerate semantic outgoing links | `LinkRelationships` |
| Backlinks/reachability across an explicit set | `BuildDocumentGraph` |
| Diagnose link/fragment/workspace state | `ValidateWorkspace` |
| Add application-owned aliases/tags/logical edges | `BuildKnowledgeIndex` |

For exact signatures, use the [API Reference](../api-reference.md#links-navigation-and-workspaces).
