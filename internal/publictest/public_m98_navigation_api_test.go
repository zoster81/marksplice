package publictest

import (
	"errors"
	"reflect"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM98HeadingAnchorsFollowGitHubRulesAndReserveDuplicates(t *testing.T) {
	t.Parallel()

	source := []byte("# Sample Section\n## This'll be a _Helpful_ Section About the Greek Letter Θ!\n## Same\n## Same\n## Same-1\n## Same\n## [Linked](https://example.com) `code`\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchors := doc.HeadingAnchors()
	got := make([]string, len(anchors))
	for i, anchor := range anchors {
		got[i] = anchor.Value()
		if anchor.HeadingID().String() == "" {
			t.Fatalf("HeadingAnchors()[%d] has empty heading ID", i)
		}
		resolved, ok := doc.ResolveFragment("#" + anchor.Value())
		if !ok || resolved.Kind() != marksplice.FragmentTargetHeading || resolved.NodeID() != anchor.HeadingID() {
			t.Fatalf("ResolveFragment(%q) = %+v/%v", "#"+anchor.Value(), resolved, ok)
		}
	}
	want := []string{
		"sample-section",
		"thisll-be-a-helpful-section-about-the-greek-letter-θ",
		"same",
		"same-1",
		"same-1-1",
		"same-2",
		"linked-code",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("HeadingAnchors() = %#v, want %#v", got, want)
	}
	for _, anchor := range anchors {
		byID, ok := doc.HeadingAnchor(anchor.HeadingID())
		if !ok || byID != anchor {
			t.Fatalf("HeadingAnchor(%v) = %+v/%v, want %+v", anchor.HeadingID(), byID, ok, anchor)
		}
	}
}

func TestM98FragmentResolutionSupportsEncodingExplicitAnchorsAndAmbiguity(t *testing.T) {
	t.Parallel()

	source := []byte("# Café\n\n<a name=\"custom-point\"></a>\n\n## Custom Point\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	cafe, ok := doc.ResolveFragment("#caf%C3%A9")
	if !ok || cafe.Kind() != marksplice.FragmentTargetHeading || cafe.Value() != "café" {
		t.Fatalf("encoded cafe fragment = %+v/%v", cafe, ok)
	}
	custom, ok := doc.ResolveFragment("custom-point")
	if ok {
		t.Fatalf("ambiguous custom-point resolved as %+v", custom)
	}
	if doc.ValidateFragment("#custom-point") {
		t.Fatal("ValidateFragment accepted ambiguous heading/explicit anchor")
	}
	if doc.ValidateFragment("#caf%ZZ") || doc.ValidateFragment("") || doc.ValidateFragment("#missing") || doc.ValidateFragment("#one#two") {
		t.Fatal("ValidateFragment accepted malformed/empty/missing fragment")
	}

	explicitDoc, err := marksplice.Parse([]byte("# Root\n\n<a id=\"only-explicit\"></a>\n"))
	if err != nil {
		t.Fatalf("Parse(explicit) error = %v", err)
	}
	explicit, ok := explicitDoc.ResolveFragment("#only-explicit")
	if !ok || explicit.Kind() != marksplice.FragmentTargetHTMLAnchor || explicit.Value() != "only-explicit" {
		t.Fatalf("explicit fragment = %+v/%v", explicit, ok)
	}
}

func TestM98GenerateTOCUsesSectionHierarchyAndEscapesLabels(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n## Child [one]\n#### Deep \\ slash\n## Child [one]\n# Tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	got := string(doc.GenerateTOC())
	want := "- [Root](#root)\n  - [Child \\[one\\]](#child-one)\n    - [Deep \\\\ slash](#deep--slash)\n  - [Child \\[one\\]](#child-one-1)\n- [Tail](#tail)\n"
	if got != want {
		t.Fatalf("GenerateTOC() = %q, want %q", got, want)
	}
}

func TestM98TOCStalenessAndSourcePreservingSynchronization(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\r\n\r\n## Contents\r\n\r\n- [Root](#old-root)\r\n- [Contents](#contents)\r\n- [Child](#child)\r\n\r\n## Child\r\nbody\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sections := publicSectionsByHeadingText(t, doc, source, doc.Sections())
	contents := sections["Contents"]
	stale, ok := doc.TOCStale(contents.HeadingID())
	if !ok || !stale {
		t.Fatalf("TOCStale(Contents) = %v/%v, want true/true", stale, ok)
	}
	change, err := doc.PrepareSyncTOC(contents.HeadingID())
	if err != nil {
		t.Fatalf("PrepareSyncTOC() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("# Root\r\n\r\n## Contents\r\n\r\n- [Root](#root)\r\n  - [Contents](#contents)\r\n  - [Child](#child)\r\n\r\n## Child\r\nbody\r\n")
	if string(got) != string(want) {
		t.Fatalf("synced source = %q, want %q", got, want)
	}
	if string(got[:contents.BodyRange().Start]) != string(source[:contents.BodyRange().Start]) {
		t.Fatal("sync changed bytes before the owned TOC body")
	}

	synced, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(synced) error = %v", err)
	}
	syncedContents := publicSectionsByHeadingText(t, synced, got, synced.Sections())["Contents"]
	stale, ok = synced.TOCStale(syncedContents.HeadingID())
	if !ok || stale {
		t.Fatalf("TOCStale(synced Contents) = %v/%v, want false/true", stale, ok)
	}
}

func TestM98TOCSyncPreservesBlankOnlyBodyPrefix(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n\n## Contents\n\n## Child\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	contents := publicSectionsByHeadingText(t, doc, source, doc.Sections())["Contents"]
	change, err := doc.PrepareSyncTOC(contents.HeadingID())
	if err != nil {
		t.Fatalf("PrepareSyncTOC() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("# Root\n\n## Contents\n\n- [Root](#root)\n  - [Contents](#contents)\n  - [Child](#child)\n## Child\n")
	if string(got) != string(want) {
		t.Fatalf("synced blank-only body = %q, want %q", got, want)
	}
}

func TestM98TOCSyncFailsClosedForArbitrarySectionBodies(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n\n## Notes\nDo not overwrite me.\n\n## Child\nbody\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	notes := publicSectionsByHeadingText(t, doc, source, doc.Sections())["Notes"]
	if stale, ok := doc.TOCStale(notes.HeadingID()); ok || stale {
		t.Fatalf("TOCStale(Notes) = %v/%v, want false/false", stale, ok)
	}
	if _, err := doc.PrepareSyncTOC(notes.HeadingID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareSyncTOC(Notes) error = %v, want ErrInvalidTargetKind", err)
	}

	var nilDoc *marksplice.Document
	if nilDoc.GenerateTOC() != nil || nilDoc.ValidateFragment("#root") {
		t.Fatal("nil document navigation methods returned data")
	}
}
