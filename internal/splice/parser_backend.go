package splice

import (
	"github.com/zoster81/marksplice/internal/parser"
	goldmarkparser "github.com/zoster81/marksplice/internal/parser/goldmark"
)

// newParserBackend is the temporary production parser-backend bridge. M111 keeps
// Goldmark ownership centralized here so later native-parser cutover cannot leave
// hidden backend imports throughout splice/model/construction code.
func newParserBackend() parser.Backend {
	return goldmarkparser.New()
}

// defaultReferenceLabelKey keeps the hot pure reference-normalization path from
// constructing a complete parser backend for every label while preserving one
// centralized temporary Goldmark dependency.
func defaultReferenceLabelKey(label string) string {
	return goldmarkparser.ReferenceLabelKey(label)
}
