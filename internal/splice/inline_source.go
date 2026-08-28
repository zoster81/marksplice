package splice

import "github.com/zoster81/marksplice/internal/source"

func remapStrikethroughSource(input []byte, node Node) (source.StrikethroughMapping, bool) {
	if node.Kind != KindStrikethrough || !node.Editable {
		return source.StrikethroughMapping{}, false
	}
	mapping, err := source.MapSimpleStrikethrough(input, node.ContentRange)
	return mapping, err == nil
}

func remapInlineLinkSource(input []byte, node Node) (source.InlineLinkMapping, bool) {
	if node.Kind != KindInlineLink || !node.Editable {
		return source.InlineLinkMapping{}, false
	}
	mapping, err := source.MapSimpleInlineLink(input, node.Anchor, node.Range, node.Destination, node.Title, node.HasTitle)
	return mapping, err == nil
}

func remapImageSource(input []byte, node Node) (source.ImageMapping, bool) {
	if node.Kind != KindImage || !node.Editable {
		return source.ImageMapping{}, false
	}
	mapping, err := source.MapSimpleImage(input, node.Anchor, node.Range)
	return mapping, err == nil
}

func remapReferenceDefinitionSource(input []byte, node Node) (source.ReferenceDefinitionMapping, bool) {
	if node.Kind != KindReferenceDefinition || !node.Editable {
		return source.ReferenceDefinitionMapping{}, false
	}
	mapping, err := source.MapSingleLineReferenceDefinition(input, node.Range, node.Label, node.Destination, node.Title, node.HasTitle)
	return mapping, err == nil
}

func remapAutoLinkSource(input []byte, node Node) (source.AutoLinkMapping, bool) {
	if node.Kind != KindAutoLink || !node.Editable {
		return source.AutoLinkMapping{}, false
	}
	mapping, err := source.MapAutoLink(input, node.Anchor, node.ContentRange, node.Value, node.AutoLinkEmail)
	return mapping, err == nil
}

func remapCodeSpanSource(input []byte, node Node) (source.CodeSpanMapping, bool) {
	if node.Kind != KindCodeSpan || !node.Editable {
		return source.CodeSpanMapping{}, false
	}
	mapping, err := source.MapSimpleCodeSpan(input, node.Anchor, node.ContentRange)
	return mapping, err == nil
}

func remapEmphasisSource(input []byte, node Node) (source.EmphasisMapping, bool) {
	if (node.Kind != KindEmphasis && node.Kind != KindStrong) || !node.Editable {
		return source.EmphasisMapping{}, false
	}
	mapping, err := source.MapSimpleEmphasis(input, node.Anchor, node.ContentRange, node.Level)
	return mapping, err == nil
}

// InlineLinkSource remaps the exact source capability for one editable inline link.
func (d *Document) InlineLinkSource(id NodeID) (source.InlineLinkMapping, bool) {
	node, ok := d.nodeByID(id)
	if !ok {
		return source.InlineLinkMapping{}, false
	}
	return remapInlineLinkSource(d.source, node)
}

// ImageSource remaps the exact source capability for one editable inline image.
func (d *Document) ImageSource(id NodeID) (source.ImageMapping, bool) {
	node, ok := d.nodeByID(id)
	if !ok {
		return source.ImageMapping{}, false
	}
	return remapImageSource(d.source, node)
}

// ReferenceDefinitionSource remaps the exact source capability for one editable reference definition.
func (d *Document) ReferenceDefinitionSource(id NodeID) (source.ReferenceDefinitionMapping, bool) {
	node, ok := d.nodeByID(id)
	if !ok {
		return source.ReferenceDefinitionMapping{}, false
	}
	return remapReferenceDefinitionSource(d.source, node)
}

// AutoLinkSource remaps the exact source capability for one editable autolink.
func (d *Document) AutoLinkSource(id NodeID) (source.AutoLinkMapping, bool) {
	node, ok := d.nodeByID(id)
	if !ok {
		return source.AutoLinkMapping{}, false
	}
	return remapAutoLinkSource(d.source, node)
}
