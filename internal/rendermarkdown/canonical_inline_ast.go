package rendermarkdown

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
)

const noCanonicalInlineASTNode = -1

// canonicalInlineAST is the operation-local structural representation used by
// canonical inline rendering. It is built only from Native semantic events.
type canonicalInlineAST struct {
	events []parser.SemanticEvent
	nodes  []canonicalInlineASTNode
	root   int
}

type canonicalInlineASTNode struct {
	eventIndex  int
	exitIndex   int
	parent      int
	firstChild  int
	nextSibling int
}

type canonicalInlineASTBuildFrame struct {
	node      int
	lastChild int
}

func buildCanonicalInlineAST(events []parser.SemanticEvent) (canonicalInlineAST, error) {
	ast := canonicalInlineAST{
		events: append([]parser.SemanticEvent(nil), events...),
		nodes: []canonicalInlineASTNode{{
			eventIndex:  noCanonicalInlineASTNode,
			exitIndex:   noCanonicalInlineASTNode,
			parent:      noCanonicalInlineASTNode,
			firstChild:  noCanonicalInlineASTNode,
			nextSibling: noCanonicalInlineASTNode,
		}},
		root: 0,
	}
	stack := []canonicalInlineASTBuildFrame{{node: ast.root, lastChild: noCanonicalInlineASTNode}}

	appendNode := func(eventIndex int) int {
		parentFrame := &stack[len(stack)-1]
		nodeIndex := len(ast.nodes)
		ast.nodes = append(ast.nodes, canonicalInlineASTNode{
			eventIndex:  eventIndex,
			exitIndex:   noCanonicalInlineASTNode,
			parent:      parentFrame.node,
			firstChild:  noCanonicalInlineASTNode,
			nextSibling: noCanonicalInlineASTNode,
		})
		if parentFrame.lastChild == noCanonicalInlineASTNode {
			ast.nodes[parentFrame.node].firstChild = nodeIndex
		} else {
			ast.nodes[parentFrame.lastChild].nextSibling = nodeIndex
		}
		parentFrame.lastChild = nodeIndex
		return nodeIndex
	}

	for eventIndex, event := range ast.events {
		switch event.Phase {
		case parser.SemanticLeaf:
			appendNode(eventIndex)
		case parser.SemanticEnter:
			nodeIndex := appendNode(eventIndex)
			stack = append(stack, canonicalInlineASTBuildFrame{
				node:      nodeIndex,
				lastChild: noCanonicalInlineASTNode,
			})
		case parser.SemanticExit:
			if len(stack) == 1 {
				return canonicalInlineAST{}, fmt.Errorf("%w: inline AST close without open kind %d", ErrInvalidInput, event.Kind)
			}
			frame := stack[len(stack)-1]
			open := ast.events[ast.nodes[frame.node].eventIndex]
			if open.Kind != event.Kind {
				return canonicalInlineAST{}, fmt.Errorf("%w: inline AST close kind %d for open kind %d", ErrInvalidInput, event.Kind, open.Kind)
			}
			ast.nodes[frame.node].exitIndex = eventIndex
			stack = stack[:len(stack)-1]
		default:
			return canonicalInlineAST{}, fmt.Errorf("%w: inline AST phase %d", ErrInvalidInput, event.Phase)
		}
	}
	if len(stack) != 1 {
		frame := stack[len(stack)-1]
		open := ast.events[ast.nodes[frame.node].eventIndex]
		return canonicalInlineAST{}, fmt.Errorf("%w: unterminated inline AST kind %d", ErrInvalidInput, open.Kind)
	}
	return ast, nil
}
