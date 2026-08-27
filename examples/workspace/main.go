package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/zoster81/marksplice"
)

var workspaceFiles = []struct {
	key  marksplice.DocumentKey
	path string
}{
	{key: "index", path: "examples/workspace/docs/README.md"},
	{key: "getting-started", path: "examples/workspace/docs/getting-started.md"},
	{key: "configuration", path: "examples/workspace/docs/configuration.md"},
	{key: "troubleshooting", path: "examples/workspace/docs/troubleshooting.md"},
}

func main() {
	log.SetFlags(0)

	documents, targetByName := loadWorkspace()
	graph, err := marksplice.BuildDocumentGraph(documents, graphResolver(targetByName))
	if err != nil {
		log.Fatalf("build graph: %v", err)
	}

	backlinks, _ := graph.Backlinks("configuration")
	reachable, _ := graph.ReachableFrom("index")
	fmt.Printf("documents=%d edges=%d configuration-backlinks=%d reachable-from-index=%d\n",
		len(graph.DocumentKeys()), len(graph.Edges()), len(backlinks), len(reachable))

	report, err := marksplice.ValidateWorkspace(
		documents,
		workspaceResolver(targetByName),
		marksplice.WorkspaceValidationOptions{Roots: []marksplice.DocumentKey{"index"}},
	)
	if err != nil {
		log.Fatalf("validate workspace: %v", err)
	}
	for _, diagnostic := range report.Diagnostics() {
		switch diagnostic.Kind() {
		case marksplice.WorkspaceDiagnosticOrphanDocument:
			target, _ := diagnostic.TargetDocument()
			fmt.Printf("diagnostic=orphan-document target=%s\n", target)
		case marksplice.WorkspaceDiagnosticMissingDocument:
			target, _ := diagnostic.TargetDocument()
			fmt.Printf("diagnostic=missing-document target=%s\n", target)
		case marksplice.WorkspaceDiagnosticMissingFragment:
			fragment, _ := diagnostic.Fragment()
			fmt.Printf("diagnostic=missing-fragment fragment=%s\n", fragment)
		}
	}
}

func loadWorkspace() ([]marksplice.GraphDocument, map[string]marksplice.DocumentKey) {
	documents := make([]marksplice.GraphDocument, 0, len(workspaceFiles))
	targetByName := make(map[string]marksplice.DocumentKey, len(workspaceFiles))
	for _, item := range workspaceFiles {
		source, err := os.ReadFile(item.path)
		if err != nil {
			log.Fatalf("read %s: %v", item.path, err)
		}
		doc, err := marksplice.Parse(source)
		if err != nil {
			log.Fatalf("parse %s: %v", item.path, err)
		}
		documents = append(documents, marksplice.GraphDocument{Key: item.key, Document: doc})
		targetByName[filepath.Base(item.path)] = item.key
	}
	return documents, targetByName
}

func graphResolver(targetByName map[string]marksplice.DocumentKey) marksplice.DocumentResolver {
	return func(_ marksplice.DocumentKey, relationship marksplice.LinkRelationship) (marksplice.DocumentResolution, bool) {
		name, fragment := splitDestination(relationship.Destination())
		target, ok := targetByName[name]
		if !ok {
			return marksplice.DocumentResolution{}, false
		}
		return marksplice.DocumentResolution{Target: target, Fragment: fragment}, true
	}
}

func workspaceResolver(targetByName map[string]marksplice.DocumentKey) marksplice.WorkspaceResolver {
	return func(_ marksplice.DocumentKey, relationship marksplice.LinkRelationship) marksplice.WorkspaceResolution {
		name, fragment := splitDestination(relationship.Destination())
		if target, ok := targetByName[name]; ok {
			return marksplice.WorkspaceResolution{
				Kind:     marksplice.WorkspaceResolutionResolved,
				Target:   target,
				Fragment: fragment,
			}
		}
		return marksplice.WorkspaceResolution{Kind: marksplice.WorkspaceResolutionIgnore}
	}
}

func splitDestination(destination string) (string, string) {
	if index := strings.IndexByte(destination, '#'); index >= 0 {
		return destination[:index], destination[index:]
	}
	return destination, ""
}
