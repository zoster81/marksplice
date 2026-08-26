package marksplice

import (
	"errors"
	"fmt"
	"strings"

	"github.com/zoster81/marksplice/internal/splice"
)

var (
	// ErrNodeNotFound reports that a snapshot-local node ID does not exist.
	ErrNodeNotFound = errors.New("node not found")
	// ErrInvalidReplacement reports a requested mutation that cannot preserve the required structure.
	ErrInvalidReplacement = errors.New("invalid replacement")
	// ErrInvalidTargetKind reports that an operation does not support the targeted node kind.
	ErrInvalidTargetKind = errors.New("invalid target kind")
	// ErrSourceConflict reports that a prepared change was applied to a different source snapshot.
	ErrSourceConflict = errors.New("source snapshot conflict")
	// ErrInvalidConstruction reports new-document content that cannot be proven to produce the requested GFM structure.
	ErrInvalidConstruction = errors.New("invalid construction")
	// ErrInvalidQuery reports an unbounded or malformed structural query.
	ErrInvalidQuery = errors.New("invalid query")
	// ErrInvalidGraph reports malformed explicit document-graph input or resolution.
	ErrInvalidGraph = errors.New("invalid document graph")
	// ErrInvalidWorkspace reports malformed workspace validation authority or targets.
	ErrInvalidWorkspace = errors.New("invalid workspace validation")
	// ErrInvalidKnowledge reports malformed syntax-independent knowledge metadata or references.
	ErrInvalidKnowledge = errors.New("invalid knowledge index")
	// ErrInvalidExtension reports malformed third-party extension configuration or observations.
	ErrInvalidExtension = errors.New("invalid extension")
)

// Kind identifies a Marksplice core structural Markdown node category.
// Third-party extension identities are intentionally outside this core enum.
type Kind uint8

const (
	KindUnknown Kind = iota
	KindParagraph
	KindHeading
	KindListItem
	KindTask
	KindTableCell
	KindFencedCode
	KindStrikethrough
	KindCodeSpan
	KindEmphasis
	KindStrong
	KindInlineLink
	KindReferenceDefinition
	KindAutoLink
	KindFrontMatterField
	KindHTMLComment
	KindHTMLAnchor
	KindImage
	KindTableRow
	KindTable
	KindThematicBreak
	KindBlockquote
	KindFootnoteDefinition
	KindMathExpression
)

// NodeID identifies a node within one parsed source snapshot.
//
// Node IDs are deterministic for a snapshot, but they are not durable identities
// across arbitrary source changes or reparses. The representation is opaque;
// String is for diagnostics and does not define a persistence or round-trip format.
type NodeID struct {
	value string
}

// String returns a diagnostic representation of the snapshot-scoped ID.
func (id NodeID) String() string {
	return id.value
}

// Range is a half-open byte range [Start, End) in a source snapshot.
// The accessor returning a Range defines the semantic meaning of that span.
type Range struct {
	Start int
	End   int
}

// Valid reports whether r is ordered and contained in a source of total bytes.
func (r Range) Valid(total int) bool {
	return r.Start >= 0 && r.End >= r.Start && r.End <= total
}

// Node is an immutable public summary of one promoted structural node.
//
// Syntax-specific details and source ranges are intentionally not part of this
// common value until their public semantics are reviewed.
type Node struct {
	id   NodeID
	kind Kind
}

// ID returns the snapshot-scoped node identity.
func (n Node) ID() NodeID {
	return n.id
}

// Kind returns the structural node category.
func (n Node) Kind() Kind {
	return n.kind
}

// Document is an immutable parsed Markdown source snapshot.
type Document struct {
	document       *splice.Document
	extensionNodes []ExtensionNode
}

// Parse copies and parses source into an immutable document snapshot.
func Parse(source []byte) (*Document, error) {
	document, err := splice.Parse(source)
	if err != nil {
		return nil, publicError(err)
	}
	return &Document{document: document}, nil
}

