package rendermarkdown

import (
	"reflect"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

func TestEmitCanonicalInlineASTUsesTreeAndExplicitMarkerPlan(t *testing.T) {
	events := []parser.SemanticEvent{
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "a"},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticStrong},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "b"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticStrong},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis},
	}
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	plan := newCanonicalInlinePlan(ast)
	plan.nodes[1].marker = '*'
	plan.nodes[3].marker = '_'
	got, err := emitCanonicalInlineAST(ast, normalization, plan, canonicalInlineEmitContext{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "*a__b__*" {
		t.Fatalf("canonical inline = %q", got)
	}
}

func TestNewCanonicalInlinePlanDefaultsStrikethroughToDoubleTilde(t *testing.T) {
	events := []parser.SemanticEvent{
		{Phase: parser.SemanticEnter, Kind: parser.SemanticStrikethrough},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "x"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticStrikethrough},
	}
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	plan := newCanonicalInlinePlan(ast)
	if got := plan.nodes[1].strikethroughWidth; got != 2 {
		t.Fatalf("strikethrough width = %d, want 2", got)
	}
	got, err := emitCanonicalInlineAST(ast, normalization, plan, canonicalInlineEmitContext{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "~~x~~" {
		t.Fatalf("canonical inline = %q, want %q", got, "~~x~~")
	}
}

func TestCanonicalInlineWrapperDelimitersRejectsUnplannedStrikethroughWidth(t *testing.T) {
	if _, _, err := canonicalInlineWrapperDelimiters(parser.SemanticStrikethrough, 0, 0); err == nil {
		t.Fatal("expected zero strikethrough width to be rejected")
	}
}

func TestEmitCanonicalInlineASTKeepsContainerLocalTextState(t *testing.T) {
	events := []parser.SemanticEvent{
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: " x"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis},
	}
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	plan := newCanonicalInlinePlan(ast)
	plan.nodes[1].marker = '*'
	got, err := emitCanonicalInlineAST(ast, normalization, plan, canonicalInlineEmitContext{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "*&#32;x*" {
		t.Fatalf("canonical inline = %q", got)
	}
}

func TestEmitCanonicalInlineASTUsesValidatedRawTabPayloadChoice(t *testing.T) {
	events := []parser.SemanticEvent{
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "!\t"},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "x"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis},
	}
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	plan := newCanonicalInlinePlan(ast)
	plan.nodes[1].payload = canonicalInlineRawTabPayload
	plan.nodes[2].marker = '*'
	got, err := emitCanonicalInlineAST(ast, normalization, plan, canonicalInlineEmitContext{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "\\!\t*x*" {
		t.Fatalf("canonical inline = %q", got)
	}
}

func TestEmitCanonicalInlineASTUsesRawTabAfterBareWWWContinuation(t *testing.T) {
	events := []parser.SemanticEvent{
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticAutoLink, Value: "www.example.com", Destination: "http://www.example.com", AutoLinkForm: parser.AutoLinkExtendedWWW},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: ")\t"},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "x"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis},
	}
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	if got := normalization.nodes[2].trailingAlternative; got != canonicalInlineRawTabBoundary {
		t.Fatalf("trailing alternative = %d, want raw TAB", got)
	}
	plan := newCanonicalInlinePlan(ast)
	plan.nodes[2].payload = canonicalInlineRawTabPayload
	plan.nodes[3].marker = '*'
	got, err := emitCanonicalInlineAST(ast, normalization, plan, canonicalInlineEmitContext{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := "www.example.com)\t*x*"; string(got) != want {
		t.Fatalf("canonical inline = %q, want %q", got, want)
	}
}

func TestEmitCanonicalInlineASTRejectsUnavailablePayloadChoice(t *testing.T) {
	events := []parser.SemanticEvent{
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "!\t"},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "x"},
	}
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	plan := newCanonicalInlinePlan(ast)
	plan.nodes[1].payload = canonicalInlineRawTabPayload
	if _, err := emitCanonicalInlineAST(ast, normalization, plan, canonicalInlineEmitContext{}, false); err == nil {
		t.Fatal("expected unavailable payload choice error")
	}
}

