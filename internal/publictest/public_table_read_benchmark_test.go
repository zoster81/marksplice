package publictest

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/zoster81/marksplice"
)

var benchmarkTableReadSink int

// Exercise typed table reads independently of parsing and node enumeration.
func BenchmarkTableReadScaling(b *testing.B) {
	for _, rowCount := range []int{16, 256, 1024} {
		source := append([]byte("| left | right |\n| :--- | ---: |\n"), bytes.Repeat([]byte("| a | b |\n"), rowCount)...)
		document, err := marksplice.Parse(source)
		if err != nil {
			b.Fatal(err)
		}
		var tableID marksplice.NodeID
		var rows, cells []marksplice.NodeID
		for _, node := range document.Nodes() {
			switch node.Kind() {
			case marksplice.KindTable:
				tableID = node.ID()
			case marksplice.KindTableRow:
				rows = append(rows, node.ID())
			case marksplice.KindTableCell:
				cells = append(cells, node.ID())
			}
		}
		if len(rows) != rowCount || len(cells) != 2*(rowCount+1) {
			b.Fatalf("unexpected table: rows=%d cells=%d", len(rows), len(cells))
		}
		b.Run(fmt.Sprintf("%dRows", rowCount), func(b *testing.B) {
			b.Run("Table", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					value, ok := document.Table(tableID)
					if !ok {
						b.Fatal("missing table")
					}
					benchmarkTableReadSink = value.ColumnCount()
				}
			})
			b.Run("Rows", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					for _, id := range rows {
						value, ok := document.TableRow(id)
						if !ok {
							b.Fatal("missing row")
						}
						benchmarkTableReadSink = value.ColumnCount()
					}
				}
			})
			b.Run("Cells", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					for _, id := range cells {
						value, ok := document.TableCell(id)
						if !ok {
							b.Fatal("missing cell")
						}
						benchmarkTableReadSink = value.Column()
					}
				}
			})
			b.Run("RowCellIDs", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					for _, id := range rows {
						values, ok := document.TableRowCellIDs(id)
						if !ok {
							b.Fatal("missing row cells")
						}
						benchmarkTableReadSink = len(values)
					}
				}
			})
		})
	}
}
