package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM103FencedBlocksExposeExactContainerInfoLanguageAndBodyRanges(t *testing.T) {
	t.Parallel()

	source := []byte("before\n\n  ~~~~  mermaid diagram  \n  graph TD\n  A-->B\n ~~~~~~   \n\nafter\n")
	doc := mustParseM103Document(t, source)
	blocks := doc.FencedBlocks()
	if len(blocks) != 1 {
		t.Fatalf("len(FencedBlocks()) = %d, want 1", len(blocks))
	}
	block := blocks[0]
	if block.FenceChar() != '~' || block.OpeningFenceLength() != 4 || block.OpeningIndent() != 2 {
		t.Fatalf("opening fence = char %q length %d indent %d", block.FenceChar(), block.OpeningFenceLength(), block.OpeningIndent())
	}
	if !block.Closed() {
		t.Fatal("Closed() = false, want true")
	}
	closingRange, ok := block.ClosingFenceRange()
	if !ok {
		t.Fatal("ClosingFenceRange() ok = false")
	}
	closingLength, ok := block.ClosingFenceLength()
	if !ok || closingLength != 6 {
		t.Fatalf("ClosingFenceLength() = %d/%v, want 6/true", closingLength, ok)
	}
	closingIndent, ok := block.ClosingIndent()
	if !ok || closingIndent != 1 {
		t.Fatalf("ClosingIndent() = %d/%v, want 1/true", closingIndent, ok)
	}
	assertM103RangeSource(t, doc, block.OpeningFenceRange(), "~~~~")
	assertM103RangeSource(t, doc, closingRange, "~~~~~~")
	assertM103RangeSource(t, doc, block.Range(), "  ~~~~  mermaid diagram  \n  graph TD\n  A-->B\n ~~~~~~   \n")

	info, ok := block.Info()
	if !ok || info != "mermaid diagram" {
		t.Fatalf("Info() = %q/%v, want mermaid diagram/true", info, ok)
	}
	infoRange, ok := block.InfoRange()
	if !ok {
		t.Fatal("InfoRange() ok = false")
	}
	assertM103RangeSource(t, doc, infoRange, "mermaid diagram")
	language, ok := block.Language()
	if !ok || language != "mermaid" {
		t.Fatalf("Language() = %q/%v, want mermaid/true", language, ok)
	}

	body, ok := doc.FencedBlockContentRanges(block.ID())
	if !ok || len(body) != 2 {
		t.Fatalf("FencedBlockContentRanges() = %v/%v, want 2 ranges", body, ok)
	}
	assertM103RangeSource(t, doc, body[0], "graph TD")
	assertM103RangeSource(t, doc, body[1], "A-->B")
	original := body[0]
	body[0] = marksplice.Range{}
	again, ok := doc.FencedBlockContentRanges(block.ID())
	if !ok || len(again) != 2 || again[0] != original {
		t.Fatalf("second FencedBlockContentRanges() = %v/%v, caller mutation leaked", again, ok)
	}

	if _, ok := doc.FencedCode(block.ID()); ok {
		t.Fatal("FencedCode(indented multiline block) ok = true; legacy contiguous replacement range must remain unavailable")
	}
	blocks[0] = marksplice.FencedBlock{}
	againBlocks := doc.FencedBlocks()
	if len(againBlocks) != 1 || againBlocks[0].ID() != block.ID() {
		t.Fatalf("second FencedBlocks() = %+v, caller mutation leaked", againBlocks)
	}
}

func TestM103EmptyFencedBlockIsReadableAndConstructible(t *testing.T) {
	t.Parallel()

	source := []byte("```math\n```\n")
	doc := mustParseM103Document(t, source)
	blocks := doc.FencedBlocks()
	if len(blocks) != 1 {
		t.Fatalf("len(FencedBlocks()) = %d, want 1", len(blocks))
	}
	block := blocks[0]
	if !block.Closed() || block.FenceChar() != '`' || block.OpeningFenceLength() != 3 {
		t.Fatalf("empty block fence facts = closed %v char %q length %d", block.Closed(), block.FenceChar(), block.OpeningFenceLength())
	}
	if info, ok := block.Info(); !ok || info != "math" {
		t.Fatalf("Info() = %q/%v, want math/true", info, ok)
	}
	if language, ok := block.Language(); !ok || language != "math" {
		t.Fatalf("Language() = %q/%v, want math/true", language, ok)
	}
	body, ok := doc.FencedBlockContentRanges(block.ID())
	if !ok || len(body) != 0 {
		t.Fatalf("FencedBlockContentRanges() = %v/%v, want empty/true", body, ok)
	}
	if _, ok := doc.FencedCode(block.ID()); ok {
		t.Fatal("FencedCode(empty block) ok = true; no legacy replacement span exists")
	}

	builder := marksplice.NewDocumentBuilder()
	if err := builder.AppendFencedCode("", "math"); err != nil {
		t.Fatalf("AppendFencedCode(empty) error = %v", err)
	}
	markdown, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	if want := []byte("```math\n```\n"); !bytes.Equal(markdown, want) {
		t.Fatalf("Markdown() = %q, want %q", markdown, want)
	}
	built := mustParseM103Document(t, markdown).FencedBlocks()
	if len(built) != 1 {
		t.Fatalf("built len(FencedBlocks()) = %d, want 1", len(built))
	}
	if ranges, ok := mustParseM103Document(t, markdown).FencedBlockContentRanges(built[0].ID()); !ok || len(ranges) != 0 {
		t.Fatalf("built content ranges = %v/%v, want empty/true", ranges, ok)
	}
}

