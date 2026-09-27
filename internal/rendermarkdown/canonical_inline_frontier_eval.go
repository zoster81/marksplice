package rendermarkdown

import (
	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

func canonicalInlineTextOrdinals(ast canonicalInlineAST) ([]int, int) {
	ordinals := make([]int, len(ast.nodes))
	for index := range ordinals {
		ordinals[index] = noCanonicalInlineASTNode
	}
	count := 0
	for nodeIndex := 1; nodeIndex < len(ast.nodes); nodeIndex++ {
		eventIndex := ast.nodes[nodeIndex].eventIndex
		if eventIndex < 0 || eventIndex >= len(ast.events) {
			continue
		}
		event := ast.events[eventIndex]
		if event.Phase == parser.SemanticLeaf && event.Kind == parser.SemanticText {
			ordinals[nodeIndex] = count
			count++
		}
	}
	return ordinals, count
}

func canonicalInlinePayloadVariableNodes(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
) []int {
	if len(normalization.nodes) != len(ast.nodes) {
		return nil
	}
	var nodes []int
	for nodeIndex := 1; nodeIndex < len(ast.nodes); nodeIndex++ {
		eventIndex := ast.nodes[nodeIndex].eventIndex
		if eventIndex < 0 || eventIndex >= len(ast.events) {
			continue
		}
		event := ast.events[eventIndex]
		if event.Phase == parser.SemanticLeaf && event.Kind == parser.SemanticText &&
			normalization.nodes[nodeIndex].trailingAlternative != canonicalInlineNoBoundaryAlternative {
			nodes = append(nodes, nodeIndex)
		}
	}
	return nodes
}

func canonicalInlinePayloadChoices(
	normalization canonicalInlineNodeNormalization,
) []canonicalInlinePayloadChoice {
	choices := []canonicalInlinePayloadChoice{canonicalInlinePreferredPayload}
	if normalization.trailingAlternative == canonicalInlineRawTabBoundary {
		choices = append(choices, canonicalInlineRawTabPayload)
	}
	return choices
}

func canonicalInlinePayloadChoiceAvailable(
	choice canonicalInlinePayloadChoice,
	choices []canonicalInlinePayloadChoice,
) bool {
	for _, available := range choices {
		if available == choice {
			return true
		}
	}
	return false
}

func canonicalInlineDelimiterPlanNodes(ast canonicalInlineAST) (markers, strikes []int) {
	for nodeIndex := 1; nodeIndex < len(ast.nodes); nodeIndex++ {
		eventIndex := ast.nodes[nodeIndex].eventIndex
		if eventIndex < 0 || eventIndex >= len(ast.events) {
			continue
		}
		event := ast.events[eventIndex]
		if event.Phase != parser.SemanticEnter {
			continue
		}
		switch event.Kind {
		case parser.SemanticEmphasis, parser.SemanticStrong:
			markers = append(markers, nodeIndex)
		case parser.SemanticStrikethrough:
			strikes = append(strikes, nodeIndex)
		}
	}
	return markers, strikes
}

func canonicalInlinePlanForFrontierCandidate(
	ast canonicalInlineAST,
	candidate canonicalInlineFrontierCandidate,
	textOrdinals []int,
	markerNodes []int,
	strikeNodes []int,
) (canonicalInlinePlan, bool) {
	if len(candidate.markers) != len(markerNodes) ||
		len(candidate.strikeWidths) != len(strikeNodes) ||
		len(textOrdinals) != len(ast.nodes) {
		return canonicalInlinePlan{}, false
	}
	plan := newCanonicalInlinePlan(ast)
	for ordinal, node := range markerNodes {
		plan.nodes[node].marker = candidate.markers[ordinal]
	}
	for ordinal, node := range strikeNodes {
		plan.nodes[node].strikethroughWidth = candidate.strikeWidths[ordinal]
	}
	for nodeIndex, textOrdinal := range textOrdinals {
		if textOrdinal == noCanonicalInlineASTNode {
			continue
		}
		if textOrdinal < 0 || textOrdinal >= len(candidate.payloads) {
			return canonicalInlinePlan{}, false
		}
		plan.nodes[nodeIndex].payload = candidate.payloads[textOrdinal]
	}
	return plan, true
}

func canonicalInlineSelectorStateFromProof(
	proof canonicalInlineCandidate,
	topology *native.DelimiterTopologyCandidate,
	withTilde bool,
	deferred bool,
) (canonicalInlineSelectorState, bool) {
	facts, ok := topology.TransferFacts(withTilde && !deferred)
	if !ok {
		return canonicalInlineSelectorState{}, false
	}
	state := canonicalInlineSelectorState{
		base:                 canonicalInlineTransferStateFromFacts(proof.output, facts),
		extendedWWW:          proof.bareWWW,
		deferredContinuation: deferred,
		tildeOpener:          facts.TildeForbidden,
	}
	return state, true
}

type canonicalInlineFrontierPayloadSpace struct {
	markerNodes      []int
	strikeNodes      []int
	textOrdinals     []int
	choicesByOrdinal [][]canonicalInlinePayloadChoice
	unknown          []int
	knownValid       bool
}

func canonicalInlineFrontierPayloadSpaceFor(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	candidate canonicalInlineFrontierCandidate,
) (canonicalInlineFrontierPayloadSpace, bool) {
	markerNodes, strikeNodes := canonicalInlineDelimiterPlanNodes(ast)
	textOrdinals, textCount := canonicalInlineTextOrdinals(ast)
	if len(candidate.payloads) != textCount ||
		len(candidate.payloadKnown) != textCount ||
		len(candidate.payloadRawTab) != textCount ||
		len(candidate.markers) != len(markerNodes) ||
		len(candidate.strikeWidths) != len(strikeNodes) {
		return canonicalInlineFrontierPayloadSpace{}, false
	}
	space := canonicalInlineFrontierPayloadSpace{
		markerNodes:      markerNodes,
		strikeNodes:      strikeNodes,
		textOrdinals:     textOrdinals,
		choicesByOrdinal: make([][]canonicalInlinePayloadChoice, textCount),
		knownValid:       true,
	}
	available := make([]bool, textCount)
	for _, node := range canonicalInlinePayloadVariableNodes(ast, normalization) {
		ordinal := textOrdinals[node]
		if ordinal < 0 {
			return canonicalInlineFrontierPayloadSpace{}, false
		}
		available[ordinal] = true
		space.choicesByOrdinal[ordinal] = canonicalInlinePayloadChoices(normalization.nodes[node])
		if !candidate.payloadKnown[ordinal] {
			space.unknown = append(space.unknown, ordinal)
		}
	}
	for ordinal, known := range candidate.payloadKnown {
		if known && (!available[ordinal] ||
			!canonicalInlinePayloadChoiceAvailable(candidate.payloads[ordinal], space.choicesByOrdinal[ordinal])) {
			space.knownValid = false
			break
		}
	}
	return space, true
}

func canonicalInlineFrontierPayloadVariants(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	candidate canonicalInlineFrontierCandidate,
) ([]canonicalInlineFrontierCandidate, []int, []int, []int, bool) {
	space, ok := canonicalInlineFrontierPayloadSpaceFor(ast, normalization, candidate)
	if !ok {
		return nil, nil, nil, nil, false
	}
	if !space.knownValid {
		return nil, space.markerNodes, space.strikeNodes, space.textOrdinals, true
	}
	variants := []canonicalInlineFrontierCandidate{candidate.clone()}
	for _, ordinal := range space.unknown {
		next := make([]canonicalInlineFrontierCandidate, 0, len(variants)*len(space.choicesByOrdinal[ordinal]))
		for _, partial := range variants {
			for _, choice := range space.choicesByOrdinal[ordinal] {
				variant := partial.clone()
				variant.payloadKnown[ordinal] = true
				variant.payloads[ordinal] = choice
				next = append(next, variant)
			}
		}
		variants = next
	}
	return variants, space.markerNodes, space.strikeNodes, space.textOrdinals, true
}

func canonicalInlineEvaluateFrontierCandidate(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	candidate canonicalInlineFrontierCandidate,
	needsTilde bool,
	context canonicalInlineEmitContext,
	tableCell bool,
	topology *native.DelimiterTopologyCandidate,
) ([]canonicalInlineFrontierCandidate, error) {
	variants, markerNodes, strikeNodes, textOrdinals, ok :=
		canonicalInlineFrontierPayloadVariants(ast, normalization, candidate)
	if !ok {
		return nil, ErrInvalidInput
	}
	result := make([]canonicalInlineFrontierCandidate, 0, len(variants))
	for _, variant := range variants {
		plan, ok := canonicalInlinePlanForFrontierCandidate(
			ast, variant, textOrdinals, markerNodes, strikeNodes,
		)
		if !ok {
			return nil, ErrInvalidInput
		}
		proof, err := canonicalInlineCandidateForPlan(
			ast,
			normalization,
			plan,
			context,
			tableCell,
		)
		if err != nil {
			return nil, err
		}
		if !topology.Reset(proof.output, proof.owners, proof.pairs) {
			continue
		}
		structuralValid := topology.Matches()
		deferred := false
		if !structuralValid {
			deferred = canonicalInlineDeferredRecoverable(proof.output, proof.owners, proof.pairs)
			if !deferred {
				continue
			}
		}
		variant.alternationCost = proof.alternationCost
		if len(markerNodes)+len(strikeNodes) == 0 {
			result = append(result, variant)
			continue
		}
		state, ok := canonicalInlineSelectorStateFromProof(
			proof,
			topology,
			len(strikeNodes) != 0 || needsTilde,
			deferred,
		)
		if !ok {
			return nil, ErrInvalidInput
		}
		variant.state = state
		result = append(result, variant)
	}
	return result, nil
}

func canonicalInlinePendingRawTabNode(
	ast canonicalInlineAST,
	candidate canonicalInlineFrontierCandidate,
) ([]int, int, bool) {
	if len(candidate.payloadRawTab) == 0 ||
		len(candidate.payloadKnown) != len(candidate.payloadRawTab) ||
		len(candidate.payloads) != len(candidate.payloadRawTab) {
		return nil, noCanonicalInlineASTNode, false
	}
	ordinal := len(candidate.payloadRawTab) - 1
	if !candidate.payloadRawTab[ordinal] || candidate.payloadKnown[ordinal] {
		return nil, noCanonicalInlineASTNode, false
	}
	textOrdinals, textCount := canonicalInlineTextOrdinals(ast)
	if textCount != len(candidate.payloads) {
		return nil, noCanonicalInlineASTNode, false
	}
	for nodeIndex, textOrdinal := range textOrdinals {
		if textOrdinal == ordinal {
			return textOrdinals, nodeIndex, true
		}
	}
	return nil, noCanonicalInlineASTNode, false
}

func canonicalInlinePendingRawTabProof(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	candidate canonicalInlineFrontierCandidate,
	textOrdinals []int,
	rawNode int,
	context canonicalInlineEmitContext,
	tableCell bool,
) (canonicalInlineCandidate, error) {
	pendingNormalization := canonicalInlineNormalization{
		nodes: append([]canonicalInlineNodeNormalization(nil), normalization.nodes...),
	}
	pendingNormalization.nodes[rawNode].trailingAlternative = canonicalInlineRawTabBoundary
	markerNodes, strikeNodes := canonicalInlineDelimiterPlanNodes(ast)
	plan, ok := canonicalInlinePlanForFrontierCandidate(
		ast, candidate, textOrdinals, markerNodes, strikeNodes,
	)
	if !ok {
		return canonicalInlineCandidate{}, ErrInvalidInput
	}
	plan.nodes[rawNode].payload = canonicalInlineRawTabPayload
	return canonicalInlineCandidateForPlan(
		ast,
		pendingNormalization,
		plan,
		context,
		tableCell,
	)
}

func canonicalInlinePendingRawTabResponse(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	candidate canonicalInlineFrontierCandidate,
	context canonicalInlineEmitContext,
	tableCell bool,
	topology *native.DelimiterTopologyCandidate,
) (canonicalInlineSelectorState, bool, error) {
	textOrdinals, rawNode, ok := canonicalInlinePendingRawTabNode(ast, candidate)
	if !ok {
		return canonicalInlineSelectorState{}, false, nil
	}
	proof, err := canonicalInlinePendingRawTabProof(
		ast, normalization, candidate, textOrdinals, rawNode, context, tableCell,
	)
	if err != nil {
		return canonicalInlineSelectorState{}, false, err
	}
	if !topology.Reset(proof.output, proof.owners, proof.pairs) {
		return canonicalInlineSelectorState{}, false, nil
	}
	if topology.Matches() {
		state, ok := canonicalInlineSelectorStateFromProof(proof, topology, true, false)
		if !ok {
			return canonicalInlineSelectorState{}, false, ErrInvalidInput
		}
		return state, true, nil
	}
	if !canonicalInlineDeferredRecoverable(proof.output, proof.owners, proof.pairs) {
		return canonicalInlineSelectorState{}, false, nil
	}
	state, ok := canonicalInlineSelectorStateFromProof(proof, topology, false, true)
	if !ok {
		return canonicalInlineSelectorState{}, false, ErrInvalidInput
	}
	return state, true, nil
}

type canonicalInlinePruneAccumulator struct {
	markerless    canonicalInlineFrontierCandidate
	markerlessSet bool
	deferred      []canonicalInlineFrontierCandidate
	keyed         []canonicalInlineFrontierCandidate
	byKey         map[canonicalInlinePruneKey]int
}

func (a *canonicalInlinePruneAccumulator) observe(
	candidate canonicalInlineFrontierCandidate,
	disposition canonicalInlinePruneDisposition,
	key canonicalInlinePruneKey,
) {
	switch disposition {
	case canonicalInlinePruneMarkerless:
		if !a.markerlessSet || canonicalInlineChoiceBetter(candidate.choiceCost(), a.markerless.choiceCost()) {
			a.markerless = candidate
			a.markerlessSet = true
		}
	case canonicalInlinePruneDeferred:
		a.deferred = append(a.deferred, candidate)
	case canonicalInlinePruneKeyed:
		if a.byKey == nil {
			a.byKey = make(map[canonicalInlinePruneKey]int)
		}
		if index, exists := a.byKey[key]; exists {
			if canonicalInlineChoiceBetter(candidate.choiceCost(), a.keyed[index].choiceCost()) {
				a.keyed[index] = candidate
			}
			return
		}
		a.byKey[key] = len(a.keyed)
		a.keyed = append(a.keyed, candidate)
	}
}

// result consumes the accumulator; a single owned result list needs no copy.
func (a canonicalInlinePruneAccumulator) result() []canonicalInlineFrontierCandidate {
	count := len(a.deferred) + len(a.keyed)
	if a.markerlessSet {
		count++
	}
	if count != 0 && count == len(a.keyed) {
		return a.keyed
	}
	if count != 0 && count == len(a.deferred) {
		return a.deferred
	}
	result := make([]canonicalInlineFrontierCandidate, 0, count)
	if a.markerlessSet {
		result = append(result, a.markerless)
	}
	result = append(result, a.deferred...)
	result = append(result, a.keyed...)
	return result
}

func canonicalInlinePruneFrontier(
	ast canonicalInlineAST,
	candidates []canonicalInlineFrontierCandidate,
	needsTilde bool,
	context canonicalInlineEmitContext,
	tableCell bool,
	topology *native.DelimiterTopologyCandidate,
) ([]canonicalInlineFrontierCandidate, error) {
	if len(candidates) == 0 {
		return nil, nil
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		return nil, err
	}
	selected := canonicalInlinePruneAccumulator{}
	for _, candidate := range candidates {
		emittedVariants, err := canonicalInlineEvaluateFrontierCandidate(
			ast, normalization, candidate, needsTilde, context, tableCell, topology,
		)
		if err != nil {
			return nil, err
		}
		if len(emittedVariants) == 0 {
			pendingState, pending, err := canonicalInlinePendingRawTabResponse(
				ast, normalization, candidate, context, tableCell, topology,
			)
			if err != nil {
				return nil, err
			}
			if key, keyed := canonicalInlinePendingOnlyPruneKey(pendingState, pending); keyed {
				selected.observe(candidate, canonicalInlinePruneKeyed, key)
			}
			continue
		}
		for _, emitted := range emittedVariants {
			disposition := canonicalInlinePruneDispositionForEmitted(
				len(emitted.markers)+len(emitted.strikeWidths) != 0,
				emitted.state,
			)
			key := canonicalInlinePruneKey{}
			if disposition == canonicalInlinePruneKeyed {
				pendingState, pending, err := canonicalInlinePendingRawTabResponse(
					ast, normalization, emitted, context, tableCell, topology,
				)
				if err != nil {
					return nil, err
				}
				key = canonicalInlinePruneKeyWithPending(emitted.state, pendingState, pending)
			}
			selected.observe(emitted, disposition, key)
		}
	}
	return selected.result(), nil
}

func canonicalInlineBottomUpHostInContext(
	ast canonicalInlineAST,
	context canonicalInlineEmitContext,
	tableCell bool,
) ([]canonicalInlineFrontierCandidate, bool, error) {
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		return nil, false, err
	}
	_, strikeNodes := canonicalInlineDelimiterPlanNodes(ast)
	// Proof results retained by the frontier are values, so sibling and parent
	// candidates can share resolver scratch during this serial host operation.
	var topology native.DelimiterTopologyCandidate
	prune := func(
		view canonicalInlineAST,
		candidates []canonicalInlineFrontierCandidate,
		needsTilde bool,
	) ([]canonicalInlineFrontierCandidate, error) {
		return canonicalInlinePruneFrontier(
			view, candidates, needsTilde, context, tableCell, &topology,
		)
	}
	return canonicalInlineBottomUpHostWithNormalization(
		ast,
		normalization,
		len(strikeNodes) != 0,
		prune,
	)
}

