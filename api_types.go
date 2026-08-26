package marksplice

// HeadingStyle identifies the source syntax of a promoted heading.
type HeadingStyle uint8

const (
	HeadingStyleUnknown HeadingStyle = iota
	HeadingStyleATX
	HeadingStyleSetext
)

// FrontMatterFormat identifies the source format of a recognized front-matter envelope or promoted field.
type FrontMatterFormat uint8

const (
	FrontMatterFormatUnknown FrontMatterFormat = iota
	FrontMatterFormatYAML
	FrontMatterFormatTOML
)

// FrontMatter is immutable source ownership for one recognized document-leading metadata envelope.
// It is document-envelope state rather than a structural Markdown node.
type FrontMatter struct {
	format       FrontMatterFormat
	sourceRange  Range
	openingRange Range
	closingRange Range
}

// Format returns whether the envelope uses the reviewed YAML or TOML delimiters.
func (f FrontMatter) Format() FrontMatterFormat { return f.format }

// Range returns the complete envelope from the opening delimiter through the closing delimiter.
// A physical line terminator following the closing delimiter is outside this range.
func (f FrontMatter) Range() Range { return f.sourceRange }

// OpeningRange returns the exact opening delimiter bytes.
func (f FrontMatter) OpeningRange() Range { return f.openingRange }

// ClosingRange returns the exact closing delimiter bytes.
func (f FrontMatter) ClosingRange() Range { return f.closingRange }

// HTMLAnchorAttribute identifies the semantic anchor attribute targeted by an HTMLAnchor.
type HTMLAnchorAttribute uint8

const (
	HTMLAnchorAttributeUnknown HTMLAnchorAttribute = iota
	HTMLAnchorAttributeID
	HTMLAnchorAttributeName
)

// Paragraph is immutable typed detail for one promoted top-level paragraph.
type Paragraph struct {
	id          NodeID
	sourceRange Range
}

// ID returns the paragraph's snapshot-scoped node identity.
func (p Paragraph) ID() NodeID {
	return p.id
}

// Range returns the exact paragraph byte span replaced by PrepareReplaceParagraph.
// A line ending immediately following the paragraph is outside this range.
func (p Paragraph) Range() Range {
	return p.sourceRange
}

// Heading is immutable typed detail for one promoted top-level heading.
type Heading struct {
	id          NodeID
	sourceRange Range
	level       int
	style       HeadingStyle
}

// ID returns the heading's snapshot-scoped node identity.
func (h Heading) ID() NodeID {
	return h.id
}

// Range returns the exact heading-content byte span replaced by PrepareRenameHeading.
// ATX markers, optional closing markers, Setext underlines, and line endings are outside this range.
func (h Heading) Range() Range {
	return h.sourceRange
}

// Level returns the GFM heading level from 1 through 6.
func (h Heading) Level() int {
	return h.level
}

// Style returns whether the heading uses ATX or Setext source syntax.
func (h Heading) Style() HeadingStyle {
	return h.style
}

// ListItem is immutable typed detail for one promoted single-line list item.
type ListItem struct {
	id              NodeID
	sourceRange     Range
	subtreeRange    Range
	hasSubtreeRange bool
	ordered         bool
	marker          byte
	parentID        NodeID
	childIDs        []NodeID
	hasChildren     bool
}

// ID returns the list item's snapshot-scoped node identity.
func (i ListItem) ID() NodeID {
	return i.id
}

// Range returns the exact list-item content span replaced by PrepareReplaceListItem.
// Indentation, list numbering, marker/delimiter bytes, post-marker spacing, and line endings are outside this range.
func (i ListItem) Range() Range {
	return i.sourceRange
}

// SubtreeRange returns the exact complete supported subtree source span used by structural list-item operations.
// The boolean is false when Marksplice cannot prove that every semantic descendant belongs to the supported list-item model.
func (i ListItem) SubtreeRange() (Range, bool) {
	if !i.hasSubtreeRange {
		return Range{}, false
	}
	return i.subtreeRange, true
}

// Ordered reports whether the item belongs to an ordered list.
func (i ListItem) Ordered() bool {
	return i.ordered
}

// Marker returns the source marker/delimiter byte.
// Unordered items use '-', '*', or '+'; ordered items use '.' or ')'.
func (i ListItem) Marker() byte {
	return i.marker
}

// ParentID returns the immediate supported list-item parent's snapshot-scoped identity.
// The boolean is false for root items and when the semantic parent exists but is not publicly promoted.
func (i ListItem) ParentID() (NodeID, bool) {
	if i.parentID.value == "" {
		return NodeID{}, false
	}
	return i.parentID, true
}

