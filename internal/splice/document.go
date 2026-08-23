package splice

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
	goldmarkparser "github.com/zoster81/marksplice/internal/parser/goldmark"
	"github.com/zoster81/marksplice/internal/source"
)

var (
	ErrNodeNotFound       = errors.New("node not found")
	ErrInvalidReplacement = errors.New("invalid replacement")
	ErrInvalidTargetKind  = errors.New("invalid target kind")
	ErrSourceConflict     = source.ErrConflict

	errDuplicateNodeID = errors.New("duplicate node ID")
)

// Kind identifies a Marksplice structural node kind.
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
	KindYAMLFrontMatterField
	KindTOMLFrontMatterField
	KindHTMLComment
	KindHTMLAnchor
	KindHTMLOpaque
	KindImage
	KindTableRow
	KindThematicBreak
	KindBlockquote
	KindTable
)

// HeadingStyle identifies the source syntax of a heading.
type HeadingStyle uint8

const (
	HeadingStyleUnknown HeadingStyle = HeadingStyle(source.HeadingStyleUnknown)
	HeadingStyleATX     HeadingStyle = HeadingStyle(source.HeadingStyleATX)
	HeadingStyleSetext  HeadingStyle = HeadingStyle(source.HeadingStyleSetext)
)

// TableAlignment identifies the semantic alignment of one GFM table column.
type TableAlignment = parser.TableAlignment

const (
	TableAlignmentDefault = parser.TableAlignmentDefault
	TableAlignmentLeft    = parser.TableAlignmentLeft
	TableAlignmentRight   = parser.TableAlignmentRight
	TableAlignmentCenter  = parser.TableAlignmentCenter
)

// FrontMatterFormat identifies a recognized leading metadata envelope format.
type FrontMatterFormat uint8

const (
	FrontMatterFormatUnknown FrontMatterFormat = FrontMatterFormat(source.FrontMatterUnknown)
	FrontMatterFormatYAML    FrontMatterFormat = FrontMatterFormat(source.FrontMatterYAML)
	FrontMatterFormatTOML    FrontMatterFormat = FrontMatterFormat(source.FrontMatterTOML)
)

// NodeID is deterministic within one source snapshot.
type NodeID string

// Range is a half-open byte range in the source snapshot.
type Range = source.Range

// Node is the minimal Marksplice-owned structural view used by the feasibility slice.
type Node struct {
	ID                        NodeID
	Kind                      Kind
	Range                     Range
	ContentRange              Range
	Level                     int
	HeadingStyle              HeadingStyle
	Checked                   bool
	ListOrdered               bool
	ListMarker                byte
	ListHasParent             bool
	ListParentAnchor          int
	ListContainerAnchor       int
	ListParentID              NodeID
	ListHasChildren           bool
	ListDirectChildCount      int
	ListChildStart            int
	ListChildCount            int
	ListSubtreeComplete       bool
	ListSubtreeEnd            int
	ListItemSource            source.ListItemMapping
	TableHeader               bool
	TableColumn               int
	TableRowAnchor            int
	TableRowID                NodeID
	TableAnchor               int
	TableColumnCount          int
	TableAlignments           []TableAlignment
	TableID                   NodeID
	TableBodyRowCount         int
	TableLastBodyRowAnchor    int
	TablePromotedRowStart     int
	TablePromotedRowCount     int
	TableOwnedHeaderCellStart int
	TableOwnedHeaderCellCount int
	TablePreviousRowID        NodeID
	TableNextRowID            NodeID
	TableRowCellStart         int
	TableRowCellCount         int
	TableHeaderCellStart      int
	TableHeaderCellCount      int
	Editable                  bool
	TableCellSource           source.TableCellMapping
	TableRowSource            source.TableRowMapping
	TableSource               source.TableMapping
	FencedCodeSource          source.FencedCodeMapping
	StrikethroughSource       source.StrikethroughMapping
	InlineLinkSource          source.InlineLinkMapping
	ImageSource               source.ImageMapping
	ReferenceDefinitionSource source.ReferenceDefinitionMapping
	AutoLinkSource            source.AutoLinkMapping
	CodeSpanSource            source.CodeSpanMapping
	EmphasisSource            source.EmphasisMapping
	ThematicBreakSource       source.ThematicBreakMapping
	BlockquoteSource          source.BlockquoteMapping
	Anchor                    int
	Destination               string
	Label                     string
	Title                     string
	HasTitle                  bool
	Value                     string
	AutoLinkEmail             bool
	Key                       string
	FrontMatterFormat         FrontMatterFormat
	FrontMatterStyle          source.FrontMatterValueStyle
	HTMLAttribute             string
	HTMLQuote                 byte
	TopLevel                  bool
}

