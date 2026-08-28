package splice

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
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
	KindFootnoteDefinition
	KindMathExpression
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
	HeadingText               string
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
	ListItemLineRange         Range
	TableHeader               bool
	TableColumn               int
	TableRowAnchor            int
	TableRowSourceAnchor      int
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
	SourceDetailIndex         uint32
	TableCellRange            Range
	MathStyle                 MathExpressionStyle
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

// FrontMatterEnvelope is immutable source ownership for one recognized leading metadata envelope.
type FrontMatterEnvelope struct {
	Format       FrontMatterFormat
	Range        Range
	OpeningRange Range
	ClosingRange Range
}

// Document is an immutable parsed source snapshot used by the feasibility slice.
type Document struct {
	source                    []byte
	nodes                     []Node
	nodeIndex                 map[NodeID]int
	listChildIDs              []NodeID
	listItemIndexes           []int
	tableCellIDs              []NodeID
	tableHeaderCellIDs        []NodeID
	tableRowIndexes           []int
	tableCellIndexes          []int
	tableRowIDs               []NodeID
	tableOwnedHeaderCellIDs   []NodeID
	fencedSources             []fencedSourceDetail
	blockquoteSources         []source.BlockquoteMapping
	footnoteSources           []source.FootnoteDefinitionMapping
	sections                  []Section
	sectionIndex              map[NodeID]int
	frontMatter               frontMatterEnvelope
	linkUsages                []parser.LinkUsage
	unresolvedReferenceUsages []parser.UnresolvedReferenceUsage
	footnoteReferences        []FootnoteReference
}

// Parse creates a snapshot-local Marksplice model using the active internal parser backend.
func Parse(input []byte) (*Document, error) {
	return parseWithBackend(input, newParserBackend())
}

func parseWithBackend(input []byte, semanticParser parser.Backend) (*Document, error) {
	if semanticParser == nil {
		return nil, fmt.Errorf("parse markdown: nil parser backend")
	}
	return parseWithValidatedBackend(input, semanticParser)
}

func parseWithValidatedBackend(input []byte, semanticParser parser.Backend) (*Document, error) {
	snapshot := append([]byte(nil), input...)
	frontMatter, hasFrontMatter := source.MapLeadingFrontMatter(snapshot)
	observed, err := semanticParser.ParseDocument(snapshot)
	if err != nil {
		return nil, fmt.Errorf("parse markdown: %w", err)
	}
	observations := observed.Nodes
	parserDetails := parserNodeDetails{
		blockquotes: observed.BlockquoteDetails,
		fencedCode:  observed.FencedCodeDetails,
		tables:      observed.TableDetails,
		tableRows:   observed.TableRowDetails,
		tableCells:  observed.TableCellDetails,
	}
	linkUsages := observed.LinkUsages
	unresolvedReferenceUsages := observed.UnresolvedReferenceUsages
	footnoteDefinitions := observed.FootnoteDefinitions
	footnoteReferences := observed.FootnoteReferences
	mathExpressions := observed.MathExpressions
	if hasFrontMatter {
		linkUsages = linkUsagesOutsideRange(linkUsages, frontMatter.Range)
		unresolvedReferenceUsages = unresolvedReferenceUsagesOutsideRange(unresolvedReferenceUsages, frontMatter.Range)
		footnoteDefinitions = footnoteDefinitionsOutsideRange(footnoteDefinitions, frontMatter.Range)
		footnoteReferences = footnoteReferencesOutsideRange(footnoteReferences, frontMatter.Range)
		mathExpressions = mathExpressionsOutsideRange(mathExpressions, frontMatter.Range)
	}

	fingerprint := source.Sum(snapshot)
	nodes := make([]Node, 0, len(observations)+len(frontMatter.Fields)+len(footnoteDefinitions)+len(mathExpressions))
	tableRows := make(map[int]tableRowSourceResult)
	tableSources := make(map[int]source.TableMapping)
	fencedCapacity, blockquoteCapacity := sourceDetailCapacities(observations)
	fencedSources := make([]fencedSourceDetail, 0, fencedCapacity)
	blockquoteSources := make([]source.BlockquoteMapping, 0, blockquoteCapacity)
	footnoteSources := make([]source.FootnoteDefinitionMapping, 0, len(footnoteDefinitions))
	if hasFrontMatter {
		nodes = append(nodes, frontMatterNodes(fingerprint, frontMatter)...)
	}
	for _, observation := range observations {
		observationRange := Range{Start: observation.Range.Start, End: observation.Range.End}
		if hasFrontMatter && rangesOverlap(frontMatter.Range, observationRange) {
			continue
		}
		node, err := nodeFromObservation(snapshot, fingerprint, observation, parserDetails, tableRows, tableSources, &fencedSources, &blockquoteSources)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	nodes, err = promoteSupplementalNodes(snapshot, fingerprint, nodes, mathExpressions, footnoteDefinitions, &footnoteSources)
	if err != nil {
		return nil, err
	}
	listModel, err := resolveListItemModel(nodes)
	if err != nil {
		return nil, fmt.Errorf("resolve list item model: %w", err)
	}
	tableModel, err := resolveTableRowCells(nodes)
	if err != nil {
		return nil, fmt.Errorf("resolve table row cells: %w", err)
	}
	tableOwnerModel, err := resolveTables(nodes, tableSources)
	if err != nil {
		return nil, fmt.Errorf("resolve tables: %w", err)
	}

	nodeIndex, err := indexNodes(nodes)
	if err != nil {
		return nil, fmt.Errorf("index structural nodes: %w", err)
	}
	resolvedFootnoteReferences := resolveFootnoteReferences(nodes, footnoteReferences)
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
		source:                    snapshot,
		nodes:                     nodes,
		nodeIndex:                 nodeIndex,
		listChildIDs:              listModel.childIDs,
		listItemIndexes:           listModel.itemIndexes,
		tableCellIDs:              tableModel.cellIDs,
		tableHeaderCellIDs:        tableModel.headerCellIDs,
		tableRowIndexes:           tableModel.rowIndexes,
		tableCellIndexes:          tableModel.cellIndexes,
		tableRowIDs:               tableOwnerModel.rowIDs,
		tableOwnedHeaderCellIDs:   tableOwnerModel.headerCellIDs,
		fencedSources:             fencedSources,
		blockquoteSources:         blockquoteSources,
		footnoteSources:           footnoteSources,
		sections:                  sections,
		sectionIndex:              sectionIndex,
		frontMatter:               storedFrontMatter,
		linkUsages:                append([]parser.LinkUsage(nil), linkUsages...),
		unresolvedReferenceUsages: append([]parser.UnresolvedReferenceUsage(nil), unresolvedReferenceUsages...),
		footnoteReferences:        resolvedFootnoteReferences,
	}, nil
}

