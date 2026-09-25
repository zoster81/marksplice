package rendermarkdown

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
)

type inlineDelimiterNodeConstraint struct {
	allowed inlineMarkerMask
}

type inlineDelimiterPairMask uint8

const (
	inlinePairStarStar inlineDelimiterPairMask = 1 << iota
	inlinePairStarUnderscore
	inlinePairUnderscoreStar
	inlinePairUnderscoreUnderscore
)

type inlineDelimiterPairConstraint struct {
	first   int
	second  int
	allowed inlineDelimiterPairMask
}

type inlineDelimiterConstraintSet struct {
	nodes []inlineDelimiterNodeConstraint
	pairs []inlineDelimiterPairConstraint
}

type inlineDelimiterPairKey struct {
	first  int
	second int
}

type inlinePairNodeGeometry struct {
	open     inlineDelimiterCandidateGeometry
	close    inlineDelimiterCandidateGeometry
	hasOpen  bool
	hasClose bool
}

func buildInlineDelimiterBooleanConstraints(
	ast inlineAST,
	neighborhood inlineDelimiterNeighborhood,
	delimiters inlineDelimiterAnalysis,
) (inlineDelimiterConstraintSet, error) {
	if len(neighborhood.sites) != len(delimiters.nodes)*2 {
		return inlineDelimiterConstraintSet{}, fmt.Errorf(
			"%w: delimiter site count %d",
			ErrInvalidInput,
			len(neighborhood.sites),
		)
	}
	constraints := inlineDelimiterConstraintSet{
		nodes: make([]inlineDelimiterNodeConstraint, len(delimiters.nodes)),
		pairs: make([]inlineDelimiterPairConstraint, 0),
	}
	for delimiter := range constraints.nodes {
		constraints.nodes[delimiter].allowed = inlineMarkerStar | inlineMarkerUnderscore
		restrictInlineDelimiterNode(
			&constraints.nodes[delimiter],
			neighborhood,
			delimiters,
			delimiter,
		)
	}

	coalescences := buildInlineDelimiterCoalescences(neighborhood, delimiters)
	pairGroups := make(map[inlineDelimiterPairKey][]inlineDelimiterCoalescence)
	for _, edge := range coalescences {
		if !edge.resolved {
			continue
		}
		allowed, ok := inlineResolvedPairMask(ast, neighborhood, delimiters, edge)
		if !ok {
			continue
		}
		if edge.first.delimiter == edge.second.delimiter {
			constraints.nodes[edge.first.delimiter].allowed &= inlineSelfPairMarkerMask(allowed)
			continue
		}
		constraints.pairs = append(constraints.pairs, inlineDelimiterPairConstraint{
			first:   edge.first.delimiter,
			second:  edge.second.delimiter,
			allowed: allowed,
		})
		key := inlineDelimiterPairKeyFor(edge.first.delimiter, edge.second.delimiter)
		pairGroups[key] = append(pairGroups[key], edge)
	}
	appendInlineDelimiterPairGroupConstraints(
		&constraints,
		ast,
		neighborhood,
		delimiters,
		pairGroups,
	)
	appendInlineOwnedSharedOpenConstraints(&constraints, delimiters, coalescences)
	appendInlineOwnedOpenCloseBridgeConstraints(&constraints, delimiters, coalescences)
	appendInlineOwnedAlternatingChainConstraints(&constraints, delimiters, coalescences)
	return constraints, nil
}

func appendInlineOwnedAlternatingChainConstraints(
	constraints *inlineDelimiterConstraintSet,
	delimiters inlineDelimiterAnalysis,
	coalescences []inlineDelimiterCoalescence,
) {
	if constraints == nil {
		return
	}
	openParents := inlineOwnedAlternatingOpenParents(delimiters, coalescences)
	appendInlineOwnedAlternatingCloseConstraints(
		constraints,
		delimiters,
		coalescences,
		openParents,
	)
}

func inlineOwnedAlternatingOpenParents(
	delimiters inlineDelimiterAnalysis,
	coalescences []inlineDelimiterCoalescence,
) []int {
	openParents := make([]int, len(delimiters.nodes))
	for index := range openParents {
		openParents[index] = -1
	}
	for _, edge := range coalescences {
		if !edge.resolved || edge.first.side != inlineDelimiterOpen ||
			edge.second.side != inlineDelimiterOpen {
			continue
		}
		key := inlineDelimiterPairKeyFor(edge.first.delimiter, edge.second.delimiter)
		parent, child, ok := inlineDelimiterParentChild(delimiters, key)
		if ok && inlineOwnedNodesUseOppositeMarkers(delimiters, parent, child) {
			openParents[child] = parent
		}
	}
	return openParents
}

