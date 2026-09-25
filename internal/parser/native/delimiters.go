package native

import (
	"cmp"
	"slices"
	"sort"
	"unicode/utf8"

	"github.com/zoster81/marksplice/internal/parser"
)

type delimiterRun struct {
	segment  int
	start    int
	end      int
	marker   byte
	length   int
	canOpen  bool
	canClose bool
}

type delimiterMatch struct {
	marker            byte
	level             int
	opener            int
	closer            int
	startSegment      int
	endSegment        int
	syntaxStart       int
	syntaxEnd         int
	openingConsumed   parser.Range
	closingConsumed   parser.Range
	content           parser.Range
	hasDelimiterChild bool
}

type delimiterParseResult struct {
	nodes      []parser.Node
	matches    []delimiterMatch
	composites []compositeInline
}

func parseDelimiterObservations(source []byte, block inlineBlock, owners []inlineSpan, barriers []backtickRun, definitions []referenceDefinitionParse) delimiterParseResult {
	return parseDelimiterObservationsIndexed(source, block, owners, barriers, basicReferenceDefinitions(definitions))
}

func parseDelimiterObservationsIndexed(source []byte, block inlineBlock, owners []inlineSpan, barriers []backtickRun, definitions referenceDefinitionIndex) delimiterParseResult {
	ownerExclusions := inlineOwnerExclusions(block, owners, nil)
	composites := collectCompositeInlinesIndexed(source, block, owners, definitions, ownerExclusions)
	exclusions := appendCompositeDelimiterExclusions(ownerExclusions, block, composites)
	return parseDelimiterObservationsWithExclusions(source, block, owners, barriers, composites, exclusions)
}

func parseDelimiterObservationsWithExclusions(source []byte, block inlineBlock, owners []inlineSpan, barriers []backtickRun, composites []compositeInline, exclusions [][]parser.Range) delimiterParseResult {
	barriers = activeBacktickBarriers(barriers, exclusions)
	runs := collectDelimiterRuns(source, block, exclusions)
	matches := processDelimiters(runs)
	projection := newDelimiterProjectionIndex(len(block.segments), owners, composites, matches, barriers, runs)
	nodes := make([]parser.Node, 0, len(matches))
	for index, match := range matches {
		if !simpleDelimiterMatch(source, block, match, index, projection) {
			continue
		}
		switch match.marker {
		case '*', '_':
			kind := parser.KindEmphasis
			if match.level == 2 {
				kind = parser.KindStrong
			}
			nodes = append(nodes, parser.Node{
				Kind:   kind,
				Range:  match.content,
				Anchor: match.syntaxStart,
				Level:  match.level,
			})
		case '~':
			nodes = append(nodes, parser.Node{Kind: parser.KindStrikethrough, Range: match.content})
		}
	}
	if len(nodes) > 1 {
		slices.SortStableFunc(nodes, func(left, right parser.Node) int {
			if order := cmp.Compare(left.Range.Start, right.Range.Start); order != 0 {
				return order
			}
			return cmp.Compare(left.Range.End, right.Range.End)
		})
	}
	return delimiterParseResult{nodes: nodes, matches: matches, composites: composites}
}

func inlineOwnerExclusions(block inlineBlock, owners []inlineSpan, composites []compositeInline) [][]parser.Range {
	exclusions := inlineBlockExclusions(block)
	if len(exclusions) == 0 && (len(owners) != 0 || hasActiveComposite(composites)) {
		exclusions = ensureInlineExclusions(exclusions, len(block.segments))
	}
	for _, owner := range owners {
		if owner.segment < 0 || owner.endSegment >= len(block.segments) || owner.segment > owner.endSegment {
			continue
		}
		for segmentIndex := owner.segment; segmentIndex <= owner.endSegment; segmentIndex++ {
			segment := block.segments[segmentIndex]
			start, end := segment.Start, segment.End
			if segmentIndex == owner.segment {
				start = owner.start
			}
			if segmentIndex == owner.endSegment {
				end = owner.end
			}
			if start < end {
				exclusions[segmentIndex] = append(exclusions[segmentIndex], parser.Range{Start: start, End: end})
			}
		}
	}
	for _, composite := range composites {
		appendCompositeExclusions(exclusions, block, composite)
	}
	normalizeInlineExclusions(exclusions)
	return exclusions
}

