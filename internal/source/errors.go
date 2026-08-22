package source

import "errors"

var (
	ErrUnsupportedHeadingShape             = errors.New("unsupported heading source shape")
	ErrUnsupportedTaskMarker               = errors.New("unsupported task-list marker source shape")
	ErrUnsupportedListItemShape            = errors.New("unsupported list-item source shape")
	ErrUnsupportedTableCellShape           = errors.New("unsupported table-cell source shape")
	ErrUnsupportedFencedCodeShape          = errors.New("unsupported fenced-code source shape")
	ErrUnsupportedStrikethroughShape       = errors.New("unsupported strikethrough source shape")
	ErrUnsupportedInlineLinkShape          = errors.New("unsupported inline-link source shape")
	ErrUnsupportedImageShape               = errors.New("unsupported image source shape")
	ErrUnsupportedReferenceDefinitionShape = errors.New("unsupported reference-definition source shape")
	ErrUnsupportedAutoLinkShape            = errors.New("unsupported autolink source shape")
	ErrUnsupportedCodeSpanShape            = errors.New("unsupported code-span source shape")
	ErrUnsupportedEmphasisShape            = errors.New("unsupported emphasis source shape")
)
