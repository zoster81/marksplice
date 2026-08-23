package publictest

import (
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM96PromotedLinkFamilyExposesProvenSemanticValues(t *testing.T) {
	t.Parallel()

	source := []byte("[label](<dest/path> \"Link title\") <https://example.test/path> <user@example.test>\n\n[Docs]: <ref/path> 'Reference title'\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	seenLink := false
	seenReference := false
	seenURL := false
	seenEmail := false
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindInlineLink:
			detail, ok := doc.InlineLink(node.ID())
			if !ok {
				t.Fatal("InlineLink() ok = false")
			}
			if detail.Destination() != "dest/path" {
				t.Fatalf("InlineLink.Destination() = %q, want dest/path", detail.Destination())
			}
			title, ok := detail.Title()
			if !ok || title != "Link title" {
				t.Fatalf("InlineLink.Title() = %q/%v, want Link title/true", title, ok)
			}
			seenLink = true
		case marksplice.KindReferenceDefinition:
			detail, ok := doc.ReferenceDefinition(node.ID())
			if !ok {
				t.Fatal("ReferenceDefinition() ok = false")
			}
			if detail.Label() != "Docs" || detail.Destination() != "ref/path" {
				t.Fatalf("reference semantic values = label %q destination %q", detail.Label(), detail.Destination())
			}
			title, ok := detail.Title()
			if !ok || title != "Reference title" {
				t.Fatalf("ReferenceDefinition.Title() = %q/%v, want Reference title/true", title, ok)
			}
			seenReference = true
		case marksplice.KindAutoLink:
			detail, ok := doc.AutoLink(node.ID())
			if !ok {
				t.Fatal("AutoLink() ok = false")
			}
			switch detail.Value() {
			case "https://example.test/path":
				if detail.IsEmail() {
					t.Fatal("URL autolink IsEmail() = true")
				}
				seenURL = true
			case "user@example.test":
				if !detail.IsEmail() {
					t.Fatal("email autolink IsEmail() = false")
				}
				seenEmail = true
			}
		}
	}
	if !seenLink || !seenReference || !seenURL || !seenEmail {
		t.Fatalf("semantic promoted kinds found = link %v reference %v url %v email %v", seenLink, seenReference, seenURL, seenEmail)
	}
}

func TestM96LinkFamilySemanticZeroValues(t *testing.T) {
	t.Parallel()

	var link marksplice.InlineLink
	if link.Destination() != "" {
		t.Fatalf("zero InlineLink.Destination() = %q", link.Destination())
	}
	if title, ok := link.Title(); ok || title != "" {
		t.Fatalf("zero InlineLink.Title() = %q/%v", title, ok)
	}

	var definition marksplice.ReferenceDefinition
	if definition.Label() != "" || definition.Destination() != "" {
		t.Fatalf("zero ReferenceDefinition = label %q destination %q", definition.Label(), definition.Destination())
	}
	if title, ok := definition.Title(); ok || title != "" {
		t.Fatalf("zero ReferenceDefinition.Title() = %q/%v", title, ok)
	}

	var autoLink marksplice.AutoLink
	if autoLink.Value() != "" || autoLink.IsEmail() {
		t.Fatalf("zero AutoLink = value %q email %v", autoLink.Value(), autoLink.IsEmail())
	}
}
