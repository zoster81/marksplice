package source

import (
	"errors"
	"testing"
)

func TestMapSimpleInlineLinkPreservesDestinationBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    []byte
		anchor    int
		label     Range
		dest      string
		title     string
		hasTitle  bool
		wantRange Range
		wantDest  Range
		wantTitle Range
		wantAngle bool
	}{
		{
			name:      "raw destination with title and CRLF",
			source:    []byte("before [label](old/path  \"A title\") after\r\n"),
			anchor:    7,
			label:     Range{Start: 8, End: 13},
			dest:      "old/path",
			title:     "A title",
			hasTitle:  true,
			wantRange: Range{Start: 7, End: 35},
			wantDest:  Range{Start: 15, End: 23},
			wantTitle: Range{Start: 26, End: 33},
		},
		{
			name:      "angle destination preserves wrapper outside range",
			source:    []byte("[label](  <old path> 'title' )\n"),
			anchor:    0,
			label:     Range{Start: 1, End: 6},
			dest:      "old path",
			title:     "title",
			hasTitle:  true,
			wantRange: Range{Start: 0, End: 30},
			wantDest:  Range{Start: 11, End: 19},
			wantTitle: Range{Start: 22, End: 27},
			wantAngle: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := MapSimpleInlineLink(tt.source, tt.anchor, tt.label, tt.dest, tt.title, tt.hasTitle)
			if err != nil {
				t.Fatalf("MapSimpleInlineLink() error = %v", err)
			}
			if got.Range != tt.wantRange || got.LabelRange != tt.label || got.DestinationRange != tt.wantDest || got.TitleRange != tt.wantTitle || got.AngleDestination != tt.wantAngle || got.HasTitle != tt.hasTitle {
				t.Fatalf("mapping = %+v, want range %v label %v destination %v title %v angle %v", got, tt.wantRange, tt.label, tt.wantDest, tt.wantTitle, tt.wantAngle)
			}
		})
	}
}

func TestMapSimpleInlineLinkRejectsUnprovenShape(t *testing.T) {
	t.Parallel()

	source := []byte("[label](old/path \"title\")\n")
	tests := []struct {
		name  string
		label Range
		dest  string
		title string
	}{
		{name: "wrong label boundary", label: Range{Start: 1, End: 5}, dest: "old/path", title: "title"},
		{name: "semantic destination mismatch", label: Range{Start: 1, End: 6}, dest: "other/path", title: "title"},
		{name: "semantic title mismatch", label: Range{Start: 1, End: 6}, dest: "old/path", title: "other"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := MapSimpleInlineLink(source, 0, tt.label, tt.dest, tt.title, true)
			if !errors.Is(err, ErrUnsupportedInlineLinkShape) {
				t.Fatalf("MapSimpleInlineLink() error = %v, want ErrUnsupportedInlineLinkShape", err)
			}
		})
	}
}

func TestMapSingleLineReferenceDefinitionPreservesDestinationBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    []byte
		obs       Range
		label     string
		dest      string
		title     string
		hasTitle  bool
		wantRange Range
		wantDest  Range
		wantTitle Range
		wantAngle bool
	}{
		{
			name:      "angle destination with CRLF",
			source:    []byte("[id]: <old path>  \"Title\"  \r\n"),
			obs:       Range{Start: 0, End: 27},
			label:     "id",
			dest:      "old path",
			title:     "Title",
			hasTitle:  true,
			wantRange: Range{Start: 0, End: 27},
			wantDest:  Range{Start: 7, End: 15},
			wantTitle: Range{Start: 19, End: 24},
			wantAngle: true,
		},
		{
			name:      "raw destination with indentation and tab",
			source:    []byte("  [id]: old/path\t'title'   \n"),
			obs:       Range{Start: 2, End: 24},
			label:     "id",
			dest:      "old/path",
			title:     "title",
			hasTitle:  true,
			wantRange: Range{Start: 0, End: 27},
			wantDest:  Range{Start: 8, End: 16},
			wantTitle: Range{Start: 18, End: 23},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := MapSingleLineReferenceDefinition(tt.source, tt.obs, tt.label, tt.dest, tt.title, tt.hasTitle)
			if err != nil {
				t.Fatalf("MapSingleLineReferenceDefinition() error = %v", err)
			}
			if got.Range != tt.wantRange || got.DestinationRange != tt.wantDest || got.TitleRange != tt.wantTitle || got.AngleDestination != tt.wantAngle || got.HasTitle != tt.hasTitle {
				t.Fatalf("mapping = %+v, want range %v destination %v title %v angle %v", got, tt.wantRange, tt.wantDest, tt.wantTitle, tt.wantAngle)
			}
		})
	}
}

