package splice

import (
	"errors"
	"fmt"
	"slices"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

// FootnoteReference is one immutable parser-proven reference to a footnote definition.
type FootnoteReference struct {
	Range         Range
	LabelRange    Range
	Label         string
	DefinitionID  NodeID
	HasDefinition bool
	Occurrence    int
}

func footnoteDefinitionNode(snapshot []byte, fingerprint source.Fingerprint, observation parser.FootnoteDefinitionObservation, footnoteSources *[]source.FootnoteDefinitionMapping) (Node, bool, error) {
	bodyRanges := make([]Range, len(observation.BodyRanges))
	for index, range_ := range observation.BodyRanges {
		bodyRanges[index] = Range{Start: range_.Start, End: range_.End}
	}
	mapping, err := source.MapTopLevelFootnoteDefinition(snapshot, observation.Anchor, observation.Label, bodyRanges)
	if err != nil {
		if errors.Is(err, source.ErrUnsupportedFootnoteShape) {
			return Node{}, false, nil
		}
		return Node{}, false, fmt.Errorf("map footnote definition source: %w", err)
	}
	index, ok := appendSourceDetail(footnoteSources, mapping)
	if !ok {
		return Node{}, false, fmt.Errorf("map footnote definition source: sidecar capacity exceeded")
	}
	node := Node{
		Kind:              KindFootnoteDefinition,
		Range:             mapping.Range,
		ContentRange:      mapping.BodyRange,
		SourceDetailIndex: index,
		Anchor:            observation.Anchor,
		Label:             observation.Label,
		TopLevel:          true,
		Editable:          true,
	}
	node.ID = makeNodeID(fingerprint, node.Kind, node.Range)
	return node, true, nil
}

func promoteFootnoteDefinitionNodes(snapshot []byte, fingerprint source.Fingerprint, definitions []parser.FootnoteDefinitionObservation, footnoteSources *[]source.FootnoteDefinitionMapping) ([]Node, error) {
	result := make([]Node, 0, len(definitions))
	for _, definition := range definitions {
		node, ok, err := footnoteDefinitionNode(snapshot, fingerprint, definition, footnoteSources)
		if err != nil {
			return nil, err
		}
		if ok {
			result = append(result, node)
		}
	}
	return result, nil
}

func footnoteDefinitionsOutsideRange(definitions []parser.FootnoteDefinitionObservation, excluded Range) []parser.FootnoteDefinitionObservation {
	result := make([]parser.FootnoteDefinitionObservation, 0, len(definitions))
	for _, definition := range definitions {
		if definition.Anchor >= excluded.Start && definition.Anchor < excluded.End {
			continue
		}
		result = append(result, definition)
	}
	return result
}

func footnoteReferencesOutsideRange(references []parser.FootnoteReferenceObservation, excluded Range) []parser.FootnoteReferenceObservation {
	result := make([]parser.FootnoteReferenceObservation, 0, len(references))
	for _, reference := range references {
		if reference.Range.Start >= excluded.Start && reference.Range.Start < excluded.End {
			continue
		}
		result = append(result, reference)
	}
	return result
}

func mergeSourceOrderedNodes(nodes, additions []Node) []Node {
	if len(additions) == 0 {
		return nodes
	}
	originalLen := len(nodes)
	nodes = slices.Grow(nodes, len(additions))
	nodes = nodes[:originalLen+len(additions)]
	nodeIndex := originalLen - 1
	additionIndex := len(additions) - 1
	for writeIndex := len(nodes) - 1; writeIndex >= 0 && additionIndex >= 0; writeIndex-- {
		if nodeIndex >= 0 && nodes[nodeIndex].Range.Start > additions[additionIndex].Range.Start {
			nodes[writeIndex] = nodes[nodeIndex]
			nodeIndex--
			continue
		}
		nodes[writeIndex] = additions[additionIndex]
		additionIndex--
	}
	return nodes
}

func resolveFootnoteReferences(nodes []Node, observed []parser.FootnoteReferenceObservation) []FootnoteReference {
	definitionIDs := make(map[int]NodeID)
	for _, node := range nodes {
		if node.Kind == KindFootnoteDefinition {
			definitionIDs[node.Anchor] = node.ID
		}
	}
	result := make([]FootnoteReference, len(observed))
	for index, reference := range observed {
		result[index] = FootnoteReference{
			Range:         Range{Start: reference.Range.Start, End: reference.Range.End},
			LabelRange:    Range{Start: reference.LabelRange.Start, End: reference.LabelRange.End},
			Label:         reference.Label,
			Occurrence:    reference.Occurrence,
			DefinitionID:  definitionIDs[reference.DefinitionAnchor],
			HasDefinition: definitionIDs[reference.DefinitionAnchor] != "",
		}
	}
	return result
}

// FootnoteReferences returns caller-owned parser-proven footnote relationships in source order.
func (d *Document) FootnoteReferences() []FootnoteReference {
	if d == nil {
		return nil
	}
	return append([]FootnoteReference(nil), d.footnoteReferences...)
}

func (d *Document) footnoteSource(node Node) (source.FootnoteDefinitionMapping, bool) {
	if d == nil || node.Kind != KindFootnoteDefinition || !node.Editable || !node.TopLevel {
		return source.FootnoteDefinitionMapping{}, false
	}
	index, ok := sourceDetailIndex(node.SourceDetailIndex, len(d.footnoteSources))
	if !ok {
		return source.FootnoteDefinitionMapping{}, false
	}
	mapping := d.footnoteSources[index]
	if !validFootnoteSourceMapping(d.source, node, mapping) {
		return source.FootnoteDefinitionMapping{}, false
	}
	return mapping, true
}

func validFootnoteSourceMapping(input []byte, node Node, mapping source.FootnoteDefinitionMapping) bool {
	if mapping.Range != node.Range || mapping.BodyRange != node.ContentRange || !mapping.Range.Valid(len(input)) || mapping.Range.Start >= mapping.Range.End {
		return false
	}
	if !mapping.LabelRange.Valid(len(input)) || mapping.LabelRange.Start >= mapping.LabelRange.End {
		return false
	}
	return sourceRangesWithin(mapping.BodyRanges, mapping.Range, len(input))
}

// FootnoteSource returns a caller-owned source mapping for one promoted top-level footnote definition.
func (d *Document) FootnoteSource(id NodeID) (source.FootnoteDefinitionMapping, bool) {
	node, ok := d.nodeByID(id)
	if !ok {
		return source.FootnoteDefinitionMapping{}, false
	}
	mapping, ok := d.footnoteSource(node)
	if !ok {
		return source.FootnoteDefinitionMapping{}, false
	}
	mapping.BodyRanges = append([]source.Range(nil), mapping.BodyRanges...)
	return mapping, true
}

// FootnoteDefinitionBodyRanges returns caller-owned parser-proven semantic body ranges.
func (d *Document) FootnoteDefinitionBodyRanges(id NodeID) ([]Range, bool) {
	if d == nil {
		return nil, false
	}
	node, ok := d.nodeByID(id)
	if !ok {
		return nil, false
	}
	mapping, ok := d.footnoteSource(node)
	if !ok {
		return nil, false
	}
	return append([]Range(nil), mapping.BodyRanges...), true
}
