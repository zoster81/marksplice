package workspacefs_test

import (
	"errors"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/zoster81/marksplice"
	"github.com/zoster81/marksplice/workspacefs"
)

func TestScanDiscoversNestedMarkdownDeterministicallyAndBuildsExistingGraph(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{
		"site/README.md":           markdown("# Home\n\n[guide](docs/guide.md#guide) [local](#home) [external](https://example.com)\n"),
		"site/docs/guide.md":       markdown("# Guide\n\n[home](../README.md)\n"),
		"site/docs/notes.markdown": markdown("# Notes\n"),
		"site/docs/asset.txt":      {Data: []byte("not markdown")},
	}

	workspace, err := workspacefs.Scan(files, "site", testOptions())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if got := documentKeys(workspace.Documents()); !reflect.DeepEqual(got, []marksplice.DocumentKey{"README.md", "docs/guide.md", "docs/notes.markdown"}) {
		t.Fatalf("document keys = %#v", got)
	}

	graph, err := workspace.BuildGraph()
	if err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}
	if got := graph.DocumentKeys(); !reflect.DeepEqual(got, []marksplice.DocumentKey{"README.md", "docs/guide.md", "docs/notes.markdown"}) {
		t.Fatalf("graph document keys = %#v", got)
	}
	if edges := graph.Edges(); len(edges) != 3 {
		t.Fatalf("graph edges = %d, want 3 (guide, local fragment, home)", len(edges))
	}
	if got, ok := graph.ReachableFrom("README.md"); !ok || !reflect.DeepEqual(got, []marksplice.DocumentKey{"docs/guide.md"}) {
		t.Fatalf("ReachableFrom(README.md) = %#v/%v", got, ok)
	}

	report, err := workspace.Validate(marksplice.WorkspaceValidationOptions{Roots: []marksplice.DocumentKey{"README.md"}})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if diagnostics := report.Diagnostics(); len(diagnostics) != 1 || diagnostics[0].Kind() != marksplice.WorkspaceDiagnosticOrphanDocument {
		t.Fatalf("diagnostics = %#v, want one orphan document", diagnostics)
	}
	if target, ok := report.Diagnostics()[0].TargetDocument(); !ok || target != "docs/notes.markdown" {
		t.Fatalf("orphan target = %q/%v", target, ok)
	}
}

func TestFollowTraversesLocalMarkdownOnceAcrossCyclesAndReportsMissingTargets(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{
		"README.md": markdown("# Root\n\n[a](docs/a.md) [missing](missing.md?download=1#part) [external](https://example.com/x.md)\n"),
		"docs/a.md": markdown("# A\n\n[root](../README.md) [b](b.md)\n"),
		"docs/b.md": markdown("# B\n\n[local](#b) [mail](mailto:test@example.com)\n"),
	}

	workspace, err := workspacefs.Follow(files, ".", []string{"README.md"}, testOptions())
	if err != nil {
		t.Fatalf("Follow() error = %v", err)
	}
	if got := documentKeys(workspace.Documents()); !reflect.DeepEqual(got, []marksplice.DocumentKey{"README.md", "docs/a.md", "docs/b.md"}) {
		t.Fatalf("document keys = %#v", got)
	}

	graph, err := workspace.BuildGraph()
	if err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}
	if edges := graph.Edges(); len(edges) != 4 {
		t.Fatalf("graph edges = %d, want 4 local/resolved edges", len(edges))
	}
	if got, ok := graph.ReachableFrom("README.md"); !ok || !reflect.DeepEqual(got, []marksplice.DocumentKey{"docs/a.md", "docs/b.md"}) {
		t.Fatalf("ReachableFrom(README.md) = %#v/%v", got, ok)
	}

	report, err := workspace.Validate(marksplice.WorkspaceValidationOptions{Roots: []marksplice.DocumentKey{"README.md"}})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	diagnostics := report.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Kind() != marksplice.WorkspaceDiagnosticMissingDocument {
		t.Fatalf("diagnostics = %#v, want one missing document", diagnostics)
	}
	if target, ok := diagnostics[0].TargetDocument(); !ok || target != "missing.md" {
		t.Fatalf("missing target = %q/%v", target, ok)
	}
}

func TestBudgetsFailClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		files   fstest.MapFS
		options workspacefs.Options
	}{
		{
			name: "documents",
			files: fstest.MapFS{
				"a.md": markdown("# A\n"),
				"b.md": markdown("# B\n"),
			},
			options: optionsWith(workspacefs.Limits{MaxDocuments: 1, MaxBytes: 1024, MaxDepth: 8, MaxRelationships: 8}),
		},
		{
			name:    "bytes",
			files:   fstest.MapFS{"a.md": markdown("# 123456789\n")},
			options: optionsWith(workspacefs.Limits{MaxDocuments: 8, MaxBytes: 4, MaxDepth: 8, MaxRelationships: 8}),
		},
		{
			name:    "depth",
			files:   fstest.MapFS{"nested/a.md": markdown("# A\n")},
			options: optionsWith(workspacefs.Limits{MaxDocuments: 8, MaxBytes: 1024, MaxDepth: 0, MaxRelationships: 8}),
		},
		{
			name: "relationships",
			files: fstest.MapFS{
				"a.md": markdown("# A\n\n[b](b.md) [c](c.md)\n"),
				"b.md": markdown("# B\n"),
				"c.md": markdown("# C\n"),
			},
			options: optionsWith(workspacefs.Limits{MaxDocuments: 8, MaxBytes: 4096, MaxDepth: 8, MaxRelationships: 1}),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := workspacefs.Scan(test.files, ".", test.options); !errors.Is(err, workspacefs.ErrBudgetExceeded) {
				t.Fatalf("Scan() error = %v, want ErrBudgetExceeded", err)
			}
		})
	}

	followFiles := fstest.MapFS{
		"a.md": markdown("# A\n\n[b](b.md)\n"),
		"b.md": markdown("# B\n"),
	}
	followOptions := optionsWith(workspacefs.Limits{MaxDocuments: 8, MaxBytes: 4096, MaxDepth: 0, MaxRelationships: 8})
	if _, err := workspacefs.Follow(followFiles, ".", []string{"a.md"}, followOptions); !errors.Is(err, workspacefs.ErrBudgetExceeded) {
		t.Fatalf("Follow() depth error = %v, want ErrBudgetExceeded", err)
	}
}

func TestHostileOrNonLocalDestinationsAreNotFollowedAndMalformedInputsFail(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{
		"root.md":   markdown("# Root\n\n[escape](../inside.md) [absolute](/inside.md) [protocol](//example.com/inside.md) [scheme](https://example.com/inside.md) [encoded](%2e%2e/inside.md) [backslash](docs\\inside.md)\n"),
		"inside.md": markdown("# Inside\n"),
	}
	workspace, err := workspacefs.Follow(files, ".", []string{"root.md"}, testOptions())
	if err != nil {
		t.Fatalf("Follow() error = %v", err)
	}
	if got := documentKeys(workspace.Documents()); !reflect.DeepEqual(got, []marksplice.DocumentKey{"root.md"}) {
		t.Fatalf("non-local destinations were followed: %#v", got)
	}
	report, err := workspace.Validate(marksplice.WorkspaceValidationOptions{Roots: []marksplice.DocumentKey{"root.md"}})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if diagnostics := report.Diagnostics(); len(diagnostics) != 0 {
		t.Fatalf("non-local destinations produced diagnostics: %#v", diagnostics)
	}

	if _, err := workspacefs.Scan(files, "../escape", testOptions()); !errors.Is(err, workspacefs.ErrInvalidInput) {
		t.Fatalf("Scan(invalid root) error = %v, want ErrInvalidInput", err)
	}
	if _, err := workspacefs.Follow(files, ".", []string{"../escape.md"}, testOptions()); !errors.Is(err, workspacefs.ErrInvalidInput) {
		t.Fatalf("Follow(invalid entry) error = %v, want ErrInvalidInput", err)
	}
	if _, err := workspacefs.Scan(files, ".", workspacefs.Options{}); !errors.Is(err, workspacefs.ErrInvalidInput) {
		t.Fatalf("Scan(zero options) error = %v, want ErrInvalidInput", err)
	}
}

