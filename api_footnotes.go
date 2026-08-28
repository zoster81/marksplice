package marksplice

import "github.com/zoster81/marksplice/internal/splice"

// FootnoteDefinition is immutable typed detail for one source-proven top-level
// footnote definition. Range owns the complete physical definition container;
// BodyRange is available only for the conservative simple editable subset.
type FootnoteDefinition struct {
	id          NodeID
	sourceRange Range
	labelRange  Range
	bodyRange   Range
	label       string
	hasBody     bool
}

// ID returns the definition's snapshot-scoped structural identity.
func (f FootnoteDefinition) ID() NodeID { return f.id }

// Range returns the exact complete physical source owned by the definition.
func (f FootnoteDefinition) Range() Range { return f.sourceRange }

// Label returns the parser-proven footnote label.
func (f FootnoteDefinition) Label() string { return f.label }

// LabelRange returns the exact source bytes containing Label.
func (f FootnoteDefinition) LabelRange() Range { return f.labelRange }

// BodyRange returns the exact simple body span suitable for source-preserving
// replacement. Segmented or multiline definitions return false; use
// Document.FootnoteDefinitionBodyRanges for read-only semantic body segments.
func (f FootnoteDefinition) BodyRange() (Range, bool) {
	return f.bodyRange, f.hasBody
}

// FootnoteReference is one immutable parser-proven footnote reference occurrence.
// It is relationship data and does not grant generic mutation authority.
type FootnoteReference struct {
	sourceRange   Range
	labelRange    Range
	label         string
	definitionID  NodeID
	hasDefinition bool
	occurrence    int
}

// Range returns the exact `[^label]` source token span.
func (r FootnoteReference) Range() Range { return r.sourceRange }

// LabelRange returns the exact source bytes containing Label.
func (r FootnoteReference) LabelRange() Range { return r.labelRange }

// Label returns the parser-proven footnote label.
func (r FootnoteReference) Label() string { return r.label }

// DefinitionID returns the promoted definition that owns this reference when
// complete top-level source ownership is proven.
func (r FootnoteReference) DefinitionID() (NodeID, bool) {
	return r.definitionID, r.hasDefinition
}

// Occurrence returns the zero-based source-order occurrence for this definition.
func (r FootnoteReference) Occurrence() int { return r.occurrence }

// FootnoteDefinitions returns every source-proven top-level footnote definition
// in physical source order, including unused definitions.
func (d *Document) FootnoteDefinitions() []FootnoteDefinition {
	if d == nil || d.document == nil {
		return nil
	}
	result := make([]FootnoteDefinition, 0)
	for index := 0; index < d.document.NodeCount(); index++ {
		summary, ok := d.document.NodeSummaryAt(index)
		if !ok || summary.Kind != splice.KindFootnoteDefinition {
			continue
		}
		node, ok := d.document.Node(summary.ID)
		if !ok {
			continue
		}
		definition, ok := publicFootnoteDefinition(d.document, node)
		if ok {
			result = append(result, definition)
		}
	}
	return result
}

// FootnoteDefinition returns one source-proven top-level definition by snapshot ID.
func (d *Document) FootnoteDefinition(id NodeID) (FootnoteDefinition, bool) {
	node, ok := d.internalNode(id)
	if !ok {
		return FootnoteDefinition{}, false
	}
	return publicFootnoteDefinition(d.document, node)
}

// FootnoteDefinitionBodyRanges returns caller-owned parser-proven body segments
// in physical source order. They are read-only source metadata, not generic edit spans.
func (d *Document) FootnoteDefinitionBodyRanges(id NodeID) ([]Range, bool) {
	if d == nil || d.document == nil {
		return nil, false
	}
	ranges, ok := d.document.FootnoteDefinitionBodyRanges(internalNodeID(id))
	if !ok {
		return nil, false
	}
	return publicRanges(ranges), true
}

// FootnoteReferences returns every parser-proven footnote reference in source order.
// The returned slice is caller-owned and no relationship index is retained.
func (d *Document) FootnoteReferences() []FootnoteReference {
	if d == nil || d.document == nil {
		return nil
	}
	internal := d.document.FootnoteReferences()
	result := make([]FootnoteReference, len(internal))
	for index, reference := range internal {
		result[index] = FootnoteReference{
			sourceRange:   publicRange(reference.Range),
			labelRange:    publicRange(reference.LabelRange),
			label:         reference.Label,
			definitionID:  publicNodeID(reference.DefinitionID),
			hasDefinition: reference.HasDefinition,
			occurrence:    reference.Occurrence,
		}
	}
	return result
}

func publicFootnoteDefinition(document *splice.Document, node splice.Node) (FootnoteDefinition, bool) {
	if document == nil || node.Kind != splice.KindFootnoteDefinition || !node.Editable || !node.TopLevel {
		return FootnoteDefinition{}, false
	}
	mapping, ok := document.FootnoteSource(node.ID)
	if !ok ||
		mapping.Range.Start >= mapping.Range.End || mapping.LabelRange.Start >= mapping.LabelRange.End {
		return FootnoteDefinition{}, false
	}
	return FootnoteDefinition{
		id:          publicNodeID(node.ID),
		sourceRange: publicRange(mapping.Range),
		labelRange:  publicRange(mapping.LabelRange),
		bodyRange:   publicRange(mapping.BodyRange),
		label:       node.Label,
		hasBody:     mapping.BodyRange.Start < mapping.BodyRange.End,
	}, true
}
