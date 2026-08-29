package native_test

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

var _ parser.SemanticBackend = native.New()

type semanticSnapshot struct {
	phase       parser.SemanticPhase
	kind        parser.SemanticKind
	value       string
	level       int
	destination string
}

func TestWalkSemanticFoundationEvents(t *testing.T) {
	t.Parallel()

	source := []byte("# Hello *world* [docs](guide.md)\n\nfirst\nsecond  \nthird\n")
	events := make([]parser.SemanticEvent, 0, 24)
	err := native.New().WalkSemantic(source, func(event parser.SemanticEvent) error {
		if !event.Range.Valid(len(source)) {
			t.Fatalf("event range = %v, source bytes = %d", event.Range, len(source))
		}
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkSemantic() error = %v", err)
	}

	got := make([]semanticSnapshot, len(events))
	for index, event := range events {
		got[index] = semanticSnapshot{
			phase:       event.Phase,
			kind:        event.Kind,
			value:       event.Value,
			level:       event.Level,
			destination: event.Destination,
		}
	}
	want := []semanticSnapshot{
		{phase: parser.SemanticEnter, kind: parser.SemanticDocument},
		{phase: parser.SemanticEnter, kind: parser.SemanticHeading, level: 1},
		{phase: parser.SemanticLeaf, kind: parser.SemanticText, value: "Hello "},
		{phase: parser.SemanticEnter, kind: parser.SemanticEmphasis},
		{phase: parser.SemanticLeaf, kind: parser.SemanticText, value: "world"},
		{phase: parser.SemanticExit, kind: parser.SemanticEmphasis},
		{phase: parser.SemanticLeaf, kind: parser.SemanticText, value: " "},
		{phase: parser.SemanticEnter, kind: parser.SemanticLink, destination: "guide.md"},
		{phase: parser.SemanticLeaf, kind: parser.SemanticText, value: "docs"},
		{phase: parser.SemanticExit, kind: parser.SemanticLink},
		{phase: parser.SemanticExit, kind: parser.SemanticHeading, level: 1},
		{phase: parser.SemanticEnter, kind: parser.SemanticParagraph},
		{phase: parser.SemanticLeaf, kind: parser.SemanticText, value: "first"},
		{phase: parser.SemanticLeaf, kind: parser.SemanticSoftBreak},
		{phase: parser.SemanticLeaf, kind: parser.SemanticText, value: "second"},
		{phase: parser.SemanticLeaf, kind: parser.SemanticHardBreak},
		{phase: parser.SemanticLeaf, kind: parser.SemanticText, value: "third"},
		{phase: parser.SemanticExit, kind: parser.SemanticParagraph},
		{phase: parser.SemanticExit, kind: parser.SemanticDocument},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("semantic events changed\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestWalkSemanticFoundationInlineTerminals(t *testing.T) {
	t.Parallel()

	source := []byte("escaped \\* &amp; `code` <https://example.com> <em>raw</em> ![alt](image.png)\n")
	events := collectSemanticEvents(t, source)
	if !hasSemanticEvent(events, parser.SemanticText, func(event parser.SemanticEvent) bool { return event.Value == "escaped * & " }) {
		t.Fatalf("decoded semantic text missing from %#v", events)
	}
	if !hasSemanticEvent(events, parser.SemanticCodeSpan, func(event parser.SemanticEvent) bool { return event.Value == "code" }) {
		t.Fatalf("code-span semantic event missing from %#v", events)
	}
	if !hasSemanticEvent(events, parser.SemanticAutoLink, func(event parser.SemanticEvent) bool { return event.Destination == "https://example.com" }) {
		t.Fatalf("autolink semantic event missing from %#v", events)
	}
	if !hasSemanticEvent(events, parser.SemanticRawHTML, func(event parser.SemanticEvent) bool { return event.Value == "<em>" }) {
		t.Fatalf("raw HTML semantic event missing from %#v", events)
	}
	if !hasSemanticEvent(events, parser.SemanticImage, func(event parser.SemanticEvent) bool { return event.Destination == "image.png" }) {
		t.Fatalf("image semantic event missing from %#v", events)
	}
}

func TestWalkSemanticFoundationBlockAndSupplementalFamilies(t *testing.T) {
	t.Parallel()

	source := []byte("> quote\n\n- [x] task\n- item\n\n| A | B |\n| --- | :---: |\n| x | y |\n\n---\n\n```go\nfmt.Println(\"x\")\n```\n\n<div>raw</div>\n\n[^note]: footnote body\n\nreference[^note] and $x$\n")
	events := collectSemanticEvents(t, source)
	for _, kind := range []parser.SemanticKind{
		parser.SemanticBlockquote,
		parser.SemanticList,
		parser.SemanticListItem,
		parser.SemanticTaskItem,
		parser.SemanticTable,
		parser.SemanticTableRow,
		parser.SemanticTableCell,
		parser.SemanticThematicBreak,
		parser.SemanticCodeBlock,
		parser.SemanticHTMLBlock,
		parser.SemanticFootnoteDefinition,
		parser.SemanticFootnoteReference,
		parser.SemanticMath,
	} {
		if !hasSemanticEvent(events, kind, nil) {
			t.Fatalf("semantic kind %v missing from %#v", kind, events)
		}
	}
	if !hasSemanticEvent(events, parser.SemanticTaskItem, func(event parser.SemanticEvent) bool { return event.Checked }) {
		t.Fatal("checked task semantics missing")
	}
	if !hasSemanticEvent(events, parser.SemanticTableCell, func(event parser.SemanticEvent) bool {
		return event.Header && event.Column == 1 && event.Alignment == parser.TableAlignmentCenter
	}) {
		t.Fatal("table header/alignment semantics missing")
	}
	if !hasSemanticEvent(events, parser.SemanticCodeBlock, func(event parser.SemanticEvent) bool {
		return event.Fenced && event.Language == "go" && event.Info == "go"
	}) {
		t.Fatal("fenced-code semantics missing")
	}
	if !hasSemanticEvent(events, parser.SemanticFootnoteDefinition, func(event parser.SemanticEvent) bool { return event.Label == "note" }) {
		t.Fatal("footnote-definition semantics missing")
	}
	if !hasSemanticEvent(events, parser.SemanticFootnoteReference, func(event parser.SemanticEvent) bool { return event.Label == "note" }) {
		t.Fatal("footnote-reference semantics missing")
	}
	if !hasSemanticEvent(events, parser.SemanticMath, func(event parser.SemanticEvent) bool {
		return event.Value == "x" && event.MathStyle == parser.MathExpressionInlineDollar
	}) {
		t.Fatal("math semantics missing")
	}
}

func TestM120SemanticDelimiterConsumptionPreservesNestedAndResidualRuns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		want   []semanticSnapshot
	}{
		{
			name:   "nested strong inside emphasis",
			source: "***foo** bar*\n",
			want: []semanticSnapshot{
				{phase: parser.SemanticEnter, kind: parser.SemanticDocument},
				{phase: parser.SemanticEnter, kind: parser.SemanticParagraph},
				{phase: parser.SemanticEnter, kind: parser.SemanticEmphasis},
				{phase: parser.SemanticEnter, kind: parser.SemanticStrong},
				{phase: parser.SemanticLeaf, kind: parser.SemanticText, value: "foo"},
				{phase: parser.SemanticExit, kind: parser.SemanticStrong},
				{phase: parser.SemanticLeaf, kind: parser.SemanticText, value: " bar"},
				{phase: parser.SemanticExit, kind: parser.SemanticEmphasis},
				{phase: parser.SemanticExit, kind: parser.SemanticParagraph},
				{phase: parser.SemanticExit, kind: parser.SemanticDocument},
			},
		},
		{
			name:   "unconsumed opener byte remains text",
			source: "**foo*\n",
			want: []semanticSnapshot{
				{phase: parser.SemanticEnter, kind: parser.SemanticDocument},
				{phase: parser.SemanticEnter, kind: parser.SemanticParagraph},
				{phase: parser.SemanticLeaf, kind: parser.SemanticText, value: "*"},
				{phase: parser.SemanticEnter, kind: parser.SemanticEmphasis},
				{phase: parser.SemanticLeaf, kind: parser.SemanticText, value: "foo"},
				{phase: parser.SemanticExit, kind: parser.SemanticEmphasis},
				{phase: parser.SemanticExit, kind: parser.SemanticParagraph},
				{phase: parser.SemanticExit, kind: parser.SemanticDocument},
			},
		},
		{
			name:   "same-run strong nesting",
			source: "****foo****\n",
			want: []semanticSnapshot{
				{phase: parser.SemanticEnter, kind: parser.SemanticDocument},
				{phase: parser.SemanticEnter, kind: parser.SemanticParagraph},
				{phase: parser.SemanticEnter, kind: parser.SemanticStrong},
				{phase: parser.SemanticEnter, kind: parser.SemanticStrong},
				{phase: parser.SemanticLeaf, kind: parser.SemanticText, value: "foo"},
				{phase: parser.SemanticExit, kind: parser.SemanticStrong},
				{phase: parser.SemanticExit, kind: parser.SemanticStrong},
				{phase: parser.SemanticExit, kind: parser.SemanticParagraph},
				{phase: parser.SemanticExit, kind: parser.SemanticDocument},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := semanticSnapshots(collectSemanticEvents(t, []byte(test.source)))
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("semantic events changed\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}

func TestWalkSemanticStopsOnVisitorError(t *testing.T) {
	t.Parallel()

	stop := errors.New("stop semantic walk")
	calls := 0
	err := native.New().WalkSemantic([]byte("# heading\n"), func(parser.SemanticEvent) error {
		calls++
		if calls == 2 {
			return stop
		}
		return nil
	})
	if !errors.Is(err, stop) {
		t.Fatalf("WalkSemantic() error = %v, want %v", err, stop)
	}
	if calls != 2 {
		t.Fatalf("visitor calls = %d, want 2", calls)
	}
}

func TestWalkSemanticRejectsNilVisitor(t *testing.T) {
	t.Parallel()

	if err := native.New().WalkSemantic([]byte("text\n"), nil); !errors.Is(err, parser.ErrSemanticVisitorRequired) {
		t.Fatalf("WalkSemantic(nil) error = %v, want ErrSemanticVisitorRequired", err)
	}
}

func TestWalkSemanticEmptyDocumentIsBalanced(t *testing.T) {
	t.Parallel()

	got := collectSemanticEvents(t, nil)
	want := []parser.SemanticEvent{
		{Phase: parser.SemanticEnter, Kind: parser.SemanticDocument},
		{Phase: parser.SemanticExit, Kind: parser.SemanticDocument},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("empty semantic events = %#v, want %#v", got, want)
	}
}

func TestWalkSemanticIsDeterministicAndRetainsNoSourceSlice(t *testing.T) {
	t.Parallel()

	source := []byte("# A &amp; B\n\n- [x] item\n\n| A | B |\n| --- | --- |\n| x | y |\n")
	first := collectSemanticEvents(t, source)
	second := collectSemanticEvents(t, source)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("semantic walk is nondeterministic\nfirst:  %#v\nsecond: %#v", first, second)
	}
	for index := range source {
		source[index] = 'z'
	}
	for _, event := range first {
		if event.Kind == parser.SemanticText && event.Value == "A & B" {
			return
		}
	}
	t.Fatalf("retained semantic values changed after caller source mutation: %#v", first)
}

func TestM119SemanticPathologicalInputsRemainBalanced(t *testing.T) {
	t.Parallel()

	deepQuote := []byte(strings.Repeat("> ", 256) + "body\n")
	for _, tt := range []struct {
		name   string
		source []byte
	}{
		{name: "deep blockquote", source: deepQuote},
		{name: "crlf overlays", source: []byte("---\r\ntitle: demo\r\n---\r\n\r\n> [!NOTE]\r\n> body\r\n\r\n`a\r\nb`\r\n")},
		{name: "unclosed syntax", source: []byte("```go\n[broken **~~` <https://example.invalid\n")},
		{name: "invalid utf8", source: []byte{0xff, 0x00, '>', ' ', '[', '!', 'N', 'O', 'T', 'E', ']', '\n', 0xfe}},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			first := collectSemanticEvents(t, tt.source)
			second := collectSemanticEvents(t, tt.source)
			if !reflect.DeepEqual(first, second) {
				t.Fatalf("semantic walk is nondeterministic\nfirst:  %#v\nsecond: %#v", first, second)
			}
			assertSemanticEventsBalanced(t, first)
		})
	}
}

func FuzzM119SemanticWalkRemainsSourceBound(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("# Title\n\nparagraph with *em* [link](<target>)\n"),
		[]byte("---\ntitle: demo\n---\n\n> [!NOTE]\n> body\n\n- [x] task\n"),
		[]byte("[^n]: *note*\n\nuse[^n] and $x$ plus $`y`$\n"),
		[]byte("| A | B |\n| :- | -: |\n| x | y |\n"),
		[]byte("``\nfoo\nbar  \nbaz\n``\n"),
		{0xff, 0x00, '#', ' ', 0xfe, '\r', '\n'},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source []byte) {
		if len(source) > 64<<10 {
			return
		}
		before := bytes.Clone(source)
		first := collectSemanticEvents(t, source)
		if !bytes.Equal(source, before) {
			t.Fatal("WalkSemantic() mutated fuzz source")
		}
		second := collectSemanticEvents(t, source)
		if !reflect.DeepEqual(first, second) {
			t.Fatal("WalkSemantic() is nondeterministic for fuzz source")
		}
		assertSemanticEventsBalanced(t, first)
	})
}

func TestWalkSemanticStopsDuringSupplementalEvents(t *testing.T) {
	t.Parallel()

	stop := errors.New("stop supplemental semantic walk")
	calls := 0
	err := native.New().WalkSemantic([]byte("[^n]: note\n\nreference[^n] and $x$\n"), func(event parser.SemanticEvent) error {
		calls++
		if event.Kind == parser.SemanticFootnoteDefinition {
			return stop
		}
		return nil
	})
	if !errors.Is(err, stop) {
		t.Fatalf("WalkSemantic() supplemental error = %v, want %v", err, stop)
	}
	if calls == 0 {
		t.Fatal("supplemental visitor was not called")
	}
}

func TestSemanticVocabularyReservesM119PolicyFamilies(t *testing.T) {
	t.Parallel()

	for _, kind := range []parser.SemanticKind{parser.SemanticReferenceDefinition, parser.SemanticAlert, parser.SemanticFrontMatter} {
		if kind == parser.SemanticUnknown {
			t.Fatalf("semantic policy kind %v is not reserved", kind)
		}
	}
}

func TestM119SemanticNestedBlockOwnershipAndListFacts(t *testing.T) {
	t.Parallel()

	source := []byte("> outer\n>\n> 3. first\n>    - child\n> 4. second\n")
	events := collectSemanticEvents(t, source)
	got := semanticPhaseKinds(events)
	want := []semanticPhaseKind{
		{parser.SemanticEnter, parser.SemanticDocument},
		{parser.SemanticEnter, parser.SemanticBlockquote},
		{parser.SemanticEnter, parser.SemanticParagraph},
		{parser.SemanticLeaf, parser.SemanticText},
		{parser.SemanticExit, parser.SemanticParagraph},
		{parser.SemanticEnter, parser.SemanticList},
		{parser.SemanticEnter, parser.SemanticListItem},
		{parser.SemanticEnter, parser.SemanticParagraph},
		{parser.SemanticLeaf, parser.SemanticText},
		{parser.SemanticExit, parser.SemanticParagraph},
		{parser.SemanticEnter, parser.SemanticList},
		{parser.SemanticEnter, parser.SemanticListItem},
		{parser.SemanticEnter, parser.SemanticParagraph},
		{parser.SemanticLeaf, parser.SemanticText},
		{parser.SemanticExit, parser.SemanticParagraph},
		{parser.SemanticExit, parser.SemanticListItem},
		{parser.SemanticExit, parser.SemanticList},
		{parser.SemanticExit, parser.SemanticListItem},
		{parser.SemanticEnter, parser.SemanticListItem},
		{parser.SemanticEnter, parser.SemanticParagraph},
		{parser.SemanticLeaf, parser.SemanticText},
		{parser.SemanticExit, parser.SemanticParagraph},
		{parser.SemanticExit, parser.SemanticListItem},
		{parser.SemanticExit, parser.SemanticList},
		{parser.SemanticExit, parser.SemanticBlockquote},
		{parser.SemanticExit, parser.SemanticDocument},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("semantic block hierarchy changed\ngot:  %#v\nwant: %#v", got, want)
	}
	lists := semanticEventsOfKind(events, parser.SemanticList, parser.SemanticEnter)
	if len(lists) != 2 {
		t.Fatalf("list enter events = %d, want 2", len(lists))
	}
	if !lists[0].Ordered || lists[0].Start != 3 || !lists[0].Tight {
		t.Fatalf("outer list facts = %#v, want ordered start=3 tight", lists[0])
	}
	if lists[1].Ordered || lists[1].Start != 0 || !lists[1].Tight {
		t.Fatalf("nested list facts = %#v, want unordered tight", lists[1])
	}
}

func TestM119SemanticLooseListFacts(t *testing.T) {
	t.Parallel()

	events := collectSemanticEvents(t, []byte("3. one\n\n4. two\n"))
	lists := semanticEventsOfKind(events, parser.SemanticList, parser.SemanticEnter)
	if len(lists) != 1 {
		t.Fatalf("list enter events = %d, want 1", len(lists))
	}
	if !lists[0].Ordered || lists[0].Start != 3 || lists[0].Tight {
		t.Fatalf("loose list facts = %#v, want ordered start=3 loose", lists[0])
	}
}

func TestM119SemanticIndentedCodeAndReferenceDefinition(t *testing.T) {
	t.Parallel()

	source := []byte("    code\n    next\n\n[label]: /target \"title\"\n\n[use][label]\n")
	events := collectSemanticEvents(t, source)
	if !hasSemanticEvent(events, parser.SemanticCodeBlock, func(event parser.SemanticEvent) bool {
		return !event.Fenced && event.Value == "code\nnext\n"
	}) {
		t.Fatalf("indented code semantics missing from %#v", events)
	}
	if !hasSemanticEvent(events, parser.SemanticReferenceDefinition, func(event parser.SemanticEvent) bool {
		return event.Label == "label" && event.Destination == "/target" && event.Title == "title" && event.HasTitle
	}) {
		t.Fatalf("reference-definition semantics missing from %#v", events)
	}
}

func TestM119SemanticFrontMatterEnvelope(t *testing.T) {
	t.Parallel()

	source := []byte("---\ntitle: demo\n---\n\n# Body\n")
	events := collectSemanticEvents(t, source)
	got := semanticPhaseKinds(events)
	want := []semanticPhaseKind{
		{parser.SemanticEnter, parser.SemanticDocument},
		{parser.SemanticLeaf, parser.SemanticFrontMatter},
		{parser.SemanticEnter, parser.SemanticHeading},
		{parser.SemanticLeaf, parser.SemanticText},
		{parser.SemanticExit, parser.SemanticHeading},
		{parser.SemanticExit, parser.SemanticDocument},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("front-matter semantic hierarchy changed\ngot:  %#v\nwant: %#v", got, want)
	}
	frontMatter := semanticEventsOfKind(events, parser.SemanticFrontMatter, parser.SemanticLeaf)
	if len(frontMatter) != 1 {
		t.Fatalf("front-matter events = %d, want 1", len(frontMatter))
	}
	if frontMatter[0].FrontMatterFormat != parser.SemanticFrontMatterYAML || frontMatter[0].Value != "---\ntitle: demo\n---" {
		t.Fatalf("front-matter event = %#v", frontMatter[0])
	}
}

func TestM119SemanticAlertOwnsBodyWithoutMarkerText(t *testing.T) {
	t.Parallel()

	source := []byte("> [!NOTE]\n> **Body**\n")
	events := collectSemanticEvents(t, source)
	got := semanticPhaseKinds(events)
	want := []semanticPhaseKind{
		{parser.SemanticEnter, parser.SemanticDocument},
		{parser.SemanticEnter, parser.SemanticAlert},
		{parser.SemanticEnter, parser.SemanticParagraph},
		{parser.SemanticEnter, parser.SemanticStrong},
		{parser.SemanticLeaf, parser.SemanticText},
		{parser.SemanticExit, parser.SemanticStrong},
		{parser.SemanticExit, parser.SemanticParagraph},
		{parser.SemanticExit, parser.SemanticAlert},
		{parser.SemanticExit, parser.SemanticDocument},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("alert semantic hierarchy changed\ngot:  %#v\nwant: %#v", got, want)
	}
	alerts := semanticEventsOfKind(events, parser.SemanticAlert, parser.SemanticEnter)
	if len(alerts) != 1 || alerts[0].AlertKind != parser.SemanticAlertNote {
		t.Fatalf("alert event = %#v, want one NOTE", alerts)
	}
	if hasSemanticEvent(events, parser.SemanticText, func(event parser.SemanticEvent) bool { return event.Value == "[!NOTE]" }) {
		t.Fatalf("alert marker leaked as semantic text: %#v", events)
	}
}

func TestM119UnknownAlertMarkerRemainsBlockquote(t *testing.T) {
	t.Parallel()

	events := collectSemanticEvents(t, []byte("> [!CUSTOM]\n> body\n"))
	if hasSemanticEvent(events, parser.SemanticAlert, nil) {
		t.Fatalf("unknown alert marker promoted: %#v", events)
	}
	if !hasSemanticEvent(events, parser.SemanticBlockquote, nil) {
		t.Fatalf("unknown alert marker lost blockquote semantics: %#v", events)
	}
}

func TestM119SemanticFootnoteDefinitionAndReferenceInPlace(t *testing.T) {
	t.Parallel()

	events := collectSemanticEvents(t, []byte("before[^n] after\n\n[^n]: *note* body\n"))
	before := semanticEventIndex(events, parser.SemanticText, func(event parser.SemanticEvent) bool { return event.Value == "before" })
	reference := semanticEventIndex(events, parser.SemanticFootnoteReference, func(event parser.SemanticEvent) bool { return event.Label == "n" })
	after := semanticEventIndex(events, parser.SemanticText, func(event parser.SemanticEvent) bool { return event.Value == " after" })
	definition := semanticEventIndex(events, parser.SemanticFootnoteDefinition, func(event parser.SemanticEvent) bool {
		return event.Label == "n" && event.Phase == parser.SemanticEnter
	})
	if before < 0 || reference <= before || after <= reference || definition <= after {
		t.Fatalf("footnote source order invalid: before=%d reference=%d after=%d definition=%d events=%#v", before, reference, after, definition, events)
	}
	if hasSemanticEvent(events, parser.SemanticReferenceDefinition, func(event parser.SemanticEvent) bool { return event.Label == "^n" }) {
		t.Fatalf("footnote leaked ordinary reference-definition semantics: %#v", events)
	}
	if !hasSemanticEvent(events[definition:], parser.SemanticEmphasis, nil) {
		t.Fatalf("footnote body emphasis missing: %#v", events)
	}
}

func TestM119SemanticInlineMathReplacesSourceText(t *testing.T) {
	t.Parallel()

	events := collectSemanticEvents(t, []byte("a $x$ b $`y`$ c\n"))
	got := make([]string, 0)
	for _, event := range events {
		switch event.Kind {
		case parser.SemanticText:
			got = append(got, "text:"+event.Value)
		case parser.SemanticMath:
			got = append(got, fmt.Sprintf("math:%d:%s", event.MathStyle, event.Value))
		}
	}
	want := []string{
		"text:a ",
		fmt.Sprintf("math:%d:x", parser.MathExpressionInlineDollar),
		"text: b ",
		fmt.Sprintf("math:%d:y", parser.MathExpressionInlineBacktick),
		"text: c",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("inline math semantic sequence = %#v, want %#v", got, want)
	}
}

func TestM119SemanticBlockMathReplacesParagraph(t *testing.T) {
	t.Parallel()

	events := collectSemanticEvents(t, []byte("$$x+y$$\n"))
	got := semanticPhaseKinds(events)
	want := []semanticPhaseKind{
		{parser.SemanticEnter, parser.SemanticDocument},
		{parser.SemanticLeaf, parser.SemanticMath},
		{parser.SemanticExit, parser.SemanticDocument},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("block math hierarchy = %#v, want %#v", got, want)
	}
	math := semanticEventsOfKind(events, parser.SemanticMath, parser.SemanticLeaf)
	if len(math) != 1 || math[0].MathStyle != parser.MathExpressionBlockDollar || math[0].Value != "x+y" {
		t.Fatalf("block math event = %#v", math)
	}
}

func TestM119SemanticTaskMarkerIsMetadataNotText(t *testing.T) {
	t.Parallel()

	events := collectSemanticEvents(t, []byte("- [ ] foo\n- [x] bar\n"))
	texts := make([]string, 0)
	checks := make([]bool, 0)
	for _, event := range events {
		switch event.Kind {
		case parser.SemanticText:
			texts = append(texts, event.Value)
		case parser.SemanticTaskItem:
			checks = append(checks, event.Checked)
		}
	}
	if want := []string{" foo", " bar"}; !reflect.DeepEqual(texts, want) {
		t.Fatalf("task semantic text = %#v, want %#v", texts, want)
	}
	if want := []bool{false, true}; !reflect.DeepEqual(checks, want) {
		t.Fatalf("task checked states = %#v, want %#v", checks, want)
	}
}

func TestM119SemanticEmailAutolinkCarriesEmailFact(t *testing.T) {
	t.Parallel()

	events := collectSemanticEvents(t, []byte("<foo@bar.example.com>\n"))
	links := semanticEventsOfKind(events, parser.SemanticAutoLink, parser.SemanticLeaf)
	if len(links) != 1 {
		t.Fatalf("autolink events = %d, want 1: %#v", len(links), events)
	}
	if links[0].Value != "foo@bar.example.com" || links[0].Destination != "mailto:foo@bar.example.com" || !links[0].AutoLinkEmail {
		t.Fatalf("email autolink = %#v", links[0])
	}
}

func TestM119SemanticCodeSpanValueMatchesCommonMarkNormalization(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		source string
		want   string
	}{
		{name: "example 328", source: "`foo`\n", want: "foo"},
		{name: "example 329", source: "`` foo ` bar ``\n", want: "foo ` bar"},
		{name: "example 331", source: "`  ``  `\n", want: " `` "},
		{name: "example 335", source: "``\nfoo\nbar  \nbaz\n``\n", want: "foo bar   baz"},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			events := collectSemanticEvents(t, []byte(tt.source))
			spans := semanticEventsOfKind(events, parser.SemanticCodeSpan, parser.SemanticLeaf)
			if len(spans) != 1 || spans[0].Value != tt.want {
				t.Fatalf("code span = %#v, want value %q", spans, tt.want)
			}
		})
	}
}

