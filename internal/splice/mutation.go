package splice

import (
	"bytes"
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
	goldmarkparser "github.com/zoster81/marksplice/internal/parser/goldmark"
	"github.com/zoster81/marksplice/internal/source"
)

func (d *Document) targetNode(id NodeID, expected Kind) (Node, error) {
	target, ok := d.nodeByID(id)
	if !ok {
		return Node{}, ErrNodeNotFound
	}
	if target.Kind != expected {
		return Node{}, ErrInvalidTargetKind
	}
	return target, nil
}

func (d *Document) editableTargetNode(id NodeID, expected Kind, description string) (Node, error) {
	target, err := d.targetNode(id, expected)
	if err != nil {
		return Node{}, err
	}
	if !target.Editable {
		return Node{}, fmt.Errorf("%w: %s source shape is not editable", ErrInvalidTargetKind, description)
	}
	return target, nil
}

func validateNonEmptySingleLine(replacement []byte) error {
	if len(replacement) == 0 || bytes.ContainsAny(replacement, "\r\n") {
		return ErrInvalidReplacement
	}
	return nil
}

func (d *Document) newChange(range_ Range, replacement []byte, operation string) (ChangeSet, error) {
	change, err := source.NewChangeSet(d.source, []source.Patch{{Range: range_, Replacement: replacement}})
	if err != nil {
		return ChangeSet{}, fmt.Errorf("prepare %s: %w", operation, err)
	}
	return change, nil
}

func (d *Document) prepareCandidateChange(range_ Range, replacement []byte, operation string) (ChangeSet, []byte, error) {
	change, err := d.newChange(range_, replacement, operation)
	if err != nil {
		return ChangeSet{}, nil, err
	}
	candidate, err := change.Apply(d.source)
	if err != nil {
		return ChangeSet{}, nil, fmt.Errorf("render %s candidate: %w", operation, err)
	}
	return change, candidate, nil
}

func parseCandidate(candidate []byte) ([]parser.Node, error) {
	observations, err := goldmarkparser.New().Parse(candidate)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidReplacement, err)
	}
	return observations, nil
}

func shiftedEnd(range_ Range, delta int) Range {
	return Range{Start: range_.Start, End: range_.End + delta}
}

func rangeWithLength(start, length int) Range {
	return Range{Start: start, End: start + length}
}
