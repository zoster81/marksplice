package marksplice

import "github.com/zoster81/marksplice/internal/splice"

// Section is an immutable source-bound view governed by one promoted document heading.
type Section struct {
	headingID       NodeID
	level           int
	sourceRange     Range
	bodyRange       Range
	parentHeadingID NodeID
	hasParent       bool
}

// HeadingID returns the snapshot-scoped heading node identity governing this section.
func (s Section) HeadingID() NodeID { return s.headingID }

// Level returns the governing GFM heading level from 1 through 6.
func (s Section) Level() int { return s.level }

// Range returns the complete section subtree source span, including its heading.
// The range ends immediately before the next heading of equal or higher level, or at end of source.
func (s Section) Range() Range { return s.sourceRange }

// BodyRange returns the direct body source span after the heading line and before the next heading of any level.
// Nested subsection headings and their content are outside this range.
func (s Section) BodyRange() Range { return s.bodyRange }

// ParentHeadingID returns the governing heading ID of the nearest enclosing section.
func (s Section) ParentHeadingID() (NodeID, bool) { return s.parentHeadingID, s.hasParent }

// Sections returns all derived document sections in source order.
func (d *Document) Sections() []Section {
	if d == nil || d.document == nil {
		return nil
	}
	count := d.document.SectionCount()
	sections := make([]Section, 0, count)
	for i := 0; i < count; i++ {
		section, ok := d.document.SectionAt(i)
		if !ok {
			continue
		}
		sections = append(sections, publicSection(section))
	}
	return sections
}

// Section returns the derived section governed by headingID.
func (d *Document) Section(headingID NodeID) (Section, bool) {
	if d == nil || d.document == nil {
		return Section{}, false
	}
	section, ok := d.document.SectionByHeadingID(internalNodeID(headingID))
	if !ok {
		return Section{}, false
	}
	return publicSection(section), true
}

// SectionChildHeadingIDs returns one section's immediate child heading identities in source order.
func (d *Document) SectionChildHeadingIDs(headingID NodeID) ([]NodeID, bool) {
	if d == nil || d.document == nil {
		return nil, false
	}
	internalIDs, ok := d.document.SectionChildHeadingIDs(internalNodeID(headingID))
	if !ok {
		return nil, false
	}
	return publicNodeIDs(internalIDs), true
}

func publicSection(section splice.Section) Section {
	result := Section{
		headingID:   publicNodeID(section.HeadingID),
		level:       section.Level,
		sourceRange: Range{Start: section.Range.Start, End: section.Range.End},
		bodyRange:   Range{Start: section.BodyRange.Start, End: section.BodyRange.End},
		hasParent:   section.HasParent,
	}
	if section.HasParent {
		result.parentHeadingID = publicNodeID(section.ParentHeadingID)
	}
	return result
}