func appendInlineOwnedAlternatingCloseConstraints(
	constraints *inlineDelimiterConstraintSet,
	delimiters inlineDelimiterAnalysis,
	coalescences []inlineDelimiterCoalescence,
	openParents []int,
) {
	same := inlinePairStarStar | inlinePairUnderscoreUnderscore
	for _, edge := range coalescences {
		if !edge.resolved || edge.first.side != inlineDelimiterClose ||
			edge.second.side != inlineDelimiterClose {
			continue
		}
		key := inlineDelimiterPairKeyFor(edge.first.delimiter, edge.second.delimiter)
		parent, child, ok := inlineDelimiterParentChild(delimiters, key)
		if !ok || parent < 0 || parent >= len(openParents) ||
			openParents[parent] < 0 || delimiters.nodes[child].sourceOwnedPrefix {
			continue
		}
		bridge := inlineDelimiterPairKeyFor(openParents[parent], child)
		constraints.pairs = append(constraints.pairs, inlineDelimiterPairConstraint{
			first: bridge.first, second: bridge.second, allowed: same,
		})
	}
}

func inlineOwnedNodesUseOppositeMarkers(
	delimiters inlineDelimiterAnalysis,
	first, second int,
) bool {
	if first < 0 || first >= len(delimiters.nodes) ||
		second < 0 || second >= len(delimiters.nodes) {
		return false
	}
	firstNode, secondNode := delimiters.nodes[first], delimiters.nodes[second]
	return firstNode.sourceOwnedMarker != 0 &&
		firstNode.sourceOwnedMarker == firstNode.sourceMarker &&
		secondNode.sourceOwnedMarker != 0 &&
		secondNode.sourceOwnedMarker == secondNode.sourceMarker &&
		firstNode.sourceOwnedMarker != secondNode.sourceOwnedMarker &&
		!firstNode.sourceOwnedPrefix && !secondNode.sourceOwnedPrefix
}

func appendInlineOwnedOpenCloseBridgeConstraints(
	constraints *inlineDelimiterConstraintSet,
	delimiters inlineDelimiterAnalysis,
	coalescences []inlineDelimiterCoalescence,
) {
	if constraints == nil {
		return
	}
	openPeers, closePeers := inlineOwnedOpenClosePeers(delimiters, coalescences)
	all := inlinePairStarStar | inlinePairStarUnderscore |
		inlinePairUnderscoreStar | inlinePairUnderscoreUnderscore
	for owned, outer := range openPeers {
		child := closePeers[owned]
		if outer < 0 || child < 0 || outer == child {
			continue
		}
		key := inlineDelimiterPairKeyFor(outer, child)
		forbidden := inlineOwnedBridgeForbiddenPair(delimiters, key, owned, outer)
		if forbidden == 0 {
			continue
		}
		constraints.pairs = append(constraints.pairs, inlineDelimiterPairConstraint{
			first: key.first, second: key.second, allowed: all &^ forbidden,
		})
	}
}

func inlineOwnedOpenClosePeers(
	delimiters inlineDelimiterAnalysis,
	coalescences []inlineDelimiterCoalescence,
) ([]int, []int) {
	openPeers := make([]int, len(delimiters.nodes))
	closePeers := make([]int, len(delimiters.nodes))
	for index := range openPeers {
		openPeers[index] = -1
		closePeers[index] = -1
	}
	for _, edge := range coalescences {
		if owned, other, ok := inlineOwnedSharedOpenPair(delimiters, edge); ok {
			openPeers[owned] = other
		}
		if parent, child, ok := inlineOwnedCloseParentChild(delimiters, edge); ok {
			closePeers[parent] = child
		}
	}
	return openPeers, closePeers
}

func inlineOwnedCloseParentChild(
	delimiters inlineDelimiterAnalysis,
	edge inlineDelimiterCoalescence,
) (parent, child int, ok bool) {
	if !edge.resolved || edge.first.side != inlineDelimiterClose ||
		edge.second.side != inlineDelimiterClose {
		return 0, 0, false
	}
	key := inlineDelimiterPairKeyFor(edge.first.delimiter, edge.second.delimiter)
	parent, child, ok = inlineDelimiterParentChild(delimiters, key)
	if !ok {
		return 0, 0, false
	}
	parentNode := delimiters.nodes[parent]
	if parentNode.sourceOwnedMarker == 0 ||
		parentNode.sourceOwnedMarker != parentNode.sourceMarker ||
		delimiters.nodes[child].sourceOwnedPrefix {
		return 0, 0, false
	}
	return parent, child, true
}

func inlineOwnedBridgeForbiddenPair(
	delimiters inlineDelimiterAnalysis,
	key inlineDelimiterPairKey,
	owned, outer int,
) inlineDelimiterPairMask {
	if owned < 0 || owned >= len(delimiters.nodes) {
		return 0
	}
	marker := delimiters.nodes[owned].sourceOwnedMarker
	opposite, ok := inlineOppositeMarkerByte(marker)
	if !ok {
		return 0
	}
	return inlinePairBitForIndexedMarkers(key, outer, opposite, marker)
}