func TestFollowResolvesNormalizedURIPathsQueriesAndFragments(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{
		"README.md":        markdown("# Root\n\n[dot](./docs/guide.md) [space](docs/My%20Guide.md) [unicode](docs/%E2%9C%93.md) [query](docs/query.md?view=%ZZ#part%2Done) [colon](docs/a:b.md)\n"),
		"docs/guide.md":    markdown("# Guide\n\n[parent](../README.md) [escape](../../outside.md)\n"),
		"docs/My Guide.md": markdown("# Spaced\n"),
		"docs/✓.md":        markdown("# Unicode\n"),
		"docs/query.md":    markdown("# Part One\n"),
		"docs/a:b.md":      markdown("# Colon\n"),
	}

	workspace, err := workspacefs.Follow(files, ".", []string{"README.md"}, testOptions())
	if err != nil {
		t.Fatalf("Follow() error = %v", err)
	}
	wantKeys := []marksplice.DocumentKey{"README.md", "docs/guide.md", "docs/My Guide.md", "docs/✓.md", "docs/query.md", "docs/a:b.md"}
	if got := documentKeys(workspace.Documents()); !reflect.DeepEqual(got, wantKeys) {
		t.Fatalf("document keys = %#v, want %#v", got, wantKeys)
	}

	graph, err := workspace.BuildGraph()
	if err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}
	edges := graph.Edges()
	if len(edges) != 6 {
		t.Fatalf("graph edges = %d, want 6", len(edges))
	}
	var queryEdge *marksplice.GraphEdge
	for index := range edges {
		if edges[index].Relationship().Destination() == "docs/query.md?view=%ZZ#part%2Done" {
			queryEdge = &edges[index]
			break
		}
	}
	if queryEdge == nil {
		t.Fatal("query-bearing relationship was not resolved")
		return
	}
	if got := queryEdge.TargetDocument(); got != "docs/query.md" {
		t.Fatalf("query target = %q, want docs/query.md", got)
	}
	if fragment, ok := queryEdge.Fragment(); !ok || fragment != "#part%2Done" {
		t.Fatalf("query fragment = %q/%v, want #part%%2Done/true", fragment, ok)
	}
	if _, ok := queryEdge.FragmentTarget(); !ok {
		t.Fatal("query fragment did not resolve in target document")
	}
}

func TestFollowRejectsEncodedTraversalAndSeparatorsWithoutDoubleDecoding(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{
		"root.md":          markdown("# Root\n\n[parent](%2e%2e/inside.md) [mixed](%2e./inside.md) [slash](docs%2Finside.md) [backslash](docs%5Cinside.md) [malformed](docs/%ZZ.md) [nul](docs/%00.md) [invalid-utf8](docs/%FF.md) [double](%252e%252e/inside.md) [duplicate](docs//inside.md) [absolute](/inside.md) [protocol](//example.com/inside.md) [scheme](https://example.com/inside.md) [drive](C:/inside.md) [raw-backslash](docs\\inside.md) [directory](docs/) [extensionless](docs/inside)\n"),
		"inside.md":        markdown("# Inside\n"),
		"docs/inside.md":   markdown("# Docs Inside\n"),
		"%2e%2e/inside.md": markdown("# Literal Percent Directory\n"),
	}

	workspace, err := workspacefs.Follow(files, ".", []string{"root.md"}, testOptions())
	if err != nil {
		t.Fatalf("Follow() error = %v", err)
	}
	wantKeys := []marksplice.DocumentKey{"root.md", "%2e%2e/inside.md"}
	if got := documentKeys(workspace.Documents()); !reflect.DeepEqual(got, wantKeys) {
		t.Fatalf("document keys = %#v, want single-decode result %#v", got, wantKeys)
	}
	report, err := workspace.Validate(marksplice.WorkspaceValidationOptions{Roots: []marksplice.DocumentKey{"root.md"}})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if diagnostics := report.Diagnostics(); len(diagnostics) != 0 {
		t.Fatalf("ignored hostile/non-filesystem destinations produced diagnostics: %#v", diagnostics)
	}
}

