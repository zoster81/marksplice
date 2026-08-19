package goldmark

import (
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extensionast "github.com/yuin/goldmark/extension/ast"
	goldmarkparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/zoster81/marksplice/internal/parser"
)

// Adapter isolates Goldmark behind Marksplice-owned parser types.
type Adapter struct {
	markdown goldmark.Markdown
}

// New returns a Goldmark-backed semantic parser configured for Marksplice's single GFM profile.
func New() *Adapter {
	return &Adapter{markdown: newMarkdown()}
}

func newMarkdown(options ...goldmark.Option) goldmark.Markdown {
	return newMarkdownWithExtensions([]goldmark.Extender{extension.GFM}, options...)
}

func newMarkdownWithExtensions(extenders []goldmark.Extender, options ...goldmark.Option) goldmark.Markdown {
	parserOptions := []goldmarkparser.Option{
		goldmarkparser.WithInlineParsers(
			util.Prioritized(&gfmCompatibilityInlineParser{}, 399),
		),
	}
	if hasGFMExtendedAutolinks(extenders) {
		parserOptions = append(parserOptions,
			goldmarkparser.WithInlineParsers(
				util.Prioritized(&gfmProtocolAutolinkParser{}, 998),
			),
			extension.WithLinkifyAllowedProtocols([]string{"http:", "https:"}),
		)
	}
	options = append(options, goldmark.WithParserOptions(parserOptions...))
	if len(extenders) != 0 {
		options = append(options, goldmark.WithExtensions(extenders...))
	}
	return goldmark.New(options...)
}

func hasGFMExtendedAutolinks(extenders []goldmark.Extender) bool {
	for _, extender := range extenders {
		if extender == extension.GFM || extender == extension.Linkify {
			return true
		}
	}
	return false
}

// Parse returns parser-independent semantic observations tied to source byte ranges.
func (a *Adapter) Parse(source []byte) ([]parser.Node, error) {
	parseSource := normalizeIsolatedCR(source)
	root := a.markdown.Parser().Parse(text.NewReader(parseSource))
	nodes := make([]parser.Node, 0)

	err := ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch typed := node.(type) {
		case *ast.Paragraph:
			lines := typed.Lines()
			if lines.Len() == 0 {
				return ast.WalkContinue, nil
			}

			first := lines.At(0)
			last := lines.At(lines.Len() - 1)
			range_ := parser.Range{Start: first.Start, End: paragraphContentEnd(source, last.Stop)}
			if !range_.Valid(len(source)) {
				return ast.WalkStop, fmt.Errorf("goldmark paragraph range [%d,%d) is outside source length %d", range_.Start, range_.End, len(source))
			}

			nodes = append(nodes, parser.Node{
				Kind:  parser.KindParagraph,
				Range: range_,
			})
		case *ast.Heading:
			if typed.Parent() == nil || typed.Parent().Kind() != ast.KindDocument {
				return ast.WalkContinue, nil
			}
			lines := typed.Lines()
			if lines.Len() == 0 {
				return ast.WalkContinue, nil
			}

			first := lines.At(0)
			last := lines.At(lines.Len() - 1)
			range_ := parser.Range{Start: first.Start, End: last.Stop}
			if !range_.Valid(len(source)) {
				return ast.WalkStop, fmt.Errorf("goldmark heading content range [%d,%d) is outside source length %d", range_.Start, range_.End, len(source))
			}

			nodes = append(nodes, parser.Node{
				Kind:  parser.KindHeading,
				Range: range_,
				Level: typed.Level,
			})
		case *extensionast.TaskCheckBox:
			parent, ok := typed.Parent().(*ast.TextBlock)
			if !ok || parent.Parent() == nil || parent.Parent().Kind() != ast.KindListItem {
				return ast.WalkContinue, nil
			}
			lines := parent.Lines()
			if lines.Len() == 0 {
				return ast.WalkContinue, nil
			}
			first := lines.At(0)
			range_ := parser.Range{Start: first.Start, End: first.Stop}
			if !range_.Valid(len(source)) {
				return ast.WalkStop, fmt.Errorf("goldmark task anchor range [%d,%d) is outside source length %d", range_.Start, range_.End, len(source))
			}
			nodes = append(nodes, parser.Node{
				Kind:    parser.KindTask,
				Range:   range_,
				Checked: typed.IsChecked,
			})
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk goldmark AST: %w", err)
	}
	return nodes, nil
}

func normalizeIsolatedCR(source []byte) []byte {
	var normalized []byte
	for i, b := range source {
		if b != '\r' || (i+1 < len(source) && source[i+1] == '\n') {
			continue
		}
		if normalized == nil {
			normalized = append([]byte(nil), source...)
		}
		normalized[i] = '\n'
	}
	if normalized == nil {
		return source
	}
	return normalized
}

func paragraphContentEnd(source []byte, semanticEnd int) int {
	end := semanticEnd
	for end < len(source) && source[end] != '\n' && source[end] != '\r' {
		end++
	}
	return end
}

var _ parser.Adapter = (*Adapter)(nil)
