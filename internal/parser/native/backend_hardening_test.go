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

func TestM114CommonMark0312InlineHTMLGrammar(t *testing.T) {
	backend := native.New()
	tests := []struct {
		name    string
		source  []byte
		wantRaw bool
	}{
		{name: "processing instruction opener cannot overlap closer", source: []byte("0<?>"), wantRaw: false},
		{name: "processing instruction with explicit closer", source: []byte("0<??>"), wantRaw: true},
		{name: "lowercase declaration", source: []byte("0<!a0>"), wantRaw: true},
		{name: "short comment", source: []byte("0<!-->"), wantRaw: true},
		{name: "short hyphen comment", source: []byte("0<!--->"), wantRaw: true},
		{name: "comment may contain double hyphen", source: []byte("0<!-- a--b -->"), wantRaw: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observed, err := backend.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			gotRaw := false
			for _, node := range observed.Nodes {
				gotRaw = gotRaw || node.Kind == parser.KindRawHTML
			}
			if gotRaw != tt.wantRaw {
				t.Fatalf("RawHTML presence = %v, want %v; observations = %#v", gotRaw, tt.wantRaw, observed.Nodes)
			}
		})
	}
}

func TestM114PublishedGFMAutolinkGrammar(t *testing.T) {
	backend := native.New()
	tests := []struct {
		name      string
		source    []byte
		wantValue string
		wantEmail bool
		wantLink  bool
	}{
		{name: "www suffix still requires a period", source: []byte("www.000"), wantLink: false},
		{name: "numeric final URL segment is valid", source: []byte("www.example.000"), wantValue: "www.example.000", wantLink: true},
		{name: "uppercase final URL segment is valid", source: []byte("www.example.COM"), wantValue: "www.example.COM", wantLink: true},
		{name: "mixed alphanumeric final URL segment is valid", source: []byte("www.example.a00"), wantValue: "www.example.a00", wantLink: true},
		{name: "numeric final HTTP segment is valid", source: []byte("http://example.000"), wantValue: "http://example.000", wantLink: true},
		{name: "numeric HTTP domain is valid", source: []byte("http://0.0"), wantValue: "http://0.0", wantLink: true},
		{name: "email domain permits underscore", source: []byte("a@a_b.0"), wantValue: "a@a_b.0", wantEmail: true, wantLink: true},
		{name: "terminal underscore invalidates complete email", source: []byte("0@0.0_"), wantLink: false},
		{name: "terminal hyphen invalidates complete email", source: []byte("0@0.0-"), wantLink: false},
		{name: "terminal dot is excluded from otherwise valid email", source: []byte("0@0.0."), wantValue: "0@0.0", wantEmail: true, wantLink: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observed, err := backend.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			var links []parser.Node
			for _, node := range observed.Nodes {
				if node.Kind == parser.KindAutoLink {
					links = append(links, node)
				}
			}
			if !tt.wantLink {
				if len(links) != 0 {
					t.Fatalf("AutoLink nodes = %#v, want none", links)
				}
				return
			}
			if len(links) != 1 || links[0].Value != tt.wantValue || links[0].AutoLinkEmail != tt.wantEmail {
				t.Fatalf("AutoLink nodes = %#v, want value %q email=%v", links, tt.wantValue, tt.wantEmail)
			}
		})
	}
}

