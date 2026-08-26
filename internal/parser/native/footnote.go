package native

import (
	"bytes"
	"sort"

	"github.com/zoster81/marksplice/internal/parser"
)

type nativeFootnoteDefinition struct {
	observation parser.FootnoteDefinitionObservation
	bodyBlocks  []inlineBlock
}

func nativeFootnoteObservations(source []byte, blocks blockParseResult) ([]parser.FootnoteDefinitionObservation, []parser.FootnoteReferenceObservation, []parser.LinkUsage) {
	parsed := appendNativeContainerFootnoteDefinitions(source, scanNativeFootnoteDefinitions(source), blocks.nodes)
	definitions := make([]parser.FootnoteDefinitionObservation, len(parsed))
	for index := range parsed {
		definitions[index] = parsed[index].observation
	}
	ordinaryDefinitions := nativeOrdinaryReferenceDefinitions(source, blocks.references, definitions)
	references := scanNativeFootnoteReferences(source, blocks.inlines, parsed, ordinaryDefinitions)
	bodyUsages := nativeFootnoteBodyLinkUsages(source, parsed, ordinaryDefinitions)
	return definitions, references, bodyUsages
}

func scanNativeFootnoteDefinitions(source []byte) []nativeFootnoteDefinition {
	lines := physicalLines(source)
	result := make([]nativeFootnoteDefinition, 0)
	for index := 0; index < len(lines); {
		anchor, label, bodyStart, ok := nativeFootnoteOpening(source, lines[index])
		if !ok {
			index++
			continue
		}
		childLines, next := nativeFootnoteChildLines(source, lines, index, bodyStart)
		if nativeFootnoteHasNestedDefinition(source, childLines) {
			index = next
			continue
		}
		child := parseBlockLines(source, childLines, false)
		bodyRanges := normalizeNativeFootnoteBodyRanges(source, child.semantic)
		result = append(result, nativeFootnoteDefinition{
			observation: parser.FootnoteDefinitionObservation{
				Anchor:     anchor,
				Label:      label,
				BodyRanges: bodyRanges,
			},
			bodyBlocks: append([]inlineBlock(nil), child.inlines...),
		})
		index = next
	}
	return result
}

func appendNativeContainerFootnoteDefinitions(source []byte, definitions []nativeFootnoteDefinition, nodes []parser.Node) []nativeFootnoteDefinition {
	seen := make(map[int]struct{}, len(definitions))
	for _, definition := range definitions {
		seen[definition.observation.Anchor] = struct{}{}
	}
	for _, node := range nodes {
		if !nativeFootnoteContainerAnchorCandidate(node) {
			continue
		}
		if _, exists := seen[node.Range.Start]; exists {
			continue
		}
		definition, ok := nativeContainerFootnoteDefinitionAt(source, node.Range.Start)
		if !ok {
			continue
		}
		definitions = append(definitions, definition)
		seen[definition.observation.Anchor] = struct{}{}
	}
	sort.SliceStable(definitions, func(left, right int) bool {
		return definitions[left].observation.Anchor < definitions[right].observation.Anchor
	})
	return definitions
}

func nativeFootnoteContainerAnchorCandidate(node parser.Node) bool {
	if node.Kind == parser.KindListItem {
		return true
	}
	return node.Kind == parser.KindReferenceDefinition && len(node.Label) >= 2 && node.Label[0] == '^'
}

func nativeContainerFootnoteDefinitionAt(source []byte, anchor int) (nativeFootnoteDefinition, bool) {
	if anchor < 0 || anchor >= len(source) {
		return nativeFootnoteDefinition{}, false
	}
	lines, openingIndex := nativeContainerFootnoteLines(source, anchor)
	if openingIndex < 0 {
		return nativeFootnoteDefinition{}, false
	}
	opening := lines[openingIndex]
	definitionAnchor, label, bodyStart, ok := nativeFootnoteOpening(source, opening)
	if !ok || definitionAnchor != anchor {
		return nativeFootnoteDefinition{}, false
	}
	childLines, _ := nativeFootnoteChildLines(source, lines, openingIndex, bodyStart)
	if nativeFootnoteHasNestedDefinition(source, childLines) {
		return nativeFootnoteDefinition{}, false
	}
	child := parseBlockLines(source, childLines, false)
	return nativeFootnoteDefinition{
		observation: parser.FootnoteDefinitionObservation{
			Anchor:     anchor,
			Label:      label,
			BodyRanges: normalizeNativeFootnoteBodyRanges(source, child.semantic),
		},
		bodyBlocks: append([]inlineBlock(nil), child.inlines...),
	}, true
}

