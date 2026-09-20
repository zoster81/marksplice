package parser

import (
	"unicode"
	"unicode/utf8"
)

// DelimiterFlanking reports whether one delimiter run can open and close
// according to the CommonMark/GFM flanking rules used by the Native parser.
func DelimiterFlanking(source []byte, segment Range, start, end int, marker byte) (bool, bool) {
	if !segment.Valid(len(source)) || start < segment.Start || start >= end || end > segment.End {
		return false, false
	}
	beforeWhitespace, beforePunctuation := delimiterPrecedingClass(source, segment, start)
	afterWhitespace, afterPunctuation := delimiterFollowingClass(source, start, end, segment.End)
	return delimiterFlankingFromClasses(beforeWhitespace, beforePunctuation, afterWhitespace, afterPunctuation, marker)
}

// DelimiterRunsHaveModuloThreeConflict reports whether CommonMark's
// emphasis odd-match rule prevents an otherwise flanking-compatible pair.
func DelimiterRunsHaveModuloThreeConflict(openLength int, openCanClose bool, closeLength int, closeCanOpen bool) bool {
	if openLength <= 0 || closeLength <= 0 {
		return false
	}
	return (openCanClose || closeCanOpen) &&
		(openLength+closeLength)%3 == 0 && closeLength%3 != 0
}

func delimiterFlankingFromClasses(beforeWhitespace, beforePunctuation, afterWhitespace, afterPunctuation bool, marker byte) (bool, bool) {
	leftFlanking := !afterWhitespace && (!afterPunctuation || beforeWhitespace || beforePunctuation)
	rightFlanking := !beforeWhitespace && (!beforePunctuation || afterWhitespace || afterPunctuation)
	if marker == '_' {
		return leftFlanking && (!rightFlanking || beforePunctuation), rightFlanking && (!leftFlanking || afterPunctuation)
	}
	return leftFlanking, rightFlanking
}

func delimiterPrecedingClass(source []byte, segment Range, position int) (bool, bool) {
	if position <= segment.Start {
		return true, false
	}
	index := position - 1
	for index >= segment.Start && !utf8.RuneStart(source[index]) {
		index--
	}
	if index < segment.Start {
		return true, false
	}
	rune_, _ := utf8.DecodeRune(source[index:position])
	return delimiterRuneClass(rune_)
}

func delimiterFollowingClass(source []byte, runStart, position, segmentEnd int) (bool, bool) {
	if position >= segmentEnd {
		return true, false
	}
	index := position
	for index > runStart && !utf8.RuneStart(source[index]) {
		index--
	}
	rune_, _ := utf8.DecodeRune(source[index:segmentEnd])
	return delimiterRuneClass(rune_)
}

func delimiterRuneClass(rune_ rune) (bool, bool) {
	return unicode.IsSpace(rune_), unicode.IsPunct(rune_) || unicode.IsSymbol(rune_)
}