func appendInlineOwnedSharedOpenConstraints(
	constraints *inlineDelimiterConstraintSet,
	delimiters inlineDelimiterAnalysis,
	coalescences []inlineDelimiterCoalescence,
) {
	if constraints == nil {
		return
	}
	pairMasks := inlineEffectivePairMasks(constraints.pairs)
	oppositeChild := inlineOwnedDelimiterOppositeChildren(*constraints, delimiters, pairMasks)
	same := inlinePairStarStar | inlinePairUnderscoreUnderscore
	for _, edge := range coalescences {
		owned, other, ok := inlineOwnedSharedOpenPair(delimiters, edge)
		if !ok || !oppositeChild[owned] {
			continue
		}
		key := inlineDelimiterPairKeyFor(owned, other)
		constraints.pairs = append(constraints.pairs, inlineDelimiterPairConstraint{
			first: key.first, second: key.second, allowed: same,
		})
	}
}

func inlineEffectivePairMasks(
	pairs []inlineDelimiterPairConstraint,
) map[inlineDelimiterPairKey]inlineDelimiterPairMask {
	masks := make(map[inlineDelimiterPairKey]inlineDelimiterPairMask, len(pairs))
	for _, pair := range pairs {
		key := inlineDelimiterPairKeyFor(pair.first, pair.second)
		allowed := pair.allowed
		if pair.first != key.first {
			allowed = inlineTransposePairMask(allowed)
		}
		if current, ok := masks[key]; ok {
			masks[key] = current & allowed
			continue
		}
		masks[key] = allowed
	}
	return masks
}

func inlineOwnedDelimiterOppositeChildren(
	constraints inlineDelimiterConstraintSet,
	delimiters inlineDelimiterAnalysis,
	pairMasks map[inlineDelimiterPairKey]inlineDelimiterPairMask,
) []bool {
	result := make([]bool, len(delimiters.nodes))
	for key, allowed := range pairMasks {
		parent, child, ok := inlineDelimiterParentChild(delimiters, key)
		if !ok {
			continue
		}
		parentNode := delimiters.nodes[parent]
		childNode := delimiters.nodes[child]
		if parentNode.sourceOwnedMarker == 0 ||
			parentNode.sourceOwnedMarker != parentNode.sourceMarker ||
			childNode.sourceOwnedPrefix {
			continue
		}
		sameMarker := inlineMarkerMaskForByte(parentNode.sourceOwnedMarker)
		oppositeMarker := inlineOppositeMarkerMask(parentNode.sourceOwnedMarker)
		if sameMarker == 0 || oppositeMarker == 0 {
			continue
		}
		sameBit := inlinePairBitForMarkers(parentNode.sourceOwnedMarker, parentNode.sourceOwnedMarker)
		oppositeByte, ok := inlineOppositeMarkerByte(parentNode.sourceOwnedMarker)
		if !ok {
			continue
		}
		oppositeBit := inlinePairBitForIndexedMarkers(key, parent, parentNode.sourceOwnedMarker, oppositeByte)
		sameFeasible := allowed&sameBit != 0 && constraints.nodes[child].allowed&sameMarker != 0
		oppositeFeasible := allowed&oppositeBit != 0 && constraints.nodes[child].allowed&oppositeMarker != 0
		if !sameFeasible && oppositeFeasible {
			result[parent] = true
		}
	}
	return result
}

func inlineDelimiterParentChild(
	delimiters inlineDelimiterAnalysis,
	key inlineDelimiterPairKey,
) (parent, child int, ok bool) {
	if key.first < 0 || key.first >= len(delimiters.nodes) ||
		key.second < 0 || key.second >= len(delimiters.nodes) {
		return 0, 0, false
	}
	switch {
	case delimiters.nodes[key.second].parent == key.first:
		return key.first, key.second, true
	case delimiters.nodes[key.first].parent == key.second:
		return key.second, key.first, true
	default:
		return 0, 0, false
	}
}

func inlineOwnedSharedOpenPair(
	delimiters inlineDelimiterAnalysis,
	edge inlineDelimiterCoalescence,
) (owned, other int, ok bool) {
	if !edge.resolved || edge.first.side != inlineDelimiterOpen ||
		edge.second.side != inlineDelimiterOpen {
		return 0, 0, false
	}
	firstIndex, secondIndex := edge.first.delimiter, edge.second.delimiter
	if firstIndex < 0 || firstIndex >= len(delimiters.nodes) ||
		secondIndex < 0 || secondIndex >= len(delimiters.nodes) {
		return 0, 0, false
	}
	first, second := delimiters.nodes[firstIndex], delimiters.nodes[secondIndex]
	if !inlineDelimiterNodesShareSourceOpen(first, second) {
		return 0, 0, false
	}
	firstOwned := first.sourceOwnedMarker == first.sourceMarker
	secondOwned := second.sourceOwnedMarker == second.sourceMarker
	if firstOwned == secondOwned {
		return 0, 0, false
	}
	if firstOwned {
		return firstIndex, secondIndex, true
	}
	return secondIndex, firstIndex, true
}

