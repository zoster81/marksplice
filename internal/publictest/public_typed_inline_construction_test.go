package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicDocumentBuilderTypedTextConstructionEscapesGFM(t *testing.T) {
	t.Parallel()

	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendHeadingContent(2, marksplice.TextInline("Title *literal*")); err != nil {
		t.Fatalf("AppendHeadingContent() error = %v", err)
	}
	if err := builder.AppendParagraphContent(
		marksplice.TextInline("Use [label](dest), `code`, "),
		marksplice.TextInline("#hash & Unicode π."),
	); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}
	if err := builder.AppendBlockquoteContent(marksplice.TextInline("> quoted?")); err != nil {
		t.Fatalf("AppendBlockquoteContent() error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("## Title \\*literal\\*\n\nUse \\[label\\]\\(dest\\)\\, \\`code\\`\\, \\#hash \\& Unicode π\\.\n\n> \\> quoted\\?\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindEmphasis, marksplice.KindStrong, marksplice.KindCodeSpan, marksplice.KindInlineLink, marksplice.KindImage:
			t.Fatalf("typed text unexpectedly created inline syntax kind %v", node.Kind())
		}
	}
}

func TestPublicTypedTextInlineCompositionPreservesSemanticSpacing(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendParagraphContent(
		marksplice.TextInline("Hello "),
		marksplice.TextInline("world!"),
	); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	if want := []byte("Hello world\\!\n"); !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicDocumentBuilderTypedStructuredInlineConstruction(t *testing.T) {
	t.Parallel()

	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendParagraphContent(
		marksplice.TextInline("Before "),
		marksplice.EmphasisInline(marksplice.TextInline("em")),
		marksplice.TextInline(" and "),
		marksplice.StrongInline(marksplice.TextInline("strong")),
		marksplice.TextInline(" plus "),
		marksplice.CodeInline("a`b"),
		marksplice.TextInline("."),
	); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("Before *em* and **strong** plus ``a`b``\\.\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	seen := map[marksplice.Kind]string{}
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindEmphasis:
			detail, ok := doc.Emphasis(node.ID())
			if !ok {
				t.Fatal("Emphasis() ok = false")
			}
			seen[node.Kind()] = string(got[detail.Range().Start:detail.Range().End])
		case marksplice.KindStrong:
			detail, ok := doc.Strong(node.ID())
			if !ok {
				t.Fatal("Strong() ok = false")
			}
			seen[node.Kind()] = string(got[detail.Range().Start:detail.Range().End])
		case marksplice.KindCodeSpan:
			detail, ok := doc.CodeSpan(node.ID())
			if !ok {
				t.Fatal("CodeSpan() ok = false")
			}
			seen[node.Kind()] = string(got[detail.Range().Start:detail.Range().End])
		}
	}
	if seen[marksplice.KindEmphasis] != "em" || seen[marksplice.KindStrong] != "strong" || seen[marksplice.KindCodeSpan] != "a`b" {
		t.Fatalf("typed structured inline proof = %v", seen)
	}
}

