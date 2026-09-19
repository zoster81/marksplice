package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestComposeChangesAndSyncTOCUsesFinalCandidateHeadings(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n\n## Contents\n\n- [Root](#old-root)\n- [Contents](#contents)\n- [Child](#child)\n\n## Child\nbody\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sections := publicSectionsByHeadingText(t, doc, source, doc.Sections())
	contents := sections["Contents"]
	child := sections["Child"]
	rename, err := doc.PrepareRenameHeading(child.HeadingID(), []byte("Renamed"))
	if err != nil {
		t.Fatalf("PrepareRenameHeading() error = %v", err)
	}

	change, err := doc.ComposeChangesAndSyncTOC(contents.HeadingID(), rename)
	if err != nil {
		t.Fatalf("ComposeChangesAndSyncTOC() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("# Root\n\n## Contents\n\n- [Root](#root)\n  - [Contents](#contents)\n  - [Renamed](#renamed)\n\n## Renamed\nbody\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("final-state TOC composition = %q, want %q", got, want)
	}
	updated, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(updated) error = %v", err)
	}
	updatedContents := publicSectionsByHeadingText(t, updated, got, updated.Sections())["Contents"]
	if stale, recognized := updated.TOCStale(updatedContents.HeadingID()); !recognized || stale {
		t.Fatalf("final TOC stale/recognized = %v/%v, want false/true", stale, recognized)
	}
	staleSource := append(append([]byte(nil), source...), 'x')
	if _, err := change.Apply(staleSource); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale source) error = %v, want ErrSourceConflict", err)
	}
}

func TestComposeChangesAndSyncTOCUsesFinalCandidateAfterMultipleChanges(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n\n## Contents\n\n- [Root](#old-root)\n- [Contents](#contents)\n- [Child](#child)\n\n## Child\nbody\n\n## Tail\nend\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sections := publicSectionsByHeadingText(t, doc, source, doc.Sections())
	rename, err := doc.PrepareRenameHeading(sections["Child"].HeadingID(), []byte("Renamed"))
	if err != nil {
		t.Fatalf("PrepareRenameHeading() error = %v", err)
	}
	insert, err := doc.PrepareInsertSectionBefore(sections["Tail"].HeadingID(), []byte("## Added\nnew\n\n"))
	if err != nil {
		t.Fatalf("PrepareInsertSectionBefore() error = %v", err)
	}
	change, err := doc.ComposeChangesAndSyncTOC(sections["Contents"].HeadingID(), rename, insert)
	if err != nil {
		t.Fatalf("ComposeChangesAndSyncTOC() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	updated, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(updated) error = %v", err)
	}
	updatedSections := publicSectionsByHeadingText(t, updated, got, updated.Sections())
	if stale, recognized := updated.TOCStale(updatedSections["Contents"].HeadingID()); !recognized || stale {
		t.Fatalf("final TOC stale/recognized = %v/%v, want false/true", stale, recognized)
	}
	if _, ok := updatedSections["Renamed"]; !ok {
		t.Fatal("renamed section missing")
	}
	if _, ok := updatedSections["Added"]; !ok {
		t.Fatal("inserted section missing")
	}
}

func TestComposeChangesAndSyncTOCRejectsDirectManagedBodyChange(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n\n## Contents\n\n- [Root](#root)\n  - [Contents](#contents)\n\n## Child\nbody\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	contents := publicSectionsByHeadingText(t, doc, source, doc.Sections())["Contents"]
	replace, err := doc.PrepareReplaceSectionBody(contents.HeadingID(), []byte("\n- [Root](#root)\n"))
	if err != nil {
		t.Fatalf("PrepareReplaceSectionBody() error = %v", err)
	}
	if _, err := doc.ComposeChangesAndSyncTOC(contents.HeadingID(), replace); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("ComposeChangesAndSyncTOC(body change) error = %v, want ErrInvalidReplacement", err)
	}
}
