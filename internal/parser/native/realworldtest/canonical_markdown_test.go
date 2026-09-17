package realworldtest

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
	"github.com/zoster81/marksplice/internal/rendermarkdown"
)

type canonicalSemanticFact struct {
	phase       parser.SemanticPhase
	kind        parser.SemanticKind
	value       string
	level       int
	destination string
	title       string
	hasTitle    bool
	label       string
	email       bool
	ordered     bool
	start       int
	tight       bool
	checked     bool
	header      bool
	column      int
	columns     int
	alignment   parser.TableAlignment
	language    string
	info        string
	alert       parser.SemanticAlertKind
	frontMatter parser.SemanticFrontMatterFormat
	math        parser.MathExpressionStyle
}

func TestCanonicalMarkdownRealWorldCorpus(t *testing.T) {
	files := loadRealWorldCorpus(t)
	backend := native.New()
	var totalSourceBytes, totalCanonicalBytes int64

	for _, file := range files {
		file := file
		t.Run(realWorldBenchmarkName(file.relative), func(t *testing.T) {
			before := bytes.Clone(file.source)
			originalFacts := collectCanonicalSemanticFacts(t, backend, file.source)

			var first bytes.Buffer
			if panicValue, err := renderCanonical(&first, file.source, backend); panicValue != nil {
				t.Fatalf("canonical render panic: %v", panicValue)
			} else if err != nil {
				t.Fatalf("canonical render error: %v", err)
			}
			if !bytes.Equal(file.source, before) {
				t.Fatal("canonical rendering mutated corpus source")
			}

			canonicalFacts := collectCanonicalSemanticFacts(t, backend, first.Bytes())
			if !reflect.DeepEqual(canonicalFacts, originalFacts) {
				t.Fatalf("canonical rendering changed semantic facts: %s", firstCanonicalSemanticDifference(originalFacts, canonicalFacts))
			}

			var second bytes.Buffer
			if panicValue, err := renderCanonical(&second, first.Bytes(), backend); panicValue != nil {
				t.Fatalf("second canonical render panic: %v", panicValue)
			} else if err != nil {
				t.Fatalf("second canonical render error: %v", err)
			}
			if !bytes.Equal(second.Bytes(), first.Bytes()) {
				t.Fatal("canonical rendering is not byte-idempotent")
			}

			totalSourceBytes += int64(len(file.source))
			totalCanonicalBytes += int64(first.Len())
		})
	}

	t.Logf("M123 real-world canonical corpus: files=%d source_bytes=%d canonical_bytes=%d", len(files), totalSourceBytes, totalCanonicalBytes)
}

func BenchmarkCanonicalMarkdownRealWorldCorpus(b *testing.B) {
	files := loadRealWorldCorpus(b)
	backend := native.New()
	var totalBytes int64
	for _, file := range files {
		totalBytes += int64(len(file.source))
	}
	b.SetBytes(totalBytes)
	b.ReportMetric(float64(len(files)), "files/op")
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		for _, file := range files {
			if err := rendermarkdown.Render(io.Discard, file.source, backend); err != nil {
				b.Fatalf("Render(%s): %v", file.relative, err)
			}
		}
	}
}

func BenchmarkCanonicalMarkdownLargestFiles(b *testing.B) {
	files := slices.Clone(loadRealWorldCorpus(b))
	slices.SortFunc(files, func(left, right realWorldCorpusFile) int {
		return len(right.source) - len(left.source)
	})
	files = files[:min(12, len(files))]
	backend := native.New()
	for _, file := range files {
		file := file
		b.Run(realWorldBenchmarkName(file.relative), func(b *testing.B) {
			b.SetBytes(int64(len(file.source)))
			b.ReportAllocs()
			for range b.N {
				if err := rendermarkdown.Render(io.Discard, file.source, backend); err != nil {
					b.Fatalf("Render(%s): %v", file.relative, err)
				}
			}
		})
	}
}

func renderCanonical(writer io.Writer, source []byte, backend rendermarkdown.Backend) (panicValue any, err error) {
	defer func() {
		panicValue = recover()
	}()
	err = rendermarkdown.Render(writer, source, backend)
	return nil, err
}

