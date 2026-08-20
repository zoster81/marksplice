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
)

// HeadingStyle identifies the source syntax of a heading.
type HeadingStyle uint8

const (
	HeadingStyleUnknown HeadingStyle = HeadingStyle(source.HeadingStyleUnknown)
	HeadingStyleATX     HeadingStyle = HeadingStyle(source.HeadingStyleATX)
	HeadingStyleSetext  HeadingStyle = HeadingStyle(source.HeadingStyleSetext)
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
	ListItemSource            source.ListItemMapping
	TableHeader               bool
	TableColumn               int
	Editable                  bool
	TableCellSource           source.TableCellMapping
	FencedCodeSource          source.FencedCodeMapping
	StrikethroughSource       source.StrikethroughMapping
	InlineLinkSource          source.InlineLinkMapping
	ImageSource               source.ImageMapping
	ReferenceDefinitionSource source.ReferenceDefinitionMapping
	AutoLinkSource            source.AutoLinkMapping
	CodeSpanSource            source.CodeSpanMapping
	EmphasisSource            source.EmphasisMapping
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
	source       []byte
	nodes        []Node
	nodeIndex    map[NodeID]int
	sections     []Section
	sectionIndex map[NodeID]int
	frontMatter  frontMatterEnvelope
}

// Parse creates a snapshot-local Marksplice model using the internal Goldmark adapter.
func Parse(input []byte) (*Document, error) {
	semanticParser := goldmarkparser.New()
	snapshot := append([]byte(nil), input...)
	frontMatter, hasFrontMatter := source.MapLeadingFrontMatter(snapshot)
	observations, err := semanticParser.Parse(snapshot)
	if err != nil {
		return nil, fmt.Errorf("parse markdown: %w", err)
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
		source:       snapshot,
		nodes:        nodes,
		nodeIndex:    nodeIndex,
		sections:     sections,
		sectionIndex: sectionIndex,
		frontMatter:  storedFrontMatter,
	}, nil
}

type tableRowSourceResult struct {
	mapping  source.TableRowMapping
	editable bool
}