func TestM103UnclosedContiguousFenceKeepsLegacyReplacementContract(t *testing.T) {
	t.Parallel()

	source := []byte("~~~ geojson\n{\"type\":\"Point\"}")
	doc := mustParseM103Document(t, source)
	blocks := doc.FencedBlocks()
	if len(blocks) != 1 {
		t.Fatalf("len(FencedBlocks()) = %d, want 1", len(blocks))
	}
	block := blocks[0]
	if block.Closed() {
		t.Fatal("Closed() = true, want false")
	}
	if _, ok := block.ClosingFenceRange(); ok {
		t.Fatal("ClosingFenceRange() ok = true for unclosed block")
	}
	if _, ok := block.ClosingFenceLength(); ok {
		t.Fatal("ClosingFenceLength() ok = true for unclosed block")
	}
	if _, ok := block.ClosingIndent(); ok {
		t.Fatal("ClosingIndent() ok = true for unclosed block")
	}
	assertM103RangeSource(t, doc, block.Range(), string(source))
	if language, ok := block.Language(); !ok || language != "geojson" {
		t.Fatalf("Language() = %q/%v, want geojson/true", language, ok)
	}

	legacy, ok := doc.FencedCode(block.ID())
	if !ok {
		t.Fatal("FencedCode(unclosed contiguous block) ok = false")
	}
	assertM103RangeSource(t, doc, legacy.Range(), "{\"type\":\"Point\"}")
	change, err := doc.PrepareReplaceFencedCode(block.ID(), []byte("{\"type\":\"LineString\"}"))
	if err != nil {
		t.Fatalf("PrepareReplaceFencedCode() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("~~~ geojson\n{\"type\":\"LineString\"}")
	if !bytes.Equal(got, want) {
		t.Fatalf("Apply() = %q, want %q", got, want)
	}
}

func TestM103HistoricalFencedCodeRangeAndClosedReplacementRemainStable(t *testing.T) {
	t.Parallel()

	source := []byte("```` go extra\nline one\nline two\n  `````  \n")
	doc := mustParseM103Document(t, source)
	blocks := doc.FencedBlocks()
	if len(blocks) != 1 {
		t.Fatalf("len(FencedBlocks()) = %d, want 1", len(blocks))
	}
	block := blocks[0]
	legacy, ok := doc.FencedCode(block.ID())
	if !ok {
		t.Fatal("FencedCode() ok = false")
	}
	assertM103RangeSource(t, doc, legacy.Range(), "line one\nline two")
	if block.OpeningFenceLength() != 4 {
		t.Fatalf("OpeningFenceLength() = %d, want 4", block.OpeningFenceLength())
	}
	if closing, ok := block.ClosingFenceLength(); !ok || closing != 5 {
		t.Fatalf("ClosingFenceLength() = %d/%v, want 5/true", closing, ok)
	}
	if info, ok := block.Info(); !ok || info != "go extra" {
		t.Fatalf("Info() = %q/%v, want go extra/true", info, ok)
	}
	if language, ok := block.Language(); !ok || language != "go" {
		t.Fatalf("Language() = %q/%v, want go/true", language, ok)
	}

	change, err := doc.PrepareReplaceFencedCode(block.ID(), []byte("new one\nnew two"))
	if err != nil {
		t.Fatalf("PrepareReplaceFencedCode() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("```` go extra\nnew one\nnew two\n  `````  \n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Apply() = %q, want %q", got, want)
	}
}

func TestM103FencedBlocksStayTopLevelAndFailClosedOnUnknownIDs(t *testing.T) {
	t.Parallel()

	source := []byte("> ```note\n> nested\n> ```\n\n- item\n  ~~~ tip\n  nested\n  ~~~\n")
	doc := mustParseM103Document(t, source)
	if blocks := doc.FencedBlocks(); len(blocks) != 0 {
		t.Fatalf("FencedBlocks() = %+v, want no top-level fenced blocks", blocks)
	}
	if _, ok := doc.FencedBlock(marksplice.NodeID{}); ok {
		t.Fatal("FencedBlock(zero ID) ok = true")
	}
	if _, ok := doc.FencedBlockContentRanges(marksplice.NodeID{}); ok {
		t.Fatal("FencedBlockContentRanges(zero ID) ok = true")
	}
}

func TestM103EmptyReplacementStillFailsClosedForLegacyFencedCode(t *testing.T) {
	t.Parallel()

	source := []byte("```\nbody\n```\n")
	doc := mustParseM103Document(t, source)
	block := doc.FencedBlocks()[0]
	_, err := doc.PrepareReplaceFencedCode(block.ID(), nil)
	if !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareReplaceFencedCode(empty) error = %v, want ErrInvalidReplacement", err)
	}
}

func mustParseM103Document(t *testing.T, source []byte) *marksplice.Document {
	t.Helper()
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return doc
}

func assertM103RangeSource(t *testing.T, doc *marksplice.Document, range_ marksplice.Range, want string) {
	t.Helper()
	got, ok := doc.SourceRange(range_)
	if !ok {
		t.Fatalf("SourceRange(%v) ok = false", range_)
	}
	if string(got) != want {
		t.Fatalf("SourceRange(%v) = %q, want %q", range_, got, want)
	}
}
