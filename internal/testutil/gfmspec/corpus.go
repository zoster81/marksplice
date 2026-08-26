// Package gfmspec loads the approved published GFM specification snapshot for tests.
// It contains no specification corpus bytes; callers provide the separately provisioned file.
package gfmspec

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

// PublishedSHA256 is the approved SHA-256 of the published GFM 0.29 HTML snapshot.
const PublishedSHA256 = "b153d814fdfc8624bb6da7449162c1cd707a637f7d1c1b636eb44b9cf63fa220"

// Case is one numbered published GFM example and its section-derived extension profile.
type Case struct {
	Number     int
	Extensions []string
	Markdown   string
	HTML       string
}

// Stats summarizes the approved corpus shape.
type Stats struct {
	Total         int
	Core          int
	Table         int
	TaskList      int
	Strikethrough int
	Autolink      int
	TagFilter     int
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

var extensionBySection = map[string]string{
	"tables-extension-":              "table",
	"task-list-items-extension-":     "tasklist",
	"strikethrough-extension-":       "strikethrough",
	"autolinks-extension-":           "autolink",
	"disallowed-raw-html-extension-": "tagfilter",
}

// LoadPublished loads and verifies the approved published GFM snapshot before
// returning its numbered examples in specification order.
func LoadPublished(path string) ([]Case, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := VerifyPublishedSnapshot(source); err != nil {
		return nil, err
	}
	if !bytes.Contains(source, []byte("Version 0.29-gfm")) {
		return nil, fmt.Errorf("input is not the published GFM 0.29 specification")
	}

	sections := extractSections(source)
	matches := examplePattern.FindAllSubmatchIndex(source, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no GFM examples found in published specification")
	}

	cases := make([]Case, 0, len(matches))
	for index, match := range matches {
		number, err := strconv.Atoi(string(source[match[2]:match[3]]))
		if err != nil {
			return nil, fmt.Errorf("parse example number: %w", err)
		}
		if number != index+1 {
			return nil, fmt.Errorf("non-contiguous GFM example numbering at index %d: got %d", index, number)
		}

		case_ := Case{
			Number:   number,
			Markdown: decodeCode(source[match[4]:match[5]]),
			HTML:     decodeCode(source[match[6]:match[7]]),
		}
		if extensionName := extensionForPosition(sections, match[0]); extensionName != "" {
			case_.Extensions = []string{extensionName}
		}
		cases = append(cases, case_)
	}
	return cases, nil
}

// VerifyPublishedSnapshot rejects bytes that are not the approved GFM snapshot.
func VerifyPublishedSnapshot(source []byte) error {
	actual := fmt.Sprintf("%x", sha256.Sum256(source))
	if actual != PublishedSHA256 {
		return fmt.Errorf("published GFM specification snapshot SHA-256 = %s, want %s", actual, PublishedSHA256)
	}
	return nil
}

// Summarize returns deterministic counts for the known GFM extension sections.
func Summarize(cases []Case) Stats {
	stats := Stats{Total: len(cases)}
	for _, case_ := range cases {
		if len(case_.Extensions) == 0 {
			stats.Core++
		}
		for _, extension := range case_.Extensions {
			switch extension {
			case "table":
				stats.Table++
			case "tasklist":
				stats.TaskList++
			case "strikethrough":
				stats.Strikethrough++
			case "autolink":
				stats.Autolink++
			case "tagfilter":
				stats.TagFilter++
			}
		}
	}
	return stats
}

func extractSections(source []byte) []section {
	matches := h2Pattern.FindAllSubmatchIndex(source, -1)
	sections := make([]section, 0, len(matches))
	for _, match := range matches {
		sections = append(sections, section{start: match[0], id: string(source[match[2]:match[3]])})
	}
	return sections
}

func extensionForPosition(sections []section, position int) string {
	index := sort.Search(len(sections), func(i int) bool {
		return sections[i].start > position
	}) - 1
	if index < 0 {
		return ""
	}
	return extensionBySection[sections[index].id]
}

func decodeCode(encoded []byte) string {
	withoutMarkup := htmlTagPattern.ReplaceAll(encoded, nil)
	decoded := htmlstd.UnescapeString(string(withoutMarkup))
	return strings.ReplaceAll(decoded, "→", "\t")
}
