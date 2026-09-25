package rendermarkdown

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
)

const noInlineASTNode = -1

type inlineASTNode struct {
	eventIndex  int
	exitIndex   int
	parent      int
	firstChild  int
	nextSibling int
}

type inlineAST struct {
	events []parser.SemanticEvent
	nodes  []inlineASTNode
	root   int
}

type inlineASTWalkCursor struct {
	node      int
	nextChild int
}

const noInlineDelimiterNode = -1

type inlineDelimiterNode struct {
	astNode           int
	parent            int
	kind              parser.SemanticKind
	range_            parser.Range
	width             int
	sourceMarker      byte
	sourceOwnedMarker byte
	sourceOwnedPrefix bool
}

type inlineDelimiterAnalysis struct {
	nodes []inlineDelimiterNode
}

func analyzeInlineASTDelimiters(source []byte, ast inlineAST) (inlineDelimiterAnalysis, error) {
	if ast.root < 0 || ast.root >= len(ast.nodes) {
		return inlineDelimiterAnalysis{}, ErrInvalidInput
	}
	nearest := make([]int, len(ast.nodes))
	previousSibling := make([]int, len(ast.nodes))
	for index := range nearest {
		nearest[index] = noInlineDelimiterNode
		previousSibling[index] = noInlineASTNode
	}
	analysis := inlineDelimiterAnalysis{
		nodes: make([]inlineDelimiterNode, 0, 8),
	}
	for astNode := ast.root + 1; astNode < len(ast.nodes); astNode++ {
		node := ast.nodes[astNode]
		if node.parent < 0 || node.parent >= astNode {
			return inlineDelimiterAnalysis{}, fmt.Errorf("%w: inline AST parent index %d", ErrInvalidInput, node.parent)
		}
		parentDelimiter := nearest[node.parent]
		nearest[astNode] = parentDelimiter
		previous := previousSibling[node.parent]
		previousSibling[node.parent] = astNode
		if node.eventIndex < 0 || node.eventIndex >= len(ast.events) {
			return inlineDelimiterAnalysis{}, fmt.Errorf("%w: inline AST event index %d", ErrInvalidInput, node.eventIndex)
		}
		event := ast.events[node.eventIndex]
		width := delimiterWidth(event.Kind)
		if width == 0 {
			continue
		}
		if event.Phase != parser.SemanticEnter {
			return inlineDelimiterAnalysis{}, fmt.Errorf("%w: delimiter node phase %d", ErrInvalidInput, event.Phase)
		}
		marker, _ := sourceEmphasisMarker(source, event.Kind, event.Range)
		ownedMarker, ownedPrefix := inlineSourceOwnedMarker(source, ast, previous, event, marker)
		delimiterIndex := len(analysis.nodes)
		analysis.nodes = append(analysis.nodes, inlineDelimiterNode{
			astNode:           astNode,
			parent:            parentDelimiter,
			kind:              event.Kind,
			range_:            event.Range,
			width:             width,
			sourceMarker:      marker,
			sourceOwnedMarker: ownedMarker,
			sourceOwnedPrefix: ownedPrefix,
		})
		nearest[astNode] = delimiterIndex
	}
	applyInlineSharedRunSourceOwnership(&analysis)
	applyInlineAncestorSourceOwnership(&analysis)
	return analysis, nil
}

func applyInlineSharedRunSourceOwnership(analysis *inlineDelimiterAnalysis) {
	if analysis == nil {
		return
	}
	for currentIndex := range analysis.nodes {
		current := analysis.nodes[currentIndex]
		if current.width <= 0 || current.sourceMarker == 0 {
			continue
		}
		previous := current
		component := make([]int, 0, 2)
		sameMarker := true
		for ancestorIndex := current.parent; ancestorIndex >= 0; ancestorIndex = analysis.nodes[ancestorIndex].parent {
			ancestor := analysis.nodes[ancestorIndex]
			if ancestor.width != current.width ||
				ancestor.sourceMarker == 0 ||
				!emphasisDelimiterRangesShareSourceRun(ancestor.range_, previous.range_, current.width) {
				break
			}
			sameMarker = sameMarker && ancestor.sourceMarker == current.sourceMarker
			component = append(component, ancestorIndex)
			previous = ancestor
		}
		if len(component) < 2 || !sameMarker {
			continue
		}
		for _, ancestorIndex := range component {
			analysis.nodes[ancestorIndex].sourceOwnedMarker = analysis.nodes[ancestorIndex].sourceMarker
		}
	}
}

