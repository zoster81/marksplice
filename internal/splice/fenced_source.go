package splice

import "github.com/zoster81/marksplice/internal/source"

type fencedSourceDetail struct {
	block    source.FencedBlockMapping
	code     source.FencedCodeMapping
	info     string
	language string
}

func appendSourceDetail[T any](details *[]T, detail T) (uint32, bool) {
	if uint64(len(*details)) >= uint64(^uint32(0)) {
		return 0, false
	}
	*details = append(*details, detail)
	return uint32(len(*details)), true
}

func sourceDetailIndex(index uint32, count int) (int, bool) {
	if index == 0 || uint64(index) > uint64(count) {
		return 0, false
	}
	return int(index) - 1, true
}

func sourceRangesWithin(ranges []source.Range, outer source.Range, sourceLen int) bool {
	for _, range_ := range ranges {
		if !range_.Valid(sourceLen) || range_.Start < outer.Start || range_.End > outer.End {
			return false
		}
	}
	return true
}

func (d *Document) fencedSource(node Node) (fencedSourceDetail, bool) {
	if d == nil || node.Kind != KindFencedCode {
		return fencedSourceDetail{}, false
	}
	index, ok := sourceDetailIndex(node.SourceDetailIndex, len(d.fencedSources))
	if !ok {
		return fencedSourceDetail{}, false
	}
	return d.fencedSources[index], true
}

// FencedBlockSource returns a caller-owned source mapping and metadata for one source-proven top-level fenced block.
func (d *Document) FencedBlockSource(id NodeID) (source.FencedBlockMapping, string, string, bool) {
	node, ok := d.nodeByID(id)
	if !ok || !node.TopLevel {
		return source.FencedBlockMapping{}, "", "", false
	}
	detail, ok := d.fencedSource(node)
	if !ok || detail.block.OpeningFenceLength < 3 || detail.block.OpeningFenceRange.Start >= detail.block.OpeningFenceRange.End {
		return source.FencedBlockMapping{}, "", "", false
	}
	mapping := detail.block
	mapping.ContentRanges = append([]source.Range(nil), mapping.ContentRanges...)
	return mapping, detail.info, detail.language, true
}

// FencedCodeSource returns the source mapping for one editable contiguous fenced-code payload.
func (d *Document) FencedCodeSource(id NodeID) (source.FencedCodeMapping, bool) {
	node, ok := d.nodeByID(id)
	if !ok || !node.Editable {
		return source.FencedCodeMapping{}, false
	}
	detail, ok := d.fencedSource(node)
	if !ok || detail.code.FenceLength < 3 || detail.code.ContentRange != node.ContentRange {
		return source.FencedCodeMapping{}, false
	}
	return detail.code, true
}
