package marksplice_test

import (
	"fmt"
	"strings"

	"github.com/zoster81/marksplice"
)

func ExampleDocumentBuilder() {
	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendHeadingContent(1, marksplice.TextInline("Marksplice")); err != nil {
		panic(err)
	}
	if err := builder.AppendParagraphContent(marksplice.TextInline("Source preserving GFM")); err != nil {
		panic(err)
	}

	source, err := builder.Markdown()
	if err != nil {
		panic(err)
	}
	fmt.Print(string(source))

	// Output:
	// # Marksplice
	//
	// Source preserving GFM
}

func ExampleParse() {
	source := []byte("# Title\n\nBody.\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		panic(err)
	}

	for _, node := range document.Nodes() {
		if node.Kind() != marksplice.KindHeading {
			continue
		}
		heading, ok := document.Heading(node.ID())
		if !ok {
			continue
		}
		content, ok := document.SourceRange(heading.Range())
		if !ok {
			panic("heading range is not readable")
		}
		fmt.Printf("level=%d text=%s\n", heading.Level(), content)
	}

	// Output:
	// level=1 text=Title
}

func ExampleParseWithOptions() {
	source := []byte("See [[guide]].\n")
	wiki := marksplice.Extension{
		ID: "example.org/wiki",
		Recognize: func(input marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
			text := input.Text()
			start := strings.Index(text, "[[")
			if start < 0 {
				return nil, nil
			}
			close := strings.Index(text[start+2:], "]]")
			if close < 0 {
				return nil, nil
			}
			end := start + 2 + close + 2
			return []marksplice.ExtensionMatch{{
				Kind:  "wikilink",
				Range: marksplice.Range{Start: start, End: end},
				Attributes: []marksplice.ExtensionAttribute{{
					Name:  "target",
					Value: text[start+2 : end-2],
				}},
			}}, nil
		},
	}
	document, err := marksplice.ParseWithOptions(source, marksplice.ParseOptions{
		Extensions: []marksplice.Extension{wiki},
		ExtensionLimits: marksplice.ExtensionLimits{
			MaxNodes:         8,
			MaxMetadataBytes: 256,
		},
	})
	if err != nil {
		panic(err)
	}
	node := document.ExtensionNodes()[0]
	raw, _ := document.SourceRange(node.Range())
	target, _ := node.Attribute("target")
	fmt.Printf("%s -> %s\n", raw, target)

	// Output:
	// [[guide]] -> guide
}

func ExampleDocument_QueryNodes() {
	source := []byte("# One\n\nBody.\n\n## Two\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		panic(err)
	}
	matches, err := document.QueryNodes(marksplice.NodeQuery{
		Kinds: []marksplice.Kind{marksplice.KindHeading},
		Limit: 10,
	})
	if err != nil {
		panic(err)
	}
	for _, match := range matches {
		content, ok := document.SourceRange(match.Range())
		if !ok {
			panic("query range is not readable")
		}
		fmt.Println(string(content))
	}

	// Output:
	// One
	// Two
}

