package native

import (
	"sort"

	"github.com/zoster81/marksplice/internal/parser"
)

// DelimiterTopologyPair describes one semantic delimiter pair expected in an
// emitted inline candidate. Ranges identify the exact delimiter bytes owned by
// that semantic wrapper, including its share of any coalesced physical run.
type DelimiterTopologyPair struct {
	Marker  byte
	Opening parser.Range
	Closing parser.Range
}

// DelimiterTopologyBoundaryDemand records the saturated level-one semantic
// demand owned by one physical boundary run.
type DelimiterTopologyBoundaryDemand struct {
	OpenLevelOne  uint8
	CloseLevelOne uint8
}

// DelimiterTopologyBoundaryFacts describes the physical first/last delimiter
// runs that contain the expected semantic topology.
type DelimiterTopologyBoundaryFacts struct {
	Empty       bool
	First       parser.DelimiterRun
	Last        parser.DelimiterRun
	FirstDemand DelimiterTopologyBoundaryDemand
	LastDemand  DelimiterTopologyBoundaryDemand
	SingleRun   bool
}

// DelimiterTopologyTransferFacts is the complete parser-owned compact transfer
// observation derived from one scan of a canonical inline candidate.
type DelimiterTopologyTransferFacts struct {
	Boundary            DelimiterTopologyBoundaryFacts
	IncomingOpener      uint8
	StarForbidden       uint8
	UnderscoreForbidden uint8
	TildeForbidden      uint8
}

// DelimiterTopologyBoundaryFactsForCandidate derives physical boundary-run
// facts with the same Native scanner and owner exclusions used for topology
// validation. Expected pairs remain the semantic authority for level-one demand.
func DelimiterTopologyBoundaryFactsForCandidate(
	source []byte,
	owners []parser.Range,
	expected []DelimiterTopologyPair,
) (DelimiterTopologyBoundaryFacts, bool) {
	runs, _, ok := delimiterTopologyLeadingRunPreparation(source, owners, expected)
	if !ok {
		return DelimiterTopologyBoundaryFacts{}, false
	}
	return delimiterTopologyBoundaryFactsFromPrepared(runs, expected)
}

func delimiterTopologyBoundaryFactsFromPrepared(
	runs []delimiterRun,
	expected []DelimiterTopologyPair,
) (DelimiterTopologyBoundaryFacts, bool) {
	if len(runs) == 0 {
		return DelimiterTopologyBoundaryFacts{Empty: true}, len(expected) == 0
	}
	demands, ok := delimiterTopologyBoundaryDemands(runs, expected)
	if !ok {
		return DelimiterTopologyBoundaryFacts{}, false
	}
	return DelimiterTopologyBoundaryFacts{
		First:       delimiterTopologySharedRun(runs[0]),
		Last:        delimiterTopologySharedRun(runs[len(runs)-1]),
		FirstDemand: demands[0],
		LastDemand:  demands[len(demands)-1],
		SingleRun:   len(runs) == 1,
	}, true
}

func delimiterTopologyBoundaryDemands(
	runs []delimiterRun,
	expected []DelimiterTopologyPair,
) ([]DelimiterTopologyBoundaryDemand, bool) {
	demands := make([]DelimiterTopologyBoundaryDemand, len(runs))
	for _, pair := range expected {
		opener := delimiterTopologyContainingRun(runs, pair.Marker, pair.Opening)
		closer := delimiterTopologyContainingRun(runs, pair.Marker, pair.Closing)
		if opener < 0 || closer < 0 {
			return nil, false
		}
		if pair.Opening.End-pair.Opening.Start != 1 {
			continue
		}
		delimiterTopologySaturatingIncrement(&demands[opener].OpenLevelOne)
		delimiterTopologySaturatingIncrement(&demands[closer].CloseLevelOne)
	}
	return demands, true
}

func delimiterTopologyContainingRun(
	runs []delimiterRun,
	marker byte,
	range_ parser.Range,
) int {
	// Native scanning produces non-overlapping runs in source order. Expected
	// pairs have non-empty ranges, so at most one run can own this start byte.
	index := sort.Search(len(runs), func(i int) bool { return runs[i].end > range_.Start })
	if index < len(runs) {
		run := runs[index]
		if run.marker == marker && run.start <= range_.Start && range_.End <= run.end {
			return index
		}
	}
	return -1
}

func delimiterTopologySharedRun(run delimiterRun) parser.DelimiterRun {
	return parser.DelimiterRun{
		Start: run.start, End: run.end, Marker: run.marker,
		CanOpen: run.canOpen, CanClose: run.canClose,
	}
}

func delimiterTopologySaturatingIncrement(value *uint8) {
	if *value < 2 {
		*value++
	}
}

