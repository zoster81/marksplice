package renderhtml

import "testing"

func TestDecodeMarkdownStringPreservesUnchangedPrefixesAndBytes(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"", ""},
		{"https://example.test/a", "https://example.test/a"},
		{`prefix \*x\* &amp;`, "prefix *x* &"},
		{`a\q`, `a\q`},
		{"tail\\", "tail\\"},
		{"&not-an-entity;", "&not-an-entity;"},
		{"before &#x41; &#0;", "before A \ufffd"},
		{"\xff &amp; \xfe", "\xff & \xfe"},
	} {
		if got := decodeMarkdownString(test.input); got != test.want {
			t.Errorf("decode(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestPercentEncodeURLKeepsAuthorityBracketsAndExistingEscapes(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"", ""},
		{"https://example.test/a%2fb?q=x#part", "https://example.test/a%2fb?q=x#part"},
		{"https://[2001:db8::1]/x[y]", "https://[2001:db8::1]/x%5By%5D"},
		{"//[::1]/a b", "//[::1]/a%20b"},
		{"/[a]?q=\u00e9", "/%5Ba%5D?q=%C3%A9"},
		{" \u00e9", "%20%C3%A9"},
		{"before\xff", "before%FF"},
	} {
		if got := percentEncodeURL(test.input); got != test.want {
			t.Errorf("encode(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestHTMLEscapingKeepsTheReviewedQuoteAndEntityBehavior(t *testing.T) {
	const input = `a&<>"' &quot;`
	const want = `a&amp;&lt;&gt;&quot;' &amp;quot;`
	if escapeText(input) != want || escapeAttribute(input) != want {
		t.Fatal("text and attribute escaping changed")
	}
}
