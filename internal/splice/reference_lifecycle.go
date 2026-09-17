package splice

import (
	"bytes"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

type referenceUsageTarget struct {
	index   int
	usage   parser.LinkUsage
	mapping source.ReferenceOccurrenceMapping
}

type referenceUsageExpectation struct {
	kind        parser.Kind
	form        parser.LinkUsageForm
	reference   string
	destination string
	title       string
	hasTitle    bool
	email       bool
}

// PrepareRetargetReferenceOccurrence retargets one parser-proven reference link/image
// occurrence to one existing uniquely normalized reference definition. Collapsed and
// shortcut occurrences are promoted to full form while preserving their visible label.
func (d *Document) PrepareRetargetReferenceOccurrence(sourceOffset int, replacement []byte) (ChangeSet, error) {
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}
	target, err := d.referenceUsageTarget(sourceOffset)
	if err != nil {
		return ChangeSet{}, err
	}
	definition, ok := d.uniqueReferenceDefinitionByLabel(string(replacement))
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if referenceLabelKey(target.usage.Reference) == referenceLabelKey(string(replacement)) &&
		target.usage.Destination == definition.Destination && target.usage.Title == definition.Title && target.usage.HasTitle == definition.HasTitle {
		return d.newChanges(nil, "reference occurrence retarget")
	}
	patch := referenceOccurrencePatch(target.mapping, replacement)
	change, candidate, err := d.prepareCandidateChanges([]source.Patch{patch}, "reference occurrence retarget")
	if err != nil {
		return ChangeSet{}, err
	}
	transforms := patchTransforms([]source.Patch{patch})
	expected := referenceUsageExpectation{
		kind:        target.usage.Kind,
		form:        parser.LinkUsageFull,
		reference:   string(replacement),
		destination: definition.Destination,
		title:       definition.Title,
		hasTitle:    definition.HasTitle,
		email:       target.usage.AutoLinkEmail,
	}
	if err := d.validateReferenceUsageCandidate(candidate, transforms, map[int]referenceUsageExpectation{target.index: expected}); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareRenameReferenceDefinition atomically renames one uniquely normalized editable
// reference definition and every parser-proven occurrence bound to it. Collapsed and
// shortcut occurrences are promoted to full form so their visible labels remain unchanged.
func (d *Document) PrepareRenameReferenceDefinition(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindReferenceDefinition, "reference definition")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateNonEmptySingleLine(replacement); err != nil {
		return ChangeSet{}, err
	}
	mapping, err := d.referenceRenameMapping(target, replacement)
	if err != nil {
		return ChangeSet{}, err
	}
	if bytes.Equal(d.source[mapping.LabelRange.Start:mapping.LabelRange.End], replacement) {
		return d.newChanges(nil, "reference definition rename")
	}
	patches, expected, err := d.referenceRenamePatches(target, mapping, replacement)
	if err != nil {
		return ChangeSet{}, err
	}
	change, candidate, err := d.prepareCandidateChanges(patches, "reference definition rename")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateReferenceRenameCandidate(candidate, target, replacement, patches, expected); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func (d *Document) referenceRenameMapping(target Node, replacement []byte) (source.ReferenceDefinitionMapping, error) {
	owner, ok := d.uniqueReferenceDefinitionByLabel(target.Label)
	if !ok || owner.ID != target.ID || referenceLabelKey(string(replacement)) == "" || d.referenceDefinitionLabelCollides(target.ID, string(replacement)) {
		return source.ReferenceDefinitionMapping{}, ErrInvalidReplacement
	}
	mapping, ok := remapReferenceDefinitionSource(d.source, target)
	if !ok || !mapping.LabelRange.Valid(len(d.source)) {
		return source.ReferenceDefinitionMapping{}, ErrInvalidReplacement
	}
	return mapping, nil
}

func (d *Document) referenceRenamePatches(target Node, mapping source.ReferenceDefinitionMapping, replacement []byte) ([]source.Patch, map[int]referenceUsageExpectation, error) {
	patches := []source.Patch{{Range: mapping.LabelRange, Replacement: replacement}}
	expected := make(map[int]referenceUsageExpectation)
	for index, usage := range d.linkUsages {
		if usage.Form == parser.LinkUsageDirect || referenceLabelKey(usage.Reference) != referenceLabelKey(target.Label) {
			continue
		}
		occurrence, ok := d.mapReferenceOccurrence(usage)
		if !ok {
			return nil, nil, ErrInvalidReplacement
		}
		patches = append(patches, referenceOccurrencePatch(occurrence, replacement))
		expected[index] = referenceUsageExpectation{kind: usage.Kind, form: parser.LinkUsageFull, reference: string(replacement), destination: target.Destination, title: target.Title, hasTitle: target.HasTitle, email: usage.AutoLinkEmail}
	}
	return patches, expected, nil
}

func (d *Document) validateReferenceRenameCandidate(candidate []byte, target Node, replacement []byte, patches []source.Patch, expected map[int]referenceUsageExpectation) error {
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ErrInvalidReplacement
	}
	transforms := patchTransforms(patches)
	if !d.referenceDefinitionsMatchAfterMutation(candidateDocument, target.ID, string(replacement), target.Destination, target.Title, target.HasTitle, transforms) ||
		!d.referenceUsagesMatch(candidateDocument, transforms, expected) ||
		!sameUnresolvedReferenceUsagesAfterPatches(d.unresolvedReferenceUsages, candidateDocument.unresolvedReferenceUsages, transforms) {
		return ErrInvalidReplacement
	}
	return nil
}

