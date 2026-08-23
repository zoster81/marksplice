package marksplice

import (
	"fmt"
	"slices"

	"github.com/zoster81/marksplice/internal/splice"
)

type constructionExpectation struct {
	kind         splice.Kind
	level        int
	contentRange splice.Range
	sourceRange  splice.Range
	list         constructionListProof
	fence        constructionFenceProof
	reference    constructionReferenceProof
	table        constructionTableProof
	tableRow     constructionTableRowProof
	blockquote   constructionBlockquoteProof
}

type constructionBlockquoteProof struct {
	depth         int
	contentRanges []splice.Range
	innerSource   []byte
}

type constructionListProof struct {
	containerAnchor int
	markerStart     int
	ordered         bool
	marker          byte
	hasParent       bool
	parentAnchor    int
	directChildren  int
	subtreeEnd      int
	task            constructionTaskProof
}

type constructionTaskProof struct {
	present     bool
	markerRange splice.Range
	stateRange  splice.Range
	checked     bool
}

type constructionFenceProof struct {
	infoRange splice.Range
	length    int
}

type constructionReferenceProof struct {
	label       string
	destination string
	title       string
	hasTitle    bool
	titleRange  splice.Range
}

type constructionTableProof struct {
	columnCount  int
	bodyRowCount int
	alignments   []splice.TableAlignment
}

type constructionTableRowProof struct {
	tableAnchor  int
	columnCount  int
	alignments   []splice.TableAlignment
	cellRanges   []splice.Range
	cellContents []splice.Range
}

func validateConstructionDocument(source []byte, expected []constructionExpectation) error {
	if err := validateConstructionBlockquoteProofs(source, expected); err != nil {
		return err
	}
	nodeExpected := constructionNodeExpectations(expected)
	document, err := splice.Parse(source)
	if err != nil {
		return fmt.Errorf("%w: generated GFM parse: %v", ErrInvalidConstruction, err)
	}

	nodes := document.Nodes()
	tasks := make(map[int]splice.Node)
	matched := 0
	for _, node := range nodes {
		if node.Kind == splice.KindTask && node.Editable {
			if _, exists := tasks[node.Range.Start]; exists {
				return fmt.Errorf("%w: generated duplicate task marker", ErrInvalidConstruction)
			}
			tasks[node.Range.Start] = node
		}
		if isConstructionOnlyBlockquoteObservation(node, expected) || !isConstructionProofNode(node) {
			continue
		}
		if matched >= len(nodeExpected) {
			return fmt.Errorf("%w: generated unexpected top-level block", ErrInvalidConstruction)
		}
		if err := validateConstructionExpectation(node, nodeExpected[matched]); err != nil {
			return err
		}
		matched++
	}
	if matched != len(nodeExpected) {
		return fmt.Errorf("%w: generated block count changed", ErrInvalidConstruction)
	}
	return validateConstructionTaskExpectations(tasks, expected)
}

func validateConstructionBlockquoteProofs(source []byte, expected []constructionExpectation) error {
	for _, want := range expected {
		if want.kind != splice.KindBlockquote {
			continue
		}
		if len(want.blockquote.innerSource) != 0 {
			if err := splice.ValidateConstructionNestedBlockquoteBlocks(source, want.sourceRange, want.blockquote.innerSource, want.blockquote.depth); err != nil {
				return fmt.Errorf("%w: generated blockquote child sequence changed: %v", ErrInvalidConstruction, err)
			}
			continue
		}
		if want.blockquote.depth == 1 && len(want.blockquote.contentRanges) == 1 {
			continue
		}
		if err := splice.ValidateConstructionNestedBlockquoteParagraph(source, want.sourceRange, want.blockquote.contentRanges, want.blockquote.depth); err != nil {
			return fmt.Errorf("%w: generated blockquote paragraph changed: %v", ErrInvalidConstruction, err)
		}
	}
	return nil
}

func constructionNodeExpectations(expected []constructionExpectation) []constructionExpectation {
	return append([]constructionExpectation(nil), expected...)
}

func validateConstructionTaskExpectations(tasks map[int]splice.Node, expected []constructionExpectation) error {
	for _, want := range expected {
		taskProof := want.list.task
		if !taskProof.present {
			continue
		}
		task, ok := tasks[taskProof.markerRange.Start]
		if !ok || taskProof.markerRange.Start != want.contentRange.Start || task.Kind != splice.KindTask || !task.Editable ||
			task.Range != taskProof.markerRange || task.ContentRange != taskProof.stateRange || task.Checked != taskProof.checked {
			return fmt.Errorf("%w: generated task marker changed", ErrInvalidConstruction)
		}
	}
	return nil
}

