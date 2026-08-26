package source

import "bytes"

// FrontMatterFormat identifies a Marksplice-recognized document metadata envelope.
type FrontMatterFormat uint8

const (
	FrontMatterUnknown FrontMatterFormat = iota
	FrontMatterYAML
	FrontMatterTOML
)

// FrontMatterValueStyle identifies the lexical wrapper of a simple metadata scalar.
type FrontMatterValueStyle uint8

const (
	FrontMatterValueUnknown FrontMatterValueStyle = iota
	FrontMatterValuePlain
	FrontMatterValueSingleQuoted
	FrontMatterValueDoubleQuoted
)

// FrontMatterFieldMapping binds one conservatively recognized scalar field to exact source bytes.
type FrontMatterFieldMapping struct {
	Format     FrontMatterFormat
	Range      Range
	KeyRange   Range
	ValueRange Range
	Key        string
	Style      FrontMatterValueStyle
	Quote      byte
}

// FrontMatterMapping describes one leading YAML/TOML metadata envelope and its safely targetable fields.
type FrontMatterMapping struct {
	Format       FrontMatterFormat
	Range        Range
	OpeningRange Range
	ClosingRange Range
	Fields       []FrontMatterFieldMapping
}

// MapLeadingFrontMatter recognizes a closed metadata envelope only when it begins at byte zero.
// The Markdown body remains GFM; this scanner is a Marksplice-owned document-envelope layer.
func MapLeadingFrontMatter(input []byte) (FrontMatterMapping, bool) {
	format, delimiter, openingEnd, bodyStart, ok := scanFrontMatterOpening(input)
	if !ok {
		return FrontMatterMapping{}, false
	}
	closingStart, closingEnd, ok := findFrontMatterClosing(input, bodyStart, delimiter)
	if !ok {
		return FrontMatterMapping{}, false
	}
	fields, recognized := collectFrontMatterFields(input, bodyStart, closingStart, format)
	if !recognized {
		return FrontMatterMapping{}, false
	}
	return FrontMatterMapping{
		Format:       format,
		Range:        Range{Start: 0, End: closingEnd},
		OpeningRange: Range{Start: 0, End: openingEnd},
		ClosingRange: Range{Start: closingStart, End: closingEnd},
		Fields:       fields,
	}, true
}

func scanFrontMatterOpening(input []byte) (FrontMatterFormat, []byte, int, int, bool) {
	if len(input) < 7 {
		return FrontMatterUnknown, nil, 0, 0, false
	}
	openingEnd := physicalLineEnd(input, 0)
	var format FrontMatterFormat
	var delimiter []byte
	switch {
	case bytes.Equal(input[:openingEnd], []byte("---")):
		format = FrontMatterYAML
		delimiter = []byte("---")
	case bytes.Equal(input[:openingEnd], []byte("+++")):
		format = FrontMatterTOML
		delimiter = []byte("+++")
	default:
		return FrontMatterUnknown, nil, 0, 0, false
	}
	bodyStart, ok := nextPhysicalLineStart(input, openingEnd)
	if !ok {
		return FrontMatterUnknown, nil, 0, 0, false
	}
	return format, delimiter, openingEnd, bodyStart, true
}

func findFrontMatterClosing(input []byte, bodyStart int, delimiter []byte) (int, int, bool) {
	for lineStart := bodyStart; lineStart < len(input); {
		lineEnd := physicalLineEnd(input, lineStart)
		if bytes.Equal(input[lineStart:lineEnd], delimiter) {
			return lineStart, lineEnd, true
		}
		next, ok := nextPhysicalLineStart(input, lineEnd)
		if !ok {
			break
		}
		lineStart = next
	}
	return 0, 0, false
}

func collectFrontMatterFields(input []byte, bodyStart, closingStart int, format FrontMatterFormat) ([]FrontMatterFieldMapping, bool) {
	candidates := make([]FrontMatterFieldMapping, 0)
	recognized := bodyStart == closingStart
	tomlTable := false
	for lineStart := bodyStart; lineStart < closingStart; {
		lineEnd := physicalLineEnd(input, lineStart)
		line := input[lineStart:lineEnd]
		if format == FrontMatterTOML && tomlTableHeaderCandidate(line) {
			recognized = true
			tomlTable = true
		} else {
			if _, _, ok := scanFrontMatterKeySeparator(line, format); ok {
				recognized = true
			}
			if !tomlTable {
				if field, ok := mapFrontMatterField(input, lineStart, lineEnd, format); ok {
					candidates = append(candidates, field)
				}
			}
		}
		next, ok := nextPhysicalLineStart(input, lineEnd)
		if !ok || next >= closingStart {
			break
		}
		lineStart = next
	}
	return uniqueFrontMatterFields(candidates), recognized
}

