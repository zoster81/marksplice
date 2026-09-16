package marksplice

import (
	"bytes"
	"errors"
	"io"

	"github.com/zoster81/marksplice/internal/rendermarkdown"
)

// RenderCanonicalMarkdown streams deterministic canonical Markdown from this
// immutable snapshot. Canonical rendering is an explicit export path and never
// replaces source-preserving existing-document edits.
func (d *Document) RenderCanonicalMarkdown(writer io.Writer) error {
	if d == nil || d.document == nil || writer == nil {
		return ErrInvalidRender
	}
	if err := d.document.RenderCanonicalMarkdown(writer); err != nil {
		if errors.Is(err, rendermarkdown.ErrInvalidInput) {
			return translateError(err, rendermarkdown.ErrInvalidInput, ErrInvalidRender)
		}
		return err
	}
	return nil
}

// CanonicalMarkdown renders deterministic canonical Markdown into caller-owned bytes.
func (d *Document) CanonicalMarkdown() ([]byte, error) {
	if d == nil || d.document == nil {
		return nil, ErrInvalidRender
	}
	var output bytes.Buffer
	if err := d.RenderCanonicalMarkdown(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
