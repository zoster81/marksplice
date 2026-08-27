package native

import "testing"

func TestDelimiterRunIndexSingleAndMultiSegment(t *testing.T) {
	t.Parallel()

	runs := []delimiterRun{
		{segment: 0, start: 1, end: 2},
		{segment: 0, start: 4, end: 6},
		{segment: 2, start: 9, end: 10},
	}

	single := newDelimiterRunIndex(1, runs[:2])
	if single.bySegment != nil {
		t.Fatalf("single-segment index allocated segment table: %#v", single.bySegment)
	}
	if got := single.segmentRuns(0); len(got) != 2 || got[0].start != 1 || got[1].end != 6 {
		t.Fatalf("single-segment runs = %#v", got)
	}
	if got := single.segmentRuns(1); got != nil {
		t.Fatalf("out-of-range single-segment runs = %#v", got)
	}

	multi := newDelimiterRunIndex(3, runs)
	if len(multi.bySegment) != 3 {
		t.Fatalf("multi-segment table len = %d, want 3", len(multi.bySegment))
	}
	if got := multi.segmentRuns(0); len(got) != 2 || got[0].start != 1 || got[1].end != 6 {
		t.Fatalf("segment 0 runs = %#v", got)
	}
	if got := multi.segmentRuns(1); got != nil {
		t.Fatalf("empty segment runs = %#v", got)
	}
	if got := multi.segmentRuns(2); len(got) != 1 || got[0].start != 9 || got[0].end != 10 {
		t.Fatalf("segment 2 runs = %#v", got)
	}

	zero := newDelimiterRunIndex(0, nil)
	if got := zero.segmentRuns(0); got != nil {
		t.Fatalf("zero-segment runs = %#v", got)
	}
}

func TestInlineStartIndexSingleAndMultiSegment(t *testing.T) {
	t.Parallel()

	t.Run("single segment", func(t *testing.T) {
		index := newInlineStartIndex(1)
		index.add(0, 7)
		index.add(0, 3)
		index.add(0, 7)
		index.add(-1, 1)
		index.add(1, 9)
		index.finalize()
		if !index.hasAt(0, 3) || !index.hasAt(0, 7) || index.hasAt(0, 4) {
			t.Fatalf("single-segment starts = %v", index.starts(0))
		}
		if !index.anyIn(0, 4, 8) || index.anyIn(0, 8, 9) || index.anyIn(1, 0, 10) {
			t.Fatalf("single-segment range queries changed: %v", index.starts(0))
		}
	})

	t.Run("multiple segments", func(t *testing.T) {
		index := newInlineStartIndex(3)
		index.add(2, 9)
		index.add(0, 2)
		index.add(2, 4)
		index.finalize()
		if !index.hasAt(0, 2) || !index.hasAt(2, 4) || !index.hasAt(2, 9) || index.hasAt(1, 2) {
			t.Fatalf("multi-segment starts = %v / %v / %v", index.starts(0), index.starts(1), index.starts(2))
		}
	})

	t.Run("zero segments", func(t *testing.T) {
		index := newInlineStartIndex(0)
		index.add(0, 1)
		index.finalize()
		if index.hasAt(0, 1) || index.anyIn(0, 0, 2) {
			t.Fatal("zero-segment index accepted a position")
		}
	})
}
