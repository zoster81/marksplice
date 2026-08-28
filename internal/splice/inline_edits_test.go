package splice

import (
	"bytes"
	"errors"
	"testing"
)

func TestReplaceSimpleStrikethroughPreservesDelimitersAndSurroundingSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		targetText  string
		replacement []byte
		want        []byte
	}{
		{
			name:        "single tilde preserves CRLF and surrounding paragraph bytes",
			source:      []byte("before ~old~ after\r\n"),
			targetText:  "old",
			replacement: []byte("new"),
			want:        []byte("before ~new~ after\r\n"),
		},
		{
			name:        "double tilde preserves Unicode byte ranges",
			source:      []byte("prefix ~~caffè 東京~~ suffix\n"),
			targetText:  "caffè 東京",
			replacement: []byte("nuovo 東京"),
			want:        []byte("prefix ~~nuovo 東京~~ suffix\n"),
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
			strikes := nodesOfKind(doc.Nodes(), KindStrikethrough)
			var target Node
			found := false
			for _, strike := range strikes {
				if string(tt.source[strike.ContentRange.Start:strike.ContentRange.End]) == tt.targetText {
					target = strike
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("strikethrough with content %q not found; strikes = %+v", tt.targetText, strikes)
			}
			mapping, ok := remapStrikethroughSource(tt.source, target)
			if !target.Editable || !ok || mapping.ContentRange != target.ContentRange || mapping.DelimiterLength < 1 {
				t.Fatalf("strikethrough capability = editable %v mapping %+v", target.Editable, mapping)
			}

			prefix := append([]byte(nil), tt.source[:target.ContentRange.Start]...)
			suffix := append([]byte(nil), tt.source[target.ContentRange.End:]...)
			change, err := doc.PrepareReplaceStrikethrough(target.ID, tt.replacement)
			if err != nil {
				t.Fatalf("PrepareReplaceStrikethrough() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, tt.want)
			}
			if !bytes.Equal(got[:len(prefix)], prefix) {
				t.Fatal("bytes before changed strikethrough content were modified")
			}
			if !bytes.Equal(got[len(prefix)+len(tt.replacement):], suffix) {
				t.Fatal("bytes after changed strikethrough content were modified")
			}
		})
	}
}

