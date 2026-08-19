package goldmark

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark/ast"
	extensionast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

func TestDefaultProfileIsGFMOnly(t *testing.T) {
	t.Parallel()

	source := []byte("| a | b |\n| - | - |\n| 1 | 2 |\n\n- [x] task\n\n~~strike~~\n\nwww.example.com\n\nterm\n: definition\n")
	var output bytes.Buffer
	if err := New().markdown.Convert(source, &output); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	html := output.String()
	for _, marker := range []string{"<table>", "type=\"checkbox\"", "<del>strike</del>", "href=\"http://www.example.com\""} {
		if !strings.Contains(html, marker) {
			t.Fatalf("GFM output missing %q:\n%s", marker, html)
		}
	}
	if strings.Contains(html, "<dl>") {
		t.Fatalf("non-GFM definition-list extension is unexpectedly enabled:\n%s", html)
	}
}

func TestGFM029SpecCompatibilityGuards(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		kind   ast.NodeKind
		want   int
	}{
		{name: "single tilde is strikethrough", source: "~strike~\n", kind: extensionast.KindStrikethrough, want: 1},
		{name: "double tilde is strikethrough", source: "~~strike~~\n", kind: extensionast.KindStrikethrough, want: 1},
		{name: "bare ftp is not an extended GFM autolink", source: "ftp://example.com\n", kind: ast.KindAutoLink, want: 0},
		{name: "bare mailto protocol is an extended GFM autolink", source: "mailto:foo@bar.baz\n", kind: ast.KindAutoLink, want: 1},
		{name: "bare xmpp resource is an extended GFM autolink", source: "xmpp:foo@bar.baz/txt@bin.com\n", kind: ast.KindAutoLink, want: 1},
		{name: "valid HTML comment", source: "foo <!-- valid comment -->\n", kind: ast.KindRawHTML, want: 1},
		{name: "comment containing double hyphen is literal", source: "foo <!-- not a comment -- two hyphens -->\n", kind: ast.KindRawHTML, want: 0},
		{name: "comment text starting with greater-than is literal", source: "foo <!--> foo -->\n", kind: ast.KindRawHTML, want: 0},
		{name: "comment text ending with hyphen is literal", source: "foo <!-- foo--->\n", kind: ast.KindRawHTML, want: 0},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := New().markdown.Parser().Parse(text.NewReader([]byte(tt.source)))
			got := countNodeKind(root, tt.kind)
			if got != tt.want {
				t.Fatalf("node count for %s = %d, want %d", tt.kind.String(), got, tt.want)
			}
		})
	}
}

func countNodeKind(root ast.Node, kind ast.NodeKind) int {
	count := 0
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && node.Kind() == kind {
			count++
		}
		return ast.WalkContinue, nil
	})
	return count
}

func FuzzParseProducesValidSourceRanges(f *testing.F) {
	f.Add([]byte("# Title\n\nparagraph\n"))
	f.Add([]byte("Title\r\n=====\r\n\r\nparagraph  \r\n"))
	f.Add([]byte("- [ ] task\n\n```go\nfmt.Println()\n```\n"))

	f.Fuzz(func(t *testing.T, source []byte) {
		nodes, err := New().Parse(source)
		if err != nil {
			return
		}
		for i, node := range nodes {
			if !node.Range.Valid(len(source)) {
				t.Fatalf("node %d has invalid range [%d,%d) for source length %d", i, node.Range.Start, node.Range.End, len(source))
			}
		}
	})
}
