package goldmark

import (
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestObserveSimpleImageUsesPublicASTSourceBoundaries(t *testing.T) {
	t.Parallel()

	source := []byte("before ![alt](old/path \"title\") after\n")
	observations, err := New().Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	var image parser.Node
	for _, observation := range observations {
		if observation.Kind == parser.KindImage {
			image = observation
			break
		}
	}
	if image.Kind != parser.KindImage {
		t.Fatal("image observation not found")
	}
	if image.Anchor != 7 {
		t.Fatalf("image anchor = %d, want 7 at the ! marker", image.Anchor)
	}
	if image.Range != (parser.Range{Start: 9, End: 12}) {
		t.Fatalf("image alt range = %v, want [9,12)", image.Range)
	}
}

func TestObserveImageFiltersCompoundAltText(t *testing.T) {
	t.Parallel()

	observations, err := New().Parse([]byte("![**alt**](old/path)\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	for _, observation := range observations {
		if observation.Kind == parser.KindImage {
			t.Fatal("compound-alt image produced a simple image observation")
		}
	}
}
