package publictest

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM102AlertsRecognizeExactTopLevelGitHubAlertMarkers(t *testing.T) {
	t.Parallel()

	source := []byte(
		"> ordinary\n\n" +
			"> [!NOTE]\n> note body\n\n" +
			"> [!TIP]\n> tip body\n\n" +
			"> [!IMPORTANT]\n> important body\n\n" +
			"> [!WARNING]\n> warning body\n\n" +
			"> [!CAUTION]\n> caution body\n",
	)
	doc := mustParseAlertDocument(t, source)

	alerts := doc.Alerts()
	wantKinds := []marksplice.AlertKind{
		marksplice.AlertKindNote,
		marksplice.AlertKindTip,
		marksplice.AlertKindImportant,
		marksplice.AlertKindWarning,
		marksplice.AlertKindCaution,
	}
	if len(alerts) != len(wantKinds) {
		t.Fatalf("Alerts() len = %d, want %d", len(alerts), len(wantKinds))
	}
	for index, alert := range alerts {
		if alert.Kind() != wantKinds[index] {
			t.Fatalf("alert %d Kind() = %v, want %v", index, alert.Kind(), wantKinds[index])
		}
		blockquote, ok := doc.Blockquote(alert.ID())
		if !ok {
			t.Fatalf("alert %d ID() does not identify the underlying blockquote", index)
		}
		if alert.Range() != blockquote.Range() {
			t.Fatalf("alert %d Range() = %+v, blockquote Range() = %+v", index, alert.Range(), blockquote.Range())
		}
		marker, ok := doc.SourceRange(alert.MarkerRange())
		if !ok {
			t.Fatalf("alert %d MarkerRange() is not readable", index)
		}
		wantMarker := []byte(alertMarkerForTest(wantKinds[index]))
		if !bytes.Equal(marker, wantMarker) {
			t.Fatalf("alert %d marker = %q, want %q", index, marker, wantMarker)
		}
		fromID, ok := doc.Alert(alert.ID())
		if !ok || fromID != alert {
			t.Fatalf("Alert(ID) = %+v/%v, want %+v/true", fromID, ok, alert)
		}
		node, ok := doc.Node(alert.ID())
		if !ok || node.Kind() != marksplice.KindBlockquote {
			t.Fatalf("alert %d underlying node = %+v/%v, want KindBlockquote", index, node, ok)
		}
	}

	if _, ok := doc.Alert(marksplice.NodeID{}); ok {
		t.Fatal("Alert(zero ID) ok = true")
	}
	original := alerts[0]
	alerts[0] = marksplice.Alert{}
	again := doc.Alerts()
	if len(again) != len(wantKinds) || again[0] != original {
		t.Fatalf("second Alerts() = %+v, caller mutation leaked", again)
	}
}

func TestM102AlertBodyRangesPreservePhysicalSourceAndAreCallerOwned(t *testing.T) {
	t.Parallel()

	source := []byte("> [!NOTE]\r\n> first π\r\n>\r\n> second\r\n")
	doc := mustParseAlertDocument(t, source)
	alerts := doc.Alerts()
	if len(alerts) != 1 {
		t.Fatalf("Alerts() len = %d, want 1", len(alerts))
	}
	alert := alerts[0]

	ranges, ok := doc.AlertBodyRanges(alert.ID())
	if !ok || len(ranges) != 3 {
		t.Fatalf("AlertBodyRanges() = %v/%v, want three ranges", ranges, ok)
	}
	want := [][]byte{[]byte("first π"), nil, []byte("second")}
	for index, range_ := range ranges {
		got, ok := doc.SourceRange(range_)
		if !ok || !bytes.Equal(got, want[index]) {
			t.Fatalf("body range %d = %q/%v, want %q", index, got, ok, want[index])
		}
	}
	original := ranges[0]
	ranges[0] = marksplice.Range{}
	again, ok := doc.AlertBodyRanges(alert.ID())
	if !ok || len(again) != 3 || again[0] != original {
		t.Fatalf("second AlertBodyRanges() = %v/%v, caller mutation leaked", again, ok)
	}
	if _, ok := doc.AlertBodyRanges(marksplice.NodeID{}); ok {
		t.Fatal("AlertBodyRanges(zero ID) ok = true")
	}
}

