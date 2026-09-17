package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestAppendFirstListItemChildDerivesSourceProvenLayout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  []byte
		parent  string
		content []byte
		ordered bool
		want    []byte
	}{
		{
			name:    "unordered LF",
			source:  []byte("- parent\n- tail\n"),
			parent:  "parent",
			content: []byte("child"),
			want:    []byte("- parent\n  - child\n- tail\n"),
		},
		{
			name:    "ordered CRLF and unicode",
			source:  []byte("42) parent\r\n43) tail\r\n"),
			parent:  "parent",
			content: []byte("child π"),
			ordered: true,
			want:    []byte("42) parent\r\n    1. child π\r\n43) tail\r\n"),
		},
		{
			name:    "blockquote container prefix",
			source:  []byte("> - parent\n> - tail\n"),
			parent:  "parent",
			content: []byte("child"),
			want:    []byte("> - parent\n>   - child\n> - tail\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			parent := publicListItemByContent(t, doc, tt.source, tt.parent)
			change, err := doc.PrepareAppendFirstListItemChild(parent.ID(), tt.content, tt.ordered)
			if err != nil {
				t.Fatalf("PrepareAppendFirstListItemChild() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("Apply() = %q, want %q", got, tt.want)
			}
			if _, err := marksplice.Parse(got); err != nil {
				t.Fatalf("Parse(applied) error = %v", err)
			}
		})
	}
}

func TestAppendFirstListItemChildFailsClosedOutsideProvenHost(t *testing.T) {
	t.Parallel()

	withChild := []byte("- parent\n  - existing\n- tail\n")
	doc, err := marksplice.Parse(withChild)
	if err != nil {
		t.Fatalf("Parse(with child) error = %v", err)
	}
	parent := publicListItemByContent(t, doc, withChild, "parent")
	if _, err := doc.PrepareAppendFirstListItemChild(parent.ID(), []byte("new"), false); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("existing child error = %v, want ErrInvalidReplacement", err)
	}

	for _, content := range [][]byte{nil, []byte("one\ntwo"), []byte("one\rtwo")} {
		source := []byte("- parent\n- tail\n")
		doc, err := marksplice.Parse(source)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		parent := publicListItemByContent(t, doc, source, "parent")
		if _, err := doc.PrepareAppendFirstListItemChild(parent.ID(), content, false); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("content %q error = %v, want ErrInvalidReplacement", content, err)
		}
	}

	atEOF := []byte("- parent")
	eofDoc, err := marksplice.Parse(atEOF)
	if err != nil {
		t.Fatalf("Parse(EOF) error = %v", err)
	}
	eofParent := publicListItemByContent(t, eofDoc, atEOF, "parent")
	if _, err := eofDoc.PrepareAppendFirstListItemChild(eofParent.ID(), []byte("child"), false); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("EOF parent error = %v, want ErrInvalidReplacement", err)
	}
}

func TestAppendFirstListItemChildRemainsSnapshotBound(t *testing.T) {
	t.Parallel()

	source := []byte("- parent\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := publicListItemByContent(t, doc, source, "parent")
	change, err := doc.PrepareAppendFirstListItemChild(parent.ID(), []byte("child"), false)
	if err != nil {
		t.Fatalf("PrepareAppendFirstListItemChild() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[len(stale)-2] = 'X'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}
