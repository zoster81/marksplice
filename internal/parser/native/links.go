package native

import (
	"sort"
	"strings"

	"golang.org/x/text/cases"

	"github.com/zoster81/marksplice/internal/parser"
)

var referenceCaseFolder = cases.Fold()

// compositeInline records syntax that owns inline children but must not hide its
// label from later delimiter parsing. M113 initially uses it to distinguish a
// genuinely plain emphasis child from one containing a link/image owner.
type compositeInline struct {
	kind            parser.Kind
	segment         int
	start           int
	endSegment      int
	end             int
	label           parser.Range
	labelEndSegment int
	destination     string
	title           string
	hasTitle        bool
	form            parser.LinkUsageForm
	reference       string
	active          bool
}

type inlineDestinationTail struct {
	endSegment  int
	end         int
	destination string
	title       string
	hasTitle    bool
}

func collectCompositeInlines(source []byte, block inlineBlock, owners []inlineSpan, definitions []referenceDefinitionParse) []compositeInline {
	primaryExclusions := inlineOwnerExclusions(block, owners, nil)
	composites := make([]compositeInline, 0)
	for segmentIndex, segment := range block.segments {
		exclusions := primaryExclusions[segmentIndex]
		for position := segment.Start; position < segment.End; position++ {
			if inlineRangesContainPosition(exclusions, position) {
				continue
			}
			image := source[position] == '!' && position+1 < segment.End && source[position+1] == '['
			if source[position] == '[' && imageMarkerOwnsBracket(source, segment, position) {
				continue
			}
			if source[position] != '[' && !image || inlineByteEscaped(source, segment.Start, position) {
				continue
			}
			candidate, ok := scanDirectComposite(source, block, segmentIndex, segment, position, image, exclusions)
			if ok {
				composites = append(composites, candidate)
			}
		}
	}
	composites = append(composites, collectReferenceComposites(source, block, owners, definitions, composites)...)
	sortCompositeInlines(composites)
	activateCompositeInlines(composites, owners, block)
	return composites
}

func imageMarkerOwnsBracket(source []byte, segment parser.Range, position int) bool {
	return position > segment.Start && source[position-1] == '!' && !inlineByteEscaped(source, segment.Start, position-1)
}

func sortCompositeInlines(composites []compositeInline) {
	sort.SliceStable(composites, func(left, right int) bool {
		if composites[left].segment != composites[right].segment {
			return composites[left].segment < composites[right].segment
		}
		if composites[left].start != composites[right].start {
			return composites[left].start < composites[right].start
		}
		return composites[left].end < composites[right].end
	})
}

func scanDirectComposite(source []byte, block inlineBlock, segmentIndex int, segment parser.Range, position int, image bool, exclusions []parser.Range) (compositeInline, bool) {
	open := position
	labelStart := position + 1
	kind := parser.KindInlineLink
	if image {
		labelStart++
		kind = parser.KindImage
	}
	labelEnd, ok := scanInlineLabelEnd(source, segment, labelStart, exclusions)
	if !ok {
		return compositeInline{}, false
	}
	tail, ok := scanInlineDestinationSyntax(source, labelEnd+1, segment.End)
	if ok {
		tail.endSegment = segmentIndex
	} else {
		if segmentIndex+1 >= len(block.segments) {
			return compositeInline{}, false
		}
		tail, ok = scanInlineDestinationAcrossSegments(source, block, segmentIndex, labelEnd+1)
		if !ok {
			return compositeInline{}, false
		}
	}
	return compositeInline{
		kind:            kind,
		segment:         segmentIndex,
		start:           open,
		endSegment:      tail.endSegment,
		end:             tail.end,
		label:           parser.Range{Start: labelStart, End: labelEnd},
		labelEndSegment: segmentIndex,
		destination:     tail.destination,
		title:           tail.title,
		hasTitle:        tail.hasTitle,
		form:            parser.LinkUsageDirect,
	}, true
}

