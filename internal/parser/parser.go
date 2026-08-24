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

// Adapter parses Markdown into Marksplice-owned semantic observations.
type Adapter interface {
	Parse(source []byte) ([]Node, error)
}
