package marksplice

import (
	"bytes"
	"errors"
	"io"
	"sort"

	"github.com/zoster81/marksplice/internal/renderhtml"
)

// HTMLOutputRange is a half-open byte range [Start, End) in one exact emitted
// HTML result. The coordinate space is output bytes, not Markdown source bytes.
type HTMLOutputRange struct {
	Start int
	End   int
}

// Valid reports whether r is ordered and contained in output of total bytes.
func (r HTMLOutputRange) Valid(total int) bool {
	return r.Start >= 0 && r.End >= r.Start && r.End <= total
}

// HTMLSourceMapEntry correlates one snapshot-local Markdown source range with
// one contiguous range in the exact HTML output produced by the same render.
// Entries describe emitted semantic events rather than total byte coverage, so
// synthetic HTML can be unmapped and nested constructs can intentionally overlap.
type HTMLSourceMapEntry struct {
	sourceRange Range
	outputRange HTMLOutputRange
}

// SourceRange returns the half-open byte range in this Document snapshot.
func (m HTMLSourceMapEntry) SourceRange() Range { return m.sourceRange }

// OutputRange returns the half-open byte range in the exact rendered HTML output.
func (m HTMLSourceMapEntry) OutputRange() HTMLOutputRange { return m.outputRange }

// RenderHTMLWithSourceMap streams a deterministic HTML fragment and returns
// source-to-output byte mappings after successful completion. Entries are sorted
// by output byte offset, with an outer range before a nested range at the same
// start. On render failure, including writer failure, the returned mapping is nil.
func (d *Document) RenderHTMLWithSourceMap(writer io.Writer, options HTMLRenderOptions) ([]HTMLSourceMapEntry, error) {
	if d == nil || d.document == nil || writer == nil {
		return nil, ErrInvalidRender
	}
	internal, err := internalHTMLRenderOptions(options)
	if err != nil {
		return nil, err
	}
	var collector htmlSourceMapCollector
	if err := d.document.RenderHTMLWithSourceMap(writer, internal, collector.add); err != nil {
		if errors.Is(err, renderhtml.ErrInvalidInput) {
			return nil, translateError(err, renderhtml.ErrInvalidInput, ErrInvalidRender)
		}
		return nil, err
	}
	return collector.result(), nil
}

// HTMLWithSourceMap renders a deterministic HTML fragment and returns caller-owned
// output bytes plus source mappings for that exact output.
func (d *Document) HTMLWithSourceMap(options HTMLRenderOptions) ([]byte, []HTMLSourceMapEntry, error) {
	if d == nil || d.document == nil {
		return nil, nil, ErrInvalidRender
	}
	var output bytes.Buffer
	mappings, err := d.RenderHTMLWithSourceMap(&output, options)
	if err != nil {
		return nil, nil, err
	}
	return output.Bytes(), mappings, nil
}

// RenderHTMLDocumentWithSourceMap streams a deterministic standalone HTML
// document and returns source mappings using absolute output-byte offsets.
// Synthetic wrapper bytes are intentionally unmapped; eligible reviewed metadata
// fields map to the HTML they emit.
func (d *Document) RenderHTMLDocumentWithSourceMap(writer io.Writer, options HTMLDocumentOptions) ([]HTMLSourceMapEntry, error) {
	if d == nil || d.document == nil || writer == nil {
		return nil, ErrInvalidRender
	}
	internal, err := internalHTMLDocumentOptions(options)
	if err != nil {
		return nil, err
	}
	var collector htmlSourceMapCollector
	if err := d.document.RenderHTMLDocumentWithSourceMap(writer, internal, collector.add); err != nil {
		if errors.Is(err, renderhtml.ErrInvalidInput) {
			return nil, translateError(err, renderhtml.ErrInvalidInput, ErrInvalidRender)
		}
		return nil, err
	}
	return collector.result(), nil
}

// HTMLDocumentWithSourceMap renders a deterministic standalone HTML document
// and returns caller-owned bytes plus mappings for that exact output.
func (d *Document) HTMLDocumentWithSourceMap(options HTMLDocumentOptions) ([]byte, []HTMLSourceMapEntry, error) {
	if d == nil || d.document == nil {
		return nil, nil, ErrInvalidRender
	}
	var output bytes.Buffer
	mappings, err := d.RenderHTMLDocumentWithSourceMap(&output, options)
	if err != nil {
		return nil, nil, err
	}
	return output.Bytes(), mappings, nil
}

const htmlSourceMapChunkSize = 1024

type htmlSourceMapCollector struct {
	chunks  [][]HTMLSourceMapEntry
	current []HTMLSourceMapEntry
	count   int
}

func (c *htmlSourceMapCollector) add(mapping renderhtml.SourceMapEntry) {
	if len(c.current) == cap(c.current) {
		if len(c.current) != 0 {
			c.chunks = append(c.chunks, c.current)
		}
		capacity := 64
		if len(c.chunks) != 0 {
			capacity = htmlSourceMapChunkSize
		}
		c.current = make([]HTMLSourceMapEntry, 0, capacity)
	}
	c.current = append(c.current, HTMLSourceMapEntry{
		sourceRange: Range{Start: mapping.Source.Start, End: mapping.Source.End},
		outputRange: HTMLOutputRange{Start: mapping.Output.Start, End: mapping.Output.End},
	})
	c.count++
}

func (c *htmlSourceMapCollector) result() []HTMLSourceMapEntry {
	if c.count == 0 {
		return nil
	}
	if len(c.chunks) == 0 {
		result := c.current
		sortPublicHTMLSourceMap(result)
		return result
	}
	result := make([]HTMLSourceMapEntry, 0, c.count)
	for _, chunk := range c.chunks {
		result = append(result, chunk...)
	}
	result = append(result, c.current...)
	sortPublicHTMLSourceMap(result)
	return result
}

func sortPublicHTMLSourceMap(mappings []HTMLSourceMapEntry) {
	sort.Slice(mappings, func(left, right int) bool {
		leftOutput := mappings[left].outputRange
		rightOutput := mappings[right].outputRange
		if leftOutput.Start != rightOutput.Start {
			return leftOutput.Start < rightOutput.Start
		}
		if leftOutput.End != rightOutput.End {
			return leftOutput.End > rightOutput.End
		}
		leftSource := mappings[left].sourceRange
		rightSource := mappings[right].sourceRange
		if leftSource.Start != rightSource.Start {
			return leftSource.Start < rightSource.Start
		}
		return leftSource.End > rightSource.End
	})
}
