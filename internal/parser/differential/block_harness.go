package differential

import (
	"bytes"
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
)

// BlockParser is the M112 native block-parser entrypoint shape. It is kept
// separate from parser.Backend until native inline parsing completes the full
// backend contract.
type BlockParser func([]byte) ([]parser.Node, error)

// BlockHarness compares only observations owned by the M112 block parser.
type BlockHarness struct {
	Oracle    parser.Backend
	Candidate BlockParser
}

// Compare requires exact block-observation parity while deliberately excluding
// heading semantic text, which is inline-derived and belongs to M113.
func (h BlockHarness) Compare(source []byte) error {
	if h.Oracle == nil || h.Candidate == nil {
		return fmt.Errorf("parser differential: block oracle and candidate are required")
	}
	oracleSource := bytes.Clone(source)
	candidateSource := bytes.Clone(source)
	observed, oracleErr := h.Oracle.ParseDocument(oracleSource)
	candidate, candidateErr := h.Candidate(candidateSource)
	if err := compareSourceMutation("block parse", source, oracleSource, candidateSource); err != nil {
		return err
	}
	if err := compareErrorOutcome("block parse", oracleErr, candidateErr); err != nil {
		return err
	}
	if oracleErr != nil {
		return nil
	}
	return compareBlockNodeSlices(projectBlockNodes(observed.Nodes), projectBlockNodes(candidate))
}

// CompareKinds compares only the selected M112 block observation families. It
// is used to close one grammar family even when a normative example also owns
// container observations that are implemented by a later M112 slice.
func (h BlockHarness) CompareKinds(source []byte, kinds ...parser.Kind) error {
	if h.Oracle == nil || h.Candidate == nil {
		return fmt.Errorf("parser differential: block oracle and candidate are required")
	}
	oracleSource := bytes.Clone(source)
	candidateSource := bytes.Clone(source)
	observed, oracleErr := h.Oracle.ParseDocument(oracleSource)
	candidate, candidateErr := h.Candidate(candidateSource)
	if err := compareSourceMutation("block parse", source, oracleSource, candidateSource); err != nil {
		return err
	}
	if err := compareErrorOutcome("block parse", oracleErr, candidateErr); err != nil {
		return err
	}
	if oracleErr != nil {
		return nil
	}
	return compareBlockNodeSlices(filterBlockKinds(projectBlockNodes(observed.Nodes), kinds), filterBlockKinds(projectBlockNodes(candidate), kinds))
}

func compareBlockNodeSlices(oracle, candidate []parser.Node) error {
	if len(oracle) != len(candidate) {
		return fmt.Errorf("parser differential: block node length differs: oracle=%d candidate=%d", len(oracle), len(candidate))
	}
	for index := range oracle {
		if !sameNode(oracle[index], candidate[index]) {
			return fmt.Errorf("parser differential: block node[%d] differs: oracle=%+v candidate=%+v", index, oracle[index], candidate[index])
		}
	}
	return nil
}

func filterBlockKinds(nodes []parser.Node, kinds []parser.Kind) []parser.Node {
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

func projectBlockNodes(nodes []parser.Node) []parser.Node {
	projected := make([]parser.Node, 0, len(nodes))
	for _, node := range nodes {
		if !m112BlockKind(node.Kind) {
			continue
		}
		if node.Kind == parser.KindHeading {
			node.HeadingText = ""
		}
		projected = append(projected, node)
	}
	return projected
}

func m112BlockKind(kind parser.Kind) bool {
	switch kind {
	case parser.KindParagraph,
		parser.KindHeading,
		parser.KindTask,
		parser.KindListItem,
		parser.KindTableCell,
		parser.KindFencedCode,
		parser.KindReferenceDefinition,
		parser.KindHTMLBlock,
		parser.KindTableRow,
		parser.KindThematicBreak,
		parser.KindBlockquote,
		parser.KindTable:
		return true
	default:
		return false
	}
}
