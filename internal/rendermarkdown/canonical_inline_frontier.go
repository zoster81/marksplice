package rendermarkdown

import "github.com/zoster81/marksplice/internal/parser"

type canonicalInlineFrontierCandidate struct {
	canonicalInlinePlanCandidate
	state canonicalInlineSelectorState
}

type canonicalInlineFrontierPruneFunc func(
	ast canonicalInlineAST,
	candidates []canonicalInlineFrontierCandidate,
	needsTilde bool,
) ([]canonicalInlineFrontierCandidate, error)

type canonicalInlineASTViewFrame struct {
	source    int
	target    int
	nextChild int
	lastChild int
}

func canonicalInlineASTView(
	ast canonicalInlineAST,
	roots []int,
) (canonicalInlineAST, error) {
	if !canonicalInlineASTViewRootsValid(ast, roots) {
		return canonicalInlineAST{}, ErrInvalidInput
	}
	view, err := newCanonicalInlineASTView(ast)
	if err != nil {
		return canonicalInlineAST{}, err
	}
	previousSourceRoot := noCanonicalInlineASTNode
	previousTargetRoot := noCanonicalInlineASTNode
	for _, sourceRoot := range roots {
		previousTargetRoot, err = canonicalInlineASTViewAppendRoot(
			&view, ast, sourceRoot, previousSourceRoot, previousTargetRoot,
		)
		if err != nil {
			return canonicalInlineAST{}, err
		}
		previousSourceRoot = sourceRoot
	}
	return view, nil
}

func newCanonicalInlineASTView(ast canonicalInlineAST) (canonicalInlineAST, error) {
	if ast.root < 0 || ast.root >= len(ast.nodes) {
		return canonicalInlineAST{}, ErrInvalidInput
	}
	return canonicalInlineAST{
		events: ast.events,
		nodes: []canonicalInlineASTNode{{
			eventIndex:  noCanonicalInlineASTNode,
			exitIndex:   noCanonicalInlineASTNode,
			parent:      noCanonicalInlineASTNode,
			firstChild:  noCanonicalInlineASTNode,
			nextSibling: noCanonicalInlineASTNode,
		}},
		root: 0,
	}, nil
}

func canonicalInlineASTViewAppendRoot(
	view *canonicalInlineAST,
	source canonicalInlineAST,
	sourceRoot int,
	previousSourceRoot int,
	previousTargetRoot int,
) (int, error) {
	if !canonicalInlineASTViewAppendRootValid(
		view, source, sourceRoot, previousSourceRoot, previousTargetRoot,
	) {
		return noCanonicalInlineASTNode, ErrInvalidInput
	}
	targetRoot := canonicalInlineASTViewAppendNode(view, source, sourceRoot, view.root)
	if previousTargetRoot == noCanonicalInlineASTNode {
		view.nodes[view.root].firstChild = targetRoot
	} else {
		view.nodes[previousTargetRoot].nextSibling = targetRoot
	}
	canonicalInlineASTViewAppendDescendants(view, source, sourceRoot, targetRoot)
	return targetRoot, nil
}

func canonicalInlineASTViewAppendRootValid(
	view *canonicalInlineAST,
	source canonicalInlineAST,
	sourceRoot int,
	previousSourceRoot int,
	previousTargetRoot int,
) bool {
	if view == nil || view.root < 0 || view.root >= len(view.nodes) ||
		sourceRoot <= source.root || sourceRoot >= len(source.nodes) {
		return false
	}
	if previousSourceRoot == noCanonicalInlineASTNode {
		return previousTargetRoot == noCanonicalInlineASTNode &&
			view.nodes[view.root].firstChild == noCanonicalInlineASTNode
	}
	return previousSourceRoot > source.root && previousSourceRoot < len(source.nodes) &&
		previousTargetRoot > view.root && previousTargetRoot < len(view.nodes) &&
		source.nodes[previousSourceRoot].parent == source.nodes[sourceRoot].parent &&
		source.nodes[previousSourceRoot].nextSibling == sourceRoot
}

func canonicalInlineASTViewRootsValid(ast canonicalInlineAST, roots []int) bool {
	if ast.root < 0 || ast.root >= len(ast.nodes) {
		return false
	}
	if len(roots) == 0 {
		return true
	}
	parent := noCanonicalInlineASTNode
	for index, root := range roots {
		if root <= ast.root || root >= len(ast.nodes) {
			return false
		}
		if index == 0 {
			parent = ast.nodes[root].parent
			continue
		}
		if ast.nodes[root].parent != parent ||
			ast.nodes[roots[index-1]].nextSibling != root {
			return false
		}
	}
	return true
}

