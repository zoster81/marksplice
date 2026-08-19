package splice

import (
	"bytes"
	"errors"
	"testing"
)

func TestReplaceParagraphPreservesUnchangedBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
	}{
		{
			name:   "LF",
			source: []byte("# Title\n\nOriginal paragraph.\n\n- keep\n- this\n"),
		},
		{
			name:   "LF with trailing spaces",
			source: []byte("# Title\n\nOriginal paragraph.  \n\n- keep\n- this\n"),
		},
		{
			name:   "CRLF",
			source: []byte("# Title\r\n\r\nOriginal paragraph.\r\n\r\n- keep\r\n- this\r\n"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, err := Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
			if len(paragraphs) != 1 {
				t.Fatalf("paragraph count = %d, want 1", len(paragraphs))
			}
			target := paragraphs[0]

			marker := []byte("Original paragraph.")
			expectedStart := bytes.Index(tt.source, marker)
			if expectedStart < 0 {
				t.Fatal("test fixture paragraph marker not found")
			}
			expectedEnd := expectedStart + len(marker)
			for expectedEnd < len(tt.source) && (tt.source[expectedEnd] == ' ' || tt.source[expectedEnd] == '\t') {
				expectedEnd++
			}
			if target.Range.Start != expectedStart || target.Range.End != expectedEnd {
				t.Fatalf("paragraph range = [%d,%d), want [%d,%d)", target.Range.Start, target.Range.End, expectedStart, expectedEnd)
			}

			prefix := append([]byte(nil), tt.source[:expectedStart]...)
			suffix := append([]byte(nil), tt.source[expectedEnd:]...)
			replacement := []byte("Replacement paragraph with **formatting**.")

			change, err := doc.PrepareReplace(target.ID, replacement)
			if err != nil {
				t.Fatalf("PrepareReplace() error = %v", err)
			}

			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}

			want := make([]byte, 0, len(prefix)+len(replacement)+len(suffix))
			want = append(want, prefix...)
			want = append(want, replacement...)
			want = append(want, suffix...)
			if !bytes.Equal(got, want) {
				t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
			}
			if !bytes.Equal(got[:len(prefix)], prefix) {
				t.Fatal("bytes before changed span were modified")
			}
			if !bytes.Equal(got[len(prefix)+len(replacement):], suffix) {
				t.Fatal("bytes after changed span were modified")
			}
		})
	}
}

