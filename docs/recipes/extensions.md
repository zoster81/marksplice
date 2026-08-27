# Read-Only Extensions

Use `ParseWithOptions` when your application wants to observe product-specific syntax without changing Marksplice's core CommonMark/GFM model.

Run the complete example:

```sh
go run ./examples/extensions
```

It reads [`../../examples/extensions/sample.md`](../../examples/extensions/sample.md) and recognizes `[[configuration]]` as a namespaced `wikilink` observation.

## Define a recognizer

An extension has a namespace and a callback:

```go
wiki := marksplice.Extension{
    ID: "example.org/wikilink",
    Recognize: func(source marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
        // Inspect source.Text() and return exact non-empty ranges.
        return matches, nil
    },
}
```

The recognizer receives the exact parsed source as an immutable string view and returns only extension-local observations:

- `Kind` — extension-local semantic name;
- `Range` — exact non-empty byte range in the snapshot;
- `Attributes` — scalar metadata.

Extension kinds live outside the closed core `Kind` enum.

## Parse with explicit limits

```go
doc, err := marksplice.ParseWithOptions(source, marksplice.ParseOptions{
    Extensions: []marksplice.Extension{wiki},
    ExtensionLimits: marksplice.ExtensionLimits{
        MaxNodes:         32,
        MaxMetadataBytes: 4 << 10,
    },
})
```

When extensions are enabled, retention limits must be positive. Malformed ranges/metadata, duplicate namespaces, recognizer errors, recognizer panics, or exhausted retention budgets fail the complete operation with `ErrInvalidExtension`.

Zero options are equivalent to ordinary `Parse`.

## Read observations

```go
for _, node := range doc.ExtensionNodes() {
    raw, _ := doc.SourceRange(node.Range())
    target, _ := node.Attribute("target")
    fmt.Printf("%s -> %s\n", raw, target)
}
```

Extension nodes are immutable read-only observations attached to the parsed snapshot.

## What an extension cannot do through Marksplice

The extension SPI does not let an extension:

- replace or reclassify core nodes;
- register new core `Kind` values;
- prepare generic source patches or `ChangeSet` values;
- add builder syntax;
- change graph resolution semantics;
- access parser internals;
- gain filesystem, network, or command authority from Marksplice.

Recognizers are ordinary caller-linked Go code, however. Marksplice validates/bounds retained observations but cannot sandbox the recognizer's own CPU, memory, goroutines, or external I/O. Only register code you trust under your application's security model.

## When to keep a feature outside core

Product-specific spellings such as wikilinks, hashtags, custom tags, or private Markdown conventions normally belong in independent application/extension code. Broad document capabilities belong in core only when they can be defined without turning Marksplice into a collection of dialects.

The architecture rationale is in [`../extension-strategy.md`](../extension-strategy.md); normal extension users usually only need this recipe and the [API Reference](../api-reference.md#read-only-extensions).
