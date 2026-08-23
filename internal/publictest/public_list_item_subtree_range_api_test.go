package publictest

import (
	"bytes"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicListItemSubtreeRangeExposesExactCompleteOwnedSource(t *testing.T) {
	t.Parallel()

	source := []byte("- root\n  - child\n    - grandchild\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	tests := []struct {
		content string
		want    []byte
	}{
		{content: "root", want: []byte("- root\n  - child\n    - grandchild\n")},
		{content: "child", want: []byte("  - child\n    - grandchild\n")},
		{content: "grandchild", want: []byte("    - grandchild\n")},
		{content: "tail", want: []byte("- tail\n")},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.content, func(t *testing.T) {
			t.Parallel()
			item := publicListItemByContent(t, doc, source, tt.content)
			range_, ok := item.SubtreeRange()
			if !ok {
				t.Fatal("SubtreeRange() ok = false, want true")
			}
			got, ok := doc.SourceRange(range_)
			if !ok {
				t.Fatalf("SourceRange(%v) ok = false", range_)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("subtree source = %q, want %q", got, tt.want)
			}
			if range_ == item.Range() {
				t.Fatalf("SubtreeRange() = content Range() = %v; structural and content semantics must remain distinct", range_)
			}
		})
	}
}

func TestPublicListItemSubtreeRangeHandlesCRLFUnicodeAndFinalUnterminatedLeaf(t *testing.T) {
	t.Parallel()

	source := []byte("10. root \u03c0\r\n    - child\r\n11. tail")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	root := publicListItemByContent(t, doc, source, "root \u03c0")
	rootRange, ok := root.SubtreeRange()
	if !ok {
		t.Fatal("root SubtreeRange() ok = false")
	}
	rootSource, ok := doc.SourceRange(rootRange)
	if !ok || !bytes.Equal(rootSource, []byte("10. root \u03c0\r\n    - child\r\n")) {
		t.Fatalf("root subtree source = %q, ok %v", rootSource, ok)
	}
	tail := publicListItemByContent(t, doc, source, "tail")
	tailRange, ok := tail.SubtreeRange()
	if !ok {
		t.Fatal("tail SubtreeRange() ok = false")
	}
	tailSource, ok := doc.SourceRange(tailRange)
	if !ok || !bytes.Equal(tailSource, []byte("11. tail")) {
		t.Fatalf("tail subtree source = %q, ok %v", tailSource, ok)
	}
}

func TestPublicListItemSubtreeRangeRejectsIncompleteSupportedParent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
	}{
		{
			name:   "unsupported direct child",
			source: []byte("- root\n  - complex\n\n    second paragraph\n- tail\n"),
		},
		{
			name:   "unsupported grandchild",
			source: []byte("- root\n  - child\n    - complex\n\n      second paragraph\n- tail\n"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			root := publicListItemByContent(t, doc, tt.source, "root")
			if !root.HasChildren() {
				t.Fatal("root HasChildren() = false, want semantic children")
			}
			if range_, ok := root.SubtreeRange(); ok || range_ != (marksplice.Range{}) {
				t.Fatalf("root SubtreeRange() = (%v,%v), want zero,false", range_, ok)
			}
		})
	}
}

func TestPublicListItemSubtreeRangeTracksReparsedStructuralMutation(t *testing.T) {
	t.Parallel()

	source := []byte("- parent\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := publicListItemByContent(t, doc, source, "parent")
	oldRange, ok := parent.SubtreeRange()
	if !ok {
		t.Fatal("initial parent SubtreeRange() ok = false")
	}
	oldSource, _ := doc.SourceRange(oldRange)
	if !bytes.Equal(oldSource, []byte("- parent\n")) {
		t.Fatalf("initial subtree source = %q", oldSource)
	}

	change, err := doc.PrepareAppendListItemChild(parent.ID(), []byte("  - child\n"))
	if err != nil {
		t.Fatalf("PrepareAppendListItemChild() error = %v", err)
	}
	updatedSource, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	updated, err := marksplice.Parse(updatedSource)
	if err != nil {
		t.Fatalf("Parse(updated) error = %v", err)
	}
	updatedParent := publicListItemByContent(t, updated, updatedSource, "parent")
	updatedRange, ok := updatedParent.SubtreeRange()
	if !ok {
		t.Fatal("updated parent SubtreeRange() ok = false")
	}
	got, _ := updated.SourceRange(updatedRange)
	if !bytes.Equal(got, []byte("- parent\n  - child\n")) {
		t.Fatalf("updated subtree source = %q", got)
	}
	if parentRange, _ := parent.SubtreeRange(); parentRange != oldRange {
		t.Fatalf("original snapshot SubtreeRange() = %v, want unchanged %v", parentRange, oldRange)
	}
}

func TestZeroValueListItemHasNoSubtreeRange(t *testing.T) {
	t.Parallel()

	var item marksplice.ListItem
	if range_, ok := item.SubtreeRange(); ok || range_ != (marksplice.Range{}) {
		t.Fatalf("zero ListItem SubtreeRange() = (%v,%v), want zero,false", range_, ok)
	}
}
