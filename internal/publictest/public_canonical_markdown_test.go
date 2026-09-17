package publictest

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestCanonicalMarkdownBasicPolicyAndIdempotence(t *testing.T) {
	t.Parallel()

	source := []byte("Title\n=====\r\n\r\nParagraph with *emphasis* and [docs](guide.md).\r\n\r\n+ one\r\n+ two\r\n\r\n    code\r\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	canonical, err := document.CanonicalMarkdown()
	if err != nil {
		t.Fatalf("CanonicalMarkdown() error = %v", err)
	}
	want := "# Title\n\nParagraph with *emphasis* and [docs](<guide.md>)\\.\n\n- one\n\n- two\n\n  code\n"
	if string(canonical) != want {
		t.Fatalf("CanonicalMarkdown() = %q, want %q", canonical, want)
	}

	var streamed bytes.Buffer
	if err := document.RenderCanonicalMarkdown(&streamed); err != nil {
		t.Fatalf("RenderCanonicalMarkdown() error = %v", err)
	}
	if !bytes.Equal(streamed.Bytes(), canonical) {
		t.Fatalf("streamed canonical = %q, buffered = %q", streamed.Bytes(), canonical)
	}

	reparsed, err := marksplice.Parse(canonical)
	if err != nil {
		t.Fatalf("Parse(canonical) error = %v", err)
	}
	second, err := reparsed.CanonicalMarkdown()
	if err != nil {
		t.Fatalf("second CanonicalMarkdown() error = %v", err)
	}
	if !bytes.Equal(second, canonical) {
		t.Fatalf("canonical rendering is not idempotent\nfirst:  %q\nsecond: %q", canonical, second)
	}
}

func TestCanonicalMarkdownRejectsInvalidInputsAndWriterFailures(t *testing.T) {
	t.Parallel()

	var nilDocument *marksplice.Document
	if err := nilDocument.RenderCanonicalMarkdown(&bytes.Buffer{}); !errors.Is(err, marksplice.ErrInvalidRender) {
		t.Fatalf("nil RenderCanonicalMarkdown() error = %v, want ErrInvalidRender", err)
	}
	if output, err := nilDocument.CanonicalMarkdown(); !errors.Is(err, marksplice.ErrInvalidRender) || output != nil {
		t.Fatalf("nil CanonicalMarkdown() = %q/%v, want nil/ErrInvalidRender", output, err)
	}

	document, err := marksplice.Parse([]byte("text\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if err := document.RenderCanonicalMarkdown(nil); !errors.Is(err, marksplice.ErrInvalidRender) {
		t.Fatalf("nil-writer RenderCanonicalMarkdown() error = %v, want ErrInvalidRender", err)
	}

	stop := errors.New("writer stopped")
	if err := document.RenderCanonicalMarkdown(errorWriter{err: stop}); !errors.Is(err, stop) {
		t.Fatalf("error-writer RenderCanonicalMarkdown() error = %v, want %v", err, stop)
	}
	if err := document.RenderCanonicalMarkdown(shortWriter{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short-writer RenderCanonicalMarkdown() error = %v, want io.ErrShortWrite", err)
	}
}
