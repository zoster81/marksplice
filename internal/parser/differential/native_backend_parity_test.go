package differential

import (
	"os"
	"slices"
	"testing"

	goldmarkparser "github.com/zoster81/marksplice/internal/parser/goldmark"
	nativeparser "github.com/zoster81/marksplice/internal/parser/native"
	"github.com/zoster81/marksplice/internal/testutil/gfmspec"
)

func TestNativeBackendMatchesPublishedGFMDifferentialCorpus(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}

	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	stats := gfmspec.Summarize(cases)
	if stats != (gfmspec.Stats{Total: 677, Core: 649, Table: 8, TaskList: 2, Strikethrough: 3, Autolink: 14, TagFilter: 1}) {
		t.Fatalf("unexpected published GFM corpus shape: %+v", stats)
	}

	harness := Harness{Oracle: goldmarkparser.New(), Candidate: nativeparser.New()}
	compared := 0
	for _, case_ := range cases {
		if slices.Contains(case_.Extensions, "tagfilter") {
			continue
		}
		source := []byte(case_.Markdown)
		if err := harness.CompareDocument(source); err != nil {
			want, wantErr := harness.Oracle.ParseDocument(source)
			got, gotErr := harness.Candidate.ParseDocument(source)
			t.Fatalf("CompareDocument(example %d) error = %v\nsource=%q\nGoldmark error=%v nodes=%#v\nnative error=%v nodes=%#v", case_.Number, err, source, wantErr, want.Nodes, gotErr, got.Nodes)
		}
		compared++
	}
	if compared != 676 {
		t.Fatalf("compared published GFM examples = %d, want 676", compared)
	}
}
