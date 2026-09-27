package splice

import (
	"slices"
	"strings"
	"testing"
)

func TestCompositionDeltasOwnReplacementStorage(t *testing.T) {
	for _, candidate := range [][]string{
		{"a", "changed", "c", "also changed"},
		{"a", "inserted", "b", "c", "d"},
		{"a", "d"},
	} {
		original := []string{"a", "b", "c", "d"}
		want := slices.Clone(candidate)
		deltas := newCompositionDeltas(original, candidate, 0)
		clear(candidate)
		got, ok := applyCompositionDeltas(original, deltas)
		if !ok || !slices.Equal(got, want) {
			t.Fatalf("replacement storage was not detached: got %v, want %v, ok %v", got, want, ok)
		}
	}
}

func TestCompositionViewsReuseAcrossChangingDocuments(t *testing.T) {
	inputs := []string{
		"# Heading\n\n[a](one) and [^note].\n\n[^note]: first note\n\n| a | b |\n| - | - |\n| c | d |\n",
		"plain\n",
		strings.Repeat("[different](two) and [^other].\n\n", 16) + "[^other]: second note\n",
		"",
		"- one\n  - nested\n\n> quoted\n",
	}
	var scratch compositionModelView
	for _, input := range inputs {
		document, err := Parse([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		previous := scratch
		scratch.reset(document)
		want := compositionModelViews(document)
		if !slices.Equal(scratch.nodes, want.nodes) || !slices.Equal(scratch.links, want.links) ||
			!slices.Equal(scratch.footnotes, want.footnotes) {
			t.Fatalf("reused views differ from fresh views for %q", input)
		}
		checkClearedCompositionTail(t, previous.nodes, len(scratch.nodes))
		checkClearedCompositionTail(t, previous.links, len(scratch.links))
		checkClearedCompositionTail(t, previous.footnotes, len(scratch.footnotes))
	}
}

func checkClearedCompositionTail[T comparable](t *testing.T, previous []T, size int) {
	t.Helper()
	var zero T
	for index := size; index < len(previous); index++ {
		if previous[index] != zero {
			t.Fatalf("truncated composition view %d retains a value", index)
		}
	}
}