func promoteSupplementalNodes(snapshot []byte, fingerprint source.Fingerprint, nodes []Node, mathExpressions []parser.MathExpressionObservation, footnotes []parser.FootnoteDefinitionObservation, footnoteSources *[]source.FootnoteDefinitionMapping) ([]Node, error) {
	mathNodes, err := promoteMathExpressionNodes(snapshot, fingerprint, mathExpressions)
	if err != nil {
		return nil, err
	}
	nodes = mergeSourceOrderedNodes(nodes, mathNodes)
	footnoteNodes, err := promoteFootnoteDefinitionNodes(snapshot, fingerprint, footnotes, footnoteSources)
	if err != nil {
		return nil, err
	}
	return mergeSourceOrderedNodes(nodes, footnoteNodes), nil
}

// FrontMatter returns the recognized document-leading metadata envelope, if present.
func (d *Document) FrontMatter() (FrontMatterEnvelope, bool) {
	if d == nil || d.frontMatter.Format == source.FrontMatterUnknown || d.frontMatter.OpeningRange.Start != 0 || d.frontMatter.ClosingRange.End < d.frontMatter.OpeningRange.End {
		return FrontMatterEnvelope{}, false
	}
	return FrontMatterEnvelope{
		Format:       FrontMatterFormat(d.frontMatter.Format),
		Range:        Range{Start: d.frontMatter.OpeningRange.Start, End: d.frontMatter.ClosingRange.End},
		OpeningRange: d.frontMatter.OpeningRange,
		ClosingRange: d.frontMatter.ClosingRange,
	}, true
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
	if !ok {
		return nil, false
	}
	mapping, ok := d.blockquoteSource(node)
	if !ok {
		return nil, false
	}
	return append([]Range(nil), mapping.ContentRanges...), true
}

func linkUsagesOutsideRange(usages []parser.LinkUsage, excluded Range) []parser.LinkUsage {
	result := make([]parser.LinkUsage, 0, len(usages))
	for _, usage := range usages {
		if usage.Anchor >= excluded.Start && usage.Anchor < excluded.End {
			continue
		}
		result = append(result, usage)
	}
	return result
}

func unresolvedReferenceUsagesOutsideRange(usages []parser.UnresolvedReferenceUsage, excluded Range) []parser.UnresolvedReferenceUsage {
	result := make([]parser.UnresolvedReferenceUsage, 0, len(usages))
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
	var input [sha256.Size + 1 + 16]byte
	copy(input[:sha256.Size], fingerprint[:])
	input[sha256.Size] = byte(kind)
	binary.BigEndian.PutUint64(input[sha256.Size+1:sha256.Size+9], uint64(range_.Start))
	binary.BigEndian.PutUint64(input[sha256.Size+9:], uint64(range_.End))

	sum := sha256.Sum256(input[:])
	var encoded [32]byte
	hex.Encode(encoded[:], sum[:16])
	return NodeID(string(encoded[:]))
}