// DelimiterTopologyMatches proves an expected delimiter topology using the
// same Native delimiter scanner, owned-object boundary classification, and
// shared resolver used by ordinary parsing. Owner ranges are lexical regions
// whose bytes must not be reinterpreted as delimiter syntax.
func DelimiterTopologyMatches(
	source []byte,
	owners []parser.Range,
	expected []DelimiterTopologyPair,
) bool {
	runs, want, ok := delimiterTopologyLeadingRunPreparation(source, owners, expected)
	if !ok {
		return false
	}
	return delimiterTopologyMatchesExpected(processDelimiters(runs), want, len(expected), 0)
}

// DelimiterTopologyMatchesInContext proves that all expected delimiter pairs
// owned by source remain exact when source is embedded between prefix and
// suffix. Extra matches are allowed only when both consumed delimiter ranges
// lie completely outside source. This lets callers represent an intermediate
// subtree whose delimiter topology is completed by an ancestor without
// reimplementing Native delimiter resolution.
func DelimiterTopologyMatchesInContext(
	source []byte,
	owners []parser.Range,
	expected []DelimiterTopologyPair,
	prefix, suffix []byte,
) bool {
	offset := len(prefix)
	combined := make([]byte, 0, len(prefix)+len(source)+len(suffix))
	combined = append(combined, prefix...)
	combined = append(combined, source...)
	combined = append(combined, suffix...)

	shiftedOwners, ok := delimiterTopologyShiftOwners(owners, len(source), offset)
	if !ok {
		return false
	}
	exclusions, ok := delimiterTopologyExclusions(combined, shiftedOwners)
	if !ok {
		return false
	}
	want, ok := delimiterTopologyExpected(source, expected)
	if !ok {
		return false
	}
	want = delimiterTopologyShiftExpected(want, offset)

	block := inlineBlock{segments: []parser.Range{{Start: 0, End: len(combined)}}}
	runs := collectDelimiterRuns(combined, block, [][]parser.Range{exclusions})
	sourceRange := parser.Range{Start: offset, End: offset + len(source)}
	return delimiterTopologyContextMatches(processDelimiters(runs), want, sourceRange)
}

func delimiterTopologyShiftOwners(
	owners []parser.Range,
	sourceLength, offset int,
) ([]parser.Range, bool) {
	shifted := make([]parser.Range, len(owners))
	for index, owner := range owners {
		if !owner.Valid(sourceLength) {
			return nil, false
		}
		shifted[index] = parser.Range{Start: owner.Start + offset, End: owner.End + offset}
	}
	return shifted, true
}

func delimiterTopologyShiftExpected(
	expected map[delimiterTopologyKey]int,
	offset int,
) map[delimiterTopologyKey]int {
	shifted := make(map[delimiterTopologyKey]int, len(expected))
	for key, count := range expected {
		key.opening.Start += offset
		key.opening.End += offset
		key.closing.Start += offset
		key.closing.End += offset
		shifted[key] += count
	}
	return shifted
}

func delimiterTopologyContextMatches(
	matches []delimiterMatch,
	want map[delimiterTopologyKey]int,
	sourceRange parser.Range,
) bool {
	got := make(map[delimiterTopologyKey]int, len(want))
	for _, match := range matches {
		key := delimiterTopologyKey{
			marker:  match.marker,
			opening: match.openingConsumed,
			closing: match.closingConsumed,
		}
		if want[key] != 0 {
			got[key]++
			continue
		}
		if delimiterTopologyRangesOverlap(match.openingConsumed, sourceRange) ||
			delimiterTopologyRangesOverlap(match.closingConsumed, sourceRange) {
			return false
		}
	}
	if len(got) != len(want) {
		return false
	}
	for key, count := range want {
		if got[key] != count {
			return false
		}
	}
	return true
}

func delimiterTopologyRangesOverlap(left, right parser.Range) bool {
	return left.Start < right.End && right.Start < left.End
}

// DelimiterTopologyPreservedWithLeadingRun reports whether adding a synthetic
// opening delimiter run before a candidate leaves the candidate's exact
// semantic delimiter topology unchanged. The candidate itself is scanned by
// Native; only the external run's already-classified shape is supplied.
func DelimiterTopologyPreservedWithLeadingRun(
	source []byte,
	owners []parser.Range,
	expected []DelimiterTopologyPair,
	marker byte,
	width int,
	canClose bool,
) bool {
	if marker != '*' && marker != '_' && marker != '~' || width <= 0 {
		return false
	}
	runs, want, ok := delimiterTopologyLeadingRunPreparation(source, owners, expected)
	if !ok {
		return false
	}
	return delimiterTopologyPreservedWithPreparedLeadingRun(
		runs, want, len(expected), marker, width, canClose,
	)
}