func collectReferenceComposites(source []byte, block inlineBlock, owners []inlineSpan, definitions []referenceDefinitionParse, direct []compositeInline) []compositeInline {
	if len(definitions) == 0 {
		return nil
	}
	primaryExclusions := inlineOwnerExclusions(block, owners, nil)
	resolved := basicReferenceDefinitions(definitions)
	directStarts := newInlineStartIndex(len(block.segments))
	for _, composite := range direct {
		directStarts.add(composite.segment, composite.start)
	}
	directStarts.finalize()
	composites := make([]compositeInline, 0)
	for segmentIndex, segment := range block.segments {
		exclusions := primaryExclusions[segmentIndex]
		for position := segment.Start; position < segment.End; position++ {
			if inlineRangesContainPosition(exclusions, position) || directStarts.hasAt(segmentIndex, position) {
				continue
			}
			image := source[position] == '!' && position+1 < segment.End && source[position+1] == '['
			if source[position] == '[' && imageMarkerOwnsBracket(source, segment, position) {
				continue
			}
			if source[position] != '[' && !image || inlineByteEscaped(source, segment.Start, position) {
				continue
			}
			candidate, ok := scanReferenceComposite(source, block, segmentIndex, position, image, primaryExclusions, resolved)
			if ok {
				composites = append(composites, candidate)
			}
		}
	}
	return composites
}

func scanReferenceComposite(source []byte, block inlineBlock, segmentIndex, position int, image bool, exclusions [][]parser.Range, definitions map[string]referenceDefinitionParse) (compositeInline, bool) {
	open := position
	labelStart := position + 1
	kind := parser.KindInlineLink
	if image {
		labelStart++
		kind = parser.KindImage
	}
	labelEndSegment, labelEnd, ok := scanInlineLabelAcrossSegments(source, block, segmentIndex, labelStart, exclusions)
	if !ok {
		return compositeInline{}, false
	}
	form := parser.LinkUsageShortcut
	reference := referenceSourceValue(source, labelStart, labelEnd)
	endSegment := labelEndSegment
	end := labelEnd + 1
	endLine := block.segments[labelEndSegment]
	if end < endLine.End && source[end] == '[' && !inlineByteEscaped(source, endLine.Start, end) {
		if secondEndSegment, secondEnd, ok := scanReferenceUsageLabelAcrossSegments(source, block, labelEndSegment, end+1); ok {
			if secondEndSegment == labelEndSegment && secondEnd == end+1 {
				form = parser.LinkUsageCollapsed
			} else {
				form = parser.LinkUsageFull
				reference = referenceSourceValue(source, end+1, secondEnd)
			}
			endSegment = secondEndSegment
			end = secondEnd + 1
		}
	}
	if len(reference) > 999 {
		return compositeInline{}, false
	}
	definition, ok := definitions[ReferenceLabelKey(reference)]
	if !ok {
		return compositeInline{}, false
	}
	return compositeInline{
		kind:            kind,
		segment:         segmentIndex,
		start:           open,
		endSegment:      endSegment,
		end:             end,
		label:           parser.Range{Start: labelStart, End: labelEnd},
		labelEndSegment: labelEndSegment,
		destination:     definition.destination,
		title:           definition.title,
		hasTitle:        definition.hasTitle,
		form:            form,
		reference:       reference,
	}, true
}

func scanReferenceUsageLabelEnd(source []byte, segment parser.Range, start int) (int, bool) {
	for position := start; position < segment.End; position++ {
		if source[position] == '\\' && position+1 < segment.End && asciiPunctuation(source[position+1]) {
			position++
			continue
		}
		if source[position] == '[' {
			return start, false
		}
		if source[position] == ']' {
			return position, true
		}
	}
	return start, false
}

func scanInlineLabelAcrossSegments(source []byte, block inlineBlock, startSegment, start int, exclusions [][]parser.Range) (int, int, bool) {
	if startSegment < 0 || startSegment >= len(block.segments) || len(exclusions) != len(block.segments) {
		return startSegment, start, false
	}
	depth := 0
	for segmentIndex := startSegment; segmentIndex < len(block.segments); segmentIndex++ {
		segment := block.segments[segmentIndex]
		position := segment.Start
		if segmentIndex == startSegment {
			position = start
		}
		end, nextDepth, ok := scanInlineLabelSegment(source, segment, position, exclusions[segmentIndex], depth)
		if ok {
			return segmentIndex, end, true
		}
		depth = nextDepth
	}
	return startSegment, start, false
}

