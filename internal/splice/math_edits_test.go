package splice

import (
	"bytes"
	"errors"
	"testing"
)

func TestPrepareReplaceMathExpressionNoOpPreservesSourceProvenNULPayload(t *testing.T) {
	t.Parallel()

	source := []byte{'$', 0, '$'}
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	nodes := nodesOfKind(doc.Nodes(), KindMathExpression)
	if len(nodes) != 1 {
		t.Fatalf("math node count = %d, want 1", len(nodes))
	}
	target := nodes[0]
	if target.ContentRange != (Range{Start: 1, End: 2}) {
		t.Fatalf("math payload range = %v, want [1,2)", target.ContentRange)
	}

	change, err := doc.PrepareReplaceMathExpression(target.ID, []byte{0})
	if err != nil {
		t.Fatalf("PrepareReplaceMathExpression(no-op) error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply(no-op) error = %v", err)
	}
	if !bytes.Equal(got, source) {
		t.Fatalf("no-op changed source: %q", got)
	}
	stale := append([]byte(nil), source...)
	stale[1] = 'x'
	if _, err := change.Apply(stale); !errors.Is(err, ErrSourceConflict) {
		t.Fatalf("Apply(stale no-op) error = %v, want ErrSourceConflict", err)
	}
}

func TestPrepareReplaceMathExpressionStillRejectsNewNULPayload(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("$x$"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	nodes := nodesOfKind(doc.Nodes(), KindMathExpression)
	if len(nodes) != 1 {
		t.Fatalf("math node count = %d, want 1", len(nodes))
	}
	if _, err := doc.PrepareReplaceMathExpression(nodes[0].ID, []byte{0}); !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceMathExpression(new NUL) error = %v, want ErrInvalidReplacement", err)
	}
}
