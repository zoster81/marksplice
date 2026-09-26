package native

import "bytes"

// physicalLine is a byte-preserving view over one physical source line.
// columnPhase is the logical visual column modulo four before virtualIndent
// at start. Only tab-stop alignment needs this value, not the absolute column.
// virtualIndent records columns left behind when a tab is only partially
// consumed by a container indentation without inventing byte offsets.
type physicalLine struct {
	physicalStart int
	start         int
	end           int
	next          int
	columnPhase   uint8
	virtualIndent uint8
}

func physicalLines(source []byte) []physicalLine {
	if len(source) == 0 {
		return nil
	}
	lines := make([]physicalLine, 0, 1+bytes.Count(source, []byte{'\n'}))
	for start := 0; start < len(source); {
		end := start
		for end < len(source) && source[end] != '\r' && source[end] != '\n' {
			end++
		}
		next := end
		if next < len(source) && source[next] == '\r' {
			next++
			if next < len(source) && source[next] == '\n' {
				next++
			}
		} else if next < len(source) {
			next++
		}
		lines = append(lines, physicalLine{physicalStart: start, start: start, end: end, next: next})
		start = next
	}
	return lines
}

func blankLine(source []byte, line physicalLine) bool {
	for _, value := range source[line.start:line.end] {
		if value != ' ' && value != '\t' {
			return false
		}
	}
	return true
}

func ordinaryIndent(source []byte, line physicalLine) (int, bool) {
	bytes_, columns := leadingIndent(source, line)
	return bytes_, columns <= 3
}

func sourceIndentHasTab(source []byte, line physicalLine, bytes_ int) bool {
	return bytes_ > 0 && bytes.IndexByte(source[line.start:line.start+bytes_], '\t') >= 0
}

func leadingIndent(source []byte, line physicalLine) (bytes_, columns int) {
	position := line.start
	columns = int(line.virtualIndent)
	for position < line.end {
		switch source[position] {
		case ' ':
			columns++
		case '\t':
			columns += 4 - (int(line.columnPhase)+columns)%4
		default:
			return position - line.start, columns
		}
		position++
	}
	return position - line.start, columns
}

func advancePhysicalLineStart(source []byte, line physicalLine, start, virtualIndent int) physicalLine {
	column := int(line.columnPhase) + int(line.virtualIndent)
	for position := line.start; position < start; position++ {
		if source[position] == '\t' {
			column += 4 - column%4
		} else {
			column++
		}
	}
	line.start = start
	line.virtualIndent = uint8(virtualIndent)
	line.columnPhase = uint8(column-virtualIndent) & 3
	return line
}

func indentedCodeLine(source []byte, line physicalLine) bool {
	if blankLine(source, line) {
		return false
	}
	_, columns := leadingIndent(source, line)
	return columns >= 4
}

func stripIndentColumns(source []byte, line physicalLine, columns int) physicalLine {
	if columns <= 0 {
		return line
	}
	remaining := columns
	if int(line.virtualIndent) >= remaining {
		line.columnPhase = (line.columnPhase + uint8(remaining)) & 3
		line.virtualIndent -= uint8(remaining)
		return line
	}
	line.columnPhase = (line.columnPhase + line.virtualIndent) & 3
	remaining -= int(line.virtualIndent)
	line.virtualIndent = 0
	position := line.start
	for position < line.end && remaining > 0 {
		width := 0
		switch source[position] {
		case ' ':
			width = 1
		case '\t':
			width = 4 - int(line.columnPhase)
		default:
			line.start = position
			return line
		}
		position++
		if width > remaining {
			line.columnPhase = (line.columnPhase + uint8(remaining)) & 3
			line.start = position
			line.virtualIndent = uint8(width - remaining)
			return line
		}
		line.columnPhase = (line.columnPhase + uint8(width)) & 3
		remaining -= width
	}
	line.start = position
	return line
}
