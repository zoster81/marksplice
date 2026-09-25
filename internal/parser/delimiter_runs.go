package parser

import "slices"

// DelimiterRun describes one already-scanned Markdown delimiter run.
type DelimiterRun struct {
	Start    int
	End      int
	Marker   byte
	CanOpen  bool
	CanClose bool
}

// DelimiterRunMatch describes one delimiter consumption selected by CommonMark matching.
type DelimiterRunMatch struct {
	Marker          byte
	Level           int
	OpenerRun       int
	CloserRun       int
	OpeningConsumed Range
	ClosingConsumed Range
}

type delimiterRunState struct {
	DelimiterRun
	length      int
	remaining   int
	openActive  bool
	openVersion int
}

type delimiterRunRef struct {
	index   int
	version int
}

type delimiterRunOpenerIndex struct {
	byCategory [3][6][]delimiterRunRef
	active     []int
}

// ResolveDelimiterRuns resolves already-scanned delimiter runs using CommonMark
// emphasis matching. The input slice is not modified.
func ResolveDelimiterRuns(runs []DelimiterRun) []DelimiterRunMatch {
	var resolver DelimiterRunResolver
	return resolver.Resolve(runs)
}

// DelimiterRunResolver reuses scratch storage for a sequence of delimiter
// proofs within one operation. It must not be shared by concurrent callers.
// Results belong to the resolver and remain valid until its next Resolve call.
type DelimiterRunResolver struct {
	state   []delimiterRunState
	index   delimiterRunOpenerIndex
	matches []DelimiterRunMatch
}

// Resolve uses the same matching rules as ResolveDelimiterRuns without
// modifying runs. All logical state is reset before the input is processed.
func (r *DelimiterRunResolver) Resolve(runs []DelimiterRun) []DelimiterRunMatch {
	r.state = slices.Grow(r.state[:0], len(runs))[:len(runs)]
	state := r.state
	for index, run := range runs {
		length := run.End - run.Start
		if length < 0 {
			length = 0
		}
		state[index] = delimiterRunState{
			DelimiterRun: run,
			length:       length,
			remaining:    length,
		}
	}
	index := &r.index
	index.active = slices.Grow(index.active[:0], len(state))
	for marker := range index.byCategory {
		for category := range index.byCategory[marker] {
			index.byCategory[marker][category] = index.byCategory[marker][category][:0]
		}
	}
	matches := r.matches[:0]
	if matches == nil {
		matches = []DelimiterRunMatch{}
	}
	for closerIndex := range state {
		closer := &state[closerIndex]
		for closer.CanClose && closer.remaining > 0 {
			openerIndex := nearestDelimiterRunOpener(state, index, *closer)
			if openerIndex < 0 {
				break
			}
			use := delimiterRunConsumption(state[openerIndex], *closer)
			matches = append(matches, delimiterRunMatchFor(state[openerIndex], *closer, openerIndex, closerIndex, use))
			invalidateDelimiterRunOpenersAbove(state, index, openerIndex)
			consumeDelimiterRunOpener(state, index, openerIndex, use)
			closer.remaining -= use
		}
		if closer.CanOpen && closer.remaining > 0 {
			activateDelimiterRunOpener(state, index, closerIndex)
		}
	}
	r.matches = matches
	return matches
}

func nearestDelimiterRunOpener(runs []delimiterRunState, index *delimiterRunOpenerIndex, closer delimiterRunState) int {
	markerIndex := delimiterRunMarkerIndex(closer.Marker)
	best := -1
	for category := 0; category < len(index.byCategory[markerIndex]); category++ {
		candidate := topDelimiterRunCategoryOpener(runs, index, markerIndex, category)
		if candidate < 0 || delimiterRunConsumption(runs[candidate], closer) == 0 {
			continue
		}
		if candidate > best {
			best = candidate
		}
	}
	return best
}

func topDelimiterRunCategoryOpener(runs []delimiterRunState, index *delimiterRunOpenerIndex, markerIndex, category int) int {
	stack := &index.byCategory[markerIndex][category]
	for len(*stack) != 0 {
		ref := (*stack)[len(*stack)-1]
		run := runs[ref.index]
		if ref.version == run.openVersion && run.openActive && run.remaining > 0 && delimiterRunCategory(run) == category {
			return ref.index
		}
		*stack = (*stack)[:len(*stack)-1]
	}
	return -1
}

func delimiterRunConsumption(opener, closer delimiterRunState) int {
	if opener.Marker != closer.Marker || !opener.CanOpen || !closer.CanClose || opener.remaining == 0 || closer.remaining == 0 {
		return 0
	}
	if opener.Marker == '~' {
		if opener.length != closer.length || opener.length < 1 || opener.length > 2 ||
			opener.remaining != opener.length || closer.remaining != closer.length {
			return 0
		}
		return opener.length
	}
	if DelimiterRunsHaveModuloThreeConflict(opener.length, opener.CanClose, closer.length, closer.CanOpen) {
		return 0
	}
	if opener.remaining >= 2 && closer.remaining >= 2 {
		return 2
	}
	return 1
}

func activateDelimiterRunOpener(runs []delimiterRunState, index *delimiterRunOpenerIndex, runIndex int) {
	run := &runs[runIndex]
	if run.openActive || !run.CanOpen || run.remaining == 0 {
		return
	}
	run.openActive = true
	index.active = append(index.active, runIndex)
	pushDelimiterRunCategory(runs, index, runIndex)
}

func pushDelimiterRunCategory(runs []delimiterRunState, index *delimiterRunOpenerIndex, runIndex int) {
	run := runs[runIndex]
	markerIndex := delimiterRunMarkerIndex(run.Marker)
	category := delimiterRunCategory(run)
	index.byCategory[markerIndex][category] = append(
		index.byCategory[markerIndex][category],
		delimiterRunRef{index: runIndex, version: run.openVersion},
	)
}

func consumeDelimiterRunOpener(runs []delimiterRunState, index *delimiterRunOpenerIndex, runIndex, count int) {
	run := &runs[runIndex]
	run.remaining -= count
	run.openVersion++
	if run.remaining == 0 {
		run.openActive = false
		if len(index.active) != 0 && index.active[len(index.active)-1] == runIndex {
			index.active = index.active[:len(index.active)-1]
		}
		return
	}
	pushDelimiterRunCategory(runs, index, runIndex)
}

func invalidateDelimiterRunOpenersAbove(runs []delimiterRunState, index *delimiterRunOpenerIndex, opener int) {
	for len(index.active) != 0 && index.active[len(index.active)-1] > opener {
		runIndex := index.active[len(index.active)-1]
		index.active = index.active[:len(index.active)-1]
		runs[runIndex].openActive = false
		runs[runIndex].openVersion++
	}
}

func delimiterRunMarkerIndex(marker byte) int {
	switch marker {
	case '_':
		return 1
	case '~':
		return 2
	default:
		return 0
	}
}

func delimiterRunCategory(run delimiterRunState) int {
	category := run.length % 3
	if run.CanClose {
		category += 3
	}
	return category
}

func delimiterRunMatchFor(opener, closer delimiterRunState, openerIndex, closerIndex, level int) DelimiterRunMatch {
	openingEnd := opener.Start + opener.remaining
	closingStart := closer.End - closer.remaining
	return DelimiterRunMatch{
		Marker:          opener.Marker,
		Level:           level,
		OpenerRun:       openerIndex,
		CloserRun:       closerIndex,
		OpeningConsumed: Range{Start: openingEnd - level, End: openingEnd},
		ClosingConsumed: Range{Start: closingStart, End: closingStart + level},
	}
}