// ChildIDs returns the immediate supported list-item child identities in source order.
// Semantic children outside the promoted public subset are omitted.
func (i ListItem) ChildIDs() []NodeID {
	return append([]NodeID(nil), i.childIDs...)
}

// HasChildren reports whether the supported single-line item owns one or more semantic direct child list items.
// It can be true even when ChildIDs is empty because unsupported children are not assigned public identities.
func (i ListItem) HasChildren() bool {
	return i.hasChildren
}

// Task is immutable typed detail for one promoted GFM task marker.
type Task struct {
	id          NodeID
	sourceRange Range
	checked     bool
}

// ID returns the task's snapshot-scoped node identity.
func (t Task) ID() NodeID {
	return t.id
}

// Range returns the exact one-byte task state span changed by PrepareSetTaskChecked.
// The surrounding brackets and list-item source are outside this range.
func (t Task) Range() Range {
	return t.sourceRange
}

// Checked reports the semantic task state.
func (t Task) Checked() bool {
	return t.checked
}

// TableCell is immutable typed detail for one promoted non-empty GFM table cell.
type TableCell struct {
	id          NodeID
	sourceRange Range
	tableID     NodeID
	rowID       NodeID
	header      bool
	column      int
}

// ID returns the table cell's snapshot-scoped node identity.
func (c TableCell) ID() NodeID {
	return c.id
}

// Range returns the exact table-cell content span replaced by PrepareReplaceTableCell.
// Pipes, cell padding, alignment syntax, neighboring cells, and line endings are outside this range.
func (c TableCell) Range() Range {
	return c.sourceRange
}

// Header reports whether the cell belongs to the table header row.
func (c TableCell) Header() bool {
	return c.header
}

// TableID returns the promoted GFM table that owns this cell.
// The boolean is false when no promoted table identity is available.
func (c TableCell) TableID() (NodeID, bool) {
	if c.tableID.value == "" {
		return NodeID{}, false
	}
	return c.tableID, true
}

// RowID returns the promoted GFM body row that owns this cell.
// The boolean is false for header cells and when no promoted body-row identity is available.
func (c TableCell) RowID() (NodeID, bool) {
	if c.rowID.value == "" {
		return NodeID{}, false
	}
	return c.rowID, true
}

// Column returns the zero-based column index within the mapped table row.
func (c TableCell) Column() int {
	return c.column
}

// Table is immutable typed detail for one promoted GFM table.
type Table struct {
	id           NodeID
	sourceRange  Range
	columnCount  int
	bodyRowCount int
}

// ID returns the table's snapshot-scoped node identity.
func (t Table) ID() NodeID { return t.id }

// Range returns the exact complete table source span.
// It owns the header row, delimiter row, and every semantic body row; when present, the final owned line terminator is included.
func (t Table) Range() Range { return t.sourceRange }

// ColumnCount returns the semantic/source-proven number of table columns.
func (t Table) ColumnCount() int { return t.columnCount }

// BodyRowCount returns the semantic number of body rows, including rows outside the promoted public row subset.
func (t Table) BodyRowCount() int { return t.bodyRowCount }

// TableRow is immutable typed detail for one promoted GFM table body row.
type TableRow struct {
	id          NodeID
	sourceRange Range
	tableID     NodeID
	columnCount int
	previousID  NodeID
	nextID      NodeID
}

// ID returns the table row's snapshot-scoped node identity.
func (r TableRow) ID() NodeID { return r.id }

// Range returns the exact complete physical body-row span used by structural row operations.
// When present, the row's own line terminator is included; header and delimiter rows are never part of this range.
func (r TableRow) Range() Range { return r.sourceRange }

// ColumnCount returns the semantic/source-proven number of columns in the body row.
func (r TableRow) ColumnCount() int { return r.columnCount }

// TableID returns the promoted GFM table that owns this body row.
// The boolean is false when no promoted table identity is available.
func (r TableRow) TableID() (NodeID, bool) {
	if r.tableID.value == "" {
		return NodeID{}, false
	}
	return r.tableID, true
}

// PreviousID returns the nearest promoted body row before this row in the same table.
func (r TableRow) PreviousID() (NodeID, bool) {
	if r.previousID.value == "" {
		return NodeID{}, false
	}
	return r.previousID, true
}

