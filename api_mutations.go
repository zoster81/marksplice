package marksplice

import "github.com/zoster81/marksplice/internal/splice"

// ComposeChanges combines already-prepared mutations from this exact document
// snapshot into one atomic source-bound change. Overlapping or semantically
// interacting prepared mutations fail closed.
func (d *Document) ComposeChanges(changes ...ChangeSet) (ChangeSet, error) {
	if d == nil || d.document == nil {
		return ChangeSet{}, ErrSourceConflict
	}
	internal := make([]splice.ChangeSet, len(changes))
	for index, change := range changes {
		internal[index] = change.change
	}
	return publicChangeSet(d.document.ComposeChanges(internal...))
}

// PrepareReplaceParagraph prepares a source-preserving paragraph replacement.
func (d *Document) PrepareReplaceParagraph(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindParagraph, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplace(internalNodeID(id), replacement))
}

// PrepareRemoveParagraph prepares removal of one complete promoted top-level paragraph.
func (d *Document) PrepareRemoveParagraph(id NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindParagraph, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRemoveParagraph(internalNodeID(id)))
}

// PrepareInsertParagraphBefore prepares insertion of one paragraph payload before a promoted top-level paragraph.
func (d *Document) PrepareInsertParagraphBefore(id NodeID, content []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindParagraph, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareInsertParagraphBefore(internalNodeID(id), content))
}

// PrepareInsertParagraphAfter prepares insertion of one paragraph payload after a promoted top-level paragraph.
func (d *Document) PrepareInsertParagraphAfter(id NodeID, content []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindParagraph, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareInsertParagraphAfter(internalNodeID(id), content))
}

// PrepareRenameHeading prepares a source-preserving rename of promoted heading content.
func (d *Document) PrepareRenameHeading(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRenameHeading(internalNodeID(id), replacement))
}

// PrepareSetHeadingLevel prepares a source-preserving level change for one promoted top-level heading.
func (d *Document) PrepareSetHeadingLevel(id NodeID, level int) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareSetHeadingLevel(internalNodeID(id), level))
}

// PrepareRemoveThematicBreak prepares source-preserving removal of one complete promoted top-level thematic-break line.
func (d *Document) PrepareRemoveThematicBreak(id NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindThematicBreak, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRemoveThematicBreak(internalNodeID(id)))
}

// PrepareRemoveBlockquote prepares source-preserving removal of one complete promoted top-level blockquote container.
func (d *Document) PrepareRemoveBlockquote(id NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindBlockquote, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRemoveBlockquote(internalNodeID(id)))
}

// PrepareReplaceBlockquoteContent replaces the complete inner content of one
// promoted non-alert blockquote when its physical lines share one source-proven
// marker prefix and line-ending style.
func (d *Document) PrepareReplaceBlockquoteContent(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindBlockquote, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceBlockquoteContent(internalNodeID(id), replacement))
}

// PrepareSetAlertKind changes the reviewed GitHub alert kind while preserving
// the blockquote marker prefix, body source, spacing, and line endings.
func (d *Document) PrepareSetAlertKind(id NodeID, kind AlertKind) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindBlockquote, true); err != nil {
		return ChangeSet{}, err
	}
	if _, ok := d.Alert(id); !ok {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	marker, ok := alertMarker(kind)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return publicChangeSet(d.document.PrepareSetAlertMarker(internalNodeID(id), []byte(marker)))
}

// PrepareReplaceAlertBody replaces all body lines after the alert marker when
// those lines share one source-proven blockquote marker prefix and EOL style.
func (d *Document) PrepareReplaceAlertBody(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindBlockquote, true); err != nil {
		return ChangeSet{}, err
	}
	if _, ok := d.Alert(id); !ok {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	return publicChangeSet(d.document.PrepareReplaceAlertBody(internalNodeID(id), replacement))
}

// PrepareRemoveSection prepares source-preserving removal of one complete promoted section subtree.
func (d *Document) PrepareRemoveSection(headingID NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(headingID, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRemoveSection(internalNodeID(headingID)))
}

// PrepareReplaceSection prepares source-preserving replacement of one complete promoted section subtree.
func (d *Document) PrepareReplaceSection(headingID NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(headingID, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceSection(internalNodeID(headingID), replacement))
}