func nativeContainerFootnoteLines(source []byte, anchor int) ([]physicalLine, int) {
	lines := physicalLines(source)
	openingIndex := sort.Search(len(lines), func(index int) bool { return lines[index].next > anchor })
	if openingIndex >= len(lines) || anchor < lines[openingIndex].physicalStart || anchor > lines[openingIndex].end {
		return nil, -1
	}
	containerIndent := nativeFootnoteListContainerIndent(source, lines[openingIndex], anchor)
	lines[openingIndex] = advancePhysicalLineStart(source, lines[openingIndex], anchor, 0)
	if containerIndent == 0 {
		return lines, openingIndex
	}
	for index := openingIndex + 1; index < len(lines); index++ {
		if blankLine(source, lines[index]) {
			continue
		}
		_, columns := leadingIndent(source, lines[index])
		if columns < containerIndent {
			return lines[:index], openingIndex
		}
		lines[index] = stripIndentColumns(source, lines[index], containerIndent)
	}
	return lines, openingIndex
}

func nativeFootnoteListContainerIndent(source []byte, line physicalLine, anchor int) int {
	marker, ok := parseListMarker(source, line)
	if !ok || marker.contentStart > anchor {
		return 0
	}
	return marker.contentIndent
}

func nativeFootnoteOpening(source []byte, line physicalLine) (int, string, int, bool) {
	indent, ordinary := ordinaryIndent(source, line)
	if !ordinary {
		return 0, "", 0, false
	}
	anchor := line.start + indent
	if anchor+4 > line.end || source[anchor] != '[' || source[anchor+1] != '^' {
		return 0, "", 0, false
	}
	close := nativeFootnoteLabelEnd(source, anchor+2, line.end)
	if close < 0 || close+1 >= line.end || source[close+1] != ':' {
		return 0, "", 0, false
	}
	labelBytes := source[anchor+2 : close]
	if len(bytes.Trim(labelBytes, " \t")) == 0 {
		return 0, "", 0, false
	}
	return anchor, string(labelBytes), close + 2, true
}

func nativeFootnoteLabelEnd(source []byte, start, limit int) int {
	for position := start; position < limit; position++ {
		switch source[position] {
		case '[':
			return -1
		case ']':
			if !nativeSourceByteEscaped(source, position) {
				return position
			}
		}
	}
	return -1
}

func nativeFootnoteChildLines(source []byte, lines []physicalLine, index, bodyStart int) ([]physicalLine, int) {
	opening := lines[index]
	children := make([]physicalLine, 0, 2)
	allowLazy := false
	if bodyStart < opening.end {
		opening = advancePhysicalLineStart(source, opening, bodyStart, 0)
		children = append(children, opening)
		allowLazy = nestedParagraphEligibleLine(source, opening)
	}
	next := index + 1
	for next < len(lines) {
		line := lines[next]
		if blankLine(source, line) {
			children = append(children, line)
			allowLazy = false
			next++
			continue
		}
		_, columns := leadingIndent(source, line)
		if columns >= 4 {
			line = stripIndentColumns(source, line, 4)
			children = append(children, line)
			allowLazy = nestedParagraphEligibleLine(source, line)
			next++
			continue
		}
		if _, _, _, nestedOpening := nativeFootnoteOpening(source, line); nestedOpening {
			break
		}
		if allowLazy && strictContainerLazyParagraphLine(source, line) {
			children = append(children, line)
			next++
			continue
		}
		break
	}
	return children, next
}

func nativeFootnoteHasNestedDefinition(source []byte, lines []physicalLine) bool {
	for _, line := range lines {
		if _, _, _, ok := nativeFootnoteOpening(source, line); ok {
			return true
		}
	}
	return false
}

