package splice

import "fmt"

// Section is a derived source-bound view governed by one supported document heading.
type Section struct {
	HeadingID        NodeID
	Level            int
	Range            Range
	BodyRange        Range
	ParentHeadingID  NodeID
	HasParent        bool
	firstChildIndex  int
	nextSiblingIndex int
	childCount       int
}

type sectionHeading struct {
	ID    NodeID
	Level int
	Range Range
}

func buildSections(source []byte, nodes []Node) ([]Section, map[NodeID]int, error) {
	headings, err := collectSectionHeadings(source, nodes)
	if err != nil {
		return nil, nil, err
	}

	sections := make([]Section, len(headings))
	index := make(map[NodeID]int, len(headings))
	stack := make([]int, 0, len(headings))
	lastChildIndex := make([]int, len(headings))
	for i, heading := range headings {
		nextHeadingStart := len(source)
		if i+1 < len(headings) {
			nextHeadingStart = headings[i+1].Range.Start
		}
		section, err := newSection(source, heading, nextHeadingStart)
		if err != nil {
			return nil, nil, err
		}

		for len(stack) != 0 && sections[stack[len(stack)-1]].Level >= heading.Level {
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			sections[open].Range.End = heading.Range.Start
		}
		if _, exists := index[heading.ID]; exists {
			return nil, nil, fmt.Errorf("duplicate section heading ID %q", heading.ID)
		}
		if len(stack) != 0 {
			parentIndex := stack[len(stack)-1]
			section.ParentHeadingID = sections[parentIndex].HeadingID
			section.HasParent = true
			if err := linkSectionChild(sections, lastChildIndex, parentIndex, i); err != nil {
				return nil, nil, err
			}
		}

		sections[i] = section
		index[heading.ID] = i
		stack = append(stack, i)
	}
	return sections, index, nil
}

func collectSectionHeadings(source []byte, nodes []Node) ([]sectionHeading, error) {
	headings := make([]sectionHeading, 0)
	for _, node := range nodes {
		if node.Kind != KindHeading || !node.Editable || !node.TopLevel {
			continue
		}
		if !node.Range.Valid(len(source)) || node.Range.Start == node.Range.End || node.Level < 1 || node.Level > 6 {
			return nil, fmt.Errorf("invalid section heading %q: level %d range [%d,%d)", node.ID, node.Level, node.Range.Start, node.Range.End)
		}
		if len(headings) != 0 && node.Range.Start <= headings[len(headings)-1].Range.Start {
			previous := headings[len(headings)-1]
			return nil, fmt.Errorf("section headings out of source order: %q at %d after %q at %d", node.ID, node.Range.Start, previous.ID, previous.Range.Start)
		}
		headings = append(headings, sectionHeading{ID: node.ID, Level: node.Level, Range: node.Range})
	}
	return headings, nil
}

func newSection(source []byte, heading sectionHeading, nextHeadingStart int) (Section, error) {
	bodyStart, err := sectionBodyStart(source, heading.Range.End)
	if err != nil {
		return Section{}, fmt.Errorf("section heading %q: %w", heading.ID, err)
	}
	if bodyStart > nextHeadingStart {
		return Section{}, fmt.Errorf("section heading %q body starts at %d after next heading at %d", heading.ID, bodyStart, nextHeadingStart)
	}
	return Section{
		HeadingID:        heading.ID,
		Level:            heading.Level,
		Range:            Range{Start: heading.Range.Start, End: len(source)},
		BodyRange:        Range{Start: bodyStart, End: nextHeadingStart},
		firstChildIndex:  -1,
		nextSiblingIndex: -1,
	}, nil
}