func inlineDelimiterNodesShareSourceOpen(first, second inlineDelimiterNode) bool {
	if first.width <= 0 || first.width != second.width ||
		first.sourceMarker == 0 || first.sourceMarker != second.sourceMarker {
		return false
	}
	return first.range_.Start+first.width == second.range_.Start ||
		second.range_.Start+second.width == first.range_.Start
}

func inlineTransposePairMask(mask inlineDelimiterPairMask) inlineDelimiterPairMask {
	out := mask & (inlinePairStarStar | inlinePairUnderscoreUnderscore)
	if mask&inlinePairStarUnderscore != 0 {
		out |= inlinePairUnderscoreStar
	}
	if mask&inlinePairUnderscoreStar != 0 {
		out |= inlinePairStarUnderscore
	}
	return out
}

func inlineOppositeMarkerMask(marker byte) inlineMarkerMask {
	switch marker {
	case '*':
		return inlineMarkerUnderscore
	case '_':
		return inlineMarkerStar
	default:
		return 0
	}
}

func inlinePairBitForMarkers(first, second byte) inlineDelimiterPairMask {
	switch {
	case first == '*' && second == '*':
		return inlinePairStarStar
	case first == '*' && second == '_':
		return inlinePairStarUnderscore
	case first == '_' && second == '*':
		return inlinePairUnderscoreStar
	case first == '_' && second == '_':
		return inlinePairUnderscoreUnderscore
	default:
		return 0
	}
}

func inlineOppositeMarkerByte(marker byte) (byte, bool) {
	switch marker {
	case '*':
		return '_', true
	case '_':
		return '*', true
	default:
		return 0, false
	}
}

func inlinePairBitForIndexedMarkers(
	key inlineDelimiterPairKey,
	firstIndex int,
	firstMarker, secondMarker byte,
) inlineDelimiterPairMask {
	if firstIndex == key.first {
		return inlinePairBitForMarkers(firstMarker, secondMarker)
	}
	return inlinePairBitForMarkers(secondMarker, firstMarker)
}

func inlineDelimiterPairKeyFor(first, second int) inlineDelimiterPairKey {
	if second < first {
		first, second = second, first
	}
	return inlineDelimiterPairKey{first: first, second: second}
}

func appendInlineDelimiterPairGroupConstraints(
	constraints *inlineDelimiterConstraintSet,
	ast inlineAST,
	neighborhood inlineDelimiterNeighborhood,
	delimiters inlineDelimiterAnalysis,
	groups map[inlineDelimiterPairKey][]inlineDelimiterCoalescence,
) {
	if constraints == nil {
		return
	}
	for key, edges := range groups {
		if len(edges) < 2 {
			continue
		}
		allowed := inlineResolvedPairGroupMask(ast, neighborhood, delimiters, key, edges)
		constraints.pairs = append(constraints.pairs, inlineDelimiterPairConstraint{
			first: key.first, second: key.second, allowed: allowed,
		})
	}
}

func inlineResolvedPairGroupMask(
	ast inlineAST,
	neighborhood inlineDelimiterNeighborhood,
	delimiters inlineDelimiterAnalysis,
	key inlineDelimiterPairKey,
	edges []inlineDelimiterCoalescence,
) inlineDelimiterPairMask {
	assignments := [...]struct {
		first  byte
		second byte
		bit    inlineDelimiterPairMask
	}{
		{first: '*', second: '*', bit: inlinePairStarStar},
		{first: '*', second: '_', bit: inlinePairStarUnderscore},
		{first: '_', second: '*', bit: inlinePairUnderscoreStar},
		{first: '_', second: '_', bit: inlinePairUnderscoreUnderscore},
	}
	allowed := inlineDelimiterPairMask(0)
	for _, assignment := range assignments {
		if inlinePairGroupAssignmentAllowed(
			ast, neighborhood, delimiters, key, edges, assignment.first, assignment.second,
		) {
			allowed |= assignment.bit
		}
	}
	return allowed
}

