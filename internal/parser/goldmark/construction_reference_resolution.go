package goldmark

import (
	"fmt"

	"github.com/yuin/goldmark/util"

	"github.com/zoster81/marksplice/internal/parser"
)

// ResolveConstructionReference resolves one GFM reference label against a
// construction-only definition set. Ambiguous normalized labels fail closed.
func ResolveConstructionReference(label string, definitions []parser.ConstructionReferenceDefinition) (parser.ConstructionReferenceDefinition, error) {
	key := ReferenceLabelKey(label)
	var match parser.ConstructionReferenceDefinition
	count := 0
	for _, definition := range definitions {
		if ReferenceLabelKey(definition.Label) != key {
			continue
		}
		match = definition
		count++
	}
	if count != 1 {
		return parser.ConstructionReferenceDefinition{}, fmt.Errorf("reference label %q must match exactly one normalized definition", label)
	}
	return match, nil
}

// ReferenceLabelKey returns the parser-defined GFM normalization key for one
// reference label. The representation is internal and must not be persisted.
func ReferenceLabelKey(label string) string {
	return util.ToLinkReference([]byte(label))
}

// ConstructionReferenceLabelsEquivalent reports whether two source labels use
// the same GFM reference-label normalization key.
func ConstructionReferenceLabelsEquivalent(left, right string) bool {
	return ReferenceLabelKey(left) == ReferenceLabelKey(right)
}