func applyInlineAncestorSourceOwnership(analysis *inlineDelimiterAnalysis) {
	if analysis == nil {
		return
	}
	for currentIndex := range analysis.nodes {
		current := analysis.nodes[currentIndex]
		if current.width <= 0 || current.sourceMarker == 0 {
			continue
		}
		for ancestorIndex := current.parent; ancestorIndex >= 0; ancestorIndex = analysis.nodes[ancestorIndex].parent {
			ancestor := analysis.nodes[ancestorIndex]
			if ancestor.width != current.width {
				continue
			}
			if ancestor.sourceMarker != 0 && ancestor.sourceMarker != current.sourceMarker {
				analysis.nodes[ancestorIndex].sourceOwnedMarker = ancestor.sourceMarker
			}
			break
		}
	}
}

func inlineSourceOwnedMarker(
	source []byte,
	ast inlineAST,
	previousNode int,
	event parser.SemanticEvent,
	marker byte,
) (byte, bool) {
	previousEvent, ok := inlinePreviousTextSiblingEvent(ast, previousNode, event)
	if marker == 0 || !ok {
		return 0, false
	}
	count, ok := inlineUnescapedTrailingSourceRun(source, previousEvent, marker)
	if !ok || !escapedTextMarkerRunTail([]byte(escapeText(previousEvent.Value)), marker, count) {
		return 0, false
	}
	return marker, true
}

func inlinePreviousTextSiblingEvent(
	ast inlineAST,
	previousNode int,
	event parser.SemanticEvent,
) (parser.SemanticEvent, bool) {
	if previousNode <= ast.root || previousNode >= len(ast.nodes) {
		return parser.SemanticEvent{}, false
	}
	previous := ast.nodes[previousNode]
	if previous.eventIndex < 0 || previous.eventIndex >= len(ast.events) {
		return parser.SemanticEvent{}, false
	}
	previousEvent := ast.events[previous.eventIndex]
	if previousEvent.Phase != parser.SemanticLeaf ||
		previousEvent.Kind != parser.SemanticText ||
		previousEvent.Range.End != event.Range.Start {
		return parser.SemanticEvent{}, false
	}
	return previousEvent, true
}

func inlineUnescapedTrailingSourceRun(
	source []byte,
	event parser.SemanticEvent,
	marker byte,
) (int, bool) {
	previousMarker, ok := trailingEmphasisMarker(event.Value)
	if !ok || previousMarker != marker {
		return 0, false
	}
	markerPosition := event.Range.End - 1
	if markerPosition < event.Range.Start || markerPosition >= len(source) ||
		source[markerPosition] != marker || sourceByteEscapedAt(source, markerPosition) {
		return 0, false
	}
	start := markerPosition
	for start > event.Range.Start &&
		source[start-1] == marker &&
		!sourceByteEscapedAt(source, start-1) {
		start--
	}
	return markerPosition - start + 1, true
}

type inlineASTWorkspace struct {
	events     []parser.SemanticEvent
	nodes      []inlineASTNode
	buildStack []int
	lastChild  []int
	walkStack  []inlineASTWalkCursor
}

func newInlineASTWorkspace() inlineASTWorkspace {
	return inlineASTWorkspace{
		events:     make([]parser.SemanticEvent, 0, 16),
		nodes:      make([]inlineASTNode, 0, 17),
		buildStack: make([]int, 0, 8),
		lastChild:  make([]int, 0, 17),
		walkStack:  make([]inlineASTWalkCursor, 0, 8),
	}
}

func buildInlineAST(events []parser.SemanticEvent) (inlineAST, error) {
	workspace := newInlineASTWorkspace()
	workspace.beginBuild()
	for _, event := range events {
		if err := workspace.appendEvent(event); err != nil {
			return inlineAST{}, err
		}
	}
	return workspace.finishBuild()
}

