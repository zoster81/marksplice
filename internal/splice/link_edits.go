package splice

import (
	"bytes"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

// PrepareReplaceInlineLinkDestination prepares a source-preserving destination replacement for one simple inline link.
func (d *Document) PrepareReplaceInlineLinkDestination(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindInlineLink, "inline link")
	if err != nil {
		return ChangeSet{}, err
	}
	if change, ok := d.unchangedRangeChange(target.ContentRange, replacement); ok {
		return change, nil
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	mapping, ok := remapInlineLinkSource(d.source, target)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
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

// PrepareReplaceInlineLinkLabel prepares a source-preserving replacement of one simple inline-link label payload.
func (d *Document) PrepareReplaceInlineLinkLabel(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindInlineLink, "inline link")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := remapInlineLinkSource(d.source, target)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if change, ok := d.unchangedRangeChange(mapping.LabelRange, replacement); ok {
		return change, nil
	}
	change, candidate, err := d.prepareCandidateChange(mapping.LabelRange, replacement, "inline link label replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateInlineLinkLabelReplacement(candidate, target, mapping, replacement); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareReplaceInlineLinkTitle prepares a source-preserving replacement of one existing simple inline-link title payload.
func (d *Document) PrepareReplaceInlineLinkTitle(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindInlineLink, "inline link")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := remapInlineLinkSource(d.source, target)
	if !ok || !mapping.HasTitle || mapping.TitleRange.Start == mapping.TitleRange.End {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if change, ok := d.unchangedRangeChange(mapping.TitleRange, replacement); ok {
		return change, nil
	}
	change, candidate, err := d.prepareCandidateChange(mapping.TitleRange, replacement, "inline link title replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateInlineLinkTitleReplacement(candidate, target, mapping, replacement); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareReplaceImageDestination prepares a source-preserving destination replacement for one simple inline image.
func (d *Document) PrepareReplaceImageDestination(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindImage, "image")
	if err != nil {
		return ChangeSet{}, err
	}
	if change, ok := d.unchangedRangeChange(target.ContentRange, replacement); ok {
		return change, nil
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	mapping, ok := remapImageSource(d.source, target)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(mapping.DestinationRange, replacement, "image destination replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateImageDestinationReplacement(candidate, target, mapping, replacement); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareReplaceImageAlt prepares a source-preserving replacement of one simple inline-image alt payload.
func (d *Document) PrepareReplaceImageAlt(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindImage, "image")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := remapImageSource(d.source, target)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if change, ok := d.unchangedRangeChange(mapping.AltRange, replacement); ok {
		return change, nil
	}
	change, candidate, err := d.prepareCandidateChange(mapping.AltRange, replacement, "image alt replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateImageAltReplacement(candidate, target, mapping, replacement); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareReplaceImageTitle prepares a source-preserving replacement of one existing simple inline-image title payload.
func (d *Document) PrepareReplaceImageTitle(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindImage, "image")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := remapImageSource(d.source, target)
	if !ok || !mapping.HasTitle || mapping.TitleRange.Start == mapping.TitleRange.End {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if change, ok := d.unchangedRangeChange(mapping.TitleRange, replacement); ok {
		return change, nil
	}
	change, candidate, err := d.prepareCandidateChange(mapping.TitleRange, replacement, "image title replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateImageTitleReplacement(candidate, target, mapping, replacement); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareReplaceReferenceDefinitionDestination prepares a source-preserving destination replacement for one single-line reference definition.
func (d *Document) PrepareReplaceReferenceDefinitionDestination(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindReferenceDefinition, "reference definition")
	if err != nil {
		return ChangeSet{}, err
	}
	if change, ok := d.unchangedRangeChange(target.ContentRange, replacement); ok {
		return change, nil
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	mapping, ok := remapReferenceDefinitionSource(d.source, target)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
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

// PrepareReplaceReferenceDefinitionTitle prepares a source-preserving replacement of one existing single-line reference-definition title payload.
func (d *Document) PrepareReplaceReferenceDefinitionTitle(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindReferenceDefinition, "reference definition")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	mapping, ok := remapReferenceDefinitionSource(d.source, target)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if !mapping.HasTitle || mapping.TitleRange.Start == mapping.TitleRange.End {
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(mapping.TitleRange, replacement, "reference definition title replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateReferenceDefinitionTitleReplacement(candidate, target, mapping, replacement); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareRemoveReferenceDefinition prepares removal of one complete source-owned single-line reference definition.
func (d *Document) PrepareRemoveReferenceDefinition(id NodeID) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindReferenceDefinition, "reference definition")
	if err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := remapReferenceDefinitionSource(d.source, target)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	removeRange := mapping.LineRange
	if !removeRange.Valid(len(d.source)) || removeRange.Start >= removeRange.End || !rangesOverlap(target.Range, removeRange) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(removeRange, nil, "reference definition removal")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateNodeSurvivorsAfterRemoval(candidate, removeRange); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func validateInlineLinkLabelReplacement(candidate []byte, target Node, original source.InlineLinkMapping, replacement []byte) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := len(replacement) - (original.LabelRange.End - original.LabelRange.Start)
	for _, observation := range observations {
		mapping, ok := matchingInlineLinkMapping(candidate, observation, target)
		if ok && inlineLinkLabelMappingMatches(mapping, original, len(replacement), delta) {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func matchingInlineLinkMapping(candidate []byte, observation parser.Node, target Node) (source.InlineLinkMapping, bool) {
	if observation.Kind != parser.KindInlineLink || observation.Anchor != target.Anchor ||
		observation.Destination != target.Destination || observation.Title != target.Title || observation.HasTitle != target.HasTitle {
		return source.InlineLinkMapping{}, false
	}
	mapping, err := source.MapSimpleInlineLink(candidate, observation.Anchor, Range{Start: observation.Range.Start, End: observation.Range.End}, observation.Destination, observation.Title, observation.HasTitle)
	return mapping, err == nil
}

func inlineLinkLabelMappingMatches(mapping, original source.InlineLinkMapping, replacementLength, delta int) bool {
	if mapping.Range != shiftedEnd(original.Range, delta) || mapping.LabelRange != rangeWithLength(original.LabelRange.Start, replacementLength) {
		return false
	}
	if mapping.DestinationRange != shiftedRange(original.DestinationRange, delta) || mapping.AngleDestination != original.AngleDestination {
		return false
	}
	if mapping.HasTitle != original.HasTitle {
		return false
	}
	return !original.HasTitle || mapping.TitleRange == shiftedRange(original.TitleRange, delta)
}

func validateInlineLinkTitleReplacement(candidate []byte, target Node, original source.InlineLinkMapping, replacement []byte) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := len(replacement) - (original.TitleRange.End - original.TitleRange.Start)
	for _, observation := range observations {
		if observation.Kind != parser.KindInlineLink || observation.Anchor != target.Anchor || observation.Destination != target.Destination || observation.Title != string(replacement) || !observation.HasTitle {
			continue
		}
		mapping, err := source.MapSimpleInlineLink(candidate, observation.Anchor, Range{Start: observation.Range.Start, End: observation.Range.End}, observation.Destination, observation.Title, observation.HasTitle)
		if err != nil {
			continue
		}
		if mapping.Range == shiftedEnd(original.Range, delta) && mapping.LabelRange == original.LabelRange &&
			mapping.DestinationRange == original.DestinationRange && mapping.TitleRange == rangeWithLength(original.TitleRange.Start, len(replacement)) &&
			mapping.AngleDestination == original.AngleDestination && mapping.HasTitle == original.HasTitle {
			return nil
		}
	}
	return ErrInvalidReplacement
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

func validateImageDestinationReplacement(candidate []byte, target Node, original source.ImageMapping, replacement []byte) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := len(replacement) - (original.DestinationRange.End - original.DestinationRange.Start)
	for _, observation := range observations {
		if observation.Kind != parser.KindImage || observation.Anchor != target.Anchor {
			continue
		}
		mapping, err := source.MapSimpleImage(candidate, observation.Anchor, Range{Start: observation.Range.Start, End: observation.Range.End})
		if err != nil {
			continue
		}
		titleMatches := !original.HasTitle || mapping.TitleRange == (Range{Start: original.TitleRange.Start + delta, End: original.TitleRange.End + delta})
		if mapping.Range == shiftedEnd(original.Range, delta) &&
			mapping.AltRange == original.AltRange &&
			mapping.DestinationRange == rangeWithLength(original.DestinationRange.Start, len(replacement)) &&
			mapping.AngleDestination == original.AngleDestination && mapping.HasTitle == original.HasTitle && titleMatches {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func validateImageAltReplacement(candidate []byte, target Node, original source.ImageMapping, replacement []byte) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := len(replacement) - (original.AltRange.End - original.AltRange.Start)
	for _, observation := range observations {
		mapping, ok := matchingImageMapping(candidate, observation, target)
		if ok && imageAltMappingMatches(mapping, original, len(replacement), delta) {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func matchingImageMapping(candidate []byte, observation parser.Node, target Node) (source.ImageMapping, bool) {
	if observation.Kind != parser.KindImage || observation.Anchor != target.Anchor {
		return source.ImageMapping{}, false
	}
	mapping, err := source.MapSimpleImage(candidate, observation.Anchor, Range{Start: observation.Range.Start, End: observation.Range.End})
	return mapping, err == nil
}

func imageAltMappingMatches(mapping, original source.ImageMapping, replacementLength, delta int) bool {
	if mapping.Range != shiftedEnd(original.Range, delta) || mapping.AltRange != rangeWithLength(original.AltRange.Start, replacementLength) {
		return false
	}
	if mapping.DestinationRange != shiftedRange(original.DestinationRange, delta) || mapping.AngleDestination != original.AngleDestination {
		return false
	}
	if mapping.HasTitle != original.HasTitle {
		return false
	}
	return !original.HasTitle || mapping.TitleRange == shiftedRange(original.TitleRange, delta)
}

func validateImageTitleReplacement(candidate []byte, target Node, original source.ImageMapping, replacement []byte) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := len(replacement) - (original.TitleRange.End - original.TitleRange.Start)
	for _, observation := range observations {
		mapping, ok := matchingImageMapping(candidate, observation, target)
		if ok && imageTitleMappingMatches(candidate, mapping, original, replacement, delta) {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func imageTitleMappingMatches(candidate []byte, mapping, original source.ImageMapping, replacement []byte, delta int) bool {
	if mapping.Range != shiftedEnd(original.Range, delta) || mapping.AltRange != original.AltRange {
		return false
	}
	if mapping.DestinationRange != original.DestinationRange || mapping.TitleRange != rangeWithLength(original.TitleRange.Start, len(replacement)) {
		return false
	}
	if mapping.AngleDestination != original.AngleDestination || mapping.HasTitle != original.HasTitle || !mapping.TitleRange.Valid(len(candidate)) {
		return false
	}
	return bytes.Equal(candidate[mapping.TitleRange.Start:mapping.TitleRange.End], replacement)
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
		if mapping.Range == shiftedEnd(original.Range, delta) && mapping.LineRange == shiftedEnd(original.LineRange, delta) &&
			mapping.DestinationRange == rangeWithLength(original.DestinationRange.Start, len(replacement)) &&
			mapping.AngleDestination == original.AngleDestination && mapping.HasTitle == original.HasTitle {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func validateReferenceDefinitionTitleReplacement(candidate []byte, target Node, original source.ReferenceDefinitionMapping, replacement []byte) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := len(replacement) - (original.TitleRange.End - original.TitleRange.Start)
	for _, observation := range observations {
		if observation.Kind != parser.KindReferenceDefinition || observation.Label != target.Label || observation.Destination != target.Destination || observation.Title != string(replacement) || !observation.HasTitle {
			continue
		}
		mapping, err := source.MapSingleLineReferenceDefinition(candidate, Range{Start: observation.Range.Start, End: observation.Range.End}, observation.Label, observation.Destination, observation.Title, observation.HasTitle)
		if err != nil {
			continue
		}
		if mapping.Range == shiftedEnd(original.Range, delta) && mapping.LineRange == shiftedEnd(original.LineRange, delta) &&
			mapping.DestinationRange == original.DestinationRange && mapping.TitleRange == rangeWithLength(original.TitleRange.Start, len(replacement)) &&
			mapping.AngleDestination == original.AngleDestination && mapping.HasTitle == original.HasTitle {
			return nil
		}
	}
	return ErrInvalidReplacement
}