// PrepareAppendReferenceDefinition appends one canonical top-level single-line
// reference definition while preserving the document's proven line-ending style.
func (d *Document) PrepareAppendReferenceDefinition(label, destination []byte) (ChangeSet, error) {
	return d.prepareAppendReferenceDefinition(label, destination, nil, false)
}

// PrepareAppendReferenceDefinitionWithTitle appends one canonical top-level single-line
// reference definition with a canonical double-quoted title.
func (d *Document) PrepareAppendReferenceDefinitionWithTitle(label, destination, title []byte) (ChangeSet, error) {
	return d.prepareAppendReferenceDefinition(label, destination, title, true)
}

func (d *Document) prepareAppendReferenceDefinition(label, destination, title []byte, hasTitle bool) (ChangeSet, error) {
	if d == nil || d.uniqueReferenceDefinitionKeyExists(string(label)) || referenceLabelKey(string(label)) == "" {
		return ChangeSet{}, ErrInvalidReplacement
	}
	prefix, eol, ok := d.referenceDefinitionAppendLayout()
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	definition, ok := canonicalReferenceDefinition(label, destination, title, hasTitle, eol)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	replacement := make([]byte, 0, len(prefix)+len(definition))
	replacement = append(replacement, prefix...)
	replacement = append(replacement, definition...)
	patch := source.Patch{Range: Range{Start: len(d.source), End: len(d.source)}, Replacement: replacement}
	change, candidate, err := d.prepareCandidateChanges([]source.Patch{patch}, "reference definition append")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	definitionOffset := len(d.source) + len(prefix)
	if !d.validateAppendedReferenceDefinition(candidateDocument, definitionOffset, string(label), string(destination), string(title), hasTitle) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return change, nil
}

