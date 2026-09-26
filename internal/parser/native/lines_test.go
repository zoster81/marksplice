package native

import (
	"strings"
	"testing"
)

func TestIndentationRetainsTabAlignmentAfterLongPrefixes(t *testing.T) {
	for _, prefixLength := range []int{0, 1, 2, 3, 4, 127, 255, 256, 257, 1023, 4095} {
		for _, indentation := range []string{"", " ", "\t", "\t \t", "   \t\t  "} {
			source := []byte(strings.Repeat("x", prefixLength) + indentation + "text\r\n")
			line := advancePhysicalLineStart(source, physicalLines(source)[0], prefixLength, 0)
			column := prefixLength
			for _, value := range indentation {
				column++
				if value == '\t' {
					for column%4 != 0 {
						column++
					}
				}
			}
			width := column - prefixLength
			for first := 0; first <= width+2; first++ {
				stripped := stripIndentColumns(source, line, first)
				_, got := leadingIndent(source, stripped)
				if want := max(0, width-first); got != want {
					t.Fatalf("prefix=%d indent=%q strip=%d: columns=%d want=%d", prefixLength, indentation, first, got, want)
				}
				for second := 0; second <= width+2; second++ {
					again := stripIndentColumns(source, stripped, second)
					direct := stripIndentColumns(source, line, first+second)
					if again != direct {
						t.Fatalf("prefix=%d indent=%q strips=%d+%d: sequential=%+v direct=%+v", prefixLength, indentation, first, second, again, direct)
					}
				}
			}
		}
	}
}
