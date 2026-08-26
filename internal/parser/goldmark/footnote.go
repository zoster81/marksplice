package goldmark

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extensionast "github.com/yuin/goldmark/extension/ast"
	goldmarkparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/zoster81/marksplice/internal/parser"
)

var footnoteObservationKey = goldmarkparser.NewContextKey()

type footnoteCollector struct{}

type footnoteCollection struct {
	definitions []collectedFootnoteDefinition
	references  []rawFootnoteReference
	linkUsages  []parser.LinkUsage
	err         error
}

type collectedFootnoteDefinition struct {
	observation parser.FootnoteDefinitionObservation
	index       int
}

type rawFootnoteReference struct {
	index int
	pos   int
}

type indexedFootnoteDefinition struct {
	anchor int
	label  string
}

func newFootnoteMarkdown() goldmark.Markdown {
	return newMarkdownWithExtensions(
		[]goldmark.Extender{extension.GFM, extension.Footnote},
		goldmark.WithParserOptions(
			goldmarkparser.WithASTTransformers(
				util.Prioritized(&footnoteCollector{}, 998),
			),
		),
	)
}

// ParseDocument returns the complete parser-independent semantic observation set.
// The normative GFM parse remains authoritative for GFM syntax. Footnotes are
// observed by an isolated second parser pass before Goldmark reorders/removes its
// footnote AST nodes; no Goldmark AST or parser context survives this call.
func (a *Adapter) ParseDocument(source []byte) (parser.DocumentObservations, error) {
	nodes, usages, unresolved, mathExpressions, err := a.parseGFM(source)
	if err != nil {
		return parser.DocumentObservations{}, err
	}
	definitions, references, footnoteUsages, err := a.parseFootnotes(source)
	if err != nil {
		return parser.DocumentObservations{}, err
	}
	nodes, usages = removeFootnoteGFMConflicts(source, nodes, usages, definitions)
	unresolved = removeFootnoteUnresolvedConflicts(source, unresolved, definitions)
	usages = mergeFootnoteLinkUsages(usages, footnoteUsages)
	return parser.DocumentObservations{
		Nodes:                     nodes,
		LinkUsages:                usages,
		UnresolvedReferenceUsages: unresolved,
		FootnoteDefinitions:       definitions,
		FootnoteReferences:        references,
		MathExpressions:           mathExpressions,
	}, nil
}

func (a *Adapter) parseFootnotes(source []byte) ([]parser.FootnoteDefinitionObservation, []parser.FootnoteReferenceObservation, []parser.LinkUsage, error) {
	parseSource := normalizeIsolatedCR(source)
	context := goldmarkparser.NewContext()
	_ = a.footnotes.Parser().Parse(text.NewReader(parseSource), goldmarkparser.WithContext(context))
	value := context.Get(footnoteObservationKey)
	if value == nil {
		return nil, nil, nil, nil
	}
	collection, ok := value.(footnoteCollection)
	if !ok {
		return nil, nil, nil, fmt.Errorf("collect footnote semantics: invalid parser context value")
	}
	if collection.err != nil {
		return nil, nil, nil, fmt.Errorf("collect footnote semantics: %w", collection.err)
	}

	definitions := make([]parser.FootnoteDefinitionObservation, 0, len(collection.definitions))
	indexed := make(map[int]indexedFootnoteDefinition, len(collection.definitions))
	for _, definition := range collection.definitions {
		if !footnoteDefinitionTopLevel(source, definition.observation.Anchor) {
			continue
		}
		definitions = append(definitions, definition.observation)
		if definition.index > 0 {
			indexed[definition.index] = indexedFootnoteDefinition{
				anchor: definition.observation.Anchor,
				label:  definition.observation.Label,
			}
		}
	}
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Anchor < definitions[j].Anchor })

	references := make([]parser.FootnoteReferenceObservation, 0, len(collection.references))
	for _, raw := range collection.references {
		definition, ok := indexed[raw.index]
		if !ok {
			continue
		}
		range_, labelRange, ok := footnoteReferenceRanges(source, raw.pos, definition.label)
		if !ok {
			continue
		}
		references = append(references, parser.FootnoteReferenceObservation{
			Range:            range_,
			LabelRange:       labelRange,
			Label:            definition.label,
			DefinitionAnchor: definition.anchor,
		})
	}
	sort.Slice(references, func(i, j int) bool { return references[i].Range.Start < references[j].Range.Start })
	occurrences := make(map[int]int, len(definitions))
	for index := range references {
		anchor := references[index].DefinitionAnchor
		references[index].Occurrence = occurrences[anchor]
		occurrences[anchor]++
	}
	claims := footnoteClaimedRanges(source, definitions)
	linkUsages := make([]parser.LinkUsage, 0, len(collection.linkUsages))
	for _, usage := range collection.linkUsages {
		if offsetInsideAny(usage.Anchor, claims) {
			linkUsages = append(linkUsages, usage)
		}
	}
	sort.SliceStable(linkUsages, func(i, j int) bool { return linkUsages[i].Anchor < linkUsages[j].Anchor })
	return definitions, references, linkUsages, nil
}

