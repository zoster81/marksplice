# Render HTML Fragments

Use this workflow when you already have Markdown bytes and want deterministic HTML output. Rendering is separate from source-preserving editing: it does not mutate the parsed `Document` or replace your Markdown source.

## Stream HTML to a writer

`RenderHTML` is the primary API. It writes incrementally to the `io.Writer` you supply and stops on the first writer error.

```go
source, err := os.ReadFile("README.md")
if err != nil {
    return err
}

doc, err := marksplice.Parse(source)
if err != nil {
    return err
}

if err := doc.RenderHTML(os.Stdout, marksplice.DefaultHTMLRenderOptions()); err != nil {
    return err
}
```

The renderer performs no filesystem discovery, URL fetching, command execution, syntax highlighting, template execution, or math-engine execution. Link and image destinations are emitted as references only.

## Get caller-owned bytes

Use `HTML` when collecting the complete fragment in memory is more convenient:

```go
htmlFragment, err := doc.HTML(marksplice.DefaultHTMLRenderOptions())
if err != nil {
    return err
}
```

For large output or an HTTP/file pipeline, prefer `RenderHTML` so the HTML result itself does not need to be accumulated by Marksplice.

## Understand the default safety policy

The zero value of `HTMLRenderOptions` is the documented default:

```go
options := marksplice.HTMLRenderOptions{}
```

It:

- preserves parser-proven raw HTML;
- applies the published GFM disallowed-tag filter;
- suppresses dangerous URL schemes by emitting an empty destination.

Preserving raw HTML is **not** an HTML sanitization boundary. If Markdown is untrusted and the output will cross an HTML security boundary, choose the explicit escape policy or use an application-appropriate sanitizer after rendering.

```go
options := marksplice.HTMLRenderOptions{
    RawHTML: marksplice.HTMLRawEscape,
}
```

## Change policies explicitly

To preserve raw HTML without the GFM tag filter:

```go
options := marksplice.HTMLRenderOptions{
    TagFilter: marksplice.HTMLTagFilterDisabled,
}
```

To allow destinations that the default policy suppresses:

```go
options := marksplice.HTMLRenderOptions{
    UnsafeURLs: marksplice.HTMLUnsafeURLAllow,
}
```

`HTMLUnsafeURLAllow` is an explicit trust decision. Marksplice does not fetch the URL, but emitting a dangerous scheme can matter when another system later consumes the HTML.

## Output scope

M120 produces an **HTML fragment**, not a complete standalone document. There is no generated `<!doctype html>`, `<html>`, `<head>`, stylesheet, or template wrapper. Front matter and reference-definition declarations are semantic source metadata and are not emitted as visible HTML blocks.

Mathematical payload remains opaque data. Marksplice emits deterministic wrappers for the reviewed math forms but does not interpret or execute LaTeX, MathJax, KaTeX, or another math engine.

For exact option and method signatures, see the [API Reference](../api-reference.md). For the current renderer boundary, see [Capabilities](../capabilities.md).
