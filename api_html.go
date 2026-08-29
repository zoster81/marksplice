package marksplice

import (
	"bytes"
	"errors"
	"io"

	"github.com/zoster81/marksplice/internal/renderhtml"
	"github.com/zoster81/marksplice/internal/splice"
)

// ErrInvalidRender reports invalid HTML renderer input or options.
var ErrInvalidRender = errors.New("invalid render")

// HTMLRawPolicy controls renderer handling of parser-proven raw HTML.
type HTMLRawPolicy uint8

const (
	// HTMLRawPreserve preserves parser-proven raw HTML. With tag filtering enabled,
	// GFM-disallowed tag starts are escaped before writing.
	HTMLRawPreserve HTMLRawPolicy = iota
	// HTMLRawEscape escapes all parser-proven raw HTML as text.
	HTMLRawEscape
)

// HTMLUnsafeURLPolicy controls renderer handling of dangerous URL schemes.
type HTMLUnsafeURLPolicy uint8

const (
	// HTMLUnsafeURLSuppress replaces dangerous link/image destinations with an empty URL.
	HTMLUnsafeURLSuppress HTMLUnsafeURLPolicy = iota
	// HTMLUnsafeURLAllow emits parser-resolved destinations without scheme suppression.
	HTMLUnsafeURLAllow
)

// HTMLTagFilterPolicy controls the GFM disallowed-raw-HTML tag filter.
type HTMLTagFilterPolicy uint8

const (
	// HTMLTagFilterEnabled applies the published GFM disallowed-tag filter when raw HTML is preserved.
	HTMLTagFilterEnabled HTMLTagFilterPolicy = iota
	// HTMLTagFilterDisabled preserves parser-proven raw HTML without GFM tag filtering.
	HTMLTagFilterDisabled
)

// HTMLRenderOptions controls deterministic HTML-fragment rendering.
// Its zero value preserves raw HTML with the GFM tag filter enabled and suppresses
// dangerous URL schemes. Preserved raw HTML can still contain active markup; use
// HTMLRawEscape when rendering untrusted Markdown into an HTML security boundary.
type HTMLRenderOptions struct {
	RawHTML    HTMLRawPolicy
	UnsafeURLs HTMLUnsafeURLPolicy
	TagFilter  HTMLTagFilterPolicy
}

// DefaultHTMLRenderOptions returns the documented zero-value rendering policy.
func DefaultHTMLRenderOptions() HTMLRenderOptions {
	return HTMLRenderOptions{}
}

// RenderHTML streams a deterministic HTML fragment from this immutable snapshot.
// It performs no filesystem or network access and stops immediately on writer error.
func (d *Document) RenderHTML(writer io.Writer, options HTMLRenderOptions) error {
	if d == nil || d.document == nil || writer == nil {
		return ErrInvalidRender
	}
	internal, err := internalHTMLRenderOptions(options)
	if err != nil {
		return err
	}
	if err := d.document.RenderHTML(writer, internal); err != nil {
		if errors.Is(err, renderhtml.ErrInvalidInput) {
			return translateError(err, renderhtml.ErrInvalidInput, ErrInvalidRender)
		}
		return err
	}
	return nil
}

// HTML renders a deterministic HTML fragment into caller-owned bytes.
func (d *Document) HTML(options HTMLRenderOptions) ([]byte, error) {
	if d == nil || d.document == nil {
		return nil, ErrInvalidRender
	}
	var output bytes.Buffer
	if err := d.RenderHTML(&output, options); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func internalHTMLRenderOptions(options HTMLRenderOptions) (splice.HTMLRenderOptions, error) {
	if options.RawHTML > HTMLRawEscape || options.UnsafeURLs > HTMLUnsafeURLAllow || options.TagFilter > HTMLTagFilterDisabled {
		return splice.HTMLRenderOptions{}, ErrInvalidRender
	}
	return splice.HTMLRenderOptions{
		RawHTML:    splice.HTMLRawPolicy(options.RawHTML),
		UnsafeURLs: splice.HTMLUnsafeURLPolicy(options.UnsafeURLs),
		TagFilter:  splice.HTMLTagFilterPolicy(options.TagFilter),
	}, nil
}