// PrepareAddReferenceDefinitionTitle inserts one canonical double-quoted title into
// an editable single-line reference definition that currently has no title.
func (d *Document) PrepareAddReferenceDefinitionTitle(id NodeID, title []byte) (ChangeSet, error) {
	if err := validateNonEmptySingleLine(title); err != nil || bytes.ContainsRune(title, '"') {
		return ChangeSet{}, ErrInvalidReplacement
	}
	target, err := d.editableTargetNode(id, KindReferenceDefinition, "reference definition")
	if err != nil {
		return ChangeSet{}, err
	}
	owner, unique := d.uniqueReferenceDefinitionByLabel(target.Label)
	if !unique || owner.ID != target.ID {
		return ChangeSet{}, ErrInvalidReplacement
	}
	mapping, ok := remapReferenceDefinitionSource(d.source, target)
	if !ok || mapping.HasTitle {
		return ChangeSet{}, ErrInvalidReplacement
	}
	insertAt := mapping.Range.End
	fragment := canonicalReferenceTitleInsertion(d.source, mapping, title)
	if len(fragment) == 0 {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return d.prepareReferenceDefinitionTitleMutation(target, Range{Start: insertAt, End: insertAt}, fragment, string(title), true, "reference definition title addition")
}

// PrepareRemoveReferenceDefinitionTitle removes the source-owned title syntax from
// one editable single-line reference definition while preserving any extra authored spacing.
func (d *Document) PrepareRemoveReferenceDefinitionTitle(id NodeID) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindReferenceDefinition, "reference definition")
	if err != nil {
		return ChangeSet{}, err
	}
	owner, unique := d.uniqueReferenceDefinitionByLabel(target.Label)
	if !unique || owner.ID != target.ID {
		return ChangeSet{}, ErrInvalidReplacement
	}
	mapping, ok := remapReferenceDefinitionSource(d.source, target)
	if !ok || !mapping.HasTitle {
		return ChangeSet{}, ErrInvalidReplacement
	}
	removeRange, ok := referenceDefinitionTitleSyntaxRange(d.source, mapping)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return d.prepareReferenceDefinitionTitleMutation(target, removeRange, nil, "", false, "reference definition title removal")
}

func (d *Document) prepareReferenceDefinitionTitleMutation(target Node, patchRange Range, replacement []byte, expectedTitle string, expectedHasTitle bool, operation string) (ChangeSet, error) {
	patch := source.Patch{Range: patchRange, Replacement: replacement}
	change, candidate, err := d.prepareCandidateChanges([]source.Patch{patch}, operation)
	if err != nil {
		return ChangeSet{}, err
	}
	transforms := patchTransforms([]source.Patch{patch})
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ChangeSet{}, ErrInvalidReplacement
	}
	expectedUsages := d.referenceDefinitionUsageExpectations(target, expectedTitle, expectedHasTitle)
	if !d.referenceDefinitionsMatchAfterMutation(candidateDocument, target.ID, target.Label, target.Destination, expectedTitle, expectedHasTitle, transforms) ||
		!d.referenceUsagesMatch(candidateDocument, transforms, expectedUsages) ||
		!sameUnresolvedReferenceUsagesAfterPatches(d.unresolvedReferenceUsages, candidateDocument.unresolvedReferenceUsages, transforms) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	return change, nil
}

func (d *Document) referenceUsageTarget(sourceOffset int) (referenceUsageTarget, error) {
	found := referenceUsageTarget{index: -1}
	for index, usage := range d.linkUsages {
		if usage.Anchor != sourceOffset || usage.Form == parser.LinkUsageDirect {
			continue
		}
		if found.index >= 0 {
			return referenceUsageTarget{}, ErrInvalidReplacement
		}
		mapping, ok := d.mapReferenceOccurrence(usage)
		if !ok {
			return referenceUsageTarget{}, ErrInvalidReplacement
		}
		found = referenceUsageTarget{index: index, usage: usage, mapping: mapping}
	}
	if found.index < 0 {
		return referenceUsageTarget{}, ErrInvalidReplacement
	}
	return found, nil
}

func (d *Document) mapReferenceOccurrence(usage parser.LinkUsage) (source.ReferenceOccurrenceMapping, bool) {
	form, ok := sourceReferenceOccurrenceForm(usage.Form)
	if !ok {
		return source.ReferenceOccurrenceMapping{}, false
	}
	image, ok := referenceUsageImage(usage.Kind)
	if !ok {
		return source.ReferenceOccurrenceMapping{}, false
	}
	mapping, err := source.MapSimpleReferenceOccurrence(d.source, usage.Anchor, image, form, usage.Reference)
	return mapping, err == nil
}

