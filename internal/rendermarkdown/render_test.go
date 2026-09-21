package rendermarkdown

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
	"github.com/zoster81/marksplice/internal/testutil/commonmarkspec"
	"github.com/zoster81/marksplice/internal/testutil/gfmspec"
)

type semanticFact struct {
	phase       parser.SemanticPhase
	kind        parser.SemanticKind
	value       string
	level       int
	destination string
	title       string
	hasTitle    bool
	label       string
	email       bool
	ordered     bool
	start       int
	tight       bool
	checked     bool
	header      bool
	column      int
	columns     int
	alignment   parser.TableAlignment
	language    string
	info        string
	alert       parser.SemanticAlertKind
	frontMatter parser.SemanticFrontMatterFormat
	math        parser.MathExpressionStyle
}

func TestCanonicalMarkdownSemanticRoundTripComplexFamilies(t *testing.T) {
	t.Parallel()

	source := []byte("---\r\ntitle: demo\r\n---\r\n\r\nHeading\r\n=======\r\n\r\nParagraph with *em **strong** ~~strike~~*, [direct](target.md 'a \\\"title\\\"'), [reference][ref], ![image *alt*](image.png), <https://example.test/a>, www.example.test, `code`, and $x$ plus $`y`$.  \r\nnext line\r\n\r\n[ref]: /reference 'reference title'\r\n\r\n> quote\r\n>\r\n> 3. outer\r\n>    - inner\r\n\r\n> [!NOTE]\r\n> alert **body**\r\n>\r\n> - child\r\n\r\n- [x] task\r\n- plain\r\n\r\n3. loose one\r\n\r\n4. loose two\r\n\r\n| Left | Right |\r\n| :--- | ---: |\r\n| `a\\|b` | ~~cell~~ |\r\n\r\n~~~go`x\r\n```\r\nbody\r\n~~~\r\n\r\n    indented\r\n    code\r\n\r\n<div data-x=\"a|b\">\r\nraw\r\n</div>\r\n\r\nuse[^note]\r\n\r\n[^note]:\r\n    first *footnote* block\r\n\r\n    - nested item\r\n\r\n$$z^2$$\r\n")
	backend := native.New()
	firstFacts := collectSemanticFacts(t, backend, source)

	var canonical bytes.Buffer
	if err := Render(&canonical, source, backend); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if bytes.Contains(canonical.Bytes(), []byte("\r")) {
		t.Fatalf("canonical output contains CR bytes: %q", canonical.Bytes())
	}
	secondFacts := collectSemanticFacts(t, backend, canonical.Bytes())
	if !reflect.DeepEqual(secondFacts, firstFacts) {
		t.Fatalf("semantic round trip changed: %s\noutput:\n%s", semanticFactsDifference(firstFacts, secondFacts), canonical.Bytes())
	}

	var second bytes.Buffer
	if err := Render(&second, canonical.Bytes(), backend); err != nil {
		t.Fatalf("second Render() error = %v", err)
	}
	if !bytes.Equal(second.Bytes(), canonical.Bytes()) {
		t.Fatalf("canonical output is not idempotent\nfirst:\n%s\nsecond:\n%s", canonical.Bytes(), second.Bytes())
	}
}

func TestPublishedCommonMarkCanonicalSemanticRoundTrip(t *testing.T) {
	path := os.Getenv("MARKSPLICE_COMMONMARK_SPEC_HTML")
	if path == "" {
		t.Skip("MARKSPLICE_COMMONMARK_SPEC_HTML is not set")
	}
	cases, err := commonmarkspec.LoadPublished(path)
	if err != nil {
		t.Fatalf("load CommonMark spec: %v", err)
	}
	if len(cases) != 652 {
		t.Fatalf("CommonMark case count = %d, want 652", len(cases))
	}
	backend := native.New()
	for _, case_ := range cases {
		case_ := case_
		t.Run(strconv.Itoa(case_.Number), func(t *testing.T) {
			assertCanonicalSemanticRoundTrip(t, backend, []byte(case_.Markdown))
		})
	}
}

