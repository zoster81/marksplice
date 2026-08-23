package splice

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
	goldmarkparser "github.com/zoster81/marksplice/internal/parser/goldmark"
)

// ConstructionReferenceDefinition is one construction-only reference target.
type ConstructionReferenceDefinition struct {
	Label       string
	Destination string
	Title       string
	HasTitle    bool
}

// ResolveConstructionReference resolves one source label through the isolated
// parser implementation and fails closed on normalized ambiguity.
func ResolveConstructionReference(label string, definitions []ConstructionReferenceDefinition) (ConstructionReferenceDefinition, error) {
	converted := make([]parser.ConstructionReferenceDefinition, len(definitions))
	for index, definition := range definitions {
		converted[index] = parser.ConstructionReferenceDefinition{
			Label:       definition.Label,
			Destination: definition.Destination,
			Title:       definition.Title,
			HasTitle:    definition.HasTitle,
		}
	}
	resolved, err := goldmarkparser.ResolveConstructionReference(label, converted)
	if err != nil {
		return ConstructionReferenceDefinition{}, fmt.Errorf("resolve construction reference: %w", err)
	}
	return ConstructionReferenceDefinition{
		Label:       resolved.Label,
		Destination: resolved.Destination,
		Title:       resolved.Title,
		HasTitle:    resolved.HasTitle,
	}, nil
}

// ConstructionReferenceLabelsEquivalent reports whether two labels share one
// parser-defined GFM reference normalization key.
func ConstructionReferenceLabelsEquivalent(left, right string) bool {
	return goldmarkparser.ConstructionReferenceLabelsEquivalent(left, right)
}
