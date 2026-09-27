package splice

import (
	"slices"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestMapLinkRelationshipsOrderOwnershipAndFailure(t *testing.T) {
	document, err := Parse([]byte("# Target\n\n[a](first) [b](#target) [c][ref]\n\n[ref]: last\n"))
	if err != nil {
		t.Fatal(err)
	}
	project := func(relationship LinkRelationship) (string, bool) {
		return relationship.Destination, true
	}
	want := []string{"first", "#target", "last"}
	got, ok := MapLinkRelationships(document, project)
	if !ok || !slices.Equal(got, want) {
		t.Fatalf("projection = %v/%v, want %v", got, ok, want)
	}
	got[0] = "changed"
	again, ok := MapLinkRelationships(document, project)
	if !ok || !slices.Equal(again, want) {
		t.Fatal("projection shares mutable result storage")
	}
	calls := 0
	partial, ok := MapLinkRelationships(document, func(relationship LinkRelationship) (string, bool) {
		calls++
		return relationship.Destination, calls < 2
	})
	if ok || partial != nil || calls != 2 {
		t.Fatalf("failed conversion returned %v/%v after %d calls", partial, ok, calls)
	}
	for _, item := range []struct {
		document *Document
		wantNil  bool
	}{{nil, true}, {&Document{}, false}} {
		empty, ok := MapLinkRelationships(item.document, project)
		if !ok || len(empty) != 0 || (empty == nil) != item.wantNil {
			t.Fatalf("empty projection = %v/%v, want nil=%v", empty, ok, item.wantNil)
		}
	}
}

func TestMapLinkRelationshipsDiscardsInvalidSemanticProjection(t *testing.T) {
	document := &Document{linkUsages: []parser.LinkUsage{
		{Kind: parser.KindInlineLink, Form: parser.LinkUsageDirect, Destination: "valid"},
		{Kind: parser.KindInlineLink, Form: parser.LinkUsageDirect, Anchor: -1},
	}}
	calls := 0
	result, ok := MapLinkRelationships(document, func(relationship LinkRelationship) (string, bool) {
		calls++
		return relationship.Destination, true
	})
	if ok || result != nil || calls != 1 {
		t.Fatalf("invalid relationship returned %v/%v after %d conversions", result, ok, calls)
	}
}