func collectCanonicalSemanticFacts(t *testing.T, backend rendermarkdown.Backend, source []byte) []canonicalSemanticFact {
	t.Helper()
	facts := make([]canonicalSemanticFact, 0, 32)
	footnotes := make([][]canonicalSemanticFact, 0, 4)
	var currentFootnote []canonicalSemanticFact
	if err := backend.WalkSemantic(source, func(event parser.SemanticEvent) error {
		if event.Kind == parser.SemanticReferenceDefinition {
			return nil
		}
		fact := canonicalSemanticFactForEvent(event)
		if currentFootnote != nil {
			currentFootnote = append(currentFootnote, fact)
			if event.Kind == parser.SemanticFootnoteDefinition && event.Phase == parser.SemanticExit {
				footnotes = append(footnotes, currentFootnote)
				currentFootnote = nil
			}
			return nil
		}
		if event.Kind == parser.SemanticFootnoteDefinition && event.Phase == parser.SemanticEnter {
			currentFootnote = []canonicalSemanticFact{fact}
			return nil
		}
		facts = append(facts, fact)
		return nil
	}); err != nil {
		t.Fatalf("WalkSemantic() error: %v", err)
	}
	if currentFootnote != nil {
		t.Fatal("WalkSemantic() ended with an open footnote definition")
	}
	slices.SortFunc(footnotes, func(left, right []canonicalSemanticFact) int {
		return strings.Compare(fmt.Sprintf("%#v", left), fmt.Sprintf("%#v", right))
	})
	for _, footnote := range footnotes {
		facts = append(facts, footnote...)
	}
	return facts
}

func canonicalSemanticFactForEvent(event parser.SemanticEvent) canonicalSemanticFact {
	value := event.Value
	switch event.Kind {
	case parser.SemanticFrontMatter, parser.SemanticRawHTML, parser.SemanticCodeBlock:
		value = normalizeCanonicalLineEndings(value)
	case parser.SemanticHTMLBlock:
		value = strings.TrimSuffix(normalizeCanonicalLineEndings(value), "\n")
	}
	label := event.Label
	if event.Kind == parser.SemanticLink || event.Kind == parser.SemanticImage {
		label = ""
	}
	return canonicalSemanticFact{
		phase:       event.Phase,
		kind:        event.Kind,
		value:       value,
		level:       event.Level,
		destination: event.Destination,
		title:       decodeCanonicalMarkdownSyntaxString(event.Title),
		hasTitle:    event.HasTitle,
		label:       label,
		email:       event.AutoLinkEmail,
		ordered:     event.Ordered,
		start:       event.Start,
		tight:       event.Tight,
		checked:     event.Checked,
		header:      event.Header,
		column:      event.Column,
		columns:     event.Columns,
		alignment:   event.Alignment,
		language:    event.Language,
		info:        event.Info,
		alert:       event.AlertKind,
		frontMatter: event.FrontMatterFormat,
		math:        event.MathStyle,
	}
}

func firstCanonicalSemanticDifference(before, after []canonicalSemanticFact) string {
	limit := len(before)
	if len(after) < limit {
		limit = len(after)
	}
	for index := 0; index < limit; index++ {
		if before[index] != after[index] {
			return fmt.Sprintf("index=%d before=%#v after=%#v", index, before[index], after[index])
		}
	}
	if len(before) != len(after) {
		return "semantic event count changed"
	}
	return "semantic facts differ"
}

func normalizeCanonicalLineEndings(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(value, "\r", "\n")
}

func decodeCanonicalMarkdownSyntaxString(value string) string {
	var output strings.Builder
	for position := 0; position < len(value); {
		if value[position] == '\\' && position+1 < len(value) && isCanonicalASCIIPunctuation(value[position+1]) {
			output.WriteByte(value[position+1])
			position += 2
			continue
		}
		output.WriteByte(value[position])
		position++
	}
	return output.String()
}

func isCanonicalASCIIPunctuation(value byte) bool {
	return value >= '!' && value <= '/' || value >= ':' && value <= '@' || value >= '[' && value <= '`' || value >= '{' && value <= '~'
}
