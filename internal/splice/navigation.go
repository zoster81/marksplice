package splice

import (
	"bytes"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/zoster81/marksplice/internal/source"
)

// HeadingAnchor is one source-ordered GitHub-compatible heading anchor.
type HeadingAnchor struct {
	HeadingID NodeID
	Value     string
}

// FragmentTargetKind identifies one supported intra-document fragment target.
type FragmentTargetKind uint8

const (
	FragmentTargetUnknown FragmentTargetKind = iota
	FragmentTargetHeading
	FragmentTargetHTMLAnchor
)

// FragmentTarget identifies one uniquely resolved intra-document fragment target.
type FragmentTarget struct {
	Kind   FragmentTargetKind
	NodeID NodeID
	Value  string
}

func (d *Document) HeadingAnchors() []HeadingAnchor {
	if d == nil || len(d.sections) == 0 {
		return nil
	}
	anchors := make([]HeadingAnchor, 0, len(d.sections))
	used := make(map[string]int, len(d.sections))
	for _, section := range d.sections {
		node, ok := d.nodeByID(section.HeadingID)
		if !ok || node.Kind != KindHeading || !node.Editable || !node.TopLevel {
			continue
		}
		anchors = append(anchors, HeadingAnchor{
			HeadingID: node.ID,
			Value:     uniqueHeadingSlug(githubHeadingSlug(node.HeadingText), used),
		})
	}
	return anchors
}

func (d *Document) HeadingAnchor(id NodeID) (HeadingAnchor, bool) {
	for _, anchor := range d.HeadingAnchors() {
		if anchor.HeadingID == id {
			return anchor, true
		}
	}
	return HeadingAnchor{}, false
}

func githubHeadingSlug(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	var slug strings.Builder
	slug.Grow(len(text))
	for _, r := range text {
		switch {
		case r == ' ':
			slug.WriteByte('-')
		case r == '-' || r == '_':
			slug.WriteRune(r)
		case unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r):
			slug.WriteRune(r)
		}
	}
	return slug.String()
}

func uniqueHeadingSlug(base string, used map[string]int) string {
	if _, exists := used[base]; !exists {
		used[base] = 0
		return base
	}
	for suffix := used[base] + 1; ; suffix++ {
		candidate := base + "-" + strconv.Itoa(suffix)
		if _, exists := used[candidate]; exists {
			continue
		}
		used[base] = suffix
		used[candidate] = 0
		return candidate
	}
}

func normalizeFragment(fragment string) (string, bool) {
	fragment = strings.TrimPrefix(fragment, "#")
	if fragment == "" || strings.Contains(fragment, "#") {
		return "", false
	}
	decoded, err := url.PathUnescape(fragment)
	if err != nil || decoded == "" {
		return "", false
	}
	return decoded, true
}

type fragmentResolutionStatus uint8

const (
	fragmentResolutionInvalid fragmentResolutionStatus = iota
	fragmentResolutionMissing
	fragmentResolutionAmbiguous
	fragmentResolutionResolved
)

func (d *Document) ResolveFragment(fragment string) (FragmentTarget, bool) {
	target, status := d.resolveFragment(fragment)
	return target, status == fragmentResolutionResolved
}

type fragmentCatalogEntry struct {
	target FragmentTarget
	count  int
}

type fragmentCatalog map[string]fragmentCatalogEntry

func (d *Document) resolveFragment(fragment string) (FragmentTarget, fragmentResolutionStatus) {
	if d == nil {
		return FragmentTarget{}, fragmentResolutionInvalid
	}
	return resolveFragmentFromCatalog(fragment, d.fragmentCatalog())
}

// FragmentResolver returns an ephemeral resolver backed by one fragment catalog
// derived from this immutable snapshot. The catalog is owned only by the returned
// closure and is never retained by Document.
func (d *Document) FragmentResolver() func(string) (FragmentTarget, bool) {
	resolveStatus := d.FragmentStatusResolver()
	return func(fragment string) (FragmentTarget, bool) {
		target, status := resolveStatus(fragment)
		return target, status == LinkFragmentResolved
	}
}

// FragmentStatusResolver returns an ephemeral resolver that reuses one fragment
// catalog while preserving exact invalid/missing/ambiguous/resolved classification.
func (d *Document) FragmentStatusResolver() func(string) (FragmentTarget, LinkFragmentStatus) {
	if d == nil {
		return func(string) (FragmentTarget, LinkFragmentStatus) {
			return FragmentTarget{}, LinkFragmentInvalid
		}
	}
	catalog := d.fragmentCatalog()
	return func(fragment string) (FragmentTarget, LinkFragmentStatus) {
		target, status := resolveFragmentFromCatalog(fragment, catalog)
		return target, linkFragmentStatus(status)
	}
}

