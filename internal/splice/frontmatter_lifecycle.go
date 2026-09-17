package splice

import (
	"bytes"

	"github.com/zoster81/marksplice/internal/source"
)

// PrepareRenameFrontMatterField renames one promoted simple field key while preserving its value wrapper and line trivia.
func (d *Document) PrepareRenameFrontMatterField(id NodeID, key []byte) (ChangeSet, error) {
	target, err := d.frontMatterFieldTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	if !source.ValidCanonicalFrontMatterKey(key) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	mapping, field, ok := d.frontMatterFieldMapping(target)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if bytes.Equal(d.source[field.KeyRange.Start:field.KeyRange.End], key) {
		change, ok := d.unchangedRangeChange(field.KeyRange, key)
		if !ok {
			return ChangeSet{}, ErrInvalidReplacement
		}
		return change, nil
	}
	for _, existing := range mapping.Fields {
		if existing.Range != field.Range && existing.Key == string(key) {
			return ChangeSet{}, ErrInvalidReplacement
		}
	}
	change, candidate, err := d.prepareCandidateChange(field.KeyRange, key, "front-matter field rename")
	if err != nil {
		return ChangeSet{}, err
	}
	patches := []patchTransform{{Range: field.KeyRange, ReplacementLength: len(key)}}
	if err := d.validateFrontMatterFieldRename(candidate, mapping, field, key, patches); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareRemoveFrontMatterField removes one complete promoted simple field physical line.
func (d *Document) PrepareRemoveFrontMatterField(id NodeID) (ChangeSet, error) {
	target, err := d.frontMatterFieldTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	mapping, field, ok := d.frontMatterFieldMapping(target)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	removeRange := field.Range
	removeRange.End += len(physicalLineEndingAt(d.source, field.Range.End))
	if removeRange.End <= removeRange.Start {
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(removeRange, nil, "front-matter field removal")
	if err != nil {
		return ChangeSet{}, err
	}
	patches := []patchTransform{{Range: removeRange}}
	if err := d.validateFrontMatterFieldRemoval(candidate, mapping, field, patches); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareAppendFrontMatterField appends one canonical simple field immediately before the existing closing delimiter.
func (d *Document) PrepareAppendFrontMatterField(key, value []byte) (ChangeSet, error) {
	mapping, ok := source.MapLeadingFrontMatter(d.source)
	if d == nil || !ok || !source.ValidCanonicalFrontMatterKey(key) || !source.ValidCanonicalFrontMatterValue(value) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	eol := physicalLineEndingAt(d.source, mapping.OpeningRange.End)
	fragment, ok := source.CanonicalFrontMatterFieldLine(mapping.Format, key, value, eol)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	insertAt := mapping.ClosingRange.Start
	patch := Range{Start: insertAt, End: insertAt}
	change, candidate, err := d.prepareCandidateChange(patch, fragment, "front-matter field append")
	if err != nil {
		return ChangeSet{}, err
	}
	patches := []patchTransform{{Range: patch, ReplacementLength: len(fragment)}}
	if err := d.validateFrontMatterFieldAppend(candidate, mapping, key, value, fragment, eol, patches); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareAddFrontMatter inserts one empty leading YAML/TOML envelope using the document's proven EOL style.
func (d *Document) PrepareAddFrontMatter(format FrontMatterFormat) (ChangeSet, error) {
	if d == nil || d.frontMatter.Format != source.FrontMatterUnknown {
		return ChangeSet{}, ErrInvalidReplacement
	}
	eol := []byte(d.preferredLineEnding())
	fragment, ok := source.CanonicalEmptyFrontMatter(source.FrontMatterFormat(format), eol)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	patch := Range{Start: 0, End: 0}
	change, candidate, err := d.prepareCandidateChange(patch, fragment, "front-matter envelope insertion")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	mapping, ok := source.MapLeadingFrontMatter(candidate)
	if !ok || mapping.Format != source.FrontMatterFormat(format) || len(mapping.Fields) != 0 || mapping.OpeningRange.Start != 0 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	patches := []patchTransform{{Range: patch, ReplacementLength: len(fragment)}}
	if !d.frontMatterBodySurvives(candidateDocument, patches) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return change, nil
}

// PrepareRemoveFrontMatter removes the complete leading envelope plus its owned closing EOL and one immediate blank separator line when present.
func (d *Document) PrepareRemoveFrontMatter() (ChangeSet, error) {
	if d == nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	mapping, ok := source.MapLeadingFrontMatter(d.source)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	removeRange := frontMatterRemovalRange(d.source, mapping)
	if removeRange.Start != 0 || removeRange.End <= 0 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(removeRange, nil, "front-matter envelope removal")
	if err != nil {
		return ChangeSet{}, err
	}
	if _, stillFrontMatter := source.MapLeadingFrontMatter(candidate); stillFrontMatter {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if err := d.validateNodeSurvivorsAfterRemoval(candidate, removeRange); err != nil {
		return ChangeSet{}, err
	}
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	patches := []patchTransform{{Range: removeRange}}
	if !d.frontMatterAuxiliarySemanticsSurvive(candidateDocument, patches) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return change, nil
}

func (d *Document) frontMatterFieldTarget(id NodeID) (Node, error) {
	target, ok := d.nodeByID(id)
	if !ok {
		return Node{}, ErrNodeNotFound
	}
	if !target.Editable || target.Kind != KindYAMLFrontMatterField && target.Kind != KindTOMLFrontMatterField {
		return Node{}, ErrInvalidTargetKind
	}
	return target, nil
}

func (d *Document) frontMatterFieldMapping(target Node) (source.FrontMatterMapping, source.FrontMatterFieldMapping, bool) {
	mapping, ok := source.MapLeadingFrontMatter(d.source)
	if !ok || mapping.Format != source.FrontMatterFormat(target.FrontMatterFormat) {
		return source.FrontMatterMapping{}, source.FrontMatterFieldMapping{}, false
	}
	for _, field := range mapping.Fields {
		if field.Key == target.Key && field.Range == target.Range && field.ValueRange == target.ContentRange {
			return mapping, field, true
		}
	}
	return source.FrontMatterMapping{}, source.FrontMatterFieldMapping{}, false
}

func (d *Document) validateFrontMatterFieldRename(candidate []byte, original source.FrontMatterMapping, target source.FrontMatterFieldMapping, key []byte, patches []patchTransform) error {
	candidateMapping, candidateDocument, ok := frontMatterCandidate(candidate, original.Format, len(original.Fields))
	if !ok || !sameShiftedFrontMatterEnvelope(original, candidateMapping, patches) {
		return ErrInvalidReplacement
	}
	for index, field := range original.Fields {
		current := candidateMapping.Fields[index]
		if field.Range == target.Range {
			if !renamedFrontMatterFieldMatches(field, current, key, patches) {
				return ErrInvalidReplacement
			}
			continue
		}
		if !sameShiftedFrontMatterField(field, current, patches) {
			return ErrInvalidReplacement
		}
	}
	if !d.frontMatterBodySurvives(candidateDocument, patches) {
		return ErrInvalidReplacement
	}
	return nil
}

func (d *Document) validateFrontMatterFieldRemoval(candidate []byte, original source.FrontMatterMapping, target source.FrontMatterFieldMapping, patches []patchTransform) error {
	candidateMapping, candidateDocument, ok := frontMatterCandidate(candidate, original.Format, len(original.Fields)-1)
	if !ok || !sameShiftedFrontMatterEnvelope(original, candidateMapping, patches) {
		return ErrInvalidReplacement
	}
	candidateIndex := 0
	for _, field := range original.Fields {
		if field.Range == target.Range {
			continue
		}
		if candidateIndex >= len(candidateMapping.Fields) || !sameShiftedFrontMatterField(field, candidateMapping.Fields[candidateIndex], patches) {
			return ErrInvalidReplacement
		}
		candidateIndex++
	}
	if candidateIndex != len(candidateMapping.Fields) || !d.frontMatterBodySurvives(candidateDocument, patches) {
		return ErrInvalidReplacement
	}
	return nil
}

func (d *Document) validateFrontMatterFieldAppend(candidate []byte, original source.FrontMatterMapping, key, value, fragment, eol []byte, patches []patchTransform) error {
	candidateMapping, candidateDocument, ok := frontMatterCandidate(candidate, original.Format, len(original.Fields)+1)
	if !ok || !sameShiftedFrontMatterEnvelope(original, candidateMapping, patches) {
		return ErrInvalidReplacement
	}
	for index, field := range original.Fields {
		if !sameShiftedFrontMatterField(field, candidateMapping.Fields[index], patches) {
			return ErrInvalidReplacement
		}
	}
	added := candidateMapping.Fields[len(candidateMapping.Fields)-1]
	lineLength := len(fragment) - len(eol)
	if added.Format != original.Format || added.Key != string(key) || added.Style != source.FrontMatterValueDoubleQuoted || added.Quote != '"' ||
		added.Range != (Range{Start: original.ClosingRange.Start, End: original.ClosingRange.Start + lineLength}) ||
		!added.KeyRange.Valid(len(candidate)) || !added.ValueRange.Valid(len(candidate)) ||
		!bytes.Equal(candidate[added.KeyRange.Start:added.KeyRange.End], key) || !bytes.Equal(candidate[added.ValueRange.Start:added.ValueRange.End], value) {
		return ErrInvalidReplacement
	}
	if !d.frontMatterBodySurvives(candidateDocument, patches) {
		return ErrInvalidReplacement
	}
	return nil
}

func frontMatterCandidate(candidate []byte, format source.FrontMatterFormat, fieldCount int) (source.FrontMatterMapping, *Document, bool) {
	mapping, ok := source.MapLeadingFrontMatter(candidate)
	if !ok || mapping.Format != format || len(mapping.Fields) != fieldCount {
		return source.FrontMatterMapping{}, nil, false
	}
	document, err := Parse(candidate)
	return mapping, document, err == nil
}

func sameShiftedFrontMatterEnvelope(original, candidate source.FrontMatterMapping, patches []patchTransform) bool {
	if candidate.Format != original.Format || candidate.OpeningRange != original.OpeningRange {
		return false
	}
	expectedClosing, ok := rangeAfterPatches(original.ClosingRange, patches)
	return ok && candidate.ClosingRange == expectedClosing && candidate.Range == (Range{Start: 0, End: expectedClosing.End})
}

func sameShiftedFrontMatterField(original, candidate source.FrontMatterFieldMapping, patches []patchTransform) bool {
	if candidate.Format != original.Format || candidate.Key != original.Key || candidate.Style != original.Style || candidate.Quote != original.Quote {
		return false
	}
	expectedRange, ok := rangeAfterPatches(original.Range, patches)
	if !ok || candidate.Range != expectedRange {
		return false
	}
	expectedKey, ok := rangeAfterPatches(original.KeyRange, patches)
	if !ok || candidate.KeyRange != expectedKey {
		return false
	}
	expectedValue, ok := rangeAfterPatches(original.ValueRange, patches)
	return ok && candidate.ValueRange == expectedValue
}

func renamedFrontMatterFieldMatches(original, candidate source.FrontMatterFieldMapping, key []byte, patches []patchTransform) bool {
	if candidate.Format != original.Format || candidate.Key != string(key) || candidate.Style != original.Style || candidate.Quote != original.Quote {
		return false
	}
	delta := len(key) - (original.KeyRange.End - original.KeyRange.Start)
	if candidate.Range != (Range{Start: original.Range.Start, End: original.Range.End + delta}) ||
		candidate.KeyRange != (Range{Start: original.KeyRange.Start, End: original.KeyRange.Start + len(key)}) {
		return false
	}
	expectedValue, ok := rangeAfterPatches(original.ValueRange, patches)
	return ok && candidate.ValueRange == expectedValue
}

func (d *Document) frontMatterBodySurvives(candidate *Document, patches []patchTransform) bool {
	if d == nil || candidate == nil {
		return false
	}
	matched := make([]bool, len(candidate.nodes))
	originalCount := 0
	candidateCount := 0
	for _, node := range candidate.nodes {
		if !frontMatterFieldKind(node.Kind) {
			candidateCount++
		}
	}
	for _, original := range d.nodes {
		if frontMatterFieldKind(original.Kind) {
			continue
		}
		originalCount++
		index := matchingInsertionSurvivor(d, candidate, matched, original, patches)
		if index < 0 {
			return false
		}
		matched[index] = true
	}
	if originalCount != candidateCount || !d.frontMatterAuxiliarySemanticsSurvive(candidate, patches) {
		return false
	}
	return d.validateSectionHeadingsAfterPatches(candidate.source, candidate, patches) == nil
}

func (d *Document) frontMatterAuxiliarySemanticsSurvive(candidate *Document, patches []patchTransform) bool {
	if !sameLinkUsagesAfterPatches(d.linkUsages, candidate.linkUsages, patches) ||
		!sameUnresolvedReferenceUsagesAfterPatches(d.unresolvedReferenceUsages, candidate.unresolvedReferenceUsages, patches) ||
		len(d.footnoteReferences) != len(candidate.footnoteReferences) {
		return false
	}
	leftDefinitions := footnoteDefinitionIndexes(d)
	rightDefinitions := footnoteDefinitionIndexes(candidate)
	for index, reference := range d.footnoteReferences {
		if !sameShiftedFootnoteReference(reference, candidate.footnoteReferences[index], reference.Label, patches, leftDefinitions, rightDefinitions) {
			return false
		}
	}
	return true
}

func frontMatterFieldKind(kind Kind) bool {
	return kind == KindYAMLFrontMatterField || kind == KindTOMLFrontMatterField
}

func frontMatterRemovalRange(input []byte, mapping source.FrontMatterMapping) Range {
	owned := mapping.Range
	owned.End += len(physicalLineEndingAt(input, owned.End))
	if owned.End >= len(input) {
		return owned
	}
	lineEnd, next, _ := physicalLine(input, owned.End)
	if len(bytes.Trim(input[owned.End:lineEnd], " \t")) == 0 {
		owned.End = next
	}
	return owned
}
