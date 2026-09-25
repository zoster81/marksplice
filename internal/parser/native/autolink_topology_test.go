package native

import (
	"bytes"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestExtendedAutolinkOwnersMatchRejectsBoundaryExpansion(t *testing.T) {
	t.Parallel()

	const value = "https://encrypted.google.com/search?q=Markup+(business)"
	source := []byte("\\(Visit " + value + "\\)")
	start := bytes.Index(source, []byte(value))
	expected := []ExtendedAutolinkOwner{{
		Range: parser.Range{Start: start, End: start + len(value)},
		Form:  parser.AutoLinkExtendedURL,
		Value: value,
	}}
	if ExtendedAutolinkOwnersMatch(source, expected) {
		t.Fatal("extended URL owner accepted trailing escape byte")
	}

	source = []byte("(Visit " + value + ")")
	start = bytes.Index(source, []byte(value))
	expected[0].Range = parser.Range{Start: start, End: start + len(value)}
	if !ExtendedAutolinkOwnersMatch(source, expected) {
		t.Fatal("balanced extended URL owner was rejected")
	}
}

func TestExtendedAutolinkOwnersMatchStopsAtInlineSegmentEnd(t *testing.T) {
	t.Parallel()

	const value = "https://example.test/path"
	for _, separator := range []string{"\n", "\r\n", "\r"} {
		source := []byte(value + separator + "next")
		expected := []ExtendedAutolinkOwner{{
			Range: parser.Range{Start: 0, End: len(value)},
			Form:  parser.AutoLinkExtendedURL,
			Value: value,
		}}
		if !ExtendedAutolinkOwnersMatch(source, expected) {
			t.Fatalf("extended URL owner crossed inline segment for separator %q", separator)
		}
	}
}

func TestExtendedAutolinkSafeTrailingTextPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		form  parser.AutoLinkForm
		value string
		email bool
		text  string
		want  int
	}{
		{name: "url closing parenthesis", form: parser.AutoLinkExtendedURL, value: "https://example.test/a(b)", text: ")", want: 1},
		{name: "url trailing period", form: parser.AutoLinkExtendedURL, value: "https://example.test/path", text: ". next", want: 1},
		{name: "url extending text", form: parser.AutoLinkExtendedURL, value: "https://example.test/path", text: ".more", want: 0},
		{name: "email trailing period", form: parser.AutoLinkExtendedEmail, value: "foo@example.test", email: true, text: ".", want: 1},
		{name: "protocol trailing period", form: parser.AutoLinkExtendedProtocol, value: "mailto:foo@example.test", text: ".", want: 1},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := ExtendedAutolinkSafeTrailingTextPrefix(test.form, test.value, test.email, test.text); got != test.want {
				t.Fatalf("safe prefix = %d, want %d", got, test.want)
			}
		})
	}
}

func TestExtendedAutolinkOwnersMatchProtocolAndEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		source string
		value  string
		form   parser.AutoLinkForm
		email  bool
	}{
		{source: "mail mailto:foo@example.test end", value: "mailto:foo@example.test", form: parser.AutoLinkExtendedProtocol},
		{source: "mail foo@example.test end", value: "foo@example.test", form: parser.AutoLinkExtendedEmail, email: true},
	}
	for _, test := range tests {
		start := bytes.Index([]byte(test.source), []byte(test.value))
		expected := []ExtendedAutolinkOwner{{
			Range: parser.Range{Start: start, End: start + len(test.value)},
			Form:  test.form,
			Value: test.value,
			Email: test.email,
		}}
		if !ExtendedAutolinkOwnersMatch([]byte(test.source), expected) {
			t.Fatalf("owner rejected for %q", test.source)
		}
	}
}