func ExampleDocument_GenerateTOC() {
	source := []byte("# Root\n\n## Child\n\n## Child\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(document.GenerateTOC()))

	// Output:
	// - [Root](#root)
	//   - [Child](#child)
	//   - [Child](#child-1)
}

func ExampleDocument_LinkRelationships() {
	source := []byte("# Guide\n\n[local](#guide) [web](https://example.com)\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		panic(err)
	}
	for _, relationship := range document.LinkRelationships() {
		_, local := relationship.FragmentTarget()
		fmt.Printf("%s local=%t\n", relationship.Destination(), local)
	}

	// Output:
	// #guide local=true
	// https://example.com local=false
}

func ExampleBuildDocumentGraph() {
	index, err := marksplice.Parse([]byte("# Index\n\n[guide](guide.md#guide)\n"))
	if err != nil {
		panic(err)
	}
	guide, err := marksplice.Parse([]byte("# Guide\n"))
	if err != nil {
		panic(err)
	}
	graph, err := marksplice.BuildDocumentGraph([]marksplice.GraphDocument{
		{Key: "index", Document: index},
		{Key: "guide", Document: guide},
	}, func(_ marksplice.DocumentKey, relationship marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
		if relationship.Destination() != "guide.md#guide" {
			return marksplice.DocumentResolution{}, false
		}
		return marksplice.DocumentResolution{Target: "guide", Fragment: "#guide"}, true
	})
	if err != nil {
		panic(err)
	}
	for _, edge := range graph.Edges() {
		_, fragmentResolved := edge.FragmentTarget()
		fmt.Printf("%s -> %s fragment=%t\n", edge.SourceDocument(), edge.TargetDocument(), fragmentResolved)
	}

	// Output:
	// index -> guide fragment=true
}

func ExampleBuildKnowledgeIndex() {
	a, err := marksplice.Parse([]byte("# A\n\n[to-b](b.md)\n"))
	if err != nil {
		panic(err)
	}
	b, err := marksplice.Parse([]byte("# B\n"))
	if err != nil {
		panic(err)
	}
	graph, err := marksplice.BuildDocumentGraph([]marksplice.GraphDocument{
		{Key: "a", Document: a},
		{Key: "b", Document: b},
	}, func(_ marksplice.DocumentKey, relationship marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
		if relationship.Destination() == "b.md" {
			return marksplice.DocumentResolution{Target: "b"}, true
		}
		return marksplice.DocumentResolution{}, false
	})
	if err != nil {
		panic(err)
	}
	knowledge, err := marksplice.BuildKnowledgeIndex(graph, []marksplice.KnowledgeDocument{
		{Document: "a", Aliases: []marksplice.KnowledgeAlias{"start"}, Tags: []marksplice.KnowledgeTag{"guide"}},
		{Document: "b", Tags: []marksplice.KnowledgeTag{"guide"}, References: []marksplice.DocumentKey{"a"}},
	})
	if err != nil {
		panic(err)
	}
	alias, _ := knowledge.ResolveAlias("start")
	related, _ := knowledge.RelatedDocuments("a")
	fmt.Printf("alias=%s tagged=%d related=%v\n", alias, len(knowledge.DocumentsWithTag("guide")), related)

	// Output:
	// alias=a tagged=2 related=[b]
}

func ExampleValidateWorkspace() {
	document, err := marksplice.Parse([]byte("# Guide\n\n[missing](#missing)\n"))
	if err != nil {
		panic(err)
	}
	report, err := marksplice.ValidateWorkspace([]marksplice.GraphDocument{
		{Key: "guide", Document: document},
	}, nil, marksplice.WorkspaceValidationOptions{})
	if err != nil {
		panic(err)
	}
	diagnostic := report.Diagnostics()[0]
	fragment, _ := diagnostic.Fragment()
	fmt.Printf("missing fragment=%t %s\n", diagnostic.Kind() == marksplice.WorkspaceDiagnosticMissingFragment, fragment)
	fmt.Printf("repairs=%d\n", len(report.RepairPlan().Repairs()))

	// Output:
	// missing fragment=true #missing
	// repairs=0
}

func ExampleDocument_Alerts() {
	document, err := marksplice.Parse([]byte("> [!WARNING]\n> Back up first.\n"))
	if err != nil {
		panic(err)
	}
	alert := document.Alerts()[0]
	marker, ok := document.SourceRange(alert.MarkerRange())
	if !ok {
		panic("alert marker is not readable")
	}
	bodyRanges, ok := document.AlertBodyRanges(alert.ID())
	if !ok || len(bodyRanges) == 0 {
		panic("alert body is not readable")
	}
	body, ok := document.SourceRange(bodyRanges[0])
	if !ok {
		panic("alert body range is not readable")
	}
	fmt.Printf("%s: %s\n", marker, body)

	// Output:
	// [!WARNING]: Back up first.
}

func ExampleDocument_FencedBlocks() {
	document, err := marksplice.Parse([]byte("```mermaid\ngraph TD\n```\n"))
	if err != nil {
		panic(err)
	}
	block := document.FencedBlocks()[0]
	language, ok := block.Language()
	if !ok {
		panic("fenced block language is not available")
	}
	bodyRanges, ok := document.FencedBlockContentRanges(block.ID())
	if !ok || len(bodyRanges) != 1 {
		panic("fenced block body is not readable")
	}
	body, ok := document.SourceRange(bodyRanges[0])
	if !ok {
		panic("fenced block body range is not readable")
	}
	fmt.Printf("language=%s closed=%t body=%s\n", language, block.Closed(), body)

	// Output:
	// language=mermaid closed=true body=graph TD
}

func ExampleDocument_FootnoteDefinitions() {
	source := []byte("Text[^note].\n\n[^note]: Original body.\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		panic(err)
	}
	definition := document.FootnoteDefinitions()[0]
	references := document.FootnoteReferences()
	fmt.Printf("label=%s references=%d occurrence=%d\n", definition.Label(), len(references), references[0].Occurrence())

	change, err := document.PrepareRenameFootnote(definition.ID(), []byte("renamed"))
	if err != nil {
		panic(err)
	}
	updated, err := change.Apply(source)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(updated))

	// Output:
	// label=note references=1 occurrence=0
	// Text[^renamed].
	//
	// [^renamed]: Original body.
}