func nodeFromObservation(snapshot []byte, fingerprint source.Fingerprint, observation parser.Node, tableRows map[int]tableRowSourceResult) (Node, error) {
	if observation.Kind == parser.KindRawHTML {
		return nodeFromRawHTMLObservation(snapshot, fingerprint, observation)
	}
	kind, err := mapKind(observation.Kind)
	if err != nil {
		return Node{}, err
	}

	contentRange := Range{Start: observation.Range.Start, End: observation.Range.End}
	if !contentRange.Valid(len(snapshot)) {
		return Node{}, fmt.Errorf("semantic node range [%d,%d) is outside source length %d", contentRange.Start, contentRange.End, len(snapshot))
	}

	node := Node{
		Kind:          kind,
		Range:         contentRange,
		ContentRange:  contentRange,
		Level:         observation.Level,
		Checked:       observation.Checked,
		ListOrdered:   observation.Ordered,
		ListMarker:    observation.Marker,
		TableHeader:   observation.TableHeader,
		TableColumn:   observation.TableColumn,
		Anchor:        observation.Anchor,
		Destination:   observation.Destination,
		Label:         observation.Label,
		Title:         observation.Title,
		HasTitle:      observation.HasTitle,
		Value:         observation.Value,
		AutoLinkEmail: observation.AutoLinkEmail,
		TopLevel:      observation.TopLevel,
	}
	if kind == KindParagraph && observation.TopLevel {
		node.Editable = true
	}
	switch kind {
	case KindHeading:
		mapping, err := source.MapTopLevelHeading(snapshot, contentRange, observation.Level)
		if err != nil {
			return Node{}, fmt.Errorf("map heading source: %w", err)
		}
		node.Range = mapping.Range
		node.ContentRange = mapping.ContentRange
		node.HeadingStyle = HeadingStyle(mapping.Style)
		node.Editable = true
	case KindTask:
		mapping, err := source.MapTaskMarker(snapshot, observation.Range.Start)
		if err != nil {
			return Node{}, fmt.Errorf("map task marker: %w", err)
		}
		if mapping.Checked != observation.Checked {
			return Node{}, fmt.Errorf("map task marker: semantic checked state %v disagrees with source state %v", observation.Checked, mapping.Checked)
		}
		node.Range = mapping.Range
		node.ContentRange = mapping.ContentRange
		node.Checked = mapping.Checked
		node.Editable = true
	case KindListItem:
		mapping, err := source.MapSingleLineListItem(snapshot, contentRange, observation.Ordered, observation.Marker)
		if err != nil {
			return Node{}, fmt.Errorf("map list item source: %w", err)
		}
		node.Range = mapping.Range
		node.ContentRange = mapping.ContentRange
		node.ListOrdered = mapping.Ordered
		node.ListMarker = mapping.Marker
		node.ListItemSource = mapping
		node.Editable = true
	case KindTableCell:
		mapping, editable, err := mapTableCellSource(snapshot, observation, contentRange, tableRows)
		if err != nil {
			return Node{}, fmt.Errorf("map table cell source: %w", err)
		}
		if editable {
			node.TableCellSource = mapping
			node.Editable = true
		}
	case KindFencedCode:
		mapping, err := source.MapSingleLineFencedCode(snapshot, contentRange)
		if err == nil {
			node.FencedCodeSource = mapping
			node.Editable = true
		} else if !errors.Is(err, source.ErrUnsupportedFencedCodeShape) {
			return Node{}, fmt.Errorf("map fenced code source: %w", err)
		}
	case KindStrikethrough:
		mapping, err := source.MapSimpleStrikethrough(snapshot, contentRange)
		if err == nil {
			node.StrikethroughSource = mapping
			node.Editable = true
		} else if !errors.Is(err, source.ErrUnsupportedStrikethroughShape) {
			return Node{}, fmt.Errorf("map strikethrough source: %w", err)
		}
	case KindInlineLink:
		mapping, err := source.MapSimpleInlineLink(snapshot, observation.Anchor, contentRange, observation.Destination, observation.Title, observation.HasTitle)
		if err == nil {
			node.ContentRange = mapping.DestinationRange
			node.InlineLinkSource = mapping
			node.Editable = true
		} else if !errors.Is(err, source.ErrUnsupportedInlineLinkShape) {
			return Node{}, fmt.Errorf("map inline link source: %w", err)
		}
	case KindImage:
		mapping, err := source.MapSimpleImage(snapshot, observation.Anchor, contentRange)
		if err == nil {
			node.ContentRange = mapping.DestinationRange
			node.ImageSource = mapping
			node.Editable = true
		} else if !errors.Is(err, source.ErrUnsupportedImageShape) {
			return Node{}, fmt.Errorf("map image source: %w", err)
		}
	case KindReferenceDefinition:
		mapping, err := source.MapSingleLineReferenceDefinition(snapshot, contentRange, observation.Label, observation.Destination, observation.Title, observation.HasTitle)
		if err == nil {
			node.ContentRange = mapping.DestinationRange
			node.ReferenceDefinitionSource = mapping
			node.Editable = true
		} else if !errors.Is(err, source.ErrUnsupportedReferenceDefinitionShape) {
			return Node{}, fmt.Errorf("map reference definition source: %w", err)
		}
	case KindAutoLink:
		mapping, err := source.MapAutoLink(snapshot, observation.Anchor, contentRange, observation.Value, observation.AutoLinkEmail)
		if err == nil {
			node.ContentRange = mapping.ContentRange
			node.AutoLinkSource = mapping
			node.Editable = true
		} else if !errors.Is(err, source.ErrUnsupportedAutoLinkShape) {
			return Node{}, fmt.Errorf("map autolink source: %w", err)
		}
	case KindCodeSpan:
		mapping, err := source.MapSimpleCodeSpan(snapshot, observation.Anchor, contentRange)
		if err == nil {
			node.CodeSpanSource = mapping
			node.Editable = true
		} else if !errors.Is(err, source.ErrUnsupportedCodeSpanShape) {
			return Node{}, fmt.Errorf("map code span source: %w", err)
		}
	case KindEmphasis, KindStrong:
		mapping, err := source.MapSimpleEmphasis(snapshot, observation.Anchor, contentRange, observation.Level)
		if err == nil {
			node.EmphasisSource = mapping
			node.Editable = true
		} else if !errors.Is(err, source.ErrUnsupportedEmphasisShape) {
			return Node{}, fmt.Errorf("map emphasis source: %w", err)
		}
	}
	node.ID = makeNodeID(fingerprint, kind, node.Range)
	return node, nil
}

