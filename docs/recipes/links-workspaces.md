# Links and Workspaces

Marksplice can reason about navigation and relationships without hidden I/O. The root package stays in-memory; the optional `workspacefs` package performs read-only discovery only through an `fs.FS` explicitly supplied by your application.

Run the complete multi-file example:

```sh
go run ./examples/workspace
```

It gives `workspacefs` an `os.DirFS`, scans four Markdown files under [`../../examples/workspace/docs/`](../../examples/workspace/docs/), builds the existing graph, checks backlinks/reachability, and reports `troubleshooting.md` as unreachable from the configured root while all reviewed local links remain valid.

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

## Load a caller-authorized filesystem workspace

Use `workspacefs.Scan` when you want every Markdown document under one supplied filesystem root:

```go
workspace, err := workspacefs.Scan(
    os.DirFS("."),
    "docs",
    workspacefs.DefaultOptions(),
)
if err != nil {
    return err
}

graph, err := workspace.BuildGraph()
```

`Scan` discovers `.md` and `.markdown` files in deterministic slash-relative order. The returned document keys are paths relative to the supplied workspace root, such as `README.md` or `guide/setup.md`.

Use `workspacefs.Follow` instead when you want to start from explicit entry documents and load only reviewed local Markdown targets:

```go
workspace, err := workspacefs.Follow(
    os.DirFS("."),
    "docs",
    []string{"README.md"},
    workspacefs.DefaultOptions(),
)
```

Both operations are read-only and require finite document, byte, depth, and relationship limits. `Scan` interprets `MaxDepth` as directory depth below the workspace root; `Follow` interprets it as relationship-hop depth. Budget exhaustion fails with `workspacefs.ErrBudgetExceeded`.

`Follow` treats a relationship destination as a URI reference and uses only its path component for filesystem lookup. Literal relative `.` and `..` segments are normalized against the source document and must remain inside the supplied `fs.FS` namespace. Each path component is percent-decoded exactly once, so `docs/My%20Guide.md` can address `docs/My Guide.md`, while encoded traversal such as `%2e%2e/secret.md` and encoded separators such as `docs%2Fsecret.md` are rejected. Query text does not affect lookup: `guide/setup.md?print=1#install` looks up `guide/setup.md` and preserves `#install` for ordinary fragment resolution.

Absolute paths, URI schemes, protocol-relative targets, raw backslashes, empty path segments such as `docs//guide.md`, malformed encoding, directory targets, and extensionless targets are not followed. Marksplice does not invent `index.md`, append extensions, or rewrite case. Case sensitivity, symlink behavior, and other host semantics remain properties of the `fs.FS` supplied by your application. The adapter never fetches URLs, executes commands, or writes files.

Once loaded, `Workspace.BuildGraph` and `Workspace.Validate` delegate to the same immutable graph and validation APIs described below.

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

The only automatic repair planning is for deterministic managed-TOC synchronization. Root `ValidateWorkspace` does not discover files beyond the caller-provided document set; `workspacefs.Workspace.Validate` validates exactly the finite set loaded by its preceding `Scan` or `Follow` operation.

See [`examples/workspace/main.go`](../../examples/workspace/main.go) for the complete `workspacefs.Scan` flow and a caller-provided root that surfaces an orphan-document diagnostic.

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
| Discover all Markdown under an explicit `fs.FS` root | `workspacefs.Scan` |
| Follow local Markdown from explicit entries | `workspacefs.Follow` |
| Backlinks/reachability across an explicit set | `BuildDocumentGraph` or `workspacefs.Workspace.BuildGraph` |
| Diagnose link/fragment/workspace state | `ValidateWorkspace` |
| Add application-owned aliases/tags/logical edges | `BuildKnowledgeIndex` |

For exact signatures, use the [API Reference](../api-reference.md#links-navigation-and-workspaces).
