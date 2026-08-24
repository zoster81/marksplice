package marksplice

import "fmt"

// DocumentBuilder constructs a new GFM document independently from parsed source snapshots.
//
// M44-M103 support one optional document-leading YAML/TOML front-matter
// envelope, top-level ATX headings, parser-proven paragraphs, thematic breaks,
// parser-proven single-paragraph and reviewed multi-block blockquotes, exact
// non-nested GitHub alerts layered over canonical top-level blockquotes, flat or
// homogeneous nested unordered/ordered lists and task lists, supported fenced code,
// simple reference definitions,
// canonical unaligned/aligned tables, and typed inline construction for
// semantic text plus conservative code/emphasis/strong, strikethrough, link,
// image, and angle-autolink content, including bounded reviewed structured
// inline nesting. Generated documents use LF line endings,
// one blank line between GFM blocks, one blank line between retained front
// matter and a non-empty body, and one final line ending.
type DocumentBuilder struct {
	frontMatter        *constructionFrontMatter
	blocks             []constructionBlock
	deferredReferences []constructionBlock
}

// NewDocumentBuilder returns an empty new-document builder.
func NewDocumentBuilder() *DocumentBuilder {
	return &DocumentBuilder{}
}

// TaskListItem is structured input for one newly constructed GFM task-list item.
// InlineGFM is caller-provided inline GFM source; Checked selects the canonical
// '[x]' or '[ ]' task marker written before that content.
type TaskListItem struct {
	InlineGFM string
	Checked   bool
}

// ListItemInput is construction-only structured input for one item in a nested list.
//
// Depth is structural depth, not source indentation: zero denotes a top-level
// item and each child is exactly one level deeper than its parent. DocumentBuilder
// derives canonical indentation from the generated parent marker width.
type ListItemInput struct {
	InlineGFM string
	Depth     int
}

// TaskListItemInput is construction-only structured input for one item in a
// nested task list. Depth follows the same structural contract as ListItemInput.
type TaskListItemInput struct {
	InlineGFM string
	Checked   bool
	Depth     int
}

// TableAlignment identifies the semantic alignment of a GFM table column.
// It is used for new-document construction and read-only parsed table alignment access.
type TableAlignment uint8

const (
	TableAlignmentDefault TableAlignment = iota
	TableAlignmentLeft
	TableAlignmentRight
	TableAlignmentCenter
)

// AppendHeading appends one top-level ATX heading.
//
// inlineGFM must be one non-empty physical line of valid UTF-8 GFM source. The
// generated source is accepted only when reparsing proves the requested heading
// level and exact content range.
func (b *DocumentBuilder) AppendHeading(level int, inlineGFM string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	if level < 1 || level > 6 {
		return fmt.Errorf("%w: heading level must be between 1 and 6", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:      constructionHeading,
		level:     level,
		inlineGFM: inlineGFM,
	})
}

// AppendParagraph appends one top-level paragraph.
//
// M54 accepts non-empty valid UTF-8 GFM source with canonical LF line endings.
// The complete input must reparse as exactly one top-level paragraph; input that
// becomes multiple blocks or another block kind fails closed instead of being
// escaped or normalized implicitly.
func (b *DocumentBuilder) AppendParagraph(inlineGFM string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:      constructionParagraph,
		inlineGFM: inlineGFM,
	})
}

// AppendBlockquote appends one top-level blockquote containing one paragraph.
//
// M56 introduced the canonical single-line '> ' form. M81 extends the same API
// to non-empty LF-separated paragraph GFM and writes canonical '> ' on every
// physical line. The block is retained only when construction-only source and
// semantic proof reproduce exactly one top-level blockquote paragraph; broader
// existing-source blockquote promotion remains unchanged.
func (b *DocumentBuilder) AppendBlockquote(inlineGFM string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:      constructionBlockquote,
		depth:     1,
		inlineGFM: inlineGFM,
	})
}

// AppendNestedBlockquote appends one explicitly nested blockquote containing one paragraph.
//
// depth is structural container depth and must be between 2 and 64. The writer
// derives the canonical repeated "> " prefix on every physical line; caller
// content remains raw paragraph GFM and must not introduce container structure
// that changes the requested nesting hierarchy.
func (b *DocumentBuilder) AppendNestedBlockquote(depth int, inlineGFM string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	if depth < 2 || depth > maxConstructionBlockquoteDepth {
		return fmt.Errorf("%w: nested blockquote depth must be between 2 and %d", ErrInvalidConstruction, maxConstructionBlockquoteDepth)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:      constructionBlockquote,
		depth:     depth,
		inlineGFM: inlineGFM,
	})
}

