package publictest

import (
	"io"
	"testing"

	"github.com/zoster81/marksplice"
)

var hTMLSourceMapSink []marksplice.HTMLSourceMapEntry

func BenchmarkHTMLSourceMap256KiB(b *testing.B) {
	body := realisticSource(256 << 10)
	source := append([]byte("---\ntitle: \"Benchmark document\"\ndescription: Representative source mapping\nauthor: Marksplice\nlang: en-US\n---\n"), body...)
	document, err := marksplice.Parse(source)
	if err != nil {
		b.Fatal(err)
	}
	fragmentOptions := marksplice.DefaultHTMLRenderOptions()
	documentOptions := marksplice.DefaultHTMLDocumentOptions()

	b.Run("FragmentMappingOff", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(source)))
		for iteration := 0; iteration < b.N; iteration++ {
			if err := document.RenderHTML(io.Discard, fragmentOptions); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("FragmentMappingOn", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(source)))
		mappingCount := 0
		for iteration := 0; iteration < b.N; iteration++ {
			mappings, err := document.RenderHTMLWithSourceMap(io.Discard, fragmentOptions)
			if err != nil {
				b.Fatal(err)
			}
			mappingCount = len(mappings)
			hTMLSourceMapSink = mappings
		}
		b.ReportMetric(float64(mappingCount), "maps/op")
	})
	b.Run("DocumentMappingOff", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(source)))
		for iteration := 0; iteration < b.N; iteration++ {
			if err := document.RenderHTMLDocument(io.Discard, documentOptions); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("DocumentMappingOn", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(source)))
		mappingCount := 0
		for iteration := 0; iteration < b.N; iteration++ {
			mappings, err := document.RenderHTMLDocumentWithSourceMap(io.Discard, documentOptions)
			if err != nil {
				b.Fatal(err)
			}
			mappingCount = len(mappings)
			hTMLSourceMapSink = mappings
		}
		b.ReportMetric(float64(mappingCount), "maps/op")
	})
}