func linkSectionChild(sections []Section, lastChildIndex []int, parentIndex, childIndex int) error {
	if parentIndex < 0 || parentIndex >= childIndex || childIndex >= len(sections) || parentIndex >= len(lastChildIndex) {
		return fmt.Errorf("invalid section child linkage indexes parent=%d child=%d", parentIndex, childIndex)
	}
	parent := &sections[parentIndex]
	if parent.childCount < 0 || parent.firstChildIndex < -1 {
		return fmt.Errorf("invalid section child linkage for parent %q", parent.HeadingID)
	}
	if parent.childCount == 0 {
		if parent.firstChildIndex != -1 {
			return fmt.Errorf("invalid section first-child linkage for parent %q", parent.HeadingID)
		}
		parent.firstChildIndex = childIndex
	} else {
		previousChild := lastChildIndex[parentIndex]
		if previousChild <= parentIndex || previousChild >= childIndex || sections[previousChild].nextSiblingIndex != -1 {
			return fmt.Errorf("invalid section sibling linkage for parent %q", parent.HeadingID)
		}
		sections[previousChild].nextSiblingIndex = childIndex
	}
	lastChildIndex[parentIndex] = childIndex
	parent.childCount++
	return nil
}

func sectionBodyStart(source []byte, headingEnd int) (int, error) {
	if headingEnd < 0 || headingEnd > len(source) {
		return 0, fmt.Errorf("heading end %d is outside source length %d", headingEnd, len(source))
	}
	if headingEnd == len(source) {
		return headingEnd, nil
	}
	switch source[headingEnd] {
	case '\n':
		return headingEnd + 1, nil
	case '\r':
		if headingEnd+1 < len(source) && source[headingEnd+1] == '\n' {
			return headingEnd + 2, nil
		}
		return headingEnd + 1, nil
	default:
		return 0, fmt.Errorf("heading range ends before non-line-ending byte 0x%02x at %d", source[headingEnd], headingEnd)
	}
}

// SectionCount returns the number of derived document sections.
func (d *Document) SectionCount() int {
	if d == nil {
		return 0
	}
	return len(d.sections)
}

// SectionAt returns one derived section by source order.
func (d *Document) SectionAt(index int) (Section, bool) {
	if d == nil || index < 0 || index >= len(d.sections) {
		return Section{}, false
	}
	return d.sections[index], true
}

func (d *Document) sectionByHeadingID(id NodeID) (Section, int, bool) {
	if d == nil {
		return Section{}, 0, false
	}
	index, ok := d.sectionIndex[id]
	if !ok || index < 0 || index >= len(d.sections) {
		return Section{}, 0, false
	}
	section := d.sections[index]
	if section.HeadingID != id {
		return Section{}, 0, false
	}
	return section, index, true
}

// SectionByHeadingID returns the section governed by one heading node ID.
func (d *Document) SectionByHeadingID(id NodeID) (Section, bool) {
	section, _, ok := d.sectionByHeadingID(id)
	return section, ok
}

// SectionChildHeadingIDs returns one section's immediate child heading IDs in source order.
func (d *Document) SectionChildHeadingIDs(id NodeID) ([]NodeID, bool) {
	parent, parentIndex, ok := d.sectionByHeadingID(id)
	if !ok {
		return nil, false
	}
	if parent.firstChildIndex < -1 || parent.childCount < 0 {
		return nil, false
	}
	children := make([]NodeID, 0, parent.childCount)
	previousChildIndex := parentIndex
	for childIndex := parent.firstChildIndex; childIndex >= 0; {
		if childIndex <= previousChildIndex || childIndex >= len(d.sections) || len(children) >= parent.childCount {
			return nil, false
		}
		child := d.sections[childIndex]
		if !child.HasParent || child.ParentHeadingID != parent.HeadingID || child.nextSiblingIndex < -1 {
			return nil, false
		}
		_, indexedChild, ok := d.sectionByHeadingID(child.HeadingID)
		if !ok || indexedChild != childIndex {
			return nil, false
		}
		children = append(children, child.HeadingID)
		previousChildIndex = childIndex
		childIndex = child.nextSiblingIndex
	}
	if len(children) != parent.childCount {
		return nil, false
	}
	return children, true
}
