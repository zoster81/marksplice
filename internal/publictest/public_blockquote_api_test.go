package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicBlockquotePromotesSimpleExactPhysicalLine(t *testing.T) {
	t.Parallel()

	source := []byte("before\r\n\r\n  > quoted *text*  \r\n\r\nafter\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var found marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindBlockquote {
			found = node
			break
		}
	}
	if found.ID() == (marksplice.NodeID{}) {
		t.Fatal("simple blockquote was not promoted")
	}
	detail, ok := doc.Blockquote(found.ID())
	if !ok {
		t.Fatal("Blockquote() ok = false")
	}
	line, ok := doc.SourceRange(detail.Range())
	if !ok || !bytes.Equal(line, []byte("  > quoted *text*  \r\n")) {
		t.Fatalf("blockquote line = %q/%v, want exact physical line", line, ok)
	}
	content, ok := doc.SourceRange(detail.ContentRange())
	if !ok || !bytes.Equal(content, []byte("quoted *text*  ")) {
		t.Fatalf("blockquote content = %q/%v, want exact inner paragraph source", content, ok)
	}
	if detail.ID() != found.ID() {
		t.Fatalf("detail ID = %v, want %v", detail.ID(), found.ID())
	}
}

func TestPublicBlockquoteSupportsMarkerWithoutFollowingSpace(t *testing.T) {
	t.Parallel()

	source := []byte(">quoted\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindBlockquote {
			continue
		}
		detail, ok := doc.Blockquote(node.ID())
		if !ok {
			t.Fatal("Blockquote() ok = false")
		}
		content, ok := doc.SourceRange(detail.ContentRange())
		if !ok || !bytes.Equal(content, []byte("quoted")) {
			t.Fatalf("content = %q/%v, want quoted", content, ok)
		}
		return
	}
	t.Fatal("block quote without following marker space was not promoted")
}

func TestPublicBlockquotePromotesCompleteExistingSourceContainers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   []byte
		want     []byte
		contents [][]byte
	}{
		{name: "multiline", source: []byte("> one\n> two\n"), want: []byte("> one\n> two\n"), contents: [][]byte{[]byte("one"), []byte("two")}},
		{name: "nested", source: []byte("> > nested\n"), want: []byte("> > nested\n"), contents: [][]byte{[]byte("> nested")}},
		{name: "lazy continuation", source: []byte("> first\nsecond\n"), want: []byte("> first\nsecond\n"), contents: [][]byte{[]byte("first"), []byte("second")}},
		{name: "multi block with empty marker line", source: []byte("> first\n>\n> - item\n"), want: []byte("> first\n>\n> - item\n"), contents: [][]byte{[]byte("first"), {}, []byte("- item")}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			var ids []marksplice.NodeID
			for _, node := range doc.Nodes() {
				if node.Kind() == marksplice.KindBlockquote {
					ids = append(ids, node.ID())
				}
			}
			if len(ids) != 1 {
				t.Fatalf("public blockquote count = %d, want exactly one top-level container", len(ids))
			}
			detail, ok := doc.Blockquote(ids[0])
			if !ok {
				t.Fatal("Blockquote() ok = false")
			}
			got, ok := doc.SourceRange(detail.Range())
			if !ok || !bytes.Equal(got, tt.want) {
				t.Fatalf("Range() source = %q/%v, want %q", got, ok, tt.want)
			}
			ranges, ok := doc.BlockquoteContentRanges(ids[0])
			if !ok || len(ranges) != len(tt.contents) {
				t.Fatalf("BlockquoteContentRanges() = %v/%v, want %d ranges", ranges, ok, len(tt.contents))
			}
			for index, range_ := range ranges {
				content, ok := doc.SourceRange(range_)
				if !ok || !bytes.Equal(content, tt.contents[index]) {
					t.Fatalf("content range %d = %q/%v, want %q", index, content, ok, tt.contents[index])
				}
			}
			if len(ranges) != 1 && detail.ContentRange() != (marksplice.Range{}) {
				t.Fatalf("ContentRange() = %+v, want zero range for segmented content", detail.ContentRange())
			}
		})
	}
}

func TestPublicBlockquoteBroaderSingleSegmentUsesContentRangesOnly(t *testing.T) {
	t.Parallel()

	source := []byte("> - item\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindBlockquote {
			continue
		}
		detail, ok := doc.Blockquote(node.ID())
		if !ok {
			t.Fatal("Blockquote() ok = false")
		}
		if detail.ContentRange() != (marksplice.Range{}) {
			t.Fatalf("ContentRange() = %+v, want zero range for non-paragraph structural child", detail.ContentRange())
		}
		ranges, ok := doc.BlockquoteContentRanges(node.ID())
		if !ok || len(ranges) != 1 {
			t.Fatalf("BlockquoteContentRanges() = %v/%v, want one lexical inner segment", ranges, ok)
		}
		content, ok := doc.SourceRange(ranges[0])
		if !ok || !bytes.Equal(content, []byte("- item")) {
			t.Fatalf("content = %q/%v, want exact structural child source", content, ok)
		}
		return
	}
	t.Fatal("structural-child blockquote was not promoted")
}

