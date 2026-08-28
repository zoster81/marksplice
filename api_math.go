package marksplice

import "github.com/zoster81/marksplice/internal/splice"

// MathExpressionStyle identifies one reviewed GitHub-compatible mathematical source form.
type MathExpressionStyle uint8

const (
	MathExpressionUnknown MathExpressionStyle = iota
	MathExpressionInlineDollar
	MathExpressionInlineBacktick
	MathExpressionBlockDollar
	MathExpressionFencedBlock
)

// MathExpression is immutable typed detail for one reviewed mathematical source form.
// Mathematical payload is opaque data; Marksplice does not parse or render LaTeX.
type MathExpression struct {
	id           NodeID
	style        MathExpressionStyle
	sourceRange  Range
	payloadRange Range
	hasPayload   bool
}

// ID returns the snapshot-scoped identity. Fenced math shares the underlying FencedBlock ID.
func (m MathExpression) ID() NodeID { return m.id }

// Style returns the exact reviewed source delimiter/container form.
func (m MathExpression) Style() MathExpressionStyle { return m.style }

// Range returns the complete source-owned mathematical syntax/container.
func (m MathExpression) Range() Range { return m.sourceRange }

// PayloadRange returns one exact contiguous payload span when available.
// Dollar/backtick forms always expose it; fenced math may be non-contiguous or empty.
func (m MathExpression) PayloadRange() (Range, bool) {
	if !m.hasPayload {
		return Range{}, false
	}
	return m.payloadRange, true
}

// MathExpressions returns reviewed mathematical expressions in source order.
// Exact-info `math` fenced blocks reuse their existing FencedBlock identity rather
// than creating a second structural node.
func (d *Document) MathExpressions() []MathExpression {
	if d == nil || d.document == nil {
		return nil
	}
	result := make([]MathExpression, 0)
	for index := 0; index < d.document.NodeCount(); index++ {
		summary, ok := d.document.NodeSummaryAt(index)
		if !ok {
			continue
		}
		node, ok := d.document.Node(summary.ID)
		if !ok {
			continue
		}
		expression, ok := publicMathExpression(d.document, node)
		if ok {
			result = append(result, expression)
		}
	}
	return result
}

// MathExpression returns one reviewed mathematical expression by snapshot ID.
func (d *Document) MathExpression(id NodeID) (MathExpression, bool) {
	node, ok := d.internalNode(id)
	if !ok {
		return MathExpression{}, false
	}
	return publicMathExpression(d.document, node)
}

// PrepareReplaceMathExpression prepares a source-preserving replacement of one
// reviewed mathematical payload while retaining its exact delimiter/container form.
func (d *Document) PrepareReplaceMathExpression(id NodeID, replacement []byte) (ChangeSet, error) {
	if d == nil || d.document == nil {
		return ChangeSet{}, ErrNodeNotFound
	}
	node, ok := d.internalNode(id)
	if !ok {
		return ChangeSet{}, ErrNodeNotFound
	}
	if _, ok := publicMathExpression(d.document, node); !ok {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	return publicChangeSet(d.document.PrepareReplaceMathExpression(internalNodeID(id), replacement))
}

// MathExpressionPayloadRanges returns caller-owned source-backed payload ranges.
// Fenced math may expose zero, one, or multiple physical payload ranges.
func (d *Document) MathExpressionPayloadRanges(id NodeID) ([]Range, bool) {
	if d == nil || d.document == nil {
		return nil, false
	}
	ranges, ok := d.document.MathExpressionPayloadRanges(internalNodeID(id))
	if !ok {
		return nil, false
	}
	return publicRanges(ranges), true
}

func publicMathExpression(document *splice.Document, node splice.Node) (MathExpression, bool) {
	if node.Kind == splice.KindMathExpression && node.Editable {
		style, ok := publicMathExpressionStyle(node.MathStyle)
		if !ok || node.Range.Start >= node.Range.End || node.ContentRange.Start >= node.ContentRange.End {
			return MathExpression{}, false
		}
		return MathExpression{
			id:           publicNodeID(node.ID),
			style:        style,
			sourceRange:  publicRange(node.Range),
			payloadRange: publicRange(node.ContentRange),
			hasPayload:   true,
		}, true
	}
	if document == nil || node.Kind != splice.KindFencedCode || !node.TopLevel {
		return MathExpression{}, false
	}
	block, info, _, ok := document.FencedBlockSource(node.ID)
	if !ok || info != "math" || block.Range.Start >= block.Range.End {
		return MathExpression{}, false
	}
	code, codeOK := document.FencedCodeSource(node.ID)
	payload := Range{}
	if codeOK {
		payload = publicRange(code.ContentRange)
	}
	return MathExpression{
		id:           publicNodeID(node.ID),
		style:        MathExpressionFencedBlock,
		sourceRange:  publicRange(block.Range),
		payloadRange: payload,
		hasPayload:   codeOK && payload.Start < payload.End,
	}, true
}

func publicMathExpressionStyle(style splice.MathExpressionStyle) (MathExpressionStyle, bool) {
	switch style {
	case splice.MathExpressionInlineDollar:
		return MathExpressionInlineDollar, true
	case splice.MathExpressionInlineBacktick:
		return MathExpressionInlineBacktick, true
	case splice.MathExpressionBlockDollar:
		return MathExpressionBlockDollar, true
	default:
		return MathExpressionUnknown, false
	}
}