// DelimiterTopologyTransferFactsForCandidate derives all compact transfer
// facts from one Native scan of the candidate. Tilde probes are optional and
// use width-one/two categories (bits 1, 2, 4, 5) when requested.
func DelimiterTopologyTransferFactsForCandidate(
	source []byte,
	owners []parser.Range,
	expected []DelimiterTopologyPair,
	withTilde bool,
) (DelimiterTopologyTransferFacts, bool) {
	runs, want, ok := delimiterTopologyLeadingRunPreparation(source, owners, expected)
	if !ok {
		return DelimiterTopologyTransferFacts{}, false
	}
	boundary, ok := delimiterTopologyBoundaryFactsFromPrepared(runs, expected)
	if !ok {
		return DelimiterTopologyTransferFacts{}, false
	}
	incoming, ok := delimiterTopologyIncomingOpenerMaskPrepared(
		source, runs, want, len(expected),
	)
	if !ok {
		return DelimiterTopologyTransferFacts{}, false
	}
	star, underscore, tilde := delimiterTopologyLeadingRunForbiddenMasksPrepared(
		runs, want, len(expected), withTilde,
	)
	return DelimiterTopologyTransferFacts{
		Boundary:            boundary,
		IncomingOpener:      incoming,
		StarForbidden:       star,
		UnderscoreForbidden: underscore,
		TildeForbidden:      tilde,
	}, true
}

// DelimiterTopologyIncomingOpenerMask returns the six-bit survival response
// for a same-marker opener coalesced into the candidate's first physical run.
// Bits are ordered by width 1 then 2, each with external-before classes
// whitespace, punctuation, and other. One Native scan is reused for all probes.
func DelimiterTopologyIncomingOpenerMask(
	source []byte,
	owners []parser.Range,
	expected []DelimiterTopologyPair,
) (uint8, bool) {
	runs, want, ok := delimiterTopologyLeadingRunPreparation(source, owners, expected)
	if !ok {
		return 0, false
	}
	return delimiterTopologyIncomingOpenerMaskPrepared(source, runs, want, len(expected))
}

func delimiterTopologyIncomingOpenerMaskPrepared(
	source []byte,
	runs []delimiterRun,
	want map[delimiterTopologyKey]int,
	expectedCount int,
) (uint8, bool) {
	if len(runs) == 0 {
		return 0, expectedCount == 0
	}
	first := runs[0]
	afterWhitespace, afterPunctuation := parser.DelimiterFollowingClass(
		source, first.start, first.end, len(source),
	)

	probe := make([]delimiterRun, len(runs))
	var resolver delimiterResolver
	var mask uint8
	var bit uint8
	for width := 1; width <= 2; width++ {
		for class := 0; class < 3; class++ {
			beforeWhitespace, beforePunctuation := delimiterTopologyIncomingBoundaryClass(class)
			copy(probe, runs)
			probe[0].start -= width
			probe[0].length += width
			probe[0].canOpen, probe[0].canClose = parser.DelimiterFlankingFromClasses(
				beforeWhitespace, beforePunctuation,
				afterWhitespace, afterPunctuation,
				first.marker,
			)
			if probe[0].canOpen &&
				delimiterTopologyMatchesExpected(resolver.resolve(probe), want, expectedCount, 0) {
				mask |= 1 << bit
			}
			bit++
		}
	}
	return mask, true
}

func delimiterTopologyIncomingBoundaryClass(class int) (whitespace, punctuation bool) {
	switch class {
	case 0:
		return true, false
	case 1:
		return false, true
	default:
		return false, false
	}
}

func delimiterTopologyMatchesExpected(
	matches []delimiterMatch,
	want map[delimiterTopologyKey]int,
	expectedCount int,
	coordinateOffset int,
) bool {
	if len(matches) != expectedCount {
		return false
	}
	got := make(map[delimiterTopologyKey]int, len(matches))
	for _, match := range matches {
		got[delimiterTopologyKey{
			marker: match.marker,
			opening: parser.Range{
				Start: match.openingConsumed.Start - coordinateOffset,
				End:   match.openingConsumed.End - coordinateOffset,
			},
			closing: parser.Range{
				Start: match.closingConsumed.Start - coordinateOffset,
				End:   match.closingConsumed.End - coordinateOffset,
			},
		}]++
	}
	if len(got) != len(want) {
		return false
	}
	for key, count := range want {
		if got[key] != count {
			return false
		}
	}
	return true
}

