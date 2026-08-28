package native_test

import (
	"errors"
	"reflect"
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

func collectSemanticEvents(t *testing.T, source []byte) []parser.SemanticEvent {
	t.Helper()
	events := make([]parser.SemanticEvent, 0, 64)
	if err := native.New().WalkSemantic(source, func(event parser.SemanticEvent) error {
		if !event.Range.Valid(len(source)) {
			t.Fatalf("event range = %v, source bytes = %d", event.Range, len(source))
		}
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatalf("WalkSemantic() error = %v", err)
	}
	return events
}

func hasSemanticEvent(events []parser.SemanticEvent, kind parser.SemanticKind, match func(parser.SemanticEvent) bool) bool {
	for _, event := range events {
		if event.Kind == kind && (match == nil || match(event)) {
			return true
		}
	}
	return false
}
