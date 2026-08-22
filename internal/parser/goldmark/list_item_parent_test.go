package goldmark

import (
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestListItemObservationsExposeContainerAndImmediateParentAnchors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		source          []byte
		content         string
		containerAnchor int
		hasParent       bool
		parentLine      int
	}{
		{
			name:            "root leaf anchors its list container",
			source:          []byte("- root\n"),
			content:         "root",
			containerAnchor: 0,
			hasParent:       false,
			parentLine:      0,
		},
		{
			name:            "nested siblings share nested list anchor",
			source:          []byte("- root\n  - child\n  - sibling\n"),
			content:         "child",
			containerAnchor: len("- root\n"),
			hasParent:       true,
			parentLine:      0,
		},
		{
			name:            "deep leaf keeps separate container and immediate parent anchors",
			source:          []byte("1. root\n   1) parent\n      + child\n"),
			content:         "child",
			containerAnchor: len("1. root\n   1) parent\n"),
			hasParent:       true,
			parentLine:      len("1. root\n"),
		},
		{
			name:            "blank line alone does not invent a second list container",
			source:          []byte("- one\n\n- two\n"),
			content:         "two",
			containerAnchor: 0,
			hasParent:       false,
			parentLine:      0,
		},
		{
			name:            "intervening paragraph starts a distinct list container",
			source:          []byte("- one\n\nParagraph.\n\n- two\n"),
			content:         "two",
			containerAnchor: len("- one\n\nParagraph.\n\n"),
			hasParent:       false,
			parentLine:      0,
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
			if got.ListContainerAnchor != tt.containerAnchor {
				t.Fatalf("container anchor = %d, want %d", got.ListContainerAnchor, tt.containerAnchor)
			}
			if got.HasListParent != tt.hasParent || got.ListParentAnchor != tt.parentLine {
				t.Fatalf("parent metadata = has %v anchor %d, want has %v anchor %d", got.HasListParent, got.ListParentAnchor, tt.hasParent, tt.parentLine)
			}
		})
	}
}
