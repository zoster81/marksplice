package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestDirectLinkAndImageTitleLifecyclePreservesUnownedTrivia(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
		kind   marksplice.Kind
		add    []byte
		added  []byte
		final  []byte
	}{
		{
			name:   "link with angle destination and authored spacing",
			source: []byte("[label](<dest path>   )\n"),
			kind:   marksplice.KindInlineLink,
			add:    []byte("new title"),
			added:  []byte("[label](<dest path>    \"new title\")\n"),
			final:  []byte("[label](<dest path>    )\n"),
		},
		{
			name:   "image CRLF",
			source: []byte("![alt](image.png)\r\n"),
			kind:   marksplice.KindImage,
			add:    []byte("caption π"),
			added:  []byte("![alt](image.png \"caption π\")\r\n"),
			final:  []byte("![alt](image.png )\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatal(err)
			}
			node := findNodeOfKind(t, doc, tt.kind)
			var addChange marksplice.ChangeSet
			switch tt.kind {
			case marksplice.KindInlineLink:
				addChange, err = doc.PrepareAddInlineLinkTitle(node.ID(), tt.add)
			case marksplice.KindImage:
				addChange, err = doc.PrepareAddImageTitle(node.ID(), tt.add)
			}
			if err != nil {
				t.Fatalf("add title error = %v", err)
			}
			added, err := addChange.Apply(tt.source)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(added, tt.added) {
				t.Fatalf("added = %q, want %q", added, tt.added)
			}

			addedDoc, err := marksplice.Parse(added)
			if err != nil {
				t.Fatal(err)
			}
			addedNode := findNodeOfKind(t, addedDoc, tt.kind)
			var removeChange marksplice.ChangeSet
			switch tt.kind {
			case marksplice.KindInlineLink:
				removeChange, err = addedDoc.PrepareRemoveInlineLinkTitle(addedNode.ID())
			case marksplice.KindImage:
				removeChange, err = addedDoc.PrepareRemoveImageTitle(addedNode.ID())
			}
			if err != nil {
				t.Fatalf("remove title error = %v", err)
			}
			final, err := removeChange.Apply(added)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(final, tt.final) {
				t.Fatalf("removed = %q, want %q", final, tt.final)
			}
		})
	}
}

func TestDirectTitleLifecycleFailsClosedAndRemainsSnapshotBound(t *testing.T) {
	t.Parallel()

	withoutTitle := []byte("[label](dest)\n")
	doc, err := marksplice.Parse(withoutTitle)
	if err != nil {
		t.Fatal(err)
	}
	link := findNodeOfKind(t, doc, marksplice.KindInlineLink)
	if _, err := doc.PrepareRemoveInlineLinkTitle(link.ID()); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("remove absent title error = %v, want ErrInvalidReplacement", err)
	}
	for _, title := range [][]byte{nil, []byte("bad\ntitle"), []byte("bad\rtitle")} {
		if _, err := doc.PrepareAddInlineLinkTitle(link.ID(), title); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("add title %q error = %v, want ErrInvalidReplacement", title, err)
		}
	}

	withTitle := []byte("![alt](dest 'old')\n")
	imageDoc, err := marksplice.Parse(withTitle)
	if err != nil {
		t.Fatal(err)
	}
	image := findNodeOfKind(t, imageDoc, marksplice.KindImage)
	if _, err := imageDoc.PrepareAddImageTitle(image.ID(), []byte("second")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("add existing image title error = %v, want ErrInvalidReplacement", err)
	}
	change, err := imageDoc.PrepareRemoveImageTitle(image.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveImageTitle() error = %v", err)
	}
	stale := append([]byte(nil), withTitle...)
	stale[2] = 'X'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}
