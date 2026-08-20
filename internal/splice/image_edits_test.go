package splice

import (
	"bytes"
	"errors"
	"testing"
)

func TestImageMappingPersistsAtParseTimeAndUnsupportedShapesStayNonEditable(t *testing.T) {
	t.Parallel()

	source := []byte("![alt](<old path> \"title\")\n\n![ref][id]\n\n[id]: old/ref\n\n![empty]()\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	images := nodesOfKind(doc.Nodes(), KindImage)
	if len(images) != 3 {
		t.Fatalf("image count = %d, want 3; nodes = %+v", len(images), doc.Nodes())
	}

	var editable Node
	unsupported := 0
	for _, image := range images {
		if image.Editable {
			editable = image
			continue
		}
		unsupported++
		if image.ImageSource.DestinationRange != (Range{}) {
			t.Fatalf("unsupported image retained destination mapping %v", image.ImageSource.DestinationRange)
		}
	}
	if editable.ID == "" || unsupported != 2 {
		t.Fatalf("editable image ID = %q unsupported = %d, want non-empty/2", editable.ID, unsupported)
	}
	if editable.ImageSource.DestinationRange != editable.ContentRange || !editable.ImageSource.AngleDestination || !editable.ImageSource.HasTitle {
		t.Fatalf("editable image mapping = %+v content = %v", editable.ImageSource, editable.ContentRange)
	}
	if got := string(source[editable.ContentRange.Start:editable.ContentRange.End]); got != "old path" {
		t.Fatalf("image destination bytes = %q, want old path", got)
	}
	for _, image := range images {
		if image.Editable {
			continue
		}
		if _, err := doc.PrepareReplaceImageDestination(image.ID, []byte("new/path")); !errors.Is(err, ErrInvalidTargetKind) {
			t.Fatalf("unsupported image mutation error = %v, want ErrInvalidTargetKind", err)
		}
	}
}

func TestReplaceSimpleImageDestinationPreservesSourceSyntax(t *testing.T) {
	t.Parallel()

	source := []byte("before ![alt](  <old path> 'title' ) after\r\n")
	want := []byte("before ![alt](  <new path> 'title' ) after\r\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	images := nodesOfKind(doc.Nodes(), KindImage)
	if len(images) != 1 || !images[0].Editable {
		t.Fatalf("images = %+v, want one editable image", images)
	}
	change, err := doc.PrepareReplaceImageDestination(images[0].ID, []byte("new path"))
	if err != nil {
		t.Fatalf("PrepareReplaceImageDestination() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}
