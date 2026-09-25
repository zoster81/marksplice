package rendermarkdown

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
)

type inlineDelimiterSide uint8

const (
	inlineDelimiterOpen inlineDelimiterSide = iota
	inlineDelimiterClose
)

type inlineDelimiterNeighborKind uint8

const (
	inlineDelimiterNeighborUnresolved inlineDelimiterNeighborKind = iota
	inlineDelimiterNeighborFixed
	inlineDelimiterNeighborEndpoint
)

type inlineDelimiterNeighbor struct {
	kind      inlineDelimiterNeighborKind
	edge      inlineBoundaryEdge
	delimiter int
	side      inlineDelimiterSide
}

type inlineDelimiterSite struct {
	delimiter int
	side      inlineDelimiterSide
	inner     inlineDelimiterNeighbor
	outer     inlineDelimiterNeighbor
}

type inlineDelimiterNeighborhood struct {
	sites []inlineDelimiterSite
}

func (n inlineDelimiterNeighborhood) site(delimiter int, side inlineDelimiterSide) inlineDelimiterSite {
	index := delimiter*2 + int(side)
	if delimiter < 0 || side > inlineDelimiterClose || index < 0 || index >= len(n.sites) {
		return inlineDelimiterSite{}
	}
	return n.sites[index]
}

func buildInlineDelimiterNeighborhood(
	ast inlineAST,
	normalization inlineASTNormalization,
	delimiters inlineDelimiterAnalysis,
) (inlineDelimiterNeighborhood, error) {
	if ast.root < 0 || ast.root >= len(ast.nodes) ||
		len(normalization.nodes) != len(ast.nodes) {
		return inlineDelimiterNeighborhood{}, ErrInvalidInput
	}

	delimiterByAST, err := indexInlineDelimiterNodes(ast, delimiters)
	if err != nil {
		return inlineDelimiterNeighborhood{}, err
	}
	previousSibling, lastChild, err := indexInlineSiblingEdges(ast)
	if err != nil {
		return inlineDelimiterNeighborhood{}, err
	}

	graph := inlineDelimiterNeighborhood{
		sites: make([]inlineDelimiterSite, len(delimiters.nodes)*2),
	}
	for delimiter := range delimiters.nodes {
		astNode := delimiters.nodes[delimiter].astNode
		open := inlineDelimiterSite{
			delimiter: delimiter,
			side:      inlineDelimiterOpen,
		}
		close := inlineDelimiterSite{
			delimiter: delimiter,
			side:      inlineDelimiterClose,
		}
		open.inner = inlineInnerNeighbor(
			ast,
			normalization,
			delimiterByAST,
			ast.nodes[astNode].firstChild,
			delimiter,
			inlineDelimiterOpen,
		)
		close.inner = inlineInnerNeighbor(
			ast,
			normalization,
			delimiterByAST,
			lastChild[astNode],
			delimiter,
			inlineDelimiterClose,
		)
		open.outer = inlineOuterNeighbor(
			ast,
			normalization,
			delimiterByAST,
			previousSibling,
			astNode,
			inlineDelimiterOpen,
		)
		close.outer = inlineOuterNeighbor(
			ast,
			normalization,
			delimiterByAST,
			previousSibling,
			astNode,
			inlineDelimiterClose,
		)
		graph.sites[delimiter*2] = open
		graph.sites[delimiter*2+1] = close
	}
	return graph, nil
}

func indexInlineDelimiterNodes(ast inlineAST, delimiters inlineDelimiterAnalysis) ([]int, error) {
	byAST := make([]int, len(ast.nodes))
	for index := range byAST {
		byAST[index] = noInlineDelimiterNode
	}
	for delimiter, node := range delimiters.nodes {
		if node.astNode <= ast.root || node.astNode >= len(ast.nodes) {
			return nil, fmt.Errorf("%w: delimiter AST node index %d", ErrInvalidInput, node.astNode)
		}
		if byAST[node.astNode] != noInlineDelimiterNode {
			return nil, fmt.Errorf("%w: duplicate delimiter AST node index %d", ErrInvalidInput, node.astNode)
		}
		byAST[node.astNode] = delimiter
	}
	return byAST, nil
}

