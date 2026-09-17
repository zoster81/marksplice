package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestR30FootnoteMultilineBodyMutationPreservesContainerLayout(t *testing.T) {
	t.Parallel()

	source := []byte("See[^n]\r\n\r\n[^n]: first\r\n\r\n    second\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	definition := publicFootnoteDefinitionN(t, doc)
	change, err := doc.PrepareReplaceFootnoteDefinitionBodyMultiline(definition.ID(), []byte("alpha\n\nbeta"))
	if err != nil {
		t.Fatalf("PrepareReplaceFootnoteDefinitionBodyMultiline() error = %v", err)
	}
	updated, err := change.Apply(source)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("See[^n]\r\n\r\n[^n]: alpha\r\n\r\n    beta\r\n")
	if !bytes.Equal(updated, want) {
		t.Fatalf("Apply() = %q, want %q", updated, want)
	}
	candidate, err := marksplice.Parse(updated)
	if err != nil {
		t.Fatal(err)
	}
	bodyRanges, ok := candidate.FootnoteDefinitionBodyRanges(publicFootnoteDefinitionN(t, candidate).ID())
	if !ok || len(bodyRanges) != 2 {
		t.Fatalf("candidate body ranges = %v/%v, want two semantic ranges", bodyRanges, ok)
	}
	if got, _ := candidate.SourceRange(bodyRanges[0]); string(got) != "alpha" {
		t.Fatalf("first body range = %q, want alpha", got)
	}
	if got, _ := candidate.SourceRange(bodyRanges[1]); string(got) != "beta" {
		t.Fatalf("second body range = %q, want beta", got)
	}
}

func TestR30AppendFootnoteDefinitionResolvesExistingReference(t *testing.T) {
	t.Parallel()

	source := []byte("See[^new]\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if refs := doc.FootnoteReferences(); len(refs) != 0 {
		t.Fatalf("pre-append refs = %d, want 0 before a definition exists", len(refs))
	}
	change, err := doc.PrepareAppendFootnoteDefinition([]byte("new"), []byte("first\n\nsecond"))
	if err != nil {
		t.Fatalf("PrepareAppendFootnoteDefinition() error = %v", err)
	}
	updated, err := change.Apply(source)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("See[^new]\n\n[^new]: first\n\n    second\n")
	if !bytes.Equal(updated, want) {
		t.Fatalf("Apply() = %q, want %q", updated, want)
	}
	candidate, err := marksplice.Parse(updated)
	if err != nil {
		t.Fatal(err)
	}
	refs := candidate.FootnoteReferences()
	if len(refs) != 1 {
		t.Fatalf("post-append refs = %d, want 1", len(refs))
	}
	if _, ok := refs[0].DefinitionID(); !ok {
		t.Fatal("post-append DefinitionID() ok = false")
	}
}

func TestR30RemoveFootnoteDefinitionLeavesOccurrenceUnresolved(t *testing.T) {
	t.Parallel()

	source := []byte("See[^n]\n\n[^n]: body\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	definition := publicFootnoteDefinitionN(t, doc)
	change, err := doc.PrepareRemoveFootnoteDefinition(definition.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveFootnoteDefinition() error = %v", err)
	}
	updated, err := change.Apply(source)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("See[^n]\n\n")
	if !bytes.Equal(updated, want) {
		t.Fatalf("Apply() = %q, want %q", updated, want)
	}
	candidate, err := marksplice.Parse(updated)
	if err != nil {
		t.Fatal(err)
	}
	if refs := candidate.FootnoteReferences(); len(refs) != 0 {
		t.Fatalf("refs = %d, want 0 after definition removal", len(refs))
	}
	if got := candidate.FootnoteDefinitions(); len(got) != 0 {
		t.Fatalf("definitions = %d, want 0", len(got))
	}
}

func TestR30FootnoteLifecycleFailsClosed(t *testing.T) {
	t.Parallel()

	source := []byte("[^n]: body\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	definition := publicFootnoteDefinitionN(t, doc)
	if _, err := doc.PrepareReplaceFootnoteDefinitionBodyMultiline(definition.ID(), []byte("one\r\ntwo")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("CRLF logical body error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareAppendFootnoteDefinition([]byte("n"), []byte("other")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("duplicate append error = %v, want ErrInvalidReplacement", err)
	}
}

func publicFootnoteDefinitionN(t *testing.T, doc *marksplice.Document) marksplice.FootnoteDefinition {
	t.Helper()
	for _, definition := range doc.FootnoteDefinitions() {
		if definition.Label() == "n" {
			return definition
		}
	}
	t.Fatal("footnote definition \"n\" not found")
	return marksplice.FootnoteDefinition{}
}
