package marksplice

import (
	"bytes"
	"errors"
	"io"

	"github.com/zoster81/marksplice/internal/renderhtml"
	"github.com/zoster81/marksplice/internal/splice"
)

// HTMLMetadataPolicy controls whether standalone HTML maps reviewed front-matter metadata.
type HTMLMetadataPolicy uint8

const (
	// HTMLMetadataFrontMatter maps the reviewed title, description, author, and lang
	// fields only when they are already source-proven simple front-matter scalars.
	HTMLMetadataFrontMatter HTMLMetadataPolicy = iota
	// HTMLMetadataOmit omits all front-matter-derived HTML metadata.
	HTMLMetadataOmit
)

// HTMLDocumentOptions controls deterministic standalone HTML rendering.
// Its zero value uses the normal fragment-rendering defaults and maps the small
// reviewed front-matter metadata set when those fields are safely available.
type HTMLDocumentOptions struct {
	Body     HTMLRenderOptions
	Metadata HTMLMetadataPolicy
}

// DefaultHTMLDocumentOptions returns the documented zero-value standalone policy.
func DefaultHTMLDocumentOptions() HTMLDocumentOptions {
	return HTMLDocumentOptions{}
}

// RenderHTMLDocument streams a deterministic standalone HTML document.
// It performs no filesystem, network, asset, template, or command access.
func (d *Document) RenderHTMLDocument(writer io.Writer, options HTMLDocumentOptions) error {
	if d == nil || d.document == nil || writer == nil {
		return ErrInvalidRender
	}
	internal, err := internalHTMLDocumentOptions(options)
	if err != nil {
		return err
	}
	if err := d.document.RenderHTMLDocument(writer, internal); err != nil {
		if errors.Is(err, renderhtml.ErrInvalidInput) {
			return translateError(err, renderhtml.ErrInvalidInput, ErrInvalidRender)
		}
		return err
	}
	return nil
}

// HTMLDocument renders a deterministic standalone HTML document into caller-owned bytes.
func (d *Document) HTMLDocument(options HTMLDocumentOptions) ([]byte, error) {
	if d == nil || d.document == nil {
		return nil, ErrInvalidRender
	}
	var output bytes.Buffer
	if err := d.RenderHTMLDocument(&output, options); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func internalHTMLDocumentOptions(options HTMLDocumentOptions) (splice.HTMLDocumentOptions, error) {
	body, err := internalHTMLRenderOptions(options.Body)
	if err != nil {
		return splice.HTMLDocumentOptions{}, err
	}
	if options.Metadata > HTMLMetadataOmit {
		return splice.HTMLDocumentOptions{}, ErrInvalidRender
	}
	return splice.HTMLDocumentOptions{
		Body:     body,
		Metadata: splice.HTMLMetadataPolicy(options.Metadata),
	}, nil
}
