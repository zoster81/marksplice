package source

func containsLineBreak(input []byte) bool {
	for _, b := range input {
		if b == '\r' || b == '\n' {
			return true
		}
	}
	return false
}

func trimHorizontalSpaceRange(line []byte, range_ Range) Range {
	for range_.Start < range_.End && isHorizontalSpace(line[range_.Start]) {
		range_.Start++
	}
	for range_.End > range_.Start && isHorizontalSpace(line[range_.End-1]) {
		range_.End--
	}
	return range_
}

func allHorizontalSpace(input []byte) bool {
	for _, b := range input {
		if !isHorizontalSpace(b) {
			return false
		}
	}
	return true
}

func isHorizontalSpace(b byte) bool {
	return b == ' ' || b == '\t'
}

func skipHorizontalSpace(input []byte, pos, limit int) int {
	for pos < limit && isHorizontalSpace(input[pos]) {
		pos++
	}
	return pos
}

func previousPhysicalLineStart(input []byte, lineStart int) (int, bool) {
	if lineStart <= 0 || lineStart > len(input) {
		return 0, false
	}

	pos := lineStart - 1
	switch input[pos] {
	case '\n':
		pos--
		if pos >= 0 && input[pos] == '\r' {
			pos--
		}
	case '\r':
		pos--
	default:
		return 0, false
	}
	for pos >= 0 {
		if input[pos] == '\n' || input[pos] == '\r' {
			return pos + 1, true
		}
		pos--
	}
	return 0, true
}

func physicalLineStart(input []byte, pos int) int {
	if pos > len(input) {
		pos = len(input)
	}
	for pos > 0 {
		if input[pos-1] == '\n' || input[pos-1] == '\r' {
			break
		}
		pos--
	}
	return pos
}

func physicalLineEnd(input []byte, pos int) int {
	if pos < 0 {
		pos = 0
	}
	if pos > len(input) {
		pos = len(input)
	}
	for pos < len(input) && input[pos] != '\n' && input[pos] != '\r' {
		pos++
	}
	return pos
}

func nextPhysicalLineStart(input []byte, lineEnd int) (int, bool) {
	if lineEnd >= len(input) {
		return 0, false
	}
	switch input[lineEnd] {
	case '\r':
		lineEnd++
		if lineEnd < len(input) && input[lineEnd] == '\n' {
			lineEnd++
		}
	case '\n':
		lineEnd++
	default:
		return 0, false
	}
	if lineEnd > len(input) {
		return 0, false
	}
	return lineEnd, true
}