// PrepareInsertSectionBefore prepares insertion of one sibling section subtree immediately before the target section.
func (d *Document) PrepareInsertSectionBefore(headingID NodeID, fragment []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(headingID, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareInsertSectionBefore(internalNodeID(headingID), fragment))
}

// PrepareInsertSectionAfter prepares insertion of one sibling section subtree immediately after the target section subtree.
func (d *Document) PrepareInsertSectionAfter(headingID NodeID, fragment []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(headingID, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareInsertSectionAfter(internalNodeID(headingID), fragment))
}

// PrepareAppendSectionChild prepares appending one direct child section subtree to a promoted parent section.
func (d *Document) PrepareAppendSectionChild(parentHeadingID NodeID, fragment []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(parentHeadingID, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareAppendSectionChild(internalNodeID(parentHeadingID), fragment))
}

// PrepareMoveSectionBefore prepares moving one complete promoted section subtree immediately before a same-level anchor section.
func (d *Document) PrepareMoveSectionBefore(headingID, anchorHeadingID NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(headingID, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	if _, err := d.promotedNode(anchorHeadingID, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareMoveSectionBefore(internalNodeID(headingID), internalNodeID(anchorHeadingID)))
}

// PrepareMoveSectionAfter prepares moving one complete promoted section subtree immediately after a same-level anchor subtree.
func (d *Document) PrepareMoveSectionAfter(headingID, anchorHeadingID NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(headingID, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	if _, err := d.promotedNode(anchorHeadingID, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareMoveSectionAfter(internalNodeID(headingID), internalNodeID(anchorHeadingID)))
}

// PrepareReplaceSectionBody prepares source-preserving replacement of one promoted section's direct body.
func (d *Document) PrepareReplaceSectionBody(headingID NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(headingID, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceSectionBody(internalNodeID(headingID), replacement))
}

// PrepareReplaceListItem prepares a source-preserving replacement of promoted list-item content.
func (d *Document) PrepareReplaceListItem(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindListItem, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceListItem(internalNodeID(id), replacement))
}

// PrepareReplaceListItemSubtree prepares replacement of one complete supported list-item subtree while preserving its external sibling shape and semantic parent.
func (d *Document) PrepareReplaceListItemSubtree(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindListItem, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceListItemSubtree(internalNodeID(id), replacement))
}

// PrepareRemoveListItem prepares removal of one complete supported list-item subtree.
func (d *Document) PrepareRemoveListItem(id NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindListItem, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRemoveListItem(internalNodeID(id)))
}

// PrepareInsertListItemBefore prepares insertion of one complete same-shape supported list-item subtree immediately before a complete supported anchor subtree.
func (d *Document) PrepareInsertListItemBefore(anchorID NodeID, fragment []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(anchorID, splice.KindListItem, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareInsertListItemBefore(internalNodeID(anchorID), fragment))
}

// PrepareInsertListItemAfter prepares insertion of one complete same-shape supported list-item subtree immediately after a complete supported anchor subtree.
func (d *Document) PrepareInsertListItemAfter(anchorID NodeID, fragment []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(anchorID, splice.KindListItem, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareInsertListItemAfter(internalNodeID(anchorID), fragment))
}

// PrepareAppendFirstListItemChild prepares the first direct child item from caller-owned content. Marksplice derives the source-proven container prefix/indentation and emits canonical '-' or '1.' child marker syntax.
func (d *Document) PrepareAppendFirstListItemChild(parentID NodeID, content []byte, ordered bool) (ChangeSet, error) {
	if _, err := d.promotedNode(parentID, splice.KindListItem, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareAppendFirstListItemChild(internalNodeID(parentID), content, ordered))
}

// PrepareAppendListItemChild prepares appending one complete direct-child subtree to a fully supported list-item subtree.
func (d *Document) PrepareAppendListItemChild(parentID NodeID, fragment []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(parentID, splice.KindListItem, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareAppendListItemChild(internalNodeID(parentID), fragment))
}

// PrepareMoveListItemBefore prepares moving one complete supported list-item subtree immediately before a complete same-shape anchor subtree.
func (d *Document) PrepareMoveListItemBefore(id, anchorID NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindListItem, false); err != nil {
		return ChangeSet{}, err
	}
	if _, err := d.promotedNode(anchorID, splice.KindListItem, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareMoveListItemBefore(internalNodeID(id), internalNodeID(anchorID)))
}

