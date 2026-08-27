package splice

import (
	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

// newParserBackend returns the Marksplice-native production parser backend.
func newParserBackend() parser.Backend {
	return native.New()
}

// defaultReferenceLabelKey uses the same native normalization as the production backend
// without constructing a complete backend for every label.
func defaultReferenceLabelKey(label string) string {
	return native.ReferenceLabelKey(label)
}
