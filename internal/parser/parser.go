package parser

// Kind identifies a semantic node kind without exposing parser-specific types.
type Kind uint8

const (
	KindUnknown Kind = iota
	KindParagraph
	KindHeading
	KindTask
	KindListItem
	KindTableCell
	KindFencedCode
	KindStrikethrough
	KindInlineLink
	KindReferenceDefinition
	KindAutoLink
	KindCodeSpan
	KindEmphasis
	KindStrong
	KindRawHTML
	KindHTMLBlock
	KindImage
	KindTableRow
	KindThematicBreak
	KindBlockquote
	KindTable
)

// TableAlignment identifies the semantic alignment of one GFM table column.
type TableAlignment uint8

const (
	TableAlignmentDefault TableAlignment = iota
	TableAlignmentLeft
	TableAlignmentRight
	TableAlignmentCenter
)

// Range is a half-open byte range [Start, End) in the parsed source snapshot.
type Range struct {
	Start int
	End   int
}

// Valid reports whether the range is ordered and contained in a source of total bytes.
func (r Range) Valid(total int) bool {
	return r.Start >= 0 && r.End >= r.Start && r.End <= total
}

// Node is a parser-independent semantic observation used by Marksplice internals.
type Node struct {
	Kind                     Kind
	Range                    Range
	BlockquoteContentRange   Range
	BlockquoteSemanticRanges []Range
	FencedCodeContentRanges  []Range
	FencedCodeInfo           string
	FencedCodeLanguage       string
	Level                    int
	HeadingText              string
	Checked                  bool
	Ordered                  bool
	Marker                   byte
	HasListParent            bool
	ListParentAnchor         int
	ListContainerAnchor      int
	HasListChildren          bool
	ListDirectChildCount     int
	TableHeader              bool
	TableColumn              int
	TableRowAnchor           int
	TableAnchor              int
	TableColumnCount         int
	TableAlignments          []TableAlignment
	TableBodyRowCount        int
	TableLastBodyRowAnchor   int
	Anchor                   int
	Destination              string
	Label                    string
	Title                    string
	HasTitle                 bool
	Value                    string
	AutoLinkEmail            bool
	TopLevel                 bool
}

// LinkUsageForm identifies the parsed source form of one semantic link/image usage.
type LinkUsageForm uint8

const (
	LinkUsageUnknown LinkUsageForm = iota
	LinkUsageFull
	LinkUsageCollapsed
	LinkUsageShortcut
	LinkUsageDirect
)

// LinkUsage records one parser-resolved link, image, or autolink relationship
// independently from ordinary public node promotion/source-editability.
type LinkUsage struct {
	Kind          Kind
	Form          LinkUsageForm
	Anchor        int
	Reference     string
	Destination   string
	Title         string
	HasTitle      bool
	AutoLinkEmail bool
}

// UnresolvedReferenceUsage records one conservative explicit full/collapsed
// reference-looking source form for which the parser context has no definition.
// Shortcut bracket text is intentionally excluded because it is ambiguous with
// ordinary Markdown text when no reference definition exists.
type UnresolvedReferenceUsage struct {
	Kind      Kind
	Form      LinkUsageForm
	Anchor    int
	Reference string
}

// MathExpressionStyle identifies one reviewed GitHub-compatible mathematical source form.
type MathExpressionStyle uint8

const (
	MathExpressionUnknown MathExpressionStyle = iota
	MathExpressionInlineDollar
	MathExpressionInlineBacktick
	MathExpressionBlockDollar
)

// MathExpressionObservation records one Marksplice-recognized mathematical source form
// whose location is proven against the ordinary GFM parse context. Payload remains opaque.
type MathExpressionObservation struct {
	Style        MathExpressionStyle
	Range        Range
	PayloadRange Range
	TopLevel     bool
}

// FootnoteDefinitionObservation records one parser-proven footnote definition
// independently from ordinary GFM block promotion. BodyRanges are semantic
// source-backed content segments in physical source order.
type FootnoteDefinitionObservation struct {
	Anchor     int
	Label      string
	BodyRanges []Range
}

// FootnoteReferenceObservation records one parser-proven footnote reference.
// Occurrence is zero-based among references to the same definition in source order.
type FootnoteReferenceObservation struct {
	Range            Range
	LabelRange       Range
	Label            string
	DefinitionAnchor int
	Occurrence       int
}

// DocumentObservations groups one immutable parse pass worth of parser-independent
// semantic facts without retaining parser AST or context state. Slice order is semantic
// source order unless the individual observation type documents a narrower ordering rule.
type DocumentObservations struct {
	Nodes                     []Node
	LinkUsages                []LinkUsage
	UnresolvedReferenceUsages []UnresolvedReferenceUsage
	FootnoteDefinitions       []FootnoteDefinitionObservation
	FootnoteReferences        []FootnoteReferenceObservation
	MathExpressions           []MathExpressionObservation
}

// Backend is the complete parser-independent semantic contract consumed by Marksplice.
// It deliberately contains both document observations and construction-only semantic
// proof/reference operations so replacing one parser backend cannot leave hidden backend
// dependencies elsewhere in the source-preserving or construction layers.
//
// Implementations must not mutate or retain caller source/expectation slices after a method
// returns. Byte ranges are half-open offsets into the exact source argument. Successful
// document observations and construction proofs must be deterministic for identical input.
type Backend interface {
	ParseDocument(source []byte) (DocumentObservations, error)
	ValidateNestedBlockquoteBlocks(source []byte, outer Range, innerSource []byte, depth int) error
	ValidateNestedBlockquoteParagraph(source []byte, outer Range, contentLines []Range, depth int) error
	ValidateConstructionInlineHierarchy(source []byte, expected []ConstructionInlineExpectation, references []ConstructionReferenceInlineExpectation) error
	ValidateConstructionLinkImages(source []byte, expected []ConstructionLinkImageExpectation) error
	ValidateConstructionReferenceInlines(source []byte, expected []ConstructionReferenceInlineExpectation) error
	ResolveConstructionReference(label string, definitions []ConstructionReferenceDefinition) (ConstructionReferenceDefinition, error)
	ReferenceLabelKey(label string) string
}
