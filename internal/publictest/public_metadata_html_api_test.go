package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicFrontMatterAndHTMLDetailsPreserveSource(t *testing.T) {
	t.Parallel()

	t.Run("YAML front matter field", testPublicYAMLFrontMatterFieldPreservesSource)
	t.Run("TOML front matter field", testPublicTOMLFrontMatterFieldPreservesSource)
	t.Run("HTML comment", testPublicHTMLCommentPreservesSource)
	t.Run("HTML anchor id", testPublicHTMLAnchorIDPreservesSource)
	t.Run("HTML anchor name", testPublicHTMLAnchorNameDetail)
}

func testPublicYAMLFrontMatterFieldPreservesSource(t *testing.T) {
	t.Parallel()

	source := []byte("---\r\ntitle: 'old title'  # keep\r\ncount: 2\r\n---\r\n\r\nbody\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	field := findNodeOfKind(t, doc, marksplice.KindFrontMatterField)
	detail, ok := doc.FrontMatterField(field.ID())
	if !ok {
		t.Fatalf("FrontMatterField(%q) ok = false", field.ID())
	}
	if detail.Key() != "title" || detail.Format() != marksplice.FrontMatterFormatYAML {
		t.Fatalf("front matter detail = key %q format %v, want title/YAML", detail.Key(), detail.Format())
	}
	if got := string(source[detail.Range().Start:detail.Range().End]); got != "old title" {
		t.Fatalf("front matter range bytes = %q, want old title", got)
	}

	change, err := doc.PrepareReplaceFrontMatterValue(field.ID(), []byte("new title"))
	if err != nil {
		t.Fatalf("PrepareReplaceFrontMatterValue() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("---\r\ntitle: 'new title'  # keep\r\ncount: 2\r\n---\r\n\r\nbody\r\n")
	assertReplacementPreservesOutsideRange(t, source, got, want, detail.Range(), []byte("new title"))
}

func testPublicTOMLFrontMatterFieldPreservesSource(t *testing.T) {
	t.Parallel()

	source := []byte("+++\ntitle = \"old title\" # keep\nenabled = true\n+++\n\nbody\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var field marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindFrontMatterField {
			continue
		}
		detail, ok := doc.FrontMatterField(node.ID())
		if ok && detail.Key() == "title" {
			field = node
			break
		}
	}
	if field.ID().String() == "" {
		t.Fatal("public TOML title field not found")
	}
	detail, _ := doc.FrontMatterField(field.ID())
	if detail.Format() != marksplice.FrontMatterFormatTOML {
		t.Fatalf("FrontMatterField.Format() = %v, want TOML", detail.Format())
	}
	change, err := doc.PrepareReplaceFrontMatterValue(field.ID(), []byte("new title"))
	if err != nil {
		t.Fatalf("PrepareReplaceFrontMatterValue() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("+++\ntitle = \"new title\" # keep\nenabled = true\n+++\n\nbody\n")
	assertReplacementPreservesOutsideRange(t, source, got, want, detail.Range(), []byte("new title"))
}

func testPublicHTMLCommentPreservesSource(t *testing.T) {
	t.Parallel()

	source := []byte("before <!--  old comment  --> after\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	node := findNodeOfKind(t, doc, marksplice.KindHTMLComment)
	detail, ok := doc.HTMLComment(node.ID())
	if !ok {
		t.Fatalf("HTMLComment(%q) ok = false", node.ID())
	}
	if got := string(source[detail.Range().Start:detail.Range().End]); got != "old comment" {
		t.Fatalf("HTML comment range bytes = %q, want old comment", got)
	}
	change, err := doc.PrepareReplaceHTMLComment(node.ID(), []byte("new comment"))
	if err != nil {
		t.Fatalf("PrepareReplaceHTMLComment() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("before <!--  new comment  --> after\r\n")
	assertReplacementPreservesOutsideRange(t, source, got, want, detail.Range(), []byte("new comment"))
}

func testPublicHTMLAnchorIDPreservesSource(t *testing.T) {
	t.Parallel()

	source := []byte("before <A class='x' ID=\"old-anchor\">text</A> after\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	node := findNodeOfKind(t, doc, marksplice.KindHTMLAnchor)
	detail, ok := doc.HTMLAnchor(node.ID())
	if !ok {
		t.Fatalf("HTMLAnchor(%q) ok = false", node.ID())
	}
	if detail.Attribute() != marksplice.HTMLAnchorAttributeID {
		t.Fatalf("HTMLAnchor.Attribute() = %v, want ID", detail.Attribute())
	}
	if got := string(source[detail.Range().Start:detail.Range().End]); got != "old-anchor" {
		t.Fatalf("HTML anchor range bytes = %q, want old-anchor", got)
	}
	change, err := doc.PrepareReplaceHTMLAnchor(node.ID(), []byte("new-anchor"))
	if err != nil {
		t.Fatalf("PrepareReplaceHTMLAnchor() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("before <A class='x' ID=\"new-anchor\">text</A> after\n")
	assertReplacementPreservesOutsideRange(t, source, got, want, detail.Range(), []byte("new-anchor"))
}

func testPublicHTMLAnchorNameDetail(t *testing.T) {
	t.Parallel()

	source := []byte("<a NAME='old-name'>text</a>\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	node := findNodeOfKind(t, doc, marksplice.KindHTMLAnchor)
	detail, ok := doc.HTMLAnchor(node.ID())
	if !ok || detail.Attribute() != marksplice.HTMLAnchorAttributeName {
		t.Fatalf("HTMLAnchor detail = %+v, %v; want name attribute", detail, ok)
	}
}

func TestPublicFrontMatterAndHTMLFilterUnsupportedShapesAndPreserveErrors(t *testing.T) {
	t.Parallel()

	duplicateOnly, err := marksplice.Parse([]byte("---\ntitle: one\ntitle: two\n---\n"))
	if err != nil {
		t.Fatalf("Parse(duplicate front matter) error = %v", err)
	}
	for _, node := range duplicateOnly.Nodes() {
		if node.Kind() == marksplice.KindFrontMatterField {
			t.Fatal("duplicate-only front matter was promoted publicly")
		}
	}

	opaqueHTML, err := marksplice.Parse([]byte("before <span id=\"x\">text</span> after\n"))
	if err != nil {
		t.Fatalf("Parse(opaque HTML) error = %v", err)
	}
	for _, node := range opaqueHTML.Nodes() {
		if node.Kind() == marksplice.KindHTMLAnchor || node.Kind() == marksplice.KindHTMLComment {
			t.Fatalf("unsupported raw HTML kind %v was promoted publicly", node.Kind())
		}
	}

	source := []byte("---\ntitle: old\n---\n\nparagraph <!-- old --> <a id=\"old-anchor\">x</a>\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	field := findNodeOfKind(t, doc, marksplice.KindFrontMatterField)
	comment := findNodeOfKind(t, doc, marksplice.KindHTMLComment)
	anchor := findNodeOfKind(t, doc, marksplice.KindHTMLAnchor)
	paragraph := findNodeOfKind(t, doc, marksplice.KindParagraph)

	if _, err := doc.PrepareReplaceFrontMatterValue(marksplice.NodeID{}, []byte("new")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareReplaceFrontMatterValue(zero ID) error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareReplaceFrontMatterValue(paragraph.ID(), []byte("new")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceFrontMatterValue(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareReplaceFrontMatterValue(field.ID(), []byte("one\ntwo")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceFrontMatterValue(multiline) error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareReplaceHTMLComment(comment.ID(), []byte("bad -- split")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceHTMLComment(double hyphen) error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareReplaceHTMLAnchor(anchor.ID(), []byte("bad\"anchor")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceHTMLAnchor(quote) error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareReplaceHTMLComment(paragraph.ID(), []byte("new")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceHTMLComment(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}

func findNodeOfKind(t *testing.T, doc *marksplice.Document, kind marksplice.Kind) marksplice.Node {
	t.Helper()
	for _, node := range doc.Nodes() {
		if node.Kind() == kind {
			return node
		}
	}
	t.Fatalf("public node kind %v not found; nodes = %+v", kind, doc.Nodes())
	return marksplice.Node{}
}

func assertReplacementPreservesOutsideRange(t *testing.T, source, got, want []byte, sourceRange marksplice.Range, replacement []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
	prefix := source[:sourceRange.Start]
	suffix := source[sourceRange.End:]
	if !bytes.Equal(got[:len(prefix)], prefix) || !bytes.Equal(got[len(prefix)+len(replacement):], suffix) {
		t.Fatal("replacement modified bytes outside typed range")
	}
}