// PrepareMoveListItemAfter prepares moving one complete supported list-item subtree immediately after a complete same-shape anchor subtree.
func (d *Document) PrepareMoveListItemAfter(id, anchorID NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindListItem, false); err != nil {
		return ChangeSet{}, err
	}
	if _, err := d.promotedNode(anchorID, splice.KindListItem, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareMoveListItemAfter(internalNodeID(id), internalNodeID(anchorID)))
}

// PrepareSetTaskChecked prepares a source-preserving GFM task state change.
func (d *Document) PrepareSetTaskChecked(id NodeID, checked bool) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindTask, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareSetTaskChecked(internalNodeID(id), checked))
}

// PrepareSetTableColumnAlignment prepares a source-preserving alignment change for one promoted GFM table column.
func (d *Document) PrepareSetTableColumnAlignment(id NodeID, column int, alignment TableAlignment) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindTable, false); err != nil {
		return ChangeSet{}, err
	}
	internalAlignment, ok := internalTableAlignment(alignment)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return publicChangeSet(d.document.PrepareSetTableColumnAlignment(internalNodeID(id), column, internalAlignment))
}

// PrepareSetTableAlignments prepares one atomic source-preserving alignment update for every promoted GFM table column.
func (d *Document) PrepareSetTableAlignments(id NodeID, alignments []TableAlignment) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindTable, false); err != nil {
		return ChangeSet{}, err
	}
	internalAlignments := make([]splice.TableAlignment, len(alignments))
	for index, alignment := range alignments {
		converted, ok := internalTableAlignment(alignment)
		if !ok {
			return ChangeSet{}, ErrInvalidReplacement
		}
		internalAlignments[index] = converted
	}
	return publicChangeSet(d.document.PrepareSetTableAlignments(internalNodeID(id), internalAlignments))
}

// PrepareAppendTableRow prepares appending one caller-owned compatible body row to a promoted GFM table.
func (d *Document) PrepareAppendTableRow(id NodeID, fragment []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindTable, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareAppendTableRow(internalNodeID(id), fragment))
}

// PrepareInsertTableColumn prepares source-preserving insertion of one complete promoted GFM table column.
func (d *Document) PrepareInsertTableColumn(id NodeID, column int, header []byte, alignment TableAlignment, body [][]byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindTable, false); err != nil {
		return ChangeSet{}, err
	}
	internalAlignment, ok := internalTableAlignment(alignment)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return publicChangeSet(d.document.PrepareInsertTableColumn(internalNodeID(id), column, header, internalAlignment, body))
}

// PrepareRemoveTableColumn prepares source-preserving removal of one complete promoted GFM table column.
func (d *Document) PrepareRemoveTableColumn(id NodeID, column int) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindTable, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRemoveTableColumn(internalNodeID(id), column))
}

// PrepareMoveTableColumn prepares moving one complete promoted GFM table column to a new zero-based position.
func (d *Document) PrepareMoveTableColumn(id NodeID, from, to int) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindTable, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareMoveTableColumn(internalNodeID(id), from, to))
}

// PrepareReplaceTableRow prepares source-preserving replacement of one complete promoted GFM table body row.
func (d *Document) PrepareReplaceTableRow(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindTableRow, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceTableRow(internalNodeID(id), replacement))
}

// PrepareRemoveTableRow prepares source-preserving removal of one promoted GFM table body row.
func (d *Document) PrepareRemoveTableRow(id NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindTableRow, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRemoveTableRow(internalNodeID(id)))
}

// PrepareInsertTableRowBefore prepares insertion of one complete compatible body row before a promoted row.
func (d *Document) PrepareInsertTableRowBefore(anchorID NodeID, fragment []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(anchorID, splice.KindTableRow, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareInsertTableRowBefore(internalNodeID(anchorID), fragment))
}

// PrepareInsertTableRowAfter prepares insertion of one complete compatible body row after a promoted row.
func (d *Document) PrepareInsertTableRowAfter(anchorID NodeID, fragment []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(anchorID, splice.KindTableRow, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareInsertTableRowAfter(internalNodeID(anchorID), fragment))
}

