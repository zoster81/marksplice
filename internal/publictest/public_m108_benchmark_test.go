package publictest

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/zoster81/marksplice"
)

var (
	m108DocumentSink  *marksplice.Document
	m108GraphSink     *marksplice.DocumentGraph
	m108WorkspaceSink *marksplice.WorkspaceReport
	m108KnowledgeSink *marksplice.KnowledgeIndex
	m108NodeSink      []marksplice.NodeMatch
	m108SectionSink   []marksplice.Section
	m108AnchorSink    []marksplice.HeadingAnchor
	m108BytesSink     []byte
	m108LinksSink     []marksplice.LinkRelationship
	m108KeysSink      []marksplice.DocumentKey
	m108ChangeSink    marksplice.ChangeSet
)

func BenchmarkM108ParseRealistic(b *testing.B) {
	for _, sizeKiB := range []int{64, 256, 1024} {
		source := m108RealisticSource(sizeKiB << 10)
		b.Run(fmt.Sprintf("%dKiB", sizeKiB), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				document, err := marksplice.Parse(source)
				if err != nil {
					b.Fatal(err)
				}
				m108DocumentSink = document
			}
		})
	}
}

func BenchmarkM108DocumentIntelligence(b *testing.B) {
	source := m108RealisticSource(256 << 10)
	document, err := marksplice.Parse(source)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("M97QueryNodes", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			matches, err := document.QueryNodes(marksplice.NodeQuery{Limit: 256})
			if err != nil {
				b.Fatal(err)
			}
			m108NodeSink = matches
		}
	})
	b.Run("M97QuerySections", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			sections, err := document.QuerySections(marksplice.SectionQuery{Limit: 256})
			if err != nil {
				b.Fatal(err)
			}
			m108SectionSink = sections
		}
	})
	b.Run("M98HeadingAnchors", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			m108AnchorSink = document.HeadingAnchors()
		}
	})
	b.Run("M98GenerateTOC", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			m108BytesSink = document.GenerateTOC()
		}
	})
	b.Run("M99LinkRelationships", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			m108LinksSink = document.LinkRelationships()
		}
	})
	b.Run("M102Alerts", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = document.Alerts()
		}
	})
	b.Run("M103FencedBlocks", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = document.FencedBlocks()
		}
	})
	b.Run("M104Footnotes", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = document.FootnoteDefinitions()
			_ = document.FootnoteReferences()
		}
	})
	b.Run("M105Math", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = document.MathExpressions()
		}
	})
	b.Run("M106FrontMatter", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = document.FrontMatter()
		}
	})
}

func BenchmarkM108MutationPlanning(b *testing.B) {
	for _, sizeKiB := range []int{64, 256, 1024} {
		source := m108RealisticSource(sizeKiB << 10)
		document, err := marksplice.Parse(source)
		if err != nil {
			b.Fatal(err)
		}
		matches, err := document.QueryNodes(marksplice.NodeQuery{Kinds: []marksplice.Kind{marksplice.KindParagraph}, Limit: 1})
		if err != nil || len(matches) != 1 {
			b.Fatalf("paragraph query = %d, %v", len(matches), err)
		}
		payload, ok := document.SourceRange(matches[0].Range())
		if !ok {
			b.Fatal("paragraph range unavailable")
		}
		id := matches[0].Node().ID()
		b.Run(fmt.Sprintf("%dKiB", sizeKiB), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				change, err := document.PrepareReplaceParagraph(id, payload)
				if err != nil {
					b.Fatal(err)
				}
				m108ChangeSink = change
			}
		})
	}
}