func TestPublicTypedStructuredInlineNestingConstruction(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendParagraphContent(
		marksplice.EmphasisInline(
			marksplice.TextInline("before "),
			marksplice.CodeInline("a`b"),
			marksplice.TextInline(" after"),
		),
		marksplice.TextInline(" "),
		marksplice.StrongInline(marksplice.EmphasisInline(marksplice.TextInline("inside"))),
		marksplice.TextInline(" "),
		marksplice.EmphasisInline(marksplice.StrikethroughInline(marksplice.TextInline("gone"))),
	); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("*before ``a`b`` after* **_inside_** *~~gone~~*\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicTypedStructuredInlineNestingBoundsAndRejectsUnsupportedShapes(t *testing.T) {
	t.Parallel()

	inline := marksplice.CodeInline("x")
	for depth := 0; depth < 64; depth++ {
		children := []marksplice.Inline{
			marksplice.TextInline("left "),
			inline,
			marksplice.TextInline(" right"),
		}
		if depth%2 == 0 {
			inline = marksplice.EmphasisInline(children...)
		} else {
			inline = marksplice.StrongInline(children...)
		}
	}
	var accepted marksplice.DocumentBuilder
	if err := accepted.AppendParagraphContent(inline); err != nil {
		t.Fatalf("AppendParagraphContent(depth=64) error = %v", err)
	}
	var linkAccepted marksplice.DocumentBuilder
	if err := linkAccepted.AppendParagraphContent(marksplice.LinkInline("target", inline)); err != nil {
		t.Fatalf("AppendParagraphContent(link label depth=64) error = %v", err)
	}

	tooDeep := marksplice.EmphasisInline(inline)
	var rejected marksplice.DocumentBuilder
	if err := rejected.AppendParagraphContent(tooDeep); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendParagraphContent(depth=65) error = %v, want ErrInvalidConstruction", err)
	}
	var linkRejected marksplice.DocumentBuilder
	if err := linkRejected.AppendParagraphContent(marksplice.LinkInline("target", tooDeep)); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendParagraphContent(link label depth=65) error = %v, want ErrInvalidConstruction", err)
	}

	invalid := []marksplice.Inline{
		marksplice.StrikethroughInline(marksplice.StrikethroughInline(marksplice.TextInline("nested"))),
		marksplice.EmphasisInline(marksplice.LinkInline("target", marksplice.TextInline("label"))),
		marksplice.StrongInline(marksplice.ImageInline("target", marksplice.TextInline("alt"))),
		marksplice.EmphasisInline(marksplice.AutoLinkInline("user@example.test")),
	}
	for _, candidate := range invalid {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendParagraphContent(candidate); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendParagraphContent(unsupported nested inline) error = %v, want ErrInvalidConstruction", err)
		}
	}
}

func TestPublicDocumentBuilderTypedLinkAndImageConstruction(t *testing.T) {
	t.Parallel()

	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendParagraphContent(
		marksplice.TextInline("See "),
		marksplice.LinkInline("https://example.test/a?b=1", marksplice.TextInline("docs [v1]")),
		marksplice.TextInline(" and "),
		marksplice.ImageInline("images/logo.png", marksplice.TextInline("logo *literal*")),
		marksplice.TextInline("."),
	); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("See [docs \\[v1\\]](<https://example.test/a?b=1>) and ![logo \\*literal\\*](<images/logo.png>)\\.\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	seen := map[marksplice.Kind]string{}
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindInlineLink:
			detail, ok := doc.InlineLink(node.ID())
			if !ok {
				t.Fatal("InlineLink() ok = false")
			}
			seen[node.Kind()] = string(got[detail.Range().Start:detail.Range().End])
		case marksplice.KindImage:
			detail, ok := doc.Image(node.ID())
			if !ok {
				t.Fatal("Image() ok = false")
			}
			seen[node.Kind()] = string(got[detail.Range().Start:detail.Range().End])
		}
	}
	if seen[marksplice.KindInlineLink] != "https://example.test/a?b=1" || seen[marksplice.KindImage] != "images/logo.png" {
		t.Fatalf("typed link/image destination proof = %v", seen)
	}
}

