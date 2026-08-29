package publictest

import (
	"os"
	"strconv"
	"testing"

	"github.com/zoster81/marksplice"
	"github.com/zoster81/marksplice/internal/testutil/commonmarkspec"
	"github.com/zoster81/marksplice/internal/testutil/gfmspec"
)

func TestM120PublishedCommonMarkHTMLContract(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_COMMONMARK_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_COMMONMARK_SPEC_HTML is not set")
	}
	cases, err := commonmarkspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published CommonMark spec: %v", err)
	}
	for _, number := range []int{5, 6, 7, 28, 80, 278, 279, 335, 473, 474, 521, 527, 534, 564, 572, 603, 604, 633} {
		case_ := cases[number-1]
		if case_.Number != number {
			t.Fatalf("CommonMark example identity changed: got %d want %d", case_.Number, number)
		}
		t.Run("example-"+strconv.Itoa(number), func(t *testing.T) {
			assertPublishedHTML(t, case_.Markdown, case_.HTML)
		})
	}
}

func TestM120PublishedGFMHTMLContract(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	for _, number := range []int{49, 82, 108, 115, 143, 193, 196, 199, 200, 202, 204, 278, 279, 295, 298, 301, 324, 344, 441, 482, 483, 491, 514, 530, 543, 573, 622, 630, 638, 639, 648, 649, 650, 657, 661, 662, 667, 668, 674} {
		case_ := cases[number-1]
		if case_.Number != number {
			t.Fatalf("GFM example identity changed: got %d want %d", case_.Number, number)
		}
		t.Run("example-"+strconv.Itoa(number), func(t *testing.T) {
			assertPublishedHTML(t, case_.Markdown, case_.HTML)
		})
	}
}

func assertPublishedHTML(t *testing.T, markdown, want string) {
	t.Helper()
	document, err := marksplice.Parse([]byte(markdown))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	got, err := document.HTML(marksplice.HTMLRenderOptions{})
	if err != nil {
		t.Fatalf("HTML() error = %v", err)
	}
	if string(got) != want {
		t.Fatalf("HTML mismatch\ngot:  %q\nwant: %q", got, want)
	}
}