func indexInlineSiblingEdges(ast inlineAST) ([]int, []int, error) {
	previous := make([]int, len(ast.nodes))
	lastChild := make([]int, len(ast.nodes))
	for index := range ast.nodes {
		previous[index] = noInlineASTNode
		lastChild[index] = noInlineASTNode
	}
	for parent := ast.root; parent < len(ast.nodes); parent++ {
		prior := noInlineASTNode
		for child := ast.nodes[parent].firstChild; child != noInlineASTNode; child = ast.nodes[child].nextSibling {
			if child <= ast.root || child >= len(ast.nodes) || ast.nodes[child].parent != parent {
				return nil, nil, fmt.Errorf("%w: invalid inline AST sibling index %d", ErrInvalidInput, child)
			}
			previous[child] = prior
			prior = child
			lastChild[parent] = child
		}
	}
	return previous, lastChild, nil
}

func inlineInnerNeighbor(
	ast inlineAST,
	normalization inlineASTNormalization,
	delimiterByAST []int,
	child int,
	owner int,
	side inlineDelimiterSide,
) inlineDelimiterNeighbor {
	if child == noInlineASTNode {
		other := inlineDelimiterClose
		if side == inlineDelimiterClose {
			other = inlineDelimiterOpen
		}
		return inlineEndpointNeighbor(owner, other)
	}
	return inlineNodeBoundaryNeighbor(ast, normalization, delimiterByAST, child, side)
}

func inlineOuterNeighbor(
	ast inlineAST,
	normalization inlineASTNormalization,
	delimiterByAST []int,
	previousSibling []int,
	astNode int,
	side inlineDelimiterSide,
) inlineDelimiterNeighbor {
	if side == inlineDelimiterOpen {
		if previous := previousSibling[astNode]; previous != noInlineASTNode {
			return inlineNodeBoundaryNeighbor(ast, normalization, delimiterByAST, previous, inlineDelimiterClose)
		}
	} else if next := ast.nodes[astNode].nextSibling; next != noInlineASTNode {
		return inlineNodeBoundaryNeighbor(ast, normalization, delimiterByAST, next, inlineDelimiterOpen)
	}
	return inlineParentBoundaryNeighbor(ast, delimiterByAST, ast.nodes[astNode].parent, side)
}

func inlineNodeBoundaryNeighbor(
	ast inlineAST,
	normalization inlineASTNormalization,
	delimiterByAST []int,
	astNode int,
	side inlineDelimiterSide,
) inlineDelimiterNeighbor {
	if astNode < 0 || astNode >= len(normalization.nodes) || astNode >= len(delimiterByAST) {
		return inlineDelimiterNeighbor{}
	}
	if delimiter := delimiterByAST[astNode]; delimiter != noInlineDelimiterNode {
		return inlineEndpointNeighbor(delimiter, side)
	}
	payload := normalization.nodes[astNode]
	if payload.boundaryReady && !payload.boundary.empty {
		edge := payload.boundary.left
		if side == inlineDelimiterClose {
			edge = payload.boundary.right
		}
		return inlineFixedNeighbor(edge)
	}
	if side == inlineDelimiterOpen && inlineTextStartsWithSpace(ast, astNode) {
		return inlineFixedNeighbor(inlineBoundaryEdge{whitespace: true})
	}
	return inlineDelimiterNeighbor{}
}

func inlineTextStartsWithSpace(ast inlineAST, astNode int) bool {
	if astNode < 0 || astNode >= len(ast.nodes) {
		return false
	}
	eventIndex := ast.nodes[astNode].eventIndex
	if eventIndex < 0 || eventIndex >= len(ast.events) {
		return false
	}
	event := ast.events[eventIndex]
	return event.Kind == parser.SemanticText && event.Value != "" && event.Value[0] == ' '
}

func inlineParentBoundaryNeighbor(
	ast inlineAST,
	delimiterByAST []int,
	parent int,
	side inlineDelimiterSide,
) inlineDelimiterNeighbor {
	if parent == ast.root {
		return inlineFixedNeighbor(inlineBoundaryEdge{whitespace: true})
	}
	if parent < 0 || parent >= len(ast.nodes) {
		return inlineDelimiterNeighbor{}
	}
	if delimiter := delimiterByAST[parent]; delimiter != noInlineDelimiterNode {
		return inlineEndpointNeighbor(delimiter, side)
	}
	eventIndex := ast.nodes[parent].eventIndex
	if eventIndex < 0 || eventIndex >= len(ast.events) {
		return inlineDelimiterNeighbor{}
	}
	switch ast.events[eventIndex].Kind {
	case parser.SemanticStrikethrough, parser.SemanticLink, parser.SemanticImage:
		return inlineFixedNeighbor(inlineBoundaryEdge{punctuation: true})
	default:
		return inlineDelimiterNeighbor{}
	}
}