func TestPublicDocumentBuilderTypedLinkAndImageTitleConstruction(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendParagraphContent(
		marksplice.LinkInlineWithTitle("https://example.test/docs", "Guide v1", marksplice.TextInline("docs")),
		marksplice.TextInline(" and "),
		marksplice.ImageInlineWithTitle("images/logo.png", "Project logo", marksplice.TextInline("logo")),
	); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("[docs](<https://example.test/docs> \"Guide v1\") and ![logo](<images/logo.png> \"Project logo\")\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicTypedLinkAndImageStructuredLabelAndAltConstruction(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendParagraphContent(
		marksplice.LinkInline(
			"https://example.test/docs",
			marksplice.TextInline("read "),
			marksplice.StrongInline(marksplice.TextInline("docs")),
			marksplice.TextInline(" "),
			marksplice.CodeInline("v1"),
		),
		marksplice.TextInline(" and "),
		marksplice.ImageInlineWithTitle(
			"images/logo.png",
			"Project logo",
			marksplice.EmphasisInline(marksplice.TextInline("project")),
			marksplice.TextInline(" "),
			marksplice.StrikethroughInline(marksplice.TextInline("old")),
		),
	); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("[read **docs** `v1`](<https://example.test/docs>) and ![*project* ~~old~~](<images/logo.png> \"Project logo\")\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindInlineLink || node.Kind() == marksplice.KindImage {
			t.Fatalf("structured generated link/image was unexpectedly promoted by ordinary Parse: %v", node.Kind())
		}
	}
}

func TestPublicTypedLinkAndImageTitleConstructionRejectsUnsafeTitles(t *testing.T) {
	t.Parallel()

	invalidTitles := []string{
		"",
		"line\nbreak",
		"contains\x00nul",
		string([]byte{0xff}),
		`quote"title`,
		`escape\\title`,
		"entity&amp;title",
	}
	for _, title := range invalidTitles {
		for _, inline := range []marksplice.Inline{
			marksplice.LinkInlineWithTitle("target", title, marksplice.TextInline("label")),
			marksplice.ImageInlineWithTitle("target", title, marksplice.TextInline("alt")),
		} {
			var builder marksplice.DocumentBuilder
			if err := builder.AppendParagraphContent(inline); !errors.Is(err, marksplice.ErrInvalidConstruction) {
				t.Fatalf("AppendParagraphContent(unsafe title %q) error = %v, want ErrInvalidConstruction", title, err)
			}
		}
	}
}

func TestPublicDocumentBuilderTypedStrikethroughConstruction(t *testing.T) {
	t.Parallel()

	children := []marksplice.Inline{marksplice.TextInline("removed")}
	strike := marksplice.StrikethroughInline(children...)
	children[0] = marksplice.TextInline("mutated")

	var builder marksplice.DocumentBuilder
	if err := builder.AppendParagraphContent(
		marksplice.TextInline("Keep "),
		strike,
		marksplice.TextInline("."),
	); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	if want := []byte("Keep ~~removed~~\\.\n"); !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindStrikethrough {
			continue
		}
		detail, ok := doc.Strikethrough(node.ID())
		if !ok {
			t.Fatal("Strikethrough() ok = false")
		}
		if content := string(got[detail.Range().Start:detail.Range().End]); content != "removed" {
			t.Fatalf("strikethrough content = %q, want removed", content)
		}
		return
	}
	t.Fatal("generated strikethrough was not promoted")
}

func TestPublicDocumentBuilderTypedAutoLinkConstruction(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendParagraphContent(
		marksplice.AutoLinkInline("https://example.test/path?q=1"),
		marksplice.TextInline(" "),
		marksplice.AutoLinkInline("user@example.test"),
	); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("<https://example.test/path?q=1> <user@example.test>\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	values := []string{}
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindAutoLink {
			continue
		}
		detail, ok := doc.AutoLink(node.ID())
		if !ok {
			t.Fatal("AutoLink() ok = false")
		}
		values = append(values, string(got[detail.Range().Start:detail.Range().End]))
	}
	if len(values) != 2 || values[0] != "https://example.test/path?q=1" || values[1] != "user@example.test" {
		t.Fatalf("autolink values = %v", values)
	}
}