func TestEmitCanonicalInlineASTRendersInlineMathFromSemanticStyle(t *testing.T) {
	for _, test := range []struct {
		name  string
		style parser.MathExpressionStyle
		want  string
	}{
		{name: "dollar", style: parser.MathExpressionInlineDollar, want: "$x+y$"},
		{name: "backtick", style: parser.MathExpressionInlineBacktick, want: "$`x+y`$"},
	} {
		t.Run(test.name, func(t *testing.T) {
			events := []parser.SemanticEvent{{
				Phase:     parser.SemanticLeaf,
				Kind:      parser.SemanticMath,
				Value:     "x+y",
				MathStyle: test.style,
			}}
			ast, err := buildCanonicalInlineAST(events)
			if err != nil {
				t.Fatal(err)
			}
			normalization, err := normalizeCanonicalInlineAST(ast)
			if err != nil {
				t.Fatal(err)
			}
			got, err := emitCanonicalInlineAST(ast, normalization, newCanonicalInlinePlan(ast), canonicalInlineEmitContext{}, false)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want {
				t.Fatalf("canonical inline = %q, want %q", got, test.want)
			}
		})
	}
}

func TestEmitCanonicalInlineASTRejectsBlockMathInsideInlineHost(t *testing.T) {
	events := []parser.SemanticEvent{{
		Phase:     parser.SemanticLeaf,
		Kind:      parser.SemanticMath,
		Value:     "x",
		MathStyle: parser.MathExpressionBlockDollar,
	}}
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := emitCanonicalInlineAST(ast, normalization, newCanonicalInlinePlan(ast), canonicalInlineEmitContext{}, false); err == nil {
		t.Fatal("expected block math rejection")
	}
}