// PrepareMoveTableRowBefore prepares moving one complete body row before another promoted row in the same table.
func (d *Document) PrepareMoveTableRowBefore(id, anchorID NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindTableRow, false); err != nil {
		return ChangeSet{}, err
	}
	if _, err := d.promotedNode(anchorID, splice.KindTableRow, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareMoveTableRowBefore(internalNodeID(id), internalNodeID(anchorID)))
}

// PrepareMoveTableRowAfter prepares moving one complete body row after another promoted row in the same table.
func (d *Document) PrepareMoveTableRowAfter(id, anchorID NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindTableRow, false); err != nil {
		return ChangeSet{}, err
	}
	if _, err := d.promotedNode(anchorID, splice.KindTableRow, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareMoveTableRowAfter(internalNodeID(id), internalNodeID(anchorID)))
}

// PrepareReplaceTableCell prepares a source-preserving replacement of promoted table-cell content.
func (d *Document) PrepareReplaceTableCell(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindTableCell, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceTableCell(internalNodeID(id), replacement))
}

// PrepareReplaceFencedCode prepares a source-preserving replacement of promoted fenced-code content.
// It also populates a source-proven empty closed fenced block while preserving its fence trivia.
func (d *Document) PrepareReplaceFencedCode(id NodeID, replacement []byte) (ChangeSet, error) {
	if err := d.sourceProvenFencedBlock(id); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceFencedCode(internalNodeID(id), replacement))
}

// PrepareSetFencedBlockInfo prepares a source-preserving set, replacement, or clear
// of the parser-proven info string on one source-proven top-level fenced block.
func (d *Document) PrepareSetFencedBlockInfo(id NodeID, info []byte) (ChangeSet, error) {
	if err := d.sourceProvenFencedBlock(id); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareSetFencedBlockInfo(internalNodeID(id), info))
}

// PrepareReplaceStrikethrough prepares a source-preserving replacement of promoted strikethrough content.
func (d *Document) PrepareReplaceStrikethrough(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindStrikethrough, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceStrikethrough(internalNodeID(id), replacement))
}

// PrepareReplaceCodeSpan prepares a source-preserving replacement of promoted code-span content.
func (d *Document) PrepareReplaceCodeSpan(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindCodeSpan, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceCodeSpan(internalNodeID(id), replacement))
}

// PrepareReplaceEmphasis prepares a source-preserving replacement of promoted emphasis content.
func (d *Document) PrepareReplaceEmphasis(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindEmphasis, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceEmphasis(internalNodeID(id), replacement))
}

// PrepareReplaceStrong prepares a source-preserving replacement of promoted strong-emphasis content.
func (d *Document) PrepareReplaceStrong(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindStrong, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceStrong(internalNodeID(id), replacement))
}

// PrepareReplaceInlineLinkDestination prepares a source-preserving replacement of a promoted inline-link destination.
func (d *Document) PrepareReplaceInlineLinkDestination(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindInlineLink, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceInlineLinkDestination(internalNodeID(id), replacement))
}

// PrepareReplaceInlineLinkLabel prepares a source-preserving replacement of a promoted simple inline-link label payload.
func (d *Document) PrepareReplaceInlineLinkLabel(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindInlineLink, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceInlineLinkLabel(internalNodeID(id), replacement))
}

// PrepareReplaceInlineLinkTitle prepares a source-preserving replacement of an existing promoted simple inline-link title payload.
func (d *Document) PrepareReplaceInlineLinkTitle(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindInlineLink, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceInlineLinkTitle(internalNodeID(id), replacement))
}

// PrepareAddInlineLinkTitle inserts a canonical double-quoted title into a promoted simple inline link that currently has no title.
func (d *Document) PrepareAddInlineLinkTitle(id NodeID, title []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindInlineLink, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareAddInlineLinkTitle(internalNodeID(id), title))
}

// PrepareRemoveInlineLinkTitle removes the delimiters and payload of an existing promoted simple inline-link title while preserving separator whitespace.
func (d *Document) PrepareRemoveInlineLinkTitle(id NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindInlineLink, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRemoveInlineLinkTitle(internalNodeID(id)))
}

// PrepareReplaceImageDestination prepares a source-preserving replacement of a promoted image destination.
func (d *Document) PrepareReplaceImageDestination(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindImage, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceImageDestination(internalNodeID(id), replacement))
}

