package differential

import (
	"errors"
	"strings"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestHarnessAcceptsEquivalentDocumentObservations(t *testing.T) {
	t.Parallel()

	observed := parser.DocumentObservations{
		Nodes: []parser.Node{{Kind: parser.KindParagraph, Range: parser.Range{Start: 0, End: 4}, TopLevel: true}},
	}
	oracle := &fakeBackend{observed: observed}
	candidate := &fakeBackend{observed: parser.DocumentObservations{
		Nodes:                     append([]parser.Node(nil), observed.Nodes...),
		LinkUsages:                []parser.LinkUsage{},
		UnresolvedReferenceUsages: []parser.UnresolvedReferenceUsage{},
		FootnoteDefinitions:       []parser.FootnoteDefinitionObservation{},
		FootnoteReferences:        []parser.FootnoteReferenceObservation{},
		MathExpressions:           []parser.MathExpressionObservation{},
	}}

	if err := (Harness{Oracle: oracle, Candidate: candidate}).CompareDocument([]byte("text")); err != nil {
		t.Fatalf("CompareDocument() error = %v", err)
	}
}

func TestHarnessReportsDeterministicDocumentObservationMismatch(t *testing.T) {
	t.Parallel()

	oracle := &fakeBackend{observed: parser.DocumentObservations{
		Nodes: []parser.Node{{Kind: parser.KindParagraph, Range: parser.Range{Start: 0, End: 4}, TopLevel: true}},
	}}
	candidate := &fakeBackend{observed: parser.DocumentObservations{
		Nodes: []parser.Node{{Kind: parser.KindHeading, Range: parser.Range{Start: 0, End: 4}, Level: 1, TopLevel: true}},
	}}

	err := (Harness{Oracle: oracle, Candidate: candidate}).CompareDocument([]byte("text"))
	if err == nil || !strings.Contains(err.Error(), "Nodes[0]") {
		t.Fatalf("CompareDocument() error = %v, want deterministic Nodes[0] mismatch", err)
	}
}

func TestHarnessComparesConstructionProofOutcomes(t *testing.T) {
	t.Parallel()

	oracle := &fakeBackend{}
	candidate := &fakeBackend{linkImageErr: errors.New("candidate rejected")}
	harness := Harness{Oracle: oracle, Candidate: candidate}
	err := harness.CompareConstructionLinkImages([]byte("[x](y)"), []parser.ConstructionLinkImageExpectation{{
		Kind:        parser.KindInlineLink,
		SyntaxRange: parser.Range{Start: 0, End: 6},
		LabelRange:  parser.Range{Start: 1, End: 2},
		Destination: "y",
	}})
	if err == nil || !strings.Contains(err.Error(), "construction link/image") {
		t.Fatalf("CompareConstructionLinkImages() error = %v, want parity mismatch", err)
	}
}

func TestHarnessRejectsBackendInputMutation(t *testing.T) {
	t.Parallel()

	source := []byte("text")
	oracle := &mutatingBackend{fakeBackend: &fakeBackend{observed: parser.DocumentObservations{
		Nodes: []parser.Node{{Kind: parser.KindParagraph, Range: parser.Range{Start: 0, End: len(source)}, TopLevel: true}},
	}}}
	candidate := &fakeBackend{observed: oracle.observed}
	err := (Harness{Oracle: oracle, Candidate: candidate}).CompareDocument(source)
	if err == nil || !strings.Contains(err.Error(), "mutated source") {
		t.Fatalf("CompareDocument() error = %v, want source mutation rejection", err)
	}
	if string(source) != "text" {
		t.Fatalf("caller source changed to %q", source)
	}
}

func TestHarnessComparesReferenceSemantics(t *testing.T) {
	t.Parallel()

	oracle := &fakeBackend{labelKey: "same", resolved: parser.ConstructionReferenceDefinition{Label: "A", Destination: "/a"}}
	candidate := &fakeBackend{labelKey: "different", resolved: parser.ConstructionReferenceDefinition{Label: "A", Destination: "/b"}}
	harness := Harness{Oracle: oracle, Candidate: candidate}

	if err := harness.CompareReferenceLabelKey(" A "); err == nil || !strings.Contains(err.Error(), "reference label key") {
		t.Fatalf("CompareReferenceLabelKey() error = %v, want mismatch", err)
	}
	if err := harness.CompareConstructionReferenceResolution("A", []parser.ConstructionReferenceDefinition{{Label: "A", Destination: "/a"}}); err == nil || !strings.Contains(err.Error(), "reference resolution") {
		t.Fatalf("CompareConstructionReferenceResolution() error = %v, want mismatch", err)
	}
}

type fakeBackend struct {
	observed     parser.DocumentObservations
	parseErr     error
	linkImageErr error
	labelKey     string
	resolved     parser.ConstructionReferenceDefinition
	resolveErr   error
}

func (f *fakeBackend) ParseDocument([]byte) (parser.DocumentObservations, error) {
	return f.observed, f.parseErr
}

func (*fakeBackend) ValidateNestedBlockquoteBlocks([]byte, parser.Range, []byte, int) error {
	return nil
}

func (*fakeBackend) ValidateNestedBlockquoteParagraph([]byte, parser.Range, []parser.Range, int) error {
	return nil
}

func (*fakeBackend) ValidateConstructionInlineHierarchy([]byte, []parser.ConstructionInlineExpectation, []parser.ConstructionReferenceInlineExpectation) error {
	return nil
}

func (f *fakeBackend) ValidateConstructionLinkImages([]byte, []parser.ConstructionLinkImageExpectation) error {
	return f.linkImageErr
}

func (*fakeBackend) ValidateConstructionReferenceInlines([]byte, []parser.ConstructionReferenceInlineExpectation) error {
	return nil
}

func (f *fakeBackend) ResolveConstructionReference(string, []parser.ConstructionReferenceDefinition) (parser.ConstructionReferenceDefinition, error) {
	return f.resolved, f.resolveErr
}

func (f *fakeBackend) ReferenceLabelKey(string) string {
	return f.labelKey
}

type mutatingBackend struct {
	*fakeBackend
}

func (m *mutatingBackend) ParseDocument(source []byte) (parser.DocumentObservations, error) {
	if len(source) != 0 {
		source[0] = 'X'
	}
	return m.fakeBackend.ParseDocument(source)
}

var _ parser.Backend = (*fakeBackend)(nil)
var _ parser.Backend = (*mutatingBackend)(nil)
