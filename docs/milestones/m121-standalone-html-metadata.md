# M121 — Standalone HTML and Metadata

Date: 2026-08-29
Status: Complete locally; unreleased pending milestone freeze commit, push, and exact remote CI closure

## Goal

M121 adds deterministic standalone HTML output on top of the M120 fragment renderer without introducing another Markdown renderer, a template engine, a YAML/TOML parser, an asset system, or new I/O authority.

The public surface is deliberately small:

- `Document.RenderHTMLDocument(io.Writer, HTMLDocumentOptions)` for streaming output;
- `Document.HTMLDocument(HTMLDocumentOptions)` for caller-owned complete bytes;
- `DefaultHTMLDocumentOptions()`;
- `HTMLMetadataFrontMatter` as the zero-value reviewed metadata policy;
- `HTMLMetadataOmit` for explicit metadata omission.

`HTMLDocumentOptions.Body` is exactly the existing `HTMLRenderOptions` contract.

## Requirements and edge cases

The standalone path emits deterministic doctype/html/head/UTF-8-charset/body markup, invokes the exact M120 fragment renderer, remains writer-oriented, and never buffers the complete document unless the caller chooses `HTMLDocument`. It fabricates no title from headings or filenames and performs no stylesheet injection, asset fetching, template execution, filesystem/network access, command execution, syntax highlighting, or mathematical execution.

Metadata is HTML-escaped and fail-closed. Duplicate, complex, nested, invalid-UTF-8, control-containing, escape-dependent, or unsupported values are omitted without rejecting an otherwise valid Markdown document.

## Architecture

`internal/renderhtml.RenderDocument` owns only the wrapper and delegates body output to `Render`. `internal/splice` projects metadata from the immutable snapshot because that layer already owns the source-proven front-matter nodes.

Projection is bounded to the recognized leading front-matter envelope and considers only existing promoted YAML/TOML field nodes plus four exact lower-case keys: `title`, `description`, `author`, and `lang`. No metadata lexer or parser is added. Plain simple scalar bytes are treated as text. Quoted values are admitted only when their source can be used without interpreting YAML/TOML escape syntax. `lang` also uses a conservative ASCII alphanumeric/hyphen token check.

## TDD and devil's-advocate review

The initial black-box RED failed only because the proposed M121 public API did not exist. The smallest GREEN proved wrapper output, streaming/buffered equivalence, M120 body-policy reuse, metadata omission, invalid-option handling, and writer-error propagation.

Hardening covers YAML/TOML fixed-key mapping, HTML escaping, TOML table scope, duplicate keys, escape-dependent quoted values, invalid language tokens, invalid UTF-8/control data, deterministic repeat output, and caller-source immutability.

The design review focused on four failure modes:

1. metadata silently becoming a YAML/TOML parser — prevented by consuming only already-promoted M106 fields and omitting escape-dependent values;
2. standalone rendering diverging from M120 — prevented by calling the same fragment renderer and retaining the exact parser/semantic/full-profile conformance gates;
3. metadata creating HTML injection — prevented by renderer text/attribute escaping and no raw metadata insertion;
4. convenience wrapping hiding whole-output buffering — prevented by a streaming primary API; only `HTMLDocument` owns a complete result buffer.

## Performance boundary

Five focused runs on the 256 KiB realistic workload with four reviewed metadata fields place streaming standalone rendering at roughly 41.72 MB/op and 258.4k allocations. Buffered `HTMLDocument` is roughly 43.82 MB/op at the same allocation-count scale because it accumulates the complete output. Wall-clock values remain host-sensitive engineering evidence.

## Conformance boundary

M121 defines no new Markdown-to-HTML dialect. The body remains governed by the M120 profile-aware expected-HTML gates against the official CommonMark 0.31.2 and published-GFM snapshots. Standalone wrapper and metadata behavior are separate Marksplice-owned export semantics.

## Exit boundary

M121 is locally complete only when its exact documented tree passes the applicable focused/full, race, static/complexity, security, supported-Go/cross-build, documentation/API, hygiene, and M120 renderer-conformance gates. Ordinary milestone commit/push and exact GitHub CI verification are required before M122 begins.
