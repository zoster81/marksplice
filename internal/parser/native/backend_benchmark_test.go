package native_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

var m114ObservationsSink parser.DocumentObservations

func BenchmarkM114NativeBackendScaling(b *testing.B) {
	backend := native.New()
	families := []struct {
		name string
		unit string
	}{
		{name: "DirectLinks", unit: "[label](<target>) "},
		{name: "Delimiters", unit: "*a* **b** ~~c~~ `d` "},
	}
	for _, family := range families {
		for _, count := range []int{256, 1024, 4096} {
			source := []byte(strings.Repeat(family.unit, count) + "\n")
			b.Run(fmt.Sprintf("%s/%d", family.name, count), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(source)))
				for iteration := 0; iteration < b.N; iteration++ {
					observations, err := backend.ParseDocument(source)
					if err != nil {
						b.Fatal(err)
					}
					m114ObservationsSink = observations
				}
			})
		}
	}
	for _, depth := range []int{256, 1024, 4096} {
		source := []byte(strings.Repeat("> ", depth) + "payload\n")
		b.Run(fmt.Sprintf("DeepBlockquote/%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for iteration := 0; iteration < b.N; iteration++ {
				observations, err := backend.ParseDocument(source)
				if err != nil {
					b.Fatal(err)
				}
				m114ObservationsSink = observations
			}
		})
	}
}

func BenchmarkM114NativeBackendRealisticScaling(b *testing.B) {
	backend := native.New()
	for _, sizeKiB := range []int{64, 256, 1024} {
		source := m114RealisticSource(sizeKiB << 10)
		b.Run(fmt.Sprintf("%dKiB", sizeKiB), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for iteration := 0; iteration < b.N; iteration++ {
				observations, err := backend.ParseDocument(source)
				if err != nil {
					b.Fatal(err)
				}
				m114ObservationsSink = observations
			}
		})
	}
}

func m114RealisticSource(minBytes int) []byte {
	var source strings.Builder
	source.Grow(minBytes + 4096)
	source.WriteString("---\ntitle: \"M108 benchmark\"\nowner: \"marksplice\"\n---\n\n")
	for index := 0; source.Len() < minBytes; index++ {
		fmt.Fprintf(&source, "# Section %d\n\n", index)
		fmt.Fprintf(&source, "Paragraph %d with *emphasis*, **strong**, ~~strike~~, `code`, [local](#section-%d), [external](doc-%d.md), and $x_%d+y$.\n\n", index, index, index+1, index)
		source.WriteString("> [!NOTE]\n> benchmark alert body\n\n")
		fmt.Fprintf(&source, "```go\nfmt.Println(%d)\n```\n\n", index)
		fmt.Fprintf(&source, "[^note-%d]: footnote body with [link](doc-%d.md)\n\nreference[^note-%d]\n\n", index, index+1, index)
		source.WriteString("| Left | Right |\n| :--- | ---: |\n| a | b |\n\n")
	}
	return []byte(source.String())
}

func BenchmarkM114NativeReferenceDefinitionReuse(b *testing.B) {
	backend := native.New()
	for _, test := range []struct {
		definitions int
		blocks      int
	}{
		{definitions: 64, blocks: 256},
		{definitions: 256, blocks: 1024},
	} {
		source := m114ReferenceDefinitionSource(test.definitions, test.blocks)
		b.Run(fmt.Sprintf("Definitions%d/Blocks%d", test.definitions, test.blocks), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for iteration := 0; iteration < b.N; iteration++ {
				observations, err := backend.ParseDocument(source)
				if err != nil {
					b.Fatal(err)
				}
				m114ObservationsSink = observations
			}
		})
	}
}

func m114ReferenceDefinitionSource(definitions, blocks int) []byte {
	var source strings.Builder
	for index := range definitions {
		fmt.Fprintf(&source, "[label-%d]: <target-%d>\n", index, index)
	}
	source.WriteByte('\n')
	for index := range blocks {
		fmt.Fprintf(&source, "paragraph %d using [label-%d].\n\n", index, index%definitions)
	}
	return []byte(source.String())
}

func BenchmarkM114NativeFootnoteReferenceDefinitionReuse(b *testing.B) {
	backend := native.New()
	for _, test := range []struct {
		definitions int
		blocks      int
	}{
		{definitions: 64, blocks: 256},
		{definitions: 256, blocks: 1024},
	} {
		source := m114FootnoteReferenceDefinitionSource(test.definitions, test.blocks)
		b.Run(fmt.Sprintf("Definitions%d/Blocks%d", test.definitions, test.blocks), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for iteration := 0; iteration < b.N; iteration++ {
				observations, err := backend.ParseDocument(source)
				if err != nil {
					b.Fatal(err)
				}
				m114ObservationsSink = observations
			}
		})
	}
}

func m114FootnoteReferenceDefinitionSource(definitions, blocks int) []byte {
	var source strings.Builder
	source.WriteString("[^note]: footnote body\n\n")
	for index := range definitions {
		fmt.Fprintf(&source, "[label-%d]: <target-%d>\n", index, index)
	}
	source.WriteByte('\n')
	for index := range blocks {
		fmt.Fprintf(&source, "paragraph %d using [label-%d] and footnote [^note].\n\n", index, index%definitions)
	}
	return []byte(source.String())
}

func BenchmarkM114NativeConstructionProofScaling(b *testing.B) {
	backend := native.New()
	for _, count := range []int{256, 1024, 4096} {
		source, expected := m114SiblingEmphasisProof(count)
		b.Run(fmt.Sprintf("SiblingEmphasis/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for iteration := 0; iteration < b.N; iteration++ {
				if err := backend.ValidateConstructionInlineHierarchy(source, expected, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func m114SiblingEmphasisProof(count int) ([]byte, []parser.ConstructionInlineExpectation) {
	var source strings.Builder
	source.Grow(count * 4)
	expected := make([]parser.ConstructionInlineExpectation, count)
	for index := range count {
		start := source.Len()
		source.WriteString("*x*")
		if index+1 < count {
			source.WriteByte(' ')
		}
		expected[index] = parser.ConstructionInlineExpectation{
			Kind:            parser.KindEmphasis,
			SyntaxRange:     parser.Range{Start: start, End: start + 3},
			ContentRange:    parser.Range{Start: start + 1, End: start + 2},
			Marker:          '*',
			DelimiterLength: 1,
			Parent:          -1,
		}
	}
	return []byte(source.String()), expected
}
