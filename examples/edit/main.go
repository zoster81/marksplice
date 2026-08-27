package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/zoster81/marksplice"
)

const fixture = "examples/edit/release-plan.md"

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

	headingID := findHeading(doc, "Release Plan")
	paragraphID := findParagraph(doc, "The next beta focuses on documentation clarity.")
	taskID := findUncheckedTask(doc)
	cellID := findCell(doc, "Draft")

	rename, err := doc.PrepareRenameHeading(headingID, []byte("Release Readiness"))
	if err != nil {
		log.Fatalf("prepare heading rename: %v", err)
	}
	replaceParagraph, err := doc.PrepareReplaceParagraph(paragraphID, []byte("The next beta focuses on documentation UX and runnable examples."))
	if err != nil {
		log.Fatalf("prepare paragraph replacement: %v", err)
	}
	checkTask, err := doc.PrepareSetTaskChecked(taskID, true)
	if err != nil {
		log.Fatalf("prepare task update: %v", err)
	}
	updateCell, err := doc.PrepareReplaceTableCell(cellID, []byte("Ready"))
	if err != nil {
		log.Fatalf("prepare table-cell update: %v", err)
	}

	combined, err := doc.ComposeChanges(rename, replaceParagraph, checkTask, updateCell)
	if err != nil {
		log.Fatalf("compose changes: %v", err)
	}
	updated, err := combined.Apply(source)
	if err != nil {
		log.Fatalf("apply changes: %v", err)
	}

	preserved := bytes.Contains(updated, []byte("- [x] Keep source-preserving edits.")) &&
		bytes.Contains(updated, []byte("| API | Stable | Team |"))
	fmt.Printf("fixture-unchanged=%t unrelated-source-preserved=%t\n", fileStillMatches(fixture, source), preserved)
	fmt.Print(string(updated))
}

func findHeading(doc *marksplice.Document, text string) marksplice.NodeID {
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindHeading {
			continue
		}
		heading, ok := doc.Heading(node.ID())
		if !ok {
			continue
		}
		value, ok := doc.SourceRange(heading.Range())
		if ok && string(value) == text {
			return heading.ID()
		}
	}
	log.Fatalf("heading %q not found", text)
	return marksplice.NodeID{}
}

func findParagraph(doc *marksplice.Document, text string) marksplice.NodeID {
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindParagraph {
			continue
		}
		paragraph, ok := doc.Paragraph(node.ID())
		if !ok {
			continue
		}
		value, ok := doc.SourceRange(paragraph.Range())
		if ok && string(value) == text {
			return paragraph.ID()
		}
	}
	log.Fatalf("paragraph %q not found", text)
	return marksplice.NodeID{}
}

func findUncheckedTask(doc *marksplice.Document) marksplice.NodeID {
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindTask {
			continue
		}
		task, ok := doc.Task(node.ID())
		if ok && !task.Checked() {
			return task.ID()
		}
	}
	log.Fatal("unchecked task not found")
	return marksplice.NodeID{}
}

func findCell(doc *marksplice.Document, text string) marksplice.NodeID {
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindTableCell {
			continue
		}
		cell, ok := doc.TableCell(node.ID())
		if !ok {
			continue
		}
		value, ok := doc.SourceRange(cell.Range())
		if ok && strings.TrimSpace(string(value)) == text {
			return cell.ID()
		}
	}
	log.Fatalf("table cell %q not found", text)
	return marksplice.NodeID{}
}

func fileStillMatches(path string, original []byte) bool {
	current, err := os.ReadFile(path)
	return err == nil && bytes.Equal(current, original)
}