func inlinePairGroupAssignmentAllowed(
	ast inlineAST,
	neighborhood inlineDelimiterNeighborhood,
	delimiters inlineDelimiterAnalysis,
	key inlineDelimiterPairKey,
	edges []inlineDelimiterCoalescence,
	firstMarker, secondMarker byte,
) bool {
	var geometry [2]inlinePairNodeGeometry
	for _, edge := range edges {
		if !inlineRecordPairEdgeGeometry(
			ast, neighborhood, delimiters, key, edge,
			firstMarker, secondMarker, &geometry,
		) {
			return false
		}
	}
	for _, node := range geometry {
		if node.hasOpen && node.hasClose &&
			parser.DelimiterRunsHaveModuloThreeConflict(
				node.open.runLength,
				node.open.canClose,
				node.close.runLength,
				node.close.canOpen,
			) {
			return false
		}
	}
	if firstMarker == secondMarker &&
		inlinePairGroupSharesBothBoundaries(edges) &&
		!inlinePairGroupResolutionLevelsAllowed(key, geometry, delimiters, firstMarker) {
		return false
	}
	return true
}

func inlinePairGroupSharesBothBoundaries(edges []inlineDelimiterCoalescence) bool {
	hasOpen := false
	hasClose := false
	for _, edge := range edges {
		switch {
		case edge.first.side == inlineDelimiterOpen && edge.second.side == inlineDelimiterOpen:
			hasOpen = true
		case edge.first.side == inlineDelimiterClose && edge.second.side == inlineDelimiterClose:
			hasClose = true
		}
	}
	return hasOpen && hasClose
}

func inlinePairGroupResolutionLevelsAllowed(
	key inlineDelimiterPairKey,
	geometry [2]inlinePairNodeGeometry,
	delimiters inlineDelimiterAnalysis,
	marker byte,
) bool {
	if !geometry[0].hasOpen || !geometry[0].hasClose ||
		!geometry[1].hasOpen || !geometry[1].hasClose {
		return true
	}
	firstWidth, firstOK := inlineDelimiterWidth(delimiters, key.first)
	secondWidth, secondOK := inlineDelimiterWidth(delimiters, key.second)
	if !firstOK || !secondOK {
		return true
	}
	totalWidth := firstWidth + secondWidth
	if geometry[0].open.runLength != totalWidth ||
		geometry[0].close.runLength != totalWidth ||
		geometry[1].open != geometry[0].open ||
		geometry[1].close != geometry[0].close {
		return true
	}
	gap := geometry[0].open.runLength + 1
	matches := parser.ResolveDelimiterRuns([]parser.DelimiterRun{
		{
			Start: 0, End: geometry[0].open.runLength, Marker: marker,
			CanOpen: geometry[0].open.canOpen, CanClose: geometry[0].open.canClose,
		},
		{
			Start: gap, End: gap + geometry[0].close.runLength, Marker: marker,
			CanOpen: geometry[0].close.canOpen, CanClose: geometry[0].close.canClose,
		},
	})
	if len(matches) != 2 {
		return false
	}
	return (matches[0].Level == firstWidth && matches[1].Level == secondWidth) ||
		(matches[0].Level == secondWidth && matches[1].Level == firstWidth)
}

func inlineRecordPairEdgeGeometry(
	ast inlineAST,
	neighborhood inlineDelimiterNeighborhood,
	delimiters inlineDelimiterAnalysis,
	key inlineDelimiterPairKey,
	edge inlineDelimiterCoalescence,
	firstMarker, secondMarker byte,
	geometry *[2]inlinePairNodeGeometry,
) bool {
	if geometry == nil {
		return false
	}
	firstSite, firstOK := inlineDelimiterSiteByRef(neighborhood, edge.first)
	secondSite, secondOK := inlineDelimiterSiteByRef(neighborhood, edge.second)
	firstWidth, firstWidthOK := inlineDelimiterWidth(delimiters, edge.first.delimiter)
	secondWidth, secondWidthOK := inlineDelimiterWidth(delimiters, edge.second.delimiter)
	if !firstOK || !secondOK || !firstWidthOK || !secondWidthOK {
		return false
	}
	edgeFirstMarker, firstIndex, firstMarkerOK := inlinePairGroupMarker(
		key, edge.first.delimiter, firstMarker, secondMarker,
	)
	edgeSecondMarker, secondIndex, secondMarkerOK := inlinePairGroupMarker(
		key, edge.second.delimiter, firstMarker, secondMarker,
	)
	if !firstMarkerOK || !secondMarkerOK {
		return false
	}
	firstCandidate, secondCandidate, ok := inlinePairAssignmentCandidates(
		ast, delimiters, edge, firstSite, secondSite,
		firstWidth, secondWidth, edgeFirstMarker, edgeSecondMarker,
	)
	return ok &&
		inlineDelimiterCandidateAllowsSite(firstSite, firstCandidate) &&
		inlineDelimiterCandidateAllowsSite(secondSite, secondCandidate) &&
		inlineRecordPairGeometry(&geometry[firstIndex], firstSite.side, firstCandidate) &&
		inlineRecordPairGeometry(&geometry[secondIndex], secondSite.side, secondCandidate)
}

