package parser

import "strings"

type bareWWWPhase uint8

const (
	bareWWWDomain bareWWWPhase = iota
	bareWWWPath
	bareWWWTerminated
	bareWWWDead
)

// BareWWWContinuation incrementally tracks the parser-owned lexical ownership
// of one GFM bare autolink candidate that starts with "www.".
type BareWWWContinuation struct {
	phase        bareWWWPhase
	domain       bareWWWDomainState
	path         bareWWWPathState
	ownerPresent bool
	ownerEnd     int
}

type bareWWWDomainState struct {
	rawEnd                int
	ownerEnd              int
	ownerPresent          bool
	labelCount            int
	trailingDots          uint8
	currentHasUnderscore  bool
	previousHasUnderscore bool
	dead                  bool
}

type bareWWWPathState struct {
	rawEnd         int
	ownerEnd       int
	parenBalance   int
	punctuationLen int
	closeLen       int
	entityPrefix   int
	entityComplete int
	terminated     bool
}

// StartBareWWWContinuation starts a candidate immediately after the "www."
// prefix. The candidate is initially active but does not yet own an autolink.
func StartBareWWWContinuation() BareWWWContinuation {
	const prefixLength = len("www.")
	return BareWWWContinuation{
		phase: bareWWWDomain,
		domain: bareWWWDomainState{
			rawEnd:   prefixLength,
			ownerEnd: prefixLength,
		},
		ownerEnd: prefixLength,
	}
}

// BareWWWContinuationFromAcceptedValue rebuilds continuation state from one
// already-accepted bare "www." autolink semantic value.
func BareWWWContinuationFromAcceptedValue(value string) (BareWWWContinuation, bool) {
	const prefix = "www."
	if !strings.HasPrefix(value, prefix) || len(value) == len(prefix) {
		return BareWWWContinuation{}, false
	}
	state := StartBareWWWContinuation()
	for index := len(prefix); index < len(value); index++ {
		if !state.AppendByte(value[index]) {
			return BareWWWContinuation{}, false
		}
	}
	end, ok := state.Owner()
	if !ok || end != len(value) {
		return BareWWWContinuation{}, false
	}
	return state, true
}

// AppendByte advances the candidate by one byte. It reports whether later
// bytes can still extend this lexical candidate.
func (state *BareWWWContinuation) AppendByte(b byte) bool {
	switch state.phase {
	case bareWWWTerminated, bareWWWDead:
		return false
	case bareWWWPath:
		state.appendPathByte(b)
	case bareWWWDomain:
		state.appendDomainByte(b)
	default:
		state.phase = bareWWWDead
		state.ownerPresent = false
	}
	return state.phase != bareWWWTerminated && state.phase != bareWWWDead
}

// Owner reports the currently materialized bare-autolink owner end offset,
// measured from the beginning of "www.", when the candidate currently owns one.
func (state BareWWWContinuation) Owner() (int, bool) {
	if !state.ownerPresent {
		return 0, false
	}
	return state.ownerEnd, true
}

func (state *BareWWWContinuation) appendDomainByte(b byte) {
	if bareWWWTerminatorByte(b) {
		state.ownerPresent = state.domain.ownerPresent
		state.ownerEnd = state.domain.ownerEnd
		state.phase = bareWWWTerminated
		return
	}
	if bareWWWDomainByte(b) {
		state.domain.append(b)
		state.ownerPresent = state.domain.ownerPresent
		state.ownerEnd = state.domain.ownerEnd
		if state.domain.dead {
			state.phase = bareWWWDead
		}
		return
	}
	if state.domain.trailingDots != 0 {
		state.ownerPresent = state.domain.ownerPresent
		state.ownerEnd = state.domain.ownerEnd
		state.phase = bareWWWTerminated
		return
	}
	if !state.domain.ownerPresent || state.domain.dead {
		state.ownerPresent = false
		state.phase = bareWWWDead
		return
	}
	state.path = bareWWWPathState{rawEnd: state.domain.rawEnd, ownerEnd: state.domain.rawEnd}
	state.path.append(b)
	state.phase = bareWWWPath
	state.ownerPresent = true
	state.ownerEnd = state.path.ownerEnd
}

