package main

import (
	"fmt"
	"log"
	"os"

	"github.com/zoster81/marksplice"
	"github.com/zoster81/marksplice/workspacefs"
)

const workspaceRoot = "examples/workspace/docs"

func main() {
	log.SetFlags(0)

	workspace, err := workspacefs.Scan(os.DirFS("."), workspaceRoot, workspacefs.DefaultOptions())
	if err != nil {
		log.Fatalf("scan workspace: %v", err)
	}
	graph, err := workspace.BuildGraph()
	if err != nil {
		log.Fatalf("build graph: %v", err)
	}

	backlinks, _ := graph.Backlinks("configuration.md")
	reachable, _ := graph.ReachableFrom("README.md")
	fmt.Printf("documents=%d edges=%d configuration-backlinks=%d reachable-from-readme=%d\n",
		len(graph.DocumentKeys()), len(graph.Edges()), len(backlinks), len(reachable))

	report, err := workspace.Validate(marksplice.WorkspaceValidationOptions{
		Roots: []marksplice.DocumentKey{"README.md"},
	})
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