func TestResolutionPreservesCallerFilesystemCaseSemantics(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{
		"root.md":       markdown("# Root\n\n[case](docs/guide.md)\n"),
		"docs/Guide.md": markdown("# Guide\n"),
	}
	workspace, err := workspacefs.Follow(files, ".", []string{"root.md"}, testOptions())
	if err != nil {
		t.Fatalf("Follow() error = %v", err)
	}
	if got := documentKeys(workspace.Documents()); !reflect.DeepEqual(got, []marksplice.DocumentKey{"root.md"}) {
		t.Fatalf("case-mismatched path was followed: %#v", got)
	}
	report, err := workspace.Validate(marksplice.WorkspaceValidationOptions{Roots: []marksplice.DocumentKey{"root.md"}})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	diagnostics := report.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Kind() != marksplice.WorkspaceDiagnosticMissingDocument {
		t.Fatalf("diagnostics = %#v, want one missing document", diagnostics)
	}
	if target, ok := diagnostics[0].TargetDocument(); !ok || target != "docs/guide.md" {
		t.Fatalf("missing target = %q/%v", target, ok)
	}
}

func TestWorkspaceOwnershipDefaultsAndEntryOrdering(t *testing.T) {
	t.Parallel()

	defaults := workspacefs.DefaultOptions().Limits
	if defaults.MaxDocuments <= 0 || defaults.MaxBytes <= 0 || defaults.MaxDepth < 0 || defaults.MaxRelationships <= 0 {
		t.Fatalf("DefaultOptions() returned non-finite limits: %+v", defaults)
	}

	files := fstest.MapFS{
		"a.md": markdown("# A\n"),
		"b.md": markdown("# B\n"),
	}
	workspace, err := workspacefs.Follow(files, ".", []string{"b.md", "a.md", "b.md"}, testOptions())
	if err != nil {
		t.Fatalf("Follow() error = %v", err)
	}
	if got := documentKeys(workspace.Documents()); !reflect.DeepEqual(got, []marksplice.DocumentKey{"a.md", "b.md"}) {
		t.Fatalf("entry ordering = %#v", got)
	}
	documents := workspace.Documents()
	documents[0].Key = "mutated"
	if got := documentKeys(workspace.Documents()); !reflect.DeepEqual(got, []marksplice.DocumentKey{"a.md", "b.md"}) {
		t.Fatalf("Documents() caller mutation leaked: %#v", got)
	}

	var nilWorkspace *workspacefs.Workspace
	if nilWorkspace.Documents() != nil {
		t.Fatal("nil Workspace.Documents() should return nil")
	}
	if _, err := nilWorkspace.BuildGraph(); !errors.Is(err, workspacefs.ErrInvalidInput) {
		t.Fatalf("nil BuildGraph() error = %v, want ErrInvalidInput", err)
	}
	if _, err := nilWorkspace.Validate(marksplice.WorkspaceValidationOptions{}); !errors.Is(err, workspacefs.ErrInvalidInput) {
		t.Fatalf("nil Validate() error = %v, want ErrInvalidInput", err)
	}
}

func markdown(source string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(source)}
}

func testOptions() workspacefs.Options {
	return optionsWith(workspacefs.Limits{
		MaxDocuments:     64,
		MaxBytes:         1 << 20,
		MaxDepth:         16,
		MaxRelationships: 256,
	})
}

func optionsWith(limits workspacefs.Limits) workspacefs.Options {
	return workspacefs.Options{Limits: limits}
}

func documentKeys(documents []marksplice.GraphDocument) []marksplice.DocumentKey {
	result := make([]marksplice.DocumentKey, len(documents))
	for index, document := range documents {
		result[index] = document.Key
	}
	return result
}
