package publictest

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zoster81/marksplice"
)

var benchmarkListReadSink int

// Measure list-only snapshots, including child ownership and construction proof.
func BenchmarkListReadScaling(b *testing.B) {
	for _, nested := range []bool{false, true} {
		for _, count := range []int{16, 256, 1024} {
			items := make([]marksplice.ListItemInput, count)
			var source strings.Builder
			for index := range items {
				depth := 0
				if nested {
					depth = index % 4
				}
				items[index] = marksplice.ListItemInput{InlineGFM: "item", Depth: depth}
				source.WriteString(strings.Repeat("  ", depth) + "- item\n")
			}
			input := []byte(source.String())
			document, err := marksplice.Parse(input)
			if err != nil {
				b.Fatal(err)
			}
			var ids []marksplice.NodeID
			for _, node := range document.Nodes() {
				if node.Kind() == marksplice.KindListItem {
					ids = append(ids, node.ID())
				}
			}
			if len(ids) != count {
				b.Fatalf("list contains %d items, want %d", len(ids), count)
			}
			b.Run(fmt.Sprintf("Nested%t/%dItems", nested, count), func(b *testing.B) {
				b.Run("TypedReads", func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						for _, id := range ids {
							item, ok := document.ListItem(id)
							if !ok {
								b.Fatal("missing list item")
							}
							benchmarkListReadSink = int(item.Marker())
						}
					}
				})
				b.Run("Parse", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for i := 0; i < b.N; i++ {
						parsed, err := marksplice.Parse(input)
						if err != nil {
							b.Fatal(err)
						}
						benchmarkListReadSink = len(parsed.Nodes())
					}
				})
				b.Run("Construction", func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						builder := marksplice.NewDocumentBuilder()
						if err := builder.AppendNestedUnorderedList(items...); err != nil {
							b.Fatal(err)
						}
						output, err := builder.Markdown()
						if err != nil {
							b.Fatal(err)
						}
						benchmarkListReadSink = len(output)
					}
				})
			})
		}
	}
}