// DelimiterTopologyLeadingRunForbiddenMasks returns the six-category leading
// run rejection masks for '*' and '_' from one Native scan of the candidate.
func DelimiterTopologyLeadingRunForbiddenMasks(
	source []byte,
	owners []parser.Range,
	expected []DelimiterTopologyPair,
) (star, underscore uint8, ok bool) {
	runs, want, ok := delimiterTopologyLeadingRunPreparation(source, owners, expected)
	if !ok {
		return 0, 0, false
	}
	star, underscore, _ = delimiterTopologyLeadingRunForbiddenMasksPrepared(
		runs, want, len(expected), false,
	)
	return star, underscore, true
}

func delimiterTopologyLeadingRunForbiddenMasksPrepared(
	runs []delimiterRun,
	want map[delimiterTopologyKey]int,
	expectedCount int,
	withTilde bool,
) (star, underscore, tilde uint8) {
	probe := make([]delimiterRun, 0, len(runs)+1)
	var resolver delimiterResolver
	for category := 0; category < 6; category++ {
		width := category % 3
		if width == 0 {
			width = 3
		}
		canClose := category >= 3
		if !delimiterTopologyPreservedWithPreparedLeadingRunUsing(
			probe, runs, want, expectedCount, '*', width, canClose, &resolver,
		) {
			star |= 1 << category
		}
		if !delimiterTopologyPreservedWithPreparedLeadingRunUsing(
			probe, runs, want, expectedCount, '_', width, canClose, &resolver,
		) {
			underscore |= 1 << category
		}
		if withTilde && width != 3 && !delimiterTopologyPreservedWithPreparedLeadingRunUsing(
			probe, runs, want, expectedCount, '~', width, canClose, &resolver,
		) {
			tilde |= 1 << category
		}
	}
	return star, underscore, tilde
}

func delimiterTopologyLeadingRunPreparation(
	source []byte,
	owners []parser.Range,
	expected []DelimiterTopologyPair,
) ([]delimiterRun, map[delimiterTopologyKey]int, bool) {
	exclusions, ok := delimiterTopologyExclusions(source, owners)
	if !ok {
		return nil, nil, false
	}
	want, ok := delimiterTopologyExpected(source, expected)
	if !ok {
		return nil, nil, false
	}
	block := inlineBlock{segments: []parser.Range{{Start: 0, End: len(source)}}}
	return collectDelimiterRuns(source, block, [][]parser.Range{exclusions}), want, true
}

func delimiterTopologyPreservedWithPreparedLeadingRun(
	runs []delimiterRun,
	want map[delimiterTopologyKey]int,
	expectedCount int,
	marker byte,
	width int,
	canClose bool,
) bool {
	var resolver delimiterResolver
	return delimiterTopologyPreservedWithPreparedLeadingRunUsing(
		make([]delimiterRun, 0, len(runs)+1),
		runs, want, expectedCount, marker, width, canClose, &resolver,
	)
}

func delimiterTopologyPreservedWithPreparedLeadingRunUsing(
	probe []delimiterRun,
	runs []delimiterRun,
	want map[delimiterTopologyKey]int,
	expectedCount int,
	marker byte,
	width int,
	canClose bool,
	resolver *delimiterResolver,
) bool {
	delta := width + 1
	probe = probe[:0]
	probe = append(probe, delimiterRun{
		segment: 0, start: 0, end: width, marker: marker,
		length: width, canOpen: true, canClose: canClose,
	})
	for _, run := range runs {
		run.start += delta
		run.end += delta
		probe = append(probe, run)
	}
	return delimiterTopologyMatchesExpected(resolver.resolve(probe), want, expectedCount, delta)
}

type delimiterTopologyKey struct {
	marker  byte
	opening parser.Range
	closing parser.Range
}

func delimiterTopologyExclusions(source []byte, owners []parser.Range) ([]parser.Range, bool) {
	exclusions := append([]parser.Range(nil), owners...)
	for _, range_ := range exclusions {
		if !range_.Valid(len(source)) {
			return nil, false
		}
	}
	return normalizeInlineRanges(exclusions), true
}

func delimiterTopologyExpected(
	source []byte,
	expected []DelimiterTopologyPair,
) (map[delimiterTopologyKey]int, bool) {
	result := make(map[delimiterTopologyKey]int, len(expected))
	for _, pair := range expected {
		if pair.Marker != '*' && pair.Marker != '_' && pair.Marker != '~' {
			return nil, false
		}
		if !pair.Opening.Valid(len(source)) || pair.Opening.Start >= pair.Opening.End ||
			!pair.Closing.Valid(len(source)) || pair.Closing.Start >= pair.Closing.End {
			return nil, false
		}
		if pair.Opening.End-pair.Opening.Start != pair.Closing.End-pair.Closing.Start {
			return nil, false
		}
		key := delimiterTopologyKey{
			marker:  pair.Marker,
			opening: pair.Opening,
			closing: pair.Closing,
		}
		result[key]++
	}
	return result, true
}