func (w *inlineASTWorkspace) beginBuild() {
	w.events = w.events[:0]
	w.nodes = w.nodes[:0]
	w.nodes = append(w.nodes, inlineASTNode{
		eventIndex:  noInlineASTNode,
		exitIndex:   noInlineASTNode,
		parent:      noInlineASTNode,
		firstChild:  noInlineASTNode,
		nextSibling: noInlineASTNode,
	})
	w.buildStack = append(w.buildStack[:0], 0)
	w.lastChild = append(w.lastChild[:0], noInlineASTNode)
}

func (w *inlineASTWorkspace) appendEvent(event parser.SemanticEvent) error {
	if w == nil || len(w.buildStack) == 0 {
		return ErrInvalidInput
	}
	eventIndex := len(w.events)
	w.events = append(w.events, event)
	switch event.Phase {
	case parser.SemanticEnter:
		w.appendContainer(eventIndex)
		return nil
	case parser.SemanticLeaf:
		w.appendLeaf(eventIndex)
		return nil
	case parser.SemanticExit:
		return w.closeContainer(eventIndex, event)
	default:
		return fmt.Errorf("%w: unknown semantic phase %d", ErrInvalidInput, event.Phase)
	}
}

func (w *inlineASTWorkspace) appendContainer(eventIndex int) {
	parent := w.buildStack[len(w.buildStack)-1]
	index := w.appendNode(eventIndex, noInlineASTNode, parent)
	w.buildStack = append(w.buildStack, index)
}

func (w *inlineASTWorkspace) appendLeaf(eventIndex int) {
	parent := w.buildStack[len(w.buildStack)-1]
	w.appendNode(eventIndex, eventIndex, parent)
}

func (w *inlineASTWorkspace) appendNode(eventIndex, exitIndex, parent int) int {
	index := len(w.nodes)
	w.nodes = append(w.nodes, inlineASTNode{
		eventIndex:  eventIndex,
		exitIndex:   exitIndex,
		parent:      parent,
		firstChild:  noInlineASTNode,
		nextSibling: noInlineASTNode,
	})
	w.lastChild = append(w.lastChild, noInlineASTNode)
	if w.nodes[parent].firstChild == noInlineASTNode {
		w.nodes[parent].firstChild = index
	} else {
		w.nodes[w.lastChild[parent]].nextSibling = index
	}
	w.lastChild[parent] = index
	return index
}

func (w *inlineASTWorkspace) closeContainer(eventIndex int, event parser.SemanticEvent) error {
	if len(w.buildStack) <= 1 {
		return fmt.Errorf("%w: inline AST close without open", ErrInvalidInput)
	}
	current := w.buildStack[len(w.buildStack)-1]
	open := w.events[w.nodes[current].eventIndex]
	if open.Kind != event.Kind {
		return fmt.Errorf(
			"%w: inline AST close kind %d does not match open kind %d",
			ErrInvalidInput,
			event.Kind,
			open.Kind,
		)
	}
	w.nodes[current].exitIndex = eventIndex
	w.buildStack = w.buildStack[:len(w.buildStack)-1]
	return nil
}

func (w *inlineASTWorkspace) finishBuild() (inlineAST, error) {
	if w == nil || len(w.buildStack) == 0 {
		return inlineAST{}, ErrInvalidInput
	}
	if len(w.buildStack) != 1 {
		return inlineAST{}, fmt.Errorf("%w: inline AST has %d unclosed node(s)", ErrInvalidInput, len(w.buildStack)-1)
	}
	return inlineAST{events: w.events, nodes: w.nodes, root: 0}, nil
}

func walkInlineAST(ast inlineAST, visit parser.SemanticVisitor) error {
	var workspace inlineASTWorkspace
	return workspace.walk(ast, visit)
}

func (w *inlineASTWorkspace) walk(ast inlineAST, visit parser.SemanticVisitor) error {
	if w == nil || visit == nil || ast.root < 0 || ast.root >= len(ast.nodes) {
		return ErrInvalidInput
	}

	w.walkStack = append(w.walkStack[:0], inlineASTWalkCursor{
		node:      ast.root,
		nextChild: ast.nodes[ast.root].firstChild,
	})
	for len(w.walkStack) != 0 {
		current := &w.walkStack[len(w.walkStack)-1]
		if current.nextChild != noInlineASTNode {
			if err := w.walkChild(ast, current, visit); err != nil {
				return err
			}
			continue
		}
		if err := w.closeWalkCursor(ast, visit); err != nil {
			return err
		}
	}
	return nil
}

