package goldmark

import (
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestListItemObservationsPromoteSingleLineHeadWithOnlyChildLists(t *testing.T) {
	t.Parallel()

	source := []byte("- parent\n  - child\n- leaf\n\n- complex\n\n  second paragraph\n")
	observations, err := New().Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	seen := make(map[string]parser.Node)
	for _, observation := range observations {
		if observation.Kind != parser.KindListItem {
			continue
		}
		seen[string(source[observation.Range.Start:observation.Range.End])] = observation
	}

	parent, ok := seen["parent"]
	if !ok {
		t.Fatal("single-line parent with only child list was not observed")
	}
	if !parent.HasListChildren {
		t.Fatal("parent HasListChildren = false, want true")
	}
	leaf, ok := seen["leaf"]
	if !ok || leaf.HasListChildren {
		t.Fatalf("leaf observation = %+v, want present with HasListChildren=false", leaf)
	}
	child, ok := seen["child"]
	if !ok || child.HasListChildren || !child.HasListParent {
		t.Fatalf("child observation = %+v, want nested leaf", child)
	}
	if _, ok := seen["complex"]; ok {
		t.Fatal("multi-block non-list item was observed as a supported list item")
	}
}

func TestListItemObservationsRejectParentWithNonListTrailingBlock(t *testing.T) {
	t.Parallel()

	tests := [][]byte{
		[]byte("- parent\n\n  second paragraph\n"),
		[]byte("- parent\n\n      code\n"),
		[]byte("- parent\n\n  > quote\n"),
	}
	for _, source := range tests {
		observations, err := New().Parse(source)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", source, err)
		}
		for _, observation := range observations {
			if observation.Kind == parser.KindListItem && string(source[observation.Range.Start:observation.Range.End]) == "parent" {
				t.Fatalf("unsupported parent observed for %q", source)
			}
		}
	}
}
