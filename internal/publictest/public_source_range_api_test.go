package publictest

import (
	"bytes"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicSourceRangeReadsCopiedSnapshotBytes(t *testing.T) {
	t.Parallel()

	source := []byte("prefix\r\n# One\r\nbody 🙂\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	start := bytes.Index(source, []byte("# One"))
	end := len(source)
	want := append([]byte(nil), source[start:end]...)

	got, ok := doc.SourceRange(marksplice.Range{Start: start, End: end})
	if !ok {
		t.Fatal("SourceRange(valid) ok = false, want true")
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("SourceRange(valid) = %q, want %q", got, want)
	}

	got[0] = 'X'
	again, ok := doc.SourceRange(marksplice.Range{Start: start, End: end})
	if !ok || !bytes.Equal(again, want) {
		t.Fatalf("SourceRange() after caller mutation = %q, %v; want unchanged %q, true", again, ok, want)
	}

	source[start] = 'Y'
	again, ok = doc.SourceRange(marksplice.Range{Start: start, End: end})
	if !ok || !bytes.Equal(again, want) {
		t.Fatalf("SourceRange() after input mutation = %q, %v; want snapshot %q, true", again, ok, want)
	}
}

func TestPublicSourceRangeConsumesSectionRanges(t *testing.T) {
	t.Parallel()

	source := []byte("preamble\n\n# One\nintro\n\n## Child\nchild\n\n# Two\ntail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sections := doc.Sections()
	if len(sections) != 3 {
		t.Fatalf("Sections() count = %d, want 3", len(sections))
	}

	one := sections[0]
	subtree, ok := doc.SourceRange(one.Range())
	if !ok {
		t.Fatal("SourceRange(section.Range()) ok = false, want true")
	}
	if want := []byte("# One\nintro\n\n## Child\nchild\n\n"); !bytes.Equal(subtree, want) {
		t.Fatalf("section subtree = %q, want %q", subtree, want)
	}

	body, ok := doc.SourceRange(one.BodyRange())
	if !ok {
		t.Fatal("SourceRange(section.BodyRange()) ok = false, want true")
	}
	if want := []byte("intro\n\n"); !bytes.Equal(body, want) {
		t.Fatalf("section body = %q, want %q", body, want)
	}
}

func TestPublicSourceRangeRejectsInvalidRangesAndAcceptsEmptyRange(t *testing.T) {
	t.Parallel()

	doc, err := marksplice.Parse([]byte("abc"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	for _, range_ := range []marksplice.Range{
		{Start: -1, End: 0},
		{Start: 2, End: 1},
		{Start: 0, End: 4},
	} {
		if got, ok := doc.SourceRange(range_); ok || got != nil {
			t.Fatalf("SourceRange(%v) = %q, %v; want nil, false", range_, got, ok)
		}
	}

	empty, ok := doc.SourceRange(marksplice.Range{Start: 2, End: 2})
	if !ok || len(empty) != 0 {
		t.Fatalf("SourceRange(empty) = %q, %v; want empty, true", empty, ok)
	}

	var nilDoc *marksplice.Document
	if got, ok := nilDoc.SourceRange(marksplice.Range{}); ok || got != nil {
		t.Fatalf("nil Document.SourceRange() = %q, %v; want nil, false", got, ok)
	}
}
