package rendermarkdown

import (
	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

type canonicalInlineBoundaryClass uint8

const (
	canonicalInlineBoundaryWhitespace canonicalInlineBoundaryClass = iota
	canonicalInlineBoundaryPunctuation
	canonicalInlineBoundaryOther
)

func canonicalInlineBoundaryClassFromFlags(
	whitespace, punctuation bool,
) canonicalInlineBoundaryClass {
	switch {
	case whitespace:
		return canonicalInlineBoundaryWhitespace
	case punctuation:
		return canonicalInlineBoundaryPunctuation
	default:
		return canonicalInlineBoundaryOther
	}
}

func canonicalInlineBoundaryRunFromEmission(
	output []byte,
	run parser.DelimiterRun,
) canonicalInlineBoundaryRun {
	segment := parser.Range{Start: 0, End: len(output)}
	beforeWhitespace, beforePunctuation := parser.DelimiterPrecedingClass(
		output, segment, run.Start,
	)
	afterWhitespace, afterPunctuation := parser.DelimiterFollowingClass(
		output, run.Start, run.End, segment.End,
	)
	length := run.End - run.Start
	bucket := uint8(2)
	if length <= 1 {
		bucket = 1
	}
	return canonicalInlineBoundaryRun{
		present:           true,
		marker:            run.Marker,
		lengthMod3:        uint8(length % 3),
		lengthBucket:      bucket,
		beforeWhitespace:  beforeWhitespace,
		beforePunctuation: beforePunctuation,
		afterWhitespace:   afterWhitespace,
		afterPunctuation:  afterPunctuation,
	}
}

func canonicalInlineEdgeClass(output []byte, left bool) canonicalInlineBoundaryClass {
	if len(output) == 0 {
		return canonicalInlineBoundaryWhitespace
	}
	if left {
		whitespace, punctuation := parser.DelimiterFollowingClass(output, 0, 0, len(output))
		return canonicalInlineBoundaryClassFromFlags(whitespace, punctuation)
	}
	whitespace, punctuation := parser.DelimiterPrecedingClass(
		output, parser.Range{Start: 0, End: len(output)}, len(output),
	)
	return canonicalInlineBoundaryClassFromFlags(whitespace, punctuation)
}

func canonicalInlineForeignForbiddenMasks(
	source []byte,
	owners []parser.Range,
	expected []native.DelimiterTopologyPair,
) (star, underscore uint8, ok bool) {
	return native.DelimiterTopologyLeadingRunForbiddenMasks(source, owners, expected)
}

func canonicalInlineIncomingOpenerMask(
	source []byte,
	owners []parser.Range,
	expected []native.DelimiterTopologyPair,
) (uint8, bool) {
	return native.DelimiterTopologyIncomingOpenerMask(source, owners, expected)
}

func canonicalInlineTransferStateFromFacts(
	source []byte,
	facts native.DelimiterTopologyTransferFacts,
) canonicalInlineTransferState {
	state := canonicalInlineTransferStateFromBoundaryFacts(source, facts.Boundary)
	state.incomingOpener = facts.IncomingOpener
	state.starForbidden = facts.StarForbidden
	state.underscoreForbidden = facts.UnderscoreForbidden
	return state
}

func canonicalInlineTransferBoundaryState(
	source []byte,
	owners []parser.Range,
	expected []native.DelimiterTopologyPair,
) (canonicalInlineTransferState, bool) {
	facts, ok := native.DelimiterTopologyBoundaryFactsForCandidate(source, owners, expected)
	if !ok {
		return canonicalInlineTransferState{}, false
	}
	return canonicalInlineTransferStateFromBoundaryFacts(source, facts), true
}

func canonicalInlineTransferStateFromBoundaryFacts(
	source []byte,
	facts native.DelimiterTopologyBoundaryFacts,
) canonicalInlineTransferState {
	if facts.Empty {
		return canonicalInlineTransferState{empty: true}
	}
	return canonicalInlineTransferState{
		first:            canonicalInlineBoundaryRunFromEmission(source, facts.First),
		last:             canonicalInlineBoundaryRunFromEmission(source, facts.Last),
		firstDemand:      canonicalInlineBoundaryDemand{openLevelOne: facts.FirstDemand.OpenLevelOne, closeLevelOne: facts.FirstDemand.CloseLevelOne},
		lastDemand:       canonicalInlineBoundaryDemand{openLevelOne: facts.LastDemand.OpenLevelOne, closeLevelOne: facts.LastDemand.CloseLevelOne},
		leftEdge:         canonicalInlineEdgeClass(source, true),
		rightEdge:        canonicalInlineEdgeClass(source, false),
		firstTouchesLeft: facts.First.Start == 0,
		lastTouchesRight: facts.Last.End == len(source),
		singleRun:        facts.SingleRun,
	}
}

func canonicalInlineForeignForbiddenMask(
	source []byte,
	owners []parser.Range,
	expected []native.DelimiterTopologyPair,
	marker byte,
) uint8 {
	star, underscore, ok := canonicalInlineForeignForbiddenMasks(source, owners, expected)
	if !ok {
		return 0x3f
	}
	if marker == '*' {
		return star
	}
	if marker == '_' {
		return underscore
	}
	return 0x3f
}

type canonicalInlineBoundaryRun struct {
	present           bool
	marker            byte
	lengthMod3        uint8
	lengthBucket      uint8
	beforeWhitespace  bool
	beforePunctuation bool
	afterWhitespace   bool
	afterPunctuation  bool
}

type canonicalInlineBoundaryDemand struct {
	openLevelOne  uint8
	closeLevelOne uint8
}

type canonicalInlineTransferState struct {
	empty               bool
	starForbidden       uint8
	underscoreForbidden uint8
	first               canonicalInlineBoundaryRun
	last                canonicalInlineBoundaryRun
	firstDemand         canonicalInlineBoundaryDemand
	lastDemand          canonicalInlineBoundaryDemand
	incomingOpener      uint8
	leftEdge            canonicalInlineBoundaryClass
	rightEdge           canonicalInlineBoundaryClass
	firstTouchesLeft    bool
	lastTouchesRight    bool
	singleRun           bool
}

type canonicalInlineSelectorState struct {
	base                 canonicalInlineTransferState
	extendedWWW          canonicalBareWWWContinuation
	tildeOpener          uint8
	deferredContinuation bool
}

type canonicalInlinePruneKey struct {
	state              canonicalInlineSelectorState
	pendingRawTab      bool
	pendingRawTabState canonicalInlineSelectorState
}

type canonicalInlinePruneDisposition uint8

const (
	canonicalInlinePruneMarkerless canonicalInlinePruneDisposition = iota
	canonicalInlinePruneDeferred
	canonicalInlinePruneKeyed
)

func canonicalInlinePruneDispositionForEmitted(
	hasDelimiters bool,
	state canonicalInlineSelectorState,
) canonicalInlinePruneDisposition {
	if !hasDelimiters {
		return canonicalInlinePruneMarkerless
	}
	if state.deferredContinuation {
		return canonicalInlinePruneDeferred
	}
	return canonicalInlinePruneKeyed
}

func canonicalInlinePruneKeyWithPending(
	state canonicalInlineSelectorState,
	pendingState canonicalInlineSelectorState,
	pending bool,
) canonicalInlinePruneKey {
	key := canonicalInlinePruneKey{state: state}
	if pending {
		key.pendingRawTab = true
		key.pendingRawTabState = pendingState
	}
	return key
}

func canonicalInlinePendingOnlyPruneKey(
	pendingState canonicalInlineSelectorState,
	pending bool,
) (canonicalInlinePruneKey, bool) {
	if !pending {
		return canonicalInlinePruneKey{}, false
	}
	return canonicalInlinePruneKey{
		pendingRawTab:      true,
		pendingRawTabState: pendingState,
	}, true
}
