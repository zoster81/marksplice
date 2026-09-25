package workspacefs_test

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/zoster81/marksplice/workspacefs"
)

func BenchmarkWorkspaceLoadingScaling(b *testing.B) {
	for _, count := range []int{16, 64, 256} {
		files := make(fstest.MapFS, count)
		for i := 0; i < count; i++ {
			files[fmt.Sprintf("doc%d.md", i)] = &fstest.MapFile{
				Data: []byte(fmt.Sprintf("# Heading\n\n[Next](doc%d.md#heading) and *text*.\n", (i+1)%count)),
			}
		}
		options := workspacefs.DefaultOptions()
		options.Limits.MaxDepth = count
		for _, operation := range []string{"Scan", "Follow"} {
			b.Run(fmt.Sprintf("%dDocuments/%s", count, operation), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					var workspace *workspacefs.Workspace
					var err error
					if operation == "Scan" {
						workspace, err = workspacefs.Scan(files, ".", options)
					} else {
						workspace, err = workspacefs.Follow(files, ".", []string{"doc0.md"}, options)
					}
					if err != nil || workspace == nil {
						b.Fatalf("workspace loading failed: %v", err)
					}
				}
			})
		}
	}
}
