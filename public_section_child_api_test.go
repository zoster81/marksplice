package marksplice_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicAppendSectionChildAddsFirstChildAndPreservesBytes(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\r\nbody π\r\n# Tail\r\ntail\r\n")
	fragment := []byte("## Child\r\nchild\r\n### Deep\r\ndeep\r\n")
	want := []byte("# Root\r\nbody π\r\n## Child\r\nchild\r\n### Deep\r\ndeep\r\n# Tail\r\ntail\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	root := publicSectionsByHeadingText(t, doc, source, doc.Sections())["Root"]
	insertAt := root.Range().End
	prefix := append([]byte(nil), source[:insertAt]...)
	suffix := append([]byte(nil), source[insertAt:]...)

	change, err := doc.PrepareAppendSectionChild(root.HeadingID(), fragment)
	if err != nil {
		t.Fatalf("PrepareAppendSectionChild() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
	if !bytes.Equal(got[:len(prefix)], prefix) || !bytes.Equal(got[len(prefix)+len(fragment):], suffix) {
		t.Fatal("append child modified bytes outside the zero-width parent subtree boundary")
	}

	updated, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(result) error = %v", err)
	}
	sections := publicSectionsByHeadingText(t, updated, got, updated.Sections())
	child := sections["Child"]
	parent, ok := child.ParentHeadingID()
	if !ok || parent != sections["Root"].HeadingID() {
		t.Fatalf("Child parent = %v, %v; want Root", parent, ok)
	}
	deepParent, ok := sections["Deep"].ParentHeadingID()
	if !ok || deepParent != child.HeadingID() {
		t.Fatalf("Deep parent = %v, %v; want Child", deepParent, ok)
	}
}

func TestPublicAppendSectionChildAppendsAfterExistingDescendants(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n## First\none\n### Deep\nd\n# Tail\ntail\n")
	fragment := []byte("## Last\nlast\n")
	want := []byte("# Root\n## First\none\n### Deep\nd\n## Last\nlast\n# Tail\ntail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	root := publicSectionsByHeadingText(t, doc, source, doc.Sections())["Root"]
	change, err := doc.PrepareAppendSectionChild(root.HeadingID(), fragment)
	if err != nil {
		t.Fatalf("PrepareAppendSectionChild() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}

	updated, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(result) error = %v", err)
	}
	sections := publicSectionsByHeadingText(t, updated, got, updated.Sections())
	lastParent, ok := sections["Last"].ParentHeadingID()
	if !ok || lastParent != sections["Root"].HeadingID() {
		t.Fatalf("Last parent = %v, %v; want Root", lastParent, ok)
	}
}

func TestPublicAppendSectionChildSupportsSafeEOFAndRejectsUnsafeEOF(t *testing.T) {
	t.Parallel()

	safe := []byte("# Root\nbody\n")
	doc, err := marksplice.Parse(safe)
	if err != nil {
		t.Fatalf("Parse(safe) error = %v", err)
	}
	root := publicSectionsByHeadingText(t, doc, safe, doc.Sections())["Root"]
	change, err := doc.PrepareAppendSectionChild(root.HeadingID(), []byte("## Child\nchild"))
	if err != nil {
		t.Fatalf("PrepareAppendSectionChild(safe) error = %v", err)
	}
	got, err := change.Apply(safe)
	if err != nil {
		t.Fatalf("Apply(safe) error = %v", err)
	}
	if want := []byte("# Root\nbody\n## Child\nchild"); !bytes.Equal(got, want) {
		t.Fatalf("safe result = %q, want %q", got, want)
	}

	unsafe := []byte("# Root\nbody")
	unsafeDoc, err := marksplice.Parse(unsafe)
	if err != nil {
		t.Fatalf("Parse(unsafe) error = %v", err)
	}
	unsafeRoot := publicSectionsByHeadingText(t, unsafeDoc, unsafe, unsafeDoc.Sections())["Root"]
	if _, err := unsafeDoc.PrepareAppendSectionChild(unsafeRoot.HeadingID(), []byte("## Child\nchild\n")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareAppendSectionChild(unsafe EOF) error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPublicAppendSectionChildRejectsInvalidFragmentsAndTargets(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\nbody\n###### Max\nend\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sections := publicSectionsByHeadingText(t, doc, source, doc.Sections())
	root := sections["Root"]
	max := sections["Max"]

	for _, tt := range []struct {
		name     string
		fragment []byte
	}{
		{name: "empty", fragment: nil},
		{name: "wrong level", fragment: []byte("### Too Deep\nbody\n")},
		{name: "preamble", fragment: []byte("text\n## Child\nbody\n")},
		{name: "multiple child roots", fragment: []byte("## One\na\n## Two\nb\n")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := doc.PrepareAppendSectionChild(root.HeadingID(), tt.fragment); !errors.Is(err, marksplice.ErrInvalidReplacement) {
				t.Fatalf("PrepareAppendSectionChild() error = %v, want ErrInvalidReplacement", err)
			}
		})
	}
	if _, err := doc.PrepareAppendSectionChild(max.HeadingID(), []byte("###### Child\nbody\n")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("level-6 append error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareAppendSectionChild(marksplice.NodeID{}, []byte("## Child\nbody\n")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("missing parent error = %v, want ErrNodeNotFound", err)
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
	if _, err := doc.PrepareAppendSectionChild(paragraph.ID(), []byte("## Child\nbody\n")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("wrong parent kind error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestPublicAppendSectionChildPreparedChangeRejectsStaleSource(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\nbody\n# Tail\ntail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	root := publicSectionsByHeadingText(t, doc, source, doc.Sections())["Root"]
	change, err := doc.PrepareAppendSectionChild(root.HeadingID(), []byte("## Child\nbody\n"))
	if err != nil {
		t.Fatalf("PrepareAppendSectionChild() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '!'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}
