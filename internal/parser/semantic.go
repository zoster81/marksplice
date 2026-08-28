package parser

import "errors"

// SemanticPhase identifies whether a semantic event opens a container, is a
// leaf value, or closes a container. Enter/exit pairs are emitted in strict
// nesting order by SemanticBackend implementations.
type SemanticPhase uint8

const (
	SemanticPhaseUnknown SemanticPhase = iota
	SemanticEnter
	SemanticLeaf
	SemanticExit
)

// SemanticKind identifies renderer-oriented semantics without exposing a
// parser AST. M118 establishes the event vocabulary; completeness is proven by
// the following semantic-conformance milestone.
type SemanticKind uint8

const (
	SemanticUnknown SemanticKind = iota
	SemanticDocument
	SemanticParagraph
	SemanticHeading
	SemanticText
	SemanticSoftBreak
	SemanticHardBreak
	SemanticEmphasis
	SemanticStrong
	SemanticStrikethrough
	SemanticCodeSpan
	SemanticLink
	SemanticImage
	SemanticAutoLink
	SemanticRawHTML
	SemanticBlockquote
	SemanticList
	SemanticListItem
	SemanticTaskItem
	SemanticThematicBreak
	SemanticTable
	SemanticTableRow
	SemanticTableCell
	SemanticCodeBlock
	SemanticHTMLBlock
	SemanticFootnoteDefinition
	SemanticFootnoteReference
	SemanticMath
	SemanticReferenceDefinition
	SemanticAlert
	SemanticFrontMatter
)

// SemanticAlertKind identifies one reviewed GitHub alert semantic kind.
type SemanticAlertKind uint8

const (
	SemanticAlertUnknown SemanticAlertKind = iota
	SemanticAlertNote
	SemanticAlertTip
	SemanticAlertImportant
	SemanticAlertWarning
	SemanticAlertCaution
)

// SemanticFrontMatterFormat identifies the source envelope format.
type SemanticFrontMatterFormat uint8

const (
	SemanticFrontMatterUnknown SemanticFrontMatterFormat = iota
	SemanticFrontMatterYAML
	SemanticFrontMatterTOML
)

// SemanticEvent is one ephemeral renderer-oriented event. Range and
// ContentRange are offsets into the exact source passed to WalkSemantic and are
// snapshot-local metadata, not durable identities. Kind-specific scalar fields
// are populated only when relevant to that event.
type SemanticEvent struct {
	Phase             SemanticPhase
	Kind              SemanticKind
	Range             Range
	ContentRange      Range
	Value             string
	Level             int
	Destination       string
	Title             string
	HasTitle          bool
	Label             string
	AutoLinkEmail     bool
	Ordered           bool
	Start             int
	Tight             bool
	Checked           bool
	Marker            byte
	Header            bool
	Column            int
	Columns           int
	Alignment         TableAlignment
	Info              string
	Language          string
	Fenced            bool
	MathStyle         MathExpressionStyle
	AlertKind         SemanticAlertKind
	FrontMatterFormat SemanticFrontMatterFormat
	DefinitionAnchor  int
	Occurrence        int
}

// SemanticVisitor consumes events synchronously. Implementations must stop
// immediately and return the visitor error unchanged when the visitor fails.
type SemanticVisitor func(SemanticEvent) error

// ErrSemanticVisitorRequired classifies a nil semantic visitor.
var ErrSemanticVisitorRequired = errors.New("parser: semantic visitor is required")

// SemanticBackend is the optional on-demand semantic rendering contract. It is
// deliberately separate from Backend so ordinary document parsing does not
// retain a second AST or pay for rendering projection when callers do not use
// it.
type SemanticBackend interface {
	WalkSemantic(source []byte, visit SemanticVisitor) error
}