func inlineBlockExclusions(block inlineBlock) [][]parser.Range {
	if block.prefixExclusion.Start >= block.prefixExclusion.End {
		return nil
	}
	exclusions := make([][]parser.Range, len(block.segments))
	for segmentIndex, segment := range block.segments {
		start := max(segment.Start, block.prefixExclusion.Start)
		end := min(segment.End, block.prefixExclusion.End)
		if start < end {
			exclusions[segmentIndex] = append(exclusions[segmentIndex], parser.Range{Start: start, End: end})
		}
	}
	return exclusions
}

func appendCompositeDelimiterExclusions(exclusions [][]parser.Range, block inlineBlock, composites []compositeInline) [][]parser.Range {
	if len(exclusions) == 0 && hasActiveComposite(composites) {
		exclusions = ensureInlineExclusions(exclusions, len(block.segments))
	}
	for _, composite := range composites {
		appendCompositeExclusions(exclusions, block, composite)
	}
	normalizeInlineExclusions(exclusions)
	return exclusions
}

func appendCompositeExclusions(exclusions [][]parser.Range, block inlineBlock, composite compositeInline) {
	if !composite.active || composite.segment < 0 || composite.labelEndSegment < composite.segment || composite.endSegment < composite.labelEndSegment || composite.endSegment >= len(block.segments) {
		return
	}
	appendRelationshipExclusion(exclusions, block, composite.segment, composite.start, composite.segment, composite.label.Start)
	appendRelationshipExclusion(exclusions, block, composite.labelEndSegment, composite.label.End, composite.endSegment, composite.end)
}

func activeBacktickBarriers(barriers []backtickRun, exclusions [][]parser.Range) []backtickRun {
	if len(exclusions) == 0 {
		return barriers
	}
	result := make([]backtickRun, 0, len(barriers))
	currentSegment := -1
	excludedIndex := 0
	for _, barrier := range barriers {
		if barrier.segment < 0 || barrier.segment >= len(exclusions) {
			continue
		}
		if barrier.segment != currentSegment {
			currentSegment = barrier.segment
			excludedIndex = 0
		}
		ranges := exclusions[currentSegment]
		for excludedIndex < len(ranges) && ranges[excludedIndex].End <= barrier.start {
			excludedIndex++
		}
		if excludedIndex < len(ranges) && ranges[excludedIndex].Start <= barrier.start && barrier.start < ranges[excludedIndex].End {
			continue
		}
		result = append(result, barrier)
	}
	return result
}

func collectDelimiterRuns(source []byte, block inlineBlock, exclusions [][]parser.Range) []delimiterRun {
	runs := make([]delimiterRun, 0)
	for segmentIndex, segment := range block.segments {
		segmentExclusions := inlineExclusionsAt(exclusions, segmentIndex)
		excludedIndex := 0
		for position := segment.Start; position < segment.End; {
			for excludedIndex < len(segmentExclusions) && position >= segmentExclusions[excludedIndex].End {
				excludedIndex++
			}
			if excludedIndex < len(segmentExclusions) && position >= segmentExclusions[excludedIndex].Start {
				position = segmentExclusions[excludedIndex].End
				continue
			}
			marker := source[position]
			if marker != '*' && marker != '_' && marker != '~' || inlineByteEscaped(source, segment.Start, position) {
				position++
				continue
			}
			start := position
			for position < segment.End && source[position] == marker {
				position++
			}
			length := position - start
			if marker == '~' && !strikethroughRunEligible(source, segment, start, length) {
				continue
			}
			canOpen, canClose := delimiterFlankingAtOwnedBoundary(
				source,
				segment,
				start,
				position,
				marker,
				segmentExclusions,
			)
			runs = append(runs, delimiterRun{
				segment:  segmentIndex,
				start:    start,
				end:      position,
				marker:   marker,
				length:   length,
				canOpen:  canOpen,
				canClose: canClose,
			})
		}
	}
	return runs
}