func TestSetTaskCheckedChangesOnlyCheckboxStateByte(t *testing.T) {
	t.Parallel()

	source := []byte("- [ ] first\r\n* [X] second\r\n+ [x] third\r\n\r\nplain [ ] text\r\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	tasks := nodesOfKind(doc.Nodes(), KindTask)
	if len(tasks) != 3 {
		t.Fatalf("task count = %d, want 3", len(tasks))
	}
	if tasks[0].Checked || !tasks[1].Checked || !tasks[2].Checked {
		t.Fatalf("task checked states = %v, %v, %v; want false, true, true", tasks[0].Checked, tasks[1].Checked, tasks[2].Checked)
	}
	if got, want := string(source[tasks[0].Range.Start:tasks[0].Range.End]), "[ ]"; got != want {
		t.Fatalf("task marker range = %q, want %q", got, want)
	}
	if got, want := string(source[tasks[0].ContentRange.Start:tasks[0].ContentRange.End]), " "; got != want {
		t.Fatalf("task state range = %q, want %q", got, want)
	}

	change, err := doc.PrepareSetTaskChecked(tasks[0].ID, true)
	if err != nil {
		t.Fatalf("PrepareSetTaskChecked() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("- [x] first\r\n* [X] second\r\n+ [x] third\r\n\r\nplain [ ] text\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
}

func TestSetTaskUncheckedPreservesListMarkerAndNoOpPreservesUppercaseX(t *testing.T) {
	t.Parallel()

	source := []byte("* [X] Keep style\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	tasks := nodesOfKind(doc.Nodes(), KindTask)
	if len(tasks) != 1 {
		t.Fatalf("task count = %d, want 1", len(tasks))
	}

	noOp, err := doc.PrepareSetTaskChecked(tasks[0].ID, true)
	if err != nil {
		t.Fatalf("PrepareSetTaskChecked(no-op) error = %v", err)
	}
	unchanged, err := noOp.Apply(source)
	if err != nil {
		t.Fatalf("Apply(no-op) error = %v", err)
	}
	if !bytes.Equal(unchanged, source) {
		t.Fatalf("no-op changed source: %q", unchanged)
	}

	change, err := doc.PrepareSetTaskChecked(tasks[0].ID, false)
	if err != nil {
		t.Fatalf("PrepareSetTaskChecked(false) error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply(false) error = %v", err)
	}
	want := []byte("* [ ] Keep style\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestNestedTaskListItemIsMappedWithoutChangingIndentation(t *testing.T) {
	t.Parallel()

	source := []byte("1. parent\n   - [ ] nested\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	tasks := nodesOfKind(doc.Nodes(), KindTask)
	if len(tasks) != 1 {
		t.Fatalf("task count = %d, want 1", len(tasks))
	}

	change, err := doc.PrepareSetTaskChecked(tasks[0].ID, true)
	if err != nil {
		t.Fatalf("PrepareSetTaskChecked() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("1. parent\n   - [x] nested\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestPrepareSetTaskCheckedRejectsNonTask(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("Paragraph.\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(paragraphs) != 1 {
		t.Fatalf("paragraph count = %d, want 1", len(paragraphs))
	}
	if _, err := doc.PrepareSetTaskChecked(paragraphs[0].ID, true); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareSetTaskChecked(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestRenameHeadingPreservesATXSourceStyle(t *testing.T) {
	t.Parallel()

	source := []byte("  ### Old *heading* ###  \n\nParagraph.\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	headings := nodesOfKind(doc.Nodes(), KindHeading)
	if len(headings) != 1 {
		t.Fatalf("heading count = %d, want 1", len(headings))
	}
	target := headings[0]
	if target.Level != 3 || target.HeadingStyle != HeadingStyleATX {
		t.Fatalf("heading metadata = level %d style %d, want level 3 ATX", target.Level, target.HeadingStyle)
	}
	if got, want := string(source[target.Range.Start:target.Range.End]), "  ### Old *heading* ###  "; got != want {
		t.Fatalf("heading source range = %q, want %q", got, want)
	}
	if got, want := string(source[target.ContentRange.Start:target.ContentRange.End]), "Old *heading*"; got != want {
		t.Fatalf("heading content range = %q, want %q", got, want)
	}

	replacement := []byte("New **heading**")
	change, err := doc.PrepareRenameHeading(target.ID, replacement)
	if err != nil {
		t.Fatalf("PrepareRenameHeading() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("  ### New **heading** ###  \n\nParagraph.\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
}

func TestRenameHeadingPreservesSetextSourceStyleAndCRLF(t *testing.T) {
	t.Parallel()

	source := []byte("Old heading\r\n-----------   \r\n\r\nParagraph.\r\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	headings := nodesOfKind(doc.Nodes(), KindHeading)
	if len(headings) != 1 {
		t.Fatalf("heading count = %d, want 1", len(headings))
	}
	target := headings[0]
	if target.Level != 2 || target.HeadingStyle != HeadingStyleSetext {
		t.Fatalf("heading metadata = level %d style %d, want level 2 Setext", target.Level, target.HeadingStyle)
	}
	if got, want := string(source[target.Range.Start:target.Range.End]), "Old heading\r\n-----------   "; got != want {
		t.Fatalf("heading source range = %q, want %q", got, want)
	}
	if got, want := string(source[target.ContentRange.Start:target.ContentRange.End]), "Old heading"; got != want {
		t.Fatalf("heading content range = %q, want %q", got, want)
	}

	change, err := doc.PrepareRenameHeading(target.ID, []byte("New heading"))
	if err != nil {
		t.Fatalf("PrepareRenameHeading() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("New heading\r\n-----------   \r\n\r\nParagraph.\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
}

func TestRenameHeadingPreservesSetextLevelOneAndCRLineEndings(t *testing.T) {
	t.Parallel()

	source := []byte("Old heading\r=========  \r\rParagraph.\r")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	headings := nodesOfKind(doc.Nodes(), KindHeading)
	if len(headings) != 1 {
		t.Fatalf("heading count = %d, want 1", len(headings))
	}
	target := headings[0]
	if target.Level != 1 || target.HeadingStyle != HeadingStyleSetext {
		t.Fatalf("heading metadata = level %d style %d, want level 1 Setext", target.Level, target.HeadingStyle)
	}

	change, err := doc.PrepareRenameHeading(target.ID, []byte("New heading"))
	if err != nil {
		t.Fatalf("PrepareRenameHeading() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("New heading\r=========  \r\rParagraph.\r")
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
}

func TestPrepareRenameHeadingRejectsNonHeadingAndMultilineReplacement(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("# Heading\n\nParagraph.\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(paragraphs) != 1 {
		t.Fatalf("paragraph count = %d, want 1", len(paragraphs))
	}
	if _, err := doc.PrepareRenameHeading(paragraphs[0].ID, []byte("new")); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareRenameHeading(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}

	headings := nodesOfKind(doc.Nodes(), KindHeading)
	if len(headings) != 1 {
		t.Fatalf("heading count = %d, want 1", len(headings))
	}
	if _, err := doc.PrepareRenameHeading(headings[0].ID, []byte("line one\nline two")); !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("PrepareRenameHeading(multiline) error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPrepareRenameHeadingRejectsSetextReplacementThatChangesBlockKind(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("Heading\n-------\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	headings := nodesOfKind(doc.Nodes(), KindHeading)
	if len(headings) != 1 {
		t.Fatalf("heading count = %d, want 1", len(headings))
	}

	_, err = doc.PrepareRenameHeading(headings[0].ID, []byte("---"))
	if !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("PrepareRenameHeading() error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPreparedChangeRejectsStaleSource(t *testing.T) {
	t.Parallel()

	source := []byte("# Title\n\nOriginal paragraph.\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(paragraphs) != 1 {
		t.Fatalf("paragraph count = %d, want 1", len(paragraphs))
	}

	change, err := doc.PrepareReplace(paragraphs[0].ID, []byte("Replacement."))
	if err != nil {
		t.Fatalf("PrepareReplace() error = %v", err)
	}

	stale := append([]byte(nil), source...)
	stale[0] = '*'
	_, err = change.Apply(stale)
	if !errors.Is(err, ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}

func TestDuplicateParagraphsHaveDistinctDeterministicIDs(t *testing.T) {
	t.Parallel()

	source := []byte("same\n\nsame\n")
	first, err := Parse(source)
	if err != nil {
		t.Fatalf("first Parse() error = %v", err)
	}
	second, err := Parse(source)
	if err != nil {
		t.Fatalf("second Parse() error = %v", err)
	}

	firstParagraphs := nodesOfKind(first.Nodes(), KindParagraph)
	secondParagraphs := nodesOfKind(second.Nodes(), KindParagraph)
	if len(firstParagraphs) != 2 || len(secondParagraphs) != 2 {
		t.Fatalf("paragraph counts = %d and %d, want 2 and 2", len(firstParagraphs), len(secondParagraphs))
	}
	if firstParagraphs[0].ID == firstParagraphs[1].ID {
		t.Fatal("duplicate paragraph content produced duplicate node IDs")
	}
	for i := range firstParagraphs {
		if firstParagraphs[i].ID != secondParagraphs[i].ID {
			t.Fatalf("node %d ID is not deterministic: %q != %q", i, firstParagraphs[i].ID, secondParagraphs[i].ID)
		}
	}
}

func TestPrepareReplaceRejectsUnknownNode(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("paragraph\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	_, err = doc.PrepareReplace(NodeID("missing"), []byte("replacement"))
	if !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("PrepareReplace() error = %v, want ErrNodeNotFound", err)
	}
}

func TestPrepareReplaceRejectsDifferentBlockKind(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("paragraph\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(paragraphs) != 1 {
		t.Fatalf("paragraph count = %d, want 1", len(paragraphs))
	}

	_, err = doc.PrepareReplace(paragraphs[0].ID, []byte("# heading"))
	if !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("PrepareReplace() error = %v, want ErrInvalidReplacement", err)
	}
}

func nodesOfKind(nodes []Node, kind Kind) []Node {
	var result []Node
	for _, node := range nodes {
		if node.Kind == kind {
			result = append(result, node)
		}
	}
	return result
}
