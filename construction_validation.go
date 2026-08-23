package marksplice

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type constructionBlockKind uint8

const (
	constructionParagraph constructionBlockKind = iota + 1
	constructionHeading
	constructionUnorderedList
	constructionOrderedList
	constructionUnorderedTaskList
	constructionOrderedTaskList
	constructionFencedCode
	constructionReferenceDefinition
	constructionTableBlock
	constructionThematicBreak
	constructionBlockquote
	constructionBlockquoteBlocks
	constructionNestedUnorderedList
	constructionNestedOrderedList

	maxConstructionBlockquoteDepth = 64
)

type constructionListItem struct {
	inlineGFM string
	depth     int
	task      bool
	checked   bool
}

type constructionTable struct {
	header     []string
	alignments []TableAlignment
	rows       [][]string
}

type constructionBlock struct {
	kind        constructionBlockKind
	level       int
	depth       int
	inlineGFM   string
	info        string
	label       string
	destination string
	title       string
	hasTitle    bool
	items       []constructionListItem
	table       constructionTable
	children    []constructionBlock
}

func (b *DocumentBuilder) appendConstructionBlock(block constructionBlock) error {
	if err := validateConstructionBlock(block); err != nil {
		return err
	}
	source, expected := writeConstructionBlocks([]constructionBlock{block})
	if err := validateConstructionDocument(source, expected); err != nil {
		return err
	}
	b.blocks = append(b.blocks, block)
	return nil
}

func validateConstructionBlock(block constructionBlock) error {
	switch block.kind {
	case constructionHeading:
		if block.level < 1 || block.level > 6 {
			return fmt.Errorf("%w: heading level must be between 1 and 6", ErrInvalidConstruction)
		}
		return validateConstructionInlineGFM(block.inlineGFM)
	case constructionParagraph:
		return validateConstructionParagraphGFM(block.inlineGFM)
	case constructionThematicBreak:
		return nil
	case constructionBlockquote:
		if err := validateConstructionBlockquoteDepth(block.depth); err != nil {
			return err
		}
		return validateConstructionBlockquoteGFM(block.inlineGFM)
	case constructionBlockquoteBlocks:
		if err := validateConstructionBlockquoteDepth(block.depth); err != nil {
			return err
		}
		return validateConstructionBlockquoteChildren(block.children, block.depth)
	case constructionUnorderedList, constructionOrderedList, constructionUnorderedTaskList, constructionOrderedTaskList,
		constructionNestedUnorderedList, constructionNestedOrderedList:
		return validateConstructionListItems(block.items)
	case constructionFencedCode:
		if err := validateConstructionFenceContent(block.inlineGFM); err != nil {
			return err
		}
		return validateConstructionFenceInfo(block.info)
	case constructionReferenceDefinition:
		return validateConstructionReferenceDefinition(block.label, block.destination, block.title, block.hasTitle)
	case constructionTableBlock:
		return validateConstructionTable(block.table)
	default:
		return fmt.Errorf("%w: unsupported construction block", ErrInvalidConstruction)
	}
}

func validateConstructionBlockquoteDepth(depth int) error {
	if depth < 1 || depth > maxConstructionBlockquoteDepth {
		return fmt.Errorf("%w: blockquote depth must be between 1 and %d", ErrInvalidConstruction, maxConstructionBlockquoteDepth)
	}
	return nil
}

type constructionBlockquoteValidationFrame struct {
	children    []constructionBlock
	parentDepth int
}

func validateConstructionBlockquoteChildren(children []constructionBlock, outerDepth int) error {
	if len(children) == 0 {
		return fmt.Errorf("%w: blockquote child sequence is empty", ErrInvalidConstruction)
	}
	stack := []constructionBlockquoteValidationFrame{{children: children, parentDepth: outerDepth}}
	for len(stack) != 0 {
		frame := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for index, child := range frame.children {
			if !supportedConstructionBlockquoteChild(child.kind) {
				return fmt.Errorf("%w: unsupported blockquote child at index %d", ErrInvalidConstruction, index)
			}
			if err := validateConstructionBlockquoteChild(child); err != nil {
				return fmt.Errorf("%w: blockquote child at index %d: %v", ErrInvalidConstruction, index, err)
			}
			if child.kind != constructionBlockquote && child.kind != constructionBlockquoteBlocks {
				continue
			}
			totalDepth := frame.parentDepth + child.depth
			if totalDepth > maxConstructionBlockquoteDepth {
				return fmt.Errorf("%w: total blockquote depth exceeds %d", ErrInvalidConstruction, maxConstructionBlockquoteDepth)
			}
			if child.kind == constructionBlockquoteBlocks {
				stack = append(stack, constructionBlockquoteValidationFrame{children: child.children, parentDepth: totalDepth})
			}
		}
	}
	source, expected := writeConstructionBlocks(children)
	if err := validateConstructionDocument(source, expected); err != nil {
		return fmt.Errorf("%w: blockquote child sequence: %v", ErrInvalidConstruction, err)
	}
	return nil
}

