package native_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	goldmarkparser "github.com/zoster81/marksplice/internal/parser/goldmark"
	"github.com/zoster81/marksplice/internal/parser/native"
)

func TestBackendImplementsFrozenParserContract(t *testing.T) {
	t.Parallel()
	var _ parser.Backend = native.New()
}

func TestBackendParseDocumentMatchesGoldmarkFocusedM114Surface(t *testing.T) {
	t.Parallel()

	source := []byte("# Head\n\n> quote *em*\n\n- [x] task [docs][ref]\n\n| A | B |\n| :- | -: |\n| x | y |\n\n[ref]: <https://example.test/a> \"Guide\"\n\nplain ~~gone~~ `code` <https://example.test>\n")
	want, err := goldmarkparser.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("Goldmark ParseDocument() error = %v", err)
	}
	before := bytes.Clone(source)
	got, err := native.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("native ParseDocument() error = %v", err)
	}
	if !bytes.Equal(source, before) {
		t.Fatal("native ParseDocument() mutated source")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("native ParseDocument() mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBackendParseDocumentMatchesGoldmarkMultilineSetextHeading(t *testing.T) {
	t.Parallel()

	source := []byte("  Foo *bar\nbaz*\t\n====\n")
	want, err := goldmarkparser.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("Goldmark ParseDocument() error = %v", err)
	}
	got, err := native.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("native ParseDocument() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("native ParseDocument() mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBackendParseDocumentMatchesGoldmarkSetextHeadingVisibleBackslash(t *testing.T) {
	t.Parallel()

	source := []byte("Foo\\\n----\n")
	want, err := goldmarkparser.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("Goldmark ParseDocument() error = %v", err)
	}
	got, err := native.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("native ParseDocument() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("native ParseDocument() mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBackendParseDocumentMatchesGoldmarkEmptyBlockquote(t *testing.T) {
	t.Parallel()

	source := []byte(">")
	want, err := goldmarkparser.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("Goldmark ParseDocument() error = %v", err)
	}
	got, err := native.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("native ParseDocument() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("native ParseDocument() mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBackendParseDocumentMatchesGoldmarkBlockquoteLazyContinuationBoundaries(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte(">0000000\n*"),
		[]byte(">0000000\n2. item"),
		[]byte(">0000000\n2."),
		[]byte(">0000000\nplain"),
		[]byte(">0000000\n    indented"),
		[]byte(">0000000\n<span>"),
		[]byte(">0000000\n<div>"),
		[]byte(">0\r\t 0"),
		[]byte(">0\n\t 0"),
		[]byte(">0\r\n\t 0"),
		[]byte(">0\r    0"),
		[]byte(">0\r\t0"),
		[]byte(">0\r  0"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkCompactTaskMarker(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("* [X]0"),
		[]byte("* [X] "),
		[]byte("* [x]\t"),
		[]byte("* [ ]   "),
		[]byte("* [X] text  "),
		[]byte("* [X][]"),
		[]byte("* [X](u)"),
		[]byte("* [X][ref]\n\n[ref]: /target\n"),
		[]byte("* [X] [ref]\n\n[ref]: /target\n"),
		[]byte("* [X]\n0"),
		[]byte("* [X] \n0"),
		[]byte("* [X]\r\n0"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkEmptyNestedListBlankBoundary(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("* *\n\n  0"),
		[]byte("* -\n\n  0"),
		[]byte("* *\n  0"),
		[]byte("* a\n\n  0"),
		[]byte("* *\n\n   0"),
		[]byte("* *\n\n\t0"),
		[]byte("* *\n\n    0"),
		[]byte("* *\n\n  *"),
		[]byte("* *\n\n  -"),
	} {
		source := source
		t.Run(string(source), func(t *testing.T) {
			want, err := goldmarkparser.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
			}
			got, err := native.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("native ParseDocument(%q) error = %v", source, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
			}
		})
	}
}

func TestBackendParseDocumentMatchesGoldmarkListMarkerTabPadding(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("*\t  0"),
		[]byte("*\t 0"),
		[]byte("*\t0"),
		[]byte("*\t   0"),
		[]byte("* \t0"),
		[]byte("*  \t0"),
		[]byte("*   \t0"),
		[]byte("*    0"),
		[]byte("*     0"),
		[]byte("-\t  0"),
		[]byte("1.\t  0"),
	} {
		source := source
		t.Run(string(source), func(t *testing.T) {
			want, err := goldmarkparser.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
			}
			got, err := native.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("native ParseDocument(%q) error = %v", source, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
			}
		})
	}
}

func TestBackendParseDocumentMatchesGoldmarkMultilineReferenceUsages(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("[A]:0\n[\nA]"),
		[]byte("[A]:0\n![\nA]"),
		[]byte("[A]:0\n[ \nA ]"),
		[]byte("[A]:0\n[x\n][A]"),
		[]byte("[A]:0\n[x][\nA]"),
		[]byte("[A]:0\n[A\n][]"),
		[]byte("[A]:0\n[\rA]"),
		[]byte("[A]:0\r\n[\r\nA]"),
	} {
		source := source
		t.Run(string(source), func(t *testing.T) {
			want, err := goldmarkparser.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
			}
			got, err := native.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("native ParseDocument(%q) error = %v", source, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
			}
		})
	}
}

func TestBackendParseDocumentMatchesGoldmarkIncompleteReferenceTailFallback(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("[A]:0\n000[A]["),
		[]byte("[A]:0\n[A][missing"),
		[]byte("[A]:0\n![A]["),
		[]byte("[A]:0\n[A][missing]"),
		[]byte("[A]:0\n[A][]"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkEmptyLabelFullReferences(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("[A]:0\n[][A]0"),
		[]byte("[A]:0\n![][A]0"),
		[]byte("[A]:0\n[][A] []"),
		[]byte("[A]:0\n[][missing]"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkUnresolvedReferenceDelimiterBoundaries(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("[][*]*"),
		[]byte("[x][*]*"),
		[]byte("![][*]*"),
		[]byte("[][missing]"),
		[]byte("[x][missing]"),
		[]byte("[x][]"),
		[]byte("*[x][missing]*"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkReferenceDefinitionNextLineBoundary(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("[0]:\n#"),
		[]byte("[0]:\n## heading"),
		[]byte("[0]:\n-"),
		[]byte("[0]:\n- item"),
		[]byte("[0]:\n1. item"),
		[]byte("[0]:\n> quote"),
		[]byte("[0]:\n---"),
		[]byte("[0]:\n="),
		[]byte("[0]:\nfoo"),
		[]byte("[0]:\n<foo>"),
		[]byte("[0]:\n    foo"),
	} {
		source := source
		t.Run(string(source), func(t *testing.T) {
			want, err := goldmarkparser.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
			}
			got, err := native.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("native ParseDocument(%q) error = %v", source, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
			}
		})
	}
}

func TestBackendParseDocumentMatchesGoldmarkReferenceDefinitionMultilineLabelBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
	}{
		{name: "plain continuation", source: []byte("[a\nb]:0")},
		{name: "crlf continuation", source: []byte("[a\r\nb]:0")},
		{name: "isolated cr continuation", source: []byte("[a\rb]:0")},
		{name: "atx heading boundary", source: []byte("[a\n# b]:0")},
		{name: "bullet list boundary", source: []byte("[a\n- b]:0")},
		{name: "ordered list boundary", source: []byte("[a\n1. b]:0")},
		{name: "blockquote boundary", source: []byte("[a\n> b]:0")},
		{name: "thematic or setext boundary", source: []byte("[a\n---]:0")},
		{name: "setext equals boundary", source: []byte("[a\n=]:0")},
		{name: "invalid utf8 before heading boundary", source: []byte{'[', 0xf5, '\r', '0', '\n', '#', ' ', '0', '0', ']', ':', '0'}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := goldmarkparser.New().ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("Goldmark ParseDocument(%q) error = %v", tt.source, err)
			}
			got, err := native.New().ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("native ParseDocument(%q) error = %v", tt.source, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", tt.source, got, want)
			}
		})
	}
}

func TestBackendParseDocumentMatchesGoldmarkIndentedReferenceDefinitionRange(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("[0]:0"),
		[]byte(" [0]:0"),
		[]byte("  [0]:0"),
		[]byte("   [0]:0"),
		[]byte("    [0]:0"),
		[]byte("\t[0]:0"),
		[]byte(" [0]: <target> \"title\""),
	} {
		source := source
		t.Run(string(source), func(t *testing.T) {
			want, err := goldmarkparser.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
			}
			got, err := native.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("native ParseDocument(%q) error = %v", source, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
			}
		})
	}
}