func TestM114NativeBackendPathologicalInputsRemainSourceBound(t *testing.T) {
	backend := native.New()
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "one mebibyte paragraph", source: []byte(strings.Repeat("x", 1<<20) + "\n")},
		{name: "one mebibyte unclosed fence", source: []byte("```text\n" + strings.Repeat("x", 1<<20))},
		{name: "deep blockquote", source: []byte(strings.Repeat("> ", 4096) + "payload\n")},
		{name: "dense delimiters", source: []byte(strings.Repeat("*_~`", 64<<10) + "\n")},
		{name: "malformed link storm", source: []byte(strings.Repeat("[broken]( ", 4096) + "\n")},
		{name: "dense direct links", source: []byte(strings.Repeat("[label](<target>) ", 4096) + "\n")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := backend.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(tt.source, before) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := backend.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkPathologicalSubset(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "large paragraph", source: []byte(strings.Repeat("x", 64<<10) + "\n")},
		{name: "deep blockquote", source: []byte(strings.Repeat("> ", 1024) + "payload\n")},
		{name: "dense delimiters", source: []byte(strings.Repeat("*_~`", 16<<10) + "\n")},
		{name: "malformed links", source: []byte(strings.Repeat("[broken]( ", 1024) + "\n")},
	}
	oracle := goldmarkparser.New()
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, wantErr := oracle.ParseDocument(tt.source)
			got, gotErr := candidate.ParseDocument(tt.source)
			if (gotErr != nil) != (wantErr != nil) {
				t.Fatalf("error parity mismatch: native=%v Goldmark=%v", gotErr, wantErr)
			}
			if gotErr == nil && !reflect.DeepEqual(got, want) {
				t.Fatal("pathological backend observations differ from Goldmark")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkBracketedDelimiterContent(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "opening bracket at content start closes after emphasis", source: []byte("__[*_ 0]")},
		{name: "opening bracket only content closes after emphasis", source: []byte("__[_ 0]")},
		{name: "opening bracket at content end closes after emphasis", source: []byte("__a[_ 0]")},
		{name: "opening bracket before text closes after emphasis", source: []byte("__[a_ 0]")},
		{name: "opening bracket without later close", source: []byte("__[*_ 0")},
		{name: "opening bracket closes immediately after emphasis", source: []byte("__[*_]")},
		{name: "double closer run", source: []byte("__[*__ 0]")},
		{name: "three-character opener run", source: []byte("___[*_ 0]")},
		{name: "different marker does not pair", source: []byte("**[*_ 0]")},
		{name: "single underscore around bracket star", source: []byte("_0[*_ ]")},
		{name: "single underscore around bracket text", source: []byte("_0[a_ ]")},
		{name: "single underscore around trailing bracket", source: []byte("_0[_ ]")},
		{name: "single underscore immediate bracket close", source: []byte("_0[*_]")},
		{name: "single underscore bracket closes after text", source: []byte("_0[*_ x]")},
		{name: "single underscore bracket closes after isolated cr", source: []byte("_[*_ \r]")},
		{name: "single underscore bracket closes after lf", source: []byte("_[*_ \n]")},
		{name: "single underscore bracket closes after crlf", source: []byte("_[*_ \r\n]")},
		{name: "single underscore bracket closes after isolated cr text", source: []byte("_[*_ \r x]")},
		{name: "single star bracket closes after isolated cr", source: []byte("*[_* \r]")},
		{name: "single underscore bracket remains open after isolated cr", source: []byte("_[*_ \r")},
		{name: "single underscore bracket remains open after lf", source: []byte("_[*_ \n")},
		{name: "single underscore bracket never closes", source: []byte("_0[*_")},
		{name: "single underscore double star inside bracket", source: []byte("_0[**_ ]")},
		{name: "single star around bracket underscore", source: []byte("*0[_* ]")},
		{name: "single underscore escaped star in bracket", source: []byte("_0[\\*_ ]")},
		{name: "single underscore crossing direct link", source: []byte("_0[*_](x)")},
		{name: "single underscore crossing reference link", source: []byte("_0[*_][r]\n[r]:x")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkInlineHTMLDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "artifact declaration without whitespace", source: []byte("0<!A0>")},
		{name: "minimal declaration", source: []byte("<!A>")},
		{name: "declaration after text", source: []byte("0<!A>")},
		{name: "declaration with body text", source: []byte("<!ABC foo>")},
		{name: "declaration with digit body", source: []byte("<!A0>")},
		{name: "declaration with punctuation body", source: []byte("<!A->")},
		{name: "declaration across lf", source: []byte("0<!A\n0>")},
		{name: "declaration across isolated cr", source: []byte("0<!A\r0>")},
		{name: "declaration across crlf", source: []byte("0<!A\r\n0>")},
		{name: "declaration across lf owns delimiters", source: []byte("0<!A\n*0*>")},
		{name: "declaration across isolated cr owns delimiters", source: []byte("0<!A\r*0*>")},
		{name: "declaration across crlf owns delimiters", source: []byte("0<!A\r\n*0*>")},
		{name: "declaration across two lines owns delimiters", source: []byte("0<!A\n*0*\n>")},
		{name: "exact fuzz artifact", source: []byte("00000000000000000000000000<!A\r*0*>")},
		{name: "lowercase multiline prefix does not own delimiters", source: []byte("0<!a\r*0*>")},
		{name: "declaration closes on next line", source: []byte("0<!A\n>")},
		{name: "lowercase declaration prefix is not declaration", source: []byte("0<!a0>")},
		{name: "digit declaration prefix is not declaration", source: []byte("0<!0>")},
		{name: "empty declaration prefix is not declaration", source: []byte("0<!>")},
		{name: "unclosed declaration", source: []byte("0<!A")},
		{name: "cdata remains independent", source: []byte("0<![CDATA[x]]>")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkExtendedWWWAutolinks(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "exact round 54 artifact", source: []byte("www.000")},
		{name: "alphabetic tld", source: []byte("www.com")},
		{name: "single alphabetic tld", source: []byte("www.a")},
		{name: "mixed tld starts digit", source: []byte("www.0a0")},
		{name: "mixed tld starts letter", source: []byte("www.a00")},
		{name: "mixed tld ends letter", source: []byte("www.000a")},
		{name: "mixed tld ends digit", source: []byte("www.com0")},
		{name: "numeric intermediate label", source: []byte("www.000.com")},
		{name: "numeric final label after name", source: []byte("www.example.000")},
		{name: "www mixed tld starts digit", source: []byte("www.example.0a0")},
		{name: "www mixed tld starts letter", source: []byte("www.example.a00")},
		{name: "www mixed tld ends letter", source: []byte("www.example.000a")},
		{name: "www mixed tld ends digit", source: []byte("www.example.com0")},
		{name: "www single letter final tld", source: []byte("www.example.a")},
		{name: "www two letter final tld", source: []byte("www.example.ab")},
		{name: "www uppercase final tld", source: []byte("www.example.COM")},
		{name: "numeric two-level domain", source: []byte("www.0.0")},
		{name: "http ordinary domain", source: []byte("http://example.com")},
		{name: "http single letter final tld", source: []byte("http://example.a")},
		{name: "http two letter final tld", source: []byte("http://example.ab")},
		{name: "http uppercase final tld", source: []byte("http://example.COM")},
		{name: "http numeric tld", source: []byte("http://example.000")},
		{name: "http mixed tld starts digit", source: []byte("http://example.0a0")},
		{name: "http mixed tld starts letter", source: []byte("http://example.a00")},
		{name: "http mixed tld ends letter", source: []byte("http://example.000a")},
		{name: "http mixed tld ends digit", source: []byte("http://example.com0")},
		{name: "http numeric host", source: []byte("http://0.0")},
		{name: "https numeric host", source: []byte("https://0.0")},
		{name: "http www numeric tld", source: []byte("http://www.000")},
		{name: "www path after numeric tld", source: []byte("www.000/path")},
		{name: "www punctuation after numeric tld", source: []byte("www.000,")},
		{name: "www alphabetic tld path", source: []byte("www.example.com/path")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkPartiallyConsumedDelimiterRuleOfThree(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "exact round 55 artifact", source: []byte("*0 **0*~*~")},
		{name: "no residual opener", source: []byte("*0 *0*~*~")},
		{name: "double tilde intermediate", source: []byte("*0 **0*~~*~~")},
		{name: "residual opener with text before crossing closer", source: []byte("*0 **0*~x*~")},
		{name: "underscore analogue", source: []byte("_0 __0_~_~")},
		{name: "partially consumed star control", source: []byte("*a***b*")},
		{name: "partially consumed underscore control", source: []byte("_!___!_ 00")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114CommonMark0312RuleOfThreeUsesOriginalDelimiterRunLength(t *testing.T) {
	source := []byte("*0 **0*~*~")
	observed, err := native.New().ParseDocument(source)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}
	for _, node := range observed.Nodes {
		if node.Kind == parser.KindStrikethrough {
			t.Fatalf("ParseDocument() projected spurious strikethrough for %q: %+v", source, node)
		}
	}
}

func TestM114NativeBackendMatchesGoldmarkInlineHTMLProcessingInstructions(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "exact round 53 artifact", source: []byte("0<?>")},
		{name: "minimal overlapping close", source: []byte("<?>")},
		{name: "explicit empty body", source: []byte("<??>")},
		{name: "ordinary body", source: []byte("<?x?>")},
		{name: "ordinary body after text", source: []byte("0<?x?>")},
		{name: "artifact with trailing text", source: []byte("0<?>x")},
		{name: "unclosed prefix", source: []byte("0<?")},
		{name: "body without question close", source: []byte("0<?x>")},
		{name: "space without question close", source: []byte("0<? >")},
		{name: "across lf", source: []byte("0<?x\n?>")},
		{name: "across isolated cr", source: []byte("0<?x\r?>")},
		{name: "across crlf", source: []byte("0<?x\r\n?>")},
		{name: "across lf owns delimiters", source: []byte("0<?x\n*0*?>")},
		{name: "across isolated cr owns delimiters", source: []byte("0<?x\r*0*?>")},
		{name: "across crlf owns delimiters", source: []byte("0<?x\r\n*0*?>")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkMixedLineEndings(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "isolated cr before crlf", source: []byte("\r*\r\n")},
		{name: "isolated cr before lf", source: []byte("\r*\n")},
		{name: "leading lf before crlf", source: []byte("\n*\r\n")},
		{name: "leading crlf", source: []byte("\r\n*\r\n")},
		{name: "bare bullet lf", source: []byte("*\n")},
		{name: "bare bullet crlf", source: []byte("*\r\n")},
		{name: "bare dash crlf", source: []byte("-\r\n")},
		{name: "bare plus crlf", source: []byte("+\r\n")},
		{name: "bare ordered crlf", source: []byte("1.\r\n")},
		{name: "spaced bullet crlf", source: []byte("* \r\n")},
		{name: "spaced dash crlf", source: []byte("- \r\n")},
		{name: "spaced ordered crlf", source: []byte("1. \r\n")},
		{name: "isolated cr terminator", source: []byte("*\r")},
		{name: "two isolated cr boundaries", source: []byte("\r\r*\r")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114M104NestedFootnoteDefinitionAmbiguityFailsClosed(t *testing.T) {
	backend := native.New()
	tests := []struct {
		name   string
		source []byte
		want   int
	}{
		{name: "round 58 ordered container definition", source: []byte("[^0]:0) [^0]:")},
		{name: "bullet nested definition", source: []byte("[^0]:- [^1]:")},
		{name: "direct nested definition", source: []byte("[^0]: [^1]:")},
		{name: "ordered reference token is ordinary body", source: []byte("[^0]:0) [^0]"), want: 1},
		{name: "plain body", source: []byte("[^0]:0) plain"), want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := backend.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if len(got.FootnoteDefinitions) != tt.want {
				t.Fatalf("footnote definitions = %+v, want %d", got.FootnoteDefinitions, tt.want)
			}
		})
	}
}
func TestM114M104FootnoteDefinitionsRemainTopLevel(t *testing.T) {
	backend := native.New()
	tests := []struct {
		name   string
		source []byte
		want   int
	}{
		{name: "top level", source: []byte("[^a]: note\n"), want: 1},
		{name: "three leading spaces", source: []byte("   [^a]: note\n"), want: 1},
		{name: "blockquote", source: []byte("> [^a]: note\n")},
		{name: "bullet list", source: []byte("- [^a]: note\n")},
		{name: "ordered list", source: []byte("1. [^a]: note\n")},
		{name: "nested containers", source: []byte("- > [^a]: note\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := backend.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if len(got.FootnoteDefinitions) != tt.want {
				t.Fatalf("footnote definitions = %+v, want %d", got.FootnoteDefinitions, tt.want)
			}
		})
	}
}
func TestM114M104FootnoteCaretPrecedence(t *testing.T) {
	backend := native.New()
	tests := []struct {
		name             string
		source           []byte
		wantFootnoteRefs int
	}{
		{name: "original M104 caret conflict", source: []byte("foot[^n] [normal][^n]\n\n[^n]: note\n"), wantFootnoteRefs: 2},
		{name: "round 59 multiline conflict", source: []byte("[^n]\n\n[^n]:\n0"), wantFootnoteRefs: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := backend.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if len(got.FootnoteDefinitions) != 1 || len(got.FootnoteReferences) != tt.wantFootnoteRefs {
				t.Fatalf("footnote observations = definitions %+v references %+v", got.FootnoteDefinitions, got.FootnoteReferences)
			}
			for _, usage := range got.LinkUsages {
				if usage.Reference == "^n" {
					t.Fatalf("caret source leaked as ordinary GFM relationship: %+v", usage)
				}
			}
			for _, usage := range got.UnresolvedReferenceUsages {
				if usage.Reference == "^n" {
					t.Fatalf("caret source leaked as unresolved GFM relationship: %+v", usage)
				}
			}
		})
	}
}
func TestM114NativeBackendMatchesGoldmarkFootnoteReferenceOverlayProjection(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "exact round 59 multiline destination", source: []byte("[^n]\n\n[^n]:\n0")},
		{name: "multiline slash destination", source: []byte("[^n]\n\n[^n]:\n/target")},
		{name: "multiline destination full reference", source: []byte("[text][^n]\n\n[^n]:\n/target")},
		{name: "multiline destination crlf", source: []byte("[^n]\r\n\r\n[^n]:\r\n0")},
		{name: "same line destination is suppressed", source: []byte("[^n]\n\n[^n]: /target")},
		{name: "same line destination full reference suppressed", source: []byte("[text][^n]\n\n[^n]: /target")},
		{name: "ordinary reference remains", source: []byte("[^n] [ok][docs]\n\n[^n]: note\n\n[docs]: /target")},
		{name: "definition without gfm destination", source: []byte("[^n]\n\n[^n]:")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeNestedFootnoteInputsRemainSourceBound(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "exact round 58 artifact", source: []byte("[^0]:0) [^0]:")},
		{name: "ordered nested other label", source: []byte("[^0]:0) [^1]:")},
		{name: "ordered nested definition with body", source: []byte("[^0]:0) [^1]: child")},
		{name: "dot ordered nested definition", source: []byte("[^0]:1. [^1]:")},
		{name: "bullet nested definition", source: []byte("[^0]:- [^1]:")},
		{name: "bullet nested definition with body", source: []byte("[^0]:- [^1]: child")},
		{name: "ordered reference token remains valid", source: []byte("[^0]:0) [^0]")},
		{name: "ordered plain body remains valid", source: []byte("[^0]:0) x")},
		{name: "immediate plain body remains valid", source: []byte("[^0]:x")},
		{name: "direct nested definition remains rejected", source: []byte("[^0]: [^1]:")},
		{name: "top level ordered container definition remains valid", source: []byte("0) [^0]:")},
		{name: "top level bullet container definition remains valid", source: []byte("- [^0]: body")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkFootnoteLabelWhitespace(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "empty", source: []byte("[^]:")},
		{name: "space", source: []byte("[^ ]:")},
		{name: "tab", source: []byte("[^\t]:")},
		{name: "vertical tab", source: []byte{'[', '^', '\v', ']', ':'}},
		{name: "form feed", source: []byte{'[', '^', '\f', ']', ':'}},
		{name: "spaces around form feed", source: []byte{'[', '^', ' ', '\f', ' ', ']', ':'}},
		{name: "non breaking space", source: []byte("[^\u00a0]:")},
		{name: "embedded space", source: []byte("[^a b]:")},
		{name: "nul", source: []byte{'[', '^', 0, ']', ':'}},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkLooseTaskMarkers(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "tight terminal marker", source: []byte("- [X]")},
		{name: "tight following item", source: []byte("- [X]\n- next")},
		{name: "loose blank item", source: []byte("- [X]\n\n-")},
		{name: "loose nonempty item", source: []byte("- [X]\n\n- next")},
		{name: "loose same item continuation", source: []byte("- [X]\n\n  continuation")},
		{name: "loose marker with reference definition", source: []byte("- [X]\n\n-\n\n[X]: /target")},
		{name: "tight marker with trailing space", source: []byte("- [X] \n- next")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkReferenceLabelWhitespace(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "empty", source: []byte("[]:0")},
		{name: "space", source: []byte("[ ]:0")},
		{name: "tab", source: []byte("[\t]:0")},
		{name: "vertical tab", source: []byte{'[', '\v', ']', ':', '0'}},
		{name: "form feed", source: []byte{'[', '\f', ']', ':', '0'}},
		{name: "spaces around vertical tab", source: []byte{'[', ' ', '\v', ' ', ']', ':', '0'}},
		{name: "non breaking space", source: []byte("[\u00a0]:0")},
		{name: "embedded space", source: []byte("[a b]:0")},
		{name: "nul", source: []byte{'[', 0, ']', ':', '0'}},
		{name: "line break only", source: []byte("[\n]:0")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkExtendedEmailTrailingDots(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "no trailing dot", source: []byte("0@00.000")},
		{name: "one trailing dot", source: []byte("0@00.000.")},
		{name: "two trailing dots", source: []byte("0@00.000..")},
		{name: "three trailing dots", source: []byte("0@00.000...")},
		{name: "trailing dots before comma", source: []byte("0@00.000..,")},
		{name: "internal empty domain label", source: []byte("0@00..000")},
		{name: "domain only dots", source: []byte("0@...")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkExtendedEmailDomainSuffixes(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "exact round 51 artifact", source: []byte("0@0.0._")},
		{name: "terminal underscore", source: []byte("0@0.0_")},
		{name: "terminal hyphen", source: []byte("0@0.0-")},
		{name: "dot hyphen suffix", source: []byte("0@0.0.-")},
		{name: "double underscore suffix", source: []byte("0@0.0.__")},
		{name: "underscore then letter", source: []byte("0@0.0._x")},
		{name: "letter then underscore", source: []byte("0@0.0.x_")},
		{name: "letter then hyphen", source: []byte("0@0.0.x-")},
		{name: "suffix before comma", source: []byte("0@0.0._,")},
		{name: "double dot before underscore", source: []byte("0@0.0.._")},
		{name: "valid internal underscore", source: []byte("0@a_b.0")},
		{name: "internal empty label", source: []byte("0@0..0")},
		{name: "mailto control", source: []byte("mailto:0@0.0._")},
		{name: "xmpp control", source: []byte("xmpp:0@0.0._/r")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkExtendedEmailLocalPunctuation(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "exclamation", source: []byte("0!@0.0")},
		{name: "hash", source: []byte("0#@0.0")},
		{name: "dollar", source: []byte("0$@0.0")},
		{name: "percent", source: []byte("0%@0.0")},
		{name: "ampersand", source: []byte("0&@0.0")},
		{name: "apostrophe", source: []byte("0'@0.0")},
		{name: "asterisk", source: []byte("0*@0.0")},
		{name: "plus", source: []byte("0+@0.0")},
		{name: "slash", source: []byte("0/@0.0")},
		{name: "equals", source: []byte("0=@0.0")},
		{name: "question", source: []byte("0?@0.0")},
		{name: "caret", source: []byte("0^@0.0")},
		{name: "backtick", source: []byte("0`@0.0")},
		{name: "braces", source: []byte("0{@0.0")},
		{name: "pipe", source: []byte("0|@0.0")},
		{name: "tilde", source: []byte("0~@0.0")},
		{name: "colon", source: []byte("0:@0.0")},
		{name: "comma", source: []byte("0,@0.0")},
		{name: "leading exclamation", source: []byte("!0@0.0")},
		{name: "leading dot", source: []byte(".0@0.0")},
		{name: "leading plus", source: []byte("+0@0.0")},
		{name: "leading underscore", source: []byte("_0@0.0")},
		{name: "mailto broad punctuation rejected", source: []byte("mailto:0!@0.0")},
		{name: "mailto gfm punctuation accepted", source: []byte("mailto:0+@0.0")},
		{name: "xmpp broad punctuation rejected", source: []byte("xmpp:0!@0.0/resource")},
		{name: "xmpp gfm punctuation accepted", source: []byte("xmpp:0+@0.0/resource")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkContainerLazyContinuation(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "exact round 52 artifact", source: []byte("><A>\n0")},
		{name: "complete tag spaced", source: []byte("> <A>\n0")},
		{name: "named tag", source: []byte("> <div>\n0")},
		{name: "comment", source: []byte("> <!--x-->\n0")},
		{name: "processing instruction", source: []byte("> <?x?>\n0")},
		{name: "declaration", source: []byte("> <!A>\n0")},
		{name: "cdata", source: []byte("> <![CDATA[x]]>\n0")},
		{name: "raw tag", source: []byte("> <script></script>\n0")},
		{name: "complete tag after paragraph", source: []byte("> x\n> <A>\n0")},
		{name: "named tag after paragraph", source: []byte("> x\n> <div>\n0")},
		{name: "complete html block remains open", source: []byte("> <A>\n> y\n0")},
		{name: "unmarked complete tag continues paragraph", source: []byte("> x\n<A>\n0")},
		{name: "unmarked named tag interrupts paragraph", source: []byte("> x\n<div>\n0")},
		{name: "nested blockquote html block", source: []byte("> > <A>\n0")},
		{name: "list item html block", source: []byte("> - <A>\n0")},
		{name: "list item paragraph", source: []byte("> - x\n0")},
		{name: "unclosed fenced code", source: []byte("> ```\n> code\n0")},
		{name: "closed fenced code", source: []byte("> ```\n> code\n> ```\n0")},
		{name: "table leaf", source: []byte("> a | b\n> - | -\n0")},
		{name: "table at eof", source: []byte("> a | b\n> - | -")},
		{name: "table with body then outside", source: []byte("> a | b\n> - | -\n> x | y\n0")},
		{name: "table with long delimiters", source: []byte("> a | b\n> --- | ---\n0")},
		{name: "table before blank", source: []byte("> a | b\n> - | -\n>\n0")},
		{name: "list paragraph before marked blank", source: []byte("> - x\n>\n0")},
		{name: "list reference before marked blank", source: []byte("> - [a]: /url\n>\n0")},
		{name: "pipe table with body", source: []byte("> | A | B |\n> | - | - |\n> | x | y |\n0")},
		{name: "list nested table", source: []byte("- a | b\n  - | -\n  x | y")},
		{name: "top level ambiguous delimiter", source: []byte("a | b\n- | -")},
		{name: "top level pipe delimiter", source: []byte("| a | b |\n| - | - |")},
		{name: "reference definition leaf", source: []byte("> [a]: /url\n0")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkBlockquoteTabPadding(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "no padding", source: []byte(">0")},
		{name: "one space", source: []byte("> 0")},
		{name: "tab only", source: []byte(">\t0")},
		{name: "two spaces", source: []byte(">  0")},
		{name: "three spaces", source: []byte(">   0")},
		{name: "four spaces", source: []byte(">    0")},
		{name: "five spaces", source: []byte(">     0")},
		{name: "space then tab", source: []byte("> \t0")},
		{name: "two spaces then tab", source: []byte(">  \t0")},
		{name: "tab then space", source: []byte(">\t 0")},
		{name: "tab then two spaces", source: []byte(">\t  0")},
		{name: "tab then three spaces", source: []byte(">\t   0")},
		{name: "tab space tab", source: []byte(">\t \t0")},
		{name: "tab two spaces tab", source: []byte(">\t  \t0")},
		{name: "tab three spaces tab", source: []byte(">\t   \t0")},
		{name: "tab tab", source: []byte(">\t\t0")},
		{name: "space tab tab", source: []byte("> \t\t0")},
		{name: "space tab space", source: []byte("> \t 0")},
		{name: "space then tab before text", source: []byte("> \tabc")},
		{name: "nested compact", source: []byte(">>0")},
		{name: "nested one space", source: []byte(">> 0")},
		{name: "nested two spaces", source: []byte(">>  0")},
		{name: "nested space tab", source: []byte(">> \t0")},
		{name: "nested two spaces tab", source: []byte(">>  \t0")},
		{name: "nested three spaces tab", source: []byte(">>   \t0")},
		{name: "spaced nested two spaces tab", source: []byte("> >  \t0")},
		{name: "triple nested two spaces tab", source: []byte(">>>  \t0")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBlockquoteStructuralTabPaddingRemainsSourceBound(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "consumed tab empty heading", source: []byte(">\t#")},
		{name: "consumed tab heading", source: []byte(">\t# x")},
		{name: "space then tab empty heading", source: []byte("> \t#")},
		{name: "space then tab level two empty heading", source: []byte("> \t##")},
		{name: "space then tab empty heading newline", source: []byte("> \t#\n")},
		{name: "space then tab empty heading trailing space", source: []byte("> \t# ")},
		{name: "space then tab heading", source: []byte("> \t# x")},
		{name: "two spaces then tab empty heading", source: []byte(">  \t#")},
		{name: "two spaces then tab heading", source: []byte(">  \t# x")},
		{name: "space then tab bullet", source: []byte("> \t- x")},
		{name: "two spaces then tab bullet", source: []byte(">  \t- x")},
		{name: "space then tab ordered", source: []byte("> \t1. x")},
		{name: "two spaces then tab ordered", source: []byte(">  \t1. x")},
		{name: "space then tab nested quote", source: []byte("> \t> x")},
		{name: "space then tab fence", source: []byte("> \t```\nx\n```")},
		{name: "consumed tab fence exact round 56", source: []byte(">\t```")},
		{name: "consumed tab tilde fence", source: []byte(">\t~~~")},
		{name: "consumed tab fence newline", source: []byte(">\t```\n")},
		{name: "consumed tab fence body", source: []byte(">\t```\n> x\n> ```")},
		{name: "one leading space consumed tab fence", source: []byte(" >\t```")},
		{name: "two leading spaces consumed tab fence", source: []byte("  >\t```")},
		{name: "three leading spaces consumed tab fence", source: []byte("   >\t```")},
		{name: "space then tab text", source: []byte("> \ttext")},
		{name: "one leading space space then tab empty heading", source: []byte(" > \t#")},
		{name: "two leading spaces space then tab empty heading", source: []byte("  > \t#")},
		{name: "three leading spaces space then tab empty heading", source: []byte("   > \t#")},
		{name: "one leading space space then tab heading", source: []byte(" > \t# x")},
		{name: "one leading space space then tab text", source: []byte(" > \ttext")},
		{name: "one leading space space then tab bullet", source: []byte(" > \t- x")},
		{name: "one leading space tab padding empty heading", source: []byte(" >\t#")},
		{name: "two leading spaces tab padding empty heading", source: []byte("  >\t#")},
		{name: "three leading spaces tab padding empty heading", source: []byte("   >\t#")},
		{name: "one leading space two spaces tab empty heading", source: []byte(" >  \t#")},
		{name: "two leading spaces two spaces tab empty heading", source: []byte("  >  \t#")},
		{name: "three leading spaces two spaces tab empty heading exact round 57", source: []byte("   >  \t#")},
		{name: "one leading space three spaces tab empty heading", source: []byte(" >   \t#")},
		{name: "two leading spaces three spaces tab empty heading", source: []byte("  >   \t#")},
		{name: "three leading spaces three spaces tab empty heading", source: []byte("   >   \t#")},
		{name: "three leading spaces two spaces tab heading text", source: []byte("   >  \t# x")},
		{name: "three leading spaces two spaces tab level two empty heading", source: []byte("   >  \t##")},
		{name: "three leading spaces two spaces tab ordinary text", source: []byte("   >  \ttext")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkNestedListTabPadding(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "nested marker tab", source: []byte("* *\t0")},
		{name: "nested marker tab one space", source: []byte("* *\t 0")},
		{name: "nested marker tab two spaces", source: []byte("* *\t  0")},
		{name: "nested marker tab three spaces", source: []byte("* *\t   0")},
		{name: "nested marker space tab", source: []byte("* * \t0")},
		{name: "nested marker two spaces tab", source: []byte("* *  \t0")},
		{name: "ordinary nested list", source: []byte("* - child")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkEmptyListSiblingAfterBlank(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "same marker no indent", source: []byte("*\n\n* 0")},
		{name: "same marker one space", source: []byte("*\n\n * 0")},
		{name: "same marker two spaces", source: []byte("*\n\n  * 0")},
		{name: "same marker three spaces", source: []byte("*\n\n   * 0")},
		{name: "same marker four spaces is code", source: []byte("*\n\n    * 0")},
		{name: "different marker is separate list", source: []byte("*\n\n   - 0")},
		{name: "no blank is nested list", source: []byte("*\n   * 0")},
		{name: "same marker empty sibling", source: []byte("*\n\n   *")},
		{name: "dash same marker three spaces", source: []byte("-\n\n   - 0")},
		{name: "ordered same punctuation", source: []byte("1.\n\n   2. 0")},
		{name: "ordered different punctuation", source: []byte("1.\n\n   2) 0")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkNestedListWidePaddingContinuationBoundary(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "artifact isolated cr", source: []byte("*\n\n+\r  *     0\n  0")},
		{name: "artifact lf", source: []byte("*\n\n+\n  *     0\n  0")},
		{name: "artifact crlf", source: []byte("*\n\n+\r\n  *     0\n  0")},
		{name: "no preceding list", source: []byte("+\r  *     0\n  0")},
		{name: "same preceding marker", source: []byte("*\n\n*\r  *     0\n  0")},
		{name: "nested padding one", source: []byte("*\n\n+\r  * 0\n  0")},
		{name: "nested padding two", source: []byte("*\n\n+\r  *  0\n  0")},
		{name: "nested padding three", source: []byte("*\n\n+\r  *   0\n  0")},
		{name: "nested padding four", source: []byte("*\n\n+\r  *    0\n  0")},
		{name: "nested padding five", source: []byte("*\n\n+\r  *     0\n  0")},
		{name: "nested padding six", source: []byte("*\n\n+\r  *      0\n  0")},
		{name: "final indent zero", source: []byte("*\n\n+\r  *     0\n0")},
		{name: "final indent one", source: []byte("*\n\n+\r  *     0\n 0")},
		{name: "final indent two", source: []byte("*\n\n+\r  *     0\n  0")},
		{name: "final indent three", source: []byte("*\n\n+\r  *     0\n   0")},
		{name: "final indent four", source: []byte("*\n\n+\r  *     0\n    0")},
		{name: "alternate nested marker", source: []byte("*\n\n+\r  -     0\n  0")},
		{name: "alternate outer marker", source: []byte("+\n\n*\r  +     0\n  0")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkListReferenceDefinitionLineEndings(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "ordered lf", source: []byte("0) [0]:0\n0")},
		{name: "ordered crlf", source: []byte("0) [0]:0\r\n0")},
		{name: "ordered isolated cr", source: []byte("0) [0]:0\r0")},
		{name: "bullet lf", source: []byte("- [0]:0\n0")},
		{name: "bullet crlf", source: []byte("- [0]:0\r\n0")},
		{name: "bullet isolated cr", source: []byte("- [0]:0\r0")},
		{name: "top level lf", source: []byte("[0]:0\n0")},
		{name: "top level crlf", source: []byte("[0]:0\r\n0")},
		{name: "top level isolated cr", source: []byte("[0]:0\r0")},
		{name: "ordered indented continuation lf", source: []byte("0) [0]:0\n   0")},
		{name: "ordered indented continuation cr", source: []byte("0) [0]:0\r   0")},
		{name: "tight ordinary paragraph", source: []byte("- first\n  second")},
		{name: "reference without residual", source: []byte("- [0]:0")},
		{name: "reference then indented residual", source: []byte("- [0]:0\n  residual")},
		{name: "reference then multiline residual", source: []byte("- [0]:0\n  first\n  second")},
		{name: "two references then residual", source: []byte("- [a]:0\n  [b]:1\n  residual")},
		{name: "ordinary paragraph then reference text", source: []byte("- first\n  [0]:0")},
		{name: "reference then nested list", source: []byte("- [0]:0\n  - child")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func TestM114NativeBackendMatchesGoldmarkWhitespaceOnlyMultilineCodeSpanProjection(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
	}{
		{name: "one space lf", source: []byte("` \n`")},
		{name: "two spaces lf", source: []byte("`  \n`")},
		{name: "three spaces lf", source: []byte("`   \n`")},
		{name: "four spaces lf", source: []byte("`    \n`")},
		{name: "two spaces crlf", source: []byte("`  \r\n`")},
		{name: "two spaces isolated cr", source: []byte("`  \r`")},
		{name: "space tab lf", source: []byte("` \t\n`")},
		{name: "tab only lf", source: []byte("`\t\n`")},
		{name: "space tab text lf", source: []byte("` \t0\n`")},
		{name: "tab text lf", source: []byte("`\t0\n`")},
		{name: "space text lf", source: []byte("` 0\n`")},
		{name: "two spaces text lf", source: []byte("`  0\n`")},
		{name: "double backtick spaces", source: []byte("``  \n``")},
		{name: "second line text", source: []byte("`  \n x`")},
	}
	candidate := native.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.source)
			first, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if !bytes.Equal(before, tt.source) {
				t.Fatal("ParseDocument() mutated source")
			}
			assertM114ObservationsSourceBound(t, first, len(tt.source))
			second, err := candidate.ParseDocument(tt.source)
			if err != nil {
				t.Fatalf("second ParseDocument() error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("ParseDocument() is not deterministic")
			}
		})
	}
}

func FuzzM114NativeBackendObservationsRemainSourceBound(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("# Title\n\nparagraph with *em* [link](<target>)\n"),
		[]byte("Title\r\n=====\r\n\r\n> quote\r\n"),
		[]byte("| A | B |\n| :- | -: |\n| x | y |\n"),
		[]byte("[ref]: <target> \"Guide\"\n\n[docs][ref] ![img][ref]\n"),
		{0xff, 0x00, '#', ' ', 0xfe, '\n'},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source []byte) {
		if len(source) > 64<<10 {
			return
		}
		before := bytes.Clone(source)
		first, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("ParseDocument() error = %v", err)
		}
		if !bytes.Equal(source, before) {
			t.Fatal("ParseDocument() mutated fuzz source")
		}
		assertM114ObservationsSourceBound(t, first, len(source))
		second, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("second ParseDocument() error = %v", err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Fatal("ParseDocument() is nondeterministic for fuzz source")
		}
	})
}

func FuzzM114NativeBackendLegacyDifferentialCorpusRemainsSourceBound(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("plain\n"),
		[]byte("# *head* `code`\n"),
		[]byte("> > nested\n\n- [x] task\n"),
		[]byte("[a]: /target\n\n[a] [x][a]\n"),
		[]byte("<div>\nopaque\n</div>\n"),
		[]byte(">"),
		[]byte("* [X]0"),
		[]byte("[A]:0(00000"),
		[]byte{'[', '0', ']', ':', 0},
		[]byte("# *00000`0*"),
		[]byte("*$[$"),
		[]byte("use[^n]\n\n[^n]: body\n"),
		[]byte("~~~0"),
		[]byte("~~0~"),
		[]byte("*[*"),
		[]byte("*[0*"),
		[]byte("0 *`*"),
		[]byte(">0000000\n*"),
		[]byte("[A]:0\n[][A]0"),
		[]byte("[00]]:00"),
		[]byte{'~', 0x88, '~'},
		[]byte("<000@0>0"),
		[]byte{'~', 0xcc, '*', '~', ' '},
		[]byte("*$*$"),
		[]byte("* [X] "),
		[]byte("000000\n[^0]:0000000 "),
		[]byte("*!***!* 00"),
		[]byte("~0~~~0~"),
		[]byte("# *0`0**"),
		[]byte("*[]*"),
		[]byte("* [X][]"),
		{'~', 0x9c, '~', '0', '~'},
		[]byte("[^0]:0\n0"),
		[]byte("*~*~"),
		[]byte("*0[]*"),
		[]byte("*!**~* 000"),
		[]byte("*0 * 0*"),
		[]byte("~!*~!*!*"),
		[]byte("$$\n$$"),
		[]byte("* [X]\n0"),
		[]byte("*0 *~** "),
		[]byte("[A]:0\n000[A]["),
		[]byte("0) [^0]:000"),
		[]byte("[^0]:[^0]:"),
		[]byte("0) [^0]:"),
		[]byte("~![~"),
		[]byte("~~~0\t0"),
		[]byte("*0[*]"),
		[]byte(">0\r\t 0"),
		[]byte("[][*]*"),
		[]byte("* *\n\n  0"),
		[]byte("` 0\n`000000000"),
		[]byte("[A]:0\n[\nA]"),
		[]byte("[0]:0\n-"),
		[]byte("*\t  0"),
		[]byte("[][^n]\n[^n]:0"),
		[]byte(" [0]:0"),
		[]byte("[0]:\n#"),
		[]byte("\r*\r\n"),
		{'[', '^', '\f', ']', ':'},
		[]byte("- [X]\n\n-"),
		{'[', '\v', ']', ':', '0'},
		[]byte("0@00.000.."),
		[]byte("> \t0"),
		[]byte("> \t#"),
		[]byte(">\t  0"),
		[]byte(">\t  \t0"),
		[]byte(">>  \t0"),
		[]byte("* *\t  0"),
		[]byte("0) [0]:0\r0"),
		[]byte("_0[*_ ]"),
		[]byte("_[*_ \r]"),
		[]byte("0<!A0>"),
		[]byte("00000000000000000000000000<!A\r*0*>"),
		[]byte(" > \t#"),
		[]byte("*\n\n+\r  *     0\n  0"),
		[]byte("*\n\n   * 0"),
		[]byte("`  \n`"),
		[]byte("[^0]:0\n[0]:0"),
		[]byte("[\xf5\r0\n# 00]:0"),
		[]byte("__[*_ 0]"),
		[]byte("0!@0.0"),
		[]byte("0@0.0._"),
		[]byte("><A>\n0"),
		[]byte("0<?>"),
		[]byte("www.000"),
		[]byte("*0 **0*~*~"),
		[]byte(">\t```"),
		[]byte("   >  \t#"),
		[]byte("[^0]:0) [^0]:"),
		[]byte("[^n]\n\n[^n]:\n0"),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source []byte) {
		if len(source) > 16<<10 {
			return
		}
		before := bytes.Clone(source)
		first, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("ParseDocument() error = %v", err)
		}
		if !bytes.Equal(source, before) {
			t.Fatal("ParseDocument() mutated fuzz source")
		}
		assertM114ObservationsSourceBound(t, first, len(source))
		second, err := native.New().ParseDocument(source)
		if err != nil {
			t.Fatalf("second ParseDocument() error = %v", err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Fatal("ParseDocument() is nondeterministic for legacy differential corpus fuzz source")
		}
	})
}

func FuzzM114NativeDirectLinkProofMatchesGoldmark(f *testing.F) {
	f.Add(uint8(0), uint16(0))
	f.Add(uint8(1), uint16(3))
	f.Add(uint8(2), uint16(12))

	f.Fuzz(func(t *testing.T, selector uint8, offset uint16) {
		source := []byte("[docs](<target>)")
		want := parser.ConstructionLinkImageExpectation{
			Kind:        parser.KindInlineLink,
			SyntaxRange: parser.Range{Start: 0, End: len(source)},
			LabelRange:  parser.Range{Start: 1, End: 5},
			Destination: "target",
		}
		switch selector % 5 {
		case 1:
			want.Destination = "other"
		case 2:
			want.SyntaxRange.End = int(offset) % (len(source) + 2)
		case 3:
			want.LabelRange.End = int(offset) % (len(source) + 1)
		case 4:
			want.Kind = parser.KindImage
		}
		compareM114ProofErrors(t,
			func() error {
				return goldmarkparser.New().ValidateConstructionLinkImages(source, []parser.ConstructionLinkImageExpectation{want})
			},
			func() error {
				return native.New().ValidateConstructionLinkImages(source, []parser.ConstructionLinkImageExpectation{want})
			},
		)
	})
}

func FuzzM114NativeReferenceProofMatchesGoldmark(f *testing.F) {
	f.Add(uint8(0), uint16(0))
	f.Add(uint8(1), uint16(5))
	f.Add(uint8(2), uint16(9))

	f.Fuzz(func(t *testing.T, selector uint8, offset uint16) {
		source := []byte("[docs][ref]")
		want := parser.ConstructionReferenceInlineExpectation{
			Kind:           parser.KindInlineLink,
			Form:           parser.ConstructionReferenceInlineFull,
			SyntaxRange:    parser.Range{Start: 0, End: len(source)},
			LabelRange:     parser.Range{Start: 1, End: 5},
			ReferenceRange: parser.Range{Start: 7, End: 10},
			Reference:      "ref",
			Destination:    "target",
		}
		switch selector % 6 {
		case 1:
			want.Reference = "changed"
		case 2:
			want.ReferenceRange.End = int(offset) % (len(source) + 1)
		case 3:
			want.LabelRange.Start = int(offset) % (len(source) + 1)
		case 4:
			want.Form = parser.ConstructionReferenceInlineCollapsed
		case 5:
			want.StructuredLabel = true
		}
		compareM114ProofErrors(t,
			func() error {
				return goldmarkparser.New().ValidateConstructionReferenceInlines(source, []parser.ConstructionReferenceInlineExpectation{want})
			},
			func() error {
				return native.New().ValidateConstructionReferenceInlines(source, []parser.ConstructionReferenceInlineExpectation{want})
			},
		)
	})
}

func assertM114ObservationsSourceBound(t testing.TB, observations parser.DocumentObservations, total int) {
	t.Helper()
	for index, node := range observations.Nodes {
		if !node.Range.Valid(total) {
			t.Fatalf("node %d range %v invalid for source length %d", index, node.Range, total)
		}
		if node.BlockquoteContentRange != (parser.Range{}) && !node.BlockquoteContentRange.Valid(total) {
			t.Fatalf("node %d blockquote content range %v invalid", index, node.BlockquoteContentRange)
		}
		for _, range_ := range node.BlockquoteSemanticRanges {
			if !range_.Valid(total) {
				t.Fatalf("node %d blockquote semantic range %v invalid", index, range_)
			}
		}
		for _, range_ := range node.FencedCodeContentRanges {
			if !range_.Valid(total) {
				t.Fatalf("node %d fenced content range %v invalid", index, range_)
			}
		}
	}
	for index, usage := range observations.LinkUsages {
		if usage.Anchor < 0 || usage.Anchor > total {
			t.Fatalf("link usage %d anchor %d invalid for source length %d", index, usage.Anchor, total)
		}
	}
	for index, usage := range observations.UnresolvedReferenceUsages {
		if usage.Anchor < 0 || usage.Anchor > total {
			t.Fatalf("unresolved usage %d anchor %d invalid for source length %d", index, usage.Anchor, total)
		}
	}
	for index, definition := range observations.FootnoteDefinitions {
		if definition.Anchor < 0 || definition.Anchor > total {
			t.Fatalf("footnote definition %d anchor %d invalid", index, definition.Anchor)
		}
		for _, range_ := range definition.BodyRanges {
			if !range_.Valid(total) {
				t.Fatalf("footnote definition %d body range %v invalid", index, range_)
			}
		}
	}
	for index, reference := range observations.FootnoteReferences {
		if !reference.Range.Valid(total) || !reference.LabelRange.Valid(total) || reference.DefinitionAnchor < 0 || reference.DefinitionAnchor > total {
			t.Fatalf("footnote reference %d is not source-bound: %#v", index, reference)
		}
	}
	for index, expression := range observations.MathExpressions {
		if !expression.Range.Valid(total) || !expression.PayloadRange.Valid(total) {
			t.Fatalf("math expression %d is not source-bound: %#v", index, expression)
		}
	}
}

func compareM114ProofErrors(t testing.TB, oracleCall, nativeCall func() error) {
	t.Helper()
	oracleErr := oracleCall()
	nativeErr := nativeCall()
	if (oracleErr != nil) != (nativeErr != nil) {
		t.Fatalf("proof acceptance mismatch: native=%v Goldmark=%v", nativeErr, oracleErr)
	}
}
