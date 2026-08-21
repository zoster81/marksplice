package goldmark

import (
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestListItemObservationsExposeImmediateParentLineAnchor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     []byte
		content    string
		hasParent  bool
		parentLine int
	}{
		{
			name:       "root leaf has no list-item parent",
			source:     []byte("- root\n"),
			content:    "root",
			hasParent:  false,
			parentLine: 0,
		},
		{
			name:       "nested leaf points to root item line",
			source:     []byte("- root\n  - child\n  - sibling\n"),
			content:    "child",
			hasParent:  true,
			parentLine: 0,
		},
		{
			name:       "deep leaf points to immediate parent line",
			source:     []byte("1. root\n   1) parent\n      + child\n"),
			content:    "child",
			hasParent:  true,
			parentLine: len("1. root\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			observations, err := New().Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			var got parser.Node
			found := false
			for _, observation := range observations {
				if observation.Kind != parser.KindListItem {
					continue
				}
				if string(tt.source[observation.Range.Start:observation.Range.End]) == tt.content {
					got = observation
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("list item %q not observed", tt.content)
			}
			if got.HasListParent != tt.hasParent || got.ListParentAnchor != tt.parentLine {
				t.Fatalf("parent metadata = has %v anchor %d, want has %v anchor %d", got.HasListParent, got.ListParentAnchor, tt.hasParent, tt.parentLine)
			}
		})
	}
}