func TestPublicBlockquoteContentRangesAreCallerOwned(t *testing.T) {
	t.Parallel()

	source := []byte("> one\r\n  > two π\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var id marksplice.NodeID
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindBlockquote {
			id = node.ID()
			break
		}
	}
	first, ok := doc.BlockquoteContentRanges(id)
	if !ok || len(first) != 2 {
		t.Fatalf("BlockquoteContentRanges() = %v/%v, want two CRLF segments", first, ok)
	}
	original := first[0]
	first[0] = marksplice.Range{}
	second, ok := doc.BlockquoteContentRanges(id)
	if !ok || len(second) != 2 || second[0] != original {
		t.Fatalf("second BlockquoteContentRanges() = %v/%v, caller mutation leaked", second, ok)
	}
	if _, ok := doc.BlockquoteContentRanges(marksplice.NodeID{}); ok {
		t.Fatal("BlockquoteContentRanges(zero ID) ok = true")
	}
}

func TestPublicBlockquoteDoesNotAbsorbUnownedFollowingSource(t *testing.T) {
	t.Parallel()

	source := []byte("> quoted\n\noutside\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindBlockquote {
			continue
		}
		detail, ok := doc.Blockquote(node.ID())
		if !ok {
			t.Fatal("Blockquote() ok = false")
		}
		got, ok := doc.SourceRange(detail.Range())
		if !ok || !bytes.Equal(got, []byte("> quoted\n")) {
			t.Fatalf("Range() source = %q/%v, want exact blockquote only", got, ok)
		}
		return
	}
	t.Fatal("blockquote was not promoted")
}

func TestPrepareRemoveBlockquoteRemovesExactPhysicalLine(t *testing.T) {
	t.Parallel()

	source := []byte("before\r\n\r\n> quoted\r\n\r\nafter\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var id marksplice.NodeID
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindBlockquote {
			id = node.ID()
			break
		}
	}
	change, err := doc.PrepareRemoveBlockquote(id)
	if err != nil {
		t.Fatalf("PrepareRemoveBlockquote() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("before\r\n\r\n\r\nafter\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestPrepareRemoveBlockquoteRemovesCompleteExistingSourceContainers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
		want   []byte
	}{
		{name: "multiline CRLF", source: []byte("before\r\n\r\n  > one\r\n  > two π\r\n\r\nafter\r\n"), want: []byte("before\r\n\r\n\r\nafter\r\n")},
		{name: "lazy continuation", source: []byte("before\n\n> first\nsecond\n\nafter\n"), want: []byte("before\n\n\nafter\n")},
		{name: "nested", source: []byte("before\n\n> > nested\n\nafter\n"), want: []byte("before\n\n\nafter\n")},
		{name: "multi block", source: []byte("before\n\n> first\n>\n> - item\n\nafter\n"), want: []byte("before\n\n\nafter\n")},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			var id marksplice.NodeID
			for _, node := range doc.Nodes() {
				if node.Kind() == marksplice.KindBlockquote {
					id = node.ID()
					break
				}
			}
			change, err := doc.PrepareRemoveBlockquote(id)
			if err != nil {
				t.Fatalf("PrepareRemoveBlockquote() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrepareRemoveBlockquotePreservesLaterComplexBlockquoteMapping(t *testing.T) {
	t.Parallel()

	source := []byte("> remove\n\nmiddle\n\n> keep\n> second\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var first marksplice.NodeID
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindBlockquote {
			first = node.ID()
			break
		}
	}
	change, err := doc.PrepareRemoveBlockquote(first)
	if err != nil {
		t.Fatalf("PrepareRemoveBlockquote() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("\nmiddle\n\n> keep\n> second\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestPrepareRemoveBlockquoteAllowsOwnedReferenceUsageToDisappear(t *testing.T) {
	t.Parallel()

	source := []byte("[docs]: <target>\n\n> [docs]\n\nafter\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var id marksplice.NodeID
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindBlockquote {
			id = node.ID()
			break
		}
	}
	change, err := doc.PrepareRemoveBlockquote(id)
	if err != nil {
		t.Fatalf("PrepareRemoveBlockquote() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("[docs]: <target>\n\n\nafter\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestPrepareRemoveBlockquoteFailsClosedWhenJoinChangesParagraphStructure(t *testing.T) {
	t.Parallel()

	source := []byte("before\n> quoted\n---\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var id marksplice.NodeID
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindBlockquote {
			id = node.ID()
			break
		}
	}
	if _, err := doc.PrepareRemoveBlockquote(id); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareRemoveBlockquote() error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPrepareRemoveBlockquoteIsSourceBound(t *testing.T) {
	t.Parallel()

	source := []byte("> quoted\n\nTail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var id marksplice.NodeID
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindBlockquote {
			id = node.ID()
			break
		}
	}
	change, err := doc.PrepareRemoveBlockquote(id)
	if err != nil {
		t.Fatalf("PrepareRemoveBlockquote() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[len(stale)-2] = 'x'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}

func TestKindBlockquoteAppendsAfterThematicBreak(t *testing.T) {
	t.Parallel()
	if marksplice.KindBlockquote != marksplice.KindThematicBreak+1 {
		t.Fatalf("KindBlockquote = %d, want KindThematicBreak+1 = %d", marksplice.KindBlockquote, marksplice.KindThematicBreak+1)
	}
}