// AppendBlockquoteBlocks appends one blockquote container from an existing child builder.
//
// depth must be between 1 and 64. content is treated as an immutable construction
// snapshot: its current reviewed body blocks are copied into the new container,
// while later changes to content do not affect this builder. M83-M86 accept
// every reviewed body-block construction family, including recursive blockquote
// children whose total structural depth remains at most 64. Front matter remains
// a document envelope and is never accepted as a blockquote child.
func (b *DocumentBuilder) AppendBlockquoteBlocks(depth int, content *DocumentBuilder) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	if content == nil {
		return fmt.Errorf("%w: nil blockquote child builder", ErrInvalidConstruction)
	}
	if content == b {
		return fmt.Errorf("%w: blockquote child builder cannot be the destination builder", ErrInvalidConstruction)
	}
	if err := validateConstructionBlockquoteDepth(depth); err != nil {
		return err
	}
	if content.frontMatter != nil {
		return fmt.Errorf("%w: blockquote child builder cannot contain front matter", ErrInvalidConstruction)
	}
	if len(content.deferredReferences) != 0 {
		return fmt.Errorf("%w: blockquote child builder cannot contain deferred reference definitions", ErrInvalidConstruction)
	}
	children := append([]constructionBlock(nil), content.blocks...)
	return b.appendConstructionBlock(constructionBlock{
		kind:     constructionBlockquoteBlocks,
		depth:    depth,
		children: children,
	})
}

// AppendThematicBreak appends one canonical top-level thematic break.
//
// M55 writes exactly three hyphens and retains the block only when the internal
// GFM parser observes one top-level thematic break over those exact bytes.
func (b *DocumentBuilder) AppendThematicBreak() error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{kind: constructionThematicBreak})
}

// AppendUnorderedList appends one flat top-level unordered list.
//
// Each item must be one non-empty physical line of valid UTF-8 inline GFM. M45
// writes the canonical '-' marker and accepts the block only when reparsing proves
// that every generated item belongs to the one requested list container.
func (b *DocumentBuilder) AppendUnorderedList(items ...string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:  constructionUnorderedList,
		items: constructionListItems(items),
	})
}

// AppendOrderedList appends one flat top-level ordered list.
//
// M46 writes canonical sequential decimal markers beginning at 1 with '.' as the
// delimiter. The generated items must reparse as one ordered list container.
func (b *DocumentBuilder) AppendOrderedList(items ...string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:  constructionOrderedList,
		items: constructionListItems(items),
	})
}

// AppendNestedUnorderedList appends one homogeneous nested unordered list.
//
// M57 accepts source-ordered ListItemInput values whose Depth describes the
// parent/child hierarchy. The writer uses canonical '-' markers and derives
// each nested indentation from the generated parent's exact content column.
func (b *DocumentBuilder) AppendNestedUnorderedList(items ...ListItemInput) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:  constructionNestedUnorderedList,
		items: constructionNestedListItems(items),
	})
}

// AppendNestedOrderedList appends one homogeneous nested ordered list.
//
// M58 uses the same structural depth contract as M57. Decimal numbering starts
// at 1 in every list container and indentation follows the actual generated
// parent marker width, including transitions such as '9.' to '10.'.
func (b *DocumentBuilder) AppendNestedOrderedList(items ...ListItemInput) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:  constructionNestedOrderedList,
		items: constructionNestedListItems(items),
	})
}

// AppendNestedUnorderedTaskList appends one homogeneous nested unordered GFM task list.
// M59 combines M57 structural depth with the exact M47 task-marker/state proof.
func (b *DocumentBuilder) AppendNestedUnorderedTaskList(items ...TaskListItemInput) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:  constructionNestedUnorderedList,
		items: constructionNestedTaskListItems(items),
	})
}

// AppendNestedOrderedTaskList appends one homogeneous nested ordered GFM task list.
// M60 combines M58 container-local numbering/indentation with M48 task proof.
func (b *DocumentBuilder) AppendNestedOrderedTaskList(items ...TaskListItemInput) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:  constructionNestedOrderedList,
		items: constructionNestedTaskListItems(items),
	})
}