func ExampleDocumentBuilder_DeferFootnoteDefinition() {
	builder := marksplice.NewDocumentBuilder()
	if err := builder.DeferFootnoteDefinition("note", "Built body."); err != nil {
		panic(err)
	}
	if err := builder.AppendParagraphContent(marksplice.TextInline("Text"), marksplice.FootnoteReferenceInline("note")); err != nil {
		panic(err)
	}
	markdown, err := builder.Markdown()
	if err != nil {
		panic(err)
	}
	fmt.Print(string(markdown))

	// Output:
	// Text[^note]
	//
	// [^note]: Built body.
}

func ExampleDocument_FrontMatter() {
	source := []byte("---\ntags:\n  - source-preserving\n---\n\n# Marksplice\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		panic(err)
	}
	frontMatter, ok := document.FrontMatter()
	if !ok {
		panic("front matter is not available")
	}
	metadata, ok := document.SourceRange(frontMatter.Range())
	if !ok {
		panic("front matter range is not readable")
	}
	fmt.Printf("format=%d bytes=%d\n", frontMatter.Format(), len(metadata))

	// Output:
	// format=1 bytes=35
}

func ExampleDocument_MathExpressions() {
	source := []byte("Inline $x+1$.\n\n$$x^2$$\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		panic(err)
	}
	for _, expression := range document.MathExpressions() {
		payload, ok := expression.PayloadRange()
		if !ok {
			continue
		}
		value, _ := document.SourceRange(payload)
		fmt.Printf("style=%d payload=%s\n", expression.Style(), value)
	}

	// Output:
	// style=1 payload=x+1
	// style=3 payload=x^2
}

func ExampleDocumentBuilder_AppendMathBlock() {
	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendParagraphContent(marksplice.TextInline("Value "), marksplice.MathInline("x+1")); err != nil {
		panic(err)
	}
	if err := builder.AppendMathBlock("x^2+y^2"); err != nil {
		panic(err)
	}
	markdown, err := builder.Markdown()
	if err != nil {
		panic(err)
	}
	fmt.Print(string(markdown))

	// Output:
	// Value $x+1$
	//
	// $$x^2+y^2$$
}

func ExampleDocument_ComposeChanges() {
	source := []byte("# Old title\n\nOld body.\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		panic(err)
	}

	var headingID, paragraphID marksplice.NodeID
	for _, node := range document.Nodes() {
		switch node.Kind() {
		case marksplice.KindHeading:
			headingID = node.ID()
		case marksplice.KindParagraph:
			paragraphID = node.ID()
		}
	}
	rename, err := document.PrepareRenameHeading(headingID, []byte("New title"))
	if err != nil {
		panic(err)
	}
	replace, err := document.PrepareReplaceParagraph(paragraphID, []byte("New body."))
	if err != nil {
		panic(err)
	}
	combined, err := document.ComposeChanges(rename, replace)
	if err != nil {
		panic(err)
	}
	updated, err := combined.Apply(source)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(updated))

	// Output:
	// # New title
	//
	// New body.
}

func ExampleDocument_PrepareRenameHeading() {
	source := []byte("##  Old title  ##\n\nBody.\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		panic(err)
	}

	var heading marksplice.Heading
	for _, node := range document.Nodes() {
		if node.Kind() != marksplice.KindHeading {
			continue
		}
		var ok bool
		heading, ok = document.Heading(node.ID())
		if ok {
			break
		}
	}

	change, err := document.PrepareRenameHeading(heading.ID(), []byte("New title"))
	if err != nil {
		panic(err)
	}
	updated, err := change.Apply(source)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(updated))

	// Output:
	// ##  New title  ##
	//
	// Body.
}
