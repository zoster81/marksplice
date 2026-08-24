package splice

import "github.com/zoster81/marksplice/internal/parser"

// UnresolvedReference is one conservative explicit full/collapsed reference
// usage for which the parser context contains no matching definition.
type UnresolvedReference struct {
	SourceOffset int
	Reference    string
	Form         ReferenceForm
	Image        bool
}

// UnresolvedReferences returns parser-proven conservative unresolved explicit
// reference usages in source order. Shortcut bracket text is intentionally absent.
func (d *Document) UnresolvedReferences() ([]UnresolvedReference, bool) {
	if d == nil {
		return nil, true
	}
	result := make([]UnresolvedReference, 0, len(d.unresolvedReferenceUsages))
	for _, usage := range d.unresolvedReferenceUsages {
		form, ok := relationshipReferenceForm(usage.Form)
		if !ok || (form != ReferenceFormFull && form != ReferenceFormCollapsed) || usage.Anchor < 0 {
			return nil, false
		}
		image, ok := unresolvedReferenceImage(usage.Kind)
		if !ok || usage.Reference == "" {
			return nil, false
		}
		result = append(result, UnresolvedReference{
			SourceOffset: usage.Anchor,
			Reference:    usage.Reference,
			Form:         form,
			Image:        image,
		})
	}
	return result, true
}

func unresolvedReferenceImage(kind parser.Kind) (bool, bool) {
	switch kind {
	case parser.KindInlineLink:
		return false, true
	case parser.KindImage:
		return true, true
	default:
		return false, false
	}
}
