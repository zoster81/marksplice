package publictest

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM99LinkRelationshipsCoverDirectReferenceImageAndAutoLinkForms(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n\n## Target\n\n<a id=\"explicit\"></a>\n\n[complex *label*](#target) [simple](#target) ![alt *value*](asset.png)\n\n[full][Ref] ![image][Ref] [collapsed][] [shortcut]\n\n<https://example.com/path> <mail@example.com>\n\n[Ref]: <doc.md#part> \"Title\"\n[collapsed]: #target\n[shortcut]: #explicit\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	relationships := doc.LinkRelationships()
	if len(relationships) != 9 {
		t.Fatalf("LinkRelationships() count = %d, want 9: %+v", len(relationships), relationships)
	}
	for index := 1; index < len(relationships); index++ {
		if relationships[index].SourceOffset() <= relationships[index-1].SourceOffset() {
			t.Fatalf("relationships not in strict source order at %d: %d <= %d", index, relationships[index].SourceOffset(), relationships[index-1].SourceOffset())
		}
	}

	kinds := make([]marksplice.LinkRelationshipKind, len(relationships))
	for index, relationship := range relationships {
		kinds[index] = relationship.Kind()
	}
	wantKinds := []marksplice.LinkRelationshipKind{
		marksplice.LinkRelationshipInlineLink,
		marksplice.LinkRelationshipInlineLink,
		marksplice.LinkRelationshipInlineImage,
		marksplice.LinkRelationshipReferenceLink,
		marksplice.LinkRelationshipReferenceImage,
		marksplice.LinkRelationshipReferenceLink,
		marksplice.LinkRelationshipReferenceLink,
		marksplice.LinkRelationshipAutoLink,
		marksplice.LinkRelationshipAutoLink,
	}
	if !reflect.DeepEqual(kinds, wantKinds) {
		t.Fatalf("relationship kinds = %#v, want %#v", kinds, wantKinds)
	}

	if _, ok := relationships[0].SourceNodeID(); ok {
		t.Fatal("complex direct link unexpectedly exposes a promoted source node ID")
	}
	if _, ok := relationships[1].SourceNodeID(); !ok {
		t.Fatal("simple direct link does not expose its promoted source node ID")
	}
	if _, ok := relationships[2].SourceNodeID(); ok {
		t.Fatal("complex direct image unexpectedly exposes a promoted source node ID")
	}

	full := relationships[3]
	if reference, form, ok := full.Reference(); !ok || reference != "Ref" || form != marksplice.ReferenceFormFull {
		t.Fatalf("full Reference() = %q/%v/%v", reference, form, ok)
	}
	if full.Destination() != "doc.md#part" {
		t.Fatalf("full destination = %q, want doc.md#part", full.Destination())
	}
	if title, ok := full.Title(); !ok || title != "Title" {
		t.Fatalf("full title = %q/%v, want Title/true", title, ok)
	}
	definitionID, ok := full.ReferenceDefinitionID()
	if !ok || definitionID.String() == "" {
		t.Fatalf("full ReferenceDefinitionID() = %v/%v", definitionID, ok)
	}
	for _, index := range []int{4} {
		if got, ok := relationships[index].ReferenceDefinitionID(); !ok || got != definitionID {
			t.Fatalf("shared Ref definition ID at %d = %v/%v, want %v", index, got, ok, definitionID)
		}
	}
	if _, _, ok := relationships[0].Reference(); ok {
		t.Fatal("direct relationship unexpectedly reports reference syntax")
	}

	if relationships[7].Destination() != "https://example.com/path" || relationships[7].IsEmail() {
		t.Fatalf("URL autolink = destination %q email %v", relationships[7].Destination(), relationships[7].IsEmail())
	}
	if relationships[8].Destination() != "mail@example.com" || !relationships[8].IsEmail() {
		t.Fatalf("email autolink = destination %q email %v", relationships[8].Destination(), relationships[8].IsEmail())
	}

	copyOut := doc.LinkRelationships()
	copyOut[0] = marksplice.LinkRelationship{}
	again := doc.LinkRelationships()
	if len(again) != len(relationships) || again[0].Kind() == marksplice.LinkRelationshipUnknown {
		t.Fatal("caller mutation leaked into relationship state")
	}
}

