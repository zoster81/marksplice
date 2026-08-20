package marksplice

import "github.com/zoster81/marksplice/internal/splice"

// PrepareReplaceParagraph prepares a source-preserving paragraph replacement.
func (d *Document) PrepareReplaceParagraph(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindParagraph, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplace(internalNodeID(id), replacement))
}

// PrepareRenameHeading prepares a source-preserving rename of promoted heading content.
func (d *Document) PrepareRenameHeading(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareRenameHeading(internalNodeID(id), replacement))
}

// PrepareReplaceListItem prepares a source-preserving replacement of promoted list-item content.
func (d *Document) PrepareReplaceListItem(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindListItem, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceListItem(internalNodeID(id), replacement))
}

// PrepareSetTaskChecked prepares a source-preserving GFM task state change.
func (d *Document) PrepareSetTaskChecked(id NodeID, checked bool) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindTask, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareSetTaskChecked(internalNodeID(id), checked))
}

// PrepareReplaceTableCell prepares a source-preserving replacement of promoted table-cell content.
func (d *Document) PrepareReplaceTableCell(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindTableCell, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceTableCell(internalNodeID(id), replacement))
}

// PrepareReplaceFencedCode prepares a source-preserving replacement of promoted fenced-code content.
func (d *Document) PrepareReplaceFencedCode(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindFencedCode, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceFencedCode(internalNodeID(id), replacement))
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

// PrepareReplaceReferenceDefinitionDestination prepares a source-preserving replacement of a promoted reference-definition destination.
func (d *Document) PrepareReplaceReferenceDefinitionDestination(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindReferenceDefinition, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceReferenceDefinitionDestination(internalNodeID(id), replacement))
}

// PrepareReplaceAutoLink prepares a source-preserving replacement of a promoted GFM autolink token.
func (d *Document) PrepareReplaceAutoLink(id NodeID, replacement []byte) (ChangeSet, error) {
	if _, err := d.promotedNode(id, splice.KindAutoLink, false); err != nil {
		return ChangeSet{}, err
	}
	return publicChangeSet(d.document.PrepareReplaceAutoLink(internalNodeID(id), replacement))
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