func scanInlineLabelSegment(source []byte, segment parser.Range, position int, exclusions []parser.Range, depth int) (int, int, bool) {
	exclusionIndex := 0
	for position < segment.End {
		for exclusionIndex < len(exclusions) && position >= exclusions[exclusionIndex].End {
			exclusionIndex++
		}
		if exclusionIndex < len(exclusions) && position >= exclusions[exclusionIndex].Start {
			position = exclusions[exclusionIndex].End
			continue
		}
		if inlineByteEscaped(source, segment.Start, position) {
			position++
			continue
		}
		switch source[position] {
		case '[':
			depth++
		case ']':
			if depth == 0 {
				return position, depth, true
			}
			depth--
		}
		position++
	}
	return position, depth, false
}

func scanReferenceUsageLabelAcrossSegments(source []byte, block inlineBlock, startSegment, start int) (int, int, bool) {
	if startSegment < 0 || startSegment >= len(block.segments) {
		return startSegment, start, false
	}
	if end, ok := scanReferenceUsageLabelEnd(source, block.segments[startSegment], start); ok {
		return startSegment, end, true
	}
	for segmentIndex := startSegment; segmentIndex < len(block.segments); segmentIndex++ {
		segment := block.segments[segmentIndex]
		position := segment.Start
		if segmentIndex == startSegment {
			position = start
		}
		for ; position < segment.End; position++ {
			if source[position] == '\\' && position+1 < segment.End && asciiPunctuation(source[position+1]) {
				position++
				continue
			}
			if source[position] == '[' {
				return startSegment, start, false
			}
			if source[position] == ']' {
				return segmentIndex, position, true
			}
		}
	}
	return startSegment, start, false
}

func referenceSourceValue(source []byte, start, end int) string {
	if start < 0 || end < start || end > len(source) {
		return ""
	}
	value := append([]byte(nil), source[start:end]...)
	for index := 0; index < len(value); index++ {
		if value[index] == '\r' && (index+1 == len(value) || value[index+1] != '\n') {
			value[index] = '\n'
		}
	}
	return string(value)
}

func basicReferenceDefinitions(definitions []referenceDefinitionParse) map[string]referenceDefinitionParse {
	result := make(map[string]referenceDefinitionParse, len(definitions))
	for _, definition := range definitions {
		key := ReferenceLabelKey(definition.label)
		if key == "" {
			continue
		}
		if _, exists := result[key]; !exists {
			result[key] = definition
		}
	}
	return result
}

// ReferenceLabelKey returns the M113-native GFM normalization key for one
// reference label: full Unicode case folding plus whitespace normalization.
func ReferenceLabelKey(label string) string {
	folded := referenceCaseFolder.String(label)
	var normalized strings.Builder
	normalized.Grow(len(folded))
	pendingSpace := false
	wrote := false
	for _, value := range folded {
		if referenceLabelWhitespace(value) {
			if wrote {
				pendingSpace = true
			}
			continue
		}
		if pendingSpace {
			normalized.WriteByte(' ')
			pendingSpace = false
		}
		normalized.WriteRune(value)
		wrote = true
	}
	return normalized.String()
}

func referenceLabelWhitespace(value rune) bool {
	switch value {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	default:
		return false
	}
}

func compositeLinkUsages(composites []compositeInline) []parser.LinkUsage {
	usages := make([]parser.LinkUsage, 0, len(composites))
	for _, composite := range composites {
		if !composite.active || composite.form == parser.LinkUsageUnknown {
			continue
		}
		usage := parser.LinkUsage{
			Kind:        composite.kind,
			Form:        composite.form,
			Anchor:      composite.start,
			Destination: composite.destination,
			Title:       composite.title,
			HasTitle:    composite.hasTitle,
		}
		if composite.form != parser.LinkUsageDirect {
			usage.Reference = composite.reference
		}
		usages = append(usages, usage)
	}
	return usages
}

func filterPrimaryCompositeSyntaxNodes(block inlineBlock, nodes []parser.Node, composites []compositeInline) []parser.Node {
	exclusions := flattenInlineExclusions(inlineOwnerExclusions(block, nil, composites))
	filtered := nodes[:0]
	for _, node := range nodes {
		position := node.Anchor
		if node.Kind == parser.KindRawHTML {
			position = node.Range.Start
		}
		if inlineRangesContainPosition(exclusions, position) {
			continue
		}
		filtered = append(filtered, node)
	}
	clear(nodes[len(filtered):])
	return filtered
}

