package splice

import (
	"io"

	"github.com/zoster81/marksplice/internal/rendermarkdown"
)

// RenderCanonicalMarkdown streams deterministic canonical Markdown from this immutable snapshot.
func (d *Document) RenderCanonicalMarkdown(writer io.Writer) error {
	if d == nil {
		return rendermarkdown.ErrInvalidInput
	}
	return rendermarkdown.Render(writer, d.source, newSemanticBackend())
}
