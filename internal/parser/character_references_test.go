package parser

import "testing"

func TestDecodeCommonMarkCharacterReference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw  string
		want string
		ok   bool
	}{
		{raw: "&amp;", want: "&", ok: true},
		{raw: "&ngE;", want: "≧̸", ok: true},
		{raw: "&nGt;", want: "≫⃒", ok: true},
		{raw: "&nLt;", want: "≪⃒", ok: true},
		{raw: "&#128;", want: "\u0080", ok: true},
		{raw: "&#0;", want: "�", ok: true},
		{raw: "&#xD800;", want: "�", ok: true},
		{raw: "&#x110000;", want: "�", ok: true},
		{raw: "&copy", ok: false},
		{raw: "&MadeUpEntity;", ok: false},
		{raw: "&#87654321;", ok: false},
		{raw: "&#abcdef0;", ok: false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.raw, func(t *testing.T) {
			got, ok := DecodeCommonMarkCharacterReference(tt.raw)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("DecodeCommonMarkCharacterReference(%q) = %q, %v; want %q, %v", tt.raw, got, ok, tt.want, tt.ok)
			}
		})
	}
}