// PrepareReplaceImageAlt prepares a source-preserving replacement of a promoted simple inline-image alt payload.
func (d *Document) PrepareReplaceImageAlt(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindImage, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceImageAlt(internalNodeID(id), replacement))
}

// PrepareReplaceImageTitle prepares a source-preserving replacement of an existing promoted simple inline-image title payload.
func (d *Document) PrepareReplaceImageTitle(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindImage, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceImageTitle(internalNodeID(id), replacement))
}

// PrepareAddImageTitle inserts a canonical double-quoted title into a promoted simple inline image that currently has no title.
func (d *Document) PrepareAddImageTitle(id NodeID, title []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindImage, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareAddImageTitle(internalNodeID(id), title))
}

// PrepareRemoveImageTitle removes the delimiters and payload of an existing promoted simple inline-image title while preserving separator whitespace.
func (d *Document) PrepareRemoveImageTitle(id NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindImage, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRemoveImageTitle(internalNodeID(id)))
}

// PrepareReplaceFootnoteDefinitionBody prepares a source-preserving replacement
// of the conservative simple editable body of one promoted footnote definition.
func (d *Document) PrepareReplaceFootnoteDefinitionBody(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindFootnoteDefinition, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceFootnoteDefinitionBody(internalNodeID(id), replacement))
}

// PrepareReplaceFootnoteDefinitionBodyMultiline replaces one source-proven footnote body from logical LF-separated content while preserving its container EOL/indentation style.
func (d *Document) PrepareReplaceFootnoteDefinitionBodyMultiline(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindFootnoteDefinition, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceFootnoteDefinitionBodyMultiline(internalNodeID(id), replacement))
}

// PrepareAppendFootnoteDefinition appends one canonical top-level footnote definition from logical LF-separated body content.
func (d *Document) PrepareAppendFootnoteDefinition(label, body []byte) (ChangeSet, error) {
	if d == nil || d.document == nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return publicChangeSet(d.document.PrepareAppendFootnoteDefinition(label, body))
}

// PrepareRemoveFootnoteDefinition removes one complete promoted top-level footnote definition while preserving source occurrences outside its owned container.
func (d *Document) PrepareRemoveFootnoteDefinition(id NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindFootnoteDefinition, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRemoveFootnoteDefinition(internalNodeID(id)))
}

// PrepareRenameFootnote atomically renames one promoted footnote definition and
// every parser-proven reference occurrence bound to that definition.
func (d *Document) PrepareRenameFootnote(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindFootnoteDefinition, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRenameFootnote(internalNodeID(id), replacement))
}

// PrepareRetargetReferenceOccurrence retargets one parser-proven reference link/image occurrence at sourceOffset to an existing uniquely normalized definition.
func (d *Document) PrepareRetargetReferenceOccurrence(sourceOffset int, reference []byte) (ChangeSet, error) {
	if d == nil || d.document == nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return publicChangeSet(d.document.PrepareRetargetReferenceOccurrence(sourceOffset, reference))
}

// PrepareRenameReferenceDefinition atomically renames one promoted reference definition and every parser-proven occurrence bound to it.
func (d *Document) PrepareRenameReferenceDefinition(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindReferenceDefinition, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRenameReferenceDefinition(internalNodeID(id), replacement))
}

// PrepareAppendReferenceDefinition appends one canonical single-line reference definition using the document's proven line-ending style.
func (d *Document) PrepareAppendReferenceDefinition(label, destination []byte) (ChangeSet, error) {
	if d == nil || d.document == nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return publicChangeSet(d.document.PrepareAppendReferenceDefinition(label, destination))
}

// PrepareAppendReferenceDefinitionWithTitle appends one canonical single-line reference definition with a double-quoted title.
func (d *Document) PrepareAppendReferenceDefinitionWithTitle(label, destination, title []byte) (ChangeSet, error) {
	if d == nil || d.document == nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return publicChangeSet(d.document.PrepareAppendReferenceDefinitionWithTitle(label, destination, title))
}

// PrepareAddReferenceDefinitionTitle inserts a canonical double-quoted title into a promoted single-line reference definition that currently has no title.
func (d *Document) PrepareAddReferenceDefinitionTitle(id NodeID, title []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindReferenceDefinition, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareAddReferenceDefinitionTitle(internalNodeID(id), title))
}

