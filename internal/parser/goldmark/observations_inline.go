package goldmark

import (
	"github.com/yuin/goldmark/ast"
	extensionast "github.com/yuin/goldmark/extension/ast"

	"github.com/zoster81/marksplice/internal/parser"
)

func observeAutoLink(source []byte, link *ast.AutoLink) (parser.Node, bool, error) {
	anchor, range_, value, ok := autoLinkSourceRange(source, link)
	if !ok {
		return parser.Node{}, false, nil
	}
	return parser.Node{
		Kind:          parser.KindAutoLink,
		Range:         range_,
		Anchor:        anchor,
		Value:         value,
		AutoLinkEmail: link.AutoLinkType == ast.AutoLinkEmail,
	}, true, nil
}

func observeCodeSpan(source []byte, span *ast.CodeSpan) (parser.Node, bool, error) {
	range_, ok := simplePlainTextInlineRange(source, span)
	if !ok || span.Pos() < 0 {
		return parser.Node{}, false, nil
	}
	return parser.Node{Kind: parser.KindCodeSpan, Range: range_, Anchor: span.Pos()}, true, nil
}

func observeEmphasis(source []byte, emphasis *ast.Emphasis) (parser.Node, bool, error) {
	if emphasis.Level != 1 && emphasis.Level != 2 {
		return parser.Node{}, false, nil
	}
	range_, ok := simplePlainTextInlineRange(source, emphasis)
	if !ok || emphasis.Pos() < 0 {
		return parser.Node{}, false, nil
	}
	kind := parser.KindEmphasis
	if emphasis.Level == 2 {
		kind = parser.KindStrong
	}
	return parser.Node{Kind: kind, Range: range_, Anchor: emphasis.Pos(), Level: emphasis.Level}, true, nil
}

func observeRawHTML(source []byte, raw *ast.RawHTML) (parser.Node, bool, error) {
	range_, ok := singleLineRawHTMLRange(source, raw)
	if !ok {
		return parser.Node{}, false, nil
	}
	return parser.Node{Kind: parser.KindRawHTML, Range: range_}, true, nil
}

func observeInlineLink(source []byte, link *ast.Link) (parser.Node, bool, error) {
	if link.Reference != nil {
		return parser.Node{}, false, nil
	}
	range_, ok := simpleInlineLinkLabelRange(source, link)
	if !ok {
		return parser.Node{}, false, nil
	}
	return parser.Node{
		Kind:        parser.KindInlineLink,
		Range:       range_,
		Anchor:      link.Pos(),
		Destination: string(link.Destination),
		Title:       string(link.Title),
		HasTitle:    link.Title != nil,
	}, true, nil
}

func observeImage(source []byte, image *ast.Image) (parser.Node, bool, error) {
	if image.Pos() < 0 {
		return parser.Node{}, false, nil
	}
	range_, ok := simplePlainTextInlineRange(source, image)
	if !ok {
		return parser.Node{}, false, nil
	}
	return parser.Node{Kind: parser.KindImage, Range: range_, Anchor: image.Pos()}, true, nil
}

func observeStrikethrough(source []byte, strike *extensionast.Strikethrough) (parser.Node, bool, error) {
	range_, ok := simpleStrikethroughContentRange(source, strike)
	if !ok {
		return parser.Node{}, false, nil
	}
	return parser.Node{Kind: parser.KindStrikethrough, Range: range_}, true, nil
}