func autoLinkUsages(nodes []parser.Node) []parser.LinkUsage {
	usages := make([]parser.LinkUsage, 0)
	for _, node := range nodes {
		if node.Kind != parser.KindAutoLink {
			continue
		}
		usages = append(usages, parser.LinkUsage{
			Kind:          parser.KindAutoLink,
			Form:          parser.LinkUsageDirect,
			Anchor:        node.Anchor,
			Destination:   autoLinkDestination(node),
			AutoLinkEmail: node.AutoLinkEmail,
		})
	}
	return usages
}

func autoLinkDestination(node parser.Node) string {
	if strings.HasPrefix(node.Value, "www.") {
		return "http://" + node.Value
	}
	return node.Value
}

func unresolvedReferenceUsages(source []byte, block inlineBlock, owners []inlineSpan, composites []compositeInline, matches []delimiterMatch, definitions []referenceDefinitionParse) []parser.UnresolvedReferenceUsage {
	resolved := basicReferenceDefinitions(definitions)
	exclusions := relationshipExclusions(block, owners, composites)
	appendDelimiterRelationshipExclusions(exclusions, block, matches)
	normalizeInlineExclusions(exclusions)
	result := make([]parser.UnresolvedReferenceUsage, 0)
	for segmentIndex, segment := range block.segments {
		exclusionIndex := 0
		for position := segment.Start; position < segment.End; {
			for exclusionIndex < len(exclusions[segmentIndex]) && position >= exclusions[segmentIndex][exclusionIndex].End {
				exclusionIndex++
			}
			if exclusionIndex < len(exclusions[segmentIndex]) && position >= exclusions[segmentIndex][exclusionIndex].Start {
				position = exclusions[segmentIndex][exclusionIndex].End
				continue
			}
			limit := segment.End
			if exclusionIndex < len(exclusions[segmentIndex]) && exclusions[segmentIndex][exclusionIndex].Start > position {
				limit = exclusions[segmentIndex][exclusionIndex].Start
			}
			usage, end, ok := scanUnresolvedReferenceUsage(source, parser.Range{Start: segment.Start, End: limit}, position, resolved)
			if !ok {
				position++
				continue
			}
			result = append(result, usage)
			position = end
		}
	}
	return result
}

func appendDelimiterRelationshipExclusions(exclusions [][]parser.Range, block inlineBlock, matches []delimiterMatch) {
	for _, match := range matches {
		appendDelimiterRelationshipExclusion(exclusions, block, match.startSegment, match.openingConsumed)
		appendDelimiterRelationshipExclusion(exclusions, block, match.endSegment, match.closingConsumed)
	}
}

func appendDelimiterRelationshipExclusion(exclusions [][]parser.Range, block inlineBlock, segmentIndex int, range_ parser.Range) {
	if segmentIndex < 0 || segmentIndex >= len(block.segments) || range_.Start >= range_.End {
		return
	}
	segment := block.segments[segmentIndex]
	start := max(segment.Start, range_.Start)
	end := min(segment.End, range_.End)
	if start < end {
		exclusions[segmentIndex] = append(exclusions[segmentIndex], parser.Range{Start: start, End: end})
	}
}

func relationshipExclusions(block inlineBlock, owners []inlineSpan, composites []compositeInline) [][]parser.Range {
	exclusions := inlineBlockExclusions(block)
	for _, owner := range owners {
		appendRelationshipExclusion(exclusions, block, owner.segment, owner.start, owner.endSegment, owner.end)
	}
	for _, composite := range composites {
		if !composite.active {
			continue
		}
		appendRelationshipExclusion(exclusions, block, composite.segment, composite.start, composite.endSegment, composite.end)
	}
	normalizeInlineExclusions(exclusions)
	return exclusions
}

func appendRelationshipExclusion(exclusions [][]parser.Range, block inlineBlock, startSegment, start, endSegment, end int) {
	if startSegment < 0 || endSegment < startSegment || endSegment >= len(block.segments) {
		return
	}
	for segmentIndex := startSegment; segmentIndex <= endSegment; segmentIndex++ {
		segment := block.segments[segmentIndex]
		rangeStart, rangeEnd := segment.Start, segment.End
		if segmentIndex == startSegment {
			rangeStart = start
		}
		if segmentIndex == endSegment {
			rangeEnd = end
		}
		if rangeStart < rangeEnd {
			exclusions[segmentIndex] = append(exclusions[segmentIndex], parser.Range{Start: rangeStart, End: rangeEnd})
		}
	}
}