func inlineEndpointNeighbor(delimiter int, side inlineDelimiterSide) inlineDelimiterNeighbor {
	return inlineDelimiterNeighbor{
		kind:      inlineDelimiterNeighborEndpoint,
		delimiter: delimiter,
		side:      side,
	}
}

func inlineFixedNeighbor(edge inlineBoundaryEdge) inlineDelimiterNeighbor {
	return inlineDelimiterNeighbor{
		kind: inlineDelimiterNeighborFixed,
		edge: edge,
	}
}

type inlineMarkerMask uint8

const (
	inlineMarkerStar inlineMarkerMask = 1 << iota
	inlineMarkerUnderscore
)

type inlineDelimiterCandidateGeometry struct {
	runLength int
	resolved  bool
	canOpen   bool
	canClose  bool
}

type inlineDelimiterFixedGeometry struct {
	star         inlineDelimiterCandidateGeometry
	underscore   inlineDelimiterCandidateGeometry
	allowedOpen  inlineMarkerMask
	allowedClose inlineMarkerMask
}

func analyzeInlineDelimiterFixedGeometry(site inlineDelimiterSite, width int) inlineDelimiterFixedGeometry {
	if width <= 0 {
		return inlineDelimiterFixedGeometry{}
	}
	before, after := site.outer, site.inner
	if site.side == inlineDelimiterClose {
		before, after = site.inner, site.outer
	}
	star := inlineFixedDelimiterCandidate(before, after, width, '*')
	underscore := inlineFixedDelimiterCandidate(before, after, width, '_')
	geometry := inlineDelimiterFixedGeometry{
		star:       star,
		underscore: underscore,
	}
	if star.resolved && star.canOpen {
		geometry.allowedOpen |= inlineMarkerStar
	}
	if underscore.resolved && underscore.canOpen {
		geometry.allowedOpen |= inlineMarkerUnderscore
	}
	if star.resolved && star.canClose {
		geometry.allowedClose |= inlineMarkerStar
	}
	if underscore.resolved && underscore.canClose {
		geometry.allowedClose |= inlineMarkerUnderscore
	}
	return geometry
}

func inlineFixedDelimiterCandidate(
	before, after inlineDelimiterNeighbor,
	width int,
	marker byte,
) inlineDelimiterCandidateGeometry {
	beforeWhitespace, beforePunctuation, beforeExtension, beforeOK :=
		inlineFixedBoundaryClass(before, marker)
	afterWhitespace, afterPunctuation, afterExtension, afterOK :=
		inlineFixedBoundaryClass(after, marker)
	if !beforeOK || !afterOK {
		return inlineDelimiterCandidateGeometry{}
	}
	canOpen, canClose := parser.DelimiterFlankingFromClasses(
		beforeWhitespace,
		beforePunctuation,
		afterWhitespace,
		afterPunctuation,
		marker,
	)
	if before.kind == inlineDelimiterNeighborFixed && before.edge.preservableTab {
		rawOpen, rawClose := parser.DelimiterFlankingFromClasses(
			true,
			false,
			afterWhitespace,
			afterPunctuation,
			marker,
		)
		if rawOpen != canOpen || rawClose != canClose {
			canOpen, canClose = rawOpen, rawClose
		}
	}
	return inlineDelimiterCandidateGeometry{
		runLength: width + beforeExtension + afterExtension,
		resolved:  true,
		canOpen:   canOpen,
		canClose:  canClose,
	}
}

func inlineFixedBoundaryClass(
	neighbor inlineDelimiterNeighbor,
	marker byte,
) (whitespace, punctuation bool, extension int, ok bool) {
	if neighbor.kind != inlineDelimiterNeighborFixed {
		return false, false, 0, false
	}
	edge := neighbor.edge
	if edge.marker == 0 {
		return edge.whitespace, edge.punctuation, 0, true
	}
	if edge.marker == marker {
		return edge.whitespace, edge.punctuation, edge.runLength, true
	}
	return false, true, 0, true
}

type inlineDelimiterSiteRef struct {
	delimiter int
	side      inlineDelimiterSide
}

