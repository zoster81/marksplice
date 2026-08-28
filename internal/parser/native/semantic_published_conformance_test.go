package native_test

import (
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
	"github.com/zoster81/marksplice/internal/testutil/commonmarkspec"
	"github.com/zoster81/marksplice/internal/testutil/gfmspec"
)

type publishedSemanticEvent struct {
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
}

type publishedSemanticExpectation struct {
	section string
	events  []publishedSemanticEvent
}

func TestM119PublishedCommonMarkSemanticContract(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_COMMONMARK_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_COMMONMARK_SPEC_HTML is not set")
	}
	cases, err := commonmarkspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published CommonMark spec: %v", err)
	}

	expected := map[int]publishedSemanticExpectation{
		80: {
			section: "setext-headings",
			events: []publishedSemanticEvent{
				publishedEnter(parser.SemanticDocument),
				publishedHeadingEnter(1), publishedText("Foo "), publishedEnter(parser.SemanticEmphasis), publishedText("bar"), publishedExit(parser.SemanticEmphasis), publishedHeadingExit(1),
				publishedHeadingEnter(2), publishedText("Foo "), publishedEnter(parser.SemanticEmphasis), publishedText("bar"), publishedExit(parser.SemanticEmphasis), publishedHeadingExit(2),
				publishedExit(parser.SemanticDocument),
			},
		},
		335: {
			section: "code-spans",
			events: []publishedSemanticEvent{
				publishedEnter(parser.SemanticDocument), publishedEnter(parser.SemanticParagraph), publishedCodeSpan("foo bar   baz"), publishedExit(parser.SemanticParagraph), publishedExit(parser.SemanticDocument),
			},
		},
		527: {
			section: "links",
			events: []publishedSemanticEvent{
				publishedEnter(parser.SemanticDocument), publishedEnter(parser.SemanticParagraph),
				publishedLinkEnter("/url", "title", true, "bar"), publishedText("foo"), publishedExit(parser.SemanticLink),
				publishedExit(parser.SemanticParagraph), publishedReferenceDefinition("bar", "/url", "title", true), publishedExit(parser.SemanticDocument),
			},
		},
		572: {
			section: "images",
			events: []publishedSemanticEvent{
				publishedEnter(parser.SemanticDocument), publishedEnter(parser.SemanticParagraph), publishedImageEnter("/url", "title", true), publishedText("foo"), publishedExit(parser.SemanticImage), publishedExit(parser.SemanticParagraph), publishedExit(parser.SemanticDocument),
			},
		},
		604: {
			section: "autolinks",
			events: []publishedSemanticEvent{
				publishedEnter(parser.SemanticDocument), publishedEnter(parser.SemanticParagraph), publishedAutoLink("foo@bar.example.com", "mailto:foo@bar.example.com", true), publishedExit(parser.SemanticParagraph), publishedExit(parser.SemanticDocument),
			},
		},
		633: {
			section: "hard-line-breaks",
			events: []publishedSemanticEvent{
				publishedEnter(parser.SemanticDocument), publishedEnter(parser.SemanticParagraph), publishedText("foo"), publishedLeaf(parser.SemanticHardBreak), publishedText("baz"), publishedExit(parser.SemanticParagraph), publishedExit(parser.SemanticDocument),
			},
		},
	}

	assertPublishedCommonMarkSemanticCases(t, cases, expected)
}

func TestM119PublishedGFMSemanticContract(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}

	expected := map[int][]publishedSemanticEvent{
		199: {
			publishedEnter(parser.SemanticDocument),
			publishedTableEnter(2), publishedRowEnter(true, 2), publishedCellEnter(true, 0, parser.TableAlignmentCenter), publishedText("abc"), publishedExit(parser.SemanticTableCell), publishedCellEnter(true, 1, parser.TableAlignmentRight), publishedText("defghi"), publishedExit(parser.SemanticTableCell), publishedExit(parser.SemanticTableRow),
			publishedRowEnter(false, 2), publishedCellEnter(false, 0, parser.TableAlignmentCenter), publishedText("bar"), publishedExit(parser.SemanticTableCell), publishedCellEnter(false, 1, parser.TableAlignmentRight), publishedText("baz"), publishedExit(parser.SemanticTableCell), publishedExit(parser.SemanticTableRow),
			publishedExit(parser.SemanticTable), publishedExit(parser.SemanticDocument),
		},
		279: {
			publishedEnter(parser.SemanticDocument), publishedListEnter(false, 0, true),
			publishedEnter(parser.SemanticListItem), publishedTask(false), publishedEnter(parser.SemanticParagraph), publishedText(" foo"), publishedExit(parser.SemanticParagraph), publishedExit(parser.SemanticListItem),
			publishedEnter(parser.SemanticListItem), publishedTask(true), publishedEnter(parser.SemanticParagraph), publishedText(" bar"), publishedExit(parser.SemanticParagraph), publishedExit(parser.SemanticListItem),
			publishedExit(parser.SemanticList), publishedExit(parser.SemanticDocument),
		},
		491: {
			publishedEnter(parser.SemanticDocument), publishedEnter(parser.SemanticParagraph), publishedEnter(parser.SemanticStrikethrough), publishedText("Hi"), publishedExit(parser.SemanticStrikethrough), publishedText(" Hello, "), publishedEnter(parser.SemanticStrikethrough), publishedText("there"), publishedExit(parser.SemanticStrikethrough), publishedText(" world!"), publishedExit(parser.SemanticParagraph), publishedExit(parser.SemanticDocument),
		},
		622: {
			publishedEnter(parser.SemanticDocument), publishedEnter(parser.SemanticParagraph), publishedAutoLink("www.commonmark.org", "http://www.commonmark.org", false), publishedExit(parser.SemanticParagraph), publishedExit(parser.SemanticDocument),
		},
		630: {
			publishedEnter(parser.SemanticDocument), publishedEnter(parser.SemanticParagraph), publishedAutoLink("foo@bar.baz", "mailto:foo@bar.baz", true), publishedExit(parser.SemanticParagraph), publishedExit(parser.SemanticDocument),
		},
	}

	assertPublishedGFMSemanticCases(t, cases, expected)
}

