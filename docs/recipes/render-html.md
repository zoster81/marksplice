# Render HTML

Use this workflow when you already have Markdown bytes and want deterministic HTML output. Rendering is separate from source-preserving editing: it does not mutate the parsed `Document` or replace your Markdown source.

## Stream an HTML fragment

`RenderHTML` is the primary fragment API. It writes incrementally to the `io.Writer` you supply and stops on the first writer error.

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

Use `HTML` when collecting the complete fragment in memory is more convenient:

```go
fragment, err := doc.HTML(marksplice.DefaultHTMLRenderOptions())
```

For large output or an HTTP/file pipeline, prefer the writer form.

## Render a standalone HTML document

`RenderHTMLDocument` writes a deterministic wrapper around the exact same fragment renderer:

```go
options := marksplice.DefaultHTMLDocumentOptions()
if err := doc.RenderHTMLDocument(os.Stdout, options); err != nil {
    return err
}
```

The generated shape is deliberately small: `<!doctype html>`, `<html>`, `<head>`, UTF-8 charset metadata, optional reviewed metadata, and `<body>`. There is no template engine, stylesheet injection, asset manager, base URL, script injection, or network behavior.

Use `HTMLDocument` when caller-owned complete bytes are more convenient:

```go
page, err := doc.HTMLDocument(marksplice.DefaultHTMLDocumentOptions())
```

Run the file-based example from the repository root:

```sh
go run ./examples/render
```

It reads `examples/render/page.md` and streams a complete HTML document without modifying the fixture.

## Map Markdown source ranges to HTML output

Use the `...WithSourceMap` variants when an editor, preview, or inspection tool needs to correlate the parsed Markdown snapshot with the exact HTML produced by the same render:

```go
html, mappings, err := doc.HTMLDocumentWithSourceMap(
    marksplice.DefaultHTMLDocumentOptions(),
)
if err != nil {
    return err
}

for _, mapping := range mappings {
    sourceRange := mapping.SourceRange()
    outputRange := mapping.OutputRange()

    markdownBytes := source[sourceRange.Start:sourceRange.End]
    htmlBytes := html[outputRange.Start:outputRange.End]
    fmt.Printf("%q -> %q\n", markdownBytes, htmlBytes)
}
```

For streaming output, use `RenderHTMLWithSourceMap` or `RenderHTMLDocumentWithSourceMap`. Mapping is opt-in: ordinary `RenderHTML`/`RenderHTMLDocument` do not retain the source-map result.

Both sides use half-open **byte** ranges. A source map is not total byte coverage. It describes emitted semantic events, so:

- nested constructs can intentionally overlap, such as emphasis and its text child;
- synthetic HTML such as fragment tags or the standalone doctype/body wrapper can be unmapped;
- source declarations that emit no visible fragment bytes do not receive fabricated output ranges;
- standalone metadata fields map to the exact `<title>`, `<meta>`, or `lang` bytes they emit;
- standalone output offsets are absolute from byte zero of the complete HTML document;
- no `NodeID` is stored in the map, because source/output offsets belong only to that exact snapshot/result pair.

Results are sorted by output position, with an outer mapping before a nested mapping that begins at the same output byte. A render error returns no map. The writer may already contain partial HTML, so discard both partial output and mappings when an error is returned.

Run the same tracked example in mapping mode:

```sh
go run ./examples/render --map
```

## Front-matter metadata mapping

The standalone zero value uses `HTMLMetadataFrontMatter`. Mapping is intentionally narrow and case-sensitive. Only these exact lower-case keys are recognized:

| Front-matter key | HTML output |
| --- | --- |
| `title` | `<title>...</title>` |
| `description` | `<meta name="description" content="...">` |
| `author` | `<meta name="author" content="...">` |
| `lang` | `lang="..."` on `<html>` when the value is a conservative ASCII language token |

A field is eligible only when the existing Marksplice front-matter model already proves it as one unique top-level simple scalar. Duplicate keys, complex YAML, TOML values after table scope, invalid UTF-8, control bytes, and quoted values that require YAML/TOML escape interpretation are omitted rather than guessed. Marksplice does not add a YAML/TOML parser for HTML export.

The scalar bytes are treated as text and HTML-escaped. They are not inserted as raw markup.

Disable all front-matter-derived metadata explicitly:

```go
options := marksplice.HTMLDocumentOptions{
    Metadata: marksplice.HTMLMetadataOmit,
}
```

No heading or filename is fabricated as a fallback `<title>`.

## Body safety policy

The zero value of `HTMLRenderOptions`:

- preserves parser-proven raw HTML;
- applies the published GFM disallowed-tag filter;
- suppresses dangerous URL schemes by emitting an empty destination.

Standalone rendering reuses those exact policies through `HTMLDocumentOptions.Body`:

```go
options := marksplice.HTMLDocumentOptions{
    Body: marksplice.HTMLRenderOptions{
        RawHTML: marksplice.HTMLRawEscape,
    },
}
```

Preserving raw HTML is **not** an HTML sanitization boundary. If Markdown is untrusted and the output crosses an HTML security boundary, choose `HTMLRawEscape` or use an application-appropriate sanitizer after rendering.

`HTMLUnsafeURLAllow` remains an explicit trust decision. Marksplice emits references only; it never fetches URLs or images.

## Output and authority boundaries

Fragments contain only rendered Markdown body output. Standalone documents add only the deterministic wrapper and reviewed metadata described above. Front-matter declarations and reference-definition declarations are not emitted as visible body blocks.

Mathematical payload remains opaque data. Marksplice does not interpret or execute LaTeX, MathJax, KaTeX, fenced code, templates, or embedded languages. Rendering performs no filesystem discovery, asset fetching, network access, command execution, or syntax highlighting.

For exact option and method signatures, see the [API Reference](../api-reference.md). For the current renderer boundary, see [Capabilities](../capabilities.md).