func scanUnresolvedReferenceUsage(source []byte, segment parser.Range, anchor int, definitions map[string]referenceDefinitionParse) (parser.UnresolvedReferenceUsage, int, bool) {
	kind := parser.KindInlineLink
	open := anchor
	if source[anchor] == '!' {
		if inlineByteEscaped(source, segment.Start, anchor) {
			return parser.UnresolvedReferenceUsage{}, anchor, false
		}
		kind = parser.KindImage
		open++
	}
	if open >= segment.End || source[open] != '[' || inlineByteEscaped(source, segment.Start, open) {
		return parser.UnresolvedReferenceUsage{}, anchor, false
	}
	first, firstEnd, ok := scanPlainReferenceUsageLabel(source, open, segment.End)
	if !ok || firstEnd >= segment.End || source[firstEnd] != '[' {
		return parser.UnresolvedReferenceUsage{}, anchor, false
	}
	second, secondEnd, ok := scanPlainReferenceUsageLabel(source, firstEnd, segment.End)
	if !ok {
		return parser.UnresolvedReferenceUsage{}, anchor, false
	}
	form := parser.LinkUsageFull
	reference := second
	if second == "" {
		form = parser.LinkUsageCollapsed
		reference = first
	}
	if reference == "" || len(reference) > 999 {
		return parser.UnresolvedReferenceUsage{}, anchor, false
	}
	if _, exists := definitions[ReferenceLabelKey(reference)]; exists {
		return parser.UnresolvedReferenceUsage{}, anchor, false
	}
	return parser.UnresolvedReferenceUsage{Kind: kind, Form: form, Anchor: anchor, Reference: reference}, secondEnd, true
}

func scanPlainReferenceUsageLabel(source []byte, open, limit int) (string, int, bool) {
	if open >= limit || source[open] != '[' {
		return "", open, false
	}
	for position := open + 1; position < limit; position++ {
		switch source[position] {
		case '\r', '\n', '[', '\\':
			return "", open, false
		case ']':
			label := string(source[open+1 : position])
			if label != "" && strings.TrimSpace(label) == "" {
				return "", open, false
			}
			return label, position + 1, true
		}
	}
	return "", open, false
}

func activateCompositeInlines(composites []compositeInline, owners []inlineSpan, block inlineBlock) {
	linkStarts := newInlineStartIndex(len(block.segments))
	autoLinkStarts := newInlineStartIndex(len(block.segments))
	for _, composite := range composites {
		if composite.kind == parser.KindInlineLink {
			linkStarts.add(composite.segment, composite.start)
		}
	}
	for _, owner := range owners {
		if owner.node.Kind == parser.KindAutoLink {
			autoLinkStarts.add(owner.segment, owner.start)
		}
	}
	linkStarts.finalize()
	autoLinkStarts.finalize()
	for index := range composites {
		composite := composites[index]
		composites[index].active = composite.kind == parser.KindImage ||
			!inlineStartsInCompositeLabel(linkStarts, block, composite) &&
				!inlineStartsInCompositeLabel(autoLinkStarts, block, composite)
	}
	suppressExplicitReferenceTailComposites(composites)
}

func inlineStartsInCompositeLabel(index inlineStartIndex, block inlineBlock, composite compositeInline) bool {
	if composite.segment < 0 || composite.labelEndSegment < composite.segment || composite.labelEndSegment >= len(block.segments) {
		return false
	}
	for segmentIndex := composite.segment; segmentIndex <= composite.labelEndSegment; segmentIndex++ {
		segment := block.segments[segmentIndex]
		start, end := segment.Start, segment.End
		if segmentIndex == composite.segment {
			start = composite.label.Start
		}
		if segmentIndex == composite.labelEndSegment {
			end = composite.label.End
		}
		if index.anyIn(segmentIndex, start, end) {
			return true
		}
	}
	return false
}