// ChangeSet is a source-bound prepared mutation.
type ChangeSet = source.ChangeSet

type frontMatterEnvelope struct {
	Format       source.FrontMatterFormat
	OpeningRange Range
	ClosingRange Range
}

// Document is an immutable parsed source snapshot used by the feasibility slice.
type Document struct {
	source                  []byte
	nodes                   []Node
	nodeIndex               map[NodeID]int
	listChildIDs            []NodeID
	listItemIndexes         []int
	tableCellIDs            []NodeID
	tableHeaderCellIDs      []NodeID
	tableRowIndexes         []int
	tableCellIndexes        []int
	tableRowIDs             []NodeID
	tableOwnedHeaderCellIDs []NodeID
	sections                []Section
	sectionIndex            map[NodeID]int
	frontMatter             frontMatterEnvelope
	referenceUsages         []parser.ReferenceUsage
}

// Parse creates a snapshot-local Marksplice model using the internal Goldmark adapter.
func Parse(input []byte) (*Document, error) {
	semanticParser := goldmarkparser.New()
	snapshot := append([]byte(nil), input...)
	frontMatter, hasFrontMatter := source.MapLeadingFrontMatter(snapshot)
	observations, referenceUsages, err := semanticParser.ParseWithReferenceUsages(snapshot)
	if err != nil {
		return nil, fmt.Errorf("parse markdown: %w", err)
	}
	if hasFrontMatter {
		referenceUsages = referenceUsagesOutsideRange(referenceUsages, frontMatter.Range)
	}

	fingerprint := source.Sum(snapshot)
	nodes := make([]Node, 0, len(observations)+len(frontMatter.Fields))
	tableRows := make(map[int]tableRowSourceResult)
	if hasFrontMatter {
		nodes = append(nodes, frontMatterNodes(fingerprint, frontMatter)...)
	}
	for _, observation := range observations {
		observationRange := Range{Start: observation.Range.Start, End: observation.Range.End}
		if hasFrontMatter && rangesOverlap(frontMatter.Range, observationRange) {
			continue
		}
		node, err := nodeFromObservation(snapshot, fingerprint, observation, tableRows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	listModel, err := resolveListItemModel(nodes)
	if err != nil {
		return nil, fmt.Errorf("resolve list item model: %w", err)
	}
	tableModel, err := resolveTableRowCells(nodes)
	if err != nil {
		return nil, fmt.Errorf("resolve table row cells: %w", err)
	}
	tableOwnerModel, err := resolveTables(nodes)
	if err != nil {
		return nil, fmt.Errorf("resolve tables: %w", err)
	}

	nodeIndex, err := indexNodes(nodes)
	if err != nil {
		return nil, fmt.Errorf("index structural nodes: %w", err)
	}
	sections, sectionIndex, err := buildSections(snapshot, nodes)
	if err != nil {
		return nil, fmt.Errorf("index sections: %w", err)
	}
	storedFrontMatter := frontMatterEnvelope{}
	if hasFrontMatter {
		storedFrontMatter = frontMatterEnvelope{
			Format:       frontMatter.Format,
			OpeningRange: frontMatter.OpeningRange,
			ClosingRange: frontMatter.ClosingRange,
		}
	}
	return &Document{
		source:                  snapshot,
		nodes:                   nodes,
		nodeIndex:               nodeIndex,
		listChildIDs:            listModel.childIDs,
		listItemIndexes:         listModel.itemIndexes,
		tableCellIDs:            tableModel.cellIDs,
		tableHeaderCellIDs:      tableModel.headerCellIDs,
		tableRowIndexes:         tableModel.rowIndexes,
		tableCellIndexes:        tableModel.cellIndexes,
		tableRowIDs:             tableOwnerModel.rowIDs,
		tableOwnedHeaderCellIDs: tableOwnerModel.headerCellIDs,
		sections:                sections,
		sectionIndex:            sectionIndex,
		frontMatter:             storedFrontMatter,
		referenceUsages:         append([]parser.ReferenceUsage(nil), referenceUsages...),
	}, nil
}

// Nodes returns a copy of the snapshot-local structural nodes.
func (d *Document) Nodes() []Node {
	if d == nil {
		return nil
	}
	result := make([]Node, len(d.nodes))
	for index, node := range d.nodes {
		result[index] = cloneNode(node)
	}
	return result
}

// NodeSummary is the lightweight common structural data needed for enumeration boundaries.
type NodeSummary struct {
	ID       NodeID
	Kind     Kind
	TopLevel bool
	Editable bool
}

func summarizeNode(node Node) NodeSummary {
	return NodeSummary{ID: node.ID, Kind: node.Kind, TopLevel: node.TopLevel, Editable: node.Editable}
}

// NodeCount returns the number of snapshot-local structural nodes.
func (d *Document) NodeCount() int {
	if d == nil {
		return 0
	}
	return len(d.nodes)
}

// NodeSummaryAt returns lightweight structural data at one stable snapshot-local index.
func (d *Document) NodeSummaryAt(index int) (NodeSummary, bool) {
	if d == nil || index < 0 || index >= len(d.nodes) {
		return NodeSummary{}, false
	}
	node := d.nodes[index]
	return summarizeNode(node), true
}

// Node returns one snapshot-local structural node by ID.
func (d *Document) Node(id NodeID) (Node, bool) {
	node, ok := d.nodeByID(id)
	if !ok {
		return Node{}, false
	}
	return cloneNode(node), true
}

// SourceRange returns a copy of one valid byte range from the immutable source snapshot.
func (d *Document) SourceRange(range_ Range) ([]byte, bool) {
	if d == nil || !range_.Valid(len(d.source)) {
		return nil, false
	}
	return append([]byte(nil), d.source[range_.Start:range_.End]...), true
}

// ListItemChildIDs returns a copied, source-ordered list of one supported list item's immediate supported child IDs.
func (d *Document) ListItemChildIDs(id NodeID) ([]NodeID, bool) {
	if d == nil {
		return nil, false
	}
	node, ok := d.nodeByID(id)
	if !ok {
		return nil, false
	}
	ids, ok := d.listItemChildIDSpan(node)
	if !ok {
		return nil, false
	}
	return append([]NodeID(nil), ids...), true
}

// BlockquoteContentRanges returns caller-owned per-physical-line inner source ranges
// for one promoted top-level blockquote container.
func (d *Document) BlockquoteContentRanges(id NodeID) ([]Range, bool) {
	if d == nil {
		return nil, false
	}
	node, ok := d.nodeByID(id)
	if !ok || node.Kind != KindBlockquote || !node.Editable || !node.TopLevel || len(node.BlockquoteSource.ContentRanges) == 0 {
		return nil, false
	}
	return append([]Range(nil), node.BlockquoteSource.ContentRanges...), true
}

func referenceUsagesOutsideRange(usages []parser.ReferenceUsage, excluded Range) []parser.ReferenceUsage {
	result := make([]parser.ReferenceUsage, 0, len(usages))
	for _, usage := range usages {
		if usage.Anchor >= excluded.Start && usage.Anchor < excluded.End {
			continue
		}
		result = append(result, usage)
	}
	return result
}

func cloneNode(node Node) Node {
	node.TableAlignments = append([]TableAlignment(nil), node.TableAlignments...)
	node.TableRowSource.Cells = append([]source.TableCellMapping(nil), node.TableRowSource.Cells...)
	node.TableSource.Header.Cells = append([]source.TableCellMapping(nil), node.TableSource.Header.Cells...)
	node.TableSource.Delimiter.Cells = append([]source.TableCellMapping(nil), node.TableSource.Delimiter.Cells...)
	node.TableSource.DelimiterAlignments = append([]source.TableDelimiterAlignment(nil), node.TableSource.DelimiterAlignments...)
	node.BlockquoteSource.ContentRanges = append([]source.Range(nil), node.BlockquoteSource.ContentRanges...)
	return node
}

func indexNodes(nodes []Node) (map[NodeID]int, error) {
	index := make(map[NodeID]int, len(nodes))
	for i, node := range nodes {
		if _, exists := index[node.ID]; exists {
			return nil, fmt.Errorf("%w: %q", errDuplicateNodeID, node.ID)
		}
		index[node.ID] = i
	}
	return index, nil
}

func (d *Document) nodeByID(id NodeID) (Node, bool) {
	index, ok := d.nodeIndex[id]
	if !ok || index < 0 || index >= len(d.nodes) {
		return Node{}, false
	}
	return d.nodes[index], true
}

func (d *Document) indexedEditableNode(index int, kind Kind) (Node, bool) {
	if d == nil || index < 0 || index >= len(d.nodes) {
		return Node{}, false
	}
	node := d.nodes[index]
	return node, node.Kind == kind && node.Editable
}

var spliceKindByParserKind = [parser.KindTable + 1]Kind{
	parser.KindParagraph:           KindParagraph,
	parser.KindHeading:             KindHeading,
	parser.KindTask:                KindTask,
	parser.KindListItem:            KindListItem,
	parser.KindTableCell:           KindTableCell,
	parser.KindFencedCode:          KindFencedCode,
	parser.KindStrikethrough:       KindStrikethrough,
	parser.KindInlineLink:          KindInlineLink,
	parser.KindReferenceDefinition: KindReferenceDefinition,
	parser.KindAutoLink:            KindAutoLink,
	parser.KindCodeSpan:            KindCodeSpan,
	parser.KindEmphasis:            KindEmphasis,
	parser.KindStrong:              KindStrong,
	parser.KindRawHTML:             KindHTMLOpaque,
	parser.KindHTMLBlock:           KindHTMLOpaque,
	parser.KindImage:               KindImage,
	parser.KindTableRow:            KindTableRow,
	parser.KindThematicBreak:       KindThematicBreak,
	parser.KindBlockquote:          KindBlockquote,
	parser.KindTable:               KindTable,
}

func mapKind(kind parser.Kind) (Kind, error) {
	if kind <= parser.KindUnknown || int(kind) >= len(spliceKindByParserKind) {
		return KindUnknown, fmt.Errorf("unsupported parser node kind %d", kind)
	}
	mapped := spliceKindByParserKind[kind]
	if mapped == KindUnknown {
		return KindUnknown, fmt.Errorf("unsupported parser node kind %d", kind)
	}
	return mapped, nil
}

func makeNodeID(fingerprint source.Fingerprint, kind Kind, range_ Range) NodeID {
	hash := sha256.New()
	_, _ = hash.Write(fingerprint[:])
	_, _ = hash.Write([]byte{byte(kind)})

	var offsets [16]byte
	binary.BigEndian.PutUint64(offsets[:8], uint64(range_.Start))
	binary.BigEndian.PutUint64(offsets[8:], uint64(range_.End))
	_, _ = hash.Write(offsets[:])

	sum := hash.Sum(nil)
	return NodeID(hex.EncodeToString(sum[:16]))
}
