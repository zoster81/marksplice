package marksplice

import "github.com/zoster81/marksplice/internal/splice"

// HeadingStyle identifies the source syntax of a promoted heading.
type HeadingStyle uint8

const (
	HeadingStyleUnknown HeadingStyle = iota
	HeadingStyleATX
	HeadingStyleSetext
)

// FrontMatterFormat identifies the source envelope format of a promoted front-matter field.
type FrontMatterFormat uint8

const (
	FrontMatterFormatUnknown FrontMatterFormat = iota
	FrontMatterFormatYAML
	FrontMatterFormatTOML
)

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

// Column returns the zero-based column index within the mapped table row.
func (c TableCell) Column() int {
	return c.column
}

// FencedCode is immutable typed detail for one promoted single-line fenced code block.
type FencedCode struct {
	id          NodeID
	sourceRange Range
}

// ID returns the fenced code block's snapshot-scoped node identity.
func (f FencedCode) ID() NodeID {
	return f.id
}

// Range returns the exact fenced-code content span replaced by PrepareReplaceFencedCode.
// Fence delimiters, indentation, info-string source, and line endings are outside this range.
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
}

// ID returns the inline link's snapshot-scoped node identity.
func (l InlineLink) ID() NodeID { return l.id }

// Range returns the exact destination span replaced by PrepareReplaceInlineLinkDestination.
// Label, parentheses, destination wrappers, title syntax, and surrounding source are outside this range.
func (l InlineLink) Range() Range { return l.sourceRange }

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
}

// ID returns the reference definition's snapshot-scoped node identity.
func (r ReferenceDefinition) ID() NodeID { return r.id }

// Range returns the exact destination span replaced by PrepareReplaceReferenceDefinitionDestination.
// Label, colon, destination wrappers, title syntax, indentation, trailing spaces, and line endings are outside this range.
func (r ReferenceDefinition) Range() Range { return r.sourceRange }

// AutoLink is immutable typed detail for one promoted single-line GFM autolink.
type AutoLink struct {
	id          NodeID
	sourceRange Range
}

// ID returns the autolink's snapshot-scoped node identity.
func (a AutoLink) ID() NodeID { return a.id }

// Range returns the exact autolink token content replaced by PrepareReplaceAutoLink.
// Angle brackets, when present, and surrounding source are outside this range.
func (a AutoLink) Range() Range { return a.sourceRange }

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
func (d *Document) Paragraph(id NodeID) (Paragraph, bool) {
	node, err := d.promotedNode(id, splice.KindParagraph, true)
	if err != nil {
		return Paragraph{}, false
	}
	return Paragraph{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.Range.Start, End: node.Range.End},
	}, true
}

// Heading returns typed detail for one promoted top-level heading.
func (d *Document) Heading(id NodeID) (Heading, bool) {
	node, err := d.promotedNode(id, splice.KindHeading, true)
	if err != nil {
		return Heading{}, false
	}
	style, ok := publicHeadingStyle(node.HeadingStyle)
	if !ok {
		return Heading{}, false
	}
	return Heading{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		level:       node.Level,
		style:       style,
	}, true
}

// ListItem returns typed detail for one promoted single-line list item.
func (d *Document) ListItem(id NodeID) (ListItem, bool) {
	node, err := d.promotedNode(id, splice.KindListItem, false)
	if err != nil {
		return ListItem{}, false
	}
	internalChildIDs, ok := d.document.ListItemChildIDs(node.ID)
	if !ok {
		return ListItem{}, false
	}
	childIDs := make([]NodeID, len(internalChildIDs))
	for index, childID := range internalChildIDs {
		childIDs[index] = publicNodeID(childID)
	}
	item := ListItem{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		ordered:     node.ListOrdered,
		marker:      node.ListMarker,
		parentID:    publicNodeID(node.ListParentID),
		childIDs:    childIDs,
		hasChildren: node.ListHasChildren,
	}
	if node.ListSubtreeComplete {
		item.subtreeRange = Range{Start: node.ListItemSource.LineRange.Start, End: node.ListSubtreeEnd}
		item.hasSubtreeRange = true
	}
	return item, true
}

// Task returns typed detail for one promoted GFM task marker.
func (d *Document) Task(id NodeID) (Task, bool) {
	node, err := d.promotedNode(id, splice.KindTask, false)
	if err != nil {
		return Task{}, false
	}
	return Task{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		checked:     node.Checked,
	}, true
}

// TableCell returns typed detail for one promoted non-empty GFM table cell.
func (d *Document) TableCell(id NodeID) (TableCell, bool) {
	node, err := d.promotedNode(id, splice.KindTableCell, false)
	if err != nil {
		return TableCell{}, false
	}
	return TableCell{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		header:      node.TableHeader,
		column:      node.TableColumn,
	}, true
}

// FencedCode returns typed detail for one promoted single-line fenced code block.
func (d *Document) FencedCode(id NodeID) (FencedCode, bool) {
	node, err := d.promotedNode(id, splice.KindFencedCode, false)
	if err != nil {
		return FencedCode{}, false
	}
	return FencedCode{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
	}, true
}

