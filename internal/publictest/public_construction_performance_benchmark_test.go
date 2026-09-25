package publictest

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/zoster81/marksplice"
)

// Include both per-block proof and final-document proof in construction cost.
func BenchmarkBuilderScaling(b *testing.B) {
	for _, count := range []int{16, 64, 256} {
		b.Run(fmt.Sprintf("%dBlocks", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				builder := marksplice.NewDocumentBuilder()
				for block := 0; block < count; block++ {
					if err := builder.AppendParagraph("Text with *emphasis*, **strong**, and [a link](target.md)."); err != nil {
						b.Fatal(err)
					}
				}
				output, err := builder.Markdown()
				if err != nil || len(output) == 0 {
					b.Fatalf("construction failed: %v", err)
				}
			}
		})
	}
}

// Unlike the no-op planning benchmark, this changes content and applies the
// snapshot-bound patch. Parsing the original document is outside the timer.
func BenchmarkParagraphEditAndApplyScaling(b *testing.B) {
	for _, sizeKiB := range []int{16, 64, 256} {
		source := realisticSource(sizeKiB << 10)
		document, err := marksplice.Parse(source)
		if err != nil {
			b.Fatal(err)
		}
		node := benchmarkNodeOfKind(b, document, marksplice.KindParagraph)
		b.Run(fmt.Sprintf("%dKiB", sizeKiB), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for i := 0; i < b.N; i++ {
				change, err := document.PrepareReplaceParagraph(node.ID(), []byte("A replacement with **different** content."))
				if err != nil {
					b.Fatal(err)
				}
				output, err := change.Apply(source)
				if err != nil || bytes.Equal(output, source) {
					b.Fatalf("edit was not applied: %v", err)
				}
			}
		})
	}
}
