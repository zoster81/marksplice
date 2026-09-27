package native

import (
	"strings"
	"testing"
)

func TestASCIILetterEveryByte(t *testing.T) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	for value := 0; value < 256; value++ {
		want := strings.IndexByte(alphabet, byte(value)) >= 0
		if got := asciiLetter(byte(value)); got != want {
			t.Fatalf("asciiLetter(%#02x) = %v, want %v", value, got, want)
		}
	}
}
