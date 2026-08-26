package splice

import "bytes"

// PrepareReplaceMathExpression prepares a source-preserving replacement of one
// reviewed mathematical payload while retaining its exact delimiter/container form.
func (d *Document) PrepareReplaceMathExpression(id NodeID, replacement []byte) (ChangeSet, error) {
	if d == nil {
		return ChangeSet{}, ErrNodeNotFound
	}
	target, ok := d.nodeByID(id)
	if !ok {
		return ChangeSet{}, ErrNodeNotFound
	}
	if isFencedMathNode(target) {
		return d.PrepareReplaceFencedCode(id, replacement)
	}
	if target.Kind != KindMathExpression || !target.Editable {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	if err := validateMathReplacement(target.MathSource.Style, replacement); err != nil {
		return ChangeSet{}, err
	}

	change, candidate, err := d.prepareCandidateChange(target.MathSource.PayloadRange, replacement, "math expression replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateMathExpressionReplacement(candidate, target, len(replacement)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func validateMathReplacement(style MathExpressionStyle, replacement []byte) error {
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return err
	}
	if bytes.IndexByte(replacement, 0) >= 0 || hasUnescapedMathReplacementDollar(replacement) {
		return ErrInvalidReplacement
	}
	if style == MathExpressionInlineBacktick && bytes.IndexByte(replacement, '`') >= 0 {
		return ErrInvalidReplacement
	}
	return nil
}

func hasUnescapedMathReplacementDollar(value []byte) bool {
	for index, current := range value {
		if current != '$' {
			continue
		}
		backslashes := 0
		for position := index - 1; position >= 0 && value[position] == '\\'; position-- {
			backslashes++
		}
		if backslashes%2 == 0 {
			return true
		}
	}
	return false
}

func validateMathExpressionReplacement(candidate []byte, target Node, replacementLength int) error {
	parsed, err := Parse(candidate)
	if err != nil {
		return ErrInvalidReplacement
	}
	delta := replacementLength - (target.MathSource.PayloadRange.End - target.MathSource.PayloadRange.Start)
	wantRange := shiftedEnd(target.MathSource.Range, delta)
	wantPayload := rangeWithLength(target.MathSource.PayloadRange.Start, replacementLength)
	for _, node := range parsed.nodes {
		if node.Kind != KindMathExpression || node.Anchor != target.Anchor || node.TopLevel != target.TopLevel ||
			node.MathSource.Style != target.MathSource.Style || node.MathSource.Range != wantRange || node.MathSource.PayloadRange != wantPayload {
			continue
		}
		return nil
	}
	return ErrInvalidReplacement
}