func normalizeNativeFootnoteBodyRanges(source []byte, ranges []parser.Range) []parser.Range {
	if len(ranges) == 0 {
		return []parser.Range{}
	}
	result := ranges[:0]
	for _, range_ := range ranges {
		if !range_.Valid(len(source)) || range_.Start >= range_.End {
			continue
		}
		if lineEnd := nativePhysicalLineEnd(source, range_.Start); lineEnd < range_.End {
			range_.End = nativePhysicalLineRangeEnd(source, range_.Start)
		}
		range_.End = trimHorizontalEnd(source, range_.Start, range_.End)
		if range_.Start < range_.End {
			result = append(result, range_)
		}
	}
	sort.SliceStable(result, func(left, right int) bool {
		if result[left].Start != result[right].Start {
			return result[left].Start < result[right].Start
		}
		return result[left].End < result[right].End
	})
	if len(result) < 2 {
		return result
	}
	deduplicated := result[:1]
	for _, range_ := range result[1:] {
		if deduplicated[len(deduplicated)-1] != range_ {
			deduplicated = append(deduplicated, range_)
		}
	}
	return deduplicated
}

func nativeOrdinaryReferenceDefinitions(source []byte, definitions []referenceDefinitionParse, footnotes []parser.FootnoteDefinitionObservation) []referenceDefinitionParse {
	claimedLabels := make(map[string]struct{}, len(footnotes))
	for _, definition := range footnotes {
		claimedLabels[ReferenceLabelKey("^"+definition.Label)] = struct{}{}
	}
	claimedRanges := nativeFootnoteClaimedRanges(source, footnotes)
	result := make([]referenceDefinitionParse, 0, len(definitions))
	for _, definition := range definitions {
		if _, suppressed := claimedLabels[ReferenceLabelKey(definition.label)]; suppressed {
			continue
		}
		if definition.hasAnchor && nativeOffsetInsideAny(definition.anchor, claimedRanges) {
			continue
		}
		result = append(result, definition)
	}
	return result
}

func scanNativeFootnoteReferences(source []byte, blocks []inlineBlock, definitions []nativeFootnoteDefinition, ordinaryDefinitions []referenceDefinitionParse) []parser.FootnoteReferenceObservation {
	byLabel := make(map[string]int, len(definitions))
	observations := make([]parser.FootnoteDefinitionObservation, len(definitions))
	for index, definition := range definitions {
		observations[index] = definition.observation
		if _, exists := byLabel[definition.observation.Label]; !exists {
			byLabel[definition.observation.Label] = definition.observation.Anchor
		}
	}
	claims := nativeFootnoteClaimedRanges(source, observations)
	result := make([]parser.FootnoteReferenceObservation, 0)
	for _, block := range blocks {
		for _, reference := range scanNativeFootnoteReferenceBlock(source, block, byLabel, ordinaryDefinitions) {
			if !nativeRangeInsideAny(reference.Range, claims) {
				result = append(result, reference)
			}
		}
	}
	for _, definition := range definitions {
		for _, block := range definition.bodyBlocks {
			result = append(result, scanNativeFootnoteReferenceBlock(source, block, byLabel, ordinaryDefinitions)...)
		}
	}
	sort.SliceStable(result, func(left, right int) bool { return result[left].Range.Start < result[right].Range.Start })
	result = deduplicateNativeFootnoteReferences(result)
	occurrences := make(map[int]int, len(definitions))
	for index := range result {
		anchor := result[index].DefinitionAnchor
		result[index].Occurrence = occurrences[anchor]
		occurrences[anchor]++
	}
	return result
}

func scanNativeFootnoteReferenceBlock(source []byte, block inlineBlock, definitions map[string]int, ordinaryDefinitions []referenceDefinitionParse) []parser.FootnoteReferenceObservation {
	runs := collectBacktickRuns(source, block)
	nextSame := nextSameLengthRuns(runs)
	spans := collectInlineSpans(source, block)
	owners, _, barriers := resolvePrimaryInlineOwners(source, block, runs, nextSame, spans)
	delimiters := parseDelimiterObservations(source, block, owners, barriers, ordinaryDefinitions)
	exclusions := nativeFootnoteInlineExclusions(block, owners, delimiters.composites)
	result := make([]parser.FootnoteReferenceObservation, 0)
	for segmentIndex, segment := range block.segments {
		for position := segment.Start; position < segment.End; position++ {
			if inlineRangesContainPosition(exclusions[segmentIndex], position) || source[position] != '[' {
				continue
			}
			observation, end, ok := nativeFootnoteReferenceAt(source, position, segment.End, definitions)
			if !ok {
				continue
			}
			result = append(result, observation)
			position = end - 1
		}
	}
	return result
}