func footnoteDefinitionTopLevel(source []byte, anchor int) bool {
	if anchor < 0 || anchor >= len(source) {
		return false
	}
	start := footnotePhysicalLineStart(source, anchor)
	if anchor-start > 3 {
		return false
	}
	for _, value := range source[start:anchor] {
		if value != ' ' {
			return false
		}
	}
	return true
}

func (c *footnoteCollector) Transform(root *ast.Document, reader text.Reader, context goldmarkparser.Context) {
	collection := footnoteCollection{}
	source := reader.Source()
	footnoteDepth := 0

	err := ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			if _, ok := node.(*extensionast.Footnote); ok {
				footnoteDepth--
			}
			return ast.WalkContinue, nil
		}
		switch typed := node.(type) {
		case *extensionast.Footnote:
			footnoteDepth++
			anchor := typed.Pos()
			if anchor < 0 || anchor >= len(source) {
				return ast.WalkStop, fmt.Errorf("footnote definition anchor %d is outside source length %d", anchor, len(source))
			}
			definition := parser.FootnoteDefinitionObservation{
				Anchor:     anchor,
				Label:      string(typed.Ref),
				BodyRanges: footnoteBodyRanges(source, typed),
			}
			collection.definitions = append(collection.definitions, collectedFootnoteDefinition{
				observation: definition,
				index:       typed.Index,
			})
		case *extensionast.FootnoteLink:
			collection.references = append(collection.references, rawFootnoteReference{index: typed.Index, pos: typed.Pos()})
		default:
			if footnoteDepth > 0 {
				if usage, ok := linkUsage(source, node); ok {
					collection.linkUsages = append(collection.linkUsages, usage)
				}
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		collection.err = err
	}
	context.Set(footnoteObservationKey, collection)
}

func footnoteBodyRanges(source []byte, definition *extensionast.Footnote) []parser.Range {
	seen := make(map[parser.Range]struct{})
	ranges := make([]parser.Range, 0)
	_ = ast.Walk(definition, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || node == definition || node.Type() != ast.TypeBlock {
			return ast.WalkContinue, nil
		}
		lines := node.Lines()
		if lines == nil {
			return ast.WalkContinue, nil
		}
		for index := 0; index < lines.Len(); index++ {
			segment := lines.At(index)
			range_ := parser.Range{Start: segment.Start, End: segment.Stop}
			if !range_.Valid(len(source)) || range_.Start == range_.End {
				continue
			}
			if _, exists := seen[range_]; exists {
				continue
			}
			seen[range_] = struct{}{}
			ranges = append(ranges, range_)
		}
		return ast.WalkContinue, nil
	})
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].Start != ranges[j].Start {
			return ranges[i].Start < ranges[j].Start
		}
		return ranges[i].End < ranges[j].End
	})
	return ranges
}

func footnoteReferenceRanges(source []byte, pos int, label string) (parser.Range, parser.Range, bool) {
	token := []byte("[^" + label + "]")
	for _, start := range []int{pos, pos + 1} {
		if start < 0 || start+len(token) > len(source) || !bytes.Equal(source[start:start+len(token)], token) {
			continue
		}
		return parser.Range{Start: start, End: start + len(token)},
			parser.Range{Start: start + 2, End: start + 2 + len(label)}, true
	}
	return parser.Range{}, parser.Range{}, false
}

func removeFootnoteGFMConflicts(source []byte, nodes []parser.Node, usages []parser.LinkUsage, definitions []parser.FootnoteDefinitionObservation) ([]parser.Node, []parser.LinkUsage) {
	if len(definitions) == 0 {
		return nodes, usages
	}
	anchors := make(map[int]struct{}, len(definitions))
	caretKeys := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		anchors[definition.Anchor] = struct{}{}
		caretKeys[ReferenceLabelKey("^"+definition.Label)] = struct{}{}
	}
	claims := footnoteClaimedRanges(source, definitions)
	filteredNodes := filterFootnoteConflictNodes(nodes, claims, anchors)
	filteredUsages, suppressedUsageAnchors := filterFootnoteConflictUsages(usages, claims, caretKeys)
	return removeSuppressedReferenceNodes(filteredNodes, suppressedUsageAnchors), filteredUsages
}

func filterFootnoteConflictNodes(nodes []parser.Node, claims []parser.Range, anchors map[int]struct{}) []parser.Node {
	filtered := nodes[:0]
	for _, node := range nodes {
		insideClaim := rangeInsideAny(node.Range, claims)
		_, claimedAnchor := anchors[node.Range.Start]
		if insideClaim || node.Kind == parser.KindReferenceDefinition && claimedAnchor {
			continue
		}
		filtered = append(filtered, node)
	}
	clear(nodes[len(filtered):])
	return filtered
}