type inlineDelimiterCoalescence struct {
	first               inlineDelimiterSiteRef
	second              inlineDelimiterSiteRef
	baseRunLength       int
	starRunLength       int
	underscoreRunLength int
	resolved            bool
}

func buildInlineDelimiterCoalescences(
	neighborhood inlineDelimiterNeighborhood,
	delimiters inlineDelimiterAnalysis,
) []inlineDelimiterCoalescence {
	edges := make([]inlineDelimiterCoalescence, 0)
	for siteIndex, site := range neighborhood.sites {
		for _, neighbor := range []inlineDelimiterNeighbor{site.inner, site.outer} {
			edge, ok := inlineDelimiterCoalescenceFor(
				neighborhood,
				delimiters,
				siteIndex,
				site,
				neighbor,
			)
			if ok {
				edges = append(edges, edge)
			}
		}
	}
	return edges
}

func inlineDelimiterCoalescenceFor(
	neighborhood inlineDelimiterNeighborhood,
	delimiters inlineDelimiterAnalysis,
	siteIndex int,
	site inlineDelimiterSite,
	neighbor inlineDelimiterNeighbor,
) (inlineDelimiterCoalescence, bool) {
	if neighbor.kind != inlineDelimiterNeighborEndpoint {
		return inlineDelimiterCoalescence{}, false
	}
	otherIndex := neighbor.delimiter*2 + int(neighbor.side)
	if otherIndex <= siteIndex || otherIndex < 0 || otherIndex >= len(neighborhood.sites) {
		return inlineDelimiterCoalescence{}, false
	}
	first := inlineDelimiterSiteRef{delimiter: site.delimiter, side: site.side}
	second := inlineDelimiterSiteRef{delimiter: neighbor.delimiter, side: neighbor.side}
	other := neighborhood.sites[otherIndex]
	if !inlineDelimiterSitesAreReciprocal(other, first) {
		return inlineDelimiterCoalescence{}, false
	}
	edge := inlineDelimiterCoalescence{
		first:  first,
		second: second,
	}
	if first.delimiter < 0 || first.delimiter >= len(delimiters.nodes) ||
		second.delimiter < 0 || second.delimiter >= len(delimiters.nodes) {
		return edge, true
	}
	edge.baseRunLength = delimiters.nodes[first.delimiter].width +
		delimiters.nodes[second.delimiter].width
	resolveInlineDelimiterCoalescence(&edge, site, other)
	return edge, true
}

func resolveInlineDelimiterCoalescence(
	edge *inlineDelimiterCoalescence,
	firstSite, secondSite inlineDelimiterSite,
) {
	firstFar, firstOK := inlineDelimiterFarNeighbor(firstSite, edge.second)
	secondFar, secondOK := inlineDelimiterFarNeighbor(secondSite, edge.first)
	if !firstOK || !secondOK ||
		firstFar.kind != inlineDelimiterNeighborFixed ||
		secondFar.kind != inlineDelimiterNeighborFixed {
		return
	}
	edge.resolved = true
	edge.starRunLength = edge.baseRunLength +
		inlineFixedRunExtension(firstFar, '*') +
		inlineFixedRunExtension(secondFar, '*')
	edge.underscoreRunLength = edge.baseRunLength +
		inlineFixedRunExtension(firstFar, '_') +
		inlineFixedRunExtension(secondFar, '_')
}

func inlineDelimiterSitesAreReciprocal(site inlineDelimiterSite, other inlineDelimiterSiteRef) bool {
	return inlineDelimiterNeighborMatches(site.inner, other) ||
		inlineDelimiterNeighborMatches(site.outer, other)
}

func inlineDelimiterFarNeighbor(
	site inlineDelimiterSite,
	other inlineDelimiterSiteRef,
) (inlineDelimiterNeighbor, bool) {
	innerMatches := inlineDelimiterNeighborMatches(site.inner, other)
	outerMatches := inlineDelimiterNeighborMatches(site.outer, other)
	switch {
	case innerMatches && !outerMatches:
		return site.outer, true
	case outerMatches && !innerMatches:
		return site.inner, true
	default:
		return inlineDelimiterNeighbor{}, false
	}
}

func inlineDelimiterNeighborMatches(
	neighbor inlineDelimiterNeighbor,
	ref inlineDelimiterSiteRef,
) bool {
	return neighbor.kind == inlineDelimiterNeighborEndpoint &&
		neighbor.delimiter == ref.delimiter &&
		neighbor.side == ref.side
}

