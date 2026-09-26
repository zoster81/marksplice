package splice

import (
	"reflect"
	"testing"

	"github.com/zoster81/marksplice/internal/source"
)

func TestAlertMetadataPreservesSourceValidationAndOwnership(t *testing.T) {
	document, err := Parse([]byte("> [!NOTE]\r\n> first\r\n>\r\n> last\r\n\nparagraph\n"))
	if err != nil {
		t.Fatal(err)
	}
	var alertID NodeID
	for _, node := range document.Nodes() {
		metadata, ok := document.AlertMetadata(node.ID)
		if node.Kind != KindBlockquote {
			if ok {
				t.Fatal("non-blockquote has alert metadata")
			}
			continue
		}
		alertID = node.ID
		mapping, mapped := document.BlockquoteSource(node.ID)
		if !ok || !mapped || metadata.Kind != source.AlertNote || metadata.Range != mapping.LineRange || metadata.MarkerRange != mapping.ContentRanges[0] {
			t.Fatalf("metadata does not match source mapping: %+v", metadata)
		}
		body, ok := document.AlertBodyRanges(node.ID)
		if !ok || len(body) != 3 || !reflect.DeepEqual(body, mapping.ContentRanges[1:]) {
			t.Fatalf("body ranges = %v/%v", body, ok)
		}
		body[0] = Range{}
		again, ok := document.AlertBodyRanges(node.ID)
		if !ok || !reflect.DeepEqual(again, mapping.ContentRanges[1:]) {
			t.Fatal("body ranges alias the snapshot")
		}
	}
	if alertID == "" {
		t.Fatal("missing test alert")
	}
	for _, value := range []*Document{nil, {}, document} {
		if _, ok := value.AlertMetadata("missing"); ok {
			t.Fatal("missing node has metadata")
		}
		if _, ok := value.AlertBodyRanges("missing"); ok {
			t.Fatal("missing node has body ranges")
		}
	}
	// A source detail that no longer proves the complete blockquote must be
	// rejected even when its first line still contains a valid alert marker.
	document.blockquoteSources[0].ContentRanges[1].End = len(document.source) + 1
	if _, ok := document.AlertMetadata(alertID); ok {
		t.Fatal("invalid source proof has metadata")
	}
	if _, ok := document.AlertBodyRanges(alertID); ok {
		t.Fatal("invalid source proof has body ranges")
	}
}
