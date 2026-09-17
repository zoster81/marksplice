package publictest

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestHTMLSourceMapTracksNestedFragmentOutput(t *testing.T) {
	t.Parallel()

	source := []byte("hello *world*\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	output, mappings, err := document.HTMLWithSourceMap(marksplice.DefaultHTMLRenderOptions())
	if err != nil {
		t.Fatalf("HTMLWithSourceMap() error = %v", err)
	}
	if want := "<p>hello <em>world</em></p>\n"; string(output) != want {
		t.Fatalf("HTMLWithSourceMap() output = %q, want %q", output, want)
	}
	assertMappingsValidAndOrdered(t, source, output, mappings)
	assertMapping(t, source, output, mappings, "world", "world")
	assertMapping(t, source, output, mappings, "*world*", "<em>world</em>")

	var streamed bytes.Buffer
	streamMappings, err := document.RenderHTMLWithSourceMap(&streamed, marksplice.DefaultHTMLRenderOptions())
	if err != nil {
		t.Fatalf("RenderHTMLWithSourceMap() error = %v", err)
	}
	if !bytes.Equal(streamed.Bytes(), output) {
		t.Fatalf("streamed output = %q, buffered = %q", streamed.Bytes(), output)
	}
	if !equalMappings(streamMappings, mappings) {
		t.Fatalf("stream mappings = %#v, buffered mappings = %#v", streamMappings, mappings)
	}
}

func TestHTMLSourceMapTranslatesFootnoteCaptureAndImageOutput(t *testing.T) {
	t.Parallel()

	source := []byte("image ![alt](/x) note[^a]\n\n[^a]: foot text\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, mappings, err := document.HTMLWithSourceMap(marksplice.DefaultHTMLRenderOptions())
	if err != nil {
		t.Fatalf("HTMLWithSourceMap() error = %v", err)
	}
	assertMappingsValidAndOrdered(t, source, output, mappings)
	assertMappingContains(t, source, output, mappings, "![alt](/x)", "<img src=\"/x\" alt=\"alt\" />")
	assertMapping(t, source, output, mappings, "foot text", "foot text")
}

func TestStandaloneHTMLSourceMapIncludesReviewedMetadataAndAbsoluteBodyOffsets(t *testing.T) {
	t.Parallel()

	source := []byte("---\ntitle: Guide & More\nlang: en-US\n---\nbody\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, mappings, err := document.HTMLDocumentWithSourceMap(marksplice.DefaultHTMLDocumentOptions())
	if err != nil {
		t.Fatalf("HTMLDocumentWithSourceMap() error = %v", err)
	}
	assertMappingsValidAndOrdered(t, source, output, mappings)
	assertMapping(t, source, output, mappings, "Guide & More", "<title>Guide &amp; More</title>\n")
	assertMapping(t, source, output, mappings, "en-US", " lang=\"en-US\"")
	assertMapping(t, source, output, mappings, "body", "body")

	var streamed bytes.Buffer
	streamMappings, err := document.RenderHTMLDocumentWithSourceMap(&streamed, marksplice.DefaultHTMLDocumentOptions())
	if err != nil {
		t.Fatalf("RenderHTMLDocumentWithSourceMap() error = %v", err)
	}
	if !bytes.Equal(streamed.Bytes(), output) || !equalMappings(streamMappings, mappings) {
		t.Fatalf("standalone streaming result differs from buffered result")
	}
}

func TestHTMLSourceMapRejectsInvalidInputsAndFailsClosedWithWriterErrors(t *testing.T) {
	t.Parallel()

	var nilDocument *marksplice.Document
	if mappings, err := nilDocument.RenderHTMLWithSourceMap(&bytes.Buffer{}, marksplice.DefaultHTMLRenderOptions()); !errors.Is(err, marksplice.ErrInvalidRender) || mappings != nil {
		t.Fatalf("nil RenderHTMLWithSourceMap() = %#v/%v, want nil/ErrInvalidRender", mappings, err)
	}
	if _, mappings, err := nilDocument.HTMLWithSourceMap(marksplice.DefaultHTMLRenderOptions()); !errors.Is(err, marksplice.ErrInvalidRender) || mappings != nil {
		t.Fatalf("nil HTMLWithSourceMap() mappings/error = %#v/%v, want nil/ErrInvalidRender", mappings, err)
	}
	if mappings, err := nilDocument.RenderHTMLDocumentWithSourceMap(&bytes.Buffer{}, marksplice.DefaultHTMLDocumentOptions()); !errors.Is(err, marksplice.ErrInvalidRender) || mappings != nil {
		t.Fatalf("nil RenderHTMLDocumentWithSourceMap() = %#v/%v, want nil/ErrInvalidRender", mappings, err)
	}
	if _, mappings, err := nilDocument.HTMLDocumentWithSourceMap(marksplice.DefaultHTMLDocumentOptions()); !errors.Is(err, marksplice.ErrInvalidRender) || mappings != nil {
		t.Fatalf("nil HTMLDocumentWithSourceMap() mappings/error = %#v/%v, want nil/ErrInvalidRender", mappings, err)
	}

	document, err := marksplice.Parse([]byte("text\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if mappings, err := document.RenderHTMLWithSourceMap(nil, marksplice.DefaultHTMLRenderOptions()); !errors.Is(err, marksplice.ErrInvalidRender) || mappings != nil {
		t.Fatalf("nil-writer RenderHTMLWithSourceMap() = %#v/%v, want nil/ErrInvalidRender", mappings, err)
	}
	invalidBody := marksplice.HTMLRenderOptions{RawHTML: marksplice.HTMLRawPolicy(255)}
	if mappings, err := document.RenderHTMLWithSourceMap(&bytes.Buffer{}, invalidBody); !errors.Is(err, marksplice.ErrInvalidRender) || mappings != nil {
		t.Fatalf("invalid RenderHTMLWithSourceMap() = %#v/%v, want nil/ErrInvalidRender", mappings, err)
	}
	invalidDocument := marksplice.HTMLDocumentOptions{Metadata: marksplice.HTMLMetadataPolicy(255)}
	if mappings, err := document.RenderHTMLDocumentWithSourceMap(&bytes.Buffer{}, invalidDocument); !errors.Is(err, marksplice.ErrInvalidRender) || mappings != nil {
		t.Fatalf("invalid RenderHTMLDocumentWithSourceMap() = %#v/%v, want nil/ErrInvalidRender", mappings, err)
	}

	stop := errors.New("writer stopped")
	mappings, err := document.RenderHTMLWithSourceMap(errorWriter{err: stop}, marksplice.DefaultHTMLRenderOptions())
	if !errors.Is(err, stop) {
		t.Fatalf("RenderHTMLWithSourceMap() error = %v, want %v", err, stop)
	}
	if mappings != nil {
		t.Fatalf("RenderHTMLWithSourceMap() mappings = %#v, want nil on failed output", mappings)
	}

	mappings, err = document.RenderHTMLDocumentWithSourceMap(errorWriter{err: stop}, marksplice.DefaultHTMLDocumentOptions())
	if !errors.Is(err, stop) {
		t.Fatalf("RenderHTMLDocumentWithSourceMap() error = %v, want %v", err, stop)
	}
	if mappings != nil {
		t.Fatalf("RenderHTMLDocumentWithSourceMap() mappings = %#v, want nil on failed output", mappings)
	}
}

func TestHTMLSourceMapPreservesExactRendererOutputAcrossComplexFamilies(t *testing.T) {
	t.Parallel()

	source := []byte("- item with [unsafe](javascript:alert(1))\n- second *item*\n\n| A | B |\n| :--- | ---: |\n| cell | <span>raw</span> |\n\n> nested **quote**\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	options := marksplice.HTMLRenderOptions{}
	plain, err := document.HTML(options)
	if err != nil {
		t.Fatalf("HTML() error = %v", err)
	}
	mapped, mappings, err := document.HTMLWithSourceMap(options)
	if err != nil {
		t.Fatalf("HTMLWithSourceMap() error = %v", err)
	}
	if !bytes.Equal(mapped, plain) {
		t.Fatalf("mapped HTML differs from ordinary HTML\nmapped=%q\nplain=%q", mapped, plain)
	}
	assertMappingsValidAndOrdered(t, source, mapped, mappings)
	assertMapping(t, source, mapped, mappings, "cell", "cell")
	assertMapping(t, source, mapped, mappings, "<span>", "<span>")
	assertMapping(t, source, mapped, mappings, "raw", "raw")
	assertMapping(t, source, mapped, mappings, "</span>", "</span>")
	assertMappingContains(t, source, mapped, mappings, "[unsafe](javascript:alert(1))", "<a href=\"\">unsafe</a>")
	assertMapping(t, source, mapped, mappings, "**quote**", "<strong>quote</strong>")
}

func TestHTMLSourceMapUsesByteOffsetsForUnicodeAndReturnsCallerOwnedMappings(t *testing.T) {
	t.Parallel()

	source := []byte("π *λ*\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, mappings, err := document.HTMLWithSourceMap(marksplice.DefaultHTMLRenderOptions())
	if err != nil {
		t.Fatalf("HTMLWithSourceMap() error = %v", err)
	}
	assertMappingsValidAndOrdered(t, source, output, mappings)
	assertMapping(t, source, output, mappings, "π ", "π ")
	assertMapping(t, source, output, mappings, "*λ*", "<em>λ</em>")

	if len(mappings) == 0 {
		t.Fatal("source map unexpectedly empty")
	}
	mappings[0] = marksplice.HTMLSourceMapEntry{}
	_, again, err := document.HTMLWithSourceMap(marksplice.DefaultHTMLRenderOptions())
	if err != nil {
		t.Fatalf("second HTMLWithSourceMap() error = %v", err)
	}
	if len(again) == 0 || again[0].SourceRange() == (marksplice.Range{}) {
		t.Fatal("caller mutation leaked into a later source-map result")
	}
}

func TestStandaloneHTMLSourceMapMetadataOmissionLeavesHeadUnmapped(t *testing.T) {
	t.Parallel()

	source := []byte("---\ntitle: Hidden\ndescription: Also hidden\n---\nbody\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	options := marksplice.HTMLDocumentOptions{Metadata: marksplice.HTMLMetadataOmit}
	output, mappings, err := document.HTMLDocumentWithSourceMap(options)
	if err != nil {
		t.Fatalf("HTMLDocumentWithSourceMap() error = %v", err)
	}
	assertMappingsValidAndOrdered(t, source, output, mappings)
	for _, mapping := range mappings {
		sourceRange := mapping.SourceRange()
		mappedSource := string(source[sourceRange.Start:sourceRange.End])
		if strings.Contains(mappedSource, "Hidden") || strings.Contains(mappedSource, "Also hidden") {
			t.Fatalf("metadata-omitted render returned metadata mapping %#v", mapping)
		}
	}
	assertMapping(t, source, output, mappings, "body", "body")
}

func TestHTMLSourceMapPathologicalInlineIsDeterministic(t *testing.T) {
	t.Parallel()

	source := []byte(strings.Repeat("*_~`π", 8<<10) + "\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	firstOutput, firstMap, err := document.HTMLWithSourceMap(marksplice.DefaultHTMLRenderOptions())
	if err != nil {
		t.Fatalf("first HTMLWithSourceMap() error = %v", err)
	}
	secondOutput, secondMap, err := document.HTMLWithSourceMap(marksplice.DefaultHTMLRenderOptions())
	if err != nil {
		t.Fatalf("second HTMLWithSourceMap() error = %v", err)
	}
	if !bytes.Equal(firstOutput, secondOutput) || !equalMappings(firstMap, secondMap) {
		t.Fatal("pathological source mapping is nondeterministic")
	}
	assertMappingsValidAndOrdered(t, source, firstOutput, firstMap)
}

func TestHTMLSourceMapHandlesDeepNestedBlockquotes(t *testing.T) {
	t.Parallel()

	const depth = 128
	source := []byte(strings.Repeat("> ", depth) + "deep *value*\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	plain, err := document.HTML(marksplice.DefaultHTMLRenderOptions())
	if err != nil {
		t.Fatalf("HTML() error = %v", err)
	}
	mapped, mappings, err := document.HTMLWithSourceMap(marksplice.DefaultHTMLRenderOptions())
	if err != nil {
		t.Fatalf("HTMLWithSourceMap() error = %v", err)
	}
	if !bytes.Equal(mapped, plain) {
		t.Fatal("deep mapped HTML differs from ordinary HTML")
	}
	assertMappingsValidAndOrdered(t, source, mapped, mappings)
	assertMapping(t, source, mapped, mappings, "*value*", "<em>value</em>")
}

func TestHTMLSourceMapRejectsShortWritesWithoutReturningMappings(t *testing.T) {
	t.Parallel()

	document, err := marksplice.Parse([]byte("text *value*\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	mappings, err := document.RenderHTMLWithSourceMap(shortWriter{}, marksplice.DefaultHTMLRenderOptions())
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("RenderHTMLWithSourceMap() error = %v, want io.ErrShortWrite", err)
	}
	if mappings != nil {
		t.Fatalf("RenderHTMLWithSourceMap() mappings = %#v, want nil after short write", mappings)
	}

	mappings, err = document.RenderHTMLDocumentWithSourceMap(shortWriter{}, marksplice.DefaultHTMLDocumentOptions())
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("RenderHTMLDocumentWithSourceMap() error = %v, want io.ErrShortWrite", err)
	}
	if mappings != nil {
		t.Fatalf("RenderHTMLDocumentWithSourceMap() mappings = %#v, want nil after short write", mappings)
	}
}

type shortWriter struct{}

func (shortWriter) Write(value []byte) (int, error) {
	if len(value) == 0 {
		return 0, nil
	}
	return len(value) - 1, nil
}

func assertMappingsValidAndOrdered(t *testing.T, source, output []byte, mappings []marksplice.HTMLSourceMapEntry) {
	t.Helper()
	if len(mappings) == 0 {
		t.Fatal("source map is empty")
	}
	previousStart := -1
	previousEnd := 0
	for index, mapping := range mappings {
		sourceRange := mapping.SourceRange()
		outputRange := mapping.OutputRange()
		if !sourceRange.Valid(len(source)) || sourceRange.Start == sourceRange.End {
			t.Fatalf("mapping %d source range = %+v, invalid for %d bytes", index, sourceRange, len(source))
		}
		if !outputRange.Valid(len(output)) || outputRange.Start == outputRange.End {
			t.Fatalf("mapping %d output range = %+v, invalid for %d bytes", index, outputRange, len(output))
		}
		if outputRange.Start < previousStart || outputRange.Start == previousStart && outputRange.End > previousEnd {
			t.Fatalf("mapping %d output range = %+v after start/end %d/%d; want deterministic output order with outer ranges first", index, outputRange, previousStart, previousEnd)
		}
		previousStart, previousEnd = outputRange.Start, outputRange.End
	}
}

func assertMapping(t *testing.T, source, output []byte, mappings []marksplice.HTMLSourceMapEntry, sourceText, outputText string) {
	t.Helper()
	for _, mapping := range mappings {
		sourceRange := mapping.SourceRange()
		outputRange := mapping.OutputRange()
		if string(source[sourceRange.Start:sourceRange.End]) == sourceText && string(output[outputRange.Start:outputRange.End]) == outputText {
			return
		}
	}
	t.Fatalf("missing mapping %q -> %q in %#v", sourceText, outputText, mappings)
}

func assertMappingContains(t *testing.T, source, output []byte, mappings []marksplice.HTMLSourceMapEntry, sourceText, outputText string) {
	t.Helper()
	for _, mapping := range mappings {
		sourceRange := mapping.SourceRange()
		outputRange := mapping.OutputRange()
		if string(source[sourceRange.Start:sourceRange.End]) == sourceText && strings.Contains(string(output[outputRange.Start:outputRange.End]), outputText) {
			return
		}
	}
	t.Fatalf("missing mapping %q -> output containing %q in %#v", sourceText, outputText, mappings)
}

func equalMappings(left, right []marksplice.HTMLSourceMapEntry) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].SourceRange() != right[index].SourceRange() || left[index].OutputRange() != right[index].OutputRange() {
			return false
		}
	}
	return true
}