func TestBackendParseDocumentMatchesGoldmarkReferenceDefinitionParagraphContinuation(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("[a]: first\n2. second"),
		[]byte("[a]: first\nplain"),
		[]byte("[a]: first\n1. second"),
		[]byte("[a]: first\n- item"),
		[]byte("[0]:0\n-"),
		[]byte("[0]:0\n+"),
		[]byte("[0]:0\n*"),
		[]byte("[0]:0\n1."),
		[]byte("[0]:0\n1)"),
		[]byte("[0]:0\n---"),
		[]byte("[0]:0\n="),
		[]byte("[0]:0\n--"),
	} {
		source := source
		t.Run(string(source), func(t *testing.T) {
			want, err := goldmarkparser.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
			}
			got, err := native.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("native ParseDocument(%q) error = %v", source, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
			}
		})
	}
}

func TestBackendParseDocumentMatchesGoldmarkReferenceDestinationWithOpenParenthesis(t *testing.T) {
	t.Parallel()

	source := []byte("[A]:0(00000")
	want, err := goldmarkparser.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("Goldmark ParseDocument() error = %v", err)
	}
	got, err := native.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("native ParseDocument() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("native ParseDocument() mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBackendParseDocumentMatchesGoldmarkReferenceDestinationWithNUL(t *testing.T) {
	t.Parallel()

	source := []byte{'[', '0', ']', ':', 0}
	want, err := goldmarkparser.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("Goldmark ParseDocument() error = %v", err)
	}
	got, err := native.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("native ParseDocument() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("native ParseDocument() mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBackendParseDocumentMatchesGoldmarkReferenceLabelClosingBracket(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("[00]]:00"),
		[]byte("[00]:00"),
		[]byte("[00\\]]:00"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkMalformedUTF8DelimiterNeighbors(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		{'~', 0x88, '~'},
		{'*', 0x80, '*'},
		{'_', 0xbf, '_'},
		{'~', 0xff, '~'},
		{'~', 0x9c, '~', '0', '~'},
		[]byte("~π~"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkSingleByteEmailDomainLabel(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("<000@0>0"),
		[]byte("<a@b>"),
		[]byte("<a@b-c>"),
		[]byte("<a@-b>"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkTerminalUnmatchedDelimiterProjection(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("~a*~ "),
		[]byte("~*a~ "),
		[]byte("~a*b~ "),
		[]byte("~a**~ "),
		[]byte("*a~*"),
		{'~', 0xcc, '*', '~', ' '},
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkBracketTextEmphasis(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("*[*"),
		[]byte("*[]*"),
		[]byte("*0[]*"),
		[]byte("*[]0*"),
		[]byte("*0[]1*"),
		[]byte("_0[]_"),
		[]byte("~~0[]~~"),
		[]byte("_[]_"),
		[]byte("~~[]~~"),
		[]byte("*[ ]*"),
		[]byte("*]*"),
		[]byte("*[a*"),
		[]byte("*[0*"),
		[]byte("*[bar*"),
		[]byte("*foo [bar* baz]"),
		[]byte("*foo ]bar*"),
		[]byte("*[x]*"),
		[]byte("*[x](u)*"),
		[]byte("~![~"),
		[]byte("*![*"),
		[]byte("_![_"),
		[]byte("~~![~~"),
		[]byte("~a![~"),
		[]byte("~![a~"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkBracketCloserAcrossDelimiter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
	}{
		{name: "empty label state closed after emphasis", source: []byte("*0[*]")},
		{name: "unclosed empty label state", source: []byte("*0[*")},
		{name: "empty label state after strong closer", source: []byte("**0[**]")},
		{name: "empty label state after strikethrough", source: []byte("~0[~]")},
		{name: "non-empty label state", source: []byte("*0[a*]")},
		{name: "label text before delimiter", source: []byte("*[a*]")},
		{name: "closing bracket not adjacent", source: []byte("*0[* x]")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := goldmarkparser.New().ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("Goldmark ParseDocument(%q) error = %v", tt.source, err)
			}
			got, err := native.New().ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("native ParseDocument(%q) error = %v", tt.source, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", tt.source, got, want)
			}
		})
	}
}

func TestBackendParseDocumentMatchesGoldmarkUnresolvedDelimiterText(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("*!**~* 000"),
		[]byte("*0 * 0*"),
		[]byte("_0 _ 0_"),
		[]byte("~0 ~ 0~"),
		[]byte("*0 **~*"),
		[]byte("*0 **a~*"),
		[]byte("*a**~*"),
		[]byte("_!**~_"),
		[]byte("~~!**_~~"),
		[]byte("*~~gone~~*"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkCrossingDelimiterOwnership(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("*~*~"),
		[]byte("~*~*"),
		[]byte("**~**~"),
		[]byte("~~*~~*"),
		[]byte("*~~x~~*"),
		[]byte("~**x**~"),
		[]byte("~!*~!*!*"),
		[]byte("*0 *~** "),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkAsymmetricStrikethrough(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("~~0~"),
		[]byte("~0~~"),
		[]byte("~0~"),
		[]byte("~~0~~"),
		[]byte("x ~~~0~~~"),
		[]byte("~0~~~0~"),
		[]byte("~~0~~~0~~"),
		[]byte("~0~~~~0~"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkEOFSingleByteFenceInfo(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("~~~0"),
		[]byte("~~~00"),
		[]byte("~~~0\n"),
		[]byte("~~~ 0"),
		[]byte("~~~0\t0"),
		[]byte("~~~0 0"),
		[]byte("~~~0\t0 1"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkMultilineCodeSpanProjection(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("` 0\n`000000000"),
		[]byte("` 0\n`"),
		[]byte("`` 0\n``"),
		[]byte("`  0\n`"),
		[]byte("`x\n`"),
		[]byte("`\t0\n`"),
		[]byte("` 0\n x`"),
		[]byte("` 0\n\n`"),
		[]byte("` 0\r\n`"),
		[]byte("` 0\r`"),
	} {
		source := source
		t.Run(string(source), func(t *testing.T) {
			want, err := goldmarkparser.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
			}
			got, err := native.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("native ParseDocument(%q) error = %v", source, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
			}
		})
	}
}

func TestBackendParseDocumentMatchesGoldmarkInlineDollarTextRunBoundaries(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("*$*$"),
		[]byte("$*$"),
		[]byte("*$x$*"),
		[]byte("$*x*$"),
		[]byte("~$x$~"),
		[]byte("[$x$](u)"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkMathematicalOverlay(t *testing.T) {
	t.Parallel()

	source := []byte("inline $x$ and $`code`$\n\n$$block$$\n\n[link]($hidden$) `also $hidden$`\n")
	want, err := goldmarkparser.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("Goldmark ParseDocument() error = %v", err)
	}
	got, err := native.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("native ParseDocument() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("native ParseDocument() mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBackendParseDocumentMatchesGoldmarkBlockDollarMathLineBoundaries(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("$$x$$"),
		[]byte("$$\n$$"),
		[]byte("$$\r\n$$"),
		[]byte("$$x\ny$$"),
		[]byte("before\n$$x$$\nafter"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkFootnoteOverlay(t *testing.T) {
	t.Parallel()

	source := []byte("before[^b] and again[^a] [outside](#out)\n\n[^a]: first [inside](b.md#part)\n\n    second paragraph\n\n[^unused]: unused\n[^b]: bee\n")
	want, err := goldmarkparser.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("Goldmark ParseDocument() error = %v", err)
	}
	got, err := native.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("native ParseDocument() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("native ParseDocument() mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBackendParseDocumentMatchesGoldmarkFootnoteContainerDefinitions(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("0) [^0]:000"),
		[]byte("0) [^0]:"),
		[]byte("- [^a]:"),
		[]byte("- [^a]: body"),
		[]byte("> [^a]: body"),
		[]byte("- > [^a]: body"),
		[]byte("- [^a]: first\n      second"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkFootnoteAndUnresolvedReferenceOverlap(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("[][^n]\n[^n]:0"),
		[]byte("[x][^n]\n[^n]:0"),
		[]byte("![x][^n]\n[^n]:0"),
		[]byte("[^n][]\n[^n]:0"),
		[]byte("[][^n]\n\n[^n]:0"),
		[]byte("[x][^n]\n\n[^n]:0"),
	} {
		source := source
		t.Run(string(source), func(t *testing.T) {
			want, err := goldmarkparser.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
			}
			got, err := native.New().ParseDocument(source)
			if err != nil {
				t.Fatalf("native ParseDocument(%q) error = %v", source, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
			}
		})
	}
}

func TestBackendParseDocumentMatchesGoldmarkFootnoteReferenceLikeBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
	}{
		{name: "recursive same label", source: []byte("[^0]:[^0]:")},
		{name: "recursive other label", source: []byte("[^0]:[^1]:")},
		{name: "reference token only", source: []byte("[^0]:[^1]")},
		{name: "spaced recursive label", source: []byte("[^0]: [^1]:")},
		{name: "indented recursive label", source: []byte("[^0]:\n    [^1]:")},
		{name: "indented recursive label with body", source: []byte("[^0]:\n    [^1]: child")},
		{name: "recursive label after text", source: []byte("[^0]: first\n    [^1]:")},
		{name: "recursive label after blank", source: []byte("[^0]: first\n\n    [^1]:")},
		{name: "ordinary reference definition body", source: []byte("[^0]:[a]:b")},
		{name: "lazy ordinary reference definition body", source: []byte("[^0]:0\n[0]:0")},
		{name: "lazy ordinary reference definition then token", source: []byte("[^0]:0\n[a]:b\n[a]")},
		{name: "ordinary reference usage to external definition", source: []byte("[a]: /target\n\n[^0]: use [a]")},
		{name: "plain bracket body", source: []byte("[^0]:[a]")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := goldmarkparser.New().ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("Goldmark ParseDocument(%q) error = %v", tt.source, err)
			}
			got, err := native.New().ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("native ParseDocument(%q) error = %v", tt.source, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", tt.source, got, want)
			}
		})
	}
}

func TestBackendParseDocumentMatchesGoldmarkFootnoteTrailingWhitespace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
	}{
		{name: "single trailing space", source: []byte("000000\n[^0]:0000000 ")},
		{name: "trailing tab", source: []byte("[^a]: body\t")},
		{name: "two trailing spaces", source: []byte("[^a]: body  ")},
		{name: "space tab suffix", source: []byte("[^a]: body \t")},
		{name: "continuation trailing space", source: []byte("[^a]: first\n    second ")},
		{name: "continuation two trailing spaces", source: []byte("[^a]: first\n    second  ")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := goldmarkparser.New().ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("Goldmark ParseDocument(%q) error = %v", tt.source, err)
			}
			got, err := native.New().ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("native ParseDocument(%q) error = %v", tt.source, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", tt.source, got, want)
			}
		})
	}
}

func TestBackendParseDocumentMatchesGoldmarkFootnoteLazyContinuation(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("[^0]:0\n0"),
		[]byte("[^0]: first\nsecond"),
		[]byte("[^0]: first\n2. second"),
		[]byte("[^0]: first\n# heading"),
		[]byte("[^0]: first\n- item"),
		[]byte("[^0]: first\n1. item"),
		[]byte("[^0]: first\n> quote"),
		[]byte("[^a]: first\n[^b]: second"),
		[]byte("[^a]: first\n [^b]: second"),
		[]byte("[^0]: first\n```\ncode\n```"),
		[]byte("[^0]: first\n---"),
		[]byte("[^0]: first\n\nsecond"),
		[]byte("[^0]:\nsecond"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkHeadingPartiallyConsumedDelimiterText(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("# *0`0**"),
		[]byte("# **0`0*"),
		[]byte("# *plain**"),
		[]byte("# **plain*"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkPartiallyConsumedDelimiterRuns(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("*!***!* 00"),
		[]byte("*a***b*"),
		[]byte("_!___!_ 00"),
	} {
		want, err := goldmarkparser.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("Goldmark ParseDocument(%q) error = %v", source, err)
		}
		got, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("native ParseDocument(%q) error = %v", source, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", source, got, want)
		}
	}
}

func TestBackendParseDocumentMatchesGoldmarkUnmatchedBacktickDelimiterBarrier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
	}{
		{name: "fuzz regression", source: []byte("# *00000`0*")},
		{name: "unmatched run starts emphasis payload", source: []byte("0 *`*")},
		{name: "unmatched run starts longer emphasis payload", source: []byte("*`a*")},
		{name: "unmatched run follows payload text", source: []byte("*a`*")},
		{name: "unmatched run starts strikethrough payload", source: []byte("~~`~~")},
		{name: "unmatched run follows strikethrough payload text", source: []byte("~~a`~~")},
		{name: "crossing emphasis", source: []byte("*a`b*")},
		{name: "crossing underscore emphasis", source: []byte("_a`b_")},
		{name: "crossing strikethrough", source: []byte("~~a`b~~")},
		{name: "emphasis after unmatched run", source: []byte("` *a*")},
		{name: "strikethrough after unmatched run", source: []byte("` ~~a~~")},
		{name: "matched code inside emphasis", source: []byte("*a`b`*")},
		{name: "different unmatched run length", source: []byte("*a``b*")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := goldmarkparser.New().ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("Goldmark ParseDocument(%q) error = %v", tt.source, err)
			}
			got, err := native.New().ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("native ParseDocument(%q) error = %v", tt.source, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("native ParseDocument(%q) mismatch\n got: %#v\nwant: %#v", tt.source, got, want)
			}
		})
	}
}

func TestBackendConstructionProofAcceptanceMatchesGoldmark(t *testing.T) {
	t.Parallel()

	t.Run("blockquote paragraph", func(t *testing.T) {
		source := []byte("# Title\n\n> > first *line*\n> > second π\n\nTail.\n")
		start := bytes.Index(source, []byte("> > first *line*"))
		firstStart := start + 4
		firstEnd := firstStart + len("first *line*")
		secondStart := firstEnd + 1 + 4
		secondEnd := secondStart + len("second π")
		outer := parser.Range{Start: start, End: secondEnd}
		lines := []parser.Range{{Start: firstStart, End: firstEnd}, {Start: secondStart, End: secondEnd}}
		compareErrors(t,
			func() error { return goldmarkparser.New().ValidateNestedBlockquoteParagraph(source, outer, lines, 2) },
			func() error { return native.New().ValidateNestedBlockquoteParagraph(source, outer, lines, 2) },
		)
	})

	t.Run("blockquote blocks", func(t *testing.T) {
		inner := []byte("## Head\n\n- parent\n  - child\n\n```go\nx\n```\n")
		source := []byte("> > ## Head\n> > \n> > - parent\n> >   - child\n> > \n> > ```go\n> > x\n> > ```\n")
		outer := parser.Range{Start: 0, End: len(source) - 1}
		compareErrors(t,
			func() error { return goldmarkparser.New().ValidateNestedBlockquoteBlocks(source, outer, inner, 2) },
			func() error { return native.New().ValidateNestedBlockquoteBlocks(source, outer, inner, 2) },
		)
	})

	t.Run("inline hierarchy", func(t *testing.T) {
		source := []byte("*before ``a`b`` after* **_inside_** *~~gone~~*")
		expected := []parser.ConstructionInlineExpectation{
			inlineExpectation(source, "*before ``a`b`` after*", parser.KindEmphasis, '*', 1, -1),
			inlineExpectation(source, "``a`b``", parser.KindCodeSpan, '`', 2, 0),
			inlineExpectation(source, "**_inside_**", parser.KindStrong, '*', 2, -1),
			inlineExpectation(source, "_inside_", parser.KindEmphasis, '_', 1, 2),
			inlineExpectation(source, "*~~gone~~*", parser.KindEmphasis, '*', 1, -1),
			inlineExpectation(source, "~~gone~~", parser.KindStrikethrough, '~', 2, 4),
		}
		compareErrors(t,
			func() error { return goldmarkparser.New().ValidateConstructionInlineHierarchy(source, expected, nil) },
			func() error { return native.New().ValidateConstructionInlineHierarchy(source, expected, nil) },
		)
	})

	t.Run("unmatched backtick emphasis", func(t *testing.T) {
		source := []byte("*a`b*")
		expected := []parser.ConstructionInlineExpectation{
			inlineExpectation(source, string(source), parser.KindEmphasis, '*', 1, -1),
		}
		compareErrors(t,
			func() error { return goldmarkparser.New().ValidateConstructionInlineHierarchy(source, expected, nil) },
			func() error { return native.New().ValidateConstructionInlineHierarchy(source, expected, nil) },
		)
	})

	t.Run("direct link image", func(t *testing.T) {
		source := []byte("[**docs**](<target> \"Guide\") ![*logo*](<image.png>)")
		expected := []parser.ConstructionLinkImageExpectation{
			linkImageExpectation(source, "[**docs**](<target> \"Guide\")", "**docs**", parser.KindInlineLink, "target", "Guide", true),
			linkImageExpectation(source, "![*logo*](<image.png>)", "*logo*", parser.KindImage, "image.png", "", false),
		}
		compareErrors(t,
			func() error { return goldmarkparser.New().ValidateConstructionLinkImages(source, expected) },
			func() error { return native.New().ValidateConstructionLinkImages(source, expected) },
		)
	})

	t.Run("reference inline", func(t *testing.T) {
		source := []byte("[docs][ref] and ![logo][img]")
		expected := []parser.ConstructionReferenceInlineExpectation{
			referenceExpectation(source, "[docs][ref]", "docs", "ref", parser.KindInlineLink, "https://example.test", "Guide", true),
			referenceExpectation(source, "![logo][img]", "logo", "img", parser.KindImage, "images/logo.png", "", false),
		}
		compareErrors(t,
			func() error { return goldmarkparser.New().ValidateConstructionReferenceInlines(source, expected) },
			func() error { return native.New().ValidateConstructionReferenceInlines(source, expected) },
		)
	})
}

func TestBackendConstructionProofRejectionMatchesGoldmark(t *testing.T) {
	t.Parallel()

	t.Run("blockquote depth mismatch", func(t *testing.T) {
		source := []byte("> > paragraph\n")
		outer := parser.Range{Start: 0, End: len(source) - 1}
		contentStart := len("> > ")
		content := []parser.Range{{Start: contentStart, End: len(source) - 1}}
		compareErrors(t,
			func() error { return goldmarkparser.New().ValidateNestedBlockquoteParagraph(source, outer, content, 1) },
			func() error { return native.New().ValidateNestedBlockquoteParagraph(source, outer, content, 1) },
		)
	})

	t.Run("inline parent mismatch", func(t *testing.T) {
		source := []byte("*outer `code`*")
		expected := []parser.ConstructionInlineExpectation{
			inlineExpectation(source, "*outer `code`*", parser.KindEmphasis, '*', 1, -1),
			inlineExpectation(source, "`code`", parser.KindCodeSpan, '`', 1, -1),
		}
		compareErrors(t,
			func() error { return goldmarkparser.New().ValidateConstructionInlineHierarchy(source, expected, nil) },
			func() error { return native.New().ValidateConstructionInlineHierarchy(source, expected, nil) },
		)
	})

	t.Run("direct destination mismatch", func(t *testing.T) {
		source := []byte("[docs](<target>)")
		expected := []parser.ConstructionLinkImageExpectation{
			linkImageExpectation(source, string(source), "docs", parser.KindInlineLink, "other", "", false),
		}
		compareErrors(t,
			func() error { return goldmarkparser.New().ValidateConstructionLinkImages(source, expected) },
			func() error { return native.New().ValidateConstructionLinkImages(source, expected) },
		)
	})

	t.Run("structured reference declared plain", func(t *testing.T) {
		source := []byte("[**docs** `v1`][ref]")
		expected := referenceExpectation(source, string(source), "**docs** `v1`", "ref", parser.KindInlineLink, "target", "Guide", true)
		compareErrors(t,
			func() error {
				return goldmarkparser.New().ValidateConstructionReferenceInlines(source, []parser.ConstructionReferenceInlineExpectation{expected})
			},
			func() error {
				return native.New().ValidateConstructionReferenceInlines(source, []parser.ConstructionReferenceInlineExpectation{expected})
			},
		)
	})

	t.Run("conflicting reference semantics", func(t *testing.T) {
		source := []byte("[a][ref] [b][ref]")
		first := referenceExpectation(source, "[a][ref]", "a", "ref", parser.KindInlineLink, "first", "", false)
		second := referenceExpectation(source, "[b][ref]", "b", "ref", parser.KindInlineLink, "second", "", false)
		compareErrors(t,
			func() error {
				return goldmarkparser.New().ValidateConstructionReferenceInlines(source, []parser.ConstructionReferenceInlineExpectation{first, second})
			},
			func() error {
				return native.New().ValidateConstructionReferenceInlines(source, []parser.ConstructionReferenceInlineExpectation{first, second})
			},
		)
	})
}

func TestBackendConstructionStructuredReferenceAcceptanceMatchesGoldmark(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("[**docs** `v1`][ref]"),
		[]byte("[docs][ref]"),
	} {
		label := string(source[1:bytes.IndexByte(source, ']')])
		expected := referenceExpectation(source, string(source), label, "ref", parser.KindInlineLink, "target", "Guide", true)
		expected.StructuredLabel = true
		compareErrors(t,
			func() error {
				return goldmarkparser.New().ValidateConstructionReferenceInlines(source, []parser.ConstructionReferenceInlineExpectation{expected})
			},
			func() error {
				return native.New().ValidateConstructionReferenceInlines(source, []parser.ConstructionReferenceInlineExpectation{expected})
			},
		)
	}
}

func TestBackendReferenceOperationsMatchGoldmark(t *testing.T) {
	t.Parallel()

	backend := native.New()
	labels := []string{"  A  B\tC ", "Straße", "STRASSE", "İ", "\u212a", "a\nb"}
	for _, label := range labels {
		if got, want := backend.ReferenceLabelKey(label), goldmarkparser.ReferenceLabelKey(label); got != want {
			t.Fatalf("ReferenceLabelKey(%q) = %q, want %q", label, got, want)
		}
	}

	definitions := []parser.ConstructionReferenceDefinition{
		{Label: "Straße", Destination: "one"},
		{Label: "other", Destination: "two"},
	}
	got, gotErr := backend.ResolveConstructionReference("STRASSE", definitions)
	want, wantErr := goldmarkparser.ResolveConstructionReference("STRASSE", definitions)
	if (gotErr != nil) != (wantErr != nil) || !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveConstructionReference() = %#v, %v; want %#v, %v", got, gotErr, want, wantErr)
	}

	definitions = append(definitions, parser.ConstructionReferenceDefinition{Label: "STRASSE", Destination: "duplicate"})
	compareErrors(t,
		func() error {
			_, err := goldmarkparser.ResolveConstructionReference("straße", definitions)
			return err
		},
		func() error { _, err := backend.ResolveConstructionReference("straße", definitions); return err },
	)
}

func compareErrors(t *testing.T, goldmarkCall, nativeCall func() error) {
	t.Helper()
	goldmarkErr := goldmarkCall()
	nativeErr := nativeCall()
	if (nativeErr != nil) != (goldmarkErr != nil) {
		t.Fatalf("acceptance mismatch: native error = %v; Goldmark error = %v", nativeErr, goldmarkErr)
	}
}

func inlineExpectation(source []byte, token string, kind parser.Kind, marker byte, delimiterLength, parent int) parser.ConstructionInlineExpectation {
	start := strings.Index(string(source), token)
	end := start + len(token)
	return parser.ConstructionInlineExpectation{
		Kind: kind, SyntaxRange: parser.Range{Start: start, End: end},
		ContentRange: parser.Range{Start: start + delimiterLength, End: end - delimiterLength},
		Marker:       marker, DelimiterLength: delimiterLength, Parent: parent,
	}
}

func linkImageExpectation(source []byte, token, label string, kind parser.Kind, destination, title string, hasTitle bool) parser.ConstructionLinkImageExpectation {
	start := strings.Index(string(source), token)
	prefix := 1
	if kind == parser.KindImage {
		prefix = 2
	}
	labelStart := start + prefix
	return parser.ConstructionLinkImageExpectation{
		Kind: kind, SyntaxRange: parser.Range{Start: start, End: start + len(token)},
		LabelRange:  parser.Range{Start: labelStart, End: labelStart + len(label)},
		Destination: destination, Title: title, HasTitle: hasTitle,
	}
}

func referenceExpectation(source []byte, token, label, reference string, kind parser.Kind, destination, title string, hasTitle bool) parser.ConstructionReferenceInlineExpectation {
	start := strings.Index(string(source), token)
	prefix := 1
	if kind == parser.KindImage {
		prefix = 2
	}
	labelStart := start + prefix
	labelEnd := labelStart + len(label)
	referenceStart := labelEnd + 2
	return parser.ConstructionReferenceInlineExpectation{
		Kind: kind, Form: parser.ConstructionReferenceInlineFull,
		SyntaxRange:    parser.Range{Start: start, End: start + len(token)},
		LabelRange:     parser.Range{Start: labelStart, End: labelEnd},
		ReferenceRange: parser.Range{Start: referenceStart, End: referenceStart + len(reference)},
		Reference:      reference, Destination: destination, Title: title, HasTitle: hasTitle,
	}
}
