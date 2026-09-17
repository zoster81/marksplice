package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestR30ReferenceOccurrenceRetargetsAllSupportedForms(t *testing.T) {
	t.Parallel()

	source := []byte("[one]: <dest-one>\n[two]: <dest-two>\n\n[visible][one] [one][] [one] ![alt][one]\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	relationships := doc.LinkRelationships()
	if len(relationships) != 4 {
		t.Fatalf("LinkRelationships() count = %d, want 4", len(relationships))
	}
	for _, relationship := range relationships {
		reference, form, ok := relationship.Reference()
		if !ok || reference != "one" {
			t.Fatalf("relationship reference = %q/%v/%v, want one/reference", reference, form, ok)
		}
		change, err := doc.PrepareRetargetReferenceOccurrence(relationship.SourceOffset(), []byte("two"))
		if err != nil {
			t.Fatalf("PrepareRetargetReferenceOccurrence(%v) error = %v", form, err)
		}
		updated, err := change.Apply(source)
		if err != nil {
			t.Fatal(err)
		}
		candidate, err := marksplice.Parse(updated)
		if err != nil {
			t.Fatal(err)
		}
		var found bool
		for _, got := range candidate.LinkRelationships() {
			if got.SourceOffset() != relationship.SourceOffset() {
				continue
			}
			gotReference, gotForm, ok := got.Reference()
			if !ok || gotReference != "two" || gotForm != marksplice.ReferenceFormFull || got.Destination() != "dest-two" {
				t.Fatalf("retargeted relationship = ref %q form %v dest %q ok %v", gotReference, gotForm, got.Destination(), ok)
			}
			found = true
		}
		if !found {
			t.Fatalf("retargeted relationship at %d not found in %q", relationship.SourceOffset(), updated)
		}
	}
}

func TestR30ReferenceDefinitionRenamePreservesVisibleOccurrenceLabels(t *testing.T) {
	t.Parallel()

	source := []byte("[one]: <dest>\n\n[visible][one] [one][] [one]\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	definition := publicReferenceDefinitionByLabel(t, doc, "one")
	change, err := doc.PrepareRenameReferenceDefinition(definition.ID(), []byte("renamed"))
	if err != nil {
		t.Fatalf("PrepareRenameReferenceDefinition() error = %v", err)
	}
	updated, err := change.Apply(source)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("[renamed]: <dest>\n\n[visible][renamed] [one][renamed] [one][renamed]\n")
	if !bytes.Equal(updated, want) {
		t.Fatalf("Apply() = %q, want %q", updated, want)
	}
	candidate, err := marksplice.Parse(updated)
	if err != nil {
		t.Fatal(err)
	}
	for _, relationship := range candidate.LinkRelationships() {
		reference, form, ok := relationship.Reference()
		if !ok || reference != "renamed" || form != marksplice.ReferenceFormFull || relationship.Destination() != "dest" {
			t.Fatalf("renamed relationship = ref %q form %v dest %q ok %v", reference, form, relationship.Destination(), ok)
		}
	}
}

func TestR30AppendReferenceDefinitionResolvesExplicitOccurrences(t *testing.T) {
	t.Parallel()

	source := []byte("Paragraph [visible][new] [new][].\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(doc.LinkRelationships()); got != 0 {
		t.Fatalf("pre-insert LinkRelationships() = %d, want 0", got)
	}
	change, err := doc.PrepareAppendReferenceDefinition([]byte("new"), []byte("dest"))
	if err != nil {
		t.Fatalf("PrepareAppendReferenceDefinition() error = %v", err)
	}
	updated, err := change.Apply(source)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("Paragraph [visible][new] [new][].\r\n\r\n[new]: <dest>\r\n")
	if !bytes.Equal(updated, want) {
		t.Fatalf("Apply() = %q, want %q", updated, want)
	}
	candidate, err := marksplice.Parse(updated)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(candidate.LinkRelationships()); got != 2 {
		t.Fatalf("post-insert LinkRelationships() = %d, want 2", got)
	}
	for _, relationship := range candidate.LinkRelationships() {
		if relationship.Destination() != "dest" {
			t.Fatalf("resolved destination = %q, want dest", relationship.Destination())
		}
	}
}

func TestR30ReferenceDefinitionTitleLifecycle(t *testing.T) {
	t.Parallel()

	source := []byte("[ref]: <dest>\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	definition := publicReferenceDefinitionByLabel(t, doc, "ref")
	add, err := doc.PrepareAddReferenceDefinitionTitle(definition.ID(), []byte("Title"))
	if err != nil {
		t.Fatalf("PrepareAddReferenceDefinitionTitle() error = %v", err)
	}
	withTitle, err := add.Apply(source)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte("[ref]: <dest> \"Title\"\n"); !bytes.Equal(withTitle, want) {
		t.Fatalf("add Apply() = %q, want %q", withTitle, want)
	}
	withTitleDoc, err := marksplice.Parse(withTitle)
	if err != nil {
		t.Fatal(err)
	}
	withTitleDefinition := publicReferenceDefinitionByLabel(t, withTitleDoc, "ref")
	remove, err := withTitleDoc.PrepareRemoveReferenceDefinitionTitle(withTitleDefinition.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveReferenceDefinitionTitle() error = %v", err)
	}
	roundTrip, err := remove.Apply(withTitle)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(roundTrip, source) {
		t.Fatalf("title round-trip = %q, want %q", roundTrip, source)
	}
}

func TestR30ReferenceLifecycleFailsClosed(t *testing.T) {
	t.Parallel()

	source := []byte("[one]: <dest>\n\n[x][one]\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	definition := publicReferenceDefinitionByLabel(t, doc, "one")
	if _, err := doc.PrepareRenameReferenceDefinition(definition.ID(), []byte("bad]label")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("rename invalid label error = %v, want ErrInvalidReplacement", err)
	}
	relationship := doc.LinkRelationships()[0]
	if _, err := doc.PrepareRetargetReferenceOccurrence(relationship.SourceOffset(), []byte("missing")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("retarget missing definition error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareAppendReferenceDefinition([]byte("one"), []byte("other")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("append duplicate label error = %v, want ErrInvalidReplacement", err)
	}
}

func publicReferenceDefinitionByLabel(t *testing.T, doc *marksplice.Document, label string) marksplice.ReferenceDefinition {
	t.Helper()
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindReferenceDefinition {
			continue
		}
		definition, ok := doc.ReferenceDefinition(node.ID())
		if ok && definition.Label() == label {
			return definition
		}
	}
	t.Fatalf("reference definition %q not found", label)
	return marksplice.ReferenceDefinition{}
}
