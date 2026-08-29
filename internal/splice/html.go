package splice

import (
	"io"

	"github.com/zoster81/marksplice/internal/renderhtml"
)

// HTMLRawPolicy controls renderer handling of parser-proven raw HTML.
type HTMLRawPolicy = renderhtml.RawHTMLPolicy

const (
	HTMLRawPreserve = renderhtml.RawHTMLPreserve
	HTMLRawEscape   = renderhtml.RawHTMLEscape
)

// HTMLUnsafeURLPolicy controls renderer handling of dangerous URL schemes.
type HTMLUnsafeURLPolicy = renderhtml.UnsafeURLPolicy

const (
	HTMLUnsafeURLSuppress = renderhtml.UnsafeURLSuppress
	HTMLUnsafeURLAllow    = renderhtml.UnsafeURLAllow
)

// HTMLTagFilterPolicy controls the GFM disallowed-raw-HTML tag filter.
type HTMLTagFilterPolicy = renderhtml.TagFilterPolicy

const (
	HTMLTagFilterEnabled  = renderhtml.TagFilterEnabled
	HTMLTagFilterDisabled = renderhtml.TagFilterDisabled
)

// HTMLRenderOptions controls deterministic HTML-fragment rendering.
type HTMLRenderOptions = renderhtml.Options

// RenderHTML streams a deterministic HTML fragment from this immutable snapshot.
func (d *Document) RenderHTML(writer io.Writer, options HTMLRenderOptions) error {
	if d == nil {
		return renderhtml.ErrInvalidInput
	}
	return renderhtml.Render(writer, d.source, newSemanticBackend(), options)
}