func TestEmitCanonicalInlineASTRendersAutoLinksFromSemanticFacts(t *testing.T) {
	for _, test := range []struct {
		name     string
		event    parser.SemanticEvent
		trailing string
		want     string
	}{
		{
			name:  "explicit URI uses angle form",
			event: parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticAutoLink, Value: "https://example.test/a", Destination: "https://example.test/a", AutoLinkForm: parser.AutoLinkExplicitURI},
			want:  "<https://example.test/a>",
		},
		{
			name:  "explicit email uses angle form",
			event: parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticAutoLink, Value: "foo@example.test", Destination: "mailto:foo@example.test", AutoLinkEmail: true, AutoLinkForm: parser.AutoLinkExplicitEmail},
			want:  "<foo@example.test>",
		},
		{
			name:     "GFM extended www stays extension form",
			event:    parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticAutoLink, Value: "www.example.test", Destination: "http://www.example.test", AutoLinkForm: parser.AutoLinkExtendedWWW},
			trailing: ") ",
			want:     "www.example.test) ",
		},
		{
			name:  "GFM extended URL stays extension form",
			event: parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticAutoLink, Value: "https://example.test/a", Destination: "https://example.test/a", AutoLinkForm: parser.AutoLinkExtendedURL},
			want:  "https://example.test/a",
		},
		{
			name:  "GFM extended email stays extension form",
			event: parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticAutoLink, Value: "foo@example.test", Destination: "mailto:foo@example.test", AutoLinkEmail: true, AutoLinkForm: parser.AutoLinkExtendedEmail},
			want:  "foo@example.test",
		},
		{
			name:  "GFM extended protocol stays extension form",
			event: parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticAutoLink, Value: "mailto:foo@example.test", Destination: "mailto:foo@example.test", AutoLinkForm: parser.AutoLinkExtendedProtocol},
			want:  "mailto:foo@example.test",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			events := []parser.SemanticEvent{test.event}
			if test.trailing != "" {
				events = append(events, parser.SemanticEvent{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: test.trailing})
			}
			ast, err := buildCanonicalInlineAST(events)
			if err != nil {
				t.Fatal(err)
			}
			normalization, err := normalizeCanonicalInlineAST(ast)
			if err != nil {
				t.Fatal(err)
			}
			got, err := emitCanonicalInlineAST(ast, normalization, newCanonicalInlinePlan(ast), canonicalInlineEmitContext{}, false)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want {
				t.Fatalf("canonical inline = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCanonicalInlineASTStateDistinguishesBareAutoLinkLatentTails(t *testing.T) {
	newState := func(tail string) canonicalInlineASTState {
		t.Helper()
		state := newCanonicalInlineASTState()
		if !state.startBareWWWContinuation("www.example.com") {
			t.Fatal("failed to start bare-www continuation")
		}
		state.observe([]byte(tail))
		return state
	}

	star := newState("*")
	close := newState(")")
	starEnd, starOwner := star.bareWWW.continuation.Owner()
	closeEnd, closeOwner := close.bareWWW.continuation.Owner()
	if !starOwner || !closeOwner || starEnd != closeEnd {
		t.Fatalf("precondition mismatch: star=(%d,%t) close=(%d,%t)", starEnd, starOwner, closeEnd, closeOwner)
	}

	starPrefix := star.bareWWW.rawTextPrefixEnd(")")
	closePrefix := close.bareWWW.rawTextPrefixEnd(")")
	if starPrefix != 0 || closePrefix != 1 {
		t.Fatalf("raw prefix after same append: star=%d close=%d, want 0 and 1", starPrefix, closePrefix)
	}

	entity := newCanonicalInlineASTState()
	if !entity.startBareWWWContinuation("www.example.com") {
		t.Fatal("failed to start entity bare-www continuation")
	}
	if got := entity.bareWWW.rawTextPrefixEnd("&amp;x"); got != len("&amp;") {
		t.Fatalf("entity raw prefix = %d, want %d", got, len("&amp;"))
	}

	star.observe([]byte(")"))
	close.observe([]byte(")"))
	starEnd, starOwner = star.bareWWW.continuation.Owner()
	closeEnd, closeOwner = close.bareWWW.continuation.Owner()
	if !starOwner || !closeOwner || starEnd != len("www.example.com*") || closeEnd != len("www.example.com") {
		t.Fatalf("latent transition mismatch: star=(%d,%t) close=(%d,%t)", starEnd, starOwner, closeEnd, closeOwner)
	}
}

func TestEmitCanonicalInlineASTPreservesBareAutoLinkSemanticValueReclaimedByFollowingText(t *testing.T) {
	const source = "~~www.example.com*)~~"
	backend := native.New()
	events := firstParagraphInlineEvents(t, backend, []byte(source))
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	got, err := emitCanonicalInlineAST(
		ast,
		normalization,
		newCanonicalInlinePlan(ast),
		canonicalInlineEmitContext{},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != source {
		t.Fatalf("canonical inline = %q, want %q", got, source)
	}
	wantFacts := collectSemanticFacts(t, backend, []byte(source))
	gotFacts := collectSemanticFacts(t, backend, got)
	if !reflect.DeepEqual(gotFacts, wantFacts) {
		t.Fatalf("semantic round trip mismatch\nsource: %q\ncandidate: %q\nwant: %#v\ngot: %#v", source, got, wantFacts, gotFacts)
	}
}

func TestEmitCanonicalInlineASTPreservesBareAutoLinkTailAcrossDelimiterWrapper(t *testing.T) {
	const source = "www.example.com*_*_"
	backend := native.New()
	events := firstParagraphInlineEvents(t, backend, []byte(source))
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	plan := newCanonicalInlinePlan(ast)
	for nodeIndex := 1; nodeIndex < len(ast.nodes); nodeIndex++ {
		event := ast.events[ast.nodes[nodeIndex].eventIndex]
		if event.Phase == parser.SemanticEnter && event.Kind == parser.SemanticEmphasis {
			plan.nodes[nodeIndex].marker = '_'
		}
	}
	got, err := emitCanonicalInlineAST(
		ast,
		normalization,
		plan,
		canonicalInlineEmitContext{},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != source {
		t.Fatalf("canonical inline = %q, want %q", got, source)
	}
	wantFacts := collectSemanticFacts(t, backend, []byte(source))
	gotFacts := collectSemanticFacts(t, backend, got)
	if !reflect.DeepEqual(gotFacts, wantFacts) {
		t.Fatalf("semantic round trip mismatch\nsource: %q\ncandidate: %q\nwant: %#v\ngot: %#v", source, got, wantFacts, gotFacts)
	}
}

func TestEmitCanonicalInlineASTUsesSingleTildeWhenBareAutoLinkGeometryRequiresIt(t *testing.T) {
	const source = "www.example.com*~*~~*~"
	backend := native.New()
	events := firstParagraphInlineEvents(t, backend, []byte(source))
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	plan := newCanonicalInlinePlan(ast)
	for nodeIndex := 1; nodeIndex < len(ast.nodes); nodeIndex++ {
		event := ast.events[ast.nodes[nodeIndex].eventIndex]
		if event.Phase != parser.SemanticEnter {
			continue
		}
		switch event.Kind {
		case parser.SemanticEmphasis:
			plan.nodes[nodeIndex].marker = '*'
		case parser.SemanticStrikethrough:
			plan.nodes[nodeIndex].strikethroughWidth = 1
		}
	}
	got, err := emitCanonicalInlineAST(
		ast,
		normalization,
		plan,
		canonicalInlineEmitContext{},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != source {
		t.Fatalf("canonical inline = %q, want %q", got, source)
	}
	wantFacts := collectSemanticFacts(t, backend, []byte(source))
	gotFacts := collectSemanticFacts(t, backend, got)
	if !reflect.DeepEqual(gotFacts, wantFacts) {
		t.Fatalf("semantic round trip mismatch\nsource: %q\ncandidate: %q\nwant: %#v\ngot: %#v", source, got, wantFacts, gotFacts)
	}
}

func TestEmitCanonicalInlineASTGuardsRawHTMLOnlyAfterPhysicalInlineBreak(t *testing.T) {
	tests := []struct {
		name   string
		events []parser.SemanticEvent
		want   string
	}{
		{
			name: "block tag after soft break",
			events: []parser.SemanticEvent{
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "a"},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticSoftBreak},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticRawHTML, Value: "<div>"},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "x"},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticRawHTML, Value: "</div>"},
			},
			want: "a\n    <div>x</div>",
		},
		{
			name: "comment after hard break",
			events: []parser.SemanticEvent{
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "a"},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticHardBreak},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticRawHTML, Value: "<!--x-->"},
			},
			want: "a\\\n    <!--x-->",
		},
		{
			name: "inline tag at host start",
			events: []parser.SemanticEvent{
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticRawHTML, Value: "<em>"},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "x"},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticRawHTML, Value: "</em>"},
			},
			want: "<em>x</em>",
		},
		{
			name: "normalize raw html line endings",
			events: []parser.SemanticEvent{
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "a"},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticSoftBreak},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticRawHTML, Value: "<span\r\ndata-x=x>"},
			},
			want: "a\n    <span\ndata-x=x>",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ast, err := buildCanonicalInlineAST(test.events)
			if err != nil {
				t.Fatal(err)
			}
			normalization, err := normalizeCanonicalInlineAST(ast)
			if err != nil {
				t.Fatal(err)
			}
			got, err := emitCanonicalInlineAST(ast, normalization, newCanonicalInlinePlan(ast), canonicalInlineEmitContext{}, false)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want {
				t.Fatalf("canonical inline = %q, want %q", got, test.want)
			}
		})
	}
}

