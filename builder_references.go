package marksplice

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/splice"
)

// DeferReferenceDefinition schedules one canonical top-level reference
// definition after the ordinary constructed body. M93 uses this explicit
// document-level contract for forward typed references without changing the
// M89 prior-definition contract of ReferenceLinkInline/ReferenceImageInline.
func (b *DocumentBuilder) DeferReferenceDefinition(label, destination string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.deferReferenceDefinition(constructionBlock{
		kind:        constructionReferenceDefinition,
		label:       label,
		destination: destination,
	})
}

// DeferReferenceDefinitionWithTitle schedules one canonical top-level reference
// definition with a conservative double-quoted title after the ordinary body.
func (b *DocumentBuilder) DeferReferenceDefinitionWithTitle(label, destination, title string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.deferReferenceDefinition(constructionBlock{
		kind:        constructionReferenceDefinition,
		label:       label,
		destination: destination,
		title:       title,
		hasTitle:    true,
	})
}

func (b *DocumentBuilder) deferReferenceDefinition(block constructionBlock) error {
	if err := validateConstructionBlockStandalone(block); err != nil {
		return err
	}
	if referenceDefinitionLabelCollides(block.label, b.blocks) || referenceDefinitionLabelCollides(block.label, b.deferredReferences) {
		return fmt.Errorf("%w: deferred reference label conflicts with another normalized definition", ErrInvalidConstruction)
	}
	b.deferredReferences = append(b.deferredReferences, block)
	return nil
}

func (b *DocumentBuilder) rejectDeferredReferenceCollision(label string) error {
	if referenceDefinitionLabelCollides(label, b.deferredReferences) {
		return fmt.Errorf("%w: reference label conflicts with a deferred normalized definition", ErrInvalidConstruction)
	}
	return nil
}

func referenceDefinitionLabelCollides(label string, blocks []constructionBlock) bool {
	for _, block := range blocks {
		if block.kind == constructionReferenceDefinition && splice.ConstructionReferenceLabelsEquivalent(label, block.label) {
			return true
		}
	}
	return false
}

func (b *DocumentBuilder) constructionDocumentBlocks() []constructionBlock {
	if len(b.deferredReferences) == 0 {
		return b.blocks
	}
	blocks := make([]constructionBlock, 0, len(b.blocks)+len(b.deferredReferences))
	blocks = append(blocks, b.blocks...)
	blocks = append(blocks, b.deferredReferences...)
	return blocks
}