// NextID returns the nearest promoted body row after this row in the same table.
func (r TableRow) NextID() (NodeID, bool) {
	if r.nextID.value == "" {
		return NodeID{}, false
	}
	return r.nextID, true
}

// ThematicBreak is immutable typed detail for one promoted top-level thematic break.
type ThematicBreak struct {
	id          NodeID
	sourceRange Range
}

// ID returns the thematic break's snapshot-scoped node identity.
func (t ThematicBreak) ID() NodeID { return t.id }

// Range returns the exact complete physical line owned by structural thematic-break operations.
// When present, the line terminator is included.
func (t ThematicBreak) Range() Range { return t.sourceRange }

// Blockquote is immutable typed detail for one promoted complete top-level blockquote container.
type Blockquote struct {
	id           NodeID
	sourceRange  Range
	contentRange Range
}

// ID returns the blockquote's snapshot-scoped node identity.
func (b Blockquote) ID() NodeID { return b.id }

// Range returns the exact complete physical source owned by the top-level blockquote container.
// Every owned physical line terminator is included when present.
func (b Blockquote) Range() Range { return b.sourceRange }

// ContentRange returns the historical single-line inner source span when the
// promoted blockquote owns exactly one physical content segment. It returns the
// zero Range for segmented multiline, lazy-continuation, or multi-block source;
// use Document.BlockquoteContentRanges for those containers.
func (b Blockquote) ContentRange() Range { return b.contentRange }

// FencedCode is immutable typed detail for one fenced code block whose payload
// is proven to be one exact contiguous source span suitable for the historical
// source-preserving replacement API. Use Document.FencedBlocks for broader
// read-only fenced-container ownership.
type FencedCode struct {
	id          NodeID
	sourceRange Range
}

// ID returns the fenced code block's snapshot-scoped node identity.
func (f FencedCode) ID() NodeID {
	return f.id
}

// Range returns the exact fenced-code content span replaced by PrepareReplaceFencedCode.
// Internal body line endings are part of this span. Fence lines, info-string source,
// and the final line ending immediately before a closing fence are outside it.
// For an unclosed block the payload still excludes the preserved trailing source
// line ending when one is present.
func (f FencedCode) Range() Range {
	return f.sourceRange
}

// Strikethrough is immutable typed detail for one promoted simple GFM strikethrough.
type Strikethrough struct {
	id          NodeID
	sourceRange Range
}

// ID returns the strikethrough's snapshot-scoped node identity.
func (s Strikethrough) ID() NodeID { return s.id }

// Range returns the exact strikethrough content span replaced by PrepareReplaceStrikethrough.
func (s Strikethrough) Range() Range { return s.sourceRange }

// CodeSpan is immutable typed detail for one promoted simple single-line code span.
type CodeSpan struct {
	id          NodeID
	sourceRange Range
}

// ID returns the code span's snapshot-scoped node identity.
func (c CodeSpan) ID() NodeID { return c.id }

// Range returns the exact code-span content span replaced by PrepareReplaceCodeSpan.
func (c CodeSpan) Range() Range { return c.sourceRange }

// Emphasis is immutable typed detail for one promoted simple emphasis span.
type Emphasis struct {
	id          NodeID
	sourceRange Range
}

// ID returns the emphasis span's snapshot-scoped node identity.
func (e Emphasis) ID() NodeID { return e.id }

// Range returns the exact emphasis content span replaced by PrepareReplaceEmphasis.
func (e Emphasis) Range() Range { return e.sourceRange }

// Strong is immutable typed detail for one promoted simple strong-emphasis span.
type Strong struct {
	id          NodeID
	sourceRange Range
}

// ID returns the strong span's snapshot-scoped node identity.
func (s Strong) ID() NodeID { return s.id }

// Range returns the exact strong content span replaced by PrepareReplaceStrong.
func (s Strong) Range() Range { return s.sourceRange }

// InlineLink is immutable typed detail for one promoted simple inline link.
type InlineLink struct {
	id          NodeID
	sourceRange Range
	destination string
	title       string
	hasTitle    bool
}

// ID returns the inline link's snapshot-scoped node identity.
func (l InlineLink) ID() NodeID { return l.id }

// Range returns the exact destination span replaced by PrepareReplaceInlineLinkDestination.
// Label, parentheses, destination wrappers, title syntax, and surrounding source are outside this range.
func (l InlineLink) Range() Range { return l.sourceRange }

// Destination returns the parser-proven semantic link destination.
func (l InlineLink) Destination() string { return l.destination }

// Title returns the parser-proven semantic link title when one is present.
func (l InlineLink) Title() (string, bool) {
	if !l.hasTitle {
		return "", false
	}
	return l.title, true
}