func sourceReferenceOccurrenceForm(form parser.LinkUsageForm) (source.ReferenceOccurrenceForm, bool) {
	switch form {
	case parser.LinkUsageFull:
		return source.ReferenceOccurrenceFull, true
	case parser.LinkUsageCollapsed:
		return source.ReferenceOccurrenceCollapsed, true
	case parser.LinkUsageShortcut:
		return source.ReferenceOccurrenceShortcut, true
	default:
		return source.ReferenceOccurrenceUnknown, false
	}
}

func referenceUsageImage(kind parser.Kind) (bool, bool) {
	switch kind {
	case parser.KindInlineLink:
		return false, true
	case parser.KindImage:
		return true, true
	default:
		return false, false
	}
}

func referenceOccurrencePatch(mapping source.ReferenceOccurrenceMapping, replacement []byte) source.Patch {
	payload := append([]byte(nil), replacement...)
	if mapping.Form == source.ReferenceOccurrenceShortcut {
		wrapped := make([]byte, 0, len(payload)+2)
		wrapped = append(wrapped, '[')
		wrapped = append(wrapped, payload...)
		wrapped = append(wrapped, ']')
		payload = wrapped
	}
	return source.Patch{Range: mapping.ReferenceRange, Replacement: payload}
}

func (d *Document) uniqueReferenceDefinitionByLabel(label string) (Node, bool) {
	key := referenceLabelKey(label)
	if key == "" {
		return Node{}, false
	}
	var result Node
	count := 0
	for _, node := range d.nodes {
		if node.Kind != KindReferenceDefinition || referenceLabelKey(node.Label) != key {
			continue
		}
		count++
		result = node
	}
	return result, count == 1 && result.Editable
}

func (d *Document) referenceDefinitionLabelCollides(exclude NodeID, label string) bool {
	key := referenceLabelKey(label)
	if key == "" {
		return true
	}
	for _, node := range d.nodes {
		if node.Kind == KindReferenceDefinition && node.ID != exclude && referenceLabelKey(node.Label) == key {
			return true
		}
	}
	return false
}

func (d *Document) uniqueReferenceDefinitionKeyExists(label string) bool {
	key := referenceLabelKey(label)
	for _, node := range d.nodes {
		if node.Kind == KindReferenceDefinition && referenceLabelKey(node.Label) == key {
			return true
		}
	}
	return false
}

func patchTransforms(patches []source.Patch) []patchTransform {
	transforms := make([]patchTransform, len(patches))
	for index, patch := range patches {
		transforms[index] = patchTransform{Range: patch.Range, ReplacementLength: len(patch.Replacement)}
	}
	return transforms
}

func (d *Document) validateReferenceUsageCandidate(candidate []byte, transforms []patchTransform, expected map[int]referenceUsageExpectation) error {
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ErrInvalidReplacement
	}
	if !d.referenceUsagesMatch(candidateDocument, transforms, expected) ||
		!sameUnresolvedReferenceUsagesAfterPatches(d.unresolvedReferenceUsages, candidateDocument.unresolvedReferenceUsages, transforms) ||
		!d.referenceDefinitionsMatchAfterMutation(candidateDocument, "", "", "", "", false, transforms) {
		return ErrInvalidReplacement
	}
	return nil
}

func (d *Document) referenceUsagesMatch(candidate *Document, transforms []patchTransform, expected map[int]referenceUsageExpectation) bool {
	if len(d.linkUsages) != len(candidate.linkUsages) {
		return false
	}
	for index, original := range d.linkUsages {
		anchor, ok := offsetAfterPatches(original.Anchor, transforms)
		if !ok {
			return false
		}
		if mutation, changed := expected[index]; changed {
			if !referenceUsageMatchesExpectation(candidate.linkUsages[index], anchor, mutation) {
				return false
			}
			continue
		}
		want := original
		want.Anchor = anchor
		if candidate.linkUsages[index] != want {
			return false
		}
	}
	return true
}

