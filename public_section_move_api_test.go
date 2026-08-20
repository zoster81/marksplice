package marksplice_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicMoveSectionBeforeAndAfterPreserveCompleteSubtreeBytes(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\nintro\n## Alpha\na\n### Alpha Child\nac\n## Beta\nb\n## Gamma\ng\n# Tail\ntail\n")
	tests := []struct {
		name    string
		source  string
		anchor  string
		prepare func(*marksplice.Document, marksplice.NodeID, marksplice.NodeID) (marksplice.ChangeSet, error)
		want    []byte
	}{
		{
			name:   "move subtree forward after sibling",
			source: "Alpha",
			anchor: "Beta",
			prepare: func(doc *marksplice.Document, sourceID, anchorID marksplice.NodeID) (marksplice.ChangeSet, error) {
				return doc.PrepareMoveSectionAfter(sourceID, anchorID)
			},
			want: []byte("# Root\nintro\n## Beta\nb\n## Alpha\na\n### Alpha Child\nac\n## Gamma\ng\n# Tail\ntail\n"),
		},
		{
			name:   "move subtree backward before sibling",
			source: "Gamma",
			anchor: "Alpha",
			prepare: func(doc *marksplice.Document, sourceID, anchorID marksplice.NodeID) (marksplice.ChangeSet, error) {
				return doc.PrepareMoveSectionBefore(sourceID, anchorID)
			},
			want: []byte("# Root\nintro\n## Gamma\ng\n## Alpha\na\n### Alpha Child\nac\n## Beta\nb\n# Tail\ntail\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			sections := publicSectionsByHeadingText(t, doc, source, doc.Sections())
			moved := sections[tt.source]
			anchor := sections[tt.anchor]
			movedBytes := append([]byte(nil), source[moved.Range().Start:moved.Range().End]...)

			change, err := tt.prepare(doc, moved.HeadingID(), anchor.HeadingID())
			if err != nil {
				t.Fatalf("prepare move error = %v", err)
			}
			got, err := change.Apply(source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, tt.want)
			}
			if bytes.Count(got, movedBytes) != 1 {
				t.Fatalf("moved subtree occurrence count = %d, want 1", bytes.Count(got, movedBytes))
			}

			updated, err := marksplice.Parse(got)
			if err != nil {
				t.Fatalf("Parse(result) error = %v", err)
			}
			byHeading := publicSectionsByHeadingText(t, updated, got, updated.Sections())
			movedUpdated := byHeading[tt.source]
			anchorUpdated := byHeading[tt.anchor]
			movedParent, movedHasParent := movedUpdated.ParentHeadingID()
			anchorParent, anchorHasParent := anchorUpdated.ParentHeadingID()
			if movedHasParent != anchorHasParent || (movedHasParent && movedParent != anchorParent) {
				t.Fatalf("moved/anchor parents = %v,%v / %v,%v; want same parent", movedParent, movedHasParent, anchorParent, anchorHasParent)
			}
			if tt.source == "Alpha" {
				child := byHeading["Alpha Child"]
				parent, ok := child.ParentHeadingID()
				if !ok || parent != movedUpdated.HeadingID() {
					t.Fatalf("moved child parent = %v, %v; want moved Alpha", parent, ok)
				}
			}
		})
	}
}

func TestPublicMoveSectionMayReparentAcrossSameLevelAnchors(t *testing.T) {
	t.Parallel()

	source := []byte("# One\r\n## Move\r\nπ\r\n# Two\r\n## Anchor\r\nβ\r\n")
	want := []byte("# One\r\n# Two\r\n## Anchor\r\nβ\r\n## Move\r\nπ\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sections := publicSectionsByHeadingText(t, doc, source, doc.Sections())
	change, err := doc.PrepareMoveSectionAfter(sections["Move"].HeadingID(), sections["Anchor"].HeadingID())
	if err != nil {
		t.Fatalf("PrepareMoveSectionAfter() error = %v", err)
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
	byHeading := publicSectionsByHeadingText(t, updated, got, updated.Sections())
	moveParent, ok := byHeading["Move"].ParentHeadingID()
	if !ok || moveParent != byHeading["Two"].HeadingID() {
		t.Fatalf("Move parent = %v, %v; want Two", moveParent, ok)
	}
}

func TestPublicMoveSectionAdjacentSatisfiedPositionsAreSnapshotBoundNoOps(t *testing.T) {
	t.Parallel()

	source := []byte("# A\na\n# B\nb\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sections := publicSectionsByHeadingText(t, doc, source, doc.Sections())
	for _, tt := range []struct {
		name    string
		prepare func() (marksplice.ChangeSet, error)
	}{
		{name: "A already before B", prepare: func() (marksplice.ChangeSet, error) {
			return doc.PrepareMoveSectionBefore(sections["A"].HeadingID(), sections["B"].HeadingID())
		}},
		{name: "B already after A", prepare: func() (marksplice.ChangeSet, error) {
			return doc.PrepareMoveSectionAfter(sections["B"].HeadingID(), sections["A"].HeadingID())
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			change, err := tt.prepare()
			if err != nil {
				t.Fatalf("prepare no-op move error = %v", err)
			}
			got, err := change.Apply(source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, source) {
				t.Fatalf("no-op move result = %q, want original %q", got, source)
			}
		})
	}
}

func TestPublicMoveSectionRejectsUnsafeRelationshipsAndJoins(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\nintro\n## Move\ngone\nNext\n----\ntail\n## Anchor\na\n# Other\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sections := publicSectionsByHeadingText(t, doc, source, doc.Sections())
	move := sections["Move"]
	anchor := sections["Anchor"]
	other := sections["Other"]

	if _, err := doc.PrepareMoveSectionAfter(move.HeadingID(), anchor.HeadingID()); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("unsafe source join error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareMoveSectionBefore(move.HeadingID(), move.HeadingID()); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("self move error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareMoveSectionBefore(move.HeadingID(), other.HeadingID()); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("different-level move error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareMoveSectionBefore(marksplice.NodeID{}, anchor.HeadingID()); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("missing source error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareMoveSectionBefore(move.HeadingID(), marksplice.NodeID{}); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("missing anchor error = %v, want ErrNodeNotFound", err)
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
	if _, err := doc.PrepareMoveSectionAfter(paragraph.ID(), anchor.HeadingID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("wrong source kind error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestPublicMoveSectionPreparedChangeRejectsStaleSource(t *testing.T) {
	t.Parallel()

	source := []byte("# A\na\n# B\nb\n# C\nc\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	sections := publicSectionsByHeadingText(t, doc, source, doc.Sections())
	change, err := doc.PrepareMoveSectionAfter(sections["A"].HeadingID(), sections["B"].HeadingID())
	if err != nil {
		t.Fatalf("PrepareMoveSectionAfter() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '!'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}