func BenchmarkM108ChangeCompositionScaling(b *testing.B) {
	source := m108RealisticSource(256 << 10)
	document, err := marksplice.Parse(source)
	if err != nil {
		b.Fatal(err)
	}
	matches, err := document.QueryNodes(marksplice.NodeQuery{Kinds: []marksplice.Kind{marksplice.KindParagraph}, Limit: 16})
	if err != nil || len(matches) < 16 {
		b.Fatalf("paragraph query = %d, %v", len(matches), err)
	}
	changes := make([]marksplice.ChangeSet, 16)
	for i := range changes {
		change, err := document.PrepareReplaceParagraph(matches[i].Node().ID(), []byte(fmt.Sprintf("Replacement paragraph %d.", i)))
		if err != nil {
			b.Fatalf("prepare paragraph %d: %v", i, err)
		}
		changes[i] = change
	}
	for _, count := range []int{1, 4, 16} {
		b.Run(fmt.Sprintf("Changes/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				combined, err := document.ComposeChanges(changes[:count]...)
				if err != nil {
					b.Fatal(err)
				}
				m108ChangeSink = combined
			}
		})
	}
}

func BenchmarkM108WorkspaceGraphKnowledgeScaling(b *testing.B) {
	for _, count := range []int{64, 256, 1024} {
		documents := m108GraphDocuments(b, count)
		resolver := m108DocumentResolver(count)
		workspaceResolver := m108WorkspaceResolver(count)
		graph, err := marksplice.BuildDocumentGraph(documents, resolver)
		if err != nil {
			b.Fatal(err)
		}
		metadata := m108KnowledgeDocuments(count)

		b.Run(fmt.Sprintf("BuildGraph/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				got, err := marksplice.BuildDocumentGraph(documents, resolver)
				if err != nil {
					b.Fatal(err)
				}
				m108GraphSink = got
			}
		})
		b.Run(fmt.Sprintf("ValidateWorkspace/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			options := marksplice.WorkspaceValidationOptions{Roots: []marksplice.DocumentKey{"doc-0"}}
			for i := 0; i < b.N; i++ {
				report, err := marksplice.ValidateWorkspace(documents, workspaceResolver, options)
				if err != nil {
					b.Fatal(err)
				}
				m108WorkspaceSink = report
			}
		})
		b.Run(fmt.Sprintf("BuildKnowledge/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				index, err := marksplice.BuildKnowledgeIndex(graph, metadata)
				if err != nil {
					b.Fatal(err)
				}
				m108KnowledgeSink = index
			}
		})
		index, err := marksplice.BuildKnowledgeIndex(graph, metadata)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("KnowledgeReachable/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				keys, ok := index.ReachableFrom("doc-0")
				if !ok {
					b.Fatal("root unavailable")
				}
				m108KeysSink = keys
			}
		})
		b.Run(fmt.Sprintf("KnowledgeTagLookup/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				m108KeysSink = index.DocumentsWithTag("benchmark")
			}
		})
		alias := marksplice.KnowledgeAlias(fmt.Sprintf("alias-%d", count-1))
		b.Run(fmt.Sprintf("KnowledgeAliasLookup/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = index.ResolveAlias(alias)
			}
		})
	}
}

func BenchmarkM108PathologicalParseScaling(b *testing.B) {
	for _, sizeKiB := range []int{16, 64, 256} {
		source := []byte(strings.Repeat("*_~`", (sizeKiB<<10)/4) + "\n")
		b.Run(fmt.Sprintf("DenseDelimiters/%dKiB", sizeKiB), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for i := 0; i < b.N; i++ {
				document, err := marksplice.Parse(source)
				if err != nil {
					b.Fatal(err)
				}
				m108DocumentSink = document
			}
		})
	}
	for _, depth := range []int{256, 1024, 4096} {
		source := []byte(strings.Repeat("> ", depth) + "payload\n")
		b.Run(fmt.Sprintf("DeepBlockquote/%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for i := 0; i < b.N; i++ {
				document, err := marksplice.Parse(source)
				if err != nil {
					b.Fatal(err)
				}
				m108DocumentSink = document
			}
		})
	}
}

func BenchmarkM108DenseReadProjectionScaling(b *testing.B) {
	for _, count := range []int{1024, 4096, 16384} {
		source := repeatedM108Source(count, func(i int) string {
			return fmt.Sprintf("[target-%d](#target-%d) ", i, i)
		})
		document, err := marksplice.Parse(source)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("Relationships/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				m108LinksSink = document.LinkRelationships()
			}
		})
	}
}

