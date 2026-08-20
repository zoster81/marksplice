package splice

import (
	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

// PrepareReplace prepares a minimal replacement for a parsed paragraph node.
func (d *Document) PrepareReplace(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.targetNode(id, KindParagraph)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateParagraphReplacement(replacement); err != nil {
		return ChangeSet{}, err
	}
	return d.newChange(target.Range, replacement, "paragraph replacement")
}

// PrepareReplaceListItem prepares a source-preserving replacement of one single-line list item's content.
func (d *Document) PrepareReplaceListItem(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.targetNode(id, KindListItem)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	change, candidate, err := d.prepareCandidateChange(target.ContentRange, replacement, "list item replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateListItemReplacement(candidate, target, len(replacement)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareReplaceTableCell prepares a source-preserving replacement of one non-empty GFM table cell.
func (d *Document) PrepareReplaceTableCell(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindTableCell, "table cell")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}
	mapping := target.TableCellSource
	change, candidate, err := d.prepareCandidateChange(target.ContentRange, replacement, "table cell replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	delta := len(replacement) - (target.ContentRange.End - target.ContentRange.Start)
	if err := validateTableCellReplacement(candidate, target, mapping, delta); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareReplaceFencedCode prepares a source-preserving replacement of one single-line fenced code block's content.
func (d *Document) PrepareReplaceFencedCode(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindFencedCode, "fenced code")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}
	mapping := target.FencedCodeSource
	change, candidate, err := d.prepareCandidateChange(target.ContentRange, replacement, "fenced code replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateFencedCodeReplacement(candidate, target, mapping, len(replacement)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareRenameHeading prepares a source-preserving replacement of top-level heading content.
func (d *Document) PrepareRenameHeading(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.targetNode(id, KindHeading)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}

	change, candidate, err := d.prepareCandidateChange(target.ContentRange, replacement, "heading rename")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateRenamedHeading(candidate, target); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareSetTaskChecked prepares a one-byte GFM task checkbox state change.
func (d *Document) PrepareSetTaskChecked(id NodeID, checked bool) (ChangeSet, error) {
	target, err := d.targetNode(id, KindTask)
	if err != nil {
		return ChangeSet{}, err
	}
	if target.Checked == checked {
		return source.NewChangeSet(d.source, nil)
	}

	state := byte(' ')
	if checked {
		state = 'x'
	}
	change, candidate, err := d.prepareCandidateChange(target.ContentRange, []byte{state}, "task state change")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateTaskState(candidate, target, checked); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func validateFencedCodeReplacement(candidate []byte, target Node, original source.FencedCodeMapping, replacementLength int) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	delta := replacementLength - (target.ContentRange.End - target.ContentRange.Start)
	for _, observation := range observations {
		if observation.Kind != parser.KindFencedCode {
			continue
		}
		mapping, err := source.MapSingleLineFencedCode(candidate, Range{Start: observation.Range.Start, End: observation.Range.End})
		if err != nil {
			continue
		}
		if mapping.Range == shiftedEnd(original.Range, delta) &&
			mapping.ContentRange == rangeWithLength(original.ContentRange.Start, replacementLength) &&
			mapping.InfoRange == original.InfoRange &&
			mapping.FenceChar == original.FenceChar &&
			mapping.FenceLength == original.FenceLength &&
			mapping.ClosingFenceLength == original.ClosingFenceLength &&
			mapping.OpeningIndent == original.OpeningIndent &&
			mapping.ClosingIndent == original.ClosingIndent {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func validateTableCellReplacement(candidate []byte, target Node, original source.TableCellMapping, delta int) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	for _, observation := range observations {
		if observation.Kind != parser.KindTableCell || observation.TableHeader != target.TableHeader || observation.TableColumn != target.TableColumn {
			continue
		}
		mapping, err := source.MapTableCell(candidate, Range{Start: observation.Range.Start, End: observation.Range.End}, observation.TableColumn)
		if err == nil && mapping.Range == shiftedEnd(original.Range, delta) {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func validateListItemReplacement(candidate []byte, target Node, replacementLength int) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	for _, observation := range observations {
		if observation.Kind != parser.KindListItem || observation.Range.Start != target.ContentRange.Start {
			continue
		}
		mapping, err := source.MapSingleLineListItem(candidate, Range{Start: observation.Range.Start, End: observation.Range.End}, observation.Ordered, observation.Marker)
		if err != nil {
			continue
		}
		if mapping.Range.Start == target.Range.Start &&
			mapping.ContentRange == rangeWithLength(target.ContentRange.Start, replacementLength) &&
			mapping.Ordered == target.ListOrdered && mapping.Marker == target.ListMarker {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func validateTaskState(candidate []byte, target Node, checked bool) error {
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
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
	observations, err := parseCandidate(candidate)
	if err != nil {
		return err
	}
	for _, observation := range observations {
		if observation.Kind != parser.KindHeading || observation.Level != target.Level || observation.Range.Start != target.ContentRange.Start {
			continue
		}
		mapping, err := source.MapTopLevelHeading(candidate, Range{Start: observation.Range.Start, End: observation.Range.End}, observation.Level)
		if err == nil && HeadingStyle(mapping.Style) == target.HeadingStyle && mapping.Range.Start == target.Range.Start {
			return nil
		}
	}
	return ErrInvalidReplacement
}

func validateParagraphReplacement(replacement []byte) error {
	if len(replacement) == 0 {
		return ErrInvalidReplacement
	}

	observations, err := parseCandidate(replacement)
	if err != nil {
		return err
	}
	paragraphs := 0
	for _, observation := range observations {
		switch observation.Kind {
		case parser.KindParagraph:
			if observation.Range.Start != 0 || observation.Range.End != len(replacement) {
				return ErrInvalidReplacement
			}
			paragraphs++
		case parser.KindStrikethrough, parser.KindInlineLink, parser.KindAutoLink, parser.KindCodeSpan, parser.KindEmphasis, parser.KindStrong, parser.KindRawHTML:
			// Nested inline observations are compatible with a single paragraph replacement.
		default:
			return ErrInvalidReplacement
		}
	}
	if paragraphs != 1 {
		return ErrInvalidReplacement
	}
	return nil
}
