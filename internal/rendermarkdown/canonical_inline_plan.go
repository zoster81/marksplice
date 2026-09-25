package rendermarkdown

import "github.com/zoster81/marksplice/internal/parser"

type canonicalInlinePayloadChoice uint8

const (
	canonicalInlinePreferredPayload canonicalInlinePayloadChoice = iota
	canonicalInlineRawTabPayload
)

type canonicalInlineNodePlan struct {
	marker             byte
	payload            canonicalInlinePayloadChoice
	strikethroughWidth uint8
}

type canonicalInlinePlan struct {
	nodes []canonicalInlineNodePlan
}

type canonicalInlinePlanCandidate struct {
	markers          []byte
	strikeWidths     []uint8
	payloads         []canonicalInlinePayloadChoice
	payloadKnown     []bool
	payloadRawTab    []bool
	alternationCost  int
	strikeSingleCost int
}

func (candidate canonicalInlinePlanCandidate) clone() canonicalInlinePlanCandidate {
	return canonicalInlinePlanCandidate{
		markers:          append([]byte(nil), candidate.markers...),
		strikeWidths:     append([]uint8(nil), candidate.strikeWidths...),
		payloads:         append([]canonicalInlinePayloadChoice(nil), candidate.payloads...),
		payloadKnown:     append([]bool(nil), candidate.payloadKnown...),
		payloadRawTab:    append([]bool(nil), candidate.payloadRawTab...),
		alternationCost:  candidate.alternationCost,
		strikeSingleCost: candidate.strikeSingleCost,
	}
}

func appendCanonicalInlinePlanCandidates(
	left, right canonicalInlinePlanCandidate,
) canonicalInlinePlanCandidate {
	result := canonicalInlinePlanCandidate{
		markers:          make([]byte, 0, len(left.markers)+len(right.markers)),
		strikeWidths:     make([]uint8, 0, len(left.strikeWidths)+len(right.strikeWidths)),
		payloads:         make([]canonicalInlinePayloadChoice, 0, len(left.payloads)+len(right.payloads)),
		payloadKnown:     make([]bool, 0, len(left.payloadKnown)+len(right.payloadKnown)),
		payloadRawTab:    make([]bool, 0, len(left.payloadRawTab)+len(right.payloadRawTab)),
		alternationCost:  left.alternationCost + right.alternationCost,
		strikeSingleCost: left.strikeSingleCost + right.strikeSingleCost,
	}
	result.markers = append(result.markers, left.markers...)
	result.markers = append(result.markers, right.markers...)
	result.strikeWidths = append(result.strikeWidths, left.strikeWidths...)
	result.strikeWidths = append(result.strikeWidths, right.strikeWidths...)
	result.payloads = append(result.payloads, left.payloads...)
	result.payloads = append(result.payloads, right.payloads...)
	result.payloadKnown = append(result.payloadKnown, left.payloadKnown...)
	result.payloadKnown = append(result.payloadKnown, right.payloadKnown...)
	result.payloadRawTab = append(result.payloadRawTab, left.payloadRawTab...)
	result.payloadRawTab = append(result.payloadRawTab, right.payloadRawTab...)
	return result
}

func (candidate canonicalInlinePlanCandidate) choiceCost() canonicalInlineChoiceCost {
	return canonicalInlineChoiceCost{
		alternation:  candidate.alternationCost,
		strikeSingle: candidate.strikeSingleCost,
		payloads:     candidate.payloads,
		markers:      candidate.markers,
	}
}

func newCanonicalInlinePlan(ast canonicalInlineAST) canonicalInlinePlan {
	plan := canonicalInlinePlan{
		nodes: make([]canonicalInlineNodePlan, len(ast.nodes)),
	}
	for nodeIndex := 1; nodeIndex < len(ast.nodes); nodeIndex++ {
		event := ast.events[ast.nodes[nodeIndex].eventIndex]
		if event.Phase == parser.SemanticEnter && event.Kind == parser.SemanticStrikethrough {
			plan.nodes[nodeIndex].strikethroughWidth = 2
		}
	}
	return plan
}
