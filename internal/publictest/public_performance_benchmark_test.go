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

func BenchmarkParseRealistic(b *testing.B) {
	for _, sizeKiB := range []int{64, 256, 1024} {
		source := realisticSource(sizeKiB << 10)
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

func BenchmarkDocumentIntelligence(b *testing.B) {
	source := realisticSource(256 << 10)
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

func BenchmarkGeneralMutationPlanning(b *testing.B) {
	for _, sizeKiB := range []int{64, 256, 1024} {
		source := realisticSource(sizeKiB << 10)
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

func BenchmarkStructuralMutationPlanning(b *testing.B) {
	b.Run("ReferenceRename", func(b *testing.B) {
		source := []byte("[visible][old]\n\n[old]: <https://example.com> \"title\"\n")
		document, err := marksplice.Parse(source)
		if err != nil {
			b.Fatal(err)
		}
		matches, err := document.QueryNodes(marksplice.NodeQuery{Kinds: []marksplice.Kind{marksplice.KindReferenceDefinition}, Limit: 1})
		if err != nil || len(matches) != 1 {
			b.Fatalf("reference definition query = %d, %v", len(matches), err)
		}
		id := matches[0].Node().ID()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			change, err := document.PrepareRenameReferenceDefinition(id, []byte("new"))
			if err != nil {
				b.Fatal(err)
			}
			m108ChangeSink = change
		}
	})

	b.Run("FootnoteMultiline", func(b *testing.B) {
		source := []byte("See[^n]\r\n\r\n[^n]: first\r\n\r\n    second\r\n")
		document, err := marksplice.Parse(source)
		if err != nil {
			b.Fatal(err)
		}
		definitions := document.FootnoteDefinitions()
		if len(definitions) != 1 {
			b.Fatalf("footnote definitions = %d, want 1", len(definitions))
		}
		id := definitions[0].ID()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			change, err := document.PrepareReplaceFootnoteDefinitionBodyMultiline(id, []byte("alpha\n\nbeta"))
			if err != nil {
				b.Fatal(err)
			}
			m108ChangeSink = change
		}
	})

	b.Run("FrontMatterRename", func(b *testing.B) {
		source := []byte("---\ntitle: \"old\"\n---\n\nBody.\n")
		document, err := marksplice.Parse(source)
		if err != nil {
			b.Fatal(err)
		}
		matches, err := document.QueryNodes(marksplice.NodeQuery{Kinds: []marksplice.Kind{marksplice.KindFrontMatterField}, Limit: 1})
		if err != nil || len(matches) != 1 {
			b.Fatalf("front-matter field query = %d, %v", len(matches), err)
		}
		id := matches[0].Node().ID()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			change, err := document.PrepareRenameFrontMatterField(id, []byte("name"))
			if err != nil {
				b.Fatal(err)
			}
			m108ChangeSink = change
		}
	})
}

func BenchmarkChangeCompositionScaling(b *testing.B) {
	source := realisticSource(256 << 10)
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

func BenchmarkWorkspaceGraphKnowledgeScaling(b *testing.B) {
	for _, count := range []int{64, 256, 1024} {
		documents := graphDocuments(b, count)
		resolver := documentResolver(count)
		workspaceResolver := workspaceResolver(count)
		graph, err := marksplice.BuildDocumentGraph(documents, resolver)
		if err != nil {
			b.Fatal(err)
		}
		metadata := knowledgeDocuments(count)

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

func BenchmarkPathologicalParseScaling(b *testing.B) {
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

func BenchmarkDenseReadProjectionScaling(b *testing.B) {
	for _, count := range []int{1024, 4096, 16384} {
		source := repeatedSource(count, func(i int) string {
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

func BenchmarkDuplicateHeadingAnchorScaling(b *testing.B) {
	for _, count := range []int{1024, 4096, 16384} {
		source := repeatedSource(count, func(int) string { return "# Same heading\n\n" })
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

func realisticSource(minBytes int) []byte {
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

func graphDocuments(tb testing.TB, count int) []marksplice.GraphDocument {
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

func documentResolver(count int) marksplice.DocumentResolver {
	return func(_ marksplice.DocumentKey, relationship marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
		target, ok := targetKey(relationship.Destination(), count)
		if !ok {
			return marksplice.DocumentResolution{}, false
		}
		return marksplice.DocumentResolution{Target: target}, true
	}
}

func workspaceResolver(count int) marksplice.WorkspaceResolver {
	return func(_ marksplice.DocumentKey, relationship marksplice.LinkRelationship) marksplice.WorkspaceResolution {
		target, ok := targetKey(relationship.Destination(), count)
		if !ok {
			return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionIgnore}
		}
		return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionResolved, Target: target}
	}
}

func targetKey(destination string, count int) (marksplice.DocumentKey, bool) {
	if !strings.HasPrefix(destination, "doc-") || !strings.HasSuffix(destination, ".md") {
		return "", false
	}
	value, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(destination, "doc-"), ".md"))
	if err != nil || value < 0 || value >= count {
		return "", false
	}
	return marksplice.DocumentKey(fmt.Sprintf("doc-%d", value)), true
}

func knowledgeDocuments(count int) []marksplice.KnowledgeDocument {
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