// Nodes returns summaries for node kinds promoted into the public API.
func (d *Document) Nodes() []Node {
	if d == nil || d.document == nil {
		return nil
	}
	nodes := make([]Node, 0, d.document.NodeCount())
	for i := 0; i < d.document.NodeCount(); i++ {
		summary, ok := d.document.NodeSummaryAt(i)
		if !ok {
			continue
		}
		if node, ok := publicNodeSummary(summary); ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// Node returns one node summary by snapshot-local ID.
func (d *Document) Node(id NodeID) (Node, bool) {
	node, ok := d.internalNode(id)
	if !ok {
		return Node{}, false
	}
	return publicNode(node)
}

// SourceRange returns a copy of one valid byte range from the immutable source snapshot.
// Caller mutations of the returned bytes do not affect the document.
func (d *Document) SourceRange(range_ Range) ([]byte, bool) {
	if d == nil || d.document == nil {
		return nil, false
	}
	return d.document.SourceRange(splice.Range{Start: range_.Start, End: range_.End})
}

// ChangeSet is an opaque prepared change bound to one exact source snapshot.
// Its zero value is unbound and Apply reports ErrSourceConflict.
type ChangeSet struct {
	change splice.ChangeSet
}

// Apply applies the prepared change only when source matches its original snapshot.
func (c ChangeSet) Apply(source []byte) ([]byte, error) {
	result, err := c.change.Apply(source)
	if err != nil {
		return nil, publicError(err)
	}
	return result, nil
}

func (d *Document) internalNode(id NodeID) (splice.Node, bool) {
	if d == nil || d.document == nil {
		return splice.Node{}, false
	}
	return d.document.Node(internalNodeID(id))
}

func (d *Document) promotedNode(id NodeID, expected splice.Kind, requireTopLevel bool) (splice.Node, error) {
	return d.promotedNodeKinds(id, requireTopLevel, expected)
}

func (d *Document) promotedNodeKinds(id NodeID, requireTopLevel bool, expected ...splice.Kind) (splice.Node, error) {
	node, ok := d.internalNode(id)
	if !ok {
		return splice.Node{}, ErrNodeNotFound
	}
	if !node.Editable || requireTopLevel && !node.TopLevel {
		return splice.Node{}, ErrInvalidTargetKind
	}
	for _, kind := range expected {
		if node.Kind == kind {
			return node, nil
		}
	}
	return splice.Node{}, ErrInvalidTargetKind
}

func publicNode(node splice.Node) (Node, bool) {
	return publicNodeSummary(splice.NodeSummary{ID: node.ID, Kind: node.Kind, TopLevel: node.TopLevel, Editable: node.Editable})
}

func publicNodeSummary(summary splice.NodeSummary) (Node, bool) {
	if !summary.Editable {
		return Node{}, false
	}
	if (summary.Kind == splice.KindParagraph || summary.Kind == splice.KindHeading) && !summary.TopLevel {
		return Node{}, false
	}
	kind, ok := publicKind(summary.Kind)
	if !ok {
		return Node{}, false
	}
	return Node{id: publicNodeID(summary.ID), kind: kind}, true
}

func publicNodeID(id splice.NodeID) NodeID {
	return NodeID{value: string(id)}
}

func publicNodeIDs(ids []splice.NodeID) []NodeID {
	result := make([]NodeID, len(ids))
	for index, id := range ids {
		result[index] = publicNodeID(id)
	}
	return result
}

func publicRange(value splice.Range) Range {
	return Range{Start: value.Start, End: value.End}
}

func publicRanges(values []splice.Range) []Range {
	result := make([]Range, len(values))
	for index, value := range values {
		result[index] = publicRange(value)
	}
	return result
}

func internalNodeID(id NodeID) splice.NodeID {
	return splice.NodeID(id.value)
}

var publicKindByInternalKind = [splice.KindMathExpression + 1]Kind{
	splice.KindParagraph:            KindParagraph,
	splice.KindHeading:              KindHeading,
	splice.KindTask:                 KindTask,
	splice.KindListItem:             KindListItem,
	splice.KindTableCell:            KindTableCell,
	splice.KindFencedCode:           KindFencedCode,
	splice.KindStrikethrough:        KindStrikethrough,
	splice.KindInlineLink:           KindInlineLink,
	splice.KindReferenceDefinition:  KindReferenceDefinition,
	splice.KindAutoLink:             KindAutoLink,
	splice.KindCodeSpan:             KindCodeSpan,
	splice.KindEmphasis:             KindEmphasis,
	splice.KindStrong:               KindStrong,
	splice.KindYAMLFrontMatterField: KindFrontMatterField,
	splice.KindTOMLFrontMatterField: KindFrontMatterField,
	splice.KindHTMLComment:          KindHTMLComment,
	splice.KindHTMLAnchor:           KindHTMLAnchor,
	splice.KindImage:                KindImage,
	splice.KindTableRow:             KindTableRow,
	splice.KindThematicBreak:        KindThematicBreak,
	splice.KindBlockquote:           KindBlockquote,
	splice.KindTable:                KindTable,
	splice.KindFootnoteDefinition:   KindFootnoteDefinition,
	splice.KindMathExpression:       KindMathExpression,
}

func publicKind(kind splice.Kind) (Kind, bool) {
	if kind <= splice.KindUnknown || int(kind) >= len(publicKindByInternalKind) {
		return KindUnknown, false
	}
	public := publicKindByInternalKind[kind]
	return public, public != KindUnknown
}

func publicError(err error) error {
	switch {
	case errors.Is(err, splice.ErrNodeNotFound):
		return translateError(err, splice.ErrNodeNotFound, ErrNodeNotFound)
	case errors.Is(err, splice.ErrInvalidReplacement):
		return translateError(err, splice.ErrInvalidReplacement, ErrInvalidReplacement)
	case errors.Is(err, splice.ErrInvalidTargetKind):
		return translateError(err, splice.ErrInvalidTargetKind, ErrInvalidTargetKind)
	case errors.Is(err, splice.ErrSourceConflict):
		return translateError(err, splice.ErrSourceConflict, ErrSourceConflict)
	default:
		return err
	}
}

func translateError(err, internalSentinel, publicSentinel error) error {
	if err == internalSentinel {
		return publicSentinel
	}
	message := err.Error()
	prefix := internalSentinel.Error() + ": "
	message = strings.TrimPrefix(message, prefix)
	return fmt.Errorf("%w: %s", publicSentinel, message)
}