func TestM102AlertsRejectMalformedMarkerOnlyAndNestedShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
	}{
		{name: "lowercase", source: "> [!note]\n> body\n"},
		{name: "unknown", source: "> [!INFO]\n> body\n"},
		{name: "trailing text", source: "> [!NOTE] extra\n> body\n"},
		{name: "trailing space", source: "> [!NOTE] \n> body\n"},
		{name: "leading space", source: ">  [!NOTE]\n> body\n"},
		{name: "not first line", source: "> intro\n> [!NOTE]\n> body\n"},
		{name: "marker only", source: "> [!NOTE]\n"},
		{name: "empty body", source: "> [!NOTE]\n>\n"},
		{name: "nested blockquote", source: "> outer\n> > [!NOTE]\n> > body\n"},
		{name: "list nested", source: "- > [!NOTE]\n  > body\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc := mustParseAlertDocument(t, []byte(tt.source))
			if alerts := doc.Alerts(); len(alerts) != 0 {
				t.Fatalf("Alerts() = %+v, want none", alerts)
			}
		})
	}
}

func TestM102DocumentBuilderWritesCanonicalSingleParagraphAlerts(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendAlert(marksplice.AlertKindWarning, "first *line*\nsecond"); err != nil {
		t.Fatalf("AppendAlert() error = %v", err)
	}
	if err := builder.AppendAlertContent(marksplice.AlertKindTip, marksplice.TextInline("Use *literal* punctuation.")); err != nil {
		t.Fatalf("AppendAlertContent() error = %v", err)
	}
	markdown, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte(
		"> [!WARNING]\n> first *line*\n> second\n\n" +
			"> [!TIP]\n> Use \\*literal\\* punctuation\\.\n",
	)
	if !bytes.Equal(markdown, want) {
		t.Fatalf("Markdown() = %q, want %q", markdown, want)
	}

	doc := mustParseAlertDocument(t, markdown)
	alerts := doc.Alerts()
	if got := alertKindsForTest(alerts); !reflect.DeepEqual(got, []marksplice.AlertKind{marksplice.AlertKindWarning, marksplice.AlertKindTip}) {
		t.Fatalf("parsed alert kinds = %v", got)
	}
}

func TestM102DocumentBuilderWritesCanonicalMultiBlockAlert(t *testing.T) {
	t.Parallel()

	content := marksplice.NewDocumentBuilder()
	if err := content.AppendParagraph("Intro"); err != nil {
		t.Fatalf("content.AppendParagraph() error = %v", err)
	}
	if err := content.AppendUnorderedList("one", "two"); err != nil {
		t.Fatalf("content.AppendUnorderedList() error = %v", err)
	}

	var builder marksplice.DocumentBuilder
	if err := builder.AppendAlertBlocks(marksplice.AlertKindImportant, content); err != nil {
		t.Fatalf("AppendAlertBlocks() error = %v", err)
	}
	if err := content.AppendParagraph("later mutation"); err != nil {
		t.Fatalf("content.AppendParagraph(later) error = %v", err)
	}
	markdown, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("> [!IMPORTANT]\n> Intro\n> \n> - one\n> - two\n")
	if !bytes.Equal(markdown, want) {
		t.Fatalf("Markdown() = %q, want %q", markdown, want)
	}

	doc := mustParseAlertDocument(t, markdown)
	alerts := doc.Alerts()
	if len(alerts) != 1 || alerts[0].Kind() != marksplice.AlertKindImportant {
		t.Fatalf("Alerts() = %+v, want one Important alert", alerts)
	}
	body, ok := doc.AlertBodyRanges(alerts[0].ID())
	if !ok || len(body) != 4 {
		t.Fatalf("AlertBodyRanges() = %v/%v, want four physical body ranges", body, ok)
	}
}

