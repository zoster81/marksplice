package splice

import (
	"bytes"
	"errors"
	"testing"
)

func TestPrepareReplaceMathExpressionNormalizesSourceProvenNULPayload(t *testing.T) {
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
	if target.ContentRange != (Range{Start: 1, End: 4}) {
		t.Fatalf("math payload range = %v, want [1,4)", target.ContentRange)
	}

	change, err := doc.PrepareReplaceMathExpression(target.ID, []byte{0})
	if err != nil {
		t.Fatalf("PrepareReplaceMathExpression(normalized no-op) error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply(normalized no-op) error = %v", err)
	}
	want := []byte("$�$")
	if !bytes.Equal(got, want) {
		t.Fatalf("normalized no-op source = % x (%q), want % x (%q)", got, got, want, want)
	}
	stale := append([]byte(nil), source...)
	stale[1] = 'x'
	if _, err := change.Apply(stale); !errors.Is(err, ErrSourceConflict) {
		t.Fatalf("Apply(stale normalized no-op) error = %v, want ErrSourceConflict", err)
	}
}

func TestPrepareReplaceMathExpressionNormalizesNewNULPayload(t *testing.T) {
	t.Parallel()

	source := []byte("$x$")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	nodes := nodesOfKind(doc.Nodes(), KindMathExpression)
	if len(nodes) != 1 {
		t.Fatalf("math node count = %d, want 1", len(nodes))
	}
	change, err := doc.PrepareReplaceMathExpression(nodes[0].ID, []byte{0})
	if err != nil {
		t.Fatalf("PrepareReplaceMathExpression(new NUL) error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply(new NUL) error = %v", err)
	}
	want := []byte("$�$")
	if !bytes.Equal(got, want) {
		t.Fatalf("normalized math replacement = % x (%q), want % x (%q)", got, got, want, want)
	}
}
