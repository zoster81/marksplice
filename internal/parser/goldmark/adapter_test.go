package goldmark

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark/ast"
	extensionast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	markparser "github.com/zoster81/marksplice/internal/parser"
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

func TestAdapterExposesSharedTableRowAnchors(t *testing.T) {
	t.Parallel()

	source := []byte("before\n\n| A | B |\n| - | - |\n| x | y |\n")
	nodes, err := New().Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	headerAnchor := bytes.Index(source, []byte("| A | B |"))
	bodyAnchor := bytes.Index(source, []byte("| x | y |"))
	var header, body []markparser.Node
	for _, node := range nodes {
		if node.Kind != markparser.KindTableCell {
			continue
		}
		if node.TableHeader {
			header = append(header, node)
		} else {
			body = append(body, node)
		}
	}
	if len(header) != 2 || len(body) != 2 {
		t.Fatalf("table cell counts = header %d body %d, want 2/2", len(header), len(body))
	}
	for column, node := range header {
		if node.TableRowAnchor != headerAnchor || node.TableColumn != column {
			t.Fatalf("header row/column = %d/%d, want %d/%d", node.TableRowAnchor, node.TableColumn, headerAnchor, column)
		}
	}
	for column, node := range body {
		if node.TableRowAnchor != bodyAnchor || node.TableColumn != column {
			t.Fatalf("body row/column = %d/%d, want %d/%d", node.TableRowAnchor, node.TableColumn, bodyAnchor, column)
		}
	}
}

func TestAdapterTableColumnsCountUnobservedEmptyCells(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| - | - |\n|   | value |\n")
	nodes, err := New().Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	bodyAnchor := bytes.Index(source, []byte("|   | value |"))
	for _, node := range nodes {
		if node.Kind == markparser.KindTableCell && !node.TableHeader && node.TableRowAnchor == bodyAnchor {
			if node.TableColumn != 1 || string(source[node.Range.Start:node.Range.End]) != "value" {
				t.Fatalf("body table observation = column %d source %q, want column 1 value", node.TableColumn, source[node.Range.Start:node.Range.End])
			}
			return
		}
	}
	t.Fatal("non-empty second body cell was not observed")
}

func TestAdapterExposesRawHTMLAndHTMLBlockSourceRanges(t *testing.T) {
	t.Parallel()

	source := []byte("before <!-- old --> <a id=\"anchor\">text</a>\n\n<div data-x=\"1\">\r\n*not markdown*\r\n</div>\r\n\r\nafter\r\n")
	nodes, err := New().Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	var rawValues []string
	var blockValues []string
	for _, node := range nodes {
		switch node.Kind {
		case markparser.KindRawHTML:
			rawValues = append(rawValues, string(source[node.Range.Start:node.Range.End]))
		case markparser.KindHTMLBlock:
			blockValues = append(blockValues, string(source[node.Range.Start:node.Range.End]))
		}
	}
	for _, want := range []string{"<!-- old -->", "<a id=\"anchor\">", "</a>"} {
		found := false
		for _, got := range rawValues {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("raw HTML %q not exposed; got %q", want, rawValues)
		}
	}
	if len(blockValues) != 1 || !strings.Contains(blockValues[0], "<div data-x=\"1\">") || !strings.Contains(blockValues[0], "</div>") {
		t.Fatalf("HTML block ranges = %q, want one opaque block containing opening and closing tags", blockValues)
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
