package main

import (
	"fmt"
	"log"
	"os"

	"github.com/zoster81/marksplice"
)

const fixture = "examples/render/page.md"

func main() {
	log.SetFlags(0)

	source, err := os.ReadFile(fixture)
	if err != nil {
		log.Fatalf("read %s: %v", fixture, err)
	}
	document, err := marksplice.Parse(source)
	if err != nil {
		log.Fatalf("parse %s: %v", fixture, err)
	}
	if len(os.Args) > 2 {
		log.Fatalf("usage: go run ./examples/render [--map|--markdown]")
	}
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "--map":
			renderSourceMap(document, source)
		case "--markdown":
			if err := document.RenderCanonicalMarkdown(os.Stdout); err != nil {
				log.Fatalf("render canonical Markdown %s: %v", fixture, err)
			}
		default:
			log.Fatalf("usage: go run ./examples/render [--map|--markdown]")
		}
		return
	}
	if err := document.RenderHTMLDocument(os.Stdout, marksplice.DefaultHTMLDocumentOptions()); err != nil {
		log.Fatalf("render %s: %v", fixture, err)
	}
}

func renderSourceMap(document *marksplice.Document, source []byte) {
	output, mappings, err := document.HTMLDocumentWithSourceMap(marksplice.DefaultHTMLDocumentOptions())
	if err != nil {
		log.Fatalf("render source map %s: %v", fixture, err)
	}
	for _, mapping := range mappings {
		sourceRange := mapping.SourceRange()
		outputRange := mapping.OutputRange()
		fmt.Printf(
			"source[%d:%d] -> html[%d:%d]  %q -> %q\n",
			sourceRange.Start,
			sourceRange.End,
			outputRange.Start,
			outputRange.End,
			source[sourceRange.Start:sourceRange.End],
			output[outputRange.Start:outputRange.End],
		)
	}
}