func canonicalInlineASTViewAppendNode(
	view *canonicalInlineAST,
	source canonicalInlineAST,
	sourceNode int,
	parent int,
) int {
	node := source.nodes[sourceNode]
	target := len(view.nodes)
	view.nodes = append(view.nodes, canonicalInlineASTNode{
		eventIndex:  node.eventIndex,
		exitIndex:   node.exitIndex,
		parent:      parent,
		firstChild:  noCanonicalInlineASTNode,
		nextSibling: noCanonicalInlineASTNode,
	})
	return target
}

func canonicalInlineASTViewAppendDescendants(
	view *canonicalInlineAST,
	source canonicalInlineAST,
	sourceRoot int,
	targetRoot int,
) {
	stack := []canonicalInlineASTViewFrame{{
		source: sourceRoot, target: targetRoot,
		nextChild: source.nodes[sourceRoot].firstChild,
		lastChild: noCanonicalInlineASTNode,
	}}
	for len(stack) != 0 {
		frame := &stack[len(stack)-1]
		if frame.nextChild == noCanonicalInlineASTNode {
			stack = stack[:len(stack)-1]
			continue
		}
		sourceChild := frame.nextChild
		frame.nextChild = source.nodes[sourceChild].nextSibling
		targetChild := canonicalInlineASTViewAppendNode(view, source, sourceChild, frame.target)
		if frame.lastChild == noCanonicalInlineASTNode {
			view.nodes[frame.target].firstChild = targetChild
		} else {
			view.nodes[frame.lastChild].nextSibling = targetChild
		}
		frame.lastChild = targetChild
		stack = append(stack, canonicalInlineASTViewFrame{
			source: sourceChild, target: targetChild,
			nextChild: source.nodes[sourceChild].firstChild,
			lastChild: noCanonicalInlineASTNode,
		})
	}
}

func (candidate canonicalInlineFrontierCandidate) clone() canonicalInlineFrontierCandidate {
	return canonicalInlineFrontierCandidate{
		canonicalInlinePlanCandidate: candidate.canonicalInlinePlanCandidate.clone(),
		state:                        candidate.state,
	}
}

func appendCanonicalInlineFrontierCandidates(
	left, right canonicalInlineFrontierCandidate,
) canonicalInlineFrontierCandidate {
	return canonicalInlineFrontierCandidate{
		canonicalInlinePlanCandidate: appendCanonicalInlinePlanCandidates(
			left.canonicalInlinePlanCandidate,
			right.canonicalInlinePlanCandidate,
		),
	}
}

func canonicalInlineFrontierLeaf(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	nodeIndex int,
) (canonicalInlineFrontierCandidate, bool) {
	if nodeIndex <= ast.root || nodeIndex >= len(ast.nodes) ||
		len(normalization.nodes) != len(ast.nodes) {
		return canonicalInlineFrontierCandidate{}, false
	}
	node := ast.nodes[nodeIndex]
	if node.eventIndex < 0 || node.eventIndex >= len(ast.events) {
		return canonicalInlineFrontierCandidate{}, false
	}
	event := ast.events[node.eventIndex]
	if event.Phase != parser.SemanticLeaf {
		return canonicalInlineFrontierCandidate{}, false
	}
	candidate := canonicalInlineFrontierCandidate{}
	if event.Kind == parser.SemanticText {
		candidate.payloads = []canonicalInlinePayloadChoice{canonicalInlinePreferredPayload}
		candidate.payloadKnown = []bool{false}
		candidate.payloadRawTab = []bool{
			normalization.nodes[nodeIndex].trailingAlternative == canonicalInlineRawTabBoundary,
		}
	}
	return candidate, true
}

func canonicalInlineFrontierWrap(
	ast canonicalInlineAST,
	nodeIndex int,
	child canonicalInlineFrontierCandidate,
) ([]canonicalInlineFrontierCandidate, bool) {
	if nodeIndex <= ast.root || nodeIndex >= len(ast.nodes) {
		return nil, false
	}
	eventIndex := ast.nodes[nodeIndex].eventIndex
	if eventIndex < 0 || eventIndex >= len(ast.events) {
		return nil, false
	}
	event := ast.events[eventIndex]
	if event.Phase != parser.SemanticEnter {
		return nil, false
	}
	base := child.clone()
	base.state = canonicalInlineSelectorState{}
	switch event.Kind {
	case parser.SemanticEmphasis, parser.SemanticStrong:
		result := make([]canonicalInlineFrontierCandidate, 0, 2)
		for _, marker := range []byte{'*', '_'} {
			variant := base.clone()
			variant.markers = append([]byte{marker}, variant.markers...)
			result = append(result, variant)
		}
		return result, true
	case parser.SemanticStrikethrough:
		result := make([]canonicalInlineFrontierCandidate, 0, 2)
		for _, width := range []uint8{2, 1} {
			variant := base.clone()
			variant.strikeWidths = append([]uint8{width}, variant.strikeWidths...)
			if width == 1 {
				variant.strikeSingleCost++
			}
			result = append(result, variant)
		}
		return result, true
	case parser.SemanticLink, parser.SemanticImage:
		return []canonicalInlineFrontierCandidate{base}, true
	default:
		return nil, false
	}
}

