# Render Canonical Markdown

Use this workflow when you want an explicit deterministic Markdown export from an existing parsed document. Canonical rendering is intentionally different from source-preserving editing: it produces a new normalized representation and never mutates the parsed `Document` or rewrites your original bytes.

## Stream canonical Markdown

`RenderCanonicalMarkdown` is the primary API. It writes to the `io.Writer` you supply and stops on the first writer error.

```go
source, err := os.ReadFile("README.md")
if err != nil {
    return err
}

doc, err := marksplice.Parse(source)
if err != nil {
    return err
}

if err := doc.RenderCanonicalMarkdown(os.Stdout); err != nil {
    return err
}
```

Use `CanonicalMarkdown` when caller-owned complete bytes are more convenient:

```go
canonical, err := doc.CanonicalMarkdown()
```

For large output or a file/network pipeline owned by your application, prefer the writer form.

Run the tracked file-based example from the repository root:

```sh
go run ./examples/render --markdown
```

It reads `examples/render/page.md`, writes canonical Markdown to standard output, and leaves the fixture unchanged.

## What canonical means

The renderer consumes the same Native semantic walk used by the HTML path. It does not parse Markdown delimiters a second time and it does not retain a second AST in `Document`.

The output uses one Marksplice-owned deterministic profile rather than caller-selectable formatting styles. Among other choices, it uses LF line endings, deterministic block spacing, canonical list/table/fence/reference forms where needed, and syntax that preserves the semantic facts exposed by the Native walk. Opaque payload such as code or raw HTML is preserved according to its semantic container instead of being interpreted as another language.

The contract is semantic, not source-form preserving:

```text
Parse(original) -> semantic A
semantic A -> canonical Markdown
Parse(canonical) -> semantic B
A == B
```

Rendering the canonical result again is byte-idempotent:

```text
canonical(parse(canonical(x))) == canonical(x)
```

Those properties are covered across the complete approved CommonMark 0.31.2 and published-GFM example sets plus the retained real-world regression corpus.

## Keep canonical export separate from editing

Do not use canonical rendering when your goal is a narrow edit that should preserve author formatting. Use the ordinary `Prepare...` APIs and `ChangeSet.Apply` for that job. Those operations patch only source ranges they own and preserve unrelated bytes.

Canonical rendering is appropriate when normalization itself is the requested output, for example:

- deterministic generated artifacts;
- semantic round-trip checks;
- stable comparison/snapshot output;
- explicit format normalization chosen by the caller.

The renderer performs no filesystem discovery, URL fetching, asset loading, command execution, template execution, syntax highlighting, or mathematical-engine execution.

For exact method signatures, see the [API Reference](../api-reference.md). For the current rendering boundary, see [Capabilities](../capabilities.md).
