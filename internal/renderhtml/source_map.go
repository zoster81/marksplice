package renderhtml

import (
	"io"

	"github.com/zoster81/marksplice/internal/parser"
)

// OutputRange is one half-open byte range in emitted HTML output.
type OutputRange struct {
	Start int
	End   int
}

// SourceMapEntry correlates one source-owned semantic range with one contiguous
// HTML output range. Nested semantic ranges may intentionally overlap in output.
type SourceMapEntry struct {
	Source parser.Range
	Output OutputRange
}

// SourceMapCollector consumes source-map entries synchronously during rendering.
type SourceMapCollector func(SourceMapEntry)

type mappingTarget uint8

const (
	mappingRoot mappingTarget = iota
	mappingCapture
)

type countingWriter struct {
	writer io.Writer
	offset int
}

func (w *countingWriter) Write(value []byte) (int, error) {
	written, err := w.writer.Write(value)
	w.offset += written
	return written, err
}

func (w *countingWriter) WriteString(value string) (int, error) {
	written, err := io.WriteString(w.writer, value)
	w.offset += written
	return written, err
}

func sourceMapEntry(source parser.Range, start, end int) (SourceMapEntry, bool) {
	if source.Start < 0 || source.End <= source.Start || end <= start {
		return SourceMapEntry{}, false
	}
	return SourceMapEntry{
		Source: source,
		Output: OutputRange{Start: start, End: end},
	}, true
}

func appendSourceMap(entries *[]SourceMapEntry, source parser.Range, start, end int) {
	if entries == nil {
		return
	}
	entry, ok := sourceMapEntry(source, start, end)
	if ok {
		*entries = append(*entries, entry)
	}
}

func emitSourceMap(collect SourceMapCollector, source parser.Range, start, end int) {
	if collect == nil {
		return
	}
	entry, ok := sourceMapEntry(source, start, end)
	if ok {
		collect(entry)
	}
}
