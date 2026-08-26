# M107 — Knowledge-document primitives

Status: complete.

## Objective

Add broadly useful knowledge-document semantics without adding a Markdown dialect, metadata schema, workspace crawler, or second document graph.

M107 is deliberately syntax-independent. Callers may attach exact aliases, tags, and logical document references to documents that are already present in an authorized M100 `DocumentGraph`. Marksplice does not infer these values from wikilinks, hashtags, front matter, headings, filenames, paths, URLs, or any other source spelling.

## Requirements and edge cases

The reviewed contract requires:

- one existing M100 `DocumentGraph` as the complete document/authority boundary;
- metadata may cover any subset of graph documents;
- aliases are exact, case-sensitive, non-empty valid UTF-8 scalar values without NUL/CR/LF;
- aliases are globally unique and may not collide exactly with any canonical `DocumentKey` already in the graph;
- tags are exact, case-sensitive, non-empty valid UTF-8 scalar values without NUL/CR/LF;
- duplicate tags on one document fail closed;
- logical references point directly to existing `DocumentKey` values and therefore require no resolver or alias interpretation;
- duplicate logical references from one source to the same target fail closed;
- self logical references are valid but do not make reachability/related-document output include the source itself;
- metadata input order must not change graph-document ordering semantics;
- an M100 Markdown edge and M107 logical reference may point to the same target, but combined reachability/related queries must not duplicate that document;
- every variable-length public result is caller-owned;
- malformed input fails with `ErrInvalidKnowledge`;
- no source mutation, parser callback, filesystem/network access, command execution, hidden document discovery, or syntax normalization is introduced.

`DocumentKey` remains opaque caller data. M107 does not make it a path, URI, filename, or durable source-derived identity.

## Architecture and public API

M107 adds a separate immutable overlay:

```go
type KnowledgeAlias string
type KnowledgeTag string

type KnowledgeDocument struct {
    Document   DocumentKey
    Aliases    []KnowledgeAlias
    Tags       []KnowledgeTag
    References []DocumentKey
}

type KnowledgeReference struct { /* immutable */ }
func (r KnowledgeReference) SourceDocument() DocumentKey
func (r KnowledgeReference) TargetDocument() DocumentKey

type KnowledgeIndex struct { /* immutable */ }
func BuildKnowledgeIndex(graph *DocumentGraph, documents []KnowledgeDocument) (*KnowledgeIndex, error)
func (k *KnowledgeIndex) Aliases(key DocumentKey) ([]KnowledgeAlias, bool)
func (k *KnowledgeIndex) Tags(key DocumentKey) ([]KnowledgeTag, bool)
func (k *KnowledgeIndex) ResolveAlias(alias KnowledgeAlias) (DocumentKey, bool)
func (k *KnowledgeIndex) DocumentsWithTag(tag KnowledgeTag) []DocumentKey
func (k *KnowledgeIndex) References() []KnowledgeReference
func (k *KnowledgeIndex) ReferencesFrom(key DocumentKey) ([]KnowledgeReference, bool)
func (k *KnowledgeIndex) ReferencedBy(key DocumentKey) ([]KnowledgeReference, bool)
func (k *KnowledgeIndex) ReachableFrom(key DocumentKey) ([]DocumentKey, bool)
func (k *KnowledgeIndex) RelatedDocuments(key DocumentKey) ([]DocumentKey, bool)
```

`ErrInvalidKnowledge` is the fail-closed public sentinel for malformed M107 input.

The overlay retains the supplied immutable M100 graph pointer instead of copying documents, M99 relationships, or graph edges. Per-document state retains only copied alias/tag slices. Logical references are stored once in deterministic graph-document order, with outgoing/backlink maps containing integer reference indices, matching M100's compact adjacency pattern.

The build-time unique-alias map becomes the retained exact alias lookup, so `ResolveAlias` is expected O(1) rather than rescanning every document. Input logical-reference target slices are validation/build temporaries and are not retained a second time after the authoritative `KnowledgeReference` vector is built.

M107 reuses private M100 document-membership, BFS append, neighbor-marking, and graph-order projection helpers. This keeps graph and knowledge traversal behavior aligned without creating a second graph abstraction or duplicating traversal machinery.

## Query and ordering semantics

Ordering remains deterministic:

- aliases/tags preserve caller order within the owning document;
- `DocumentsWithTag` returns matching documents in original M100 graph-input order;
- `References` is graph-document ordered, then caller reference order within each document, independent of the order of `KnowledgeDocument` inputs;
- `ReferencesFrom` and `ReferencedBy` preserve the authoritative reference-vector order;
- combined `ReachableFrom` is BFS; for each visited document, existing M100 Markdown outgoing edges are considered before M107 logical references;
- combined `RelatedDocuments` returns unique direct incoming/outgoing neighbors in original M100 graph-input order;
- self edges/references and duplicate targets across M100/M107 are deduplicated only in document-set queries, not by mutating either underlying relationship vector.

