package publictest

import (
	"io"
	"testing"

	"github.com/zoster81/marksplice"
)

var m121HTMLDocumentBytesSink []byte

func BenchmarkM121StandaloneHTML256KiB(b *testing.B) {
	body := m108RealisticSource(256 << 10)
	source := append([]byte("---\ntitle: \"Benchmark document\"\ndescription: Representative standalone rendering\nauthor: Marksplice\nlang: en-US\n---\n"), body...)
	document, err := marksplice.Parse(source)
	if err != nil {
		b.Fatal(err)
	}
	options := marksplice.DefaultHTMLDocumentOptions()

	b.Run("RenderHTMLDocumentDiscard", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(source)))
		for iteration := 0; iteration < b.N; iteration++ {
			if err := document.RenderHTMLDocument(io.Discard, options); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("HTMLDocumentBytes", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(source)))
		for iteration := 0; iteration < b.N; iteration++ {
			output, err := document.HTMLDocument(options)
			if err != nil {
				b.Fatal(err)
			}
			m121HTMLDocumentBytesSink = output
		}
	})
}