func validateConstructionBlockquoteChild(child constructionBlock) error {
	switch child.kind {
	case constructionBlockquote:
		if err := validateConstructionBlockquoteDepth(child.depth); err != nil {
			return err
		}
		return validateConstructionBlockquoteGFM(child.inlineGFM)
	case constructionBlockquoteBlocks:
		if err := validateConstructionBlockquoteDepth(child.depth); err != nil {
			return err
		}
		if len(child.children) == 0 {
			return fmt.Errorf("%w: blockquote child sequence is empty", ErrInvalidConstruction)
		}
		return nil
	default:
		return validateConstructionBlock(child)
	}
}

func supportedConstructionBlockquoteChild(kind constructionBlockKind) bool {
	switch kind {
	case constructionParagraph, constructionHeading, constructionThematicBreak, constructionFencedCode,
		constructionUnorderedList, constructionOrderedList, constructionUnorderedTaskList, constructionOrderedTaskList,
		constructionNestedUnorderedList, constructionNestedOrderedList, constructionReferenceDefinition, constructionTableBlock,
		constructionBlockquote, constructionBlockquoteBlocks:
		return true
	default:
		return false
	}
}

func constructionListItems(items []string) []constructionListItem {
	result := make([]constructionListItem, len(items))
	for index, item := range items {
		result[index].inlineGFM = item
	}
	return result
}

func constructionTaskListItems(items []TaskListItem) []constructionListItem {
	result := make([]constructionListItem, len(items))
	for index, item := range items {
		result[index] = constructionListItem{inlineGFM: item.InlineGFM, task: true, checked: item.Checked}
	}
	return result
}

func constructionNestedListItems(items []ListItemInput) []constructionListItem {
	result := make([]constructionListItem, len(items))
	for index, item := range items {
		result[index] = constructionListItem{inlineGFM: item.InlineGFM, depth: item.Depth}
	}
	return result
}

func constructionNestedTaskListItems(items []TaskListItemInput) []constructionListItem {
	result := make([]constructionListItem, len(items))
	for index, item := range items {
		result[index] = constructionListItem{inlineGFM: item.InlineGFM, depth: item.Depth, task: true, checked: item.Checked}
	}
	return result
}

func validateConstructionListItems(items []constructionListItem) error {
	if len(items) == 0 {
		return fmt.Errorf("%w: list is empty", ErrInvalidConstruction)
	}
	previousDepth := 0
	for index, item := range items {
		if item.depth < 0 || index == 0 && item.depth != 0 || index != 0 && item.depth > previousDepth+1 {
			return fmt.Errorf("%w: invalid list-item depth at index %d", ErrInvalidConstruction, index)
		}
		if err := validateConstructionInlineGFM(item.inlineGFM); err != nil {
			return err
		}
		previousDepth = item.depth
	}
	return nil
}

func validateConstructionNonEmptyUTF8(value, description string) error {
	switch {
	case value == "":
		return fmt.Errorf("%w: %s is empty", ErrInvalidConstruction, description)
	case !utf8.ValidString(value):
		return fmt.Errorf("%w: %s is not valid UTF-8", ErrInvalidConstruction, description)
	default:
		return nil
	}
}

func validateConstructionParagraphGFM(content string) error {
	if err := validateConstructionNonEmptyUTF8(content, "paragraph GFM"); err != nil {
		return err
	}
	switch {
	case strings.IndexByte(content, 0) >= 0:
		return fmt.Errorf("%w: paragraph GFM contains NUL", ErrInvalidConstruction)
	case strings.IndexByte(content, '\r') >= 0:
		return fmt.Errorf("%w: paragraph construction accepts canonical LF line endings only", ErrInvalidConstruction)
	default:
		return nil
	}
}

