package main

import (
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
	if err := document.RenderHTMLDocument(os.Stdout, marksplice.DefaultHTMLDocumentOptions()); err != nil {
		log.Fatalf("render %s: %v", fixture, err)
	}
}