func (w *inlineASTWorkspace) walkChild(
	ast inlineAST,
	current *inlineASTWalkCursor,
	visit parser.SemanticVisitor,
) error {
	child := current.nextChild
	if child < 0 || child >= len(ast.nodes) {
		return fmt.Errorf("%w: inline AST child index %d", ErrInvalidInput, child)
	}
	current.nextChild = ast.nodes[child].nextSibling
	node := ast.nodes[child]
	if node.eventIndex < 0 || node.eventIndex >= len(ast.events) {
		return fmt.Errorf("%w: inline AST event index %d", ErrInvalidInput, node.eventIndex)
	}
	event := ast.events[node.eventIndex]
	switch event.Phase {
	case parser.SemanticLeaf:
		return visit(event)
	case parser.SemanticEnter:
		if err := visit(event); err != nil {
			return err
		}
		w.walkStack = append(w.walkStack, inlineASTWalkCursor{
			node:      child,
			nextChild: node.firstChild,
		})
		return nil
	default:
		return fmt.Errorf("%w: inline AST node phase %d", ErrInvalidInput, event.Phase)
	}
}

func (w *inlineASTWorkspace) closeWalkCursor(ast inlineAST, visit parser.SemanticVisitor) error {
	current := w.walkStack[len(w.walkStack)-1]
	w.walkStack = w.walkStack[:len(w.walkStack)-1]
	if current.node == ast.root {
		return nil
	}
	node := ast.nodes[current.node]
	if node.exitIndex < 0 || node.exitIndex >= len(ast.events) {
		return fmt.Errorf("%w: inline AST missing close event", ErrInvalidInput)
	}
	exit := ast.events[node.exitIndex]
	if exit.Phase != parser.SemanticExit || exit.Kind != ast.events[node.eventIndex].Kind {
		return fmt.Errorf("%w: inline AST invalid close event", ErrInvalidInput)
	}
	return visit(exit)
}

type inlineASTHostVisitor struct {
	downstream parser.SemanticVisitor
	hostKind   parser.SemanticKind
	workspace  inlineASTWorkspace
	active     bool
}

func newInlineASTHostVisitor(downstream parser.SemanticVisitor) *inlineASTHostVisitor {
	return &inlineASTHostVisitor{
		downstream: downstream,
		workspace:  newInlineASTWorkspace(),
	}
}

func (v *inlineASTHostVisitor) visit(event parser.SemanticEvent) error {
	if v == nil || v.downstream == nil {
		return ErrInvalidInput
	}
	if !v.active {
		if event.Phase == parser.SemanticEnter && inlineASTHostKind(event.Kind) {
			if err := v.downstream(event); err != nil {
				return err
			}
			v.active = true
			v.hostKind = event.Kind
			v.workspace.beginBuild()
			return nil
		}
		return v.downstream(event)
	}

	if event.Phase == parser.SemanticEnter && inlineASTHostKind(event.Kind) {
		return fmt.Errorf("%w: nested inline AST host kind %d", ErrInvalidInput, event.Kind)
	}
	if event.Phase != parser.SemanticExit || event.Kind != v.hostKind {
		return v.workspace.appendEvent(event)
	}

	ast, err := v.workspace.finishBuild()
	if err != nil {
		return err
	}
	if err := v.workspace.walk(ast, v.downstream); err != nil {
		return err
	}
	v.active = false
	v.hostKind = parser.SemanticUnknown
	return v.downstream(event)
}

func (v *inlineASTHostVisitor) finish() error {
	if v == nil || v.downstream == nil {
		return ErrInvalidInput
	}
	if v.active {
		return fmt.Errorf("%w: unterminated inline AST host kind %d", ErrInvalidInput, v.hostKind)
	}
	return nil
}

func inlineASTHostKind(kind parser.SemanticKind) bool {
	switch kind {
	case parser.SemanticParagraph, parser.SemanticHeading, parser.SemanticTableCell:
		return true
	default:
		return false
	}
}