func semanticEventIndex(events []parser.SemanticEvent, kind parser.SemanticKind, match func(parser.SemanticEvent) bool) int {
	for index, event := range events {
		if event.Kind == kind && (match == nil || match(event)) {
			return index
		}
	}
	return -1
}

type semanticPhaseKind struct {
	phase parser.SemanticPhase
	kind  parser.SemanticKind
}

func semanticSnapshots(events []parser.SemanticEvent) []semanticSnapshot {
	result := make([]semanticSnapshot, len(events))
	for index, event := range events {
		result[index] = semanticSnapshot{
			phase:       event.Phase,
			kind:        event.Kind,
			value:       event.Value,
			level:       event.Level,
			destination: event.Destination,
		}
	}
	return result
}

func semanticPhaseKinds(events []parser.SemanticEvent) []semanticPhaseKind {
	result := make([]semanticPhaseKind, len(events))
	for index, event := range events {
		result[index] = semanticPhaseKind{phase: event.Phase, kind: event.Kind}
	}
	return result
}

func semanticEventsOfKind(events []parser.SemanticEvent, kind parser.SemanticKind, phase parser.SemanticPhase) []parser.SemanticEvent {
	result := make([]parser.SemanticEvent, 0)
	for _, event := range events {
		if event.Kind == kind && event.Phase == phase {
			result = append(result, event)
		}
	}
	return result
}