func delimiterFlankingAtOwnedBoundary(
	source []byte,
	segment parser.Range,
	start, end int,
	marker byte,
	exclusions []parser.Range,
) (bool, bool) {
	if marker != '~' || len(exclusions) == 0 {
		return parser.DelimiterFlanking(source, segment, start, end, marker)
	}
	beforeWhitespace, beforePunctuation := parser.DelimiterPrecedingClass(source, segment, start)
	afterWhitespace, afterPunctuation := parser.DelimiterFollowingClass(source, start, end, segment.End)
	if ownedInlineEndsAt(exclusions, start) {
		beforeWhitespace, beforePunctuation = false, true
	}
	return parser.DelimiterFlankingFromClasses(
		beforeWhitespace,
		beforePunctuation,
		afterWhitespace,
		afterPunctuation,
		marker,
	)
}

func ownedInlineEndsAt(exclusions []parser.Range, position int) bool {
	index := sort.Search(len(exclusions), func(index int) bool {
		return exclusions[index].End >= position
	})
	return index < len(exclusions) && exclusions[index].End == position
}

func hasActiveComposite(composites []compositeInline) bool {
	for _, composite := range composites {
		if composite.active {
			return true
		}
	}
	return false
}

func strikethroughRunEligible(source []byte, segment parser.Range, start, length int) bool {
	if length < 1 || length > 2 {
		return false
	}
	before, ok := delimiterPrecedingRune(source, segment, start)
	return !ok || before != '~' || inlineByteEscaped(source, segment.Start, start-1)
}

func delimiterPrecedingRune(source []byte, segment parser.Range, position int) (rune, bool) {
	if position <= segment.Start {
		return 0, false
	}
	index := position - 1
	for index >= segment.Start && !utf8.RuneStart(source[index]) {
		index--
	}
	if index < segment.Start {
		return 0, false
	}
	rune_, _ := utf8.DecodeRune(source[index:position])
	return rune_, true
}

func processDelimiters(runs []delimiterRun) []delimiterMatch {
	sharedRuns := make([]parser.DelimiterRun, len(runs))
	for index, run := range runs {
		sharedRuns[index] = parser.DelimiterRun{
			Start:    run.start,
			End:      run.end,
			Marker:   run.marker,
			CanOpen:  run.canOpen,
			CanClose: run.canClose,
		}
	}
	resolved := parser.ResolveDelimiterRuns(sharedRuns)
	matches := make([]delimiterMatch, 0, len(resolved))
	maxMatchedOpener := -1
	for _, resolvedMatch := range resolved {
		opener := runs[resolvedMatch.OpenerRun]
		closer := runs[resolvedMatch.CloserRun]
		match := delimiterMatch{
			marker:            resolvedMatch.Marker,
			level:             resolvedMatch.Level,
			opener:            opener.start,
			closer:            closer.start,
			startSegment:      opener.segment,
			endSegment:        closer.segment,
			syntaxStart:       opener.start,
			syntaxEnd:         closer.end,
			openingConsumed:   resolvedMatch.OpeningConsumed,
			closingConsumed:   resolvedMatch.ClosingConsumed,
			content:           parser.Range{Start: opener.end, End: closer.start},
			hasDelimiterChild: maxMatchedOpener >= resolvedMatch.OpenerRun,
		}
		matches = append(matches, match)
		maxMatchedOpener = max(maxMatchedOpener, resolvedMatch.OpenerRun)
	}
	return matches
}

type delimiterContentKey struct {
	segment int
	start   int
	end     int
}