func inlineFixedRunExtension(neighbor inlineDelimiterNeighbor, marker byte) int {
	if neighbor.kind != inlineDelimiterNeighborFixed || neighbor.edge.marker != marker {
		return 0
	}
	return neighbor.edge.runLength
}

type inlineDelimiterRunCandidate struct {
	runLength int
	resolved  bool
	allowed   bool
	canOpen   bool
	canClose  bool
}

type inlineDelimiterRunComponent struct {
	star          inlineDelimiterRunCandidate
	underscore    inlineDelimiterRunCandidate
	requiredOpen  bool
	requiredClose bool
}

func buildInlineDelimiterRunComponents(
	ast inlineAST,
	neighborhood inlineDelimiterNeighborhood,
	delimiters inlineDelimiterAnalysis,
) ([]inlineDelimiterRunComponent, error) {
	marks := make([]int, len(neighborhood.sites))
	for index := range marks {
		marks[index] = -1
	}
	components := make([]inlineDelimiterRunComponent, 0)
	componentID := 0
	for start := range neighborhood.sites {
		if marks[start] >= 0 {
			continue
		}
		members, err := collectInlineDelimiterComponent(neighborhood, marks, componentID, start)
		if err != nil {
			return nil, err
		}
		componentID++
		if len(members) < 2 {
			continue
		}
		component, err := analyzeInlineDelimiterRunComponent(
			ast,
			neighborhood,
			delimiters,
			marks,
			componentID-1,
			members,
		)
		if err != nil {
			return nil, err
		}
		components = append(components, component)
	}
	return components, nil
}

func collectInlineDelimiterComponent(
	neighborhood inlineDelimiterNeighborhood,
	marks []int,
	componentID, start int,
) ([]int, error) {
	stack := []int{start}
	marks[start] = componentID
	members := make([]int, 0, 4)
	for len(stack) != 0 {
		index := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		members = append(members, index)
		neighbors, count := inlineDelimiterEndpointNeighborIndices(neighborhood.sites[index])
		for offset := 0; offset < count; offset++ {
			neighbor := neighbors[offset]
			if neighbor < 0 || neighbor >= len(neighborhood.sites) {
				return nil, fmt.Errorf("%w: delimiter endpoint index %d", ErrInvalidInput, neighbor)
			}
			if marks[neighbor] >= 0 {
				continue
			}
			marks[neighbor] = componentID
			stack = append(stack, neighbor)
		}
	}
	return members, nil
}

func analyzeInlineDelimiterRunComponent(
	ast inlineAST,
	neighborhood inlineDelimiterNeighborhood,
	delimiters inlineDelimiterAnalysis,
	marks []int,
	componentID int,
	members []int,
) (inlineDelimiterRunComponent, error) {
	component := inlineDelimiterRunComponent{}
	baseLength := 0
	terminals := make([]int, 0, 2)
	for _, siteIndex := range members {
		site := neighborhood.sites[siteIndex]
		if site.delimiter < 0 || site.delimiter >= len(delimiters.nodes) {
			return inlineDelimiterRunComponent{}, fmt.Errorf("%w: delimiter index %d", ErrInvalidInput, site.delimiter)
		}
		baseLength += delimiters.nodes[site.delimiter].width
		if site.side == inlineDelimiterOpen {
			component.requiredOpen = true
		} else {
			component.requiredClose = true
		}
		if inlineDelimiterComponentDegree(site, marks, componentID) == 1 {
			terminals = append(terminals, siteIndex)
		}
	}
	if len(terminals) != 2 {
		return component, nil
	}
	left, right, err := orderInlineDelimiterTerminals(ast, delimiters, neighborhood, terminals)
	if err != nil {
		return inlineDelimiterRunComponent{}, err
	}
	leftFar, leftOK := inlineDelimiterTerminalFarNeighbor(neighborhood.sites[left], marks, componentID)
	rightFar, rightOK := inlineDelimiterTerminalFarNeighbor(neighborhood.sites[right], marks, componentID)
	if !leftOK || !rightOK {
		return component, nil
	}
	component.star = analyzeInlineDelimiterRunCandidate(
		leftFar, rightFar, baseLength, '*', component.requiredOpen, component.requiredClose,
	)
	component.underscore = analyzeInlineDelimiterRunCandidate(
		leftFar, rightFar, baseLength, '_', component.requiredOpen, component.requiredClose,
	)
	return component, nil
}