func referenceUsageMatchesExpectation(candidate parser.LinkUsage, anchor int, expected referenceUsageExpectation) bool {
	return candidate.Kind == expected.kind && candidate.Form == expected.form && candidate.Anchor == anchor &&
		candidate.Reference == expected.reference && candidate.Destination == expected.destination &&
		candidate.Title == expected.title && candidate.HasTitle == expected.hasTitle && candidate.AutoLinkEmail == expected.email
}

func sameUnresolvedReferenceUsagesAfterPatches(original, candidate []parser.UnresolvedReferenceUsage, transforms []patchTransform) bool {
	if len(original) != len(candidate) {
		return false
	}
	for index, usage := range original {
		anchor, ok := offsetAfterPatches(usage.Anchor, transforms)
		if !ok {
			return false
		}
		want := usage
		want.Anchor = anchor
		if candidate[index] != want {
			return false
		}
	}
	return true
}

func (d *Document) referenceDefinitionsMatchAfterMutation(candidate *Document, targetID NodeID, expectedLabel, expectedDestination, expectedTitle string, expectedHasTitle bool, transforms []patchTransform) bool {
	originalDefinitions := referenceDefinitionNodesInOrder(d)
	candidateDefinitions := referenceDefinitionNodesInOrder(candidate)
	if len(originalDefinitions) != len(candidateDefinitions) {
		return false
	}
	for index, original := range originalDefinitions {
		current := candidateDefinitions[index]
		if targetID != "" && original.ID == targetID {
			if !d.mutatedReferenceDefinitionMatches(candidate, original, current, expectedLabel, expectedDestination, expectedTitle, expectedHasTitle, transforms) {
				return false
			}
			continue
		}
		if !sameShiftedReferenceDefinition(d, candidate, original, current, transforms) {
			return false
		}
	}
	return true
}

func referenceDefinitionNodesInOrder(document *Document) []Node {
	result := make([]Node, 0)
	for _, node := range document.nodes {
		if node.Kind == KindReferenceDefinition {
			result = append(result, node)
		}
	}
	return result
}

func sameShiftedReferenceDefinition(originalDocument, candidateDocument *Document, original, candidate Node, transforms []patchTransform) bool {
	if candidate.Label != original.Label || candidate.Destination != original.Destination || candidate.Title != original.Title || candidate.HasTitle != original.HasTitle || !candidate.Editable {
		return false
	}
	originalMapping, originalOK := remapReferenceDefinitionSource(originalDocument.source, original)
	candidateMapping, candidateOK := remapReferenceDefinitionSource(candidateDocument.source, candidate)
	return originalOK && candidateOK && sameShiftedReferenceDefinitionMapping(originalMapping, candidateMapping, transforms)
}

func sameShiftedReferenceDefinitionMapping(original, candidate source.ReferenceDefinitionMapping, transforms []patchTransform) bool {
	ranges := [][2]Range{{original.Range, candidate.Range}, {original.LineRange, candidate.LineRange}, {original.LabelRange, candidate.LabelRange}, {original.DestinationRange, candidate.DestinationRange}}
	for _, pair := range ranges {
		expected, ok := rangeAfterPatches(pair[0], transforms)
		if !ok || pair[1] != expected {
			return false
		}
	}
	return candidate.AngleDestination == original.AngleDestination && sameShiftedOptionalReferenceTitle(original, candidate, transforms)
}

func sameShiftedOptionalReferenceTitle(original, candidate source.ReferenceDefinitionMapping, transforms []patchTransform) bool {
	if original.HasTitle != candidate.HasTitle {
		return false
	}
	if !original.HasTitle {
		return true
	}
	expected, ok := rangeAfterPatches(original.TitleRange, transforms)
	return ok && candidate.TitleRange == expected
}

