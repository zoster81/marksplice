package publictest

import (
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestLocalFragmentContinuityDetectsDuplicateHeadingInsertionRetarget(t *testing.T) {
	t.Parallel()

	source := []byte("[go](#same-1)\n\n# Same\none\n\n# Same\ntwo\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchors := doc.HeadingAnchors()
	if len(anchors) != 2 {
		t.Fatalf("HeadingAnchors() count = %d, want 2", len(anchors))
	}
	change, err := doc.PrepareInsertSectionBefore(anchors[0].HeadingID(), []byte("# Same\nnew\n\n"))
	if err != nil {
		t.Fatalf("PrepareInsertSectionBefore() error = %v", err)
	}

	got, err := doc.LocalFragmentContinuity(change)
	if err != nil {
		t.Fatalf("LocalFragmentContinuity() error = %v", err)
	}
	requireSingleFragmentContinuity(t, got, marksplice.FragmentContinuityTargetChanged)
}

func TestLocalFragmentContinuityTracksMovedDuplicateHeadingIdentity(t *testing.T) {
	t.Parallel()

	source := []byte("[go](#same-1)\n\n# Same\none\n\n# Same\ntwo\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchors := doc.HeadingAnchors()
	if len(anchors) != 2 {
		t.Fatalf("HeadingAnchors() count = %d, want 2", len(anchors))
	}
	change, err := doc.PrepareMoveSectionBefore(anchors[1].HeadingID(), anchors[0].HeadingID())
	if err != nil {
		t.Fatalf("PrepareMoveSectionBefore() error = %v", err)
	}

	got, err := doc.LocalFragmentContinuity(change)
	if err != nil {
		t.Fatalf("LocalFragmentContinuity() error = %v", err)
	}
	requireSingleFragmentContinuity(t, got, marksplice.FragmentContinuityTargetChanged)
}

func TestLocalFragmentContinuityReportsMissingRemovedTarget(t *testing.T) {
	t.Parallel()

	source := []byte("[go](#target)\n\n# Target\nbody\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchors := doc.HeadingAnchors()
	if len(anchors) != 1 {
		t.Fatalf("HeadingAnchors() count = %d, want 1", len(anchors))
	}
	change, err := doc.PrepareRemoveSection(anchors[0].HeadingID())
	if err != nil {
		t.Fatalf("PrepareRemoveSection() error = %v", err)
	}

	got, err := doc.LocalFragmentContinuity(change)
	if err != nil {
		t.Fatalf("LocalFragmentContinuity() error = %v", err)
	}
	requireSingleFragmentContinuity(t, got, marksplice.FragmentContinuityMissing)
}

func TestLocalFragmentContinuityPreservesMovedExplicitHTMLAnchor(t *testing.T) {
	t.Parallel()

	source := []byte("[go](#anchor)\n\n# A\n<a id=\"anchor\"></a>\n\n# B\nbody\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchors := doc.HeadingAnchors()
	if len(anchors) != 2 {
		t.Fatalf("HeadingAnchors() count = %d, want 2", len(anchors))
	}
	change, err := doc.PrepareMoveSectionAfter(anchors[0].HeadingID(), anchors[1].HeadingID())
	if err != nil {
		t.Fatalf("PrepareMoveSectionAfter() error = %v", err)
	}

	got, err := doc.LocalFragmentContinuity(change)
	if err != nil {
		t.Fatalf("LocalFragmentContinuity() error = %v", err)
	}
	requireSingleFragmentContinuity(t, got, marksplice.FragmentContinuityPreserved)
}

func TestLocalFragmentContinuityTreatsRecreatedAnchorAsChangedTarget(t *testing.T) {
	t.Parallel()

	source := []byte("[go](#anchor)\n\n# Section\n<a id=\"anchor\"></a>\nold\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchors := doc.HeadingAnchors()
	if len(anchors) != 1 {
		t.Fatalf("HeadingAnchors() count = %d, want 1", len(anchors))
	}
	change, err := doc.PrepareReplaceSectionBody(anchors[0].HeadingID(), []byte("<a id=\"anchor\"></a>\nnew\n"))
	if err != nil {
		t.Fatalf("PrepareReplaceSectionBody() error = %v", err)
	}

	got, err := doc.LocalFragmentContinuity(change)
	if err != nil {
		t.Fatalf("LocalFragmentContinuity() error = %v", err)
	}
	requireSingleFragmentContinuity(t, got, marksplice.FragmentContinuityTargetChanged)
}

func TestLocalFragmentContinuityRecognizesComposedExplicitRetarget(t *testing.T) {
	t.Parallel()

	source := []byte("[go](#old)\n\n# Old\nbody\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	link := findNodeOfKind(t, doc, marksplice.KindInlineLink)
	anchors := doc.HeadingAnchors()
	if len(anchors) != 1 {
		t.Fatalf("HeadingAnchors() count = %d, want 1", len(anchors))
	}
	retarget, err := doc.PrepareReplaceInlineLinkDestination(link.ID(), []byte("#new"))
	if err != nil {
		t.Fatalf("PrepareReplaceInlineLinkDestination() error = %v", err)
	}
	rename, err := doc.PrepareRenameHeading(anchors[0].HeadingID(), []byte("New"))
	if err != nil {
		t.Fatalf("PrepareRenameHeading() error = %v", err)
	}
	change, err := doc.ComposeChanges(retarget, rename)
	if err != nil {
		t.Fatalf("ComposeChanges() error = %v", err)
	}

	got, err := doc.LocalFragmentContinuity(change)
	if err != nil {
		t.Fatalf("LocalFragmentContinuity() error = %v", err)
	}
	requireSingleFragmentContinuity(t, got, marksplice.FragmentContinuityRetargeted)
}

func TestLocalFragmentContinuityRejectsForeignChange(t *testing.T) {
	t.Parallel()

	doc, err := marksplice.Parse([]byte("[go](#a)\n\n# A\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	other, err := marksplice.Parse([]byte("# B\n"))
	if err != nil {
		t.Fatalf("Parse(other) error = %v", err)
	}
	anchor := other.HeadingAnchors()[0]
	change, err := other.PrepareRenameHeading(anchor.HeadingID(), []byte("C"))
	if err != nil {
		t.Fatalf("PrepareRenameHeading(other) error = %v", err)
	}

	if _, err := doc.LocalFragmentContinuity(change); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("LocalFragmentContinuity(foreign) error = %v, want ErrSourceConflict", err)
	}
}

func TestLocalFragmentContinuityPreservesHeadingAcrossLevelChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
		level  int
	}{
		{name: "ATX marker replacement", source: []byte("[go](#target)\n\n## Target\nbody\n"), level: 3},
		{name: "Setext to ATX conversion", source: []byte("[go](#target)\n\nTarget\n======\nbody\n"), level: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			anchors := doc.HeadingAnchors()
			if len(anchors) != 1 {
				t.Fatalf("HeadingAnchors() count = %d, want 1", len(anchors))
			}
			change, err := doc.PrepareSetHeadingLevel(anchors[0].HeadingID(), tt.level)
			if err != nil {
				t.Fatalf("PrepareSetHeadingLevel() error = %v", err)
			}
			got, err := doc.LocalFragmentContinuity(change)
			if err != nil {
				t.Fatalf("LocalFragmentContinuity() error = %v", err)
			}
			requireSingleFragmentContinuity(t, got, marksplice.FragmentContinuityPreserved)
		})
	}
}

func TestLocalFragmentContinuityComposesTargetCorrespondenceWithEarlierEdit(t *testing.T) {
	t.Parallel()

	source := []byte("[go](#target)\n\nintro\n\n## Target\nbody\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var paragraphs []marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraphs = append(paragraphs, node)
		}
	}
	anchors := doc.HeadingAnchors()
	if len(paragraphs) < 2 || len(anchors) != 1 {
		t.Fatalf("paragraphs/anchors = %d/%d, want at least 2/1", len(paragraphs), len(anchors))
	}
	paragraph := paragraphs[1]
	replace, err := doc.PrepareReplaceParagraph(paragraph.ID(), []byte("a much longer introduction"))
	if err != nil {
		t.Fatalf("PrepareReplaceParagraph() error = %v", err)
	}
	level, err := doc.PrepareSetHeadingLevel(anchors[0].HeadingID(), 3)
	if err != nil {
		t.Fatalf("PrepareSetHeadingLevel() error = %v", err)
	}
	change, err := doc.ComposeChanges(replace, level)
	if err != nil {
		t.Fatalf("ComposeChanges() error = %v", err)
	}
	got, err := doc.LocalFragmentContinuity(change)
	if err != nil {
		t.Fatalf("LocalFragmentContinuity() error = %v", err)
	}
	requireSingleFragmentContinuity(t, got, marksplice.FragmentContinuityPreserved)
}

func TestLocalFragmentContinuityCoversDirectAndReferenceLinksAndImages(t *testing.T) {
	t.Parallel()

	source := []byte("[direct](#same-1) ![direct image](#same-1) [ref][target] ![ref image][target]\n\n[target]: #same-1\n\n# Same\none\n\n# Same\ntwo\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchors := doc.HeadingAnchors()
	if len(anchors) != 2 {
		t.Fatalf("HeadingAnchors() count = %d, want 2", len(anchors))
	}
	change, err := doc.PrepareInsertSectionBefore(anchors[0].HeadingID(), []byte("# Same\nnew\n\n"))
	if err != nil {
		t.Fatalf("PrepareInsertSectionBefore() error = %v", err)
	}

	got, err := doc.LocalFragmentContinuity(change)
	if err != nil {
		t.Fatalf("LocalFragmentContinuity() error = %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("LocalFragmentContinuity() count = %d, want 4", len(got))
	}
	for index, continuity := range got {
		if continuity.Status() != marksplice.FragmentContinuityTargetChanged {
			t.Fatalf("continuity[%d].Status() = %v, want TargetChanged", index, continuity.Status())
		}
	}
}

func TestLocalFragmentContinuityReportsNewAmbiguity(t *testing.T) {
	t.Parallel()

	source := []byte("[go](#target)\n\n# Target\nbody\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchors := doc.HeadingAnchors()
	if len(anchors) != 1 {
		t.Fatalf("HeadingAnchors() count = %d, want 1", len(anchors))
	}
	change, err := doc.PrepareInsertSectionBefore(anchors[0].HeadingID(), []byte("# Added\n<a id=\"target\"></a>\n\n"))
	if err != nil {
		t.Fatalf("PrepareInsertSectionBefore() error = %v", err)
	}

	got, err := doc.LocalFragmentContinuity(change)
	if err != nil {
		t.Fatalf("LocalFragmentContinuity() error = %v", err)
	}
	requireSingleFragmentContinuity(t, got, marksplice.FragmentContinuityAmbiguous)
}

func requireSingleFragmentContinuity(t *testing.T, got []marksplice.FragmentContinuity, want marksplice.FragmentContinuityStatus) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("LocalFragmentContinuity() count = %d, want 1: %+v", len(got), got)
	}
	if got[0].Status() != want {
		t.Fatalf("LocalFragmentContinuity().Status() = %v, want %v", got[0].Status(), want)
	}
	before := got[0].Before()
	if before.FragmentStatus() != marksplice.LinkFragmentResolved {
		t.Fatalf("Before().FragmentStatus() = %v, want Resolved", before.FragmentStatus())
	}
	if after, ok := got[0].After(); !ok && want != marksplice.FragmentContinuityInvalid {
		t.Fatalf("After() ok = false, want correlated relationship")
	} else if ok && after.SourceOffset() < 0 {
		t.Fatalf("After().SourceOffset() = %d, want non-negative", after.SourceOffset())
	}
}