func TestM102DocumentBuilderRejectsInvalidAndNestedAlertConstruction(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	invalid := []struct {
		name string
		kind marksplice.AlertKind
		body string
	}{
		{name: "unknown kind", kind: marksplice.AlertKindUnknown, body: "body"},
		{name: "out of range kind", kind: marksplice.AlertKind(255), body: "body"},
		{name: "empty body", kind: marksplice.AlertKindNote, body: ""},
		{name: "blank body", kind: marksplice.AlertKindNote, body: "first\n\nsecond"},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			if err := builder.AppendAlert(tt.kind, tt.body); !errors.Is(err, marksplice.ErrInvalidConstruction) {
				t.Fatalf("AppendAlert() error = %v, want ErrInvalidConstruction", err)
			}
		})
	}

	var nilBuilder *marksplice.DocumentBuilder
	if err := nilBuilder.AppendAlert(marksplice.AlertKindNote, "body"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil AppendAlert() error = %v, want ErrInvalidConstruction", err)
	}
	if err := builder.AppendAlertBlocks(marksplice.AlertKindNote, nil); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendAlertBlocks(nil) error = %v, want ErrInvalidConstruction", err)
	}
	if err := builder.AppendAlertBlocks(marksplice.AlertKindNote, &builder); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendAlertBlocks(self) error = %v, want ErrInvalidConstruction", err)
	}

	alertContent := marksplice.NewDocumentBuilder()
	if err := alertContent.AppendAlert(marksplice.AlertKindTip, "nested"); err != nil {
		t.Fatalf("alertContent.AppendAlert() error = %v", err)
	}
	if err := builder.AppendBlockquoteBlocks(1, alertContent); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendBlockquoteBlocks(alert) error = %v, want ErrInvalidConstruction", err)
	}
	if err := builder.AppendAlertBlocks(marksplice.AlertKindNote, alertContent); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendAlertBlocks(alert child) error = %v, want ErrInvalidConstruction", err)
	}

	frontMatterContent := marksplice.NewDocumentBuilder()
	if err := frontMatterContent.SetYAMLFrontMatter(marksplice.FrontMatterFieldInput{Key: "title", Value: "x"}); err != nil {
		t.Fatalf("frontMatterContent.SetYAMLFrontMatter() error = %v", err)
	}
	if err := frontMatterContent.AppendParagraph("body"); err != nil {
		t.Fatalf("frontMatterContent.AppendParagraph() error = %v", err)
	}
	if err := builder.AppendAlertBlocks(marksplice.AlertKindNote, frontMatterContent); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendAlertBlocks(front matter) error = %v, want ErrInvalidConstruction", err)
	}

	deferredContent := marksplice.NewDocumentBuilder()
	if err := deferredContent.AppendParagraph("body"); err != nil {
		t.Fatalf("deferredContent.AppendParagraph() error = %v", err)
	}
	if err := deferredContent.DeferReferenceDefinition("later", "/later"); err != nil {
		t.Fatalf("deferredContent.DeferReferenceDefinition() error = %v", err)
	}
	if err := builder.AppendAlertBlocks(marksplice.AlertKindNote, deferredContent); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendAlertBlocks(deferred reference) error = %v, want ErrInvalidConstruction", err)
	}
}

func TestM102AlertUnderlyingBlockquoteKeepsExistingRemovalContract(t *testing.T) {
	t.Parallel()

	source := []byte("before\n\n> [!CAUTION]\n> destructive\n\nafter\n")
	doc := mustParseAlertDocument(t, source)
	alerts := doc.Alerts()
	if len(alerts) != 1 {
		t.Fatalf("Alerts() len = %d, want 1", len(alerts))
	}
	change, err := doc.PrepareRemoveBlockquote(alerts[0].ID())
	if err != nil {
		t.Fatalf("PrepareRemoveBlockquote() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("before\n\n\nafter\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Apply() = %q, want %q", got, want)
	}
}

func mustParseAlertDocument(t *testing.T, source []byte) *marksplice.Document {
	t.Helper()
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return doc
}

func alertKindsForTest(alerts []marksplice.Alert) []marksplice.AlertKind {
	result := make([]marksplice.AlertKind, len(alerts))
	for index, alert := range alerts {
		result[index] = alert.Kind()
	}
	return result
}

func alertMarkerForTest(kind marksplice.AlertKind) string {
	switch kind {
	case marksplice.AlertKindNote:
		return "[!NOTE]"
	case marksplice.AlertKindTip:
		return "[!TIP]"
	case marksplice.AlertKindImportant:
		return "[!IMPORTANT]"
	case marksplice.AlertKindWarning:
		return "[!WARNING]"
	case marksplice.AlertKindCaution:
		return "[!CAUTION]"
	default:
		return ""
	}
}
