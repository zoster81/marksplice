package main

import (
	"fmt"
	"log"
	"os"

	"github.com/zoster81/marksplice"
)

const fixture = "examples/inspect/project-guide.md"

func main() {
	log.SetFlags(0)

	source, err := os.ReadFile(fixture)
	if err != nil {
		log.Fatalf("read %s: %v", fixture, err)
	}
	doc, err := marksplice.Parse(source)
	if err != nil {
		log.Fatalf("parse %s: %v", fixture, err)
	}

	openTasks := 0
	taskCount := 0
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindTask {
			continue
		}
		task, ok := doc.Task(node.ID())
		if !ok {
			continue
		}
		taskCount++
		if !task.Checked() {
			openTasks++
		}
	}

	fmt.Printf("file=%s\n", fixture)
	fmt.Printf("sections=%d tasks=%d open=%d fenced=%d links=%d\n",
		len(doc.Sections()), taskCount, openTasks, len(doc.FencedBlocks()), len(doc.LinkRelationships()))

	for _, section := range doc.Sections() {
		heading, ok := doc.Heading(section.HeadingID())
		if !ok {
			continue
		}
		text, ok := doc.SourceRange(heading.Range())
		if !ok {
			log.Fatalf("read heading %s", heading.ID())
		}
		fmt.Printf("heading level=%d text=%s\n", section.Level(), text)
	}

	for _, block := range doc.FencedBlocks() {
		language, _ := block.Language()
		fmt.Printf("fence language=%s closed=%t\n", language, block.Closed())
	}
}
