package publictest

import (
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM106FrontMatterEnvelopeIsReadableWithoutEditableFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  []byte
		format  marksplice.FrontMatterFormat
		closing marksplice.Range
	}{
		{
			name:    "complex YAML",
			source:  []byte("---\ntags:\n  - one\n  - two\n---\n\n# Body\n"),
			format:  marksplice.FrontMatterFormatYAML,
			closing: marksplice.Range{Start: 26, End: 29},
		},
		{
			name:    "duplicate-only YAML",
			source:  []byte("---\ntitle: one\ntitle: two\n---\n\n# Body\n"),
			format:  marksplice.FrontMatterFormatYAML,
			closing: marksplice.Range{Start: 26, End: 29},
		},
		{
			name:    "empty YAML",
			source:  []byte("---\n---\n\n# Body\n"),
			format:  marksplice.FrontMatterFormatYAML,
			closing: marksplice.Range{Start: 4, End: 7},
		},
		{
			name:    "complex TOML",
			source:  []byte("+++\n[params]\nauthor = 'Ada'\n+++\n\n# Body\n"),
			format:  marksplice.FrontMatterFormatTOML,
			closing: marksplice.Range{Start: 28, End: 31},
		},
		{
			name:    "empty TOML",
			source:  []byte("+++\n+++\n\n# Body\n"),
			format:  marksplice.FrontMatterFormatTOML,
			closing: marksplice.Range{Start: 4, End: 7},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(test.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			frontMatter, ok := doc.FrontMatter()
			if !ok {
				t.Fatal("FrontMatter() ok = false, want true")
			}
			if frontMatter.Format() != test.format {
				t.Fatalf("FrontMatter.Format() = %v, want %v", frontMatter.Format(), test.format)
			}
			if frontMatter.Range() != (marksplice.Range{Start: 0, End: test.closing.End}) ||
				frontMatter.OpeningRange() != (marksplice.Range{Start: 0, End: 3}) ||
				frontMatter.ClosingRange() != test.closing {
				t.Fatalf("FrontMatter ranges = %v/%v/%v", frontMatter.Range(), frontMatter.OpeningRange(), frontMatter.ClosingRange())
			}
			if got, _ := doc.SourceRange(frontMatter.Range()); len(got) == 0 {
				t.Fatal("FrontMatter.Range() is not source-readable")
			}
		})
	}
}

func TestM106ComplexFrontMatterIsOpaqueToMarkdownIntelligence(t *testing.T) {
	t.Parallel()

	source := []byte("---\nlinks:\n  - '[inside](#target)'\nrefs:\n  - '[inside][missing]'\n---\n\n# Target\n\n[outside](#target) [outside][missing]\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if _, ok := doc.FrontMatter(); !ok {
		t.Fatal("FrontMatter() ok = false")
	}
	if relationships := doc.LinkRelationships(); len(relationships) != 1 || relationships[0].Destination() != "#target" {
		t.Fatalf("LinkRelationships() = %+v, want only outside resolved link", relationships)
	}
	report, err := marksplice.ValidateWorkspace([]marksplice.GraphDocument{{Key: "doc", Document: doc}}, nil, marksplice.WorkspaceValidationOptions{})
	if err != nil {
		t.Fatalf("ValidateWorkspace() error = %v", err)
	}
	diagnostics := report.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Kind() != marksplice.WorkspaceDiagnosticUnresolvedReference {
		t.Fatalf("diagnostics = %+v, want only outside unresolved reference", diagnostics)
	}
}

func TestM106FrontMatterFieldPromotionRemainsConservative(t *testing.T) {
	t.Parallel()

	source := []byte("+++\ntitle = 'Top'\n[params]\nauthor = 'Nested'\n+++\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if _, ok := doc.FrontMatter(); !ok {
		t.Fatal("FrontMatter() ok = false")
	}
	var fields []marksplice.FrontMatterField
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindFrontMatterField {
			continue
		}
		field, ok := doc.FrontMatterField(node.ID())
		if ok {
			fields = append(fields, field)
		}
	}
	if len(fields) != 1 || fields[0].Key() != "title" {
		t.Fatalf("promoted fields = %+v, want only top-level title", fields)
	}
}

func TestM106EmptyLeadingYAMLUsesFrontMatterPrecedence(t *testing.T) {
	t.Parallel()

	doc, err := marksplice.Parse([]byte("---\n---\n\n# Body\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if _, ok := doc.FrontMatter(); !ok {
		t.Fatal("FrontMatter() ok = false")
	}
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindThematicBreak {
			t.Fatal("empty leading YAML delimiters were also promoted as thematic breaks")
		}
	}
}

func TestM106FrontMatterDoesNotGuessUnclosedNonLeadingOrMetadataFreeEnvelope(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("---\ntags:\n  - one\n"),
		[]byte("intro\n---\ntags:\n  - one\n---\n"),
		[]byte("---\nplain paragraph\n---\n"),
	} {
		doc, err := marksplice.Parse(source)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		if frontMatter, ok := doc.FrontMatter(); ok {
			t.Fatalf("FrontMatter() = %+v, true; want false", frontMatter)
		}
	}

	var nilDocument *marksplice.Document
	if frontMatter, ok := nilDocument.FrontMatter(); ok || frontMatter != (marksplice.FrontMatter{}) {
		t.Fatalf("nil FrontMatter() = %+v/%v", frontMatter, ok)
	}
	var zero marksplice.FrontMatter
	if zero.Format() != marksplice.FrontMatterFormatUnknown || zero.Range() != (marksplice.Range{}) ||
		zero.OpeningRange() != (marksplice.Range{}) || zero.ClosingRange() != (marksplice.Range{}) {
		t.Fatalf("zero FrontMatter = %+v", zero)
	}
}