func nativeFootnoteInlineExclusions(block inlineBlock, owners []inlineSpan, composites []compositeInline) [][]parser.Range {
	exclusions := make([][]parser.Range, len(block.segments))
	for _, owner := range owners {
		appendRelationshipExclusion(exclusions, block, owner.segment, owner.start, owner.endSegment, owner.end)
	}
	for _, composite := range composites {
		if !composite.active || composite.form != parser.LinkUsageDirect {
			continue
		}
		appendRelationshipExclusion(exclusions, block, composite.segment, composite.label.End, composite.endSegment, composite.end)
	}
	normalizeInlineExclusions(exclusions)
	return exclusions
}

func nativeFootnoteReferenceAt(source []byte, start, limit int, definitions map[string]int) (parser.FootnoteReferenceObservation, int, bool) {
	if start+4 > limit || source[start] != '[' || source[start+1] != '^' || nativeSourceByteEscaped(source, start) {
		return parser.FootnoteReferenceObservation{}, start, false
	}
	end := nativeFootnoteLabelEnd(source, start+2, limit)
	if end < 0 {
		return parser.FootnoteReferenceObservation{}, start, false
	}
	label := string(source[start+2 : end])
	definitionAnchor, ok := definitions[label]
	if !ok {
		return parser.FootnoteReferenceObservation{}, start, false
	}
	return parser.FootnoteReferenceObservation{
		Range:            parser.Range{Start: start, End: end + 1},
		LabelRange:       parser.Range{Start: start + 2, End: end},
		Label:            label,
		DefinitionAnchor: definitionAnchor,
	}, end + 1, true
}

func deduplicateNativeFootnoteReferences(references []parser.FootnoteReferenceObservation) []parser.FootnoteReferenceObservation {
	if len(references) < 2 {
		return references
	}
	result := references[:1]
	for _, reference := range references[1:] {
		last := result[len(result)-1]
		if last.Range == reference.Range && last.DefinitionAnchor == reference.DefinitionAnchor && last.Label == reference.Label {
			continue
		}
		result = append(result, reference)
	}
	return result
}

func nativeFootnoteBodyLinkUsages(source []byte, definitions []nativeFootnoteDefinition, ordinaryDefinitions []referenceDefinitionParse) []parser.LinkUsage {
	result := make([]parser.LinkUsage, 0)
	for _, definition := range definitions {
		parsed := parseInlineBlocks(source, definition.bodyBlocks, ordinaryDefinitions)
		result = append(result, parsed.usages...)
	}
	sort.SliceStable(result, func(left, right int) bool { return result[left].Anchor < result[right].Anchor })
	return result
}