func inlinePairGroupMarker(
	key inlineDelimiterPairKey,
	delimiter int,
	firstMarker, secondMarker byte,
) (byte, int, bool) {
	switch delimiter {
	case key.first:
		return firstMarker, 0, true
	case key.second:
		return secondMarker, 1, true
	default:
		return 0, 0, false
	}
}

func inlineRecordPairGeometry(
	node *inlinePairNodeGeometry,
	side inlineDelimiterSide,
	candidate inlineDelimiterCandidateGeometry,
) bool {
	if node == nil || !candidate.resolved {
		return false
	}
	slot := &node.open
	set := &node.hasOpen
	if side == inlineDelimiterClose {
		slot = &node.close
		set = &node.hasClose
	}
	if *set {
		return *slot == candidate
	}
	*slot = candidate
	*set = true
	return true
}

func restrictInlineDelimiterNode(
	constraint *inlineDelimiterNodeConstraint,
	neighborhood inlineDelimiterNeighborhood,
	delimiters inlineDelimiterAnalysis,
	delimiter int,
) {
	if constraint == nil || delimiter < 0 || delimiter >= len(delimiters.nodes) {
		return
	}
	node := delimiters.nodes[delimiter]
	width := node.width
	if owned := inlineMarkerMaskForByte(node.sourceOwnedMarker); owned != 0 {
		constraint.allowed &= owned
	}
	open := analyzeInlineDelimiterFixedGeometry(
		neighborhood.site(delimiter, inlineDelimiterOpen),
		width,
	)
	close := analyzeInlineDelimiterFixedGeometry(
		neighborhood.site(delimiter, inlineDelimiterClose),
		width,
	)
	if inlineFixedGeometryResolved(open) {
		constraint.allowed &= open.allowedOpen
	}
	if inlineFixedGeometryResolved(close) {
		constraint.allowed &= close.allowedClose
	}
	restrictInlineDelimiterModuloThree(constraint, open, close)
}

func inlineMarkerMaskForByte(marker byte) inlineMarkerMask {
	switch marker {
	case '*':
		return inlineMarkerStar
	case '_':
		return inlineMarkerUnderscore
	default:
		return 0
	}
}

func inlineFixedGeometryResolved(geometry inlineDelimiterFixedGeometry) bool {
	return geometry.star.resolved && geometry.underscore.resolved
}

func restrictInlineDelimiterModuloThree(
	constraint *inlineDelimiterNodeConstraint,
	open, close inlineDelimiterFixedGeometry,
) {
	if constraint == nil {
		return
	}
	if constraint.allowed&inlineMarkerStar != 0 &&
		open.star.resolved && close.star.resolved &&
		parser.DelimiterRunsHaveModuloThreeConflict(
			open.star.runLength,
			open.star.canClose,
			close.star.runLength,
			close.star.canOpen,
		) {
		constraint.allowed &^= inlineMarkerStar
	}
	if constraint.allowed&inlineMarkerUnderscore != 0 &&
		open.underscore.resolved && close.underscore.resolved &&
		parser.DelimiterRunsHaveModuloThreeConflict(
			open.underscore.runLength,
			open.underscore.canClose,
			close.underscore.runLength,
			close.underscore.canOpen,
		) {
		constraint.allowed &^= inlineMarkerUnderscore
	}
}

func inlineSelfPairMarkerMask(allowed inlineDelimiterPairMask) inlineMarkerMask {
	mask := inlineMarkerMask(0)
	if allowed&inlinePairStarStar != 0 {
		mask |= inlineMarkerStar
	}
	if allowed&inlinePairUnderscoreUnderscore != 0 {
		mask |= inlineMarkerUnderscore
	}
	return mask
}

func inlineResolvedPairMask(
	ast inlineAST,
	neighborhood inlineDelimiterNeighborhood,
	delimiters inlineDelimiterAnalysis,
	edge inlineDelimiterCoalescence,
) (inlineDelimiterPairMask, bool) {
	firstSite, firstOK := inlineDelimiterSiteByRef(neighborhood, edge.first)
	secondSite, secondOK := inlineDelimiterSiteByRef(neighborhood, edge.second)
	if !firstOK || !secondOK {
		return 0, false
	}
	firstWidth, firstWidthOK := inlineDelimiterWidth(delimiters, edge.first.delimiter)
	secondWidth, secondWidthOK := inlineDelimiterWidth(delimiters, edge.second.delimiter)
	if !firstWidthOK || !secondWidthOK {
		return 0, false
	}

	assignments := [...]struct {
		first  byte
		second byte
		bit    inlineDelimiterPairMask
	}{
		{first: '*', second: '*', bit: inlinePairStarStar},
		{first: '*', second: '_', bit: inlinePairStarUnderscore},
		{first: '_', second: '*', bit: inlinePairUnderscoreStar},
		{first: '_', second: '_', bit: inlinePairUnderscoreUnderscore},
	}
	allowed := inlineDelimiterPairMask(0)
	for _, assignment := range assignments {
		if inlinePairAssignmentAllowed(
			ast,
			neighborhood,
			delimiters,
			edge,
			firstSite,
			secondSite,
			firstWidth,
			secondWidth,
			assignment.first,
			assignment.second,
		) {
			allowed |= assignment.bit
		}
	}
	return allowed, true
}

