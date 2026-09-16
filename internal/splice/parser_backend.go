package splice

import (
	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

// semanticBackend is the Native rendering contract, including parser-owned
// reference-label normalization required by canonical Markdown rendering.
type semanticBackend interface {
	parser.SemanticBackend
	ReferenceLabelKey(label string) string
}

// newSemanticBackend returns the Marksplice-native semantic renderer backend.
func newSemanticBackend() semanticBackend {
	return native.New()
}

// newParserBackend returns the Marksplice-native production parser backend.
func newParserBackend() parser.Backend {
	return native.New()
}

// defaultReferenceLabelKey uses the same native normalization as the production backend
// without constructing a complete backend for every label.
func defaultReferenceLabelKey(label string) string {
	return native.ReferenceLabelKey(label)
}
