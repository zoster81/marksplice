package native_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

var m114ObservationsSink parser.DocumentObservations

func BenchmarkM114NativeBackendScaling(b *testing.B) {
	backend := native.New()
	families := []struct {
		name string
		unit string
	}{
		{name: "DirectLinks", unit: "[label](<target>) "},
		{name: "Delimiters", unit: "*a* **b** ~~c~~ `d` "},
	}
	for _, family := range families {
		for _, count := range []int{256, 1024, 4096} {
			source := []byte(strings.Repeat(family.unit, count) + "\n")
			b.Run(fmt.Sprintf("%s/%d", family.name, count), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(source)))
				for iteration := 0; iteration < b.N; iteration++ {
					observations, err := backend.ParseDocument(source)
					if err != nil {
						b.Fatal(err)
					}
					m114ObservationsSink = observations
				}
			})
		}
	}
	for _, depth := range []int{256, 1024, 4096} {
		source := []byte(strings.Repeat("> ", depth) + "payload\n")
		b.Run(fmt.Sprintf("DeepBlockquote/%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for iteration := 0; iteration < b.N; iteration++ {
				observations, err := backend.ParseDocument(source)
				if err != nil {
					b.Fatal(err)
				}
				m114ObservationsSink = observations
			}
		})
	}
}

func BenchmarkM114NativeConstructionProofScaling(b *testing.B) {
	backend := native.New()
	for _, count := range []int{256, 1024, 4096} {
		source, expected := m114SiblingEmphasisProof(count)
		b.Run(fmt.Sprintf("SiblingEmphasis/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for iteration := 0; iteration < b.N; iteration++ {
				if err := backend.ValidateConstructionInlineHierarchy(source, expected, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func m114SiblingEmphasisProof(count int) ([]byte, []parser.ConstructionInlineExpectation) {
	var source strings.Builder
	source.Grow(count * 4)
	expected := make([]parser.ConstructionInlineExpectation, count)
	for index := range count {
		start := source.Len()
		source.WriteString("*x*")
		if index+1 < count {
			source.WriteByte(' ')
		}
		expected[index] = parser.ConstructionInlineExpectation{
			Kind:            parser.KindEmphasis,
			SyntaxRange:     parser.Range{Start: start, End: start + 3},
			ContentRange:    parser.Range{Start: start + 1, End: start + 2},
			Marker:          '*',
			DelimiterLength: 1,
			Parent:          -1,
		}
	}
	return []byte(source.String()), expected
}