func uniqueFrontMatterFields(candidates []FrontMatterFieldMapping) []FrontMatterFieldMapping {
	counts := make(map[string]int, len(candidates))
	for _, field := range candidates {
		counts[field.Key]++
	}
	fields := make([]FrontMatterFieldMapping, 0, len(candidates))
	for _, field := range candidates {
		if counts[field.Key] == 1 {
			fields = append(fields, field)
		}
	}
	return fields
}

func tomlTableHeaderCandidate(line []byte) bool {
	start := skipHorizontalSpace(line, 0, len(line))
	end := len(line)
	for end > start && isHorizontalSpace(line[end-1]) {
		end--
	}
	if start >= end || line[start] != '[' {
		return false
	}
	if start+1 < end && line[start+1] == '[' {
		return end-start >= 5 && line[end-2] == ']' && line[end-1] == ']' && simpleTOMLTableKey(line[start+2:end-2])
	}
	return end-start >= 3 && line[end-1] == ']' && simpleTOMLTableKey(line[start+1:end-1])
}

func simpleTOMLTableKey(value []byte) bool {
	if len(value) == 0 {
		return false
	}
	for _, b := range value {
		if !isSimpleMetadataKeyByte(b) {
			return false
		}
	}
	return true
}

func mapFrontMatterField(input []byte, lineStart, lineEnd int, format FrontMatterFormat) (FrontMatterFieldMapping, bool) {
	if lineStart < 0 || lineEnd < lineStart || lineEnd > len(input) {
		return FrontMatterFieldMapping{}, false
	}
	line := input[lineStart:lineEnd]
	keyRange, valueStart, ok := scanFrontMatterKeyValueStart(line, format)
	if !ok {
		return FrontMatterFieldMapping{}, false
	}
	value, style, quote, ok := scanSimpleMetadataValue(line, valueStart, format)
	if !ok || value.Start == value.End {
		return FrontMatterFieldMapping{}, false
	}

	return FrontMatterFieldMapping{
		Format:     format,
		Range:      Range{Start: lineStart, End: lineEnd},
		KeyRange:   Range{Start: lineStart + keyRange.Start, End: lineStart + keyRange.End},
		ValueRange: Range{Start: lineStart + value.Start, End: lineStart + value.End},
		Key:        string(line[keyRange.Start:keyRange.End]),
		Style:      style,
		Quote:      quote,
	}, true
}

func scanFrontMatterKeyValueStart(line []byte, format FrontMatterFormat) (Range, int, bool) {
	keyRange, pos, ok := scanFrontMatterKeySeparator(line, format)
	if !ok {
		return Range{}, 0, false
	}
	pos = skipHorizontalSpace(line, pos, len(line))
	if pos >= len(line) {
		return Range{}, 0, false
	}
	return keyRange, pos, true
}

func scanFrontMatterKeySeparator(line []byte, format FrontMatterFormat) (Range, int, bool) {
	if len(line) == 0 {
		return Range{}, 0, false
	}
	pos := 0
	if format == FrontMatterTOML {
		pos = skipHorizontalSpace(line, pos, len(line))
	} else if isHorizontalSpace(line[0]) {
		return Range{}, 0, false
	}
	keyStart := pos
	for pos < len(line) && isSimpleMetadataKeyByte(line[pos]) {
		pos++
	}
	if pos == keyStart {
		return Range{}, 0, false
	}
	keyRange := Range{Start: keyStart, End: pos}
	if format == FrontMatterTOML {
		pos = skipHorizontalSpace(line, pos, len(line))
		if pos >= len(line) || line[pos] != '=' {
			return Range{}, 0, false
		}
	} else if pos >= len(line) || line[pos] != ':' {
		return Range{}, 0, false
	}
	return keyRange, pos + 1, true
}

