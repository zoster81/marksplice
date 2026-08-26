package splice

import (
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestParseWithBackendConsumesParserIndependentObservations(t *testing.T) {
	t.Parallel()

	backend := &m111FakeBackend{observed: parser.DocumentObservations{
		Nodes: []parser.Node{{
			Kind:     parser.KindParagraph,
			Range:    parser.Range{Start: 0, End: 4},
			TopLevel: true,
		}},
	}}
	document, err := parseWithBackend([]byte("text"), backend)
	if err != nil {
		t.Fatalf("parseWithBackend() error = %v", err)
	}
	if backend.parseCalls != 1 {
		t.Fatalf("ParseDocument calls = %d, want 1", backend.parseCalls)
	}
	nodes := document.Nodes()
	if len(nodes) != 1 || nodes[0].Kind != KindParagraph || nodes[0].Range != (Range{Start: 0, End: 4}) {
		t.Fatalf("Nodes() = %+v, want one paragraph [0,4)", nodes)
	}
}

func TestParseWithBackendRejectsNilBackend(t *testing.T) {
	t.Parallel()

	if document, err := parseWithBackend([]byte("text"), nil); err == nil || document != nil {
		t.Fatalf("parseWithBackend(nil) = %+v, %v; want nil/error", document, err)
	}
}

type m111FakeBackend struct {
	observed   parser.DocumentObservations
	parseCalls int
}

func (f *m111FakeBackend) ParseDocument([]byte) (parser.DocumentObservations, error) {
	f.parseCalls++
	return f.observed, nil
}

func (*m111FakeBackend) ValidateNestedBlockquoteBlocks([]byte, parser.Range, []byte, int) error {
	return nil
}

func (*m111FakeBackend) ValidateNestedBlockquoteParagraph([]byte, parser.Range, []parser.Range, int) error {
	return nil
}

func (*m111FakeBackend) ValidateConstructionInlineHierarchy([]byte, []parser.ConstructionInlineExpectation, []parser.ConstructionReferenceInlineExpectation) error {
	return nil
}

func (*m111FakeBackend) ValidateConstructionLinkImages([]byte, []parser.ConstructionLinkImageExpectation) error {
	return nil
}

func (*m111FakeBackend) ValidateConstructionReferenceInlines([]byte, []parser.ConstructionReferenceInlineExpectation) error {
	return nil
}

func (*m111FakeBackend) ResolveConstructionReference(string, []parser.ConstructionReferenceDefinition) (parser.ConstructionReferenceDefinition, error) {
	return parser.ConstructionReferenceDefinition{}, nil
}

func (*m111FakeBackend) ReferenceLabelKey(label string) string { return label }

var _ parser.Backend = (*m111FakeBackend)(nil)