func suppressExplicitReferenceTailComposites(composites []compositeInline) {
	delta := make([]int, len(composites)+1)
	activeSuppressions := 0
	for index := range composites {
		activeSuppressions += delta[index]
		if activeSuppressions > 0 {
			composites[index].active = false
			continue
		}
		parent := composites[index]
		if !parent.active || parent.form != parser.LinkUsageFull && parent.form != parser.LinkUsageCollapsed {
			continue
		}
		first := sort.Search(len(composites), func(candidate int) bool {
			return composites[candidate].start > parent.label.End
		})
		last := sort.Search(len(composites), func(candidate int) bool {
			return composites[candidate].start >= parent.end
		})
		first = max(first, index+1)
		if first < last {
			delta[first]++
			delta[last]--
		}
	}
}

func scanInlineLabelEnd(source []byte, segment parser.Range, start int, exclusions []parser.Range) (int, bool) {
	depth := 0
	exclusionIndex := 0
	for position := start; position < segment.End; position++ {
		for exclusionIndex < len(exclusions) && position >= exclusions[exclusionIndex].End {
			exclusionIndex++
		}
		if exclusionIndex < len(exclusions) && position >= exclusions[exclusionIndex].Start {
			position = exclusions[exclusionIndex].End - 1
			continue
		}
		if inlineByteEscaped(source, segment.Start, position) {
			continue
		}
		switch source[position] {
		case '[':
			depth++
		case ']':
			if depth == 0 {
				return position, true
			}
			depth--
		}
	}
	return start, false
}

func scanInlineDestinationSyntax(source []byte, start, limit int) (inlineDestinationTail, bool) {
	if start >= limit || source[start] != '(' {
		return inlineDestinationTail{}, false
	}
	position := skipInlineLinkSpace(source, start+1, limit)
	if position < limit && source[position] == ')' {
		return inlineDestinationTail{end: position + 1}, true
	}
	if position >= limit {
		return inlineDestinationTail{}, false
	}
	destination, next, ok := scanInlineLinkDestination(source, position, limit)
	if !ok {
		return inlineDestinationTail{}, false
	}
	beforeSpace := next
	position = skipInlineLinkSpace(source, next, limit)
	if position < limit && source[position] == ')' {
		return inlineDestinationTail{end: position + 1, destination: destination}, true
	}
	if position == beforeSpace {
		return inlineDestinationTail{}, false
	}
	title, next, ok := scanInlineLinkTitleValue(source, position, limit)
	if !ok {
		return inlineDestinationTail{}, false
	}
	position = skipInlineLinkSpace(source, next, limit)
	if position >= limit || source[position] != ')' {
		return inlineDestinationTail{}, false
	}
	return inlineDestinationTail{end: position + 1, destination: destination, title: title, hasTitle: true}, true
}

func scanInlineDestinationAcrossSegments(source []byte, block inlineBlock, segmentIndex, start int) (inlineDestinationTail, bool) {
	if segmentIndex < 0 || segmentIndex >= len(block.segments) || start >= block.segments[segmentIndex].End || source[start] != '(' {
		return inlineDestinationTail{}, false
	}
	cursor := inlineHTMLCursor{block: block, segment: segmentIndex, position: start + 1}
	skipInlineLinkCursorSpace(source, &cursor)
	if value, ok := cursor.peek(source); ok && value == ')' {
		cursor.advance()
		return inlineDestinationTail{endSegment: cursor.segment, end: cursor.position}, true
	}
	destination, ok := scanInlineLinkCursorDestination(source, &cursor)
	if !ok {
		return inlineDestinationTail{}, false
	}
	space := skipInlineLinkCursorSpace(source, &cursor)
	if value, ok := cursor.peek(source); ok && value == ')' {
		cursor.advance()
		return inlineDestinationTail{endSegment: cursor.segment, end: cursor.position, destination: destination}, true
	}
	if !space {
		return inlineDestinationTail{}, false
	}
	title, ok := scanInlineLinkCursorTitle(source, &cursor)
	if !ok {
		return inlineDestinationTail{}, false
	}
	skipInlineLinkCursorSpace(source, &cursor)
	value, ok := cursor.peek(source)
	if !ok || value != ')' {
		return inlineDestinationTail{}, false
	}
	cursor.advance()
	return inlineDestinationTail{endSegment: cursor.segment, end: cursor.position, destination: destination, title: title, hasTitle: true}, true
}

func skipInlineLinkCursorSpace(source []byte, cursor *inlineHTMLCursor) bool {
	consumed := false
	for {
		value, ok := cursor.peek(source)
		if !ok || value != ' ' && value != '\t' && value != '\n' && value != '\r' {
			return consumed
		}
		consumed = true
		cursor.advance()
	}
}

