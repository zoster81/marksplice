package splice

import (
	"fmt"
	"slices"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

// validateNodeSurvivorsAfterRemoval proves that deleting one owned block span
// removes only nodes whose semantic ranges overlap that span. Every other
// observed node must survive one-to-one with transformed ranges and unchanged
// semantic shape, and the candidate must not introduce new observed nodes.
func (d *Document) validateNodeSurvivorsAfterRemoval(candidate []byte, removed Range) error {
	if d == nil || !removed.Valid(len(d.source)) || removed.Start == removed.End {
		return ErrInvalidReplacement
	}
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return fmt.Errorf("%w: parse block-removal candidate: %v", ErrInvalidReplacement, err)
	}

	survivors := make([]Node, 0, len(d.nodes))
	for _, original := range d.nodes {
		if !rangesOverlap(original.Range, removed) {
			survivors = append(survivors, original)
		}
	}
	if len(candidateDocument.nodes) != len(survivors) || !linkUsagesSurviveRemoval(d.linkUsages, candidateDocument.linkUsages, removed) {
		return ErrInvalidReplacement
	}

	matched := make([]bool, len(candidateDocument.nodes))
	for _, original := range survivors {
		candidateIndex := matchingRemovalSurvivor(d, candidateDocument, matched, original, removed)
		if candidateIndex < 0 {
			return ErrInvalidReplacement
		}
		matched[candidateIndex] = true
	}
	return nil
}

func linkUsagesSurviveRemoval(original, candidate []parser.LinkUsage, removed Range) bool {
	patches := []patchTransform{{Range: removed}}
	candidateIndex := 0
	for _, usage := range original {
		if usage.Anchor >= removed.Start && usage.Anchor < removed.End {
			continue
		}
		expectedAnchor, ok := anchorAfterPatches(usage.Anchor, patches)
		if !ok || candidateIndex >= len(candidate) {
			return false
		}
		expected := usage
		expected.Anchor = expectedAnchor
		if candidate[candidateIndex] != expected {
			return false
		}
		candidateIndex++
	}
	return candidateIndex == len(candidate)
}

func matchingRemovalSurvivor(originalDocument, candidateDocument *Document, matched []bool, original Node, removed Range) int {
	expectedRange, ok := rangeAfterPatch(original.Range, removed, 0)
	if !ok {
		return -1
	}
	expectedContent, ok := rangeAfterPatch(original.ContentRange, removed, 0)
	if !ok {
		return -1
	}
	for index, candidate := range candidateDocument.nodes {
		if matched[index] || candidate.Kind != original.Kind || candidate.Range != expectedRange || candidate.ContentRange != expectedContent {
			continue
		}
		if sameRemovalSurvivorSemantics(original, candidate) &&
			sameRemovalSurvivorAnchors(original, candidate, removed) &&
			sameRemovalBlockquoteSource(originalDocument, candidateDocument, original, candidate, removed) {
			return index
		}
	}
	return -1
}

type removalSurvivorSignature struct {
	level                     int
	headingStyle              HeadingStyle
	checked                   bool
	listOrdered               bool
	listMarker                byte
	listHasParent             bool
	listHasChildren           bool
	listDirectChildCount      int
	tableHeader               bool
	tableColumn               int
	tableColumnCount          int
	tableBodyRowCount         int
	tablePromotedRowCount     int
	tableOwnedHeaderCellCount int
	tableRowCellCount         int
	tableHeaderCellCount      int
	destination               string
	label                     string
	title                     string
	hasTitle                  bool
	value                     string
	autoLinkEmail             bool
	key                       string
	frontMatterFormat         FrontMatterFormat
	frontMatterStyle          source.FrontMatterValueStyle
	htmlAttribute             string
	htmlQuote                 byte
	topLevel                  bool
	editable                  bool
}

func sameRemovalSurvivorSemantics(original, candidate Node) bool {
	return removalSurvivorSemanticSignature(candidate) == removalSurvivorSemanticSignature(original) &&
		slices.Equal(candidate.TableAlignments, original.TableAlignments)
}