func reconcileNativeFootnotes(source []byte, nodes []parser.Node, usages []parser.LinkUsage, unresolved []parser.UnresolvedReferenceUsage, definitions []parser.FootnoteDefinitionObservation, bodyUsages []parser.LinkUsage) ([]parser.Node, []parser.LinkUsage, []parser.UnresolvedReferenceUsage) {
	if len(definitions) == 0 {
		return nodes, usages, unresolved
	}
	claims := nativeFootnoteClaimedRanges(source, definitions)
	caretKeys := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		caretKeys[ReferenceLabelKey("^"+definition.Label)] = struct{}{}
	}
	filteredNodes := make([]parser.Node, 0, len(nodes))
	for _, node := range nodes {
		if nativeRangeInsideAny(node.Range, claims) {
			continue
		}
		filteredNodes = append(filteredNodes, node)
	}
	filteredUsages := make([]parser.LinkUsage, 0, len(usages)+len(bodyUsages))
	suppressedNodeAnchors := make(map[constructionSemanticKey]struct{})
	for _, usage := range usages {
		if nativeOffsetInsideAny(usage.Anchor, claims) {
			continue
		}
		if usage.Form != parser.LinkUsageDirect {
			if _, suppressed := caretKeys[ReferenceLabelKey(usage.Reference)]; suppressed {
				suppressedNodeAnchors[constructionSemanticKey{kind: usage.Kind, syntax: parser.Range{Start: usage.Anchor, End: usage.Anchor}}] = struct{}{}
				continue
			}
		}
		filteredUsages = append(filteredUsages, usage)
	}
	if len(suppressedNodeAnchors) != 0 {
		kept := filteredNodes[:0]
		for _, node := range filteredNodes {
			key := constructionSemanticKey{kind: node.Kind, syntax: parser.Range{Start: node.Anchor, End: node.Anchor}}
			if _, suppressed := suppressedNodeAnchors[key]; suppressed {
				continue
			}
			kept = append(kept, node)
		}
		filteredNodes = kept
	}
	filteredUsages = append(filteredUsages, bodyUsages...)
	sort.SliceStable(filteredUsages, func(left, right int) bool { return filteredUsages[left].Anchor < filteredUsages[right].Anchor })
	filteredUsages = deduplicateNativeLinkUsages(filteredUsages)
	filteredUnresolved := make([]parser.UnresolvedReferenceUsage, 0, len(unresolved))
	for _, usage := range unresolved {
		if !nativeOffsetInsideAny(usage.Anchor, claims) {
			filteredUnresolved = append(filteredUnresolved, usage)
		}
	}
	return filteredNodes, filteredUsages, filteredUnresolved
}

func deduplicateNativeLinkUsages(usages []parser.LinkUsage) []parser.LinkUsage {
	if len(usages) < 2 {
		return usages
	}
	result := usages[:1]
	for _, usage := range usages[1:] {
		if result[len(result)-1] == usage {
			continue
		}
		result = append(result, usage)
	}
	return result
}

func nativeFootnoteClaimedRanges(source []byte, definitions []parser.FootnoteDefinitionObservation) []parser.Range {
	claims := make([]parser.Range, 0, len(definitions))
	for _, definition := range definitions {
		if definition.Anchor < 0 || definition.Anchor >= len(source) {
			continue
		}
		start := nativePhysicalLineStart(source, definition.Anchor)
		end := nativePhysicalLineRangeEnd(source, definition.Anchor)
		for _, body := range definition.BodyRanges {
			if body.Valid(len(source)) && body.End > body.Start {
				if candidate := nativePhysicalLineRangeEnd(source, body.End); candidate > end {
					end = candidate
				}
			}
		}
		if end > start {
			claims = append(claims, parser.Range{Start: start, End: end})
		}
	}
	return normalizeNativeFootnoteClaims(claims)
}

func normalizeNativeFootnoteClaims(claims []parser.Range) []parser.Range {
	if len(claims) < 2 {
		return claims
	}
	sort.Slice(claims, func(left, right int) bool {
		if claims[left].Start != claims[right].Start {
			return claims[left].Start < claims[right].Start
		}
		return claims[left].End < claims[right].End
	})
	result := claims[:1]
	for _, claim := range claims[1:] {
		last := &result[len(result)-1]
		if claim.Start < last.End {
			if claim.End > last.End {
				last.End = claim.End
			}
			continue
		}
		result = append(result, claim)
	}
	return result
}

func nativePhysicalLineRangeEnd(source []byte, offset int) int {
	end := nativePhysicalLineEnd(source, offset)
	if end < len(source) && source[end] == '\r' {
		end++
		if end < len(source) && source[end] == '\n' {
			end++
		}
	} else if end < len(source) && source[end] == '\n' {
		end++
	}
	return end
}

func nativeRangeInsideAny(range_ parser.Range, claims []parser.Range) bool {
	if range_.Start < 0 || range_.End < range_.Start {
		return false
	}
	index := sort.Search(len(claims), func(index int) bool { return claims[index].End > range_.Start })
	return index < len(claims) && range_.Start >= claims[index].Start && range_.End <= claims[index].End
}

func nativeOffsetInsideAny(offset int, claims []parser.Range) bool {
	index := sort.Search(len(claims), func(index int) bool { return claims[index].End > offset })
	return index < len(claims) && offset >= claims[index].Start
}