func (d *Document) mutatedReferenceDefinitionMatches(candidateDocument *Document, original, candidate Node, label, destination, title string, hasTitle bool, transforms []patchTransform) bool {
	if candidate.Label != label || candidate.Destination != destination || candidate.Title != title || candidate.HasTitle != hasTitle || !candidate.Editable {
		return false
	}
	originalMapping, originalOK := remapReferenceDefinitionSource(d.source, original)
	candidateMapping, candidateOK := remapReferenceDefinitionSource(candidateDocument.source, candidate)
	return originalOK && candidateOK && mutatedReferenceMappingMatches(originalMapping, candidateMapping, label, hasTitle, transforms)
}

func mutatedReferenceMappingMatches(original, candidate source.ReferenceDefinitionMapping, label string, hasTitle bool, transforms []patchTransform) bool {
	expectedRange, rangeOK := rangeBoundariesAfterPatches(original.Range, transforms)
	expectedLine, lineOK := rangeBoundariesAfterPatches(original.LineRange, transforms)
	expectedDestination, destinationOK := rangeAfterPatches(original.DestinationRange, transforms)
	if !rangeOK || !lineOK || !destinationOK || candidate.Range != expectedRange || candidate.LineRange != expectedLine || candidate.DestinationRange != expectedDestination {
		return false
	}
	if candidate.LabelRange.End-candidate.LabelRange.Start != len(label) || candidate.AngleDestination != original.AngleDestination {
		return false
	}
	return mutatedReferenceTitleMappingMatches(original, candidate, transforms, hasTitle)
}

func mutatedReferenceTitleMappingMatches(original, candidate source.ReferenceDefinitionMapping, transforms []patchTransform, hasTitle bool) bool {
	if candidate.HasTitle != hasTitle {
		return false
	}
	if original.HasTitle && hasTitle {
		expected, ok := rangeAfterPatches(original.TitleRange, transforms)
		return ok && candidate.TitleRange == expected
	}
	return !hasTitle || candidate.TitleRange.Start < candidate.TitleRange.End
}

func (d *Document) referenceDefinitionUsageExpectations(target Node, title string, hasTitle bool) map[int]referenceUsageExpectation {
	result := make(map[int]referenceUsageExpectation)
	key := referenceLabelKey(target.Label)
	for index, usage := range d.linkUsages {
		if usage.Form == parser.LinkUsageDirect || referenceLabelKey(usage.Reference) != key || usage.Destination != target.Destination || usage.Title != target.Title || usage.HasTitle != target.HasTitle {
			continue
		}
		result[index] = referenceUsageExpectation{
			kind:        usage.Kind,
			form:        usage.Form,
			reference:   usage.Reference,
			destination: target.Destination,
			title:       title,
			hasTitle:    hasTitle,
			email:       usage.AutoLinkEmail,
		}
	}
	return result
}

func canonicalReferenceTitleInsertion(input []byte, mapping source.ReferenceDefinitionMapping, title []byte) []byte {
	destinationEnd := referenceDestinationSyntaxEnd(mapping)
	if destinationEnd < 0 || destinationEnd > mapping.Range.End || mapping.Range.End > len(input) {
		return nil
	}
	prefix := byte(' ')
	hasSeparator := mapping.Range.End > destinationEnd
	capacity := len(title) + 2
	if !hasSeparator {
		capacity++
	}
	fragment := make([]byte, 0, capacity)
	if !hasSeparator {
		fragment = append(fragment, prefix)
	}
	fragment = append(fragment, '"')
	fragment = append(fragment, title...)
	fragment = append(fragment, '"')
	return fragment
}

