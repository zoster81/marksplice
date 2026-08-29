package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM121StandaloneHTMLDocumentMapsReviewedFrontMatterMetadata(t *testing.T) {
	t.Parallel()

	source := []byte("---\ntitle: \"Project <Guide>\"\ndescription: Source & structure\nauthor: 'Ada & Bob'\nlang: en-US\nignored: value\n---\n# Hello\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	got, err := document.HTMLDocument(marksplice.HTMLDocumentOptions{})
	if err != nil {
		t.Fatalf("HTMLDocument() error = %v", err)
	}
	want := "<!doctype html>\n<html lang=\"en-US\">\n<head>\n<meta charset=\"utf-8\">\n<title>Project &lt;Guide&gt;</title>\n<meta name=\"description\" content=\"Source &amp; structure\">\n<meta name=\"author\" content=\"Ada &amp; Bob\">\n</head>\n<body>\n<h1>Hello</h1>\n</body>\n</html>\n"
	if string(got) != want {
		t.Fatalf("HTMLDocument() = %q, want %q", got, want)
	}

	var streamed bytes.Buffer
	if err := document.RenderHTMLDocument(&streamed, marksplice.DefaultHTMLDocumentOptions()); err != nil {
		t.Fatalf("RenderHTMLDocument() error = %v", err)
	}
	if !bytes.Equal(streamed.Bytes(), got) {
		t.Fatalf("streamed = %q, buffered = %q", streamed.Bytes(), got)
	}
}

func TestM121StandaloneHTMLDocumentMetadataIsNarrowAndOptional(t *testing.T) {
	t.Parallel()

	source := []byte("---\ntitle: \"line\\nbreak\"\ndescription: safe <value>\nlang: en US\n---\ntext\n")
	document, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	got, err := document.HTMLDocument(marksplice.HTMLDocumentOptions{})
	if err != nil {
		t.Fatalf("HTMLDocument(default) error = %v", err)
	}
	want := "<!doctype html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n<meta name=\"description\" content=\"safe &lt;value&gt;\">\n</head>\n<body>\n<p>text</p>\n</body>\n</html>\n"
	if string(got) != want {
		t.Fatalf("HTMLDocument(default) = %q, want %q", got, want)
	}

	got, err = document.HTMLDocument(marksplice.HTMLDocumentOptions{Metadata: marksplice.HTMLMetadataOmit})
	if err != nil {
		t.Fatalf("HTMLDocument(omit metadata) error = %v", err)
	}
	want = "<!doctype html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n</head>\n<body>\n<p>text</p>\n</body>\n</html>\n"
	if string(got) != want {
		t.Fatalf("HTMLDocument(omit metadata) = %q, want %q", got, want)
	}
}

func TestM121StandaloneHTMLDocumentReusesFragmentPoliciesAndErrors(t *testing.T) {
	t.Parallel()

	document, err := marksplice.Parse([]byte("<title>x</title>\n\n[unsafe](javascript:alert(1))\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	options := marksplice.HTMLDocumentOptions{Body: marksplice.HTMLRenderOptions{RawHTML: marksplice.HTMLRawEscape, UnsafeURLs: marksplice.HTMLUnsafeURLAllow, TagFilter: marksplice.HTMLTagFilterDisabled}}
	got, err := document.HTMLDocument(options)
	if err != nil {
		t.Fatalf("HTMLDocument() error = %v", err)
	}
	wantBody := "&lt;title&gt;x&lt;/title&gt;\n<p><a href=\"javascript:alert(1)\">unsafe</a></p>\n"
	want := "<!doctype html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n</head>\n<body>\n" + wantBody + "</body>\n</html>\n"
	if string(got) != want {
		t.Fatalf("HTMLDocument() = %q, want %q", got, want)
	}

	var nilDocument *marksplice.Document
	if err := nilDocument.RenderHTMLDocument(&bytes.Buffer{}, marksplice.HTMLDocumentOptions{}); !errors.Is(err, marksplice.ErrInvalidRender) {
		t.Fatalf("nil RenderHTMLDocument() error = %v, want ErrInvalidRender", err)
	}
	if err := document.RenderHTMLDocument(nil, marksplice.HTMLDocumentOptions{}); !errors.Is(err, marksplice.ErrInvalidRender) {
		t.Fatalf("nil-writer RenderHTMLDocument() error = %v, want ErrInvalidRender", err)
	}
	invalid := marksplice.HTMLDocumentOptions{Metadata: marksplice.HTMLMetadataPolicy(255)}
	if err := document.RenderHTMLDocument(&bytes.Buffer{}, invalid); !errors.Is(err, marksplice.ErrInvalidRender) {
		t.Fatalf("invalid-options RenderHTMLDocument() error = %v, want ErrInvalidRender", err)
	}

	stop := errors.New("writer stopped")
	if err := document.RenderHTMLDocument(errorWriter{err: stop}, marksplice.HTMLDocumentOptions{}); !errors.Is(err, stop) {
		t.Fatalf("writer RenderHTMLDocument() error = %v, want %v", err, stop)
	}
}

func TestM121StandaloneHTMLDocumentHandlesTOMLAndDuplicateMetadataFailClosed(t *testing.T) {
	t.Parallel()

	toml := []byte("+++\ntitle = \"Top & <safe>\"\ndescription = 'Useful docs'\nlang = \"it-IT\"\n[params]\nauthor = 'Nested Author'\n+++\n# Body\n")
	document, err := marksplice.Parse(toml)
	if err != nil {
		t.Fatalf("Parse(TOML) error = %v", err)
	}
	got, err := document.HTMLDocument(marksplice.HTMLDocumentOptions{})
	if err != nil {
		t.Fatalf("HTMLDocument(TOML) error = %v", err)
	}
	want := "<!doctype html>\n<html lang=\"it-IT\">\n<head>\n<meta charset=\"utf-8\">\n<title>Top &amp; &lt;safe&gt;</title>\n<meta name=\"description\" content=\"Useful docs\">\n</head>\n<body>\n<h1>Body</h1>\n</body>\n</html>\n"
	if string(got) != want {
		t.Fatalf("HTMLDocument(TOML) = %q, want %q", got, want)
	}

	duplicate := []byte("---\ntitle: first\ntitle: second\ndescription: kept\n---\ntext\n")
	document, err = marksplice.Parse(duplicate)
	if err != nil {
		t.Fatalf("Parse(duplicate) error = %v", err)
	}
	got, err = document.HTMLDocument(marksplice.HTMLDocumentOptions{})
	if err != nil {
		t.Fatalf("HTMLDocument(duplicate) error = %v", err)
	}
	want = "<!doctype html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n<meta name=\"description\" content=\"kept\">\n</head>\n<body>\n<p>text</p>\n</body>\n</html>\n"
	if string(got) != want {
		t.Fatalf("HTMLDocument(duplicate) = %q, want %q", got, want)
	}
}

func TestM121StandaloneHTMLDocumentRejectsAmbiguousMetadataValuesWithoutRejectingDocument(t *testing.T) {
	t.Parallel()

	source := append([]byte("---\ntitle: 'can''t decode without YAML semantics'\nauthor: \"escaped\\tvalue\"\nlang: en--US\ndescription: valid value\nopaque: "), 0xff)
	source = append(source, []byte("\n---\ntext\n")...)
	original := append([]byte(nil), source...)
	document, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	first, err := document.HTMLDocument(marksplice.HTMLDocumentOptions{})
	if err != nil {
		t.Fatalf("HTMLDocument(first) error = %v", err)
	}
	second, err := document.HTMLDocument(marksplice.HTMLDocumentOptions{})
	if err != nil {
		t.Fatalf("HTMLDocument(second) error = %v", err)
	}
	want := "<!doctype html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n<meta name=\"description\" content=\"valid value\">\n</head>\n<body>\n<p>text</p>\n</body>\n</html>\n"
	if string(first) != want || !bytes.Equal(first, second) {
		t.Fatalf("HTMLDocument() first=%q second=%q, want %q", first, second, want)
	}
	if !bytes.Equal(source, original) {
		t.Fatal("standalone rendering mutated caller source")
	}
}