func scanSimpleMetadataValue(line []byte, start int, format FrontMatterFormat) (Range, FrontMatterValueStyle, byte, bool) {
	if start < 0 || start >= len(line) {
		return Range{}, FrontMatterValueUnknown, 0, false
	}
	if line[start] == '\'' || line[start] == '"' {
		return scanQuotedMetadataScalar(line, start, format)
	}
	if format == FrontMatterTOML {
		return scanTOMLBareScalar(line, start)
	}
	return scanYAMLPlainScalar(line, start)
}

func scanQuotedMetadataScalar(line []byte, start int, format FrontMatterFormat) (Range, FrontMatterValueStyle, byte, bool) {
	quote := line[start]
	end, ok := scanMetadataQuotedValue(line, start, quote, format)
	if !ok || !onlyHorizontalSpaceOrComment(line, end+1) {
		return Range{}, FrontMatterValueUnknown, 0, false
	}
	style := FrontMatterValueSingleQuoted
	if quote == '"' {
		style = FrontMatterValueDoubleQuoted
	}
	return Range{Start: start + 1, End: end}, style, quote, end > start+1
}

func scanTOMLBareScalar(line []byte, start int) (Range, FrontMatterValueStyle, byte, bool) {
	end := start
	for end < len(line) && !isHorizontalSpace(line[end]) && line[end] != '#' {
		end++
	}
	if end == start || !safeTOMLBareScalar(line[start:end]) || !onlyHorizontalSpaceOrComment(line, end) {
		return Range{}, FrontMatterValueUnknown, 0, false
	}
	return Range{Start: start, End: end}, FrontMatterValuePlain, 0, true
}

func scanYAMLPlainScalar(line []byte, start int) (Range, FrontMatterValueStyle, byte, bool) {
	end := len(line)
	for i := start; i < len(line); i++ {
		if line[i] == '#' && i > start && isHorizontalSpace(line[i-1]) {
			end = i
			break
		}
	}
	for end > start && isHorizontalSpace(line[end-1]) {
		end--
	}
	if end == start || !safeYAMLPlainScalar(line[start:end]) {
		return Range{}, FrontMatterValueUnknown, 0, false
	}
	return Range{Start: start, End: end}, FrontMatterValuePlain, 0, true
}

func onlyHorizontalSpaceOrComment(line []byte, start int) bool {
	for start < len(line) && isHorizontalSpace(line[start]) {
		start++
	}
	return start == len(line) || line[start] == '#'
}

func scanMetadataQuotedValue(line []byte, start int, quote byte, format FrontMatterFormat) (int, bool) {
	for i := start + 1; i < len(line); i++ {
		if quote == '"' && line[i] == '\\' && i+1 < len(line) {
			i++
			continue
		}
		if format == FrontMatterYAML && quote == '\'' && line[i] == '\'' && i+1 < len(line) && line[i+1] == '\'' {
			i++
			continue
		}
		if line[i] == quote {
			return i, true
		}
	}
	return 0, false
}

func safeYAMLPlainScalar(value []byte) bool {
	if !safeYAMLPlainScalarStart(value) {
		return false
	}
	for index, b := range value {
		if !safeYAMLPlainScalarByte(value, index, b) {
			return false
		}
	}
	return true
}

func safeYAMLPlainScalarStart(value []byte) bool {
	if len(value) == 0 {
		return false
	}
	switch value[0] {
	case ',', '[', ']', '{', '}', '#', '&', '*', '!', '|', '>', '\'', '"', '%', '@', '`':
		return false
	case '-', '?', ':':
		return len(value) > 1 && !isHorizontalSpace(value[1])
	default:
		return true
	}
}

func safeYAMLPlainScalarByte(value []byte, index int, b byte) bool {
	if b < 0x20 && b != '\t' || b == '[' || b == ']' || b == '{' || b == '}' {
		return false
	}
	return b != ':' || index+1 >= len(value) || !isHorizontalSpace(value[index+1])
}

func safeTOMLBareScalar(value []byte) bool {
	return bytes.Equal(value, []byte("true")) || bytes.Equal(value, []byte("false"))
}

func isSimpleMetadataKeyByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '-' || b == '.'
}