func resolveFragmentFromCatalog(fragment string, catalog fragmentCatalog) (FragmentTarget, fragmentResolutionStatus) {
	value, ok := normalizeFragment(fragment)
	if !ok {
		return FragmentTarget{}, fragmentResolutionInvalid
	}
	entry, exists := catalog[value]
	if !exists || entry.count == 0 {
		return FragmentTarget{}, fragmentResolutionMissing
	}
	if entry.count != 1 {
		return FragmentTarget{}, fragmentResolutionAmbiguous
	}
	return entry.target, fragmentResolutionResolved
}

func (d *Document) fragmentCatalog() fragmentCatalog {
	catalog := make(fragmentCatalog, len(d.sections))
	for _, anchor := range d.HeadingAnchors() {
		addFragmentTarget(catalog, FragmentTarget{Kind: FragmentTargetHeading, NodeID: anchor.HeadingID, Value: anchor.Value})
	}
	for _, node := range d.nodes {
		if node.Kind != KindHTMLAnchor || !node.Editable || !node.ContentRange.Valid(len(d.source)) {
			continue
		}
		value := string(d.source[node.ContentRange.Start:node.ContentRange.End])
		addFragmentTarget(catalog, FragmentTarget{Kind: FragmentTargetHTMLAnchor, NodeID: node.ID, Value: value})
	}
	return catalog
}

func addFragmentTarget(catalog fragmentCatalog, target FragmentTarget) {
	entry := catalog[target.Value]
	entry.target = target
	entry.count++
	catalog[target.Value] = entry
}

func (d *Document) ValidateFragment(fragment string) bool {
	_, ok := d.ResolveFragment(fragment)
	return ok
}

func (d *Document) GenerateTOC() []byte {
	return d.renderTOC("\n")
}

func (d *Document) renderTOC(eol string) []byte {
	if d == nil || len(d.sections) == 0 {
		return nil
	}
	anchors := d.HeadingAnchors()
	if len(anchors) != len(d.sections) {
		return nil
	}
	depths, ok := d.sectionDepths()
	if !ok {
		return nil
	}
	var toc strings.Builder
	for index, section := range d.sections {
		node, ok := d.nodeByID(section.HeadingID)
		if !ok || node.Kind != KindHeading {
			return nil
		}
		toc.WriteString(strings.Repeat("  ", depths[index]))
		toc.WriteString("- [")
		toc.WriteString(escapeTOCLabel(node.HeadingText))
		toc.WriteString("](#")
		toc.WriteString(anchors[index].Value)
		toc.WriteByte(')')
		toc.WriteString(eol)
	}
	return []byte(toc.String())
}

func (d *Document) sectionDepths() ([]int, bool) {
	depths := make([]int, len(d.sections))
	for index, section := range d.sections {
		if !section.HasParent {
			continue
		}
		parentIndex, ok := d.sectionIndex[section.ParentHeadingID]
		if !ok || parentIndex < 0 || parentIndex >= index {
			return nil, false
		}
		depths[index] = depths[parentIndex] + 1
	}
	return depths, true
}

func escapeTOCLabel(text string) string {
	var escaped strings.Builder
	escaped.Grow(len(text))
	for _, r := range text {
		if r == '\\' || r == '[' || r == ']' {
			escaped.WriteByte('\\')
		}
		escaped.WriteRune(r)
	}
	return escaped.String()
}

type tocBodyProfile struct {
	eol      string
	leading  []byte
	trailing []byte
}

// TOCStatus records whether one explicitly designated section body is recognized
// as managed TOC source and, when recognized, whether it is stale.
type TOCStatus struct {
	HeadingID  NodeID
	Stale      bool
	Recognized bool
}

func (d *Document) TOCStale(id NodeID) (bool, bool) {
	status := d.tocStatus(id, make(map[string][]byte))
	return status.Stale, status.Recognized
}

// TOCStatuses evaluates multiple managed-TOC candidates while reusing generated
// TOC bytes for matching line-ending forms. The cache is call-local only.
func (d *Document) TOCStatuses(ids []NodeID) []TOCStatus {
	result := make([]TOCStatus, len(ids))
	cache := make(map[string][]byte)
	for index, id := range ids {
		result[index] = d.tocStatus(id, cache)
	}
	return result
}

