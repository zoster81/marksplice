package native

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkM113NativeInlineScaling(b *testing.B) {
	families := []struct {
		name string
		unit string
	}{
		{name: "DirectLinks", unit: "[label](https://example.com/path) "},
		{name: "Delimiters", unit: "*a* **b** ~~c~~ "},
	}
	for _, family := range families {
		for _, count := range []int{256, 1024, 4096} {
			source := []byte(strings.Repeat(family.unit, count) + "\n")
			b.Run(fmt.Sprintf("%s/%d", family.name, count), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(source)))
				for iteration := 0; iteration < b.N; iteration++ {
					if _, err := ParseInlineObservations(source); err != nil {
						b.Fatalf("ParseInlineObservations() error = %v", err)
					}
				}
			})
		}
	}
}
