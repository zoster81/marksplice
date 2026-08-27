package main

import (
	"fmt"
	"log"
	"os"

	"github.com/zoster81/marksplice"
)

const fixture = "examples/query/project-guide.md"

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

	sectionRange, ok := sectionRangeByHeading(doc, "Current sprint")
	if !ok {
		log.Fatal("Current sprint section not found")
	}
	matches, err := doc.QueryNodes(marksplice.NodeQuery{
		Kinds:  []marksplice.Kind{marksplice.KindTask},
		Within: &sectionRange,
		Limit:  20,
	})
	if err != nil {
		log.Fatalf("query tasks: %v", err)
	}

	unfinished := 0
	for _, match := range matches {
		task, ok := doc.Task(match.Node().ID())
		if ok && !task.Checked() {
			unfinished++
		}
	}
	fmt.Printf("section=Current sprint tasks=%d unfinished=%d\n", len(matches), unfinished)
}

func sectionRangeByHeading(doc *marksplice.Document, text string) (marksplice.Range, bool) {
	for _, section := range doc.Sections() {
		heading, ok := doc.Heading(section.HeadingID())
		if !ok {
			continue
		}
		value, ok := doc.SourceRange(heading.Range())
		if ok && string(value) == text {
			return section.Range(), true
		}
	}
	return marksplice.Range{}, false
}