func collectSemanticEvents(t *testing.T, source []byte) []parser.SemanticEvent {
	t.Helper()
	events := make([]parser.SemanticEvent, 0, 64)
	if err := native.New().WalkSemantic(source, func(event parser.SemanticEvent) error {
		if !event.Range.Valid(len(source)) {
			t.Fatalf("event range = %v, source bytes = %d", event.Range, len(source))
		}
		if event.ContentRange != (parser.Range{}) && !event.ContentRange.Valid(len(source)) {
			t.Fatalf("event content range = %v, source bytes = %d", event.ContentRange, len(source))
		}
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatalf("WalkSemantic() error = %v", err)
	}
	return events
}

func assertSemanticEventsBalanced(t *testing.T, events []parser.SemanticEvent) {
	t.Helper()
	stack := make([]parser.SemanticKind, 0, 32)
	for index, event := range events {
		switch event.Phase {
		case parser.SemanticEnter:
			stack = append(stack, event.Kind)
		case parser.SemanticExit:
			if len(stack) == 0 || stack[len(stack)-1] != event.Kind {
				t.Fatalf("event %d unbalanced exit kind=%d stack=%v", index, event.Kind, stack)
			}
			stack = stack[:len(stack)-1]
		case parser.SemanticLeaf:
		default:
			t.Fatalf("event %d has unknown phase %d", index, event.Phase)
		}
	}
	if len(stack) != 0 {
		t.Fatalf("semantic stack remains open: %v", stack)
	}
}

func hasSemanticEvent(events []parser.SemanticEvent, kind parser.SemanticKind, match func(parser.SemanticEvent) bool) bool {
	for _, event := range events {
		if event.Kind == kind && (match == nil || match(event)) {
			return true
		}
	}
	return false
}