func TestPublishedGFMCanonicalSemanticRoundTrip(t *testing.T) {
	path := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if path == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(path)
	if err != nil {
		t.Fatalf("load GFM spec: %v", err)
	}
	if len(cases) != 677 {
		t.Fatalf("GFM case count = %d, want 677", len(cases))
	}
	backend := native.New()
	for _, case_ := range cases {
		case_ := case_
		t.Run(strconv.Itoa(case_.Number), func(t *testing.T) {
			assertCanonicalSemanticRoundTrip(t, backend, []byte(case_.Markdown))
		})
	}
}

func TestCanonicalMarkdownEdgeSyntaxRoundTrips(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		check  func(*testing.T, string)
	}{
		{
			name:   "code span containing backtick",
			source: "`` ` ``\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "`` ` ``\n" {
					t.Fatalf("canonical code span = %q", canonical)
				}
			},
		},
		{
			name:   "code span containing only spaces",
			source: "`  `\n",
		},
		{
			name:   "title quote canonicalization",
			source: "[label](target(a).md 'he said \"hi\"')\n\n[ref]: /target 'he said \"again\"'\n\n[use][ref]\n",
		},
		{
			name:   "standalone reference definition is retained",
			source: "[unused]: /target 'title'\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "[unused]: </target> \"title\"\n" {
					t.Fatalf("canonical standalone reference definition = %q", canonical)
				}
			},
		},
		{
			name:   "multiline reference definition is synthesized canonically",
			source: "[foo]:\n/url\n\n[foo]\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if !strings.Contains(canonical, "[foo][foo]\n\n[foo]: </url>\n") {
					t.Fatalf("canonical multiline reference definition = %q", canonical)
				}
			},
		},
		{
			name:   "ordered list crosses digit width",
			source: "9. nine\n10. ten\n11. eleven\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if !strings.Contains(canonical, "9. nine\n10. ten\n11. eleven\n") {
					t.Fatalf("ordered markers not sequential: %q", canonical)
				}
			},
		},
		{
			name:   "nested strikethrough",
			source: "~~outer ~inner~ end~~\n",
		},
		{
			name:   "nested emphasis survives canonical text escaping",
			source: "*>\t*|**\n",
		},
		{
			name:   "escaped tilde before strikethrough",
			source: "\\~~~x~~\n",
		},
		{
			name:   "nested strikethrough preserves tab when flanking changes",
			source: "~#\t~#~~\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "~\\#\t~\\#~~\n" {
					t.Fatalf("canonical nested strike tab = %q", canonical)
				}
			},
		},
		{
			name:   "nested strikethrough keeps canonical tab entity when flanking is stable",
			source: "~#\t~a~~\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "~\\#&#9;~a~~\n" {
					t.Fatalf("canonical stable nested strike tab = %q", canonical)
				}
			},
		},
		{
			name:   "nested strikethrough across emphasis preserves tab flanking",
			source: "~*#\t~#~*~\n",
		},
		{
			name:   "nested strikethrough across emphasis keeps stable tab entity",
			source: "~*#\t~a~*~\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "~*\\#&#9;~a~*~\n" {
					t.Fatalf("canonical stable wrapped strike tab = %q", canonical)
				}
			},
		},
		{
			name:   "emphasis alternation survives strikethrough wrapper",
			source: "*~_)_~*\n",
		},
		{
			name:   "triple nested emphasis preserves marker hierarchy",
			source: "*#*b#_)_**\n",
		},
		{
			name:   "shared star delimiter run component preserves deep hierarchy",
			source: "**#*a)_)_** a*\n",
		},
		{
			name:   "shared underscore delimiter run component preserves deep hierarchy",
			source: "__#_a)*)*__ a_\n",
		},
		{
			name:   "anchored mixed star delimiter run chain preserves deep hierarchy",
			source: "*#*b#*a)_)_** a*\n",
		},
		{
			name:   "anchored mixed underscore delimiter run chain preserves deep hierarchy",
			source: "_#_b#_a)*)*__ a_\n",
		},
		{
			name:   "shared closing pair follows final outer marker",
			source: "_#_b#_a)*)*__ a*\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "\\_\\#*b\\#*a\\)_\\)_** a\\*\n" {
					t.Fatalf("canonical shared closing pair = %q", canonical)
				}
			},
		},
		{
			name:   "shared opening pair follows final outer marker",
			source: "**a_a)*)*_* a*\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "**a\\_a\\)_\\)_\\_* a*\n" {
					t.Fatalf("canonical shared opening pair = %q", canonical)
				}
			},
		},
		{
			name:   "deep shared opening pair follows final outer marker",
			source: "__#*a)_)_*_ a_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "**\\#*a\\)_\\)_** a*\n" {
					t.Fatalf("canonical deep shared opening pair = %q", canonical)
				}
			},
		},
		{
			name:   "three-level shared opener keeps leaf marker when closes are separated",
			source: "__*#a*b_]_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "***\\#a*b*\\]*\n" {
					t.Fatalf("canonical separated leaf close = %q", canonical)
				}
			},
		},
		{
			name:   "three-level shared opener alternates leaf marker when closes are adjacent",
			source: "__*#1*_ #_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "**_\\#1_* \\#*\n" {
					t.Fatalf("canonical adjacent leaf close = %q", canonical)
				}
			},
		},
		{
			name:   "three-level shared opener keeps same leaf marker across punctuation gap",
			source: "__[*#a*b_]_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "**\\[*\\#a*b*\\]*\n" {
					t.Fatalf("canonical punctuation-gap leaf = %q", canonical)
				}
			},
		},
		{
			name:   "three-level shared opener flips base across adjacent punctuation gap",
			source: "__#*#1*_ #_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "**\\#_\\#1_* \\#*\n" {
					t.Fatalf("canonical punctuation-gap adjacent leaf = %q", canonical)
				}
			},
		},
		{
			name:   "three-level shared opener same-marker spelling is a fixed point",
			source: "**\\[*\\#a*b*\\]*\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "**\\[*\\#a*b*\\]*\n" {
					t.Fatalf("canonical same-marker fixed point = %q", canonical)
				}
			},
		},
		{
			name:   "three-level separated openers preserve invalid-byte leaf before bracket",
			source: "_#_1[*\xff*_]_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*\\#*1\\[_\xff_*\\]*\n" {
					t.Fatalf("canonical separated invalid-byte bracket = %q", canonical)
				}
			},
		},
		{
			name:   "three-level separated openers preserve invalid-byte leaf before hash",
			source: "_>_Z\xff*\xfe*_#_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*\\>*Z\xff_\xfe_*\\#*\n" {
					t.Fatalf("canonical separated invalid-byte hash = %q", canonical)
				}
			},
		},
		{
			name:   "mixed-marker adjacent emphasis chain keeps three levels",
			source: "_*!\t_*\x00_*\rc_",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "**\\!&#9;_\\*\x00_*\nc*\n" {
					t.Fatalf("canonical mixed-marker adjacent chain = %q", canonical)
				}
			},
		},
		{
			name:   "mixed-marker tab before leaf preserves star outer chain",
			source: "*_\xfe\t*>*_*",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*_\xfe\t*\\>*_*\n" {
					t.Fatalf("canonical star outer tab-before-leaf chain = %q", canonical)
				}
			},
		},
		{
			name:   "mixed-marker tab before leaf preserves underscore outer chain",
			source: "_*\xff\t_]_*_",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "_*\xff\t_\\]_*_\n" {
					t.Fatalf("canonical underscore outer tab-before-leaf chain = %q", canonical)
				}
			},
		},
		{
			name:   "mixed-marker tab before leaf preserves nested flanking",
			source: "_*\x00\t_*1_*_",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "_*\x00\t_\\*1_*_\n" {
					t.Fatalf("canonical mixed-marker tab-before-leaf flanking = %q", canonical)
				}
			},
		},
		{
			name:   "nested emphasis preserves host-valid outer marker",
			source: "X*Z[_\x00\t*>\x00*2_\xff*",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "X*Z\\[_\x00\t*\\>\x00*2_\xff*\n" {
					t.Fatalf("canonical host-valid outer marker = %q", canonical)
				}
			},
		},
		{
			name:   "same-marker tab before leaf preserves host topology",
			source: "\x00*Z>*\x00\t*)3*\x00*[*",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "\x00*Z\\>*\x00\t*\\)3*\x00*\\[*\n" {
					t.Fatalf("canonical same-marker host topology = %q", canonical)
				}
			},
		},
		{
			name:   "star shared-close chain restores source tab",
			source: "*\x00\t*)*a***",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*\x00\t*\\)*a***\n" {
					t.Fatalf("canonical star shared-close tab chain = %q", canonical)
				}
			},
		},
		{
			name:   "underscore shared-close chain restores source tab",
			source: "_\x00\n_\xff\t_]___",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "_\x00\n_\xff\t_\\]___\n" {
					t.Fatalf("canonical underscore shared-close tab chain = %q", canonical)
				}
			},
		},
		{
			name:   "same-marker multi-child shared close alternates first child",
			source: "*>*\x00\t*)*\n*a***",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*\\>*\x00&#9;_\\)_\n*a***\n" {
					t.Fatalf("canonical same-marker multi-child topology = %q", canonical)
				}
			},
		},
		{
			name:   "dual-flanking multi-child prefix keeps unconsumed star escaped",
			source: "**)*_*\r*\x00**",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "\\**\\)_\\__\n_\x00_*\n" {
					t.Fatalf("canonical dual-flanking multi-child prefix = %q", canonical)
				}
			},
		},
		{
			name:   "mixed-marker adjacent openers are separate physical runs",
			source: "__*\xff*0*a*b_]_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "***\xff*0*a*b*\\]*\n" {
					t.Fatalf("canonical mixed-marker adjacent openers = %q", canonical)
				}
			},
		},
		{
			name:   "three-level shared close run preserves source marker ownership",
			source: "_#_0*a*__\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "_\\#_0*a*__\n" {
					t.Fatalf("canonical shared close run = %q", canonical)
				}
			},
		},
		{
			name:   "nearby shared close spelling keeps stable canonical output",
			source: "*#*0_a_ b**\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*\\#_0\\_a\\_ b_*\n" {
					t.Fatalf("canonical nearby shared close control = %q", canonical)
				}
			},
		},
		{
			name:   "three-level shared close run preserves middle tail",
			source: "_#_1[*\xff0*a__\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "_\\#_1\\[*\xff0*a__\n" {
					t.Fatalf("canonical shared close middle tail = %q", canonical)
				}
			},
		},
		{
			name:   "deep shared close run preserves nested component ownership",
			source: "_#_1[*\xff*0*a*__\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "_\\#_1\\[*\xff*0*a*__\n" {
					t.Fatalf("canonical deep shared close run = %q", canonical)
				}
			},
		},
		{
			name:   "deep separated boundaries choose stable marker ownership",
			source: "_#_1[*\xff*0*a*_]_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*\\#*1\\[_\xff*0*a_*\\]*\n" {
					t.Fatalf("canonical deep separated boundaries = %q", canonical)
				}
			},
		},
		{
			name:   "deep separated boundary fixed point stays unchanged",
			source: "*\\#*1\\[_\xff*0*a_*\\]*\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*\\#*1\\[_\xff*0*a_*\\]*\n" {
					t.Fatalf("canonical deep separated fixed point = %q", canonical)
				}
			},
		},
		{
			name:   "deep separated boundary encodes lowercase tail for underscore flanking",
			source: "_#_1[*\xff*0*a*b_]_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*\\#*1\\[_\xff*0*a_&#98;*\\]*\n" {
					t.Fatalf("canonical lowercase boundary entity = %q", canonical)
				}
			},
		},
		{
			name:   "deep separated boundary encodes uppercase tail for underscore flanking",
			source: "_#_1[*\xff*0*a*A_]_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*\\#*1\\[_\xff*0*a_&#65;*\\]*\n" {
					t.Fatalf("canonical uppercase boundary entity = %q", canonical)
				}
			},
		},
		{
			name:   "deep separated boundary encodes digit tail for underscore flanking",
			source: "_#_1[*\xff*0*a*0_]_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*\\#*1\\[_\xff*0*a_&#48;*\\]*\n" {
					t.Fatalf("canonical digit boundary entity = %q", canonical)
				}
			},
		},
		{
			name:   "deep separated boundary encodes compound alphanumeric tail",
			source: "_#_1[*\xff*0*a*b]_]_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*\\#*1\\[_\xff*0*a_&#98;\\]*\\]*\n" {
					t.Fatalf("canonical compound boundary entity = %q", canonical)
				}
			},
		},
		{
			name:   "final shared opener preserves tab flanking after marker reconciliation",
			source: "**~*>*:*\t1*\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "**\\~_\\>_\\:*\t1*\n" {
					t.Fatalf("canonical final shared opener tab boundary = %q", canonical)
				}
			},
		},
		{
			name:   "final shared opener combines marker repair with tab boundary",
			source: "**~*>\x00*:*\t1*\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "**\\~_\\>\x00_\\:*\t1*\n" {
					t.Fatalf("canonical final shared opener combined boundary = %q", canonical)
				}
			},
		},
		{
			name:   "unconsumed star run prefix survives shallow nested emphasis",
			source: "***a*a*\n",
		},
		{
			name:   "multiple unconsumed stars survive shallow nested emphasis",
			source: "****1*1*\n",
		},
		{
			name:   "unconsumed star run survives shallow nested emphasis after text",
			source: "x***a*a*!\n",
		},
		{
			name:   "multiple unconsumed stars survive shallow nested emphasis before text",
			source: "x****1*1*a\n",
		},
		{
			name:   "shared boundary children keep star prefix escaped",
			source: "***_*[*a**\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "\\**_\\__\\[_a_*\n" {
					t.Fatalf("canonical shared boundary star prefix = %q", canonical)
				}
			},
		},
		{
			name:   "shared boundary children keep star prefix escaped with punctuation",
			source: "***>*#*a**\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "\\**_\\>_\\#_a_*\n" {
					t.Fatalf("canonical punctuated shared boundary star prefix = %q", canonical)
				}
			},
		},
		{
			name:   "shared boundary underscore prefix remains unescaped",
			source: "___>_[_a__\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "___\\>_\\[_a__\n" {
					t.Fatalf("canonical shared boundary underscore prefix = %q", canonical)
				}
			},
		},
		{
			name:   "shared opener children keep star prefix escaped before trailing text",
			source: "***_*[*a*c*\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "\\**_\\__\\[*a*c*\n" {
					t.Fatalf("canonical shared opener trailing star prefix = %q", canonical)
				}
			},
		},
		{
			name:   "shared opener children keep star prefix escaped before numeric trailing text",
			source: "***>*#*a*1*\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "\\**_\\>_\\#*a*1*\n" {
					t.Fatalf("canonical numeric trailing star prefix = %q", canonical)
				}
			},
		},
		{
			name:   "shared opener underscore prefix remains unescaped before trailing text",
			source: "___>_[_a_!_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "___\\>_\\[_a_\\!_\n" {
					t.Fatalf("canonical trailing underscore prefix = %q", canonical)
				}
			},
		},
		{
			name:   "separated children keep dual-purpose star prefix escaped",
			source: "**~*>*#*:*#*\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "\\**\\~_\\>_\\#_\\:_\\#*\n" {
					t.Fatalf("canonical separated dual-purpose star prefix = %q", canonical)
				}
			},
		},
		{
			name:   "separated children keep star prefix escaped before escaped marker tail",
			source: "**~*a*#*_*\\**\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "\\**\\~_a_\\#_\\__\\**\n" {
					t.Fatalf("canonical escaped-tail star prefix = %q", canonical)
				}
			},
		},
		{
			name:   "separated children keep underscore prefix stable",
			source: "__~_>_#_>_#_\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "__\\~_\\>_\\#_\\>_\\#_\n" {
					t.Fatalf("canonical separated underscore prefix = %q", canonical)
				}
			},
		},
		{
			name:   "shallow star prefix alternates outer marker across punctuation tail",
			source: "**a*a*[*\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "\\*_a*a*\\[_\n" {
					t.Fatalf("canonical shallow punctuation-tail prefix = %q", canonical)
				}
			},
		},
		{
			name:   "shallow star prefix alternates outer marker across invalid-byte tail",
			source: "**1*a*\xff*\n",
		},
		{
			name:   "shallow alternate marker ignores whitespace tail controls",
			source: "**#_a)*)__* a*\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "\\**\\#\\_a\\)_\\)\\_\\__ a*\n" {
					t.Fatalf("canonical shallow whitespace-tail control = %q", canonical)
				}
			},
		},
		{
			name:   "unconsumed star run prefix survives deep emphasis",
			source: "**#*a)_)_**\n",
		},
		{
			name:   "unconsumed underscore run prefix survives deep emphasis",
			source: "__#_a)*)*__\n",
		},
		{
			name:   "unconsumed star run prefix survives sensitive emphasis siblings",
			source: "**_)_*#* a*\n",
		},
		{
			name:   "unconsumed underscore run prefix survives sensitive emphasis siblings",
			source: "__*)*_*#_ a_\n",
		},
		{
			name:   "unconsumed star run suffix survives nested emphasis",
			source: "**a*_)_**\n",
		},
		{
			name:   "unconsumed underscore run suffix survives nested emphasis",
			source: "__a_*)*__\n",
		},
		{
			name:   "two-child suffix avoids modulo-three collision",
			source: "!\t_*1*\t*c*__\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "\\!&#9;_*1*&#9;*c*_\\_\n" {
					t.Fatalf("canonical two-child suffix = %q", canonical)
				}
			},
		},
		{
			name:   "star two-child suffix avoids modulo-three collision",
			source: "!\t**1*\t*c***\n",
		},
		{
			name:   "single-child star suffix run keeps canonical escaping",
			source: "**a*_***\n",
		},
		{
			name:   "single-child underscore suffix run keeps canonical escaping",
			source: "_*a**___\n",
		},
		{
			name:   "tab after unconsumed suffix preserves run flanking",
			source: "*_)_**\t1\n",
		},
		{
			name:   "shallow unconsumed suffix keeps canonical escaping",
			source: "*x**\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*x*\\*\n" {
					t.Fatalf("canonical shallow unconsumed suffix = %q", canonical)
				}
			},
		},
		{
			name:   "tab after shallow unconsumed suffix stays canonical",
			source: "*x**\t1\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*x*\\*&#9;1\n" {
					t.Fatalf("canonical shallow suffix tab = %q", canonical)
				}
			},
		},
		{
			name:   "shallow unconsumed run prefix keeps canonical escaping",
			source: "**#*x**\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "\\**\\#_x_*\n" {
					t.Fatalf("canonical shallow unconsumed prefix = %q", canonical)
				}
			},
		},
		{
			name:   "tab after nested emphasis preserves closing flanking",
			source: "*#*a)_)_**\t1\n",
		},
		{
			name:   "tab after ordinary emphasis remains canonical entity",
			source: "*x*\t1\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*x*&#9;1\n" {
					t.Fatalf("canonical ordinary emphasis tab = %q", canonical)
				}
			},
		},
		{
			name:   "matching emphasis markers remain canonical through strikethrough",
			source: "*~*x*~*\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "*~~*x*~~*\n" {
					t.Fatalf("canonical matching emphasis markers = %q", canonical)
				}
			},
		},
		{
			name:   "raw link destination rejects ASCII control",
			source: "[](>\x00)\n",
		},
		{
			name:   "empty heading and thematic break",
			source: "#\n\n***\n",
		},
		{
			name:   "terminal indented code preserves EOF payload",
			source: "    code",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "    code" {
					t.Fatalf("terminal indented code = %q", canonical)
				}
			},
		},
		{
			name:   "terminal fenced code preserves EOF payload",
			source: "```go\ncode",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "```go\ncode" {
					t.Fatalf("terminal fenced code = %q", canonical)
				}
			},
		},
		{
			name:   "synthetic reference precedes terminal code",
			source: "[foo]:\n/url\n\n[foo]\n\n    code",
		},
		{
			name:   "footnote-shaped reference link renders inline",
			source: "[CVE][^8] and note[^8]\n\n[^8]: https://example.test/advisory\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if !strings.Contains(canonical, "[CVE](<https://example.test/advisory>)") {
					t.Fatalf("footnote-shaped reference remained ambiguous: %q", canonical)
				}
			},
		},
		{
			name:   "nested list keeps following opaque sibling at parent depth",
			source: "- outer\n    - inner\n    <!--\n    sibling\n    -->\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if !strings.Contains(canonical, "- outer\n    - inner\n    <!--\n") {
					t.Fatalf("nested list and sibling indentation = %q", canonical)
				}
			},
		},
		{
			name:   "linked reference image uses stable shortcut form",
			source: "[![Build Status]][actions]\n\n[Build Status]: https://example.test/badge.svg\n[actions]: https://example.test/actions\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if !strings.Contains(canonical, "[![Build Status]][actions]") {
					t.Fatalf("linked reference image did not use shortcut form: %q", canonical)
				}
			},
		},
		{
			name:   "shortcut image keeps reference punctuation spelling",
			source: "[![serde_derive msrv]][Rust 1.71]\n\n[serde_derive msrv]: https://example.test/badge.svg\n[Rust 1.71]: https://example.test/rust\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if !strings.Contains(canonical, "[![serde_derive msrv]][Rust 1.71]") {
					t.Fatalf("shortcut image label spelling = %q", canonical)
				}
			},
		},
	}

	backend := native.New()
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			source := []byte(test.source)
			before := collectSemanticFacts(t, backend, source)
			var output bytes.Buffer
			if err := Render(&output, source, backend); err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			after := collectSemanticFacts(t, backend, output.Bytes())
			if !reflect.DeepEqual(after, before) {
				t.Fatalf("semantic facts changed\nbefore: %#v\nafter:  %#v\ncanonical: %q", before, after, output.String())
			}
			var second bytes.Buffer
			if err := Render(&second, output.Bytes(), backend); err != nil {
				t.Fatalf("second Render() error = %v", err)
			}
			if second.String() != output.String() {
				t.Fatalf("not idempotent: first=%q second=%q", output.String(), second.String())
			}
			if test.check != nil {
				test.check(t, output.String())
			}
		})
	}
}

func assertCanonicalSemanticRoundTrip(t *testing.T, backend Backend, source []byte) {
	t.Helper()
	before := collectSemanticFacts(t, backend, source)
	var output bytes.Buffer
	if err := Render(&output, source, backend); err != nil {
		t.Fatalf("Render() error = %v\nsource=%q", err, source)
	}
	after := collectSemanticFacts(t, backend, output.Bytes())
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("semantic facts changed: %s\nsource: %q\ncanonical: %q", semanticFactsDifference(before, after), source, output.Bytes())
	}
	var second bytes.Buffer
	if err := Render(&second, output.Bytes(), backend); err != nil {
		t.Fatalf("second Render() error = %v\ncanonical=%q", err, output.Bytes())
	}
	if !bytes.Equal(second.Bytes(), output.Bytes()) {
		t.Fatalf("canonical output is not idempotent\nfirst:  %q\nsecond: %q", output.Bytes(), second.Bytes())
	}
}

func semanticFactsDifference(before, after []semanticFact) string {
	limit := len(before)
	if len(after) < limit {
		limit = len(after)
	}
	for index := 0; index < limit; index++ {
		if before[index] != after[index] {
			return fmt.Sprintf("index=%d before=%#v after=%#v", index, before[index], after[index])
		}
	}
	return fmt.Sprintf("length before=%d after=%d", len(before), len(after))
}

func collectSemanticFacts(t *testing.T, backend Backend, source []byte) []semanticFact {
	t.Helper()
	facts := make([]semanticFact, 0, 32)
	footnotes := make([][]semanticFact, 0, 4)
	var currentFootnote []semanticFact
	if err := backend.WalkSemantic(source, func(event parser.SemanticEvent) error {
		// Reference-definition leaf projection depends on source shape: Native can
		// resolve a multiline definition for link semantics without projecting a
		// SemanticReferenceDefinition leaf. Canonical rendering intentionally turns
		// such targets into a single-line definition, so definition-leaf presence is
		// not part of the semantic-equivalence oracle. Dedicated tests above still
		// require standalone definitions to be retained and missing projected
		// definitions to be synthesized deterministically.
		if event.Kind == parser.SemanticReferenceDefinition {
			return nil
		}
		fact := semanticFactForEvent(event)
		if currentFootnote != nil {
			currentFootnote = append(currentFootnote, fact)
			if event.Kind == parser.SemanticFootnoteDefinition && event.Phase == parser.SemanticExit {
				footnotes = append(footnotes, currentFootnote)
				currentFootnote = nil
			}
			return nil
		}
		if event.Kind == parser.SemanticFootnoteDefinition && event.Phase == parser.SemanticEnter {
			currentFootnote = []semanticFact{fact}
			return nil
		}
		facts = append(facts, fact)
		return nil
	}); err != nil {
		t.Fatalf("WalkSemantic() error = %v", err)
	}
	if currentFootnote != nil {
		t.Fatal("WalkSemantic() ended with an open footnote definition")
	}
	// Native may replace a captured paragraph with a footnote definition or add
	// the same definition as a top-level semantic overlay. Canonical source shape
	// can therefore move an otherwise identical definition subtree within the
	// event stream. Compare every definition subtree intact, but independently
	// from flow position and in deterministic order.
	slices.SortFunc(footnotes, func(left, right []semanticFact) int {
		return strings.Compare(fmt.Sprintf("%#v", left), fmt.Sprintf("%#v", right))
	})
	for _, footnote := range footnotes {
		facts = append(facts, footnote...)
	}
	return facts
}

func semanticFactForEvent(event parser.SemanticEvent) semanticFact {
	value := event.Value
	switch event.Kind {
	case parser.SemanticFrontMatter, parser.SemanticRawHTML, parser.SemanticCodeBlock:
		value = normalizeLineEndings(value)
	case parser.SemanticHTMLBlock:
		value = strings.TrimSuffix(normalizeLineEndings(value), "\n")
	}
	label := event.Label
	if event.Kind == parser.SemanticLink || event.Kind == parser.SemanticImage {
		// Reference-style versus inline links are source representations of the
		// same rendered relationship. Destination, title, and inline content carry
		// the semantic contract; the parser-owned label spelling does not.
		label = ""
	}
	return semanticFact{
		phase:       event.Phase,
		kind:        event.Kind,
		value:       value,
		level:       event.Level,
		destination: event.Destination,
		title:       decodeMarkdownSyntaxString(event.Title),
		hasTitle:    event.HasTitle,
		label:       label,
		email:       event.AutoLinkEmail,
		ordered:     event.Ordered,
		start:       event.Start,
		tight:       event.Tight,
		checked:     event.Checked,
		header:      event.Header,
		column:      event.Column,
		columns:     event.Columns,
		alignment:   event.Alignment,
		language:    event.Language,
		info:        event.Info,
		alert:       event.AlertKind,
		frontMatter: event.FrontMatterFormat,
		math:        event.MathStyle,
	}
}

func decodeMarkdownSyntaxString(value string) string {
	var output strings.Builder
	for position := 0; position < len(value); {
		if value[position] == '\\' && position+1 < len(value) && isASCIIPunctuation(value[position+1]) {
			output.WriteByte(value[position+1])
			position += 2
			continue
		}
		output.WriteByte(value[position])
		position++
	}
	return output.String()
}
