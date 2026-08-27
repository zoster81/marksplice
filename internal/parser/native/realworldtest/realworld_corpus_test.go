package realworldtest

import (
	"bytes"
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

const (
	realWorldCorpusEnv = "MARKSPLICE_REALWORLD_CORPUS"
	realWorldFileEnv   = "MARKSPLICE_REALWORLD_FILE"
)

type realWorldCorpusFile struct {
	relative string
	repo     string
	source   []byte
}

type realWorldParseSample struct {
	relative    string
	bytes       int
	duration    time.Duration
	nodes       int
	links       int
	unresolved  int
	nestedLists int
	nonTopLevel int
}

func TestNativeRealWorldMarkdownCorpus(t *testing.T) {
	files := loadRealWorldCorpus(t)
	backend := native.New()
	repositories := make(map[string]struct{})
	samples := make([]realWorldParseSample, 0, len(files))

	var totalBytes int64
	var totalNodes, totalLinks, totalUnresolved int
	var nonUTF8Files, bomFiles, crlfFiles, tabFiles int

	for _, file := range files {
		file := file
		repositories[file.repo] = struct{}{}
		totalBytes += int64(len(file.source))
		if !utf8.Valid(file.source) {
			nonUTF8Files++
		}
		if bytes.HasPrefix(file.source, []byte{0xef, 0xbb, 0xbf}) {
			bomFiles++
		}
		if bytes.Contains(file.source, []byte("\r\n")) {
			crlfFiles++
		}
		if bytes.Contains(file.source, []byte{'\t'}) {
			tabFiles++
		}

		t.Run(filepath.ToSlash(file.relative), func(t *testing.T) {
			before := bytes.Clone(file.source)
			started := time.Now()
			first, panicValue, parseErr := parseNativeRealWorld(backend, file.source)
			elapsed := time.Since(started)
			if panicValue != nil {
				t.Fatalf("Native ParseDocument panic: %v", panicValue)
			}
			if parseErr != nil {
				t.Fatalf("Native ParseDocument error: %v", parseErr)
			}
			if !bytes.Equal(file.source, before) {
				t.Fatal("Native ParseDocument mutated corpus source")
			}
			if err := validateSourceBound(first, len(file.source)); err != nil {
				t.Fatal(err)
			}

			second, secondPanic, secondErr := parseNativeRealWorld(backend, file.source)
			if secondPanic != nil {
				t.Fatalf("second Native ParseDocument panic: %v", secondPanic)
			}
			if secondErr != nil {
				t.Fatalf("second Native ParseDocument error: %v", secondErr)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("Native ParseDocument is nondeterministic for corpus source")
			}

			nestedLists, nonTopLevel := realWorldNestingCounts(first.Nodes)
			samples = append(samples, realWorldParseSample{
				relative:    file.relative,
				bytes:       len(file.source),
				duration:    elapsed,
				nodes:       len(first.Nodes),
				links:       len(first.LinkUsages),
				unresolved:  len(first.UnresolvedReferenceUsages),
				nestedLists: nestedLists,
				nonTopLevel: nonTopLevel,
			})
			totalNodes += len(first.Nodes)
			totalLinks += len(first.LinkUsages)
			totalUnresolved += len(first.UnresolvedReferenceUsages)
		})
	}

	if len(repositories) < 30 {
		t.Fatalf("real-world corpus repositories = %d, want at least 30", len(repositories))
	}

	slices.SortFunc(samples, func(left, right realWorldParseSample) int {
		return cmp.Compare(right.duration, left.duration)
	})
	top := min(15, len(samples))
	for index := 0; index < top; index++ {
		sample := samples[index]
		t.Logf("slow[%02d] %s bytes=%d duration=%s nodes=%d links=%d unresolved=%d nested_lists=%d non_top_level=%d",
			index+1, filepath.ToSlash(sample.relative), sample.bytes, sample.duration, sample.nodes,
			sample.links, sample.unresolved, sample.nestedLists, sample.nonTopLevel)
	}

	costSamples := make([]realWorldParseSample, 0, len(samples))
	for _, sample := range samples {
		if sample.bytes >= 4<<10 {
			costSamples = append(costSamples, sample)
		}
	}
	slices.SortFunc(costSamples, func(left, right realWorldParseSample) int {
		return cmp.Compare(realWorldNanosecondsPerByte(right), realWorldNanosecondsPerByte(left))
	})
	for index := 0; index < min(15, len(costSamples)); index++ {
		sample := costSamples[index]
		t.Logf("cost[%02d] %s bytes=%d duration=%s ns_per_byte=%.1f nodes_per_kib=%.1f links=%d unresolved=%d",
			index+1, filepath.ToSlash(sample.relative), sample.bytes, sample.duration, realWorldNanosecondsPerByte(sample),
			float64(sample.nodes)*(1<<10)/float64(sample.bytes), sample.links, sample.unresolved)
	}

	t.Logf("real-world corpus: repos=%d files=%d bytes=%d nodes=%d links=%d unresolved=%d non_utf8=%d bom=%d crlf=%d tabs=%d go=%s os=%s arch=%s",
		len(repositories), len(files), totalBytes, totalNodes, totalLinks, totalUnresolved,
		nonUTF8Files, bomFiles, crlfFiles, tabFiles, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

func TestNativeRealWorldDerivedMalformedCorpus(t *testing.T) {
	files := selectMalformedRealWorldSeeds(loadRealWorldCorpus(t), 64)
	backend := native.New()
	cases := 0
	for _, file := range files {
		for _, mutation := range realWorldMalformedMutations(file.source) {
			mutation := mutation
			cases++
			t.Run(realWorldBenchmarkName(file.relative)+"/"+mutation.name, func(t *testing.T) {
				assertNativeRealWorldStable(t, backend, mutation.source)
			})
		}
	}
	t.Logf("derived malformed corpus: seeds=%d cases=%d", len(files), cases)
}

type realWorldMutation struct {
	name   string
	source []byte
}

func selectMalformedRealWorldSeeds(files []realWorldCorpusFile, limit int) []realWorldCorpusFile {
	if len(files) <= limit {
		return slices.Clone(files)
	}
	bySize := slices.Clone(files)
	slices.SortFunc(bySize, func(left, right realWorldCorpusFile) int {
		return cmp.Compare(len(right.source), len(left.source))
	})
	largest := min(limit/2, len(bySize))
	selected := append([]realWorldCorpusFile(nil), bySize[:largest]...)
	seen := make(map[string]struct{}, limit)
	for _, file := range selected {
		seen[file.relative] = struct{}{}
	}
	byPath := slices.Clone(files)
	slices.SortFunc(byPath, func(left, right realWorldCorpusFile) int {
		return strings.Compare(filepath.ToSlash(left.relative), filepath.ToSlash(right.relative))
	})
	remaining := limit - len(selected)
	for index := 0; index < remaining; index++ {
		position := index * len(byPath) / remaining
		for position < len(byPath) {
			file := byPath[position]
			if _, exists := seen[file.relative]; !exists {
				selected = append(selected, file)
				seen[file.relative] = struct{}{}
				break
			}
			position++
		}
	}
	return selected
}

func realWorldMalformedMutations(source []byte) []realWorldMutation {
	half := len(source) / 2
	if half == 0 {
		half = len(source)
	}
	isolatedCR := bytes.Clone(source)
	if newline := bytes.IndexByte(isolatedCR, '\n'); newline >= 0 {
		isolatedCR[newline] = '\r'
	} else {
		isolatedCR = append(isolatedCR, '\r')
	}
	invalidUTF8 := append(bytes.Clone(source), 0xff, 0xfe, '[', 'x', ']')
	deepQuote := make([]byte, 0, len(source)+512)
	deepQuote = append(deepQuote, bytes.Repeat([]byte("> "), 256)...)
	deepQuote = append(deepQuote, source...)
	return []realWorldMutation{
		{name: "truncate-half", source: bytes.Clone(source[:half])},
		{name: "unclosed-fence", source: append(bytes.Clone(source), []byte("\n````marksplice-malformed\n")...)},
		{name: "unclosed-inline", source: append(bytes.Clone(source), []byte("\n[broken **~~` <https://example.invalid\n")...)},
		{name: "isolated-cr", source: isolatedCR},
		{name: "invalid-utf8-tail", source: invalidUTF8},
		{name: "deep-quote-prefix", source: deepQuote},
	}
}

func assertNativeRealWorldStable(t *testing.T, backend *native.Backend, source []byte) {
	t.Helper()
	before := bytes.Clone(source)
	first, panicValue, err := parseNativeRealWorld(backend, source)
	if panicValue != nil {
		t.Fatalf("Native ParseDocument panic: %v", panicValue)
	}
	if err != nil {
		t.Fatalf("Native ParseDocument error: %v", err)
	}
	if !bytes.Equal(source, before) {
		t.Fatal("Native ParseDocument mutated derived source")
	}
	if err := validateSourceBound(first, len(source)); err != nil {
		t.Fatal(err)
	}
	second, panicValue, err := parseNativeRealWorld(backend, source)
	if panicValue != nil {
		t.Fatalf("second Native ParseDocument panic: %v", panicValue)
	}
	if err != nil {
		t.Fatalf("second Native ParseDocument error: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("Native ParseDocument is nondeterministic for derived source")
	}
}

func BenchmarkNativeRealWorldCorpus(b *testing.B) {
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
			if _, err := backend.ParseDocument(file.source); err != nil {
				b.Fatalf("ParseDocument(%s): %v", file.relative, err)
			}
		}
	}
}

func BenchmarkNativeRealWorldSelectedFile(b *testing.B) {
	selected := filepath.ToSlash(os.Getenv(realWorldFileEnv))
	if selected == "" {
		b.Skipf("%s is not set", realWorldFileEnv)
	}
	files := loadRealWorldCorpus(b)
	for _, file := range files {
		if filepath.ToSlash(file.relative) != selected {
			continue
		}
		backend := native.New()
		b.SetBytes(int64(len(file.source)))
		b.ReportAllocs()
		for range b.N {
			if _, err := backend.ParseDocument(file.source); err != nil {
				b.Fatalf("ParseDocument(%s): %v", file.relative, err)
			}
		}
		return
	}
	b.Fatalf("%s=%q does not identify a corpus file", realWorldFileEnv, selected)
}

func BenchmarkNativeRealWorldLargestFiles(b *testing.B) {
	files := loadRealWorldCorpus(b)
	files = slices.Clone(files)
	slices.SortFunc(files, func(left, right realWorldCorpusFile) int {
		return len(right.source) - len(left.source)
	})
	files = files[:min(12, len(files))]

	for _, file := range files {
		file := file
		name := realWorldBenchmarkName(file.relative)
		b.Run(name, func(b *testing.B) {
			backend := native.New()
			b.SetBytes(int64(len(file.source)))
			b.ReportAllocs()
			for range b.N {
				if _, err := backend.ParseDocument(file.source); err != nil {
					b.Fatalf("ParseDocument(%s): %v", file.relative, err)
				}
			}
		})
	}
}

func loadRealWorldCorpus(tb testing.TB) []realWorldCorpusFile {
	tb.Helper()
	root := os.Getenv(realWorldCorpusEnv)
	if root == "" {
		tb.Skipf("%s is not set", realWorldCorpusEnv)
	}
	root, err := filepath.Abs(root)
	if err != nil {
		tb.Fatalf("resolve real-world corpus root: %v", err)
	}

	files := make([]realWorldCorpusFile, 0, 1024)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if extension != ".md" && extension != ".markdown" {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		components := strings.Split(filepath.ToSlash(relative), "/")
		if len(components) < 2 || components[0] == "" {
			return fmt.Errorf("corpus file is not repository-scoped: %s", relative)
		}
		files = append(files, realWorldCorpusFile{relative: relative, repo: components[0], source: source})
		return nil
	})
	if err != nil {
		tb.Fatalf("walk real-world corpus: %v", err)
	}
	if len(files) == 0 {
		tb.Fatal("real-world corpus contains no Markdown files")
	}
	return files
}

func parseNativeRealWorld(backend *native.Backend, source []byte) (observations parser.DocumentObservations, panicValue any, err error) {
	defer func() {
		panicValue = recover()
	}()
	observations, err = backend.ParseDocument(source)
	return observations, nil, err
}

func validateSourceBound(observations parser.DocumentObservations, total int) error {
	for index, node := range observations.Nodes {
		if !node.Range.Valid(total) {
			return fmt.Errorf("node %d range %v invalid for source length %d", index, node.Range, total)
		}
		if node.BlockquoteContentRange != (parser.Range{}) && !node.BlockquoteContentRange.Valid(total) {
			return fmt.Errorf("node %d blockquote content range %v invalid", index, node.BlockquoteContentRange)
		}
		for _, range_ := range node.BlockquoteSemanticRanges {
			if !range_.Valid(total) {
				return fmt.Errorf("node %d blockquote semantic range %v invalid", index, range_)
			}
		}
		for _, range_ := range node.FencedCodeContentRanges {
			if !range_.Valid(total) {
				return fmt.Errorf("node %d fenced content range %v invalid", index, range_)
			}
		}
	}
	for index, usage := range observations.LinkUsages {
		if usage.Anchor < 0 || usage.Anchor > total {
			return fmt.Errorf("link usage %d anchor %d invalid for source length %d", index, usage.Anchor, total)
		}
	}
	for index, usage := range observations.UnresolvedReferenceUsages {
		if usage.Anchor < 0 || usage.Anchor > total {
			return fmt.Errorf("unresolved usage %d anchor %d invalid for source length %d", index, usage.Anchor, total)
		}
	}
	for index, definition := range observations.FootnoteDefinitions {
		if definition.Anchor < 0 || definition.Anchor > total {
			return fmt.Errorf("footnote definition %d anchor %d invalid", index, definition.Anchor)
		}
		for _, range_ := range definition.BodyRanges {
			if !range_.Valid(total) {
				return fmt.Errorf("footnote definition %d body range %v invalid", index, range_)
			}
		}
	}
	for index, reference := range observations.FootnoteReferences {
		if !reference.Range.Valid(total) || !reference.LabelRange.Valid(total) || reference.DefinitionAnchor < 0 || reference.DefinitionAnchor > total {
			return fmt.Errorf("footnote reference %d is not source-bound: %#v", index, reference)
		}
	}
	for index, expression := range observations.MathExpressions {
		if !expression.Range.Valid(total) || !expression.PayloadRange.Valid(total) {
			return fmt.Errorf("math expression %d is not source-bound: %#v", index, expression)
		}
	}
	return nil
}

func realWorldNestingCounts(nodes []parser.Node) (nestedLists, nonTopLevel int) {
	for _, node := range nodes {
		if node.HasListParent {
			nestedLists++
		}
		if !node.TopLevel {
			nonTopLevel++
		}
	}
	return nestedLists, nonTopLevel
}

func realWorldNanosecondsPerByte(sample realWorldParseSample) float64 {
	if sample.bytes == 0 {
		return 0
	}
	return float64(sample.duration.Nanoseconds()) / float64(sample.bytes)
}

func realWorldBenchmarkName(relative string) string {
	name := filepath.ToSlash(relative)
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, " ", "_")
	if len(name) > 120 {
		name = name[:120]
	}
	return name
}
