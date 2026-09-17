package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestBlockquoteContentReplacementReusesProvenPrefixAndEOL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  []byte
		body []byte
		want []byte
	}{
		{
			name: "CRLF grows line count",
			src:  []byte("before\r\n\r\n  > old one\r\n  > old two\r\n\r\nafter\r\n"),
			body: []byte("new one\nnew two\nnew three"),
			want: []byte("before\r\n\r\n  > new one\r\n  > new two\r\n  > new three\r\n\r\nafter\r\n"),
		},
		{
			name: "no-space marker shrinks line count",
			src:  []byte(">old one\n>old two\n"),
			body: []byte("new"),
			want: []byte(">new\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.src)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			quote := findNodeOfKind(t, doc, marksplice.KindBlockquote)
			change, err := doc.PrepareReplaceBlockquoteContent(quote.ID(), tt.body)
			if err != nil {
				t.Fatalf("PrepareReplaceBlockquoteContent() error = %v", err)
			}
			got, err := change.Apply(tt.src)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("Apply() = %q, want %q", got, tt.want)
			}
			reparsed, err := marksplice.Parse(got)
			if err != nil {
				t.Fatalf("Parse(applied) error = %v", err)
			}
			if gotRanges := blockquoteContentRanges(t, reparsed); len(gotRanges) != bytes.Count(tt.body, []byte{'\n'})+1 {
				t.Fatalf("content range count = %d, want %d", len(gotRanges), bytes.Count(tt.body, []byte{'\n'})+1)
			}
		})
	}
}

func TestBlockquoteContentReplacementFailsClosedForMixedOrAlertOwnership(t *testing.T) {
	t.Parallel()

	for name, source := range map[string][]byte{
		"lazy continuation": []byte("> quoted\nlazy continuation\n"),
		"mixed prefix":      []byte("> one\n>two\n"),
		"alert":             []byte("> [!NOTE]\n> body\n"),
	} {
		doc, err := marksplice.Parse(source)
		if err != nil {
			t.Fatalf("Parse(%s) error = %v", name, err)
		}
		quote := findNodeOfKind(t, doc, marksplice.KindBlockquote)
		if _, err := doc.PrepareReplaceBlockquoteContent(quote.ID(), []byte("replacement")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceBlockquoteContent(%s) error = %v, want ErrInvalidReplacement", name, err)
		}
	}
}

func TestAlertKindAndBodyMutationPreserveBlockquoteTrivia(t *testing.T) {
	t.Parallel()

	source := []byte("before\r\n\r\n  > [!NOTE]\r\n  > old one\r\n  > old two\r\n\r\nafter\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	alerts := doc.Alerts()
	if len(alerts) != 1 {
		t.Fatalf("len(Alerts()) = %d, want 1", len(alerts))
	}

	kindChange, err := doc.PrepareSetAlertKind(alerts[0].ID(), marksplice.AlertKindWarning)
	if err != nil {
		t.Fatalf("PrepareSetAlertKind() error = %v", err)
	}
	kindApplied, err := kindChange.Apply(source)
	if err != nil {
		t.Fatalf("Apply(kind) error = %v", err)
	}
	wantKind := []byte("before\r\n\r\n  > [!WARNING]\r\n  > old one\r\n  > old two\r\n\r\nafter\r\n")
	if !bytes.Equal(kindApplied, wantKind) {
		t.Fatalf("kind Apply() = %q, want %q", kindApplied, wantKind)
	}
	kindDoc, err := marksplice.Parse(kindApplied)
	if err != nil {
		t.Fatalf("Parse(kind applied) error = %v", err)
	}
	if got := kindDoc.Alerts(); len(got) != 1 || got[0].Kind() != marksplice.AlertKindWarning {
		t.Fatalf("applied alert = %+v, want WARNING", got)
	}

	bodyChange, err := doc.PrepareReplaceAlertBody(alerts[0].ID(), []byte("new one\nnew two\nnew three"))
	if err != nil {
		t.Fatalf("PrepareReplaceAlertBody() error = %v", err)
	}
	bodyApplied, err := bodyChange.Apply(source)
	if err != nil {
		t.Fatalf("Apply(body) error = %v", err)
	}
	wantBody := []byte("before\r\n\r\n  > [!NOTE]\r\n  > new one\r\n  > new two\r\n  > new three\r\n\r\nafter\r\n")
	if !bytes.Equal(bodyApplied, wantBody) {
		t.Fatalf("body Apply() = %q, want %q", bodyApplied, wantBody)
	}
	bodyDoc, err := marksplice.Parse(bodyApplied)
	if err != nil {
		t.Fatalf("Parse(body applied) error = %v", err)
	}
	if got := bodyDoc.Alerts(); len(got) != 1 || got[0].Kind() != marksplice.AlertKindNote {
		t.Fatalf("body-applied alert = %+v, want NOTE", got)
	}
}

func TestAlertMutationRejectsInvalidKindMixedBodyAndStaleSource(t *testing.T) {
	t.Parallel()

	source := []byte("> [!NOTE]\n> body\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	alert := doc.Alerts()[0]
	if _, err := doc.PrepareSetAlertKind(alert.ID(), marksplice.AlertKindUnknown); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareSetAlertKind(unknown) error = %v, want ErrInvalidReplacement", err)
	}
	change, err := doc.PrepareReplaceAlertBody(alert.ID(), []byte("new body"))
	if err != nil {
		t.Fatalf("PrepareReplaceAlertBody() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[len(stale)-2] = 'X'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}

	mixedSource := []byte("> [!NOTE]\n> body one\n>body two\n")
	mixed, err := marksplice.Parse(mixedSource)
	if err != nil {
		t.Fatalf("Parse(mixed) error = %v", err)
	}
	mixedAlert := mixed.Alerts()[0]
	if _, err := mixed.PrepareReplaceAlertBody(mixedAlert.ID(), []byte("replacement")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceAlertBody(mixed prefix) error = %v, want ErrInvalidReplacement", err)
	}
}

func blockquoteContentRanges(t *testing.T, doc *marksplice.Document) []marksplice.Range {
	t.Helper()
	quote := findNodeOfKind(t, doc, marksplice.KindBlockquote)
	ranges, ok := doc.BlockquoteContentRanges(quote.ID())
	if !ok {
		t.Fatalf("BlockquoteContentRanges(%q) unavailable", quote.ID())
	}
	return ranges
}