// Image is immutable typed detail for one promoted simple inline image.
type Image struct {
	id          NodeID
	sourceRange Range
}

// ID returns the image's snapshot-scoped node identity.
func (i Image) ID() NodeID { return i.id }

// Range returns the exact destination span replaced by PrepareReplaceImageDestination.
// The image marker, alt text, parentheses, destination wrappers, title syntax, and surrounding source are outside this range.
func (i Image) Range() Range { return i.sourceRange }

// ReferenceDefinition is immutable typed detail for one promoted single-line reference definition.
type ReferenceDefinition struct {
	id          NodeID
	sourceRange Range
	label       string
	destination string
	title       string
	hasTitle    bool
}

// ID returns the reference definition's snapshot-scoped node identity.
func (r ReferenceDefinition) ID() NodeID { return r.id }

// Range returns the exact destination span replaced by PrepareReplaceReferenceDefinitionDestination.
// Label, colon, destination wrappers, title syntax, indentation, trailing spaces, and line endings are outside this range.
func (r ReferenceDefinition) Range() Range { return r.sourceRange }

// Label returns the parser-proven reference-definition label as authored.
func (r ReferenceDefinition) Label() string { return r.label }

// Destination returns the parser-proven semantic reference destination.
func (r ReferenceDefinition) Destination() string { return r.destination }

// Title returns the parser-proven semantic reference title when one is present.
func (r ReferenceDefinition) Title() (string, bool) {
	if !r.hasTitle {
		return "", false
	}
	return r.title, true
}

// AutoLink is immutable typed detail for one promoted single-line GFM autolink.
type AutoLink struct {
	id          NodeID
	sourceRange Range
	value       string
	email       bool
}

// ID returns the autolink's snapshot-scoped node identity.
func (a AutoLink) ID() NodeID { return a.id }

// Range returns the exact autolink token content replaced by PrepareReplaceAutoLink.
// Angle brackets, when present, and surrounding source are outside this range.
func (a AutoLink) Range() Range { return a.sourceRange }

// Value returns the parser-proven semantic autolink value.
func (a AutoLink) Value() string { return a.value }

// IsEmail reports whether the parser classified this as an email autolink.
func (a AutoLink) IsEmail() bool { return a.email }

// FrontMatterField is immutable typed detail for one promoted simple leading YAML/TOML scalar field.
type FrontMatterField struct {
	id          NodeID
	sourceRange Range
	key         string
	format      FrontMatterFormat
}

// ID returns the field's snapshot-scoped node identity.
func (f FrontMatterField) ID() NodeID { return f.id }

// Range returns the exact scalar value span replaced by PrepareReplaceFrontMatterValue.
// Delimiters, key spelling, separator spacing, quote wrappers, comments, and line endings are outside this range.
func (f FrontMatterField) Range() Range { return f.sourceRange }

// Key returns the recognized simple scalar field key.
func (f FrontMatterField) Key() string { return f.key }

// Format returns whether the field belongs to a YAML or TOML front-matter envelope.
func (f FrontMatterField) Format() FrontMatterFormat { return f.format }

// HTMLComment is immutable typed detail for one promoted single-line HTML comment payload.
type HTMLComment struct {
	id          NodeID
	sourceRange Range
}

// ID returns the HTML comment's snapshot-scoped node identity.
func (c HTMLComment) ID() NodeID { return c.id }

// Range returns the exact comment payload span replaced by PrepareReplaceHTMLComment.
// Comment delimiters and preserved inner horizontal padding are outside this range.
func (c HTMLComment) Range() Range { return c.sourceRange }

// HTMLAnchor is immutable typed detail for one promoted simple quoted id/name attribute on an <a> tag.
type HTMLAnchor struct {
	id          NodeID
	sourceRange Range
	attribute   HTMLAnchorAttribute
}

// ID returns the HTML anchor's snapshot-scoped node identity.
func (a HTMLAnchor) ID() NodeID { return a.id }

// Range returns the exact quoted attribute value span replaced by PrepareReplaceHTMLAnchor.
// Tag/attribute spelling, spacing, quote wrappers, and other attributes are outside this range.
func (a HTMLAnchor) Range() Range { return a.sourceRange }

// Attribute returns whether the promoted anchor targets an id or name attribute.
func (a HTMLAnchor) Attribute() HTMLAnchorAttribute { return a.attribute }

// Paragraph returns typed detail for one promoted top-level paragraph.
