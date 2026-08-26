// Package commonmarkspec loads the approved published CommonMark specification snapshot for tests.
// It contains no specification corpus bytes; callers provide the separately provisioned file.
package commonmarkspec

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	htmlstd "html"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// PublishedSHA256 is the approved SHA-256 of the published CommonMark 0.31.2 HTML snapshot.
const PublishedSHA256 = "c85f56cc2101f761cc94215c677cd2c7c1ecca9c47be016479fdfe718af60f0d"

// Case is one numbered published CommonMark example and its containing specification section.
type Case struct {
	Number   int
	Section  string
	Markdown string
	HTML     string
}

type section struct {
	start int
	id    string
}

var (
	h2Pattern      = regexp.MustCompile(`(?s)<h2 id="([^"]+)"[^>]*>`)
	examplePattern = regexp.MustCompile(
		`(?s)<div class="example" id="example-([0-9]+)">.*?<pre><code class="language-markdown">(.*?)</code></pre>.*?<pre><code class="language-html">(.*?)</code></pre>`,
	)
	htmlTagPattern = regexp.MustCompile(`<[^>]+>`)
)

// LoadPublished loads and verifies the approved CommonMark 0.31.2 snapshot before
// returning its numbered examples in specification order.
func LoadPublished(path string) ([]Case, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := VerifyPublishedSnapshot(source); err != nil {
		return nil, err
	}
	if !bytes.Contains(source, []byte("Version 0.31.2")) {
		return nil, fmt.Errorf("input is not the published CommonMark 0.31.2 specification")
	}

	sections := extractSections(source)
	matches := examplePattern.FindAllSubmatchIndex(source, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no CommonMark examples found in published specification")
	}

	cases := make([]Case, 0, len(matches))
	for index, match := range matches {
		number, err := strconv.Atoi(string(source[match[2]:match[3]]))
		if err != nil {
			return nil, fmt.Errorf("parse example number: %w", err)
		}
		if number != index+1 {
			return nil, fmt.Errorf("non-contiguous CommonMark example numbering at index %d: got %d", index, number)
		}
		cases = append(cases, Case{
			Number:   number,
			Section:  sectionForPosition(sections, match[0]),
			Markdown: decodeCode(source[match[4]:match[5]]),
			HTML:     decodeCode(source[match[6]:match[7]]),
		})
	}
	return cases, nil
}

// VerifyPublishedSnapshot rejects bytes that are not the approved CommonMark snapshot.
func VerifyPublishedSnapshot(source []byte) error {
	actual := fmt.Sprintf("%x", sha256.Sum256(source))
	if actual != PublishedSHA256 {
		return fmt.Errorf("published CommonMark specification snapshot SHA-256 = %s, want %s", actual, PublishedSHA256)
	}
	return nil
}

func extractSections(source []byte) []section {
	matches := h2Pattern.FindAllSubmatchIndex(source, -1)
	sections := make([]section, 0, len(matches))
	for _, match := range matches {
		sections = append(sections, section{start: match[0], id: string(source[match[2]:match[3]])})
	}
	return sections
}

func sectionForPosition(sections []section, position int) string {
	index := sort.Search(len(sections), func(i int) bool {
		return sections[i].start > position
	}) - 1
	if index < 0 {
		return ""
	}
	return sections[index].id
}

func decodeCode(encoded []byte) string {
	withoutMarkup := htmlTagPattern.ReplaceAll(encoded, nil)
	decoded := htmlstd.UnescapeString(string(withoutMarkup))
	return strings.ReplaceAll(decoded, "→", "\t")
}
