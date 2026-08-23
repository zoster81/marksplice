package marksplice

import (
	"fmt"
	"strings"

	"github.com/zoster81/marksplice/internal/splice"
)

type typedInlineReferenceForm uint8

const (
	typedInlineReferenceFull typedInlineReferenceForm = iota
	typedInlineReferenceCollapsed
	typedInlineReferenceShortcut
)

type typedInlineReferenceScope uint8

const (
	typedInlineReferencePriorExact typedInlineReferenceScope = iota
	typedInlineReferenceDeferredExact
	typedInlineReferenceAvailableNormalized
)

type typedInlineReferenceDefinition struct {
	label       string
	destination string
	title       string
	hasTitle    bool
}

// ReferenceLinkInline returns one conservative full reference-link construction value.
// The exact reference label must identify one already-appended top-level reference
// definition in the destination DocumentBuilder when the value is appended. M92
// permits the same reviewed bounded structured-inline label children as direct links.
func ReferenceLinkInline(reference string, label ...Inline) Inline {
	return newTypedReferenceLinkOrImage(inlineConstructionReferenceLink, typedInlineReferenceFull, typedInlineReferencePriorExact, reference, label)
}

// ForwardReferenceLinkInline returns one full reference-link construction value
// resolved only against one explicitly deferred top-level definition.
func ForwardReferenceLinkInline(reference string, label ...Inline) Inline {
	return newTypedReferenceLinkOrImage(inlineConstructionReferenceLink, typedInlineReferenceFull, typedInlineReferenceDeferredExact, reference, label)
}

// CollapsedReferenceLinkInline returns one `[label][]` construction value. The
// emitted label must resolve to exactly one available normalized definition.
func CollapsedReferenceLinkInline(label ...Inline) Inline {
	return newTypedReferenceLinkOrImage(inlineConstructionReferenceLink, typedInlineReferenceCollapsed, typedInlineReferenceAvailableNormalized, "", label)
}

// ShortcutReferenceLinkInline returns one `[label]` construction value. The
// emitted label must resolve to exactly one available normalized definition.
func ShortcutReferenceLinkInline(label ...Inline) Inline {
	return newTypedReferenceLinkOrImage(inlineConstructionReferenceLink, typedInlineReferenceShortcut, typedInlineReferenceAvailableNormalized, "", label)
}

// ReferenceImageInline returns one conservative full reference-image construction value.
// It follows the same existing exact-definition and M92 structured-alt requirements as
// ReferenceLinkInline.
func ReferenceImageInline(reference string, alt ...Inline) Inline {
	return newTypedReferenceLinkOrImage(inlineConstructionReferenceImage, typedInlineReferenceFull, typedInlineReferencePriorExact, reference, alt)
}

// ForwardReferenceImageInline returns one full reference-image construction
// value resolved only against one explicitly deferred top-level definition.
func ForwardReferenceImageInline(reference string, alt ...Inline) Inline {
	return newTypedReferenceLinkOrImage(inlineConstructionReferenceImage, typedInlineReferenceFull, typedInlineReferenceDeferredExact, reference, alt)
}

// CollapsedReferenceImageInline returns one `![alt][]` construction value.
func CollapsedReferenceImageInline(alt ...Inline) Inline {
	return newTypedReferenceLinkOrImage(inlineConstructionReferenceImage, typedInlineReferenceCollapsed, typedInlineReferenceAvailableNormalized, "", alt)
}

// ShortcutReferenceImageInline returns one `![alt]` construction value.
func ShortcutReferenceImageInline(alt ...Inline) Inline {
	return newTypedReferenceLinkOrImage(inlineConstructionReferenceImage, typedInlineReferenceShortcut, typedInlineReferenceAvailableNormalized, "", alt)
}

func newTypedReferenceLinkOrImage(kind inlineConstructionKind, form typedInlineReferenceForm, scope typedInlineReferenceScope, reference string, content []Inline) Inline {
	return Inline{
		kind:           kind,
		reference:      reference,
		referenceForm:  form,
		referenceScope: scope,
		children:       cloneTypedInlineConstruction(content),
	}
}

