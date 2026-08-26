package publictest

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM108PathologicalInputsRemainOperable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
	}{
		{name: "one mebibyte paragraph", source: []byte(strings.Repeat("x", 1<<20) + "\n")},
		{name: "one mebibyte unclosed fenced block", source: []byte("```text\n" + strings.Repeat("x", 1<<20))},
		{name: "deep blockquote", source: []byte(strings.Repeat("> ", 1024) + "payload\n")},
		{name: "dense delimiter run", source: []byte(strings.Repeat("*_~`", 16<<10) + "\n")},
		{name: "malformed inline link storm", source: repeatedM108Source(4096, func(i int) string { return fmt.Sprintf("[broken-%d]( ", i) })},
		{name: "many duplicate headings", source: repeatedM108Source(4096, func(i int) string { return "# Same heading\n\n" })},
		{name: "many local links", source: repeatedM108Source(4096, func(i int) string { return fmt.Sprintf("[target-%d](#target-%d) ", i, i) })},
		{name: "many footnotes and references", source: repeatedM108Source(2048, func(i int) string { return fmt.Sprintf("[^note-%d]: body %d\n\nref[^note-%d]\n\n", i, i, i) })},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			document, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			assertM108ReadSurfaces(t, document, tt.source)
		})
	}
}

func FuzzM108PublicReadSurfacesRemainSourceBound(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("# Title\n\nparagraph with [link](#title)\n"),
		[]byte("---\ntitle: \"Example\"\n---\n\n> [!NOTE]\n> body\n\n```math\nx+y\n```\n"),
		[]byte("[^note]: body with [link](target.md)\n\nuse[^note] and $x+y$ plus $`z`$\n"),
		[]byte("| A | B |\n| :- | -: |\n| x | y |\n"),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source []byte) {
		document, err := marksplice.Parse(source)
		if err != nil {
			return
		}
		assertM108ReadSurfaces(t, document, source)
	})
}

func FuzzM108NoopMutationsPreserveExactSource(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("# Title\n\nparagraph\n"),
		[]byte("- item\n\n```go\nbody\n```\n"),
		[]byte("text with *em* **strong** ~~strike~~ `code` and [link](target) ![image](img.png)\n"),
		[]byte("[ref]: <target>\n\n<https://example.com>\n"),
		[]byte("---\ntitle: \"Example\"\n---\n\n$x+y$\n"),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source []byte) {
		document, err := marksplice.Parse(source)
		if err != nil {
			return
		}
		matches, err := document.QueryNodes(marksplice.NodeQuery{Limit: 64})
		if err != nil {
			t.Fatalf("QueryNodes() error = %v", err)
		}
		for _, match := range matches {
			payload, ok := document.SourceRange(match.Range())
			if !ok {
				t.Fatalf("SourceRange(%v) rejected query result", match.Range())
			}
			change, supported, err := m108NoopChange(document, match.Node(), payload)
			if err != nil {
				t.Fatalf("no-op mutation for kind %v error = %v", match.Node().Kind(), err)
			}
			if !supported {
				continue
			}
			got, err := change.Apply(source)
			if err != nil {
				t.Fatalf("Apply() no-op for kind %v error = %v", match.Node().Kind(), err)
			}
			if !bytes.Equal(got, source) {
				t.Fatalf("no-op mutation for kind %v changed source", match.Node().Kind())
			}
		}
	})
}

func assertM108ReadSurfaces(t testing.TB, document *marksplice.Document, source []byte) {
	t.Helper()
	matches, err := document.QueryNodes(marksplice.NodeQuery{Limit: 256})
	if err != nil {
		t.Fatalf("QueryNodes() error = %v", err)
	}
	for _, match := range matches {
		if !match.Range().Valid(len(source)) {
			t.Fatalf("query range %v invalid for source length %d", match.Range(), len(source))
		}
		if _, ok := document.SourceRange(match.Range()); !ok {
			t.Fatalf("SourceRange(%v) rejected query result", match.Range())
		}
	}
	sections, err := document.QuerySections(marksplice.SectionQuery{Limit: 256})
	if err != nil {
		t.Fatalf("QuerySections() error = %v", err)
	}
	for _, section := range sections {
		if !section.Range().Valid(len(source)) {
			t.Fatalf("section range %v invalid for source length %d", section.Range(), len(source))
		}
	}
	for _, relationship := range document.LinkRelationships() {
		if offset := relationship.SourceOffset(); offset < 0 || offset > len(source) {
			t.Fatalf("relationship source offset %d invalid for source length %d", offset, len(source))
		}
	}
	for _, block := range document.FencedBlocks() {
		if !block.Range().Valid(len(source)) {
			t.Fatalf("fenced block range %v invalid for source length %d", block.Range(), len(source))
		}
		contentRanges, ok := document.FencedBlockContentRanges(block.ID())
		if !ok {
			t.Fatalf("FencedBlockContentRanges(%v) rejected fenced block", block.ID())
		}
		for _, range_ := range contentRanges {
			if !range_.Valid(len(source)) {
				t.Fatalf("fenced content range %v invalid for source length %d", range_, len(source))
			}
		}
	}
	for _, definition := range document.FootnoteDefinitions() {
		if !definition.Range().Valid(len(source)) {
			t.Fatalf("footnote range %v invalid for source length %d", definition.Range(), len(source))
		}
		bodyRanges, ok := document.FootnoteDefinitionBodyRanges(definition.ID())
		if !ok {
			t.Fatalf("FootnoteDefinitionBodyRanges(%v) rejected footnote definition", definition.ID())
		}
		for _, range_ := range bodyRanges {
			if !range_.Valid(len(source)) {
				t.Fatalf("footnote body range %v invalid for source length %d", range_, len(source))
			}
		}
	}
	for _, expression := range document.MathExpressions() {
		if !expression.Range().Valid(len(source)) {
			t.Fatalf("math range %v invalid for source length %d", expression.Range(), len(source))
		}
	}
	if frontMatter, ok := document.FrontMatter(); ok && !frontMatter.Range().Valid(len(source)) {
		t.Fatalf("front-matter range %v invalid for source length %d", frontMatter.Range(), len(source))
	}
	_ = document.HeadingAnchors()
	_ = document.Alerts()
}

