package rendermarkdown

import (
	"fmt"
	"strings"

	"github.com/zoster81/marksplice/internal/parser"
)

type canonicalInlineBoundaryAlternative uint8

const (
	canonicalInlineNoBoundaryAlternative canonicalInlineBoundaryAlternative = iota
	canonicalInlineRawTabBoundary
)

type canonicalInlineNodeNormalization struct {
	trailingAlternative canonicalInlineBoundaryAlternative
}

type canonicalInlineNormalization struct {
	nodes []canonicalInlineNodeNormalization
}

func normalizeCanonicalInlineAST(ast canonicalInlineAST) (canonicalInlineNormalization, error) {
	if ast.root < 0 || ast.root >= len(ast.nodes) {
		return canonicalInlineNormalization{}, ErrInvalidInput
	}
	normalization := canonicalInlineNormalization{
		nodes: make([]canonicalInlineNodeNormalization, len(ast.nodes)),
	}
	for nodeIndex := 1; nodeIndex < len(ast.nodes); nodeIndex++ {
		node := ast.nodes[nodeIndex]
		if node.eventIndex < 0 || node.eventIndex >= len(ast.events) {
			return canonicalInlineNormalization{}, fmt.Errorf("%w: inline AST event index %d", ErrInvalidInput, node.eventIndex)
		}
		event := ast.events[node.eventIndex]
		if event.Phase != parser.SemanticLeaf || event.Kind != parser.SemanticText ||
			!strings.HasSuffix(event.Value, "\t") || !canonicalInlineNextSiblingUsesDelimiter(ast, node.nextSibling) {
			continue
		}
		normalization.nodes[nodeIndex].trailingAlternative = canonicalInlineRawTabBoundary
	}
	return normalization, nil
}

func canonicalInlineNextSiblingUsesDelimiter(ast canonicalInlineAST, nodeIndex int) bool {
	return canonicalInlineNodeUsesDelimiter(ast, nodeIndex)
}

func canonicalInlineNodeUsesDelimiter(ast canonicalInlineAST, nodeIndex int) bool {
	if nodeIndex < 0 || nodeIndex >= len(ast.nodes) {
		return false
	}
	eventIndex := ast.nodes[nodeIndex].eventIndex
	if eventIndex < 0 || eventIndex >= len(ast.events) {
		return false
	}
	event := ast.events[eventIndex]
	if event.Phase != parser.SemanticEnter {
		return false
	}
	switch event.Kind {
	case parser.SemanticEmphasis, parser.SemanticStrong, parser.SemanticStrikethrough:
		return true
	default:
		return false
	}
}
