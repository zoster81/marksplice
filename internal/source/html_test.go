package source

import (
	"errors"
	"testing"
)

func TestMapHTMLCommentPreservesDelimitersAndInnerPadding(t *testing.T) {
	t.Parallel()

	source := []byte("before <!--  old comment  --> after\r\n")
	raw := Range{Start: 7, End: 29}
	got, err := MapHTMLComment(source, raw)
	if err != nil {
		t.Fatalf("MapHTMLComment() error = %v", err)
	}
	if got.Range != raw || got.ContentRange != (Range{Start: 13, End: 24}) {
		t.Fatalf("mapping = %+v, want raw %v content [13,24)", got, raw)
	}
}

func TestMapHTMLCommentRejectsUnsupportedShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
		raw    Range
	}{
		{name: "not a comment", source: []byte("<span>\n"), raw: Range{Start: 0, End: 6}},
		{name: "multiline comment", source: []byte("<!-- one\ntwo -->\n"), raw: Range{Start: 0, End: 16}},
		{name: "empty padded comment", source: []byte("<!--   -->\n"), raw: Range{Start: 0, End: 10}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := MapHTMLComment(tt.source, tt.raw)
			if !errors.Is(err, ErrUnsupportedHTMLShape) {
				t.Fatalf("MapHTMLComment() error = %v, want ErrUnsupportedHTMLShape", err)
			}
		})
	}
}

func TestMapSimpleHTMLAnchorPreservesQuotedAttributeBoundary(t *testing.T) {
	t.Parallel()

	source := []byte("before <a class=\"x\" id = 'old-anchor'>link</a> after\n")
	raw := Range{Start: 7, End: 38}
	got, err := MapSimpleHTMLAnchor(source, raw)
	if err != nil {
		t.Fatalf("MapSimpleHTMLAnchor() error = %v", err)
	}
	if got.Range != raw || got.ContentRange != (Range{Start: 26, End: 36}) || got.Attribute != "id" || got.Quote != '\'' {
		t.Fatalf("mapping = %+v, want raw %v content [26,36) id single-quote", got, raw)
	}
}

func TestMapSimpleHTMLAnchorRejectsAmbiguousOrUnquotedTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
	}{
		{name: "unquoted id", source: []byte("<a id=old>\n")},
		{name: "duplicate target attributes", source: []byte("<a id=\"one\" name=\"two\">\n")},
		{name: "non-anchor tag", source: []byte("<span id=\"old\">\n")},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raw := Range{Start: 0, End: len(tt.source) - 1}
			_, err := MapSimpleHTMLAnchor(tt.source, raw)
			if !errors.Is(err, ErrUnsupportedHTMLShape) {
				t.Fatalf("MapSimpleHTMLAnchor() error = %v, want ErrUnsupportedHTMLShape", err)
			}
		})
	}
}