func m108NoopChange(document *marksplice.Document, node marksplice.Node, payload []byte) (marksplice.ChangeSet, bool, error) {
	switch node.Kind() {
	case marksplice.KindParagraph:
		change, err := document.PrepareReplaceParagraph(node.ID(), payload)
		return change, true, err
	case marksplice.KindHeading:
		change, err := document.PrepareRenameHeading(node.ID(), payload)
		return change, true, err
	case marksplice.KindListItem:
		change, err := document.PrepareReplaceListItem(node.ID(), payload)
		return change, true, err
	case marksplice.KindTask:
		task, ok := document.Task(node.ID())
		if !ok {
			return marksplice.ChangeSet{}, true, marksplice.ErrInvalidTargetKind
		}
		change, err := document.PrepareSetTaskChecked(node.ID(), task.Checked())
		return change, true, err
	case marksplice.KindTableCell:
		change, err := document.PrepareReplaceTableCell(node.ID(), payload)
		return change, true, err
	case marksplice.KindFencedCode:
		change, err := document.PrepareReplaceFencedCode(node.ID(), payload)
		return change, true, err
	case marksplice.KindStrikethrough:
		change, err := document.PrepareReplaceStrikethrough(node.ID(), payload)
		return change, true, err
	case marksplice.KindCodeSpan:
		change, err := document.PrepareReplaceCodeSpan(node.ID(), payload)
		return change, true, err
	case marksplice.KindEmphasis:
		change, err := document.PrepareReplaceEmphasis(node.ID(), payload)
		return change, true, err
	case marksplice.KindStrong:
		change, err := document.PrepareReplaceStrong(node.ID(), payload)
		return change, true, err
	case marksplice.KindInlineLink:
		change, err := document.PrepareReplaceInlineLinkDestination(node.ID(), payload)
		return change, true, err
	case marksplice.KindImage:
		change, err := document.PrepareReplaceImageDestination(node.ID(), payload)
		return change, true, err
	case marksplice.KindReferenceDefinition:
		change, err := document.PrepareReplaceReferenceDefinitionDestination(node.ID(), payload)
		return change, true, err
	case marksplice.KindAutoLink:
		change, err := document.PrepareReplaceAutoLink(node.ID(), payload)
		return change, true, err
	case marksplice.KindFrontMatterField:
		change, err := document.PrepareReplaceFrontMatterValue(node.ID(), payload)
		return change, true, err
	case marksplice.KindHTMLComment:
		change, err := document.PrepareReplaceHTMLComment(node.ID(), payload)
		return change, true, err
	case marksplice.KindHTMLAnchor:
		change, err := document.PrepareReplaceHTMLAnchor(node.ID(), payload)
		return change, true, err
	case marksplice.KindMathExpression:
		expression, ok := document.MathExpression(node.ID())
		if !ok {
			return marksplice.ChangeSet{}, true, marksplice.ErrInvalidTargetKind
		}
		payloadRange, ok := expression.PayloadRange()
		if !ok {
			return marksplice.ChangeSet{}, false, nil
		}
		mathPayload, ok := document.SourceRange(payloadRange)
		if !ok {
			return marksplice.ChangeSet{}, true, marksplice.ErrInvalidTargetKind
		}
		change, err := document.PrepareReplaceMathExpression(node.ID(), mathPayload)
		return change, true, err
	default:
		return marksplice.ChangeSet{}, false, nil
	}
}

func repeatedM108Source(count int, line func(int) string) []byte {
	var builder strings.Builder
	for i := 0; i < count; i++ {
		builder.WriteString(line(i))
	}
	return []byte(builder.String())
}
