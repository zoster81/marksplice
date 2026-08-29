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
	Title            string
	Description      string
	Author           string
	Language         string
	TitleRange       parser.Range
	DescriptionRange parser.Range
	AuthorRange      parser.Range
	LanguageRange    parser.Range
	HasTitle         bool
	HasDescription   bool
	HasAuthor        bool
	HasLanguage      bool
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
	if err := writeDocumentOpen(writer, metadata, nil, nil); err != nil {
		return err
	}
	if err := Render(writer, source, backend, options.Body); err != nil {
		return err
	}
	return writeAllString(writer, "</body>\n</html>\n")
}

// RenderDocumentWithSourceMap streams one standalone document and synchronously
// reports absolute source-to-output byte mappings.
func RenderDocumentWithSourceMap(writer io.Writer, source []byte, backend parser.SemanticBackend, options DocumentOptions, metadata DocumentMetadata, collect SourceMapCollector) error {
	if writer == nil || backend == nil || collect == nil || options.Metadata > MetadataOmit {
		return ErrInvalidInput
	}
	if err := validateOptions(options.Body); err != nil {
		return err
	}
	if options.Metadata == MetadataOmit {
		metadata = DocumentMetadata{}
	}
	counter := &countingWriter{writer: writer}
	if err := writeDocumentOpen(counter, metadata, counter, collect); err != nil {
		return err
	}
	if err := renderMapped(counter, counter, source, backend, options.Body, collect); err != nil {
		return err
	}
	return writeAllString(counter, "</body>\n</html>\n")
}

func writeDocumentOpen(writer io.Writer, metadata DocumentMetadata, counter *countingWriter, collect SourceMapCollector) error {
	if err := writeAllString(writer, "<!doctype html>\n<html"); err != nil {
		return err
	}
	if metadata.HasLanguage {
		if err := writeDocumentMappedString(writer, counter, collect, metadata.LanguageRange, " lang=\""+escapeAttribute(metadata.Language)+"\""); err != nil {
			return err
		}
	}
	if err := writeAllString(writer, ">\n<head>\n<meta charset=\"utf-8\">\n"); err != nil {
		return err
	}
	if metadata.HasTitle {
		if err := writeDocumentMappedString(writer, counter, collect, metadata.TitleRange, "<title>"+escapeText(metadata.Title)+"</title>\n"); err != nil {
			return err
		}
	}
	if metadata.HasDescription {
		if err := writeMetadataName(writer, counter, collect, metadata.DescriptionRange, "description", metadata.Description); err != nil {
			return err
		}
	}
	if metadata.HasAuthor {
		if err := writeMetadataName(writer, counter, collect, metadata.AuthorRange, "author", metadata.Author); err != nil {
			return err
		}
	}
	return writeAllString(writer, "</head>\n<body>\n")
}

func writeMetadataName(writer io.Writer, counter *countingWriter, collect SourceMapCollector, sourceRange parser.Range, name, value string) error {
	return writeDocumentMappedString(writer, counter, collect, sourceRange, "<meta name=\""+name+"\" content=\""+escapeAttribute(value)+"\">\n")
}

func writeDocumentMappedString(writer io.Writer, counter *countingWriter, collect SourceMapCollector, sourceRange parser.Range, value string) error {
	start := 0
	if counter != nil {
		start = counter.offset
	}
	if err := writeAllString(writer, value); err != nil {
		return err
	}
	if counter != nil {
		emitSourceMap(collect, sourceRange, start, counter.offset)
	}
	return nil
}
