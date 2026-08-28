package splice

import (
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
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

func TestParseWithBackendRejectsCorruptSparseParserDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   []byte
		observed parser.DocumentObservations
	}{
		{
			name:   "ordinary node cannot claim sparse detail",
			source: []byte("text"),
			observed: parser.DocumentObservations{Nodes: []parser.Node{{
				Kind: parser.KindParagraph, DetailIndex: 1, Range: parser.Range{Start: 0, End: 4}, TopLevel: true,
			}}},
		},
		{
			name:   "blockquote detail index must resolve",
			source: []byte("> q\n"),
			observed: parser.DocumentObservations{Nodes: []parser.Node{{
				Kind: parser.KindBlockquote, DetailIndex: 1, Range: parser.Range{Start: 0, End: 3}, TopLevel: true,
			}}},
		},
		{
			name:   "blockquote detail anchor must match",
			source: []byte("> q\n"),
			observed: parser.DocumentObservations{
				Nodes:             []parser.Node{{Kind: parser.KindBlockquote, DetailIndex: 1, Range: parser.Range{Start: 0, End: 3}, TopLevel: true}},
				BlockquoteDetails: []parser.BlockquoteDetail{{Anchor: 1, ContentRange: parser.Range{Start: 2, End: 3}, SemanticRanges: []parser.Range{{Start: 2, End: 3}}}},
			},
		},
		{
			name:   "fenced detail anchor must match",
			source: []byte("```\nx\n```\n"),
			observed: parser.DocumentObservations{
				Nodes:             []parser.Node{{Kind: parser.KindFencedCode, DetailIndex: 1, Range: parser.Range{Start: 4, End: 5}, Anchor: 0, TopLevel: true}},
				FencedCodeDetails: []parser.FencedCodeDetail{{Anchor: 1, ContentRanges: []parser.Range{{Start: 4, End: 5}}}},
			},
		},
		{
			name:   "table detail index must resolve",
			source: []byte("| a |\n| - |\n"),
			observed: parser.DocumentObservations{Nodes: []parser.Node{{
				Kind: parser.KindTable, DetailIndex: 1, Range: parser.Range{Start: 0, End: 5}, TopLevel: true,
			}}},
		},
		{
			name:   "table detail anchor must match",
			source: []byte("| a |\n| - |\n"),
			observed: parser.DocumentObservations{
				Nodes:        []parser.Node{{Kind: parser.KindTable, DetailIndex: 1, Range: parser.Range{Start: 0, End: 5}, TopLevel: true}},
				TableDetails: []parser.TableDetail{{Anchor: 1, ColumnCount: 1, Alignments: []parser.TableAlignment{parser.TableAlignmentDefault}}},
			},
		},
		{
			name:   "table row detail index must resolve",
			source: []byte("| a |\n"),
			observed: parser.DocumentObservations{Nodes: []parser.Node{{
				Kind: parser.KindTableRow, DetailIndex: 1, Range: parser.Range{Start: 0, End: 5},
			}}},
		},
		{
			name:   "table row detail anchor must match",
			source: []byte("| a |\n"),
			observed: parser.DocumentObservations{
				Nodes:           []parser.Node{{Kind: parser.KindTableRow, DetailIndex: 1, Range: parser.Range{Start: 0, End: 5}}},
				TableRowDetails: []parser.TableRowDetail{{RowAnchor: 1, TableAnchor: 0, ColumnCount: 1, Alignments: []parser.TableAlignment{parser.TableAlignmentDefault}}},
			},
		},
		{
			name:   "table cell detail index must resolve",
			source: []byte("| a |\n"),
			observed: parser.DocumentObservations{Nodes: []parser.Node{{
				Kind: parser.KindTableCell, DetailIndex: 1, Range: parser.Range{Start: 2, End: 3},
			}}},
		},
		{
			name:   "table cell detail range must match",
			source: []byte("| a |\n"),
			observed: parser.DocumentObservations{
				Nodes:            []parser.Node{{Kind: parser.KindTableCell, DetailIndex: 1, Range: parser.Range{Start: 2, End: 3}}},
				TableCellDetails: []parser.TableCellDetail{{Range: parser.Range{Start: 1, End: 2}, Column: 0, RowAnchor: 0, TableAnchor: 0}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			backend := &m111FakeBackend{observed: tt.observed}
			if document, err := parseWithBackend(tt.source, backend); err == nil || document != nil {
				t.Fatalf("parseWithBackend() = %+v, %v; want nil/error", document, err)
			}
		})
	}
}

func TestParseWithBackendRejectsNilBackend(t *testing.T) {
	t.Parallel()

	if document, err := parseWithBackend([]byte("text"), nil); err == nil || document != nil {
		t.Fatalf("parseWithBackend(nil) = %+v, %v; want nil/error", document, err)
	}
}

func TestDefaultParserBackendUsesNativeParser(t *testing.T) {
	t.Parallel()

	if _, ok := newParserBackend().(*native.Backend); !ok {
		t.Fatalf("newParserBackend() = %T, want *native.Backend", newParserBackend())
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
