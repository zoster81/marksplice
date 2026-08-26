package source

import "testing"

func TestMapLeadingFrontMatterPreservesYAMLAndTOMLBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		format      FrontMatterFormat
		wantRange   Range
		wantClosing Range
		wantFields  map[string]FrontMatterFieldMapping
	}{
		{
			name:        "YAML CRLF plain scalars",
			source:      []byte("---\r\ntitle: old title  \r\ndraft: false\r\n---\r\n"),
			format:      FrontMatterYAML,
			wantRange:   Range{Start: 0, End: 42},
			wantClosing: Range{Start: 39, End: 42},
			wantFields: map[string]FrontMatterFieldMapping{
				"title": {Format: FrontMatterYAML, Range: Range{Start: 5, End: 23}, KeyRange: Range{Start: 5, End: 10}, ValueRange: Range{Start: 12, End: 21}, Key: "title", Style: FrontMatterValuePlain},
				"draft": {Format: FrontMatterYAML, Range: Range{Start: 25, End: 37}, KeyRange: Range{Start: 25, End: 30}, ValueRange: Range{Start: 32, End: 37}, Key: "draft", Style: FrontMatterValuePlain},
			},
		},
		{
			name:        "TOML LF quoted and bare scalars",
			source:      []byte("+++\n title = \"old title\"   # keep comment\ndraft = true\n+++\n"),
			format:      FrontMatterTOML,
			wantRange:   Range{Start: 0, End: 58},
			wantClosing: Range{Start: 55, End: 58},
			wantFields: map[string]FrontMatterFieldMapping{
				"title": {Format: FrontMatterTOML, Range: Range{Start: 4, End: 41}, KeyRange: Range{Start: 5, End: 10}, ValueRange: Range{Start: 14, End: 23}, Key: "title", Style: FrontMatterValueDoubleQuoted, Quote: '"'},
				"draft": {Format: FrontMatterTOML, Range: Range{Start: 42, End: 54}, KeyRange: Range{Start: 42, End: 47}, ValueRange: Range{Start: 50, End: 54}, Key: "draft", Style: FrontMatterValuePlain},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := MapLeadingFrontMatter(tt.source)
			if !ok {
				t.Fatal("MapLeadingFrontMatter() ok = false, want true")
			}
			if got.Format != tt.format || got.Range != tt.wantRange || got.OpeningRange != (Range{Start: 0, End: 3}) || got.ClosingRange != tt.wantClosing {
				t.Fatalf("mapping envelope = %+v, want format %d range %v closing %v", got, tt.format, tt.wantRange, tt.wantClosing)
			}
			if len(got.Fields) != len(tt.wantFields) {
				t.Fatalf("field count = %d, want %d: %+v", len(got.Fields), len(tt.wantFields), got.Fields)
			}
			for _, field := range got.Fields {
				want, ok := tt.wantFields[field.Key]
				if !ok {
					t.Fatalf("unexpected field %+v", field)
				}
				if field != want {
					t.Fatalf("field %q = %+v, want %+v", field.Key, field, want)
				}
			}
		})
	}
}

func TestMapLeadingFrontMatterRecognizesOpaqueAndEmptyEnvelopeWithoutEditableFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
		format FrontMatterFormat
	}{
		{name: "duplicate-only YAML", source: []byte("---\ntitle: one\ntitle: two\n---\n"), format: FrontMatterYAML},
		{name: "complex-only YAML", source: []byte("---\nitems:\n  - one\n---\n"), format: FrontMatterYAML},
		{name: "empty YAML", source: []byte("---\n---\n"), format: FrontMatterYAML},
		{name: "TOML table", source: []byte("+++\n[params]\nauthor = 'Ada'\n+++\n"), format: FrontMatterTOML},
		{name: "TOML array table", source: []byte("+++\n[[products]]\nname = 'Book'\n+++\n"), format: FrontMatterTOML},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mapping, ok := MapLeadingFrontMatter(tt.source)
			if !ok || mapping.Format != tt.format {
				t.Fatalf("MapLeadingFrontMatter() = %+v/%v, want format %v", mapping, ok, tt.format)
			}
			if len(mapping.Fields) != 0 {
				t.Fatalf("Fields = %+v, want opaque envelope with no editable fields", mapping.Fields)
			}
		})
	}
}

func TestMapLeadingFrontMatterRejectsMalformedTOMLTableEvidence(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("+++\n[\n+++\n"),
		[]byte("+++\n[params\n+++\n"),
		[]byte("+++\n[]\n+++\n"),
		[]byte("+++\n[[products]\n+++\n"),
		[]byte("+++\n[[ ]]\n+++\n"),
	} {
		if mapping, ok := MapLeadingFrontMatter(source); ok {
			t.Fatalf("MapLeadingFrontMatter(%q) = %+v, true; want false", source, mapping)
		}
	}
}

func TestMapLeadingFrontMatterStopsTopLevelTOMLFieldPromotionAtFirstTable(t *testing.T) {
	t.Parallel()

	source := []byte("+++\ntitle = 'Top'\n[params]\nauthor = 'Nested'\n+++\n")
	mapping, ok := MapLeadingFrontMatter(source)
	if !ok {
		t.Fatal("MapLeadingFrontMatter() ok = false")
	}
	if len(mapping.Fields) != 1 || mapping.Fields[0].Key != "title" {
		t.Fatalf("Fields = %+v, want only top-level title", mapping.Fields)
	}
}

func TestMapLeadingFrontMatterDoesNotTargetNonScalarYAMLOrBareTOMLStrings(t *testing.T) {
	t.Parallel()

	yamlSource := []byte("---\ntitle: # comment only\nitems: - one\ndraft: false\n---\n")
	yaml, ok := MapLeadingFrontMatter(yamlSource)
	if !ok {
		t.Fatal("MapLeadingFrontMatter(YAML) ok = false, want draft field to establish envelope")
	}
	for _, field := range yaml.Fields {
		if field.Key == "title" || field.Key == "items" {
			t.Fatalf("non-scalar YAML field unexpectedly targetable: %+v", field)
		}
	}
	if len(yaml.Fields) != 1 || yaml.Fields[0].Key != "draft" {
		t.Fatalf("YAML fields = %+v, want only draft", yaml.Fields)
	}

	tomlSource := []byte("+++\ntitle = old-title\ndraft = true\n+++\n")
	toml, ok := MapLeadingFrontMatter(tomlSource)
	if !ok {
		t.Fatal("MapLeadingFrontMatter(TOML) ok = false, want draft field to establish envelope")
	}
	for _, field := range toml.Fields {
		if field.Key == "title" {
			t.Fatalf("bare TOML string unexpectedly targetable: %+v", field)
		}
	}
	if len(toml.Fields) != 1 || toml.Fields[0].Key != "draft" {
		t.Fatalf("TOML fields = %+v, want only draft", toml.Fields)
	}
}

func TestMapLeadingFrontMatterFailsClosedOnAmbiguousOrUnsupportedEnvelope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
	}{
		{name: "not at document start", source: []byte("intro\n---\ntitle: value\n---\n")},
		{name: "unclosed YAML", source: []byte("---\ntitle: value\n")},
		{name: "ordinary thematic-break-looking content has no metadata evidence", source: []byte("---\nplain paragraph\n---\n")},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := MapLeadingFrontMatter(tt.source); ok {
				t.Fatalf("MapLeadingFrontMatter() = %+v, true; want fail-closed false", got)
			}
		})
	}
}
