package splice

import (
	"bytes"
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
)

// Kind identifies a Marksplice structural node kind.
type Kind uint8

const (
	KindUnknown Kind = iota
	KindParagraph
	KindHeading
	KindTask
)

// HeadingStyle identifies the source syntax of a heading.
type HeadingStyle uint8

const (
	HeadingStyleUnknown HeadingStyle = HeadingStyle(source.HeadingStyleUnknown)
	HeadingStyleATX     HeadingStyle = HeadingStyle(source.HeadingStyleATX)
	HeadingStyleSetext  HeadingStyle = HeadingStyle(source.HeadingStyleSetext)
)

// NodeID is deterministic within one source snapshot.
type NodeID string

// Range is a half-open byte range in the source snapshot.
type Range = source.Range

// Node is the minimal Marksplice-owned structural view used by the feasibility slice.
type Node struct {
	ID           NodeID
	Kind         Kind
	Range        Range
	ContentRange Range
	Level        int
	HeadingStyle HeadingStyle
	Checked      bool
}

// ChangeSet is a source-bound prepared mutation.
type ChangeSet = source.ChangeSet

// Document is an immutable parsed source snapshot used by the feasibility slice.
type Document struct {
	source []byte
	nodes  []Node
}

// Parse creates a snapshot-local Marksplice model using the internal Goldmark adapter.
func Parse(input []byte) (*Document, error) {
	semanticParser := goldmarkparser.New()
	snapshot := append([]byte(nil), input...)
	observations, err := semanticParser.Parse(snapshot)
	if err != nil {
		return nil, fmt.Errorf("parse markdown: %w", err)
	}

	fingerprint := source.Sum(snapshot)
	nodes := make([]Node, 0, len(observations))
	for _, observation := range observations {
		node, err := nodeFromObservation(snapshot, fingerprint, observation)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return &Document{
		source: snapshot,
		nodes:  nodes,
	}, nil
}

func nodeFromObservation(snapshot []byte, fingerprint source.Fingerprint, observation parser.Node) (Node, error) {
	kind, err := mapKind(observation.Kind)
	if err != nil {
		return Node{}, err
	}

	contentRange := Range{Start: observation.Range.Start, End: observation.Range.End}
	if !contentRange.Valid(len(snapshot)) {
		return Node{}, fmt.Errorf("semantic node range [%d,%d) is outside source length %d", contentRange.Start, contentRange.End, len(snapshot))
	}

	node := Node{
		Kind:         kind,
		Range:        contentRange,
		ContentRange: contentRange,
		Level:        observation.Level,
		Checked:      observation.Checked,
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
	}
	node.ID = makeNodeID(fingerprint, kind, node.Range)
	return node, nil
}

// Nodes returns a copy of the snapshot-local structural nodes.
func (d *Document) Nodes() []Node {
	return append([]Node(nil), d.nodes...)
}

// PrepareReplace prepares a minimal replacement for a parsed paragraph node.
func (d *Document) PrepareReplace(id NodeID, replacement []byte) (ChangeSet, error) {
	target, ok := d.nodeByID(id)
	if !ok {
		return ChangeSet{}, ErrNodeNotFound
	}
	if err := d.validateReplacement(target.Kind, replacement); err != nil {
		return ChangeSet{}, err
	}

	change, err := source.NewChangeSet(d.source, []source.Patch{{
		Range:       target.Range,
		Replacement: replacement,
	}})
	if err != nil {
		return ChangeSet{}, fmt.Errorf("prepare replacement: %w", err)
	}
	return change, nil
}

// PrepareRenameHeading prepares a source-preserving replacement of top-level heading content.
func (d *Document) PrepareRenameHeading(id NodeID, replacement []byte) (ChangeSet, error) {
	target, ok := d.nodeByID(id)
	if !ok {
		return ChangeSet{}, ErrNodeNotFound
	}
	if target.Kind != KindHeading {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	if len(replacement) == 0 || bytes.ContainsAny(replacement, "\r\n") {
		return ChangeSet{}, ErrInvalidReplacement
	}

	change, err := source.NewChangeSet(d.source, []source.Patch{{
		Range:       target.ContentRange,
		Replacement: replacement,
	}})
	if err != nil {
		return ChangeSet{}, fmt.Errorf("prepare heading rename: %w", err)
	}
	candidate, err := change.Apply(d.source)
	if err != nil {
		return ChangeSet{}, fmt.Errorf("render heading rename candidate: %w", err)
	}
	if err := validateRenamedHeading(candidate, target); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareSetTaskChecked prepares a one-byte GFM task checkbox state change.
func (d *Document) PrepareSetTaskChecked(id NodeID, checked bool) (ChangeSet, error) {
	target, ok := d.nodeByID(id)
	if !ok {
		return ChangeSet{}, ErrNodeNotFound
	}
	if target.Kind != KindTask {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	if target.Checked == checked {
		return source.NewChangeSet(d.source, nil)
	}

	state := byte(' ')
	if checked {
		state = 'x'
	}
	change, err := source.NewChangeSet(d.source, []source.Patch{{
		Range:       target.ContentRange,
		Replacement: []byte{state},
	}})
	if err != nil {
		return ChangeSet{}, fmt.Errorf("prepare task state change: %w", err)
	}
	candidate, err := change.Apply(d.source)
	if err != nil {
		return ChangeSet{}, fmt.Errorf("render task state candidate: %w", err)
	}
	if err := validateTaskState(candidate, target, checked); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func validateTaskState(candidate []byte, target Node, checked bool) error {
	observations, err := goldmarkparser.New().Parse(candidate)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidReplacement, err)
	}
	for _, observation := range observations {
		if observation.Kind != parser.KindTask || observation.Range.Start != target.Range.Start || observation.Checked != checked {
			continue
		}
		mapping, err := source.MapTaskMarker(candidate, observation.Range.Start)
		if err == nil && mapping.Range == target.Range && mapping.Checked == checked {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func validateRenamedHeading(candidate []byte, target Node) error {
	observations, err := goldmarkparser.New().Parse(candidate)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidReplacement, err)
	}
	for _, observation := range observations {
		if observation.Kind != parser.KindHeading || observation.Level != target.Level || observation.Range.Start != target.ContentRange.Start {
			continue
		}
		mapping, err := source.MapTopLevelHeading(candidate, Range{Start: observation.Range.Start, End: observation.Range.End}, observation.Level)
		if err != nil {
			continue
		}
		if HeadingStyle(mapping.Style) == target.HeadingStyle && mapping.Range.Start == target.Range.Start {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func (d *Document) nodeByID(id NodeID) (Node, bool) {
	for _, node := range d.nodes {
		if node.ID == id {
			return node, true
		}
	}
	return Node{}, false
}

func (d *Document) validateReplacement(kind Kind, replacement []byte) error {
	if kind != KindParagraph || len(replacement) == 0 {
		return ErrInvalidReplacement
	}

	observations, err := goldmarkparser.New().Parse(replacement)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidReplacement, err)
	}
	if len(observations) != 1 || observations[0].Kind != parser.KindParagraph || observations[0].Range.Start != 0 || observations[0].Range.End != len(replacement) {
		return ErrInvalidReplacement
	}
	return nil
}

func mapKind(kind parser.Kind) (Kind, error) {
	switch kind {
	case parser.KindParagraph:
		return KindParagraph, nil
	case parser.KindHeading:
		return KindHeading, nil
	case parser.KindTask:
		return KindTask, nil
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
