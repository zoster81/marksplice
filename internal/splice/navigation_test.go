package splice

import (
	"reflect"
	"testing"
)

func TestGitHubHeadingSlugRules(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		" Sample Section ":                      "sample-section",
		"This'll be Helpful!":                   "thisll-be-helpful",
		"two  spaces":                           "two--spaces",
		"tab\tseparated":                        "tabseparated",
		"keep-under_score-and-hyphen":           "keep-under_score-and-hyphen",
		"Привет non-latin 你好":                   "привет-non-latin-你好",
		"😄 emoji":                               "-emoji",
		"This'll be About the Greek Letter Θ!":  "thisll-be-about-the-greek-letter-θ",
		"punctuation: []{}()!?/\\.,;:\"'@#$%&*": "punctuation-",
	}
	for input, want := range tests {
		input, want := input, want
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			if got := githubHeadingSlug(input); got != want {
				t.Fatalf("githubHeadingSlug(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestHeadingSemanticTextResolvesMarkupEntitiesButPreservesCode(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("# A &amp; B\n## [Linked](https://example.com) _value_\n## `A &amp; B`\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchors := doc.HeadingAnchors()
	got := make([]string, len(anchors))
	for index, anchor := range anchors {
		got[index] = anchor.Value
	}
	want := []string{"a--b", "linked-value", "a-amp-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("HeadingAnchors() = %#v, want %#v", got, want)
	}
}

func TestFragmentResolverReusesM98ResolutionSemantics(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("# Café\n\n<a id=\"dup\"></a>\n\n## Dup\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	resolve := doc.FragmentResolver()
	for _, fragment := range []string{"#caf%C3%A9", "caf%C3%A9", "#dup", "#missing", "#"} {
		want, wantOK := doc.ResolveFragment(fragment)
		got, gotOK := resolve(fragment)
		if gotOK != wantOK || got != want {
			t.Fatalf("FragmentResolver(%q) = %+v/%v, ResolveFragment = %+v/%v", fragment, got, gotOK, want, wantOK)
		}
	}

	var nilDoc *Document
	if _, ok := nilDoc.FragmentResolver()("#anything"); ok {
		t.Fatal("nil FragmentResolver unexpectedly resolved a fragment")
	}
}

func TestTOCBodyProfileAcceptsOnlyManagedLocalLinkShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		ok   bool
	}{
		{name: "empty", body: "", ok: true},
		{name: "blank", body: "\n", ok: true},
		{name: "valid", body: "\n- [Root](#old)\n  - [Child](#child)\n\n", ok: true},
		{name: "internal blank", body: "- [Root](#root)\n\n- [Child](#child)\n", ok: false},
		{name: "odd indent", body: "- [Root](#root)\n   - [Child](#child)\n", ok: false},
		{name: "depth jump", body: "- [Root](#root)\n    - [Deep](#deep)\n", ok: false},
		{name: "plain list", body: "- ordinary\n", ok: false},
		{name: "external link", body: "- [External](https://example.com)\n", ok: false},
		{name: "raw second hash", body: "- [Bad](#one#two)\n", ok: false},
		{name: "mixed line endings", body: "- [Root](#root)\r\n- [Child](#child)\n", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source := []byte("# Contents\n" + tt.body)
			doc, err := Parse(source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			section, ok := doc.SectionAt(0)
			if !ok {
				t.Fatal("SectionAt(0) missing")
			}
			_, got := doc.tocBodyProfile(section.BodyRange)
			if got != tt.ok {
				t.Fatalf("tocBodyProfile(%q) ok = %v, want %v", tt.body, got, tt.ok)
			}
		})
	}
}
