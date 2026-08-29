package splice

import (
	"io"
	"unicode/utf8"

	"github.com/zoster81/marksplice/internal/renderhtml"
	"github.com/zoster81/marksplice/internal/source"
)

// HTMLMetadataPolicy controls reviewed front-matter metadata emission.
type HTMLMetadataPolicy = renderhtml.MetadataPolicy

const (
	HTMLMetadataFrontMatter = renderhtml.MetadataFrontMatter
	HTMLMetadataOmit        = renderhtml.MetadataOmit
)

// HTMLDocumentOptions controls deterministic standalone HTML rendering.
type HTMLDocumentOptions = renderhtml.DocumentOptions

// RenderHTMLDocument streams a deterministic standalone HTML document.
func (d *Document) RenderHTMLDocument(writer io.Writer, options HTMLDocumentOptions) error {
	if d == nil {
		return renderhtml.ErrInvalidInput
	}
	metadata := renderhtml.DocumentMetadata{}
	if options.Metadata == renderhtml.MetadataFrontMatter {
		metadata = d.htmlDocumentMetadata()
	}
	return renderhtml.RenderDocument(writer, d.source, newSemanticBackend(), options, metadata)
}

func (d *Document) htmlDocumentMetadata() renderhtml.DocumentMetadata {
	var metadata renderhtml.DocumentMetadata
	if d == nil || d.frontMatter.Format == source.FrontMatterUnknown {
		return metadata
	}
	for _, node := range d.nodes {
		if node.Range.Start >= d.frontMatter.ClosingRange.End {
			break
		}
		if node.Kind != KindYAMLFrontMatterField && node.Kind != KindTOMLFrontMatterField {
			continue
		}
		value, ok := d.safeHTMLMetadataScalar(node)
		if !ok {
			continue
		}
		switch node.Key {
		case "title":
			metadata.Title, metadata.HasTitle = value, true
		case "description":
			metadata.Description, metadata.HasDescription = value, true
		case "author":
			metadata.Author, metadata.HasAuthor = value, true
		case "lang":
			if safeHTMLLanguage(value) {
				metadata.Language, metadata.HasLanguage = value, true
			}
		}
	}
	return metadata
}

func (d *Document) safeHTMLMetadataScalar(node Node) (string, bool) {
	if d == nil || !node.ContentRange.Valid(len(d.source)) || node.ContentRange.Start == node.ContentRange.End {
		return "", false
	}
	value := d.source[node.ContentRange.Start:node.ContentRange.End]
	if !utf8.Valid(value) || hasHTMLMetadataControl(value) {
		return "", false
	}
	switch node.FrontMatterStyle {
	case source.FrontMatterValuePlain:
		return string(value), true
	case source.FrontMatterValueSingleQuoted:
		if node.FrontMatterFormat == FrontMatterFormatYAML {
			for _, current := range value {
				if current == '\'' {
					return "", false
				}
			}
		}
		return string(value), true
	case source.FrontMatterValueDoubleQuoted:
		for _, current := range value {
			if current == '\\' {
				return "", false
			}
		}
		return string(value), true
	default:
		return "", false
	}
}

func hasHTMLMetadataControl(value []byte) bool {
	for _, current := range value {
		if current < 0x20 || current == 0x7f {
			return true
		}
	}
	return false
}

func safeHTMLLanguage(value string) bool {
	if value == "" || len(value) > 255 {
		return false
	}
	previousHyphen := false
	for index := 0; index < len(value); index++ {
		current := value[index]
		alpha := current >= 'a' && current <= 'z' || current >= 'A' && current <= 'Z'
		digit := current >= '0' && current <= '9'
		if index == 0 && !alpha {
			return false
		}
		if alpha || digit {
			previousHyphen = false
			continue
		}
		if current != '-' || previousHyphen || index+1 == len(value) {
			return false
		}
		previousHyphen = true
	}
	return true
}