func assertPublishedCommonMarkSemanticCases(t *testing.T, cases []commonmarkspec.Case, expected map[int]publishedSemanticExpectation) {
	t.Helper()
	for number, want := range expected {
		if number <= 0 || number > len(cases) {
			t.Fatalf("CommonMark example %d is outside corpus of %d", number, len(cases))
		}
		case_ := cases[number-1]
		if case_.Number != number || case_.Section != want.section {
			t.Fatalf("CommonMark example %d identity changed: number=%d section=%q, want section %q", number, case_.Number, case_.Section, want.section)
		}
		t.Run("example-"+strconv.Itoa(number), func(t *testing.T) {
			got := collectPublishedSemanticEvents(t, []byte(case_.Markdown))
			if !reflect.DeepEqual(got, want.events) {
				t.Fatalf("CommonMark semantic contract changed\ngot:  %#v\nwant: %#v", got, want.events)
			}
		})
	}
}

func assertPublishedGFMSemanticCases(t *testing.T, cases []gfmspec.Case, expected map[int][]publishedSemanticEvent) {
	t.Helper()
	for number, want := range expected {
		if number <= 0 || number > len(cases) {
			t.Fatalf("GFM example %d is outside corpus of %d", number, len(cases))
		}
		case_ := cases[number-1]
		if case_.Number != number || len(case_.Extensions) != 1 {
			t.Fatalf("GFM example %d identity changed: number=%d extensions=%v", number, case_.Number, case_.Extensions)
		}
		t.Run("example-"+strconv.Itoa(number), func(t *testing.T) {
			got := collectPublishedSemanticEvents(t, []byte(case_.Markdown))
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("GFM semantic contract changed\ngot:  %#v\nwant: %#v", got, want)
			}
		})
	}
}

func collectPublishedSemanticEvents(t *testing.T, source []byte) []publishedSemanticEvent {
	t.Helper()
	result := make([]publishedSemanticEvent, 0, 32)
	if err := native.New().WalkSemantic(source, func(event parser.SemanticEvent) error {
		result = append(result, publishedSemanticEvent{
			phase:       event.Phase,
			kind:        event.Kind,
			value:       event.Value,
			level:       event.Level,
			destination: event.Destination,
			title:       event.Title,
			hasTitle:    event.HasTitle,
			label:       event.Label,
			email:       event.AutoLinkEmail,
			ordered:     event.Ordered,
			start:       event.Start,
			tight:       event.Tight,
			checked:     event.Checked,
			header:      event.Header,
			column:      event.Column,
			columns:     event.Columns,
			alignment:   event.Alignment,
		})
		return nil
	}); err != nil {
		t.Fatalf("WalkSemantic() error = %v", err)
	}
	return result
}

func publishedEnter(kind parser.SemanticKind) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticEnter, kind: kind}
}

func publishedExit(kind parser.SemanticKind) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticExit, kind: kind}
}

func publishedLeaf(kind parser.SemanticKind) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticLeaf, kind: kind}
}

func publishedText(value string) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticLeaf, kind: parser.SemanticText, value: value}
}

func publishedCodeSpan(value string) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticLeaf, kind: parser.SemanticCodeSpan, value: value}
}

func publishedHeadingEnter(level int) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticEnter, kind: parser.SemanticHeading, level: level}
}

func publishedHeadingExit(level int) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticExit, kind: parser.SemanticHeading, level: level}
}

func publishedLinkEnter(destination, title string, hasTitle bool, label string) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticEnter, kind: parser.SemanticLink, destination: destination, title: title, hasTitle: hasTitle, label: label}
}

func publishedImageEnter(destination, title string, hasTitle bool) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticEnter, kind: parser.SemanticImage, destination: destination, title: title, hasTitle: hasTitle}
}

func publishedReferenceDefinition(label, destination, title string, hasTitle bool) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticLeaf, kind: parser.SemanticReferenceDefinition, label: label, destination: destination, title: title, hasTitle: hasTitle}
}

func publishedAutoLink(value, destination string, email bool) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticLeaf, kind: parser.SemanticAutoLink, value: value, destination: destination, email: email}
}

func publishedListEnter(ordered bool, start int, tight bool) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticEnter, kind: parser.SemanticList, ordered: ordered, start: start, tight: tight}
}

func publishedTask(checked bool) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticLeaf, kind: parser.SemanticTaskItem, checked: checked}
}

func publishedTableEnter(columns int) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticEnter, kind: parser.SemanticTable, columns: columns}
}

func publishedRowEnter(header bool, columns int) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticEnter, kind: parser.SemanticTableRow, header: header, columns: columns}
}

func publishedCellEnter(header bool, column int, alignment parser.TableAlignment) publishedSemanticEvent {
	return publishedSemanticEvent{phase: parser.SemanticEnter, kind: parser.SemanticTableCell, header: header, column: column, alignment: alignment}
}