func (d *Document) tocStatus(id NodeID, cache map[string][]byte) TOCStatus {
	status := TOCStatus{HeadingID: id}
	section, _, ok := d.sectionByHeadingID(id)
	if !ok || !section.BodyRange.Valid(len(d.source)) {
		return status
	}
	profile, ok := d.tocBodyProfile(section.BodyRange)
	if !ok {
		return status
	}
	expected := profile.render(d.cachedRenderedTOC(cache, profile.eol))
	actual := d.source[section.BodyRange.Start:section.BodyRange.End]
	status.Stale = !bytes.Equal(actual, expected)
	status.Recognized = true
	return status
}

func (d *Document) cachedRenderedTOC(cache map[string][]byte, eol string) []byte {
	if toc, ok := cache[eol]; ok {
		return toc
	}
	toc := d.renderTOC(eol)
	cache[eol] = toc
	return toc
}

// PrepareSyncTOC prepares replacement of one explicitly designated TOC-shaped
// section body while preserving the section hierarchy and untouched source.
func (d *Document) PrepareSyncTOC(id NodeID) (ChangeSet, error) {
	section, _, err := d.sectionTarget(id)
	if err != nil {
		return ChangeSet{}, err
	}
	profile, ok := d.tocBodyProfile(section.BodyRange)
	if !ok {
		return ChangeSet{}, ErrInvalidTargetKind
	}
	replacement := profile.render(d.renderTOC(profile.eol))
	return d.PrepareReplaceSectionBody(id, replacement)
}

type tocSyncPlan struct {
	section      Section
	sectionIndex int
	replacement  []byte
}

// PrepareSyncTOCs atomically synchronizes multiple explicitly designated managed
// TOC bodies from one immutable snapshot and validates one combined candidate.
func (d *Document) PrepareSyncTOCs(ids []NodeID) (ChangeSet, error) {
	if len(ids) == 0 {
		return d.newChanges(nil, "managed TOC synchronization")
	}
	plans, err := d.planTOCSyncs(ids)
	if err != nil {
		return ChangeSet{}, err
	}
	patches := make([]source.Patch, len(plans))
	transforms := make([]patchTransform, len(plans))
	for index, plan := range plans {
		patches[index] = source.Patch{Range: plan.section.BodyRange, Replacement: plan.replacement}
		transforms[index] = patchTransform{Range: plan.section.BodyRange, ReplacementLength: len(plan.replacement)}
	}
	change, candidate, err := d.prepareCandidateChanges(patches, "managed TOC synchronization")
	if err != nil {
		return ChangeSet{}, err
	}
	candidateDocument, err := parseSectionMutationCandidate(candidate)
	if err != nil {
		return ChangeSet{}, err
	}
	if err := d.validateSectionHeadingsAfterPatches(candidate, candidateDocument, transforms); err != nil {
		return ChangeSet{}, err
	}
	if err := validateTOCSyncBodyRanges(candidateDocument, plans); err != nil {
		return ChangeSet{}, err
	}
	return change, nil
}

func (d *Document) planTOCSyncs(ids []NodeID) ([]tocSyncPlan, error) {
	plans := make([]tocSyncPlan, 0, len(ids))
	seen := make(map[NodeID]bool, len(ids))
	tocCache := make(map[string][]byte)
	for _, id := range ids {
		if seen[id] {
			return nil, ErrInvalidReplacement
		}
		seen[id] = true
		section, sectionIndex, err := d.sectionTarget(id)
		if err != nil {
			return nil, err
		}
		profile, ok := d.tocBodyProfile(section.BodyRange)
		if !ok {
			return nil, ErrInvalidTargetKind
		}
		plans = append(plans, tocSyncPlan{
			section:      section,
			sectionIndex: sectionIndex,
			replacement:  profile.render(d.cachedRenderedTOC(tocCache, profile.eol)),
		})
	}
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].section.BodyRange.Start < plans[j].section.BodyRange.Start
	})
	return plans, nil
}

func validateTOCSyncBodyRanges(candidateDocument *Document, plans []tocSyncPlan) error {
	delta := 0
	for _, plan := range plans {
		candidateSection, ok := candidateDocument.SectionAt(plan.sectionIndex)
		if !ok {
			return ErrInvalidReplacement
		}
		start := plan.section.BodyRange.Start + delta
		expected := Range{Start: start, End: start + len(plan.replacement)}
		if candidateSection.BodyRange != expected {
			return ErrInvalidReplacement
		}
		delta += len(plan.replacement) - (plan.section.BodyRange.End - plan.section.BodyRange.Start)
	}
	return nil
}