func writeTypedInlineReferenceLinkOrImage(output *strings.Builder, inline Inline, context typedInlineWriteContext) error {
	if len(inline.children) == 0 {
		return fmt.Errorf("reference link or image label is empty")
	}
	kind, syntaxStart := typedInlineReferenceOwner(output, inline)
	structured := !typedInlineChildrenAreText(inline.children)
	proofIndex := appendTypedInlineReferenceOwnerProof(context, kind, structured)

	output.WriteByte('[')
	labelStart := output.Len()
	childContext := context
	childContext.policy = typedInlineStructuredNested
	childContext.parent = proofIndex
	for _, child := range inline.children {
		if err := writeTypedInlineConstruction(output, child, childContext); err != nil {
			return err
		}
	}
	labelEnd := output.Len()
	reference, definition, form, err := resolveTypedInlineReference(inline, output.String()[labelStart:labelEnd], context)
	if err != nil {
		return err
	}
	referenceRange := writeTypedInlineReferenceSuffix(output, form, reference)
	if structured {
		(*context.hierarchy)[proofIndex].SyntaxRange = splice.Range{Start: syntaxStart, End: output.Len()}
		(*context.hierarchy)[proofIndex].ContentRange = splice.Range{Start: labelStart, End: labelEnd}
	}
	*context.referenceExpected = append(*context.referenceExpected, splice.ConstructionReferenceInlineExpectation{
		Kind:            kind,
		Form:            form,
		SyntaxRange:     splice.Range{Start: syntaxStart, End: output.Len()},
		LabelRange:      splice.Range{Start: labelStart, End: labelEnd},
		ReferenceRange:  referenceRange,
		Reference:       reference,
		Destination:     definition.destination,
		Title:           definition.title,
		HasTitle:        definition.hasTitle,
		StructuredLabel: structured,
	})
	return nil
}

func typedInlineReferenceOwner(output *strings.Builder, inline Inline) (splice.Kind, int) {
	kind := splice.KindInlineLink
	syntaxStart := output.Len()
	if inline.kind == inlineConstructionReferenceImage {
		kind = splice.KindImage
		output.WriteByte('!')
	}
	return kind, syntaxStart
}

func appendTypedInlineReferenceOwnerProof(context typedInlineWriteContext, kind splice.Kind, structured bool) int {
	if !structured {
		return -1
	}
	index := len(*context.hierarchy)
	*context.hierarchy = append(*context.hierarchy, splice.ConstructionInlineExpectation{
		Kind:   kind,
		Parent: context.parent,
	})
	return index
}

func resolveTypedInlineReference(inline Inline, labelSource string, context typedInlineWriteContext) (string, typedInlineReferenceDefinition, splice.ConstructionReferenceInlineForm, error) {
	form, ok := spliceTypedInlineReferenceForm(inline.referenceForm)
	if !ok {
		return "", typedInlineReferenceDefinition{}, form, fmt.Errorf("unsupported typed reference form")
	}
	if inline.referenceForm != typedInlineReferenceFull {
		definition, err := resolveNormalizedTypedInlineReference(labelSource, availableTypedInlineReferenceDefinitions(context))
		return labelSource, definition, form, err
	}
	if err := validateTypedInlineReference(inline.reference); err != nil {
		return "", typedInlineReferenceDefinition{}, form, err
	}
	definition, err := resolveExactTypedInlineReference(inline.reference, inline.referenceScope, context)
	return inline.reference, definition, form, err
}

func resolveExactTypedInlineReference(reference string, scope typedInlineReferenceScope, context typedInlineWriteContext) (typedInlineReferenceDefinition, error) {
	switch scope {
	case typedInlineReferencePriorExact:
		return resolveExactTypedInlineReferenceFrom(reference, context.referenceDefinitions, context.referenceDefinitions)
	case typedInlineReferenceDeferredExact:
		return resolveExactTypedInlineReferenceFrom(reference, context.deferredReferenceDefinitions, availableTypedInlineReferenceDefinitions(context))
	default:
		return typedInlineReferenceDefinition{}, fmt.Errorf("unsupported typed reference scope")
	}
}

