package goldmark

import (
	"bytes"
	"slices"
	"testing"

	markparser "github.com/zoster81/marksplice/internal/parser"
)

func TestAdapterCollectsFootnotesBeforeGoldmarkReordersThem(t *testing.T) {
	t.Parallel()

	source := []byte("before[^b] and again[^a] and again[^a]\n\n[^a]: first line\n\n    second paragraph\n\n[^unused]: unused\n[^b]: bee\n")
	got, err := New().ParseDocument(source)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	aAnchor := bytes.Index(source, []byte("[^a]:"))
	unusedAnchor := bytes.Index(source, []byte("[^unused]:"))
	bAnchor := bytes.Index(source, []byte("[^b]:"))
	wantDefinitions := []markparser.FootnoteDefinitionObservation{
		{
			Anchor: aAnchor,
			Label:  "a",
			BodyRanges: []markparser.Range{
				{Start: bytes.Index(source, []byte("first line")), End: bytes.Index(source, []byte("first line")) + len("first line")},
				{Start: bytes.Index(source, []byte("second paragraph")), End: bytes.Index(source, []byte("second paragraph")) + len("second paragraph")},
			},
		},
		{
			Anchor: unusedAnchor,
			Label:  "unused",
			BodyRanges: []markparser.Range{
				{Start: bytes.Index(source, []byte("unused\n")), End: bytes.Index(source, []byte("unused\n")) + len("unused")},
			},
		},
		{
			Anchor: bAnchor,
			Label:  "b",
			BodyRanges: []markparser.Range{
				{Start: bytes.LastIndex(source, []byte("bee")), End: bytes.LastIndex(source, []byte("bee")) + len("bee")},
			},
		},
	}
	if !slices.EqualFunc(got.FootnoteDefinitions, wantDefinitions, sameFootnoteDefinitionObservation) {
		t.Fatalf("footnote definitions = %+v, want %+v", got.FootnoteDefinitions, wantDefinitions)
	}

	firstB := bytes.Index(source, []byte("[^b]"))
	firstA := bytes.Index(source, []byte("[^a]"))
	secondA := bytes.Index(source[firstA+1:], []byte("[^a]")) + firstA + 1
	wantReferences := []markparser.FootnoteReferenceObservation{
		footnoteReferenceWant(firstB, "b", bAnchor, 0),
		footnoteReferenceWant(firstA, "a", aAnchor, 0),
		footnoteReferenceWant(secondA, "a", aAnchor, 1),
	}
	if !slices.Equal(got.FootnoteReferences, wantReferences) {
		t.Fatalf("footnote references = %+v, want %+v", got.FootnoteReferences, wantReferences)
	}
}

func TestAdapterFootnotesSupersedeConflictingGFMReferenceSemantics(t *testing.T) {
	t.Parallel()

	source := []byte("foot[^n] [normal][^n] [ok][docs]\n\n[^n]: note\n\n[docs]: /target\n")
	got, err := New().ParseDocument(source)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}
	if len(got.FootnoteDefinitions) != 1 || got.FootnoteDefinitions[0].Label != "n" {
		t.Fatalf("footnote definitions = %+v, want one n definition", got.FootnoteDefinitions)
	}
	if len(got.FootnoteReferences) != 2 || got.FootnoteReferences[0].Label != "n" || got.FootnoteReferences[1].Label != "n" {
		t.Fatalf("footnote references = %+v, want two n references", got.FootnoteReferences)
	}

	for _, node := range got.Nodes {
		if node.Kind == markparser.KindReferenceDefinition && node.Label == "^n" {
			t.Fatalf("footnote definition leaked as GFM reference definition: %+v", node)
		}
	}
	if len(got.LinkUsages) != 1 {
		t.Fatalf("link usages = %+v, want only ordinary docs reference", got.LinkUsages)
	}
	usage := got.LinkUsages[0]
	if usage.Reference != "docs" || usage.Destination != "/target" || usage.Form != markparser.LinkUsageFull {
		t.Fatalf("ordinary reference usage = %+v, want docs -> /target full reference", usage)
	}
}

func TestAdapterCollectsLinkUsagesInsideFootnoteDefinitions(t *testing.T) {
	t.Parallel()

	source := []byte("[outside](#out)\n\n[^n]: [inside](b.md#part) <https://example.com>\n")
	got, err := New().ParseDocument(source)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}
	if len(got.LinkUsages) != 3 {
		t.Fatalf("LinkUsages() = %+v, want three relationships", got.LinkUsages)
	}
	wantDestinations := []string{"#out", "b.md#part", "https://example.com"}
	for index, want := range wantDestinations {
		if got.LinkUsages[index].Destination != want {
			t.Fatalf("LinkUsages[%d].Destination = %q, want %q", index, got.LinkUsages[index].Destination, want)
		}
		if index != 0 && got.LinkUsages[index].Anchor <= got.LinkUsages[index-1].Anchor {
			t.Fatalf("LinkUsages source order changed at %d: %+v", index, got.LinkUsages)
		}
	}
}

func footnoteReferenceWant(anchor int, label string, definitionAnchor, occurrence int) markparser.FootnoteReferenceObservation {
	return markparser.FootnoteReferenceObservation{
		Range:            markparser.Range{Start: anchor, End: anchor + len(label) + 3},
		LabelRange:       markparser.Range{Start: anchor + 2, End: anchor + 2 + len(label)},
		Label:            label,
		DefinitionAnchor: definitionAnchor,
		Occurrence:       occurrence,
	}
}

func sameFootnoteDefinitionObservation(left, right markparser.FootnoteDefinitionObservation) bool {
	return left.Anchor == right.Anchor && left.Label == right.Label && slices.Equal(left.BodyRanges, right.BodyRanges)
}
