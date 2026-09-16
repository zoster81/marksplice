package publictest

import (
	"testing"

	"github.com/zoster81/marksplice"
)

var r30ChangeSink marksplice.ChangeSet

func BenchmarkR30MutationPlanning(b *testing.B) {
	benchmarks := []struct {
		name   string
		source []byte
		setup  func(testing.TB, *marksplice.Document) func() (marksplice.ChangeSet, error)
	}{
		{
			name:   "FrontMatterValue",
			source: []byte("---\ntitle: \"short\"\n---\n\nbody\n"),
			setup: func(tb testing.TB, doc *marksplice.Document) func() (marksplice.ChangeSet, error) {
				node := r30BenchmarkNodeOfKind(tb, doc, marksplice.KindFrontMatterField)
				return func() (marksplice.ChangeSet, error) {
					return doc.PrepareReplaceFrontMatterValue(node.ID(), []byte("a substantially longer title"))
				}
			},
		},
		{
			name:   "BlockquoteContent",
			source: []byte("> alpha\n> beta\n\nafter\n"),
			setup: func(tb testing.TB, doc *marksplice.Document) func() (marksplice.ChangeSet, error) {
				node := r30BenchmarkNodeOfKind(tb, doc, marksplice.KindBlockquote)
				return func() (marksplice.ChangeSet, error) {
					return doc.PrepareReplaceBlockquoteContent(node.ID(), []byte("replaced alpha\nreplaced beta"))
				}
			},
		},
		{
			name:   "FencedInfo",
			source: []byte("```go\nfmt.Println(\"hi\")\n```\n"),
			setup: func(tb testing.TB, doc *marksplice.Document) func() (marksplice.ChangeSet, error) {
				blocks := doc.FencedBlocks()
				if len(blocks) != 1 {
					tb.Fatalf("fenced blocks = %d", len(blocks))
				}
				id := blocks[0].ID()
				return func() (marksplice.ChangeSet, error) {
					return doc.PrepareSetFencedBlockInfo(id, []byte("go linenos"))
				}
			},
		},
		{
			name:   "InlineLinkLabel",
			source: []byte("[short](dest \"title\")\n"),
			setup: func(tb testing.TB, doc *marksplice.Document) func() (marksplice.ChangeSet, error) {
				node := r30BenchmarkNodeOfKind(tb, doc, marksplice.KindInlineLink)
				return func() (marksplice.ChangeSet, error) {
					return doc.PrepareReplaceInlineLinkLabel(node.ID(), []byte("a longer label"))
				}
			},
		},
		{
			name:   "ImageTitle",
			source: []byte("![alt](dest  (old title))\r\n"),
			setup: func(tb testing.TB, doc *marksplice.Document) func() (marksplice.ChangeSet, error) {
				node := r30BenchmarkNodeOfKind(tb, doc, marksplice.KindImage)
				return func() (marksplice.ChangeSet, error) {
					return doc.PrepareReplaceImageTitle(node.ID(), []byte("a longer title"))
				}
			},
		},
	}

	for _, benchmark := range benchmarks {
		benchmark := benchmark
		b.Run(benchmark.name, func(b *testing.B) {
			doc, err := marksplice.Parse(benchmark.source)
			if err != nil {
				b.Fatal(err)
			}
			prepare := benchmark.setup(b, doc)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				change, err := prepare()
				if err != nil {
					b.Fatal(err)
				}
				r30ChangeSink = change
			}
		})
	}
}

func r30BenchmarkNodeOfKind(tb testing.TB, doc *marksplice.Document, kind marksplice.Kind) marksplice.Node {
	tb.Helper()
	for _, node := range doc.Nodes() {
		if node.Kind() == kind {
			return node
		}
	}
	tb.Fatalf("node of kind %v not found", kind)
	return marksplice.Node{}
}
