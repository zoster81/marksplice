package main

import (
	"fmt"
	"log"

	"github.com/zoster81/marksplice"
)

func main() {
	log.SetFlags(0)

	builder := marksplice.NewDocumentBuilder()
	must(builder.SetYAMLFrontMatter(
		marksplice.FrontMatterFieldInput{Key: "project", Value: "marksplice"},
		marksplice.FrontMatterFieldInput{Key: "status", Value: "draft"},
	))
	must(builder.AppendHeadingContent(1, marksplice.TextInline("Release brief")))
	must(builder.AppendParagraphContent(
		marksplice.TextInline("Generated with "),
		marksplice.LinkInline("https://github.com/zoster81/marksplice", marksplice.TextInline("Marksplice")),
		marksplice.TextInline("."),
	))
	must(builder.AppendUnorderedTaskList(
		marksplice.TaskListItem{InlineGFM: "Review API changes", Checked: true},
		marksplice.TaskListItem{InlineGFM: "Publish migration notes", Checked: false},
	))
	must(builder.AppendTableWithAlignments(
		[]string{"Area", "Owner", "Status"},
		[]marksplice.TableAlignment{marksplice.TableAlignmentLeft, marksplice.TableAlignmentLeft, marksplice.TableAlignmentCenter},
		[]string{"Docs", "Team", "Ready"},
		[]string{"Release", "Maintainer", "Pending"},
	))
	must(builder.AppendFencedCode("go test ./...\ngo vet ./...", "sh"))

	source, err := builder.Markdown()
	if err != nil {
		log.Fatalf("render document: %v", err)
	}
	fmt.Print(string(source))
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
