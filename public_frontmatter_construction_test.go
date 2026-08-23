package marksplice_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicDocumentBuilderWritesCanonicalYAMLFrontMatter(t *testing.T) {
	t.Parallel()

	fields := []marksplice.FrontMatterFieldInput{
		{Key: "title", Value: "Marksplice"},
		{Key: "description", Value: "GFM: # safe π"},
	}
	var builder marksplice.DocumentBuilder
	if err := builder.AppendHeading(1, "Title"); err != nil {
		t.Fatalf("AppendHeading() error = %v", err)
	}
	if err := builder.SetYAMLFrontMatter(fields...); err != nil {
		t.Fatalf("SetYAMLFrontMatter() error = %v", err)
	}
	fields[0].Key = "mutated"
	fields[0].Value = "mutated"

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("---\ntitle: \"Marksplice\"\ndescription: \"GFM: # safe π\"\n---\n\n# Title\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
	assertGeneratedFrontMatter(t, got, marksplice.FrontMatterFormatYAML, map[string]string{
		"title":       "Marksplice",
		"description": "GFM: # safe π",
	})
}

func TestPublicDocumentBuilderWritesCanonicalTOMLFrontMatter(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.SetTOMLFrontMatter(
		marksplice.FrontMatterFieldInput{Key: "title", Value: "Marksplice"},
		marksplice.FrontMatterFieldInput{Key: "owner.name", Value: "Giovanni π"},
	); err != nil {
		t.Fatalf("SetTOMLFrontMatter() error = %v", err)
	}
	if err := builder.AppendParagraph("Body."); err != nil {
		t.Fatalf("AppendParagraph() error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("+++\ntitle = \"Marksplice\"\nowner.name = \"Giovanni π\"\n+++\n\nBody.\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
	assertGeneratedFrontMatter(t, got, marksplice.FrontMatterFormatTOML, map[string]string{
		"title":      "Marksplice",
		"owner.name": "Giovanni π",
	})
}

func TestPublicDocumentBuilderWritesFrontMatterOnlyDocument(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.SetYAMLFrontMatter(marksplice.FrontMatterFieldInput{Key: "title", Value: "Only metadata"}); err != nil {
		t.Fatalf("SetYAMLFrontMatter() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("---\ntitle: \"Only metadata\"\n---\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
	assertGeneratedFrontMatter(t, got, marksplice.FrontMatterFormatYAML, map[string]string{"title": "Only metadata"})
}

func TestPublicDocumentBuilderRejectsInvalidFrontMatterConstruction(t *testing.T) {
	t.Parallel()

	invalidFields := [][]marksplice.FrontMatterFieldInput{
		nil,
		{},
		{{Key: "", Value: "value"}},
		{{Key: "bad key", Value: "value"}},
		{{Key: "title", Value: ""}},
		{{Key: "title", Value: "line\nbreak"}},
		{{Key: "title", Value: "line\rbreak"}},
		{{Key: "title", Value: "quote\"inside"}},
		{{Key: "title", Value: "back\\slash"}},
		{{Key: "title", Value: "contains\x00nul"}},
		{{Key: "title", Value: string([]byte{0xff})}},
		{{Key: "title", Value: "one"}, {Key: "title", Value: "two"}},
	}
	for _, fields := range invalidFields {
		for _, set := range []func(*marksplice.DocumentBuilder) error{
			func(builder *marksplice.DocumentBuilder) error { return builder.SetYAMLFrontMatter(fields...) },
			func(builder *marksplice.DocumentBuilder) error { return builder.SetTOMLFrontMatter(fields...) },
		} {
			var builder marksplice.DocumentBuilder
			if err := set(&builder); !errors.Is(err, marksplice.ErrInvalidConstruction) {
				t.Fatalf("front-matter setter(%v) error = %v, want ErrInvalidConstruction", fields, err)
			}
			if got, err := builder.Markdown(); err != nil || len(got) != 0 {
				t.Fatalf("builder after rejected front matter Markdown() = %q, %v; want empty, nil", got, err)
			}
		}
	}
}

func TestPublicDocumentBuilderRejectsSecondFrontMatterEnvelope(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	first := marksplice.FrontMatterFieldInput{Key: "title", Value: "first"}
	if err := builder.SetYAMLFrontMatter(first); err != nil {
		t.Fatalf("SetYAMLFrontMatter(first) error = %v", err)
	}
	if err := builder.SetYAMLFrontMatter(marksplice.FrontMatterFieldInput{Key: "title", Value: "second"}); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("SetYAMLFrontMatter(second) error = %v, want ErrInvalidConstruction", err)
	}
	if err := builder.SetTOMLFrontMatter(marksplice.FrontMatterFieldInput{Key: "title", Value: "second"}); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("SetTOMLFrontMatter(second) error = %v, want ErrInvalidConstruction", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	if want := []byte("---\ntitle: \"first\"\n---\n"); !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicNilDocumentBuilderRejectsFrontMatter(t *testing.T) {
	t.Parallel()

	var builder *marksplice.DocumentBuilder
	field := marksplice.FrontMatterFieldInput{Key: "title", Value: "value"}
	if err := builder.SetYAMLFrontMatter(field); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil SetYAMLFrontMatter() error = %v, want ErrInvalidConstruction", err)
	}
	if err := builder.SetTOMLFrontMatter(field); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil SetTOMLFrontMatter() error = %v, want ErrInvalidConstruction", err)
	}
}

func assertGeneratedFrontMatter(t *testing.T, source []byte, format marksplice.FrontMatterFormat, want map[string]string) {
	t.Helper()
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	got := make(map[string]string)
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindFrontMatterField {
			continue
		}
		field, ok := doc.FrontMatterField(node.ID())
		if !ok {
			t.Fatalf("FrontMatterField(%q) ok = false", node.ID())
		}
		if field.Format() != format {
			t.Fatalf("field %q format = %v, want %v", field.Key(), field.Format(), format)
		}
		got[field.Key()] = string(source[field.Range().Start:field.Range().End])
	}
	if len(got) != len(want) {
		t.Fatalf("generated front-matter fields = %v, want %v", got, want)
	}
	for key, value := range want {
		if got[key] != value {
			t.Fatalf("generated front-matter field %q = %q, want %q", key, got[key], value)
		}
	}
}
