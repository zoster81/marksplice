package native

import (
	"cmp"
	"slices"
	"sort"

	"github.com/zoster81/marksplice/internal/parser"
)

type inlineStartEntry struct {
	segment int
	offset  int
}

type inlineStartIndex struct {
	segmentCount int
	single       []int
	entries      []inlineStartEntry
}

func newInlineStartIndex(segmentCount int) inlineStartIndex {
	if segmentCount <= 0 {
		return inlineStartIndex{}
	}
	return inlineStartIndex{segmentCount: segmentCount}
}

func (index *inlineStartIndex) add(segment, offset int) {
	if segment < 0 || segment >= index.segmentCount {
		return
	}
	if index.segmentCount == 1 {
		if index.single == nil {
			index.single = make([]int, 0, 4)
		}
		index.single = append(index.single, offset)
		return
	}
	if index.entries == nil {
		index.entries = make([]inlineStartEntry, 0, 4)
	}
	index.entries = append(index.entries, inlineStartEntry{segment: segment, offset: offset})
}

func (index *inlineStartIndex) finalize() {
	if index.segmentCount == 1 {
		index.single = normalizeInlineStarts(index.single)
		return
	}
	if len(index.entries) > 1 {
		slices.SortFunc(index.entries, func(left, right inlineStartEntry) int {
			if order := cmp.Compare(left.segment, right.segment); order != 0 {
				return order
			}
			return cmp.Compare(left.offset, right.offset)
		})
	}
	write := 0
	for _, entry := range index.entries {
		if write != 0 && index.entries[write-1] == entry {
			continue
		}
		index.entries[write] = entry
		write++
	}
	clear(index.entries[write:])
	index.entries = index.entries[:write]
}

func normalizeInlineStarts(starts []int) []int {
	sort.Ints(starts)
	if len(starts) < 2 {
		return starts
	}
	write := 1
	for read := 1; read < len(starts); read++ {
		if starts[read] == starts[write-1] {
			continue
		}
		starts[write] = starts[read]
		write++
	}
	clear(starts[write:])
	return starts[:write]
}

func (index inlineStartIndex) anyIn(segment, start, end int) bool {
	if segment < 0 || segment >= index.segmentCount || start >= end {
		return false
	}
	if index.segmentCount == 1 {
		position := sort.SearchInts(index.single, start)
		return position < len(index.single) && index.single[position] < end
	}
	position := sort.Search(len(index.entries), func(position int) bool {
		entry := index.entries[position]
		return entry.segment > segment || entry.segment == segment && entry.offset >= start
	})
	return position < len(index.entries) && index.entries[position].segment == segment && index.entries[position].offset < end
}

func (index inlineStartIndex) hasAt(segment, offset int) bool {
	if segment < 0 || segment >= index.segmentCount {
		return false
	}
	if index.segmentCount == 1 {
		position := sort.SearchInts(index.single, offset)
		return position < len(index.single) && index.single[position] == offset
	}
	position := sort.Search(len(index.entries), func(position int) bool {
		entry := index.entries[position]
		return entry.segment > segment || entry.segment == segment && entry.offset >= offset
	})
	return position < len(index.entries) && index.entries[position] == (inlineStartEntry{segment: segment, offset: offset})
}

type inlineInterval struct {
	start int
	end   int
}

type inlineIntervalIndex struct {
	bySegment     [][]inlineInterval
	suffixMinEnds [][]int
}

func newInlineIntervalIndex(segmentCount int) inlineIntervalIndex {
	return inlineIntervalIndex{
		bySegment:     make([][]inlineInterval, segmentCount),
		suffixMinEnds: make([][]int, segmentCount),
	}
}

func (index *inlineIntervalIndex) add(segment, start, end int) {
	if segment < 0 || segment >= len(index.bySegment) || start >= end {
		return
	}
	index.bySegment[segment] = append(index.bySegment[segment], inlineInterval{start: start, end: end})
}

func (index *inlineIntervalIndex) finalize() {
	for segment := range index.bySegment {
		intervals := index.bySegment[segment]
		if len(intervals) == 0 {
			continue
		}
		if len(intervals) > 1 {
			slices.SortStableFunc(intervals, func(left, right inlineInterval) int {
				if order := cmp.Compare(left.start, right.start); order != 0 {
					return order
				}
				return cmp.Compare(left.end, right.end)
			})
		}
		suffix := make([]int, len(intervals))
		minEnd := intervals[len(intervals)-1].end
		for position := len(intervals) - 1; position >= 0; position-- {
			if intervals[position].end < minEnd {
				minEnd = intervals[position].end
			}
			suffix[position] = minEnd
		}
		index.suffixMinEnds[segment] = suffix
	}
}

func (index inlineIntervalIndex) anyContained(segment, start, end int) bool {
	if segment < 0 || segment >= len(index.bySegment) || start >= end {
		return false
	}
	intervals := index.bySegment[segment]
	position := sort.Search(len(intervals), func(position int) bool {
		return intervals[position].start >= start
	})
	return position < len(intervals) && index.suffixMinEnds[segment][position] <= end
}

func normalizeInlineExclusions(exclusions [][]parser.Range) {
	for segment := range exclusions {
		exclusions[segment] = normalizeInlineRanges(exclusions[segment])
	}
}

func normalizeInlineRanges(ranges []parser.Range) []parser.Range {
	if len(ranges) > 1 {
		slices.SortStableFunc(ranges, func(left, right parser.Range) int {
			if order := cmp.Compare(left.Start, right.Start); order != 0 {
				return order
			}
			return cmp.Compare(left.End, right.End)
		})
	}
	normalized := ranges[:0]
	for _, range_ := range ranges {
		if range_.Start >= range_.End {
			continue
		}
		if len(normalized) == 0 || range_.Start > normalized[len(normalized)-1].End {
			normalized = append(normalized, range_)
			continue
		}
		if range_.End > normalized[len(normalized)-1].End {
			normalized[len(normalized)-1].End = range_.End
		}
	}
	clear(ranges[len(normalized):])
	return normalized
}

func inlineRangesContainPosition(ranges []parser.Range, position int) bool {
	index := sort.Search(len(ranges), func(index int) bool {
		return ranges[index].End > position
	})
	return index < len(ranges) && ranges[index].Start <= position
}

func inlineExclusionsAt(exclusions [][]parser.Range, segment int) []parser.Range {
	if segment < 0 || segment >= len(exclusions) {
		return nil
	}
	return exclusions[segment]
}

func ensureInlineExclusions(exclusions [][]parser.Range, segmentCount int) [][]parser.Range {
	if segmentCount <= 0 || len(exclusions) == segmentCount {
		return exclusions
	}
	result := make([][]parser.Range, segmentCount)
	copy(result, exclusions)
	return result
}
