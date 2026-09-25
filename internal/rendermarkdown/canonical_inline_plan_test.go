package rendermarkdown

import "testing"

func TestCanonicalInlinePlanCandidateCloneAppendAndCost(t *testing.T) {
	t.Parallel()

	left := canonicalInlinePlanCandidate{
		markers:          []byte{'*'},
		strikeWidths:     []uint8{2},
		payloads:         []canonicalInlinePayloadChoice{canonicalInlinePreferredPayload},
		payloadKnown:     []bool{true},
		payloadRawTab:    []bool{false},
		alternationCost:  1,
		strikeSingleCost: 0,
	}
	right := canonicalInlinePlanCandidate{
		markers:          []byte{'_'},
		strikeWidths:     []uint8{1},
		payloads:         []canonicalInlinePayloadChoice{canonicalInlineRawTabPayload},
		payloadKnown:     []bool{false},
		payloadRawTab:    []bool{true},
		alternationCost:  2,
		strikeSingleCost: 1,
	}

	combined := appendCanonicalInlinePlanCandidates(left, right)
	if got := string(combined.markers); got != "*_" {
		t.Fatalf("markers = %q, want %q", got, "*_")
	}
	if len(combined.strikeWidths) != 2 || combined.strikeWidths[0] != 2 || combined.strikeWidths[1] != 1 {
		t.Fatalf("strike widths = %v, want [2 1]", combined.strikeWidths)
	}
	if len(combined.payloads) != 2 ||
		combined.payloads[0] != canonicalInlinePreferredPayload ||
		combined.payloads[1] != canonicalInlineRawTabPayload {
		t.Fatalf("payloads = %v", combined.payloads)
	}
	if len(combined.payloadKnown) != 2 || !combined.payloadKnown[0] || combined.payloadKnown[1] {
		t.Fatalf("payload known = %v", combined.payloadKnown)
	}
	if len(combined.payloadRawTab) != 2 || combined.payloadRawTab[0] || !combined.payloadRawTab[1] {
		t.Fatalf("payload raw TAB = %v", combined.payloadRawTab)
	}
	if combined.alternationCost != 3 || combined.strikeSingleCost != 1 {
		t.Fatalf(
			"costs = alternation %d strike-single %d, want 3/1",
			combined.alternationCost,
			combined.strikeSingleCost,
		)
	}

	cost := combined.choiceCost()
	if cost.alternation != 3 || cost.strikeSingle != 1 ||
		string(cost.markers) != "*_" || len(cost.payloads) != 2 {
		t.Fatalf("choice cost = %+v", cost)
	}

	clone := combined.clone()
	clone.markers[0] = '_'
	clone.strikeWidths[0] = 1
	clone.payloads[0] = canonicalInlineRawTabPayload
	clone.payloadKnown[0] = false
	clone.payloadRawTab[0] = true
	if combined.markers[0] != '*' ||
		combined.strikeWidths[0] != 2 ||
		combined.payloads[0] != canonicalInlinePreferredPayload ||
		!combined.payloadKnown[0] ||
		combined.payloadRawTab[0] {
		t.Fatal("clone aliases candidate slices")
	}
}