func (p tocBodyProfile) render(toc []byte) []byte {
	result := make([]byte, 0, len(p.leading)+len(toc)+len(p.trailing))
	result = append(result, p.leading...)
	result = append(result, toc...)
	return append(result, p.trailing...)
}

func (d *Document) tocBodyProfile(range_ Range) (tocBodyProfile, bool) {
	if d == nil || !range_.Valid(len(d.source)) {
		return tocBodyProfile{}, false
	}
	body := d.source[range_.Start:range_.End]
	scan := tocBodyScan{firstEntryStart: -1}
	for offset := 0; offset < len(body); {
		lineEnd, next, lineEOL := physicalLine(body, offset)
		if !scan.observeLine(body[offset:lineEnd], offset, next, lineEOL) {
			return tocBodyProfile{}, false
		}
		offset = next
	}
	return scan.profile(body, d.preferredLineEnding()), true
}

type tocBodyScan struct {
	eol                  string
	firstEntryStart      int
	lastEntryEnd         int
	previousDepth        int
	hasEntry             bool
	trailingBlankStarted bool
}

func (s *tocBodyScan) observeLine(line []byte, offset, next int, eol string) bool {
	if !s.observeLineEnding(eol) {
		return false
	}
	if len(bytes.TrimSpace(line)) == 0 {
		if s.hasEntry {
			s.trailingBlankStarted = true
		}
		return true
	}
	if s.trailingBlankStarted {
		return false
	}
	depth, ok := tocEntryDepth(line)
	if !ok || !validTOCEntryDepth(s.hasEntry, s.previousDepth, depth) {
		return false
	}
	if s.firstEntryStart < 0 {
		s.firstEntryStart = offset
	}
	s.previousDepth = depth
	s.hasEntry = true
	s.lastEntryEnd = next
	return true
}

func (s *tocBodyScan) observeLineEnding(eol string) bool {
	if eol == "" {
		return true
	}
	if s.eol == "" {
		s.eol = eol
		return true
	}
	return s.eol == eol
}

func validTOCEntryDepth(hasEntry bool, previousDepth, depth int) bool {
	if !hasEntry {
		return depth == 0
	}
	return depth <= previousDepth+1
}

func (s tocBodyScan) profile(body []byte, fallbackEOL string) tocBodyProfile {
	eol := s.eol
	if eol == "" {
		eol = fallbackEOL
	}
	if eol == "" {
		eol = "\n"
	}
	if !s.hasEntry {
		return tocBodyProfile{eol: eol, leading: append([]byte(nil), body...)}
	}
	return tocBodyProfile{
		eol:      eol,
		leading:  append([]byte(nil), body[:s.firstEntryStart]...),
		trailing: append([]byte(nil), body[s.lastEntryEnd:]...),
	}
}

func physicalLine(source []byte, start int) (lineEnd, next int, eol string) {
	for index := start; index < len(source); index++ {
		switch source[index] {
		case '\n':
			return index, index + 1, "\n"
		case '\r':
			if index+1 < len(source) && source[index+1] == '\n' {
				return index, index + 2, "\r\n"
			}
			return index, index + 1, "\r"
		}
	}
	return len(source), len(source), ""
}

func tocEntryDepth(line []byte) (int, bool) {
	spaces := 0
	for spaces < len(line) && line[spaces] == ' ' {
		spaces++
	}
	if spaces%2 != 0 || spaces+4 > len(line) || string(line[spaces:spaces+3]) != "- [" {
		return 0, false
	}
	labelEnd := tocLabelEnd(line, spaces+3)
	if labelEnd < 0 || labelEnd+4 > len(line) || string(line[labelEnd:labelEnd+3]) != "](#" || line[len(line)-1] != ')' {
		return 0, false
	}
	fragment := string(line[labelEnd+3 : len(line)-1])
	if fragment != "" {
		if _, ok := normalizeFragment(fragment); !ok {
			return 0, false
		}
	}
	return spaces / 2, true
}

func tocLabelEnd(line []byte, start int) int {
	escaped := false
	for index := start; index < len(line); index++ {
		if escaped {
			escaped = false
			continue
		}
		switch line[index] {
		case '\\':
			escaped = true
		case ']':
			return index
		}
	}
	return -1
}

func (d *Document) preferredLineEnding() string {
	for index := 0; index < len(d.source); index++ {
		switch d.source[index] {
		case '\n':
			return "\n"
		case '\r':
			if index+1 < len(d.source) && d.source[index+1] == '\n' {
				return "\r\n"
			}
			return "\r"
		}
	}
	return ""
}
