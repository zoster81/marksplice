package marksplice

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/zoster81/marksplice/internal/source"
)

// FrontMatterFieldInput is construction-only input for one canonical simple
// YAML/TOML string scalar.
//
// Key uses the existing conservative front-matter key alphabet. Value is
// written as one double-quoted string and therefore must not require escaping.
type FrontMatterFieldInput struct {
	Key   string
	Value string
}

type constructionFrontMatter struct {
	format FrontMatterFormat
	fields []FrontMatterFieldInput
}

// SetYAMLFrontMatter configures one canonical leading YAML front-matter envelope.
// A DocumentBuilder can own at most one front-matter envelope.
func (b *DocumentBuilder) SetYAMLFrontMatter(fields ...FrontMatterFieldInput) error {
	return b.setFrontMatter(FrontMatterFormatYAML, fields)
}

// SetTOMLFrontMatter configures one canonical leading TOML front-matter envelope.
// A DocumentBuilder can own at most one front-matter envelope.
func (b *DocumentBuilder) SetTOMLFrontMatter(fields ...FrontMatterFieldInput) error {
	return b.setFrontMatter(FrontMatterFormatTOML, fields)
}

func (b *DocumentBuilder) setFrontMatter(format FrontMatterFormat, fields []FrontMatterFieldInput) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	if b.frontMatter != nil {
		return fmt.Errorf("%w: document builder already has front matter", ErrInvalidConstruction)
	}
	frontMatter := constructionFrontMatter{format: format, fields: append([]FrontMatterFieldInput(nil), fields...)}
	if err := validateConstructionFrontMatterInput(frontMatter); err != nil {
		return err
	}
	candidate := writeConstructionFrontMatter(frontMatter)
	if err := validateConstructionFrontMatter(candidate, frontMatter); err != nil {
		return err
	}
	b.frontMatter = &frontMatter
	return nil
}

func validateConstructionFrontMatterInput(frontMatter constructionFrontMatter) error {
	if frontMatter.format != FrontMatterFormatYAML && frontMatter.format != FrontMatterFormatTOML {
		return fmt.Errorf("%w: unsupported front-matter format", ErrInvalidConstruction)
	}
	if len(frontMatter.fields) == 0 {
		return fmt.Errorf("%w: front matter is empty", ErrInvalidConstruction)
	}
	seen := make(map[string]struct{}, len(frontMatter.fields))
	for index, field := range frontMatter.fields {
		if err := validateConstructionFrontMatterKey(field.Key); err != nil {
			return fmt.Errorf("%w: field %d key: %v", ErrInvalidConstruction, index, err)
		}
		if _, exists := seen[field.Key]; exists {
			return fmt.Errorf("%w: duplicate front-matter key %q", ErrInvalidConstruction, field.Key)
		}
		seen[field.Key] = struct{}{}
		if err := validateConstructionFrontMatterValue(field.Value); err != nil {
			return fmt.Errorf("%w: field %q value: %v", ErrInvalidConstruction, field.Key, err)
		}
	}
	return nil
}

func validateConstructionFrontMatterKey(key string) error {
	if key == "" {
		return fmt.Errorf("key is empty")
	}
	for index := 0; index < len(key); index++ {
		if !isConstructionFrontMatterKeyByte(key[index]) {
			return fmt.Errorf("key contains unsupported byte")
		}
	}
	return nil
}

func isConstructionFrontMatterKeyByte(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9' || value == '_' || value == '-' || value == '.'
}

func validateConstructionFrontMatterValue(value string) error {
	if value == "" {
		return fmt.Errorf("value is empty")
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("value is not valid UTF-8")
	}
	if strings.ContainsAny(value, "\r\n\"\\") || strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("value requires escaping or crosses a physical line")
	}
	for index := 0; index < len(value); index++ {
		if value[index] < 0x20 {
			return fmt.Errorf("value contains an ASCII control byte")
		}
	}
	return nil
}

func writeConstructionFrontMatter(frontMatter constructionFrontMatter) []byte {
	var output bytes.Buffer
	writeConstructionFrontMatterTo(&output, frontMatter)
	return output.Bytes()
}

func writeConstructionFrontMatterTo(output *bytes.Buffer, frontMatter constructionFrontMatter) {
	delimiter := "---"
	separator := ": "
	if frontMatter.format == FrontMatterFormatTOML {
		delimiter = "+++"
		separator = " = "
	}
	output.WriteString(delimiter)
	output.WriteByte('\n')
	for _, field := range frontMatter.fields {
		output.WriteString(field.Key)
		output.WriteString(separator)
		output.WriteByte('"')
		output.WriteString(field.Value)
		output.WriteByte('"')
		output.WriteByte('\n')
	}
	output.WriteString(delimiter)
	output.WriteByte('\n')
}

func validateConstructionFrontMatter(candidate []byte, expected constructionFrontMatter) error {
	mapping, format, err := constructionFrontMatterMapping(candidate, expected)
	if err != nil {
		return err
	}
	for index, want := range expected.fields {
		if !sameConstructionFrontMatterField(candidate, mapping.Fields[index], want, format) {
			return fmt.Errorf("%w: generated front-matter field %d changed", ErrInvalidConstruction, index)
		}
	}
	return nil
}

func constructionFrontMatterMapping(candidate []byte, expected constructionFrontMatter) (source.FrontMatterMapping, source.FrontMatterFormat, error) {
	mapping, ok := source.MapLeadingFrontMatter(candidate)
	format, okFormat := constructionSourceFrontMatterFormat(expected.format)
	if !ok || !okFormat || mapping.Format != format || len(mapping.Fields) != len(expected.fields) {
		return source.FrontMatterMapping{}, source.FrontMatterUnknown, fmt.Errorf("%w: generated front-matter envelope changed", ErrInvalidConstruction)
	}
	if mapping.Range.Start != 0 || mapping.OpeningRange.Start != 0 {
		return source.FrontMatterMapping{}, source.FrontMatterUnknown, fmt.Errorf("%w: generated front matter is not document-leading", ErrInvalidConstruction)
	}
	return mapping, format, nil
}

func sameConstructionFrontMatterField(candidate []byte, got source.FrontMatterFieldMapping, want FrontMatterFieldInput, format source.FrontMatterFormat) bool {
	return got.Key == want.Key && got.Format == format && got.Style == source.FrontMatterValueDoubleQuoted && got.Quote == '"' &&
		got.KeyRange.Valid(len(candidate)) && got.ValueRange.Valid(len(candidate)) &&
		string(candidate[got.KeyRange.Start:got.KeyRange.End]) == want.Key &&
		string(candidate[got.ValueRange.Start:got.ValueRange.End]) == want.Value
}

func constructionSourceFrontMatterFormat(format FrontMatterFormat) (source.FrontMatterFormat, bool) {
	switch format {
	case FrontMatterFormatYAML:
		return source.FrontMatterYAML, true
	case FrontMatterFormatTOML:
		return source.FrontMatterTOML, true
	default:
		return source.FrontMatterUnknown, false
	}
}