func canonicalInlineBottomUpHostWithNormalization(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	needsTilde bool,
	prune canonicalInlineFrontierPruneFunc,
) ([]canonicalInlineFrontierCandidate, bool, error) {
	if prune == nil || ast.root < 0 || ast.root >= len(ast.nodes) ||
		len(normalization.nodes) != len(ast.nodes) {
		return nil, false, ErrInvalidInput
	}
	frontier := []canonicalInlineFrontierCandidate{{}}
	view, err := newCanonicalInlineASTView(ast)
	if err != nil {
		return nil, false, err
	}
	previousSourceRoot := noCanonicalInlineASTNode
	previousTargetRoot := noCanonicalInlineASTNode
	for child := ast.nodes[ast.root].firstChild; child != noCanonicalInlineASTNode; child = ast.nodes[child].nextSibling {
		childFrontier, supported, err := canonicalInlineBottomUpNodeWithNormalization(
			ast, normalization, child, needsTilde, prune,
		)
		if err != nil || !supported {
			return nil, supported, err
		}
		frontier = canonicalInlineCombineFrontiers(frontier, childFrontier)
		previousTargetRoot, err = canonicalInlineASTViewAppendRoot(
			&view, ast, child, previousSourceRoot, previousTargetRoot,
		)
		if err != nil {
			return nil, false, err
		}
		previousSourceRoot = child
		frontier, err = prune(view, frontier, needsTilde)
		if err != nil || len(frontier) == 0 {
			return frontier, true, err
		}
	}
	return frontier, true, nil
}

func canonicalInlineBottomUpNodeWithNormalization(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	nodeIndex int,
	needsTilde bool,
	prune canonicalInlineFrontierPruneFunc,
) ([]canonicalInlineFrontierCandidate, bool, error) {
	if candidate, leaf := canonicalInlineFrontierLeaf(ast, normalization, nodeIndex); leaf {
		return []canonicalInlineFrontierCandidate{candidate}, true, nil
	}
	if nodeIndex <= ast.root || nodeIndex >= len(ast.nodes) || prune == nil {
		return nil, false, nil
	}
	children, supported, err := canonicalInlineBottomUpChildrenWithNormalization(
		ast, normalization, nodeIndex, needsTilde, prune,
	)
	if err != nil || !supported {
		return nil, supported, err
	}
	wrapped := make([]canonicalInlineFrontierCandidate, 0, len(children)*2)
	for _, child := range children {
		variants, supported := canonicalInlineFrontierWrap(ast, nodeIndex, child)
		if !supported {
			return nil, false, nil
		}
		wrapped = append(wrapped, variants...)
	}
	view, err := canonicalInlineASTView(ast, []int{nodeIndex})
	if err != nil {
		return nil, false, err
	}
	wrapped, err = prune(view, wrapped, needsTilde)
	return wrapped, true, err
}

func canonicalInlineBottomUpChildrenWithNormalization(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	nodeIndex int,
	needsTilde bool,
	prune canonicalInlineFrontierPruneFunc,
) ([]canonicalInlineFrontierCandidate, bool, error) {
	node := ast.nodes[nodeIndex]
	children := []canonicalInlineFrontierCandidate{{}}
	view, err := newCanonicalInlineASTView(ast)
	if err != nil {
		return nil, false, err
	}
	previousSourceRoot := noCanonicalInlineASTNode
	previousTargetRoot := noCanonicalInlineASTNode
	for child := node.firstChild; child != noCanonicalInlineASTNode; child = ast.nodes[child].nextSibling {
		childFrontier, supported, err := canonicalInlineBottomUpNodeWithNormalization(
			ast, normalization, child, needsTilde, prune,
		)
		if err != nil || !supported {
			return nil, supported, err
		}
		children = canonicalInlineCombineFrontiers(children, childFrontier)
		previousTargetRoot, err = canonicalInlineASTViewAppendRoot(
			&view, ast, child, previousSourceRoot, previousTargetRoot,
		)
		if err != nil {
			return nil, false, err
		}
		previousSourceRoot = child
		children, err = prune(view, children, needsTilde)
		if err != nil || len(children) == 0 {
			return children, true, err
		}
	}
	return children, true, nil
}

func canonicalInlineCombineFrontiers(
	left, right []canonicalInlineFrontierCandidate,
) []canonicalInlineFrontierCandidate {
	result := make([]canonicalInlineFrontierCandidate, 0, len(left)*len(right))
	for _, prefix := range left {
		for _, suffix := range right {
			result = append(result, appendCanonicalInlineFrontierCandidates(prefix, suffix))
		}
	}
	return result
}