func BenchmarkM108DuplicateHeadingAnchorScaling(b *testing.B) {
	for _, count := range []int{1024, 4096, 16384} {
		source := repeatedM108Source(count, func(int) string { return "# Same heading\n\n" })
		document, err := marksplice.Parse(source)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("Anchors/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				m108AnchorSink = document.HeadingAnchors()
			}
		})
	}
}

func m108RealisticSource(minBytes int) []byte {
	var builder strings.Builder
	builder.Grow(minBytes + 4096)
	builder.WriteString("---\ntitle: \"M108 benchmark\"\nowner: \"marksplice\"\n---\n\n")
	for i := 0; builder.Len() < minBytes; i++ {
		fmt.Fprintf(&builder, "# Section %d\n\n", i)
		fmt.Fprintf(&builder, "Paragraph %d with *emphasis*, **strong**, ~~strike~~, `code`, [local](#section-%d), [external](doc-%d.md), and $x_%d+y$.\n\n", i, i, i+1, i)
		builder.WriteString("> [!NOTE]\n> benchmark alert body\n\n")
		fmt.Fprintf(&builder, "```go\nfmt.Println(%d)\n```\n\n", i)
		fmt.Fprintf(&builder, "[^note-%d]: footnote body with [link](doc-%d.md)\n\nreference[^note-%d]\n\n", i, i+1, i)
		builder.WriteString("| Left | Right |\n| :--- | ---: |\n| a | b |\n\n")
	}
	return []byte(builder.String())
}

func m108GraphDocuments(tb testing.TB, count int) []marksplice.GraphDocument {
	tb.Helper()
	result := make([]marksplice.GraphDocument, count)
	for i := 0; i < count; i++ {
		var source string
		if i+1 < count {
			source = fmt.Sprintf("# Document %d\n\n[next](doc-%d.md)\n", i, i+1)
		} else {
			source = fmt.Sprintf("# Document %d\n", i)
		}
		document, err := marksplice.Parse([]byte(source))
		if err != nil {
			tb.Fatalf("parse graph document %d: %v", i, err)
		}
		result[i] = marksplice.GraphDocument{Key: marksplice.DocumentKey(fmt.Sprintf("doc-%d", i)), Document: document}
	}
	return result
}

func m108DocumentResolver(count int) marksplice.DocumentResolver {
	return func(_ marksplice.DocumentKey, relationship marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
		target, ok := m108TargetKey(relationship.Destination(), count)
		if !ok {
			return marksplice.DocumentResolution{}, false
		}
		return marksplice.DocumentResolution{Target: target}, true
	}
}

func m108WorkspaceResolver(count int) marksplice.WorkspaceResolver {
	return func(_ marksplice.DocumentKey, relationship marksplice.LinkRelationship) marksplice.WorkspaceResolution {
		target, ok := m108TargetKey(relationship.Destination(), count)
		if !ok {
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionIgnore}
		}
		return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionResolved, Target: target}
	}
}

func m108TargetKey(destination string, count int) (marksplice.DocumentKey, bool) {
	if !strings.HasPrefix(destination, "doc-") || !strings.HasSuffix(destination, ".md") {
		return "", false
	}
	value, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(destination, "doc-"), ".md"))
	if err != nil || value < 0 || value >= count {
		return "", false
	}
	return marksplice.DocumentKey(fmt.Sprintf("doc-%d", value)), true
}

func m108KnowledgeDocuments(count int) []marksplice.KnowledgeDocument {
	result := make([]marksplice.KnowledgeDocument, count)
	for i := 0; i < count; i++ {
		item := marksplice.KnowledgeDocument{
			Document: marksplice.DocumentKey(fmt.Sprintf("doc-%d", i)),
			Aliases:  []marksplice.KnowledgeAlias{marksplice.KnowledgeAlias(fmt.Sprintf("alias-%d", i))},
			Tags:     []marksplice.KnowledgeTag{"benchmark"},
		}
		if i+2 < count {
			item.References = []marksplice.DocumentKey{marksplice.DocumentKey(fmt.Sprintf("doc-%d", i+2))}
		}
		result[i] = item
	}
	return result
}