type delimiterRunSegment struct {
	start int
	end   int
}

type delimiterRunIndex struct {
	segmentCount int
	runs         []delimiterRun
	single       delimiterRunSegment
	bySegment    []delimiterRunSegment
}

func newDelimiterRunIndex(segmentCount int, runs []delimiterRun) delimiterRunIndex {
	if segmentCount <= 0 {
		return delimiterRunIndex{runs: runs}
	}
	index := delimiterRunIndex{segmentCount: segmentCount, runs: runs}
	if segmentCount > 1 {
		index.bySegment = make([]delimiterRunSegment, segmentCount)
	}
	for runIndex, run := range runs {
		if run.segment < 0 || run.segment >= segmentCount || run.start >= run.end {
			continue
		}
		segment := index.segmentSpan(run.segment)
		if segment.start == segment.end {
			segment.start = runIndex
		}
		segment.end = runIndex + 1
	}
	return index
}

func (index *delimiterRunIndex) segmentSpan(segment int) *delimiterRunSegment {
	if index.segmentCount == 1 {
		return &index.single
	}
	return &index.bySegment[segment]
}

func (index delimiterRunIndex) segmentRuns(segment int) []delimiterRun {
	if segment < 0 || segment >= index.segmentCount {
		return nil
	}
	span := index.single
	if index.segmentCount > 1 {
		span = index.bySegment[segment]
	}
	if span.end <= span.start {
		return nil
	}
	return index.runs[span.start:span.end]
}

func (index delimiterRunIndex) preservesSingleTextChild(segment, start, end int) bool {
	if start >= end {
		return true
	}
	runs := index.segmentRuns(segment)
	position := sort.Search(len(runs), func(position int) bool { return runs[position].start >= start })
	if position == len(runs) || runs[position].start >= end {
		return true
	}
	return contiguousDelimiterRunsReach(runs, position, runs[position].start, end)
}

func (index delimiterRunIndex) coversRange(segment, start, end int) bool {
	if start >= end {
		return false
	}
	runs := index.segmentRuns(segment)
	position := sort.Search(len(runs), func(position int) bool { return runs[position].start >= start })
	return position < len(runs) && runs[position].start == start && contiguousDelimiterRunsReach(runs, position, start, end)
}

func contiguousDelimiterRunsReach(runs []delimiterRun, position, start, end int) bool {
	cursor := start
	for position < len(runs) && cursor < end {
		run := runs[position]
		if run.start != cursor || run.end > end {
			return false
		}
		cursor = run.end
		position++
	}
	return cursor == end
}

type delimiterProjectionIndex struct {
	ownerStarts     inlineStartIndex
	compositeStarts inlineStartIndex
	barrierStarts   inlineStartIndex
	delimiterRuns   delimiterRunIndex
	firstContent    map[delimiterContentKey]int
}

func newDelimiterProjectionIndex(segmentCount int, owners []inlineSpan, composites []compositeInline, matches []delimiterMatch, barriers []backtickRun, runs []delimiterRun) delimiterProjectionIndex {
	index := delimiterProjectionIndex{
		ownerStarts:     newInlineStartIndex(segmentCount),
		compositeStarts: newInlineStartIndex(segmentCount),
		barrierStarts:   newInlineStartIndex(segmentCount),
		delimiterRuns:   newDelimiterRunIndex(segmentCount, runs),
		firstContent:    make(map[delimiterContentKey]int, len(matches)),
	}
	for _, owner := range owners {
		index.ownerStarts.add(owner.segment, owner.start)
	}
	for _, composite := range composites {
		if composite.active {
			index.compositeStarts.add(composite.segment, composite.start)
		}
	}
	for _, barrier := range barriers {
		index.barrierStarts.add(barrier.segment, barrier.start)
	}
	for matchIndex, match := range matches {
		key := delimiterContentKey{segment: match.startSegment, start: match.content.Start, end: match.content.End}
		if _, exists := index.firstContent[key]; !exists {
			index.firstContent[key] = matchIndex
		}
	}
	index.ownerStarts.finalize()
	index.compositeStarts.finalize()
	index.barrierStarts.finalize()
	return index
}

