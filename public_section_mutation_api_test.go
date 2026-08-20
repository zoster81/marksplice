package marksplice_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicRemoveSectionPreservesBytesOutsideCompleteSubtree(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\r\nintro π\r\n\r\n## Remove\r\nremove body\r\n\r\n### Deep\r\ndeep body\r\n\r\n## Keep\r\nkeep body\r\n\r\n# Tail\r\ntail\r\n")
	want := []byte("# Root\r\nintro π\r\n\r\n## Keep\r\nkeep body\r\n\r\n# Tail\r\ntail\r\n")

	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sections := publicSectionsByHeadingText(t, doc, source, doc.Sections())
	remove := sections["Remove"]
	if remove.HeadingID().String() == "" {
		t.Fatal("Remove section not found")
	}

	prefix := append([]byte(nil), source[:remove.Range().Start]...)
	suffix := append([]byte(nil), source[remove.Range().End:]...)
	change, err := doc.PrepareRemoveSection(remove.HeadingID())
	if err != nil {
		t.Fatalf("PrepareRemoveSection() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
	if !bytes.Equal(got[:len(prefix)], prefix) || !bytes.Equal(got[len(prefix):], suffix) {
		t.Fatal("section removal modified bytes outside Section.Range()")
	}

	stale := append([]byte(nil), source...)
	stale[0] = '!'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}

func TestPublicRemoveSectionSupportsRootAndEOFSections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  []byte
		heading string
		want    []byte
	}{
		{
			name:    "root section keeps preamble and following root",
			source:  []byte("preamble\n\n# One\nbody\n\n## Child\nchild\n\n# Two\ntail\n"),
			heading: "One",
			want:    []byte("preamble\n\n# Two\ntail\n"),
		},
		{
			name:    "last section removes through EOF",
			source:  []byte("# One\nbody\n\n# Last\nlast\n"),
			heading: "Last",
			want:    []byte("# One\nbody\n\n"),
		},
		{
			name:    "isolated CR boundaries remain byte exact",
			source:  []byte("# One\rbody\r# Remove\rgone\r# Two\rtail\r"),
			heading: "Remove",
			want:    []byte("# One\rbody\r# Two\rtail\r"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			section := publicSectionsByHeadingText(t, doc, tt.source, doc.Sections())[tt.heading]
			if section.HeadingID().String() == "" {
				t.Fatalf("section %q not found", tt.heading)
			}
			change, err := doc.PrepareRemoveSection(section.HeadingID())
			if err != nil {
				t.Fatalf("PrepareRemoveSection() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPublicRemoveSectionFailsClosedWhenJoinChangesSurvivingHeading(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\nintro\n## Remove\ngone\nNext\n----\ntail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sections := publicSectionsByHeadingText(t, doc, source, doc.Sections())
	remove := sections["Remove"]
	if remove.HeadingID().String() == "" {
		t.Fatal("Remove section not found")
	}
	if _, err := doc.PrepareRemoveSection(remove.HeadingID()); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareRemoveSection() error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPublicRemoveSectionPreservesTargetErrors(t *testing.T) {
	t.Parallel()

	source := []byte("# One\nbody\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if _, err := doc.PrepareRemoveSection(marksplice.NodeID{}); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareRemoveSection(zero ID) error = %v, want ErrNodeNotFound", err)
	}

	var paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraph = node
			break
		}
	}
	if paragraph.ID().String() == "" {
		t.Fatal("paragraph not found")
	}
	if _, err := doc.PrepareRemoveSection(paragraph.ID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareRemoveSection(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}