func scanInlineLinkCursorDestination(source []byte, cursor *inlineHTMLCursor) (string, bool) {
	value, ok := cursor.peek(source)
	if !ok {
		return "", false
	}
	if value == '<' {
		return scanInlineAngleCursorDestination(source, cursor)
	}
	return scanInlineRawCursorDestination(source, cursor)
}

func scanInlineAngleCursorDestination(source []byte, cursor *inlineHTMLCursor) (string, bool) {
	cursor.advance()
	value := make([]byte, 0)
	for {
		current, ok := cursor.peek(source)
		if !ok || current == '\n' || current == '\r' || current == '<' {
			return "", false
		}
		if current == '>' {
			cursor.advance()
			return string(value), true
		}
		value = append(value, current)
		cursor.advance()
	}
}

func scanInlineRawCursorDestination(source []byte, cursor *inlineHTMLCursor) (string, bool) {
	value := make([]byte, 0)
	depth := 0
	for {
		current, ok := cursor.peek(source)
		if !ok || rawCursorDestinationBoundary(current, depth) {
			return string(value), len(value) != 0
		}
		if current == '\\' {
			var escapedOK bool
			value, escapedOK = appendInlineCursorEscape(source, cursor, value)
			if !escapedOK {
				return "", false
			}
			continue
		}
		var depthOK bool
		depth, depthOK = rawCursorDestinationDepth(depth, current)
		if !depthOK {
			return "", false
		}
		value = append(value, current)
		cursor.advance()
	}
}

func rawCursorDestinationBoundary(current byte, depth int) bool {
	switch current {
	case ' ', '\t', '\n', '\r', '<':
		return true
	case ')':
		return depth == 0
	default:
		return false
	}
}

func appendInlineCursorEscape(source []byte, cursor *inlineHTMLCursor, value []byte) ([]byte, bool) {
	value = append(value, '\\')
	cursor.advance()
	escaped, ok := cursor.peek(source)
	if !ok || escaped == '\n' || escaped == '\r' {
		return value, false
	}
	value = append(value, escaped)
	cursor.advance()
	return value, true
}

func rawCursorDestinationDepth(depth int, current byte) (int, bool) {
	switch current {
	case '(':
		depth++
		return depth, depth <= 32
	case ')':
		return depth - 1, true
	default:
		return depth, true
	}
}

func scanInlineLinkCursorTitle(source []byte, cursor *inlineHTMLCursor) (string, bool) {
	opening, ok := cursor.peek(source)
	if !ok || opening != '\'' && opening != '"' && opening != '(' {
		return "", false
	}
	closing := opening
	if opening == '(' {
		closing = ')'
	}
	cursor.advance()
	value := make([]byte, 0)
	for {
		current, ok := cursor.peek(source)
		if !ok {
			return "", false
		}
		if current == closing {
			cursor.advance()
			return string(value), true
		}
		if current == '\\' {
			value = append(value, current)
			cursor.advance()
			escaped, ok := cursor.peek(source)
			if !ok {
				return "", false
			}
			value = append(value, escaped)
			cursor.advance()
			continue
		}
		value = append(value, current)
		cursor.advance()
	}
}

func scanInlineLinkDestination(source []byte, start, limit int) (string, int, bool) {
	if source[start] == '<' {
		end, ok := scanAngleLinkDestination(source, start, limit)
		if !ok {
			return "", start, false
		}
		return string(source[start+1 : end-1]), end, true
	}
	end, ok := scanRawLinkDestination(source, start, limit)
	if !ok || end == start {
		return "", start, false
	}
	return string(source[start:end]), end, true
}

func scanAngleLinkDestination(source []byte, start, limit int) (int, bool) {
	for position := start + 1; position < limit; position++ {
		if source[position] == '\n' || source[position] == '\r' || source[position] == '<' {
			return start, false
		}
		if source[position] == '>' && !inlineByteEscaped(source, start+1, position) {
			return position + 1, true
		}
	}
	return start, false
}

