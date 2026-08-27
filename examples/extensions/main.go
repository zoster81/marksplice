package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/zoster81/marksplice"
)

const fixture = "examples/extensions/sample.md"

func main() {
	log.SetFlags(0)

	source, err := os.ReadFile(fixture)
	if err != nil {
		log.Fatalf("read %s: %v", fixture, err)
	}

	wiki := marksplice.Extension{
		ID:        "example.org/wikilink",
		Recognize: recognizeWikiLinks,
	}
	doc, err := marksplice.ParseWithOptions(source, marksplice.ParseOptions{
		Extensions: []marksplice.Extension{wiki},
		ExtensionLimits: marksplice.ExtensionLimits{
			MaxNodes:         32,
			MaxMetadataBytes: 4 << 10,
		},
	})
	if err != nil {
		log.Fatalf("parse with extension: %v", err)
	}

	for _, node := range doc.ExtensionNodes() {
		raw, ok := doc.SourceRange(node.Range())
		if !ok {
			log.Fatal("extension range is not readable")
		}
		target, _ := node.Attribute("target")
		fmt.Printf("kind=%s source=%s target=%s\n", node.Kind(), raw, target)
	}
}

func recognizeWikiLinks(source marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
	text := source.Text()
	matches := make([]marksplice.ExtensionMatch, 0, 4)
	for offset := 0; offset < len(text); {
		open := strings.Index(text[offset:], "[[")
		if open < 0 {
			break
		}
		start := offset + open
		close := strings.Index(text[start+2:], "]]")
		if close < 0 {
			break
		}
		end := start + 2 + close + 2
		matches = append(matches, marksplice.ExtensionMatch{
			Kind:  "wikilink",
			Range: marksplice.Range{Start: start, End: end},
			Attributes: []marksplice.ExtensionAttribute{{
				Name:  "target",
				Value: text[start+2 : end-2],
			}},
		})
		offset = end
	}
	return matches, nil
}