func inlinePairAssignmentAllowed(
	ast inlineAST,
	neighborhood inlineDelimiterNeighborhood,
	delimiters inlineDelimiterAnalysis,
	edge inlineDelimiterCoalescence,
	firstSite, secondSite inlineDelimiterSite,
	firstWidth, secondWidth int,
	firstMarker, secondMarker byte,
) bool {
	firstCandidate, secondCandidate, ok := inlinePairAssignmentCandidates(
		ast,
		delimiters,
		edge,
		firstSite,
		secondSite,
		firstWidth,
		secondWidth,
		firstMarker,
		secondMarker,
	)
	if !ok ||
		!inlineDelimiterCandidateAllowsSite(firstSite, firstCandidate) ||
		!inlineDelimiterCandidateAllowsSite(secondSite, secondCandidate) {
		return false
	}
	if firstMarker == secondMarker &&
		inlinePairIncludesSourceOwnedPrefix(delimiters, firstSite, secondSite) {
		return true
	}
	return inlinePairSiteModuloThreeAllowed(
		neighborhood,
		delimiters,
		firstSite,
		firstMarker,
		firstCandidate,
	) && inlinePairSiteModuloThreeAllowed(
		neighborhood,
		delimiters,
		secondSite,
		secondMarker,
		secondCandidate,
	)
}

func inlinePairIncludesSourceOwnedPrefix(
	delimiters inlineDelimiterAnalysis,
	firstSite, secondSite inlineDelimiterSite,
) bool {
	for _, delimiter := range [...]int{firstSite.delimiter, secondSite.delimiter} {
		if delimiter >= 0 && delimiter < len(delimiters.nodes) &&
			delimiters.nodes[delimiter].sourceOwnedPrefix {
			return true
		}
	}
	return false
}

func inlinePairAssignmentCandidates(
	ast inlineAST,
	delimiters inlineDelimiterAnalysis,
	edge inlineDelimiterCoalescence,
	firstSite, secondSite inlineDelimiterSite,
	firstWidth, secondWidth int,
	firstMarker, secondMarker byte,
) (inlineDelimiterCandidateGeometry, inlineDelimiterCandidateGeometry, bool) {
	if firstMarker == secondMarker {
		candidate, ok := inlineSameMarkerPairCandidate(
			ast,
			delimiters,
			edge,
			firstSite,
			secondSite,
			firstMarker,
		)
		return candidate, candidate, ok
	}
	firstAdjusted, firstOK := inlineDelimiterSiteWithAdjacentMarker(
		firstSite,
		edge.second,
		secondMarker,
		secondWidth,
	)
	secondAdjusted, secondOK := inlineDelimiterSiteWithAdjacentMarker(
		secondSite,
		edge.first,
		firstMarker,
		firstWidth,
	)
	if !firstOK || !secondOK {
		return inlineDelimiterCandidateGeometry{}, inlineDelimiterCandidateGeometry{}, false
	}
	firstCandidate, firstOK := inlineDelimiterSiteCandidate(firstAdjusted, firstWidth, firstMarker)
	secondCandidate, secondOK := inlineDelimiterSiteCandidate(secondAdjusted, secondWidth, secondMarker)
	return firstCandidate, secondCandidate, firstOK && secondOK
}

func inlineSameMarkerPairCandidate(
	ast inlineAST,
	delimiters inlineDelimiterAnalysis,
	edge inlineDelimiterCoalescence,
	firstSite, secondSite inlineDelimiterSite,
	marker byte,
) (inlineDelimiterCandidateGeometry, bool) {
	firstFar, firstOK := inlineDelimiterFarNeighbor(firstSite, edge.second)
	secondFar, secondOK := inlineDelimiterFarNeighbor(secondSite, edge.first)
	if !firstOK || !secondOK {
		return inlineDelimiterCandidateGeometry{}, false
	}
	firstOrder, firstOrderOK := inlineDelimiterSiteOrderValue(ast, delimiters, firstSite)
	secondOrder, secondOrderOK := inlineDelimiterSiteOrderValue(ast, delimiters, secondSite)
	if !firstOrderOK || !secondOrderOK {
		return inlineDelimiterCandidateGeometry{}, false
	}
	before, after := firstFar, secondFar
	if secondOrder < firstOrder {
		before, after = secondFar, firstFar
	}
	requiredOpen := firstSite.side == inlineDelimiterOpen ||
		secondSite.side == inlineDelimiterOpen
	requiredClose := firstSite.side == inlineDelimiterClose ||
		secondSite.side == inlineDelimiterClose
	candidate := analyzeInlineDelimiterRunCandidate(
		before,
		after,
		edge.baseRunLength,
		marker,
		requiredOpen,
		requiredClose,
	)
	if !candidate.resolved || !candidate.allowed {
		return inlineDelimiterCandidateGeometry{}, false
	}
	return inlineDelimiterCandidateGeometry{
		runLength: candidate.runLength,
		resolved:  true,
		canOpen:   candidate.canOpen,
		canClose:  candidate.canClose,
	}, true
}

