package splice

import (
	"bytes"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

// PrepareAddInlineLinkTitle prepares insertion of one canonical double-quoted title into a simple direct inline link that currently has no title.
func (d *Document) PrepareAddInlineLinkTitle(id NodeID, title []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindInlineLink, "inline link")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(title); err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := remapInlineLinkSource(d.source, target)
	if !ok || mapping.HasTitle || mapping.Range.End <= mapping.Range.Start || d.source[mapping.Range.End-1] != ')' {
		return ChangeSet{}, ErrInvalidReplacement
	}
	insertAt := mapping.Range.End - 1
	fragment := quotedTitleFragment(title)
	patch := Range{Start: insertAt, End: insertAt}
	change, candidate, err := d.prepareCandidateChange(patch, fragment, "inline link title insertion")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateAddedInlineLinkTitle(candidate, target, mapping, title, insertAt, len(fragment)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareRemoveInlineLinkTitle prepares removal of the delimiters and payload of one existing simple direct inline-link title while preserving authored separator whitespace.
func (d *Document) PrepareRemoveInlineLinkTitle(id NodeID) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindInlineLink, "inline link")
	if err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := remapInlineLinkSource(d.source, target)
	if !ok || !mapping.HasTitle {
		return ChangeSet{}, ErrInvalidReplacement
	}
	remove, ok := ownedInlineTitleSyntaxRange(d.source, mapping.TitleRange, mapping.Range)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(remove, nil, "inline link title removal")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateRemovedInlineLinkTitle(candidate, target, mapping, remove); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareAddImageTitle prepares insertion of one canonical double-quoted title into a simple direct image that currently has no title.
func (d *Document) PrepareAddImageTitle(id NodeID, title []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindImage, "image")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(title); err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := remapImageSource(d.source, target)
	if !ok || mapping.HasTitle || mapping.Range.End <= mapping.Range.Start || d.source[mapping.Range.End-1] != ')' {
		return ChangeSet{}, ErrInvalidReplacement
	}
	insertAt := mapping.Range.End - 1
	fragment := quotedTitleFragment(title)
	patch := Range{Start: insertAt, End: insertAt}
	change, candidate, err := d.prepareCandidateChange(patch, fragment, "image title insertion")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateAddedImageTitle(candidate, target, mapping, title, insertAt, len(fragment)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareRemoveImageTitle prepares removal of the delimiters and payload of one existing simple direct-image title while preserving authored separator whitespace.
func (d *Document) PrepareRemoveImageTitle(id NodeID) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindImage, "image")
	if err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := remapImageSource(d.source, target)
	if !ok || !mapping.HasTitle {
		return ChangeSet{}, ErrInvalidReplacement
	}
	remove, ok := ownedInlineTitleSyntaxRange(d.source, mapping.TitleRange, mapping.Range)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(remove, nil, "image title removal")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateRemovedImageTitle(candidate, target, mapping, remove); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func quotedTitleFragment(title []byte) []byte {
	fragment := make([]byte, 0, len(title)+3)
	fragment = append(fragment, ' ', '"')
	fragment = append(fragment, title...)
	fragment = append(fragment, '"')
	return fragment
}

func ownedInlineTitleSyntaxRange(input []byte, title, owner Range) (Range, bool) {
	if !title.Valid(len(input)) || title.Start >= title.End || !owner.Valid(len(input)) || title.Start <= owner.Start || title.End >= owner.End {
		return Range{}, false
	}
	open := input[title.Start-1]
	close := input[title.End]
	if open == '"' && close != '"' || open == '\'' && close != '\'' || open == '(' && close != ')' {
		return Range{}, false
	}
	if open != '"' && open != '\'' && open != '(' {
		return Range{}, false
	}
	return Range{Start: title.Start - 1, End: title.End + 1}, true
}

func validateAddedInlineLinkTitle(candidate []byte, target Node, original source.InlineLinkMapping, title []byte, insertAt, delta int) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	for _, observation := range observations {
		if observation.Kind != parser.KindInlineLink || observation.Anchor != target.Anchor || observation.Destination != target.Destination || observation.Title != string(title) || !observation.HasTitle {
			continue
		}
		mapping, err := source.MapSimpleInlineLink(candidate, observation.Anchor, Range{Start: observation.Range.Start, End: observation.Range.End}, observation.Destination, observation.Title, observation.HasTitle)
		if err != nil {
			continue
		}
		if addedInlineTitleMappingMatches(mapping, original, title, insertAt, delta, candidate) {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func addedInlineTitleMappingMatches(mapping, original source.InlineLinkMapping, title []byte, insertAt, delta int, candidate []byte) bool {
	expectedTitle := Range{Start: insertAt + 2, End: insertAt + 2 + len(title)}
	return mapping.Range == shiftedEnd(original.Range, delta) && mapping.LabelRange == original.LabelRange &&
		mapping.DestinationRange == original.DestinationRange && mapping.TitleRange == expectedTitle &&
		mapping.AngleDestination == original.AngleDestination && mapping.HasTitle &&
		expectedTitle.Valid(len(candidate)) && bytes.Equal(candidate[expectedTitle.Start:expectedTitle.End], title)
}

func validateRemovedInlineLinkTitle(candidate []byte, target Node, original source.InlineLinkMapping, removed Range) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := -(removed.End - removed.Start)
	for _, observation := range observations {
		if observation.Kind != parser.KindInlineLink || observation.Anchor != target.Anchor || observation.Destination != target.Destination || observation.HasTitle {
			continue
		}
		mapping, err := source.MapSimpleInlineLink(candidate, observation.Anchor, Range{Start: observation.Range.Start, End: observation.Range.End}, observation.Destination, "", false)
		if err == nil && mapping.Range == shiftedEnd(original.Range, delta) && mapping.LabelRange == original.LabelRange &&
			mapping.DestinationRange == original.DestinationRange && mapping.AngleDestination == original.AngleDestination && !mapping.HasTitle {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func validateAddedImageTitle(candidate []byte, target Node, original source.ImageMapping, title []byte, insertAt, delta int) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	for _, observation := range observations {
		if observation.Kind != parser.KindImage || observation.Anchor != target.Anchor {
			continue
		}
		mapping, err := source.MapSimpleImage(candidate, observation.Anchor, Range{Start: observation.Range.Start, End: observation.Range.End})
		if err != nil {
			continue
		}
		expectedTitle := Range{Start: insertAt + 2, End: insertAt + 2 + len(title)}
		if mapping.Range == shiftedEnd(original.Range, delta) && mapping.AltRange == original.AltRange && mapping.DestinationRange == original.DestinationRange &&
			mapping.TitleRange == expectedTitle && mapping.AngleDestination == original.AngleDestination && mapping.HasTitle &&
			expectedTitle.Valid(len(candidate)) && bytes.Equal(candidate[expectedTitle.Start:expectedTitle.End], title) {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func validateRemovedImageTitle(candidate []byte, target Node, original source.ImageMapping, removed Range) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := -(removed.End - removed.Start)
	for _, observation := range observations {
		if observation.Kind != parser.KindImage || observation.Anchor != target.Anchor {
			continue
		}
		mapping, err := source.MapSimpleImage(candidate, observation.Anchor, Range{Start: observation.Range.Start, End: observation.Range.End})
		if err == nil && mapping.Range == shiftedEnd(original.Range, delta) && mapping.AltRange == original.AltRange &&
			mapping.DestinationRange == original.DestinationRange && mapping.AngleDestination == original.AngleDestination && !mapping.HasTitle {
			return nil
		}
	}
	return ErrInvalidReplacement
}
