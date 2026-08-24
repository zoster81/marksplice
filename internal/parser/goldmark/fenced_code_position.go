package goldmark

import (
	"github.com/yuin/goldmark/ast"
	goldmarkparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// fencedCodePositionParser delegates the complete fenced-code grammar to
// Goldmark while recording the exact opening-fence byte anchor on the resulting
// AST node. Goldmark otherwise leaves FencedCodeBlock.Pos undefined, including
// for empty bodies where Lines() contains no source segment.
type fencedCodePositionParser struct {
	delegate goldmarkparser.BlockParser
}

func newFencedCodePositionParser() goldmarkparser.BlockParser {
	return &fencedCodePositionParser{delegate: goldmarkparser.NewFencedCodeBlockParser()}
}

func (p *fencedCodePositionParser) Trigger() []byte {
	return p.delegate.Trigger()
}

func (p *fencedCodePositionParser) Open(parent ast.Node, reader text.Reader, pc goldmarkparser.Context) (ast.Node, goldmarkparser.State) {
	_, segment := reader.PeekLine()
	blockOffset := pc.BlockOffset()
	node, state := p.delegate.Open(parent, reader, pc)
	if node != nil && blockOffset >= 0 {
		node.SetPos(segment.Start - segment.Padding + blockOffset)
	}
	return node, state
}

func (p *fencedCodePositionParser) Continue(node ast.Node, reader text.Reader, pc goldmarkparser.Context) goldmarkparser.State {
	return p.delegate.Continue(node, reader, pc)
}

func (p *fencedCodePositionParser) Close(node ast.Node, reader text.Reader, pc goldmarkparser.Context) {
	p.delegate.Close(node, reader, pc)
}

func (p *fencedCodePositionParser) CanInterruptParagraph() bool {
	return p.delegate.CanInterruptParagraph()
}

func (p *fencedCodePositionParser) CanAcceptIndentedLine() bool {
	return p.delegate.CanAcceptIndentedLine()
}
