package splice

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

// PrepareReplaceInlineLinkDestination prepares a source-preserving destination replacement for one simple inline link.
func (d *Document) PrepareReplaceInlineLinkDestination(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.targetNode(id, KindInlineLink)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	mapping, err := source.MapSimpleInlineLink(d.source, target.Anchor, target.ContentRange, target.Destination, target.Title, target.HasTitle)
	if err != nil {
		return ChangeSet{}, fmt.Errorf("map inline link source: %w", err)
	}
	change, candidate, err := d.prepareCandidateChange(mapping.DestinationRange, replacement, "inline link destination replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateInlineLinkDestinationReplacement(candidate, target, mapping, replacement); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareReplaceReferenceDefinitionDestination prepares a source-preserving destination replacement for one single-line reference definition.
func (d *Document) PrepareReplaceReferenceDefinitionDestination(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.targetNode(id, KindReferenceDefinition)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	mapping, err := source.MapSingleLineReferenceDefinition(d.source, target.Range, target.Label, target.Destination, target.Title, target.HasTitle)
	if err != nil {
		return ChangeSet{}, fmt.Errorf("map reference definition source: %w", err)
	}
	change, candidate, err := d.prepareCandidateChange(mapping.DestinationRange, replacement, "reference definition destination replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateReferenceDefinitionDestinationReplacement(candidate, target, mapping, replacement); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func validateInlineLinkDestinationReplacement(candidate []byte, target Node, original source.InlineLinkMapping, replacement []byte) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := len(replacement) - (original.DestinationRange.End - original.DestinationRange.Start)
	for _, observation := range observations {
		if observation.Kind != parser.KindInlineLink || observation.Anchor != target.Anchor || observation.Destination != string(replacement) || observation.Title != target.Title || observation.HasTitle != target.HasTitle {
			continue
		}
		mapping, err := source.MapSimpleInlineLink(candidate, observation.Anchor, Range{Start: observation.Range.Start, End: observation.Range.End}, observation.Destination, observation.Title, observation.HasTitle)
		if err != nil {
			continue
		}
		if mapping.Range == shiftedEnd(original.Range, delta) &&
			mapping.LabelRange == original.LabelRange &&
			mapping.DestinationRange == rangeWithLength(original.DestinationRange.Start, len(replacement)) &&
			mapping.AngleDestination == original.AngleDestination && mapping.HasTitle == original.HasTitle {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func validateReferenceDefinitionDestinationReplacement(candidate []byte, target Node, original source.ReferenceDefinitionMapping, replacement []byte) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := len(replacement) - (original.DestinationRange.End - original.DestinationRange.Start)
	for _, observation := range observations {
		if observation.Kind != parser.KindReferenceDefinition || observation.Label != target.Label || observation.Destination != string(replacement) || observation.Title != target.Title || observation.HasTitle != target.HasTitle {
			continue
		}
		mapping, err := source.MapSingleLineReferenceDefinition(candidate, Range{Start: observation.Range.Start, End: observation.Range.End}, observation.Label, observation.Destination, observation.Title, observation.HasTitle)
		if err != nil {
			continue
		}
		if mapping.Range == shiftedEnd(original.Range, delta) &&
			mapping.DestinationRange == rangeWithLength(original.DestinationRange.Start, len(replacement)) &&
			mapping.AngleDestination == original.AngleDestination && mapping.HasTitle == original.HasTitle {
			return nil
		}
	}
	return ErrInvalidReplacement
}