func inlineDelimiterSiteOrderValue(
	ast inlineAST,
	delimiters inlineDelimiterAnalysis,
	site inlineDelimiterSite,
) (int, bool) {
	order, err := inlineDelimiterSiteOrder(ast, delimiters, site)
	return order, err == nil
}

func inlineDelimiterSiteWithAdjacentMarker(
	site inlineDelimiterSite,
	other inlineDelimiterSiteRef,
	marker byte,
	width int,
) (inlineDelimiterSite, bool) {
	replacement := inlineFixedNeighbor(inlineBoundaryEdge{
		marker:    marker,
		runLength: width,
	})
	innerMatches := inlineDelimiterNeighborMatches(site.inner, other)
	outerMatches := inlineDelimiterNeighborMatches(site.outer, other)
	switch {
	case innerMatches && !outerMatches:
		site.inner = replacement
	case outerMatches && !innerMatches:
		site.outer = replacement
	default:
		return inlineDelimiterSite{}, false
	}
	return site, true
}

func inlineDelimiterSiteCandidate(
	site inlineDelimiterSite,
	width int,
	marker byte,
) (inlineDelimiterCandidateGeometry, bool) {
	geometry := analyzeInlineDelimiterFixedGeometry(site, width)
	candidate := geometry.star
	if marker == '_' {
		candidate = geometry.underscore
	}
	return candidate, candidate.resolved
}

func inlineDelimiterCandidateAllowsSite(
	site inlineDelimiterSite,
	candidate inlineDelimiterCandidateGeometry,
) bool {
	if !candidate.resolved {
		return false
	}
	if site.side == inlineDelimiterOpen {
		return candidate.canOpen
	}
	return candidate.canClose
}

func inlinePairSiteModuloThreeAllowed(
	neighborhood inlineDelimiterNeighborhood,
	delimiters inlineDelimiterAnalysis,
	site inlineDelimiterSite,
	marker byte,
	candidate inlineDelimiterCandidateGeometry,
) bool {
	if site.delimiter >= 0 &&
		site.delimiter < len(delimiters.nodes) &&
		delimiters.nodes[site.delimiter].sourceOwnedPrefix {
		return true
	}
	width, ok := inlineDelimiterWidth(delimiters, site.delimiter)
	if !ok {
		return true
	}
	oppositeSide := inlineDelimiterClose
	if site.side == inlineDelimiterClose {
		oppositeSide = inlineDelimiterOpen
	}
	opposite, ok := inlineDelimiterSiteCandidate(
		neighborhood.site(site.delimiter, oppositeSide),
		width,
		marker,
	)
	if !ok {
		return true
	}
	if site.side == inlineDelimiterOpen {
		return !parser.DelimiterRunsHaveModuloThreeConflict(
			candidate.runLength,
			candidate.canClose,
			opposite.runLength,
			opposite.canOpen,
		)
	}
	return !parser.DelimiterRunsHaveModuloThreeConflict(
		opposite.runLength,
		opposite.canClose,
		candidate.runLength,
		candidate.canOpen,
	)
}

func inlineDelimiterSiteByRef(
	neighborhood inlineDelimiterNeighborhood,
	ref inlineDelimiterSiteRef,
) (inlineDelimiterSite, bool) {
	index := ref.delimiter*2 + int(ref.side)
	if ref.delimiter < 0 || ref.side > inlineDelimiterClose ||
		index < 0 || index >= len(neighborhood.sites) {
		return inlineDelimiterSite{}, false
	}
	site := neighborhood.sites[index]
	return site, site.delimiter == ref.delimiter && site.side == ref.side
}

func inlineDelimiterWidth(delimiters inlineDelimiterAnalysis, delimiter int) (int, bool) {
	if delimiter < 0 || delimiter >= len(delimiters.nodes) {
		return 0, false
	}
	width := delimiters.nodes[delimiter].width
	return width, width > 0
}