func removalSurvivorSemanticSignature(node Node) removalSurvivorSignature {
	return removalSurvivorSignature{
		level:                     node.Level,
		headingStyle:              node.HeadingStyle,
		checked:                   node.Checked,
		listOrdered:               node.ListOrdered,
		listMarker:                node.ListMarker,
		listHasParent:             node.ListHasParent,
		listHasChildren:           node.ListHasChildren,
		listDirectChildCount:      node.ListDirectChildCount,
		tableHeader:               node.TableHeader,
		tableColumn:               node.TableColumn,
		tableColumnCount:          node.TableColumnCount,
		tableBodyRowCount:         node.TableBodyRowCount,
		tablePromotedRowCount:     node.TablePromotedRowCount,
		tableOwnedHeaderCellCount: node.TableOwnedHeaderCellCount,
		tableRowCellCount:         node.TableRowCellCount,
		tableHeaderCellCount:      node.TableHeaderCellCount,
		destination:               node.Destination,
		label:                     node.Label,
		title:                     node.Title,
		hasTitle:                  node.HasTitle,
		value:                     node.Value,
		autoLinkEmail:             node.AutoLinkEmail,
		key:                       node.Key,
		frontMatterFormat:         node.FrontMatterFormat,
		frontMatterStyle:          node.FrontMatterStyle,
		htmlAttribute:             node.HTMLAttribute,
		htmlQuote:                 node.HTMLQuote,
		topLevel:                  node.TopLevel,
		editable:                  node.Editable,
	}
}

func sameRemovalBlockquoteSource(originalDocument, candidateDocument *Document, original, candidate Node, removed Range) bool {
	if original.Kind != KindBlockquote {
		return true
	}
	originalSource, originalOK := originalDocument.blockquoteSource(original)
	candidateSource, candidateOK := candidateDocument.blockquoteSource(candidate)
	if !originalOK || !candidateOK {
		return false
	}
	expectedSource, ok := rangeAfterPatch(originalSource.LineRange, removed, 0)
	if !ok || candidateSource.LineRange != expectedSource {
		return false
	}
	expectedMarker, ok := rangeAfterPatch(originalSource.MarkerRange, removed, 0)
	if !ok || candidateSource.MarkerRange != expectedMarker || len(candidateSource.ContentRanges) != len(originalSource.ContentRanges) {
		return false
	}
	for index, content := range originalSource.ContentRanges {
		expected, ok := rangeAfterPatch(content, removed, 0)
		if !ok || candidateSource.ContentRanges[index] != expected {
			return false
		}
	}
	return true
}

func sameRemovalSurvivorAnchors(original, candidate Node, removed Range) bool {
	patches := []patchTransform{{Range: removed}}
	return sameRemovalListAnchors(original, candidate, patches) &&
		sameRemovalTableAnchors(original, candidate, patches) &&
		sameRemovalInlineAnchor(original, candidate, patches)
}

func sameRemovalListAnchors(original, candidate Node, patches []patchTransform) bool {
	if original.Kind != KindListItem {
		return true
	}
	if !sameShiftedRemovalAnchor(original.ListContainerAnchor, candidate.ListContainerAnchor, patches) {
		return false
	}
	return !original.ListHasParent || sameShiftedRemovalAnchor(original.ListParentAnchor, candidate.ListParentAnchor, patches)
}

func sameRemovalTableAnchors(original, candidate Node, patches []patchTransform) bool {
	if original.Kind != KindTable && original.Kind != KindTableRow && original.Kind != KindTableCell {
		return true
	}
	if !sameShiftedRemovalAnchor(original.TableAnchor, candidate.TableAnchor, patches) {
		return false
	}
	if original.Kind != KindTable && !sameShiftedRemovalAnchor(original.TableRowAnchor, candidate.TableRowAnchor, patches) {
		return false
	}
	return original.Kind != KindTable || original.TableBodyRowCount == 0 ||
		sameShiftedRemovalAnchor(original.TableLastBodyRowAnchor, candidate.TableLastBodyRowAnchor, patches)
}

func sameRemovalInlineAnchor(original, candidate Node, patches []patchTransform) bool {
	switch original.Kind {
	case KindInlineLink, KindImage, KindAutoLink, KindCodeSpan, KindEmphasis, KindStrong:
		return sameShiftedRemovalAnchor(original.Anchor, candidate.Anchor, patches)
	default:
		return true
	}
}

func sameShiftedRemovalAnchor(original, candidate int, patches []patchTransform) bool {
	expected, ok := anchorAfterPatches(original, patches)
	return ok && candidate == expected
}