M107 does not merge logical references into `DocumentGraph.Edges()`. Markdown relationships remain M100/M99 facts with source metadata; caller-declared logical references remain a separate syntax-independent relationship family with no fabricated source offset.

## Scope decisions

M107 intentionally does **not** add:

- wikilink, hashtag, definition-list, heading-attribute, fenced-container, emoji, Obsidian, Discord, or other dialect syntax;
- automatic extraction of aliases/tags from YAML/TOML front matter;
- alias-based target resolution for logical references;
- arbitrary key/value metadata or an application schema;
- free-form relationship attributes/types whose semantics Marksplice cannot define generally yet;
- mutation APIs that write aliases/tags/references back into Markdown;
- another structural `Kind`, parser observation, AST, or persisted workspace model.

Those omissions keep the M110 third-party syntax boundary clean and leave M109 free to review the public API before stabilization.

## Performance and retained state

Let:

- `V` = graph documents;
- `E` = existing M100 graph edges;
- `A` = aliases;
- `T` = tags;
- `L` = logical references.

Build work is expected O(V + A + T + L), using the existing graph document map for membership checks. Persistent M107 state is O(A + T + L) plus one alias lookup and reference adjacency indices; it holds one pointer to the immutable M100 graph and does not duplicate `V` document snapshots or `E` graph edges.

Query costs are:

- alias lookup: expected O(1);
- aliases/tags for one document: O(result) defensive copy;
- tag query: O(V + T) worst-case scan, retaining no second tag-to-document index before M108 measurement justifies one;
- references/outgoing/backlinks: O(result) copy;
- combined reachability: O(V + E + L) worst case;
- combined related documents: O(V + incident E + incident L).

There are no hidden document/reference/tag limits. Resource use remains bounded by the explicit finite caller graph and metadata input. M108 owns measurement-driven budgets and any benchmark-justified indexing changes.

## Devil's advocate review

### Risk 1 — M107 becomes a second graph that can drift from M100

Mitigation: `KnowledgeIndex` retains the authoritative immutable `DocumentGraph`; it does not copy documents or Markdown edges. Logical references remain one separate vector plus index adjacency, and combined traversal directly reuses M100 internals/helpers.

### Risk 2 — aliases accidentally become path/syntax resolution authority

Mitigation: aliases are exact opaque scalar labels only. They never resolve Markdown destinations, filenames, paths, URLs, or logical-reference targets. References name existing `DocumentKey` values directly.

### Risk 3 — retained state duplicates explicit input unnecessarily

The first implementation retained each logical target both inside per-document state and again as `KnowledgeReference` values, and `ResolveAlias` rescanned all aliases. Refactor removes retained target duplication and preserves the already-built alias-owner map as the lookup index.

### Risk 4 — combined graph traversal becomes nondeterministic or duplicates targets

Mitigation: BFS uses the same visited-before-enqueue helper as M100, considers M100 adjacency before M107 adjacency for each source, and projects related-document sets in authoritative graph order. Focused tests cover a target present through both relationship families.

### Risk 5 — generic metadata turns core into a schema system

Mitigation: M107 supports only the reviewed general primitives. It does not add arbitrary maps, typed application properties, source extraction, or serialization. Relationship attributes are deferred rather than inventing semantics merely for symmetry.

## Verification

M107 was developed through focused RED/GREEN tests for exact alias/tag/reference validation, deterministic graph-document ordering, caller-owned results, M100+M107 traversal union/deduplication, retained-state minimization, and graph/knowledge complexity boundaries.

The documented tree passed repeated complete regressions, actual race detection, the executable knowledge-index example and API documentation checks, `go vet`, `go build`, Staticcheck, golangci-lint, production `gocyclo <= 15`, production/test-inclusive unparam, `go mod tidy -diff`, published GFM 0.29 conformance, Go 1.27 test/vet/build, govulncheck, Gitleaks, actionlint, strict text/artifact hygiene, and `git diff --check`.

Cross-package statement coverage measured **86.9% aggregate** and **84.6% through `internal/publictest`**.

## Exit decision

M107 is complete. The syntax-independent knowledge overlay reuses the authoritative M100 graph, keeps logical/source-backed relationships separate, and has passed implementation/refactor/source-of-truth/release-quality verification. The next roadmap boundary is **M108 — Fuzzing, pathological input, and performance hardening**.
