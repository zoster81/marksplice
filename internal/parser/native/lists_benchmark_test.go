package native

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

var benchmarkListObservations parser.DocumentObservations
var benchmarkListEventCount int

func BenchmarkListParsing(b *testing.B) {
	for _, shape := range []struct{ name, item string }{
		{"Tight", "- item\n"},
		{"Loose", "- item\n\n"},
		{"Separate", "- item\n\nparagraph\n\n"},
		{"Nested", "- parent\n  - first\n  - second\n"},
		{"StoppedContinuation", "- ```\n  code\n  ```\nparagraph\n\n"},
	} {
		for _, count := range []int{1, 16, 256, 1024} {
			source := bytes.Repeat([]byte(shape.item), count)
			b.Run(fmt.Sprintf("%s/%dItems", shape.name, count), func(b *testing.B) {
				backend := New()
				b.Run("Observations", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(source)))
					for i := 0; i < b.N; i++ {
						observed, err := backend.ParseDocument(source)
						if err != nil {
							b.Fatal(err)
						}
						benchmarkListObservations = observed
					}
				})
				b.Run("Semantic", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(source)))
					for i := 0; i < b.N; i++ {
						count := 0
						err := backend.WalkSemantic(source, func(parser.SemanticEvent) error {
							count++
							return nil
						})
						if err != nil {
							b.Fatal(err)
						}
						benchmarkListEventCount = count
					}
				})
			})
		}
	}
}