func isConstructionOnlyBlockquoteObservation(node splice.Node, expected []constructionExpectation) bool {
	if node.Range.Start >= node.Range.End {
		return false
	}
	for _, want := range expected {
		if want.kind != splice.KindBlockquote || len(want.blockquote.innerSource) == 0 {
			continue
		}
		if node.Kind == splice.KindBlockquote && node.TopLevel && node.Editable && node.Range.Start == want.sourceRange.Start {
			return false
		}
		if node.Range.Start >= want.sourceRange.Start && node.Range.End <= want.sourceRange.End {
			return true
		}
	}
	return false
}

func isConstructionProofNode(node splice.Node) bool {
	switch node.Kind {
	case splice.KindThematicBreak, splice.KindBlockquote:
		return node.TopLevel
	}
	if !node.Editable {
		return false
	}
	switch node.Kind {
	case splice.KindParagraph, splice.KindHeading:
		return node.TopLevel
	case splice.KindListItem, splice.KindFencedCode, splice.KindReferenceDefinition, splice.KindTable, splice.KindTableRow:
		return true
	default:
		return false
	}
}

func validateConstructionExpectation(node splice.Node, want constructionExpectation) error {
	if node.Kind != want.kind {
		return fmt.Errorf("%w: generated block kind changed", ErrInvalidConstruction)
	}
	switch node.Kind {
	case splice.KindHeading:
		return validateConstructionHeadingExpectation(node, want)
	case splice.KindParagraph:
		return validateConstructionParagraphExpectation(node, want)
	case splice.KindThematicBreak:
		return validateConstructionThematicBreakExpectation(node, want)
	case splice.KindBlockquote:
		return validateConstructionBlockquoteExpectation(node, want)
	case splice.KindListItem:
		return validateConstructionListItemExpectation(node, want)
	case splice.KindFencedCode:
		return validateConstructionFencedCodeExpectation(node, want)
	case splice.KindReferenceDefinition:
		return validateConstructionReferenceDefinitionExpectation(node, want)
	case splice.KindTable:
		return validateConstructionTableExpectation(node, want)
	case splice.KindTableRow:
		return validateConstructionTableRowExpectation(node, want)
	default:
		return fmt.Errorf("%w: generated unsupported proof block", ErrInvalidConstruction)
	}
}

func validateConstructionHeadingExpectation(node splice.Node, want constructionExpectation) error {
	if node.Level != want.level || node.HeadingStyle != splice.HeadingStyleATX || node.ContentRange != want.contentRange {
		return fmt.Errorf("%w: generated heading mapping changed", ErrInvalidConstruction)
	}
	return nil
}

func validateConstructionParagraphExpectation(node splice.Node, want constructionExpectation) error {
	if node.Range != want.contentRange {
		return fmt.Errorf("%w: generated paragraph mapping changed", ErrInvalidConstruction)
	}
	return nil
}

func validateConstructionThematicBreakExpectation(node splice.Node, want constructionExpectation) error {
	if node.Range != want.contentRange || !node.TopLevel || !node.Editable || node.ThematicBreakSource.Range != want.contentRange {
		return fmt.Errorf("%w: generated thematic-break mapping changed", ErrInvalidConstruction)
	}
	return nil
}

func validateConstructionBlockquoteExpectation(node splice.Node, want constructionExpectation) error {
	mapping := node.BlockquoteSource
	expectedOwned := splice.Range{Start: want.sourceRange.Start, End: want.sourceRange.End + 1}
	if !node.TopLevel || !node.Editable || node.Range.Start != want.sourceRange.Start ||
		mapping.LineRange != expectedOwned || mapping.MarkerRange != (splice.Range{Start: want.sourceRange.Start, End: want.sourceRange.Start + 1}) {
		return fmt.Errorf("%w: generated blockquote mapping changed", ErrInvalidConstruction)
	}
	if len(want.blockquote.innerSource) == 0 && want.blockquote.depth == 1 && len(want.blockquote.contentRanges) == 1 {
		if want.blockquote.contentRanges[0] != want.contentRange || node.Range != want.sourceRange || node.ContentRange != want.contentRange ||
			mapping.Range != want.sourceRange || mapping.ContentRange != want.contentRange {
			return fmt.Errorf("%w: generated simple blockquote mapping changed", ErrInvalidConstruction)
		}
	}
	return nil
}

