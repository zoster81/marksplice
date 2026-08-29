package publictest

import (
	"io"
	"testing"

	"github.com/zoster81/marksplice"
	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

var (
	m120SemanticEventSink int
	m120HTMLBytesSink     []byte
)

func BenchmarkM120HTMLRendering256KiB(b *testing.B) {
	source := m108RealisticSource(256 << 10)
	options := marksplice.DefaultHTMLRenderOptions()

	b.Run("Parse", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(source)))
		for iteration := 0; iteration < b.N; iteration++ {
			document, err := marksplice.Parse(source)
			if err != nil {
				b.Fatal(err)
			}
			m108DocumentSink = document
		}
	})

	backend := native.New()
	b.Run("WalkSemantic", func(b *testing.B) {
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
			m120SemanticEventSink = count
		}
	})

	document, err := marksplice.Parse(source)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("RenderHTMLDiscard", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(source)))
		for iteration := 0; iteration < b.N; iteration++ {
			if err := document.RenderHTML(io.Discard, options); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("HTMLBytes", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(source)))
		for iteration := 0; iteration < b.N; iteration++ {
			output, err := document.HTML(options)
			if err != nil {
				b.Fatal(err)
			}
			m120HTMLBytesSink = output
		}
	})
}
