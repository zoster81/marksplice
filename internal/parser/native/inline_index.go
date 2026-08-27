package native

import (
	"cmp"
	"slices"
	"sort"

	"github.com/zoster81/marksplice/internal/parser"
)

type inlineStartIndex struct {
	segmentCount int
	single       []int
	bySegment    [][]int
}

func newInlineStartIndex(segmentCount int) inlineStartIndex {
	if segmentCount <= 0 {
		return inlineStartIndex{}
	}
	index := inlineStartIndex{segmentCount: segmentCount}
	if segmentCount > 1 {
		index.bySegment = make([][]int, segmentCount)
	}
	return index
}

func (index *inlineStartIndex) add(segment, offset int) {
	if segment < 0 || segment >= index.segmentCount {
		return
	}
	if index.segmentCount == 1 {
		index.single = append(index.single, offset)
		return
	}
	index.bySegment[segment] = append(index.bySegment[segment], offset)
}

func (index *inlineStartIndex) finalize() {
	if index.segmentCount == 1 {
		index.single = normalizeInlineStarts(index.single)
		return
	}
	for segment := range index.bySegment {
		index.bySegment[segment] = normalizeInlineStarts(index.bySegment[segment])
	}
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
	if start >= end {
		return false
	}
	starts := index.starts(segment)
	position := sort.SearchInts(starts, start)
	return position < len(starts) && starts[position] < end
}

func (index inlineStartIndex) hasAt(segment, offset int) bool {
	starts := index.starts(segment)
	position := sort.SearchInts(starts, offset)
	return position < len(starts) && starts[position] == offset
}

func (index inlineStartIndex) starts(segment int) []int {
	if segment < 0 || segment >= index.segmentCount {
		return nil
	}
	if index.segmentCount == 1 {
		return index.single
	}
	return index.bySegment[segment]
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