func TestEmitCanonicalInlineASTUsesTableCellCodeSpanEscaping(t *testing.T) {
	events := []parser.SemanticEvent{{
		Phase: parser.SemanticLeaf,
		Kind:  parser.SemanticCodeSpan,
		Value: "a|b",
	}}
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	got, err := emitCanonicalInlineAST(ast, normalization, newCanonicalInlinePlan(ast), canonicalInlineEmitContext{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if want := renderCodeSpan("a|b", true); string(got) != want {
		t.Fatalf("canonical inline = %q, want %q", got, want)
	}
}

func TestEmitCanonicalInlineASTRendersLinkImageForms(t *testing.T) {
	tests := []struct {
		name   string
		events []parser.SemanticEvent
		want   string
	}{
		{
			name: "direct link",
			events: []parser.SemanticEvent{
				{Phase: parser.SemanticEnter, Kind: parser.SemanticLink, Destination: "/target", Title: "a \"title\"", HasTitle: true},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "label"},
				{Phase: parser.SemanticExit, Kind: parser.SemanticLink},
			},
			want: "[label](</target> \"a \\\"title\\\"\")",
		},
		{
			name: "reference link",
			events: []parser.SemanticEvent{
				{Phase: parser.SemanticEnter, Kind: parser.SemanticLink, Destination: "/target", Label: "ref"},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "label"},
				{Phase: parser.SemanticExit, Kind: parser.SemanticLink},
			},
			want: "[label][ref]",
		},
		{
			name: "footnote shaped ordinary reference renders direct",
			events: []parser.SemanticEvent{
				{Phase: parser.SemanticEnter, Kind: parser.SemanticLink, Destination: "https://example.test/advisory", Label: "^8"},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "CVE"},
				{Phase: parser.SemanticExit, Kind: parser.SemanticLink},
			},
			want: "[CVE](<https://example.test/advisory>)",
		},
		{
			name: "shortcut image restores parser proven reference spelling",
			events: []parser.SemanticEvent{
				{Phase: parser.SemanticEnter, Kind: parser.SemanticImage, Destination: "/badge", Label: "serde_derive msrv"},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "serde_derive msrv"},
				{Phase: parser.SemanticExit, Kind: parser.SemanticImage},
			},
			want: "![serde_derive msrv]",
		},
		{
			name: "linked shortcut image preserves composite shape",
			events: []parser.SemanticEvent{
				{Phase: parser.SemanticEnter, Kind: parser.SemanticLink, Destination: "/actions", Label: "actions"},
				{Phase: parser.SemanticEnter, Kind: parser.SemanticImage, Destination: "/badge", Label: "Build Status"},
				{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "Build Status"},
				{Phase: parser.SemanticExit, Kind: parser.SemanticImage},
				{Phase: parser.SemanticExit, Kind: parser.SemanticLink},
			},
			want: "[![Build Status]][actions]",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ast, err := buildCanonicalInlineAST(test.events)
			if err != nil {
				t.Fatal(err)
			}
			normalization, err := normalizeCanonicalInlineAST(ast)
			if err != nil {
				t.Fatal(err)
			}
			got, err := emitCanonicalInlineAST(
				ast,
				normalization,
				newCanonicalInlinePlan(ast),
				canonicalInlineEmitContext{referenceLabels: native.New()},
				false,
			)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want {
				t.Fatalf("canonical inline = %q, want %q", got, test.want)
			}
		})
	}
}