func scanRawLinkDestination(source []byte, start, limit int) (int, bool) {
	position := start
	depth := 0
	for position < limit {
		value := source[position]
		if value == '\\' && position+1 < limit {
			position += 2
			continue
		}
		switch value {
		case '(':
			depth++
			if depth > 32 {
				return start, false
			}
		case ')':
			if depth == 0 {
				return position, true
			}
			depth--
		case ' ', '\t', '\n', '\r', '<':
			return position, true
		}
		position++
	}
	return position, true
}

func scanInlineLinkTitleValue(source []byte, start, limit int) (string, int, bool) {
	if start >= limit {
		return "", start, false
	}
	opening := source[start]
	closing := opening
	if opening == '(' {
		closing = ')'
	} else if opening != '\'' && opening != '"' {
		return "", start, false
	}
	for position := start + 1; position < limit; position++ {
		if source[position] == '\\' && position+1 < limit {
			position++
			continue
		}
		if source[position] == closing {
			return string(source[start+1 : position]), position + 1, true
		}
		if source[position] == '\n' || source[position] == '\r' {
			return "", start, false
		}
	}
	return "", start, false
}

func skipInlineLinkSpace(source []byte, start, limit int) int {
	for start < limit && (source[start] == ' ' || source[start] == '\t') {
		start++
	}
	return start
}

func projectCompositeObservations(source []byte, block inlineBlock, owners []inlineSpan, composites []compositeInline, matches []delimiterMatch) []parser.Node {
	ownerStarts := newInlineStartIndex(len(block.segments))
	childStarts := newInlineStartIndex(len(block.segments))
	delimiterIntervals := newInlineIntervalIndex(len(block.segments))
	for _, owner := range owners {
		ownerStarts.add(owner.segment, owner.start)
	}
	for _, composite := range composites {
		if composite.active {
			childStarts.add(composite.segment, composite.start)
		}
	}
	for _, match := range matches {
		if match.startSegment == match.endSegment {
			delimiterIntervals.add(match.startSegment, match.syntaxStart, match.syntaxEnd)
		}
	}
	ownerStarts.finalize()
	childStarts.finalize()
	delimiterIntervals.finalize()

	nodes := make([]parser.Node, 0, len(composites))
	for _, composite := range composites {
		if !simpleCompositeInline(source, block, composite, ownerStarts, childStarts, delimiterIntervals) {
			continue
		}
		node := parser.Node{Kind: composite.kind, Range: composite.label, Anchor: composite.start}
		if composite.kind == parser.KindInlineLink {
			if composite.form != parser.LinkUsageDirect {
				continue
			}
			node.Destination = composite.destination
			node.Title = composite.title
			node.HasTitle = composite.hasTitle
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func simpleCompositeInline(source []byte, block inlineBlock, composite compositeInline, ownerStarts, childStarts inlineStartIndex, delimiterIntervals inlineIntervalIndex) bool {
	if !composite.active || composite.label.Start >= composite.label.End || composite.segment < 0 || composite.segment >= len(block.segments) || composite.labelEndSegment != composite.segment {
		return false
	}
	segment := block.segments[composite.segment]
	if composite.label.Start < segment.Start || composite.label.End > segment.End || labelHasUnescapedBracket(source, segment, composite.label) || labelHasUnresolvedOpener(source, segment, composite.label) {
		return false
	}
	if ownerStarts.anyIn(composite.segment, composite.label.Start, composite.label.End) || childStarts.anyIn(composite.segment, composite.label.Start, composite.label.End) {
		return false
	}
	return !delimiterIntervals.anyContained(composite.segment, composite.label.Start, composite.label.End)
}

func labelHasUnescapedBracket(source []byte, segment parser.Range, label parser.Range) bool {
	for position := label.Start; position < label.End; position++ {
		if (source[position] == '[' || source[position] == ']') && !inlineByteEscaped(source, segment.Start, position) {
			return true
		}
	}
	return false
}

func labelHasUnresolvedOpener(source []byte, segment, label parser.Range) bool {
	for position := label.Start; position < label.End; {
		marker := source[position]
		if marker != '*' && marker != '_' && marker != '~' || inlineByteEscaped(source, segment.Start, position) {
			position++
			continue
		}
		start := position
		for position < label.End && source[position] == marker {
			position++
		}
		if marker == '~' && position-start > 2 {
			continue
		}
		canOpen, _ := delimiterFlanking(source, segment, start, position, marker)
		if canOpen {
			return true
		}
	}
	return false
}
