package marksplice_test

import (
	"fmt"

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
