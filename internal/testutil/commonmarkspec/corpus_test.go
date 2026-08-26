package commonmarkspec

import (
	"os"
	"testing"
)

func TestPublishedCommonMark0312Corpus(t *testing.T) {
	path := os.Getenv("MARKSPLICE_COMMONMARK_SPEC_HTML")
	if path == "" {
		t.Skip("MARKSPLICE_COMMONMARK_SPEC_HTML is not set")
	}
	cases, err := LoadPublished(path)
	if err != nil {
		t.Fatalf("LoadPublished() error = %v", err)
	}
	if len(cases) != 652 {
		t.Fatalf("published CommonMark example count = %d, want 652", len(cases))
	}
	if cases[0].Number != 1 || cases[0].Section != "tabs" {
		t.Fatalf("first case = %+v, want example 1 in tabs", cases[0])
	}
	if cases[len(cases)-1].Number != 652 {
		t.Fatalf("last example number = %d, want 652", cases[len(cases)-1].Number)
	}
}