func TestPublicTypedBareAutoLinkConstruction(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	values := []string{
		"https://example.test/path?q=1",
		"www.example.test/path",
		"user@example.test",
		"mailto:foo@bar.baz",
		"xmpp:foo@bar.baz/resource",
	}
	for index, value := range values {
		if index != 0 {
			if err := builder.AppendParagraphContent(marksplice.TextInline("separator")); err != nil {
				t.Fatalf("AppendParagraphContent(separator) error = %v", err)
			}
		}
		if err := builder.AppendParagraphContent(marksplice.BareAutoLinkInline(value)); err != nil {
			t.Fatalf("AppendParagraphContent(BareAutoLinkInline(%q)) error = %v", value, err)
		}
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("https://example.test/path?q=1\n\nseparator\n\nwww.example.test/path\n\nseparator\n\nuser@example.test\n\nseparator\n\nmailto:foo@bar.baz\n\nseparator\n\nxmpp:foo@bar.baz/resource\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicTypedBareAutoLinkConstructionRejectsNonExactTokens(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "not a link", "ftp://example.test", "https://example.test/path).", "line\nbreak@example.test"} {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendParagraphContent(marksplice.BareAutoLinkInline(value)); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendParagraphContent(BareAutoLinkInline(%q)) error = %v, want ErrInvalidConstruction", value, err)
		}
	}
}

func TestPublicTypedAutoLinkConstructionRejectsUnsafeShapes(t *testing.T) {
	t.Parallel()

	invalid := []marksplice.Inline{
		marksplice.AutoLinkInline(""),
		marksplice.AutoLinkInline("not a link"),
		marksplice.AutoLinkInline("www.example.test"),
		marksplice.AutoLinkInline("line\nbreak@example.test"),
		marksplice.AutoLinkInline("left<angle@example.test"),
		marksplice.AutoLinkInline("right>angle@example.test"),
		marksplice.AutoLinkInline("contains\x00nul@example.test"),
		marksplice.AutoLinkInline(string([]byte{0xff})),
	}
	for _, inline := range invalid {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendParagraphContent(inline); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendParagraphContent(unsafe autolink) error = %v, want ErrInvalidConstruction", err)
		}
	}
}

func TestPublicTypedStrikethroughConstructionRejectsUnsafeShapes(t *testing.T) {
	t.Parallel()

	invalid := []marksplice.Inline{
		marksplice.StrikethroughInline(),
		marksplice.StrikethroughInline(marksplice.TextInline(" leading")),
		marksplice.StrikethroughInline(marksplice.TextInline("trailing ")),
	}
	for _, inline := range invalid {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendParagraphContent(inline); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendParagraphContent(unsafe strikethrough) error = %v, want ErrInvalidConstruction", err)
		}
	}
}

func TestPublicTypedLinkAndImageConstructionCopyChildrenAndRejectUnsafeShapes(t *testing.T) {
	t.Parallel()

	label := []marksplice.Inline{marksplice.TextInline("original")}
	link := marksplice.LinkInline("target", label...)
	label[0] = marksplice.TextInline("mutated")
	var copied marksplice.DocumentBuilder
	if err := copied.AppendParagraphContent(link); err != nil {
		t.Fatalf("AppendParagraphContent(copied link) error = %v", err)
	}
	got, err := copied.Markdown()
	if err != nil || !bytes.Equal(got, []byte("[original](<target>)\n")) {
		t.Fatalf("copied link Markdown() = %q, %v", got, err)
	}

	invalid := []marksplice.Inline{
		marksplice.LinkInline("", marksplice.TextInline("label")),
		marksplice.LinkInline("line\nbreak", marksplice.TextInline("label")),
		marksplice.LinkInline("contains\x00nul", marksplice.TextInline("label")),
		marksplice.LinkInline(string([]byte{0xff}), marksplice.TextInline("label")),
		marksplice.LinkInline("left<angle", marksplice.TextInline("label")),
		marksplice.LinkInline("right>angle", marksplice.TextInline("label")),
		marksplice.LinkInline(`escape\\path`, marksplice.TextInline("label")),
		marksplice.LinkInline("entity&value", marksplice.TextInline("label")),
		marksplice.LinkInline("target"),
		marksplice.ImageInline("target"),
		marksplice.LinkInline("target", marksplice.LinkInline("nested", marksplice.TextInline("label"))),
		marksplice.LinkInline("target", marksplice.ImageInline("nested", marksplice.TextInline("alt"))),
		marksplice.ImageInline("target", marksplice.AutoLinkInline("user@example.test")),
		marksplice.LinkInline("target", marksplice.ReferenceLinkInline("doc", marksplice.TextInline("reference"))),
	}
	for _, inline := range invalid {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendParagraphContent(inline); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendParagraphContent(unsafe link/image) error = %v, want ErrInvalidConstruction", err)
		}
	}
}