func simpleDelimiterMatch(source []byte, block inlineBlock, match delimiterMatch, index int, projection delimiterProjectionIndex) bool {
	if !simpleDelimiterRange(source, block, match, projection.delimiterRuns) {
		return false
	}
	if projection.ownerStarts.anyIn(match.startSegment, match.content.Start, match.content.End) ||
		projection.compositeStarts.anyIn(match.startSegment, match.content.Start, match.content.End) ||
		projection.barrierStarts.anyIn(match.startSegment, match.content.Start+1, match.content.End) {
		return false
	}
	if match.hasDelimiterChild ||
		!projection.delimiterRuns.preservesSingleTextChild(match.startSegment, match.content.Start, match.content.End) {
		return false
	}
	key := delimiterContentKey{segment: match.startSegment, start: match.content.Start, end: match.content.End}
	return projection.firstContent[key] == index
}

func simpleDelimiterRange(source []byte, block inlineBlock, match delimiterMatch, runs delimiterRunIndex) bool {
	if match.startSegment != match.endSegment || match.content.Start >= match.content.End {
		return false
	}
	segment := block.segments[match.startSegment]
	if match.content.Start < segment.Start || match.content.End > segment.End {
		return false
	}
	return simpleDelimiterTextContent(source, block, match, runs)
}

func simpleDelimiterTextContent(source []byte, block inlineBlock, match delimiterMatch, runs delimiterRunIndex) bool {
	segment := block.segments[match.startSegment]
	content := match.content
	if content.End-content.Start == 2 && source[content.Start] == '!' && source[content.Start+1] == '[' {
		return true
	}
	closesAfterMatch := false
	closesAfterMatchKnown := false
	for position := content.Start; position < content.End; position++ {
		if inlineByteEscaped(source, segment.Start, position) || source[position] != '[' {
			continue
		}
		next, ok := projectableDelimiterBracket(source, block, match, runs, position, &closesAfterMatch, &closesAfterMatchKnown)
		if !ok {
			return false
		}
		position = next
	}
	return true
}

func projectableDelimiterBracket(source []byte, block inlineBlock, match delimiterMatch, runs delimiterRunIndex, position int, closesAfterMatch, closesAfterMatchKnown *bool) (int, bool) {
	content := match.content
	if position+1 < content.End && source[position+1] == ']' {
		return position + 1, true
	}
	if position == content.Start && content.End-content.Start == 1 {
		return position, true
	}
	if !delimiterBracketTailMergeable(match, runs, position) {
		return position, false
	}
	if !*closesAfterMatchKnown {
		*closesAfterMatch = delimiterLabelStateClosesAfterMatch(source, block, match)
		*closesAfterMatchKnown = true
	}
	return position, *closesAfterMatch
}

func delimiterBracketTailMergeable(match delimiterMatch, runs delimiterRunIndex, position int) bool {
	return position == match.content.End-1 ||
		position+1 < match.content.End && runs.coversRange(match.startSegment, position+1, match.content.End)
}

func delimiterLabelStateClosesAfterMatch(source []byte, block inlineBlock, match delimiterMatch) bool {
	depth := 1
	for segmentIndex := match.endSegment; segmentIndex < len(block.segments); segmentIndex++ {
		segment := block.segments[segmentIndex]
		position := segment.Start
		if segmentIndex == match.endSegment {
			position = max(position, match.syntaxEnd)
		}
		for ; position < segment.End; position++ {
			if inlineByteEscaped(source, segment.Start, position) {
				continue
			}
			switch source[position] {
			case '[':
				depth++
			case ']':
				depth--
				if depth == 0 {
					return true
				}
			}
		}
	}
	return false
}
