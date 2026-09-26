package native

import (
	"bytes"
	"testing"
)

func TestExtendedDomainValidationMatchesLabelRules(t *testing.T) {
	// Enumerate short labels, separators, invalid bytes, and underscore positions.
	alphabet := []byte{'a', '9', '-', '_', '.', '/', 0xff}
	var visit func([]byte)
	visit = func(domain []byte) {
		if got, want := validExtendedDomain(domain), labelBasedExtendedDomain(domain); got != want {
			t.Fatalf("domain=%q: valid=%v want=%v", domain, got, want)
		}
		if len(domain) < 6 {
			for _, value := range alphabet {
				visit(append(domain, value))
			}
		}
	}
	visit(nil)
}

func labelBasedExtendedDomain(domain []byte) bool {
	labels := bytes.Split(domain, []byte{'.'})
	if len(labels) < 2 {
		return false
	}
	for index, label := range labels {
		if len(label) == 0 {
			return false
		}
		for _, value := range label {
			if !(value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z' || value >= '0' && value <= '9' || value == '-' || value == '_') {
				return false
			}
			if value == '_' && index >= len(labels)-2 {
				return false
			}
		}
	}
	return true
}