func (state *BareWWWContinuation) appendPathByte(b byte) {
	state.path.append(b)
	state.ownerPresent = true
	state.ownerEnd = state.path.ownerEnd
	if state.path.terminated {
		state.phase = bareWWWTerminated
	}
}

func (state *bareWWWDomainState) append(b byte) {
	if state.dead {
		state.rawEnd++
		state.ownerPresent = false
		return
	}
	state.rawEnd++
	if b == '.' {
		if state.trailingDots < 2 {
			state.trailingDots++
		}
		return
	}
	if state.trailingDots >= 2 {
		state.dead = true
		state.ownerPresent = false
		return
	}
	if !state.finishPendingLabel() {
		return
	}
	if b == '_' {
		state.currentHasUnderscore = true
	}
	state.ownerPresent = state.labelCount >= 2 &&
		!state.currentHasUnderscore && !state.previousHasUnderscore
	if state.ownerPresent {
		state.ownerEnd = state.rawEnd
	}
}

func (state *bareWWWDomainState) finishPendingLabel() bool {
	if state.trailingDots == 0 {
		if state.labelCount == 0 {
			state.labelCount = 1
		}
		return true
	}
	if state.labelCount == 0 {
		state.dead = true
		state.ownerPresent = false
		return false
	}
	state.previousHasUnderscore = state.currentHasUnderscore
	state.currentHasUnderscore = false
	state.trailingDots = 0
	state.labelCount++
	return true
}

func (state *bareWWWPathState) append(b byte) {
	if state.terminated {
		return
	}
	if bareWWWTerminatorByte(b) {
		state.terminated = true
		return
	}

	oldPunctuation := state.punctuationLen
	oldClose := state.closeLen
	oldEntityPrefix := state.entityPrefix
	state.rawEnd++

	switch {
	case bareWWWTrailingPunctuation(b):
		state.punctuationLen++
		state.entityPrefix = 0
	case b == ')':
		state.parenBalance--
		state.punctuationLen = 0
		state.entityPrefix = 0
		if oldPunctuation != 0 {
			state.closeLen = 1
			state.entityComplete = 0
		} else {
			state.closeLen = oldClose + 1
		}
	default:
		state.appendOrdinaryPathByte(b, oldPunctuation, oldClose, oldEntityPrefix)
	}
	state.refreshOwnerEnd()
}

func (state *bareWWWPathState) appendOrdinaryPathByte(b byte, oldPunctuation, oldClose, oldEntityPrefix int) {
	if b == '(' {
		state.parenBalance++
	}
	state.punctuationLen = 0
	state.closeLen = 0
	state.entityComplete = 0
	switch {
	case b == '&':
		state.entityPrefix = 1
	case bareWWWAlphaNumeric(b) && oldPunctuation == 0 && oldClose == 0 && oldEntityPrefix != 0:
		state.entityPrefix = oldEntityPrefix + 1
	case b == ';' && oldPunctuation == 0 && oldClose == 0 && oldEntityPrefix >= 2:
		state.entityPrefix = 0
		state.entityComplete = oldEntityPrefix + 1
	default:
		state.entityPrefix = 0
	}
}

func (state *bareWWWPathState) refreshOwnerEnd() {
	end := state.rawEnd - state.punctuationLen
	closeTrim := 0
	if state.parenBalance < 0 {
		closeTrim = -state.parenBalance
		if closeTrim > state.closeLen {
			closeTrim = state.closeLen
		}
	}
	end -= closeTrim
	if closeTrim == state.closeLen && state.entityComplete != 0 {
		end -= state.entityComplete
	}
	state.ownerEnd = end
}

func bareWWWDomainByte(b byte) bool {
	return bareWWWAlphaNumeric(b) || b == '_' || b == '-' || b == '.'
}

func bareWWWTerminatorByte(b byte) bool {
	return b == '<' || b == ' ' || b == '\t'
}

func bareWWWTrailingPunctuation(b byte) bool {
	switch b {
	case '?', '!', '.', ',', ':', '*', '_', '~':
		return true
	default:
		return false
	}
}

func bareWWWAlphaNumeric(b byte) bool {
	return b >= 'a' && b <= 'z' ||
		b >= 'A' && b <= 'Z' ||
		b >= '0' && b <= '9'
}
