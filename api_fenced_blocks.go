package marksplice

import "github.com/zoster81/marksplice/internal/splice"

// FencedBlock is immutable read-only detail for one source-proven top-level GFM
// fenced block. It owns the complete physical container independently from the
// narrower historical FencedCode replacement capability.
type FencedBlock struct {
	id                 NodeID
	sourceRange        Range
	openingFenceRange  Range
	infoRange          Range
	closingFenceRange  Range
	info               string
	language           string
	fenceChar          byte
	openingFenceLength int
	closingFenceLength int
	openingIndent      int
	closingIndent      int
	hasInfo            bool
	hasLanguage        bool
	closed             bool
}

// ID returns the snapshot-scoped identity shared with FencedCode when the same
// block also satisfies the historical contiguous replacement contract.
func (f FencedBlock) ID() NodeID { return f.id }

// Range returns the exact complete physical source owned by the fenced block.
// A closing-fence line terminator is included when present; an unclosed block
// owns source through EOF.
func (f FencedBlock) Range() Range { return f.sourceRange }

// OpeningFenceRange returns the exact opening delimiter run, excluding indentation
// and info-string source.
func (f FencedBlock) OpeningFenceRange() Range { return f.openingFenceRange }

// FenceChar returns the opening delimiter byte, either '`' or '~'.
func (f FencedBlock) FenceChar() byte { return f.fenceChar }

// OpeningFenceLength returns the number of delimiter bytes in the opening fence.
func (f FencedBlock) OpeningFenceLength() int { return f.openingFenceLength }

// OpeningIndent returns the source indentation before the opening delimiter.
func (f FencedBlock) OpeningIndent() int { return f.openingIndent }

// Info returns the parser-proven trimmed info string when one is present.
func (f FencedBlock) Info() (string, bool) {
	if !f.hasInfo {
		return "", false
	}
	return f.info, true
}

// InfoRange returns the exact source bytes corresponding to Info.
func (f FencedBlock) InfoRange() (Range, bool) {
	if !f.hasInfo {
		return Range{}, false
	}
	return f.infoRange, true
}

// Language returns the parser-proven language token derived from the info string.
// Marksplice treats the value only as metadata and does not interpret the payload.
func (f FencedBlock) Language() (string, bool) {
	if !f.hasLanguage {
		return "", false
	}
	return f.language, true
}

// Closed reports whether a matching closing fence is present in source.
func (f FencedBlock) Closed() bool { return f.closed }

// ClosingFenceRange returns the exact closing delimiter run when the block is closed.
func (f FencedBlock) ClosingFenceRange() (Range, bool) {
	if !f.closed {
		return Range{}, false
	}
	return f.closingFenceRange, true
}

// ClosingFenceLength returns the number of delimiter bytes in the closing fence.
func (f FencedBlock) ClosingFenceLength() (int, bool) {
	if !f.closed {
		return 0, false
	}
	return f.closingFenceLength, true
}

// ClosingIndent returns the source indentation before the closing delimiter.
func (f FencedBlock) ClosingIndent() (int, bool) {
	if !f.closed {
		return 0, false
	}
	return f.closingIndent, true
}

// FencedBlocks returns every source-proven top-level fenced block in source order.
// Readability is broader than the historical FencedCode edit capability: empty,
// non-contiguous, or unclosed blocks may be returned without gaining mutation authority.
func (d *Document) FencedBlocks() []FencedBlock {
	if d == nil || d.document == nil {
		return nil
	}
	blocks := make([]FencedBlock, 0)
	for index := 0; index < d.document.NodeCount(); index++ {
		summary, ok := d.document.NodeSummaryAt(index)
		if !ok || summary.Kind != splice.KindFencedCode {
			continue
		}
		node, ok := d.document.Node(summary.ID)
		if !ok {
			continue
		}
		block, ok := publicFencedBlock(d.document, node)
		if ok {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

// FencedBlock returns one source-proven top-level fenced block by snapshot ID.
func (d *Document) FencedBlock(id NodeID) (FencedBlock, bool) {
	node, ok := d.internalNode(id)
	if !ok {
		return FencedBlock{}, false
	}
	return publicFencedBlock(d.document, node)
}

// FencedBlockContentRanges returns caller-owned source-backed payload ranges, one
// per parser-proven physical body line. Empty payloads return an empty slice with
// ok=true. These ranges are read-only source ownership, not generic mutation spans.
func (d *Document) FencedBlockContentRanges(id NodeID) ([]Range, bool) {
	node, ok := d.internalNode(id)
	if !ok {
		return nil, false
	}
	if node.Kind != splice.KindFencedCode || !node.TopLevel {
		return nil, false
	}
	mapping, _, _, ok := d.document.FencedBlockSource(node.ID)
	if !ok {
		return nil, false
	}
	return publicRanges(mapping.ContentRanges), true
}

func publicFencedBlock(document *splice.Document, node splice.Node) (FencedBlock, bool) {
	if document == nil || node.Kind != splice.KindFencedCode || !node.TopLevel {
		return FencedBlock{}, false
	}
	mapping, info, language, ok := document.FencedBlockSource(node.ID)
	if !ok || mapping.OpeningFenceLength < 3 ||
		mapping.OpeningFenceRange.Start >= mapping.OpeningFenceRange.End {
		return FencedBlock{}, false
	}
	return FencedBlock{
		id:                 publicNodeID(node.ID),
		sourceRange:        Range{Start: mapping.Range.Start, End: mapping.Range.End},
		openingFenceRange:  Range{Start: mapping.OpeningFenceRange.Start, End: mapping.OpeningFenceRange.End},
		infoRange:          Range{Start: mapping.InfoRange.Start, End: mapping.InfoRange.End},
		closingFenceRange:  Range{Start: mapping.ClosingFenceRange.Start, End: mapping.ClosingFenceRange.End},
		info:               info,
		language:           language,
		fenceChar:          mapping.FenceChar,
		openingFenceLength: mapping.OpeningFenceLength,
		closingFenceLength: mapping.ClosingFenceLength,
		openingIndent:      mapping.OpeningIndent,
		closingIndent:      mapping.ClosingIndent,
		hasInfo:            info != "",
		hasLanguage:        language != "",
		closed:             mapping.Closed,
	}, true
}
