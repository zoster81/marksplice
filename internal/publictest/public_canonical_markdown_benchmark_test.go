package publictest

import (
	"fmt"
	"io"
	"testing"

	"github.com/zoster81/marksplice"
	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

var (
	m123CanonicalMarkdownBytesSink []byte
	m123SemanticEventSink          int
)

func BenchmarkCanonicalMarkdownRealisticScaling(b *testing.B) {
	for _, sizeKiB := range []int{64, 256, 1024} {
		source := realisticSource(sizeKiB << 10)
		document, err := marksplice.Parse(source)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("%dKiB", sizeKiB), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for iteration := 0; iteration < b.N; iteration++ {
				if err := document.RenderCanonicalMarkdown(io.Discard); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkSemanticWalkRealisticScaling(b *testing.B) {
	backend := native.New()
	for _, sizeKiB := range []int{64, 256, 1024} {
		source := realisticSource(sizeKiB << 10)
		b.Run(fmt.Sprintf("%dKiB", sizeKiB), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for iteration := 0; iteration < b.N; iteration++ {
				count := 0
				if err := backend.WalkSemantic(source, func(parser.SemanticEvent) error {
					count++
					return nil
				}); err != nil {
					b.Fatal(err)
				}
				m123SemanticEventSink = count
			}
		})
	}
}

func BenchmarkCanonicalMarkdown256KiB(b *testing.B) {
	source := realisticSource(256 << 10)
	document, err := marksplice.Parse(source)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("StreamingDiscard", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(source)))
		for iteration := 0; iteration < b.N; iteration++ {
			if err := document.RenderCanonicalMarkdown(io.Discard); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("BufferedBytes", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(source)))
		for iteration := 0; iteration < b.N; iteration++ {
			output, err := document.CanonicalMarkdown()
			if err != nil {
				b.Fatal(err)
			}
			m123CanonicalMarkdownBytesSink = output
		}
	})
}
