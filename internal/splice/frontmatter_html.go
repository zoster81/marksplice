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
			Editable:          true,
			Key:               field.Key,
			FrontMatterFormat: FrontMatterFormat(field.Format),
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
		node.Editable = true
	} else if mapping, err := source.MapSimpleHTMLAnchor(snapshot, raw); err == nil {
		node.Kind = KindHTMLAnchor
		node.Range = mapping.Range
		node.ContentRange = mapping.ContentRange
		node.HTMLAttribute = mapping.Attribute
		node.HTMLQuote = mapping.Quote
		node.Editable = true
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
	if (target.Kind != KindYAMLFrontMatterField && target.Kind != KindTOMLFrontMatterField) || !target.Editable {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	mapping := d.frontMatter
	if mapping.Format == source.FrontMatterUnknown || mapping.Format != source.FrontMatterFormat(target.FrontMatterFormat) {
		return ChangeSet{}, ErrInvalidReplacement
	}

	change, candidate, err := d.prepareCandidateChange(target.ContentRange, replacement, "front-matter replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateFrontMatterReplacement(candidate, target, mapping, len(replacement)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func validateFrontMatterReplacement(candidate []byte, target Node, original frontMatterEnvelope, replacementLength int) error {
	mapping, ok := source.MapLeadingFrontMatter(candidate)
	if !ok || mapping.Format != original.Format || mapping.OpeningRange != original.OpeningRange {
		return ErrInvalidReplacement
	}
	delta := replacementLength - (target.ContentRange.End - target.ContentRange.Start)
	if mapping.ClosingRange != shiftedEnd(original.ClosingRange, delta) {
		return ErrInvalidReplacement
	}
	for _, field := range mapping.Fields {
		if field.Key != target.Key || field.Format != source.FrontMatterFormat(target.FrontMatterFormat) || field.Style != target.FrontMatterStyle {
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
	target, err := d.editableTargetNode(id, KindHTMLComment, "HTML comment")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	change, candidate, err := d.prepareCandidateChange(target.ContentRange, replacement, "HTML comment replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateHTMLCommentReplacement(candidate, target, len(replacement)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func validateHTMLCommentReplacement(candidate []byte, target Node, replacementLength int) error {
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
			mapping.Range == shiftedEnd(target.Range, delta) &&
			mapping.ContentRange == rangeWithLength(target.ContentRange.Start, replacementLength) {
			return nil
		}
	}
	return ErrInvalidReplacement
}

// PrepareReplaceHTMLAnchor prepares a source-preserving replacement of one simple quoted id/name attribute on an <a> tag.
func (d *Document) PrepareReplaceHTMLAnchor(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindHTMLAnchor, "HTML anchor")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	change, candidate, err := d.prepareCandidateChange(target.ContentRange, replacement, "HTML anchor replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateHTMLAnchorReplacement(candidate, target, len(replacement)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func validateHTMLAnchorReplacement(candidate []byte, target Node, replacementLength int) error {
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
			mapping.Range == shiftedEnd(target.Range, delta) &&
			mapping.ContentRange == rangeWithLength(target.ContentRange.Start, replacementLength) &&
			mapping.Attribute == target.HTMLAttribute && mapping.Quote == target.HTMLQuote {
			return nil
		}
	}
	return ErrInvalidReplacement
}