func filterFootnoteConflictUsages(usages []parser.LinkUsage, claims []parser.Range, caretKeys map[string]struct{}) ([]parser.LinkUsage, map[linkUsageNodeKey]struct{}) {
	filtered := make([]parser.LinkUsage, 0, len(usages))
	suppressedAnchors := make(map[linkUsageNodeKey]struct{})
	for _, usage := range usages {
		if offsetInsideAny(usage.Anchor, claims) {
			continue
		}
		if usage.Form != parser.LinkUsageDirect {
			if _, suppressed := caretKeys[ReferenceLabelKey(usage.Reference)]; suppressed {
				suppressedAnchors[linkUsageNodeKey{kind: usage.Kind, anchor: usage.Anchor}] = struct{}{}
				continue
			}
		}
		filtered = append(filtered, usage)
	}
	return filtered, suppressedAnchors
}

func removeSuppressedReferenceNodes(nodes []parser.Node, suppressed map[linkUsageNodeKey]struct{}) []parser.Node {
	if len(suppressed) == 0 {
		return nodes
	}
	result := nodes[:0]
	for _, node := range nodes {
		if _, remove := suppressed[linkUsageNodeKey{kind: node.Kind, anchor: node.Anchor}]; !remove {
			result = append(result, node)
		}
	}
	clear(nodes[len(result):])
	return result
}

func removeFootnoteUnresolvedConflicts(source []byte, unresolved []parser.UnresolvedReferenceUsage, definitions []parser.FootnoteDefinitionObservation) []parser.UnresolvedReferenceUsage {
	claims := footnoteClaimedRanges(source, definitions)
	if len(claims) == 0 {
		return unresolved
	}
	result := make([]parser.UnresolvedReferenceUsage, 0, len(unresolved))
	for _, usage := range unresolved {
		if !offsetInsideAny(usage.Anchor, claims) {
			result = append(result, usage)
		}
	}
	return result
}

func footnoteClaimedRanges(source []byte, definitions []parser.FootnoteDefinitionObservation) []parser.Range {
	claims := make([]parser.Range, 0, len(definitions))
	for _, definition := range definitions {
		if definition.Anchor < 0 || definition.Anchor >= len(source) {
			continue
		}
		start := footnotePhysicalLineStart(source, definition.Anchor)
		end := footnotePhysicalLineRangeEnd(source, definition.Anchor)
		for _, body := range definition.BodyRanges {
			if body.Valid(len(source)) && body.End > body.Start {
				if candidate := footnotePhysicalLineRangeEnd(source, body.End); candidate > end {
					end = candidate
				}
			}
		}
		if end > start {
			claims = append(claims, parser.Range{Start: start, End: end})
		}
	}
	return normalizeFootnoteClaims(claims)
}

func normalizeFootnoteClaims(claims []parser.Range) []parser.Range {
	if len(claims) < 2 {
		return claims
	}
	sort.Slice(claims, func(i, j int) bool {
		if claims[i].Start != claims[j].Start {
			return claims[i].Start < claims[j].Start
		}
		return claims[i].End < claims[j].End
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

func footnotePhysicalLineStart(source []byte, offset int) int {
	if offset > len(source) {
		offset = len(source)
	}
	for offset > 0 && source[offset-1] != '\n' && source[offset-1] != '\r' {
		offset--
	}
	return offset
}

func footnotePhysicalLineRangeEnd(source []byte, offset int) int {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source) {
		offset = len(source)
	}
	for offset < len(source) && source[offset] != '\n' && source[offset] != '\r' {
		offset++
	}
	if offset < len(source) && source[offset] == '\r' {
		offset++
		if offset < len(source) && source[offset] == '\n' {
			offset++
		}
	} else if offset < len(source) && source[offset] == '\n' {
		offset++
	}
	return offset
}

func rangeInsideAny(range_ parser.Range, claims []parser.Range) bool {
	if range_.Start < 0 || range_.End < range_.Start {
		return false
	}
	index := sort.Search(len(claims), func(index int) bool { return claims[index].End > range_.Start })
	return index < len(claims) && range_.Start >= claims[index].Start && range_.End <= claims[index].End
}

func offsetInsideAny(offset int, claims []parser.Range) bool {
	index := sort.Search(len(claims), func(index int) bool { return claims[index].End > offset })
	return index < len(claims) && offset >= claims[index].Start
}

func mergeFootnoteLinkUsages(gfm, footnotes []parser.LinkUsage) []parser.LinkUsage {
	if len(footnotes) == 0 {
		return gfm
	}
	merged := make([]parser.LinkUsage, 0, len(gfm)+len(footnotes))
	merged = append(merged, gfm...)
	merged = append(merged, footnotes...)
	sort.SliceStable(merged, func(i, j int) bool { return merged[i].Anchor < merged[j].Anchor })
	result := merged[:0]
	for _, usage := range merged {
		if len(result) != 0 && result[len(result)-1] == usage {
			continue
		}
		result = append(result, usage)
	}
	return result
}

type linkUsageNodeKey struct {
	kind   parser.Kind
	anchor int
}