func TestMapSingleLineReferenceDefinitionRejectsUnprovenShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
		obs    Range
		label  string
		dest   string
		title  string
	}{
		{name: "semantic label mismatch", source: []byte("[id]: old/path\n"), obs: Range{Start: 0, End: 14}, label: "other", dest: "old/path"},
		{name: "semantic destination mismatch", source: []byte("[id]: old/path\n"), obs: Range{Start: 0, End: 14}, label: "id", dest: "other/path"},
		{name: "observation crosses physical lines", source: []byte("[id]: old/path\nnext\n"), obs: Range{Start: 0, End: 19}, label: "id", dest: "old/path"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := MapSingleLineReferenceDefinition(tt.source, tt.obs, tt.label, tt.dest, tt.title, false)
			if !errors.Is(err, ErrUnsupportedReferenceDefinitionShape) {
				t.Fatalf("MapSingleLineReferenceDefinition() error = %v, want ErrUnsupportedReferenceDefinitionShape", err)
			}
		})
	}
}

func TestMapAutoLinkPreservesAngleAndBareBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    []byte
		anchor    int
		content   Range
		value     string
		wantRange Range
		wantAngle bool
	}{
		{
			name:      "angle autolink",
			source:    []byte("before <https://old.example/path> after\r\n"),
			anchor:    7,
			content:   Range{Start: 8, End: 32},
			value:     "https://old.example/path",
			wantRange: Range{Start: 7, End: 33},
			wantAngle: true,
		},
		{
			name:      "bare autolink",
			source:    []byte("before https://old.example/path after\n"),
			anchor:    7,
			content:   Range{Start: 7, End: 31},
			value:     "https://old.example/path",
			wantRange: Range{Start: 7, End: 31},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := MapAutoLink(tt.source, tt.anchor, tt.content, tt.value, false)
			if err != nil {
				t.Fatalf("MapAutoLink() error = %v", err)
			}
			if got.Range != tt.wantRange || got.ContentRange != tt.content || got.Angle != tt.wantAngle || got.Email {
				t.Fatalf("mapping = %+v, want range %v content %v angle %v", got, tt.wantRange, tt.content, tt.wantAngle)
			}
		})
	}
}

func TestMapAutoLinkRejectsUnprovenShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  []byte
		anchor  int
		content Range
		value   string
	}{
		{name: "semantic value mismatch", source: []byte("https://old.example\n"), anchor: 0, content: Range{Start: 0, End: 19}, value: "https://new.example"},
		{name: "angle closing bracket missing", source: []byte("<https://old.example\n"), anchor: 0, content: Range{Start: 1, End: 20}, value: "https://old.example"},
		{name: "bare anchor mismatch", source: []byte("xhttps://old.example\n"), anchor: 0, content: Range{Start: 1, End: 20}, value: "https://old.example"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := MapAutoLink(tt.source, tt.anchor, tt.content, tt.value, false)
			if !errors.Is(err, ErrUnsupportedAutoLinkShape) {
				t.Fatalf("MapAutoLink() error = %v, want ErrUnsupportedAutoLinkShape", err)
			}
		})
	}
}
