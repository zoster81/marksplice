package splice

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

// PrepareReplaceStrikethrough prepares a source-preserving replacement of one simple GFM strikethrough's plain-text content.
func (d *Document) PrepareReplaceStrikethrough(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindStrikethrough, "strikethrough")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}
	mapping := target.StrikethroughSource
	change, candidate, err := d.prepareCandidateChange(target.ContentRange, replacement, "strikethrough replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateStrikethroughReplacement(candidate, target, mapping, len(replacement)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareReplaceAutoLink prepares a source-preserving replacement of one angle or bare autolink token.
func (d *Document) PrepareReplaceAutoLink(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.targetNode(id, KindAutoLink)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	mapping, err := source.MapAutoLink(d.source, target.Anchor, target.ContentRange, target.Value, target.AutoLinkEmail)
	if err != nil {
		return ChangeSet{}, fmt.Errorf("map autolink source: %w", err)
	}
	change, candidate, err := d.prepareCandidateChange(target.ContentRange, replacement, "autolink replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateAutoLinkReplacement(candidate, target, mapping, replacement); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareReplaceCodeSpan prepares a source-preserving replacement of one simple single-line code span.
func (d *Document) PrepareReplaceCodeSpan(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindCodeSpan, "code span")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}
	mapping := target.CodeSpanSource
	change, candidate, err := d.prepareCandidateChange(target.ContentRange, replacement, "code span replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateCodeSpanReplacement(candidate, target, mapping, len(replacement)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareReplaceEmphasis prepares a source-preserving replacement of one simple emphasis span.
func (d *Document) PrepareReplaceEmphasis(id NodeID, replacement []byte) (ChangeSet, error) {
	return d.prepareReplaceEmphasisLike(id, replacement, KindEmphasis, parser.KindEmphasis, 1)
}

// PrepareReplaceStrong prepares a source-preserving replacement of one simple strong span.
func (d *Document) PrepareReplaceStrong(id NodeID, replacement []byte) (ChangeSet, error) {
	return d.prepareReplaceEmphasisLike(id, replacement, KindStrong, parser.KindStrong, 2)
}

func (d *Document) prepareReplaceEmphasisLike(id NodeID, replacement []byte, kind Kind, parserKind parser.Kind, level int) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, kind, "emphasis")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}
	mapping := target.EmphasisSource
	change, candidate, err := d.prepareCandidateChange(target.ContentRange, replacement, "emphasis replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateEmphasisReplacement(candidate, target, mapping, parserKind, level, len(replacement)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func validateAutoLinkReplacement(candidate []byte, target Node, original source.AutoLinkMapping, replacement []byte) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := len(replacement) - (target.ContentRange.End - target.ContentRange.Start)
	for _, observation := range observations {
		if observation.Kind != parser.KindAutoLink || observation.Anchor != target.Anchor || observation.Value != string(replacement) || observation.AutoLinkEmail != target.AutoLinkEmail {
			continue
		}
		mapping, err := source.MapAutoLink(candidate, observation.Anchor, Range{Start: observation.Range.Start, End: observation.Range.End}, observation.Value, observation.AutoLinkEmail)
		if err != nil {
			continue
		}
		if mapping.Range == shiftedEnd(original.Range, delta) &&
			mapping.ContentRange == rangeWithLength(original.ContentRange.Start, len(replacement)) &&
			mapping.Angle == original.Angle && mapping.Email == original.Email {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func validateCodeSpanReplacement(candidate []byte, target Node, original source.CodeSpanMapping, replacementLength int) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := replacementLength - (target.ContentRange.End - target.ContentRange.Start)
	for _, observation := range observations {
		if observation.Kind != parser.KindCodeSpan || observation.Anchor != target.Anchor {
			continue
		}
		mapping, err := source.MapSimpleCodeSpan(candidate, observation.Anchor, Range{Start: observation.Range.Start, End: observation.Range.End})
		if err != nil {
			continue
		}
		if mapping.Range == shiftedEnd(original.Range, delta) &&
			mapping.ContentRange == rangeWithLength(original.ContentRange.Start, replacementLength) &&
			mapping.FenceLength == original.FenceLength {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func validateEmphasisReplacement(candidate []byte, target Node, original source.EmphasisMapping, expectedKind parser.Kind, level, replacementLength int) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := replacementLength - (target.ContentRange.End - target.ContentRange.Start)
	for _, observation := range observations {
		if observation.Kind != expectedKind || observation.Anchor != target.Anchor || observation.Level != level {
			continue
		}
		mapping, err := source.MapSimpleEmphasis(candidate, observation.Anchor, Range{Start: observation.Range.Start, End: observation.Range.End}, level)
		if err != nil {
			continue
		}
		if mapping.Range == shiftedEnd(original.Range, delta) &&
			mapping.ContentRange == rangeWithLength(original.ContentRange.Start, replacementLength) &&
			mapping.Marker == original.Marker && mapping.Level == original.Level {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func validateStrikethroughReplacement(candidate []byte, target Node, original source.StrikethroughMapping, replacementLength int) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := replacementLength - (target.ContentRange.End - target.ContentRange.Start)
	for _, observation := range observations {
		if observation.Kind != parser.KindStrikethrough || observation.Range.Start != original.ContentRange.Start {
			continue
		}
		mapping, err := source.MapSimpleStrikethrough(candidate, Range{Start: observation.Range.Start, End: observation.Range.End})
		if err != nil {
			continue
		}
		if mapping.Range == shiftedEnd(original.Range, delta) &&
			mapping.ContentRange == rangeWithLength(original.ContentRange.Start, replacementLength) &&
			mapping.DelimiterLength == original.DelimiterLength {
			return nil
		}
	}
	return ErrInvalidReplacement
}
