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
	Kind                 Kind
	Range                Range
	Level                int
	Checked              bool
	Ordered              bool
	Marker               byte
	HasListParent        bool
	ListParentAnchor     int
	HasListChildren      bool
	ListDirectChildCount int
	TableHeader          bool
	TableColumn          int
	TableRowAnchor       int
	Anchor               int
	Destination          string
	Label                string
	Title                string
	HasTitle             bool
	Value                string
	AutoLinkEmail        bool
	TopLevel             bool
}

// Adapter parses Markdown into Marksplice-owned semantic observations.
type Adapter interface {
	Parse(source []byte) ([]Node, error)
}
