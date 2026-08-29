package publictest

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM120PublicHTMLRendererStreamsDeterministicFragment(t *testing.T) {
	t.Parallel()

	document, err := marksplice.Parse([]byte("# Hello *world*\n\ntext & more  \nnext\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var output bytes.Buffer
	if err := document.RenderHTML(&output, marksplice.HTMLRenderOptions{}); err != nil {
		t.Fatalf("RenderHTML() error = %v", err)
	}
	want := "<h1>Hello <em>world</em></h1>\n<p>text &amp; more<br />\nnext</p>\n"
	if got := output.String(); got != want {
		t.Fatalf("RenderHTML() = %q, want %q", got, want)
	}

	convenience, err := document.HTML(marksplice.HTMLRenderOptions{})
	if err != nil {
		t.Fatalf("HTML() error = %v", err)
	}
	if !bytes.Equal(convenience, output.Bytes()) {
		t.Fatalf("HTML() = %q, streamed = %q", convenience, output.Bytes())
	}
}

func TestM120PublicHTMLRendererPoliciesAreExplicit(t *testing.T) {
	t.Parallel()

	document, err := marksplice.Parse([]byte("<title>x</title>\n\n[unsafe](javascript:alert(1))\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	got, err := document.HTML(marksplice.HTMLRenderOptions{})
	if err != nil {
		t.Fatalf("HTML(default) error = %v", err)
	}
	want := "&lt;title>x&lt;/title>\n<p><a href=\"\">unsafe</a></p>\n"
	if string(got) != want {
		t.Fatalf("HTML(default) = %q, want %q", got, want)
	}

	got, err = document.HTML(marksplice.HTMLRenderOptions{
		RawHTML:    marksplice.HTMLRawEscape,
		UnsafeURLs: marksplice.HTMLUnsafeURLAllow,
		TagFilter:  marksplice.HTMLTagFilterDisabled,
	})
	if err != nil {
		t.Fatalf("HTML(explicit policies) error = %v", err)
	}
	want = "&lt;title&gt;x&lt;/title&gt;\n<p><a href=\"javascript:alert(1)\">unsafe</a></p>\n"
	if string(got) != want {
		t.Fatalf("HTML(explicit policies) = %q, want %q", got, want)
	}
}

func TestM120PublicHTMLRendererNormalizesDestinationsTitlesAndFenceLanguage(t *testing.T) {
	t.Parallel()

	source := []byte("[link](/f&ouml;&ouml; \"ti\\*tle &quot;x&quot;\") <https://example.com?find=\\*> [unsafe](javascript&#x3A;alert(1)) ![alt](/φου)\n\n```foo\\+bar\nbody\n```\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	got, err := document.HTML(marksplice.HTMLRenderOptions{})
	if err != nil {
		t.Fatalf("HTML() error = %v", err)
	}
	want := "<p><a href=\"/f%C3%B6%C3%B6\" title=\"ti*tle &quot;x&quot;\">link</a> <a href=\"https://example.com?find=%5C*\">https://example.com?find=\\*</a> <a href=\"\">unsafe</a> <img src=\"/%CF%86%CE%BF%CF%85\" alt=\"alt\" /></p>\n<pre><code class=\"language-foo+bar\">body\n</code></pre>\n"
	if string(got) != want {
		t.Fatalf("HTML() = %q, want %q", got, want)
	}
}

func TestM120PublicHTMLRendererFlattensNestedImageAltSemantics(t *testing.T) {
	t.Parallel()

	document, err := marksplice.Parse([]byte("![foo ![bar](/url)](/url2)\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	got, err := document.HTML(marksplice.HTMLRenderOptions{})
	if err != nil {
		t.Fatalf("HTML() error = %v", err)
	}
	want := "<p><img src=\"/url2\" alt=\"foo bar\" /></p>\n"
	if string(got) != want {
		t.Fatalf("HTML() = %q, want %q", got, want)
	}
}

func TestM120PublicHTMLRendererPathologicalInputIsDeterministicAndSourcePreserving(t *testing.T) {
	t.Parallel()

	source := []byte(strings.Repeat("*_~`", 16<<10) + "\n")
	original := append([]byte(nil), source...)
	document, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	first, err := document.HTML(marksplice.HTMLRenderOptions{})
	if err != nil {
		t.Fatalf("first HTML() error = %v", err)
	}
	second, err := document.HTML(marksplice.HTMLRenderOptions{})
	if err != nil {
		t.Fatalf("second HTML() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("pathological HTML rendering is nondeterministic")
	}
	if !bytes.Equal(source, original) {
		t.Fatal("HTML rendering mutated caller source")
	}
}

func TestM120PublicHTMLRendererRejectsInvalidInputsAndPropagatesWriterError(t *testing.T) {
	t.Parallel()

	var nilDocument *marksplice.Document
	if err := nilDocument.RenderHTML(&bytes.Buffer{}, marksplice.HTMLRenderOptions{}); !errors.Is(err, marksplice.ErrInvalidRender) {
		t.Fatalf("nil RenderHTML() error = %v, want ErrInvalidRender", err)
	}

	document, err := marksplice.Parse([]byte("text\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if err := document.RenderHTML(nil, marksplice.HTMLRenderOptions{}); !errors.Is(err, marksplice.ErrInvalidRender) {
		t.Fatalf("nil-writer RenderHTML() error = %v, want ErrInvalidRender", err)
	}
	invalid := marksplice.HTMLRenderOptions{RawHTML: marksplice.HTMLRawPolicy(255)}
	if err := document.RenderHTML(&bytes.Buffer{}, invalid); !errors.Is(err, marksplice.ErrInvalidRender) {
		t.Fatalf("invalid-options RenderHTML() error = %v, want ErrInvalidRender", err)
	}

	stop := errors.New("writer stopped")
	if err := document.RenderHTML(errorWriter{err: stop}, marksplice.HTMLRenderOptions{}); !errors.Is(err, stop) {
		t.Fatalf("writer RenderHTML() error = %v, want %v", err, stop)
	}
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write([]byte) (int, error) {
	return 0, w.err
}
