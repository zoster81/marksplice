package splice

import (
	"bytes"
	"errors"
	"testing"
)

func TestReplaceSimpleInlineLinkDestinationPreservesSourceSyntax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		replacement []byte
		want        []byte
	}{
		{
			name:        "raw destination preserves label spacing title and CRLF",
			source:      []byte("before [label](old/path  \"A title\") after\r\n"),
			replacement: []byte("new/path"),
			want:        []byte("before [label](new/path  \"A title\") after\r\n"),
		},
		{
			name:        "angle destination preserves wrappers and spaces",
			source:      []byte("[label](  <old path> 'title' )\n"),
			replacement: []byte("new path"),
			want:        []byte("[label](  <new path> 'title' )\n"),
		},
		{
			name:        "balanced parentheses remain outside replacement logic",
			source:      []byte("[label](foo(bar) \"title\")\n"),
			replacement: []byte("next(baz)"),
			want:        []byte("[label](next(baz) \"title\")\n"),
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
			links := nodesOfKind(doc.Nodes(), KindInlineLink)
			if len(links) != 1 {
				t.Fatalf("inline link count = %d, want 1; nodes = %+v", len(links), doc.Nodes())
			}
			change, err := doc.PrepareReplaceInlineLinkDestination(links[0].ID, tt.replacement)
			if err != nil {
				t.Fatalf("PrepareReplaceInlineLinkDestination() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestPrepareReplaceInlineLinkDestinationRejectsUnsafeReplacementAndWrongTarget(t *testing.T) {
	t.Parallel()

	source := []byte("[label](old/path \"title\")\n\nparagraph\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	links := nodesOfKind(doc.Nodes(), KindInlineLink)
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(links) != 1 || len(paragraphs) != 2 {
		t.Fatalf("link/paragraph counts = %d/%d, want 1/2", len(links), len(paragraphs))
	}

	for _, replacement := range [][]byte{nil, []byte("line one\nline two"), []byte("new path"), []byte("new)tail")} {
		if _, err := doc.PrepareReplaceInlineLinkDestination(links[0].ID, replacement); !errors.Is(err, ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceInlineLinkDestination(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
	if _, err := doc.PrepareReplaceInlineLinkDestination(paragraphs[1].ID, []byte("new")); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceInlineLinkDestination(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestReplaceReferenceDefinitionDestinationPreservesReferenceLinkStyles(t *testing.T) {
	t.Parallel()

	source := []byte("[id]: <old path>  \"Title\"  \r\n\r\n[full][id] [id][] [id]\r\n")
	want := []byte("[id]: <new path>  \"Title\"  \r\n\r\n[full][id] [id][] [id]\r\n")

	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	definitions := nodesOfKind(doc.Nodes(), KindReferenceDefinition)
	if len(definitions) != 1 {
		t.Fatalf("reference definition count = %d, want 1; nodes = %+v", len(definitions), doc.Nodes())
	}

	refsStart := bytes.Index(source, []byte("[full][id]"))
	if refsStart < 0 {
		t.Fatal("reference links not found in source")
	}
	originalRefs := append([]byte(nil), source[refsStart:]...)

	change, err := doc.PrepareReplaceReferenceDefinitionDestination(definitions[0].ID, []byte("new path"))
	if err != nil {
		t.Fatalf("PrepareReplaceReferenceDefinitionDestination() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
	newRefsStart := bytes.Index(got, []byte("[full][id]"))
	if newRefsStart < 0 || !bytes.Equal(got[newRefsStart:], originalRefs) {
		t.Fatal("full, collapsed, or shortcut reference-link source changed")
	}
}

func TestReplaceRawReferenceDefinitionDestinationPreservesSpacingAndTitle(t *testing.T) {
	t.Parallel()

	source := []byte("  [id]: old/path\t'title'   \n\n[id]\n")
	want := []byte("  [id]: new/path\t'title'   \n\n[id]\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	definitions := nodesOfKind(doc.Nodes(), KindReferenceDefinition)
	if len(definitions) != 1 {
		t.Fatalf("reference definition count = %d, want 1", len(definitions))
	}
	change, err := doc.PrepareReplaceReferenceDefinitionDestination(definitions[0].ID, []byte("new/path"))
	if err != nil {
		t.Fatalf("PrepareReplaceReferenceDefinitionDestination() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
}

func TestPrepareReplaceReferenceDefinitionDestinationRejectsUnsafeReplacementAndWrongTarget(t *testing.T) {
	t.Parallel()

	source := []byte("[id]: old/path \"title\"\n\nparagraph\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	definitions := nodesOfKind(doc.Nodes(), KindReferenceDefinition)
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(definitions) != 1 || len(paragraphs) != 1 {
		t.Fatalf("definition/paragraph counts = %d/%d, want 1/1", len(definitions), len(paragraphs))
	}

	for _, replacement := range [][]byte{nil, []byte("line one\nline two"), []byte("new path"), []byte("new)tail")} {
		if _, err := doc.PrepareReplaceReferenceDefinitionDestination(definitions[0].ID, replacement); !errors.Is(err, ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceReferenceDefinitionDestination(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
	if _, err := doc.PrepareReplaceReferenceDefinitionDestination(paragraphs[0].ID, []byte("new")); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceReferenceDefinitionDestination(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}
