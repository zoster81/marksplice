package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestFrontMatterFieldLifecyclePreservesEnvelopeAndBody(t *testing.T) {
	t.Parallel()

	source := []byte("---\r\ntitle: \"Old\"\r\nauthor: \"Ada\"\r\n---\r\n\r\n# Body\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	title := publicFrontMatterFieldByKey(t, doc, "title")

	rename, err := doc.PrepareRenameFrontMatterField(title.ID(), []byte("name"))
	if err != nil {
		t.Fatalf("PrepareRenameFrontMatterField() error = %v", err)
	}
	renamed, err := rename.Apply(source)
	if err != nil {
		t.Fatal(err)
	}
	wantRenamed := []byte("---\r\nname: \"Old\"\r\nauthor: \"Ada\"\r\n---\r\n\r\n# Body\r\n")
	if !bytes.Equal(renamed, wantRenamed) {
		t.Fatalf("rename Apply() = %q, want %q", renamed, wantRenamed)
	}

	remove, err := doc.PrepareRemoveFrontMatterField(title.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveFrontMatterField() error = %v", err)
	}
	removed, err := remove.Apply(source)
	if err != nil {
		t.Fatal(err)
	}
	wantRemoved := []byte("---\r\nauthor: \"Ada\"\r\n---\r\n\r\n# Body\r\n")
	if !bytes.Equal(removed, wantRemoved) {
		t.Fatalf("remove Apply() = %q, want %q", removed, wantRemoved)
	}

	appendField, err := doc.PrepareAppendFrontMatterField([]byte("lang"), []byte("it-IT"))
	if err != nil {
		t.Fatalf("PrepareAppendFrontMatterField() error = %v", err)
	}
	appended, err := appendField.Apply(source)
	if err != nil {
		t.Fatal(err)
	}
	wantAppended := []byte("---\r\ntitle: \"Old\"\r\nauthor: \"Ada\"\r\nlang: \"it-IT\"\r\n---\r\n\r\n# Body\r\n")
	if !bytes.Equal(appended, wantAppended) {
		t.Fatalf("append Apply() = %q, want %q", appended, wantAppended)
	}
	candidate, err := marksplice.Parse(appended)
	if err != nil {
		t.Fatal(err)
	}
	if got := publicFrontMatterFieldByKey(t, candidate, "lang"); got.Format() != marksplice.FrontMatterFormatYAML {
		t.Fatalf("lang format = %v, want YAML", got.Format())
	}
}

func TestFrontMatterEnvelopeLifecycleUsesProvenEOL(t *testing.T) {
	t.Parallel()

	source := []byte("# Body\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	add, err := doc.PrepareAddFrontMatter(marksplice.FrontMatterFormatTOML)
	if err != nil {
		t.Fatalf("PrepareAddFrontMatter() error = %v", err)
	}
	withFrontMatter, err := add.Apply(source)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("+++\r\n+++\r\n\r\n# Body\r\n")
	if !bytes.Equal(withFrontMatter, want) {
		t.Fatalf("add Apply() = %q, want %q", withFrontMatter, want)
	}
	candidate, err := marksplice.Parse(withFrontMatter)
	if err != nil {
		t.Fatal(err)
	}
	frontMatter, ok := candidate.FrontMatter()
	if !ok || frontMatter.Format() != marksplice.FrontMatterFormatTOML {
		t.Fatalf("FrontMatter() = %+v/%v", frontMatter, ok)
	}
	remove, err := candidate.PrepareRemoveFrontMatter()
	if err != nil {
		t.Fatalf("PrepareRemoveFrontMatter() error = %v", err)
	}
	roundTrip, err := remove.Apply(withFrontMatter)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(roundTrip, source) {
		t.Fatalf("remove Apply() = %q, want %q", roundTrip, source)
	}
}

func TestFrontMatterLifecycleFailsClosedOnAmbiguityAndUnsafeSource(t *testing.T) {
	t.Parallel()

	source := []byte("---\ntitle: \"one\"\nauthor: \"Ada\"\n---\nbody\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	title := publicFrontMatterFieldByKey(t, doc, "title")
	for _, key := range [][]byte{nil, []byte("bad key"), []byte("author")} {
		if _, err := doc.PrepareRenameFrontMatterField(title.ID(), key); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("rename key %q error = %v, want ErrInvalidReplacement", key, err)
		}
	}
	for _, value := range [][]byte{nil, []byte("bad\nvalue"), []byte(`bad"value`), []byte(`bad\value`)} {
		if _, err := doc.PrepareAppendFrontMatterField([]byte("extra"), value); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("append value %q error = %v, want ErrInvalidReplacement", value, err)
		}
	}
	if _, err := doc.PrepareAppendFrontMatterField([]byte("title"), []byte("duplicate")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("duplicate append error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareAddFrontMatter(marksplice.FrontMatterFormatYAML); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("add existing front matter error = %v, want ErrInvalidReplacement", err)
	}

	complexTOML, err := marksplice.Parse([]byte("+++\n[params]\nauthor = 'Ada'\n+++\nbody\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := complexTOML.PrepareAppendFrontMatterField([]byte("title"), []byte("x")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("append after TOML table error = %v, want ErrInvalidReplacement", err)
	}

	noEOL, err := marksplice.Parse([]byte("body"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := noEOL.PrepareAddFrontMatter(marksplice.FrontMatterFormatYAML); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("add without proven EOL error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := noEOL.PrepareRemoveFrontMatter(); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("remove absent front matter error = %v, want ErrInvalidReplacement", err)
	}

	change, err := doc.PrepareRemoveFrontMatterField(title.ID())
	if err != nil {
		t.Fatal(err)
	}
	stale := append([]byte(nil), source...)
	stale[len(stale)-2] = 'X'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}

func publicFrontMatterFieldByKey(t *testing.T, doc *marksplice.Document, key string) marksplice.FrontMatterField {
	t.Helper()
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindFrontMatterField {
			continue
		}
		field, ok := doc.FrontMatterField(node.ID())
		if ok && field.Key() == key {
			return field
		}
	}
	t.Fatalf("front-matter field %q not found", key)
	return marksplice.FrontMatterField{}
}
