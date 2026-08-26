// Package marksplice provides structured GitHub Flavored Markdown creation and source-preserving manipulation.
//
// Marksplice is currently pre-v1 beta software under active development. Public
// APIs may change incompatibly between v0 releases until a stable v1 contract is
// explicitly published.
//
// Marksplice exposes reviewed new-document construction, snapshot-scoped structural
// views, copied bounded source reads, and named source-preserving mutations while
// keeping parser and lossless source-mapping implementation details internal.
//
// A successfully parsed Document and the immutable DocumentGraph, KnowledgeIndex,
// WorkspaceReport, and ChangeSet values derived from immutable snapshots may be read,
// queried, used for mutation planning, or applied concurrently. Public variable-length
// results are caller-owned unless an API explicitly documents otherwise. Callers must
// not concurrently mutate byte slices they pass as arguments to an operation.
//
// DocumentBuilder is mutable and is not safe for concurrent use without caller
// synchronization. Resolver callbacks supplied to graph/workspace builders are invoked
// synchronously during that call, are not invoked concurrently by Marksplice, and are
// never retained after the call returns.
//
// Core operations are synchronous and perform no implicit filesystem or network I/O.
// Structural queries require an explicit positive result limit; graph, workspace, and
// knowledge operations are bounded by the finite document collections supplied by the
// caller and never discover additional documents on their own.
//
// Public sentinel errors classify actionable failure families. Callers should use
// errors.Is rather than compare diagnostic error strings, whose wording is not a
// compatibility contract.
//
// Optional third-party extensions are explicit read-only semantic/source overlays evaluated
// only by ParseWithOptions after the ordinary core parse succeeds. Extension observations
// never enter the core Kind namespace or gain mutation/construction authority. Recognizers
// run synchronously as ordinary statically linked caller code and are not retained; Marksplice
// validates and bounds only the observations it retains and does not sandbox recognizer code.
package marksplice