func referenceDefinitionTitleSyntaxRange(input []byte, mapping source.ReferenceDefinitionMapping) (Range, bool) {
	if !mapping.HasTitle || !mapping.TitleRange.Valid(len(input)) || mapping.TitleRange.Start <= 0 || mapping.TitleRange.End >= len(input) {
		return Range{}, false
	}
	opener := mapping.TitleRange.Start - 1
	closer := mapping.TitleRange.End
	if !matchingReferenceTitleDelimiters(input[opener], input[closer]) {
		return Range{}, false
	}
	start := opener
	destinationEnd := referenceDestinationSyntaxEnd(mapping)
	if opener == destinationEnd+1 && destinationEnd < len(input) && input[destinationEnd] == ' ' {
		start = destinationEnd
	}
	return Range{Start: start, End: closer + 1}, true
}

func matchingReferenceTitleDelimiters(open, close byte) bool {
	return open == '"' && close == '"' || open == '\'' && close == '\'' || open == '(' && close == ')'
}

func referenceDestinationSyntaxEnd(mapping source.ReferenceDefinitionMapping) int {
	end := mapping.DestinationRange.End
	if mapping.AngleDestination {
		end++
	}
	return end
}

func (d *Document) referenceDefinitionAppendLayout() ([]byte, []byte, bool) {
	if len(d.source) == 0 {
		return nil, []byte("\n"), true
	}
	eol := []byte(d.preferredLineEnding())
	if len(eol) == 0 {
		return nil, nil, false
	}
	if !sourceEndsWithLineEnding(d.source) {
		prefix := make([]byte, 0, 2*len(eol))
		prefix = append(prefix, eol...)
		prefix = append(prefix, eol...)
		return prefix, eol, true
	}
	if sourceEndsWithBlankPhysicalLine(d.source) {
		return nil, eol, true
	}
	return append([]byte(nil), eol...), eol, true
}

func sourceEndsWithLineEnding(input []byte) bool {
	if len(input) == 0 {
		return false
	}
	return input[len(input)-1] == '\n' || input[len(input)-1] == '\r'
}

func sourceEndsWithBlankPhysicalLine(input []byte) bool {
	if !sourceEndsWithLineEnding(input) {
		return false
	}
	for start := 0; start < len(input); {
		lineEnd, next, _ := physicalLine(input, start)
		if next == len(input) {
			return len(bytes.Trim(input[start:lineEnd], " \t")) == 0
		}
		if next <= start {
			return false
		}
		start = next
	}
	return false
}

func canonicalReferenceDefinition(label, destination, title []byte, hasTitle bool, eol []byte) ([]byte, bool) {
	if !validCanonicalReferenceDefinitionInput(label, destination, title, hasTitle, eol) {
		return nil, false
	}
	fragment := make([]byte, 0, len(label)+len(destination)+len(title)+12+len(eol))
	fragment = append(fragment, '[')
	fragment = append(fragment, label...)
	fragment = append(fragment, ']', ':', ' ', '<')
	fragment = append(fragment, destination...)
	fragment = append(fragment, '>')
	if hasTitle {
		fragment = append(fragment, ' ', '"')
		fragment = append(fragment, title...)
		fragment = append(fragment, '"')
	}
	fragment = append(fragment, eol...)
	return fragment, canonicalReferenceDefinitionMatches(fragment, label, destination, title, hasTitle)
}

func validCanonicalReferenceDefinitionInput(label, destination, title []byte, hasTitle bool, eol []byte) bool {
	return len(label) != 0 && len(destination) != 0 && len(eol) != 0 &&
		!bytes.ContainsAny(label, "\r\n") && !bytes.ContainsAny(destination, "\r\n>") &&
		(!hasTitle || len(title) != 0 && !bytes.ContainsAny(title, "\r\n\""))
}