func validateConstructionInlineGFM(inlineGFM string) error {
	if err := validateConstructionNonEmptyUTF8(inlineGFM, "inline GFM"); err != nil {
		return err
	}
	switch {
	case strings.ContainsAny(inlineGFM, "\r\n"):
		return fmt.Errorf("%w: inline GFM must stay on one physical line", ErrInvalidConstruction)
	case strings.IndexByte(inlineGFM, 0) >= 0:
		return fmt.Errorf("%w: inline GFM contains NUL", ErrInvalidConstruction)
	default:
		return nil
	}
}

func validateConstructionBlockquoteGFM(content string) error {
	if err := validateConstructionParagraphGFM(content); err != nil {
		return err
	}
	for _, line := range strings.Split(content, "\n") {
		if line == "" {
			return fmt.Errorf("%w: blockquote paragraph contains an empty physical line", ErrInvalidConstruction)
		}
	}
	return nil
}

func validateConstructionFenceContent(content string) error {
	if err := validateConstructionNonEmptyUTF8(content, "fenced-code content"); err != nil {
		return err
	}
	switch {
	case strings.IndexByte(content, 0) >= 0:
		return fmt.Errorf("%w: fenced-code content contains NUL", ErrInvalidConstruction)
	case strings.IndexByte(content, '\r') >= 0:
		return fmt.Errorf("%w: fenced-code construction accepts canonical LF line endings only", ErrInvalidConstruction)
	default:
		return nil
	}
}

func validateConstructionFenceInfo(info string) error {
	if info == "" {
		return nil
	}
	if err := validateConstructionInlineGFM(info); err != nil {
		return err
	}
	if strings.IndexByte(info, '`') >= 0 {
		return fmt.Errorf("%w: backtick fence info contains a backtick", ErrInvalidConstruction)
	}
	return nil
}

func validateConstructionReferenceDefinition(label, destination, title string, hasTitle bool) error {
	if err := validateConstructionInlineGFM(label); err != nil {
		return err
	}
	if err := validateConstructionInlineGFM(destination); err != nil {
		return err
	}
	if strings.ContainsAny(label, "[]") {
		return fmt.Errorf("%w: reference label contains a bracket", ErrInvalidConstruction)
	}
	if strings.ContainsAny(destination, "<>") {
		return fmt.Errorf("%w: angle destination contains an angle bracket", ErrInvalidConstruction)
	}
	if !hasTitle {
		return nil
	}
	if err := validateConstructionInlineGFM(title); err != nil {
		return err
	}
	if strings.ContainsAny(title, "\"\\") {
		return fmt.Errorf("%w: reference title requires escaping", ErrInvalidConstruction)
	}
	return nil
}

func newConstructionTable(header []string, alignments []TableAlignment, rows [][]string) constructionTable {
	result := constructionTable{
		header:     append([]string(nil), header...),
		alignments: append([]TableAlignment(nil), alignments...),
		rows:       make([][]string, len(rows)),
	}
	for index, row := range rows {
		result.rows[index] = append([]string(nil), row...)
	}
	return result
}

func defaultConstructionTableAlignments(count int) []TableAlignment {
	if count <= 0 {
		return nil
	}
	return make([]TableAlignment, count)
}

func validateConstructionTable(table constructionTable) error {
	if len(table.header) == 0 || len(table.rows) == 0 {
		return fmt.Errorf("%w: table requires header columns and a body row", ErrInvalidConstruction)
	}
	if len(table.alignments) != len(table.header) {
		return fmt.Errorf("%w: table alignment width changed", ErrInvalidConstruction)
	}
	for _, alignment := range table.alignments {
		if !validConstructionTableAlignment(alignment) {
			return fmt.Errorf("%w: unsupported table alignment", ErrInvalidConstruction)
		}
	}
	for _, cell := range table.header {
		if err := validateConstructionTableCell(cell); err != nil {
			return err
		}
	}
	for _, row := range table.rows {
		if len(row) != len(table.header) {
			return fmt.Errorf("%w: table body row width changed", ErrInvalidConstruction)
		}
		for _, cell := range row {
			if err := validateConstructionTableCell(cell); err != nil {
				return err
			}
		}
	}
	return nil
}

func validConstructionTableAlignment(alignment TableAlignment) bool {
	return alignment >= TableAlignmentDefault && alignment <= TableAlignmentCenter
}

func validateConstructionTableCell(cell string) error {
	if cell == "" {
		return nil
	}
	if err := validateConstructionInlineGFM(cell); err != nil {
		return err
	}
	if strings.Trim(cell, " \t") != cell {
		return fmt.Errorf("%w: table cell has outer horizontal space", ErrInvalidConstruction)
	}
	return nil
}
