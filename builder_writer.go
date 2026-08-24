package marksplice

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/zoster81/marksplice/internal/splice"
)

func writeConstructionBlocks(blocks []constructionBlock) ([]byte, []constructionExpectation) {
	var output bytes.Buffer
	expected := appendConstructionBlocks(&output, blocks)
	return output.Bytes(), expected
}

func writeConstructionDocument(frontMatter *constructionFrontMatter, blocks []constructionBlock) ([]byte, []constructionExpectation) {
	var output bytes.Buffer
	if frontMatter != nil {
		writeConstructionFrontMatterTo(&output, *frontMatter)
		if len(blocks) != 0 {
			output.WriteByte('\n')
		}
	}
	expected := appendConstructionBlocks(&output, blocks)
	return output.Bytes(), expected
}

func appendConstructionBlocks(output *bytes.Buffer, blocks []constructionBlock) []constructionExpectation {
	expected := make([]constructionExpectation, 0, len(blocks))
	for index, block := range blocks {
		if index != 0 {
			output.WriteByte('\n')
		}
		expected = append(expected, writeConstructionBlock(output, block)...)
	}
	return expected
}

func writeConstructionBlock(output *bytes.Buffer, block constructionBlock) []constructionExpectation {
	switch block.kind {
	case constructionHeading:
		output.WriteString(strings.Repeat("#", block.level))
		output.WriteByte(' ')
		start := output.Len()
		output.WriteString(block.inlineGFM)
		expectation := constructionExpectation{
			kind:         splice.KindHeading,
			level:        block.level,
			contentRange: splice.Range{Start: start, End: output.Len()},
		}
		output.WriteByte('\n')
		return []constructionExpectation{expectation}
	case constructionParagraph:
		start := output.Len()
		output.WriteString(block.inlineGFM)
		expectation := constructionExpectation{
			kind:         splice.KindParagraph,
			contentRange: splice.Range{Start: start, End: output.Len()},
		}
		output.WriteByte('\n')
		return []constructionExpectation{expectation}
	case constructionThematicBreak:
		start := output.Len()
		output.WriteString("---")
		expectation := constructionExpectation{
			kind:         splice.KindThematicBreak,
			contentRange: splice.Range{Start: start, End: output.Len()},
		}
		output.WriteByte('\n')
		return []constructionExpectation{expectation}
	case constructionBlockquote:
		return []constructionExpectation{writeConstructionBlockquote(output, block.depth, block.inlineGFM)}
	case constructionBlockquoteBlocks:
		return []constructionExpectation{writeConstructionBlockquoteBlocks(output, block.depth, block.children)}
	case constructionAlert:
		return []constructionExpectation{writeConstructionAlert(output, block.alertKind, block.inlineGFM)}
	case constructionAlertBlocks:
		return []constructionExpectation{writeConstructionAlertBlocks(output, block.alertKind, block.children)}
	case constructionUnorderedList, constructionOrderedList, constructionUnorderedTaskList, constructionOrderedTaskList,
		constructionNestedUnorderedList, constructionNestedOrderedList:
		ordered := block.kind == constructionOrderedList || block.kind == constructionOrderedTaskList || block.kind == constructionNestedOrderedList
		return writeConstructionList(output, block.items, ordered)
	case constructionFencedCode:
		return []constructionExpectation{writeConstructionFencedCode(output, block.inlineGFM, block.info)}
	case constructionReferenceDefinition:
		return []constructionExpectation{writeConstructionReferenceDefinition(output, block.label, block.destination, block.title, block.hasTitle)}
	case constructionTableBlock:
		return writeConstructionTable(output, block.table)
	default:
		return nil
	}
}

func writeConstructionBlockquoteBlocks(output *bytes.Buffer, depth int, blocks []constructionBlock) constructionExpectation {
	innerSource, _ := writeConstructionBlocks(blocks)
	return writeConstructionBlockquoteInnerSource(output, depth, innerSource)
}