func TestPrepareReplaceStrikethroughRejectsUnsafeReplacementAndWrongTarget(t *testing.T) {
	t.Parallel()

	source := []byte("before ~~old~~ after\n\nparagraph\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	strikes := nodesOfKind(doc.Nodes(), KindStrikethrough)
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(strikes) != 1 || len(paragraphs) != 2 {
		t.Fatalf("strikethrough/paragraph counts = %d/%d, want 1/2", len(strikes), len(paragraphs))
	}

	for _, replacement := range [][]byte{nil, []byte("line one\nline two"), []byte("~new"), []byte("*new*")} {
		if _, err := doc.PrepareReplaceStrikethrough(strikes[0].ID, replacement); !errors.Is(err, ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceStrikethrough(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
	if _, err := doc.PrepareReplaceStrikethrough(paragraphs[1].ID, []byte("new")); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceStrikethrough(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestReplaceAutoLinkPreservesSourceStyle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		targetText  string
		replacement []byte
		want        []byte
	}{
		{
			name:        "angle URL preserves brackets and CRLF",
			source:      []byte("before <https://old.example/path> after\r\n"),
			targetText:  "https://old.example/path",
			replacement: []byte("https://new.example/next"),
			want:        []byte("before <https://new.example/next> after\r\n"),
		},
		{
			name:        "bare HTTPS preserves surrounding bytes",
			source:      []byte("before https://old.example/path after\n"),
			targetText:  "https://old.example/path",
			replacement: []byte("https://new.example/next"),
			want:        []byte("before https://new.example/next after\n"),
		},
		{
			name:        "bare www remains extended autolink",
			source:      []byte("www.old.example/path tail\n"),
			targetText:  "www.old.example/path",
			replacement: []byte("www.new.example/next"),
			want:        []byte("www.new.example/next tail\n"),
		},
		{
			name:        "published mailto protocol form remains extended autolink",
			source:      []byte("mailto:foo@bar.baz tail\n"),
			targetText:  "mailto:foo@bar.baz",
			replacement: []byte("mailto:new@bar.baz"),
			want:        []byte("mailto:new@bar.baz tail\n"),
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
			autolinks := nodesOfKind(doc.Nodes(), KindAutoLink)
			var target Node
			found := false
			for _, autolink := range autolinks {
				if string(tt.source[autolink.ContentRange.Start:autolink.ContentRange.End]) == tt.targetText {
					target = autolink
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("autolink with source %q not found; nodes = %+v", tt.targetText, doc.Nodes())
			}
			change, err := doc.PrepareReplaceAutoLink(target.ID, tt.replacement)
			if err != nil {
				t.Fatalf("PrepareReplaceAutoLink() error = %v", err)
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

func TestPrepareReplaceAutoLinkRejectsUnsafeReplacementAndWrongTarget(t *testing.T) {
	t.Parallel()

	source := []byte("https://old.example/path\n\nparagraph\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	autolinks := nodesOfKind(doc.Nodes(), KindAutoLink)
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(autolinks) != 1 || len(paragraphs) != 2 {
		t.Fatalf("autolink/paragraph counts = %d/%d, want 1/2", len(autolinks), len(paragraphs))
	}
	for _, replacement := range [][]byte{nil, []byte("line one\nline two"), []byte("not a link"), []byte("ftp://example.com")} {
		if _, err := doc.PrepareReplaceAutoLink(autolinks[0].ID, replacement); !errors.Is(err, ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceAutoLink(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
	if _, err := doc.PrepareReplaceAutoLink(paragraphs[1].ID, []byte("https://new.example")); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceAutoLink(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestReplaceSimpleCodeSpanPreservesFenceRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		targetText  string
		replacement []byte
		want        []byte
	}{
		{
			name:        "single backtick preserves CRLF",
			source:      []byte("before `old` after\r\n"),
			targetText:  "old",
			replacement: []byte("new"),
			want:        []byte("before `new` after\r\n"),
		},
		{
			name:        "double backtick permits single backtick content",
			source:      []byte("before ``old`code`` after\n"),
			targetText:  "old`code",
			replacement: []byte("new`code"),
			want:        []byte("before ``new`code`` after\n"),
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
			spans := nodesOfKind(doc.Nodes(), KindCodeSpan)
			var target Node
			found := false
			for _, span := range spans {
				if string(tt.source[span.ContentRange.Start:span.ContentRange.End]) == tt.targetText {
					target = span
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("code span with content %q not found; nodes = %+v", tt.targetText, doc.Nodes())
			}
			mapping, ok := remapCodeSpanSource(tt.source, target)
			if !target.Editable || !ok || mapping.ContentRange != target.ContentRange || mapping.FenceLength < 1 {
				t.Fatalf("code-span capability = editable %v mapping %+v", target.Editable, mapping)
			}
			change, err := doc.PrepareReplaceCodeSpan(target.ID, tt.replacement)
			if err != nil {
				t.Fatalf("PrepareReplaceCodeSpan() error = %v", err)
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

func TestPrepareReplaceCodeSpanRejectsUnsafeReplacementAndWrongTarget(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("`old`\n\nparagraph\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	spans := nodesOfKind(doc.Nodes(), KindCodeSpan)
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(spans) != 1 || len(paragraphs) != 2 {
		t.Fatalf("code-span/paragraph counts = %d/%d, want 1/2", len(spans), len(paragraphs))
	}
	for _, replacement := range [][]byte{nil, []byte("line one\nline two"), []byte("`")} {
		if _, err := doc.PrepareReplaceCodeSpan(spans[0].ID, replacement); !errors.Is(err, ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceCodeSpan(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
	if _, err := doc.PrepareReplaceCodeSpan(paragraphs[1].ID, []byte("new")); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceCodeSpan(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestReplaceSimpleEmphasisAndStrongPreservesDelimiterStyle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		kind        Kind
		replacement []byte
		want        []byte
	}{
		{name: "asterisk emphasis", source: []byte("before *old* after\r\n"), kind: KindEmphasis, replacement: []byte("new"), want: []byte("before *new* after\r\n")},
		{name: "underscore emphasis", source: []byte("before _old_ after\n"), kind: KindEmphasis, replacement: []byte("new"), want: []byte("before _new_ after\n")},
		{name: "asterisk strong", source: []byte("before **old** after\r\n"), kind: KindStrong, replacement: []byte("new"), want: []byte("before **new** after\r\n")},
		{name: "underscore strong", source: []byte("before __old__ after\n"), kind: KindStrong, replacement: []byte("new"), want: []byte("before __new__ after\n")},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			nodes := nodesOfKind(doc.Nodes(), tt.kind)
			if len(nodes) != 1 {
				t.Fatalf("node count for kind %d = %d, want 1; nodes = %+v", tt.kind, len(nodes), doc.Nodes())
			}
			mapping, ok := remapEmphasisSource(tt.source, nodes[0])
			if !nodes[0].Editable || !ok || mapping.ContentRange != nodes[0].ContentRange || mapping.Level < 1 {
				t.Fatalf("emphasis capability = editable %v mapping %+v", nodes[0].Editable, mapping)
			}
			var change ChangeSet
			if tt.kind == KindEmphasis {
				change, err = doc.PrepareReplaceEmphasis(nodes[0].ID, tt.replacement)
			} else {
				change, err = doc.PrepareReplaceStrong(nodes[0].ID, tt.replacement)
			}
			if err != nil {
				t.Fatalf("prepare inline emphasis/strong replacement error = %v", err)
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

func TestPrepareReplaceEmphasisAndStrongRejectUnsafeReplacement(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("*old* and **bold**\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	emphasis := nodesOfKind(doc.Nodes(), KindEmphasis)
	strong := nodesOfKind(doc.Nodes(), KindStrong)
	if len(emphasis) != 1 || len(strong) != 1 {
		t.Fatalf("emphasis/strong counts = %d/%d, want 1/1; nodes = %+v", len(emphasis), len(strong), doc.Nodes())
	}
	for _, replacement := range [][]byte{nil, []byte("line one\nline two"), []byte("*new")} {
		if _, err := doc.PrepareReplaceEmphasis(emphasis[0].ID, replacement); !errors.Is(err, ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceEmphasis(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
	for _, replacement := range [][]byte{nil, []byte("line one\nline two"), []byte("**new")} {
		if _, err := doc.PrepareReplaceStrong(strong[0].ID, replacement); !errors.Is(err, ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceStrong(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
}

func TestUnsupportedInlineSourceShapesRemainParsableButFailClosed(t *testing.T) {
	t.Parallel()

	codeSource := []byte("` old `\n")
	codeDoc, err := Parse(codeSource)
	if err != nil {
		t.Fatalf("Parse(normalized code span) error = %v", err)
	}
	codeSpans := nodesOfKind(codeDoc.Nodes(), KindCodeSpan)
	if len(codeSpans) != 1 {
		t.Fatalf("normalized-space code span count = %d, want 1 semantic observation", len(codeSpans))
	}
	if codeSpans[0].Editable {
		t.Fatalf("normalized-space code span capability = editable %v, want false", codeSpans[0].Editable)
	}
	if _, ok := remapCodeSpanSource(codeSource, codeSpans[0]); ok {
		t.Fatal("normalized-space code span unexpectedly remapped as editable source")
	}
	if _, err := codeDoc.PrepareReplaceCodeSpan(codeSpans[0].ID, []byte("new")); err == nil {
		t.Fatal("PrepareReplaceCodeSpan(normalized-space source) error = nil, want fail-closed unsupported source shape")
	}

	compoundSource := []byte("***old***\n")
	compoundDoc, err := Parse(compoundSource)
	if err != nil {
		t.Fatalf("Parse(compound emphasis) error = %v", err)
	}
	candidates := append(nodesOfKind(compoundDoc.Nodes(), KindEmphasis), nodesOfKind(compoundDoc.Nodes(), KindStrong)...)
	if len(candidates) == 0 {
		t.Fatal("compound emphasis produced no semantic emphasis/strong observation")
	}
	for _, candidate := range candidates {
		if candidate.Editable {
			t.Fatalf("compound emphasis capability = editable %v, want false", candidate.Editable)
		}
		if _, ok := remapEmphasisSource(compoundSource, candidate); ok {
			t.Fatal("compound emphasis unexpectedly remapped as editable source")
		}
		var err error
		if candidate.Kind == KindEmphasis {
			_, err = compoundDoc.PrepareReplaceEmphasis(candidate.ID, []byte("new"))
		} else {
			_, err = compoundDoc.PrepareReplaceStrong(candidate.ID, []byte("new"))
		}
		if err == nil {
			t.Fatalf("compound emphasis candidate %+v unexpectedly accepted for mutation", candidate)
		}
	}
}