// AppendUnorderedTaskList appends one flat top-level unordered GFM task list.
//
// M47 writes canonical '-' list markers and '[ ]'/'[x]' task markers. The block
// is retained only when reparsing proves both the requested list container and
// each requested semantic task marker/state.
func (b *DocumentBuilder) AppendUnorderedTaskList(items ...TaskListItem) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:  constructionUnorderedTaskList,
		items: constructionTaskListItems(items),
	})
}

// AppendOrderedTaskList appends one flat top-level ordered GFM task list.
//
// M48 combines canonical sequential '1.', '2.', ... list markers with the same
// semantic task proof used by unordered task lists.
func (b *DocumentBuilder) AppendOrderedTaskList(items ...TaskListItem) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:  constructionOrderedTaskList,
		items: constructionTaskListItems(items),
	})
}

// AppendFencedCode appends one top-level fenced code block.
//
// M49 introduced the canonical unindented backtick form; M53 extends content to
// LF-separated multiline text, and M103 permits an empty payload. The fence is at
// least three bytes and grows beyond every potentially closing backtick run in a
// non-empty body. info is an optional single-line raw GFM info string and must not
// contain backticks. Empty content produces adjacent opening/closing fence lines
// without inventing a payload line.
func (b *DocumentBuilder) AppendFencedCode(content, info string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:      constructionFencedCode,
		inlineGFM: content,
		info:      info,
	})
}

// AppendReferenceDefinition appends one top-level single-line link reference definition.
//
// M50 writes canonical angle-bracket destination syntax without a title. The
// generated definition is retained only when reparsing reproduces the exact
// label, destination, and source mapping.
func (b *DocumentBuilder) AppendReferenceDefinition(label, destination string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	if err := b.rejectDeferredReferenceCollision(label); err != nil {
		return err
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:        constructionReferenceDefinition,
		label:       label,
		destination: destination,
	})
}

// AppendReferenceDefinitionWithTitle appends one top-level single-line link
// reference definition with a canonical double-quoted title.
//
// M61 keeps the existing M50 angle-bracket destination form and accepts the
// block only when reparsing reproduces the exact label, destination, title, and
// source mapping. title must not require escaping in the canonical form.
func (b *DocumentBuilder) AppendReferenceDefinitionWithTitle(label, destination, title string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	if err := b.rejectDeferredReferenceCollision(label); err != nil {
		return err
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:        constructionReferenceDefinition,
		label:       label,
		destination: destination,
		title:       title,
		hasTitle:    true,
	})
}

// AppendTable appends one top-level unaligned GFM table.
//
// M96 requires at least one header column; body rows are optional. Every body row
// must have the same width as header. Cell strings are caller-provided single-line
// GFM source; empty cells are allowed. The builder writes canonical outer pipes
// and '---' delimiter cells and retains the table only after exact table-container
// proof plus body-row proof for every row that is present.
func (b *DocumentBuilder) AppendTable(header []string, rows ...[]string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:  constructionTableBlock,
		table: newConstructionTable(header, defaultConstructionTableAlignments(len(header)), rows),
	})
}

// AppendTableWithAlignments appends one top-level GFM table with explicit semantic column alignments.
//
// M96 preserves the canonical outer-pipe/padding policy while writing delimiter
// cells as '---', ':---', '---:', or ':---:' for default, left, right, or center
// alignment. alignments must have exactly one entry per header column; body rows
// remain optional.
func (b *DocumentBuilder) AppendTableWithAlignments(header []string, alignments []TableAlignment, rows ...[]string) error {
	if b == nil {
		return fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	return b.appendConstructionBlock(constructionBlock{
		kind:  constructionTableBlock,
		table: newConstructionTable(header, alignments, rows),
	})
}

// Markdown returns newly generated canonical GFM source.
//
// The returned bytes are caller-owned. The zero-value builder produces an empty
// document. A nil builder reports ErrInvalidConstruction.
func (b *DocumentBuilder) Markdown() ([]byte, error) {
	if b == nil {
		return nil, fmt.Errorf("%w: nil document builder", ErrInvalidConstruction)
	}
	source, expected := writeConstructionDocument(b.frontMatter, b.constructionDocumentBlocks())
	if b.frontMatter != nil {
		if err := validateConstructionFrontMatter(source, *b.frontMatter); err != nil {
			return nil, err
		}
	}
	if err := validateConstructionDocument(source, expected); err != nil {
		return nil, err
	}
	return source, nil
}