func writeConstructionAlert(output *bytes.Buffer, kind AlertKind, content string) constructionExpectation {
	marker, _ := alertMarker(kind)
	expectation := writeConstructionBlockquote(output, 1, marker+"\n"+content)
	expectation.alertKind = kind
	expectation.alertMarkerRange = expectation.blockquote.contentRanges[0]
	return expectation
}

func writeConstructionAlertBlocks(output *bytes.Buffer, kind AlertKind, blocks []constructionBlock) constructionExpectation {
	bodySource, _ := writeConstructionBlocks(blocks)
	marker, _ := alertMarker(kind)
	innerSource := make([]byte, 0, len(marker)+1+len(bodySource))
	innerSource = append(innerSource, marker...)
	innerSource = append(innerSource, '\n')
	innerSource = append(innerSource, bodySource...)
	expectation := writeConstructionBlockquoteInnerSource(output, 1, innerSource)
	expectation.alertKind = kind
	expectation.alertMarkerRange = splice.Range{Start: expectation.sourceRange.Start + 2, End: expectation.sourceRange.Start + 2 + len(marker)}
	return expectation
}

func writeConstructionBlockquoteInnerSource(output *bytes.Buffer, depth int, innerSource []byte) constructionExpectation {
	start := output.Len()
	prefix := strings.Repeat("> ", depth)
	lineStart := 0
	for lineStart < len(innerSource) {
		lineEnd := bytes.IndexByte(innerSource[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(innerSource)
		} else {
			lineEnd += lineStart
		}
		output.WriteString(prefix)
		output.Write(innerSource[lineStart:lineEnd])
		if lineEnd == len(innerSource) {
			break
		}
		output.WriteByte('\n')
		lineStart = lineEnd + 1
	}
	end := output.Len() - 1
	return constructionExpectation{
		kind:        splice.KindBlockquote,
		sourceRange: splice.Range{Start: start, End: end},
		blockquote: constructionBlockquoteProof{
			depth:       depth,
			innerSource: append([]byte(nil), innerSource...),
		},
	}
}

func writeConstructionBlockquote(output *bytes.Buffer, depth int, content string) constructionExpectation {
	start := output.Len()
	lines := strings.Split(content, "\n")
	contentRanges := make([]splice.Range, len(lines))
	prefix := strings.Repeat("> ", depth)
	for index, line := range lines {
		if index != 0 {
			output.WriteByte('\n')
		}
		output.WriteString(prefix)
		contentStart := output.Len()
		output.WriteString(line)
		contentRanges[index] = splice.Range{Start: contentStart, End: output.Len()}
	}
	end := output.Len()
	expectation := constructionExpectation{
		kind:         splice.KindBlockquote,
		contentRange: contentRanges[0],
		sourceRange:  splice.Range{Start: start, End: end},
		blockquote: constructionBlockquoteProof{
			depth:         depth,
			contentRanges: contentRanges,
		},
	}
	output.WriteByte('\n')
	return expectation
}

type constructionListFrame struct {
	containerAnchor int
	indent          int
	nextOrdinal     int
}

func writeConstructionList(output *bytes.Buffer, items []constructionListItem, ordered bool) []constructionExpectation {
	expected := make([]constructionExpectation, 0, len(items))
	frames := make([]constructionListFrame, 0, 4)
	openItems := make([]int, 0, 4)
	contentIndents := make([]int, 0, 4)
	marker := byte('-')
	if ordered {
		marker = '.'
	}

	previousLineEnd := output.Len()
	for _, item := range items {
		for len(openItems) > item.depth {
			index := openItems[len(openItems)-1]
			expected[index].list.subtreeEnd = previousLineEnd
			openItems = openItems[:len(openItems)-1]
			contentIndents = contentIndents[:len(contentIndents)-1]
		}
		if len(frames) > item.depth+1 {
			frames = frames[:item.depth+1]
		}

		lineStart := output.Len()
		if len(frames) == item.depth {
			indent := 0
			if item.depth != 0 {
				indent = contentIndents[item.depth-1]
			}
			frames = append(frames, constructionListFrame{containerAnchor: lineStart, indent: indent, nextOrdinal: 1})
		}
		frame := &frames[item.depth]
		output.WriteString(strings.Repeat(" ", frame.indent))
		markerStart := output.Len()
		if ordered {
			output.WriteString(strconv.Itoa(frame.nextOrdinal))
			output.WriteString(". ")
			frame.nextOrdinal++
		} else {
			output.WriteString("- ")
		}
		contentStart := output.Len()
		expectation := constructionExpectation{
			kind: splice.KindListItem,
			list: constructionListProof{
				containerAnchor: frame.containerAnchor,
				markerStart:     markerStart,
				ordered:         ordered,
				marker:          marker,
			},
		}
		if item.depth != 0 {
			parentIndex := openItems[item.depth-1]
			expectation.list.hasParent = true
			expectation.list.parentAnchor = expected[parentIndex].sourceRange.Start
			expected[parentIndex].list.directChildren++
		}
		if item.task {
			expectation.list.task.present = true
			expectation.list.task.checked = item.checked
			expectation.list.task.markerRange = splice.Range{Start: output.Len(), End: output.Len() + 3}
			output.WriteByte('[')
			expectation.list.task.stateRange = splice.Range{Start: output.Len(), End: output.Len() + 1}
			if item.checked {
				output.WriteByte('x')
			} else {
				output.WriteByte(' ')
			}
			output.WriteString("] ")
		}
		output.WriteString(item.inlineGFM)
		contentEnd := output.Len()
		output.WriteByte('\n')
		previousLineEnd = output.Len()
		expectation.contentRange = splice.Range{Start: contentStart, End: contentEnd}
		expectation.sourceRange = splice.Range{Start: lineStart, End: previousLineEnd}
		expected = append(expected, expectation)
		openItems = append(openItems, len(expected)-1)
		contentIndents = append(contentIndents, contentStart-lineStart)
	}
	for len(openItems) != 0 {
		index := openItems[len(openItems)-1]
		expected[index].list.subtreeEnd = previousLineEnd
		openItems = openItems[:len(openItems)-1]
	}
	return expected
}

func constructionFenceLength(content string) int {
	maxRun := 0
	for lineStart := 0; lineStart <= len(content); {
		lineEnd := strings.IndexByte(content[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(content)
		} else {
			lineEnd += lineStart
		}
		line := content[lineStart:lineEnd]
		pos := 0
		for pos < len(line) && pos < 3 && line[pos] == ' ' {
			pos++
		}
		run := 0
		for pos+run < len(line) && line[pos+run] == '`' {
			run++
		}
		if run > maxRun {
			maxRun = run
		}
		if lineEnd == len(content) {
			break
		}
		lineStart = lineEnd + 1
	}
	if maxRun < 3 {
		return 3
	}
	return maxRun + 1
}

func writeConstructionFencedCode(output *bytes.Buffer, content, info string) constructionExpectation {
	start := output.Len()
	fenceLength := constructionFenceLength(content)
	fence := strings.Repeat("`", fenceLength)
	output.WriteString(fence)
	infoStart := output.Len()
	output.WriteString(info)
	infoRange := splice.Range{Start: infoStart, End: output.Len()}
	output.WriteByte('\n')
	contentStart := output.Len()
	output.WriteString(content)
	contentRange := splice.Range{Start: contentStart, End: output.Len()}
	if content != "" {
		output.WriteByte('\n')
	}
	output.WriteString(fence)
	mappingEnd := output.Len()
	output.WriteByte('\n')
	return constructionExpectation{
		kind:         splice.KindFencedCode,
		contentRange: contentRange,
		sourceRange:  splice.Range{Start: start, End: mappingEnd},
		fence: constructionFenceProof{
			infoRange:      infoRange,
			containerRange: splice.Range{Start: start, End: output.Len()},
			length:         fenceLength,
		},
	}
}

func writeConstructionReferenceDefinition(output *bytes.Buffer, label, destination, title string, hasTitle bool) constructionExpectation {
	start := output.Len()
	output.WriteByte('[')
	output.WriteString(label)
	output.WriteString("]: <")
	destinationStart := output.Len()
	output.WriteString(destination)
	destinationRange := splice.Range{Start: destinationStart, End: output.Len()}
	output.WriteByte('>')
	titleRange := splice.Range{}
	if hasTitle {
		output.WriteString(" \"")
		titleStart := output.Len()
		output.WriteString(title)
		titleRange = splice.Range{Start: titleStart, End: output.Len()}
		output.WriteByte('"')
	}
	mappingEnd := output.Len()
	output.WriteByte('\n')
	return constructionExpectation{
		kind:         splice.KindReferenceDefinition,
		contentRange: destinationRange,
		sourceRange:  splice.Range{Start: start, End: mappingEnd},
		reference: constructionReferenceProof{
			label:       label,
			destination: destination,
			title:       title,
			hasTitle:    hasTitle,
			titleRange:  titleRange,
		},
	}
}

func writeConstructionTable(output *bytes.Buffer, table constructionTable) []constructionExpectation {
	tableAnchor := output.Len()
	writeConstructionTableLine(output, table.header)
	delimiters := make([]string, len(table.alignments))
	for index, alignment := range table.alignments {
		delimiters[index] = constructionTableDelimiter(alignment)
	}
	writeConstructionTableLine(output, delimiters)
	alignments := constructionSpliceTableAlignments(table.alignments)

	expected := make([]constructionExpectation, 1, len(table.rows)+1)
	for _, row := range table.rows {
		lineStart := output.Len()
		lineRange, cellRanges, contentRanges := writeConstructionTableLine(output, row)
		expected = append(expected, constructionExpectation{
			kind:         splice.KindTableRow,
			contentRange: lineRange,
			sourceRange:  splice.Range{Start: lineStart, End: output.Len()},
			tableRow: constructionTableRowProof{
				tableAnchor:  tableAnchor,
				columnCount:  len(table.header),
				alignments:   append([]splice.TableAlignment(nil), alignments...),
				cellRanges:   cellRanges,
				cellContents: contentRanges,
			},
		})
	}
	tableRange := splice.Range{Start: tableAnchor, End: output.Len()}
	expected[0] = constructionExpectation{
		kind:         splice.KindTable,
		contentRange: tableRange,
		sourceRange:  tableRange,
		table: constructionTableProof{
			columnCount:  len(table.header),
			bodyRowCount: len(table.rows),
			alignments:   append([]splice.TableAlignment(nil), alignments...),
		},
	}
	return expected
}

func constructionTableDelimiter(alignment TableAlignment) string {
	switch alignment {
	case TableAlignmentLeft:
		return ":---"
	case TableAlignmentRight:
		return "---:"
	case TableAlignmentCenter:
		return ":---:"
	default:
		return "---"
	}
}

func constructionSpliceTableAlignments(alignments []TableAlignment) []splice.TableAlignment {
	result := make([]splice.TableAlignment, len(alignments))
	for index, alignment := range alignments {
		converted, ok := internalTableAlignment(alignment)
		if !ok {
			converted = splice.TableAlignmentDefault
		}
		result[index] = converted
	}
	return result
}

func writeConstructionTableLine(output *bytes.Buffer, cells []string) (splice.Range, []splice.Range, []splice.Range) {
	lineStart := output.Len()
	output.WriteByte('|')
	rawRanges := make([]splice.Range, len(cells))
	contentRanges := make([]splice.Range, len(cells))
	for index, cell := range cells {
		rawStart := output.Len()
		output.WriteByte(' ')
		contentStart := output.Len()
		output.WriteString(cell)
		contentEnd := output.Len()
		output.WriteByte(' ')
		rawEnd := output.Len()
		rawRanges[index] = splice.Range{Start: rawStart, End: rawEnd}
		if cell == "" {
			contentRanges[index] = splice.Range{Start: rawEnd, End: rawEnd}
		} else {
			contentRanges[index] = splice.Range{Start: contentStart, End: contentEnd}
		}
		output.WriteByte('|')
	}
	lineEnd := output.Len()
	output.WriteByte('\n')
	return splice.Range{Start: lineStart, End: lineEnd}, rawRanges, contentRanges
}