func TestEmitCanonicalInlineASTLinkImageNativeRoundTrip(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		definitions string
	}{
		{
			name:        "linked shortcut image composite",
			source:      "[![Build Status]][actions]\n\n[Build Status]: /badge\n[actions]: /actions\n",
			definitions: "[Build Status]: /badge\n[actions]: /actions\n",
		},
		{
			name:        "footnote shaped ordinary reference",
			source:      "[CVE][^8] and note[^8]\n\n[^8]: https://example.test/advisory\n",
			definitions: "[^8]: https://example.test/advisory\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backend := native.New()
			events := firstParagraphInlineEvents(t, backend, []byte(test.source))
			ast, err := buildCanonicalInlineAST(events)
			if err != nil {
				t.Fatal(err)
			}
			normalization, err := normalizeCanonicalInlineAST(ast)
			if err != nil {
				t.Fatal(err)
			}
			inline, err := emitCanonicalInlineAST(
				ast,
				normalization,
				newCanonicalInlinePlan(ast),
				canonicalInlineEmitContext{referenceLabels: backend},
				false,
			)
			if err != nil {
				t.Fatal(err)
			}
			candidate := append(append(append([]byte(nil), inline...), '\n', '\n'), []byte(test.definitions)...)
			want := collectSemanticFacts(t, backend, []byte(test.source))
			got := collectSemanticFacts(t, backend, candidate)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("semantic round trip mismatch\nsource: %q\ncandidate: %q\nwant: %#v\ngot: %#v", test.source, candidate, want, got)
			}
		})
	}
}

func firstParagraphInlineEvents(t *testing.T, backend Backend, source []byte) []parser.SemanticEvent {
	t.Helper()
	var events []parser.SemanticEvent
	inParagraph := false
	done := false
	if err := backend.WalkSemantic(source, func(event parser.SemanticEvent) error {
		if done {
			return nil
		}
		if event.Kind == parser.SemanticParagraph {
			switch event.Phase {
			case parser.SemanticEnter:
				if !inParagraph {
					inParagraph = true
				}
			case parser.SemanticExit:
				if inParagraph {
					done = true
					inParagraph = false
				}
			}
			return nil
		}
		if inParagraph {
			events = append(events, event)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("first paragraph has no inline semantic events")
	}
	return events
}