func inlineDelimiterEndpointNeighborIndices(site inlineDelimiterSite) ([2]int, int) {
	var indices [2]int
	count := 0
	for _, neighbor := range []inlineDelimiterNeighbor{site.inner, site.outer} {
		if neighbor.kind != inlineDelimiterNeighborEndpoint {
			continue
		}
		indices[count] = neighbor.delimiter*2 + int(neighbor.side)
		count++
	}
	return indices, count
}

func inlineDelimiterComponentDegree(site inlineDelimiterSite, marks []int, componentID int) int {
	neighbors, count := inlineDelimiterEndpointNeighborIndices(site)
	degree := 0
	for index := 0; index < count; index++ {
		neighbor := neighbors[index]
		if neighbor >= 0 && neighbor < len(marks) && marks[neighbor] == componentID {
			degree++
		}
	}
	return degree
}

func orderInlineDelimiterTerminals(
	ast inlineAST,
	delimiters inlineDelimiterAnalysis,
	neighborhood inlineDelimiterNeighborhood,
	terminals []int,
) (int, int, error) {
	firstOrder, err := inlineDelimiterSiteOrder(ast, delimiters, neighborhood.sites[terminals[0]])
	if err != nil {
		return 0, 0, err
	}
	secondOrder, err := inlineDelimiterSiteOrder(ast, delimiters, neighborhood.sites[terminals[1]])
	if err != nil {
		return 0, 0, err
	}
	if firstOrder <= secondOrder {
		return terminals[0], terminals[1], nil
	}
	return terminals[1], terminals[0], nil
}

func inlineDelimiterSiteOrder(
	ast inlineAST,
	delimiters inlineDelimiterAnalysis,
	site inlineDelimiterSite,
) (int, error) {
	if site.delimiter < 0 || site.delimiter >= len(delimiters.nodes) {
		return 0, ErrInvalidInput
	}
	astNode := delimiters.nodes[site.delimiter].astNode
	if astNode <= ast.root || astNode >= len(ast.nodes) {
		return 0, ErrInvalidInput
	}
	node := ast.nodes[astNode]
	if site.side == inlineDelimiterOpen {
		return node.eventIndex, nil
	}
	return node.exitIndex, nil
}

func inlineDelimiterTerminalFarNeighbor(
	site inlineDelimiterSite,
	marks []int,
	componentID int,
) (inlineDelimiterNeighbor, bool) {
	innerIn := inlineDelimiterNeighborInComponent(site.inner, marks, componentID)
	outerIn := inlineDelimiterNeighborInComponent(site.outer, marks, componentID)
	switch {
	case innerIn && !outerIn:
		return site.outer, true
	case outerIn && !innerIn:
		return site.inner, true
	default:
		return inlineDelimiterNeighbor{}, false
	}
}

func inlineDelimiterNeighborInComponent(
	neighbor inlineDelimiterNeighbor,
	marks []int,
	componentID int,
) bool {
	if neighbor.kind != inlineDelimiterNeighborEndpoint {
		return false
	}
	index := neighbor.delimiter*2 + int(neighbor.side)
	return index >= 0 && index < len(marks) && marks[index] == componentID
}

func analyzeInlineDelimiterRunCandidate(
	before, after inlineDelimiterNeighbor,
	baseLength int,
	marker byte,
	requiredOpen, requiredClose bool,
) inlineDelimiterRunCandidate {
	beforeWhitespace, beforePunctuation, beforeExtension, beforeOK :=
		inlineFixedBoundaryClass(before, marker)
	afterWhitespace, afterPunctuation, afterExtension, afterOK :=
		inlineFixedBoundaryClass(after, marker)
	if !beforeOK || !afterOK {
		return inlineDelimiterRunCandidate{}
	}
	canOpen, canClose := parser.DelimiterFlankingFromClasses(
		beforeWhitespace,
		beforePunctuation,
		afterWhitespace,
		afterPunctuation,
		marker,
	)
	return inlineDelimiterRunCandidate{
		runLength: baseLength + beforeExtension + afterExtension,
		resolved:  true,
		allowed:   (!requiredOpen || canOpen) && (!requiredClose || canClose),
		canOpen:   canOpen,
		canClose:  canClose,
	}
}
