package splice

import (
	"bytes"
	"unicode/utf8"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

// PrepareReplaceFootnoteDefinitionBody prepares a source-preserving replacement
// of the conservative simple body span of one promoted footnote definition.
func (d *Document) PrepareReplaceFootnoteDefinitionBody(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.targetNode(id, KindFootnoteDefinition)
	if err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := d.footnoteSource(target)
	if !ok {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	body := mapping.BodyRange
	if body.Start >= body.End || !body.Valid(len(d.source)) {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	if err := validateFootnoteBodyReplacement(replacement); err != nil {
		return ChangeSet{}, err
	}

	change, candidate, err := d.prepareCandidateChange(body, replacement, "footnote definition body replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	transforms := []patchTransform{{Range: body, ReplacementLength: len(replacement)}}
	if !sameFootnoteDefinitionsAfterBodyReplacement(d, candidateDocument, target, replacement, transforms) ||
		!sameFootnoteReferencesOutsideOwnedBody(d, candidateDocument, target, transforms) ||
		!sameLinkUsagesOutsideOwnedBody(d, candidateDocument, target, transforms) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return change, nil
}

// PrepareRenameFootnote atomically renames one promoted definition and every
// parser-proven reference occurrence bound to that definition.
func (d *Document) PrepareRenameFootnote(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.targetNode(id, KindFootnoteDefinition)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateFootnoteLabelReplacement(replacement); err != nil {
		return ChangeSet{}, err
	}
	label := string(replacement)
	if d.footnoteLabelCollides(target.ID, label) {
		return ChangeSet{}, ErrInvalidReplacement
	}

	mapping, ok := d.footnoteSource(target)
	if !ok {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	patches := []source.Patch{{Range: mapping.LabelRange, Replacement: replacement}}
	for _, reference := range d.footnoteReferences {
		if reference.HasDefinition && reference.DefinitionID == target.ID {
			patches = append(patches, source.Patch{Range: reference.LabelRange, Replacement: replacement})
		}
	}
	change, candidate, err := d.prepareCandidateChanges(patches, "footnote rename")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	transforms := make([]patchTransform, len(patches))
	for index, patch := range patches {
		transforms[index] = patchTransform{Range: patch.Range, ReplacementLength: len(patch.Replacement)}
	}
	if !sameFootnoteDefinitionsAfterRename(d, candidateDocument, target, label, transforms) ||
		!sameFootnoteReferencesAfterRename(d, candidateDocument, target, label, transforms) ||
		!sameLinkUsagesAfterPatches(d.linkUsages, candidateDocument.linkUsages, transforms) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return change, nil
}

func validateFootnoteBodyReplacement(replacement []byte) error {
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return err
	}
	if !utf8.Valid(replacement) || bytes.IndexByte(replacement, 0) >= 0 {
		return ErrInvalidReplacement
	}
	return nil
}

func validateFootnoteLabelReplacement(replacement []byte) error {
	if err := validateFootnoteBodyReplacement(replacement); err != nil {
		return err
	}
	if bytes.ContainsAny(replacement, "[]") {
		return ErrInvalidReplacement
	}
	return nil
}

func (d *Document) footnoteLabelCollides(target NodeID, label string) bool {
	for _, node := range d.nodes {
		if node.Kind == KindFootnoteDefinition && node.ID != target && node.Label == label {
			return true
		}
	}
	return false
}

func sameFootnoteDefinitionsAfterBodyReplacement(original, candidate *Document, target Node, replacement []byte, transforms []patchTransform) bool {
	left := footnoteDefinitionNodes(original)
	right := footnoteDefinitionNodes(candidate)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].ID == target.ID {
			if !sameReplacedFootnoteDefinition(original, candidate, left[index], right[index], replacement, transforms) {
				return false
			}
			continue
		}
		if !sameShiftedFootnoteDefinition(original, candidate, left[index], right[index], transforms, left[index].Label) {
			return false
		}
	}
	return true
}

func sameFootnoteDefinitionsAfterRename(original, candidate *Document, target Node, label string, transforms []patchTransform) bool {
	left := footnoteDefinitionNodes(original)
	right := footnoteDefinitionNodes(candidate)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		expectedLabel := left[index].Label
		if left[index].ID == target.ID {
			expectedLabel = label
		}
		if !sameShiftedFootnoteDefinition(original, candidate, left[index], right[index], transforms, expectedLabel) {
			return false
		}
	}
	return true
}

func sameShiftedFootnoteDefinition(originalDocument, candidateDocument *Document, original, candidate Node, transforms []patchTransform, expectedLabel string) bool {
	originalSource, originalOK := originalDocument.footnoteSource(original)
	candidateSource, candidateOK := candidateDocument.footnoteSource(candidate)
	if !originalOK || !candidateOK {
		return false
	}
	expectedRange, ok := rangeBoundariesAfterPatches(originalSource.Range, transforms)
	if !ok {
		return false
	}
	expectedAnchor, ok := offsetAfterPatches(original.Anchor, transforms)
	if !ok {
		return false
	}
	expectedLabelRange := Range{Start: expectedAnchor + 2, End: expectedAnchor + 2 + len(expectedLabel)}
	if candidate.Kind != KindFootnoteDefinition || !candidate.TopLevel || !candidate.Editable || candidate.Label != expectedLabel ||
		candidate.Anchor != expectedAnchor || candidateSource.Range != expectedRange || candidateSource.LabelRange != expectedLabelRange {
		return false
	}
	return sameShiftedFootnoteBodyRanges(originalSource, candidateSource, transforms)
}

func sameReplacedFootnoteDefinition(originalDocument, candidateDocument *Document, original, candidate Node, replacement []byte, transforms []patchTransform) bool {
	originalSource, originalOK := originalDocument.footnoteSource(original)
	candidateSource, candidateOK := candidateDocument.footnoteSource(candidate)
	if !originalOK || !candidateOK || !sameShiftedFootnoteDefinitionBase(original, candidate, originalSource, candidateSource, transforms, original.Label) {
		return false
	}
	body := originalSource.BodyRange
	expectedBody := Range{Start: body.Start, End: body.Start + len(replacement)}
	return candidateSource.BodyRange == expectedBody && candidate.ContentRange == expectedBody &&
		len(candidateSource.BodyRanges) == 1 && candidateSource.BodyRanges[0] == expectedBody
}

func sameShiftedFootnoteDefinitionBase(original, candidate Node, originalSource, candidateSource source.FootnoteDefinitionMapping, transforms []patchTransform, expectedLabel string) bool {
	expectedRange, ok := rangeBoundariesAfterPatches(originalSource.Range, transforms)
	if !ok {
		return false
	}
	expectedAnchor, ok := offsetAfterPatches(original.Anchor, transforms)
	if !ok {
		return false
	}
	expectedLabelRange, ok := rangeBoundariesAfterPatches(originalSource.LabelRange, transforms)
	if !ok {
		return false
	}
	return candidate.Kind == KindFootnoteDefinition && candidate.TopLevel && candidate.Editable && candidate.Label == expectedLabel &&
		candidate.Anchor == expectedAnchor && candidateSource.Range == expectedRange && candidateSource.LabelRange == expectedLabelRange
}

func sameShiftedFootnoteBodyRanges(original, candidate source.FootnoteDefinitionMapping, transforms []patchTransform) bool {
	if len(original.BodyRanges) != len(candidate.BodyRanges) {
		return false
	}
	for index, range_ := range original.BodyRanges {
		expected, ok := rangeBoundariesAfterPatches(range_, transforms)
		if !ok || candidate.BodyRanges[index] != expected {
			return false
		}
	}
	if original.BodyRange.Start == original.BodyRange.End {
		return candidate.BodyRange.Start == candidate.BodyRange.End
	}
	expectedBody, ok := rangeBoundariesAfterPatches(original.BodyRange, transforms)
	return ok && candidate.BodyRange == expectedBody
}

func footnoteDefinitionNodes(document *Document) []Node {
	result := make([]Node, 0)
	for _, node := range document.nodes {
		if node.Kind == KindFootnoteDefinition {
			result = append(result, node)
		}
	}
	return result
}

func sameFootnoteReferencesAfterRename(original, candidate *Document, target Node, label string, transforms []patchTransform) bool {
	if len(original.footnoteReferences) != len(candidate.footnoteReferences) {
		return false
	}
	leftDefinitions := footnoteDefinitionIndexes(original)
	rightDefinitions := footnoteDefinitionIndexes(candidate)
	for index, reference := range original.footnoteReferences {
		expectedLabel := reference.Label
		if reference.HasDefinition && reference.DefinitionID == target.ID {
			expectedLabel = label
		}
		if !sameShiftedFootnoteReference(reference, candidate.footnoteReferences[index], expectedLabel, transforms, leftDefinitions, rightDefinitions) {
			return false
		}
	}
	return true
}

func sameFootnoteReferencesOutsideOwnedBody(original, candidate *Document, target Node, transforms []patchTransform) bool {
	mapping, ok := original.footnoteSource(target)
	if !ok {
		return false
	}
	oldBody := mapping.BodyRange
	newBody := Range{Start: oldBody.Start, End: oldBody.Start + transforms[0].ReplacementLength}
	left := footnoteReferencesOutsideOwnedRange(original.footnoteReferences, oldBody)
	right := footnoteReferencesOutsideOwnedRange(candidate.footnoteReferences, newBody)
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

func footnoteReferencesOutsideOwnedRange(references []FootnoteReference, excluded Range) []FootnoteReference {
	result := make([]FootnoteReference, 0, len(references))
	for _, reference := range references {
		if reference.Range.Start >= excluded.Start && reference.Range.Start < excluded.End {
			continue
		}
		result = append(result, reference)
	}
	return result
}

func sameShiftedFootnoteReference(original, candidate FootnoteReference, expectedLabel string, transforms []patchTransform, leftDefinitions, rightDefinitions map[NodeID]int) bool {
	expectedRange, ok := rangeBoundariesAfterPatches(original.Range, transforms)
	if !ok {
		return false
	}
	expectedLabelRange := Range{Start: expectedRange.Start + 2, End: expectedRange.Start + 2 + len(expectedLabel)}
	if candidate.Range != expectedRange || candidate.LabelRange != expectedLabelRange || candidate.Label != expectedLabel || candidate.Occurrence != original.Occurrence ||
		candidate.HasDefinition != original.HasDefinition {
		return false
	}
	if !original.HasDefinition {
		return true
	}
	leftIndex, leftOK := leftDefinitions[original.DefinitionID]
	rightIndex, rightOK := rightDefinitions[candidate.DefinitionID]
	return leftOK && rightOK && leftIndex == rightIndex
}

func footnoteDefinitionIndexes(document *Document) map[NodeID]int {
	result := make(map[NodeID]int)
	index := 0
	for _, node := range document.nodes {
		if node.Kind != KindFootnoteDefinition {
			continue
		}
		result[node.ID] = index
		index++
	}
	return result
}

func sameLinkUsagesAfterPatches(original, candidate []parser.LinkUsage, transforms []patchTransform) bool {
	if len(original) != len(candidate) {
		return false
	}
	for index, usage := range original {
		expectedAnchor, ok := offsetAfterPatches(usage.Anchor, transforms)
		if !ok {
			return false
		}
		expected := usage
		expected.Anchor = expectedAnchor
		if candidate[index] != expected {
			return false
		}
	}
	return true
}

func sameLinkUsagesOutsideOwnedBody(original, candidate *Document, target Node, transforms []patchTransform) bool {
	mapping, ok := original.footnoteSource(target)
	if !ok {
		return false
	}
	oldBody := mapping.BodyRange
	newBody := Range{Start: oldBody.Start, End: oldBody.Start + transforms[0].ReplacementLength}
	left := linkUsagesOutsideOwnedRange(original.linkUsages, oldBody)
	right := linkUsagesOutsideOwnedRange(candidate.linkUsages, newBody)
	return sameLinkUsagesAfterPatches(left, right, transforms)
}

func linkUsagesOutsideOwnedRange(usages []parser.LinkUsage, excluded Range) []parser.LinkUsage {
	result := make([]parser.LinkUsage, 0, len(usages))
	for _, usage := range usages {
		if usage.Anchor >= excluded.Start && usage.Anchor < excluded.End {
			continue
		}
		result = append(result, usage)
	}
	return result
}

func rangeBoundariesAfterPatches(range_ Range, patches []patchTransform) (Range, bool) {
	start, ok := offsetAfterPatches(range_.Start, patches)
	if !ok {
		return Range{}, false
	}
	end, ok := offsetAfterPatches(range_.End, patches)
	if !ok || end < start {
		return Range{}, false
	}
	return Range{Start: start, End: end}, true
}

func offsetAfterPatches(offset int, patches []patchTransform) (int, bool) {
	ordered, ok := orderedPatchTransforms(patches)
	if !ok || offset < 0 {
		return 0, false
	}
	delta := 0
	for _, patch := range ordered {
		if offset < patch.Range.Start {
			return offset + delta, true
		}
		if offset < patch.Range.End {
			return 0, false
		}
		delta += patch.ReplacementLength - (patch.Range.End - patch.Range.Start)
	}
	return offset + delta, true
}