// Strikethrough returns typed detail for one promoted simple GFM strikethrough.
func (d *Document) Strikethrough(id NodeID) (Strikethrough, bool) {
	node, err := d.promotedNode(id, splice.KindStrikethrough, false)
	if err != nil {
		return Strikethrough{}, false
	}
	return Strikethrough{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// CodeSpan returns typed detail for one promoted simple single-line code span.
func (d *Document) CodeSpan(id NodeID) (CodeSpan, bool) {
	node, err := d.promotedNode(id, splice.KindCodeSpan, false)
	if err != nil {
		return CodeSpan{}, false
	}
	return CodeSpan{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// Emphasis returns typed detail for one promoted simple emphasis span.
func (d *Document) Emphasis(id NodeID) (Emphasis, bool) {
	node, err := d.promotedNode(id, splice.KindEmphasis, false)
	if err != nil {
		return Emphasis{}, false
	}
	return Emphasis{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// Strong returns typed detail for one promoted simple strong-emphasis span.
func (d *Document) Strong(id NodeID) (Strong, bool) {
	node, err := d.promotedNode(id, splice.KindStrong, false)
	if err != nil {
		return Strong{}, false
	}
	return Strong{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// InlineLink returns typed detail for one promoted simple inline link.
func (d *Document) InlineLink(id NodeID) (InlineLink, bool) {
	node, err := d.promotedNode(id, splice.KindInlineLink, false)
	if err != nil {
		return InlineLink{}, false
	}
	return InlineLink{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// Image returns typed detail for one promoted simple inline image.
func (d *Document) Image(id NodeID) (Image, bool) {
	node, err := d.promotedNode(id, splice.KindImage, false)
	if err != nil {
		return Image{}, false
	}
	return Image{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// ReferenceDefinition returns typed detail for one promoted single-line reference definition.
func (d *Document) ReferenceDefinition(id NodeID) (ReferenceDefinition, bool) {
	node, err := d.promotedNode(id, splice.KindReferenceDefinition, false)
	if err != nil {
		return ReferenceDefinition{}, false
	}
	return ReferenceDefinition{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// AutoLink returns typed detail for one promoted single-line GFM autolink.
func (d *Document) AutoLink(id NodeID) (AutoLink, bool) {
	node, err := d.promotedNode(id, splice.KindAutoLink, false)
	if err != nil {
		return AutoLink{}, false
	}
	return AutoLink{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// FrontMatterField returns typed detail for one promoted simple leading YAML/TOML scalar field.
func (d *Document) FrontMatterField(id NodeID) (FrontMatterField, bool) {
	node, err := d.promotedNodeKinds(id, false, splice.KindYAMLFrontMatterField, splice.KindTOMLFrontMatterField)
	if err != nil {
		return FrontMatterField{}, false
	}
	format, ok := publicFrontMatterFormat(node.FrontMatterFormat)
	if !ok {
		return FrontMatterField{}, false
	}
	return FrontMatterField{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		key:         node.Key,
		format:      format,
	}, true
}

// HTMLComment returns typed detail for one promoted single-line HTML comment.
func (d *Document) HTMLComment(id NodeID) (HTMLComment, bool) {
	node, err := d.promotedNode(id, splice.KindHTMLComment, false)
	if err != nil {
		return HTMLComment{}, false
	}
	return HTMLComment{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// HTMLAnchor returns typed detail for one promoted simple quoted id/name attribute on an <a> tag.
func (d *Document) HTMLAnchor(id NodeID) (HTMLAnchor, bool) {
	node, err := d.promotedNode(id, splice.KindHTMLAnchor, false)
	if err != nil {
		return HTMLAnchor{}, false
	}
	attribute, ok := publicHTMLAnchorAttribute(node.HTMLAttribute)
	if !ok {
		return HTMLAnchor{}, false
	}
	return HTMLAnchor{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		attribute:   attribute,
	}, true
}

func publicHeadingStyle(style splice.HeadingStyle) (HeadingStyle, bool) {
	switch style {
	case splice.HeadingStyleATX:
		return HeadingStyleATX, true
	case splice.HeadingStyleSetext:
		return HeadingStyleSetext, true
	default:
		return HeadingStyleUnknown, false
	}
}

func publicFrontMatterFormat(format splice.FrontMatterFormat) (FrontMatterFormat, bool) {
	switch format {
	case splice.FrontMatterFormatYAML:
		return FrontMatterFormatYAML, true
	case splice.FrontMatterFormatTOML:
		return FrontMatterFormatTOML, true
	default:
		return FrontMatterFormatUnknown, false
	}
}

func publicHTMLAnchorAttribute(attribute string) (HTMLAnchorAttribute, bool) {
	switch attribute {
	case "id":
		return HTMLAnchorAttributeID, true
	case "name":
		return HTMLAnchorAttributeName, true
	default:
		return HTMLAnchorAttributeUnknown, false
	}
}
