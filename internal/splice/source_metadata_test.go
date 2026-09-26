package splice

import (
	"reflect"
	"testing"
)

func TestSourceMetadataPreservesValidationAndOwnership(t *testing.T) {
	doc, err := Parse([]byte("# heading\n\n```go\nfirst\nsecond\n```\n\n[^note]: first\n\n    second\n"))
	if err != nil {
		t.Fatal(err)
	}
	var fenceCount, footnoteCount int
	for _, node := range doc.Nodes() {
		full, info, language, fullOK := doc.FencedBlockSource(node.ID)
		metadata, metaInfo, metaLanguage, metadataOK := doc.FencedBlockMetadata(node.ID)
		if metadata.ContentRanges != nil || fullOK != metadataOK || info != metaInfo || language != metaLanguage {
			t.Fatalf("fenced metadata mismatch for %v", node.Kind)
		}
		if fullOK {
			fenceCount++
			if len(full.ContentRanges) != 2 {
				t.Fatalf("fenced content ranges = %v", full.ContentRanges)
			}
			expected := full.ContentRanges[0]
			full.ContentRanges[0].Start++
			again, _, _, _ := doc.FencedBlockSource(node.ID)
			if again.ContentRanges[0] != expected {
				t.Fatal("fenced source aliases document storage")
			}
		}
		full.ContentRanges = nil
		if !reflect.DeepEqual(full, metadata) {
			t.Fatalf("fenced scalar mapping mismatch for %v", node.Kind)
		}
		body, bodyOK := doc.FootnoteSource(node.ID)
		bodyMetadata, bodyMetadataOK := doc.FootnoteMetadata(node.ID)
		if bodyMetadata.BodyRanges != nil || bodyOK != bodyMetadataOK {
			t.Fatalf("footnote metadata mismatch for %v", node.Kind)
		}
		if bodyOK {
			footnoteCount++
			if len(body.BodyRanges) != 2 {
				t.Fatalf("footnote body ranges = %v", body.BodyRanges)
			}
			expected := body.BodyRanges[0]
			body.BodyRanges[0].Start++
			again, _ := doc.FootnoteSource(node.ID)
			if again.BodyRanges[0] != expected {
				t.Fatal("footnote source aliases document storage")
			}
		}
		body.BodyRanges = nil
		if !reflect.DeepEqual(body, bodyMetadata) {
			t.Fatalf("footnote scalar mapping mismatch for %v", node.Kind)
		}
	}
	if fenceCount != 1 || footnoteCount != 1 {
		t.Fatalf("tested fences=%d, footnotes=%d", fenceCount, footnoteCount)
	}
	for _, document := range []*Document{nil, {}, doc} {
		if _, _, _, ok := document.FencedBlockMetadata("missing"); ok {
			t.Fatal("fenced metadata accepted missing target")
		}
		if _, ok := document.FootnoteMetadata("missing"); ok {
			t.Fatal("footnote metadata accepted missing target")
		}
	}
}
