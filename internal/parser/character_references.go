package parser

import (
	"html"
	"strconv"
	"unicode/utf8"
)

// DecodeCommonMarkCharacterReference decodes one complete CommonMark 0.31.2
// entity or numeric character reference. It requires the terminating semicolon.
func DecodeCommonMarkCharacterReference(raw string) (string, bool) {
	if len(raw) < 4 || raw[0] != '&' || raw[len(raw)-1] != ';' {
		return "", false
	}
	if raw[1] == '#' {
		return decodeCommonMarkNumericReference(raw)
	}
	if !asciiEntityLetter(raw[1]) {
		return "", false
	}
	for index := 2; index < len(raw)-1; index++ {
		if !asciiEntityLetter(raw[index]) && !asciiEntityDigit(raw[index]) {
			return "", false
		}
	}

	switch raw {
	case "&nGt;":
		return "≫⃒", true
	case "&nLt;":
		return "≪⃒", true
	}
	decoded := html.UnescapeString(raw)
	if decoded == raw {
		return "", false
	}
	return decoded, true
}

func decodeCommonMarkNumericReference(raw string) (string, bool) {
	digits := raw[2 : len(raw)-1]
	base := 10
	maxDigits := 7
	if len(digits) != 0 && (digits[0] == 'x' || digits[0] == 'X') {
		base = 16
		maxDigits = 6
		digits = digits[1:]
	}
	if len(digits) == 0 || len(digits) > maxDigits {
		return "", false
	}
	for index := 0; index < len(digits); index++ {
		if base == 10 {
			if !asciiEntityDigit(digits[index]) {
				return "", false
			}
			continue
		}
		if !asciiEntityHexDigit(digits[index]) {
			return "", false
		}
	}
	value, err := strconv.ParseUint(digits, base, 32)
	if err != nil {
		return "", false
	}
	codepoint := rune(value)
	if value == 0 || value > utf8.MaxRune || value >= 0xD800 && value <= 0xDFFF {
		codepoint = utf8.RuneError
	}
	return string(codepoint), true
}

func asciiEntityLetter(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func asciiEntityDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func asciiEntityHexDigit(value byte) bool {
	return asciiEntityDigit(value) || value >= 'a' && value <= 'f' || value >= 'A' && value <= 'F'
}
