package splice

import (
	"bytes"

	"github.com/zoster81/marksplice/internal/source"
)

// PrepareRemoveBlockquote prepares exact removal of one complete promoted top-level blockquote container.
func (d *Document) PrepareRemoveBlockquote(id NodeID) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindBlockquote, "blockquote")
	if err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := d.blockquoteSource(target)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	removeRange := mapping.LineRange
	if !removeRange.Valid(len(d.source)) || removeRange.Start >= removeRange.End || !rangesOverlap(target.Range, removeRange) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	change, candidate, err := d.prepareCandidateChange(removeRange, nil, "blockquote removal")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateNodeSurvivorsAfterRemoval(candidate, removeRange); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareReplaceBlockquoteContent replaces the complete inner content of one
// source-proven top-level blockquote whose physical lines use one uniform proven
// marker prefix and line-ending style. Alert-shaped blockquotes require the
// alert-specific mutation APIs.
func (d *Document) PrepareReplaceBlockquoteContent(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindBlockquote, "blockquote")
	if err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := d.blockquoteSource(target)
	if !ok || blockquoteMappingAlertKind(d.source, mapping) != source.AlertUnknown {
		return ChangeSet{}, ErrInvalidReplacement
	}
	lines, ok := splitBlockquoteReplacement(replacement)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	profile, ok := uniformBlockquoteProfile(d.source, mapping.LineRange, mapping.ContentRanges)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	fragment, ok := buildQuotedReplacement(profile, lines)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if change, ok := d.unchangedRangeChange(mapping.LineRange, fragment); ok {
		return change, nil
	}
	change, candidate, err := d.prepareCandidateChange(mapping.LineRange, fragment, "blockquote content replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateBlockquoteContentReplacement(candidate, mapping, profile, lines, false); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareSetAlertMarker replaces the marker payload of one source-proven GitHub alert.
func (d *Document) PrepareSetAlertMarker(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindBlockquote, "blockquote")
	if err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := d.blockquoteSource(target)
	if !ok || len(mapping.ContentRanges) < 2 || blockquoteMappingAlertKind(d.source, mapping) == source.AlertUnknown || source.AlertKindFromMarker(replacement) == source.AlertUnknown {
		return ChangeSet{}, ErrInvalidReplacement
	}
	markerRange := mapping.ContentRanges[0]
	if change, ok := d.unchangedRangeChange(markerRange, replacement); ok {
		return change, nil
	}
	change, candidate, err := d.prepareCandidateChange(markerRange, replacement, "alert kind replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateAlertMarkerReplacement(candidate, d.source, mapping, replacement); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

// PrepareReplaceAlertBody replaces all body lines after one source-proven alert
// marker while preserving the existing uniform body marker prefix and EOL style.
func (d *Document) PrepareReplaceAlertBody(id NodeID, replacement []byte) (ChangeSet, error) {
	target, err := d.editableTargetNode(id, KindBlockquote, "blockquote")
	if err != nil {
		return ChangeSet{}, err
	}
	mapping, ok := d.blockquoteSource(target)
	if !ok || len(mapping.ContentRanges) < 2 || blockquoteMappingAlertKind(d.source, mapping) == source.AlertUnknown {
		return ChangeSet{}, ErrInvalidReplacement
	}
	lines, ok := splitBlockquoteReplacement(replacement)
	if !ok || !blockquoteReplacementHasContent(lines) {
		return ChangeSet{}, ErrInvalidReplacement
	}
	bodyRanges := mapping.ContentRanges[1:]
	bodyStart := blockquotePhysicalLineStart(d.source, bodyRanges[0].Start, mapping.LineRange.Start)
	if bodyStart <= mapping.LineRange.Start || bodyStart >= mapping.LineRange.End {
		return ChangeSet{}, ErrInvalidReplacement
	}
	bodyRange := Range{Start: bodyStart, End: mapping.LineRange.End}
	profile, ok := uniformBlockquoteProfile(d.source, bodyRange, bodyRanges)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	fragment, ok := buildQuotedReplacement(profile, lines)
	if !ok {
		return ChangeSet{}, ErrInvalidReplacement
	}
	if change, ok := d.unchangedRangeChange(bodyRange, fragment); ok {
		return change, nil
	}
	change, candidate, err := d.prepareCandidateChange(bodyRange, fragment, "alert body replacement")
	if err != nil {
		return ChangeSet{}, err
	}
	if err := validateAlertBodyReplacement(candidate, d.source, mapping, profile, lines); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

type blockquotePrefixProfile struct {
	prefix      []byte
	eol         []byte
	trailingEOL bool
}

func uniformBlockquoteProfile(input []byte, owned Range, contentRanges []Range) (blockquotePrefixProfile, bool) {
	if !owned.Valid(len(input)) || owned.Start >= owned.End || len(contentRanges) == 0 {
		return blockquotePrefixProfile{}, false
	}
	var profile blockquotePrefixProfile
	for index, content := range contentRanges {
		prefix, eol, ok := blockquoteLineProfile(input, owned, content)
		if !ok || !mergeBlockquotePrefix(&profile, prefix, index == 0) {
			return blockquotePrefixProfile{}, false
		}
		if !mergeBlockquoteLineEnding(&profile, eol, index+1 == len(contentRanges)) {
			return blockquotePrefixProfile{}, false
		}
	}
	return profile, true
}

func blockquoteLineProfile(input []byte, owned, content Range) ([]byte, []byte, bool) {
	if !content.Valid(len(input)) || content.Start < owned.Start || content.End > owned.End {
		return nil, nil, false
	}
	lineStart := blockquotePhysicalLineStart(input, content.Start, owned.Start)
	if lineStart < owned.Start || lineStart > content.Start {
		return nil, nil, false
	}
	prefix := input[lineStart:content.Start]
	if len(prefix) == 0 || !bytes.Contains(prefix, []byte{'>'}) {
		return nil, nil, false
	}
	return prefix, blockquoteLineEndingAt(input, content.End, owned.End), true
}

func mergeBlockquotePrefix(profile *blockquotePrefixProfile, prefix []byte, first bool) bool {
	if first {
		profile.prefix = append([]byte(nil), prefix...)
		return true
	}
	return bytes.Equal(prefix, profile.prefix)
}

func mergeBlockquoteLineEnding(profile *blockquotePrefixProfile, eol []byte, last bool) bool {
	if !last && len(eol) == 0 {
		return false
	}
	if len(eol) == 0 {
		return true
	}
	if len(profile.eol) == 0 {
		profile.eol = append([]byte(nil), eol...)
	} else if !bytes.Equal(eol, profile.eol) {
		return false
	}
	if last {
		profile.trailingEOL = true
	}
	return true
}

func blockquotePhysicalLineStart(input []byte, offset, lower int) int {
	if offset < lower || offset > len(input) {
		return -1
	}
	start := offset
	for start > lower && input[start-1] != '\n' && input[start-1] != '\r' {
		start--
	}
	return start
}

func blockquoteLineEndingAt(input []byte, offset, limit int) []byte {
	if offset < 0 || offset > limit || limit > len(input) {
		return nil
	}
	if offset+1 < limit && input[offset] == '\r' && input[offset+1] == '\n' {
		return input[offset : offset+2]
	}
	if offset < limit && (input[offset] == '\n' || input[offset] == '\r') {
		return input[offset : offset+1]
	}
	return nil
}

func splitBlockquoteReplacement(replacement []byte) ([][]byte, bool) {
	if len(replacement) == 0 || replacement[len(replacement)-1] == '\r' || replacement[len(replacement)-1] == '\n' {
		return nil, false
	}
	lines := make([][]byte, 0, bytes.Count(replacement, []byte{'\n'})+1)
	start := 0
	for index := 0; index < len(replacement); index++ {
		switch replacement[index] {
		case '\r':
			lines = append(lines, replacement[start:index])
			if index+1 < len(replacement) && replacement[index+1] == '\n' {
				index++
			}
			start = index + 1
		case '\n':
			lines = append(lines, replacement[start:index])
			start = index + 1
		}
	}
	lines = append(lines, replacement[start:])
	return lines, len(lines) != 0
}

func blockquoteReplacementHasContent(lines [][]byte) bool {
	for _, line := range lines {
		if len(line) != 0 {
			return true
		}
	}
	return false
}

func buildQuotedReplacement(profile blockquotePrefixProfile, lines [][]byte) ([]byte, bool) {
	if len(profile.prefix) == 0 || len(lines) == 0 || len(lines) > 1 && len(profile.eol) == 0 {
		return nil, false
	}
	capacity := 0
	for _, line := range lines {
		capacity += len(profile.prefix) + len(line)
	}
	capacity += max(0, len(lines)-1) * len(profile.eol)
	if profile.trailingEOL {
		capacity += len(profile.eol)
	}
	fragment := make([]byte, 0, capacity)
	for index, line := range lines {
		fragment = append(fragment, profile.prefix...)
		fragment = append(fragment, line...)
		if index+1 < len(lines) || profile.trailingEOL {
			fragment = append(fragment, profile.eol...)
		}
	}
	return fragment, true
}

func blockquoteMappingAlertKind(input []byte, mapping source.BlockquoteMapping) source.AlertKind {
	if len(mapping.ContentRanges) < 2 || !mapping.ContentRanges[0].Valid(len(input)) {
		return source.AlertUnknown
	}
	return source.AlertKindFromMarker(input[mapping.ContentRanges[0].Start:mapping.ContentRanges[0].End])
}

func validateBlockquoteContentReplacement(candidate []byte, original source.BlockquoteMapping, originalProfile blockquotePrefixProfile, lines [][]byte, requireAlert bool) error {
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ErrInvalidReplacement
	}
	mapping, ok := candidateBlockquoteAt(candidateDocument, original.LineRange.Start)
	if !ok || len(mapping.ContentRanges) != len(lines) || mapping.LineRange.Start != original.LineRange.Start {
		return ErrInvalidReplacement
	}
	if requireAlert != (blockquoteMappingAlertKind(candidate, mapping) != source.AlertUnknown) {
		return ErrInvalidReplacement
	}
	profile, ok := uniformBlockquoteProfile(candidate, mapping.LineRange, mapping.ContentRanges)
	if !ok || !sameBlockquoteProfile(originalProfile, profile) || !blockquoteContentsEqual(candidate, mapping.ContentRanges, lines) {
		return ErrInvalidReplacement
	}
	return nil
}

func validateAlertMarkerReplacement(candidate, originalSource []byte, original source.BlockquoteMapping, replacement []byte) error {
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ErrInvalidReplacement
	}
	mapping, ok := candidateBlockquoteAt(candidateDocument, original.LineRange.Start)
	if !ok || len(mapping.ContentRanges) != len(original.ContentRanges) || source.AlertKindFromMarker(replacement) == source.AlertUnknown {
		return ErrInvalidReplacement
	}
	delta := len(replacement) - (original.ContentRanges[0].End - original.ContentRanges[0].Start)
	if mapping.LineRange != shiftedEnd(original.LineRange, delta) || mapping.ContentRanges[0] != rangeWithLength(original.ContentRanges[0].Start, len(replacement)) || blockquoteMappingAlertKind(candidate, mapping) != source.AlertKindFromMarker(replacement) {
		return ErrInvalidReplacement
	}
	for index := 1; index < len(original.ContentRanges); index++ {
		if mapping.ContentRanges[index] != shiftedRange(original.ContentRanges[index], delta) {
			return ErrInvalidReplacement
		}
		oldRange := original.ContentRanges[index]
		newRange := mapping.ContentRanges[index]
		if !oldRange.Valid(len(originalSource)) || !newRange.Valid(len(candidate)) || !bytes.Equal(candidate[newRange.Start:newRange.End], originalSource[oldRange.Start:oldRange.End]) {
			return ErrInvalidReplacement
		}
	}
	return nil
}

func validateAlertBodyReplacement(candidate, originalSource []byte, original source.BlockquoteMapping, originalProfile blockquotePrefixProfile, lines [][]byte) error {
	candidateDocument, err := Parse(candidate)
	if err != nil {
		return ErrInvalidReplacement
	}
	mapping, ok := candidateBlockquoteAt(candidateDocument, original.LineRange.Start)
	if !ok || len(mapping.ContentRanges) != len(lines)+1 || mapping.ContentRanges[0] != original.ContentRanges[0] {
		return ErrInvalidReplacement
	}
	if blockquoteMappingAlertKind(candidate, mapping) != blockquoteMappingAlertKind(originalSource, original) {
		return ErrInvalidReplacement
	}
	bodyStart := blockquotePhysicalLineStart(candidate, mapping.ContentRanges[1].Start, mapping.LineRange.Start)
	bodyOwned := Range{Start: bodyStart, End: mapping.LineRange.End}
	profile, ok := uniformBlockquoteProfile(candidate, bodyOwned, mapping.ContentRanges[1:])
	if !ok || !sameBlockquoteProfile(originalProfile, profile) || !blockquoteContentsEqual(candidate, mapping.ContentRanges[1:], lines) {
		return ErrInvalidReplacement
	}
	return nil
}

func candidateBlockquoteAt(document *Document, lineStart int) (source.BlockquoteMapping, bool) {
	if document == nil {
		return source.BlockquoteMapping{}, false
	}
	for _, node := range document.nodes {
		if node.Kind != KindBlockquote || !node.TopLevel || !node.Editable {
			continue
		}
		mapping, ok := document.blockquoteSource(node)
		if ok && mapping.LineRange.Start == lineStart {
			return mapping, true
		}
	}
	return source.BlockquoteMapping{}, false
}

func sameBlockquoteProfile(left, right blockquotePrefixProfile) bool {
	return bytes.Equal(left.prefix, right.prefix) && bytes.Equal(left.eol, right.eol) && left.trailingEOL == right.trailingEOL
}

func blockquoteContentsEqual(input []byte, ranges []Range, lines [][]byte) bool {
	if len(ranges) != len(lines) {
		return false
	}
	for index, range_ := range ranges {
		if !range_.Valid(len(input)) || !bytes.Equal(input[range_.Start:range_.End], lines[index]) {
			return false
		}
	}
	return true
}
