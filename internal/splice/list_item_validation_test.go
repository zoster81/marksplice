package splice

import (
	"errors"
	"testing"
)

func TestValidateOriginalListItemsRejectsUnexpectedDirectChildCount(t *testing.T) {
	t.Parallel()

	sourceBytes := []byte("- parent\n  - child one\n  - child two\n")
	doc, err := Parse(sourceBytes)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := testListItemNodeByContent(t, doc, sourceBytes, "parent")
	candidateItems, err := candidateListItemMappings(sourceBytes)
	if err != nil {
		t.Fatalf("candidateListItemMappings() error = %v", err)
	}
	candidateParent := candidateItems[parent.ListItemSource.LineRange.Start]
	candidateParent.DirectChildCount--
	candidateItems[parent.ListItemSource.LineRange.Start] = candidateParent

	if err := doc.validateOriginalListItemsAfterPatches(sourceBytes, candidateItems, nil, nil, nil); !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("validateOriginalListItemsAfterPatches() error = %v, want ErrInvalidReplacement", err)
	}
}

func TestValidateListItemReplacementRejectsUnexpectedDirectChildCount(t *testing.T) {
	t.Parallel()

	sourceBytes := []byte("- parent\n  - child one\n  - child two\n")
	doc, err := Parse(sourceBytes)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := testListItemNodeByContent(t, doc, sourceBytes, "parent")
	candidateItems, err := candidateListItemMappings(sourceBytes)
	if err != nil {
		t.Fatalf("candidateListItemMappings() error = %v", err)
	}
	candidateParent := candidateItems[parent.ListItemSource.LineRange.Start]
	candidateParent.DirectChildCount--
	candidateItems[parent.ListItemSource.LineRange.Start] = candidateParent

	replacementLength := parent.ContentRange.End - parent.ContentRange.Start
	if err := validateListItemReplacement(candidateItems, parent, replacementLength); !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("validateListItemReplacement() error = %v, want ErrInvalidReplacement", err)
	}
}

func TestListItemDirectChildCountDeltas(t *testing.T) {
	t.Parallel()

	parentA := NodeID("parent-a")
	parentB := NodeID("parent-b")
	tests := []struct {
		name    string
		removed NodeID
		added   NodeID
		want    map[NodeID]int
	}{
		{name: "no parent", want: nil},
		{name: "same parent", removed: parentA, added: parentA, want: nil},
		{name: "remove", removed: parentA, want: map[NodeID]int{parentA: -1}},
		{name: "add", added: parentA, want: map[NodeID]int{parentA: 1}},
		{name: "transfer", removed: parentA, added: parentB, want: map[NodeID]int{parentA: -1, parentB: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := listItemDirectChildCountDeltas(tt.removed, tt.added)
			if len(got) != len(tt.want) {
				t.Fatalf("listItemDirectChildCountDeltas() = %v, want %v", got, tt.want)
			}
			for id, delta := range tt.want {
				if got[id] != delta {
					t.Fatalf("listItemDirectChildCountDeltas()[%q] = %d, want %d", id, got[id], delta)
				}
			}
		})
	}
}

func testListItemNodeByContent(t *testing.T, doc *Document, sourceBytes []byte, content string) Node {
	t.Helper()
	for _, node := range doc.nodes {
		if node.Kind == KindListItem && string(sourceBytes[node.ContentRange.Start:node.ContentRange.End]) == content {
			return node
		}
	}
	t.Fatalf("list item %q not found", content)
	return Node{}
}
