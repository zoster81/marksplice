package publictest

import (
	"bytes"
	"errors"
	"testing"

	marksplice "github.com/zoster81/marksplice"
)

func TestPublicReferenceDefinitionTitleReplacementPreservesSourceStyle(t *testing.T) {
	t.Parallel()

	source := []byte("  [docs]: <old/path>\t'Old title'   \r\n\r\nparagraph\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	definition := publicFirstReferenceDefinition(t, doc)
	change, err := doc.PrepareReplaceReferenceDefinitionTitle(definition.ID(), []byte("New title"))
	if err != nil {
		t.Fatalf("PrepareReplaceReferenceDefinitionTitle() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("  [docs]: <old/path>\t'New title'   \r\n\r\nparagraph\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("updated source = %q, want %q", got, want)
	}
}

func TestPublicReferenceDefinitionTitleReplacementRejectsMissingOrUnsafeTitle(t *testing.T) {
	t.Parallel()

	withoutTitle, err := marksplice.Parse([]byte("[docs]: <target>\n"))
	if err != nil {
		t.Fatalf("Parse(without title) error = %v", err)
	}
	without := publicFirstReferenceDefinition(t, withoutTitle)
	if _, err := withoutTitle.PrepareReplaceReferenceDefinitionTitle(without.ID(), []byte("new")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceReferenceDefinitionTitle(no title) error = %v, want ErrInvalidReplacement", err)
	}

	withTitle, err := marksplice.Parse([]byte("[docs]: <target> \"old\"\n"))
	if err != nil {
		t.Fatalf("Parse(with title) error = %v", err)
	}
	with := publicFirstReferenceDefinition(t, withTitle)
	for _, replacement := range [][]byte{nil, []byte("line\nbreak"), []byte("quote\"break")} {
		if _, err := withTitle.PrepareReplaceReferenceDefinitionTitle(with.ID(), replacement); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceReferenceDefinitionTitle(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
}

func TestPublicRemoveUnusedReferenceDefinitionOwnsPhysicalLine(t *testing.T) {
	t.Parallel()

	source := []byte("before\r\n\r\n  [unused]: <target> \"Title\"   \r\n\r\nafter\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	definition := publicFirstReferenceDefinition(t, doc)
	change, err := doc.PrepareRemoveReferenceDefinition(definition.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveReferenceDefinition() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("before\r\n\r\n\r\nafter\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("updated source = %q, want %q", got, want)
	}
}

func TestPublicRemoveUnusedReferenceDefinitionPreservesShiftedUsedReference(t *testing.T) {
	t.Parallel()

	source := []byte("[unused]: <old>\n[used]: <target>\n\n[used]\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	definition := publicFirstReferenceDefinition(t, doc)
	change, err := doc.PrepareRemoveReferenceDefinition(definition.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveReferenceDefinition() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("[used]: <target>\n\n[used]\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("updated source = %q, want %q", got, want)
	}
}

func TestPublicRemoveReferenceDefinitionIgnoresFrontMatterPseudoUsage(t *testing.T) {
	t.Parallel()

	source := []byte("---\ntitle: docs\nnote: [docs]\n---\n\n[docs]: <target>\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	definition := publicFirstReferenceDefinition(t, doc)
	change, err := doc.PrepareRemoveReferenceDefinition(definition.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveReferenceDefinition() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("---\ntitle: docs\nnote: [docs]\n---\n\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("updated source = %q, want %q", got, want)
	}
}

func TestPublicRemoveUsedReferenceDefinitionFailsClosed(t *testing.T) {
	t.Parallel()

	tests := []string{
		"[docs]: <target>\n\n[full][docs]\n",
		"[docs]: <target>\n\n[docs][]\n",
		"[docs]: <target>\n\n[docs]\n",
		"[docs]: <target>\n\n![docs]\n",
	}
	for _, sourceText := range tests {
		source := []byte(sourceText)
		doc, err := marksplice.Parse(source)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", sourceText, err)
		}
		definition := publicFirstReferenceDefinition(t, doc)
		if _, err := doc.PrepareRemoveReferenceDefinition(definition.ID()); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("PrepareRemoveReferenceDefinition(%q) error = %v, want ErrInvalidReplacement", sourceText, err)
		}
	}
}

func publicFirstReferenceDefinition(t *testing.T, doc *marksplice.Document) marksplice.ReferenceDefinition {
	t.Helper()
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindReferenceDefinition {
			continue
		}
		definition, ok := doc.ReferenceDefinition(node.ID())
		if ok {
			return definition
		}
	}
	t.Fatal("no promoted reference definition")
	return marksplice.ReferenceDefinition{}
}
