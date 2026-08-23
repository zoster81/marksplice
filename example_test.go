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
