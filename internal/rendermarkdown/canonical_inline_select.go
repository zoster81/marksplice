package rendermarkdown

import (
	"slices"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

type canonicalInlineCandidate struct {
	output            []byte
	delimiterSites    []canonicalInlineDelimiterSite
	owners            []parser.Range
	pairs             []native.DelimiterTopologyPair
	extendedAutolinks []native.ExtendedAutolinkOwner
	bareWWW           canonicalBareWWWContinuation
	alternationCost   int
}

// canonicalInlineProofWorkspace reuses temporary proof indexes within one
// serial host selection. Candidate results remain independently owned.
type canonicalInlineProofWorkspace struct {
	emitter        canonicalInlineEmitWorkspace
	topology       native.DelimiterTopologyCandidate
	delimiterNodes []canonicalInlineDelimiterNodePair
}

type canonicalInlineChoiceCost struct {
	alternation  int
	strikeSingle int
	payloads     []canonicalInlinePayloadChoice
	markers      []byte
}

func canonicalInlinePayloadChoicesLess(left, right []canonicalInlinePayloadChoice) bool {
	count := len(left)
	if len(right) < count {
		count = len(right)
	}
	for index := 0; index < count; index++ {
		if left[index] == right[index] {
			continue
		}
		return left[index] < right[index]
	}
	return len(left) < len(right)
}

func canonicalInlineMarkerChoicesLess(left, right []byte) bool {
	count := len(left)
	if len(right) < count {
		count = len(right)
	}
	for index := 0; index < count; index++ {
		if left[index] == right[index] {
			continue
		}
		return left[index] == '*' && right[index] == '_'
	}
	return len(left) < len(right)
}

func canonicalInlineChoiceBetter(left, right canonicalInlineChoiceCost) bool {
	if left.alternation != right.alternation {
		return left.alternation < right.alternation
	}
	if left.strikeSingle != right.strikeSingle {
		return left.strikeSingle < right.strikeSingle
	}
	if canonicalInlinePayloadChoicesLess(left.payloads, right.payloads) {
		return true
	}
	if canonicalInlinePayloadChoicesLess(right.payloads, left.payloads) {
		return false
	}
	return canonicalInlineMarkerChoicesLess(left.markers, right.markers)
}

type canonicalInlinePruneObservation struct {
	disposition canonicalInlinePruneDisposition
	key         canonicalInlinePruneKey
	cost        canonicalInlineChoiceCost
}

func canonicalInlinePruneSelection(
	observations []canonicalInlinePruneObservation,
) []int {
	markerless := -1
	deferred := make([]int, 0)
	byKey := make(map[canonicalInlinePruneKey]int)
	keyOrder := make([]canonicalInlinePruneKey, 0)

	for index, observation := range observations {
		switch observation.disposition {
		case canonicalInlinePruneMarkerless:
			if markerless < 0 ||
				canonicalInlineChoiceBetter(observation.cost, observations[markerless].cost) {
				markerless = index
			}
		case canonicalInlinePruneDeferred:
			deferred = append(deferred, index)
		case canonicalInlinePruneKeyed:
			best, exists := byKey[observation.key]
			if !exists {
				byKey[observation.key] = index
				keyOrder = append(keyOrder, observation.key)
			} else if canonicalInlineChoiceBetter(observation.cost, observations[best].cost) {
				byKey[observation.key] = index
			}
		}
	}

	result := make([]int, 0, len(deferred)+len(keyOrder)+1)
	if markerless >= 0 {
		result = append(result, markerless)
	}
	result = append(result, deferred...)
	for _, key := range keyOrder {
		result = append(result, byKey[key])
	}
	return result
}

func canonicalInlineCandidateForPlan(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	plan canonicalInlinePlan,
	context canonicalInlineEmitContext,
	tableCell bool,
) (canonicalInlineCandidate, error) {
	var workspace canonicalInlineProofWorkspace
	return workspace.candidateForPlan(ast, normalization, plan, context, tableCell)
}

func (w *canonicalInlineProofWorkspace) candidateForPlan(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	plan canonicalInlinePlan,
	context canonicalInlineEmitContext,
	tableCell bool,
) (canonicalInlineCandidate, error) {
	metadata := canonicalInlineCandidateMetadata{}
	output, err := w.emitter.emit(
		ast,
		normalization,
		plan,
		context,
		tableCell,
		&metadata,
	)
	if err != nil {
		return canonicalInlineCandidate{}, err
	}
	pairs, alternationCost, ok := w.candidateProof(ast, metadata.delimiterSites)
	if !ok {
		return canonicalInlineCandidate{}, ErrInvalidInput
	}
	return canonicalInlineCandidate{
		output:            output,
		delimiterSites:    metadata.delimiterSites,
		owners:            metadata.owners,
		pairs:             pairs,
		extendedAutolinks: metadata.extendedAutolinks,
		bareWWW:           metadata.bareWWW,
		alternationCost:   alternationCost,
	}, nil
}

type canonicalInlineDelimiterNodePair struct {
	marker   byte
	opening  parser.Range
	closing  parser.Range
	openSet  bool
	closeSet bool
}

func canonicalInlineDelimiterNodePairs(
	ast canonicalInlineAST,
	sites []canonicalInlineDelimiterSite,
) ([]canonicalInlineDelimiterNodePair, bool) {
	return canonicalInlineDelimiterNodePairsUsing(ast, sites, nil)
}

func canonicalInlineDelimiterNodePairsUsing(
	ast canonicalInlineAST,
	sites []canonicalInlineDelimiterSite,
	nodes []canonicalInlineDelimiterNodePair,
) ([]canonicalInlineDelimiterNodePair, bool) {
	nodes = slices.Grow(nodes[:0], len(ast.nodes))[:len(ast.nodes)]
	clear(nodes)
	for _, site := range sites {
		if site.node <= ast.root || site.node >= len(ast.nodes) {
			return nil, false
		}
		eventIndex := ast.nodes[site.node].eventIndex
		if eventIndex < 0 || eventIndex >= len(ast.events) {
			return nil, false
		}
		switch ast.events[eventIndex].Kind {
		case parser.SemanticEmphasis, parser.SemanticStrong, parser.SemanticStrikethrough:
		default:
			return nil, false
		}
		node := &nodes[site.node]
		if node.marker != 0 && node.marker != site.marker {
			return nil, false
		}
		node.marker = site.marker
		if site.open {
			if node.openSet {
				return nil, false
			}
			node.opening = site.range_
			node.openSet = true
			continue
		}
		if node.closeSet {
			return nil, false
		}
		node.closing = site.range_
		node.closeSet = true
	}
	return nodes, true
}

func canonicalInlineAlternationNode(ast canonicalInlineAST, nodeIndex int) bool {
	if nodeIndex <= ast.root || nodeIndex >= len(ast.nodes) {
		return false
	}
	eventIndex := ast.nodes[nodeIndex].eventIndex
	if eventIndex < 0 || eventIndex >= len(ast.events) {
		return false
	}
	event := ast.events[eventIndex]
	return event.Phase == parser.SemanticEnter &&
		(event.Kind == parser.SemanticEmphasis || event.Kind == parser.SemanticStrong)
}

func canonicalInlineCompleteAlternationPair(pair canonicalInlineDelimiterNodePair) bool {
	return pair.openSet && pair.closeSet && (pair.marker == '*' || pair.marker == '_')
}

func canonicalInlineAlternationCostFromNodes(
	ast canonicalInlineAST,
	nodes []canonicalInlineDelimiterNodePair,
) (int, bool) {
	if len(nodes) != len(ast.nodes) {
		return 0, false
	}
	cost := 0
	for nodeIndex := 1; nodeIndex < len(ast.nodes); nodeIndex++ {
		if !canonicalInlineAlternationNode(ast, nodeIndex) {
			continue
		}
		child := nodes[nodeIndex]
		if !canonicalInlineCompleteAlternationPair(child) {
			return 0, false
		}
		parentIndex := ast.nodes[nodeIndex].parent
		if !canonicalInlineAlternationNode(ast, parentIndex) {
			continue
		}
		parent := nodes[parentIndex]
		if !canonicalInlineCompleteAlternationPair(parent) {
			return 0, false
		}
		if child.marker == parent.marker &&
			(parent.opening.End == child.opening.Start ||
				child.closing.End == parent.closing.Start) {
			cost++
		}
	}
	return cost, true
}

func canonicalInlineAlternationCost(
	ast canonicalInlineAST,
	sites []canonicalInlineDelimiterSite,
) (int, bool) {
	nodes, ok := canonicalInlineDelimiterNodePairs(ast, sites)
	if !ok {
		return 0, false
	}
	return canonicalInlineAlternationCostFromNodes(ast, nodes)
}

func canonicalInlineDelimiterTopologyPairsFromNodes(
	nodes []canonicalInlineDelimiterNodePair,
) ([]native.DelimiterTopologyPair, bool) {
	pairs := make([]native.DelimiterTopologyPair, 0, len(nodes)/2)
	for nodeIndex := 1; nodeIndex < len(nodes); nodeIndex++ {
		node := nodes[nodeIndex]
		if !node.openSet && !node.closeSet {
			continue
		}
		if !node.openSet || !node.closeSet || node.marker == 0 {
			return nil, false
		}
		pairs = append(pairs, native.DelimiterTopologyPair{
			Marker:  node.marker,
			Opening: node.opening,
			Closing: node.closing,
		})
	}
	return pairs, true
}

func (w *canonicalInlineProofWorkspace) candidateProof(
	ast canonicalInlineAST,
	sites []canonicalInlineDelimiterSite,
) ([]native.DelimiterTopologyPair, int, bool) {
	nodes, ok := canonicalInlineDelimiterNodePairsUsing(ast, sites, w.delimiterNodes)
	w.delimiterNodes = nodes
	if !ok {
		return nil, 0, false
	}
	pairs, ok := canonicalInlineDelimiterTopologyPairsFromNodes(nodes)
	if !ok {
		return nil, 0, false
	}
	alternationCost, ok := canonicalInlineAlternationCostFromNodes(ast, nodes)
	if !ok {
		return nil, 0, false
	}
	return pairs, alternationCost, true
}

// canonicalInlineDeferredRecoverable reports whether Native can preserve the
// expected delimiter topology after adding one bounded delimiter context around
// a candidate already proven invalid in isolation.
//
// The caller must perform the standalone topology check first. Avoiding that
// duplicate parse keeps this helper suitable for the bottom-up hot path. The
// matched context is deliberately not returned: it is construction-local proof
// only and must not become exported subtree state.
func canonicalInlineDeferredRecoverable(
	output []byte,
	owners []parser.Range,
	pairs []native.DelimiterTopologyPair,
) bool {
	if len(pairs) == 0 {
		return false
	}
	contexts := [...]struct {
		prefix string
		suffix string
	}{
		{prefix: "*"}, {prefix: "**"}, {prefix: "_"}, {prefix: "__"}, {prefix: "~"}, {prefix: "~~"},
		{suffix: "*"}, {suffix: "**"}, {suffix: "_"}, {suffix: "__"}, {suffix: "~"}, {suffix: "~~"},
		{prefix: "*", suffix: "*"}, {prefix: "**", suffix: "**"},
		{prefix: "_", suffix: "_"}, {prefix: "__", suffix: "__"},
		{prefix: "~", suffix: "~"}, {prefix: "~~", suffix: "~~"},
	}
	for _, context := range contexts {
		if native.DelimiterTopologyMatchesInContext(
			output,
			owners,
			pairs,
			[]byte(context.prefix),
			[]byte(context.suffix),
		) {
			return true
		}
	}
	return false
}
