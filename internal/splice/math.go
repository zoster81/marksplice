package splice

import (
	"errors"
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

// MathExpressionStyle identifies one reviewed GitHub-compatible mathematical source form.
type MathExpressionStyle = source.MathExpressionStyle

const (
	MathExpressionUnknown        = source.MathExpressionUnknown
	MathExpressionInlineDollar   = source.MathExpressionInlineDollar
	MathExpressionInlineBacktick = source.MathExpressionInlineBacktick
	MathExpressionBlockDollar    = source.MathExpressionBlockDollar
)

func mathExpressionStyleFromParser(style parser.MathExpressionStyle) (source.MathExpressionStyle, bool) {
	switch style {
	case parser.MathExpressionInlineDollar:
		return source.MathExpressionInlineDollar, true
	case parser.MathExpressionInlineBacktick:
		return source.MathExpressionInlineBacktick, true
	case parser.MathExpressionBlockDollar:
		return source.MathExpressionBlockDollar, true
	default:
		return source.MathExpressionUnknown, false
	}
}

func mathExpressionsOutsideRange(expressions []parser.MathExpressionObservation, excluded Range) []parser.MathExpressionObservation {
	result := make([]parser.MathExpressionObservation, 0, len(expressions))
	for _, expression := range expressions {
		if expression.Range.Start >= excluded.Start && expression.Range.Start < excluded.End {
			continue
		}
		result = append(result, expression)
	}
	return result
}

func promoteMathExpressionNodes(snapshot []byte, fingerprint source.Fingerprint, observations []parser.MathExpressionObservation) ([]Node, error) {
	result := make([]Node, 0, len(observations))
	for _, observation := range observations {
		node, ok, err := mathExpressionNode(snapshot, fingerprint, observation)
		if err != nil {
			return nil, err
		}
		if ok {
			result = append(result, node)
		}
	}
	return result, nil
}

func mathExpressionNode(snapshot []byte, fingerprint source.Fingerprint, observation parser.MathExpressionObservation) (Node, bool, error) {
	style, ok := mathExpressionStyleFromParser(observation.Style)
	if !ok {
		return Node{}, false, nil
	}
	mapping, err := source.MapMathExpression(snapshot, style,
		Range{Start: observation.Range.Start, End: observation.Range.End},
		Range{Start: observation.PayloadRange.Start, End: observation.PayloadRange.End})
	if err != nil {
		if errors.Is(err, source.ErrUnsupportedMathExpressionShape) {
			return Node{}, false, nil
		}
		return Node{}, false, fmt.Errorf("map math expression source: %w", err)
	}
	node := Node{
		Kind:         KindMathExpression,
		Range:        mapping.Range,
		ContentRange: mapping.PayloadRange,
		MathStyle:    style,
		Anchor:       observation.Range.Start,
		TopLevel:     observation.TopLevel,
		Editable:     true,
	}
	node.ID = makeNodeID(fingerprint, node.Kind, node.Range)
	return node, true, nil
}

func (d *Document) isFencedMathNode(node Node) bool {
	detail, ok := d.fencedSource(node)
	return ok && node.Kind == KindFencedCode && node.TopLevel && detail.block.OpeningFenceLength >= 3 && detail.info == "math"
}

// MathExpressionPayloadRanges returns caller-owned payload source ranges for one
// dollar/backtick math expression or one exact-info `math` fenced block.
func (d *Document) MathExpressionPayloadRanges(id NodeID) ([]Range, bool) {
	if d == nil {
		return nil, false
	}
	node, ok := d.nodeByID(id)
	if !ok {
		return nil, false
	}
	if node.Kind == KindMathExpression && node.Editable {
		return []Range{node.ContentRange}, true
	}
	if d.isFencedMathNode(node) {
		detail, _ := d.fencedSource(node)
		return append([]Range(nil), detail.block.ContentRanges...), true
	}
	return nil, false
}
