package publictest

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/zoster81/marksplice"
)

var benchmarkAlertReadSink int

func BenchmarkAlertReadScaling(b *testing.B) {
	for _, count := range []int{16, 256, 1024} {
		document, err := marksplice.Parse(bytes.Repeat([]byte("> [!NOTE]\n> first\n>\n> second\n\n"), count))
		if err != nil {
			b.Fatal(err)
		}
		alerts := document.Alerts()
		if len(alerts) != count {
			b.Fatalf("alerts = %d, want %d", len(alerts), count)
		}
		b.Run(fmt.Sprintf("%dAlerts", count), func(b *testing.B) {
			b.Run("Metadata", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					for _, alert := range alerts {
						value, ok := document.Alert(alert.ID())
						if !ok {
							b.Fatal("missing alert")
						}
						benchmarkAlertReadSink = int(value.Kind())
					}
				}
			})
			b.Run("BodyRanges", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					for _, alert := range alerts {
						value, ok := document.AlertBodyRanges(alert.ID())
						if !ok {
							b.Fatal("missing alert body")
						}
						benchmarkAlertReadSink = len(value)
					}
				}
			})
		})
	}
}