func mapTableCellSource(snapshot []byte, observation parser.Node, contentRange Range, cache map[int]tableRowSourceResult) (source.TableCellMapping, bool, error) {
	result, ok := cache[observation.TableRowAnchor]
	if !ok {
		row, err := source.MapTableRow(snapshot, observation.TableRowAnchor)
		if err != nil {
			if errors.Is(err, source.ErrUnsupportedTableCellShape) {
				cache[observation.TableRowAnchor] = tableRowSourceResult{}
				return source.TableCellMapping{}, false, nil
			}
			return source.TableCellMapping{}, false, err
		}
		result = tableRowSourceResult{mapping: row, editable: true}
		cache[observation.TableRowAnchor] = result
	}
	if !result.editable || observation.TableColumn < 0 || observation.TableColumn >= len(result.mapping.Cells) {
		return source.TableCellMapping{}, false, nil
	}
	mapping := result.mapping.Cells[observation.TableColumn]
	if mapping.ContentRange != contentRange {
		return source.TableCellMapping{}, false, nil
	}
	return mapping, true, nil
}

// Nodes returns a copy of the snapshot-local structural nodes.
func (d *Document) Nodes() []Node {
	return append([]Node(nil), d.nodes...)
}

// NodeSummary is the lightweight common structural data needed for enumeration boundaries.
type NodeSummary struct {
	ID       NodeID
	Kind     Kind
	TopLevel bool
	Editable bool
}

// NodeCount returns the number of snapshot-local structural nodes.
func (d *Document) NodeCount() int {
	return len(d.nodes)
}

// NodeSummaryAt returns lightweight structural data at one stable snapshot-local index.
func (d *Document) NodeSummaryAt(index int) (NodeSummary, bool) {
	if index < 0 || index >= len(d.nodes) {
		return NodeSummary{}, false
	}
	node := d.nodes[index]
	return NodeSummary{ID: node.ID, Kind: node.Kind, TopLevel: node.TopLevel, Editable: node.Editable}, true
}

// Node returns one snapshot-local structural node by ID.
func (d *Document) Node(id NodeID) (Node, bool) {
	return d.nodeByID(id)
}

// SourceRange returns a copy of one valid byte range from the immutable source snapshot.
func (d *Document) SourceRange(range_ Range) ([]byte, bool) {
	if d == nil || !range_.Valid(len(d.source)) {
		return nil, false
	}
	return append([]byte(nil), d.source[range_.Start:range_.End]...), true
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

func mapKind(kind parser.Kind) (Kind, error) {
	switch kind {
	case parser.KindParagraph:
		return KindParagraph, nil
	case parser.KindHeading:
		return KindHeading, nil
	case parser.KindTask:
		return KindTask, nil
	case parser.KindListItem:
		return KindListItem, nil
	case parser.KindTableCell:
		return KindTableCell, nil
	case parser.KindFencedCode:
		return KindFencedCode, nil
	case parser.KindStrikethrough:
		return KindStrikethrough, nil
	case parser.KindInlineLink:
		return KindInlineLink, nil
	case parser.KindReferenceDefinition:
		return KindReferenceDefinition, nil
	case parser.KindAutoLink:
		return KindAutoLink, nil
	case parser.KindCodeSpan:
		return KindCodeSpan, nil
	case parser.KindEmphasis:
		return KindEmphasis, nil
	case parser.KindStrong:
		return KindStrong, nil
	case parser.KindHTMLBlock:
		return KindHTMLOpaque, nil
	case parser.KindRawHTML:
		return KindHTMLOpaque, nil
	case parser.KindImage:
		return KindImage, nil
	default:
		return KindUnknown, fmt.Errorf("unsupported parser node kind %d", kind)
	}
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
