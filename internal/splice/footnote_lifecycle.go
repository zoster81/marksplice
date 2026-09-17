package splice

import (
	"bytes"
	"unicode/utf8"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

type footnoteBodyLayout struct {
	leading            []byte
	continuationPrefix []byte
	eol                []byte
	suffix             []byte
	firstOnOpening     bool
}

// PrepareReplaceFootnoteDefinitionBodyMultiline replaces the complete logical body
// of one source-proven footnote definition. replacement uses LF as its logical line
// separator; Marksplice renders the source with the definition's proven EOL and
// continuation-prefix style.
func (d *Document) PrepareReplaceFootnoteDefinitionBodyMultiline(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindFootnoteDefinition, "footnote definition")
	if err != nil {
		return ChangeSet{}, err
	}
	lines, err := logicalFootnoteBodyLines(replacement)
	if err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := d.footnoteSource(target)
	if !ok || len(mapping.BodyRanges) == 0 {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	layout, ok := sourceFootnoteBodyLayout(d.source, mapping)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	fragment := renderFootnoteBody(layout, lines)
	if len(fragment) == 0 || bytes.Equal(fragment, d.source[mapping.Range.Start:mapping.Range.End]) {
		if bytes.Equal(fragment, d.source[mapping.Range.Start:mapping.Range.End]) {
			return d.newChanges(nil, "multiline footnote body replacement")
		}
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(mapping.Range, fragment, "multiline footnote body replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateMultilineFootnoteBodyCandidate(candidate, target, mapping, len(fragment)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareAppendFootnoteDefinition appends one canonical top-level footnote definition.
// body uses LF as its logical line separator; continuation lines are rendered with
// canonical four-space indentation and the document's proven EOL style.
func (d *Document) PrepareAppendFootnoteDefinition(label, body []byte) (ChangeSet, error) {
	if d == nil || d.footnoteLabelExists(string(label)) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if err := validateFootnoteLabelReplacement(label); err != nil {
		return ChangeSet{}, err
	}
	lines, err := logicalFootnoteBodyLines(body)
	if err != nil {
		return ChangeSet{}, err
	}
	prefix, eol, ok := d.referenceDefinitionAppendLayout()
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	definition, ok := canonicalFootnoteDefinition(label, lines, eol)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	replacement := make([]byte, 0, len(prefix)+len(definition))
	replacement = append(replacement, prefix...)
	replacement = append(replacement, definition...)
	patch := Range{Start: len(d.source), End: len(d.source)}
	change, candidate, err := d.prepareCandidateChange(patch, replacement, "footnote definition append")
	if err != nil {
		return ChangeSet{}, err
	}
	definitionOffset := len(d.source) + len(prefix)
	if err := d.validateAppendedFootnoteCandidate(candidate, string(label), definitionOffset, len(definition)); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareRemoveFootnoteDefinition removes one complete source-proven footnote
// definition while leaving all source occurrences outside the owned container intact.
func (d *Document) PrepareRemoveFootnoteDefinition(id NodeID) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindFootnoteDefinition, "footnote definition")
	if err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := d.footnoteSource(target)
	if !ok || !mapping.Range.Valid(len(d.source)) || mapping.Range.Start >= mapping.Range.End {
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(mapping.Range, nil, "footnote definition removal")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateNodeSurvivorsAfterRemoval(candidate, mapping.Range); err != nil {
		return ChangeSet{}, err
	}
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	transform := []patchTransform{{Range: mapping.Range}}
	if !d.footnoteDefinitionsSurviveRemoval(candidateDocument, target, transform) ||
		!d.footnoteReferencesSurviveDefinitionRemoval(candidateDocument, target, mapping.Range, transform) ||
		!sameUnresolvedReferenceUsagesAfterPatches(d.unresolvedReferenceUsages, candidateDocument.unresolvedReferenceUsages, transform) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return change, nil
}

func logicalFootnoteBodyLines(body []byte) ([][]byte, error) {
	if len(body) == 0 || !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0 || bytes.IndexByte(body, '\r') >= 0 {
		return nil, ErrInvalidReplacement
	}
	lines := bytes.Split(body, []byte{'\n'})
	if len(lines[0]) == 0 || len(lines[len(lines)-1]) == 0 {
		return nil, ErrInvalidReplacement
	}
	result := make([][]byte, len(lines))
	for index := range lines {
		result[index] = append([]byte(nil), lines[index]...)
	}
	return result, nil
}

func sourceFootnoteBodyLayout(input []byte, mapping source.FootnoteDefinitionMapping) (footnoteBodyLayout, bool) {
	if !mapping.Range.Valid(len(input)) || mapping.Range.Start >= mapping.Range.End || len(mapping.BodyRanges) == 0 {
		return footnoteBodyLayout{}, false
	}
	first := mapping.BodyRanges[0]
	last := mapping.BodyRanges[len(mapping.BodyRanges)-1]
	if !first.Valid(len(input)) || !last.Valid(len(input)) || first.Start >= first.End || last.Start >= last.End {
		return footnoteBodyLayout{}, false
	}
	eol, ok := uniformFootnoteLineEnding(input, mapping.Range)
	if !ok {
		return footnoteBodyLayout{}, false
	}
	openingStart := mapping.Range.Start
	firstLineStart := physicalLineStartAt(input, first.Start)
	firstOnOpening := firstLineStart == openingStart
	continuationPrefix, ok := sourceFootnoteContinuationPrefix(input, mapping, openingStart)
	if !ok {
		return footnoteBodyLayout{}, false
	}
	layout := footnoteBodyLayout{
		continuationPrefix: continuationPrefix,
		eol:                eol,
		firstOnOpening:     firstOnOpening,
		suffix:             append([]byte(nil), input[last.End:mapping.Range.End]...),
	}
	if firstOnOpening {
		layout.leading = append([]byte(nil), input[mapping.Range.Start:first.Start]...)
		return layout, true
	}
	_, openingNext, openingEOL := physicalLine(input, mapping.Range.Start)
	if openingNext <= mapping.Range.Start || openingNext > firstLineStart || openingEOL == "" {
		return footnoteBodyLayout{}, false
	}
	layout.leading = append([]byte(nil), input[mapping.Range.Start:openingNext]...)
	return layout, true
}

func uniformFootnoteLineEnding(input []byte, owned Range) ([]byte, bool) {
	var eol []byte
	for start := owned.Start; start < owned.End; {
		_, next, lineEOL := physicalLine(input, start)
		if lineEOL != "" {
			current := []byte(lineEOL)
			if len(eol) == 0 {
				eol = append([]byte(nil), current...)
			} else if !bytes.Equal(eol, current) {
				return nil, false
			}
		}
		if next <= start || next > owned.End {
			break
		}
		start = next
	}
	return eol, len(eol) != 0
}

func sourceFootnoteContinuationPrefix(input []byte, mapping source.FootnoteDefinitionMapping, openingStart int) ([]byte, bool) {
	var prefix []byte
	for _, body := range mapping.BodyRanges {
		lineStart := physicalLineStartAt(input, body.Start)
		if lineStart == openingStart {
			continue
		}
		candidate := input[lineStart:body.Start]
		if !validFootnoteContinuationPrefix(candidate) {
			return nil, false
		}
		if prefix == nil {
			prefix = append([]byte(nil), candidate...)
		} else if !bytes.Equal(prefix, candidate) {
			return nil, false
		}
	}
	if prefix == nil {
		prefix = []byte("    ")
	}
	return prefix, true
}

func validFootnoteContinuationPrefix(prefix []byte) bool {
	if len(prefix) == 0 {
		return false
	}
	column := 0
	for _, value := range prefix {
		switch value {
		case ' ':
			column++
		case '\t':
			column += 4 - column%4
		default:
			return false
		}
	}
	return column >= 4
}

func physicalLineStartAt(input []byte, offset int) int {
	if offset < 0 {
		return 0
	}
	if offset > len(input) {
		offset = len(input)
	}
	for offset > 0 && input[offset-1] != '\n' && input[offset-1] != '\r' {
		offset--
	}
	return offset
}

func renderFootnoteBody(layout footnoteBodyLayout, lines [][]byte) []byte {
	capacity := len(layout.leading) + len(layout.suffix)
	for _, line := range lines {
		capacity += len(line) + len(layout.eol) + len(layout.continuationPrefix)
	}
	fragment := make([]byte, 0, capacity)
	fragment = append(fragment, layout.leading...)
	for index, line := range lines {
		if index > 0 {
			fragment = append(fragment, layout.eol...)
		}
		if len(line) == 0 {
			continue
		}
		if index > 0 || !layout.firstOnOpening {
			fragment = append(fragment, layout.continuationPrefix...)
		}
		fragment = append(fragment, line...)
	}
	fragment = append(fragment, layout.suffix...)
	return fragment
}

func (d *Document) validateMultilineFootnoteBodyCandidate(candidate []byte, target Node, original source.FootnoteDefinitionMapping, replacementLength int) error {
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ErrInvalidReplacement
	}
	transform := []patchTransform{{Range: original.Range, ReplacementLength: replacementLength}}
	originalDefinitions := footnoteDefinitionNodes(d)
	candidateDefinitions := footnoteDefinitionNodes(candidateDocument)
	if len(originalDefinitions) != len(candidateDefinitions) {
		return ErrInvalidReplacement
	}
	for index, originalNode := range originalDefinitions {
		if originalNode.ID == target.ID {
			if !replacementFootnoteDefinitionMatches(candidateDocument, candidateDefinitions[index], target, original, replacementLength) {
				return ErrInvalidReplacement
			}
			continue
		}
		if !sameShiftedFootnoteDefinition(d, candidateDocument, originalNode, candidateDefinitions[index], transform, originalNode.Label) {
			return ErrInvalidReplacement
		}
	}
	oldRange := original.Range
	newRange := Range{Start: original.Range.Start, End: original.Range.Start + replacementLength}
	if !sameFootnoteReferencesOutsideRanges(d, candidateDocument, oldRange, newRange, transform) ||
		!sameLinkUsagesOutsideRanges(d.linkUsages, candidateDocument.linkUsages, oldRange, newRange, transform) ||
		!sameUnresolvedReferencesOutsideRanges(d.unresolvedReferenceUsages, candidateDocument.unresolvedReferenceUsages, oldRange, newRange, transform) {
		return ErrInvalidReplacement
	}
	return nil
}

func replacementFootnoteDefinitionMatches(candidateDocument *Document, candidate, original Node, mapping source.FootnoteDefinitionMapping, replacementLength int) bool {
	candidateMapping, ok := candidateDocument.footnoteSource(candidate)
	if !ok || candidate.Kind != KindFootnoteDefinition || candidate.Label != original.Label || !candidate.Editable || !candidate.TopLevel {
		return false
	}
	expectedRange := Range{Start: mapping.Range.Start, End: mapping.Range.Start + replacementLength}
	return candidateMapping.Range == expectedRange && candidateMapping.LabelRange == mapping.LabelRange && len(candidateMapping.BodyRanges) != 0
}

func sameFootnoteReferencesOutsideRanges(original, candidate *Document, oldRange, newRange Range, transforms []patchTransform) bool {
	left := footnoteReferencesOutsideOwnedRange(original.footnoteReferences, oldRange)
	right := footnoteReferencesOutsideOwnedRange(candidate.footnoteReferences, newRange)
	if len(left) != len(right) {
		return false
	}
	leftDefinitions := footnoteDefinitionIndexes(original)
	rightDefinitions := footnoteDefinitionIndexes(candidate)
	for index := range left {
		if !sameShiftedFootnoteReference(left[index], right[index], left[index].Label, transforms, leftDefinitions, rightDefinitions) {
			return false
		}
	}
	return true
}

func sameLinkUsagesOutsideRanges(original, candidate []parser.LinkUsage, oldRange, newRange Range, transforms []patchTransform) bool {
	left := linkUsagesOutsideOwnedRange(original, oldRange)
	right := linkUsagesOutsideOwnedRange(candidate, newRange)
	return sameLinkUsagesAfterPatches(left, right, transforms)
}

func sameUnresolvedReferencesOutsideRanges(original, candidate []parser.UnresolvedReferenceUsage, oldRange, newRange Range, transforms []patchTransform) bool {
	left := unresolvedReferenceUsagesOutsideRange(original, oldRange)
	right := unresolvedReferenceUsagesOutsideRange(candidate, newRange)
	return sameUnresolvedReferenceUsagesAfterPatches(left, right, transforms)
}

func (d *Document) footnoteLabelExists(label string) bool {
	for _, node := range d.nodes {
		if node.Kind == KindFootnoteDefinition && node.Label == label {
			return true
		}
	}
	return false
}

func canonicalFootnoteDefinition(label []byte, lines [][]byte, eol []byte) ([]byte, bool) {
	if len(label) == 0 || len(lines) == 0 || len(eol) == 0 {
		return nil, false
	}
	fragment := make([]byte, 0)
	fragment = append(fragment, '[', '^')
	fragment = append(fragment, label...)
	fragment = append(fragment, ']', ':', ' ')
	fragment = append(fragment, lines[0]...)
	fragment = append(fragment, eol...)
	for _, line := range lines[1:] {
		if len(line) != 0 {
			fragment = append(fragment, ' ', ' ', ' ', ' ')
			fragment = append(fragment, line...)
		}
		fragment = append(fragment, eol...)
	}
	parsed, err := Parse(fragment)
	if err != nil {
		return nil, false
	}
	definitions := footnoteDefinitionNodes(parsed)
	if len(definitions) != 1 || definitions[0].Label != string(label) {
		return nil, false
	}
	mapping, ok := parsed.footnoteSource(definitions[0])
	return fragment, ok && mapping.Range == (Range{Start: 0, End: len(fragment)}) && len(mapping.BodyRanges) != 0
}

func (d *Document) validateAppendedFootnoteCandidate(candidate []byte, label string, definitionOffset, definitionLength int) error {
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ErrInvalidReplacement
	}
	originalDefinitions := footnoteDefinitionNodes(d)
	candidateDefinitions := footnoteDefinitionNodes(candidateDocument)
	if len(candidateDefinitions) != len(originalDefinitions)+1 {
		return ErrInvalidReplacement
	}
	for index := range originalDefinitions {
		if !sameShiftedFootnoteDefinition(d, candidateDocument, originalDefinitions[index], candidateDefinitions[index], nil, originalDefinitions[index].Label) {
			return ErrInvalidReplacement
		}
	}
	appended := candidateDefinitions[len(candidateDefinitions)-1]
	mapping, ok := candidateDocument.footnoteSource(appended)
	if !ok || appended.Label != label || mapping.Range != (Range{Start: definitionOffset, End: definitionOffset + definitionLength}) {
		return ErrInvalidReplacement
	}
	appendedRange := mapping.Range
	if !d.footnoteReferencesMatchAfterAppend(candidateDocument, appended, appendedRange) ||
		!sameLinkUsagesAfterPatches(d.linkUsages, linkUsagesOutsideOwnedRange(candidateDocument.linkUsages, appendedRange), nil) ||
		!sameUnresolvedReferenceUsagesAfterPatches(d.unresolvedReferenceUsages, unresolvedReferenceUsagesOutsideRange(candidateDocument.unresolvedReferenceUsages, appendedRange), nil) {
		return ErrInvalidReplacement
	}
	return nil
}

func (d *Document) footnoteReferencesMatchAfterAppend(candidate *Document, appended Node, appendedRange Range) bool {
	matchedOriginal := make([]bool, len(d.footnoteReferences))
	appendedDefinitions := footnoteDefinitionIndexes(candidate)
	appendedIndex, ok := appendedDefinitions[appended.ID]
	if !ok {
		return false
	}
	for _, reference := range candidate.footnoteReferences {
		if reference.Range.Start >= appendedRange.Start && reference.Range.Start < appendedRange.End {
			continue
		}
		if matchOriginalFootnoteReference(d.footnoteReferences, reference, matchedOriginal) {
			continue
		}
		if reference.Label != appended.Label || !reference.HasDefinition || appendedDefinitions[reference.DefinitionID] != appendedIndex ||
			!reference.Range.Valid(len(d.source)) || !bytes.Equal(d.source[reference.Range.Start:reference.Range.End], []byte("[^"+appended.Label+"]")) {
			return false
		}
	}
	for _, matched := range matchedOriginal {
		if !matched {
			return false
		}
	}
	return true
}

func matchOriginalFootnoteReference(original []FootnoteReference, candidate FootnoteReference, matched []bool) bool {
	for index, reference := range original {
		if matched[index] || candidate.Range != reference.Range || candidate.LabelRange != reference.LabelRange || candidate.Label != reference.Label || candidate.Occurrence != reference.Occurrence {
			continue
		}
		matched[index] = true
		return true
	}
	return false
}

func (d *Document) footnoteDefinitionsSurviveRemoval(candidate *Document, removed Node, transforms []patchTransform) bool {
	left := footnoteDefinitionNodes(d)
	right := footnoteDefinitionNodes(candidate)
	if len(right) != len(left)-1 {
		return false
	}
	candidateIndex := 0
	for _, original := range left {
		if original.ID == removed.ID {
			continue
		}
		if candidateIndex >= len(right) || !sameShiftedFootnoteDefinition(d, candidate, original, right[candidateIndex], transforms, original.Label) {
			return false
		}
		candidateIndex++
	}
	return candidateIndex == len(right)
}

func (d *Document) footnoteReferencesSurviveDefinitionRemoval(candidate *Document, removed Node, removedRange Range, transforms []patchTransform) bool {
	expected := remainingFootnoteReferencesAfterDefinitionRemoval(d.footnoteReferences, removed, removedRange)
	if len(expected) != len(candidate.footnoteReferences) {
		return false
	}
	leftDefinitions := footnoteDefinitionLabels(d)
	rightDefinitions := footnoteDefinitionLabels(candidate)
	for index, original := range expected {
		if !sameFootnoteReferenceAfterDefinitionRemoval(original, candidate.footnoteReferences[index], transforms, leftDefinitions, rightDefinitions) {
			return false
		}
	}
	return true
}

func remainingFootnoteReferencesAfterDefinitionRemoval(references []FootnoteReference, removed Node, removedRange Range) []FootnoteReference {
	result := make([]FootnoteReference, 0, len(references))
	for _, reference := range references {
		insideRemoved := reference.Range.Start >= removedRange.Start && reference.Range.Start < removedRange.End
		boundToRemoved := reference.HasDefinition && reference.DefinitionID == removed.ID
		if !insideRemoved && !boundToRemoved {
			result = append(result, reference)
		}
	}
	return result
}

func sameFootnoteReferenceAfterDefinitionRemoval(original, candidate FootnoteReference, transforms []patchTransform, leftDefinitions, rightDefinitions map[NodeID]string) bool {
	expectedRange, rangeOK := rangeAfterPatches(original.Range, transforms)
	expectedLabelRange, labelOK := rangeAfterPatches(original.LabelRange, transforms)
	if !rangeOK || !labelOK || candidate.Range != expectedRange || candidate.LabelRange != expectedLabelRange {
		return false
	}
	if candidate.Label != original.Label || candidate.Occurrence != original.Occurrence || candidate.HasDefinition != original.HasDefinition {
		return false
	}
	if !original.HasDefinition {
		return true
	}
	label := leftDefinitions[original.DefinitionID]
	return label != "" && rightDefinitions[candidate.DefinitionID] == label
}

func footnoteDefinitionLabels(document *Document) map[NodeID]string {
	result := make(map[NodeID]string)
	for _, node := range document.nodes {
		if node.Kind == KindFootnoteDefinition {
			result[node.ID] = node.Label
		}
	}
	return result
}
