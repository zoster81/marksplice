package splice

import (
	"bytes"
	"errors"
	"testing"
)

func TestReplaceYAMLFrontMatterValuePreservesEnvelopeAndUnrelatedSource(t *testing.T) {
	t.Parallel()

	source := []byte("---\r\ntitle: old title  \r\ndraft: false\r\ncomplex:\r\n  nested: keep\r\n---\r\n# Heading\r\n\r\nbody\r\n")
	want := []byte("---\r\ntitle: new title  \r\ndraft: false\r\ncomplex:\r\n  nested: keep\r\n---\r\n# Heading\r\n\r\nbody\r\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	fields := nodesOfKind(doc.Nodes(), KindYAMLFrontMatterField)
	var title Node
	found := false
	for _, field := range fields {
		if field.Key == "title" {
			title = field
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("YAML title field not found; nodes = %+v", doc.Nodes())
	}
	if got := string(source[title.ContentRange.Start:title.ContentRange.End]); got != "old title" {
		t.Fatalf("title content = %q, want %q", got, "old title")
	}

	change, err := doc.PrepareReplaceFrontMatterValue(title.ID, []byte("new title"))
	if err != nil {
		t.Fatalf("PrepareReplaceFrontMatterValue() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}

	for _, paragraph := range nodesOfKind(doc.Nodes(), KindParagraph) {
		if paragraph.Range.Start < bytes.Index(source, []byte("# Heading")) {
			t.Fatalf("front-matter bytes leaked as Markdown paragraph: %+v", paragraph)
		}
	}
}

func TestReplaceTOMLFrontMatterValuePreservesQuotesCommentAndSpacing(t *testing.T) {
	t.Parallel()

	source := []byte("+++\n title = \"old title\"   # keep comment\ndraft = true\n+++\n# Heading\n")
	want := []byte("+++\n title = \"new title\"   # keep comment\ndraft = true\n+++\n# Heading\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	fields := nodesOfKind(doc.Nodes(), KindTOMLFrontMatterField)
	var title Node
	found := false
	for _, field := range fields {
		if field.Key == "title" {
			title = field
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("TOML title field not found; nodes = %+v", doc.Nodes())
	}
	change, err := doc.PrepareReplaceFrontMatterValue(title.ID, []byte("new title"))
	if err != nil {
		t.Fatalf("PrepareReplaceFrontMatterValue() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
}

func TestFrontMatterMustStartAtDocumentStartAndDuplicateKeysAreNotMutable(t *testing.T) {
	t.Parallel()

	notFrontMatter := []byte("intro\n---\ntitle: value\n---\n")
	doc, err := Parse(notFrontMatter)
	if err != nil {
		t.Fatalf("Parse(non-leading front matter) error = %v", err)
	}
	if got := len(nodesOfKind(doc.Nodes(), KindYAMLFrontMatterField)); got != 0 {
		t.Fatalf("non-leading YAML field count = %d, want 0", got)
	}

	duplicate := []byte("---\ntitle: one\ntitle: two\nunique: keep\n---\n")
	doc, err = Parse(duplicate)
	if err != nil {
		t.Fatalf("Parse(duplicate YAML keys) error = %v", err)
	}
	fields := nodesOfKind(doc.Nodes(), KindYAMLFrontMatterField)
	for _, field := range fields {
		if field.Key == "title" {
			t.Fatalf("duplicate YAML key unexpectedly targetable: %+v", field)
		}
	}
	if len(fields) != 1 || fields[0].Key != "unique" {
		t.Fatalf("unique YAML field mapping = %+v, want only unique", fields)
	}
}

func TestReplaceInlineHTMLCommentPreservesDelimitersPaddingAndCRLF(t *testing.T) {
	t.Parallel()

	source := []byte("before <!--  old comment  --> after\r\n")
	want := []byte("before <!--  new comment  --> after\r\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	comments := nodesOfKind(doc.Nodes(), KindHTMLComment)
	if len(comments) != 1 {
		t.Fatalf("HTML comment count = %d, want 1; nodes = %+v", len(comments), doc.Nodes())
	}
	change, err := doc.PrepareReplaceHTMLComment(comments[0].ID, []byte("new comment"))
	if err != nil {
		t.Fatalf("PrepareReplaceHTMLComment() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
	if _, err := doc.PrepareReplaceHTMLComment(comments[0].ID, []byte("bad -- split")); !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceHTMLComment(double hyphen) error = %v, want ErrInvalidReplacement", err)
	}
}

func TestReplaceSimpleHTMLAnchorPreservesTagAttributeStyleAndSurroundingSource(t *testing.T) {
	t.Parallel()

	source := []byte("before <a class=\"x\" id = 'old-anchor'>link</a> after\n")
	want := []byte("before <a class=\"x\" id = 'new-anchor'>link</a> after\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchors := nodesOfKind(doc.Nodes(), KindHTMLAnchor)
	if len(anchors) != 1 {
		t.Fatalf("HTML anchor count = %d, want 1; nodes = %+v", len(anchors), doc.Nodes())
	}
	if anchors[0].HTMLAttribute != "id" {
		t.Fatalf("HTML anchor attribute = %q, want id", anchors[0].HTMLAttribute)
	}
	change, err := doc.PrepareReplaceHTMLAnchor(anchors[0].ID, []byte("new-anchor"))
	if err != nil {
		t.Fatalf("PrepareReplaceHTMLAnchor() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
	}
	if _, err := doc.PrepareReplaceHTMLAnchor(anchors[0].ID, []byte("bad'anchor")); !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceHTMLAnchor(quote) error = %v, want ErrInvalidReplacement", err)
	}
}

func TestHTMLBlockIsMappedAsOpaqueSourceRegion(t *testing.T) {
	t.Parallel()

	source := []byte("<div data-x=\"1\">\r\n*not markdown*\r\n</div>\r\n\r\nafter\r\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	opaque := nodesOfKind(doc.Nodes(), KindHTMLOpaque)
	if len(opaque) == 0 {
		t.Fatalf("opaque HTML node not found; nodes = %+v", doc.Nodes())
	}
	blockEnd := bytes.Index(source, []byte("\r\n\r\nafter"))
	if blockEnd < 0 {
		t.Fatal("opaque HTML fixture boundary not found")
	}
	found := false
	for _, node := range opaque {
		if node.Range.Start == 0 && node.Range.End >= blockEnd {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("opaque HTML block range not mapped; nodes = %+v", opaque)
	}
}

func TestPrepareReplaceAllowsRawHTMLObservationsInsideParagraph(t *testing.T) {
	t.Parallel()

	source := []byte("old paragraph\n")
	replacement := []byte("new <!-- valid --> <a id=\"anchor\">text</a>")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(paragraphs) != 1 {
		t.Fatalf("paragraph count = %d, want 1", len(paragraphs))
	}
	change, err := doc.PrepareReplace(paragraphs[0].ID, replacement)
	if err != nil {
		t.Fatalf("PrepareReplace() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, append(replacement, '\n')) {
		t.Fatalf("result = %q, want %q", got, append(replacement, '\n'))
	}
}

func TestPrepareReplaceFrontMatterAndHTMLRejectWrongTargetOrMultiline(t *testing.T) {
	t.Parallel()

	source := []byte("---\ntitle: old\n---\n\nparagraph <!-- old -->\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	fields := nodesOfKind(doc.Nodes(), KindYAMLFrontMatterField)
	comments := nodesOfKind(doc.Nodes(), KindHTMLComment)
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(fields) != 1 || len(comments) != 1 || len(paragraphs) != 1 {
		t.Fatalf("field/comment/paragraph counts = %d/%d/%d, want 1/1/1", len(fields), len(comments), len(paragraphs))
	}
	if _, err := doc.PrepareReplaceFrontMatterValue(fields[0].ID, []byte("one\ntwo")); !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceFrontMatterValue(multiline) error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareReplaceFrontMatterValue(paragraphs[0].ID, []byte("new")); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceFrontMatterValue(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareReplaceHTMLComment(paragraphs[0].ID, []byte("new")); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceHTMLComment(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}
