package renderhtml

import (
	"io"

	"github.com/zoster81/marksplice/internal/parser"
)

// MetadataPolicy controls reviewed front-matter metadata emission in standalone HTML.
type MetadataPolicy uint8

const (
	MetadataFrontMatter MetadataPolicy = iota
	MetadataOmit
)

// DocumentOptions controls deterministic standalone HTML rendering.
type DocumentOptions struct {
	Body     Options
	Metadata MetadataPolicy
}

// DocumentMetadata is the already-reviewed scalar metadata available to the HTML wrapper.
type DocumentMetadata struct {
	Title          string
	Description    string
	Author         string
	Language       string
	HasTitle       bool
	HasDescription bool
	HasAuthor      bool
	HasLanguage    bool
}

// RenderDocument streams one deterministic standalone HTML document around the fragment renderer.
func RenderDocument(writer io.Writer, source []byte, backend parser.SemanticBackend, options DocumentOptions, metadata DocumentMetadata) error {
	if writer == nil || backend == nil || options.Metadata > MetadataOmit {
		return ErrInvalidInput
	}
	if err := validateOptions(options.Body); err != nil {
		return err
	}
	if options.Metadata == MetadataOmit {
		metadata = DocumentMetadata{}
	}
	if err := writeDocumentOpen(writer, metadata); err != nil {
		return err
	}
	if err := Render(writer, source, backend, options.Body); err != nil {
		return err
	}
	return writeAllString(writer, "</body>\n</html>\n")
}

func writeDocumentOpen(writer io.Writer, metadata DocumentMetadata) error {
	if err := writeAllString(writer, "<!doctype html>\n<html"); err != nil {
		return err
	}
	if metadata.HasLanguage {
		if err := writeAllString(writer, " lang=\""+escapeAttribute(metadata.Language)+"\""); err != nil {
			return err
		}
	}
	if err := writeAllString(writer, ">\n<head>\n<meta charset=\"utf-8\">\n"); err != nil {
		return err
	}
	if metadata.HasTitle {
		if err := writeAllString(writer, "<title>"+escapeText(metadata.Title)+"</title>\n"); err != nil {
			return err
		}
	}
	if metadata.HasDescription {
		if err := writeMetadataName(writer, "description", metadata.Description); err != nil {
			return err
		}
	}
	if metadata.HasAuthor {
		if err := writeMetadataName(writer, "author", metadata.Author); err != nil {
			return err
		}
	}
	return writeAllString(writer, "</head>\n<body>\n")
}

func writeMetadataName(writer io.Writer, name, value string) error {
	return writeAllString(writer, "<meta name=\""+name+"\" content=\""+escapeAttribute(value)+"\">\n")
}
