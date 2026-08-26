package differential

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/zoster81/marksplice/internal/parser"
)

// InlineParser is the M113 native inline-parser entrypoint shape. It remains
// separate from parser.Backend until the native parser satisfies the complete
// backend contract and later cutover gates.
type InlineParser func([]byte) ([]parser.Node, error)

// InlineObservationParser is the M113 relationship-observation entrypoint shape.
// It remains separate from parser.Backend until the native parser satisfies the
// complete backend contract and later cutover gates.
type InlineObservationParser func([]byte) (parser.DocumentObservations, error)

// InlineHarness compares only parser observations owned by the M113 inline
// parser while Goldmark remains the temporary differential oracle.
type InlineHarness struct {
	Oracle    parser.Backend
	Candidate InlineParser
}

// InlineRelationshipHarness compares the relationship observations owned by the
// M113 inline parser without claiming parity for unrelated document families.
type InlineRelationshipHarness struct {
	Oracle    parser.Backend
	Candidate InlineObservationParser
}

// CompareKinds compares only the selected M113 inline observation families.
// Family-scoped parity lets the native grammar advance slice by slice without
// claiming that later inline families are already implemented.
func (h InlineHarness) CompareKinds(source []byte, kinds ...parser.Kind) error {
	if h.Oracle == nil || h.Candidate == nil {
		return fmt.Errorf("parser differential: inline oracle and candidate are required")
	}
	oracleSource := bytes.Clone(source)
	candidateSource := bytes.Clone(source)
	observed, oracleErr := h.Oracle.ParseDocument(oracleSource)
	candidate, candidateErr := h.Candidate(candidateSource)
	if err := compareSourceMutation("inline parse", source, oracleSource, candidateSource); err != nil {
		return err
	}
	if err := compareErrorOutcome("inline parse", oracleErr, candidateErr); err != nil {
		return err
	}
	if oracleErr != nil {
		return nil
	}
	return compareInlineNodeSlices(filterInlineKinds(projectInlineNodes(observed.Nodes), kinds), filterInlineKinds(projectInlineNodes(candidate), kinds))
}

// CompareRelationships requires exact semantic-source-order parity for resolved
// link/image/autolink usages and conservative unresolved explicit references.
func (h InlineRelationshipHarness) CompareRelationships(source []byte) error {
	if h.Oracle == nil || h.Candidate == nil {
		return fmt.Errorf("parser differential: inline relationship oracle and candidate are required")
	}
	oracleSource := bytes.Clone(source)
	candidateSource := bytes.Clone(source)
	oracle, oracleErr := h.Oracle.ParseDocument(oracleSource)
	candidate, candidateErr := h.Candidate(candidateSource)
	if err := compareSourceMutation("inline relationship parse", source, oracleSource, candidateSource); err != nil {
		return err
	}
	if err := compareErrorOutcome("inline relationship parse", oracleErr, candidateErr); err != nil {
		return err
	}
	if oracleErr != nil {
		return nil
	}
	if !slices.Equal(oracle.LinkUsages, candidate.LinkUsages) {
		return fmt.Errorf("parser differential: inline LinkUsages differ: oracle=%+v candidate=%+v", oracle.LinkUsages, candidate.LinkUsages)
	}
	if !slices.Equal(oracle.UnresolvedReferenceUsages, candidate.UnresolvedReferenceUsages) {
		return fmt.Errorf("parser differential: inline UnresolvedReferenceUsages differ: oracle=%+v candidate=%+v", oracle.UnresolvedReferenceUsages, candidate.UnresolvedReferenceUsages)
	}
	return nil
}

func compareInlineNodeSlices(oracle, candidate []parser.Node) error {
	if len(oracle) != len(candidate) {
		return fmt.Errorf("parser differential: inline node length differs: oracle=%d candidate=%d oracleNodes=%+v candidateNodes=%+v", len(oracle), len(candidate), oracle, candidate)
	}
	for index := range oracle {
		if !sameNode(oracle[index], candidate[index]) {
			return fmt.Errorf("parser differential: inline node[%d] differs: oracle=%+v candidate=%+v", index, oracle[index], candidate[index])
		}
	}
	return nil
}

func filterInlineKinds(nodes []parser.Node, kinds []parser.Kind) []parser.Node {
	filtered := make([]parser.Node, 0, len(nodes))
	for _, node := range nodes {
		for _, kind := range kinds {
			if node.Kind == kind {
				filtered = append(filtered, node)
				break
			}
		}
	}
	return filtered
}

func projectInlineNodes(nodes []parser.Node) []parser.Node {
	projected := make([]parser.Node, 0, len(nodes))
	for _, node := range nodes {
		if m113InlineKind(node.Kind) {
			projected = append(projected, node)
		}
	}
	return projected
}

func m113InlineKind(kind parser.Kind) bool {
	switch kind {
	case parser.KindStrikethrough,
		parser.KindInlineLink,
		parser.KindAutoLink,
		parser.KindCodeSpan,
		parser.KindEmphasis,
		parser.KindStrong,
		parser.KindRawHTML,
		parser.KindImage:
		return true
	default:
		return false
	}
}
