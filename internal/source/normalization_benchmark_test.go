package source

import (
	"bytes"
	"testing"
)

var normalizedBytesSink []byte
var normalizedStringSink string

func BenchmarkCommonMarkNormalization(b *testing.B) {
	for _, unit := range []struct{ name, text string }{
		{"Unchanged", "ordinary Markdown text\n"},
		{"NUL", "ordinary\x00Markdown text\n"},
	} {
		input := bytes.Repeat([]byte(unit.text), 3072)
		b.Run(unit.name+"/Bytes", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))
			for i := 0; i < b.N; i++ {
				normalizedBytesSink = NormalizeCommonMarkInput(input)
			}
		})
		text := string(input)
		b.Run(unit.name+"/String", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(text)))
			for i := 0; i < b.N; i++ {
				normalizedStringSink = NormalizeCommonMarkString(text)
			}
		})
	}
}

func TestCommonMarkNormalizationUnchangedInput(t *testing.T) {
	if NormalizeCommonMarkInput(nil) != nil {
		t.Fatal("nil input changed")
	}
	input := []byte("ordinary text\xff\r\n")
	got := NormalizeCommonMarkInput(input)
	if len(got) != len(input) || &got[0] != &input[0] {
		t.Fatal("unchanged input must be returned without a copy")
	}
	if NormalizeCommonMarkString(string(input)) != string(input) {
		t.Fatal("normalization must preserve non-NUL bytes")
	}
}