func canonicalInlineBottomUpHost(
	ast canonicalInlineAST,
) ([]canonicalInlineFrontierCandidate, bool, error) {
	return canonicalInlineBottomUpHostInContext(
		ast,
		canonicalInlineEmitContext{referenceLabels: native.New()},
		false,
	)
}

func canonicalInlineFinalHostProof(
	ast canonicalInlineAST,
	normalization canonicalInlineNormalization,
	candidate canonicalInlineFrontierCandidate,
	context canonicalInlineEmitContext,
	tableCell bool,
) (canonicalInlineCandidate, bool, error) {
	if candidate.state.deferredContinuation {
		return canonicalInlineCandidate{}, false, nil
	}
	textOrdinals, _ := canonicalInlineTextOrdinals(ast)
	markerNodes, strikeNodes := canonicalInlineDelimiterPlanNodes(ast)
	plan, ok := canonicalInlinePlanForFrontierCandidate(
		ast, candidate, textOrdinals, markerNodes, strikeNodes,
	)
	if !ok {
		return canonicalInlineCandidate{}, false, ErrInvalidInput
	}
	proof, err := canonicalInlineCandidateForPlan(
		ast, normalization, plan, context, tableCell,
	)
	if err != nil {
		return canonicalInlineCandidate{}, false, err
	}
	if proof.alternationCost != candidate.alternationCost {
		return canonicalInlineCandidate{}, false, ErrInvalidInput
	}
	if !native.DelimiterTopologyMatches(proof.output, proof.owners, proof.pairs) ||
		!native.ExtendedAutolinkOwnersMatch(proof.output, proof.extendedAutolinks) ||
		!proof.bareWWW.semanticOwnerValid() {
		return canonicalInlineCandidate{}, false, nil
	}
	return proof, true, nil
}

func canonicalInlineRenderHost(
	ast canonicalInlineAST,
	context canonicalInlineEmitContext,
	tableCell bool,
) ([]byte, bool, error) {
	frontier, supported, err := canonicalInlineBottomUpHostInContext(
		ast, context, tableCell,
	)
	if err != nil || !supported {
		return nil, false, err
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		return nil, false, err
	}
	var best canonicalInlineFrontierCandidate
	var bestProof canonicalInlineCandidate
	found := false
	for _, candidate := range frontier {
		proof, valid, err := canonicalInlineFinalHostProof(
			ast, normalization, candidate, context, tableCell,
		)
		if err != nil {
			return nil, false, err
		}
		if !valid {
			continue
		}
		if !found || canonicalInlineChoiceBetter(candidate.choiceCost(), best.choiceCost()) {
			best = candidate
			bestProof = proof
			found = true
		}
	}
	if !found {
		return nil, false, nil
	}
	return bestProof.output, true, nil
}
