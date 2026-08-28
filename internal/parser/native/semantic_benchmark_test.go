package native_test

import (
	"fmt"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

var m118SemanticEventSink int

func BenchmarkM118SemanticWalkRealisticScaling(b *testing.B) {
	backend := native.New()
	for _, sizeKiB := range []int{64, 256, 1024} {
		source := m114RealisticSource(sizeKiB << 10)
		b.Run(fmt.Sprintf("%dKiB", sizeKiB), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for iteration := 0; iteration < b.N; iteration++ {
				count := 0
				err := backend.WalkSemantic(source, func(parser.SemanticEvent) error {
					count++
					return nil
				})
				if err != nil {
					b.Fatal(err)
				}
				m118SemanticEventSink = count
			}
		})
	}
}
