package native

import (
	"sort"

	"github.com/zoster81/marksplice/internal/parser"
)

type inlineStartIndex struct {
	bySegment [][]int
}

func newInlineStartIndex(segmentCount int) inlineStartIndex {
	return inlineStartIndex{bySegment: make([][]int, segmentCount)}
}

func (index *inlineStartIndex) add(segment, offset int) {
	if segment < 0 || segment >= len(index.bySegment) {
		return
	}
	index.bySegment[segment] = append(index.bySegment[segment], offset)
}

func (index *inlineStartIndex) finalize() {
	for segment := range index.bySegment {
		starts := index.bySegment[segment]
		sort.Ints(starts)
		if len(starts) < 2 {
			continue
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
		index.bySegment[segment] = starts[:write]
	}
}

func (index inlineStartIndex) anyIn(segment, start, end int) bool {
	if segment < 0 || segment >= len(index.bySegment) || start >= end {
		return false
	}
	starts := index.bySegment[segment]
	position := sort.SearchInts(starts, start)
	return position < len(starts) && starts[position] < end
}

func (index inlineStartIndex) hasAt(segment, offset int) bool {
	if segment < 0 || segment >= len(index.bySegment) {
		return false
	}
	starts := index.bySegment[segment]
	position := sort.SearchInts(starts, offset)
	return position < len(starts) && starts[position] == offset
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
		sort.SliceStable(intervals, func(left, right int) bool {
			if intervals[left].start != intervals[right].start {
				return intervals[left].start < intervals[right].start
			}
			return intervals[left].end < intervals[right].end
		})
		if len(intervals) == 0 {
			continue
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
	sort.SliceStable(ranges, func(left, right int) bool {
		if ranges[left].Start != ranges[right].Start {
			return ranges[left].Start < ranges[right].Start
		}
		return ranges[left].End < ranges[right].End
	})
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

func flattenInlineExclusions(exclusions [][]parser.Range) []parser.Range {
	total := 0
	for _, ranges := range exclusions {
		total += len(ranges)
	}
	flat := make([]parser.Range, 0, total)
	for _, ranges := range exclusions {
		flat = append(flat, ranges...)
	}
	return normalizeInlineRanges(flat)
}
