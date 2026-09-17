package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestR30SetHeadingLevelPreservesReviewedSourceTrivia(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		src   []byte
		text  string
		level int
		want  []byte
	}{
		{
			name:  "ATX promote preserves indentation spacing and closing sequence",
			src:   []byte("  ## Heading π   ##  \n\nbody\n"),
			text:  "Heading π",
			level: 4,
			want:  []byte("  #### Heading π   ##  \n\nbody\n"),
		},
		{
			name:  "ATX demote preserves CRLF",
			src:   []byte("#### Heading\r\n\r\nbody\r\n"),
			text:  "Heading",
			level: 1,
			want:  []byte("# Heading\r\n\r\nbody\r\n"),
		},
		{
			name:  "Setext level two to one preserves underline run and trivia",
			src:   []byte("Heading\r\n  ---------   \r\n\r\nbody\r\n"),
			text:  "Heading",
			level: 1,
			want:  []byte("Heading\r\n  =========   \r\n\r\nbody\r\n"),
		},
		{
			name:  "Setext converts to ATX when target level needs it",
			src:   []byte("  Heading π   \r\n  ========   \r\n\r\nbody\r\n"),
			text:  "Heading π",
			level: 3,
			want:  []byte("  ### Heading π   \r\n\r\nbody\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.src)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			heading := publicHeadingByText(t, doc, tt.text)
			change, err := doc.PrepareSetHeadingLevel(heading.ID(), tt.level)
			if err != nil {
				t.Fatalf("PrepareSetHeadingLevel() error = %v", err)
			}
			got, err := change.Apply(tt.src)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("Apply() = %q, want %q", got, tt.want)
			}
			updated, err := marksplice.Parse(got)
			if err != nil {
				t.Fatalf("Parse(applied) error = %v", err)
			}
			updatedHeading := publicHeadingByText(t, updated, tt.text)
			if updatedHeading.Level() != tt.level {
				t.Fatalf("Heading.Level() = %d, want %d", updatedHeading.Level(), tt.level)
			}
		})
	}
}

func TestR30SetHeadingLevelRecomputesSectionHierarchyWithoutChangingOtherHeadings(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\nintro\n## Child\nbody\n### Grandchild\nleaf\n## Peer\nend\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	child := publicHeadingByText(t, doc, "Child")
	change, err := doc.PrepareSetHeadingLevel(child.ID(), 1)
	if err != nil {
		t.Fatalf("PrepareSetHeadingLevel() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("# Root\nintro\n# Child\nbody\n### Grandchild\nleaf\n## Peer\nend\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Apply() = %q, want %q", got, want)
	}
	updated, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(applied) error = %v", err)
	}
	root := publicHeadingByText(t, updated, "Root")
	updatedChild := publicHeadingByText(t, updated, "Child")
	grandchild := publicHeadingByText(t, updated, "Grandchild")
	peer := publicHeadingByText(t, updated, "Peer")
	if updatedChild.Level() != 1 || root.Level() != 1 || grandchild.Level() != 3 || peer.Level() != 2 {
		t.Fatalf("levels = root %d child %d grandchild %d peer %d", root.Level(), updatedChild.Level(), grandchild.Level(), peer.Level())
	}
	grandSection, ok := updated.Section(grandchild.ID())
	if !ok {
		t.Fatal("grandchild section unavailable")
	}
	parentID, ok := grandSection.ParentHeadingID()
	if !ok || parentID != updatedChild.ID() {
		t.Fatalf("grandchild parent = %v, %v; want Child", parentID, ok)
	}
	peerSection, ok := updated.Section(peer.ID())
	if !ok {
		t.Fatal("peer section unavailable")
	}
	peerParent, ok := peerSection.ParentHeadingID()
	if !ok || peerParent != updatedChild.ID() {
		t.Fatalf("peer parent = %v, %v; want Child", peerParent, ok)
	}
}

func TestR30SetHeadingLevelFailsClosedForInvalidOrUnrepresentableTargets(t *testing.T) {
	t.Parallel()

	source := []byte("# Heading\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	heading := publicHeadingByText(t, doc, "Heading")
	for _, level := range []int{0, 7} {
		if _, err := doc.PrepareSetHeadingLevel(heading.ID(), level); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("level %d error = %v, want ErrInvalidReplacement", level, err)
		}
	}

	multiline := []byte("00\n0\n-\n")
	multiDoc, err := marksplice.Parse(multiline)
	if err != nil {
		t.Fatalf("Parse(multiline Setext) error = %v", err)
	}
	multiHeading := publicHeadingByLevel(t, multiDoc, 2)
	if _, err := multiDoc.PrepareSetHeadingLevel(multiHeading.ID(), 3); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("multiline Setext->ATX error = %v, want ErrInvalidReplacement", err)
	}

	change, err := doc.PrepareSetHeadingLevel(heading.ID(), 2)
	if err != nil {
		t.Fatalf("PrepareSetHeadingLevel() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[len(stale)-2] = 'X'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}

func publicHeadingByText(t *testing.T, doc *marksplice.Document, text string) marksplice.Heading {
	t.Helper()
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindHeading {
			continue
		}
		heading, ok := doc.Heading(node.ID())
		if ok && heading.Text() == text {
			return heading
		}
	}
	t.Fatalf("heading %q not found", text)
	return marksplice.Heading{}
}

func publicHeadingByLevel(t *testing.T, doc *marksplice.Document, level int) marksplice.Heading {
	t.Helper()
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindHeading {
			continue
		}
		heading, ok := doc.Heading(node.ID())
		if ok && heading.Level() == level {
			return heading
		}
	}
	t.Fatalf("heading level %d not found", level)
	return marksplice.Heading{}
}