func TestM99LinkRelationshipsIncludeBareGFMAutolinks(t *testing.T) {
	t.Parallel()

	source := []byte("https://example.com/path www.example.com mail@example.com\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	relationships := doc.LinkRelationships()
	if len(relationships) != 3 {
		t.Fatalf("LinkRelationships() count = %d, want 3", len(relationships))
	}
	wantDestinations := []string{"https://example.com/path", "http://www.example.com", "mail@example.com"}
	wantEmail := []bool{false, false, true}
	for index := range relationships {
		if relationships[index].Kind() != marksplice.LinkRelationshipAutoLink || relationships[index].Destination() != wantDestinations[index] || relationships[index].IsEmail() != wantEmail[index] {
			t.Fatalf("relationship[%d] = kind %v destination %q email %v", index, relationships[index].Kind(), relationships[index].Destination(), relationships[index].IsEmail())
		}
		if relationships[index].SourceOffset() != bytes.Index(source, []byte(strings.TrimPrefix(wantDestinations[index], "http://"))) {
			t.Fatalf("SourceOffset[%d] = %d", index, relationships[index].SourceOffset())
		}
	}
}

func TestM99LinkRelationshipsExcludeRecognizedFrontMatterEnvelope(t *testing.T) {
	t.Parallel()

	source := []byte("---\nlink: \"[inside](#target)\"\nurl: \"https://inside.example\"\n---\n\n# Target\n\n[outside](#target)\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	relationships := doc.LinkRelationships()
	if len(relationships) != 1 || relationships[0].Destination() != "#target" {
		t.Fatalf("LinkRelationships() = %+v, want only outside fragment link", relationships)
	}
}

func TestM99ReferenceRelationshipKeepsSemanticResolutionWithoutPromotedDefinitionOwner(t *testing.T) {
	t.Parallel()

	source := []byte("[use][target]\n\n[target]: /docs\n  \"Multiline title\"\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	relationships := doc.LinkRelationships()
	if len(relationships) != 1 {
		t.Fatalf("LinkRelationships() count = %d, want 1", len(relationships))
	}
	relationship := relationships[0]
	if relationship.Kind() != marksplice.LinkRelationshipReferenceLink || relationship.Destination() != "/docs" {
		t.Fatalf("relationship = kind %v destination %q", relationship.Kind(), relationship.Destination())
	}
	if title, ok := relationship.Title(); !ok || title != "Multiline title" {
		t.Fatalf("Title() = %q/%v, want Multiline title/true", title, ok)
	}
	if _, ok := relationship.ReferenceDefinitionID(); ok {
		t.Fatal("non-promoted multiline definition unexpectedly exposes ReferenceDefinitionID")
	}
}

func TestM99UnresolvedReferenceSyntaxDoesNotCreateRelationship(t *testing.T) {
	t.Parallel()

	source := []byte("[use][target]\n[target]: /docs\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if relationships := doc.LinkRelationships(); len(relationships) != 0 {
		t.Fatalf("LinkRelationships() = %+v, want no parser-resolved relationships", relationships)
	}
}

func TestM99LinkRelationshipFragmentStatusReusesM98Resolution(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\n\n## Café\n\n<a id=\"explicit\"></a>\n<a id=\"dup\"></a>\n\n## Dup\n\n[heading](#caf%C3%A9) [explicit](#explicit) [missing](#missing) [ambiguous](#dup) [invalid](#) [other](other.md#café)\n\n[via-ref][target]\n\n[target]: #caf%C3%A9\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	relationships := doc.LinkRelationships()
	if len(relationships) != 7 {
		details := make([]string, len(relationships))
		for index, relationship := range relationships {
			details[index] = relationship.Destination()
		}
		t.Fatalf("LinkRelationships() count = %d, want 7; destinations=%#v", len(relationships), details)
	}
	wantStatus := []marksplice.LinkFragmentStatus{
		marksplice.LinkFragmentResolved,
		marksplice.LinkFragmentResolved,
		marksplice.LinkFragmentMissing,
		marksplice.LinkFragmentAmbiguous,
		marksplice.LinkFragmentInvalid,
		marksplice.LinkFragmentNotApplicable,
		marksplice.LinkFragmentResolved,
	}
	for index, want := range wantStatus {
		if got := relationships[index].FragmentStatus(); got != want {
			t.Fatalf("FragmentStatus[%d] = %v, want %v; destination=%q", index, got, want, relationships[index].Destination())
		}
		target, ok := relationships[index].FragmentTarget()
		if want == marksplice.LinkFragmentResolved {
			if !ok || target.Value() == "" {
				t.Fatalf("resolved FragmentTarget[%d] = %+v/%v", index, target, ok)
			}
		} else if ok {
			t.Fatalf("non-resolved FragmentTarget[%d] unexpectedly available: %+v", index, target)
		}
	}
	if target, ok := doc.ResolveFragment("#caf%C3%A9"); !ok {
		t.Fatal("M98 ResolveFragment(#café) failed")
	} else if relationTarget, ok := relationships[0].FragmentTarget(); !ok || relationTarget != target {
		t.Fatalf("relationship target = %+v/%v, M98 target = %+v", relationTarget, ok, target)
	}

	var nilDoc *marksplice.Document
	if got := nilDoc.LinkRelationships(); got != nil {
		t.Fatalf("nil LinkRelationships() = %#v, want nil", got)
	}
}