func TestPublicTypedStructuredInlineConstructionCopiesChildrenAndRejectsUnsafeShapes(t *testing.T) {
	t.Parallel()

	children := []marksplice.Inline{marksplice.TextInline("original")}
	emphasis := marksplice.EmphasisInline(children...)
	children[0] = marksplice.TextInline("mutated")
	var copied marksplice.DocumentBuilder
	if err := copied.AppendParagraphContent(emphasis); err != nil {
		t.Fatalf("AppendParagraphContent(copied emphasis) error = %v", err)
	}
	got, err := copied.Markdown()
	if err != nil || !bytes.Equal(got, []byte("*original*\n")) {
		t.Fatalf("copied emphasis Markdown() = %q, %v", got, err)
	}

	invalid := []marksplice.Inline{
		marksplice.CodeInline(""),
		marksplice.CodeInline(" leading"),
		marksplice.CodeInline("trailing "),
		marksplice.CodeInline("`leading"),
		marksplice.CodeInline("trailing`"),
		marksplice.CodeInline("line\nbreak"),
		marksplice.EmphasisInline(),
		marksplice.StrongInline(),
		marksplice.EmphasisInline(marksplice.TextInline(" leading")),
	}
	for _, inline := range invalid {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendParagraphContent(inline); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendParagraphContent(unsafe structured inline) error = %v, want ErrInvalidConstruction", err)
		}
	}
}

func TestPublicTypedTextConstructionRejectsInvalidInlineValues(t *testing.T) {
	t.Parallel()

	invalid := []marksplice.Inline{
		{},
		marksplice.TextInline(""),
		marksplice.TextInline("line one\nline two"),
		marksplice.TextInline("line one\rline two"),
		marksplice.TextInline("contains\x00nul"),
		marksplice.TextInline(string([]byte{0xff})),
	}
	for _, inline := range invalid {
		for _, appendInline := range []func(*marksplice.DocumentBuilder) error{
			func(builder *marksplice.DocumentBuilder) error { return builder.AppendHeadingContent(1, inline) },
			func(builder *marksplice.DocumentBuilder) error { return builder.AppendParagraphContent(inline) },
			func(builder *marksplice.DocumentBuilder) error { return builder.AppendBlockquoteContent(inline) },
		} {
			var builder marksplice.DocumentBuilder
			if err := appendInline(&builder); !errors.Is(err, marksplice.ErrInvalidConstruction) {
				t.Fatalf("typed append with invalid inline error = %v, want ErrInvalidConstruction", err)
			}
			if got, err := builder.Markdown(); err != nil || len(got) != 0 {
				t.Fatalf("builder after rejected typed inline Markdown() = %q, %v; want empty, nil", got, err)
			}
		}
	}

	var builder marksplice.DocumentBuilder
	if err := builder.AppendParagraphContent(); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendParagraphContent(empty) error = %v, want ErrInvalidConstruction", err)
	}
	if err := builder.AppendHeadingContent(0, marksplice.TextInline("title")); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendHeadingContent(level=0) error = %v, want ErrInvalidConstruction", err)
	}

	var nilBuilder *marksplice.DocumentBuilder
	if err := nilBuilder.AppendParagraphContent(marksplice.TextInline("text")); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil AppendParagraphContent() error = %v, want ErrInvalidConstruction", err)
	}
}