func resolveExactTypedInlineReferenceFrom(reference string, candidates, universe []typedInlineReferenceDefinition) (typedInlineReferenceDefinition, error) {
	var exact typedInlineReferenceDefinition
	count := 0
	for _, definition := range candidates {
		if definition.label != reference {
			continue
		}
		exact = definition
		count++
	}
	if count != 1 {
		return typedInlineReferenceDefinition{}, fmt.Errorf("reference %q must match exactly one definition in its required scope", reference)
	}
	resolved, err := resolveNormalizedTypedInlineReference(reference, universe)
	if err != nil {
		return typedInlineReferenceDefinition{}, err
	}
	if resolved != exact {
		return typedInlineReferenceDefinition{}, fmt.Errorf("reference %q normalized resolution changed", reference)
	}
	return exact, nil
}

func resolveNormalizedTypedInlineReference(reference string, definitions []typedInlineReferenceDefinition) (typedInlineReferenceDefinition, error) {
	resolved, err := splice.ResolveConstructionReference(reference, spliceTypedInlineReferenceDefinitions(definitions))
	if err != nil {
		return typedInlineReferenceDefinition{}, err
	}
	return typedInlineReferenceDefinition{
		label:       resolved.Label,
		destination: resolved.Destination,
		title:       resolved.Title,
		hasTitle:    resolved.HasTitle,
	}, nil
}

func typedInlineReferenceDefinitions(blocks []constructionBlock) []typedInlineReferenceDefinition {
	result := make([]typedInlineReferenceDefinition, 0)
	for _, block := range blocks {
		if block.kind != constructionReferenceDefinition {
			continue
		}
		result = append(result, typedInlineReferenceDefinition{
			label:       block.label,
			destination: block.destination,
			title:       block.title,
			hasTitle:    block.hasTitle,
		})
	}
	return result
}

func availableTypedInlineReferenceDefinitions(context typedInlineWriteContext) []typedInlineReferenceDefinition {
	definitions := make([]typedInlineReferenceDefinition, 0, len(context.referenceDefinitions)+len(context.deferredReferenceDefinitions))
	definitions = append(definitions, context.referenceDefinitions...)
	definitions = append(definitions, context.deferredReferenceDefinitions...)
	return definitions
}

func spliceTypedInlineReferenceDefinitions(definitions []typedInlineReferenceDefinition) []splice.ConstructionReferenceDefinition {
	converted := make([]splice.ConstructionReferenceDefinition, len(definitions))
	for index, definition := range definitions {
		converted[index] = splice.ConstructionReferenceDefinition{
			Label:       definition.label,
			Destination: definition.destination,
			Title:       definition.title,
			HasTitle:    definition.hasTitle,
		}
	}
	return converted
}

func spliceTypedInlineReferenceForm(form typedInlineReferenceForm) (splice.ConstructionReferenceInlineForm, bool) {
	switch form {
	case typedInlineReferenceFull:
		return splice.ConstructionReferenceInlineFull, true
	case typedInlineReferenceCollapsed:
		return splice.ConstructionReferenceInlineCollapsed, true
	case typedInlineReferenceShortcut:
		return splice.ConstructionReferenceInlineShortcut, true
	default:
		return splice.ConstructionReferenceInlineFull, false
	}
}

func writeTypedInlineReferenceSuffix(output *strings.Builder, form splice.ConstructionReferenceInlineForm, reference string) splice.Range {
	switch form {
	case splice.ConstructionReferenceInlineFull:
		output.WriteString("][")
		start := output.Len()
		output.WriteString(reference)
		end := output.Len()
		output.WriteByte(']')
		return splice.Range{Start: start, End: end}
	case splice.ConstructionReferenceInlineCollapsed:
		output.WriteString("][")
		position := output.Len()
		output.WriteByte(']')
		return splice.Range{Start: position, End: position}
	default:
		output.WriteByte(']')
		position := output.Len()
		return splice.Range{Start: position, End: position}
	}
}

func validateTypedInlineReference(reference string) error {
	if err := validateTypedInlineText(reference); err != nil {
		return fmt.Errorf("reference label %v", err)
	}
	if strings.ContainsAny(reference, "[]\\&") {
		return fmt.Errorf("reference label requires escaping or entity interpretation")
	}
	return nil
}