// PrepareRemoveReferenceDefinitionTitle removes the source-owned title syntax of an existing promoted single-line reference definition.
func (d *Document) PrepareRemoveReferenceDefinitionTitle(id NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindReferenceDefinition, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRemoveReferenceDefinitionTitle(internalNodeID(id)))
}

// PrepareReplaceReferenceDefinitionDestination prepares a source-preserving replacement of a promoted reference-definition destination.
func (d *Document) PrepareReplaceReferenceDefinitionDestination(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindReferenceDefinition, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceReferenceDefinitionDestination(internalNodeID(id), replacement))
}

// PrepareReplaceReferenceDefinitionTitle prepares a source-preserving replacement of an existing promoted reference-definition title payload.
func (d *Document) PrepareReplaceReferenceDefinitionTitle(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindReferenceDefinition, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceReferenceDefinitionTitle(internalNodeID(id), replacement))
}

// PrepareRemoveReferenceDefinition prepares source-preserving removal of one complete promoted single-line reference-definition line.
func (d *Document) PrepareRemoveReferenceDefinition(id NodeID) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindReferenceDefinition, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRemoveReferenceDefinition(internalNodeID(id)))
}

// PrepareReplaceAutoLink prepares a source-preserving replacement of a promoted GFM autolink token.
func (d *Document) PrepareReplaceAutoLink(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindAutoLink, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceAutoLink(internalNodeID(id), replacement))
}

// PrepareRenameFrontMatterField renames one promoted simple YAML/TOML front-matter field key while preserving its value wrapper and line trivia.
func (d *Document) PrepareRenameFrontMatterField(id NodeID, key []byte) (ChangeSet, error) {
	if _, err := d.promotedNodeKinds(id, false, splice.KindYAMLFrontMatterField, splice.KindTOMLFrontMatterField); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRenameFrontMatterField(internalNodeID(id), key))
}

// PrepareRemoveFrontMatterField removes one complete promoted simple front-matter field physical line.
func (d *Document) PrepareRemoveFrontMatterField(id NodeID) (ChangeSet, error) {
	if _, err := d.promotedNodeKinds(id, false, splice.KindYAMLFrontMatterField, splice.KindTOMLFrontMatterField); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRemoveFrontMatterField(internalNodeID(id)))
}

// PrepareAppendFrontMatterField appends one canonical double-quoted simple field before the existing front-matter closing delimiter.
func (d *Document) PrepareAppendFrontMatterField(key, value []byte) (ChangeSet, error) {
	if d == nil || d.document == nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return publicChangeSet(d.document.PrepareAppendFrontMatterField(key, value))
}

// PrepareAddFrontMatter inserts one empty leading YAML or TOML front-matter envelope using the document's source-proven line ending.
func (d *Document) PrepareAddFrontMatter(format FrontMatterFormat) (ChangeSet, error) {
	if d == nil || d.document == nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	internal, ok := internalFrontMatterFormat(format)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return publicChangeSet(d.document.PrepareAddFrontMatter(internal))
}

// PrepareRemoveFrontMatter removes the complete leading front-matter envelope and its owned separator before the Markdown body.
func (d *Document) PrepareRemoveFrontMatter() (ChangeSet, error) {
	if d == nil || d.document == nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return publicChangeSet(d.document.PrepareRemoveFrontMatter())
}

// PrepareReplaceFrontMatterValue prepares a source-preserving replacement of a promoted simple front-matter scalar value.
func (d *Document) PrepareReplaceFrontMatterValue(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNodeKinds(id, false, splice.KindYAMLFrontMatterField, splice.KindTOMLFrontMatterField); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceFrontMatterValue(internalNodeID(id), replacement))
}

// PrepareReplaceHTMLComment prepares a source-preserving replacement of a promoted HTML comment payload.
func (d *Document) PrepareReplaceHTMLComment(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindHTMLComment, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceHTMLComment(internalNodeID(id), replacement))
}

// PrepareReplaceHTMLAnchor prepares a source-preserving replacement of a promoted HTML anchor id/name value.
func (d *Document) PrepareReplaceHTMLAnchor(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindHTMLAnchor, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceHTMLAnchor(internalNodeID(id), replacement))
}

func publicChangeSet(change splice.ChangeSet, err error) (ChangeSet, error) {
	if err != nil {
		return ChangeSet{}, publicError(err)
	}
	return ChangeSet{change: change}, nil
}
