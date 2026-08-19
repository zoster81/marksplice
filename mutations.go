package marksplice

import "github.com/zoster81/marksplice/internal/splice"

// PrepareReplaceParagraph prepares a source-preserving paragraph replacement.
func (d *Document) PrepareReplaceParagraph(id NodeID, replacement []byte) (ChangeSet, error) {
	if err := d.validateMutationTarget(id, splice.KindParagraph, true); err != nil {
		return ChangeSet{}, err
	}
	change, err := d.document.PrepareReplace(internalNodeID(id), replacement)
	if err != nil {
		return ChangeSet{}, publicError(err)
	}
	return ChangeSet{change: change}, nil
}

// PrepareRenameHeading prepares a source-preserving rename of promoted heading content.
func (d *Document) PrepareRenameHeading(id NodeID, replacement []byte) (ChangeSet, error) {
	if err := d.validateMutationTarget(id, splice.KindHeading, true); err != nil {
		return ChangeSet{}, err
	}
	change, err := d.document.PrepareRenameHeading(internalNodeID(id), replacement)
	if err != nil {
		return ChangeSet{}, publicError(err)
	}
	return ChangeSet{change: change}, nil
}

// PrepareReplaceListItem prepares a source-preserving replacement of promoted list-item content.
func (d *Document) PrepareReplaceListItem(id NodeID, replacement []byte) (ChangeSet, error) {
	if err := d.validateMutationTarget(id, splice.KindListItem, false); err != nil {
		return ChangeSet{}, err
	}
	change, err := d.document.PrepareReplaceListItem(internalNodeID(id), replacement)
	if err != nil {
		return ChangeSet{}, publicError(err)
	}
	return ChangeSet{change: change}, nil
}

// PrepareSetTaskChecked prepares a source-preserving GFM task state change.
func (d *Document) PrepareSetTaskChecked(id NodeID, checked bool) (ChangeSet, error) {
	if err := d.validateMutationTarget(id, splice.KindTask, false); err != nil {
		return ChangeSet{}, err
	}
	change, err := d.document.PrepareSetTaskChecked(internalNodeID(id), checked)
	if err != nil {
		return ChangeSet{}, publicError(err)
	}
	return ChangeSet{change: change}, nil
}

// PrepareReplaceTableCell prepares a source-preserving replacement of promoted table-cell content.
func (d *Document) PrepareReplaceTableCell(id NodeID, replacement []byte) (ChangeSet, error) {
	if err := d.validateMutationTarget(id, splice.KindTableCell, false); err != nil {
		return ChangeSet{}, err
	}
	change, err := d.document.PrepareReplaceTableCell(internalNodeID(id), replacement)
	if err != nil {
		return ChangeSet{}, publicError(err)
	}
	return ChangeSet{change: change}, nil
}

// PrepareReplaceFencedCode prepares a source-preserving replacement of promoted fenced-code content.
func (d *Document) PrepareReplaceFencedCode(id NodeID, replacement []byte) (ChangeSet, error) {
	if err := d.validateMutationTarget(id, splice.KindFencedCode, false); err != nil {
		return ChangeSet{}, err
	}
	change, err := d.document.PrepareReplaceFencedCode(internalNodeID(id), replacement)
	if err != nil {
		return ChangeSet{}, publicError(err)
	}
	return ChangeSet{change: change}, nil
}

// PrepareReplaceStrikethrough prepares a source-preserving replacement of promoted strikethrough content.
func (d *Document) PrepareReplaceStrikethrough(id NodeID, replacement []byte) (ChangeSet, error) {
	if err := d.validateMutationTarget(id, splice.KindStrikethrough, false); err != nil {
		return ChangeSet{}, err
	}
	change, err := d.document.PrepareReplaceStrikethrough(internalNodeID(id), replacement)
	if err != nil {
		return ChangeSet{}, publicError(err)
	}
	return ChangeSet{change: change}, nil
}

// PrepareReplaceCodeSpan prepares a source-preserving replacement of promoted code-span content.
func (d *Document) PrepareReplaceCodeSpan(id NodeID, replacement []byte) (ChangeSet, error) {
	if err := d.validateMutationTarget(id, splice.KindCodeSpan, false); err != nil {
		return ChangeSet{}, err
	}
	change, err := d.document.PrepareReplaceCodeSpan(internalNodeID(id), replacement)
	if err != nil {
		return ChangeSet{}, publicError(err)
	}
	return ChangeSet{change: change}, nil
}

// PrepareReplaceEmphasis prepares a source-preserving replacement of promoted emphasis content.
func (d *Document) PrepareReplaceEmphasis(id NodeID, replacement []byte) (ChangeSet, error) {
	if err := d.validateMutationTarget(id, splice.KindEmphasis, false); err != nil {
		return ChangeSet{}, err
	}
	change, err := d.document.PrepareReplaceEmphasis(internalNodeID(id), replacement)
	if err != nil {
		return ChangeSet{}, publicError(err)
	}
	return ChangeSet{change: change}, nil
}

// PrepareReplaceStrong prepares a source-preserving replacement of promoted strong-emphasis content.
func (d *Document) PrepareReplaceStrong(id NodeID, replacement []byte) (ChangeSet, error) {
	if err := d.validateMutationTarget(id, splice.KindStrong, false); err != nil {
		return ChangeSet{}, err
	}
	change, err := d.document.PrepareReplaceStrong(internalNodeID(id), replacement)
	if err != nil {
		return ChangeSet{}, publicError(err)
	}
	return ChangeSet{change: change}, nil
}

func (d *Document) validateMutationTarget(id NodeID, expected splice.Kind, requireTopLevel bool) error {
	node, ok := d.internalNode(id)
	if !ok {
		return ErrNodeNotFound
	}
	if node.Kind != expected || !node.Editable || requireTopLevel && !node.TopLevel {
		return ErrInvalidTargetKind
	}
	return nil
}