func canonicalReferenceDefinitionMatches(fragment, label, destination, title []byte, hasTitle bool) bool {
	parsed, err := Parse(fragment)
	if err != nil {
		return false
	}
	definitions := referenceDefinitionNodesInOrder(parsed)
	if len(definitions) != 1 || definitions[0].Label != string(label) || definitions[0].Destination != string(destination) || definitions[0].Title != string(title) || definitions[0].HasTitle != hasTitle {
		return false
	}
	mapping, ok := remapReferenceDefinitionSource(parsed.source, definitions[0])
	return ok && mapping.LineRange == (Range{Start: 0, End: len(fragment)})
}

func (d *Document) validateAppendedReferenceDefinition(candidate *Document, offset int, label, destination, title string, hasTitle bool) bool {
	originalDefinitions := referenceDefinitionNodesInOrder(d)
	candidateDefinitions := referenceDefinitionNodesInOrder(candidate)
	if len(candidateDefinitions) != len(originalDefinitions)+1 {
		return false
	}
	for index := range originalDefinitions {
		if !sameShiftedReferenceDefinition(d, candidate, originalDefinitions[index], candidateDefinitions[index], nil) {
			return false
		}
	}
	appended := candidateDefinitions[len(candidateDefinitions)-1]
	mapping, ok := remapReferenceDefinitionSource(candidate.source, appended)
	if !ok || mapping.LineRange.Start != offset || appended.Label != label || appended.Destination != destination || appended.Title != title || appended.HasTitle != hasTitle {
		return false
	}
	return d.referenceResolutionAfterAppendMatches(candidate, label, destination, title, hasTitle)
}

func (d *Document) referenceResolutionAfterAppendMatches(candidate *Document, label, destination, title string, hasTitle bool) bool {
	resolved := make(map[int]parser.UnresolvedReferenceUsage)
	key := referenceLabelKey(label)
	for _, usage := range d.unresolvedReferenceUsages {
		if referenceLabelKey(usage.Reference) == key {
			resolved[usage.Anchor] = usage
		}
	}
	seenOriginal := make(map[int]bool, len(d.linkUsages))
	seenResolved := make(map[int]bool, len(resolved))
	for _, usage := range candidate.linkUsages {
		if matchOriginalLinkUsage(d.linkUsages, usage, seenOriginal) {
			continue
		}
		unresolved, ok := resolved[usage.Anchor]
		if !ok || seenResolved[usage.Anchor] || !resolvedUsageMatchesDefinition(unresolved, usage, label, destination, title, hasTitle) {
			return false
		}
		seenResolved[usage.Anchor] = true
	}
	if len(seenOriginal) != len(d.linkUsages) || len(seenResolved) != len(resolved) {
		return false
	}
	return unresolvedReferencesAfterAppendMatch(d.unresolvedReferenceUsages, candidate.unresolvedReferenceUsages, key)
}

func matchOriginalLinkUsage(original []parser.LinkUsage, candidate parser.LinkUsage, seen map[int]bool) bool {
	for index, usage := range original {
		if !seen[index] && usage == candidate {
			seen[index] = true
			return true
		}
	}
	return false
}

func resolvedUsageMatchesDefinition(unresolved parser.UnresolvedReferenceUsage, candidate parser.LinkUsage, label, destination, title string, hasTitle bool) bool {
	return candidate.Anchor == unresolved.Anchor && candidate.Kind == unresolved.Kind && candidate.Form == unresolved.Form &&
		referenceLabelKey(candidate.Reference) == referenceLabelKey(label) && candidate.Destination == destination && candidate.Title == title && candidate.HasTitle == hasTitle
}

func unresolvedReferencesAfterAppendMatch(original, candidate []parser.UnresolvedReferenceUsage, resolvedKey string) bool {
	expected := make([]parser.UnresolvedReferenceUsage, 0, len(original))
	for _, usage := range original {
		if referenceLabelKey(usage.Reference) != resolvedKey {
			expected = append(expected, usage)
		}
	}
	if len(expected) != len(candidate) {
		return false
	}
	for index := range expected {
		if expected[index] != candidate[index] {
			return false
		}
	}
	return true
}