func validateConstructionListItemExpectation(node splice.Node, want constructionExpectation) error {
	expectedRange := splice.Range{Start: want.list.markerStart, End: want.contentRange.End}
	if node.Range != expectedRange || node.ContentRange != want.contentRange || node.ListItemSource.LineRange != want.sourceRange ||
		node.ListOrdered != want.list.ordered || node.ListMarker != want.list.marker || node.ListContainerAnchor != want.list.containerAnchor ||
		node.ListHasParent != want.list.hasParent || node.ListHasChildren != (want.list.directChildren != 0) ||
		node.ListDirectChildCount != want.list.directChildren || node.ListChildCount != want.list.directChildren ||
		!node.ListSubtreeComplete || node.ListSubtreeEnd != want.list.subtreeEnd {
		return fmt.Errorf("%w: generated list-item mapping changed", ErrInvalidConstruction)
	}
	if want.list.hasParent && node.ListParentAnchor != want.list.parentAnchor {
		return fmt.Errorf("%w: generated list-item parent changed", ErrInvalidConstruction)
	}
	return nil
}

func validateConstructionFencedCodeExpectation(node splice.Node, want constructionExpectation) error {
	mapping := node.FencedCodeSource
	if node.Range != want.contentRange || node.ContentRange != want.contentRange ||
		mapping.Range != want.sourceRange || mapping.ContentRange != want.contentRange || mapping.InfoRange != want.fence.infoRange ||
		mapping.FenceChar != '`' || mapping.FenceLength != want.fence.length || mapping.ClosingFenceLength != want.fence.length ||
		mapping.OpeningIndent != 0 || mapping.ClosingIndent != 0 {
		return fmt.Errorf("%w: generated fenced-code mapping changed", ErrInvalidConstruction)
	}
	return nil
}

func validateConstructionReferenceDefinitionExpectation(node splice.Node, want constructionExpectation) error {
	mapping := node.ReferenceDefinitionSource
	if node.Range != want.sourceRange || node.ContentRange != want.contentRange ||
		node.Label != want.reference.label || node.Destination != want.reference.destination ||
		node.Title != want.reference.title || node.HasTitle != want.reference.hasTitle ||
		mapping.Range != want.sourceRange || mapping.DestinationRange != want.contentRange || mapping.TitleRange != want.reference.titleRange ||
		!mapping.AngleDestination || mapping.HasTitle != want.reference.hasTitle {
		return fmt.Errorf("%w: generated reference-definition mapping changed", ErrInvalidConstruction)
	}
	return nil
}

func validateConstructionTableExpectation(node splice.Node, want constructionExpectation) error {
	mapping := node.TableSource
	if !node.Editable || node.Range != want.sourceRange || node.ContentRange != want.sourceRange ||
		node.TableAnchor != want.sourceRange.Start || node.TableColumnCount != want.table.columnCount ||
		node.TableBodyRowCount != want.table.bodyRowCount || !slices.Equal(node.TableAlignments, want.table.alignments) ||
		mapping.Range != want.sourceRange || len(mapping.Header.Cells) != want.table.columnCount ||
		len(mapping.Delimiter.Cells) != want.table.columnCount || len(mapping.DelimiterAlignments) != want.table.columnCount {
		return fmt.Errorf("%w: generated table mapping changed", ErrInvalidConstruction)
	}
	return nil
}

func validateConstructionTableRowExpectation(node splice.Node, want constructionExpectation) error {
	mapping := node.TableRowSource
	if node.Range != want.sourceRange || node.ContentRange != want.contentRange ||
		node.TableRowAnchor != want.contentRange.Start || node.TableAnchor != want.tableRow.tableAnchor || node.TableColumnCount != want.tableRow.columnCount ||
		!slices.Equal(node.TableAlignments, want.tableRow.alignments) ||
		mapping.Range != want.contentRange || mapping.LineRange != want.sourceRange || len(mapping.Cells) != len(want.tableRow.cellRanges) {
		return fmt.Errorf("%w: generated table-row mapping changed", ErrInvalidConstruction)
	}
	for index, cell := range mapping.Cells {
		if cell.Column != index || cell.Range != want.tableRow.cellRanges[index] || cell.ContentRange != want.tableRow.cellContents[index] {
			return fmt.Errorf("%w: generated table-cell mapping changed", ErrInvalidConstruction)
		}
	}
	return nil
}
