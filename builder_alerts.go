package marksplice

import "fmt"

// AppendAlert appends one canonical top-level GitHub alert containing one
// parser-proven paragraph. kind must be one of Note, Tip, Important, Warning,
// or Caution. inlineGFM follows the same LF-only paragraph contract as AppendBlockquote.
func (b *DocumentBuilder) AppendAlert(kind AlertKind, inlineGFM string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	if _, ok := alertMarker(kind); !ok {
		return fmt.Errorf("%w: unsupported alert kind", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:      constructionAlert,
		alertKind: kind,
		inlineGFM: inlineGFM,
	})
}

// AppendAlertContent appends one canonical top-level GitHub alert from typed
// inline paragraph content.
func (b *DocumentBuilder) AppendAlertContent(kind AlertKind, content ...Inline) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	inlineGFM, err := b.renderTypedInlineConstruction(content)
	if err != nil {
		return err
	}
	return b.AppendAlert(kind, inlineGFM)
}

// AppendAlertBlocks appends one canonical top-level GitHub alert from the current
// reviewed body blocks of content. The child builder is snapshotted and later
// changes do not affect this builder. Alerts cannot be nested inside blockquotes
// or other alerts, so child alert blocks are rejected.
func (b *DocumentBuilder) AppendAlertBlocks(kind AlertKind, content *DocumentBuilder) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	if _, ok := alertMarker(kind); !ok {
		return fmt.Errorf("%w: unsupported alert kind", ErrInvalidConstruction)
	}
	if content == nil {
		return fmt.Errorf("%w: nil alert child builder", ErrInvalidConstruction)
	}
	if content == b {
		return fmt.Errorf("%w: alert child builder cannot be the destination builder", ErrInvalidConstruction)
	}
	if content.frontMatter != nil {
		return fmt.Errorf("%w: alert child builder cannot contain front matter", ErrInvalidConstruction)
	}
	if len(content.deferredReferences) != 0 {
		return fmt.Errorf("%w: alert child builder cannot contain deferred reference definitions", ErrInvalidConstruction)
	}
	if len(content.deferredFootnotes) != 0 {
		return fmt.Errorf("%w: alert child builder cannot contain deferred footnote definitions", ErrInvalidConstruction)
	}
	children := append([]constructionBlock(nil), content.blocks...)
	return b.appendConstructionBlock(constructionBlock{
		kind:      constructionAlertBlocks,
		alertKind: kind,
		children:  children,
	})
}
