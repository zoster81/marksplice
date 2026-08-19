package splice

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

func frontMatterNodes(fingerprint source.Fingerprint, mapping source.FrontMatterMapping) []Node {
	var kind Kind
	switch mapping.Format {
	case source.FrontMatterYAML:
		kind = KindYAMLFrontMatterField
	case source.FrontMatterTOML:
		kind = KindTOMLFrontMatterField
	default:
		return nil
	}

	nodes := make([]Node, 0, len(mapping.Fields))
	for _, field := range mapping.Fields {
		node := Node{
			Kind:              kind,
			Range:             field.Range,
			ContentRange:      field.ValueRange,
			Key:               field.Key,
			FrontMatterFormat: field.Format,
			FrontMatterStyle:  field.Style,
		}
		node.ID = makeNodeID(fingerprint, kind, node.Range)
		nodes = append(nodes, node)
	}
	return nodes
}

func rangesOverlap(a, b Range) bool {
	return a.Start < b.End && b.Start < a.End
}

func nodeFromRawHTMLObservation(snapshot []byte, fingerprint source.Fingerprint, observation parser.Node) (Node, error) {
	raw := Range{Start: observation.Range.Start, End: observation.Range.End}
	if !raw.Valid(len(snapshot)) || raw.Start == raw.End {
		return Node{}, fmt.Errorf("raw HTML range [%d,%d) is outside source length %d", raw.Start, raw.End, len(snapshot))
	}

	node := Node{Kind: KindHTMLOpaque, Range: raw, ContentRange: raw}
	if mapping, err := source.MapHTMLComment(snapshot, raw); err == nil {
		node.Kind = KindHTMLComment
		node.Range = mapping.Range
		node.ContentRange = mapping.ContentRange
	} else if mapping, err := source.MapSimpleHTMLAnchor(snapshot, raw); err == nil {
		node.Kind = KindHTMLAnchor
		node.Range = mapping.Range
		node.ContentRange = mapping.ContentRange
		node.HTMLAttribute = mapping.Attribute
		node.HTMLQuote = mapping.Quote
	}
	node.ID = makeNodeID(fingerprint, node.Kind, node.Range)
	return node, nil
}

// PrepareReplaceFrontMatterValue prepares a source-preserving replacement of one simple leading YAML/TOML scalar value.
func (d *Document) PrepareReplaceFrontMatterValue(id NodeID, replacement []byte) (ChangeSet, error) {
	target, ok := d.nodeByID(id)
	if !ok {
		return ChangeSet{}, ErrNodeNotFound
	}
	if target.Kind != KindYAMLFrontMatterField && target.Kind != KindTOMLFrontMatterField {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	mapping, ok := source.MapLeadingFrontMatter(d.source)
	if !ok || mapping.Format != target.FrontMatterFormat {
		return ChangeSet{}, ErrInvalidReplacement
	}
	field, ok := frontMatterFieldForTarget(mapping, target)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}

	change, candidate, err := d.prepareCandidateChange(target.ContentRange, replacement, "front-matter replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateFrontMatterReplacement(candidate, target, mapping, field, len(replacement)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func frontMatterFieldForTarget(mapping source.FrontMatterMapping, target Node) (source.FrontMatterFieldMapping, bool) {
	for _, field := range mapping.Fields {
		if field.Key == target.Key && field.Format == target.FrontMatterFormat && field.Style == target.FrontMatterStyle &&
			field.Range == target.Range && field.ValueRange == target.ContentRange {
			return field, true
		}
	}
	return source.FrontMatterFieldMapping{}, false
}

func validateFrontMatterReplacement(candidate []byte, target Node, original source.FrontMatterMapping, originalField source.FrontMatterFieldMapping, replacementLength int) error {
	mapping, ok := source.MapLeadingFrontMatter(candidate)
	if !ok || mapping.Format != original.Format || mapping.OpeningRange != original.OpeningRange {
		return ErrInvalidReplacement
	}
	delta := replacementLength - (originalField.ValueRange.End - originalField.ValueRange.Start)
	if mapping.ClosingRange != shiftedEnd(original.ClosingRange, delta) {
		return ErrInvalidReplacement
	}
	for _, field := range mapping.Fields {
		if field.Key != target.Key || field.Format != target.FrontMatterFormat || field.Style != target.FrontMatterStyle {
			continue
		}
		if field.Range == shiftedEnd(target.Range, delta) &&
			field.ValueRange == rangeWithLength(target.ContentRange.Start, replacementLength) {
			return nil
		}
	}
	return ErrInvalidReplacement
}

// PrepareReplaceHTMLComment prepares a source-preserving replacement of one simple inline HTML comment payload.
func (d *Document) PrepareReplaceHTMLComment(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.targetNode(id, KindHTMLComment)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	mapping, err := source.MapHTMLComment(d.source, target.Range)
	if err != nil || mapping.ContentRange != target.ContentRange {
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(target.ContentRange, replacement, "HTML comment replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateHTMLCommentReplacement(candidate, target, mapping, len(replacement)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func validateHTMLCommentReplacement(candidate []byte, target Node, original source.HTMLCommentMapping, replacementLength int) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := replacementLength - (target.ContentRange.End - target.ContentRange.Start)
	for _, observation := range observations {
		if observation.Kind != parser.KindRawHTML || observation.Range.Start != target.Range.Start {
			continue
		}
		raw := Range{Start: observation.Range.Start, End: observation.Range.End}
		mapping, err := source.MapHTMLComment(candidate, raw)
		if err == nil &&
			mapping.Range == shiftedEnd(original.Range, delta) &&
			mapping.ContentRange == rangeWithLength(original.ContentRange.Start, replacementLength) {
			return nil
		}
	}
	return ErrInvalidReplacement
}

// PrepareReplaceHTMLAnchor prepares a source-preserving replacement of one simple quoted id/name attribute on an <a> tag.
func (d *Document) PrepareReplaceHTMLAnchor(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.targetNode(id, KindHTMLAnchor)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	mapping, err := source.MapSimpleHTMLAnchor(d.source, target.Range)
	if err != nil || mapping.ContentRange != target.ContentRange || mapping.Attribute != target.HTMLAttribute || mapping.Quote != target.HTMLQuote {
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(target.ContentRange, replacement, "HTML anchor replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateHTMLAnchorReplacement(candidate, target, mapping, len(replacement)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func validateHTMLAnchorReplacement(candidate []byte, target Node, original source.HTMLAnchorMapping, replacementLength int) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := replacementLength - (target.ContentRange.End - target.ContentRange.Start)
	for _, observation := range observations {
		if observation.Kind != parser.KindRawHTML || observation.Range.Start != target.Range.Start {
			continue
		}
		raw := Range{Start: observation.Range.Start, End: observation.Range.End}
		mapping, err := source.MapSimpleHTMLAnchor(candidate, raw)
		if err == nil &&
			mapping.Range == shiftedEnd(original.Range, delta) &&
			mapping.ContentRange == rangeWithLength(original.ContentRange.Start, replacementLength) &&
			mapping.Attribute == original.Attribute && mapping.Quote == original.Quote {
			return nil
		}
	}
	return ErrInvalidReplacement
}
